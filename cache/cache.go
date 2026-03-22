// Package cache provides an optional caching layer for the reaction system.
// It implements LRU eviction with TTL support and provides metrics for monitoring.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Entry represents a cached item with expiration information.
type Entry struct {
	Key        string
	Value      interface{}
	Expiration time.Time
}

// IsExpired returns true if the entry has expired.
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.Expiration)
}

// Cache defines the interface for cache implementations.
type Cache interface {
	// Get retrieves a value from the cache.
	// Returns (value, true) if found and not expired.
	// Returns (nil, false) if not found or expired.
	Get(key string) (interface{}, bool)

	// Set stores a value in the cache with the specified TTL.
	Set(key string, value interface{}, ttl time.Duration)

	// Delete removes a value from the cache.
	Delete(key string)

	// DeleteByPrefix removes all values with keys matching the given prefix.
	DeleteByPrefix(prefix string)

	// Clear removes all entries from the cache.
	Clear()

	// Len returns the number of entries in the cache.
	Len() int

	// Stats returns cache statistics.
	Stats() Stats
}

// Stats holds cache statistics.
type Stats struct {
	// Hits is the number of cache hits.
	Hits int64

	// Misses is the number of cache misses.
	Misses int64

	// Entries is the current number of entries in the cache.
	Entries int

	// MaxEntries is the maximum number of entries allowed.
	MaxEntries int
}

// HitRatio returns the ratio of hits to total requests.
func (s Stats) HitRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// MissRatio returns the ratio of misses to total requests.
func (s Stats) MissRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Misses) / float64(total)
}

// lruCache implements Cache with LRU eviction and TTL support.
type lruCache struct {
	maxEntries int
	items      map[string]*list.Element
	order      *list.List // Front is most recently used, Back is least recently used
	mu         sync.RWMutex
	hits       int64
	misses     int64
}

// New creates a new LRU cache with the specified maximum number of entries.
func New(maxEntries int) Cache {
	return &lruCache{
		maxEntries: maxEntries,
		items:      make(map[string]*list.Element),
		order:      list.New(),
	}
}

// Get retrieves a value from the cache.
func (c *lruCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		c.misses++
		return nil, false
	}

	entry := elem.Value.(*Entry)
	if entry.IsExpired() {
		c.removeElement(elem)
		c.misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	c.hits++
	return entry.Value, true
}

// Set stores a value in the cache with the specified TTL.
func (c *lruCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if elem, ok := c.items[key]; ok {
		// Update existing entry
		entry := elem.Value.(*Entry)
		entry.Value = value
		entry.Expiration = time.Now().Add(ttl)
		c.order.MoveToFront(elem)
		return
	}

	// Add new entry
	entry := &Entry{
		Key:        key,
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
	elem := c.order.PushFront(entry)
	c.items[key] = elem

	// Evict oldest entries if over capacity
	for c.order.Len() > c.maxEntries {
		c.removeOldest()
	}
}

// Delete removes a value from the cache.
func (c *lruCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// DeleteByPrefix removes all values with keys matching the given prefix.
func (c *lruCache) DeleteByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, elem := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.removeElement(elem)
		}
	}
}

// Clear removes all entries from the cache.
func (c *lruCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
}

// Len returns the number of entries in the cache.
func (c *lruCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.order.Len()
}

// Stats returns cache statistics.
func (c *lruCache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Stats{
		Hits:       c.hits,
		Misses:     c.misses,
		Entries:    c.order.Len(),
		MaxEntries: c.maxEntries,
	}
}

// removeOldest removes the least recently used entry.
func (c *lruCache) removeOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes an element from the cache.
func (c *lruCache) removeElement(elem *list.Element) {
	c.order.Remove(elem)
	entry := elem.Value.(*Entry)
	delete(c.items, entry.Key)
}
