package store

import (
	"context"
	"database/sql"
	"errors"
)

type ActivateUserTxParams struct {
	Token string
}

var ErrInvalidActivationToken = errors.New("invalid activation token")

func (s *SQLStore) ActivateUserTx(ctx context.Context, arg ActivateUserTxParams) error {
	err := s.execTx(ctx, func(q Querier) error {
		// Get user ID from the invitation token
		userId, err := q.GetUserFromInvitation(ctx, arg.Token)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				return ErrInvalidActivationToken
			default:
				return err
			}
		}

		// Mark user as verified
		if err := q.VerifyUser(ctx, userId); err != nil {
			return err
		}

		// Delete the invitation token as it's now used
		if err := q.DeleteUserInvitation(ctx, userId); err != nil {
			return err
		}

		return nil
	})

	return err
}
