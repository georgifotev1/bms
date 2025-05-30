package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	dateLayout = "2006-01-02"
)

type CreateBookingPayload struct {
	CustomerID int64     `json:"customerId" validate:"required,min=0"`
	ServiceID  uuid.UUID `json:"serviceId" validate:"required"`
	UserID     int64     `json:"userId" validate:"required,min=0"`
	BrandID    int32     `json:"brandId" validate:"required,min=0"`
	StartTime  time.Time `json:"startTime" validate:"required,gt=now"`
	EndTime    time.Time `json:"endTime" validate:"required,gtfield=StartTime"`
	Comment    string    `json:"comment"`
}

type BookingResponse struct {
	ID           int64     `json:"id"`
	CustomerID   int64     `json:"customerId"`
	ServiceID    uuid.UUID `json:"serviceId"`
	UserID       int64     `json:"userId"`
	BrandID      int32     `json:"brandId"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	CustomerName string    `json:"customerName"`
	ServiceName  string    `json:"serviceName"`
	UserName     string    `json:"userName"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// createBookingHandler creates a new booking in the system
//
//	@Summary		Create a new booking
//	@Description	Creates a new booking with validation for timeslot availability
//	@Tags			bookings
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CreateBookingPayload	true	"Booking details"
//	@Success		201		{object}	BookingResponse			"Booking created successfully"
//	@Failure		400		{object}	error					"Bad request - invalid input"
//	@Failure		409		{object}	error					"Conflict - timeslot already booked"
//	@Failure		500		{object}	error					"Internal server error"
//	@Router			/bookings [post]
func (app *application) createBookingHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateBookingPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	ctx := r.Context()
	isAvailable, err := app.store.CheckSpecificTimeslotAvailability(ctx, store.CheckSpecificTimeslotAvailabilityParams{
		UserID:    payload.UserID,
		ServiceID: payload.ServiceID,
		StartTime: payload.StartTime.UTC(),
		EndTime:   payload.EndTime.UTC(),
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if isAvailable == false {
		app.conflictRespone(w, r, errors.New("The requested timeslot is not available for booking"))
		return
	}

	user, err := app.getUser(ctx, payload.UserID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	customer, err := app.getCustomer(ctx, payload.CustomerID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	service, err := app.store.GetService(ctx, payload.ServiceID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	booking, err := app.store.CreateBooking(ctx, store.CreateBookingParams{
		CustomerID: payload.CustomerID,
		ServiceID:  payload.ServiceID,
		UserID:     payload.UserID,
		BrandID:    payload.BrandID,
		StartTime:  payload.StartTime.UTC(),
		EndTime:    payload.EndTime.UTC(),
		Comment: sql.NullString{
			String: payload.Comment,
			Valid:  payload.Comment != "",
		},
		CustomerName: customer.Name,
		UserName:     user.Name,
		ServiceName:  service.Title,
	})

	if err != nil {
		if pgError, ok := err.(*pq.Error); ok {
			if isPgError(err, foreignKeyViolation) {
				switch {
				case strings.Contains(pgError.Message, "bookings_customer_id_fkey"):
					app.badRequestResponse(w, r, errors.New("invalid customer"))
				case strings.Contains(pgError.Message, "bookings_service_id_fkey"):
					app.badRequestResponse(w, r, errors.New("invalid service"))
				case strings.Contains(pgError.Message, "bookings_user_id_fkey"):
					app.badRequestResponse(w, r, errors.New("invalid user"))
				case strings.Contains(pgError.Message, "bookings_brand_id_fkey"):
					app.badRequestResponse(w, r, errors.New("invalid brand"))
				default:
					app.badRequestResponse(w, r, errors.New("one or more referenced entities not found"))
				}
				return
			}
		}
		app.internalServerError(w, r, err)
		return
	}

	if err = writeJSON(w, http.StatusCreated, bookingResponseMapper(booking)); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) getBookingsByDayHandler(w http.ResponseWriter, r *http.Request) {}

// getBookingsByWeekHandler List all bookings of a brand in a specific week
//
//	@Summary		List all bookings of a brand in a specific week
//	@Description	List all bookings of a brand in a specific week and validate the user input
//	@Tags			bookings
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			startDate	query		string				true	"Start date in YYYY-MM-DD format"	example(2025-05-19)
//	@Param			endDate		query		string				true	"End date in YYYY-MM-DD format"		example(2025-05-20)
//	@Param			brandId		query		integer				true	"Brand ID"							minimum(1)	example(1)
//	@Success		200			{array}		[]BookingResponse	"List of brands"
//	@Failure		400			{object}	error				"Bad request - invalid input"
//	@Failure		409			{object}	error				"Conflict - timeslot already booked"
//	@Failure		500			{object}	error				"Internal server error"
//	@Router			/bookings/week [get]
func (app *application) getBookingsByWeekHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	startDateTime, err := time.Parse(dateLayout, startDate)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("Invalid startDate format. Must be YYYY-MM-DD"))
		return
	}

	endDateTime, err := time.Parse(dateLayout, endDate)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("Invalid endDate format. Must be YYYY-MM-DD"))
		return
	}

	bookings, err := app.store.GetBookingsByWeek(r.Context(), store.GetBookingsByWeekParams{
		StartDate: startDateTime,
		EndDate:   endDateTime,
		BrandID:   ctxUser.BrandID.Int32,
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var result []BookingResponse
	for _, v := range bookings {
		result = append(result, bookingResponseMapper(v))
	}

	if len(result) == 0 {
		result = []BookingResponse{}
	}

	if err = writeJSON(w, http.StatusOK, result); err != nil {
		app.internalServerError(w, r, err)
	}
}
