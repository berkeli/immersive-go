package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"multiple-servers/api/images"
	. "multiple-servers/api/types"
	"net/http"

	"github.com/jackc/pgx/v4"
)

type Config struct {
	Port   int    `json:"path"`
	DB_URL string `json:"port"`
}

type Server struct {
	db *pgx.Conn
}

func Run(c Config) error {
	conn, err := pgx.Connect(context.Background(), c.DB_URL)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	s := Server{db: conn}

	http.HandleFunc("/", s.IndexHandler)
	http.HandleFunc("/images.json", s.ImagesHandler)

	err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.EscapedPath())
	enableCors(&w)

	switch r.Method {
	case "GET":
		w.Write([]byte("Welcome to the API"))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) ImagesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.EscapedPath())
	enableCors(&w)

	indent := 1
	queryParams := r.URL.Query()
	if queryParams.Get("indent") != "" {
		i, err := ValidateIndent(queryParams.Get("indent"))
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		indent = i
	}

	switch r.Method {
	case "GET":
		images, err := images.GetAll(s.db)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to fetch images"))
			return
		}

		if images == nil {
			images = []Image{}
		}

		serialized, err := serializeIndented(images, indent)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to serialize json"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(serialized)
	case "POST":
		fmt.Println(r.Body)
		decoder := json.NewDecoder(r.Body)
		var image Image
		err := decoder.Decode(&image)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Unable to parse json"))
			return
		}

		err = images.InsertOne(s.db, image)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to insert image"))
			return
		}

		w.WriteHeader(http.StatusCreated)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}
