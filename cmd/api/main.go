package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-management/internal/config"
	"user-management/internal/handler"
	"user-management/internal/repository"
	"user-management/internal/server"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
)

var connectDatabase = config.ConnectDatabase

func handleServerError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

var runServer = func(router *gin.Engine) error {
	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return handleServerError(err)

	case <-shutdownContext.Done():
		gracefulContext, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		return httpServer.Shutdown(gracefulContext)
	}
}

func main() {
	// 1. Initialize Infrastructure
	db, err := connectDatabase()
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

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
