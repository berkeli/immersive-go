package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"multiple-servers/api/images"
	. "multiple-servers/api/types"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
)

func ConnectToDB(DB_URL string) *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), DB_URL)

	if err != nil {
		log.Fatal("Unable to connect to database: " + err.Error())
	}

	return conn
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func Run(DB_URL string, port int) {
	conn := ConnectToDB(DB_URL)
	defer conn.Close(context.Background())

	http.HandleFunc("/images.json", func(w http.ResponseWriter, r *http.Request) {
		ImagesHandler(w, r, conn)
	})
	http.HandleFunc("/", IndexHandler)

	log.Printf("Listening on :%d...", port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to image server API")
}

func ImagesHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
	log.Println(r.Method, r.URL.EscapedPath())
	enableCors(&w)

	queryParams := r.URL.Query()
	indent := 1
	if queryParams.Get("indent") != "" {
		indent, _ = strconv.Atoi(queryParams.Get("indent"))
	}

	switch r.Method {
	case "GET":
		images, err := images.GetAll(conn)

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
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var image Image
		err := decoder.Decode(&image)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := images.Insert(conn, image)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to insert image"))
			return
		}
		b, err := json.MarshalIndent(res, "", strings.Repeat(" ", indent))
		w.WriteHeader(http.StatusCreated)
		w.Write(b)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)

	}

}
