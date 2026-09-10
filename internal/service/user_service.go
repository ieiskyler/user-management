package service

import (
	"user-management/internal/models"
	"user-management/internal/repository"
)

type UserService interface {
	GetAllUsers(page, limit int) ([]models.User, int64, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo}
}

func (s *userService) GetAllUsers(page, limit int) ([]models.User, int64, error) {
	offset := (page - 1) * limit
	return s.repo.FindAll(offset, limit)
}
