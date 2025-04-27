package store

import (
	"context"
	"database/sql"

	"github.com/stretchr/testify/mock"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateUser(ctx context.Context, arg CreateUserParams) (*User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(*User), args.Error(1)
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

func (m *MockQuerier) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockQuerier) GetUserById(ctx context.Context, id int64) (*User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockQuerier) AddBrandSocialLink(ctx context.Context, arg AddBrandSocialLinkParams) (*BrandSocialLink, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(*BrandSocialLink), args.Error(1)
}

func (m *MockQuerier) AssociateUserWithBrand(ctx context.Context, arg AssociateUserWithBrandParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) CreateBrand(ctx context.Context, arg CreateBrandParams) (*Brand, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(*Brand), args.Error(1)
}

func (m *MockQuerier) DeleteBrandSocialLink(ctx context.Context, arg DeleteBrandSocialLinkParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) GetBrandProfile(ctx context.Context, id int32) (*GetBrandProfileRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*GetBrandProfileRow), args.Error(1)
}

func (m *MockQuerier) GetBrandUsers(ctx context.Context, brandID sql.NullInt32) ([]*User, error) {
	args := m.Called(ctx, brandID)
	return args.Get(0).([]*User), args.Error(1)
}

func (m *MockQuerier) GetBrandWorkingHours(ctx context.Context, brandID int32) ([]*BrandWorkingHour, error) {
	args := m.Called(ctx, brandID)
	return args.Get(0).([]*BrandWorkingHour), args.Error(1)
}

func (m *MockQuerier) UpdateBrand(ctx context.Context, arg UpdateBrandParams) (*Brand, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(*Brand), args.Error(1)
}

func (m *MockQuerier) UpdateBrandPartial(ctx context.Context, arg UpdateBrandPartialParams) (*Brand, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(*Brand), args.Error(1)
}

func (m *MockQuerier) UpdateBrandWorkingHours(ctx context.Context, arg UpdateBrandWorkingHoursParams) (*BrandWorkingHour, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(*BrandWorkingHour), args.Error(1)
}

func (m *MockQuerier) GetBrandByUrl(ctx context.Context, pageUrl string) (string, error) {
	args := m.Called(ctx, pageUrl)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}
