package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Expecting format with "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format. Format must be Bearer {token}"})
			c.Abort()
			return
		}

		// Parse and validate the JWT Token
		secretKey := os.Getenv("JWT_SECRET")
		if secretKey == "" {
			secretKey = "supersecretkey" // Fallback secret key for local development
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method ensuring it using HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract claims and set them in the context for further use (next handlers)
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("userID", claims["sub"])
			c.Set("email", claims["email"])
		}

		// Proceed to the next handler
		c.Next()
	}
}
