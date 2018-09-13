package httpsvc

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Server is a wrapper around a http.ServeMux which sets up necessary endpoints
// to run our operator.
type Server struct {
	http.ServeMux

	Addr string
	Port int
}

// Start starts the HTTP Server.
func (s *Server) Start(stopCh <-chan struct{}) error {
	srv := http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Addr, s.Port),
		Handler:      &s.ServeMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-stopCh

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	return srv.ListenAndServe()
}
