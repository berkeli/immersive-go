package routes

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
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

		body := fmt.Sprintf("%s%s", HTMLHeader, b)
		w.Write([]byte(body))
	}
}

func Handle200(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, world"))
}

func Handle404(w http.ResponseWriter, r *http.Request) {
	http.NotFoundHandler()
}

func Handle500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Internal Server Error"))
}

func HandleAuth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 - Unauthorized"))
}
