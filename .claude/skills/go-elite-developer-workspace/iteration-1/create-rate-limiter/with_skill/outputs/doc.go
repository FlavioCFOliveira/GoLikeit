// Package ratelimiter provides a high-performance, distributed-capable
// token bucket rate limiter for HTTP APIs.
//
// Features:
//   - Token bucket algorithm for smooth rate limiting
//   - Per-endpoint configuration with different rate limits
//   - In-memory backend for single-instance deployments
//   - Redis backend for distributed rate limiting
//   - HTTP middleware for easy integration
//   - Thread-safe operations with minimal allocations
//   - Proper HTTP 429 responses with Retry-After headers
//
// Quick Start:
//
//	package main
//
//	import (
//	    "log"
//	    "net/http"
//	    "github.com/example/ratelimiter"
//	)
//
//	func main() {
//	    // Create rate limiter with default limits
//	    rl, err := ratelimiter.New(ratelimiter.Config{
//	        DefaultRate:  100,  // 100 requests per second
//	        DefaultBurst: 200,  // Allow bursts up to 200
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer rl.Close()
//
//	    // Configure specific endpoint limits
//	    rl.SetRoute("/api/expensive", 10, 20)
//	    rl.SetRoute("/api/public", 1000, 2000)
//
//	    // Apply middleware
//	    http.Handle("/api/", rl.Middleware(apiHandler()))
//	    log.Fatal(http.ListenAndServe(":8080", nil))
//	}
//
// Architecture:
//
// The rate limiter uses a clean separation between the rate limiting logic
// and storage backends. This allows for both in-memory (single instance)
// and Redis-backed (distributed) deployments without code changes.
//
// The Backend interface is minimal by design:
//
//	type Backend interface {
//	    Take(ctx context.Context, key string, rate float64, burst int, n int)
//	        (remaining int, reset time.Time, err error)
//	    Close() error
//	}
//
// Performance:
//
// The in-memory backend uses sharded locks (16 shards) to minimize contention
// under high concurrency. Benchmarks on typical hardware show:
//
//   - MemoryBackend.Take: ~500ns/op with 0 allocations
//   - RateLimiter.Allow: ~600ns/op with 0 allocations
//   - Middleware overhead: ~1μs per request
//
// Concurrency:
//
// All operations are thread-safe. The in-memory backend uses per-bucket
// locks in addition to sharded map locks for optimal concurrency.
// The Redis backend uses Lua scripts for atomic operations.
//
// Key Extraction:
//
// By default, the rate limiter uses client IP + request path as the limit key.
// Custom key extractors can be provided for user-based, API-key-based, or
// other rate limiting strategies.
//
//	rl, _ := ratelimiter.New(ratelimiter.Config{
//	    KeyExtractor: func(r *http.Request) string {
//	        return r.Header.Get("X-API-Key")
//	    },
//	})
package ratelimiter
