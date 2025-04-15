package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=3,max=72"`
	Username string `json:"username" validate:"required,min=2,max=100"`
}

// @Summary Register a new user
// @Description Register a new user with email, password and username
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterUserPayload true "User registration details"
// @Success 201 {object} struct{User *store.User "user details"; Token string "activation token"}
// @Failure 400 {object} ErrorResponse "Invalid request payload"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /register [post]
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
		app.parsePGError(w, r, err)
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
		app.parsePGError(w, r, err)
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
			app.parsePGError(w, r, err)
			return
		}

		app.internalServerError(w, r, err)
		return
	}

	app.logger.Infow("Email sent", "status code", status)

	userWithToken := struct {
		User  *store.User
		Token string `json:"token"`
	}{
		User:  &user,
		Token: plainToken,
	}

	if err := writeJSON(w, http.StatusCreated, userWithToken); err != nil {
		app.internalServerError(w, r, err)
	}
}

// @Summary Activate a user account
// @Description Activates a user account using the token sent in the activation email
// @Tags auth
// @Accept json
// @Produce json
// @Param token path string true "Activation token"
// @Success 200 {object} nil "User successfully activated"
// @Failure 400 {object} ErrorResponse "Invalid activation token"
// @Failure 404 {object} ErrorResponse "Token not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /confirm/{token} [get]
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		app.badRequestResponse(w, r, errors.New("invalid activation link"))
		return
	}

	ctx := r.Context()
	hashToken := app.hashToken(token)

	userId, err := app.store.GetUserFromInvitation(ctx, hashToken)
	if err != nil {
		app.parsePGError(w, r, err)
		return
	}

	if err := app.store.VerifyUser(ctx, userId); err != nil {
		app.parsePGError(w, r, err)
		return
	}

	if err := app.store.DeleteUserInvitation(ctx, userId); err != nil {
		app.parsePGError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusNoContent, ""); err != nil {
		app.internalServerError(w, r, err)
	}
}
