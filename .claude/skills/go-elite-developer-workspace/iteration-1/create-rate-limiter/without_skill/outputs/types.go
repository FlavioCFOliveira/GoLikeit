// Package ratelimiter provides a high-performance token bucket rate limiter
// with support for multiple backends (in-memory and Redis).
package ratelimiter

import (
	"context"
	"net/http"
	"time"
)

// Backend defines the interface for rate limiter storage backends.
// Implementations must be thread-safe.
type Backend interface {
	// Take attempts to take tokens from the bucket for the given key.
	// Returns the number of tokens remaining, time until next token is available,
	// and any error encountered.
	Take(ctx context.Context, key string, tokens int, rate Rate) (TakeResult, error)

	// Close releases any resources held by the backend.
	Close() error
}

// Rate defines the token bucket rate configuration.
type Rate struct {
	// Limit is the maximum number of tokens in the bucket.
	Limit int

	// Burst is the maximum number of tokens that can be consumed in a single request.
	Burst int

	// Period is the time period for refilling tokens.
	Period time.Duration
}

// TokensPerSecond calculates tokens per second for this rate.
func (r Rate) TokensPerSecond() float64 {
	return float64(r.Limit) / r.Period.Seconds()
}

// IsZero returns true if the rate is not configured.
func (r Rate) IsZero() bool {
	return r.Limit == 0 && r.Burst == 0
}

// TakeResult contains the result of a Take operation.
type TakeResult struct {
	// Allowed indicates whether the tokens were successfully consumed.
	Allowed bool

	// Remaining is the number of tokens remaining in the bucket after the take.
	Remaining int

	// ResetTime is the time when the bucket will be full again.
	ResetTime time.Time

	// RetryAfter is the duration to wait before the next token is available.
	// Only set when Allowed is false.
	RetryAfter time.Duration
}

// KeyFunc generates a rate limit key from an HTTP request.
// The returned key is used to identify the bucket for the request.
type KeyFunc func(r *http.Request) string

// DefaultKeyFunc generates keys based on remote IP address.
func DefaultKeyFunc() KeyFunc {
	return func(r *http.Request) string {
		return r.RemoteAddr
	}
}

// UserKeyFunc generates keys based on a user identifier from headers.
func UserKeyFunc(headerName string) KeyFunc {
	return func(r *http.Request) string {
		if userID := r.Header.Get(headerName); userID != "" {
			return userID
		}
		return r.RemoteAddr
	}
}

// CompositeKeyFunc generates a composite key using multiple parts.
func CompositeKeyFunc(parts ...KeyFunc) KeyFunc {
	return func(r *http.Request) string {
		key := ""
		for _, part := range parts {
			if key != "" {
				key += ":"
			}
			key += part(r)
		}
		return key
	}
}
