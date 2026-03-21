package ratelimiter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_AllowsRequests(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := Config{
		Enabled: true,
		Endpoints: []EndpointConfig{
			{
				Path: "/api/test",
				Rate: Rate{Limit: 10, Burst: 10, Period: time.Minute},
			},
		},
		DefaultRate:    Rate{Limit: 100, Burst: 100, Period: time.Minute},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	rl := New(backend, config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Check rate limit headers
	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("expected X-RateLimit-Limit header")
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

func TestMiddleware_BlocksRequests(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := Config{
		Enabled: true,
		Endpoints: []EndpointConfig{
			{
				Path: "/api/limited",
				Rate: Rate{Limit: 2, Burst: 2, Period: time.Minute},
			},
		},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	rl := New(backend, config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/limited", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	// Third request should be blocked
	req := httptest.NewRequest("GET", "/api/limited", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	// Check retry-after header
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on blocked request")
	}
}

func TestMiddlewareFunc(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := Config{
		Enabled:        true,
		DefaultRate:    Rate{Limit: 100, Burst: 100, Period: time.Minute},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	rl := New(backend, config)

	fn := rl.MiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	fn(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandler(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := Config{
		Enabled: true,
		Endpoints: []EndpointConfig{
			{
				Path: "/api/specific",
				Rate: Rate{Limit: 5, Burst: 5, Period: time.Minute},
			},
		},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	rl := New(backend, config)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with specific path config
	wrapped := rl.Handler("/api/specific", innerHandler)

	req := httptest.NewRequest("GET", "/different/path", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should still apply /api/specific config
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestWrap(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := Config{
		Enabled:        true,
		DefaultRate:    Rate{Limit: 100, Burst: 100, Period: time.Minute},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	rl := New(backend, config)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with custom config
	customConfig := EndpointConfig{
		Path: "/custom",
		Rate: Rate{Limit: 3, Burst: 3, Period: time.Minute},
	}

	wrapped := rl.Wrap(handler, customConfig)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/custom", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	// Fourth request should be blocked
	req := httptest.NewRequest("GET", "/custom", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}
}

func TestMiddlewareWithSkip(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := Config{
		Enabled: true,
		Endpoints: []EndpointConfig{
			{
				Path: "/api/*",
				Rate: Rate{Limit: 1, Burst: 1, Period: time.Minute},
			},
		},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	rl := New(backend, config)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware that skips /health
	wrapped := rl.MiddlewareWithSkip(innerHandler, []string{"/health"})

	// First request to /api/test uses the limit
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("first request: expected status 200, got %d", rec1.Code)
	}

	// Second request to /api/test should be blocked (limit is 1)
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected status 429, got %d", rec2.Code)
	}

	// Request to /health should never be blocked
	req3 := httptest.NewRequest("GET", "/health", nil)
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("health check: expected status 200, got %d", rec3.Code)
	}
}

func TestSkipRateLimit(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		skipPaths []string
		expected  bool
	}{
		{
			name:      "exact match",
			path:      "/health",
			skipPaths: []string{"/health"},
			expected:  true,
		},
		{
			name:      "no match",
			path:      "/api/users",
			skipPaths: []string{"/health"},
			expected:  false,
		},
		{
			name:      "empty skip list",
			path:      "/health",
			skipPaths: []string{},
			expected:  false,
		},
		{
			name:      "multiple skip paths",
			path:      "/metrics",
			skipPaths: []string{"/health", "/metrics", "/ready"},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			got := skipRateLimit(req, tt.skipPaths)
			if got != tt.expected {
				t.Errorf("skipRateLimit() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: rec}

	// Test that WriteHeader captures the status
	wrapped.WriteHeader(http.StatusCreated)
	if wrapped.statusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, wrapped.statusCode)
	}

	// Test that Write sets status to OK if not set
	rec2 := httptest.NewRecorder()
	wrapped2 := &responseWriter{ResponseWriter: rec2}
	wrapped2.Write([]byte("test"))
	if wrapped2.statusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, wrapped2.statusCode)
	}
}

func BenchmarkMiddleware(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	config := Config{
		Enabled:        true,
		DefaultRate:    Rate{Limit: 100000, Burst: 100000, Period: time.Minute},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	rl := New(backend, config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/benchmark", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}
