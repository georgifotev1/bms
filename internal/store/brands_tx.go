package store

import (
	"context"
	"database/sql"
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

func (s *SQLStore) CreateBrandTx(ctx context.Context, arg CreateBrandTxParams) (*Brand, error) {
	var result Brand

	err := s.execTx(ctx, func(q Querier) error {
		brand, err := q.CreateBrand(ctx, CreateBrandParams{
			Name:    arg.Name,
			PageUrl: arg.PageUrl,
		})
		if err != nil {
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

		result = *brand
		return nil
	})

	return &result, err
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
