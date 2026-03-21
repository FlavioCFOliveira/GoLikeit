package ratelimiter_test

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/example/ratelimiter"
	"github.com/redis/go-redis/v9"
)

// ExampleNew demonstrates creating a basic rate limiter with default settings.
func ExampleNew() {
	rl, err := ratelimiter.New(ratelimiter.Config{
		DefaultRate:  100, // 100 requests per second
		DefaultBurst: 200, // Allow bursts up to 200
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	fmt.Println("Rate limiter created successfully")
	// Output: Rate limiter created successfully
}

// ExampleRateLimiter_Middleware shows how to use the rate limiter middleware.
func ExampleRateLimiter_Middleware() {
	rl, err := ratelimiter.New(ratelimiter.Config{
		DefaultRate:  100,
		DefaultBurst: 200,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	// Create your handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Wrap with rate limiter middleware
	rateLimitedHandler := rl.Middleware(handler)

	// Use rateLimitedHandler in your server
	http.Handle("/api/", rateLimitedHandler)
}

// ExampleRateLimiter_SetRoute demonstrates per-endpoint rate limiting.
func ExampleRateLimiter_SetRoute() {
	rl, err := ratelimiter.New(ratelimiter.Config{
		DefaultRate:  100,
		DefaultBurst: 200,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	// Set stricter limits for expensive endpoints
	rl.SetRoute("/api/generate-report", 1, 5)
	rl.SetRoute("/api/upload", 10, 20)

	// More lenient for public endpoints
	rl.SetRoute("/api/public/status", 1000, 2000)
}

// ExampleNewRedisBackend shows distributed rate limiting with Redis.
func ExampleNewRedisBackend() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  // default DB
	})

	// Create Redis-backed rate limiter
	backend := ratelimiter.NewRedisBackend(redisClient)

	rl, err := ratelimiter.New(ratelimiter.Config{
		Backend:      backend,
		DefaultRate:  100,
		DefaultBurst: 200,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	fmt.Println("Distributed rate limiter created")
	// Output: Distributed rate limiter created
}

// ExampleKeyExtractor shows custom key extraction for user-based rate limiting.
func ExampleConfig() {
	rl, err := ratelimiter.New(ratelimiter.Config{
		DefaultRate:  100,
		DefaultBurst: 200,
		// Custom key extractor: rate limit by API key instead of IP
		KeyExtractor: func(r *http.Request) string {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				// Fall back to IP if no API key
				return ratelimiter.ExtractIP(r)
			}
			return apiKey
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	// Now users are rate-limited by API key, not IP address
	_ = rl
}

// ExampleRateLimiter_Allow demonstrates direct rate limit checking.
func ExampleRateLimiter_Allow() {
	rl, _ := ratelimiter.New(ratelimiter.Config{
		DefaultRate:  10,
		DefaultBurst: 20,
	})
	defer rl.Close()

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	allowed, remaining, reset, err := rl.Allow(req)
	if err != nil {
		log.Fatal(err)
	}

	if allowed {
		fmt.Printf("Request allowed. Remaining: %d, Reset: %v\n", remaining, reset)
	} else {
		fmt.Printf("Rate limit exceeded. Retry after: %v\n", time.Until(reset))
	}
}

// Example_completeServer shows a complete HTTP server with rate limiting.
func Example_completeServer() {
	// Create rate limiter
	rl, err := ratelimiter.New(ratelimiter.Config{
		DefaultRate:  100,
		DefaultBurst: 200,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	// Configure different limits for different endpoints
	rl.SetRoute("/api/health", 10000, 10000) // Very permissive for health checks
	rl.SetRoute("/api/login", 5, 10)         // Strict for login (prevent brute force)
	rl.SetRoute("/api/webhook", 50, 100)     // Moderate for webhooks

	// Create handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"token":"abc123"}`))
	})
	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	})

	// Apply rate limiting to all routes
	handler := rl.Middleware(mux)

	// Start server
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// Example_memoryVsRedis shows when to use each backend.
func Example_memoryVsRedis() {
	// Option 1: In-memory backend (single instance)
	memoryLimiter, _ := ratelimiter.New(ratelimiter.Config{
		DefaultRate:  100,
		DefaultBurst: 200,
		// Backend is nil, so memory backend is used automatically
	})
	defer memoryLimiter.Close()

	// Option 2: Redis backend (distributed/multi-instance)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "redis-cluster:6379",
	})
	redisLimiter, _ := ratelimiter.New(ratelimiter.Config{
		Backend:      ratelimiter.NewRedisBackend(redisClient),
		DefaultRate:  100,
		DefaultBurst: 200,
	})
	defer redisLimiter.Close()

	fmt.Println("Both limiters created successfully")
	// Output: Both limiters created successfully
}
