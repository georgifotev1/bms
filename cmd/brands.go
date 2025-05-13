package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/georgifotev1/bms/internal/store"
)

func (app *application) getBrand(ctx context.Context, brandID int32) (*store.GetBrandProfileRow, error) {
	if !app.config.cache.enabled {
		return app.store.GetBrandProfile(ctx, brandID)
	}

	brand, err := app.cache.Brands.Get(ctx, int64(brandID))
	if err != nil {
		return nil, err
	}

	if brand == nil {
		brand, err = app.store.GetBrandProfile(ctx, brandID)
		if err != nil {
			return nil, err
		}

		if err := app.cache.Brands.Set(ctx, brand); err != nil {
			return nil, err
		}
	}

	return brand, nil
}

type BrandResponse struct {
	ID           int32                     `json:"id"`
	Name         string                    `json:"name"`
	PageUrl      string                    `json:"pageUrl"`
	Description  string                    `json:"description"`
	Email        string                    `json:"email"`
	Phone        string                    `json:"phone"`
	Country      string                    `json:"country"`
	State        string                    `json:"state"`
	ZipCode      string                    `json:"zipCode"`
	City         string                    `json:"city"`
	Address      string                    `json:"address"`
	LogoUrl      string                    `json:"logoUrl"`
	BannerUrl    string                    `json:"bannerUrl"`
	Currency     string                    `json:"currency"`
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
	SocialLinks  []*store.BrandSocialLink  `json:"socialLinks"`
	WorkingHours []*store.BrandWorkingHour `json:"workingHours"`
}

type CreateBrandPayload struct {
	Name string `json:"name" validate:"required,min=3,max=100"`
}

// @Summary		Create a new brand
// @Description	Creates a new brand and associates it with the owner user
// @Tags			brand
// @Accept			json
// @Produce		json
//
//	@Security		ApiKeyAuth
//
// @Param			payload	body		CreateBrandPayload	true	"Brand creation data"
// @Success		201		{object}	BrandResponse		"Created brand"
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

	brand, err := app.store.CreateBrand(ctx, store.CreateBrandParams{
		Name:    payload.Name,
		PageUrl: pageUrl,
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	err = app.store.AssociateUserWithBrand(ctx, store.AssociateUserWithBrandParams{
		BrandID: sql.NullInt32{
			Valid: true,
			Int32: brand.ID,
		},
		ID: ctxUser.ID,
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
