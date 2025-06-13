package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

func (app *application) AuthUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SESSION_TOKEN)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("session cookie is missing"))
			return
		}

		sessionId, err := uuid.Parse(cookie.Value)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}

		ctx := r.Context()

		session, err := app.store.GetUserSessionById(ctx, sessionId)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("session not found"))
			return
		}

		if time.Now().After(session.ExpiresAt) {
			app.ClearCookie(w, SESSION_TOKEN)
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("session expired"))
			return
		}

		user, err := app.getUser(ctx, session.UserID)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx = context.WithValue(ctx, userCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) AuthCustomerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CUSTOMER_SESSION_TOKEN)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("session cookie is missing"))
			return
		}

		sessionId, err := uuid.Parse(cookie.Value)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}

		ctx := r.Context()

		session, err := app.store.GetCustomerSessionById(ctx, sessionId)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("session not found"))
			return
		}

		if time.Now().After(session.ExpiresAt) {
			app.ClearCookie(w, CUSTOMER_SESSION_TOKEN)
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("session expired"))
			return
		}

		customer, err := app.getCustomer(ctx, session.CustomerID)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx = context.WithValue(ctx, userCtx, customer)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

func (app *application) BrandMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		parts := strings.Split(host, ".")

		// Handle development scenarios where swagger does not have the brand in the subdomain
		if app.config.env == "development" {
			brandIDHeader := r.Header.Get("X-Brand-ID")
			if brandIDHeader != "" {
				id, err := strconv.ParseInt(brandIDHeader, 10, 32)
				if err == nil {
					ctx := context.WithValue(r.Context(), brandIDCtx, int32(id))
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		brand := parts[0]

		id, err := app.store.GetBrandByUrl(context.Background(), brand)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				app.notFoundResponse(w, r, err)
			default:
				app.internalServerError(w, r, err)
			}
			return
		}

		ctx := context.WithValue(r.Context(), brandIDCtx, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
