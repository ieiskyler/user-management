package config

import (
	"log"
	"os"
	"user-management/internal/models"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDatabase() *gorm.DB {
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load()
	}

	dsn := "host=" + os.Getenv("DB_HOST") +
		" user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" port=" + os.Getenv("DB_PORT") +
		" sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		log.Panic("Failed to connect to database:", err)
	}
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Panic("Failed to migrate database:", err)
	}

	return db
}
