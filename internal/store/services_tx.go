package store

import (
	"context"
	"database/sql"
	"errors"
)

type CreateServiceTxParams struct {
	Title       string
	Description string
	Duration    int64
	BufferTime  int64
	Cost        string
	IsVisible   bool
	ImageURL    string
	BrandID     int32
	UserIDs     []int64
}

type CreateServiceTxResult struct {
	Service   *Service
	Providers []int64
}

var ErrInvalidUserIDs = errors.New("one or more user IDs are invalid or don't belong to your brand")

func (s *SQLStore) CreateServiceTx(ctx context.Context, arg CreateServiceTxParams) (CreateServiceTxResult, error) {
	var result CreateServiceTxResult

	err := s.execTx(ctx, func(q Querier) error {
		// If user IDs are provided, validate they exist and belong to the brand
		if len(arg.UserIDs) > 0 {
			count, err := q.ValidateUsersCount(ctx, ValidateUsersCountParams{
				Ids: arg.UserIDs,
				BrandID: sql.NullInt32{
					Int32: arg.BrandID,
					Valid: arg.BrandID != 0,
				},
			})
			if err != nil {
				return err
			}

			if int(count) != len(arg.UserIDs) {
				return ErrInvalidUserIDs
			}
		}

		// Create the service
		service, err := q.CreateService(ctx, CreateServiceParams{
			Title: arg.Title,
			Description: sql.NullString{
				String: arg.Description,
				Valid:  arg.Description != "",
			},
			Duration: arg.Duration,
			BufferTime: sql.NullInt64{
				Int64: arg.BufferTime,
				Valid: arg.BufferTime > 0,
			},
			Cost: sql.NullString{
				String: arg.Cost,
				Valid:  arg.Cost != "",
			},
			IsVisible: arg.IsVisible,
			ImageUrl: sql.NullString{
				String: arg.ImageURL,
				Valid:  arg.ImageURL != "",
			},
			BrandID: arg.BrandID,
		})
		if err != nil {
			return err
		}

		// Assign service to the users
		for _, userID := range arg.UserIDs {
			err := q.AssignServiceToUser(ctx, AssignServiceToUserParams{
				ServiceID: service.ID,
				UserID:    userID,
			})
			if err != nil {
				return err
			}
		}

		result.Service = service
		result.Providers = arg.UserIDs
		return nil
	})

	return result, err
}
