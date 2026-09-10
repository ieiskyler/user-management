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
	total int64
	err   error
}

func (mock *mockUserService) GetAllUsers(page, limit int) ([]models.User, int64, error) {
	return mock.users, mock.total, mock.err
}

func TestUserHandlerGetUsers(t *testing.T) {
	createdAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name           string
		queryString    string
		users          []models.User
		total          int64
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
			name:           "custom page and limit are echoed back",
			queryString:    "?page=2&limit=5",
			users:          []models.User{},
			total:          12,
			expectedStatus: http.StatusOK,
			expectedBody:   `"users":[],"page":2,"limit":5,"total":12,"total_pages":3`,
		},
		{
			name:           "service error",
			err:            errors.New("database unavailable"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"code":"FAILED_TO_RETRIEVE_USERS"`,
		},
		{
			name:           "invalid limit is rejected before hitting the service",
			queryString:    "?limit=abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"code":"INVALID_REQUEST"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewUserHandler(&mockUserService{users: test.users, total: test.total, err: test.err})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/users"+test.queryString, nil)

			handler.GetUsers(newTestContext(recorder, request))

			assert.Equal(t, test.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), test.expectedBody)
			assert.NotContains(t, recorder.Body.String(), "hashed")
		})
	}
}
