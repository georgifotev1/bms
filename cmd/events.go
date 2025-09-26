package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	dateLayout = "2006-01-02"
)

type CreateEventPayload struct {
	CustomerID int64     `json:"customerId" validate:"required,min=0"`
	ServiceID  uuid.UUID `json:"serviceId" validate:"required"`
	UserID     int64     `json:"userId" validate:"required,min=0"`
	BrandID    int32     `json:"brandId" validate:"required,min=0"`
	StartTime  time.Time `json:"startTime" validate:"required,gt=now"`
	EndTime    time.Time `json:"endTime" validate:"required,gtfield=StartTime"`
	Comment    string    `json:"comment"`
}

type EventResponse struct {
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
	BufferTime   int32     `json:"bufferTime"`
	Cost         string    `json:"cost"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type EventValidationParams struct {
	UserID     int64
	ServiceID  uuid.UUID
	CustomerID int64
	BrandID    int32
	StartTime  time.Time
	EndTime    time.Time
	Comment    string
}

type EventEntities struct {
	User     *store.User
	Customer *store.Customer
	Service  *store.Service
}

// createEventHandler creates a new event in the system
//
//	@Summary		Create a new event
//	@Description	Creates a new event with validation for timeslot availability
//	@Tags			events
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CreateEventPayload	true	"Event details"
//	@Success		201		{object}	EventResponse		"Event created successfully"
//	@Failure		400		{object}	error				"Bad request - invalid input"
//	@Failure		409		{object}	error				"Conflict - timeslot already booked"
//	@Failure		500		{object}	error				"Internal server error"
//	@Router			/events [post]
func (app *application) createEventHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateEventPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		validationError := handleValidationErrors(err)
		app.badRequestResponse(w, r, errors.New(validationError.Message))
		return
	}

	ctx := r.Context()
	validationParams := EventValidationParams{
		UserID:     payload.UserID,
		ServiceID:  payload.ServiceID,
		CustomerID: payload.CustomerID,
		BrandID:    payload.BrandID,
		StartTime:  payload.StartTime,
		EndTime:    payload.EndTime,
		Comment:    payload.Comment,
	}

	entities, err := app.validateEventEntities(ctx, validationParams)
	if err != nil {
		app.hadleEventValidationError(w, r, err)
		return
	}

	event, err := app.store.CreateEvent(ctx, store.CreateEventParams{
		CustomerID:   payload.CustomerID,
		ServiceID:    payload.ServiceID,
		UserID:       payload.UserID,
		BrandID:      payload.BrandID,
		StartTime:    payload.StartTime.UTC(),
		EndTime:      payload.EndTime.UTC(),
		Comment:      toNullString(payload.Comment),
		CustomerName: entities.Customer.Name,
		UserName:     entities.User.Name,
		Cost:         entities.Service.Cost,
		BufferTime:   entities.Service.BufferTime,
		ServiceName:  entities.Service.Title,
	})

	if err != nil {
		if app.handleEventDatabaseError(w, r, err) {
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if err = writeJSON(w, http.StatusCreated, eventResponseMapper(event)); err != nil {
		app.internalServerError(w, r, err)
	}
}

// updateEventHandler update existing event in the system
//
//	@Summary		Update an event
//	@Description	Updates an event with validation for timeslot availability
//	@Tags			events
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			payload	body		CreateEventPayload	true	"Event details"
//	@Param			eventId	path		int					true	"Event ID"
//	@Success		200		{object}	EventResponse		"Event updated successfully"
//	@Failure		400		{object}	error				"Bad request - invalid input"
//	@Failure		409		{object}	error				"Invalid timeslot"
//	@Failure		500		{object}	error				"Internal server error"
//	@Router			/events/{eventId} [put]
func (app *application) updateEventHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "eventId")
	if id == "" {
		app.badRequestResponse(w, r, errors.New("invalid event id"))
		return
	}

	eventId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var payload CreateEventPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()

	validationParams := EventValidationParams{
		UserID:     payload.UserID,
		ServiceID:  payload.ServiceID,
		CustomerID: payload.CustomerID,
		BrandID:    payload.BrandID,
		StartTime:  payload.StartTime,
		EndTime:    payload.EndTime,
		Comment:    payload.Comment,
	}

	entities, err := app.validateEventEntities(ctx, validationParams)
	if err != nil {
		app.hadleEventValidationError(w, r, err)
		return
	}

	updatedEvent, err := app.store.UpdateEvent(ctx, store.UpdateEventParams{
		ID:           eventId,
		CustomerID:   payload.CustomerID,
		ServiceID:    payload.ServiceID,
		UserID:       payload.UserID,
		BrandID:      payload.BrandID,
		StartTime:    payload.StartTime.UTC(),
		EndTime:      payload.EndTime.UTC(),
		Comment:      toNullString(payload.Comment),
		CustomerName: entities.Customer.Name,
		UserName:     entities.User.Name,
		ServiceName:  entities.Service.Title,
		Cost:         entities.Service.Cost,
		BufferTime:   entities.Service.BufferTime,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			app.notFoundResponse(w, r, err)
			return
		}
		if app.handleEventDatabaseError(w, r, err) {
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if err = writeJSON(w, http.StatusOK, eventResponseMapper(updatedEvent)); err != nil {
		app.internalServerError(w, r, err)
	}
}

// Helper function to handle common events retrieval logic
func (app *application) handleEventsByTimeStampRetrieval(w http.ResponseWriter, r *http.Request, brandID int32) {
	ctx := r.Context()

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

	events, err := app.store.GetEventsByWeek(ctx, store.GetEventsByWeekParams{
		StartDate: startDateTime,
		EndDate:   endDateTime,
		BrandID:   brandID,
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var result []EventResponse
	for _, v := range events {
		result = append(result, eventResponseMapper(v))
	}
	if len(result) == 0 {
		result = []EventResponse{}
	}

	if err = writeJSON(w, http.StatusOK, result); err != nil {
		app.internalServerError(w, r, err)
	}
}

// getEventsByWeekHandler List all events of a brand in a specific timestamp
//
//	@Summary		List all events of a brand in a specific timestamp
//	@Description	List all events of a brand in a specific timestamp and validate the user input
//	@Tags			events
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			startDate	query		string			true	"Start date in YYYY-MM-DD format"	example(2025-05-19)
//	@Param			endDate		query		string			true	"End date in YYYY-MM-DD format"		example(2025-05-20)
//	@Param			brandId		query		integer			true	"Brand ID"							minimum(1)	example(1)
//	@Success		200			{array}		[]EventResponse	"List of brands"
//	@Failure		400			{object}	error			"Bad request - invalid input"
//	@Failure		409			{object}	error			"Conflict - timeslot already booked"
//	@Failure		500			{object}	error			"Internal server error"
//	@Router			/events/timestamp [get]
func (app *application) getEventsByTimeStampHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := ctx.Value(userCtx).(*store.User)
	app.handleEventsByTimeStampRetrieval(w, r, ctxUser.BrandID.Int32)
}

// getEventsByWeekPublicHandler List all events of a brand in a specific timestamp (public)
//
//	@Summary		List all events of a brand in a specific timestamp (public)
//	@Description	List all events of a brand in a specific timestamp and validate the user input for public access
//	@Tags			events
//	@Accept			json
//	@Produce		json
//	@Param			startDate	query		string			true	"Start date in YYYY-MM-DD format"	example(2025-05-19)
//	@Param			endDate		query		string			true	"End date in YYYY-MM-DD format"		example(2025-05-20)
//	@Success		200			{array}		[]EventResponse	"List of brands"
//	@Failure		400			{object}	error			"Bad request - invalid input"
//	@Failure		409			{object}	error			"Conflict - timeslot already booked"
//	@Failure		500			{object}	error			"Internal server error"
//	@Router			/events/timestamp/public [get]
func (app *application) getEventsByTimeStampPublicHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	brandID := getBrandIDFromCtx(ctx)
	app.handleEventsByTimeStampRetrieval(w, r, brandID)
}

func (app *application) validateEventEntities(ctx context.Context, params EventValidationParams) (*EventEntities, error) {
	availabilityParams := store.CheckSpecificTimeslotAvailabilityParams{
		UserID:    params.UserID,
		ServiceID: params.ServiceID,
		StartTime: params.StartTime.UTC(),
		EndTime:   params.EndTime.UTC(),
	}

	isAvailable, err := app.store.CheckSpecificTimeslotAvailability(ctx, availabilityParams)
	if err != nil {
		fmt.Println("HERE IS THE ERROR!")
		return nil, fmt.Errorf("checking timeslot availability: %w", err)
	}
	if isAvailable == false {
		return nil, ErrTimeslotNotAvailable
	}

	var (
		user                             *store.User
		customer                         *store.Customer
		service                          *store.Service
		userErr, customerErr, serviceErr error
		wg                               sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		user, userErr = app.getUser(ctx, params.UserID)
	}()

	go func() {
		defer wg.Done()
		customer, customerErr = app.getCustomer(ctx, params.CustomerID)
	}()

	go func() {
		defer wg.Done()
		service, serviceErr = app.store.GetService(ctx, params.ServiceID)
	}()

	wg.Wait()

	if userErr != nil {
		return nil, fmt.Errorf("error getting user: %w", userErr)
	}
	if customerErr != nil {
		return nil, fmt.Errorf("error getting customer: %w", customerErr)
	}
	if serviceErr != nil {
		if errors.Is(serviceErr, sql.ErrNoRows) {
			return nil, ErrServiceNotFound
		}
		return nil, fmt.Errorf("error getting service: %w", serviceErr)
	}

	return &EventEntities{
		User:     user,
		Customer: customer,
		Service:  service,
	}, nil
}

func (app *application) handleEventDatabaseError(w http.ResponseWriter, r *http.Request, err error) bool {
	pgError, ok := err.(*pq.Error)
	if !ok || !isPgError(err, foreignKeyViolation) {
		return false
	}

	switch {
	case strings.Contains(pgError.Message, "events_customer_id_fkey"):
		app.badRequestResponse(w, r, errors.New("invalid customer"))
	case strings.Contains(pgError.Message, "events_service_id_fkey"):
		app.badRequestResponse(w, r, errors.New("invalid service"))
	case strings.Contains(pgError.Message, "events_user_id_fkey"):
		app.badRequestResponse(w, r, errors.New("invalid user"))
	case strings.Contains(pgError.Message, "events_brand_id_fkey"):
		app.badRequestResponse(w, r, errors.New("invalid brand"))
	default:
		app.badRequestResponse(w, r, errors.New("one or more referenced entities not found"))
	}
	return true
}
