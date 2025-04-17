package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=3,max=72"`
	Username string `json:"username" validate:"required,min=2,max=100"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserWithToken struct {
	UserResponse
	Token string `json:"token,omitempty"`
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

	plainToken := uuid.New().String()
	// hash the token for storage but keep the plain token for email
	hashToken := app.hashToken(plainToken)

	err = app.store.CreateUserInvitation(ctx, store.CreateUserInvitationParams{
		Token:  hashToken,
		UserID: user.ID,
		Expiry: time.Now().Add(time.Hour * 24),
	})
	if err != nil {
		switch {
		case isPgError(err, uniqueViolation):
			app.badRequestResponse(w, r, errors.New("user already exist"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	activationURL := fmt.Sprintf("%s/confirm/%s", app.config.clientUrl, plainToken)
	vars := struct {
		Username      string
		ActivationURL string
	}{
		Username:      user.Name,
		ActivationURL: activationURL,
	}

	status, err := app.mailer.Send(mailer.UserWelcomeTemplate, user.Name, user.Email, vars)
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

	userResponse := UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar.String,
		Verified:  user.Verified.Bool,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	userWithToken := UserWithToken{
		UserResponse: userResponse,
		Token:        plainToken,
	}
	if err := writeJSON(w, http.StatusCreated, userWithToken); err != nil {
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
		"sub": user.ID,
		"exp": time.Now().Add(app.config.auth.token.exp).Unix(),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"iss": app.config.auth.token.iss,
		"aud": app.config.auth.token.iss,
	}

	token, err := app.auth.GenerateToken(claims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusCreated, token); err != nil {
		app.internalServerError(w, r, err)
	}
}
