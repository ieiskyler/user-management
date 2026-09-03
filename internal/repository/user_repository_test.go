package repository

import (
	"testing"
	"user-management/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestRepository(t *testing.T) UserRepository {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.User{}))
	return NewUserRepository(database)
}

func TestUserRepositoryCreateFindAndList(t *testing.T) {
	repository := newTestRepository(t)
	user := &models.User{Username: "johndoe", Email: "john@example.com", Password: "hashed"}

	require.NoError(t, repository.Create(user))
	assert.NotEqual(t, uuid.Nil, user.ID)

	found, err := repository.FindByUsername("johndoe")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)

	users, err := repository.FindAll()
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestUserRepositoryFindByUsernameMissing(t *testing.T) {
	repository := newTestRepository(t)

	user, err := repository.FindByUsername("missing")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Equal(t, uuid.Nil, user.ID)
}
