package client

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	SP "github.com/berkeli/raft-otel/service/store"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	sc SP.StoreClient
}

func New() *Client {
	return &Client{}
}

func (c *Client) Run() error {
	storeServer := os.Getenv("STORE_SERVER")

	if storeServer == "" {
		return fmt.Errorf("STORE_SERVER env variable not set")
	}

	conn, err := grpc.Dial(
		storeServer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)

	if err != nil {
		return err
	}

	c.sc = SP.NewStoreClient(conn)

	go c.RunCli()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	<-ch

	return nil
}

func (c *Client) RunCli() {

	PrintHelp()

	for {
		fmt.Print("RAFT CMD: ")
		reader := bufio.NewReader(os.Stdin)

		text, err := reader.ReadString('\n')

		if err != nil {
			fmt.Println("ERR - ", err)
			continue
		}

		cmd := strings.Split(strings.TrimSpace(text), " ")

		switch strings.ToLower(cmd[0]) {
		case "exit":
			return
		case "get":
			if len(cmd) < 2 {
				fmt.Println("ERR - GET command requires key")
				continue
			}
			value, err := c.Get(cmd[1])
			if err != nil {
				fmt.Println("ERR - ", err)
			} else {
				fmt.Printf("GOT - %s : %s", cmd[1], value)
			}
		case "set":
			if len(cmd) < 3 {
				fmt.Println("ERR - SET command requires key and value")
				continue
			}
			err := c.Put(cmd[1], cmd[2])
			if err != nil {
				fmt.Println("ERR - ", err)
			} else {
				fmt.Printf("SET - %s : %s", cmd[1], cmd[2])
			}
		default:
		}
	}
}

func (c *Client) Get(key string) (string, error) {
	r, err := c.sc.Get(context.Background(), &SP.GetRequest{Key: key})

	st := status.Convert(err)

	details := st.Details()

	for _, detail := range details {
		switch info := detail.(type) {
		case *SP.NotLeaderResponse:
			log.Printf("RaftError: %v", info.LeaderAddr)
		default:
			log.Printf("Unexpected type: %T", info)
		}
	}

	// TODO: implement leader redirect
	if err != nil {
		return "", err
	}

	return string(r.Value), nil
}

func (c *Client) Put(key, value string) error {
	_, err := c.sc.Put(context.Background(), &SP.PutRequest{Key: key, Value: []byte(value)})
	return err
}
