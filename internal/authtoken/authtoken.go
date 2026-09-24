package authtoken

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	AccountID uuid.UUID `json:"account_id"`
	Username  string    `json:"username"`
	jwt.RegisteredClaims
}

func secret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET not set")
	}
	return []byte(secret)
}

func Issue(accountID uuid.UUID, username string, ttl time.Duration) (string, error) {
	claims := Claims{
		AccountID: accountID,
		Username:  username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString(secret())
	if err != nil {
		return "", err
	}
	return signedString, nil
}

func Verify(token string) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret(), nil
		},
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Claims{}, err
	}
	return claims, nil
}
