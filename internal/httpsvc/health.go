package httpsvc

import (
	"fmt"
	"net/http"
)

// Health represents a Health server which can be used to expose a health check.
type Health struct {
	Server
}

// Start starts the Health Server.
func (s *Health) Start(stopCh <-chan struct{}) error {
	registerHealthCheck(&s.ServeMux)

	return s.Server.Start(stopCh)
}

func registerHealthCheck(mux *http.ServeMux) {
	mux.HandleFunc("/_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
}
