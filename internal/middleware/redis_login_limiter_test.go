package middleware

import (
	"context"
	"testing"
	"time"

	"user-management/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestNewRedisLoginLimiter(t *testing.T) {
	client := redis.NewClient(&redis.Options{})

	parsedConfig := config.ParsedRateLimiterConfig{
		KeyPrefix:     "user-management:login",
		MaxAttempts:   10,
		Window:        5 * time.Minute,
		BlockDuration: 5 * time.Minute,
	}

	limiter, err := NewRedisLoginLimiter(client, parsedConfig)

	require.NoError(t, err)
	require.NotNil(t, limiter)
	require.Equal(t, client, limiter.client)
	require.Equal(t, "user-management:login", limiter.keyPrefix)
	require.Equal(t, 10, limiter.maxAttempts)
	require.Equal(t, 5*time.Minute, limiter.window)
	require.Equal(t, 5*time.Minute, limiter.blockDuration)
}

func TestNewRedisLoginLimiterRejectsNilClient(t *testing.T) {
	parsedConfig := config.ParsedRateLimiterConfig{
		KeyPrefix:     "user-management:login",
		MaxAttempts:   10,
		Window:        5 * time.Minute,
		BlockDuration: 5 * time.Minute,
	}

	limiter, err := NewRedisLoginLimiter(nil, parsedConfig)

	require.Error(t, err)
	require.Nil(t, limiter)
}

func TestRedisLoginLimiterBlocksAfterMaximumFailures(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())

	limiter, err := NewRedisLoginLimiter(
		client,
		config.ParsedRateLimiterConfig{
			KeyPrefix:     "test:user-management:login",
			MaxAttempts:   2,
			Window:        time.Minute,
			BlockDuration: 10 * time.Second,
		},
	)
	require.NoError(t, err)

	account := t.Name()
	t.Cleanup(func() {
		require.NoError(t, limiter.Reset(ctx, account))
	})

	blocked, remaining, err := limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.False(t, blocked)
	require.Zero(t, remaining)

	require.NoError(t, limiter.RecordFailure(ctx, account))

	blocked, remaining, err = limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.False(t, blocked)
	require.Zero(t, remaining)

	require.NoError(t, limiter.RecordFailure(ctx, account))

	blocked, remaining, err = limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.True(t, blocked)
	require.Greater(t, remaining, time.Duration(0))
	require.LessOrEqual(t, remaining, 10*time.Second)

	require.NoError(t, limiter.RecordFailure(ctx, account))

	blockedAfterRepeatedFailure, remainingAfterRepeatedFailure, err :=
		limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.True(t, blockedAfterRepeatedFailure)
	require.LessOrEqual(
		t,
		remainingAfterRepeatedFailure,
		remaining,
	)
}

func TestRedisLoginLimiterAllowsNewFailuresAfterBlockExpires(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())

	limiter, err := NewRedisLoginLimiter(
		client,
		config.ParsedRateLimiterConfig{
			KeyPrefix:     "test:user-management:login-expiry",
			MaxAttempts:   1,
			Window:        time.Minute,
			BlockDuration: 50 * time.Millisecond,
		},
	)
	require.NoError(t, err)

	account := t.Name()
	t.Cleanup(func() {
		require.NoError(t, limiter.Reset(ctx, account))
	})

	require.NoError(t, limiter.RecordFailure(ctx, account))

	blocked, remaining, err := limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.True(t, blocked)
	require.Greater(t, remaining, time.Duration(0))

	time.Sleep(75 * time.Millisecond)

	blocked, remaining, err = limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.False(t, blocked)
	require.Zero(t, remaining)

	require.NoError(t, limiter.RecordFailure(ctx, account))

	blocked, remaining, err = limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.True(t, blocked)
	require.Greater(t, remaining, time.Duration(0))
}

func TestRedisLoginLimiterStartsFreshWindowAfterWindowExpires(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())

	limiter, err := NewRedisLoginLimiter(
		client,
		config.ParsedRateLimiterConfig{
			KeyPrefix:     "test:user-management:login-window",
			MaxAttempts:   3,
			Window:        50 * time.Millisecond,
			BlockDuration: time.Minute,
		},
	)
	require.NoError(t, err)

	account := t.Name()
	t.Cleanup(func() {
		require.NoError(t, limiter.Reset(ctx, account))
	})

	require.NoError(t, limiter.RecordFailure(ctx, account))

	time.Sleep(75 * time.Millisecond)

	require.NoError(t, limiter.RecordFailure(ctx, account))

	blocked, remaining, err := limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.False(t, blocked)
	require.Zero(t, remaining)
}

func TestRedisLoginLimiterResetClearsAccount(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())

	limiter, err := NewRedisLoginLimiter(
		client,
		config.ParsedRateLimiterConfig{
			KeyPrefix:     "test:user-management:login-reset",
			MaxAttempts:   2,
			Window:        time.Minute,
			BlockDuration: time.Minute,
		},
	)
	require.NoError(t, err)

	account := t.Name()
	t.Cleanup(func() {
		require.NoError(t, limiter.Reset(ctx, account))
	})

	require.NoError(t, limiter.RecordFailure(ctx, account))
	require.NoError(t, limiter.Reset(ctx, account))

	require.NoError(t, limiter.RecordFailure(ctx, account))

	blocked, remaining, err := limiter.IsBlocked(ctx, account)
	require.NoError(t, err)
	require.False(t, blocked)
	require.Zero(t, remaining)
}

func TestRedisLoginLimiterKeepsAccountsSeparate(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())

	limiter, err := NewRedisLoginLimiter(
		client,
		config.ParsedRateLimiterConfig{
			KeyPrefix:     "test:user-management:login-accounts",
			MaxAttempts:   2,
			Window:        time.Minute,
			BlockDuration: time.Minute,
		},
	)
	require.NoError(t, err)

	accountA := t.Name() + ":a"
	accountB := t.Name() + ":b"

	t.Cleanup(func() {
		require.NoError(t, limiter.Reset(ctx, accountA))
		require.NoError(t, limiter.Reset(ctx, accountB))
	})

	require.NoError(t, limiter.RecordFailure(ctx, accountA))
	require.NoError(t, limiter.RecordFailure(ctx, accountA))

	blockedA, _, err := limiter.IsBlocked(ctx, accountA)
	require.NoError(t, err)
	require.True(t, blockedA)

	blockedB, remainingB, err := limiter.IsBlocked(ctx, accountB)
	require.NoError(t, err)
	require.False(t, blockedB)
	require.Zero(t, remainingB)
}
