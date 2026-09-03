package main

import (
	"log"
	"user-management/internal/config"
	"user-management/internal/handler"
	"user-management/internal/repository"
	"user-management/internal/server"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
)

var connectDatabase = config.ConnectDatabase
var runServer = func(router *gin.Engine) error { return router.Run(":8080") }

func main() {
	// 1. Initialize Infrastructure
	db := connectDatabase()

	// 2. Initialize Repositories (Data Layer)
	userRepo := repository.NewUserRepository(db)

	// 3. Initialize Services (Business Logic Layer)
	authService := service.NewAuthService(userRepo)
	userService := service.NewUserService(userRepo)

	// 4. Initialize Controllers (Transport Layer)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	r := server.NewRouter(authHandler, userHandler)

	log.Println("Server is starting on port 8080...")
	if err := runServer(r); err != nil {
		log.Fatal(err)
	}
}
