package repository

import (
	"strconv"
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
}

func TestUserRepositoryFindAll(t *testing.T) {
	seedUsers := func(t *testing.T, repository UserRepository, count int) {
		t.Helper()
		for i := 0; i < count; i++ {
			user := &models.User{
				Username: "user" + strconv.Itoa(i),
				Email:    "user" + strconv.Itoa(i) + "@example.com",
				Password: "hashed",
			}
			require.NoError(t, repository.Create(user))
		}
	}

	t.Run("empty table", func(t *testing.T) {
		repository := newTestRepository(t)

		users, total, err := repository.FindAll(0, 10)

		require.NoError(t, err)
		assert.Empty(t, users)
		assert.Equal(t, int64(0), total)
	})

	t.Run("more rows than limit", func(t *testing.T) {
		repository := newTestRepository(t)
		seedUsers(t, repository, 3)

		users, total, err := repository.FindAll(0, 2)

		require.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, int64(3), total)
	})

	t.Run("offset past the end", func(t *testing.T) {
		repository := newTestRepository(t)
		seedUsers(t, repository, 2)

		users, total, err := repository.FindAll(10, 10)

		require.NoError(t, err)
		assert.Empty(t, users)
		assert.Equal(t, int64(2), total)
	})

	t.Run("returns error when query fails", func(t *testing.T) {
		repository := newTestRepository(t)
		underlying := repository.(*userRepository)
		require.NoError(t, underlying.db.Migrator().DropTable(&models.User{}))

		users, total, err := repository.FindAll(0, 10)

		assert.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, int64(0), total)
	})
}

func TestUserRepositoryFindByUsernameMissing(t *testing.T) {
	repository := newTestRepository(t)

	user, err := repository.FindByUsername("missing")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Equal(t, uuid.Nil, user.ID)
}
