package service

import (
	"errors"
	"testing"
	"user-management/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// Mock Repository setup
type MockUserRepository struct {
	MockUsers map[string]*models.User
	CreateErr error
}

func (m *MockUserRepository) Create(user *models.User) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.MockUsers[user.Username] = user
	return nil
}

func (m *MockUserRepository) FindByUsername(username string) (*models.User, error) {
	if user, exists := m.MockUsers[username]; exists {
		return user, nil
	}
	return nil, errors.New("User not found")
}

func (m *MockUserRepository) FindAll() ([]models.User, error) {
	return nil, nil // unused in this test
}

// Test for Register function
func TestAuthService_Register(t *testing.T) {
	mockRepo := &MockUserRepository{MockUsers: make(map[string]*models.User)}
	svc := NewAuthService(mockRepo)

	tests := []struct {
		name          string
		user          *models.User
		mockError     error
		expectedError string
	}{
		{
			name:          "Success",
			user:          &models.User{Username: "newuser", Password: "password123"},
			mockError:     nil,
			expectedError: "",
		},
		{
			name:          "Database Error",
			user:          &models.User{Username: "erroruser", Password: "password123"},
			mockError:     errors.New("Database connection failed"),
			expectedError: "Database connection failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Inject Conditional mock Error
			mockRepo.CreateErr = tt.mockError

			err := svc.Register(tt.user)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			} else {
				assert.NoError(t, err)

				// Verify the business logic: Password should be hashed not plain text
				savedUser := mockRepo.MockUsers[tt.user.Username]
				assert.NotEqual(t, "password123", savedUser.Password)
			}
		})
	}
}

// Login Service Test
func TestAuthService_Login(t *testing.T) {
	t.Setenv("JWT_SECRET", "service-test-secret")

	// Pre-seed a valid user for login tests
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("securepass123"), bcrypt.DefaultCost)
	mockRepo := &MockUserRepository{
		MockUsers: map[string]*models.User{
			"johndoe": {
				ID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Username: "johndoe",
				Password: string(hashedPassword),
			},
		},
	}
	svc := NewAuthService(mockRepo)

	tests := []struct {
		name          string
		username      string
		password      string
		expectedError string
	}{
		{
			name:          "Success",
			username:      "johndoe",
			password:      "securepass123",
			expectedError: "",
		},
		{
			name:          "InvalidUsername",
			username:      "janedoe",
			password:      "securepass123",
			expectedError: "invalid credentials",
		},
		{
			name:          "InvalidPassword",
			username:      "johndoe",
			password:      "wrongpassword",
			expectedError: "invalid credentials",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.Login(tt.username, tt.password)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}
