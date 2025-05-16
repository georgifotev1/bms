package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
)

const (
	statusPending   = "pending"
	statusConfirmed = "confirmed"
	statusCompleted = "completed"
	statusCancelled = "cancelled"
	statusNoShow    = "no_show"
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
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customerId"`
	ServiceID  uuid.UUID `json:"serviceId"`
	UserID     int64     `json:"userId"`
	BrandID    int32     `json:"brandId"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	StatusID   int32     `json:"statusId"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (app *application) createBooking(w http.ResponseWriter, r *http.Request, payload CreateBookingPayload) {
	isAvailable, err := app.store.CheckSpecificTimeslotAvailability(r.Context(), store.CheckSpecificTimeslotAvailabilityParams{
		UserID:    payload.UserID,
		ServiceID: payload.ServiceID,
		StartTime: payload.StartTime,
		EndTime:   payload.EndTime,
	})

	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if isAvailable == false {
		app.conflictRespone(w, r, errors.New("The requested timeslot is not available for booking"))
		return
	}

	booking, err := app.store.CreateBooking(r.Context(), store.CreateBookingParams{
		CustomerID: payload.CustomerID,
		ServiceID:  payload.ServiceID,
		UserID:     payload.UserID,
		BrandID:    payload.BrandID,
		StartTime:  payload.StartTime,
		EndTime:    payload.EndTime,
		StatusName: statusPending,
		Comment: sql.NullString{
			String: payload.Comment,
			Valid:  payload.Comment != "",
		},
	})

	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err = writeJSON(w, http.StatusCreated, bookingResponseMapper(booking)); err != nil {
		app.internalServerError(w, r, err)
	}
}
