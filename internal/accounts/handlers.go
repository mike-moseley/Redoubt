package accounts

import (
	"net/http"

	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

type Server struct {
	q *queries.Queries
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)

	return mux
}

func NewServer(q *queries.Queries) *Server {
	return &Server{q: q}
}

func (s *Server) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}
