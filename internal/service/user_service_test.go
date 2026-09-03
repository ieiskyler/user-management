package service

import (
	"errors"
	"testing"
	"user-management/internal/models"

	"github.com/stretchr/testify/assert"
)

// Targeted mock for testing list users logic
type MockUserListRepository struct {
	MockUsers []models.User
	FindErr   error
}

func (m *MockUserListRepository) Create(user *models.User) error { return nil }
func (m *MockUserListRepository) FindByUsername(username string) (*models.User, error) {
	return nil, nil
}

func (m *MockUserListRepository) FindAll() ([]models.User, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	return m.MockUsers, nil
}

func TestUserService_GetAllUsers(t *testing.T) {
	tests := []struct {
		name          string
		mockUsers     []models.User
		mockError     error
		expectedCount int
		expectedError string
	}{
		{
			name: "Success",
			mockUsers: []models.User{
				{Username: "user1", Email: "user1@example.com"},
				{Username: "user2", Email: "user2@example.com"},
			},
			mockError:     nil,
			expectedCount: 2,
			expectedError: "",
		},
		{
			name:          "Database Error",
			mockUsers:     nil,
			mockError:     errors.New("connection lost"),
			expectedCount: 0,
			expectedError: "connection lost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUserListRepository{
				MockUsers: tt.mockUsers,
				FindErr:   tt.mockError,
			}
			svc := NewUserService(mockRepo)

			users, err := svc.GetAllUsers()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, users)
			} else {
				assert.NoError(t, err)
				assert.Len(t, users, tt.expectedCount)
			}
		})
	}
}
