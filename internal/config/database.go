package config

import (
	"errors"
	"os"
	"user-management/internal/models"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var requiredDatabaseVariables = []string{
	"DB_HOST",
	"DB_USER",
	"DB_NAME",
	"DB_PORT",
}

func validateDatabaseConfig() error {
	for _, variable := range requiredDatabaseVariables {
		if os.Getenv(variable) == "" {
			return errors.New("required database configuration is missing: " + variable)
		}
	}
	return nil
}

func databaseSSLMode() string {
	sslMode := os.Getenv("DB_SSL_MODE")
	if sslMode == "" {
		return "disable"
	}
	return sslMode
}

func ConnectDatabase() (*gorm.DB, error) {
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load()
	}

	if err := validateDatabaseConfig(); err != nil {
		return nil, err
	}

	dsn := "host=" + os.Getenv("DB_HOST") +
		" user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" port=" + os.Getenv("DB_PORT") +
		" sslmode=" + databaseSSLMode()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
