package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"user-management/internal/handler"
	"user-management/internal/models"
	"user-management/internal/ratelimit"
	"user-management/internal/repository"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type invalidCredentialsAuthService struct{}

type fakeLoginLimiter struct {
	blocked     bool
	checkedKeys []string
}

func (invalidCredentialsAuthService) Register(*models.User) error {
	return nil
}

func (invalidCredentialsAuthService) Login(string, string) (string, error) {
	return "", service.ErrInvalidCredentials
}

func (f *fakeLoginLimiter) IsBlocked(_ context.Context, account string) (bool, time.Duration, error) {
	f.checkedKeys = append(f.checkedKeys, account)
	return f.blocked, time.Minute, nil
}

func (fakeLoginLimiter) RecordFailure(context.Context, string) error {
	return nil
}

func (fakeLoginLimiter) Reset(context.Context, string) error {
	return nil
}

func newRouterRedisTestClient(t *testing.T) *redis.Client {
	t.Helper()

	address := os.Getenv("REDIS_TEST_ADDR")
	if address == "" {
		address = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: address,
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		if os.Getenv("REQUIRE_REDIS") == "1" {
			require.NoError(t, err)
		}

		t.Skipf("redis unavailable at %s: %v", address, err)
	}

	return client
}

func TestUserRegistrationLoginAndListFlow(t *testing.T) {

	// Set up Gin in test mode and configure the JWT secret for testing
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "integration-test-secret")

	// Set up an in-memory SQLite database for testing
	db, err := gorm.Open(
		sqlite.Open("file::memory:?cache=shared"),
		&gorm.Config{
			TranslateError: true,
		},
	)
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	sqlDB.SetMaxOpenConns(1) // Limit to 1 connection for in-memory SQLite

	// Auto-migrate the User model
	require.NoError(t, db.AutoMigrate(&models.User{}))

	userRepository := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepository)
	userService := service.NewUserService(userRepository)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	loginLimiter := &fakeLoginLimiter{}

	r := NewRouter(
		authHandler,
		userHandler,
		loginLimiter,
	)

	// 1. Test user registration and duplicate registration
	registerRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/register",
		strings.NewReader(`{
			"username": "johndoe",
			"email": "john@example.com",
			"password": "securepass123"
		}`),
	)
	registerRequest.Header.Set("Content-Type", "application/json")

	registerResponse := httptest.NewRecorder()
	r.ServeHTTP(registerResponse, registerRequest)

	require.Equal(t, http.StatusCreated, registerResponse.Code)
	assert.Contains(t, registerResponse.Body.String(), "johndoe")

	duplicateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/register",
		strings.NewReader(`{
			"username": "johndoe",
			"email": "john@example.com",
			"password": "securepass123"
		}`),
	)
	duplicateRequest.Header.Set("Content-Type", "application/json")

	duplicateResponse := httptest.NewRecorder()
	r.ServeHTTP(duplicateResponse, duplicateRequest)

	require.Equal(t, http.StatusConflict, duplicateResponse.Code)
	assert.Contains(t, duplicateResponse.Body.String(), `"code":"USER_ALREADY_EXISTS"`)

	// 2. Test login with the registered user
	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/login",
		strings.NewReader(`{
			"username": "johndoe",
			"password": "securepass123"
		}`),
	)
	loginRequest.Header.Set("Content-Type", "application/json")

	loginResponse := httptest.NewRecorder()
	r.ServeHTTP(loginResponse, loginRequest)

	require.Equal(t, http.StatusOK, loginResponse.Code)

	var loginBody struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(loginResponse.Body.Bytes(), &loginBody))
	require.NotEmpty(t, loginBody.Token)

	// 3. Test listing users with the obtained token
	listRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users",
		nil,
	)
	listRequest.Header.Set("Authorization", "Bearer "+loginBody.Token)

	listResponse := httptest.NewRecorder()
	r.ServeHTTP(listResponse, listRequest)

	require.Equal(t, http.StatusOK, listResponse.Code)
	assert.Contains(t, listResponse.Body.String(), "johndoe")
	assert.NotContains(t, listResponse.Body.String(), "securepass123")

}

func TestRouterLoginRateLimitBlocksBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loginLimiter := &fakeLoginLimiter{
		blocked: true,
	}

	r := NewRouter(
		handler.NewAuthHandler(nil),
		handler.NewUserHandler(nil),
		loginLimiter,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/login",
		strings.NewReader(`{"username":"alice","password":"wrongpassword"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, []string{"alice"}, loginLimiter.checkedKeys)
	require.Contains(t, recorder.Body.String(), `"code":"TOO_MANY_REQUESTS"`)
}

func TestRouterLoginRateLimitWithRealRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := newRouterRedisTestClient(t)
	ctx := context.Background()

	loginLimiter, err := ratelimit.NewRedisLoginLimiter(
		client,
		ratelimit.Options{
			KeyPrefix:     "test:router:" + t.Name(),
			MaxAttempts:   1,
			Window:        time.Minute,
			BlockDuration: time.Minute,
		},
	)
	require.NoError(t, err)

	account := "router-redis-user"

	require.NoError(t, loginLimiter.Reset(ctx, account))

	t.Cleanup(func() {
		require.NoError(t, loginLimiter.Reset(ctx, account))
	})

	authHandler := handler.NewAuthHandler(
		invalidCredentialsAuthService{},
	)

	router := NewRouter(
		authHandler,
		handler.NewUserHandler(nil),
		loginLimiter,
	)

	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/login",
		strings.NewReader(`{
            "username": "router-redis-user",
            "password": "wrongpassword"
        }`),
	)
	firstRequest.Header.Set("Content-Type", "application/json")

	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstRequest)

	require.Equal(t, http.StatusUnauthorized, firstResponse.Code)

	secondRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/login",
		strings.NewReader(`{
            "username": "router-redis-user",
            "password": "wrongpassword"
        }`),
	)
	secondRequest.Header.Set("Content-Type", "application/json")

	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, secondRequest)

	require.Equal(t, http.StatusTooManyRequests, secondResponse.Code)
	require.NotEmpty(t, secondResponse.Header().Get("Retry-After"))
	require.Contains(
		t,
		secondResponse.Body.String(),
		`"code":"TOO_MANY_REQUESTS"`,
	)
}
