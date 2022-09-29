package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func ValidateIndent(indent string) (int, error) {
	i, err := strconv.Atoi(indent)
	if err != nil {
		log.Printf("Unable to parse indent: %s", err.Error())
		return 0, fmt.Errorf("Unable to parse indent: %s", indent)
	}
	if i < 0 {
		return 0, fmt.Errorf("Indent cannot be negative: %s", indent)
	}
	return i, nil
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
