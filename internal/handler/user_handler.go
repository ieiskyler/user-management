package handler

import (
	"net/http"

	"user-management/internal/response"
	"user-management/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			response.CodeFailedToRetrieveUsers,
			"failed to retrieve users",
		)
		return
	}

	// Mapping to a strict response that excludes the hashed password
	var userResponses []map[string]interface{}
	for _, user := range users {
		userResponses = append(userResponses, map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"created_at": user.CreatedAt,
		})
	}

	// return empty array instead of null if no users found
	if userResponses == nil {
		userResponses = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{"users": userResponses})
}
