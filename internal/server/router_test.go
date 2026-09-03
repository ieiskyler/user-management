package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"user-management/internal/handler"
	"user-management/internal/models"
	"user-management/internal/repository"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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

	r := NewRouter(authHandler, userHandler)

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
	assert.Contains(t, duplicateResponse.Body.String(), "user already exists")

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
