package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	validToken := signedTestToken(t, jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": "123e4567-e89b-12d3-a456-426614174000",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	expiredToken := signedTestToken(t, jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": "user-id",
		"exp":    time.Now().Add(-time.Hour).Unix(),
	})

	wrongAlgorithmToken := signedTestToken(t, jwt.SigningMethodHS384, jwt.MapClaims{
		"userID": "123e4567-e89b-12d3-a456-426614174000",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	tests := []struct {
		name           string
		authorization  string
		expectedStatus int
		expectedUserID any
	}{
		{name: "missing header", expectedStatus: http.StatusUnauthorized},
		{name: "invalid format", authorization: "Basic token", expectedStatus: http.StatusUnauthorized},
		{name: "invalid token", authorization: "Bearer invalid", expectedStatus: http.StatusUnauthorized},
		{name: "expired token", authorization: "Bearer " + expiredToken, expectedStatus: http.StatusUnauthorized},
		{name: "wrong algorithm token", authorization: "Bearer " + wrongAlgorithmToken, expectedStatus: http.StatusUnauthorized},
		{name: "valid token", authorization: "Bearer " + validToken, expectedStatus: http.StatusOK, expectedUserID: "123e4567-e89b-12d3-a456-426614174000"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.GET("/", func(c *gin.Context) {
				userID, exists := c.Get("userID")
				assert.True(t, exists)
				assert.Equal(t, test.expectedUserID, userID)
				c.Status(http.StatusOK)
			})

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", test.authorization)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assert.Equal(t, test.expectedStatus, recorder.Code)
		})
	}
}

func signedTestToken(t *testing.T, method jwt.SigningMethod, claims jwt.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	return tokenString
}
