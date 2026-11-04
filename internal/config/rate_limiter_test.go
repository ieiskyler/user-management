package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadRedisSettings(t *testing.T) {
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_PASSWORD", "test-password")
	t.Setenv("REDIS_DATABASE", "2")

	settings, err := LoadRedisSettings()

	require.NoError(t, err)
	require.Equal(t, "redis", settings.Host)
	require.Equal(t, 6379, settings.Port)
	require.Equal(t, "test-password", settings.Password)
	require.Equal(t, 2, settings.Database)
}

func TestLoadRedisSettingsUsesDefaults(t *testing.T) {
	t.Setenv("REDIS_HOST", "")
	t.Setenv("REDIS_PORT", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DATABASE", "")

	settings, err := LoadRedisSettings()

	require.NoError(t, err)
	require.Equal(t, "localhost", settings.Host)
	require.Equal(t, 6379, settings.Port)
	require.Empty(t, settings.Password)
	require.Equal(t, 0, settings.Database)
}

func TestLoadRedisSettingsRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      string
		database  string
		errorText string
	}{
		{
			name:      "invalid port",
			host:      "redis",
			port:      "invalid",
			database:  "0",
			errorText: "REDIS_PORT",
		},
		{
			name:      "zero port",
			host:      "redis",
			port:      "0",
			database:  "0",
			errorText: "REDIS_PORT",
		},
		{
			name:      "port too large",
			host:      "redis",
			port:      "70000",
			database:  "0",
			errorText: "REDIS_PORT",
		},
		{
			name:      "invalid database",
			host:      "redis",
			port:      "6379",
			database:  "-1",
			errorText: "REDIS_DATABASE",
		},
		{
			name:      "non-integer database",
			host:      "redis",
			port:      "6379",
			database:  "invalid",
			errorText: "REDIS_DATABASE",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("REDIS_HOST", test.host)
			t.Setenv("REDIS_PORT", test.port)
			t.Setenv("REDIS_PASSWORD", "")
			t.Setenv("REDIS_DATABASE", test.database)

			_, err := LoadRedisSettings()

			require.Error(t, err)
			require.True(
				t,
				strings.Contains(err.Error(), test.errorText),
				"error %q should contain %q",
				err.Error(),
				test.errorText,
			)
		})
	}
}

func TestLoadRateLimiterConfigUsesDefaults(t *testing.T) {
	t.Setenv("RATE_LIMIT_KEY_PREFIX", "")
	t.Setenv("RATE_LIMIT_MAX_ATTEMPTS", "")
	t.Setenv("RATE_LIMIT_WINDOW", "")
	t.Setenv("RATE_LIMIT_BLOCK_DURATION", "")

	rateLimiterConfig, err := LoadRateLimiterConfig()

	require.NoError(t, err)
	require.Equal(t, "user-management:login", rateLimiterConfig.KeyPrefix)
	require.Equal(t, 10, rateLimiterConfig.MaxAttempts)
	require.Equal(t, 5*time.Minute, rateLimiterConfig.Window)
	require.Equal(t, 5*time.Minute, rateLimiterConfig.BlockDuration)
}

func TestLoadRateLimiterConfig(t *testing.T) {
	t.Setenv("RATE_LIMIT_KEY_PREFIX", "test:login")
	t.Setenv("RATE_LIMIT_MAX_ATTEMPTS", "3")
	t.Setenv("RATE_LIMIT_WINDOW", "30s")
	t.Setenv("RATE_LIMIT_BLOCK_DURATION", "2m")

	rateLimiterConfig, err := LoadRateLimiterConfig()

	require.NoError(t, err)
	require.Equal(t, "test:login", rateLimiterConfig.KeyPrefix)
	require.Equal(t, 3, rateLimiterConfig.MaxAttempts)
	require.Equal(t, 30*time.Second, rateLimiterConfig.Window)
	require.Equal(t, 2*time.Minute, rateLimiterConfig.BlockDuration)
}

func TestLoadRateLimiterConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		variable  string
		value     string
		errorText string
	}{
		{
			name:      "invalid max attempts",
			variable:  "RATE_LIMIT_MAX_ATTEMPTS",
			value:     "invalid",
			errorText: "RATE_LIMIT_MAX_ATTEMPTS",
		},
		{
			name:      "zero max attempts",
			variable:  "RATE_LIMIT_MAX_ATTEMPTS",
			value:     "0",
			errorText: "RATE_LIMIT_MAX_ATTEMPTS",
		},
		{
			name:      "invalid window",
			variable:  "RATE_LIMIT_WINDOW",
			value:     "invalid",
			errorText: "RATE_LIMIT_WINDOW",
		},
		{
			name:      "invalid block duration",
			variable:  "RATE_LIMIT_BLOCK_DURATION",
			value:     "invalid",
			errorText: "RATE_LIMIT_BLOCK_DURATION",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.variable, test.value)

			_, err := LoadRateLimiterConfig()

			require.Error(t, err)
			require.Contains(t, err.Error(), test.errorText)
		})
	}
}

func TestRateLimiterConfigValidate(t *testing.T) {
	validConfig := RateLimiterConfig{
		KeyPrefix:     "user-management:login",
		MaxAttempts:   10,
		Window:        5 * time.Minute,
		BlockDuration: 5 * time.Minute,
	}

	require.NoError(t, validConfig.Validate())

	tests := []struct {
		name   string
		config RateLimiterConfig
	}{
		{
			name: "empty key prefix",
			config: RateLimiterConfig{
				MaxAttempts:   10,
				Window:        5 * time.Minute,
				BlockDuration: 5 * time.Minute,
			},
		},
		{
			name: "zero max attempts",
			config: RateLimiterConfig{
				KeyPrefix:     "login",
				Window:        5 * time.Minute,
				BlockDuration: 5 * time.Minute,
			},
		},
		{
			name: "zero window",
			config: RateLimiterConfig{
				KeyPrefix:     "login",
				MaxAttempts:   10,
				BlockDuration: 5 * time.Minute,
			},
		},
		{
			name: "zero block duration",
			config: RateLimiterConfig{
				KeyPrefix:   "login",
				MaxAttempts: 10,
				Window:      5 * time.Minute,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, test.config.Validate())
		})
	}
}
