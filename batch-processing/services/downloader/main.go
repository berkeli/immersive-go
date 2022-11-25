package downloader

import (
	"context"
	"fmt"
	"log"
	"os"

	u "github.com/berkeli/immersive-go/batch-processing/utils"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/semaphore"
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

	pub    u.Publisher
	errPub u.Publisher
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

	kafka := &kafka.Writer{
		Addr: kafka.TCP(ds.config.KafkaBrokers...),
	}

	ds.pub = u.GetPublisher(kafka, ds.config.OutTopic)
	ds.errPub = u.GetPublisher(kafka, ERR_TOPIC)

	sem := semaphore.NewWeighted(10)

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("failed to read message: %v", err)
			continue
		}

		sem.Acquire(ctx, 1)

		go ds.processUrl(m.Value, sem)
	}
}

func (ds *DownloadService) processUrl(url []byte, sem *semaphore.Weighted) {
	defer sem.Release(1)
	fmt.Println("Downloading from url: ", string(url))

	inputPath := fmt.Sprintf("%s/%s", ds.config.OutputPath, GetMD5(url))
	file, err := os.Create(inputPath)

	if err != nil {
		log.Printf("failed to create file: %v", err)
		ds.errPub(url, []byte(fmt.Sprintf("failed to create file: %v", err)), nil)
	}

	hash, ext, err := DownloadWithBackoff(string(url), ds.config.MaxRetries, file)

	if err != nil {
		log.Println("Error downloading file: ", err)
		os.Remove(inputPath)
		ds.errPub(url, []byte(fmt.Sprintf("error downloading image: %v", err)), nil)
	}

	os.Rename(inputPath, fmt.Sprintf("%s/%s.%s", ds.config.OutputPath, hash, ext))

	ds.pub(url, []byte(fmt.Sprintf("%s.%s", hash, ext)), nil)
}
