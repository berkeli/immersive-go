package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

var (
	port         = flag.Int("port", 50051, "The server port")
	LatencyGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "latency_gauge",
		Help: "The latency of the requests to the endpoint",
	}, []string{"endpoint"})
)

// server is used to implement prober.ProberServer.
type Server struct {
	pb.UnimplementedProberServer
}

func (s *Server) DoProbes(ctx context.Context, in *pb.ProbeRequest) (*pb.ProbeReply, error) {
	// TODO: support a number of repetitions and return average latency
	totalMsecs := 0
	failed := 0
	for i := 0; i < int(in.GetNumberOfRequests()); i++ {
		start := time.Now()
		resp, err := http.Get(in.GetEndpoint())
		if err != nil {
			log.Printf("could not probe: %v", err)
			failed++
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("Received status %d during probe", resp.StatusCode)
			failed++
			continue
		}
		elapsed := time.Since(start)
		elapsedMsecs := float32(elapsed / time.Millisecond)
		LatencyGauge.WithLabelValues(in.GetEndpoint()).Set(float64(elapsedMsecs))
		totalMsecs += int(elapsedMsecs)
	}
	averageMsecs := float32(totalMsecs) / float32(in.NumberOfRequests)

	return &pb.ProbeReply{AverageResponseTime: averageMsecs, FailedRequests: int32(failed)}, nil
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
