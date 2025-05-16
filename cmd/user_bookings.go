package main

import (
	"net/http"
)

// createBookingHandler creates a new booking in the system
//
//	@Summary		Create a new booking
//	@Description	Creates a new booking with validation for timeslot availability
//	@Tags			bookings
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
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

	app.createBooking(w, r, payload)
}
