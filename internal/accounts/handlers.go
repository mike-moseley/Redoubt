package accounts

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
	"golang.org/x/crypto/bcrypt"
)

type pinger interface {
	Ping(context.Context) error
}

type Server struct {
	q      *queries.Queries
	pinger pinger
}

type signupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type signupResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("POST /accounts", s.handleSignup)

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

func (s *Server) handleSignup(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, 4096)
	jsonDec := json.NewDecoder(req.Body)
	jsonDec.DisallowUnknownFields()
	var body signupRequest
	err := jsonDec.Decode(&body)
	if err != nil {
		log.Printf("error decoding json: %v", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if (len(body.Password) < 8) || (len(body.Password) > 72) {
		writeError(w, http.StatusBadRequest, "password must be between 8 and 72 characters")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("error hashing password: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	queryResp, err := s.q.CreateAccount(req.Context(), queries.CreateAccountParams{
		Username:     body.Username,
		PasswordHash: string(hash),
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.ConstraintName {
			case "accounts_username_lower_idx":
				writeError(w, http.StatusConflict, "username taken")
				return
			case "accounts_username_length":
				writeError(w, http.StatusBadRequest, "username must be 1–20 characters with no leading or trailing spaces")
				return
			}
		}
		log.Printf("error querying database to create account: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, signupResponse{
		ID:        queryResp.ID,
		Username:  queryResp.Username,
		CreatedAt: queryResp.CreatedAt.UTC(),
	})
}
