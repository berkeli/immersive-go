package main

import (
	"log"
	"net/http"
	"os"
)

func main() {

	port := os.Getenv("HTTP_PORT")

	if port == "" {
		port = "80"
	}

	setupRoutes()

	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}

func setupRoutes() {
	s := Server{}
	http.HandleFunc("/", s.IndexHandler)
	http.HandleFunc("/ping", s.PingHandler)
}
