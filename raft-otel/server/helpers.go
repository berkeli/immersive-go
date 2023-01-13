package server

import (
	"context"
	"log"
	"os"
	"time"

	CP "github.com/berkeli/raft-otel/service/consensus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ConnectToPeer(addr string) *Peer {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		log.Printf("Failed to connect to peer on %s: %v", addr, err)
		return nil
	}

	client := CP.NewConsensusServiceClient(conn)
	return &Peer{
		client,
		addr,
	}
}

func Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func GetHostAndPort() (string, string) {
	host, port := os.Getenv("HOST_URL"), os.Getenv("PORT")

	if host == "" {
		host = "localhost"
	}

	if port == "" {
		port = "50051"
	}

	return host, port
}
