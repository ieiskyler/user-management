package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	Error(
		context,
		http.StatusUnauthorized,
		"INVALID_CREDENTIALS",
		"invalid credentials",
	)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	var body ErrorBody
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	require.Equal(t, http.StatusUnauthorized, body.Status)
	require.Equal(t, "INVALID_CREDENTIALS", body.Code)
	require.Equal(t, "invalid credentials", body.Message)
}
