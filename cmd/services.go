package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
)

type ServiceResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Duration    int64     `json:"duration"`
	BufferTime  int64     `json:"bufferTime"`
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
	Duration    int64   `json:"duration" validate:"required,gt=0"`
	BufferTime  int64   `json:"buffer_time"`
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

	var newService *store.Service
	errInvalidUserIDs := errors.New("one or more user IDs are invalid or don't belong to your brand")

	err := app.store.ExecTx(ctx, func(q store.Querier) error {
		if len(payload.UserIDs) > 0 {
			count, err := q.ValidateUsersCount(ctx, store.ValidateUsersCountParams{
				Ids: payload.UserIDs,
				BrandID: sql.NullInt32{
					Int32: ctxUserBrandId,
					Valid: ctxUserBrandId != 0,
				},
			})
			if err != nil {
				return err
			}
			fmt.Println("count", count)
			fmt.Println(len(payload.UserIDs))
			if int(count) != len(payload.UserIDs) {
				return errInvalidUserIDs
			}
		}

		service, err := q.CreateService(ctx, store.CreateServiceParams{
			Title: payload.Title,
			Description: sql.NullString{
				String: payload.Description,
				Valid:  payload.Description != "",
			},
			Duration: payload.Duration,
			BufferTime: sql.NullInt64{
				Int64: payload.BufferTime,
				Valid: payload.BufferTime > 0,
			},
			Cost: sql.NullString{
				String: payload.Cost,
				Valid:  payload.Cost != "",
			},
			IsVisible: payload.IsVisible,
			ImageUrl: sql.NullString{
				String: payload.ImageURL,
				Valid:  payload.ImageURL != "",
			},
			BrandID: ctxUser.BrandID.Int32,
		})
		if err != nil {
			return err
		}

		for _, userId := range payload.UserIDs {
			err := q.AssignServiceToUser(ctx, store.AssignServiceToUserParams{
				ServiceID: service.ID,
				UserID:    userId,
			})
			if err != nil {
				return err
			}
		}

		newService = service
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errInvalidUserIDs):
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	response := serviceResponseMapper(newService, payload.UserIDs)
	if err := writeJSON(w, http.StatusCreated, response); err != nil {
		app.internalServerError(w, r, err)
	}
}
