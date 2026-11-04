package config

import (
	"fmt"
	"time"
)

type RedisSettings struct {
	Host     string
	Port     int
	Password string
	Database int
}

type RateLimiterConfig struct {
	KeyPrefix     string
	MaxAttempts   int
	Window        time.Duration
	BlockDuration time.Duration
}

func LoadRedisSettings() (RedisSettings, error) {
	port, err := parseIntEnvironment("REDIS_PORT", 6379)
	if err != nil {
		return RedisSettings{}, err
	}
	if port > 65535 {
		return RedisSettings{}, fmt.Errorf(
			"REDIS_PORT must be a valid TCP port",
		)
	}

	database, err := parseNonNegativeIntEnvironment(
		"REDIS_DATABASE",
		0,
	)
	if err != nil {
		return RedisSettings{}, err
	}

	host := stringEnvironment("REDIS_HOST", "localhost")

	return RedisSettings{
		Host:     host,
		Port:     port,
		Password: stringEnvironment("REDIS_PASSWORD", ""),
		Database: database,
	}, nil
}

func LoadRateLimiterConfig() (RateLimiterConfig, error) {
	keyPrefix := stringEnvironment(
		"RATE_LIMIT_KEY_PREFIX",
		"user-management:login",
	)

	maxAttempts, err := parseIntEnvironment(
		"RATE_LIMIT_MAX_ATTEMPTS",
		10,
	)
	if err != nil {
		return RateLimiterConfig{}, err
	}

	window, err := parseDurationEnvironment(
		"RATE_LIMIT_WINDOW",
		5*time.Minute,
	)
	if err != nil {
		return RateLimiterConfig{}, err
	}

	blockDuration, err := parseDurationEnvironment(
		"RATE_LIMIT_BLOCK_DURATION",
		5*time.Minute,
	)
	if err != nil {
		return RateLimiterConfig{}, err
	}

	return RateLimiterConfig{
		KeyPrefix:     keyPrefix,
		MaxAttempts:   maxAttempts,
		Window:        window,
		BlockDuration: blockDuration,
	}, nil
}
