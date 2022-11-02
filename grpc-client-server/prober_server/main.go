package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	port         = flag.Int("port", 50051, "The server port")
	LatencyGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "latency_gauge",
		Help: "The latency of the requests to the endpoint",
	}, []string{"endpoint", "type"})
	timeSince = time.Since
)

type ResponseTime struct {
	Ttfb time.Duration
	Ttlb time.Duration
}

// server is used to implement prober.ProberServer.
type Server struct {
	pb.UnimplementedProberServer
}

func (s *Server) DoProbes(ctx context.Context, in *pb.ProbeRequest) (*pb.ProbeReply, error) {
	ttfbTotal := time.Duration(0)
	ttlbTotal := time.Duration(0)
	failed := 0
	latestTtfb := time.Duration(0)

	c := &http.Client{
		Transport: &TimedRoundTripper{
			defaultTripper: http.DefaultTransport,
			recordTime: func(t time.Duration) {
				latestTtfb = t
				ttfbTotal += t
			},
		},
	}

	for i := 0; i < int(in.GetNumberOfRequests()); i++ {
		ttlb, err := TimedProbe(c, in.GetEndpoint())
		if err != nil {
			log.Printf("could not probe: %v", err)
			failed++
			continue
		}
		LatencyGauge.WithLabelValues(in.GetEndpoint(), "TTFB").Set(float64(latestTtfb / time.Millisecond))
		LatencyGauge.WithLabelValues(in.GetEndpoint(), "TTLB").Set(float64(ttlb / time.Millisecond))
		ttlbTotal += ttlb
	}
	ttfbAverage := time.Duration(float32(ttfbTotal) / float32(in.NumberOfRequests-int32(failed)))
	ttlbAverage := time.Duration(float32(ttlbTotal) / float32(in.NumberOfRequests-int32(failed)))

	return &pb.ProbeReply{
		TtfbAverageResponseTime: durationpb.New(ttfbAverage),
		TtlbAverageResponseTime: durationpb.New(ttlbAverage),
		FailedRequests:          int32(failed),
	}, nil
}
func InitMonitoring() {
	prometheus.MustRegister(LatencyGauge)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()
}

func main() {
	flag.Parse()
	InitMonitoring()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterProberServer(s, &Server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func TimedProbe(c *http.Client, url string) (ttlb time.Duration, err error) {
	var start time.Time
	nullTime := time.Duration(0)

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nullTime, err
	}
	start = time.Now()

	resp, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nullTime, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ttlb, fmt.Errorf("status code %d", resp.StatusCode)
	}

	_, err = io.ReadAll(resp.Body)
	resp.Body.Close()

	ttlb = timeSince(start)

	return ttlb, nil
}

type TimedRoundTripper struct {
	defaultTripper http.RoundTripper
	recordTime     func(time.Duration)
}

func (t *TimedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.defaultTripper.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode >= http.StatusOK || resp.StatusCode < http.StatusMultipleChoices {
		t.recordTime(time.Since(start))
	}
	return resp, nil
}
