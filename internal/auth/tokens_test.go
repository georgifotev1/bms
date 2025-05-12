package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestTokenService(t *testing.T) {
	secret := "secret"
	aud := "iss"
	iss := "iss"
	exp := time.Hour

	tokenService := NewTokenService(secret, aud, iss, exp)
	t.Run("GenerateTokens", func(t *testing.T) {
		accessToken, refreshToken, err := tokenService.GenerateTokens(12345)
		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	t.Run("ValidateToken", func(t *testing.T) {
		accessToken, _, err := tokenService.GenerateTokens(12345)
		assert.NoError(t, err)

		token, err := tokenService.ValidateToken(accessToken)
		assert.NoError(t, err)
		assert.NotNil(t, token)

		claims, ok := token.Claims.(jwt.MapClaims)
		assert.True(t, ok)
		assert.Equal(t, float64(12345), claims["sub"])
		assert.Equal(t, "access", claims["type"])
	})

	t.Run("RefreshTokens", func(t *testing.T) {
		_, refreshToken, err := tokenService.GenerateTokens(12345)
		assert.NoError(t, err)

		newAccessToken, newRefreshToken, err := tokenService.RefreshTokens(refreshToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)
		assert.NotEmpty(t, newRefreshToken)

		token, err := tokenService.ValidateToken(newAccessToken)
		assert.NoError(t, err)
		assert.NotNil(t, token)

		claims, ok := token.Claims.(jwt.MapClaims)
		assert.True(t, ok)
		assert.Equal(t, float64(12345), claims["sub"])
		assert.Equal(t, "access", claims["type"])
	})
	t.Run("RefreshTokens_invalid_token", func(t *testing.T) {
		_, _, err := tokenService.RefreshTokens("invalid-token")
		assert.Error(t, err)
	})
}
