package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type MockAuthenticator struct{}

const secret = "test"

var testClaims = jwt.MapClaims{
	"aud": "test-aud",
	"iss": "test-aud",
	"sub": int64(1),
	"exp": time.Now().Add(time.Hour).Unix(),
}

func (a *MockAuthenticator) GenerateToken(claims jwt.Claims) (string, error) {
	if claims == nil {
		claims = testClaims
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString, nil
}

func (a *MockAuthenticator) ValidateToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
}
