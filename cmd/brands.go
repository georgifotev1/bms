package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/georgifotev1/bms/internal/store"
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
//
// @Security		ApiKeyAuth
//
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

// Get brand profile helper/mapper
func (app *application) getBrandProfile(ctx context.Context, brandID int32) (*store.BrandResponse, error) {
	profile, err := app.store.GetBrandProfile(ctx, brandID)
	if err != nil {
		return nil, err
	}

	var wh []store.WorkingHour
	err = json.Unmarshal(profile.WorkingHours.([]byte), &wh)
	if err != nil {
		return nil, err
	}

	var sl []store.SocialLink
	err = json.Unmarshal(profile.SocialLinks.([]byte), &sl)
	if err != nil {
		return nil, err
	}

	br := brandResponseMapper(&profile.Brand, sl, wh)
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
