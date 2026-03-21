package ratelimiter

import (
	"math"
	"sync"
	"time"
)

// tokenBucket represents a single token bucket state.
// This struct is designed to be embedded in backend-specific buckets.
type tokenBucket struct {
	mu        sync.RWMutex
	tokens    float64
	lastRefill time.Time
	limit     int
	burst     int
	period    time.Duration
}

// newTokenBucket creates a new token bucket with the given configuration.
func newTokenBucket(limit, burst int, period time.Duration) *tokenBucket {
	now := time.Now()
	return &tokenBucket{
		tokens:     float64(burst),
		lastRefill: now,
		limit:      limit,
		burst:      burst,
		period:     period,
	}
}

// refill calculates and adds tokens based on elapsed time.
// Must be called with lock held.
func (b *tokenBucket) refill(now time.Time) {
	if b.period <= 0 {
		return
	}

	elapsed := now.Sub(b.lastRefill)
	if elapsed <= 0 {
		return
	}

	// Calculate tokens to add: (elapsed / period) * limit
	rate := float64(b.limit) / b.period.Seconds()
	tokensToAdd := elapsed.Seconds() * rate

	b.tokens = math.Min(float64(b.burst), b.tokens+tokensToAdd)
	b.lastRefill = now
}

// take attempts to take tokens from the bucket.
// Returns whether the take was allowed and the remaining tokens.
func (b *tokenBucket) take(now time.Time, tokens int) (allowed bool, remaining int, retryAfter time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill(now)

	if b.tokens >= float64(tokens) {
		b.tokens -= float64(tokens)
		return true, int(b.tokens), 0
	}

	// Calculate retry after time
	needed := float64(tokens) - b.tokens
	rate := float64(b.limit) / b.period.Seconds()
	retryAfter = time.Duration(needed/rate) * time.Second

	return false, int(b.tokens), retryAfter
}

// getState returns the current bucket state for inspection.
func (b *tokenBucket) getState(now time.Time) (tokens float64, resetTime time.Time) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.refill(now)

	// Calculate when bucket will be full
	missing := float64(b.burst) - b.tokens
	rate := float64(b.limit) / b.period.Seconds()
	secondsToFull := missing / rate
	resetTime = now.Add(time.Duration(secondsToFull) * time.Second)

	return b.tokens, resetTime
}

// reset resets the bucket to full capacity.
func (b *tokenBucket) reset(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens = float64(b.burst)
	b.lastRefill = now
}
