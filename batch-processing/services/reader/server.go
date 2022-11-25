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
	u "github.com/berkeli/immersive-go/batch-processing/utils"

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
	kafka := &kafka.Writer{
		Addr:                   kafka.TCP(s.config.KafkaBrokers...),
		AllowAutoTopicCreation: true,
	}

	listen := fmt.Sprintf(":%d", s.config.Port)

	lis, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()

	rService := NewGRPCReaderService(kafka, s.config.Topic)

	rs.RegisterReaderServer(grpcServer, rService)

	var runErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = grpcServer.Serve(lis)
	}()

	wg.Wait()
	return runErr
}

type grpcReaderService struct {
	rs.UnimplementedReaderServer

	writer    u.Publisher
	errWriter u.Publisher
}

func NewGRPCReaderService(kafka *kafka.Writer, topic string) *grpcReaderService {
	return &grpcReaderService{
		writer:    u.GetPublisher(kafka, topic),
		errWriter: u.GetPublisher(kafka, ERR_TOPIC),
	}
}

func (r *grpcReaderService) ReadAndPublish(stream rs.Reader_ReadAndPublishServer) error {
	log.Println("starting to read and publish")
	var csvFile []byte
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		csvFile = append(csvFile, req.GetCsv()...)
	}

	log.Println("received csv, publishing...")

	rdr := csv.NewReader(bytes.NewBuffer(csvFile))

	rowOne, err := rdr.Read()

	if err != nil || rowOne[0] != "url" {
		log.Println("error reading first row")
	}

	err = csvToKafka(rdr, r.writer, r.errWriter)

	log.Println(err)

	if err != nil {
		return err
	}

	return stream.SendAndClose(&rs.ReaderResponse{
		Message: "success",
	})
}

func csvToKafka(r *csv.Reader, pub, pubErr u.Publisher) error {
	hash := &sync.Map{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}

		log.Println("read", record)

		if err != nil {
			log.Println("error reading row: ", err)
		}

		// check duplicates
		if _, ok := hash.LoadOrStore(record[0], true); ok {
			continue
		}
		err = pub([]byte(record[0]), []byte(record[0]), nil)

		if err != nil {
			log.Println("error publishing to kafka: ", err)
			return err
		}
	}

	return nil
}
