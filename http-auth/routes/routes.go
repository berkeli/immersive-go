package routes

import (
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

const HTMLHeader = "<!DOCTYPE html><html>"

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		b := "<em>Hello World</em>"

		queryParams := r.URL.Query()

		if len(queryParams) > 0 {
			b = "<h1>Query Params</h1><ul>"
			for key, value := range queryParams {
				b += fmt.Sprintf("<li>%s: %s</li>", key, html.EscapeString(strings.Join(value, ", ")))
			}
			b += "</ul>"
		}

		body := fmt.Sprintf("%s%s", HTMLHeader, b)
		w.Write([]byte(body))
	}

	if r.Method == "POST" {
		w.Header().Set("Content-Type", "text/html")

		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error reading request body"))
		}

		if b == nil {
			b = []byte("<em>Hello World</em>")
		}

		body := fmt.Sprintf("%s%s", HTMLHeader, b)
		w.Write([]byte(body))
	}
}

func Handle200(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, world"))
}

func Handle500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Internal Server Error"))
}

func HandleAuthenticated(w http.ResponseWriter, r *http.Request) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")

	user, pass, ok := r.BasicAuth()

	if !ok || user != username || pass != password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hello, %s", username)))

}

var limiter = rate.NewLimiter(100, 30)

func HandleRateLimit(w http.ResponseWriter, r *http.Request) {
	if !limiter.Allow() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, world"))
	} else {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("429 - Too Many Requests"))
	}
}
