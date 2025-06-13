package main

import (
	"database/sql"
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-chi/chi/v5"
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
	Title       string  `schema:"title" validate:"required,min=3,max=100"`
	Description string  `schema:"description"`
	Duration    int32   `schema:"duration" validate:"required,gt=0"`
	BufferTime  int32   `schema:"bufferTime"`
	Cost        string  `schema:"cost"`
	ImageURL    string  `schema:"imageUrl"`
	IsVisible   bool    `schema:"isVisible"`
	UserIDs     []int64 `schema:"userIds"`
}

type ImageInput struct {
	URL  string
	File *multipart.FileHeader
}

// @Summary		Create a new service
// @Description	Creates a new service for a brand and assigns it to specified providers
// @Tags			service
// @Accept			json
// @Produce		json
// @Security		CookieAuth
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

	err := r.ParseMultipartForm(20 << 20)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload CreateServicePayload
	if err := Decoder.Decode(&payload, r.PostForm); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if len(payload.UserIDs) < 1 {
		app.badRequestResponse(w, r, errors.New("At least one provider must be assigned to this service"))
		return
	}

	var imageURL string
	if file, _, err := r.FormFile("image"); err == nil {
		defer file.Close()

		uploadedURL, uploadErr := app.saveImageToCloudinary(file)
		if uploadErr != nil {
			app.badRequestResponse(w, r, uploadErr)
			return
		}
		imageURL = uploadedURL
	}

	result, err := app.store.CreateServiceTx(ctx, store.CreateServiceTxParams{
		Title:       payload.Title,
		Description: payload.Description,
		Duration:    payload.Duration,
		BufferTime:  payload.BufferTime,
		Cost:        payload.Cost,
		IsVisible:   payload.IsVisible,
		ImageURL:    imageURL,
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

// @Summary		Update a service
// @Description	Update a service and the users that can provide it
// @Tags			service
// @Accept			json
// @Produce		json
// @Security		CookieAuth
// @Param			payload		body		CreateServicePayload	true	"Service update data"
// @Param			serviceId	path		uuid.UUID				true	"service ID"
// @Success		201			{object}	ServiceResponse			"Updated service"
// @Failure		400			{object}	error					"Bad request - Invalid input"
// @Failure		401			{object}	error					"Unauthorized - Invalid or missing token"
// @Failure		403			{object}	error					"Forbidden - User does not belong to a brand"
// @Failure		404			{object}	error					"Not found - One or more providers not found"
// @Failure		500			{object}	error					"Internal server error"
// @Router			/service/id/{serviceId} [put]
func (app *application) updateServiceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)
	ctxUserBrandId := ctxUser.BrandID.Int32

	if ctxUserBrandId == 0 {
		app.forbiddenResponse(w, r, errors.New("unauthorized"))
		return
	}

	paramsServiceId := chi.URLParam(r, "serviceId")
	serviceId, err := uuid.Parse(paramsServiceId)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = r.ParseMultipartForm(20 << 20)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload CreateServicePayload
	if err := Decoder.Decode(&payload, r.PostForm); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if len(payload.UserIDs) < 1 {
		app.badRequestResponse(w, r, errors.New("At least one provider must be assigned to this service"))
		return
	}

	imageInput := &ImageInput{}

	if payload.ImageURL != "" {
		imageInput.URL = payload.ImageURL
	} else {
		if _, fileHeader, err := r.FormFile("image"); err == nil {
			imageInput.File = fileHeader
		}
	}

	imageURL, err := app.ProcessImage(imageInput)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	result, err := app.store.UpdateServiceTx(ctx, store.UpdateServiceTxParams{
		ID:          serviceId,
		Title:       payload.Title,
		Description: payload.Description,
		Duration:    payload.Duration,
		BufferTime:  payload.BufferTime,
		Cost:        payload.Cost,
		IsVisible:   payload.IsVisible,
		ImageURL:    imageURL,
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

// @Summary		Get services by brand
// @Description	Fetches all services of a brand
// @Tags			service
// @Accept			json
// @Produce		json
// @Param			brandId	path		int	true	"BrandId ID"
// @Success		200		{object}	[]ServiceResponse
// @Failure		400		{object}	error
// @Failure		404		{object}	error
// @Failure		500		{object}	error
// @Router			/service/{brandId} [get]
func (app *application) getServicesHandler(w http.ResponseWriter, r *http.Request) {
	brandIDStr := chi.URLParam(r, "brandId")
	if brandIDStr == "" {
		app.badRequestResponse(w, r, errors.New("brand ID is required"))
		return
	}

	brandId, err := strconv.Atoi(brandIDStr)
	if err != nil || brandId <= 0 {
		app.badRequestResponse(w, r, errors.New("invalid brand ID format"))
		return
	}

	ctx := r.Context()

	servicesWithProviders, err := app.store.ListServicesWithProviders(ctx, int32(brandId))
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			if err := writeJSON(w, http.StatusOK, []ServiceResponse{}); err != nil {
				app.internalServerError(w, r, err)
			}
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	serviceMap := make(map[uuid.UUID]*ServiceResponse)

	for _, row := range servicesWithProviders {
		serviceID := row.ID

		if _, exists := serviceMap[serviceID]; !exists {
			serviceMap[serviceID] = &ServiceResponse{
				ID:          row.ID,
				Title:       row.Title,
				Description: row.Description.String,
				Duration:    row.Duration,
				BufferTime:  row.BufferTime.Int32,
				Cost:        row.Cost.String,
				IsVisible:   row.IsVisible,
				ImageUrl:    row.ImageUrl.String,
				BrandID:     row.BrandID,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
				Providers:   []int64{},
			}
		}

		if row.ProviderID.Valid {
			serviceMap[serviceID].Providers = append(serviceMap[serviceID].Providers, row.ProviderID.Int64)
		}
	}

	result := make([]ServiceResponse, 0, len(serviceMap))
	for _, service := range serviceMap {
		result = append(result, *service)
	}

	if err := writeJSON(w, http.StatusOK, result); err != nil {
		app.internalServerError(w, r, err)
	}
}
