package store

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(User), args.Error(1)
}

func (m *MockQuerier) CreateUserInvitation(ctx context.Context, arg CreateUserInvitationParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) DeleteUser(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) VerifyUser(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) DeleteUserInvitation(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockQuerier) GetUserFromInvitation(ctx context.Context, token string) (int64, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(int64), args.Error(1)
}
