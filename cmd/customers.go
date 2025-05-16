package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/auth"
	"github.com/georgifotev1/bms/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type customerKey string

const (
	customerIdCtx customerKey = "customer"
)

type CustomerResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	BrandId     int32     `json:"brandId"`
	PhoneNumber string    `json:"phoneNumber"`
	Token       string    `json:"token"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type RegisterCustomerPayload struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=3,max=72"`
	Username    string `json:"username" validate:"required,min=2,max=100"`
	PhoneNumber string `json:"phoneNumber"`
}

// registerCustomerHandler godoc
//
//	@Summary		Registers a customer
//	@Description	Registers a customer
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			payload		body		RegisterCustomerPayload	true	"customer credentials"
//	@Param			X-Brand-ID	header		string					false	"Brand ID header for development. In production this header is ignored"	default(1)
//	@Success		201			{object}	CustomerResponse		"customer registered"
//	@Failure		400			{object}	error
//	@Failure		500			{object}	error
//	@Router			/customers/auth/register [post]
func (app *application) registerCustomerHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterCustomerPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	ctx := r.Context()
	ctxBrandID := getBrandIDFromCtx(ctx)
	customer, err := app.store.CreateCustomer(ctx, store.CreateCustomerParams{
		Name:     payload.Username,
		Email:    payload.Email,
		Password: hashedPass,
		BrandID:  ctxBrandID,
		PhoneNumber: sql.NullString{
			String: payload.PhoneNumber,
			Valid:  payload.PhoneNumber != "",
		},
	})
	if err != nil {
		switch {
		case isPgError(err, uniqueViolation):
			app.badRequestResponse(w, r, errors.New("customer already exists"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	accessToken, refreshToken, err := app.auth.GenerateTokens(customer.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if isBrowser(r) {
		app.SetCookie(w, CUSTOMER_REFRES_TOKEN, refreshToken)
	}

	customerResponse := customerResponseMapper(customer, accessToken)
	if err := writeJSON(w, http.StatusCreated, customerResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

type LoginCustomerPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=3,max=72"`
}

// loginCustomerHandler godoc
//
//	@Summary		Login a customer
//	@Description	Login a customer
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			payload		body		LoginCustomerPayload	true	"customer credentials"
//	@Param			X-Brand-ID	header		string					false	"Brand ID header for development. In production this header is ignored"	default(1)
//	@Success		201			{object}	CustomerResponse		"customer logged in"
//	@Failure		400			{object}	error
//	@Failure		500			{object}	error
//	@Router			/customers/auth/login [post]
func (app *application) loginCustomerHandler(w http.ResponseWriter, r *http.Request) {
	var payload LoginCustomerPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	customer, err := app.store.GetCustomerByEmail(r.Context(), payload.Email)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.unauthorizedErrorResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword(customer.Password, []byte(payload.Password))
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	accessToken, refreshToken, err := app.auth.GenerateTokens(customer.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if isBrowser(r) {
		app.SetCookie(w, CUSTOMER_REFRES_TOKEN, refreshToken)
	}

	customerResponse := customerResponseMapper(customer, accessToken)
	if err := writeJSON(w, http.StatusCreated, customerResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

// refreshCustomerTokenHandler godoc
//
//	@Summary		Refreshes the access token of customer
//	@Description	Uses a refresh token to generate a new access token
//	@Tags			customers
//	@Produce		json
//	@Success		200	{string}	string	"New access token"
//	@Failure		401	{object}	error
//	@Failure		500	{object}	error
//	@Router			/customers/auth/refresh [get]
func (app *application) refreshCustomerTokenHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(CUSTOMER_REFRES_TOKEN)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, errors.New("refresh token not found"))
		return
	}

	refreshToken := cookie.Value

	newAccessToken, newRefreshToken, err := app.auth.RefreshTokens(refreshToken)
	if err != nil {
		switch err.Error() {
		case auth.ErrTokenClaims, auth.ErrTokenType, auth.ErrTokenValidation:
			// if refresh token is invalid remove it
			app.ClearCookie(w, CUSTOMER_REFRES_TOKEN)
			app.unauthorizedErrorResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.SetCookie(w, CUSTOMER_REFRES_TOKEN, newRefreshToken)

	response := map[string]string{
		"token": newAccessToken,
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}

// logoutCustomerHandler godoc
//
//	@Summary		Logs out a customer
//	@Description	Clears the refresh token cookie to log out the customer
//	@Tags			customers
//	@Produce		json
//	@Success		200	{string}	string	"Logged out successfully"
//	@Failure		500	{object}	error
//	@Router			/customers/auth/logout [post]
func (app *application) logoutCustomerHandler(w http.ResponseWriter, r *http.Request) {
	app.ClearCookie(w, CUSTOMER_REFRES_TOKEN)

	response := map[string]string{
		"message": "Logged out successfully",
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) getCustomer(ctx context.Context, customerID int64) (*store.Customer, error) {
	if !app.config.cache.enabled {
		return app.store.GetCustomerById(ctx, customerID)
	}

	customer, err := app.cache.Customers.Get(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if customer == nil {
		customer, err = app.store.GetCustomerById(ctx, customerID)
		if err != nil {
			return nil, err
		}

		if err := app.cache.Customers.Set(ctx, customer); err != nil {
			return nil, err
		}
	}

	return customer, nil
}
