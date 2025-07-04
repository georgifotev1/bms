package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

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
	OpenTime  string    `json:"openTime"`
	CloseTime string    `json:"closeTime"`
	IsClosed  bool      `json:"isClosed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateBrandTxParams struct {
	Name    string
	PageUrl string
	UserID  int64
}

func (s *SQLStore) CreateBrandTx(ctx context.Context, arg CreateBrandTxParams) (*Brand, []*BrandWorkingHour, error) {
	var brand *Brand
	var workingHours []*BrandWorkingHour

	err := s.execTx(ctx, func(q Querier) error {
		var err error
		brand, err = q.CreateBrand(ctx, CreateBrandParams{
			Name:    arg.Name,
			PageUrl: arg.PageUrl,
		})
		if err != nil {
			fmt.Println("ERRROROROR ISSSS ", err)
			return err
		}

		err = q.AssociateUserWithBrand(ctx, AssociateUserWithBrandParams{
			BrandID: sql.NullInt32{
				Valid: true,
				Int32: brand.ID,
			},
			ID: arg.UserID,
		})
		if err != nil {
			return err
		}

		openTime, _ := time.Parse("15:04", "09:00")
		closeTime, _ := time.Parse("15:04", "17:00")

		defaultWorkingHours := []UpsertBrandWorkingHoursParams{
			{BrandID: brand.ID, DayOfWeek: 1, OpenTime: sql.NullTime{Time: openTime, Valid: true}, CloseTime: sql.NullTime{Time: closeTime, Valid: true}, IsClosed: false},
			{BrandID: brand.ID, DayOfWeek: 2, OpenTime: sql.NullTime{Time: openTime, Valid: true}, CloseTime: sql.NullTime{Time: closeTime, Valid: true}, IsClosed: false},
			{BrandID: brand.ID, DayOfWeek: 3, OpenTime: sql.NullTime{Time: openTime, Valid: true}, CloseTime: sql.NullTime{Time: closeTime, Valid: true}, IsClosed: false},
			{BrandID: brand.ID, DayOfWeek: 4, OpenTime: sql.NullTime{Time: openTime, Valid: true}, CloseTime: sql.NullTime{Time: closeTime, Valid: true}, IsClosed: false},
			{BrandID: brand.ID, DayOfWeek: 5, OpenTime: sql.NullTime{Time: openTime, Valid: true}, CloseTime: sql.NullTime{Time: closeTime, Valid: true}, IsClosed: false},
			{BrandID: brand.ID, DayOfWeek: 6, OpenTime: sql.NullTime{Time: openTime, Valid: false}, CloseTime: sql.NullTime{Time: closeTime, Valid: false}, IsClosed: true},
			{BrandID: brand.ID, DayOfWeek: 0, OpenTime: sql.NullTime{Time: openTime, Valid: false}, CloseTime: sql.NullTime{Time: closeTime, Valid: false}, IsClosed: true},
		}

		for _, wh := range defaultWorkingHours {
			if newWh, err := q.UpsertBrandWorkingHours(ctx, wh); err != nil {
				return err
			} else {
				workingHours = append(workingHours, newWh)
			}
		}

		return nil
	})

	return brand, workingHours, err
}

func (s *SQLStore) GetBrandProfileTx(ctx context.Context, brandID int32) (*Brand, []*BrandSocialLink, []*BrandWorkingHour, error) {
	var brand *Brand
	var socialLinks []*BrandSocialLink
	var workingHours []*BrandWorkingHour

	err := s.execTx(ctx, func(q Querier) error {
		var err error
		brand, err = q.GetBrand(ctx, brandID)
		if err != nil {
			return err
		}

		socialLinks, err = q.GetBrandSocialLinks(ctx, brandID)
		if err != nil {
			return err
		}

		workingHours, err = q.GetBrandWorkingHours(ctx, brandID)
		if err != nil {
			return err
		}

		return nil
	})

	return brand, socialLinks, workingHours, err
}
func (s *SQLStore) UpdateBrandProfileTx(ctx context.Context, params UpdateBrandParams, workingHoursParams []UpsertBrandWorkingHoursParams, socialLinkParams []UpsertBrandSocialLinkParams) (*Brand, []*BrandSocialLink, []*BrandWorkingHour, error) {
	var updatedBrand *Brand
	var socialLinks []*BrandSocialLink
	var workingHours []*BrandWorkingHour

	err := s.execTx(ctx, func(q Querier) error {
		var err error

		// Update the brand
		updatedBrand, err = q.UpdateBrand(ctx, params)
		if err != nil {
			return err
		}

		// Update working hours
		for _, wh := range workingHoursParams {
			if updatedWH, err := q.UpsertBrandWorkingHours(ctx, wh); err != nil {
				return err
			} else {
				workingHours = append(workingHours, updatedWH)
			}
		}

		// Update social links
		for _, sl := range socialLinkParams {
			if updatedSL, err := q.UpsertBrandSocialLink(ctx, sl); err != nil {
				return err
			} else {
				socialLinks = append(socialLinks, updatedSL)
			}
		}

		return nil
	})

	return updatedBrand, socialLinks, workingHours, err
}
