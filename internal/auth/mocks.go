package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type MockAuthenticator struct{}

const secret = "test"

func (a *MockAuthenticator) GenerateTokens(id int64) (string, string, error) {
    // Mock access token
    accessClaims := jwt.MapClaims{
        "sub":  id,
        "exp":  time.Now().Add(time.Hour).Unix(),
        "iat":  time.Now().Unix(),
        "nbf":  time.Now().Unix(),
        "iss":  "test-aud",
        "aud":  "test-aud",
        "type": "access",
    }

    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessTokenString, _ := accessToken.SignedString([]byte(secret))

    // Mock refresh token
    refreshClaims := jwt.MapClaims{
        "sub":  id,
        "exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
        "iat":  time.Now().Unix(),
        "nbf":  time.Now().Unix(),
        "iss":  "test-aud",
        "aud":  "test-aud",
        "type": "refresh",
    }

    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshTokenString, _ := refreshToken.SignedString([]byte(secret))

    return accessTokenString, refreshTokenString, nil
}

func (a *MockAuthenticator) ValidateToken(token string) (*jwt.Token, error) {
    return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
}

func (a *MockAuthenticator) RefreshTokens(refreshToken string) (access, refresh string, err error) {
    // Validate the refresh token
    jwtToken, err := a.ValidateToken(refreshToken)
    if err != nil {
        return "", "", err
    }

    claims, ok := jwtToken.Claims.(jwt.MapClaims)
    if !ok {
        return "", "", errors.New("invalid token claims")
    }

    tokenType, ok := claims["type"].(string)
    if !ok || tokenType != "refresh" {
        return "", "", errors.New("invalid token type")
    }

    id, ok := claims["sub"].(float64)
    if !ok {
        return "", "", errors.New("invalid subject in token")
    }

    // Generate new tokens
    return a.GenerateTokens(int64(id))
}