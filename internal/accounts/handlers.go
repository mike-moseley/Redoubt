package accounts

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

type pinger interface {
	Ping(context.Context) error
}

type Server struct {
	q      *queries.Queries
	pinger pinger
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)

	return mux
}

func NewServer(q *queries.Queries, p pinger) *Server {
	return &Server{q: q, pinger: p}
}

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
