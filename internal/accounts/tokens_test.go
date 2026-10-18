package accounts_test

import (
	"encoding/json"
	"net/http"
	"strings"
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

func TestCreateTokenBadAuths(t *testing.T) {
	tests := []struct {
		name    string
		request testSignupReq
	}{
		{name: "wrong password", request: testSignupReq{
			Username: "mike",
			Password: "wrong!",
		}},
		{name: "unknown user", request: testSignupReq{
			Username: "notMike",
			Password: "right!",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reset(t)
			signup(t)
			tokenRec := doJSON(t, http.MethodPost, "/tokens", tc.request)
			if tokenRec.Code != http.StatusUnauthorized {
				t.Fatalf("Mismatched http response: have %v, want %v", tokenRec.Code, http.StatusUnauthorized)
			}
			expectedErr := testErr{
				Error: "incorrect username or password",
			}
			var haveErr testErr
			err := json.Unmarshal(tokenRec.Body.Bytes(), &haveErr)
			if err != nil {
				t.Fatalf("error unmarshaling JSON: %v", err)
			}
			if haveErr != expectedErr {
				t.Fatalf("Mismatched error message: have %v, want %v", haveErr, expectedErr)
			}
		})
	}
}

func TestCreateTokenBadRequests(t *testing.T) {
	badRequests := []struct {
		name  string
		input string
	}{
		{name: "extraField", input: `{"username":"mike", "password":"password", "extra":1}`},
		{name: "malformedJSON", input: `{"username":"mike", "password":"password}`},
	}

	for _, tc := range badRequests {
		t.Run(tc.name, func(t *testing.T) {
			// No need for reset(); these should fail before hitting db
			r := strings.NewReader(tc.input)
			rec := doRequest(t, http.MethodPost, "/tokens", r)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("Mismatched http response: have %v, want %v", rec.Code, http.StatusBadRequest)
			}
		})
	}
}
