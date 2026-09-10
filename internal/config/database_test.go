package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectDatabaseReturnsErrorWhenConnectionFails(t *testing.T) {
	for key, value := range map[string]string{
		"DB_HOST":     "invalid_host",
		"DB_USER":     "test",
		"DB_PASSWORD": "test",
		"DB_NAME":     "test",
		"DB_PORT":     "5432",
	} {
		t.Setenv(key, value)
	}

	db, err := ConnectDatabase()

	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestDatabaseSSLModeUsesConfiguredValue(t *testing.T) {
	t.Setenv("DB_SSL_MODE", "require")

	assert.Equal(t, "require", databaseSSLMode())
}

func TestDatabaseSSLModeDefaultsToDisable(t *testing.T) {
	t.Setenv("DB_SSL_MODE", "")

	assert.Equal(t, "disable", databaseSSLMode())
}

func TestValidateDatabaseConfigReturnsErrorWhenVariableMissing(t *testing.T) {
	for _, variable := range requiredDatabaseVariables {
		t.Run(variable, func(t *testing.T) {
			for _, requiredVariable := range requiredDatabaseVariables {
				t.Setenv(requiredVariable, "test-value")
			}

			t.Setenv(variable, "")

			err := validateDatabaseConfig()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), variable)
		})
	}
}
