package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// MemoryBackend implements Backend using an in-memory map.
// This backend is suitable for single-instance deployments.
type MemoryBackend struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
	rates   map[string]Rate // key pattern -> rate config
	cleanup *time.Ticker
	done    chan struct{}
}

// NewMemoryBackend creates a new in-memory rate limiter backend.
func NewMemoryBackend() *MemoryBackend {
	mb := &MemoryBackend{
		buckets: make(map[string]*tokenBucket),
		rates:   make(map[string]Rate),
		done:    make(chan struct{}),
	}

	// Start cleanup goroutine
	mb.cleanup = time.NewTicker(1 * time.Minute)
	go mb.cleanupLoop()

	return mb
}

// cleanupLoop removes expired buckets periodically.
func (m *MemoryBackend) cleanupLoop() {
	for {
		select {
		case <-m.cleanup.C:
			m.removeExpiredBuckets()
		case <-m.done:
			return
		}
	}
}

// removeExpiredBuckets removes buckets that haven't been used recently.
// Buckets are considered expired after 10 minutes of inactivity.
func (m *MemoryBackend) removeExpiredBuckets() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for key, bucket := range m.buckets {
		// Check if bucket has expired
		bucket.mu.RLock()
		lastUsed := bucket.lastRefill
		bucket.mu.RUnlock()

		if lastUsed.Before(cutoff) {
			delete(m.buckets, key)
		}
	}
}

// Take attempts to take tokens from the bucket for the given key.
func (m *MemoryBackend) Take(ctx context.Context, key string, tokens int, rate Rate) (TakeResult, error) {
	now := time.Now()

	bucket := m.getOrCreateBucket(key, rate)
	allowed, remaining, retryAfter := bucket.take(now, tokens)

	_, resetTime := bucket.getState(now)

	return TakeResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}

// getOrCreateBucket returns an existing bucket or creates a new one.
func (m *MemoryBackend) getOrCreateBucket(key string, rate Rate) *tokenBucket {
	m.mu.RLock()
	bucket, exists := m.buckets[key]
	m.mu.RUnlock()

	if exists {
		return bucket
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, exists = m.buckets[key]; exists {
		return bucket
	}

	bucket = newTokenBucket(rate.Limit, rate.Burst, rate.Period)
	m.buckets[key] = bucket
	return bucket
}

// Close releases resources held by the backend.
func (m *MemoryBackend) Close() error {
	close(m.done)
	m.cleanup.Stop()

	m.mu.Lock()
	m.buckets = nil
	m.rates = nil
	m.mu.Unlock()

	return nil
}
