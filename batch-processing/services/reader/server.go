package reader

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	rs "github.com/berkeli/immersive-go/batch-processing/services/reader/service"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
)

const (
	ERR_TOPIC = "errors"
)

type Config struct {
	KafkaBrokers []string
	Topic        string
	Port         int
}

type ReaderService struct {
	config *Config
}

func NewReaderService(config *Config) *ReaderService {
	return &ReaderService{
		config: config,
	}
}

func (s *ReaderService) Run(ctx context.Context) error {
	w := &kafka.Writer{
		Addr: kafka.TCP(s.config.KafkaBrokers...),
	}

	listen := fmt.Sprintf(":%d", s.config.Port)

	lis, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()

	rService := NewGRPCReaderService(w, s.config.Topic)

	rs.RegisterReaderServer(grpcServer, rService)

	var runErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = grpcServer.Serve(lis)
	}()

	<-ctx.Done()
	wg.Wait()
	return runErr
}

type grpcReaderService struct {
	rs.UnimplementedReaderServer

	conn  *kafka.Writer
	topic string
}

func NewGRPCReaderService(conn *kafka.Writer, topic string) *grpcReaderService {
	return &grpcReaderService{
		conn:  conn,
		topic: topic,
	}
}

func (r *grpcReaderService) ReadAndPublish(stream rs.Reader_ReadAndPublishServer) error {
	var csvFile []byte
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		csvFile = append(csvFile, req.Csv...)
	}

	rdr := csv.NewReader(bytes.NewReader(csvFile))

	err := csvToKafka(rdr, r.conn, r.topic)

	if err != nil {
		return err
	}

	return stream.SendAndClose(&rs.ReaderResponse{
		Message: "success",
	})
}

func csvToKafka(r *csv.Reader, conn *kafka.Writer, topic string) error {
	hash := &sync.Map{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("error reading row: ", err)
		}

		// check duplicates
		if _, ok := hash.LoadOrStore(record[0], true); ok {
			continue
		}
		err = conn.WriteMessages(context.Background(), kafka.Message{
			Topic: topic,
			Key:   []byte(record[0]),
			Value: []byte(record[0]),
		})

		if err != nil {
			log.Println("error writing to kafka: ", err)
			conn.WriteMessages(context.Background(), kafka.Message{
				Topic: topic,
				Key:   []byte(record[0]),
				Value: []byte(record[0]),
			})
		}
	}

	return nil
}
