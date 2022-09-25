package main

import (
	"http-auth/routes"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	wantUsername := os.Getenv("USERNAME")
	wantPassword := os.Getenv("PASSWORD")
	limiter := rate.NewLimiter(100, 30)

	http.HandleFunc("/", routes.IndexHandler(io.ReadAll))

	http.HandleFunc("/200", routes.Handle200)

	http.Handle("/404", http.NotFoundHandler())

	http.HandleFunc("/500", routes.Handle500)

	http.HandleFunc("/authenticated", routes.HandleAuthenticated(wantUsername, wantPassword))

	http.HandleFunc("/limited", routes.HandleRateLimit(limiter))

	err := http.ListenAndServe(":8090", nil)

	if err != nil {
		panic(err)
	}
}
