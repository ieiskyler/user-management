package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"user-management/internal/response"

	"github.com/gin-gonic/gin"
)

type LoginLimiter interface {
	IsBlocked(string) (bool, time.Duration, error)
	RecordFailure(string) error
	Reset(string) error
}

type loginRequest struct {
	Username string `json:"username"`
}

func LoginRateLimit(limiter LoginLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		const maxBodySize = 1 << 20 // 1 MiB

		c.Request.Body = http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			maxBodySize,
		)

		rawBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			restoreRequestBody(c, rawBody)
			c.Next()
			return
		}

		restoreRequestBody(c, rawBody)

		var request loginRequest
		if err := json.Unmarshal(rawBody, &request); err != nil {
			c.Next()
			return
		}

		if request.Username == "" {
			c.Next()
			return
		}

		blocked, remaining, err := limiter.IsBlocked(request.Username)
		if err != nil {
			response.Error(
				c,
				http.StatusInternalServerError,
				response.CodeFailedToLogin,
				"login rate limiter unavailable",
			)
			c.Abort()
			return
		}

		if blocked {
			retryAfter := int64(math.Ceil(remaining.Seconds()))

			c.Header(
				"Retry-After",
				strconv.FormatInt(retryAfter, 10),
			)

			response.Error(
				c,
				http.StatusTooManyRequests,
				response.CodeTooManyRequests,
				"too many login attempts",
			)
			c.Abort()
			return
		}

		c.Next()
	}
}

func restoreRequestBody(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
}
