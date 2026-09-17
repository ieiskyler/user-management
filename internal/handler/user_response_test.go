package handler

import (
	"testing"
	"time"
	"user-management/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestToUserSummaries(t *testing.T) {
	createdAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("maps all field and excludes password", func(t *testing.T) {
		users := []models.User{
			{
				ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Username:  "johndoe",
				Email:     "john@example.com",
				Password:  "hashedpassword",
				CreatedAt: createdAt,
			},
		}

		summaries := toUserSummaries(users)

		assert.Len(t, summaries, 1)
		assert.Equal(t, users[0].ID, summaries[0].ID)
		assert.Equal(t, users[0].Username, summaries[0].Username)
		assert.Equal(t, users[0].Email, summaries[0].Email)
		assert.Equal(t, users[0].CreatedAt, summaries[0].CreatedAt)
	})

	t.Run("empty input returns a non-nil empty slice", func(t *testing.T) {
		summaries := toUserSummaries([]models.User{})

		assert.NotNil(t, summaries)
		assert.Empty(t, summaries)
	})

	t.Run("nil input returns a non-nil empty slice", func(t *testing.T) {
		summaries := toUserSummaries(nil)

		assert.NotNil(t, summaries)
		assert.Empty(t, summaries)
	})
}
