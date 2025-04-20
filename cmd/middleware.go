package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/golang-jwt/jwt/v5"
)

func (app *application) BasicAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				app.unauthorizedBasicErrorResponse(w, r, fmt.Errorf("authorization header is missing"))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Basic" {
				app.unauthorizedBasicErrorResponse(w, r, fmt.Errorf("authorization header is malformed"))
				return
			}

			decoded, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				app.unauthorizedBasicErrorResponse(w, r, err)
				return
			}

			username := app.config.auth.basic.user
			pass := app.config.auth.basic.pass

			creds := strings.SplitN(string(decoded), ":", 2)
			if len(creds) != 2 || creds[0] != username || creds[1] != pass {
				app.unauthorizedBasicErrorResponse(w, r, fmt.Errorf("invalid credentials"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (app *application) AuthTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is missing"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is malformed"))
			return
		}

		token := parts[1]
		jwtToken, err := app.auth.ValidateToken(token)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		claims, _ := jwtToken.Claims.(jwt.MapClaims)

		userID, err := strconv.ParseInt(fmt.Sprintf("%.f", claims["sub"]), 10, 64)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx := r.Context()

		user, err := app.getUser(ctx, userID)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx = context.WithValue(ctx, userCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) getUser(ctx context.Context, userID int64) (*store.User, error) {
	if !app.config.cache.enabled {
		return app.store.GetUserById(ctx, userID)
	}

	user, err := app.cache.Users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user, err = app.store.GetUserById(ctx, userID)
		if err != nil {
			return nil, err
		}

		if err := app.cache.Users.Set(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (app *application) RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.rateLimiter.Enabled {
			if allow, retryAfter := app.rateLimiter.Allow(r.RemoteAddr); !allow {
				app.rateLimitExceededResponse(w, r, retryAfter.String())
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
