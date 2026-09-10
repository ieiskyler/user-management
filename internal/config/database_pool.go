package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type databasePoolConfig struct {
	maxOpenConnections    int
	maxIdleConnections    int
	connectionMaxLifetime time.Duration
}

func loadDatabasePoolConfig() (databasePoolConfig, error) {
	maxOpenConnections, err := parseIntEnvironment(
		"DB_MAX_OPEN_CONNS",
		10,
	)
	if err != nil {
		return databasePoolConfig{}, err
	}

	maxIdleConnections, err := parseIntEnvironment(
		"DB_MAX_IDLE_CONNS",
		5,
	)
	if err != nil {
		return databasePoolConfig{}, err
	}

	connectionMaxLifetime, err := parseDurationEnvironment(
		"DB_CONN_MAX_LIFETIME",
		time.Hour,
	)
	if err != nil {
		return databasePoolConfig{}, err
	}

	if maxIdleConnections > maxOpenConnections {
		return databasePoolConfig{}, fmt.Errorf(
			"DB_MAX_IDLE_CONNS cannot exceed DB_MAX_OPEN_CONNS",
		)
	}

	return databasePoolConfig{
		maxOpenConnections:    maxOpenConnections,
		maxIdleConnections:    maxIdleConnections,
		connectionMaxLifetime: connectionMaxLifetime,
	}, nil
}

func parseIntEnvironment(name string, defaultValue int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil || parsedValue <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}

	return parsedValue, nil
}

func parseDurationEnvironment(name string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}

	parsedValue, err := time.ParseDuration(value)
	if err != nil || parsedValue <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}

	return parsedValue, nil
}
