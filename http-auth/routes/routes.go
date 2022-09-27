package routes

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"golang.org/x/time/rate"
)

const HTMLHeader = `<!DOCTYPE html>
<html>
`

type Controllers struct {
	Username string
	Password string
	Limiter  *rate.Limiter
}

func (c *Controllers) IndexHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		b := "<em>Hello World</em>"

		queryParams := r.URL.Query()

		if len(queryParams) > 0 {
			b = `<h1>Query Params</h1>
			<ul>
			`
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
			return
		}

		if b == nil {
			returnMessage = "<em>Hello World</em>"
		} else {
			returnMessage = html.EscapeString(string(b))
		}

		body := fmt.Sprintf("%s%s", HTMLHeader, returnMessage)
		w.Write([]byte(body))
	}
}

func (c *Controllers) Handle200(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, world"))
}

func (c *Controllers) Handle500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Internal Server Error"))
}

func (c *Controllers) HandleAuthenticated(w http.ResponseWriter, r *http.Request) {

	gotUsername, gotPassword, ok := r.BasicAuth()

	if !ok || gotUsername != c.Username || gotPassword != c.Password {
		w.Header().Set("WWW-Authenticate", `Basic realm="protected", charset="UTF-8"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hello, %s", gotUsername)))
}

func (c *Controllers) HandleRateLimit(w http.ResponseWriter, r *http.Request) {
	if c.Limiter.Allow() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, world"))
	} else {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("429 - Too Many Requests"))
	}
}
