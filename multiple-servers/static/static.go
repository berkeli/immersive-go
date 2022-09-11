package static

import (
	"fmt"
	"log"
	"net/http"
)

func Run(path string, port int) {

	http.Handle("/", http.FileServer(http.Dir(path)))
	log.Printf("Listening on :%d...", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
