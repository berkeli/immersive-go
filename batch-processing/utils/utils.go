package utils

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Publisher func([]byte, []byte, []kafka.Header) error

func GetPublisher(k *kafka.Writer, topic string) Publisher {
	return func(key, value []byte, headers []kafka.Header) error {
		err := k.WriteMessages(context.Background(), kafka.Message{
			Topic:   topic,
			Key:     key,
			Value:   value,
			Headers: headers,
		})

		if err != nil {
			return err
		}
		return nil
	}
}

func GetHash(m *kafka.Message) string {
	for _, h := range m.Headers {
		if string(h.Key) == "hash" {
			return string(h.Value)
		}
	}
	return ""
}

func GetExt(m *kafka.Message) string {
	for _, h := range m.Headers {
		if string(h.Key) == "ext" {
			return string(h.Value)
		}
	}
	return ""
}
