// Package main implements a client for Prober service.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	pb "github.com/Berkeli/immersive-go/grpc-client-server/prober"
	"github.com/jedib0t/go-pretty/table"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Result struct {
	Endpoint string
	Failed   int32
	Average  float32
	Err      error
}

type ArrayFlag []string

func (i *ArrayFlag) String() string {
	return strings.Join(*i, ", ")
}

func (i *ArrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	endpoints   = ArrayFlag{}
	addr        = flag.String("addr", "localhost:50051", "the address to connect to")
	nOfRequests = flag.Int("tries", 1, "number of requests to make")
	timeout     = flag.Int("timeout", 3, "timeout in seconds, how long should we allow for probing")
)

func main() {
	flag.Var(&endpoints, "endpoint", "the endpoint to probe, can specify multiple endpoints with multiple flags, e.g. --endpoint https://google.com --endpoint https://duckduckgo.com")

	flag.Parse()

	if len(endpoints) == 0 {
		endpoints.Set("http://google.com")
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewProberClient(conn)

	// Contact the server and print out its response.
	var wg sync.WaitGroup
	for _, endpoint := range endpoints {
		req := &pb.ProbeRequest{Endpoint: endpoint, NumberOfRequests: int32(*nOfRequests)}
		wg.Add(1)
		go SingleProbe(os.Stdout, c, req, &wg)
	}
	wg.Wait()
}

func SingleProbe(w io.Writer, c pb.ProberClient, req *pb.ProbeRequest, wg *sync.WaitGroup) {
	results := make(chan *Result)
	timeout := time.Duration(*timeout) * time.Second
	go CreateProgressBar(w, timeout, req.Endpoint, results, wg)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	r, err := c.DoProbes(ctx, req)
	cancel()

	if err != nil {
		results <- &Result{Endpoint: req.Endpoint, Err: err}
		return
	}

	results <- &Result{
		Endpoint: req.Endpoint,
		Failed:   r.FailedRequests,
		Average:  r.AverageResponseTime,
	}
}

func CreateProgressBar(w io.Writer, timeout time.Duration, endpoint string, results <-chan *Result, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := 200 * time.Millisecond
	bar := progressbar.NewOptions(int(timeout/ticker),
		progressbar.OptionSetWriter(w),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription(fmt.Sprintf("[cyan]Probing: %s", endpoint)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	for {
		select {
		case <-time.After(ticker):
			bar.Add(1)
		case res := <-results:
			if res.Err == nil {
				bar.Finish()
			}
			PrintResults(w, res, nOfRequests)
			return
		}
	}
}

func PrintResults(w io.Writer, res *Result, n *int) {
	fmt.Fprintln(w)
	if res.Err != nil {
		fmt.Fprintf(w, "Could not probe %s: %v", res.Endpoint, res.Err)
		return
	}
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"Average Latency", "Success rate %", "Failed Reuqests"})
	t.AppendRow(table.Row{res.Average, 100 - (float32(res.Failed) / float32(*n) * 100), res.Failed})
	t.Render()
}
