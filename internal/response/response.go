package response

import "github.com/gin-gonic/gin"

type ErrorBody struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	CodeInvalidRequest        = "INVALID_REQUEST"
	CodeUserAlreadyExists     = "USER_ALREADY_EXISTS"
	CodeFailedToRegister      = "FAILED_TO_REGISTER"
	CodeInvalidCredentials    = "INVALID_CREDENTIALS"
	CodeMissingAuthorization  = "MISSING_AUTHORIZATION"
	CodeInvalidAuthorization  = "INVALID_AUTHORIZATION"
	CodeInvalidToken          = "INVALID_TOKEN"
	CodeMissingJWTConfig      = "MISSING_JWT_CONFIG"
	CodeInvalidTokenClaims    = "INVALID_TOKEN_CLAIMS"
	CodeInvalidUserIDClaim    = "INVALID_USER_ID_CLAIM"
	CodeFailedToRetrieveUsers = "FAILED_TO_RETRIEVE_USERS"
)

func Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, ErrorBody{
		Status:  status,
		Code:    code,
		Message: message,
	})
}
