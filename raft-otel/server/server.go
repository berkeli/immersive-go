package server

import (
	"fmt"
	"log"
	"net"

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

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 50051))

	if err != nil {
		return err
	}

	s.cs.autodiscovery()

	log.Println("Server listening on port 50051")

	grpcServer := grpc.NewServer()

	CP.RegisterConsensusServiceServer(grpcServer, s.cs)
	SP.RegisterStoreServer(grpcServer, s.ss)

	err = grpcServer.Serve(lis)

	if err != nil {
		return err
	}

	return nil
}
