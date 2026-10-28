package ratelimit

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidMaxAttempts = errors.New("maxAttempts must be greater than 0")
	ErrInvalidWindow      = errors.New("window must be greater than 0")
)

type attempt struct {
	count       int
	windowStart time.Time
}

// Counts failed attempts for key
// A key is blocked for the remainder of its window once count reaches maxAttempts.
type FixedWindowLimiter struct {
	mu          sync.Mutex
	attempts    map[string]*attempt
	maxAttempts int
	window      time.Duration
	now         func() time.Time
}

func NewFixedWindowLimiter(maxAttempts int, window time.Duration) (*FixedWindowLimiter, error) {
	if maxAttempts <= 0 {
		return nil, ErrInvalidMaxAttempts
	}
	if window <= 0 {
		return nil, ErrInvalidWindow
	}
	return &FixedWindowLimiter{
		attempts:    make(map[string]*attempt),
		maxAttempts: maxAttempts,
		window:      window,
		now:         time.Now,
	}, nil
}
