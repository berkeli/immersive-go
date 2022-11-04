package main

import (
	"context"
	"testing"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"github.com/stretchr/testify/require"
)

func TestMockProbeServer(t *testing.T) {
	t.Run("Error response", func(t *testing.T) {
		server := &MockProbeServer{}
		ctx := context.Background()
		req := &pb.ProbeRequest{
			Endpoint: "http://musterror:8080",
		}

		_, err := server.DoProbes(ctx, req)
		require.Error(t, err)
	})
	t.Run("Success response", func(t *testing.T) {
		server := &MockProbeServer{}
		ctx := context.Background()
		req := &pb.ProbeRequest{
			Endpoint: "http://mustsuccess:8080",
		}

		_, err := server.DoProbes(ctx, req)
		require.NoError(t, err)
	})

}
