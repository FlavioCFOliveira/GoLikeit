package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// bucket represents a token bucket for rate limiting.
type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

// MemoryBackend implements an in-memory token bucket rate limiter.
// It uses a sharded map to reduce lock contention for high concurrency.
type MemoryBackend struct {
	shards [16]*shard
	stopCh chan struct{}
	wg     sync.WaitGroup
	ttl    time.Duration
}

// shard is a single shard of the bucket map with its own lock.
type shard struct {
	buckets map[string]*bucket
	mu      sync.RWMutex
}

// getShard returns the shard for a given key.
func (mb *MemoryBackend) getShard(key string) *shard {
	// Simple hash function: sum of bytes mod shard count
	var sum uint32
	for i := 0; i < len(key); i++ {
		sum += uint32(key[i])
	}
	return mb.shards[sum%16]
}

// NewMemoryBackend creates a new in-memory rate limiter backend.
// It starts a background goroutine to clean up expired buckets.
func NewMemoryBackend() *MemoryBackend {
	mb := &MemoryBackend{
		stopCh: make(chan struct{}),
		ttl:    time.Hour, // Buckets expire after 1 hour of inactivity
	}

	for i := 0; i < 16; i++ {
		mb.shards[i] = &shard{
			buckets: make(map[string]*bucket),
		}
	}

	// Start cleanup goroutine
	mb.wg.Add(1)
	go mb.cleanup()

	return mb
}

// Take implements the Backend interface.
func (mb *MemoryBackend) Take(ctx context.Context, key string, rate float64, burst int, n int) (int, time.Time, error) {
	shard := mb.getShard(key)

	// Get or create bucket
	shard.mu.Lock()
	b, exists := shard.buckets[key]
	if !exists {
		b = &bucket{
			tokens:   float64(burst),
			lastSeen: time.Now(),
		}
		shard.buckets[key] = b
	}
	shard.mu.Unlock()

	// Lock the bucket and process the request
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	// Calculate tokens to add since last access
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * rate
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}

	b.lastSeen = now

	// Check if we have enough tokens
	if b.tokens < float64(n) {
		// Calculate when enough tokens will be available
		needed := float64(n) - b.tokens
		secondsUntil := needed / rate
		reset := now.Add(time.Duration(secondsUntil * float64(time.Second)))
		// Return -1 to indicate rate limit exceeded
		return -1, reset, nil
	}

	// Take tokens
	b.tokens -= float64(n)
	reset := now.Add(time.Duration(float64(time.Second) / rate))

	return int(b.tokens), reset, nil
}

// Close implements the Backend interface.
func (mb *MemoryBackend) Close() error {
	close(mb.stopCh)
	mb.wg.Wait()
	return nil
}

// cleanup periodically removes expired buckets to prevent memory leaks.
func (mb *MemoryBackend) cleanup() {
	defer mb.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mb.cleanupExpired()
		case <-mb.stopCh:
			mb.cleanupExpired()
			return
		}
	}
}

// cleanupExpired removes buckets that haven't been accessed recently.
func (mb *MemoryBackend) cleanupExpired() {
	cutoff := time.Now().Add(-mb.ttl)

	for _, shard := range mb.shards {
		shard.mu.Lock()
		for key, b := range shard.buckets {
			b.mu.Lock()
			lastSeen := b.lastSeen
			b.mu.Unlock()

			if lastSeen.Before(cutoff) {
				delete(shard.buckets, key)
			}
		}
		shard.mu.Unlock()
	}
}
