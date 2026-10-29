package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedWindowLimiterBlocksAfterMaximumFailures(t *testing.T) {
	now := time.Date(
		2026,
		time.January,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	limiter, err := NewFixedWindowLimiter(
		3,
		5*time.Minute,
		2*time.Minute,
	)
	require.NoError(t, err)

	limiter.now = func() time.Time {
		return now
	}

	limiter.RecordFailure("user")

	blocked, remaining := limiter.IsBlocked("user")
	assert.False(t, blocked)
	assert.Zero(t, remaining)

	limiter.RecordFailure("user")

	blocked, remaining = limiter.IsBlocked("user")
	assert.False(t, blocked)
	assert.Zero(t, remaining)

	limiter.RecordFailure("user")

	blocked, remaining = limiter.IsBlocked("user")
	assert.True(t, blocked)
	assert.Equal(t, 2*time.Minute, remaining)
}

func TestFixedWindowLimiterDoesNotExtendBlock(t *testing.T) {
	now := time.Date(
		2026,
		time.January,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	limiter, err := NewFixedWindowLimiter(
		2,
		5*time.Minute,
		2*time.Minute,
	)
	require.NoError(t, err)

	limiter.now = func() time.Time {
		return now
	}

	limiter.RecordFailure("user")

	now = now.Add(5 * time.Minute)

	blocked, remaining := limiter.IsBlocked("user")
	assert.False(t, blocked)
	assert.Zero(t, remaining)

	limiter.RecordFailure("user")

	blocked, remaining = limiter.IsBlocked("user")
	assert.False(t, blocked)
	assert.Zero(t, remaining)
}

func TestFixedWindowLimiterReset(t *testing.T) {
	limiter, err := NewFixedWindowLimiter(
		1,
		time.Minute,
		time.Minute,
	)
	require.NoError(t, err)

	limiter.RecordFailure("user")
	limiter.Reset("user")

	blocked, remaining := limiter.IsBlocked("user")

	assert.False(t, blocked)
	assert.Zero(t, remaining)
}
