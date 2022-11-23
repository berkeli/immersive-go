package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	InvalidCSVFormat = "csv file must have a header row with 'url' as the first column"
	EmptyCSV         = "provided CSV file appears to be empty"
)

func main() {
	kafkaHost := os.Getenv("KAFKA_BROKER")

	if kafkaHost == "" {
		log.Println("KAFKA_BROKER not set, using default - localhost:9092")
		kafkaHost = "localhost:9092"
	}

	inputFilepath := flag.String("input", "", "A path to a CSV file containing image URLs to be processed")
	flag.Parse()

	if *inputFilepath == "" {
		flag.Usage()
		os.Exit(1)
	}

	csvReader, err := OpenCSVFile(*inputFilepath)

	if err != nil {
		log.Fatal(err)
	}

	conn, err := kafka.DialLeader(context.Background(), "tcp", kafkaHost, "reader", 0)
	if err != nil {
		log.Fatal("failed to dial leader:", err)
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Write the CSV file to Kafka

	err = csvToKafka(csvReader, conn)

	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}

func csvToKafka(r *csv.Reader, conn *kafka.Conn) error {
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

		_, err = conn.WriteMessages(
			kafka.Message{
				Key:   []byte(record[0]),
				Value: []byte(record[0]),
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}

/**
* Function opens the csv file, validates headers and returns the reader.
* @param: {string} filename - the path of the CSV file
* @return: {csv.Reader} csvReader - the CSV reader
* @return: {error} err - any error that occurred
 */
func OpenCSVFile(filename string) (*csv.Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = 1

	header, err := csvReader.Read()

	if err == io.EOF {
		return nil, errors.New(EmptyCSV)
	}

	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %v", err)
	}

	if strings.ToLower(header[0]) != "url" {
		return nil, errors.New(InvalidCSVFormat)
	}

	return csvReader, nil
}
