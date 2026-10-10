package accounts_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mike-moseley/redoubt/internal/accounts"
)

type fakePinger struct {
	err         error
	hadDeadline bool
}

func (f *fakePinger) Ping(ctx context.Context) error {
	_, dl := ctx.Deadline()
	f.hadDeadline = dl
	return f.err
}

func TestRouteHealthz(t *testing.T) {
	s := accounts.NewServer(nil, &fakePinger{err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, wanted %d", rec.Code, http.StatusOK)
	}
}

func TestRouteReadyz(t *testing.T) {
	testCases := []struct {
		name    string
		pingErr error
		status  int
	}{
		{"db up", nil, http.StatusOK},
		{"db down", errors.New("db down"), http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakePinger{err: tc.pingErr}
			s := accounts.NewServer(nil, fake)
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()
			s.Routes().ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Errorf("status = %d, wanted %d", rec.Code, tc.status)
			}
			if !fake.hadDeadline {
				t.Errorf("ping context has no deadline")
			}
		})
	}
}
