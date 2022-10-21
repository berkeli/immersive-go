package main

import (
	"flag"
	"log"
	"multiple-servers/static"
)

func main() {
	path := flag.String("path", "assets", "Provide path to the static files, default is assets")
	port := flag.Int("port", 8082, "Provide port where static server will listen, default is 8082")
	flag.Parse()

	err := static.Run(static.Config{
		Path: *path,
		Port: *port,
	})
	if err != nil {
		log.Fatal(err)
	}
}
