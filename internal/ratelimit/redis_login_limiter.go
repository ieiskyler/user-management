package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// isBlockedScript reports whether an account is currently blocked.
// KEYS[1] = account key; ARGV[1] = counting window in milliseconds.
// Returns {1, remainingMs} when blocked, otherwise {0, 0}.
// It never records failures; it only reads state and removes expired entries.
var isBlockedScript = redis.NewScript(`
-- Get the current state of the account from Redis.
local values = redis.call(
	"HMGET",
	KEYS[1],
	"window_start_ms",
	"blocked_until_ms"
)

-- if the account does not exist, it is not blocked.
if values[1] == false then
	return {0, 0}
end

-- convert the current time to milliseconds since epoch.
local timeParts = redis.call("TIME")
local nowMs = 
	tonumber(timeParts[1]) * 1000 +
	math.floor(tonumber(timeParts[2]) / 1000)


-- parse the window start and blocked until times from Redis.
local windowStartMs = tonumber(values[1]) or 0
local blockedUntilMs = tonumber(values[2]) or 0

-- still blocked if the current time is less than the blocked until time.
if blockedUntilMs > 0 then
	if nowMs < blockedUntilMs then
		return {1, blockedUntilMs - nowMs}
	end

	redis.call("DEL", KEYS[1])
	return {0, 0}
end

-- if the window has expired, delete the account state and return not blocked.
if nowMs - windowStartMs >= tonumber(ARGV[1]) then
	redis.call("DEL", KEYS[1])
	return {0, 0}
end

return {0, 0}
`)

// recordFailureScript counts one failed login atomically.
// Redis hash fields: count, window_start_ms, blocked_until_ms.
// KEYS[1] = account key; ARGV = window ms, max attempts, block ms.
// Returns the new failure count, or 0 if the account is already blocked.
// The window starts at the first failure and does not move; the block
// starts at the failure that reaches max attempts.
var recordFailureScript = redis.NewScript(`
-- Get the current state of the account from Redis.
local values = redis.call(
	"HMGET",
	KEYS[1],
	"count",
	"window_start_ms",
	"blocked_until_ms"
)

-- convert the current time to milliseconds since epoch.
local timeParts = redis.call("TIME")
local nowMs =
	tonumber(timeParts[1]) * 1000 +
	math.floor(tonumber(timeParts[2]) / 1000)

-- parse the current state from Redis.
local count = tonumber(values[1]) or 0
local windowStartMs = tonumber(values[2]) or 0
local blockedUntilMs = tonumber(values[3]) or 0
local windowMs = tonumber(ARGV[1])
local maxAttempts = tonumber(ARGV[2])
local blockDurationMs = tonumber(ARGV[3])

-- if the account is currently blocked, return 0 to indicate no new failure was recorded.
if blockedUntilMs > 0 and nowMs < blockedUntilMs then
	return 0
end

-- if the window has expired, reset the count and window start time.
if windowStartMs == 0 or nowMs - windowStartMs >= windowMs then
	count = 1
	windowStartMs = nowMs

	if count >= maxAttempts then
		blockedUntilMs = nowMs + blockDurationMs

		redis.call(
			"HSET",
			KEYS[1],
			"count", count,
			"window_start_ms", windowStartMs,
			"blocked_until_ms", blockedUntilMs
		)
		redis.call("PEXPIRE", KEYS[1], blockDurationMs)
		
		return count
	end

	redis.call(
		"HSET",
		KEYS[1],
		"count", count,
		"window_start_ms", windowStartMs,
		"blocked_until_ms", 0
	)
	redis.call("PEXPIRE", KEYS[1], windowMs)
	
	return count
end

-- if the account is not blocked and the window has not expired, increment the count.
count = count + 1


-- if the count reaches max attempts, set the blocked until time and expire the key.
if count >= maxAttempts then
	blockedUntilMs = nowMs + blockDurationMs

	redis.call(
		"HSET",
		KEYS[1],
		"count", count,
		"blocked_until_ms", blockedUntilMs
	)
	redis.call("PEXPIRE", KEYS[1], blockDurationMs)

	return count
end

redis.call("HSET", KEYS[1], "count", count)

return count
`)

// Options configures a Redis-backed login limiter.
type Options struct {
	KeyPrefix     string
	MaxAttempts   int
	Window        time.Duration
	BlockDuration time.Duration
}

// RedisLoginLimiter stores login failure state in Redis.
type RedisLoginLimiter struct {
	client        *redis.Client
	keyPrefix     string
	maxAttempts   int
	window        time.Duration
	blockDuration time.Duration
}

// NewRedisLoginLimiter creates a Redis-backed login limiter.
func NewRedisLoginLimiter(client *redis.Client, options Options) (*RedisLoginLimiter, error) {
	if client == nil {
		return nil, errors.New("redis client must not be nil")
	}

	if options.KeyPrefix == "" {
		return nil, errors.New("key prefix must not be empty")
	}

	if options.MaxAttempts <= 0 {
		return nil, errors.New("max attempts must be greater than zero")
	}

	if options.Window <= 0 {
		return nil, errors.New("window must be greater than zero")
	}

	if options.BlockDuration <= 0 {
		return nil, errors.New("block duration must be greater than zero")
	}

	return &RedisLoginLimiter{
		client:        client,
		keyPrefix:     options.KeyPrefix,
		maxAttempts:   options.MaxAttempts,
		window:        options.Window,
		blockDuration: options.BlockDuration,
	}, nil
}

func (l *RedisLoginLimiter) redisKey(account string) string {
	accountHash := sha256.Sum256([]byte(account))

	return l.keyPrefix + ":" + hex.EncodeToString(accountHash[:])
}

// IsBlocked reports whether an account is currently blocked.
func (l *RedisLoginLimiter) IsBlocked(ctx context.Context, account string) (bool, time.Duration, error) {
	values, err := isBlockedScript.Run(
		ctx,
		l.client,
		[]string{l.redisKey(account)},
		l.window.Milliseconds(),
	).Int64Slice()
	if err != nil {
		return false, 0, fmt.Errorf(
			"check login rate limiter: %w",
			err,
		)
	}

	if len(values) != 2 {
		return false, 0, fmt.Errorf(
			"unexpected redis rate-limit response length: %d",
			len(values),
		)
	}
	blockedValue := values[0]
	remainingMilliseconds := values[1]

	return blockedValue == 1,
		time.Duration(remainingMilliseconds) * time.Millisecond,
		nil
}

// RecordFailure records one failed login attempt.
func (l *RedisLoginLimiter) RecordFailure(ctx context.Context, account string) error {
	_, err := recordFailureScript.Run(
		ctx,
		l.client,
		[]string{l.redisKey(account)},
		l.window.Milliseconds(),
		l.maxAttempts,
		l.blockDuration.Milliseconds(),
	).Result()

	if err != nil {
		return fmt.Errorf(
			"record login failure: %w",
			err,
		)
	}
	return nil
}

// Reset clears the login failure state for an account.
func (l *RedisLoginLimiter) Reset(ctx context.Context, account string) error {
	if err := l.client.Del(
		ctx,
		l.redisKey(account),
	).Err(); err != nil {
		return fmt.Errorf(
			"reset login rate limit: %w",
			err,
		)
	}

	return nil
}
