package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
				Endpoint:         "https://www.google.com",
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
			conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer()), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()

			client := pb.NewProberClient(conn)

			resp, err := client.DoProbes(context.Background(), tc.req)

			if err == nil {
				require.NotNil(t, resp.TtfbAverageResponseTime)
			}

			require.Equal(t, tc.failedReq, resp.FailedRequests)
		})
	}

	t.Run("response time", func(t *testing.T) {
		conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer()), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := pb.NewProberClient(conn)

		timeSince = func(t time.Time) time.Duration {
			return 2 * time.Second
		}

		resp, err := client.DoProbes(context.Background(), &pb.ProbeRequest{
			Endpoint:         "https://www.google.com",
			NumberOfRequests: 1,
		})

		require.NoError(t, err)
		require.NotNil(t, resp.TtfbAverageResponseTime)
		require.NotNil(t, resp.TtlbAverageResponseTime)
		require.Greater(t, resp.TtfbAverageResponseTime.AsDuration(), time.Duration(0))
		require.Equal(t, 2*time.Second, resp.TtlbAverageResponseTime.AsDuration())
	})
}

func TestTimedProbe(t *testing.T) {
	t.Run("must return valid values", func(t *testing.T) {
		ttfbTotal := time.Duration(0)
		latestTtfb := time.Duration(0)

		timeSince = func(t time.Time) time.Duration {
			return 3 * time.Second
		}

		c := &http.Client{
			Transport: &TimedRoundTripper{
				defaultTripper: http.DefaultTransport,
				recordTime: func(t time.Duration) {
					latestTtfb = 200 * time.Millisecond
					ttfbTotal += latestTtfb
				},
			},
		}
		ttlb, err := TimedProbe(c, "https://www.google.com")

		require.NoError(t, err)
		require.Equal(t, 200*time.Millisecond, latestTtfb)
		require.Equal(t, 200*time.Millisecond, ttfbTotal)
		require.Equal(t, 3*time.Second, ttlb)
	})
}
