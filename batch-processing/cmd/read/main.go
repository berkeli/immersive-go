package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/berkeli/immersive-go/batch-processing/services/reader"
)

func main() {
	inputFilepath := flag.String("input", "", "A path to a CSV file containing image URLs to be processed")
	addr := flag.String("addr", "localhost:50051", "the address to connect to")
	flag.Parse()

	if *inputFilepath == "" {
		flag.Usage()
		os.Exit(1)
	}

	rc := reader.NewReaderClient(*addr, *inputFilepath)

	err := rc.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
