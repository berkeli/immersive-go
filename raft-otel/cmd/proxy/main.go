package main

import (
	"log"

	"github.com/berkeli/raft-otel/proxy"
)

func main() {

	p := proxy.New()

	err := p.Run()

	if err != nil {
		log.Fatal(err)
	}
}
