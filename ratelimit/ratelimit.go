// Package ratelimit provides rate limiting functionality using the sliding window algorithm.
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Config holds configuration for rate limiting.
type Config struct {
	// Enabled controls whether rate limiting is enabled.
	Enabled bool

	// RequestsPerSecond is the maximum number of requests allowed per second.
	RequestsPerSecond int

	// BurstSize is the maximum burst size allowed.
	BurstSize int

	// WindowSize is the time window for rate limiting.
	WindowSize time.Duration
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         20,
		WindowSize:        time.Second,
	}
}

// Limiter provides rate limiting using the sliding window algorithm.
type Limiter struct {
	config Config
	mu     sync.RWMutex
	// windows maps userID to their request timestamps
	windows map[string][]time.Time
	// lastCleanup tracks when we last cleaned up old entries
	lastCleanup time.Time
}

// New creates a new rate limiter with the given configuration.
func New(config Config) *Limiter {
	if !config.Enabled {
		return &Limiter{config: config}
	}

	return &Limiter{
		config:      config,
		windows:     make(map[string][]time.Time),
		lastCleanup: time.Now(),
	}
}

// Allow checks if a request from the given user is allowed.
// Returns true if the request should be allowed, false otherwise.
func (l *Limiter) Allow(userID string) bool {
	if !l.config.Enabled {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Periodic cleanup of old entries
	if time.Since(l.lastCleanup) > l.config.WindowSize*2 {
		l.cleanup()
	}

	now := time.Now()
	window := l.windows[userID]

	// Remove timestamps outside the current window
	cutoff := now.Add(-l.config.WindowSize)
	validIdx := 0
	for i, ts := range window {
		if ts.After(cutoff) {
			validIdx = i
			break
		}
	}
	window = window[validIdx:]

	// Check if request is allowed (burst limit)
	if len(window) >= l.config.BurstSize {
		return false
	}

	// Record this request
	window = append(window, now)
	l.windows[userID] = window

	return true
}

// AllowContext checks if a request is allowed with context support.
func (l *Limiter) AllowContext(ctx context.Context, userID string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return l.Allow(userID)
	}
}

// GetRemaining returns the remaining requests allowed for a user in the current window.
func (l *Limiter) GetRemaining(userID string) int {
	if !l.config.Enabled {
		return l.config.BurstSize
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	window := l.windows[userID]
	cutoff := time.Now().Add(-l.config.WindowSize)

	// Count valid requests
	validCount := 0
	for _, ts := range window {
		if ts.After(cutoff) {
			validCount++
		}
	}

	remaining := l.config.BurstSize - validCount
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// GetResetTime returns the time when the rate limit will reset for a user.
func (l *Limiter) GetResetTime(userID string) time.Time {
	if !l.config.Enabled {
		return time.Now()
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	window := l.windows[userID]
	if len(window) == 0 {
		return time.Now()
	}

	// Return the time when the oldest request in the window expires
	oldest := window[0]
	for _, ts := range window {
		if ts.Before(oldest) {
			oldest = ts
		}
	}

	return oldest.Add(l.config.WindowSize)
}

// Reset clears the rate limit window for a specific user.
func (l *Limiter) Reset(userID string) {
	if !l.config.Enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.windows, userID)
}

// ResetAll clears all rate limit windows.
func (l *Limiter) ResetAll() {
	if !l.config.Enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.windows = make(map[string][]time.Time)
}

// cleanup removes old entries to prevent memory leaks.
func (l *Limiter) cleanup() {
	cutoff := time.Now().Add(-l.config.WindowSize * 2)

	for userID, window := range l.windows {
		validIdx := 0
		for i, ts := range window {
			if ts.After(cutoff) {
				validIdx = i
				break
			}
		}

		if validIdx >= len(window) {
			delete(l.windows, userID)
		} else {
			l.windows[userID] = window[validIdx:]
		}
	}

	l.lastCleanup = time.Now()
}

// IsEnabled returns true if rate limiting is enabled.
func (l *Limiter) IsEnabled() bool {
	return l.config.Enabled
}

// GetConfig returns the current configuration.
func (l *Limiter) GetConfig() Config {
	return l.config
}

// Stats holds rate limiting statistics.
type Stats struct {
	// UserID is the user identifier.
	UserID string
	// RequestsInWindow is the number of requests in the current window.
	RequestsInWindow int
	// Remaining is the number of remaining allowed requests.
	Remaining int
	// ResetTime is when the rate limit will reset.
	ResetTime time.Time
	// Limited is true if the user is currently rate limited.
	Limited bool
}

// GetStats returns rate limiting statistics for a user.
func (l *Limiter) GetStats(userID string) Stats {
	if !l.config.Enabled {
		return Stats{
			UserID:    userID,
			Remaining: l.config.BurstSize,
			ResetTime: time.Now(),
			Limited:   false,
		}
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	window := l.windows[userID]
	cutoff := time.Now().Add(-l.config.WindowSize)

	// Count valid requests
	validCount := 0
	for _, ts := range window {
		if ts.After(cutoff) {
			validCount++
		}
	}

	remaining := l.config.BurstSize - validCount
	if remaining < 0 {
		remaining = 0
	}

	return Stats{
		UserID:             userID,
		RequestsInWindow:   validCount,
		Remaining:          remaining,
		ResetTime:          l.GetResetTime(userID),
		Limited:            validCount >= l.config.BurstSize,
	}
}

// Error represents a rate limiting error.
type Error struct {
	UserID    string
	RetryAfter time.Duration
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("rate limit exceeded for user %s, retry after %v", e.UserID, e.RetryAfter)
}
