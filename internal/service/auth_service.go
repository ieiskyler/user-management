package service

import (
	"errors"
	"os"
	"time"
	"user-management/internal/models"
	"user-management/internal/repository"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(user *models.User) error
	Login(username, password string) (string, error)
}

type authService struct {
	repo repository.UserRepository
}

func NewAuthService(repo repository.UserRepository) AuthService {
	return &authService{repo}
}

func (s *authService) Register(user *models.User) error {
	// Hash the password before saving to the database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("Failed to hash password")
	}
	user.Password = string(hashedPassword)

	return s.repo.Create(user)
}

func (s *authService) Login(username, password string) (string, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return "", errors.New("Invalid Username")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("Invalid Password")
	}

	// Generate JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"UserID": user.ID.String(),
		"exp":    time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "FallbackSecretKey" // Fallback secret key for local development
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", errors.New("Failed to generate token")
	}

	return tokenString, nil
}
