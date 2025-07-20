package main

import (
	"net/http"
	"strconv"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-chi/chi/v5"
)

// @Summary		Update brand working hours
// @Description	Update the working hours for a brand
// @Tags			brand
// @Accept			json
// @Produce		json
// @Security		CookieAuth
// @Param			payload	body		UpdateBrandWorkingHoursPayload	true	"Working hours data"
// @Param			id		path		int								true	"Brand ID"
// @Success		200		{object}	store.BrandResponse				"Updated brand with working hours"
// @Failure		400		{object}	error							"Bad request - Invalid input"
// @Failure		401		{object}	error							"Unauthorized - Invalid or missing token"
// @Failure		500		{object}	error							"Internal server error"
// @Router			/brand/{id}/working-hours [put]
func (app *application) updateBrandWorkingHoursHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	brandId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload UpdateBrandWorkingHoursPayload
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

	var workingHours []*store.BrandWorkingHour
	workingHoursParams := payload.ToWorkingHoursParams(brand.ID)

	for _, wh := range workingHoursParams {
		if updatedWH, err := app.store.UpsertBrandWorkingHours(ctx, wh); err != nil {
			app.internalServerError(w, r, err)
			return
		} else {
			workingHours = append(workingHours, updatedWH)
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
