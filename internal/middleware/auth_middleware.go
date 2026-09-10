package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"user-management/internal/config"
	"user-management/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(
				c,
				http.StatusUnauthorized,
				response.CodeMissingAuthorization,
				"authorization header is required",
			)
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(
				c,
				http.StatusUnauthorized,
				response.CodeInvalidAuthorization,
				"format must be: Bearer <token>",
			)
			c.Abort()
			return
		}

		tokenString := parts[1]
		secret, err := config.JWTSecret()
		if err != nil {
			response.Error(
				c,
				http.StatusInternalServerError,
				response.CodeMissingJWTConfig,
				"JWT configuration is missing",
			)
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			response.Error(
				c,
				http.StatusUnauthorized,
				response.CodeInvalidToken,
				"invalid or expired token",
			)
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(
				c,
				http.StatusUnauthorized,
				response.CodeInvalidTokenClaims,
				"invalid token claims",
			)
			c.Abort()
			return
		}

		userID, ok := claims["userID"].(string)
		if !ok {
			response.Error(
				c,
				http.StatusUnauthorized,
				response.CodeInvalidUserIDClaim,
				"invalid user ID claim",
			)
			c.Abort()
			return
		}

		if _, err := uuid.Parse(userID); err != nil {
			response.Error(
				c,
				http.StatusUnauthorized,
				response.CodeInvalidUserIDClaim,
				"invalid user ID claim",
			)
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
