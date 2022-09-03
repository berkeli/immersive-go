package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

type Image struct {
	Title   string
	AltText string
	Url     string
}

func main() {
	err := godotenv.Load()
	DB_URL := os.Getenv("DB_URL")
	if err != nil || DB_URL == "" {
		os.Stderr.WriteString("Error loading .env file")
		os.Exit(1)
	}

	conn, err := pgx.Connect(context.Background(), DB_URL)
	defer conn.Close(context.Background())

	if err != nil {
		os.Stderr.WriteString("Unable to connect to database: " + err.Error())
		os.Exit(1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	http.HandleFunc("/images.json", func(w http.ResponseWriter, r *http.Request) {

		queryParams := r.URL.Query()
		indent := 1
		if queryParams.Get("indent") != "" {
			indent, _ = strconv.Atoi(queryParams.Get("indent"))
		}

		if r.Method == "GET" {

			images, _ := FetchImages(conn)

			b, err := json.MarshalIndent(images, "", strings.Repeat(" ", indent))

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		} else if r.Method == "POST" {
			decoder := json.NewDecoder(r.Body)
			var image Image
			err := decoder.Decode(&image)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			_, err = conn.Exec(context.Background(), "INSERT INTO images (title, alt_text, url) VALUES ($1, $2, $3)", image.Title, image.AltText, image.Url)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Unable to insert image: " + err.Error()))
				return
			}
			b, err := json.MarshalIndent(image, "", strings.Repeat(" ", indent))
			w.WriteHeader(http.StatusCreated)
			w.Write(b)
		}

	})

	http.ListenAndServe(":8080", nil)
}

func FetchImages(conn *pgx.Conn) ([]Image, error) {
	rows, err := conn.Query(context.Background(), "SELECT title, alt_text, url FROM images")

	var images []Image

	for rows.Next() {
		var image Image
		err = rows.Scan(&image.Title, &image.AltText, &image.Url)
		if err != nil {
			os.Stderr.WriteString("Unable to scan row: " + err.Error())
			os.Exit(1)
		}
		images = append(images, image)
	}

	if err != nil {
		os.Stderr.WriteString("Unable to query database: " + err.Error())
		os.Exit(1)
	}

	return images, err
}
