package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	SP "github.com/berkeli/raft-otel/service/store"
	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
)

// Proxy is a proxy for a storage server.

type Proxy struct {
	peers                       []string
	u                           SP.StoreClient // upstream
	SP.UnimplementedStoreServer                // downstream
}

// New creates a new proxy.
func New() *Proxy {
	peers, err := readConfig("/servers.yml")

	if err != nil {
		panic(err)
	}

	conn, err := grpc.DialContext(
		context.Background(),
		peers[0],
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)

	if err != nil {
		panic(err)
	}

	return &Proxy{
		u:     SP.NewStoreClient(conn),
		peers: peers,
	}
}

// Get implements the Get method of the StoreServer interface.
func (p *Proxy) Get(ctx context.Context, req *SP.GetRequest) (*SP.GetResponse, error) {
	r, err := p.u.Get(ctx, req)

	if err != nil {
		ok, err := p.notLeaderStatus(ctx, err)

		if ok {
			if err != nil {
				log.Println("failed to connect to new leader", err)
			}

			return p.u.Get(ctx, req)
		}
	}

	return r, nil
}

// Put implements the Put method of the StoreServer interface.
func (p *Proxy) Put(ctx context.Context, req *SP.PutRequest) (*SP.PutResponse, error) {
	r, err := p.u.Put(ctx, req)

	if err != nil {
		ok, err := p.notLeaderStatus(ctx, err)

		if ok {
			if err != nil {
				log.Println("failed to connect to new leader", err)
			}

			return p.u.Put(ctx, req)
		}
	}

	return r, nil
}

// CompareAndSet implements the CompareAndSet method of the StoreServer interface.
func (p *Proxy) CompareAndSet(ctx context.Context, req *SP.CompareAndSetRequest) (*SP.CompareAndSetResponse, error) {
	r, err := p.u.CompareAndSet(ctx, req)

	if err != nil {
		ok, err := p.notLeaderStatus(ctx, err)

		if ok {
			if err != nil {
				log.Println("failed to connect to new leader", err)
			}

			return p.u.CompareAndSet(ctx, req)
		}
	}

	return r, nil
}

// Run starts the proxy.
func (p *Proxy) Run() error {
	lis, err := net.Listen("tcp", ":8080")

	if err != nil {
		return err
	}

	otelShutdown, err := launcher.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	SP.RegisterStoreServer(s, p)

	return s.Serve(lis)
}

func readConfig(path string) ([]string, error) {
	configPath := os.Getenv("SERVERS_CONFIG")

	if configPath == "" {
		log.Println("SERVERS_CONFIG env var not set, using default path '/servers.yml'")
		configPath = "/servers.yml"
	}

	config, err := os.ReadFile(configPath)

	if err != nil {
		return nil, fmt.Errorf("error while reading config file: %s", err)
	}

	var servers []string

	err = yaml.Unmarshal(config, &servers)

	if err != nil {
		return nil, fmt.Errorf("error while parsing config file: %s", err)
	}

	return servers, nil
}

func (p *Proxy) notLeaderStatus(ctx context.Context, err error) (bool, error) {
	st, ok := status.FromError(err)
	if !ok {
		return false, nil
	}

	for _, detail := range st.Details() {
		switch t := detail.(type) {
		case *SP.NotLeaderResponse:
			fmt.Println("Oops! This is not the leader!")
			fmt.Println("Redirecting to leader:", t.LeaderAddr)
			client, err := grpc.DialContext(
				ctx,
				t.LeaderAddr,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
				grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
			)

			if err != nil {
				log.Println("Error connecting to leader:", err)
				return true, err
			}
			p.u = SP.NewStoreClient(client)
			return true, nil
		}
	}

	return false, nil
}
