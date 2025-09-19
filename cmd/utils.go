package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

var Validate *validator.Validate
var Decoder *schema.Decoder

const (
	SESSION_TOKEN          string = "session_token"
	CUSTOMER_SESSION_TOKEN string = "customer_session_token"

	ownerRole      string = "owner"
	adminRole      string = "admin"
	userRole       string = "user"
	charset        string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	urlcharset     string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	passwordLength int    = 8
)

func init() {
	Decoder = schema.NewDecoder()
	Validate = validator.New(validator.WithRequiredStructEnabled())
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1_048_578 // 1mb
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) error {
	type response struct {
		Error string `json:"error"`
	}

	return writeJSON(w, status, &response{Error: message})
}

// Token helpers
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func generateRandomPassword() (string, error) {
	password := make([]byte, passwordLength)
	for i := range password {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[randomIndex.Int64()]
	}
	return string(password), nil
}

func generateSubstring(length int) string {
	result := make([]byte, length)

	for i := range result {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(urlcharset))))
		if err != nil {
			result[i] = 'x'
		}
		result[i] = urlcharset[randomIndex.Int64()]
	}

	return string(result)
}

func (app *application) SetCookie(w http.ResponseWriter, name, value string) {
	isDev := app.config.env == "development"
	sameSite := http.SameSiteStrictMode
	if isDev {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSite,
		Domain:   app.config.clientHost,
		MaxAge:   60 * 60 * 24 * 30,
	})
}

func (app *application) ClearCookie(w http.ResponseWriter, name string) {
	isDev := app.config.env == "development"
	sameSite := http.SameSiteStrictMode
	if isDev {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "", // Empty value
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSite,
		Domain:   app.config.clientHost,
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
	})
}

func getBrandIDFromCtx(ctx context.Context) int32 {
	ctxValue := ctx.Value(brandIDCtx)
	return ctxValue.(int32)
}

func getUserFromCtx(ctx context.Context) *store.User {
	ctxValue := ctx.Value(userCtx)
	return ctxValue.(*store.User)

}

func toNullString(s string) sql.NullString {
	return sql.NullString{
		Valid:  s != "",
		String: s,
	}
}

func toNullBool(b bool) sql.NullBool {
	return sql.NullBool{
		Valid: b,
		Bool:  b,
	}
}

func parseTimeString(timeStr string) sql.NullTime {
	if timeStr == "" {
		return sql.NullTime{Valid: false}
	}

	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return sql.NullTime{Valid: false}
	}

	return sql.NullTime{Time: t, Valid: true}
}
