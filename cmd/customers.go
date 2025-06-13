package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
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
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SignUpCustomerPayload struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=3,max=72"`
	Name        string `json:"name" validate:"required,min=2,max=100"`
	PhoneNumber string `json:"phoneNumber" validate:"required"`
}

// registerCustomerHandler godoc
//
//	@Summary		Registers a customer
//	@Description	Registers a customer
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			payload		body		SignUpCustomerPayload	true	"customer credentials"
//	@Param			X-Brand-ID	header		string					false	"Brand ID header for development. In production this header is ignored"	default(1)
//	@Success		201			{object}	CustomerResponse		"customer registered"
//	@Failure		400			{object}	error
//	@Failure		500			{object}	error
//	@Router			/customers/auth/signup [post]
func (app *application) signUpCustomerHandler(w http.ResponseWriter, r *http.Request) {
	var payload SignUpCustomerPayload
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
		Name: payload.Name,
		Email: sql.NullString{
			String: payload.Email,
			Valid:  payload.Email != "",
		},
		Password:    hashedPass,
		BrandID:     ctxBrandID,
		PhoneNumber: payload.PhoneNumber,
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

	session, err := app.store.CreateCustomerSession(ctx, store.CreateCustomerSessionParams{
		CustomerID: customer.ID,
		ExpiresAt:  time.Now().UTC().Add(app.config.auth.session.exp),
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.SetCookie(w, CUSTOMER_SESSION_TOKEN, session.ID.String())

	customerResponse := customerResponseMapper(customer)
	if err := writeJSON(w, http.StatusCreated, customerResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

type SignInCustomerPayload struct {
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
//	@Param			payload		body		SignInCustomerPayload	true	"customer credentials"
//	@Param			X-Brand-ID	header		string					false	"Brand ID header for development. In production this header is ignored"	default(1)
//	@Success		201			{object}	CustomerResponse		"customer logged in"
//	@Failure		400			{object}	error
//	@Failure		500			{object}	error
//	@Router			/customers/auth/signin [post]
func (app *application) signInCustomerHandler(w http.ResponseWriter, r *http.Request) {
	var payload SignInCustomerPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()

	customer, err := app.store.GetCustomerByEmail(ctx, sql.NullString{
		String: payload.Email,
		Valid:  payload.Email != "",
	})
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

	var token uuid.UUID
	session, err := app.store.GetSessionByCustomerId(ctx, customer.ID)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			newSession, err := app.store.CreateCustomerSession(ctx, store.CreateCustomerSessionParams{
				CustomerID: customer.ID,
				ExpiresAt:  time.Now().UTC().Add(app.config.auth.session.exp),
			})
			if err != nil {
				app.internalServerError(w, r, err)
				return
			}
			token = newSession.ID
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	if session.ID != uuid.Nil {
		updatedSession, err := app.store.UpdateCustomerSession(ctx, store.UpdateCustomerSessionParams{
			ID:        session.ID,
			ExpiresAt: time.Now().UTC().Add(app.config.auth.session.exp),
		})
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}
		token = updatedSession.ID
	}

	app.SetCookie(w, CUSTOMER_SESSION_TOKEN, token.String())

	customerResponse := customerResponseMapper(customer)
	if err := writeJSON(w, http.StatusOK, customerResponse); err != nil {
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
	cookie, err := r.Cookie(CUSTOMER_SESSION_TOKEN)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	sessionId, err := uuid.Parse(cookie.Value)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	_, err = app.store.UpdateCustomerSession(r.Context(), store.UpdateCustomerSessionParams{
		ID:        sessionId,
		ExpiresAt: time.Unix(0, 0),
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.ClearCookie(w, CUSTOMER_SESSION_TOKEN)

	response := map[string]string{
		"message": "Logged out successfully",
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}

type CreateGuestCustomerPayload struct {
	Email       string `json:"email,omitempty" validate:"omitempty,email"`
	Name        string `json:"name" validate:"required,min=2,max=100"`
	PhoneNumber string `json:"phoneNumber" validate:"required"`
}

// createGuestCustomerHandler godoc
//
//	@Summary		Create or get a guest (customer without session)
//	@Description	Create or get a guest
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			payload		body		CreateGuestCustomerPayload	true	"guest credentials"
//	@Param			X-Brand-ID	header		string						false	"Brand ID header for development. In production this header is ignored"	default(1)
//	@Success		201			{object}	CustomerResponse			"guest created"
//	@Success		200			{object}	CustomerResponse			"guest already exists"
//	@Failure		400			{object}	error
//	@Failure		500			{object}	error
//	@Router			/customers/guest [post]
func (app *application) createGuestCustomerHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateGuestCustomerPayload

	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()
	ctxBrandID := getBrandIDFromCtx(ctx)

	customer, exist, err := app.store.CreateGuestTx(ctx, store.CreateGuestTxParams{
		Name:        payload.Name,
		Email:       payload.Email,
		PhoneNumber: payload.PhoneNumber,
		BrandId:     ctxBrandID,
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	status := http.StatusCreated
	if exist {
		status = http.StatusOK
	}

	customerResponse := customerResponseMapper(customer)
	if err := writeJSON(w, status, customerResponse); err != nil {
		app.internalServerError(w, r, err)
	}
}

// @Summary		Get customers by brand
// @Description	Fetches all customers of a brand
// @Tags			customers
// @Accept			json
// @Produce		json
// @Success		200	{object}	[]CustomerResponse
// @Failure		400	{object}	error
// @Failure		404	{object}	error
// @Failure		500	{object}	error
// @Security		CookieAuth
// @Router			/customers [get]
func (app *application) getCustomersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)
	if ctxUser.BrandID.Int32 == 0 || !ctxUser.BrandID.Valid {
		app.forbiddenResponse(w, r, errors.New("access denied"))
		return
	}

	customers, err := app.store.GetCustomersByBrand(ctx, ctxUser.BrandID.Int32)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var result []CustomerResponse
	for _, v := range customers {
		result = append(result, customersResponseMapper(v))
	}

	if err = writeJSON(w, http.StatusOK, result); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) getCustomer(ctx context.Context, customerID int64) (*store.Customer, error) {
	if !app.config.cache.enabled {
		customer, err := app.store.GetCustomerById(ctx, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrCustomerNotFound
			}
			return nil, err
		}
		return customer, nil
	}

	customer, err := app.cache.Customers.Get(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if customer == nil {
		customer, err = app.store.GetCustomerById(ctx, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrCustomerNotFound
			}
			return nil, err
		}

		if err := app.cache.Customers.Set(ctx, customer); err != nil {
			return nil, err
		}
	}

	return customer, nil
}
