package service

import (
	"errors"
	"testing"
	"user-management/internal/models"

	"github.com/stretchr/testify/assert"
)

// Targeted mock for testing list users logic
type MockUserListRepository struct {
	MockUsers      []models.User
	MockTotal      int64
	FindErr        error
	ReceivedOffset int
	ReceivedLimit  int
}

func (m *MockUserListRepository) Create(user *models.User) error { return nil }
func (m *MockUserListRepository) FindByUsername(username string) (*models.User, error) {
	return nil, nil
}

func (m *MockUserListRepository) FindAll(offset, limit int) ([]models.User, int64, error) {
	m.ReceivedOffset = offset
	m.ReceivedLimit = limit

	if m.FindErr != nil {
		return nil, 0, m.FindErr
	}

	return m.MockUsers, m.MockTotal, nil
}

func TestUserService_GetAllUsers(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		limit          int
		mockUsers      []models.User
		mockTotal      int64
		mockError      error
		expectedCount  int
		expectedOffset int
		expectedError  string
	}{
		{
			name:  "Success page 1",
			page:  1,
			limit: 10,
			mockUsers: []models.User{
				{Username: "user1", Email: "user1@example.com"},
				{Username: "user2", Email: "user2@example.com"},
			},
			mockTotal:      2,
			mockError:      nil,
			expectedCount:  2,
			expectedOffset: 0,
			expectedError:  "",
		},
		{
			name:           "Success page 2 computes correct offset",
			page:           2,
			limit:          10,
			mockUsers:      []models.User{},
			mockTotal:      2,
			mockError:      nil,
			expectedCount:  0,
			expectedOffset: 10,
			expectedError:  "",
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
				MockTotal: tt.mockTotal,
				FindErr:   tt.mockError,
			}
			svc := NewUserService(mockRepo)

			users, total, err := svc.GetAllUsers(tt.page, tt.limit)

			assert.Equal(t, tt.expectedOffset, mockRepo.ReceivedOffset)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, users)
			} else {
				assert.NoError(t, err)
				assert.Len(t, users, tt.expectedCount)
				assert.Equal(t, tt.mockTotal, total)
			}
		})
	}
}
