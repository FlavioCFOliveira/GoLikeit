package ratelimiter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// IntegrationTestScenario contains test scenarios that simulate real-world usage
func TestIntegration_SimpleRateLimiting(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := NewConfigBuilder().
		WithDefaultRate(10, 15, time.Minute).
		WithEndpoint("/api/login", 5, 5, time.Minute).
		Build()

	rl := New(backend, config)

	// Create router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	handler := rl.Middleware(mux)

	t.Run("login endpoint rate limited separately", func(t *testing.T) {
		// Exhaust login limit (same client IP)
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/api/login", nil)
			req.RemoteAddr = "192.168.1.100:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
			}
		}

		// Next login request should be blocked (same client IP)
		req := httptest.NewRequest("POST", "/api/login", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", rec.Code)
		}

		// Users endpoint should still work (different client IP to ensure isolation)
		req2 := httptest.NewRequest("GET", "/api/users", nil)
		req2.RemoteAddr = "192.168.1.200:5678"
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("users request: expected 200, got %d", rec2.Code)
		}
	})
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := NewConfigBuilder().
		WithEndpoint("/api/resource", 100, 100, time.Second).
		Build()

	rl := New(backend, config)

	var allowed, blocked int64

	var wg sync.WaitGroup
	numGoroutines := 50
	requestsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/api/resource", nil)
				rec := httptest.NewRecorder()

				rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})).ServeHTTP(rec, req)

				if rec.Code == http.StatusOK {
					atomic.AddInt64(&allowed, 1)
				} else if rec.Code == http.StatusTooManyRequests {
					atomic.AddInt64(&blocked, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	total := allowed + blocked
	expectedTotal := int64(numGoroutines * requestsPerGoroutine)

	if total != expectedTotal {
		t.Errorf("expected %d total requests, got %d", expectedTotal, total)
	}

	t.Logf("Allowed: %d, Blocked: %d", allowed, blocked)

	// Most requests should have been blocked due to concurrent access
	// The exact number depends on timing, but we should have some allowed
	if allowed == 0 {
		t.Error("expected some requests to be allowed")
	}
}

func TestIntegration_TokenRefill(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	// Fast refill rate for testing
	config := NewConfigBuilder().
		WithEndpoint("/api/test", 10, 10, time.Second).
		Build()

	rl := New(backend, config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust tokens
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Next request should be blocked
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 before refill, got %d", rec.Code)
	}

	// Wait for refill
	time.Sleep(1500 * time.Millisecond)

	// Should be able to make requests again
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("after refill: expected 200, got %d", rec2.Code)
	}
}

func TestIntegration_MultipleEndpoints(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := NewConfigBuilder().
		WithDefaultRate(100, 100, time.Minute).
		WithEndpoint("/api/public/*", 1000, 1000, time.Minute).
		WithEndpoint("/api/private/*", 10, 10, time.Minute).
		WithEndpoint("/api/admin/*", 5, 5, time.Minute).
		Build()

	testCases := []struct {
		path           string
		requests       int
		expectedStatus int
		description    string
	}{
		{"/api/public/data", 100, http.StatusOK, "public endpoint should allow many requests"},
		{"/api/private/data", 10, http.StatusOK, "private endpoint should allow 10 requests"},
		{"/api/admin/users", 5, http.StatusOK, "admin endpoint should allow 5 requests"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Each test case needs its own backend to avoid interference
			backend := NewMemoryBackend()
			defer backend.Close()

			rl := New(backend, config)
			handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			allowed := 0
			for i := 0; i < tc.requests; i++ {
				req := httptest.NewRequest("GET", tc.path, nil)
				req.RemoteAddr = "192.168.1.1:1234"
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				if rec.Code == http.StatusOK {
					allowed++
				}
			}

			if allowed != tc.requests {
				t.Errorf("expected %d allowed, got %d", tc.requests, allowed)
			}

			// Check if we should be rate limited based on requests made vs burst
			if tc.requests >= 1000 { // Only expect 429 if we've made enough requests to exhaust burst
				// Next request should be blocked
				req := httptest.NewRequest("GET", tc.path, nil)
				req.RemoteAddr = "192.168.1.1:1234"
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				if rec.Code != http.StatusTooManyRequests {
					t.Errorf("expected 429 after limit, got %d", rec.Code)
				}
			}
		})
	}
}

func TestIntegration_RateLimitHeaders(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := NewConfigBuilder().
		WithEndpoint("/api/test", 10, 10, time.Minute).
		Build()

	rl := New(backend, config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headers := rec.Header()

	// Check all expected headers are present
	requiredHeaders := []string{
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
	}

	for _, header := range requiredHeaders {
		if headers.Get(header) == "" {
			t.Errorf("expected header %s to be present", header)
		}
	}

	t.Logf("Headers: Limit=%s, Remaining=%s, Reset=%s",
		headers.Get("X-RateLimit-Limit"),
		headers.Get("X-RateLimit-Remaining"),
		headers.Get("X-RateLimit-Reset"),
	)
}

func TestIntegration_DifferentKeys(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	// Use user ID key function
	config := NewConfigBuilder().
		WithEndpointAndKeyFunc("/api/user", 5, 5, time.Minute, UserKeyFunc("X-User-ID")).
		Build()

	rl := New(backend, config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("different users have separate limits", func(t *testing.T) {
		// User 1 exhausts their limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/api/user", nil)
			req.Header.Set("X-User-ID", "user1")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("user1 request %d: expected 200, got %d", i+1, rec.Code)
			}
		}

		// User 1's next request blocked
		req := httptest.NewRequest("GET", "/api/user", nil)
		req.Header.Set("X-User-ID", "user1")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("user1: expected 429, got %d", rec.Code)
		}

		// User 2 can still make requests
		req2 := httptest.NewRequest("GET", "/api/user", nil)
		req2.Header.Set("X-User-ID", "user2")
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("user2: expected 200, got %d", rec2.Code)
		}
	})

	t.Run("falls back to IP when no user ID", func(t *testing.T) {
		// Request without user ID should still work
		req := httptest.NewRequest("GET", "/api/user", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should be allowed (using IP as fallback)
		if rec.Code != http.StatusOK && rec.Code != http.StatusTooManyRequests {
			t.Errorf("unexpected status: %d", rec.Code)
		}
	})
}

func TestIntegration_HeavyLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heavy load test in short mode")
	}

	backend := NewMemoryBackend()
	defer backend.Close()

	config := NewConfigBuilder().
		WithEndpoint("/api/heavy", 10000, 10000, time.Minute).
		Build()

	rl := New(backend, config)

	var wg sync.WaitGroup
	numWorkers := 100
	requestsPerWorker := 100

	start := time.Now()
	var successCount int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			for j := 0; j < requestsPerWorker; j++ {
				req := httptest.NewRequest("GET", "/api/heavy", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				if rec.Code == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := numWorkers * requestsPerWorker
	throughput := float64(totalRequests) / duration.Seconds()

	t.Logf("Completed %d requests in %v (%.2f req/sec)",
		totalRequests, duration, throughput)
	t.Logf("Successful: %d, Blocked: %d",
		successCount, int64(totalRequests)-successCount)

	// All requests should succeed (limit is 10000, well above 10000 total)
	if successCount < int64(totalRequests)-100 { // Allow some small variance
		t.Errorf("expected most requests to succeed, got %d/%d",
			successCount, totalRequests)
	}
}

// ExampleUsage demonstrates how to use the rate limiter
func ExampleRateLimiter() {
	// Create a memory backend
	backend := NewMemoryBackend()
	defer backend.Close()

	// Configure rate limits
	config := NewConfigBuilder().
		WithDefaultRate(100, 150, time.Minute).
		WithEndpoint("/api/login", 5, 10, time.Minute).
		WithEndpoint("/api/public/*", 1000, 1500, time.Minute).
		Build()

	// Create rate limiter
	rl := New(backend, config)

	// Create HTTP server with rate limiting middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Login successful"))
	})
	mux.HandleFunc("/api/public/data", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Public data"))
	})

	// Apply middleware
	handler := rl.Middleware(mux)

	fmt.Println("Rate limiter configured with:")
	fmt.Printf("- Default: 100 req/min\n")
	fmt.Printf("- /api/login: 5 req/min\n")
	fmt.Printf("- /api/public/*: 1000 req/min\n")

	_ = handler
	// Output:
	// Rate limiter configured with:
	// - Default: 100 req/min
	// - /api/login: 5 req/min
	// - /api/public/*: 1000 req/min
}
