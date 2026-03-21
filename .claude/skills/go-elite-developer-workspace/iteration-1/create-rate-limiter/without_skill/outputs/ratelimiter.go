// Package ratelimiter provides a high-performance token bucket rate limiter.
//
// Usage example:
//
//	// Create a rate limiter with in-memory backend
//	backend := ratelimiter.NewMemoryBackend()
//	defer backend.Close()
//
//	config := ratelimiter.NewConfigBuilder().
//		WithDefaultRate(100, 150, time.Minute).
//		WithEndpoint("/api/login", 5, 10, time.Minute).
//		Build()
//
//	lim := ratelimiter.New(backend, config)
//
//	// Apply middleware to your router
//	http.Handle("/api/", lim.Middleware(yourHandler))
//
package ratelimiter

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RateLimiter is the main rate limiter implementation.
type RateLimiter struct {
	backend Backend
	config  Config
}

// New creates a new rate limiter with the given backend and configuration.
func New(backend Backend, config Config) *RateLimiter {
	return &RateLimiter{
		backend: backend,
		config:  config,
	}
}

// Allow checks if a single request is allowed for the given key and rate.
func (rl *RateLimiter) Allow(ctx context.Context, key string, rate Rate) (TakeResult, error) {
	if rate.IsZero() {
		return TakeResult{Allowed: true, Remaining: 1}, nil
	}
	return rl.backend.Take(ctx, key, 1, rate)
}

// AllowN checks if n tokens can be consumed for the given key.
func (rl *RateLimiter) AllowN(ctx context.Context, key string, n int, rate Rate) (TakeResult, error) {
	if rate.IsZero() {
		return TakeResult{Allowed: true, Remaining: n}, nil
	}
	return rl.backend.Take(ctx, key, n, rate)
}

// CheckHTTPRequest checks if an HTTP request should be allowed.
// Returns the TakeResult and any error encountered.
func (rl *RateLimiter) CheckHTTPRequest(r *http.Request) (TakeResult, error) {
	if !rl.config.Enabled {
		return TakeResult{Allowed: true}, nil
	}

	// Get configuration for this path
	config := rl.config.GetConfigForPath(r.URL.Path)

	// If rate is zero, allow the request
	if config.Rate.IsZero() {
		return TakeResult{Allowed: true}, nil
	}

	// Generate the rate limit key
	key := config.KeyFunc(r)

	// Take tokens
	return rl.AllowN(r.Context(), key, config.TokensConsumed, config.Rate)
}

// GetRetryAfterHeader returns the Retry-After header value in seconds.
func (rl *RateLimiter) GetRetryAfterHeader(result TakeResult) string {
	if result.RetryAfter <= 0 {
		// Calculate based on reset time
		retryAfter := time.Until(result.ResetTime)
		if retryAfter < 0 {
			return "1"
		}
		return fmt.Sprintf("%d", int(retryAfter.Seconds())+1)
	}
	return fmt.Sprintf("%d", int(result.RetryAfter.Seconds()))
}

// Close releases resources held by the rate limiter.
func (rl *RateLimiter) Close() error {
	return rl.backend.Close()
}

// GetRemainingLimit returns the remaining limit for a key.
// This is useful for debugging/monitoring purposes.
func (rl *RateLimiter) GetRemainingLimit(ctx context.Context, key string, rate Rate) (int, time.Time, error) {
	result, err := rl.backend.Take(ctx, key, 0, rate)
	if err != nil {
		return 0, time.Time{}, err
	}
	return result.Remaining, result.ResetTime, nil
}

// Stats holds rate limiter statistics for monitoring.
type Stats struct {
	// TotalRequests is the total number of requests processed.
	TotalRequests int64

	// AllowedRequests is the number of requests that were allowed.
	AllowedRequests int64

	// BlockedRequests is the number of requests that were blocked.
	BlockedRequests int64

	// AverageLatency is the average processing latency.
	AverageLatency time.Duration
}
