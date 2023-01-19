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
	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var tracer = otel.Tracer(os.Getenv("OTEL_SERVICE_NAME"))

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

	otelShutdown, err := launcher.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	conn, err := grpc.DialContext(
		context.Background(),
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
				fmt.Printf("%s : %s\n", cmd[1], value)
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
				fmt.Printf("%s = %s\n", cmd[1], cmd[2])
			}
		case "help":
			PrintHelp()
		case "setif":
			if len(cmd) < 4 {
				fmt.Println("ERR - SETIF command requires key, value and previous value")
				continue
			}

			err := c.SetIf(cmd[1], cmd[2], cmd[3])

			if err != nil {
				fmt.Println("ERR - ", err)
			} else {
				fmt.Printf("%s = %s\n", cmd[1], cmd[2])
			}
		default:
		}
	}
}

func (c *Client) Get(key string) (string, error) {
	ctx, span := tracer.Start(context.Background(), "GET")
	defer span.End()
	r, err := c.sc.Get(ctx, &SP.GetRequest{Key: key})

	if err != nil {
		return "", err
	}

	return string(r.Value), nil
}

func (c *Client) SetIf(key, value, prevValue string) error {
	ctx, span := tracer.Start(context.Background(), "CompareAndSet")
	defer span.End()
	_, err := c.sc.CompareAndSet(ctx, &SP.CompareAndSetRequest{Key: key, Value: []byte(value), PrevValue: []byte(prevValue)})
	return err
}

func (c *Client) Put(key, value string) error {
	_, err := c.sc.Put(context.Background(), &SP.PutRequest{Key: key, Value: []byte(value)})
	return err
}
