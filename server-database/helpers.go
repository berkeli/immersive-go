package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/jackc/pgx/v4"
)

func ConnectToDB(DB_URL string) *pgx.Conn {
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
