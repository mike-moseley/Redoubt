package accounts_test

import (
	"time"

	"github.com/google/uuid"
)

type testSignupReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type testSignupResp struct {
	Username  string    `json:"username"`
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type testToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type testErr struct {
	Error string `json:"error"`
}
