package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=3,max=72"`
	Username string `json:"username" validate:"required,min=2,max=100"`
}

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
	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])

	err = app.store.CreateUserInvitation(ctx, store.CreateUserInvitationParams{
		Token:  hashToken,
		UserID: user.ID,
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
