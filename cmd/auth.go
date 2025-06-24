package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SignUpUserPayload struct {
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
//	@Param			payload	body		SignUpUserPayload	true	"User credentials"
//	@Success		201		{object}	UserResponse		"Register a new user"
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
//	@Router			/auth/signup [post]
func (app *application) signUpUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload SignUpUserPayload
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

	session, err := app.store.CreateUserSession(r.Context(), store.CreateUserSessionParams{
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(app.config.auth.session.exp),
	})
	if err != nil {
		app.internalServerError(w, r, err)
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

	app.SetCookie(w, SESSION_TOKEN, session.ID.String())

	userResponse := userResponseMapper(user)

	if err := writeJSON(w, http.StatusCreated, userResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

type SignInUserPayload struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3,max=72"`
}

// createTokenHandler godoc
//
//	@Summary		Sign in user
//	@Description	Sign in user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		SignInUserPayload	true	"User credentials"
//	@Success		200		{string}	UserResponse		"User data"
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Router			/auth/signin [post]
func (app *application) signInUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload SignInUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()

	user, err := app.store.GetUserByEmail(ctx, payload.Email)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.unauthorizedErrorResponse(w, r, err)
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

	session, err := app.store.UpsertUserSession(ctx, store.UpsertUserSessionParams{
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(app.config.auth.session.exp),
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.SetCookie(w, SESSION_TOKEN, session.ID.String())

	userResponse := userResponseMapper(user)

	if err := writeJSON(w, http.StatusOK, userResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

// logoutHandler godoc
//
//	@Summary		Logs out a user
//	@Description	Clears the session cookie to log out the user
//	@Tags			auth
//	@Produce		json
//	@Success		200	{string}	string	"Logged out successfully"
//	@Failure		401	{object}	error
//	@Failure		500	{object}	error
//	@Router			/auth/logout [post]
func (app *application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SESSION_TOKEN)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	sessionId, err := uuid.Parse(cookie.Value)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	_, err = app.store.UpdateUserSession(r.Context(), store.UpdateUserSessionParams{
		ID:        sessionId,
		ExpiresAt: time.Unix(0, 0),
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.ClearCookie(w, SESSION_TOKEN)

	response := map[string]string{
		"message": "Logged out successfully",
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}
