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
	CronJobsInFlight = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cron_jobs_in_flight",
		Help: "Number of cron jobs in flight",
	})
	JobsPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_published",
		Help: "Total number of jobs published",
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
