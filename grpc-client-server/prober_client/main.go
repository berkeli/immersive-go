// Package main implements a client for Prober service.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/CodeYourFuture/immersive-go-course/grpc-client-server/prober"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr        = flag.String("addr", "localhost:50051", "the address to connect to")
	endpoint    = flag.String("endpoint", "https://www.google.com", "the endpoint to probe")
	nOfRequests = flag.Int("tries", 1, "number of requests to make")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewProberClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	r, err := c.DoProbes(ctx, &pb.ProbeRequest{Endpoint: *endpoint, NumberOfRequests: int32(*nOfRequests)})
	cancel()

	if err != nil {
		log.Fatalf("could not probe: %v", err)
	}

	if int(r.FailedRequests) == *nOfRequests {
		log.Printf("all requests failed")
	} else {
		log.Printf("Response Time: %f", r.AverageResponseTime)
		if r.FailedRequests > 0 {
			log.Printf("Failed Requests: %d", r.FailedRequests)
		}
	}
}
