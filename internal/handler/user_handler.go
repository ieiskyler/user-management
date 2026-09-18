package handler

import (
	"net/http"
	"strconv"

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

	if rawPage, exists := c.GetQuery("page"); exists {
		page, err := strconv.Atoi(rawPage)
		if err != nil || page < 1 {
			response.Error(
				c,
				http.StatusBadRequest,
				response.CodeInvalidRequest,
				"invalid pagination parameters",
			)
			return
		}
	}

	if rawLimit, exists := c.GetQuery("limit"); exists {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit < 1 || limit > maxLimit {
			response.Error(
				c,
				http.StatusBadRequest,
				response.CodeInvalidRequest,
				"invalid pagination parameters",
			)
			return
		}
	}

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

	if err := query.Validate(); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			response.CodeInvalidRequest,
			"invalid pagination parameters",
		)
		return
	}

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
