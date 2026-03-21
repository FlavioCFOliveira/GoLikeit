// Package cache provides a thread-safe LRU cache with TTL support.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Cache defines the interface for cache operations.
type Cache interface {
	// Get retrieves a value from the cache by key.
	// Returns the value and true if found, nil and false otherwise.
	Get(key string) (interface{}, bool)

	// Set adds or updates a value in the cache with the specified TTL.
	// If ttl is 0, the entry will not expire.
	Set(key string, value interface{}, ttl time.Duration)

	// Delete removes a key from the cache.
	Delete(key string)

	// Clear removes all entries from the cache.
	Clear()

	// Len returns the number of items in the cache.
	Len() int

	// Capacity returns the maximum capacity of the cache.
	Capacity() int
}

// entry represents a single cache entry with TTL support.
type entry struct {
	key        string
	value      interface{}
	expiration time.Time
}

// isExpired checks if the entry has expired.
func (e *entry) isExpired() bool {
	return !e.expiration.IsZero() && time.Now().After(e.expiration)
}

// LRUCache implements a thread-safe LRU cache with TTL support.
type LRUCache struct {
	capacity int
	items    map[string]*list.Element
	order    *list.List
	mu       sync.RWMutex
}

// NewLRUCache creates a new LRU cache with the specified capacity.
// If capacity is 0, the cache has no size limit.
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves a value from the cache.
// Moves the accessed item to the front of the list (most recently used).
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	elem, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check after acquiring write lock
	elem, exists = c.items[key]
	if !exists {
		return nil, false
	}

	ent := elem.Value.(*entry)

	// Check if expired
	if ent.isExpired() {
		c.removeElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	return ent.value, true
}

// Set adds or updates a value in the cache.
func (c *LRUCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	// If key exists, update it
	if elem, exists := c.items[key]; exists {
		ent := elem.Value.(*entry)
		ent.value = value
		ent.expiration = expiration
		c.order.MoveToFront(elem)
		return
	}

	// Add new entry
	ent := &entry{
		key:        key,
		value:      value,
		expiration: expiration,
	}
	elem := c.order.PushFront(ent)
	c.items[key] = elem

	// Evict oldest if over capacity
	if c.capacity > 0 && c.order.Len() > c.capacity {
		c.evictOldest()
	}
}

// Delete removes a key from the cache.
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

// Capacity returns the maximum capacity of the cache.
func (c *LRUCache) Capacity() int {
	return c.capacity
}

// evictOldest removes the least recently used item.
func (c *LRUCache) evictOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes an element from the cache.
func (c *LRUCache) removeElement(elem *list.Element) {
	ent := elem.Value.(*entry)
	delete(c.items, ent.key)
	c.order.Remove(elem)
}
