package main

import (
	"http-auth/routes"
	"net/http"
)

func main() {
	http.HandleFunc("/", routes.IndexHandler)

	http.HandleFunc("/200", routes.Handle200)

	http.HandleFunc("/404", routes.Handle404)

	http.HandleFunc("/500", routes.Handle500)

	http.HandleFunc("/authenticated", routes.HandleAuth)

	http.ListenAndServe(":8080", nil)
}
