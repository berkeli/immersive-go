package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
)

type MockProbeServer struct {
	pb.UnimplementedProberServer
}

func (*MockProbeServer) DoProbes(ctx context.Context, req *pb.ProbeRequest) (*pb.ProbeReply, error) {
	if req.Endpoint == "http://musterror:8080" {
		return nil, grpc.Errorf(15, "error")
	}
	return &pb.ProbeReply{
		TtfbAverageResponseTime: durationpb.New(123123 * time.Microsecond),
		TtlbAverageResponseTime: durationpb.New(223123 * time.Microsecond),
		FailedRequests:          0,
	}, nil
}

func MockDialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	server := grpc.NewServer()

	pb.RegisterProberServer(server, &MockProbeServer{})

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}
