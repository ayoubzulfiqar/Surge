package utils

import (
	"context"
	"sync"
	"time"
)

// TokenBucket is a thread-safe rate limiter.
type TokenBucket struct {
	mu         sync.Mutex
	rateBytes  float64 // tokens per second
	tokens     float64 // current tokens
	lastUpdate time.Time
	enabled    bool
}

// GlobalRateLimiter is the shared singleton for global application bandwidth limit.
var GlobalRateLimiter = NewTokenBucket(0)

// NewTokenBucket creates a new rate limiter allowing 'rate' bytes per second.
// If rate is 0, the limiter is disabled and WaitN returns immediately.
func NewTokenBucket(rate int64) *TokenBucket {
	// A max burst size is usually 1 second's worth of tokens or more, 
	// but since we update exactly, we can just use rate as our capacity.
	enabled := rate > 0
	return &TokenBucket{
		rateBytes:  float64(rate),
		tokens:     float64(rate),
		lastUpdate: time.Now(),
		enabled:    enabled,
	}
}

// SetRate dynamically updates the rate limit. 0 disables it.
func (tb *TokenBucket) SetRate(rate int64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.rateBytes = float64(rate)
	tb.enabled = rate > 0
	// Don't modify current tokens or restrict them right away, wait to drain.
}

// WaitN blocks until 'n' bytes are available according to the rate limit.
func (tb *TokenBucket) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}

	for {
		// Use a minimal block scope for locking so we don't hold the mutex during sleep
		sleepDur := time.Duration(0)
		
		tb.mu.Lock()
		
		if !tb.enabled {
			tb.mu.Unlock()
			return nil
		}

		now := time.Now()
		elapsed := now.Sub(tb.lastUpdate).Seconds()

		// Refill tokens based on elapsed time
		tb.tokens += elapsed * tb.rateBytes
		
		// Cap tokens to 1 second worth of bandwidth (max burst)
		if tb.tokens > tb.rateBytes {
			tb.tokens = tb.rateBytes
		}
		
		tb.lastUpdate = now

		reqTokens := float64(n)
		if tb.tokens >= reqTokens {
			// We have enough tokens, consume them and proceed
			tb.tokens -= reqTokens
			tb.mu.Unlock()
			return nil
		}
		
		// Not enough tokens. Calculate time to wait.
		deficit := reqTokens - tb.tokens
		sleepSecs := deficit / tb.rateBytes
		sleepDur = time.Duration(sleepSecs * float64(time.Second))

		tb.mu.Unlock()

		// Wait, while remaining respectful to the context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDur):
			// Time passed, loop again to claim tokens
		}
	}
}
