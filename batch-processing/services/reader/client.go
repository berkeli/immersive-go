package reader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	rs "github.com/berkeli/immersive-go/batch-processing/services/reader/service"
)

// ReaderService is the service that handles the reader requests.
type ReaderClient struct {
	Host      string
	InputFile string
}

func NewReaderClient(host, input string) *ReaderClient {
	return &ReaderClient{
		Host:      host,
		InputFile: input,
	}
}

func (rc *ReaderClient) Run(ctx context.Context) error {
	conn, err := grpc.Dial(rc.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	c := rs.NewReaderClient(conn)

	stream, err := c.ReadAndPublish(ctx)

	if err != nil {
		return err
	}

	f, err := os.Open(rc.InputFile)

	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}

	buf := bufio.NewReader(f)

	for {
		line, err := buf.ReadBytes('\n')

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		err = stream.Send(&rs.ReaderRequest{Csv: line})

		if err != nil {
			log.Fatal(err)
		}
	}

	resp, err := stream.CloseAndRecv()

	if err != nil {
		return err
	}

	log.Printf("sent csv to server: %s\n", resp.GetMessage())

	return nil
}
