package registry

import (
	"fmt"
	"net"
	"os"

	RP "github.com/berkeli/raft-otel/service/registry"
	"google.golang.org/grpc"
)

type Registry struct{}

func New() *Registry {
	return &Registry{}
}

func (r *Registry) Run() error {

	rs := NewRegistryService()

	port := os.Getenv("PORT")

	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", port))

	if err != nil {
		return err
	}

	fmt.Println("Registry listening on port " + port + "...")

	grpcServer := grpc.NewServer()

	RP.RegisterRegistryServer(grpcServer, rs)

	err = grpcServer.Serve(lis)

	if err != nil {
		return err
	}

	return nil
}
