package ratelimiter

import (
	"net/http"
	"strconv"
)

// ResponseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code before writing.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write captures writes and ensures status is set.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Middleware creates an HTTP middleware that applies rate limiting.
// The middleware checks the request path and applies the appropriate rate limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check rate limit
		result, err := rl.CheckHTTPRequest(r)
		if err != nil {
			// On error, allow the request but log the error
			// In production, you might want to fail closed
			http.Error(w, "Rate limiter error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.getLimitForPath(r.URL.Path)))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))

		if !result.Allowed {
			w.Header().Set("Retry-After", rl.GetRetryAfterHeader(result))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc creates a middleware function compatible with http.HandlerFunc.
func (rl *RateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return rl.Middleware(next).ServeHTTP
}

// Handler wraps an http.Handler with rate limiting.
// This allows applying rate limits to specific handlers rather than routes.
func (rl *RateLimiter) Handler(path string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Override the path for configuration lookup
		r.URL.Path = path

		// Check rate limit
		result, err := rl.CheckHTTPRequest(r)
		if err != nil {
			http.Error(w, "Rate limiter error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.getLimitForPath(path)))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))

		if !result.Allowed {
			w.Header().Set("Retry-After", rl.GetRetryAfterHeader(result))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// HandlerFunc wraps an http.HandlerFunc with rate limiting.
func (rl *RateLimiter) HandlerFunc(path string, next http.HandlerFunc) http.HandlerFunc {
	return rl.Handler(path, next).ServeHTTP
}

// Wrap wraps an http.Handler with rate limiting for a specific configuration.
// This allows using a custom rate configuration different from the path-based config.
func (rl *RateLimiter) Wrap(next http.Handler, config EndpointConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.Rate.IsZero() {
			next.ServeHTTP(w, r)
			return
		}

		// Generate key using config's key function or default
		keyFunc := config.KeyFunc
		if keyFunc == nil {
			keyFunc = rl.config.DefaultKeyFunc
		}

		// Create a wrapper to satisfy KeyFunc interface
		key := keyFunc(r)

		tokens := config.TokensConsumed
		if tokens == 0 {
			tokens = 1
		}

		result, err := rl.AllowN(r.Context(), key, tokens, config.Rate)
		if err != nil {
			http.Error(w, "Rate limiter error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Rate.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))

		if !result.Allowed {
			w.Header().Set("Retry-After", rl.GetRetryAfterHeader(result))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getLimitForPath returns the limit for the given path.
func (rl *RateLimiter) getLimitForPath(path string) int {
	config := rl.config.GetConfigForPath(path)
	return config.Rate.Limit
}

// skipRateLimit checks if rate limiting should be skipped for the request.
// This can be used to skip health checks or internal requests.
func skipRateLimit(r *http.Request, skipPaths []string) bool {
	for _, path := range skipPaths {
		if r.URL.Path == path {
			return true
		}
	}
	return false
}

// MiddlewareWithSkip creates a middleware that skips rate limiting for specified paths.
func (rl *RateLimiter) MiddlewareWithSkip(next http.Handler, skipPaths []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipRateLimit(r, skipPaths) {
			next.ServeHTTP(w, r)
			return
		}

		rl.Middleware(next).ServeHTTP(w, r)
	})
}
