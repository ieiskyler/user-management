package service

import (
	"user-management/internal/models"
	"user-management/internal/repository"
)

type UserService interface {
	GetAllUsers() ([]models.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo}
}

func (s *userService) GetAllUsers() ([]models.User, error) {
	return s.repo.FindAll()
}
