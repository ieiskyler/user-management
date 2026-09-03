package main

import (
	"log"
	"user-management/internal/config"
	"user-management/internal/handler"
	"user-management/internal/middleware"
	"user-management/internal/repository"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Initialize Infrastructure
	db := config.ConnectDatabase()

	// 2. Initialize Repositories (Data Layer)
	userRepo := repository.NewUserRepository(db)

	// 3. Initialize Services (Business Logic Layer)
	authService := service.NewAuthService(userRepo)
	userService := service.NewUserService(userRepo)

	// 4. Initialize Controllers (Transport Layer)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	// 5. Initialize Router
	r := gin.Default()

	// API v1 Group
	v1 := r.Group("/api/v1")
	{
		// Public Routes
		v1.POST("/register", authHandler.Register)
		v1.POST("/login", authHandler.Login)

		// Protected Routes
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/users", userHandler.GetUsers)
		}
	}

	log.Println("Server is starting on port 8080...")
	r.Run(":8080")
}
