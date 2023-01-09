package main

import (
	"log"

	"github.com/berkeli/raft-otel/client"
)

func main() {
	c := client.New()

	err := c.Run()

	if err != nil {
		log.Fatal(err)
	}
}
