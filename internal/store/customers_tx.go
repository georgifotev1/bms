package store

import (
	"context"
	"database/sql"
)

type CreateGuestTxParams struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Email       string `json:"email"`
	BrandId     int32  `json:"brandId"`
}

func (s *SQLStore) CreateGuestTx(ctx context.Context, arg CreateGuestTxParams) (*Customer, bool, error) {
	var result Customer
	var exists bool
	err := s.execTx(ctx, func(q Querier) error {

		customer, err := q.GetCustomerByNameAndPhone(ctx, GetCustomerByNameAndPhoneParams{
			Name:        arg.Name,
			PhoneNumber: arg.PhoneNumber,
		})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if err == nil {
			result = *customer
			exists = true
			return nil
		}

		newCustomer, err := q.CreateGuestCustomer(ctx, CreateGuestCustomerParams{
			Name:        arg.Name,
			PhoneNumber: arg.PhoneNumber,
			Email: sql.NullString{
				String: arg.Email,
				Valid:  arg.Email != "",
			},
			BrandID: arg.BrandId,
		})
		if err != nil {
			return err
		}

		result = *newCustomer
		exists = false
		return nil
	})

	return &result, exists, err
}
