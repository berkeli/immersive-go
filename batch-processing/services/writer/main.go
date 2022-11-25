package writer

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type Config struct {
	KafkaBrokers     []string
	InTopic          string
	OutputFile       string
	FailedOutputFile string
}

type WriterService struct {
	config *Config
}

func NewWriterService(config *Config) *WriterService {
	return &WriterService{
		config: config,
	}
}

func (ws *WriterService) Run(ctx context.Context) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: ws.config.KafkaBrokers,
		Topic:   ws.config.InTopic,
		GroupID: "writers",
	})

	errR := kafka.NewReader(kafka.ReaderConfig{
		Brokers: ws.config.KafkaBrokers,
		Topic:   "errors",
		GroupID: "writers",
	})

	defer r.Close()

	// read messages from kafka and flush to csv every second

	errC := make(chan error)

	go func() {
		for {
			m, err := r.ReadMessage(ctx)
			if err != nil {
				errC <- err
				return
			}

			log.Println(string(m.Value))

			// write to file
		}
	}()

	go func() {
		for {
			m, err := errR.ReadMessage(ctx)
			if err != nil {
				errC <- err
				return
			}

			log.Println(string(m.Value))

			// write to file
		}
	}()

	// go ws.ReadAndWrite(ctx, r, ws.config.OutputFile)
	// go ws.ReadAndWrite(ctx, errR, ws.config.FailedOutputFile)

	return <-errC
}

// func (ws *WriterService) ReadAndWrite(ctx context.Context, r *kafka.Reader, path string, c chan error) {
// 	var rows [][]string
// 	for {
// 		m, err := r.ReadMessage(ctx)
// 		if err != nil {
// 			c <- err
// 		}

// 		// construct rows
// 		row := []string{string(m.Key), string(m.Value)}
// 	}
// }

// func (ws *WriterService) flush(rows [][]string, path string) error {
// 	// flush to csv
// 	os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

// 	return nil
// }
