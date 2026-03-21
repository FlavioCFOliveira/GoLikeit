package ratelimiter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// getRedisClient returns a Redis client for testing, or skips if REDIS_ADDR not set.
func getRedisClient(t *testing.T) *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR not set, skipping Redis tests")
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", addr, err)
	}

	return client
}

// cleanupRedis removes test keys from Redis.
func cleanupRedis(t *testing.T, client *redis.Client, pattern string) {
	ctx := context.Background()
	iter := client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		client.Del(ctx, iter.Val())
	}
	if err := iter.Err(); err != nil {
		t.Logf("Redis cleanup error: %v", err)
	}
}

func TestRedisBackend_Take(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	// Clean up any existing test keys
	cleanupRedis(t, client, "test:*")

	backend := NewRedisBackend(client)
	defer backend.Close()

	ctx := context.Background()

	t.Run("initial bucket is full", func(t *testing.T) {
		key := "test:initial-full"
		remaining, reset, err := backend.Take(ctx, key, 10, 100, 1)
		if err != nil {
			t.Fatalf("Take() error = %v", err)
		}
		if remaining < 0 {
			t.Errorf("Take() remaining = %v, should be >= 0", remaining)
		}
		if reset.IsZero() {
			t.Error("Take() reset time should not be zero")
		}

		// Clean up
		client.Del(ctx, key)
	})

	t.Run("bucket depletes", func(t *testing.T) {
		key := "test:deplete"
		rate := float64(1000) // High rate so we don't get refills during test
		burst := 5

		// Take more than burst
		for i := 0; i < burst+1; i++ {
			remaining, _, err := backend.Take(ctx, key, rate, burst, 1)
			if err != nil {
				t.Fatalf("Take() iteration %d error = %v", i, err)
			}

			if i < burst {
				if remaining < 0 {
					t.Errorf("Take() iteration %d should be allowed, remaining = %v", i, remaining)
				}
			} else {
				// Should be rate limited
				if remaining >= 0 {
					t.Errorf("Take() iteration %d should be rate limited, remaining = %v", i, remaining)
				}
			}
		}

		// Clean up
		client.Del(ctx, key)
	})

	t.Run("token refill", func(t *testing.T) {
		key := "test:refill"
		rate := float64(10) // 10 tokens per second
		burst := 5

		// Deplete the bucket
		for i := 0; i < burst; i++ {
			backend.Take(ctx, key, rate, burst, 1)
		}

		// Wait for tokens to refill
		time.Sleep(200 * time.Millisecond)

		// Should now have tokens
		remaining, _, err := backend.Take(ctx, key, rate, burst, 1)
		if err != nil {
			t.Fatalf("Take() error = %v", err)
		}
		if remaining < 0 {
			t.Error("Should have tokens after waiting")
		}

		// Clean up
		client.Del(ctx, key)
	})

	t.Run("multiple keys are isolated", func(t *testing.T) {
		key1 := "test:isolated-1"
		key2 := "test:isolated-2"

		// Deplete key1
		for i := 0; i < 100; i++ {
			backend.Take(ctx, key1, 100, 100, 1)
		}

		// key2 should still be allowed
		remaining, _, err := backend.Take(ctx, key2, 100, 100, 1)
		if err != nil {
			t.Fatalf("Take() error = %v", err)
		}
		if remaining < 0 {
			t.Error("key2 should have available tokens")
		}

		// Clean up
		client.Del(ctx, key1, key2)
	})
}

func TestRedisBackend_Concurrency(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	cleanupRedis(t, client, "test:concurrent:*")

	backend := NewRedisBackend(client)
	defer backend.Close()

	ctx := context.Background()

	const numGoroutines = 20
	const requestsPerGoroutine = 50

	t.Run("concurrent access to same key", func(t *testing.T) {
		key := "test:concurrent:same-key"

		// Allow plenty of capacity
		rate := float64(10000)
		burst := 10000

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*requestsPerGoroutine)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < requestsPerGoroutine; j++ {
					_, _, err := backend.Take(ctx, key, rate, burst, 1)
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Logf("Error: %v", err)
				errorCount++
			}
		}

		if errorCount > 0 {
			t.Errorf("Got %d errors during concurrent access", errorCount)
		}

		// Clean up
		client.Del(ctx, key)
	})
}

func BenchmarkRedisBackend_Take(b *testing.B) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		b.Skip("REDIS_ADDR not set")
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	backend := NewRedisBackend(client)
	defer backend.Close()

	// Pre-create keys for different goroutines
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = fmt.Sprintf("bench:key:%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := keys[i%len(keys)]
			backend.Take(ctx, key, 10000, 10000, 1)
			i++
		}
	})
}

// getRedisClient returns a Redis client for testing, or skips if REDIS_ADDR not set.
