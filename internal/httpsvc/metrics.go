package httpsvc

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics represents a Metrics server which can be used to expose a prometheus
// metrics.
type Metrics struct {
	Server
	*prometheus.Registry
}

// Start starts the Health Server.
func (s *Metrics) Start(stopCh <-chan struct{}) error {
	registerMetrics(&s.ServeMux, s.Registry)
	registerHealthCheck(&s.ServeMux)

	return s.Server.Start(stopCh)
}

func registerMetrics(mux *http.ServeMux, reg *prometheus.Registry) {
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}

func registerHealthCheck(mux *http.ServeMux) {
	mux.HandleFunc("/_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
}
