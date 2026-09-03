package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-management/config"
	"user-management/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetUsers_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	// Intentionally drop the table so the SELECT query fails
	config.DB.Migrator().DropTable(&models.User{})

	r := gin.Default()
	r.GET("/users", GetUsers)

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetUsers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.GET("/users", GetUsers)

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["users"])
}

func TestGetProfileSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Simulating the middleware to inject data into the context
	r.GET("/profile", func(c *gin.Context) {
		c.Set("userID", "test-userID-1234")
		c.Set("Email", "test@example.com")
		GetProfile(c)
	})

	req, _ := http.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetProfileUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/profile", GetProfile) // No middleware, so it should return unauthorized

	req, _ := http.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateProfileUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PUT("/profile", UpdateProfile) // No middleware, so it should return unauthorized

	payload, _ := json.Marshal(map[string]string{"username": "newusername"})
	req, _ := http.NewRequest("PUT", "/profile", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateProfile_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.PUT("/profile", func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	}, UpdateProfile)

	// Send Invalid JSON missing the required "username" field
	badJSON := []byte(`{}`)
	req, _ := http.NewRequest("PUT", "/profile", bytes.NewBuffer(badJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.PUT("/profile", func(c *gin.Context) {
		c.Set("userID", uint(9999))
		c.Next()
	}, UpdateProfile)

	payload, _ := json.Marshal(map[string]string{"username": "newusername"})
	req, _ := http.NewRequest("PUT", "/profile", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateProfile_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	// Seed a test user
	user := models.User{Username: "erroruser", Email: "error@example.com", Password: "Password123"}
	config.DB.Create(&user)

	// intentionally drop the table to force an internal server error during the update
	config.DB.Migrator().DropTable(&models.User{})

	r := gin.Default()
	r.PUT("/profile", func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Next()
	}, UpdateProfile)

	payload := map[string]string{"username": "newusername"}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	// Seed a test user
	user := models.User{Username: "oldusername", Email: "update@example.com", Password: "password123"}
	config.DB.Create(&user)

	r := gin.Default()
	// Mock middleware to set userID in context
	r.PUT("/profile", func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Next()
	}, UpdateProfile)

	payload := map[string]string{"username": "newusername"}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
