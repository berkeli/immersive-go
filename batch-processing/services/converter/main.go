package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/segmentio/kafka-go"
)

func main() {
	kafkaHost := os.Getenv("KAFKA_BROKER")

	if kafkaHost == "" {
		log.Println("KAFKA_BROKER not set, using default - localhost:9092")
		kafkaHost = "localhost:9092"
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaHost},
		Topic:   "reader",
		GroupID: "converters",
	})

	for {
		msg, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Println(string(msg.Value))
	}

	r.Close()
}
