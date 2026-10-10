package accounts

import (
	"context"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReady(w http.ResponseWriter, req *http.Request) {
	ctx, cancelFn := context.WithTimeout(req.Context(), 2*time.Second)
	defer cancelFn()

	err := s.pinger.Ping(ctx)
	if err != nil {
		log.Printf("Error pinging database: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
