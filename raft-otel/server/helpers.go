package server

import (
	"context"
	"log"
	"time"

	CP "github.com/berkeli/raft-otel/service/consensus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ConnectToPeer(addr string) CP.ConsensusServiceClient {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to peer on %s: %v", addr, err)
		return nil
	}

	client := CP.NewConsensusServiceClient(conn)

	return client
}

func Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
