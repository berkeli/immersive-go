package main

import (
	"flag"
	"log"
	"multiple-servers/api"
	"os"
)

func main() {
	DB_URL := os.Getenv("DATABASE_URL")

	if DB_URL == "" {
		log.Fatal("DATABASE_URL env variable must be set for the API to run")
	}

	port := flag.Int("port", 8081, "Provide the port where api server will listen, default is 8081")
	flag.Parse()
	err := api.Run(api.Config{
		Port:   *port,
		DB_URL: DB_URL,
	})

	if err != nil {
		log.Fatalf("Could not start API: %s", err)
	}
}
