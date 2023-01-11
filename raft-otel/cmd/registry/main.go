package main

import (
	"log"

	"github.com/berkeli/raft-otel/registry"
)

func main() {

	s := registry.New()

	err := s.Run()

	if err != nil {
		log.Fatal(err)
	}
}
