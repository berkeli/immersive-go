package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	CP "github.com/berkeli/raft-otel/service/consensus"
	SP "github.com/berkeli/raft-otel/service/store"
	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	_, port := GetHostAndPort()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))

	if err != nil {
		return err
	}

	s.cs.autodiscovery()

	otelShutdown, err := launcher.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	CP.RegisterConsensusServiceServer(grpcServer, s.cs)
	SP.RegisterStoreServer(grpcServer, s.ss)

	reflection.Register(grpcServer)

	go func() {
		fmt.Println("server starting on port: ", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	<-ch

	grpcServer.GracefulStop()

	return nil
}
