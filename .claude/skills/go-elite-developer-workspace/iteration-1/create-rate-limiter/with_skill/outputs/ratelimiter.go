// Package ratelimiter provides a high-performance token bucket rate limiter
// with support for multiple backends (in-memory and Redis).
//
// The package is designed for minimal allocations and thread-safe operations,
// making it suitable for high-throughput HTTP APIs.
//
// Basic usage:
//
//	limiter := ratelimiter.New(ratelimiter.Config{
//	    DefaultRate:  100,
//	    DefaultBurst: 200,
//	})
//	defer limiter.Close()
//
//	handler := limiter.Middleware(myHandler)
package ratelimiter

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"
)

// Common errors returned by the rate limiter.
var (
	ErrRateExceeded  = errors.New("rate limit exceeded")
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrBackendClosed = errors.New("backend is closed")
)

// Backend defines the interface for rate limiter storage backends.
// Implementations must be safe for concurrent use.
type Backend interface {
	// Take attempts to take n tokens from the bucket identified by key.
	// Returns the number of tokens remaining, the time until the next token is available,
	// and any error encountered.
	Take(ctx context.Context, key string, rate float64, burst int, n int) (remaining int, reset time.Time, err error)

	// Close releases any resources held by the backend.
	Close() error
}

// KeyExtractor extracts a rate limit key from an HTTP request.
// Common implementations include IP-based, user-based, or API-key-based extractors.
type KeyExtractor func(r *http.Request) string

// Config configures the rate limiter behavior.
type Config struct {
	// Backend is the storage backend. If nil, an in-memory backend is used.
	Backend Backend

	// DefaultRate is the token refill rate per second.
	DefaultRate float64

	// DefaultBurst is the maximum bucket capacity.
	DefaultBurst int

	// KeyExtractor extracts the rate limit key from requests.
	// Defaults to IP-based extraction if nil.
	KeyExtractor KeyExtractor
}

// RateLimiter is an HTTP-aware token bucket rate limiter.
type RateLimiter struct {
	backend      Backend
	defaultRate  float64
	defaultBurst int
	extractKey   KeyExtractor
	routes       map[string]RouteConfig
}

// RouteConfig defines rate limits for a specific route pattern.
type RouteConfig struct {
	Rate  float64
	Burst int
}

// New creates a new RateLimiter with the given configuration.
// The Config must have valid DefaultRate and DefaultBurst values.
func New(cfg Config) (*RateLimiter, error) {
	if cfg.DefaultRate <= 0 {
		return nil, ErrInvalidConfig
	}
	if cfg.DefaultBurst <= 0 {
		return nil, ErrInvalidConfig
	}

	backend := cfg.Backend
	if backend == nil {
		backend = NewMemoryBackend()
	}

	extractKey := cfg.KeyExtractor
	if extractKey == nil {
		extractKey = ExtractIP
	}

	return &RateLimiter{
		backend:      backend,
		defaultRate:  cfg.DefaultRate,
		defaultBurst: cfg.DefaultBurst,
		extractKey:   extractKey,
		routes:       make(map[string]RouteConfig),
	}, nil
}

// SetRoute configures rate limits for a specific route pattern.
// The pattern is used as a key to look up configuration; actual routing
// is handled by the caller.
func (rl *RateLimiter) SetRoute(pattern string, rate float64, burst int) error {
	if rate <= 0 {
		return ErrInvalidConfig
	}
	if burst <= 0 {
		return ErrInvalidConfig
	}

	rl.routes[pattern] = RouteConfig{
		Rate:  rate,
		Burst: burst,
	}
	return nil
}

// Allow checks if the request is allowed under the rate limit.
// Returns the number of remaining requests and the reset time.
func (rl *RateLimiter) Allow(r *http.Request) (allowed bool, remaining int, reset time.Time, err error) {
	route := r.URL.Path
	config, ok := rl.routes[route]
	if !ok {
		config = RouteConfig{
			Rate:  rl.defaultRate,
			Burst: rl.defaultBurst,
		}
	}

	key := rl.extractKey(r) + ":" + route
	remaining, reset, err = rl.backend.Take(r.Context(), key, config.Rate, config.Burst, 1)
	if err != nil {
		return false, 0, time.Time{}, err
	}

	return remaining >= 0, remaining, reset, nil
}

// Middleware returns an HTTP middleware that enforces rate limiting.
// When the limit is exceeded, it responds with 429 Too Many Requests
// and includes a Retry-After header.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, remaining, reset, err := rl.Allow(r)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))

		if !allowed {
			retryAfter := time.Until(reset)
			if retryAfter < 0 {
				retryAfter = time.Second
			}
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Close releases resources held by the rate limiter.
func (rl *RateLimiter) Close() error {
	return rl.backend.Close()
}

// ExtractIP extracts the client IP from the request.
// It checks X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func ExtractIP(r *http.Request) string {
	// Check X-Forwarded-For header (common with proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs; use the first one
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := splitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// splitHostPort splits a host:port string, handling IPv6 addresses.
func splitHostPort(addr string) (string, string, error) {
	// IPv6 addresses are wrapped in brackets like [::1]:8080
	if len(addr) == 0 {
		return "", "", errors.New("empty address")
	}

	// Find the last colon which separates host and port
	colonIdx := -1
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			colonIdx = i
			break
		}
	}

	if colonIdx == -1 {
		return addr, "", nil
	}

	// Check if this is an IPv6 address in brackets
	if addr[0] == '[' {
		// Find the closing bracket
		bracketEnd := -1
		for i := 0; i < len(addr); i++ {
			if addr[i] == ']' {
				bracketEnd = i
				break
			}
		}
		if bracketEnd != -1 && colonIdx > bracketEnd {
			// Port is after the bracket
			return addr[1:bracketEnd], addr[colonIdx+1:], nil
		}
		// No port, just IPv6 address
		return addr[1 : len(addr)-1], "", nil
	}

	return addr[:colonIdx], addr[colonIdx+1:], nil
}
