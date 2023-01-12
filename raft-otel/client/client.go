package client

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	SP "github.com/berkeli/raft-otel/service/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	conn, err := grpc.Dial(storeServer, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return err
	}

	c.sc = SP.NewStoreClient(conn)
	return nil
}

func (c *Client) RunCli() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter command: ")

		text, _ := reader.ReadString('\n')

		cmd := strings.Split(text, " ")

		switch strings.ToLower(cmd[0]) {
		case "exit":
			return
		case "get":
			value, err := c.Get(cmd[1])
			if err != nil {
				fmt.Println("ERR - ", err)
			} else {
				fmt.Printf("GOT - %s : %s", cmd[1], value)
			}
		case "set":
			err := c.Put(cmd[1], cmd[2])
			if err != nil {
				fmt.Println("ERR - ", err)
			} else {
				fmt.Printf("SET - %s : %s", cmd[1], cmd[2])
			}
		default:
			fmt.Println("Unknown command")
			PrintHelp()
		}
	}
}

func (c *Client) Get(key string) (string, error) {
	r, err := c.sc.Get(context.Background(), &SP.GetRequest{Key: key})

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
