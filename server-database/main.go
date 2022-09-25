package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

type Image struct {
	Title   string
	AltText string
	Url     string
}

type Server struct {
	conn *pgx.Conn
}

func init() {
	err := godotenv.Load()
	DB_URL := os.Getenv("DB_URL")
	if err != nil || DB_URL == "" {
		os.Stderr.WriteString("Error loading .env file, please create one with 'DB_URL' set to your database connection string")
		os.Exit(1)
	}
}

func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func main() {
	conn := ConnectToDB()
	defer conn.Close(context.Background())

	s := &Server{conn: conn}

	http.HandleFunc("/", s.IndexHandler)

	http.HandleFunc("/images.json", s.ImagesHandler)

	http.ListenAndServe(":8080", nil)
}

func (s *Server) ImagesHandler(w http.ResponseWriter, r *http.Request) {
	indent := 1
	queryParams := r.URL.Query()

	if queryParams.Get("indent") != "" {
		i, err := ValidateIndent(queryParams.Get("indent"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		indent = i
	}

	if r.Method == "GET" {

		images, err := FetchImages(s.conn)

		if err != nil {
			log.Print("Error fetching images: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to fetch images"))
			return
		}

		b, err := json.MarshalIndent(images, "", strings.Repeat(" ", indent))

		if err != nil {
			log.Print("Error Parsing json", err)
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
			log.Print("Error parsing json", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = s.conn.Exec(context.Background(), "INSERT INTO images (title, alt_text, url) VALUES ($1, $2, $3)", image.Title, image.AltText, image.Url)
		if err != nil {
			log.Print("Error inserting image", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to insert image: " + err.Error()))
			return
		}
		b, err := json.MarshalIndent(image, "", strings.Repeat(" ", indent))
		if err != nil {
			log.Print("Error formatting json", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(b)
	}

}

func FetchImages(conn *pgx.Conn) ([]Image, error) {
	rows, err := conn.Query(context.Background(), "SELECT title, alt_text, url FROM images")
	if err != nil {
		log.Printf("Unable to fetch images: %s", err.Error())
		return nil, err
	}

	var images []Image

	for rows.Next() {
		var image Image
		err = rows.Scan(&image.Title, &image.AltText, &image.Url)
		if err != nil {
			log.Printf("Unable to scan row: %v", err)
			return nil, err
		}
		images = append(images, image)
	}

	return images, nil
}
