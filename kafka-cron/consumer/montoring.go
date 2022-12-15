package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics map[string]interface{}

var (
	JobsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_total",
		Help: "Total number of jobs received",
	}, []string{"topic", "description"})
	JobsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_processed",
		Help: "Total number of jobs processed",
	}, []string{"topic", "description"})
	JobsFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_failed",
		Help: "Total number of jobs failed",
	}, []string{"topic", "description"})
	JobsInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "jobs_in_flight",
		Help: "Number of jobs in flight",
	}, []string{"topic", "description"})
	JobsRetried = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_retried",
		Help: "Total number of jobs retried",
	}, []string{"topic", "description"})
	JobQueueTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "job_queue_time_seconds",
		Help: "Time in queue in seconds",
	}, []string{"topic", "description"})
	JobDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "job_duration_seconds",
		Help: "Duration of jobs in seconds",
	}, []string{"topic", "description"})
	ErrorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "error_counter",
		Help: "Number of errors",
	}, []string{"topic", "type"})
)

func InitMonitoring(port int) (Metrics, error) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}()
	return nil, nil
}
