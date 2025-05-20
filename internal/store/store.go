package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	Querier
	CreateServiceTx(ctx context.Context, arg CreateServiceTxParams) (*CreateServiceTxResult, error)
	ActivateUserTx(ctx context.Context, arg ActivateUserTxParams) error
	CreateBrandTx(ctx context.Context, arg CreateBrandTxParams) (*Brand, error)
	CreateGuestTx(ctx context.Context, arg CreateGuestTxParams) (*Customer, bool, error)
	GetBrandProfileTx(ctx context.Context, brandID int32) (*Brand, []*BrandSocialLink, []*BrandWorkingHour, error)
}

type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

func (s *SQLStore) execTx(ctx context.Context, fn func(Querier) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := s.WithTx(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx failed: %v, rollback failed: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
