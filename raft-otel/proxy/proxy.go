package proxy

import (
	"context"
	"os"

	SP "github.com/berkeli/raft-otel/service/store"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Proxy is a proxy for a storage server.

type Proxy struct {
	SP.UnimplementedStoreServer                // downstream
	u                           SP.StoreClient // upstream
}

// New creates a new proxy.
func New() *Proxy {
	host := os.Getenv("STORE_SERVER")

	if host == "" {
		panic("STORE_SERVER env variable not set")
	}

	conn, err := grpc.Dial(host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)

	if err != nil {
		panic(err)
	}

	return &Proxy{
		u: SP.NewStoreClient(conn),
	}
}

// Get implements the Get method of the StoreServer interface.
func (p *Proxy) Get(ctx context.Context, req *SP.GetRequest) (*SP.GetResponse, error) {
	r, err := p.u.Get(ctx, req)

	if err != nil {
		return nil, err
	}

	return r, nil
}
