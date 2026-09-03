package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectDatabase_Failure(t *testing.T) {
	// Provide garbage credentials to force a connection failure
	os.Setenv("DB_HOST", "invalid_host")
	os.Setenv("DB_PORT", "9999")

	assert.Panics(t, func() {
		ConnectDatabase()
	})
}

func TestConnectDatabase_Success(t *testing.T) {
	// Provide valid credentials
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "skyler")
	os.Setenv("DB_PASSWORD", "")
	os.Setenv("DB_NAME", "user_management")
	os.Setenv("DB_PORT", "5432")

	// This should not panic if the database is reachable and credentials are correct
	assert.NotPanics(t, func() {
		ConnectDatabase()
	})

	// Ensure the Global DB variable is set
	assert.NotNil(t, DB)
}

func TestConnectDatabase_MigrationError(t *testing.T) {
	// Provide valid credentials
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "skyler")
	os.Setenv("DB_PASSWORD", "")
	os.Setenv("DB_NAME", "user_management")
	os.Setenv("DB_PORT", "5432")

	// Forcing AutoMigrate to fail with temporarily point to an invalid port
	os.Setenv("DB_PORT", "1") // Invalid port that will fail connection/migration

	assert.Panics(t, func() {
		ConnectDatabase()
	}, "Expected ConnectDatabase to panic on migration/connection failure")
}
