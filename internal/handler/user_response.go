package handler

import (
	"time"
	"user-management/internal/models"

	"github.com/google/uuid"
)

type UserSummary struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserListResponse struct {
	Users      []UserSummary `json:"users"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	Total      int64         `json:"total"`
	TotalPages int           `json:"total_pages"`
}

func toUserSummaries(users []models.User) []UserSummary {
	summaries := make([]UserSummary, 0, len(users))
	for _, user := range users {
		summaries = append(summaries, UserSummary{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		})
	}
	return summaries
}
