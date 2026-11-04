package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"user-management/internal/response"

	"github.com/gin-gonic/gin"
)

type LoginLimiter interface {
	IsBlocked(context.Context, string) (bool, time.Duration, error)
	RecordFailure(context.Context, string) error
	Reset(context.Context, string) error
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

			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				response.Error(
					c,
					http.StatusRequestEntityTooLarge,
					response.CodeInvalidRequest,
					"request body too large",
				)
				_ = c.Error(err)
				c.Abort()
				return
			}
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

		blocked, remaining, err := limiter.IsBlocked(
			c.Request.Context(),
			request.Username,
		)
		if err != nil {
			_ = c.Error(err)

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

		status := c.Writer.Status()
		ctx := context.WithoutCancel(c.Request.Context())

		switch status {
		case http.StatusUnauthorized:
			if err := limiter.RecordFailure(ctx, request.Username); err != nil {
				_ = c.Error(err)
			}
		case http.StatusOK:
			if err := limiter.Reset(ctx, request.Username); err != nil {
				_ = c.Error(err)
			}
		}
	}
}

func restoreRequestBody(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
}
