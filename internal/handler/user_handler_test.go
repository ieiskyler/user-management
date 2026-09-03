package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"user-management/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockUserService struct {
	users []models.User
	err   error
}

func (mock *mockUserService) GetAllUsers() ([]models.User, error) {
	return mock.users, mock.err
}

func TestUserHandlerGetUsers(t *testing.T) {
	createdAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name           string
		users          []models.User
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			users: []models.User{{
				ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Username: "johndoe", Email: "john@example.com", Password: "hashed", CreatedAt: createdAt,
			}},
			expectedStatus: http.StatusOK,
			expectedBody:   `"username":"johndoe"`,
		},
		{
			name:           "empty result",
			expectedStatus: http.StatusOK,
			expectedBody:   `"users":[]`,
		},
		{
			name:           "service error",
			err:            errors.New("database unavailable"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"error":"Failed to retrieve users"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewUserHandler(&mockUserService{users: test.users, err: test.err})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

			handler.GetUsers(newTestContext(recorder, request))

			assert.Equal(t, test.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), test.expectedBody)
			assert.NotContains(t, recorder.Body.String(), "hashed")
		})
	}
}
