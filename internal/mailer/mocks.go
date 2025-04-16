package mailer

import (
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) Send(templateFile, username, email string, data any) (int, error) {
	args := m.Called(templateFile, username, email, data)
	return args.Int(0), args.Error(1)
}
