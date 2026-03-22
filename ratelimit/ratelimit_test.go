package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := DefaultConfig()
	limiter := New(config)

	if limiter == nil {
		t.Fatal("expected limiter to not be nil")
	}
	if !limiter.IsEnabled() {
		t.Error("expected limiter to be enabled")
	}
}

func TestNew_Disabled(t *testing.T) {
	config := Config{Enabled: false}
	limiter := New(config)

	if limiter == nil {
		t.Fatal("expected limiter to not be nil")
	}
	if limiter.IsEnabled() {
		t.Error("expected limiter to be disabled")
	}
}

func TestLimiter_Allow(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 2,
		BurstSize:         3,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	userID := "user1"

	// First 3 requests should be allowed (burst size)
	if !limiter.Allow(userID) {
		t.Error("expected first request to be allowed")
	}
	if !limiter.Allow(userID) {
		t.Error("expected second request to be allowed")
	}
	if !limiter.Allow(userID) {
		t.Error("expected third request to be allowed")
	}

	// Fourth request should be denied
	if limiter.Allow(userID) {
		t.Error("expected fourth request to be denied")
	}
}

func TestLimiter_Allow_DifferentUsers(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstSize:         1,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	// Each user should have their own limit
	if !limiter.Allow("user1") {
		t.Error("expected user1 first request to be allowed")
	}
	if !limiter.Allow("user2") {
		t.Error("expected user2 first request to be allowed")
	}

	// Second request from user1 should be denied
	if limiter.Allow("user1") {
		t.Error("expected user1 second request to be denied")
	}
}

func TestLimiter_Allow_Disabled(t *testing.T) {
	config := Config{Enabled: false}
	limiter := New(config)

	// All requests should be allowed when disabled
	for i := 0; i < 100; i++ {
		if !limiter.Allow("user1") {
			t.Errorf("expected request %d to be allowed when disabled", i)
		}
	}
}

func TestLimiter_AllowContext(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstSize:         1,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	ctx := context.Background()

	if !limiter.AllowContext(ctx, "user1") {
		t.Error("expected request to be allowed")
	}

	// Cancelled context should deny
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if limiter.AllowContext(cancelledCtx, "user1") {
		t.Error("expected request to be denied with cancelled context")
	}
}

func TestLimiter_GetRemaining(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         5,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	userID := "user1"

	// Initially should have full burst size
	if remaining := limiter.GetRemaining(userID); remaining != 5 {
		t.Errorf("expected 5 remaining, got %d", remaining)
	}

	// Make 3 requests
	for i := 0; i < 3; i++ {
		limiter.Allow(userID)
	}

	// Should have 2 remaining
	if remaining := limiter.GetRemaining(userID); remaining != 2 {
		t.Errorf("expected 2 remaining, got %d", remaining)
	}
}

func TestLimiter_GetResetTime(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstSize:         1,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	userID := "user1"

	// Before any requests, reset time should be now
	resetTime := limiter.GetResetTime(userID)
	if resetTime.After(time.Now().Add(time.Second)) {
		t.Error("expected reset time to be close to now")
	}

	// Make a request
	limiter.Allow(userID)

	// Reset time should be window size from now
	resetTime = limiter.GetResetTime(userID)
	expectedReset := time.Now().Add(time.Second)
	if resetTime.Before(expectedReset.Add(-time.Millisecond*100)) || resetTime.After(expectedReset.Add(time.Millisecond*100)) {
		t.Errorf("expected reset time around %v, got %v", expectedReset, resetTime)
	}
}

func TestLimiter_Reset(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstSize:         1,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	userID := "user1"

	// Make a request
	limiter.Allow(userID)

	// Should be rate limited now
	if limiter.Allow(userID) {
		t.Error("expected second request to be denied")
	}

	// Reset the user
	limiter.Reset(userID)

	// Should be allowed again
	if !limiter.Allow(userID) {
		t.Error("expected request to be allowed after reset")
	}
}

func TestLimiter_ResetAll(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstSize:         1,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	// Make requests from multiple users
	limiter.Allow("user1")
	limiter.Allow("user2")

	// Reset all
	limiter.ResetAll()

	// All should be allowed again
	if !limiter.Allow("user1") {
		t.Error("expected user1 request to be allowed after reset all")
	}
	if !limiter.Allow("user2") {
		t.Error("expected user2 request to be allowed after reset all")
	}
}

func TestLimiter_GetStats(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 5,
		BurstSize:         10,
		WindowSize:        time.Second,
	}
	limiter := New(config)

	userID := "user1"

	// Initially
	stats := limiter.GetStats(userID)
	if stats.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, stats.UserID)
	}
	if stats.RequestsInWindow != 0 {
		t.Errorf("expected 0 requests in window, got %d", stats.RequestsInWindow)
	}
	if stats.Remaining != 10 {
		t.Errorf("expected 10 remaining, got %d", stats.Remaining)
	}
	if stats.Limited {
		t.Error("expected not to be limited")
	}

	// Make 5 requests
	for i := 0; i < 5; i++ {
		limiter.Allow(userID)
	}

	stats = limiter.GetStats(userID)
	if stats.RequestsInWindow != 5 {
		t.Errorf("expected 5 requests in window, got %d", stats.RequestsInWindow)
	}
	if stats.Remaining != 5 {
		t.Errorf("expected 5 remaining, got %d", stats.Remaining)
	}

	// Make 5 more requests (reaching burst limit)
	for i := 0; i < 5; i++ {
		limiter.Allow(userID)
	}

	stats = limiter.GetStats(userID)
	if stats.RequestsInWindow != 10 {
		t.Errorf("expected 10 requests in window, got %d", stats.RequestsInWindow)
	}
	if stats.Remaining != 0 {
		t.Errorf("expected 0 remaining, got %d", stats.Remaining)
	}
	if !stats.Limited {
		t.Error("expected to be limited")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("expected rate limiting to be enabled by default")
	}
	if config.RequestsPerSecond != 10 {
		t.Errorf("expected 10 requests per second, got %d", config.RequestsPerSecond)
	}
	if config.BurstSize != 20 {
		t.Errorf("expected burst size of 20, got %d", config.BurstSize)
	}
	if config.WindowSize != time.Second {
		t.Errorf("expected window size of 1s, got %v", config.WindowSize)
	}
}

func TestError(t *testing.T) {
	err := &Error{
		UserID:     "user1",
		RetryAfter: time.Second,
	}

	expected := "rate limit exceeded for user user1, retry after 1s"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func BenchmarkLimiter_Allow(b *testing.B) {
	config := Config{
		Enabled:           true,
		RequestsPerSecond: 1000,
		BurstSize:         2000,
		WindowSize:        time.Second,
	}
	limiter := New(config)
	userID := "user1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(userID)
	}
}
