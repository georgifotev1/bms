package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

		var response UserResponse
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

func TestActivateUserHandler(t *testing.T) {
	app := newTestApplication(config{})
	mux := app.mount()

	t.Run("Success", func(t *testing.T) {
		token := "validtoken"
		hashedToken := app.hashToken(token)
		userId := int64(1)

		mockStore := app.store.(*store.MockQuerier)
		mockStore.On("GetUserFromInvitation", mock.Anything, hashedToken).Return(userId, nil)
		mockStore.On("VerifyUser", mock.Anything, userId).Return(nil)
		mockStore.On("DeleteUserInvitation", mock.Anything, userId).Return(nil)

		req, err := http.NewRequest(http.MethodGet, "/v1/auth/confirm/"+token, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := executeRequest(req, mux)

		checkResponseCode(t, http.StatusNoContent, rr.Code)
		mockStore.AssertExpectations(t)
	})
}
