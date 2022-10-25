package main

import (
	"context"
	"log"
	"net"
	"testing"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	server := grpc.NewServer()

	pb.RegisterProberServer(server, &Server{})

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestDoProbes(t *testing.T) {
	tests := map[string]struct {
		req       *pb.ProbeRequest
		err       error
		failedReq int32
	}{
		"simple": {
			req: &pb.ProbeRequest{
				Endpoint:         "http://google.com",
				NumberOfRequests: 1,
			},
			failedReq: 0,
		},
		"error": {
			req: &pb.ProbeRequest{
				Endpoint:         "http://musterror:8080",
				NumberOfRequests: 3,
			},
			failedReq: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer()), grpc.WithInsecure())
			require.NoError(t, err)
			defer conn.Close()

			client := pb.NewProberClient(conn)

			resp, err := client.DoProbes(context.Background(), tc.req)

			if err == nil {
				require.NotNil(t, resp.AverageResponseTime)
			}

			require.Equal(t, tc.failedReq, resp.FailedRequests)
		})
	}
}
