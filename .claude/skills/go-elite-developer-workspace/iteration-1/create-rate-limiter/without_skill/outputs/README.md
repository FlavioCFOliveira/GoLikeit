# Go Rate Limiter

A high-performance, thread-safe token bucket rate limiter for Go HTTP applications with support for multiple backends (in-memory and Redis).

## Features

- **Token Bucket Algorithm**: Smooth rate limiting with burst support
- **Per-Endpoint Configuration**: Different rate limits for different routes
- **Multiple Backends**: In-memory (single instance) and Redis (distributed)
- **HTTP Middleware**: Easy integration with `net/http` handlers
- **Thread-Safe**: Safe for concurrent use with minimal allocations
- **Configurable Key Functions**: Rate limit by IP, user ID, API key, or custom logic
- **Standard Headers**: Includes `X-RateLimit-*` and `Retry-After` headers

## Installation

```bash
go get github.com/example/ratelimiter
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"
    "time"
    "github.com/example/ratelimiter"
)

func main() {
    // Create backend (memory or Redis)
    backend := ratelimiter.NewMemoryBackend()
    defer backend.Close()

    // Configure rate limits
    config := ratelimiter.NewConfigBuilder().
        WithDefaultRate(100, 150, time.Minute).
        WithEndpoint("/api/login", 5, 10, time.Minute).
        Build()

    // Create rate limiter
    rl := ratelimiter.New(backend, config)

    // Create your handler
    mux := http.NewServeMux()
    mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Login successful"))
    })

    // Apply middleware
    handler := rl.Middleware(mux)

    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

## Configuration

### Rate Configuration

Each endpoint can have its own rate limit:

```go
rate := ratelimiter.Rate{
    Limit:  100,          // Maximum tokens in bucket
    Burst:  150,          // Maximum burst size
    Period: time.Minute,  // Time to refill limit tokens
}
```

### Per-Endpoint Configuration

```go
config := ratelimiter.NewConfigBuilder().
    // Default rate for unmatched paths
    WithDefaultRate(100, 150, time.Minute).
    // Specific endpoint with default key function (IP-based)
    WithEndpoint("/api/login", 5, 10, time.Minute).
    // Endpoint with custom key function
    WithEndpointAndKeyFunc("/api/admin/*", 50, 75, time.Minute,
        ratelimiter.UserKeyFunc("X-API-Key")).
    Build()
```

### Key Functions

Control how rate limit keys are generated:

```go
// By IP address (default)
keyFunc := ratelimiter.DefaultKeyFunc()

// By header value
keyFunc := ratelimiter.UserKeyFunc("X-User-ID")

// Composite key (IP + User)
keyFunc := ratelimiter.CompositeKeyFunc(
    ratelimiter.DefaultKeyFunc(),
    ratelimiter.UserKeyFunc("X-User-ID"),
)
```

## Backends

### Memory Backend

Suitable for single-instance deployments:

```go
backend := ratelimiter.NewMemoryBackend()
defer backend.Close()
```

### Redis Backend

For distributed rate limiting across multiple instances:

```go
backend, err := ratelimiter.NewRedisBackend(ratelimiter.RedisOptions{
    Addr:     "localhost:6379",
    Password: "",           // Leave empty if no password
    DB:       0,
    Prefix:   "ratelimit:", // Key prefix
})
if err != nil {
    log.Fatal(err)
}
defer backend.Close()
```

## Middleware Usage

### Global Middleware

```go
handler := rl.Middleware(yourHandler)
```

### Skip Specific Paths

```go
handler := rl.MiddlewareWithSkip(yourHandler, []string{"/health", "/ready"})
```

### Per-Handler Rate Limiting

```go
// Apply specific rate limit to a handler
limitedHandler := rl.Handler("/api/expensive", yourHandler)

// Or use Wrap with custom config
customConfig := ratelimiter.EndpointConfig{
    Path: "/api/special",
    Rate: ratelimiter.Rate{Limit: 1, Burst: 1, Period: time.Second},
}
wrappedHandler := rl.Wrap(yourHandler, customConfig)
```

## Response Headers

When rate limiting is applied, the following headers are included:

- `X-RateLimit-Limit`: Maximum number of requests allowed
- `X-RateLimit-Remaining`: Number of requests remaining in current window
- `X-RateLimit-Reset`: Unix timestamp when the limit resets
- `Retry-After`: Seconds until the request can be retried (429 responses only)

## Error Responses

When rate limit is exceeded:

- **Status Code**: 429 Too Many Requests
- **Body**: "Rate limit exceeded"
- **Header**: `Retry-After: <seconds>`

## Testing

Run all tests:

```bash
go test ./...
```

Run with Redis (requires Redis running locally):

```bash
REDIS_ADDR=localhost:6379 go test ./...
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Performance

Benchmarks show minimal overhead:

- Memory backend: ~100-200ns per request
- Redis backend: ~1-2ms per request (depends on network latency)
- Zero allocations in the hot path (memory backend)

## Design Considerations

1. **Thread Safety**: All operations are thread-safe using appropriate locking
2. **Memory Management**: Expired buckets are automatically cleaned up
3. **Redis Lua Scripting**: Atomic operations prevent race conditions
4. **Graceful Degradation**: On Redis errors, requests are allowed (fail open)

## License

MIT License
