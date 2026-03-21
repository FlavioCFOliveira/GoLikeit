package ratelimiter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				DefaultRate:  100,
				DefaultBurst: 200,
			},
			wantErr: false,
		},
		{
			name: "zero rate",
			cfg: Config{
				DefaultRate:  0,
				DefaultBurst: 200,
			},
			wantErr: true,
		},
		{
			name: "negative rate",
			cfg: Config{
				DefaultRate:  -1,
				DefaultBurst: 200,
			},
			wantErr: true,
		},
		{
			name: "zero burst",
			cfg: Config{
				DefaultRate:  100,
				DefaultBurst: 0,
			},
			wantErr: true,
		},
		{
			name: "negative burst",
			cfg: Config{
				DefaultRate:  100,
				DefaultBurst: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := New(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}
			defer rl.Close()
		})
	}
}

func TestMemoryBackend_Take(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	t.Run("initial bucket is full", func(t *testing.T) {
		remaining, reset, err := backend.Take(ctx, "test-key-1", 10, 100, 1)
		if err != nil {
			t.Fatalf("Take() error = %v", err)
		}
		if remaining != 99 {
			t.Errorf("Take() remaining = %v, want %v", remaining, 99)
		}
		if reset.IsZero() {
			t.Error("Take() reset time should not be zero")
		}
	})

	t.Run("bucket depletes", func(t *testing.T) {
		key := "test-key-2"
		rate := float64(10)
		burst := 5

		// Take more than burst
		for i := 0; i < burst+1; i++ {
			remaining, _, err := backend.Take(ctx, key, rate, burst, 1)
			if err != nil {
				t.Fatalf("Take() iteration %d error = %v", i, err)
			}

			if i < burst {
				expectedRemaining := burst - i - 1
				if remaining != expectedRemaining {
					t.Errorf("Take() iteration %d remaining = %v, want %v", i, remaining, expectedRemaining)
				}
			} else {
				// Should be rate limited
				if remaining >= 0 {
					t.Errorf("Take() iteration %d should be rate limited, remaining = %v", i, remaining)
				}
			}
		}
	})

	t.Run("token refill", func(t *testing.T) {
		key := "test-key-3"
		rate := float64(100) // 100 tokens per second
		burst := 10

		// Deplete the bucket
		backend.Take(ctx, key, rate, burst, burst)

		// Wait for tokens to refill
		time.Sleep(100 * time.Millisecond)

		remaining, _, err := backend.Take(ctx, key, rate, burst, 1)
		if err != nil {
			t.Fatalf("Take() error = %v", err)
		}

		// Should have some tokens refilled (roughly 10 tokens per 100ms at 100/sec)
		if remaining < 0 {
			t.Error("Take() should have tokens after waiting")
		}
	})
}

func TestRateLimiter_Allow(t *testing.T) {
	rl, err := New(Config{
		DefaultRate:  10,
		DefaultBurst: 20,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer rl.Close()

	t.Run("allows requests within burst", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		for i := 0; i < 20; i++ {
			allowed, _, _, err := rl.Allow(req)
			if err != nil {
				t.Fatalf("Allow() iteration %d error = %v", i, err)
			}
			if !allowed {
				t.Errorf("Allow() iteration %d should be allowed", i)
			}
		}
	})

	t.Run("rejects requests over burst", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test2", nil)
		req.RemoteAddr = "127.0.0.1:12346"

		// Use up burst
		for i := 0; i < 20; i++ {
			rl.Allow(req)
		}

		// Next request should be denied
		allowed, _, _, err := rl.Allow(req)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if allowed {
			t.Error("Allow() should reject request over burst")
		}
	})

	t.Run("different routes have separate limits", func(t *testing.T) {
		rl2, _ := New(Config{
			DefaultRate:  100,
			DefaultBurst: 100,
		})
		defer rl2.Close()

		req1 := httptest.NewRequest("GET", "/api/route1", nil)
		req1.RemoteAddr = "127.0.0.1:12347"

		req2 := httptest.NewRequest("GET", "/api/route2", nil)
		req2.RemoteAddr = "127.0.0.1:12347"

		// Use up burst on route1
		for i := 0; i < 100; i++ {
			rl2.Allow(req1)
		}

		// route2 should still have full burst
		allowed, remaining, _, err := rl2.Allow(req2)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if !allowed {
			t.Error("Allow() route2 should be allowed")
		}
		if remaining != 99 {
			t.Errorf("Allow() remaining = %v, want %v", remaining, 99)
		}
	})
}

func TestRateLimiter_Middleware(t *testing.T) {
	rl, err := New(Config{
		DefaultRate:  10,
		DefaultBurst: 5,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := rl.Middleware(handler)

	t.Run("successful requests", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("iteration %d: got status %d, want %d", i, rr.Code, http.StatusOK)
			}

			remaining := rr.Header().Get("X-RateLimit-Remaining")
			if remaining == "" {
				t.Error("X-RateLimit-Remaining header missing")
			}

			reset := rr.Header().Get("X-RateLimit-Reset")
			if reset == "" {
				t.Error("X-RateLimit-Reset header missing")
			}
		}
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		// Use up all tokens first
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/ratelimit", nil)
			req.RemoteAddr = "127.0.0.1:12346"
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)
		}

		// This request should be rate limited
		req := httptest.NewRequest("GET", "/ratelimit", nil)
		req.RemoteAddr = "127.0.0.1:12346"
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("got status %d, want %d", rr.Code, http.StatusTooManyRequests)
		}

		retryAfter := rr.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Error("Retry-After header missing")
		}
	})
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "simple IPv4",
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "IPv6",
			remoteAddr: "[::1]:12345",
			want:       "::1",
		},
		{
			name:       "X-Forwarded-For",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			want:       "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Real-IP": "10.0.0.1"},
			want:       "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For priority",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "10.0.0.2",
			},
			want: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			got := ExtractIP(req)
			if got != tt.want {
				t.Errorf("ExtractIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetRoute(t *testing.T) {
	rl, _ := New(Config{
		DefaultRate:  10,
		DefaultBurst: 20,
	})
	defer rl.Close()

	tests := []struct {
		name    string
		pattern string
		rate    float64
		burst   int
		wantErr bool
	}{
		{
			name:    "valid route config",
			pattern: "/api/special",
			rate:    100,
			burst:   200,
			wantErr: false,
		},
		{
			name:    "zero rate",
			pattern: "/api/invalid",
			rate:    0,
			burst:   100,
			wantErr: true,
		},
		{
			name:    "zero burst",
			pattern: "/api/invalid",
			rate:    100,
			burst:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rl.SetRoute(tt.pattern, tt.rate, tt.burst)
			if tt.wantErr {
				if err == nil {
					t.Error("SetRoute() expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("SetRoute() unexpected error: %v", err)
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	rl, _ := New(Config{
		DefaultRate:  1000,
		DefaultBurst: 1000,
	})
	defer rl.Close()

	const numGoroutines = 100
	const requestsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*requestsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/concurrent", nil)
				req.RemoteAddr = "127.0.0.1:" + strconv.Itoa(10000+id)

				_, _, _, err := rl.Allow(req)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Errorf("got %d errors during concurrent access", errorCount)
	}
}

func BenchmarkMemoryBackend_Take(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune('a' + i%26))
			backend.Take(ctx, key, 1000, 1000, 1)
			i++
		}
	})
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl, _ := New(Config{
		DefaultRate:  10000,
		DefaultBurst: 10000,
	})
	defer rl.Close()

	req := httptest.NewRequest("GET", "/benchmark", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow(req)
		}
	})
}

func BenchmarkExtractIP(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ExtractIP(req)
		}
	})
}

// splitHostPort testing
func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{
			name:     "IPv4 with port",
			addr:     "192.168.1.1:8080",
			wantHost: "192.168.1.1",
			wantPort: "8080",
		},
		{
			name:     "IPv6 with port",
			addr:     "[::1]:8080",
			wantHost: "::1",
			wantPort: "8080",
		},
		{
			name:     "IPv6 without port",
			addr:     "[::1]",
			wantHost: "::1",
			wantPort: "",
		},
		{
			name:     "empty",
			addr:     "",
			wantHost: "",
			wantPort: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := splitHostPort(tt.addr)
			if tt.wantErr {
				if err == nil {
					t.Error("splitHostPort() expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("splitHostPort() unexpected error: %v", err)
				return
			}
			if host != tt.wantHost {
				t.Errorf("splitHostPort() host = %v, want %v", host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("splitHostPort() port = %v, want %v", port, tt.wantPort)
			}
		})
	}
}

// Override the strconv usage in TestConcurrency
func TestConcurrencyWithStrconv(t *testing.T) {
	// Use strings.Builder for efficient string building
	port := func(id int) string {
		var sb strings.Builder
		sb.WriteString("127.0.0.1:")
		sb.WriteString(strconv.Itoa(id + 10000))
		return sb.String()
	}

	rl, _ := New(Config{
		DefaultRate:  1000,
		DefaultBurst: 1000,
	})
	defer rl.Close()

	const numGoroutines = 50
	const requestsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/concurrent", nil)
				req.RemoteAddr = port(id)

				rl.Allow(req)
			}
		}(i)
	}

	wg.Wait()
}
