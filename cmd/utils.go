package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

const (
	REFRESH_TOKEN         string = "refresh_token"
	CUSTOMER_REFRES_TOKEN string = "customer_refresh_token"

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
func brandResponseMapper(brand *store.Brand, links []*store.BrandSocialLink, hours []*store.BrandWorkingHour) store.BrandResponse {
	socialLinks := []store.SocialLink{}
	workingHours := []store.WorkingHour{}

	if links != nil {
		for _, link := range links {
			socialLinks = append(socialLinks, store.SocialLink{
				ID:          link.ID,
				BrandID:     link.BrandID,
				Platform:    link.Platform,
				Url:         link.Url,
				DisplayName: link.DisplayName.String,
				CreatedAt:   link.CreatedAt,
				UpdatedAt:   link.UpdatedAt,
			})
		}
	}

	if hours != nil {
		for _, hour := range hours {
			openTime := hour.OpenTime.Time.Format("15:04")
			closeTime := hour.CloseTime.Time.Format("15:04")

			workingHour := store.WorkingHour{
				ID:        hour.ID,
				BrandID:   hour.BrandID,
				DayOfWeek: hour.DayOfWeek,
				OpenTime:  openTime,
				CloseTime: closeTime,
				IsClosed:  hour.IsClosed.Bool,
				CreatedAt: hour.CreatedAt,
				UpdatedAt: hour.UpdatedAt,
			}
			workingHours = append(workingHours, workingHour)
		}
	}

	return store.BrandResponse{
		ID:           brand.ID,
		Name:         brand.Name,
		PageUrl:      brand.PageUrl,
		Description:  brand.Description.String,
		Email:        brand.Email.String,
		Phone:        brand.Phone.String,
		Country:      brand.Country.String,
		State:        brand.State.String,
		ZipCode:      brand.ZipCode.String,
		City:         brand.City.String,
		Address:      brand.Address.String,
		LogoUrl:      brand.LogoUrl.String,
		BannerUrl:    brand.BannerUrl.String,
		Currency:     brand.Currency.String,
		CreatedAt:    brand.CreatedAt,
		UpdatedAt:    brand.UpdatedAt,
		SocialLinks:  socialLinks,
		WorkingHours: workingHours,
	}
}

func serviceResponseMapper(service *store.Service, providers []int64) ServiceResponse {
	return ServiceResponse{
		ID:          service.ID,
		Title:       service.Title,
		Description: service.Description.String,
		Duration:    service.Duration,
		BufferTime:  service.BufferTime.Int32,
		Cost:        service.Cost.String,
		IsVisible:   service.IsVisible,
		ImageUrl:    service.ImageUrl.String,
		BrandID:     service.BrandID,
		Providers:   providers,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}
}

func customerResponseMapper(customer *store.Customer, token string) CustomerResponse {
	return CustomerResponse{
		ID:          customer.ID,
		Name:        customer.Name,
		Email:       customer.Email.String,
		BrandId:     customer.BrandID,
		Token:       token,
		PhoneNumber: customer.PhoneNumber,
	}
}

func customersResponseMapper(customer *store.Customer) CustomerResponse {
	return CustomerResponse{
		ID:          customer.ID,
		Name:        customer.Name,
		Email:       customer.Email.String,
		BrandId:     customer.BrandID,
		PhoneNumber: customer.PhoneNumber,
	}
}

func bookingResponseMapper(booking *store.Booking) BookingResponse {
	return BookingResponse{
		ID:           booking.ID,
		CustomerID:   booking.CustomerID,
		ServiceID:    booking.ServiceID,
		UserID:       booking.UserID,
		BrandID:      booking.BrandID,
		StartTime:    booking.StartTime,
		EndTime:      booking.EndTime,
		CustomerName: booking.CustomerName,
		UserName:     booking.UserName,
		ServiceName:  booking.ServiceName,
		Comment:      booking.Comment.String,
		CreatedAt:    booking.CreatedAt,
		UpdatedAt:    booking.UpdatedAt,
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
