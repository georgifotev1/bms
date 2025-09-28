package main

import (
	"github.com/georgifotev1/bms/internal/store"
)

type brandKey string

const (
	brandIDCtx brandKey = "brand"
)

type CreateBrandPayload struct {
	Name string `json:"name" validate:"required,min=3,max=100"`
}

type UpdateBrandPayload struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	PageUrl     string `json:"pageUrl" validate:"required"`
	Description string `json:"description"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Country     string `json:"country"`
	State       string `json:"state"`
	ZipCode     string `json:"zipCode"`
	City        string `json:"city"`
	Address     string `json:"address"`
	LogoUrl     string `json:"logoUrl"`
	BannerUrl   string `json:"bannerUrl"`
	Currency    string `json:"currency"`
}

type UpdateBrandWorkingHoursPayload struct {
	WorkingHours []UpdateBrandWorkingHours `json:"workingHours"`
}

type UpdateBrandWorkingHours struct {
	DayOfWeek int32  `json:"dayOfWeek" validate:"required,min=0,max=6"`
	OpenTime  string `json:"openTime"`
	CloseTime string `json:"closeTime"`
	IsClosed  bool   `json:"isClosed"`
}

type UpdateBrandSocialLinksPayload struct {
	SocialLinks []UpdateBrandSocialLink `json:"socialLinks"`
}

type UpdateBrandSocialLink struct {
	Platform string `json:"platform" validate:"required"`
	Url      string `json:"url" validate:"required,url"`
}

func (p *UpdateBrandWorkingHoursPayload) ToWorkingHoursParams(brandID int32) []store.UpsertBrandWorkingHoursParams {
	params := make([]store.UpsertBrandWorkingHoursParams, len(p.WorkingHours))
	for i, wh := range p.WorkingHours {
		openTime := parseTimeStringFromUserLoacation("Europe/Sofia", wh.OpenTime)
		closeTime := parseTimeStringFromUserLoacation("Europe/Sofia", wh.CloseTime)

		params[i] = store.UpsertBrandWorkingHoursParams{
			BrandID:   brandID,
			DayOfWeek: wh.DayOfWeek,
			OpenTime:  parseTimeString(openTime.Format("15:04")),
			CloseTime: parseTimeString(closeTime.Format("15:04")),
			IsClosed:  wh.IsClosed,
		}
	}
	return params
}

// Helper method for social links payload
func (p *UpdateBrandSocialLinksPayload) ToSocialLinkParams(brandID int32) []store.UpsertBrandSocialLinkParams {
	params := make([]store.UpsertBrandSocialLinkParams, len(p.SocialLinks))
	for i, sl := range p.SocialLinks {
		params[i] = store.UpsertBrandSocialLinkParams{
			Platform: sl.Platform,
			Url:      sl.Url,
			BrandID:  brandID,
		}
	}
	return params
}
