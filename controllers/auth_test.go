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
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Helper function to initialize an isolated in-memory test database
func setupTestDB() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}
	db.AutoMigrate(&models.User{})
	config.DB = db
}

func TestRegisterSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.POST("/register", Register)

	payload := RegisterInput{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: "password123",
	}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "User registered successfully", response["message"])
}

func TestRegisterDuplicateError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.POST("/register", Register)

	payload := RegisterInput{
		Username: "duplicateusre",
		Email:    "duplicateuser@example.com",
		Password: "password123",
	}
	jsonValue, _ := json.Marshal(payload)

	// First registration should succeed
	req1, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonValue))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	req2, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonValue))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestRegisterBindingError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/register", Register)

	// Send broken JSON (Missing closing quotes and brackets)
	badJSON := []byte(`{"username": "broken, "email":`)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(badJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterPasswordTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.POST("/register", Register)

	// Intentionally crashes bcrypt with a very long password (greater than 72 bytes)
	payload := map[string]string{
		"username": "longpassworduser",
		"email":    "long@example.com",
		"password": "a_very_long_password_that_exceeds_the_bcrypt_limit_of_72_bytes_" +
			"which_should_trigger_an_error_in_the_password_hashing_function_and_return_a_500_error",
	}
	jsonValue, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusCreated, w.Code)

}

func TestLoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.POST("/register", Register)
	r.POST("/login", Login)

	// Register user
	regPayload := RegisterInput{
		Username: "loginuser",
		Email:    "loginuser@example.com",
		Password: "password123",
	}
	regJSON, _ := json.Marshal(regPayload)
	reqReg, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(regJSON))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	r.ServeHTTP(wReg, reqReg)

	loginPayload := LoginInput{
		Email:    "loginuser@example.com",
		Password: "password123",
	}
	loginJSON, _ := json.Marshal(loginPayload)
	reqLogin, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginJSON))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, reqLogin)

	assert.Equal(t, http.StatusOK, wLogin.Code)

	var response map[string]interface{}
	json.Unmarshal(wLogin.Body.Bytes(), &response)
	assert.Equal(t, "Login successful", response["message"])
	assert.NotEmpty(t, response["token"])

}

func TestLoginInvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.POST("/login", Login)

	loginPayload := LoginInput{
		Email:    "nonexistent@example.com",
		Password: "nonexistent123",
	}
	loginJSON, _ := json.Marshal(loginPayload)
	reqLogin, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginJSON))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, reqLogin)

	assert.Equal(t, http.StatusUnauthorized, wLogin.Code)
}

func TestLoginInvalidPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	r := gin.Default()
	r.POST("/register", Register)
	r.POST("/login", Login)

	// Register User First
	regPayload := RegisterInput{
		Username: "wrongpassword",
		Email:    "wrongpassword@example.com",
		Password: "correctpassword123",
	}
	regJSON, _ := json.Marshal(regPayload)
	reqReg, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(regJSON))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	r.ServeHTTP(wReg, reqReg)

	// Attempt to login with wrong password

	loginPayload := LoginInput{
		Email:    "wrongpassword@example.com",
		Password: "wrongpassword123",
	}
	loginJSON, _ := json.Marshal(loginPayload)
	reqLogin, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginJSON))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, reqLogin)

	assert.Equal(t, http.StatusUnauthorized, wLogin.Code)
}

func TestLoginBindingError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/login", Login)

	// Send broken JSON (Missing closing quotes and brackets)
	badJSON := []byte(`{"email": "broken, "password":`)

	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(badJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
