package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewLRUCache(t *testing.T) {
	cache := NewLRUCache(10)
	if cache == nil {
		t.Fatal("expected cache to be non-nil")
	}
	if cache.Capacity() != 10 {
		t.Errorf("expected capacity 10, got %d", cache.Capacity())
	}
	if cache.Len() != 0 {
		t.Errorf("expected length 0, got %d", cache.Len())
	}
}

func TestCache_SetAndGet(t *testing.T) {
	cache := NewLRUCache(10)

	// Test basic set and get
	cache.Set("key1", "value1", 0)
	val, found := cache.Get("key1")
	if !found {
		t.Error("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	// Test non-existent key
	val, found = cache.Get("nonexistent")
	if found {
		t.Error("expected not to find nonexistent key")
	}
	if val != nil {
		t.Errorf("expected nil for nonexistent key, got %v", val)
	}
}

func TestCache_UpdateExistingKey(t *testing.T) {
	cache := NewLRUCache(10)

	cache.Set("key1", "value1", 0)
	cache.Set("key1", "value2", 0)

	val, found := cache.Get("key1")
	if !found {
		t.Error("expected to find key1")
	}
	if val != "value2" {
		t.Errorf("expected value2, got %v", val)
	}

	if cache.Len() != 1 {
		t.Errorf("expected length 1, got %d", cache.Len())
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	cache := NewLRUCache(10)

	// Set with 100ms TTL
	cache.Set("key1", "value1", 100*time.Millisecond)

	// Should exist immediately
	val, found := cache.Get("key1")
	if !found {
		t.Error("expected to find key1 before expiration")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	val, found = cache.Get("key1")
	if found {
		t.Error("expected key1 to be expired")
	}
	if val != nil {
		t.Errorf("expected nil for expired key, got %v", val)
	}
}

func TestCache_NoExpiration(t *testing.T) {
	cache := NewLRUCache(10)

	// Set with 0 TTL (no expiration)
	cache.Set("key1", "value1", 0)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Should still exist
	val, found := cache.Get("key1")
	if !found {
		t.Error("expected to find key1 (no expiration)")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestCache_LRUeviction(t *testing.T) {
	cache := NewLRUCache(3)

	// Fill cache
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)

	// Access key1 to make it recently used
	cache.Get("key1")

	// Add key4, should evict key2 (least recently used)
	cache.Set("key4", "value4", 0)

	if cache.Len() != 3 {
		t.Errorf("expected length 3, got %d", cache.Len())
	}

	// key1 should exist (recently accessed)
	_, found := cache.Get("key1")
	if !found {
		t.Error("expected key1 to exist (recently used)")
	}

	// key2 should be evicted
	_, found = cache.Get("key2")
	if found {
		t.Error("expected key2 to be evicted")
	}

	// key3 and key4 should exist
	_, found = cache.Get("key3")
	if !found {
		t.Error("expected key3 to exist")
	}
	_, found = cache.Get("key4")
	if !found {
		t.Error("expected key4 to exist")
	}
}

func TestCache_Delete(t *testing.T) {
	cache := NewLRUCache(10)

	cache.Set("key1", "value1", 0)
	cache.Delete("key1")

	_, found := cache.Get("key1")
	if found {
		t.Error("expected key1 to be deleted")
	}

	if cache.Len() != 0 {
		t.Errorf("expected length 0, got %d", cache.Len())
	}

	// Delete non-existent key should not panic
	cache.Delete("nonexistent")
}

func TestCache_Clear(t *testing.T) {
	cache := NewLRUCache(10)

	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)

	if cache.Len() != 3 {
		t.Errorf("expected length 3, got %d", cache.Len())
	}

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("expected length 0 after clear, got %d", cache.Len())
	}

	// All keys should be gone
	for _, key := range []string{"key1", "key2", "key3"} {
		_, found := cache.Get(key)
		if found {
			t.Errorf("expected %s to be cleared", key)
		}
	}
}

func TestCache_ZeroCapacity(t *testing.T) {
	// Zero capacity means unlimited
	cache := NewLRUCache(0)

	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i, 0)
	}

	if cache.Len() != 100 {
		t.Errorf("expected length 100, got %d", cache.Len())
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewLRUCache(100)
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Set(key, id*numOperations+j, time.Minute)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Get(key)
			}
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/2; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Delete(key)
			}
		}(i)
	}

	wg.Wait()

	// Cache should still be in a valid state
	if cache.Len() > 100 {
		t.Errorf("cache length %d exceeds capacity 100", cache.Len())
	}
}

func TestCache_ConcurrentClear(t *testing.T) {
	cache := NewLRUCache(100)
	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				cache.Set(string(rune('a'+id)), j, time.Minute)
			}
		}(i)
	}

	// Clear goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Clear()
			}
		}()
	}

	wg.Wait()

	// Should not panic and should be in valid state
	_ = cache.Len()
}

func TestCache_MixedTypes(t *testing.T) {
	cache := NewLRUCache(10)

	// Store different types
	cache.Set("string", "hello", 0)
	cache.Set("int", 42, 0)
	cache.Set("float", 3.14, 0)
	cache.Set("bool", true, 0)
	cache.Set("slice", []int{1, 2, 3}, 0)
	cache.Set("map", map[string]int{"a": 1}, 0)

	// Retrieve and verify types
	if val, _ := cache.Get("string"); val != "hello" {
		t.Errorf("expected string, got %v", val)
	}
	if val, _ := cache.Get("int"); val != 42 {
		t.Errorf("expected int, got %v", val)
	}
	if val, _ := cache.Get("float"); val != 3.14 {
		t.Errorf("expected float, got %v", val)
	}
	if val, _ := cache.Get("bool"); val != true {
		t.Errorf("expected bool, got %v", val)
	}
}

func TestCache_ExpirationDuringAccess(t *testing.T) {
	cache := NewLRUCache(10)

	// Set multiple keys with different TTLs
	cache.Set("short", 1, 50*time.Millisecond)
	cache.Set("long", 2, 200*time.Millisecond)

	// Wait for short to expire
	time.Sleep(100 * time.Millisecond)

	// Access long - should trigger cleanup of expired entries
	val, found := cache.Get("long")
	if !found {
		t.Error("expected to find 'long'")
	}
	if val != 2 {
		t.Errorf("expected 2, got %v", val)
	}

	// short should be gone
	_, found = cache.Get("short")
	if found {
		t.Error("expected 'short' to be expired")
	}
}
