package main

import (
	"http-auth/routes"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file, please provide one with Username and Password set")
	}

	c := routes.Controllers{
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
		Limiter:  rate.NewLimiter(100, 30),
	}

	http.HandleFunc("/", c.IndexHandler)

	http.HandleFunc("/200", c.Handle200)

	http.Handle("/404", http.NotFoundHandler())

	http.HandleFunc("/500", c.Handle500)

	http.HandleFunc("/authenticated", c.HandleAuthenticated)

	http.HandleFunc("/limited", c.HandleRateLimit)

	err = http.ListenAndServe(":8090", nil)

	if err != nil {
		panic(err)
	}
}
