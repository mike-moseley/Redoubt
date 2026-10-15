package accounts

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
	"github.com/mike-moseley/redoubt/internal/authtoken"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = time.Hour * 24

type signupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type signupResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type tokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
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

func (s *Server) handleCreateToken(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, 4096)
	jsonDec := json.NewDecoder(req.Body)
	jsonDec.DisallowUnknownFields()
	var body tokenRequest
	err := jsonDec.Decode(&body)
	if err != nil {
		log.Printf("error decoding json: %v", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	account, err := s.q.GetAccountByUsername(req.Context(), body.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "incorrect username or password")
			return
		}
		log.Printf("error getting account: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(body.Password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			writeError(w, http.StatusUnauthorized, "incorrect username or password")
			return
		}
		log.Printf("error comparing password & hash: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	token, expiredAt, err := authtoken.Issue(account.ID, account.Username, tokenTTL)
	if err != nil {
		log.Printf("error issuing token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{
		Token:     token,
		ExpiresAt: expiredAt.UTC(),
	})
}
