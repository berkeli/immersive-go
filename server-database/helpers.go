package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v4"
)

func ConnectToDB() *pgx.Conn {
	DB_URL := os.Getenv("DB_URL")
	conn, err := pgx.Connect(context.Background(), DB_URL)

	if err != nil {
		log.Fatalf("Unable to connect to database: %s", err.Error())
	}

	return conn
}

func ValidateIndent(indent string) (int, error) {
	i, err := strconv.Atoi(indent)
	if err != nil {
		log.Printf("Unable to parse indent: %s", err.Error())
		return 0, fmt.Errorf("Unable to parse indent: %s", indent)
	}
	if i < 0 {
		return 0, fmt.Errorf("Indent cannot be negative: %s", indent)
	}
	return i, nil
}
