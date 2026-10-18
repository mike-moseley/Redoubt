package accounts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mike-moseley/redoubt/internal/accounts"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
	"github.com/mike-moseley/redoubt/internal/testdb"
)

var pool *pgxpool.Pool

func newServer(t *testing.T) http.Handler {
	t.Helper()

	return accounts.NewServer(queries.New(pool), pool).Routes()
}

func signup(t *testing.T) (testSignupReq, testSignupResp) {
	t.Helper()
	body := testSignupReq{
		Username: "mike",
		Password: "password",
	}
	signupRec := doJSON(t, http.MethodPost, "/accounts", body)
	if signupRec.Code != http.StatusCreated {
		t.Fatalf("Error signing up; have %v, want %v", signupRec.Code, http.StatusCreated)
	}
	var signupResp testSignupResp
	err := json.Unmarshal(signupRec.Body.Bytes(), &signupResp)
	if err != nil {
		t.Fatalf("Error unmarshaling JSON response: %v", err)
	}
	return body, signupResp
}

func doJSON(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b []byte
	if body != nil {
		var err error
		b, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshalling json failed: %v", err)
		}
	}
	return doRequest(t, method, path, bytes.NewReader(b))
}

func doRequest(t *testing.T, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, req)
	return rec
}

func reset(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(t.Context(), "TRUNCATE accounts CASCADE")
	if err != nil {
		t.Fatalf("Error resetting db for tests: %v", err)
	}
}

func run(m *testing.M) int {
	ctx := context.Background()
	var cleanup func()
	var err error

	pool, cleanup, err = testdb.Start(ctx)
	if err != nil {
		log.Printf("error setting up testing db: %v", err)
		return 1
	}
	defer cleanup()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "testing_secret")
	os.Exit(run(m))
}
