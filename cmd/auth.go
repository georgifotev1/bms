package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=3,max=72"`
	Username string `json:"username" validate:"required,min=2,max=100"`
}

// registerUserHandler godoc
//
//	@Summary		Registers a user
//	@Description	Registers a user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RegisterUserPayload	true	"User credentials"
//	@Success		201		{object}	UserResponse		"User registered"
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
//	@Router			/auth/user [post]
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	ctx := r.Context()

	user, err := app.store.CreateUser(ctx, store.CreateUserParams{
		Name:     payload.Username,
		Email:    payload.Email,
		Password: hashedPass,
		Role:     "owner",
		Verified: true,
	})
	if err != nil {
		switch {
		case isPgError(err, uniqueViolation):
			app.badRequestResponse(w, r, errors.New("user already exists"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	vars := struct {
		Username string
	}{
		Username: user.Name,
	}

	status, err := app.mailer.Send(mailer.WelcomeTemplate, user.Name, user.Email, vars)
	if err != nil {
		app.logger.Errorw("error sending welcome email", "error", err)

		if err := app.store.DeleteUser(ctx, user.ID); err != nil {
			switch err {
			case sql.ErrNoRows:
				app.notFoundResponse(w, r, errors.New("user does not exist"))
			default:
				app.internalServerError(w, r, err)
			}
			return
		}

		app.internalServerError(w, r, err)
		return
	}

	app.logger.Infow("Email sent", "status code", status)

	userResponse := userResponseMapper(user)
	if err := writeJSON(w, http.StatusCreated, userResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

type CreateUserTokenPayload struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3,max=72"`
}

// createTokenHandler godoc
//
//	@Summary		Creates a token
//	@Description	Creates a token for a user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CreateUserTokenPayload	true	"User credentials"
//	@Success		200		{string}	string					"Token"
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Router			/auth/token [post]
func (app *application) createTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserTokenPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.store.GetUserByEmail(r.Context(), payload.Email)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.unauthorizedBasicErrorResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword(user.Password, []byte(payload.Password))
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	claims := jwt.MapClaims{
		"sub":  user.ID,
		"exp":  time.Now().Add(app.config.auth.token.exp).Unix(),
		"iat":  time.Now().Unix(),
		"nbf":  time.Now().Unix(),
		"iss":  app.config.auth.token.iss,
		"aud":  app.config.auth.token.iss,
		"type": "access",
	}

	accessToken, err := app.auth.GenerateToken(claims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if isBrowser(r) {
		refreshClaims := jwt.MapClaims{
			"sub":  user.ID,
			"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
			"iat":  time.Now().Unix(),
			"nbf":  time.Now().Unix(),
			"iss":  app.config.auth.token.iss,
			"aud":  app.config.auth.token.iss,
			"type": "refresh",
		}

		refreshToken, err := app.auth.GenerateToken(refreshClaims)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     REFRESH_TOKEN,
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   60 * 60 * 24 * 30,
		})
	}

	response := map[string]string{
		"token": accessToken,
	}

	if err := writeJSON(w, http.StatusCreated, response); err != nil {
		app.internalServerError(w, r, err)
	}
}

// refreshTokenHandler godoc
//
//	@Summary		Refreshes an access token
//	@Description	Uses a refresh token to generate a new access token
//	@Tags			auth
//	@Produce		json
//	@Success		200	{string}	string	"New access token"
//	@Failure		401	{object}	error
//	@Failure		500	{object}	error
//	@Router			/auth/refresh [get]
func (app *application) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(REFRESH_TOKEN)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, errors.New("refresh token not found"))
		return
	}

	refreshToken := cookie.Value

	jwtToken, err := app.auth.ValidateToken(refreshToken)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		app.unauthorizedErrorResponse(w, r, errors.New("invalid token claims"))
		return
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		app.unauthorizedErrorResponse(w, r, errors.New("invalid token type"))
		return
	}

	userID, err := strconv.ParseInt(fmt.Sprintf("%.f", claims["sub"]), 10, 64)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	accessClaims := jwt.MapClaims{
		"sub":  userID,
		"exp":  time.Now().Add(app.config.auth.token.exp).Unix(),
		"iat":  time.Now().Unix(),
		"nbf":  time.Now().Unix(),
		"iss":  app.config.auth.token.iss,
		"aud":  app.config.auth.token.iss,
		"type": "access",
	}

	newAccessToken, err := app.auth.GenerateToken(accessClaims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	refreshClaims := jwt.MapClaims{
		"sub":  userID,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"nbf":  time.Now().Unix(),
		"iss":  app.config.auth.token.iss,
		"aud":  app.config.auth.token.iss,
		"type": "refresh",
	}

	newRefreshToken, err := app.auth.GenerateToken(refreshClaims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     REFRESH_TOKEN,
		Value:    newRefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 24 * 30,
	})

	response := map[string]string{
		"token": newAccessToken,
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}

// logoutHandler godoc
// @Summary		Logs out a user
// @Description	Clears the refresh token cookie to log out the user
// @Tags		auth
// @Produce		json
// @Success		200	{string}	string	"Logged out successfully"
// @Failure		500	{object}	error
// @Router		/auth/logout [post]
func (app *application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     REFRESH_TOKEN,
		Value:    "", // Empty value
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
	})

	response := map[string]string{
		"message": "Logged out successfully",
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}
