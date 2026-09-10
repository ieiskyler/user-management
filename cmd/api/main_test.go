package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-management/internal/handler"
	"user-management/internal/models"
	"user-management/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type mockAuthService struct{}

func (mockAuthService) Register(*models.User) error          { return nil }
func (mockAuthService) Login(string, string) (string, error) { return "token", nil }

type mockUserService struct{}

func (mockUserService) GetAllUsers() ([]models.User, error) { return []models.User{}, nil }

func TestRouter(t *testing.T) {
	router := server.NewRouter(
		newAuthHandlerForTest(),
		newUserHandlerForTest(),
	)

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/register"},
		{http.MethodPost, "/api/v1/login"},
		{http.MethodGet, "/api/v1/users"},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(route.method, route.path, nil)
		router.ServeHTTP(recorder, request)
		assert.NotEqual(t, http.StatusNotFound, recorder.Code)
	}
}

func TestMain(t *testing.T) {
	originalConnectDatabase := connectDatabase
	originalRunServer := runServer
	t.Cleanup(func() {
		connectDatabase = originalConnectDatabase
		runServer = originalRunServer
	})

	connectDatabase = func() (*gorm.DB, error) {
		return &gorm.DB{}, nil
	}
	serverStarted := false
	runServer = func(*gin.Engine) error {
		serverStarted = true
		return nil
	}

	main()

	assert.True(t, serverStarted)
}

func newAuthHandlerForTest() *handler.AuthHandler {
	return handler.NewAuthHandler(mockAuthService{})
}

func newUserHandlerForTest() *handler.UserHandler {
	return handler.NewUserHandler(mockUserService{})
}

func TestHandleServerError(t *testing.T) {
	t.Run("normal server shutdown returns nil", func(t *testing.T) {
		err := handleServerError(http.ErrServerClosed)

		assert.NoError(t, err)
	})

	t.Run("unexpected server error is returned", func(t *testing.T) {
		expectedError := errors.New("server failed")

		err := handleServerError(expectedError)

		assert.ErrorIs(t, err, expectedError)
	})
}
