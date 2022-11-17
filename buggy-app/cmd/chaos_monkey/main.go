package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/CodeYourFuture/immersive-go-course/buggy-app/util"
	"github.com/XANi/loremipsum"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/semaphore"
)

const (
	USER_PWD = "apple"
	APP_URL  = "http://localhost:8090"
)

var commands = []string{"cache", "http"}

func contains(cmds []string, cmd string) bool {
	for _, c := range cmds {
		if c == cmd {
			return true
		}
	}
	return false
}

func main() {

	if len(os.Args) < 2 {
		log.Fatalf("Invalid command, please provie what to attack (%s)", strings.Join(commands, ", "))
	}

	cmd := os.Args[1]

	if !contains(commands, cmd) {
		log.Fatalf("Invalid command, please provie what to attack (%s)", strings.Join(commands, ", "))
	}

	// Set up a default POSTGRES_PASSWORD_FILE because we know where it's likely to be...
	if os.Getenv("POSTGRES_PASSWORD_FILE") == "" {
		os.Setenv("POSTGRES_PASSWORD_FILE", "volumes/secrets/postgres-passwd")
	}
	// ... and the read it. $POSTGRES_USER will still take precedence.
	dbPasswd, err := util.ReadPasswd()
	if err != nil {
		log.Fatal(err)
	}

	// The NotifyContext will signal Done when these signals are sent, allowing others to shut down safely
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	// Connect to the database
	connString := fmt.Sprintf("postgres://postgres:%s@%s/%s?sslmode=disable", dbPasswd, "localhost:5432", "app")
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}

	// Get all users
	users, err := GetAllUsers(conn)

	if err != nil {
		log.Fatalf("error getting users: %v", err)
	}

	switch cmd {
	case "cache":
		//remove notes from users so we are only attacking the cache
		clearNotes(conn, ctx)
		err = MakeRequests(ctx, users)
	case "http":
		err = CreateNotes(conn, users[0], 1000)
		if err != nil {
			log.Fatalf("error creating notes: %v", err)
		}
		err = MakeRequests(ctx, []string{users[0], users[0], users[0], users[0]})
	}

	if err != nil {
		log.Fatalf("error attacking server: %v", err)
	}
}

func clearNotes(conn *pgx.Conn, ctx context.Context) {
	// migrate notes down
	log.Println("Clearing notes table")
	path := filepath.Join("migrations", "app", "000003_create_note_table.down.sql")

	migrate := func(path string) {
		c, ioErr := os.ReadFile(path)
		if ioErr != nil {
			log.Fatal(ioErr)
		}
		sql := string(c)
		_, err := conn.Exec(ctx, sql)
		if err != nil {
			log.Fatal(err)
		}
	}

	migrate(path)

	// migrate up
	path = filepath.Join("migrations", "app", "000003_create_note_table.up.sql")

	migrate(path)

	log.Println("Finished clearing notes")
}

func GetAllUsers(conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(context.Background(), "SELECT id FROM public.user WHERE status='active'")
	if err != nil {
		return nil, err
	}

	var users []string
	for rows.Next() {
		var user string
		err := rows.Scan(&user)
		if err != nil {
			log.Println(err)
		}
		users = append(users, user)
	}

	return users, nil
}

func MakeRequests(ctx context.Context, users []string) error {
	// authenticate all users at the same time which will overload the cache
	wg := &sync.WaitGroup{}
	sem := semaphore.NewWeighted(175)
	done := make(chan bool)
	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: 9999,
		},
	}
	id := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			log.Println("Successfully crashed server")
			return nil
		default:
			ok := sem.TryAcquire(1)
			if !ok {
				continue
			} else {
				id++
				if id >= len(users) {
					id = 0
				}
				wg.Add(1)
				go AuthenticateUser(ctx, wg, sem, client, users[id], done)
			}
		}
	}
}

func AuthenticateUser(ctx context.Context, wg *sync.WaitGroup, sem *semaphore.Weighted, client *http.Client, user string, done chan bool) {
	defer sem.Release(1)
	defer wg.Done()
	req, err := http.NewRequest("GET", APP_URL+"/1/my/notes.json", nil)
	if err != nil {
		log.Println(err)
	}
	req.SetBasicAuth(user, USER_PWD)
	_, err = client.Do(req)

	if err != nil {
		ctx.Done()
		done <- true
	}
}

func CreateNotes(conn *pgx.Conn, user string, n int) error {
	// This function will make a large request to the server
	// This will cause the server to crash
	batchSize := 100
	loremIpsumGenerator := loremipsum.New()
	// add some hashtags
	content := strings.Replace(loremIpsumGenerator.Paragraph(), " ", " #", 6)
	// create 100 notes SQL statement
	sql := "INSERT INTO public.note (owner, content) VALUES "
	for i := 0; i < batchSize; i++ {
		sql += fmt.Sprintf("('%s', '%s'),", user, content)
	}

	// remove last comma
	sql = sql[:len(sql)-1]

	// execute SQL statement
	for i := 0; i < n; i += batchSize {
		_, err := conn.Exec(context.Background(), sql)
		if err != nil {
			return err
		}
	}

	return nil
}