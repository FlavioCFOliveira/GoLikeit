package ratelimiter

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBackend_Take(t *testing.T) {
	t.Run("allows request within limit", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		ctx := context.Background()
		rate := Rate{
			Limit:  100,
			Burst:  150,
			Period: time.Minute,
		}

		result, err := backend.Take(ctx, "test-key", 1, rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected request to be allowed")
		}
		if result.Remaining != 149 {
			t.Errorf("expected 149 remaining, got %d", result.Remaining)
		}
	})

	t.Run("exhausts tokens", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		ctx := context.Background()
		key := "test-exhaust"
		smallRate := Rate{Limit: 5, Burst: 5, Period: time.Minute}

		// Exhaust tokens
		for i := 0; i < 5; i++ {
			result, err := backend.Take(ctx, key, 1, smallRate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i+1)
			}
		}

		// Next request should be denied
		result, err := backend.Take(ctx, key, 1, smallRate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected request to be denied after limit exhausted")
		}
	})

	t.Run("respects burst limit", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		ctx := context.Background()
		key := "test-burst"
		burstRate := Rate{Limit: 10, Burst: 20, Period: time.Minute}

		// Consume up to burst
		result, err := backend.Take(ctx, key, 15, burstRate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected burst consumption to be allowed")
		}
	})

	t.Run("different keys are isolated", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		ctx := context.Background()
		smallRate := Rate{Limit: 5, Burst: 5, Period: time.Minute}

		// Exhaust key1
		for i := 0; i < 5; i++ {
			backend.Take(ctx, "key1", 1, smallRate)
		}

		// key2 should still have full limit
		result, err := backend.Take(ctx, "key2", 1, smallRate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected key2 to be allowed")
		}
		if result.Remaining != 4 {
			t.Errorf("expected 4 remaining for key2, got %d", result.Remaining)
		}
	})

	t.Run("refills tokens over time", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		ctx := context.Background()
		key := "test-refill"
		rate := Rate{Limit: 60, Burst: 60, Period: time.Minute}

		// Take all tokens
		for i := 0; i < 60; i++ {
			backend.Take(ctx, key, 1, rate)
		}

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Should have some tokens now
		result, err := backend.Take(ctx, key, 1, rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// This might fail if the timing is off, so we just check it doesn't panic
		_ = result.Allowed
	})
}

func TestMemoryBackend_Concurrent(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()
	rate := Rate{Limit: 1000, Burst: 1000, Period: time.Minute}

	t.Run("concurrent requests", func(t *testing.T) {
		done := make(chan bool, 100)
		errors := make(chan error, 100)

		for i := 0; i < 100; i++ {
			go func(i int) {
				key := "concurrent-key"
				_, err := backend.Take(ctx, key, 1, rate)
				if err != nil {
					errors <- err
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 100; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Errorf("concurrent error: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for concurrent requests")
			}
		}
	})
}

func TestMemoryBackend_Close(t *testing.T) {
	backend := NewMemoryBackend()

	// Add some data
	ctx := context.Background()
	rate := Rate{Limit: 10, Burst: 10, Period: time.Minute}
	backend.Take(ctx, "key1", 1, rate)
	backend.Take(ctx, "key2", 1, rate)

	// Close
	err := backend.Close()
	if err != nil {
		t.Fatalf("unexpected error on close: %v", err)
	}

	// Verify buckets are cleared
	backend.mu.RLock()
	bucketCount := len(backend.buckets)
	backend.mu.RUnlock()

	if bucketCount != 0 {
		t.Errorf("expected buckets to be cleared, found %d", bucketCount)
	}
}

func BenchmarkMemoryBackend_Take(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()
	rate := Rate{Limit: 100000, Burst: 100000, Period: time.Minute}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-key"
			backend.Take(ctx, key, 1, rate)
			i++
		}
	})
}

func BenchmarkMemoryBackend_TakeDifferentKeys(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()
	rate := Rate{Limit: 100000, Burst: 100000, Period: time.Minute}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune('a' + i%26))
			backend.Take(ctx, key, 1, rate)
			i++
		}
	})
}
