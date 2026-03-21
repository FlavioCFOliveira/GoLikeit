package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/example/ratelimiter"
)

func main() {
	// Create a memory backend (use RedisBackend for distributed systems)
	backend := ratelimiter.NewMemoryBackend()
	defer backend.Close()

	// Configure rate limits for different endpoints
	config := ratelimiter.NewConfigBuilder().
		// Default rate for all endpoints
		WithDefaultRate(100, 150, time.Minute).
		// Login endpoint - very restrictive
		WithEndpoint("/api/login", 5, 10, time.Minute).
		// Public API - generous
		WithEndpoint("/api/public/*", 1000, 1500, time.Minute).
		// Admin API - moderate
		WithEndpointAndKeyFunc("/api/admin/*", 50, 75, time.Minute,
			ratelimiter.UserKeyFunc("X-API-Key")).
		Build()

	// Create the rate limiter
	rl := ratelimiter.New(backend, config)

	// Create router
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Login successful"}`))
	})

	mux.HandleFunc("/api/public/data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "Public data"}`))
	})

	mux.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users": []}`))
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users": []}`))
	})

	// Apply rate limiting middleware
	handler := rl.Middleware(mux)

	// Start server
	port := ":8080"
	fmt.Printf("Rate limiter server starting on %s\n", port)
	fmt.Println("\nRate limits:")
	fmt.Println("- /api/login: 5 requests/minute")
	fmt.Println("- /api/public/*: 1000 requests/minute")
	fmt.Println("- /api/admin/*: 50 requests/minute (per API key)")
	fmt.Println("- Default: 100 requests/minute")
	fmt.Println("\nExample requests:")
	fmt.Printf("  curl http://localhost%s/api/login\n", port)
	fmt.Printf("  curl http://localhost%s/api/public/data\n", port)
	fmt.Printf("  curl -H 'X-API-Key: admin-key' http://localhost%s/api/admin/users\n", port)

	log.Fatal(http.ListenAndServe(port, handler))
}
