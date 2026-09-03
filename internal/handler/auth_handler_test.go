package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"user-management/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockAuthService struct {
	registerError error
	loginToken    string
	loginError    error
	registered    *models.User
}

func (mock *mockAuthService) Register(user *models.User) error {
	if mock.registerError != nil {
		return mock.registerError
	}
	mock.registered = user
	return nil
}

func (mock *mockAuthService) Login(username, password string) (string, error) {
	return mock.loginToken, mock.loginError
}

func TestAuthHandlerRegister(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		registerError  error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "success",
			body:           `{"username":"johndoe","email":"john@example.com","password":"password123"}`,
			expectedStatus: http.StatusCreated,
			expectedBody:   `"username":"johndoe"`,
		},
		{
			name:           "invalid JSON",
			body:           `{invalid`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"error"`,
		},
		{
			name:           "service error",
			body:           `{"username":"johndoe","email":"john@example.com","password":"password123"}`,
			registerError:  errors.New("database unavailable"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"error":"failed to register user"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := &mockAuthService{registerError: test.registerError}
			handler := NewAuthHandler(mockService)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			handler.Register(newTestContext(recorder, request))

			assert.Equal(t, test.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), test.expectedBody)
			if test.registerError == nil && test.expectedStatus == http.StatusCreated {
				assert.Equal(t, "johndoe", mockService.registered.Username)
				assert.NotContains(t, recorder.Body.String(), "secret")
			}
		})
	}
}

func TestAuthHandlerLogin(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		loginToken     string
		loginError     error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "success",
			body:           `{"username":"johndoe","password":"password123"}`,
			loginToken:     "signed-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `"token":"signed-token"`,
		},
		{
			name:           "invalid JSON",
			body:           `{"username":"johndoe"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"error":"Invalid Username and Password"`,
		},
		{
			name:           "authentication error",
			body:           `{"username":"johndoe","password":"wrongpassword123"}`,
			loginError:     errors.New("invalid credentials"),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `"error":"invalid credentials"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := &mockAuthService{loginToken: test.loginToken, loginError: test.loginError}
			handler := NewAuthHandler(mockService)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			handler.Login(newTestContext(recorder, request))

			assert.Equal(t, test.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), test.expectedBody)
		})
	}
}

func newTestContext(recorder *httptest.ResponseRecorder, request *http.Request) *gin.Context {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return context
}
