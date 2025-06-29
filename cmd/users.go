package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type userKey string

const (
	userCtx userKey = "user"
)

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	Verified  bool      `json:"verified"`
	BrandId   int32     `json:"brandId"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type InviteUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=2,max=100"`
}

type UserWithToken struct {
	UserResponse
	Token string `json:"token,omitempty"`
}

// @Summary		Invite a new user
// @Description	Invites a new user by creating an account and sending an activation email
// @Tags			users
// @Accept			json
// @Produce		json
// @Param			request	body		InviteUserPayload	true	"User invitation details"
// @Success		201		{object}	UserWithToken		"User created successfully with invitation token"
// @Failure		400		{object}	error				"Bad request - validation error or user already exists"
// @Failure		403		{object}	error				"Forbidden - only owner role can invite users"
// @Failure		500		{object}	error				"Internal server error"
// @Security		CookieAuth
// @Router			/users/invite [post]
func (app *application) inviteUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)
	if ctxUser.Role != ownerRole || ctxUser.BrandID.Int32 == 0 || !ctxUser.BrandID.Valid {
		app.forbiddenResponse(w, r, errors.New("access denied"))
		return
	}

	var payload InviteUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	randPw, err := generateRandomPassword()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(randPw), bcrypt.DefaultCost)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	user, err := app.store.CreateUser(ctx, store.CreateUserParams{
		Name:     payload.Username,
		Email:    payload.Email,
		Password: hashedPass,
		Role:     adminRole,
		BrandID: sql.NullInt32{
			Valid: true,
			Int32: ctxUser.BrandID.Int32,
		},
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

	err = app.store.CreateUserInvitation(ctx, store.CreateUserInvitationParams{
		Token:  hashToken(plainToken),
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
		Email         string
		Username      string
		Password      string
		ActivationUrl string
	}{
		Email:         user.Email,
		Username:      user.Name,
		Password:      randPw,
		ActivationUrl: activationURL,
	}

	status, err := app.mailer.Send(mailer.UserInvitationTemplate, user.Name, user.Email, vars)
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

	userWithToken := UserWithToken{
		UserResponse: userResponse,
		Token:        plainToken,
	}
	if err := writeJSON(w, http.StatusCreated, userWithToken); err != nil {
		app.internalServerError(w, r, err)
	}
}

type ActivationResponse struct {
	Message     string `json:"message"`
	RedirectURL string `json:"redirectUrl"`
}

// @Summary		Activate a user account
// @Description	Activates a user account using the token sent in the activation email
// @Tags			users
// @Accept			json
// @Produce		json
// @Param			token	path		string	true	"Activation token"
// @Success		204		{string}	string	"User activated"
// @Failure		404		{object}	error
// @Failure		500		{object}	error
// @Router			/users/confirm/{token} [get]
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		app.badRequestResponse(w, r, errors.New("invalid activation link"))
		return
	}

	err := app.store.ActivateUserTx(r.Context(), store.ActivateUserTxParams{
		Token: hashToken(token),
	})
	if err != nil {
		switch {
		case err == store.ErrInvalidActivationToken:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	response := ActivationResponse{
		Message:     "Account successfully activated",
		RedirectURL: app.config.clientUrl + "/login",
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetUser godoc
//
//	@Summary		Fetches a user profile
//	@Description	Fetches a user profile by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	UserResponse
//	@Failure		400	{object}	error
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Security		CookieAuth
//	@Router			/users/{id} [get]
func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.getUser(r.Context(), userID)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.notFoundResponse(w, r, err)
			return
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	userResponse := userResponseMapper(user)
	if err := writeJSON(w, http.StatusOK, userResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetUser godoc
//
//	@Summary		Get user by token
//	@Description	Fetches a user token
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	UserResponse
//	@Failure		400	{object}	error
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Security		CookieAuth
//	@Router			/users/me [get]
func (app *application) getUserProfile(w http.ResponseWriter, r *http.Request) {
	ctxUser := r.Context().Value(userCtx).(*store.User)
	userResponse := userResponseMapper(ctxUser)
	if err := writeJSON(w, http.StatusOK, userResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

// @Summary		Get users by brand
// @Description	Fetches all users of a brand
// @Tags			users
// @Accept			json
// @Produce		json
// @Success		200	{object}	[]UserResponse
// @Failure		400	{object}	error
// @Failure		404	{object}	error
// @Failure		500	{object}	error
// @Security		CookieAuth
// @Router			/users [get]
func (app *application) getUsersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)
	if ctxUser.BrandID.Int32 == 0 || !ctxUser.BrandID.Valid {
		app.forbiddenResponse(w, r, errors.New("access denied"))
		return
	}

	users, err := app.store.GetUsersByBrand(ctx, sql.NullInt32{
		Valid: true,
		Int32: ctxUser.BrandID.Int32,
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var result []UserResponse
	for _, v := range users {
		result = append(result, userResponseMapper(v))
	}

	if err = writeJSON(w, http.StatusOK, result); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) getUser(ctx context.Context, userID int64) (*store.User, error) {
	if !app.config.cache.enabled {
		user, err := app.store.GetUserById(ctx, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}
		return user, nil
	}

	user, err := app.cache.Users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user, err = app.store.GetUserById(ctx, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}
		if err := app.cache.Users.Set(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}
