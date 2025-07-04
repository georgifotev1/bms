package main

import (
	"time"

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
	Name         string                    `json:"name" validate:"required,min=3,max=100"`
	PageUrl      string                    `json:"pageUrl" validate:"required"`
	Description  string                    `json:"description"`
	Email        string                    `json:"email"`
	Phone        string                    `json:"phone"`
	Country      string                    `json:"country"`
	State        string                    `json:"state"`
	ZipCode      string                    `json:"zipCode"`
	City         string                    `json:"city"`
	Address      string                    `json:"address"`
	LogoUrl      string                    `json:"logoUrl"`
	BannerUrl    string                    `json:"bannerUrl"`
	Currency     string                    `json:"currency"`
	WorkingHours []UpdateBrandWorkingHours `json:"workingHours,omitempty"`
	SocialLinks  []UpdateBrandSocialLink   `json:"socialLinks,omitempty"`
}

type UpdateBrandWorkingHours struct {
	DayOfWeek int32     `json:"dayOfWeek" validate:"required,min=0,max=6"`
	OpenTime  time.Time `json:"openTime"`
	CloseTime time.Time `json:"closeTime"`
	IsClosed  bool      `json:"isClosed"`
}

type UpdateBrandSocialLink struct {
	Platform string `json:"platform" validate:"required"`
	Url      string `json:"url" validate:"required,url"`
}

func (p *UpdateBrandPayload) ToWorkingHoursParams(brandID int32) []store.UpsertBrandWorkingHoursParams {
	params := make([]store.UpsertBrandWorkingHoursParams, len(p.WorkingHours))
	for i, wh := range p.WorkingHours {
		params[i] = store.UpsertBrandWorkingHoursParams{
			BrandID:   brandID,
			DayOfWeek: wh.DayOfWeek,
			OpenTime:  toNullTime(wh.OpenTime),
			CloseTime: toNullTime(wh.CloseTime),
			IsClosed:  wh.IsClosed,
		}
	}
	return params
}

func (p *UpdateBrandPayload) ToSocialLinkParams(brandID int32) []store.UpsertBrandSocialLinkParams {
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
