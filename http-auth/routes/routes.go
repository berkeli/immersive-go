package routes

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/time/rate"
)

const HTMLHeader = "<!DOCTYPE html><html>"

func IsTestRun() bool {
	return os.Getenv("EXECUTION_ENVIRONMENT") == "test"
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		b := "<em>Hello World</em>"

		queryParams := r.URL.Query()

		if len(queryParams) > 0 {
			b = "<h1>Query Params</h1><ul>"
			for key, value := range queryParams {
				b += fmt.Sprintf("<li>%s: %s</li>", html.EscapeString(key), html.EscapeString(strings.Join(value, ", ")))
			}
			b += "</ul>"
		}

		body := fmt.Sprintf("%s%s", HTMLHeader, b)
		w.Write([]byte(body))
	}

	if r.Method == "POST" {
		w.Header().Set("Content-Type", "text/html")
		returnMessage := ""

		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error reading request body"))
		}

		if b == nil {
			returnMessage = html.EscapeString("<em>Hello World</em>")
		} else {
			returnMessage = html.EscapeString(string(b))
		}

		body := fmt.Sprintf("%s%s", HTMLHeader, returnMessage)
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

func HandleAuthenticated(wantUsername, wantPassword string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		gotUsername, gotPassword, ok := r.BasicAuth()

		if !ok || gotUsername != wantUsername || gotPassword != wantPassword {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("401 - Unauthorized"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Hello, %s", gotUsername)))
	}

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
