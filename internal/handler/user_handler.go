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
	var query ListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			response.CodeInvalidRequest,
			"invalid pagination parameters",
		)
		return
	}
	query.Normalize()

	users, total, err := h.userService.GetAllUsers(query.Page, query.Limit)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			response.CodeFailedToRetrieveUsers,
			"failed to retrieve users",
		)
		return
	}
	totalPages := int((total + int64(query.Limit) - 1) / int64(query.Limit))

	c.JSON(http.StatusOK, UserListResponse{
		Users:      toUserSummaries(users),
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: totalPages,
	})
}
