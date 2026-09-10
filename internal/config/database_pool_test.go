package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabasePoolConfigUsesDefaults(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")

	poolConfig, err := loadDatabasePoolConfig()

	require.NoError(t, err)
	assert.Equal(t, 10, poolConfig.maxOpenConnections)
	assert.Equal(t, 5, poolConfig.maxIdleConnections)
	assert.Equal(t, time.Hour, poolConfig.connectionMaxLifetime)
}

func TestDatabasePoolConfigParsesEnvironmentValues(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "20")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")
	t.Setenv("DB_CONN_MAX_LIFETIME", "30m")

	poolConfig, err := loadDatabasePoolConfig()

	require.NoError(t, err)
	assert.Equal(t, 20, poolConfig.maxOpenConnections)
	assert.Equal(t, 10, poolConfig.maxIdleConnections)
	assert.Equal(t, 30*time.Minute, poolConfig.connectionMaxLifetime)
}

func TestDatabasePoolConfigRejectsInvalidDuration(t *testing.T) {
	t.Setenv("DB_CONN_MAX_LIFETIME", "invalid_duration")

	_, err := loadDatabasePoolConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DB_CONN_MAX_LIFETIME")
}

func TestDatabasePoolConfigRejectsIdleConnectionsAboveOpenConnections(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "5")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")

	_, err := loadDatabasePoolConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DB_MAX_IDLE_CONNS cannot exceed DB_MAX_OPEN_CONNS")
}

func TestDatabasePoolConfigRejectsInvalidInteger(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "non-integer value",
			value: "not_an_integer",
		},
		{
			name:  "zero value",
			value: "0",
		},
		{
			name:  "negative value",
			value: "-5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DB_MAX_OPEN_CONNS", test.value)

			_, err := loadDatabasePoolConfig()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "DB_MAX_OPEN_CONNS")
		})
	}
}
