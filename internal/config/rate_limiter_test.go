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

func TestLoadRedisSettingsRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      string
		database  string
		errorText string
	}{
		{
			name:      "missing host",
			host:      "",
			port:      "6379",
			database:  "0",
			errorText: "REDIS_HOST",
		},
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

func TestLoadAppConfig(t *testing.T) {
	t.Chdir("../..")
	t.Setenv("REDIS_PASSWORD", "test-password")

	appConfig, err := LoadAppConfig()

	require.NoError(t, err)

	require.Equal(t, "user-management:login", appConfig.RateLimiter.KeyPrefix)
	require.Equal(t, 10, appConfig.RateLimiter.MaxAttempts)
	require.Equal(t, "5m", appConfig.RateLimiter.Window)
	require.Equal(t, "5m", appConfig.RateLimiter.BlockDuration)
}

func TestRateLimiterConfigParse(t *testing.T) {
	raw := RateLimiterConfig{
		KeyPrefix:     "user-management:login",
		MaxAttempts:   10,
		Window:        "5m",
		BlockDuration: "5m",
	}

	parsed, err := raw.Parse()

	require.NoError(t, err)
	require.Equal(t, "user-management:login", parsed.KeyPrefix)
	require.Equal(t, 10, parsed.MaxAttempts)
	require.Equal(t, 5*time.Minute, parsed.Window)
	require.Equal(t, 5*time.Minute, parsed.BlockDuration)
}

func TestRateLimiterConfigParseRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		config RateLimiterConfig
	}{
		{
			name: "empty key prefix",
			config: RateLimiterConfig{
				MaxAttempts:   10,
				Window:        "5m",
				BlockDuration: "5m",
			},
		},
		{
			name: "zero max attempts",
			config: RateLimiterConfig{
				KeyPrefix:     "login",
				MaxAttempts:   0,
				Window:        "5m",
				BlockDuration: "5m",
			},
		},
		{
			name: "invalid block duration",
			config: RateLimiterConfig{
				KeyPrefix:     "login",
				MaxAttempts:   10,
				Window:        "5m",
				BlockDuration: "invalid",
			},
		},
		{
			name: "invalid window",
			config: RateLimiterConfig{
				KeyPrefix:     "login",
				MaxAttempts:   10,
				Window:        "invalid",
				BlockDuration: "5m",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.config.Parse()
			require.Error(t, err)
		})
	}
}
