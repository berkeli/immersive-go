package main

import (
	"context"
	"log"

	us "github.com/berkeli/immersive-go/batch-processing/services/uploader"
)

func main() {
	u := us.NewUploaderService(&us.Config{
		KafkaBrokers: []string{"localhost:9092"},
		InTopic:      "converter",
		OutTopic:     "uploader",
		OutputPath:   "outputs",
	})

	err := u.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
