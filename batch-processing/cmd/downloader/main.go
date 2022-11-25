package main

import (
	"context"
	"flag"
	"log"

	ds "github.com/berkeli/immersive-go/batch-processing/services/downloader"
)

func main() {
	maxRetries := flag.Int("max-retries", 3, "The maximum number of times to retry a failed download with exponential backoff. Default if 3.")
	flag.Parse()

	ds := ds.NewDownloadService(&ds.Config{
		MaxRetries:   uint64(*maxRetries),
		KafkaBrokers: []string{"localhost:9092"},
		InTopic:      "reader",
		OutTopic:     "downloader",
		Partition:    0,
		OutputPath:   "outputs",
	})

	err := ds.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
