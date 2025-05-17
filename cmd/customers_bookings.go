package main

import (
	"net/http"
)

// createCustomerBookingHandler creates a new booking in the system
//
//	@Summary		Create a new custostomer booking
//	@Description	Creates a new booking with validation for timeslot availability
//	@Tags			bookings
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload		body		CreateBookingPayload	true	"Booking details"
//	@Param			X-Brand-ID	header		string					false	"Brand ID header for development. In production this header is ignored"	default(1)
//	@Success		201			{object}	BookingResponse			"Booking created successfully"
//	@Failure		400			{object}	error					"Bad request - invalid input"
//	@Failure		409			{object}	error					"Conflict - timeslot already booked"
//	@Failure		500			{object}	error					"Internal server error"
//	@Router			/customers/bookings [post]
func (app *application) createCustomerBookingHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateBookingPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	app.createBooking(w, r, payload)
}
