package ratelimiter

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// mockBackend is a test backend that can be configured to return specific results.
type mockBackend struct {
	results map[string]TakeResult
	mu      struct {
		sync.Mutex
		calls []string
	}
	err error
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		results: make(map[string]TakeResult),
	}
}

func (m *mockBackend) Take(ctx context.Context, key string, tokens int, rate Rate) (TakeResult, error) {
	m.mu.Lock()
	m.mu.calls = append(m.mu.calls, key)
	m.mu.Unlock()

	if m.err != nil {
		return TakeResult{}, m.err
	}
	if result, ok := m.results[key]; ok {
		return result, nil
	}
	return TakeResult{Allowed: true, Remaining: 100}, nil
}

func (m *mockBackend) Close() error {
	return nil
}

func TestRateLimiter_Allow(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	rl := New(backend, Config{
		Enabled:        true,
		DefaultKeyFunc: DefaultKeyFunc(),
	})

	rate := Rate{
		Limit:  10,
		Burst:  15,
		Period: time.Minute,
	}

	t.Run("allows request within limit", func(t *testing.T) {
		result, err := rl.Allow(context.Background(), "test-key", rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected request to be allowed")
		}
		if result.Remaining != 14 {
			t.Errorf("expected 14 remaining (burst - 1), got %d", result.Remaining)
		}
	})

	t.Run("exhausts tokens", func(t *testing.T) {
		key := "test-exhaust"
		rate := Rate{Limit: 5, Burst: 5, Period: time.Minute}

		// Exhaust the tokens
		for i := 0; i < 5; i++ {
			result, err := rl.Allow(context.Background(), key, rate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i+1)
			}
		}

		// Next request should be denied
		result, err := rl.Allow(context.Background(), key, rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected request to be denied after limit exhausted")
		}
		if result.RetryAfter <= 0 {
			t.Error("expected positive retry after time")
		}
	})

	t.Run("zero rate allows all", func(t *testing.T) {
		zeroRate := Rate{}
		result, err := rl.Allow(context.Background(), "any-key", zeroRate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected request to be allowed with zero rate")
		}
	})
}

func TestRateLimiter_AllowN(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	rl := New(backend, Config{
		Enabled:        true,
		DefaultKeyFunc: DefaultKeyFunc(),
	})

	rate := Rate{
		Limit:  10,
		Burst:  10,
		Period: time.Minute,
	}

	t.Run("allows when sufficient tokens", func(t *testing.T) {
		result, err := rl.AllowN(context.Background(), "test-burst", 5, rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected request to be allowed")
		}
		if result.Remaining != 5 {
			t.Errorf("expected 5 remaining, got %d", result.Remaining)
		}
	})

	t.Run("denies when insufficient tokens", func(t *testing.T) {
		result, err := rl.AllowN(context.Background(), "test-burst", 15, rate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected request to be denied")
		}
	})
}

func TestRateLimiter_CheckHTTPRequest(t *testing.T) {
	t.Run("uses endpoint config for matching path", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		config := Config{
			Enabled: true,
			Endpoints: []EndpointConfig{
				{
					Path: "/api/test",
					Rate: Rate{Limit: 5, Burst: 5, Period: time.Minute},
				},
			},
			DefaultRate:    Rate{Limit: 100, Burst: 150, Period: time.Minute},
			DefaultKeyFunc: DefaultKeyFunc(),
		}
		rl := New(backend, config)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"

		// Exhaust the endpoint limit
		for i := 0; i < 5; i++ {
			result, err := rl.CheckHTTPRequest(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i+1)
			}
		}

		// Next request should be denied
		result, err := rl.CheckHTTPRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("expected request to be denied after limit exhausted")
		}
	})

	t.Run("uses default config for non-matching path", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		config := Config{
			Enabled: true,
			Endpoints: []EndpointConfig{
				{
					Path: "/api/test",
					Rate: Rate{Limit: 5, Burst: 5, Period: time.Minute},
				},
			},
			DefaultRate:    Rate{Limit: 100, Burst: 150, Period: time.Minute},
			DefaultKeyFunc: DefaultKeyFunc(),
		}
		rl := New(backend, config)

		req := httptest.NewRequest("GET", "/other/path", nil)
		req.RemoteAddr = "192.168.1.2:1234"

		result, err := rl.CheckHTTPRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected request to be allowed with default rate")
		}
		if result.Remaining != 149 {
			t.Errorf("expected 149 remaining with default rate (burst - 1), got %d", result.Remaining)
		}
	})

	t.Run("disabled limiter allows all", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		config := Config{
			Enabled: true,
			Endpoints: []EndpointConfig{
				{
					Path: "/api/test",
					Rate: Rate{Limit: 5, Burst: 5, Period: time.Minute},
				},
			},
			DefaultRate:    Rate{Limit: 100, Burst: 150, Period: time.Minute},
			DefaultKeyFunc: DefaultKeyFunc(),
		}

		disabledConfig := config
		disabledConfig.Enabled = false
		disabledLimiter := New(backend, disabledConfig)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.3:1234"

		result, err := disabledLimiter.CheckHTTPRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("expected request to be allowed when disabled")
		}
	})
}

func TestRateLimiter_GetRetryAfterHeader(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	rl := New(backend, Config{})

	tests := []struct {
		name     string
		result   TakeResult
		expected string
	}{
		{
			name:     "with retry after duration",
			result:   TakeResult{RetryAfter: 60 * time.Second},
			expected: "60",
		},
		{
			name:     "zero retry after",
			result:   TakeResult{RetryAfter: 0, ResetTime: time.Now().Add(30 * time.Second)},
			expected: "30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := rl.GetRetryAfterHeader(tt.result)
			if header != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, header)
			}
		})
	}
}
