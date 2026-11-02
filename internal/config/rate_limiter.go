package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type RedisSettings struct {
	Host     string
	Port     int
	Password string
	Database int
}

type AppConfig struct {
	RateLimiter RateLimiterConfig `mapstructure:"rate-limiter"`
}

type RateLimiterConfig struct {
	KeyPrefix     string `mapstructure:"key-prefix"`
	MaxAttempts   int    `mapstructure:"max-attempts"`
	Window        string `mapstructure:"window"`
	BlockDuration string `mapstructure:"block-duration"`
}

type ParsedRateLimiterConfig struct {
	KeyPrefix     string
	MaxAttempts   int
	Window        time.Duration
	BlockDuration time.Duration
}

func LoadRedisSettings() (RedisSettings, error) {
	port, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil || port <= 0 || port > 65535 {
		return RedisSettings{}, fmt.Errorf(
			"REDIS_PORT must be a valid TCP port",
		)
	}

	database, err := strconv.Atoi(os.Getenv("REDIS_DATABASE"))
	if err != nil || database < 0 {
		return RedisSettings{}, fmt.Errorf(
			"REDIS_DATABASE must be zero or greater",
		)
	}

	host := os.Getenv("REDIS_HOST")
	if host == "" {
		return RedisSettings{}, errors.New("REDIS_HOST is required")
	}

	return RedisSettings{
		Host:     host,
		Port:     port,
		Password: os.Getenv("REDIS_PASSWORD"),
		Database: database,
	}, nil
}

func LoadAppConfig() (AppConfig, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return AppConfig{}, fmt.Errorf("read config.yaml: %w", err)
	}

	var appConfig AppConfig
	if err := v.Unmarshal(&appConfig); err != nil {
		return AppConfig{}, fmt.Errorf("decode config.yaml: %w", err)
	}

	return appConfig, nil
}

func (c RateLimiterConfig) Parse() (ParsedRateLimiterConfig, error) {
	if c.KeyPrefix == "" {
		return ParsedRateLimiterConfig{}, fmt.Errorf(
			"rate limiter key-prefix must not be empty",
		)
	}

	if c.MaxAttempts <= 0 {
		return ParsedRateLimiterConfig{}, fmt.Errorf(
			"rate limiter max-attempts must be greater than 0",
		)
	}

	window, err := time.ParseDuration(c.Window)
	if err != nil || window <= 0 {
		return ParsedRateLimiterConfig{}, fmt.Errorf(
			"rate limiter window must be a positive duration: %q",
			c.Window,
		)
	}

	blockDuration, err := time.ParseDuration(c.BlockDuration)
	if err != nil || blockDuration <= 0 {
		return ParsedRateLimiterConfig{}, fmt.Errorf(
			"rate limiter block-duration must be a positive duration: %q",
			c.BlockDuration,
		)
	}

	return ParsedRateLimiterConfig{
		KeyPrefix:     c.KeyPrefix,
		MaxAttempts:   c.MaxAttempts,
		Window:        window,
		BlockDuration: blockDuration,
	}, nil
}
