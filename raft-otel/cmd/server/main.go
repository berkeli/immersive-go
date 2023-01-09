package main

import (
	"log"

	"github.com/berkeli/raft-otel/server"
)

func main() {
	s := server.New(1)

	err := s.Run()

	if err != nil {
		log.Fatal(err)
	}
}
