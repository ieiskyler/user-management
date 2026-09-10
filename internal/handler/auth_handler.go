package handler

import (
	"errors"
	"net/http"

	"user-management/internal/models"
	"user-management/internal/response"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var registerRequest RegisterRequest
	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			response.CodeInvalidRequest,
			"invalid registration request",
		)
		return
	}

	user := models.User{
		Username: registerRequest.Username,
		Email:    registerRequest.Email,
		Password: registerRequest.Password,
	}

	if err := h.authService.Register(&user); err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			response.Error(
				c,
				http.StatusConflict,
				response.CodeUserAlreadyExists,
				"user already exists",
			)
			return
		}

		response.Error(
			c,
			http.StatusInternalServerError,
			response.CodeFailedToRegister,
			"failed to register user",
		)
		return
	}

	// Return success the created user (excluding the password)
	c.JSON(http.StatusCreated, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	// Use an inline struct to strictly bind the expected JSON input
	var loginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			response.CodeInvalidRequest,
			"invalid login request",
		)
		return
	}

	token, err := h.authService.Login(loginRequest.Username, loginRequest.Password)
	if err != nil {
		response.Error(
			c,
			http.StatusUnauthorized,
			response.CodeInvalidCredentials,
			"invalid credentials",
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
