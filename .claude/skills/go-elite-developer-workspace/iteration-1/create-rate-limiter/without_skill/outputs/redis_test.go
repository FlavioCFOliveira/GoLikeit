package ratelimiter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// isRedisAvailable checks if Redis is available for testing
func isRedisAvailable() bool {
	// Check environment variable
	if os.Getenv("REDIS_ADDR") == "" {
		return false
	}
	return true
}

func skipIfNoRedis(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Skipping Redis tests: REDIS_ADDR not set")
	}
}

func TestRedisBackend_Take(t *testing.T) {
	skipIfNoRedis(t)

	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")

	backend, err := NewRedisBackend(RedisOptions{
		Addr:     addr,
		Password: password,
		DB:       15, // Use separate DB for tests
		Prefix:   "test:",
	})
	if err != nil {
		t.Fatalf("failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()
	rate := Rate{Limit: 10, Burst: 10, Period: time.Minute}

	t.Run("allows request within limit", func(t *testing.T) {
		// Use unique key for this test
		key := "test-allow-" + time.Now().Format("20060102150405")

		result, err := backend.Take(ctx, key, 1, rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected request to be allowed")
		}
		if result.Remaining != 9 {
			t.Errorf("expected 9 remaining, got %d", result.Remaining)
		}
	})

	t.Run("exhausts tokens", func(t *testing.T) {
		key := "test-exhaust-" + time.Now().Format("20060102150405")

		// Exhaust tokens
		for i := 0; i < 10; i++ {
			result, err := backend.Take(ctx, key, 1, rate)
			if err != nil {
				t.Fatalf("request %d: unexpected error: %v", i+1, err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i+1)
			}
		}

		// Next should be denied
		result, err := backend.Take(ctx, key, 1, rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected request to be denied after limit exhausted")
		}
		if result.RetryAfter <= 0 {
			t.Error("expected positive retry after")
		}
	})

	t.Run("different keys are isolated", func(t *testing.T) {
		key1 := "test-isolate-1-" + time.Now().Format("20060102150405")
		key2 := "test-isolate-2-" + time.Now().Format("20060102150405")
		smallRate := Rate{Limit: 5, Burst: 5, Period: time.Minute}

		// Exhaust key1
		for i := 0; i < 5; i++ {
			backend.Take(ctx, key1, 1, smallRate)
		}

		// key2 should still work
		result, err := backend.Take(ctx, key2, 1, smallRate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected key2 to be allowed")
		}
	})
}

func TestRedisBackend_Refill(t *testing.T) {
	skipIfNoRedis(t)

	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")

	backend, err := NewRedisBackend(RedisOptions{
		Addr:     addr,
		Password: password,
		DB:       15,
		Prefix:   "test-refill:",
	})
	if err != nil {
		t.Fatalf("failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()
	key := "refill-test-" + time.Now().Format("20060102150405")
	rate := Rate{Limit: 60, Burst: 60, Period: time.Second}

	// Exhaust tokens
	for i := 0; i < 60; i++ {
		backend.Take(ctx, key, 1, rate)
	}

	// Verify exhausted
	result, _ := backend.Take(ctx, key, 1, rate)
	if result.Allowed {
		t.Fatal("expected bucket to be exhausted")
	}

	// Wait for refill
	time.Sleep(1100 * time.Millisecond)

	// Should have tokens now
	result, err = backend.Take(ctx, key, 1, rate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected request to be allowed after refill")
	}
}

func TestRedisBackend_Integration(t *testing.T) {
	skipIfNoRedis(t)

	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")

	backend, err := NewRedisBackend(RedisOptions{
		Addr:     addr,
		Password: password,
		DB:       15,
		Prefix:   "test-int:",
	})
	if err != nil {
		t.Fatalf("failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	config := NewConfigBuilder().
		WithEndpoint("/api/redis-test", 10, 10, time.Minute).
		Build()

	rl := New(backend, config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("applies rate limiting through Redis", func(t *testing.T) {
		// Make 10 allowed requests
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/redis-test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
			}
		}

		// 11th request should be blocked
		req := httptest.NewRequest("GET", "/api/redis-test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", rec.Code)
		}

		// Check retry-after header
		if rec.Header().Get("Retry-After") == "" {
			t.Error("expected Retry-After header")
		}
	})
}

// TestRedisBackendConcurrent tests concurrent access to Redis backend
func TestRedisBackend_Concurrent(t *testing.T) {
	skipIfNoRedis(t)

	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")

	backend, err := NewRedisBackend(RedisOptions{
		Addr:     addr,
		Password: password,
		DB:       15,
		Prefix:   "test-concurrent:",
	})
	if err != nil {
		t.Fatalf("failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()
	key := "concurrent-test-" + time.Now().Format("20060102150405")
	rate := Rate{Limit: 100, Burst: 100, Period: time.Minute}

	// Use a channel to coordinate goroutines
	done := make(chan int, 50)
	errors := make(chan error, 50)

	for i := 0; i < 50; i++ {
		go func() {
			_, err := backend.Take(ctx, key, 1, rate)
			if err != nil {
				errors <- err
			} else {
				done <- 1
			}
		}()
	}

	successCount := 0
	errorCount := 0
	for i := 0; i < 50; i++ {
		select {
		case <-done:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(10 * time.Second):
			t.Fatal("timeout waiting for concurrent requests")
		}
	}

	if errorCount > 0 {
		t.Errorf("got %d errors during concurrent access", errorCount)
	}

	// All 50 requests should succeed (within burst)
	if successCount != 50 {
		t.Errorf("expected 50 successes, got %d", successCount)
	}
}

// BenchmarkRedisBackend compares Redis backend performance
func BenchmarkRedisBackend(b *testing.B) {
	if !isRedisAvailable() {
		b.Skip("Skipping: REDIS_ADDR not set")
	}

	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")

	backend, err := NewRedisBackend(RedisOptions{
		Addr:     addr,
		Password: password,
		DB:       15,
		Prefix:   "bench:",
	})
	if err != nil {
		b.Fatalf("failed to create Redis backend: %v", err)
	}
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
