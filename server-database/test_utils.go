package main

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/jackc/pgx/v4"
)

const TEST_DB_URL = "postgresql://postgres:postgres@localhost:5432/go-server-database-test"

var TestDbData = []Image{
	{
		Title:   "A cute kitten",
		AltText: "A kitten looking mischievous",
		Url:     "https://placekitten.com/200/300",
		Width:   200,
		Height:  300,
	},
	{
		Title:   "A cute puppy",
		AltText: "A puppy looking mischievous",
		Url:     "https://placedog.net/200/300",
		Width:   200,
		Height:  300,
	},
}

func setupSuite(tb testing.TB) (*pgx.Conn, func(tb testing.TB)) {

	conn, err := pgx.Connect(context.Background(), TEST_DB_URL)

	if err != nil {
		tb.Fatalf("Unable to connect to database: %s", err.Error())
	}

	queries, ioErr := ioutil.ReadFile("migrations.sql")

	if ioErr != nil {
		tb.Fatalf("Unable to read migrations: %s", ioErr.Error())
	}

	_, err = conn.Exec(context.Background(), string(queries))

	if err != nil {
		tb.Fatalf("Unable to run migrations: %s", err.Error())
	}

	for _, image := range TestDbData {
		_, err = conn.Exec(context.Background(), "INSERT INTO images (title, alt_text, url, width, height) VALUES ($1, $2, $3, $4, $5)", image.Title, image.AltText, image.Url, image.Width, image.Height)

		if err != nil {
			tb.Fatalf("Unable to insert data: %s", err.Error())
		}
	}

	return conn, func(tb testing.TB) {
		// teardown the database after testing
		_, err := conn.Exec(context.Background(), "DROP TABLE images")

		if err != nil {
			tb.Fatalf("Teardown Error: Unable to delete table images: %s", err.Error())
		}
	}
}
