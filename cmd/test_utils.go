package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/georgifotev1/bms/internal/auth"
	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/store"
	"go.uber.org/zap"
)

func newTestApplication(cfg config) *application {
	mockStore := new(store.MockQuerier)
	mockMailer := new(mailer.MockClient)
	mockAuth := new(auth.MockAuthenticator)

	return &application{
		config: cfg,
		store:  mockStore,
		mailer: mockMailer,
		auth:   mockAuth,
		logger: zap.NewNop().Sugar(),
	}
}
func executeRequest(req *http.Request, mux http.Handler) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d", expected, actual)
	}
}
