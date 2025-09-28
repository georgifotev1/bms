package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/google/uuid"
)

// getAvailableTimeslotsHandler Get available timeslots for a service on a specific date
//
//	@Summary		Get available timeslots for a service on a specific date
//	@Description	Retrieves available time slots for a specific service on a given date for a selected staff member. This endpoint is public and requires brand context from middleware.
//	@Tags			timeslots
//	@Accept			json
//	@Produce		json
//	@Param			date		query		string		true	"Date in YYYY-MM-DD format"												example(2025-01-15)
//	@Param			serviceId	query		string		true	"Service ID (UUID)"														example(123e4567-e89b-12d3-a456-426614174000)
//	@Param			userId		query		string		true	"Staff member ID (UUID)"												example(987f6543-e21c-34d5-b678-123456789abc)
//	@Param			X-Brand-ID	header		string		false	"Brand ID header for development. In production this header is ignored"	default(1)
//	@Success		200			{array}		[]string	"List of available timeslots"
//	@Failure		400			{object}	error		"Bad request - Invalid date format, invalid service ID, or invalid user ID"
//	@Failure		500			{object}	error		"Internal server error"
//	@Router			/timeslots [get]
func (app *application) getAvailableTimeslotsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	brandId, err := getBrandIDFromCtx(ctx)
	if err != nil {
		app.badRequestResponse(w, r, err)
	}

	dateFromQuery := r.URL.Query().Get("date")
	date, err := time.Parse(dateLayout, dateFromQuery)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("Invalid date format. Must be YYYY-MM-DD"))
		return
	}

	if date.Before(time.Now().Truncate(24 * time.Hour)) {
		app.badRequestResponse(w, r, errors.New("Date must not be in the past"))
		return
	}

	serviceIdFromQuery := r.URL.Query().Get("serviceId")
	serviceId, err := uuid.Parse(serviceIdFromQuery)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("Invalid service ID"))
		return
	}

	userIdFromQuery := r.URL.Query().Get("userId")
	userId, err := strconv.ParseInt(userIdFromQuery, 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("Invalid user ID"))
		return
	}

	dayOfWeek := int32(date.Weekday()) // 0 = Sunday, 1 = Monday, etc.
	workingHours, err := app.store.GetBrandWorkingHours(ctx, brandId)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var dayWorkingHours *store.BrandWorkingHour
	for _, wh := range workingHours {
		if wh.DayOfWeek == dayOfWeek {
			dayWorkingHours = wh
			break
		}
	}

	if dayWorkingHours == nil || dayWorkingHours.IsClosed || !dayWorkingHours.OpenTime.Valid || !dayWorkingHours.CloseTime.Valid {
		if err := writeJSON(w, http.StatusOK, map[string]interface{}{"timeslots": []string{}}); err != nil {
			app.internalServerError(w, r, err)
		}
		return
	}

	// Get service details to check duration and buffer time
	service, err := app.store.GetService(ctx, serviceId)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Get user's existing events for the day
	userEvents, err := app.store.GetUserEventsByDay(ctx, store.GetUserEventsByDayParams{
		UserID:    userId,
		StartTime: date,
		BrandID:   brandId,
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Generate available timeslots
	timeslots := generateTimeslots(dayWorkingHours, service, userEvents, date)

	if err := writeJSON(w, http.StatusOK, map[string]interface{}{"timeslots": timeslots}); err != nil {
		app.internalServerError(w, r, err)
	}
}

func generateTimeslots(workingHours *store.BrandWorkingHour, service *store.Service, existingEvents []*store.Event, targetDate time.Time) []string {
	if !workingHours.OpenTime.Valid || !workingHours.CloseTime.Valid {
		return []string{}
	}

	year, month, day := targetDate.Date()
	location := targetDate.Location()

	openTime := workingHours.OpenTime.Time
	closeTime := workingHours.CloseTime.Time

	openDateTime := time.Date(year, month, day, openTime.Hour(), openTime.Minute(), 0, 0, location)
	closeDateTime := time.Date(year, month, day, closeTime.Hour(), closeTime.Minute(), 0, 0, location)

	fmt.Printf("Open time: %v\n", openDateTime)
	fmt.Printf("Close time: %v\n", closeDateTime)

	serviceDuration := time.Duration(service.Duration) * time.Minute
	bufferDuration := time.Duration(0)
	if service.BufferTime.Valid {
		bufferDuration = time.Duration(service.BufferTime.Int32) * time.Minute
	}
	totalSlotDuration := serviceDuration + bufferDuration

	var blockedPeriods []struct{ start, end time.Time }
	for _, event := range existingEvents {
		eventStart := event.StartTime
		eventEnd := event.EndTime

		if event.BufferTime.Valid {
			eventEnd = eventEnd.Add(time.Duration(event.BufferTime.Int32) * time.Minute)
		}

		blockedPeriods = append(blockedPeriods, struct{ start, end time.Time }{
			start: eventStart,
			end:   eventEnd,
		})

		fmt.Printf("Blocked period: %v to %v\n", eventStart.Format("15:04"), eventEnd.Format("15:04"))
	}

	slotInterval := 15 * time.Minute
	var availableSlots []string

	for currentSlotStart := openDateTime; currentSlotStart.Before(closeDateTime); currentSlotStart = currentSlotStart.Add(slotInterval) {
		currentSlotEnd := currentSlotStart.Add(totalSlotDuration)

		if currentSlotEnd.After(closeDateTime) {
			break
		}

		isAvailable := true
		for _, blocked := range blockedPeriods {
			if currentSlotStart.Before(blocked.end) && currentSlotEnd.After(blocked.start) {
				isAvailable = false
				break
			}
		}

		if isAvailable {
			availableSlots = append(availableSlots, currentSlotStart.Format("15:04"))
		}
	}

	return availableSlots
}
