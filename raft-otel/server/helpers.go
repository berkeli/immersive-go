package server

import (
	"context"
	"fmt"
	"log"
	"time"

	CP "github.com/berkeli/raft-otel/service/consensus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ConnectToPeer(id int64) CP.ConsensusServiceClient {
	host := fmt.Sprintf("server_%d:%d", 1, 50051)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to peer %d on %s: %v", id, host, err)
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
