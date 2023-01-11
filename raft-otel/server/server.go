package server

import (
	"fmt"
	"log"
	"net"
	"os"

	CP "github.com/berkeli/raft-otel/service/consensus"
	SP "github.com/berkeli/raft-otel/service/store"
	"google.golang.org/grpc"
)

type Server struct {
	cs *ConsensusServer
	ss *StorageServer
}

func New() *Server {
	s := &Server{
		cs: NewConsensusServer(),
		ss: NewStorageServer(),
	}

	return s
}

func (s *Server) Run() error {

	port := os.Getenv("PORT")

	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", port))

	if err != nil {
		return err
	}

	s.cs.autodiscovery()

	log.Println("Server listening on port " + port + "...")

	grpcServer := grpc.NewServer()

	CP.RegisterConsensusServiceServer(grpcServer, s.cs)
	SP.RegisterStoreServer(grpcServer, s.ss)

	err = grpcServer.Serve(lis)

	if err != nil {
		return err
	}

	return nil
}
