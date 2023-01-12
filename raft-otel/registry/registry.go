package registry

import (
	"fmt"
	"log"
	"net"
	"os"

	RP "github.com/berkeli/raft-otel/service/registry"
	"github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
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

	bsp := honeycomb.NewBaggageSpanProcessor()

	otelShutdown, err := launcher.ConfigureOpenTelemetry(
		launcher.WithSpanProcessor(bsp),
	)
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	RP.RegisterRegistryServer(grpcServer, rs)

	err = grpcServer.Serve(lis)

	if err != nil {
		return err
	}

	return nil
}
