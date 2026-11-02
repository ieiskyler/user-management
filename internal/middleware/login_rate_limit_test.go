package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeLoginLimiter struct {
	blocked      bool
	remaining    time.Duration
	checkError   error
	recordError  error
	resetError   error
	recordedKeys []string
	resetKeys    []string
}

func (f *fakeLoginLimiter) IsBlocked(
	context.Context,
	string,
) (bool, time.Duration, error) {
	return f.blocked, f.remaining, f.checkError
}

func (f *fakeLoginLimiter) RecordFailure(
	_ context.Context,
	key string,
) error {
	f.recordedKeys = append(f.recordedKeys, key)
	return f.recordError
}

func (f *fakeLoginLimiter) Reset(
	_ context.Context,
	key string,
) error {
	f.resetKeys = append(f.resetKeys, key)
	return f.resetError
}

func TestLoginRateLimitRecordsFailedLogin(t *testing.T) {
	limiter := &fakeLoginLimiter{}
	router := gin.New()

	router.POST(
		"/login",
		LoginRateLimit(limiter),
		func(c *gin.Context) {
			c.Status(http.StatusUnauthorized)
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"wrongpassword"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, []string{"alice"}, limiter.recordedKeys)
	require.Empty(t, limiter.resetKeys)
}

func TestLoginRateLimitResetsSuccessfulLogin(t *testing.T) {
	limiter := &fakeLoginLimiter{}
	router := gin.New()

	router.POST(
		"/login",
		LoginRateLimit(limiter),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"correctpassword"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, limiter.recordedKeys)
	require.Equal(t, []string{"alice"}, limiter.resetKeys)
}

func TestLoginRateLimitRejectsBlockedAccount(t *testing.T) {
	limiter := &fakeLoginLimiter{
		blocked:   true,
		remaining: 90 * time.Second,
	}
	router := gin.New()
	handlerCalled := false

	router.POST(
		"/login",
		LoginRateLimit(limiter),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"correctpassword"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "90", recorder.Header().Get("Retry-After"))
	require.False(t, handlerCalled)
	require.Empty(t, limiter.recordedKeys)
	require.Empty(t, limiter.resetKeys)
}
