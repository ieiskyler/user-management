package main

import (
	"fmt"
	"user-management/config"

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
			"status": "healthy",
		})
	})

	fmt.Println("Server is running on http://localhost:8080")
	router.Run(":8080")
}
