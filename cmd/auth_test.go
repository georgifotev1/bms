package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestRegisterUserHandler(t *testing.T) {
	app := newTestApplication(config{})

	t.Run("Success", func(t *testing.T) {
		userData := RegisterUserPayload{
			Email:    "test@example.com",
			Password: "password",
			Username: "testuser",
		}
		userJSON, err := json.Marshal(userData)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/v1/auth/user", bytes.NewBuffer(userJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		mockUser := store.User{
			ID:        1,
			Name:      userData.Username,
			Email:     userData.Email,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockStore := app.store.(*store.MockQuerier)
		mockMailer := app.mailer.(*mailer.MockClient)
		mockStore.On("CreateUser", mock.Anything, mock.AnythingOfType("store.CreateUserParams")).Return(mockUser, nil)
		mockStore.On("CreateUserInvitation", mock.Anything, mock.AnythingOfType("store.CreateUserInvitationParams")).Return(nil)
		mockMailer.On("Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(200, nil)

		rr := httptest.NewRecorder()

		app.registerUserHandler(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code)

		var response UserWithToken
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Equal(t, userData.Email, response.Email)
		require.Equal(t, userData.Username, response.Name)
		require.NotEmpty(t, response.Token)

		mockStore.AssertExpectations(t)
		mockMailer.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		userData := RegisterUserPayload{
			Password: "password",
			Username: "testuser",
		}
		userJSON, err := json.Marshal(userData)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/v1/auth/user", bytes.NewBuffer(userJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		app.registerUserHandler(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)

		mockStore := app.store.(*store.MockQuerier)
		mockMailer := app.mailer.(*mailer.MockClient)
		mockStore.AssertNotCalled(t, "CreateUser")
		mockStore.AssertNotCalled(t, "CreateUserInvitation")
		mockMailer.AssertNotCalled(t, "Send")
	})
}

func TestCreateTokenHandler(t *testing.T) {
	app := newTestApplication(config{})

	t.Run("Success", func(t *testing.T) {
		credentials := CreateUserTokenPayload{
			Email:    "test@example.com",
			Password: "password",
		}
		credentialsJSON, err := json.Marshal(credentials)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/v1/auth/token", bytes.NewBuffer(credentialsJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		hashedPass, err := bcrypt.GenerateFromPassword([]byte(credentials.Password), bcrypt.DefaultCost)
		require.NoError(t, err)

		mockUser := store.User{
			ID:       1,
			Name:     "testuser",
			Email:    credentials.Email,
			Password: hashedPass,
		}

		mockStore := app.store.(*store.MockQuerier)
		mockStore.On("GetUserByEmail", mock.Anything, credentials.Email).Return(mockUser, nil)

		rr := httptest.NewRecorder()
		app.createTokenHandler(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code)

		var response string
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		mockStore.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		credentials := CreateUserTokenPayload{
			// Missing email
			Password: "password",
		}
		credentialsJSON, err := json.Marshal(credentials)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/v1/auth/token", bytes.NewBuffer(credentialsJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		app.createTokenHandler(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)

		mockStore := app.store.(*store.MockQuerier)
		mockStore.AssertNotCalled(t, "GetUserByEmail")
	})

	t.Run("UserNotFound", func(t *testing.T) {
		credentials := CreateUserTokenPayload{
			Email:    "nonexistent@example.com",
			Password: "password",
		}
		credentialsJSON, err := json.Marshal(credentials)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/v1/auth/token", bytes.NewBuffer(credentialsJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		mockStore := app.store.(*store.MockQuerier)
		mockStore.On("GetUserByEmail", mock.Anything, credentials.Email).Return(store.User{}, sql.ErrNoRows)

		rr := httptest.NewRecorder()
		app.createTokenHandler(rr, req)

		require.Equal(t, http.StatusUnauthorized, rr.Code)
		mockStore.AssertExpectations(t)
	})

	t.Run("IncorrectPassword", func(t *testing.T) {
		credentials := CreateUserTokenPayload{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}
		credentialsJSON, err := json.Marshal(credentials)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/v1/auth/token", bytes.NewBuffer(credentialsJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		hashedPass, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
		require.NoError(t, err)

		mockUser := store.User{
			ID:       1,
			Name:     "testuser",
			Email:    credentials.Email,
			Password: hashedPass,
		}

		mockStore := app.store.(*store.MockQuerier)
		mockStore.On("GetUserByEmail", mock.Anything, credentials.Email).Return(mockUser, nil)

		rr := httptest.NewRecorder()
		app.createTokenHandler(rr, req)

		require.Equal(t, http.StatusUnauthorized, rr.Code)
		mockStore.AssertExpectations(t)
	})
}
