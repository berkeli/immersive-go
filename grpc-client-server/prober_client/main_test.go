package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
)

type mockProbeServer struct {
	pb.UnimplementedProberServer
}

func (*mockProbeServer) DoProbes(ctx context.Context, req *pb.ProbeRequest) (*pb.ProbeReply, error) {
	if req.Endpoint == "http://musterror:8080" {
		return nil, grpc.Errorf(15, "error")
	}
	return &pb.ProbeReply{
		AverageResponseTime: durationpb.New(123123 * time.Microsecond),
		FailedRequests:      0,
	}, nil
}

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	server := grpc.NewServer()

	pb.RegisterProberServer(server, &mockProbeServer{})

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestArrayFlag(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := map[string]struct {
			i    *ArrayFlag
			want string
		}{
			"empty": {
				i:    &ArrayFlag{},
				want: "",
			},
			"1 element": {
				i:    &ArrayFlag{"foo"},
				want: "foo",
			},
			"2 elements": {
				i:    &ArrayFlag{"foo", "bar"},
				want: "foo, bar",
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				require.Equal(t, tt.want, tt.i.String())
			})
		}
	})

	t.Run("Set", func(t *testing.T) {
		i := &ArrayFlag{}
		tests := map[string]struct {
			value string
			want  *ArrayFlag
		}{
			"add element": {
				value: "foo",
				want:  &ArrayFlag{"foo"},
			},
			"add another element": {
				value: "bar",
				want:  &ArrayFlag{"foo", "bar"},
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				require.NoError(t, i.Set(tt.value))
				require.Equal(t, tt.want, i)
			})
		}
	})
}

func TestGRpc(t *testing.T) {

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	require.NoError(t, err)

	defer conn.Close()
	client := pb.NewProberClient(conn)

	resp, err := client.DoProbes(ctx, &pb.ProbeRequest{Endpoint: "http://localhost:8080", NumberOfRequests: 1})

	require.NoError(t, err)

	require.Equal(t, resp.AverageResponseTime, durationpb.New(123123*time.Microsecond))
	require.Equal(t, resp.FailedRequests, int32(0))
}

func TestSingleProbe(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	require.NoError(t, err)

	defer conn.Close()
	client := pb.NewProberClient(conn)

	tests := map[string]struct {
		endpoint string
		want     string
	}{
		"success": {
			endpoint: "http://localhost:8080",
			want:     "Probing: http://localhost:8080 100% [===============]",
		},
		"error": {
			endpoint: "http://musterror:8080",
			want:     "Could not probe http://musterror:8080",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			req := &pb.ProbeRequest{Endpoint: tt.endpoint, NumberOfRequests: 1}
			wg := &sync.WaitGroup{}
			wg.Add(1)
			SingleProbe(&buf, client, req, wg)
			wg.Wait()
			got := stripansi.Strip(buf.String())
			require.Contains(t, got, tt.want)
		})
	}
}

func TestCreateProgressBar(t *testing.T) {
	var buf bytes.Buffer
	res := make(chan *Result, 1)
	wg := &sync.WaitGroup{}

	res <- &Result{
		Endpoint: "http://localhost:8080",
		Average:  123232 * time.Microsecond,
	}

	want := `
Probing: http://localhost:8080 100% [===============] 
+-----------------+----------------+-----------------+
| AVERAGE LATENCY | SUCCESS RATE % | FAILED REUQESTS |
+-----------------+----------------+-----------------+
| 123.232ms       |            100 |               0 |
+-----------------+----------------+-----------------+
`

	wg.Add(1)

	CreateProgressBar(&buf, 500*time.Millisecond, "http://localhost:8080", res, wg)

	wg.Wait()

	out := buf.String()

	require.Equal(t, strings.TrimSpace(want), stripansi.Strip(strings.TrimSpace(out)))
}

func TestPrintResults(t *testing.T) {
	tests := map[string]struct {
		res  *Result
		want string
	}{
		"success": {
			res: &Result{
				Endpoint: "http://localhost:8080",
				Average:  123 * time.Millisecond,
			},
			want: `
+-----------------+----------------+-----------------+
| AVERAGE LATENCY | SUCCESS RATE % | FAILED REUQESTS |
+-----------------+----------------+-----------------+
| 123ms           |            100 |               0 |
+-----------------+----------------+-----------------+
`,
		},
		"partial error": {
			res: &Result{
				Endpoint: "http://localhost:8080",
				Average:  123 * time.Millisecond,
				Failed:   1,
			},
			want: `
+-----------------+----------------+-----------------+
| AVERAGE LATENCY | SUCCESS RATE % | FAILED REUQESTS |
+-----------------+----------------+-----------------+
| 123ms           |             50 |               1 |
+-----------------+----------------+-----------------+
`,
		},
		"full error": {
			res: &Result{
				Endpoint: "http://localhost:8080",
				Err:      fmt.Errorf("error"),
			},
			want: "Could not probe http://localhost:8080: error",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			totalRequests := 2
			PrintResults(&buf, tt.res, &totalRequests)

			out := buf.String()

			require.Equal(t, strings.TrimSpace(tt.want), stripansi.Strip(strings.TrimSpace(out)))
		})
	}
}
