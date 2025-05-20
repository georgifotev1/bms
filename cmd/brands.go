package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-chi/chi/v5"
)

type brandKey string

const (
	brandIDCtx brandKey = "brand"
)

type CreateBrandPayload struct {
	Name string `json:"name" validate:"required,min=3,max=100"`
}

// @Summary		Create a new brand
// @Description	Creates a new brand and associates it with the owner user
// @Tags			brand
// @Accept			json
// @Produce		json
// @Security		ApiKeyAuth
// @Param			payload	body		CreateBrandPayload	true	"Brand creation data"
// @Success		201		{object}	store.BrandResponse	"Created brand"
// @Failure		400		{object}	error				"Bad request - Invalid input"
// @Failure		401		{object}	error				"Unauthorized - Invalid or missing token"
// @Failure		403		{object}	error				"Forbidden - User is not an owner"
// @Failure		409		{object}	error				"Conflict - Brand already exists"
// @Failure		500		{object}	error				"Internal server error"
// @Router			/brand [post]
func (app *application) createBrandHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)

	if ctxUser.Role != ownerRole {
		app.forbiddenResponse(w, r, ErrAccessDenied)
		return
	}

	if ctxUser.BrandID.Int32 != 0 {
		app.conflictRespone(w, r, errors.New("brand is already created"))
		return
	}

	var payload CreateBrandPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	pageUrl := strings.ToLower(strings.ReplaceAll(payload.Name, " ", ""))

	_, err := app.store.GetBrandByUrl(ctx, pageUrl)
	if err != nil {
		switch {
		case isPgError(err, uniqueViolation):
			pageUrl = pageUrl + generateSubstring(4)
		case err == sql.ErrNoRows:
			break
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	brand, err := app.store.CreateBrandTx(ctx, store.CreateBrandTxParams{
		Name:    payload.Name,
		PageUrl: pageUrl,
		UserID:  ctxUser.ID,
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	brandResponse := brandResponseMapper(brand, nil, nil)
	if err := writeJSON(w, http.StatusCreated, brandResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

// @Summary		Get brand by ID
// @Description	Retrieves a brand's details by its unique ID
// @Tags			brand
// @Produce		json
// @Param			id	path		int					true	"Brand ID"
// @Success		200	{object}	store.BrandResponse	"Brand details"
// @Failure		400	{object}	error				"Bad request - Invalid brand ID"
// @Failure		500	{object}	error				"Internal server error"
// @Router			/brand/{id} [get]
func (app *application) getBrandHandler(w http.ResponseWriter, r *http.Request) {
	brandId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	brandResponse, err := app.getBrand(r.Context(), int32(brandId))
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.badRequestResponse(w, r, errors.New("brand does not exist"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := writeJSON(w, http.StatusOK, brandResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

// Get brand profile helper/mapper
func (app *application) getBrandProfile(ctx context.Context, brandID int32) (*store.BrandResponse, error) {
	brand, sl, wh, err := app.store.GetBrandProfileTx(ctx, brandID)
	if err != nil {
		return nil, err
	}

	br := brandResponseMapper(brand, sl, wh)
	return &br, nil
}

// Try to get brand from cache if enabled
func (app *application) getBrand(ctx context.Context, brandID int32) (*store.BrandResponse, error) {
	if !app.config.cache.enabled {
		return app.getBrandProfile(ctx, brandID)
	}

	brand, err := app.cache.Brands.Get(ctx, brandID)
	if err != nil {
		return nil, err
	}

	if brand == nil {
		brand, err = app.getBrandProfile(ctx, brandID)
		if err != nil {
			return nil, err
		}

		if err := app.cache.Brands.Set(ctx, brand); err != nil {
			return nil, err
		}
	}

	return brand, nil
}
