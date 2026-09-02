package main

import (
	"fmt"
	"user-management/config"
	"user-management/controllers"
	"user-management/middlewares"

	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to the database
	config.ConnectDatabase()

	// Create a new Gin router
	router := gin.Default()

	// Define a simple health check route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Server and databases are alive",
		})
	})

	// Registration route
	router.POST("/register", controllers.Register)

	// Login route
	router.POST("/login", controllers.Login)

	// Protected route group
	protected := router.Group("/api")
	protected.Use(middlewares.AuthMiddleware()) // Applying the checkpoint
	{
		protected.GET("/profile", controllers.GetProfile)
	}

	fmt.Println("Server is running on http://localhost:8080")
	router.Run(":8080")
}
