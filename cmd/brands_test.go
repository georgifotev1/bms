package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateBrandsHandler(t *testing.T) {
	app := newTestApplication(t, config{})
	mux := app.mount()

	t.Run("Success", func(t *testing.T) {

		testToken, _, err := app.auth.GenerateTokens(1)
		if err != nil {
			t.Fatal(err)
		}

		mockUser := &store.User{
			ID:    1,
			Name:  "testuser",
			Email: "test@example.com",
			Role:  ownerRole,
			BrandID: sql.NullInt32{
				Valid: true,
				Int32: 0,
			},
		}

		payload := CreateBrandPayload{
			Name: "Test Brand",
		}

		mockStore := app.store.(*store.MockStore)
		mockStore.On("GetUserById", mock.Anything, mockUser.ID).Return(mockUser, nil)
		mockStore.On("GetBrandByUrl", mock.Anything, "testbrand").Return("", sql.ErrNoRows)
		mockStore.On("CreateBrandTx", mock.Anything, store.CreateBrandTxParams{
			Name:    "Test Brand",
			PageUrl: "testbrand",
			UserID:  int64(1),
		}).Return(&store.Brand{
			ID:        1,
			Name:      "Test Brand",
			PageUrl:   "testbrand",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil)

		body, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/brand", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Authorization", "Bearer "+testToken)
		req.Header.Set("Content-Type", "application/json")

		ctx := req.Context()
		ctx = context.WithValue(ctx, userCtx, mockUser)
		req = req.WithContext(ctx)

		rr := executeRequest(req, mux)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response store.BrandResponse
		err = json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, payload.Name, response.Name)

		mockStore.AssertExpectations(t)
	})

	t.Run("User already has brand id", func(t *testing.T) {
		testToken, _, err := app.auth.GenerateTokens(2)
		if err != nil {
			t.Fatal(err)
		}

		mockUser := &store.User{
			ID:    2,
			Name:  "testuser 2",
			Email: "test2@example.com",
			Role:  ownerRole,
			BrandID: sql.NullInt32{
				Valid: true,
				Int32: 1, // User has a brand, should fail call
			},
		}

		payload := CreateBrandPayload{
			Name: "Test Brand",
		}

		mockStore := app.store.(*store.MockStore)
		mockStore.On("GetUserById", mock.Anything, mockUser.ID).Return(mockUser, nil)

		body, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/brand", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Authorization", "Bearer "+testToken)
		req.Header.Set("Content-Type", "application/json")

		ctx := req.Context()
		ctx = context.WithValue(ctx, userCtx, mockUser)
		req = req.WithContext(ctx)

		rr := executeRequest(req, mux)

		checkResponseCode(t, http.StatusConflict, rr.Code)
		mockStore.AssertExpectations(t)
	})
}
