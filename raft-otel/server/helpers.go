package server

import (
	"context"
	"log"
	"os"
	"time"

	CP "github.com/berkeli/raft-otel/service/consensus"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func ConnectToPeer(addr string) *Peer {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	opts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(100 * time.Millisecond)),
		grpc_retry.WithMax(3),
		grpc_retry.WithCodes(codes.Unavailable, codes.Aborted),
	}

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_retry.UnaryClientInterceptor(opts...),
			otelgrpc.UnaryClientInterceptor(),
		)),
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
		Online,
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
