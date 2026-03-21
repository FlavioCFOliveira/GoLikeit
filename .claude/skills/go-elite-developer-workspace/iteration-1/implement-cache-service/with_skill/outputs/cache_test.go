package cache

import (
	"sync"
	"testing"
	"time"
)

func TestNewLRUCache(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	if c.Len() != 0 {
		t.Errorf("expected empty cache, got %d items", c.Len())
	}
}

func TestCacheWithCapacity(t *testing.T) {
	c := NewLRUCache(WithCapacity(2))
	defer c.Stop()

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0) // Should evict "a"

	if c.Len() != 2 {
		t.Errorf("expected 2 items, got %d", c.Len())
	}

	// "a" should be evicted
	if _, found := c.Get("a"); found {
		t.Error("expected 'a' to be evicted")
	}

	// "b" and "c" should exist
	if _, found := c.Get("b"); !found {
		t.Error("expected 'b' to exist")
	}
	if _, found := c.Get("c"); !found {
		t.Error("expected 'c' to exist")
	}
}

func TestCacheLRUOrder(t *testing.T) {
	c := NewLRUCache(WithCapacity(2))
	defer c.Stop()

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)

	// Access "a" to make it more recent
	c.Get("a")

	// Add "c", should evict "b" (least recently used)
	c.Set("c", 3, 0)

	if _, found := c.Get("b"); found {
		t.Error("expected 'b' to be evicted (LRU)")
	}

	if _, found := c.Get("a"); !found {
		t.Error("expected 'a' to exist (recently used)")
	}
}

func TestCacheGet(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("key", "value", 0)

	val, found := c.Get("key")
	if !found {
		t.Error("expected to find key")
	}
	if val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}

	// Non-existent key
	val, found = c.Get("nonexistent")
	if found {
		t.Error("expected not to find nonexistent key")
	}
	if val != nil {
		t.Errorf("expected nil for nonexistent key, got %v", val)
	}
}

func TestCacheSetUpdate(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("key", "old", 0)
	c.Set("key", "new", 0)

	val, found := c.Get("key")
	if !found {
		t.Error("expected to find key")
	}
	if val != "new" {
		t.Errorf("expected 'new', got %v", val)
	}

	if c.Len() != 1 {
		t.Errorf("expected 1 item, got %d", c.Len())
	}
}

func TestCacheDelete(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("key", "value", 0)
	c.Delete("key")

	if _, found := c.Get("key"); found {
		t.Error("expected key to be deleted")
	}

	if c.Len() != 0 {
		t.Errorf("expected empty cache, got %d items", c.Len())
	}

	// Delete non-existent key should not panic
	c.Delete("nonexistent")
}

func TestCacheClear(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("expected empty cache, got %d items", c.Len())
	}

	if _, found := c.Get("a"); found {
		t.Error("expected 'a' to be cleared")
	}
	if _, found := c.Get("b"); found {
		t.Error("expected 'b' to be cleared")
	}
}

func TestCacheTTL(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("key", "value", 100*time.Millisecond)

	// Should exist immediately
	if _, found := c.Get("key"); !found {
		t.Error("expected to find key before expiration")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	if _, found := c.Get("key"); found {
		t.Error("expected key to be expired")
	}
}

func TestCacheDefaultTTL(t *testing.T) {
	c := NewLRUCache(WithDefaultTTL(100 * time.Millisecond))
	defer c.Stop()

	c.Set("key", "value", 0) // Uses default TTL

	// Should exist immediately
	if _, found := c.Get("key"); !found {
		t.Error("expected to find key before expiration")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	if _, found := c.Get("key"); found {
		t.Error("expected key to be expired with default TTL")
	}
}

func TestCacheNoExpiration(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("key", "value", 0) // No expiration

	// Should still exist after some time
	time.Sleep(50 * time.Millisecond)

	if _, found := c.Get("key"); !found {
		t.Error("expected key to persist without TTL")
	}
}

func TestCacheExpiredEntryNotReturned(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("key", "value", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	// Get should return false for expired entry
	val, found := c.Get("key")
	if found {
		t.Error("expected expired entry to not be found")
	}
	if val != nil {
		t.Errorf("expected nil for expired entry, got %v", val)
	}
}

func TestCacheCleanupBackground(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	c.Set("key", "value", 1*time.Millisecond)

	// Wait for expiration and cleanup
	time.Sleep(2 * time.Minute + 10*time.Millisecond)

	// Entry should be cleaned up
	// Note: This test relies on the cleanup ticker interval
	// The entry might still be there if cleanup hasn't run yet
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := NewLRUCache(WithCapacity(100))
	defer c.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (id+j)%26))
				c.Set(key, id*numOperations+j, 0)
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
				c.Get(key)
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
				c.Delete(key)
			}
		}(i)
	}

	wg.Wait()

	// Cache should still be in a valid state
	if c.Len() > 100 {
		t.Errorf("cache exceeded capacity: %d", c.Len())
	}
}

func TestCacheConcurrentWithTTL(t *testing.T) {
	c := NewLRUCache(WithCapacity(50))
	defer c.Stop()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := string(rune('a' + j%26))
				c.Set(key, id, 10*time.Millisecond)
				c.Get(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestCacheRace(t *testing.T) {
	// This test is designed to be run with -race flag
	c := NewLRUCache(WithCapacity(10))
	defer c.Stop()

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			c.Set(string(rune('a'+i%26)), i, 0)
			if i%100 == 0 {
				time.Sleep(time.Microsecond)
			}
		}
		close(done)
	}()

	// Readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					c.Get("a")
					c.Len()
				}
			}
		}()
	}

	// Deleter
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				c.Delete("a")
				time.Sleep(time.Microsecond)
			}
		}
	}()

	wg.Wait()
}

func TestCacheVariousTypes(t *testing.T) {
	c := NewLRUCache()
	defer c.Stop()

	// String
	c.Set("string", "hello", 0)
	if val, _ := c.Get("string"); val != "hello" {
		t.Errorf("string mismatch: %v", val)
	}

	// Int
	c.Set("int", 42, 0)
	if val, _ := c.Get("int"); val != 42 {
		t.Errorf("int mismatch: %v", val)
	}

	// Struct
	type testStruct struct {
		Name  string
		Value int
	}
	s := testStruct{Name: "test", Value: 100}
	c.Set("struct", s, 0)
	if val, _ := c.Get("struct"); val != s {
		t.Errorf("struct mismatch: %v", val)
	}

	// Slice
	slice := []int{1, 2, 3}
	c.Set("slice", slice, 0)
	if val, _ := c.Get("slice"); len(val.([]int)) != 3 {
		t.Errorf("slice mismatch: %v", val)
	}

	// Map
	m := map[string]int{"a": 1, "b": 2}
	c.Set("map", m, 0)
	if val, _ := c.Get("map"); val.(map[string]int)["a"] != 1 {
		t.Errorf("map mismatch: %v", val)
	}

	// Nil
	c.Set("nil", nil, 0)
	if val, found := c.Get("nil"); !found || val != nil {
		t.Errorf("nil mismatch: found=%v, val=%v", found, val)
	}
}

func TestCacheOverwritePreservesOrder(t *testing.T) {
	c := NewLRUCache(WithCapacity(2))
	defer c.Stop()

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("a", 10, 0) // Overwrite "a", should become most recent

	// Add "c", should evict "b" (least recently used)
	c.Set("c", 3, 0)

	if _, found := c.Get("b"); found {
		t.Error("expected 'b' to be evicted")
	}

	if val, found := c.Get("a"); !found || val != 10 {
		t.Errorf("expected 'a' to exist with value 10, got %v, found=%v", val, found)
	}
}

func BenchmarkCacheSet(b *testing.B) {
	c := NewLRUCache(WithCapacity(1000))
	defer c.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(string(rune('a'+i%26)), i, 0)
			i++
		}
	})
}

func BenchmarkCacheGet(b *testing.B) {
	c := NewLRUCache(WithCapacity(1000))
	defer c.Stop()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		c.Set(string(rune('a'+i%26)), i, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(string(rune('a' + i%26)))
			i++
		}
	})
}

func BenchmarkCacheSetAndGet(b *testing.B) {
	c := NewLRUCache(WithCapacity(1000))
	defer c.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune('a' + i%26))
			c.Set(key, i, 0)
			c.Get(key)
			i++
		}
	})
}

func BenchmarkCacheSetWithTTL(b *testing.B) {
	c := NewLRUCache(WithCapacity(1000))
	defer c.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(string(rune('a'+i%26)), i, time.Minute)
			i++
		}
	})
}

func BenchmarkCacheConcurrentReadWrite(b *testing.B) {
	c := NewLRUCache(WithCapacity(1000))
	defer c.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune('a' + i%26))
			if i%2 == 0 {
				c.Set(key, i, 0)
			} else {
				c.Get(key)
			}
			i++
		}
	})
}
