package server

import (
	"fmt"
	"log"
	"net"

	pb "github.com/berkeli/raft-otel/service/consensus"
	"google.golang.org/grpc"
)

type Server struct {
	cs *ConsensusServer
}

func New(id int) *Server {

	return &Server{
		cs: NewConsensusServer(id),
	}
}

func (s *Server) Run() error {

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 50051))

	if err != nil {
		return err
	}

	log.Println("Server listening on port 50051")

	grpcServer := grpc.NewServer()

	pb.RegisterConsensusServiceServer(grpcServer, s.cs)

	err = grpcServer.Serve(lis)

	if err != nil {
		return err
	}

	return nil
}
