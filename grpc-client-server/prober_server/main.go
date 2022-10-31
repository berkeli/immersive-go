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
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	port         = flag.Int("port", 50051, "The server port")
	LatencyGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "latency_gauge",
		Help: "The latency of the requests to the endpoint",
	}, []string{"endpoint"})
	timeNow   = time.Now
	timeSince = time.Since
)

// server is used to implement prober.ProberServer.
type Server struct {
	pb.UnimplementedProberServer
}

func (s *Server) DoProbes(ctx context.Context, in *pb.ProbeRequest) (*pb.ProbeReply, error) {
	total := time.Duration(0)
	failed := 0
	for i := 0; i < int(in.GetNumberOfRequests()); i++ {
		start := timeNow()
		resp, err := http.Get(in.GetEndpoint())
		if err != nil {
			log.Printf("could not probe: %v", err)
			failed++
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Received status %d during probe", resp.StatusCode)
			failed++
			continue
		}
		resp.Body.Close()
		elapsed := timeSince(start)
		LatencyGauge.WithLabelValues(in.GetEndpoint()).Set(float64(elapsed / time.Millisecond))
		total += elapsed
	}
	average := time.Duration(float32(total) / float32(in.NumberOfRequests))

	return &pb.ProbeReply{AverageResponseTime: durationpb.New(average), FailedRequests: int32(failed)}, nil
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
