package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type Authenticator interface {
	GenerateTokens(id int64) (string, string, error)
	ValidateToken(token string) (*jwt.Token, error)
	RefreshTokens(refreshToken string) (access, refresh string, err error)
}
