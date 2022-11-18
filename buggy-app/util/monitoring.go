package util

import (
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Monitor() {
	// WHEN NOT TESTING:
	// start prometheus metrics server
	if !strings.HasSuffix(os.Args[0], ".test") {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.ListenAndServe(":2112", nil)
		}()
	}
}
