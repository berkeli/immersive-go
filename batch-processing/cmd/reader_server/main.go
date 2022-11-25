package main

import (
	"context"

	r "github.com/berkeli/immersive-go/batch-processing/services/reader"
)

func main() {
	s := r.NewReaderService(&r.Config{
		KafkaBrokers: []string{"localhost:9092"},
		Topic:        "reader",
		Port:         50051,
	})

	err := s.Run(context.Background())

	if err != nil {
		panic(err)
	}
}
