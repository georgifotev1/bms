package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

const (
	REFRESH_TOKEN string = "refresh_token"

	ownerRole      string = "owner"
	adminRole      string = "admin"
	userRole       string = "user"
	charset        string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	passwordLength int    = 8
)

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
}

// JSON helpers
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

// User heplers
func userResponseMapper(user *store.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar.String,
		Verified:  user.Verified,
		BrandId:   user.BrandID.Int32,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func isBrowser(r *http.Request) bool {
	userAgent := r.Header.Get("User-Agent")
	// Check for common browser identifiers
	browserIdentifiers := []string{
		"Mozilla", "Chrome", "Safari", "Firefox", "Edge", "Opera",
		"MSIE", "Trident", "Gecko", "WebKit"}

	for _, browser := range browserIdentifiers {
		if strings.Contains(userAgent, browser) {
			return true
		}
	}
	return false
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
