package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"user-management/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Format must be: Bearer <token>"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		secret, err := config.JWTSecret()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "JWT Configuration is missing"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		userID, ok := claims["userID"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID claim"})
			c.Abort()
			return
		}

		if _, err := uuid.Parse(userID); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID claim"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
