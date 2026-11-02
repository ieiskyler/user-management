package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-management/internal/config"
	"user-management/internal/handler"
	"user-management/internal/middleware"
	"user-management/internal/repository"
	"user-management/internal/server"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var connectDatabase = config.ConnectDatabase

var connectRedis = func() (*redis.Client, error) {
	redisSettings, err := config.LoadRedisSettings()
	if err != nil {
		return nil, fmt.Errorf("Load Redis settings: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisSettings.Host, redisSettings.Port),
		Password: redisSettings.Password,
		DB:       redisSettings.Database,
	})

	pingContext, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := redisClient.Ping(pingContext).Err(); err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}

	return redisClient, nil
}

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
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
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
	// Initialize Infrastructure
	db, err := connectDatabase()
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	appConfig, err := config.LoadAppConfig()
	if err != nil {
		log.Fatal("failed to load application config:", err)
	}

	rateLimiterConfig, err := appConfig.RateLimiter.Parse()
	if err != nil {
		log.Fatal("invalid rate limiter configuration:", err)
	}

	redisClient, err := connectRedis()
	if err != nil {
		log.Fatal("failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	loginLimiter, err := middleware.NewRedisLoginLimiter(
		redisClient,
		rateLimiterConfig,
	)
	if err != nil {
		log.Fatal("failed to create login rate limiter:", err)
	}

	// Initialize Repositories (Data Layer)
	userRepo := repository.NewUserRepository(db)

	// Initialize Services (Business Logic Layer)
	authService := service.NewAuthService(userRepo)
	userService := service.NewUserService(userRepo)

	// Initialize Controllers (Transport Layer)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	r := server.NewRouter(authHandler, userHandler, loginLimiter)

	log.Println("Server is starting on port 8080...")
	if err := runServer(r); err != nil {
		log.Fatal(err)
	}
}
