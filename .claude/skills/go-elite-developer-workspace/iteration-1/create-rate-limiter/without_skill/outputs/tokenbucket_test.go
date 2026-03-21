package ratelimiter

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_newTokenBucket(t *testing.T) {
	bucket := newTokenBucket(10, 15, time.Minute)

	if bucket.limit != 10 {
		t.Errorf("expected limit 10, got %d", bucket.limit)
	}
	if bucket.burst != 15 {
		t.Errorf("expected burst 15, got %d", bucket.burst)
	}
	if bucket.period != time.Minute {
		t.Errorf("expected period 1m, got %v", bucket.period)
	}
	if bucket.tokens != 15 {
		t.Errorf("expected initial tokens 15 (burst), got %f", bucket.tokens)
	}
}

func TestTokenBucket_refill(t *testing.T) {
	bucket := newTokenBucket(60, 60, time.Minute)

	// Take some tokens
	now := time.Now()
	bucket.take(now, 30)

	// Verify tokens were taken
	if bucket.tokens != 30 {
		t.Errorf("expected 30 tokens remaining, got %f", bucket.tokens)
	}

	// Refill after some time has passed
	t.Run("refills over time", func(t *testing.T) {
		bucket := newTokenBucket(60, 60, time.Minute)
		bucket.tokens = 30 // Simulate having consumed 30 tokens
		bucket.lastRefill = time.Now().Add(-30 * time.Second)

		now := time.Now()
		bucket.refill(now)

		// Should have refilled approximately 30 tokens (60 tokens/min * 0.5 min)
		if bucket.tokens < 55 || bucket.tokens > 65 {
			t.Errorf("expected approximately 60 tokens after refill, got %f", bucket.tokens)
		}
	})

	t.Run("caps at burst", func(t *testing.T) {
		bucket := newTokenBucket(60, 60, time.Minute)
		bucket.tokens = 30
		bucket.lastRefill = time.Now().Add(-2 * time.Minute) // Wait longer than period

		now := time.Now()
		bucket.refill(now)

		if bucket.tokens > 60 {
			t.Errorf("tokens should be capped at burst (60), got %f", bucket.tokens)
		}
	})
}

func TestTokenBucket_take(t *testing.T) {
	t.Run("allows when sufficient tokens", func(t *testing.T) {
		bucket := newTokenBucket(10, 10, time.Minute)
		now := time.Now()

		allowed, remaining, retryAfter := bucket.take(now, 5)

		if !allowed {
			t.Error("expected take to be allowed")
		}
		if remaining != 5 {
			t.Errorf("expected 5 remaining, got %d", remaining)
		}
		if retryAfter != 0 {
			t.Errorf("expected 0 retry after, got %v", retryAfter)
		}
	})

	t.Run("denies when insufficient tokens", func(t *testing.T) {
		bucket := newTokenBucket(10, 10, time.Minute)
		now := time.Now()

		// Take all tokens
		bucket.take(now, 10)

		// Try to take more
		allowed, remaining, retryAfter := bucket.take(now, 1)

		if allowed {
			t.Error("expected take to be denied")
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", remaining)
		}
		if retryAfter <= 0 {
			t.Errorf("expected positive retry after, got %v", retryAfter)
		}
	})

	t.Run("respects burst limit", func(t *testing.T) {
		bucket := newTokenBucket(10, 15, time.Minute)
		now := time.Now()

		// Take 5 tokens
		bucket.take(now, 5)

		// Wait for refill
		now = now.Add(1 * time.Minute)

		// Take burst amount
		allowed, _, _ := bucket.take(now, 15)

		if !allowed {
			t.Error("expected take within burst to be allowed")
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		bucket := newTokenBucket(1000, 1000, time.Minute)

		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// Spawn 100 goroutines trying to take tokens
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				now := time.Now()
				allowed, _, _ := bucket.take(now, 1)
				if allowed {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		if successCount > 100 {
			t.Errorf("expected max 100 successful takes, got %d", successCount)
		}
	})
}

func TestTokenBucket_getState(t *testing.T) {
	bucket := newTokenBucket(60, 60, time.Minute)
	now := time.Now()

	// Take some tokens
	bucket.take(now, 20)

	tokens, resetTime := bucket.getState(now)

	if tokens != 40 {
		t.Errorf("expected 40 tokens, got %f", tokens)
	}
	if resetTime.Before(now) {
		t.Error("reset time should be in the future")
	}
}

func TestTokenBucket_reset(t *testing.T) {
	bucket := newTokenBucket(60, 60, time.Minute)
	now := time.Now()

	// Take all tokens
	bucket.take(now, 60)

	// Reset
	bucket.reset(now)

	if bucket.tokens != 60 {
		t.Errorf("expected tokens to be reset to 60, got %f", bucket.tokens)
	}
	if bucket.lastRefill != now {
		t.Error("expected last refill time to be updated")
	}
}

func BenchmarkTokenBucket_take(b *testing.B) {
	bucket := newTokenBucket(10000, 10000, time.Minute)
	now := time.Now()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bucket.take(now, 1)
		}
	})
}
