package main

import (
	"net/http"
	"testing"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/stretchr/testify/mock"
)

func TestActivateUserHandler(t *testing.T) {
	app := newTestApplication(t, config{})
	mux := app.mount()

	t.Run("Success", func(t *testing.T) {
		token := "validtoken"
		mockStore := app.store.(*store.MockStore)

		mockStore.On("ExecTx", mock.Anything, mock.AnythingOfType("func(store.Querier) error")).Return(nil)

		req, err := http.NewRequest(http.MethodGet, "/v1/users/confirm/"+token, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := executeRequest(req, mux)

		checkResponseCode(t, http.StatusOK, rr.Code)
		mockStore.AssertExpectations(t)
	})
}

func TestGetUserHandler(t *testing.T) {
	app := newTestApplication(t, config{})
	mux := app.mount()

	testToken, err := app.auth.GenerateToken(nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Success", func(t *testing.T) {

		mockUser := &store.User{
			ID:    1,
			Name:  "testuser",
			Email: "test@example.com",
		}

		mockStore := app.store.(*store.MockStore)
		mockStore.On("GetUserById", mock.Anything, mockUser.ID).Return(mockUser, nil)

		req, err := http.NewRequest(http.MethodGet, "/v1/users/1", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Authorization", "Bearer "+testToken)
		rr := executeRequest(req, mux)
		checkResponseCode(t, http.StatusOK, rr.Code)
		mockStore.AssertExpectations(t)
	})
}
