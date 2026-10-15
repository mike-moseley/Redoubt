package accounts

import (
	"context"
	"net/http"

	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

type pinger interface {
	Ping(context.Context) error
}

type Server struct {
	q      *queries.Queries
	pinger pinger
}

func NewServer(q *queries.Queries, p pinger) *Server {
	return &Server{q: q, pinger: p}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)

	mux.HandleFunc("POST /accounts", s.handleSignup)
	mux.HandleFunc("POST /tokens", s.handleCreateToken)

	return mux
}
