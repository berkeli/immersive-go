package downloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	CouldNotFetchImage = "received status %d when trying to download image"
)

const (
	ERR_TOPIC = "errors"
)

type Config struct {
	MaxRetries   uint64
	KafkaBrokers []string
	InTopic      string
	OutTopic     string
	Partition    int
	OutputPath   string
}

type DownloadService struct {
	config *Config
}

func NewDownloadService(config *Config) *DownloadService {
	return &DownloadService{
		config: config,
	}
}

func (ds *DownloadService) Run(ctx context.Context) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   ds.config.KafkaBrokers,
		Topic:     ds.config.InTopic,
		GroupID:   "downloaders",
		Partition: ds.config.Partition,
	})
	defer r.Close()

	w, err := kafka.DialContext(ctx, "tcp", ds.config.KafkaBrokers[0])

	if err != nil {
		return err
	}

	defer w.Close()

	w.SetWriteDeadline(time.Now().Add(10 * time.Second))

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("failed to read message: %v", err)
			continue
		}

		filePath := fmt.Sprintf("%s/%s", ds.config.OutputPath, GetMD5Hash(m.Value))

		f, err := os.Create(filePath)

		if err != nil {
			log.Println("Failed to create file for ", string(m.Value), " error: ", err)
			w.WriteMessages(kafka.Message{
				Topic: ERR_TOPIC,
				Key:   m.Key,
				Value: []byte(fmt.Sprintf("Failed to create file, error: %s", err)),
			})
			continue
		}

		hash, ext, err := DownloadWithBackoff(string(m.Value), ds.config.MaxRetries, f)

		if err != nil {
			log.Println("Failed to download ", string(m.Value), " error: ", err)
			w.WriteMessages(kafka.Message{
				Topic: ERR_TOPIC,
				Key:   m.Key,
				Value: []byte(fmt.Sprintf("Failed to download, error: %s", err)),
			})
			continue
		}

		f.Close()

		err = os.Rename(filePath, fmt.Sprintf("%s/%s.%s", ds.config.OutputPath, hash, ext))
		fmt.Println("renamed file to ", fmt.Sprintf("%s/%s.%s", ds.config.OutputPath, hash, ext))

		if err != nil {
			log.Println("Failed to rename file ", string(m.Value), " error: ", err)
			w.WriteMessages(kafka.Message{
				Topic: ds.config.OutTopic,
				Key:   m.Key,
				Value: []byte(fmt.Sprintf("%s.%s", GetMD5Hash(m.Value), ext)),
			})
		} else {
			w.WriteMessages(kafka.Message{
				Topic: ds.config.OutTopic,
				Key:   m.Key,
				Value: []byte(fmt.Sprintf("%s.%s", hash, ext)),
			})
		}
	}
}
