package store

import "time"

type BrandResponse struct {
	ID           int32         `json:"id"`
	Name         string        `json:"name"`
	PageUrl      string        `json:"pageUrl"`
	Description  string        `json:"description"`
	Email        string        `json:"email"`
	Phone        string        `json:"phone"`
	Country      string        `json:"country"`
	State        string        `json:"state"`
	ZipCode      string        `json:"zipCode"`
	City         string        `json:"city"`
	Address      string        `json:"address"`
	LogoUrl      string        `json:"logoUrl"`
	BannerUrl    string        `json:"bannerUrl"`
	Currency     string        `json:"currency"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	SocialLinks  []SocialLink  `json:"socialLinks"`
	WorkingHours []WorkingHour `json:"workingHours"`
}

type SocialLink struct {
	ID          int32     `json:"id"`
	BrandID     int32     `json:"brandId"`
	Platform    string    `json:"platform"`
	Url         string    `json:"url"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type WorkingHour struct {
	ID        int32     `json:"id"`
	BrandID   int32     `json:"brandId"`
	DayOfWeek int32     `json:"dayOfWeek"`
	OpenTime  time.Time `json:"openTime"`
	CloseTime time.Time `json:"closeTime"`
	IsClosed  bool      `json:"isClosed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
