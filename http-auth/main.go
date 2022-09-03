package main

import (
	"http-auth/routes"
	"net/http"
)

func main() {
	http.HandleFunc("/", routes.IndexHandler)

	http.HandleFunc("/200", routes.Handle200)

	http.Handle("/404", http.NotFoundHandler())

	http.HandleFunc("/500", routes.Handle500)

	http.HandleFunc("/authenticated", routes.HandleAuthenticated)

	http.HandleFunc("/limited", routes.HandleRateLimit)

	http.ListenAndServe(":8080", nil)
}
