package ratelimit

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidMaxAttempts   = errors.New("maxAttempts must be greater than 0")
	ErrInvalidWindow        = errors.New("window must be greater than 0")
	ErrInvalidBlockDuration = errors.New("blockDuration must be greater than 0")
)

type attempt struct {
	count        int
	windowStart  time.Time
	blockedUntil time.Time
}

type FixedWindowLimiter struct {
	mu            sync.Mutex
	attempts      map[string]*attempt
	maxAttempts   int
	window        time.Duration
	blockDuration time.Duration
	now           func() time.Time
}

func NewFixedWindowLimiter(
	maxAttempts int,
	window time.Duration,
	blockDuration time.Duration,
) (*FixedWindowLimiter, error) {
	if maxAttempts <= 0 {
		return nil, ErrInvalidMaxAttempts
	}

	if window <= 0 {
		return nil, ErrInvalidWindow
	}

	if blockDuration <= 0 {
		return nil, ErrInvalidBlockDuration
	}

	return &FixedWindowLimiter{
		attempts:      make(map[string]*attempt),
		maxAttempts:   maxAttempts,
		window:        window,
		blockDuration: blockDuration,
		now:           time.Now,
	}, nil
}

func (l *FixedWindowLimiter) IsBlocked(
	key string,
) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, exists := l.attempts[key]
	if !exists {
		return false, 0
	}

	now := l.now()

	// The block takes priority over the counting window.
	if !current.blockedUntil.IsZero() {
		if now.Before(current.blockedUntil) {
			return true, current.blockedUntil.Sub(now)
		}

		// The block has expired, start completely fresh.
		delete(l.attempts, key)
		return false, 0
	}

	// The failure-counting window has expired.
	if now.Sub(current.windowStart) >= l.window {
		delete(l.attempts, key)
		return false, 0
	}
	return false, 0
}

func (l *FixedWindowLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	current, exists := l.attempts[key]

	if exists && !current.blockedUntil.IsZero() {
		// Never extend an existing block.
		if now.Before(current.blockedUntil) {
			return
		}

		// The block expired before this call.
		delete(l.attempts, key)
		exists = false
	}

	if !exists || now.Sub(current.windowStart) >= l.window {
		current = &attempt{
			count:       1,
			windowStart: now,
		}

		l.attempts[key] = current
	} else {
		current.count++
	}

	if current.count >= l.maxAttempts {
		current.blockedUntil = now.Add(l.blockDuration)
	}
}

func (l *FixedWindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, key)
}
