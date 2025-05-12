package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	secret string
	aud    string
	iss    string
	exp    time.Duration
}

func NewTokenService(secret, aud, iss string, exp time.Duration) *TokenService {
	return &TokenService{secret, iss, aud, exp}
}

const (
	ErrTokenType   = "invalid token type"
	ErrTokenClaims = "invalid token claims"
)

func (ts *TokenService) generateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(ts.secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (ts *TokenService) ValidateToken(token string) (*jwt.Token, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}

		return []byte(ts.secret), nil
	},
		jwt.WithExpirationRequired(),
		jwt.WithAudience(ts.aud),
		jwt.WithIssuer(ts.iss),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %v", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return parsedToken, nil

}

func (ts *TokenService) GenerateTokens(id int64) (access, refresh string, err error) {
	claims := jwt.MapClaims{
		"sub":  id,
		"exp":  time.Now().Add(ts.exp).Unix(),
		"iat":  time.Now().Unix(),
		"nbf":  time.Now().Unix(),
		"iss":  ts.iss,
		"aud":  ts.iss,
		"type": "access",
	}

	accessToken, err := ts.generateToken(claims)
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		"sub":  id,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"nbf":  time.Now().Unix(),
		"iss":  ts.iss,
		"aud":  ts.iss,
		"type": "refresh",
	}

	refreshToken, err := ts.generateToken(refreshClaims)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil

}
func (ts *TokenService) RefreshTokens(refreshToken string) (access, refresh string, err error) {
	jwtToken, err := ts.ValidateToken(refreshToken)
	if err != nil {
		return "", "", err
	}
	fmt.Println("CODE SHOLD NOT REACH HERE")
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New(ErrTokenClaims)
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return "", "", errors.New(ErrTokenType)
	}

	id, err := strconv.ParseInt(fmt.Sprintf("%.f", claims["sub"]), 10, 64)
	if err != nil {
		return "", "", err
	}

	return ts.GenerateTokens(id)
}
