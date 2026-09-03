package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestUserBeforeCreate(t *testing.T) {
	user := User{}

	// Trigger the BeforeCreate hook
	err := user.BeforeCreate(&gorm.DB{})

	assert.NoError(t, err)
	assert.NotEmpty(t, user.ID) // ensures the UUID is generated
}
