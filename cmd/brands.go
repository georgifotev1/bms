package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
// @Security		CookieAuth
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

	pageUrl, err := app.formatBrandUrl(ctx, payload.Name)
	if err != nil {
		app.internalServerError(w, r, err)
		return
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

	ctxUser.BrandID = sql.NullInt32{Int32: brand.ID, Valid: true}
	if err := app.cache.Users.Set(ctx, ctxUser); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	brandResponse := brandResponseMapper(brand, nil, nil)
	if err := writeJSON(w, http.StatusCreated, brandResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

type UpdateBrandPayload struct {
	Name         string                    `json:"name" validate:"required,min=3,max=100"`
	PageUrl      string                    `json:"pageUrl" validate:"required"`
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
	WorkingHours []UpdateBrandWorkingHours `json:"workingHours,omitempty"`
	SocialLinks  []UpdateBrandSocialLink   `json:"socialLinks,omitempty"`
}

type UpdateBrandWorkingHours struct {
	DayOfWeek int32     `json:"dayOfWeek" validate:"required,min=0,max=6"`
	OpenTime  time.Time `json:"openTime"`
	CloseTime time.Time `json:"closeTime"`
	IsClosed  bool      `json:"isClosed"`
}

type UpdateBrandSocialLink struct {
	Platform string `json:"platform" validate:"required"`
	Url      string `json:"url" validate:"required,url"`
}

func (p *UpdateBrandPayload) ToWorkingHoursParams(brandID int32) []store.UpsertBrandWorkingHoursParams {
	params := make([]store.UpsertBrandWorkingHoursParams, len(p.WorkingHours))
	for i, wh := range p.WorkingHours {
		params[i] = store.UpsertBrandWorkingHoursParams{
			BrandID:   brandID,
			DayOfWeek: wh.DayOfWeek,
			OpenTime: sql.NullTime{
				Valid: wh.OpenTime != time.Time{},
				Time:  wh.OpenTime,
			},
			CloseTime: sql.NullTime{
				Valid: wh.CloseTime != time.Time{},
				Time:  wh.CloseTime,
			},
			IsClosed: sql.NullBool{
				Valid: wh.IsClosed,
				Bool:  wh.IsClosed,
			},
		}
	}
	return params
}

func (p *UpdateBrandPayload) ToSocialLinkParams(brandID int32) []store.UpsertBrandSocialLinkParams {
	params := make([]store.UpsertBrandSocialLinkParams, len(p.SocialLinks))
	for i, sl := range p.SocialLinks {
		params[i] = store.UpsertBrandSocialLinkParams{
			Platform: sl.Platform,
			Url:      sl.Url,
			BrandID:  brandID,
		}
	}
	return params
}

// @Summary		Update brand
// @Description	Update the brand profile, working hours and social links
// @Tags			brand
// @Accept			json
// @Produce		json
// @Security		CookieAuth
// @Param			payload	body		UpdateBrandPayload	true	"Brand data"
// @Param			id		path		int					true	"Brand ID"
// @Success		201		{object}	store.BrandResponse	"Updated brand"
// @Failure		400		{object}	error				"Bad request - Invalid input"
// @Failure		401		{object}	error				"Unauthorized - Invalid or missing token"
// @Failure		500		{object}	error				"Internal server error"
// @Router			/brand/{id} [put]
func (app *application) updateBrandHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	brandId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	brand, err := app.getBrand(ctx, int32(brandId))
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.badRequestResponse(w, r, errors.New("brand does not exist"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	var payload UpdateBrandPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	pageUrl, err := app.formatBrandUrl(ctx, payload.PageUrl)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	updateParams := store.UpdateBrandParams{
		ID:      brand.ID,
		Name:    payload.Name,
		PageUrl: pageUrl,
		Description: sql.NullString{
			Valid:  payload.Description != "",
			String: payload.Description,
		},
		Email: sql.NullString{
			Valid:  payload.Email != "",
			String: payload.Email,
		},
		Phone: sql.NullString{
			Valid:  payload.Phone != "",
			String: payload.Phone,
		},
		Country: sql.NullString{
			Valid:  payload.Country != "",
			String: payload.Country,
		},
		State: sql.NullString{
			Valid:  payload.State != "",
			String: payload.State,
		},
		ZipCode: sql.NullString{
			Valid:  payload.ZipCode != "",
			String: payload.ZipCode,
		},
		City: sql.NullString{
			Valid:  payload.City != "",
			String: payload.City,
		},
		Address: sql.NullString{
			Valid:  payload.Address != "",
			String: payload.Address,
		},
		LogoUrl: sql.NullString{
			Valid:  payload.LogoUrl != "",
			String: payload.LogoUrl,
		},
		BannerUrl: sql.NullString{
			Valid:  payload.BannerUrl != "",
			String: payload.BannerUrl,
		},
		Currency: sql.NullString{
			Valid:  payload.Currency != "",
			String: payload.Currency,
		},
	}

	workingHoursParams := payload.ToWorkingHoursParams(brand.ID)
	socialLinkParams := payload.ToSocialLinkParams(brand.ID)

	updatedBrand, socialLinks, workingHours, err := app.store.UpdateBrandProfileTx(ctx, updateParams, workingHoursParams, socialLinkParams)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	brandResponse := brandResponseMapper(updatedBrand, socialLinks, workingHours)
	if err := writeJSON(w, http.StatusOK, brandResponse); err != nil {
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

func (app *application) formatBrandUrl(ctx context.Context, name string) (string, error) {
	pageUrl := strings.ToLower(strings.ReplaceAll(name, " ", ""))

	_, err := app.store.GetBrandByUrl(ctx, pageUrl)
	if err != nil {
		switch {
		case isPgError(err, uniqueViolation):
			pageUrl = pageUrl + generateSubstring(4)
		case err == sql.ErrNoRows:
			break
		default:
			return "", nil
		}
	}
	return pageUrl, nil
}
