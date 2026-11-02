package middleware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"user-management/internal/config"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidMaxAttempts   = errors.New("maxAttempts must be greater than 0")
	ErrInvalidWindow        = errors.New("window must be greater than 0")
	ErrInvalidBlockDuration = errors.New("blockDuration must be greater than 0")
)

var isBlockedScript = redis.NewScript(`
local values = redis.call(
	"HMGET",
	KEYS[1],
	"window_start_ms",
	"blocked_until_ms"
)

if values[1] == false then
	return {0, 0}
end

local timeParts = redis.call("TIME")
local nowMs = 
	tonumber(timeParts[1]) * 1000 +
	math.floor(tonumber(timeParts[2]) / 1000)

local windowStartMs = tonumber(values[1]) or 0
local blockedUntilMs = tonumber(values[2]) or 0

if blockedUntilMs > 0 then
	if nowMs < blockedUntilMs then
		return {1, blockedUntilMs - nowMs}
	end

	redis.call("DEL", KEYS[1])
	return {0, 0}
end

if nowMs - windowStartMs >= tonumber(ARGV[1]) then
	redis.call("DEL", KEYS[1])
	return {0, 0}
end

return {0, 0}
`)

var recordFailureScript = redis.NewScript(`
local values = redis.call(
	"HMGET",
	KEYS[1],
	"count",
	"window_start_ms",
	"blocked_until_ms"
)

local timeParts = redis.call("TIME")
local nowMs =
	tonumber(timeParts[1]) * 1000 +
	math.floor(tonumber(timeParts[2]) / 1000)

local count = tonumber(values[1]) or 0
local windowStartMs = tonumber(values[2]) or 0
local blockedUntilMs = tonumber(values[3]) or 0

local windowMs = tonumber(ARGV[1])
local maxAttempts = tonumber(ARGV[2])
local blockDurationMs = tonumber(ARGV[3])

if blockedUntilMs > 0 and nowMs < blockedUntilMs then
	return 0
end

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

count = count + 1

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

type RedisLoginLimiter struct {
	client        *redis.Client
	keyPrefix     string
	maxAttempts   int
	window        time.Duration
	blockDuration time.Duration
}

var _ LoginLimiter = (*RedisLoginLimiter)(nil)

func NewRedisLoginLimiter(
	client *redis.Client,
	config config.ParsedRateLimiterConfig,
) (*RedisLoginLimiter, error) {
	if client == nil {
		return nil, errors.New("redis client must not be nil")
	}

	return &RedisLoginLimiter{
		client:        client,
		keyPrefix:     config.KeyPrefix,
		maxAttempts:   config.MaxAttempts,
		window:        config.Window,
		blockDuration: config.BlockDuration,
	}, nil
}

func (l *RedisLoginLimiter) redisKey(account string) string {
	return l.keyPrefix + ":" + account
}

func (l *RedisLoginLimiter) IsBlocked(
	ctx context.Context,
	account string,
) (bool, time.Duration, error) {
	result, err := isBlockedScript.Run(
		ctx,
		l.client,
		[]string{l.redisKey(account)},
		l.window.Milliseconds(),
	).Result()
	if err != nil {
		return false, 0, fmt.Errorf(
			"check login rate limiter for %q: %w",
			account,
			err,
		)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, fmt.Errorf(
			"unexpected Redis rate-limit response: %T",
			result,
		)
	}

	blockedValue, ok := values[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf(
			"unexpected Redis blocked value: %T",
			values[0],
		)
	}

	remainingMilliseconds, ok := values[1].(int64)
	if !ok {
		return false, 0, fmt.Errorf(
			"unexpected Redis remaining value: %T",
			values[1],
		)
	}

	return blockedValue == 1,
		time.Duration(remainingMilliseconds) * time.Millisecond,
		nil
}

func (l *RedisLoginLimiter) RecordFailure(
	ctx context.Context,
	account string,
) error {
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
			"record login failure for %q: %w",
			account,
			err,
		)
	}
	return nil
}

func (l *RedisLoginLimiter) Reset(
	ctx context.Context,
	account string,
) error {
	if err := l.client.Del(
		ctx,
		l.redisKey(account),
	).Err(); err != nil {
		return fmt.Errorf(
			"reset login rate limit for %q: %w",
			account,
			err,
		)
	}

	return nil
}
