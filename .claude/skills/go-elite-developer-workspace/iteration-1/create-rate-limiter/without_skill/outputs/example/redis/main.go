package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/example/ratelimiter"
)

func main() {
	// Get Redis configuration from environment
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Create Redis backend
	backend, err := ratelimiter.NewRedisBackend(ratelimiter.RedisOptions{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
		Prefix:   "ratelimit:",
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer backend.Close()

	fmt.Printf("Connected to Redis at %s\n", redisAddr)

	// Configure rate limits
	config := ratelimiter.NewConfigBuilder().
		WithDefaultRate(100, 150, time.Minute).
		WithEndpoint("/api/login", 5, 10, time.Minute).
		WithEndpoint("/api/public/*", 1000, 1500, time.Minute).
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

	// Apply rate limiting middleware
	handler := rl.Middleware(mux)

	// Start server
	port := ":8080"
	fmt.Printf("Redis rate limiter server starting on %s\n", port)
	fmt.Println("\nThis server uses Redis for distributed rate limiting.")
	fmt.Println("Multiple instances can share the same Redis backend.")

	log.Fatal(http.ListenAndServe(port, handler))
}
