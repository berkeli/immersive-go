package main

import (
	"context"
	"log"

	cs "github.com/berkeli/immersive-go/batch-processing/services/converter"
)

func main() {
	c := cs.NewConverterService(&cs.Config{
		KafkaBrokers: []string{"localhost:9092"},
		InTopic:      "downloader",
		OutTopic:     "converter",
		OutputPath:   "outputs",
	})

	err := c.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
