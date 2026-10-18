package accounts

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mike-moseley/redoubt/internal/authtoken"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = time.Hour * 24

type tokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
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
