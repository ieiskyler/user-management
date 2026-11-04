package ratelimit

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newRedisTestClient(t *testing.T) *redis.Client {
	t.Helper()

	address := os.Getenv("REDIS_TEST_ADDR")
	if address == "" {
		address = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: address,
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		if os.Getenv("REQUIRE_REDIS") == "1" {
			require.NoError(t, err)
		}

		t.Skipf("Redis unavailable at %s: %v", address, err)
	}

	return client
}
func TestNewRedisLoginLimiter(t *testing.T) {
	client := redis.NewClient(&redis.Options{})

	options := Options{
		KeyPrefix:     "user-management:login",
		MaxAttempts:   10,
		Window:        5 * time.Minute,
		BlockDuration: 5 * time.Minute,
	}

	limiter, err := NewRedisLoginLimiter(client, options)

	require.NoError(t, err)
	require.NotNil(t, limiter)
	require.Equal(t, client, limiter.client)
	require.Equal(t, "user-management:login", limiter.keyPrefix)
	require.Equal(t, 10, limiter.maxAttempts)
	require.Equal(t, 5*time.Minute, limiter.window)
	require.Equal(t, 5*time.Minute, limiter.blockDuration)
}

func TestNewRedisLoginLimiterRejectsNilClient(t *testing.T) {
	options := Options{
		KeyPrefix:     "user-management:login",
		MaxAttempts:   10,
		Window:        5 * time.Minute,
		BlockDuration: 5 * time.Minute,
	}

	limiter, err := NewRedisLoginLimiter(nil, options)

	require.Error(t, err)
	require.Nil(t, limiter)
}

func TestRedisLoginLimiterBlocksAfterMaximumFailures(t *testing.T) {
	client := newRedisTestClient(t)
	ctx := context.Background()

	limiter, err := NewRedisLoginLimiter(
		client,
		Options{
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
	client := newRedisTestClient(t)
	ctx := context.Background()

	limiter, err := NewRedisLoginLimiter(
		client,
		Options{
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
	client := newRedisTestClient(t)
	ctx := context.Background()

	limiter, err := NewRedisLoginLimiter(
		client,
		Options{
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
	client := newRedisTestClient(t)
	ctx := context.Background()

	limiter, err := NewRedisLoginLimiter(
		client,
		Options{
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
	client := newRedisTestClient(t)
	ctx := context.Background()

	limiter, err := NewRedisLoginLimiter(
		client,
		Options{
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

func TestRedisKeyHashesAccount(t *testing.T) {
	limiter := &RedisLoginLimiter{
		keyPrefix: "test:user-management:login",
	}

	account := "alice"
	longAccount := strings.Repeat("x", 10000)

	key := limiter.redisKey(account)
	longKey := limiter.redisKey(longAccount)

	require.NotContains(t, key, account)
	require.NotContains(t, longKey, longAccount)
	require.LessOrEqual(t, len(longKey), len(limiter.keyPrefix)+1+64)
	require.NotEqual(t, key, longKey)
}
