package auth

import (
	"context"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	pb "github.com/CodeYourFuture/immersive-go-course/buggy-app/auth/service"
	"google.golang.org/grpc"
)

// Internal grpcAuthService struct that implements the gRPC server interface
// for testing
type mockGrpcAuthService struct {
	pb.UnimplementedAuthServer

	result *pb.VerifyResponse
	err    error

	Calls int
}

func newMockGrpcService(result *pb.VerifyResponse, err error) *mockGrpcAuthService {
	return &mockGrpcAuthService{
		result: result,
		err:    err,
	}
}

// Verify checks a Input for authentication validity
func (as *mockGrpcAuthService) Verify(ctx context.Context, in *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	as.Calls += 1
	return as.result, as.err
}

func TestClientCreate(t *testing.T) {
	config := Config{
		Port: 8010,
		Log:  log.Default(),
	}
	as := New(config)

	var err error
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = as.Run(ctx)
	}()

	client, err := newClientWithOpts(ctx, "localhost:8010", defaultOpts()...)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()

	<-time.After(100 * time.Millisecond)
	cancel()

	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientError(t *testing.T) {
	opts := append(defaultOpts(), grpc.WithBlock())
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := newClientWithOpts(ctx, "localhost:8010", opts...)
	if err == nil {
		t.Fatal("did not error")
	}
}

func TestClientClose(t *testing.T) {
	client, err := NewClient(context.Background(), "localhost:8010")
	if err != nil {
		t.Fatal(err)
	}
	err = client.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientVerify(t *testing.T) {

	tests := map[string]struct {
		pbStateExpected pb.State
		stateExpected   string
	}{
		"allow": {
			pbStateExpected: pb.State_ALLOW,
			stateExpected:   StateAllow,
		},
		"deny": {
			pbStateExpected: pb.State_DENY,
			stateExpected:   StateDeny,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			listen := "localhost:8010"
			lis, err := net.Listen("tcp", listen)
			if err != nil {
				t.Fatalf("failed to listen: %v", err)
			}
			mockService := newMockGrpcService(&pb.VerifyResponse{
				State: test.pbStateExpected,
			}, nil)

			// Set up and register the server
			grpcServer := grpc.NewServer()
			pb.RegisterAuthServer(grpcServer, mockService)

			var runErr error
			var wg sync.WaitGroup
			ctx, cancel := context.WithCancel(context.Background())

			wg.Add(1)
			go func() {
				defer wg.Done()
				runErr = grpcServer.Serve(lis)
			}()

			done := func() {
				cancel()
				grpcServer.GracefulStop()
				wg.Wait()
			}

			client, err := NewClient(ctx, listen)
			if err != nil {
				done()
				t.Fatal(err)
			}

			res, err := client.Verify(ctx, "example", "example")
			if err != nil {
				done()
				t.Fatal(err)
			}

			err = client.Close()
			if err != nil {
				done()
				t.Fatal(err)
			}

			if res.State != test.stateExpected {
				done()
				t.Fatalf("verify state: expected %s, got %s\n", test.stateExpected, res.State)
			}

			t.Run("verify cache", func(t *testing.T) {
				res2, err := client.Verify(ctx, "example", "example")
				if err != nil {
					done()
					t.Fatal(err)
				}

				if res2.State != test.stateExpected {
					done()
					t.Fatalf("verify state: expected %s, got %s\n", test.stateExpected, res2.State)
				}

				if mockService.Calls > 1 {
					done()
					t.Fatalf("verify calls: expected 1, got %d\n", mockService.Calls)
				}
			})

			done()
			if runErr != nil && runErr != grpc.ErrServerStopped {
				t.Fatal(runErr)
			}
		})
	}
}