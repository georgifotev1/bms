package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
)

type ServiceResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Duration    int32     `json:"duration"`
	BufferTime  int32     `json:"bufferTime"`
	Cost        string    `json:"cost"`
	IsVisible   bool      `json:"isVisible"`
	ImageUrl    string    `json:"imageUrl"`
	BrandID     int32     `json:"brandId"`
	Providers   []int64   `json:"providers"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateServicePayload struct {
	Title       string  `json:"title" validate:"required,min=3,max=100"`
	Description string  `json:"description"`
	Duration    int32   `json:"duration" validate:"required,gt=0"`
	BufferTime  int32   `json:"buffer_time"`
	Cost        string  `json:"cost"`
	IsVisible   bool    `json:"is_visible"`
	ImageURL    string  `json:"image_url"`
	UserIDs     []int64 `json:"user_ids"`
}

// @Summary		Create a new service
// @Description	Creates a new service for a brand and assigns it to specified providers
// @Tags			service
// @Accept			json
// @Produce		json
// @Security		ApiKeyAuth
// @Param			payload	body		CreateServicePayload	true	"Service creation data"
// @Success		201		{object}	ServiceResponse			"Created service"
// @Failure		400		{object}	error					"Bad request - Invalid input"
// @Failure		401		{object}	error					"Unauthorized - Invalid or missing token"
// @Failure		403		{object}	error					"Forbidden - User does not belong to a brand"
// @Failure		404		{object}	error					"Not found - One or more providers not found"
// @Failure		500		{object}	error					"Internal server error"
// @Router			/service [post]
func (app *application) createServiceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)
	ctxUserBrandId := ctxUser.BrandID.Int32

	if ctxUserBrandId == 0 {
		app.forbiddenResponse(w, r, errors.New("unauthorized"))
		return
	}

	var payload CreateServicePayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	result, err := app.store.CreateServiceTx(ctx, store.CreateServiceTxParams{
		Title:       payload.Title,
		Description: payload.Description,
		Duration:    payload.Duration,
		BufferTime:  payload.BufferTime,
		Cost:        payload.Cost,
		IsVisible:   payload.IsVisible,
		ImageURL:    payload.ImageURL,
		BrandID:     ctxUserBrandId,
		UserIDs:     payload.UserIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidUserIDs):
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	response := serviceResponseMapper(result.Service, result.Providers)
	if err := writeJSON(w, http.StatusCreated, response); err != nil {
		app.internalServerError(w, r, err)
	}
}
