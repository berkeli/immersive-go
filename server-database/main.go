package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

var (
	DB_URL string
	conn   *pgx.Conn
)

type Image struct {
	Title   string
	AltText string
	Url     string
}

func LoadENV() {
	err := godotenv.Load()
	DB_URL = os.Getenv("DB_URL")
	if err != nil || DB_URL == "" {
		os.Stderr.WriteString("Error loading .env file")
		os.Exit(1)
	}
}

func ConnectToDB() {
	LoadENV()
	var err error

	conn, err = pgx.Connect(context.Background(), DB_URL)

	if err != nil {
		log.Fatal("Unable to connect to database: " + err.Error())
	}

}

func main() {
	ConnectToDB()
	defer conn.Close(context.Background())

	http.HandleFunc("/", IndexHandler)

	http.HandleFunc("/images.json", ImagesHandler)

	http.ListenAndServe(":8080", nil)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func ImagesHandler(w http.ResponseWriter, r *http.Request) {

	queryParams := r.URL.Query()
	indent := 1
	if queryParams.Get("indent") != "" {
		indent, _ = strconv.Atoi(queryParams.Get("indent"))
	}

	if r.Method == "GET" {

		images, err := FetchImages(conn)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to fetch images"))
			return
		}

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

}

func FetchImages(conn *pgx.Conn) ([]Image, error) {
	rows, err := conn.Query(context.Background(), "SELECT title, alt_text, url FROM images")
	if err != nil {
		log.Printf("Unable to fetch images: %s", err.Error())
	}

	var images []Image

	for rows.Next() {
		var image Image
		err = rows.Scan(&image.Title, &image.AltText, &image.Url)
		if err != nil {
			log.Printf("Unable to scan row: %v", err)
		}
		images = append(images, image)
	}

	return images, err
}
