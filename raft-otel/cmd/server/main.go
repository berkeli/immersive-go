package main

import (
	"log"

	"github.com/berkeli/raft-otel/server"
)

func main() {

	s := server.New()

	err := s.Run()

	if err != nil {
		log.Fatal(err)
	}
}
