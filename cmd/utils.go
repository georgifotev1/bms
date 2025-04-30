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
	urlcharset     string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
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

// Mappers
func userResponseMapper(user *store.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar.String,
		Verified:  user.Verified,
		BrandId:   user.BrandID.Int32,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func brandResponseMapper(brand *store.Brand) BrandResponse {
	return BrandResponse{
		ID:          brand.ID,
		Name:        brand.Name,
		PageUrl:     brand.PageUrl,
		Description: brand.Description.String,
		Email:       brand.Email.String,
		Phone:       brand.Phone.String,
		Country:     brand.Country.String,
		State:       brand.State.String,
		ZipCode:     brand.ZipCode.String,
		City:        brand.City.String,
		Address:     brand.Address.String,
		LogoUrl:     brand.LogoUrl.String,
		BannerUrl:   brand.BannerUrl.String,
		Currency:    brand.Currency.String,
		CreatedAt:   brand.CreatedAt,
		UpdatedAt:   brand.UpdatedAt,
	}
}

func serviceResponseMapper(service *store.Service, providers []int64) ServiceResponse {
	return ServiceResponse{
		ID:          service.ID,
		Title:       service.Title,
		Description: service.Description.String,
		Duration:    service.Duration,
		BufferTime:  service.BufferTime.Int64,
		Cost:        service.Cost.String,
		IsVisible:   service.IsVisible,
		ImageUrl:    service.ImageUrl.String,
		BrandID:     service.BrandID,
		Providers:   providers,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
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
