package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeLoginLimiter struct {
	blocked       bool
	remaining     time.Duration
	checkError    error
	recordError   error
	resetError    error
	checkedKeys   []string
	recordedKeys  []string
	resetKeys     []string
	recordContext context.Context
	resetContext  context.Context
}

func (f *fakeLoginLimiter) IsBlocked(_ context.Context, key string) (bool, time.Duration, error) {
	f.checkedKeys = append(f.checkedKeys, key)
	return f.blocked, f.remaining, f.checkError
}

func (f *fakeLoginLimiter) RecordFailure(ctx context.Context, key string) error {
	f.recordContext = ctx
	f.recordedKeys = append(f.recordedKeys, key)
	return f.recordError
}

func (f *fakeLoginLimiter) Reset(ctx context.Context, key string) error {
	f.resetContext = ctx
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

func TestLoginRateLimitRejectsOversizedBody(t *testing.T) {
	limiter := &fakeLoginLimiter{}
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
		strings.NewReader(strings.Repeat("x", (1<<20)+1)),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.False(t, handlerCalled)
	require.Empty(t, limiter.checkedKeys)
	require.Empty(t, limiter.recordedKeys)
	require.Empty(t, limiter.resetKeys)
}

func TestLoginRateLimitUsesNonCancellableContextAfterHandler(t *testing.T) {
	limiter := &fakeLoginLimiter{}
	router := gin.New()

	var cancel context.CancelFunc
	router.POST(
		"/login",
		LoginRateLimit(limiter),
		func(c *gin.Context) {
			cancel()
			c.Status(http.StatusUnauthorized)
		},
	)

	requestContext, requestCancel := context.WithCancel(context.Background())
	cancel = requestCancel

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"wrongpassword"}`),
	).WithContext(requestContext)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.NotNil(t, limiter.recordContext)
	require.NoError(t, limiter.recordContext.Err())

}

func TestLoginRateLimitAllowsRequestAfterBlockRecovery(t *testing.T) {
	limiter := &fakeLoginLimiter{
		blocked:   true,
		remaining: 2 * time.Second,
	}
	router := gin.New()
	handlerCalls := 0

	router.POST(
		"/login",
		LoginRateLimit(limiter),
		func(c *gin.Context) {
			handlerCalls++
			c.Status(http.StatusOK)
		},
	)

	blockedRequest := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"correctpassword"}`),
	)
	blockedRequest.Header.Set("Content-Type", "application/json")

	blockedResponse := httptest.NewRecorder()
	router.ServeHTTP(blockedResponse, blockedRequest)

	require.Equal(t, http.StatusTooManyRequests, blockedResponse.Code)
	require.Equal(t, 0, handlerCalls)

	limiter.blocked = false
	limiter.remaining = 0

	recoveredRequest := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"correctpassword"}`),
	)
	recoveredRequest.Header.Set("Content-Type", "application/json")

	recoveredResponse := httptest.NewRecorder()
	router.ServeHTTP(recoveredResponse, recoveredRequest)

	require.Equal(t, http.StatusOK, recoveredResponse.Code)
	require.Equal(t, 1, handlerCalls)
	require.Equal(t, []string{"alice"}, limiter.resetKeys)

}
func TestLoginRateLimitFailsClosedWhenLimiterUnavailable(t *testing.T) {
	limiter := &fakeLoginLimiter{
		checkError: errors.New("redis unavailable"),
	}
	router := gin.New()
	handlerCalled := false

	var capturedError error

	router.Use(func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			capturedError = c.Errors.Last().Err
		}
	})

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

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.False(t, handlerCalled)
	require.ErrorIs(t, capturedError, limiter.checkError)
	require.Equal(t, []string{"alice"}, limiter.checkedKeys)
	require.Empty(t, limiter.recordedKeys)
	require.Empty(t, limiter.resetKeys)
}

func TestLoginRateLimitRecordsPostHandlerFailureError(t *testing.T) {
	limiterError := errors.New("redis unavailable")
	limiter := &fakeLoginLimiter{
		recordError: limiterError,
	}
	router := gin.New()
	var requestContext *gin.Context

	router.POST(
		"/login",
		LoginRateLimit(limiter),
		func(c *gin.Context) {
			requestContext = c
			c.Status(http.StatusUnauthorized)
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"wrongpassword"}`),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.NotNil(t, requestContext)
	require.Len(t, requestContext.Errors, 1)
	require.ErrorIs(t, requestContext.Errors[0], limiterError)
}

func TestLoginRateLimitRecordsPostHandlerResetError(t *testing.T) {
	limiterError := errors.New("redis unavailable")
	limiter := &fakeLoginLimiter{
		resetError: limiterError,
	}
	router := gin.New()
	var requestContext *gin.Context

	router.POST(
		"/login",
		LoginRateLimit(limiter),
		func(c *gin.Context) {
			requestContext = c
			c.Status(http.StatusOK)
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"alice","password":"correctpassword"}`),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, requestContext)
	require.Len(t, requestContext.Errors, 1)
	require.ErrorIs(t, requestContext.Errors[0], limiterError)
}
