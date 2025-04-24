package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type userKey string

const userCtx userKey = "user"

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	Verified  bool      `json:"verified"`
	BrandId   int32     `json:"brandId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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

	ctx := r.Context()

	userId, err := app.store.GetUserFromInvitation(ctx, hashToken(token))
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.notFoundResponse(w, r, errors.New("invalid token"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.store.VerifyUser(ctx, userId); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.store.DeleteUserInvitation(ctx, userId); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusNoContent, ""); err != nil {
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
//	@Security		ApiKeyAuth
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
