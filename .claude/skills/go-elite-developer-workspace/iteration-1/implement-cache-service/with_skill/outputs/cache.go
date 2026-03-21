// Package cache provides a thread-safe LRU cache with TTL support.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Cache defines the interface for cache implementations.
type Cache interface {
	// Get retrieves a value from the cache by key.
	// Returns the value and true if found, nil and false otherwise.
	Get(key string) (interface{}, bool)

	// Set stores a value in the cache with the given key and TTL.
	// If ttl is 0, the entry will use the cache's default TTL.
	Set(key string, value interface{}, ttl time.Duration)

	// Delete removes an entry from the cache by key.
	Delete(key string)

	// Clear removes all entries from the cache.
	Clear()

	// Len returns the number of items in the cache.
	Len() int
}

// entry represents a cache entry with metadata.
type entry struct {
	key        string
	value      interface{}
	expiration int64 // Unix timestamp in nanoseconds, 0 means no expiration
}

// isExpired returns true if the entry has expired.
func (e *entry) isExpired(now int64) bool {
	return e.expiration > 0 && now > e.expiration
}

// LRUCache implements a thread-safe LRU cache with TTL support.
type LRUCache struct {
	capacity    int
	defaultTTL  time.Duration
	items       map[string]*list.Element
	order       *list.List // Doubly linked list for LRU order (front = most recent)
	mu          sync.RWMutex
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// Option configures the LRUCache.
type Option func(*LRUCache)

// WithCapacity sets the maximum number of items in the cache.
// Default is 0 (unlimited).
func WithCapacity(capacity int) Option {
	return func(c *LRUCache) {
		c.capacity = capacity
	}
}

// WithDefaultTTL sets the default TTL for cache entries.
// Default is 0 (no expiration).
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *LRUCache) {
		c.defaultTTL = ttl
	}
}

// NewLRUCache creates a new LRU cache with the given options.
func NewLRUCache(opts ...Option) *LRUCache {
	c := &LRUCache{
		capacity:    0,
		defaultTTL:  0,
		items:       make(map[string]*list.Element),
		order:       list.New(),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Start background cleanup goroutine only if TTL is used
	go c.cleanupExpired()

	return c
}

// Stop gracefully stops the background cleanup goroutine.
// Should be called when the cache is no longer needed.
func (c *LRUCache) Stop() {
	close(c.stopCleanup)
	<-c.cleanupDone
}

// Get retrieves a value from the cache.
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	elem, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	elem, exists = c.items[key]
	if !exists {
		return nil, false
	}

	ent := elem.Value.(*entry)

	// Check expiration
	if ent.isExpired(time.Now().UnixNano()) {
		c.removeElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)

	return ent.value, true
}

// Set stores a value in the cache.
func (c *LRUCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use default TTL if not specified
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	// Calculate expiration time
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	// Check if key already exists
	if elem, exists := c.items[key]; exists {
		// Update existing entry
		ent := elem.Value.(*entry)
		ent.value = value
		ent.expiration = expiration
		c.order.MoveToFront(elem)
		return
	}

	// Create new entry
	ent := &entry{
		key:        key,
		value:      value,
		expiration: expiration,
	}

	elem := c.order.PushFront(ent)
	c.items[key] = elem

	// Evict oldest entries if over capacity
	if c.capacity > 0 && c.order.Len() > c.capacity {
		c.evictOldest()
	}
}

// Delete removes an entry from the cache.
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		c.removeElement(elem)
	}
}

// Clear removes all entries from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
}

// Len returns the number of items in the cache.
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.order.Len()
}

// removeElement removes an element from the cache.
// Must be called with lock held.
func (c *LRUCache) removeElement(elem *list.Element) {
	ent := elem.Value.(*entry)
	delete(c.items, ent.key)
	c.order.Remove(elem)
}

// evictOldest removes the oldest entry from the cache.
// Must be called with lock held.
func (c *LRUCache) evictOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// cleanupExpired periodically removes expired entries.
func (c *LRUCache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	defer close(c.cleanupDone)

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// removeExpired removes all expired entries.
func (c *LRUCache) removeExpired() {
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Iterate from oldest to newest
	for elem := c.order.Back(); elem != nil; {
		prev := elem.Prev()

		ent := elem.Value.(*entry)
		if ent.isExpired(now) {
			c.removeElement(elem)
		}

		elem = prev
	}
}
