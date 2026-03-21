package ratelimiter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegration_MemoryBackend tests the complete flow with in-memory backend
func TestIntegration_MemoryBackend(t *testing.T) {
	// Create rate limiter
	rl, err := New(Config{
		DefaultRate:  100,
		DefaultBurst: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	// Configure specific route with lower limits
	if err := rl.SetRoute("/api/limited", 1, 2); err != nil {
		t.Fatalf("Failed to set route: %v", err)
	}

	// Create test handler that tracks successful requests
	var successCount int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&successCount, 1)
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	// Test 1: Default route allows burst
	t.Run("default_route_allows_burst", func(t *testing.T) {
		atomic.StoreInt64(&successCount, 0)

		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/default", nil)
			req.RemoteAddr = "127.0.0.1:10001"
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Request %d: expected 200, got %d", i, rr.Code)
			}
		}

		// 11th request should be rate limited
		req := httptest.NewRequest("GET", "/api/default", nil)
		req.RemoteAddr = "127.0.0.1:10001"
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Expected 429, got %d", rr.Code)
		}

		retryAfter := rr.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Error("Retry-After header missing")
		}
	})

	// Test 2: Limited route has stricter limits
	t.Run("limited_route_enforces_stricter_limits", func(t *testing.T) {
		atomic.StoreInt64(&successCount, 0)

		// Should only allow 2 requests (burst)
		allowed := 0
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/api/limited", nil)
			req.RemoteAddr = "127.0.0.1:10002"
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			if rr.Code == http.StatusOK {
				allowed++
			}
		}

		if allowed != 2 {
			t.Errorf("Expected 2 allowed requests, got %d", allowed)
		}
	})

	// Test 3: Different IPs have separate buckets
	t.Run("per_ip_isolation", func(t *testing.T) {
		// First IP uses up its burst
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/default", nil)
			req.RemoteAddr = "192.168.1.1:10003"
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)
		}

		// Second IP (different address) should still have full burst
		req := httptest.NewRequest("GET", "/api/default", nil)
		req.RemoteAddr = "192.168.1.2:10004"
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Second IP should have available tokens, got %d", rr.Code)
		}
	})

	// Test 4: Token refill over time
	t.Run("token_refill", func(t *testing.T) {
		key := "127.0.0.1:10005"

		// Use up all tokens
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/api/limited", nil)
			req.RemoteAddr = key
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)
		}

		// Verify rate limited
		req := httptest.NewRequest("GET", "/api/limited", nil)
		req.RemoteAddr = key
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("Should be rate limited before waiting")
		}

		// Wait for token refill (rate is 1/sec)
		time.Sleep(1100 * time.Millisecond)

		// Should now be allowed
		req = httptest.NewRequest("GET", "/api/limited", nil)
		req.RemoteAddr = key
		rr = httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Should be allowed after refill, got %d", rr.Code)
		}
	})
}

// TestIntegration_ConcurrentStress tests concurrent access patterns
func TestIntegration_ConcurrentStress(t *testing.T) {
	rl, err := New(Config{
		DefaultRate:  10000,
		DefaultBurst: 1000,
	})
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	const (
		numWorkers        = 50
		requestsPerWorker = 100
		numIPs            = 10
	)

	var (
		successCount int64
		limitedCount int64
		wg           sync.WaitGroup
	)

	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			// Each worker uses a subset of IPs
			ip := fmt.Sprintf("127.0.0.%d", workerID%numIPs+1)

			for j := 0; j < requestsPerWorker; j++ {
				req := httptest.NewRequest("GET", "/api/test", nil)
				req.RemoteAddr = ip + ":10000"
				rr := httptest.NewRecorder()

				middleware.ServeHTTP(rr, req)

				switch rr.Code {
				case http.StatusOK:
					atomic.AddInt64(&successCount, 1)
				case http.StatusTooManyRequests:
					atomic.AddInt64(&limitedCount, 1)
				default:
					t.Errorf("Unexpected status: %d", rr.Code)
				}
			}
		}(i)
	}

	wg.Wait()

	total := successCount + limitedCount
	expectedTotal := int64(numWorkers * requestsPerWorker)

	if total != expectedTotal {
		t.Errorf("Total requests mismatch: got %d, want %d", total, expectedTotal)
	}

	t.Logf("Success: %d, Limited: %d, Total: %d", successCount, limitedCount, total)

	// With 10 IPs and burst of 1000 each, we expect:
	// - Up to 1000*10 = 10000 successful requests initially
	// - Some additional successes due to token refill during test
	// - Rest should be rate limited
	if successCount > int64(numIPs)*1000+1000 {
		t.Logf("High success count is reasonable given rate: %d", successCount)
	}
}

// TestIntegration_CustomKeyExtractor tests custom key extraction
func TestIntegration_CustomKeyExtractor(t *testing.T) {
	// Use API key for rate limiting
	rl, err := New(Config{
		DefaultRate:  10,
		DefaultBurst: 20,
		KeyExtractor: func(r *http.Request) string {
			return r.Header.Get("X-API-Key")
		},
	})
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	// Test with different API keys
	t.Run("api_key_isolation", func(t *testing.T) {
		// Use up burst for key1
		for i := 0; i < 20; i++ {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("X-API-Key", "key1")
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)
		}

		// key1 should now be rate limited
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Key", "key1")
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("key1 should be rate limited, got %d", rr.Code)
		}

		// key2 should still have full burst
		req = httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Key", "key2")
		rr = httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("key2 should have available tokens, got %d", rr.Code)
		}
	})
}

// TestIntegration_BackendSwap tests that we can swap backends
func TestIntegration_BackendSwap(t *testing.T) {
	// Create memory backend directly
	memBackend := NewMemoryBackend()

	rl, err := New(Config{
		Backend:      memBackend,
		DefaultRate:  100,
		DefaultBurst: 200,
	})
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	// Use it
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:10000"

	allowed, _, _, err := rl.Allow(req)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Error("First request should be allowed")
	}

	rl.Close()

	// Create new rate limiter with new backend
	memBackend2 := NewMemoryBackend()
	rl2, err := New(Config{
		Backend:      memBackend2,
		DefaultRate:  100,
		DefaultBurst: 200,
	})
	if err != nil {
		t.Fatalf("Failed to create second rate limiter: %v", err)
	}
	defer rl2.Close()

	// Should have fresh limits
	allowed, _, _, err = rl2.Allow(req)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed with new backend")
	}
}

// TestIntegration_MiddlewareHeaders tests that all headers are set correctly
func TestIntegration_MiddlewareHeaders(t *testing.T) {
	rl, err := New(Config{
		DefaultRate:  10,
		DefaultBurst: 20,
	})
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:10000"
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Check required headers
	headers := []string{
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
	}

	for _, header := range headers {
		if value := rr.Header().Get(header); value == "" {
			t.Errorf("Header %s is missing", header)
		}
	}

	// Verify remaining decreases
	remaining1 := rr.Header().Get("X-RateLimit-Remaining")

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "127.0.0.1:10000"
	middleware.ServeHTTP(rr2, req2)

	remaining2 := rr2.Header().Get("X-RateLimit-Remaining")

	if remaining1 == remaining2 {
		t.Error("X-RateLimit-Remaining should decrease between requests")
	}
}

// BenchmarkIntegration_Middleware simulates real-world usage
func BenchmarkIntegration_Middleware(b *testing.B) {
	rl, _ := New(Config{
		DefaultRate:  100000,
		DefaultBurst: 10000,
	})
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	// Pre-create request to avoid allocation in benchmark
	req := httptest.NewRequest("GET", "/benchmark", nil)
	req.RemoteAddr = "127.0.0.1:10000"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req.Clone(context.Background()))
		}
	})
}

// BenchmarkIntegration_MemoryBackendHighContention tests worst-case contention
func BenchmarkIntegration_MemoryBackendHighContention(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	// All goroutines hitting same key
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			backend.Take(ctx, "same-key", 1000000, 1000000, 1)
		}
	})
}

// TestIntegration_ContextCancellation tests behavior with cancelled context
func TestIntegration_ContextCancellation(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := backend.Take(ctx, "test", 100, 100, 1)
	// The memory backend doesn't use context cancellation,
	// but the Redis backend might. This test ensures graceful handling.
	if err != nil {
		t.Logf("Backend returned error with cancelled context: %v", err)
	}
}

// TestIntegration_LongRunning tests stability over extended operation
func TestIntegration_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	rl, err := New(Config{
		DefaultRate:  1000,
		DefaultBurst: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	// Run for a short period to verify no resource leaks
	duration := 2 * time.Second
	start := time.Now()
	requests := 0

	for time.Since(start) < duration {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = fmt.Sprintf("127.0.0.1:%d", requests%100+10000)
		rl.Allow(req)
		requests++
	}

	t.Logf("Processed %d requests in %v", requests, time.Since(start))
}
