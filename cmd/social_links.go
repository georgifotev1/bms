package main

import (
	"net/http"
	"strconv"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-chi/chi/v5"
)

// @Summary		Update brand social links
// @Description	Update the social media links for a brand
// @Tags			brand
// @Accept			json
// @Produce		json
// @Security		CookieAuth
// @Param			payload	body		UpdateBrandSocialLinksPayload	true	"Social links data"
// @Param			id		path		int								true	"Brand ID"
// @Success		200		{object}	store.BrandResponse				"Updated brand with social links"
// @Failure		400		{object}	error							"Bad request - Invalid input"
// @Failure		401		{object}	error							"Unauthorized - Invalid or missing token"
// @Failure		500		{object}	error							"Internal server error"
// @Router			/brand/{id}/social-links [put]
func (app *application) updateBrandSocialLinksHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	brandId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload UpdateBrandSocialLinksPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	brand, err := app.store.GetBrandById(ctx, int32(brandId))
	if err != nil {
		if err.Error() == "brand does not exist" {
			app.badRequestResponse(w, r, err)
		} else {
			app.internalServerError(w, r, err)
		}
		return
	}

	var socialLinks []*store.BrandSocialLink
	socialLinkParams := payload.ToSocialLinkParams(brand.ID)

	for _, sl := range socialLinkParams {
		if updatedSL, err := app.store.UpsertBrandSocialLink(ctx, sl); err != nil {
			app.internalServerError(w, r, err)
			return
		} else {
			socialLinks = append(socialLinks, updatedSL)
		}
	}

	app.cache.Brands.Delete(ctx, brand.ID)
	br, err := app.getBrand(ctx, brand.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, br); err != nil {
		app.internalServerError(w, r, err)
	}
}
