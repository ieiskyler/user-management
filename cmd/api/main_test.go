package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"user-management/internal/handler"
	"user-management/internal/models"
	"user-management/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type mockAuthService struct{}

func (mockAuthService) Register(*models.User) error          { return nil }
func (mockAuthService) Login(string, string) (string, error) { return "token", nil }

type mockUserService struct{}

type fakeLoginLimiter struct{}

func (fakeLoginLimiter) IsBlocked(
	context.Context,
	string,
) (bool, time.Duration, error) {
	return false, 0, nil
}

func (fakeLoginLimiter) RecordFailure(
	context.Context,
	string,
) error {
	return nil
}

func (fakeLoginLimiter) Reset(
	context.Context,
	string,
) error {
	return nil
}

func (mockUserService) GetAllUsers(page, limit int) ([]models.User, int64, error) {
	return []models.User{}, 0, nil
}

func TestRouter(t *testing.T) {
	router := server.NewRouter(
		newAuthHandlerForTest(),
		newUserHandlerForTest(),
		fakeLoginLimiter{},
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
	t.Chdir("../..")

	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DATABASE", "0")

	originalConnectDatabase := connectDatabase
	originalRunServer := runServer
	t.Cleanup(func() {
		connectDatabase = originalConnectDatabase
		runServer = originalRunServer
	})

	connectDatabase = func() (*gorm.DB, error) {
		return &gorm.DB{}, nil
	}

	connectRedis = func() (*redis.Client, error) {
		return redis.NewClient(&redis.Options{}), nil
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
