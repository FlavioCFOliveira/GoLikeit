// Package resilience provides retry policies and circuit breaker patterns
// for resilient operation execution.
package resilience

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Common errors for resilience operations.
var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

// RetryPolicy configures retry behavior with exponential backoff and jitter.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts (including initial).
	// Default: 3
	MaxAttempts int

	// InitialBackoff is the initial backoff duration.
	// Default: 100ms
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	// Default: 10s
	MaxBackoff time.Duration

	// Jitter is the fraction of backoff to add as random jitter (0.0-1.0).
	// Default: 0.1 (10%)
	Jitter float64

	// ShouldRetry determines if an error should trigger a retry.
	// If nil, all errors are retryable.
	ShouldRetry func(error) bool
}

// DefaultRetryPolicy returns a RetryPolicy with sensible defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Jitter:         0.1,
		ShouldRetry:    nil, // Retry all errors by default
	}
}

// ExponentialBackoffWithJitter calculates the backoff duration for a given attempt.
// Uses exponential backoff with full jitter to prevent thundering herd.
func (p RetryPolicy) ExponentialBackoffWithJitter(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential backoff: initial * 2^(attempt-1)
	backoff := float64(p.InitialBackoff) * math.Pow(2, float64(attempt-1))

	// Apply max backoff cap
	if backoff > float64(p.MaxBackoff) {
		backoff = float64(p.MaxBackoff)
	}

	// Apply jitter (full jitter: random value between 0 and backoff)
	if p.Jitter > 0 {
		jitterRange := backoff * p.Jitter
		// Random value between (backoff - jitterRange) and backoff
		backoff = backoff - jitterRange + (rand.Float64() * jitterRange)
	}

	return time.Duration(backoff)
}

// Retry executes the given function with retry logic.
// Returns nil on success, or ErrMaxRetriesExceeded if all attempts fail.
func (p RetryPolicy) Retry(fn func() error) error {
	return p.RetryWithContext(context.Background(), func(ctx context.Context) error {
		return fn()
	})
}

// RetryWithContext executes the given function with retry logic and context support.
// Returns nil on success, or ErrMaxRetriesExceeded if all attempts fail.
// Honors context cancellation between attempts.
func (p RetryPolicy) RetryWithContext(ctx context.Context, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		// Execute the function
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if p.ShouldRetry != nil && !p.ShouldRetry(err) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt < p.MaxAttempts-1 {
			backoff := p.ExponentialBackoffWithJitter(attempt + 1)

			// Wait for either backoff duration or context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Continue to next attempt
			}
		}
	}

	// All attempts exhausted
	return errors.Join(ErrMaxRetriesExceeded, lastErr)
}

// IsRetryable is a default function that classifies errors as retryable or not.
// Errors that are typically not retryable:
//   - Validation errors
//   - Not found errors
//   - Authentication errors
// Errors that are typically retryable:
//   - Network errors
//   - Timeout errors
//   - Temporary errors
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific non-retryable error types
	// These would need to be defined in the domain or checked via errors.As
	// For now, we assume all errors are retryable unless marked otherwise

	// Check for context errors (not retryable)
	if errors.Is(err, context.Canceled) {
		return false
	}

	// All other errors are potentially retryable
	return true
}

// RetryableError wraps an error to indicate it should be retried.
type RetryableError struct {
	Err error
}

// Error implements the error interface.
func (e *RetryableError) Error() string {
	if e.Err != nil {
		return "retryable: " + e.Err.Error()
	}
	return "retryable error"
}

// Unwrap returns the wrapped error.
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NonRetryableError wraps an error to indicate it should NOT be retried.
type NonRetryableError struct {
	Err error
}

// Error implements the error interface.
func (e *NonRetryableError) Error() string {
	if e.Err != nil {
		return "non-retryable: " + e.Err.Error()
	}
	return "non-retryable error"
}

// Unwrap returns the wrapped error.
func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// MarkRetryable wraps an error as retryable.
func MarkRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// MarkNonRetryable wraps an error as non-retryable.
func MarkNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &NonRetryableError{Err: err}
}

// IsRetryableError checks if an error is marked as retryable.
func IsRetryableError(err error) bool {
	var retryable *RetryableError
	return errors.As(err, &retryable)
}

// IsNonRetryableError checks if an error is marked as non-retryable.
func IsNonRetryableError(err error) bool {
	var nonRetryable *NonRetryableError
	return errors.As(err, &nonRetryable)
}
