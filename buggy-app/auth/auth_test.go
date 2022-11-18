package auth

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	pb "github.com/CodeYourFuture/immersive-go-course/buggy-app/auth/service"
	"github.com/CodeYourFuture/immersive-go-course/buggy-app/util"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestRun(t *testing.T) {
	passwd, err := util.ReadPasswd()
	if err != nil {
		t.Fatal(err)
	}

	config := Config{
		Port:        8010,
		DatabaseUrl: fmt.Sprintf("postgres://postgres:%s@postgres:5432/app", passwd),
		Log:         log.Default(),
	}
	as := New(config)

	var runErr error
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = as.Run(ctx)
	}()

	<-time.After(1000 * time.Millisecond)
	cancel()

	wg.Wait()
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func setupSuite(t *testing.T, DatabaseUrl string, ctx context.Context, user *userRow) (func(), error) {
	t.Helper()

	// Connect to DB to get a test user
	dbConn, err := pgx.Connect(ctx, DatabaseUrl)
	if err != nil {
		t.Fatalf("test failed to connect: %v", err)
	}

	// Insert test user
	if user != nil {
		err = dbConn.QueryRow(
			ctx,
			"INSERT INTO public.user (password, status) VALUES ($1, $2) RETURNING id",
			user.password,
			user.status,
		).Scan(&user.id)
	}

	if err != nil {
		return nil, err
	}

	return func() {

		// Delete test user
		if user != nil {
			_, err = dbConn.Exec(
				ctx,
				"DELETE FROM public.user WHERE id = $1",
				user.id,
			)
		}

		if err != nil {
			t.Fatalf("test failed to delete: %v", err)
		}
		dbConn.Close(ctx)
	}, nil
}

func TestVerify(t *testing.T) {
	tests := map[string]struct {
		user            *userRow
		requestPassword string
		wantState       pb.State
	}{
		"valid user and password": {
			user: &userRow{
				password: "$2y$10$O8VPlcAPa/iKHrkdyzN1cu7TvF5Goq6nRjSdaz9uXm1zPcVgRxQnK",
				status:   "active",
			},
			requestPassword: "banana",
			wantState:       pb.State_ALLOW,
		},
		"valid user with invalid password": {
			user: &userRow{
				password: "$2y$10$O8VPlcAPa/iKHrkdyzN1cu7TvF5Goq6nRjSdaz9uXm1zPcVgRxQnK",
				status:   "active",
			},
			requestPassword: "banana123",
			wantState:       pb.State_DENY,
		},
		"inactive user with valid password": {
			user: &userRow{
				password: "$2y$10$O8VPlcAPa/iKHrkdyzN1cu7TvF5Goq6nRjSdaz9uXm1zPcVgRxQnK",
				status:   "inactive",
			},
			requestPassword: "banana",
			wantState:       pb.State_DENY,
		},
		"inactive user with invalid password": {
			user: &userRow{
				password: "$2y$10$O8VPlcAPa/iKHrkdyzN1cu7TvF5Goq6nRjSdaz9uXm1zPcVgRxQnK",
				status:   "inactive",
			},
			requestPassword: "banana123",
			wantState:       pb.State_DENY,
		},
	}
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	passwd, err := util.ReadPasswd()
	if err != nil {
		t.Fatal(err)
	}

	config := Config{
		Port:        8010,
		DatabaseUrl: fmt.Sprintf("postgres://postgres:%s@postgres:5432/app", passwd),
		Log:         log.Default(),
	}
	as := New(config)

	var runErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = as.Run(ctx)
	}()

	<-time.After(100 * time.Millisecond)

	conn, err := grpc.Dial("localhost:8010", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	defer conn.Close()

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			teardown, err := setupSuite(t, config.DatabaseUrl, ctx, tc.user)

			defer teardown()

			if err != nil {
				t.Fatal(err)
			}

			client := pb.NewAuthClient(conn)

			result, err := client.Verify(ctx, &pb.VerifyRequest{
				Id:       tc.user.id,
				Password: tc.requestPassword,
			})
			if err != nil {
				cancel()
				wg.Wait()
				t.Fatalf("fail to dial: %v", err)
			}

			if result.State != tc.wantState {
				t.Fatalf("failed to verify, expected %v, got %v", tc.wantState, result.State)
			}
		})
	}
}
