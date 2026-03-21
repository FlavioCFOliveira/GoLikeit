package ratelimiter

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRate_TokensPerSecond(t *testing.T) {
	tests := []struct {
		name     string
		rate     Rate
		expected float64
	}{
		{
			name:     "100 per minute",
			rate:     Rate{Limit: 100, Burst: 100, Period: time.Minute},
			expected: 100.0 / 60.0,
		},
		{
			name:     "60 per second",
			rate:     Rate{Limit: 60, Burst: 60, Period: time.Second},
			expected: 60.0,
		},
		{
			name:     "1000 per hour",
			rate:     Rate{Limit: 1000, Burst: 1000, Period: time.Hour},
			expected: 1000.0 / 3600.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rate.TokensPerSecond()
			// Allow small floating point differences
			if got < tt.expected*0.99 || got > tt.expected*1.01 {
				t.Errorf("TokensPerSecond() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRate_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		rate     Rate
		expected bool
	}{
		{
			name:     "zero rate",
			rate:     Rate{},
			expected: true,
		},
		{
			name:     "non-zero limit",
			rate:     Rate{Limit: 100},
			expected: false,
		},
		{
			name:     "non-zero burst",
			rate:     Rate{Burst: 100},
			expected: false,
		},
		{
			name:     "full rate",
			rate:     Rate{Limit: 100, Burst: 150, Period: time.Minute},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rate.IsZero()
			if got != tt.expected {
				t.Errorf("IsZero() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true")
	}

	if config.DefaultKeyFunc == nil {
		t.Error("expected DefaultKeyFunc to be set")
	}

	if len(config.Endpoints) == 0 {
		t.Error("expected some endpoint configurations")
	}

	// Check default rate is set
	if config.DefaultRate.Limit == 0 {
		t.Error("expected DefaultRate to have a limit")
	}
}

func TestConfig_GetConfigForPath(t *testing.T) {
	config := Config{
		Endpoints: []EndpointConfig{
			{
				Path:           "/api/users",
				Rate:           Rate{Limit: 50, Burst: 50, Period: time.Minute},
				TokensConsumed: 1,
			},
			{
				Path:           "/api/*",
				Rate:           Rate{Limit: 100, Burst: 100, Period: time.Minute},
				TokensConsumed: 1,
			},
		},
		DefaultRate:    Rate{Limit: 1000, Burst: 1000, Period: time.Minute},
		DefaultKeyFunc: DefaultKeyFunc(),
	}

	tests := []struct {
		name         string
		path         string
		expectedRate int
	}{
		{
			name:         "exact match",
			path:         "/api/users",
			expectedRate: 50,
		},
		{
			name:         "wildcard match",
			path:         "/api/posts",
			expectedRate: 100,
		},
		{
			name:         "no match uses default",
			path:         "/other/path",
			expectedRate: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epConfig := config.GetConfigForPath(tt.path)
			if epConfig.Rate.Limit != tt.expectedRate {
				t.Errorf("GetConfigForPath(%q) rate = %d, want %d",
					tt.path, epConfig.Rate.Limit, tt.expectedRate)
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		// Exact matches
		{"/api/users", "/api/users", true},
		{"/api/users", "/api/posts", false},

		// Suffix wildcards
		{"/api/*", "/api/users", true},
		{"/api/*", "/api/posts/comments", true},
		{"/api/*", "/other/path", false},
		{"/api/*", "/api", false}, // /api/ is not /api

		// Prefix wildcards
		{"*/api", "/v1/api", true},
		{"*/api", "/v2/api", true},
		{"*/api", "/api", true},
		{"*/api", "/api/users", false},

		// Global wildcard
		{"*", "/anything", true},
		{"*", "", true},

		// No match
		{"/api", "/api/users", false},
		{"/api/users/extra", "/api/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got := matchWildcard(tt.pattern, tt.path)
			if got != tt.expected {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v",
					tt.pattern, tt.path, got, tt.expected)
			}
		})
	}
}

func TestNewConfigBuilder(t *testing.T) {
	builder := NewConfigBuilder()

	t.Run("default values", func(t *testing.T) {
		config := builder.Build()

		if !config.Enabled {
			t.Error("expected Enabled to be true by default")
		}
		if config.DefaultKeyFunc == nil {
			t.Error("expected DefaultKeyFunc to be set")
		}
	})

	t.Run("with default rate", func(t *testing.T) {
		config := NewConfigBuilder().
			WithDefaultRate(500, 750, time.Minute).
			Build()

		if config.DefaultRate.Limit != 500 {
			t.Errorf("expected limit 500, got %d", config.DefaultRate.Limit)
		}
		if config.DefaultRate.Burst != 750 {
			t.Errorf("expected burst 750, got %d", config.DefaultRate.Burst)
		}
	})

	t.Run("with endpoint", func(t *testing.T) {
		config := NewConfigBuilder().
			WithEndpoint("/api/users", 50, 75, time.Minute).
			Build()

		if len(config.Endpoints) != 1 {
			t.Fatalf("expected 1 endpoint, got %d", len(config.Endpoints))
		}

		ep := config.Endpoints[0]
		if ep.Path != "/api/users" {
			t.Errorf("expected path /api/users, got %s", ep.Path)
		}
		if ep.Rate.Limit != 50 {
			t.Errorf("expected limit 50, got %d", ep.Rate.Limit)
		}
	})

	t.Run("with multiple endpoints", func(t *testing.T) {
		config := NewConfigBuilder().
			WithEndpoint("/api/users", 50, 75, time.Minute).
			WithEndpoint("/api/posts", 100, 150, time.Minute).
			WithEndpoint("/login", 5, 10, time.Minute).
			Build()

		if len(config.Endpoints) != 3 {
			t.Errorf("expected 3 endpoints, got %d", len(config.Endpoints))
		}
	})

	t.Run("chained configuration", func(t *testing.T) {
		config := NewConfigBuilder().
			WithDefaultRate(1000, 1500, time.Minute).
			WithEndpoint("/api/*", 100, 150, time.Minute).
			WithEndpoint("/login", 5, 10, time.Minute).
			Build()

		if config.DefaultRate.Limit != 1000 {
			t.Errorf("expected default limit 1000, got %d", config.DefaultRate.Limit)
		}
		if len(config.Endpoints) != 2 {
			t.Errorf("expected 2 endpoints, got %d", len(config.Endpoints))
		}
	})
}

func TestKeyFunc(t *testing.T) {
	t.Run("DefaultKeyFunc", func(t *testing.T) {
		fn := DefaultKeyFunc()
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		key := fn(req)

		if key != "192.168.1.1:1234" {
			t.Errorf("expected key '192.168.1.1:1234', got %s", key)
		}
	})

	t.Run("UserKeyFunc with header", func(t *testing.T) {
		fn := UserKeyFunc("X-User-ID")
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		req.Header.Set("X-User-ID", "user123")
		key := fn(req)

		if key != "user123" {
			t.Errorf("expected key 'user123', got %s", key)
		}
	})

	t.Run("UserKeyFunc fallback to IP", func(t *testing.T) {
		fn := UserKeyFunc("X-User-ID")
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		key := fn(req)

		if key != "192.168.1.1:1234" {
			t.Errorf("expected key '192.168.1.1:1234', got %s", key)
		}
	})

	t.Run("CompositeKeyFunc", func(t *testing.T) {
		fn := CompositeKeyFunc(
			DefaultKeyFunc(),
			UserKeyFunc("X-User-ID"),
		)
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		req.Header.Set("X-User-ID", "user123")
		key := fn(req)

		expected := "192.168.1.1:1234:user123"
		if key != expected {
			t.Errorf("expected key '%s', got %s", expected, key)
		}
	})
}
