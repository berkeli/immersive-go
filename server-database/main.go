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
	Width   int
	Height  int
}

type Server struct {
	images *Images
}

func main() {
	err := godotenv.Load()
	DB_URL := os.Getenv("DB_URL")
	if err != nil || DB_URL == "" {
		log.Fatal("Error loading .env file, please create one with 'DB_URL' set to your database connection string")
	}

	conn, err := pgx.Connect(context.Background(), DB_URL)

	if err != nil {
		log.Fatalf("Unable to connect to database: %s", err.Error())
	}
	defer conn.Close(context.Background())

	s := &Server{
		images: &Images{
			conn: conn,
		},
	}

	http.HandleFunc("/", s.IndexHandler)

	http.HandleFunc("/images.json", s.ImagesHandler)

	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Fatalf("Unable to start server: %s", err.Error())
	}
}

func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
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

	switch r.Method {
	case "GET":
		images, err := s.images.GetAll()

		if err != nil {
			log.Print("Error fetching images: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to fetch images"))
			return
		}

		b, err := json.MarshalIndent(images, "", strings.Repeat(" ", indent))

		if err != nil {
			log.Print("Error serializing json", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var image Image
		err := decoder.Decode(&image)
		if err != nil {
			log.Print("Error parsing json", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = ValidateAltText(image.Title, image.AltText)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		width, height, err := ValidateImage(image.Url)

		if err != nil {
			log.Print("Error validating image: ", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		image.Width = width
		image.Height = height

		err = s.images.InsertOne(image)
		if err != nil {
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
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
