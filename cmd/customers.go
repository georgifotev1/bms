package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"golang.org/x/crypto/bcrypt"
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
	BrandId     int32  `json:"brandId" validate:"required,min=1"`
	PhoneNumber string `json:"phoneNumber"`
}

// registerCustomerHandler godoc
//
//	@Summary		Registers a customer
//	@Description	Registers a customer
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RegisterCustomerPayload	true	"User credentials"
//	@Success		201		{object}	CustomerResponse		"User registered"
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
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
	_, err = app.getBrand(ctx, payload.BrandId) // TODO: remove when create middleware
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.unauthorizedErrorResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	customer, err := app.store.CreateCustomer(ctx, store.CreateCustomerParams{
		Name:     payload.Username,
		Email:    payload.Email,
		Password: hashedPass,
		BrandID:  payload.BrandId,
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
	BrandId  int32  `json:"brandId" validate:"required,min=1"`
}

// loginCustomerHandler godoc
//
//	@Summary		Login a customer
//	@Description	Login a customer
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		LoginCustomerPayload	true	"User credentials"
//	@Success		201		{object}	CustomerResponse		"User logged in"
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
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

	ctx := r.Context()
	_, err := app.getBrand(ctx, payload.BrandId)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			app.unauthorizedErrorResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	customer, err := app.store.GetCustomerByEmail(ctx, payload.Email)
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
