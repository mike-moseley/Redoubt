package authtoken

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestRoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	token, err := Issue(uuid.New(), "mike", 10 * time.Minute)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	claims, err := Verify(token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.Username != "mike" {
		t.Errorf("username = %q, wanted %q", claims.Username, "mike")
	}
}

func TestTamper(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-tamper")
	token, err := Issue(uuid.New(), "doink", 10 * time.Minute)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	parts := strings.Split(token, ".")
	b := []byte(parts[2])
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	parts[2] = string(b)
	tampered := strings.Join(parts, ".")

	_, err = Verify(tampered)
	if !errors.Is(err, jwt.ErrSignatureInvalid) {
		t.Fatalf("Expected error `jwt.ErrSignatureInvalid`, got `%v`", err)
	}
	if err == nil {
		t.Fatal("Verify passed, should have failed.")
	}
}

func TestExpired(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-time")
	token, err := Issue(uuid.New(), "zoink", -1 * time.Minute)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	_, err = Verify(token)
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("Expected error `jwt.ErrTokenExpired`, got `%v`", err)
	}
	if err == nil {
		t.Fatal("Verify passed, should have failed.")
	}
	
}
