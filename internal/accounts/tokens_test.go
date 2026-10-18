package accounts_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mike-moseley/redoubt/internal/authtoken"
)

func TestCreateTokenRoundTrip(t *testing.T) {
	reset(t)
	body, signupResp := signup(t)
	tokenRec := doJSON(t, http.MethodPost, "/tokens", body)
	if tokenRec.Code != http.StatusOK {
		t.Fatalf("Error creating token; have %v, want %v", tokenRec.Code, http.StatusOK)
	}
	var tokenResp testToken
	err := json.Unmarshal(tokenRec.Body.Bytes(), &tokenResp)
	if err != nil {
		t.Fatalf("Error unmarshaling JSON response: %v", err)
	}
	claim, err := authtoken.Verify(tokenResp.Token)
	if err != nil {
		t.Fatalf("Error verifying token: %v", err)
	}
	if claim.AccountID != signupResp.ID {
		t.Fatalf("Mismatched IDs: have %v, want %v", claim.AccountID, signupResp.ID)
	}
	if !claim.ExpiresAt.Equal(tokenResp.ExpiresAt) {
		t.Fatalf("Mismatched token expiration: have %v, want %v", claim.ExpiresAt.UTC(), tokenResp.ExpiresAt)
	}
}

func TestWrongPassword(t *testing.T) {
	reset(t)
	body, _ := signup(t)
	body.Password = "wrong!"
	tokenRec := doJSON(t, http.MethodPost, "/tokens", body)
	if tokenRec.Code != http.StatusUnauthorized {
		t.Fatalf("Mismatched http response: have %v, want %v", tokenRec.Code, http.StatusUnauthorized)
	}
}
