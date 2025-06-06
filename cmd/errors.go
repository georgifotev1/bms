package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

var (
	ErrAccessDenied         = errors.New("access denied")
	ErrTimeslotNotAvailable = errors.New("The requested timeslot is not available for event")
	ErrUserNotFound         = errors.New("user not found")
	ErrCustomerNotFound     = errors.New("customer not found")
	ErrServiceNotFound      = errors.New("service not found")
)

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("internal server error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	writeJSONError(w, http.StatusInternalServerError, "the server encountered a problem")
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnf("bad request", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusBadRequest, err.Error())
}

func (app *application) forbiddenResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("forbidden", "method", r.Method, "path", r.URL.Path, "error")

	writeJSONError(w, http.StatusForbidden, err.Error())
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnf("not found error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusNotFound, "not found")
}

func (app *application) unauthorizedErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnf("unauthorized error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusUnauthorized, "unauthorized")
}

func (app *application) conflictRespone(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnf("confilct", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusConflict, err.Error())
}

func (app *application) unauthorizedBasicErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnf("unauthorized basic error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)

	writeJSONError(w, http.StatusUnauthorized, "unauthorized")
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request, retryAfter string) {
	app.logger.Warnw("rate limit exceeded", "method", r.Method, "path", r.URL.Path)

	w.Header().Set("Retry-After", retryAfter)

	writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded, retry after: "+retryAfter)
}

func isPgError(err error, code pq.ErrorCode) bool {
	pgErr, ok := err.(*pq.Error)
	return ok && pgErr.Code == code
}

func (app *application) hadleEventValidationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrTimeslotNotAvailable):
		app.conflictRespone(w, r, err)
	case errors.Is(err, ErrUserNotFound):
		app.badRequestResponse(w, r, err)
	case errors.Is(err, ErrCustomerNotFound):
		app.badRequestResponse(w, r, err)
	case errors.Is(err, ErrServiceNotFound):
		app.badRequestResponse(w, r, err)
	default:
		app.internalServerError(w, r, err)
	}
}

type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Errors []ValidationError
}

func handleValidationErrors(err error) ValidationError {
	var validationErrors ValidationErrors

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, validationErr := range validationErrs {
			validationError := ValidationError{
				Field:   validationErr.Field(),
				Tag:     validationErr.Tag(),
				Value:   fmt.Sprintf("%v", validationErr.Value()),
				Message: getErrorMessage(validationErr),
			}
			validationErrors.Errors = append(validationErrors.Errors, validationError)
		}
	}

	return validationErrors.Errors[0]
}

func getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fe.Field(), fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fe.Field(), fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}
