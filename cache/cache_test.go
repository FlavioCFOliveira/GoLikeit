package cache

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New(100)
	if c == nil {
		t.Fatal("expected cache to not be nil")
	}
	if c.Len() != 0 {
		t.Errorf("expected empty cache, got %d entries", c.Len())
	}
}

func TestCache_GetSet(t *testing.T) {
	c := New(100)

	// Set a value
	c.Set("key1", "value1", time.Minute)

	// Get the value
	val, ok := c.Get("key1")
	if !ok {
		t.Error("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestCache_GetNotFound(t *testing.T) {
	c := New(100)

	val, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected not to find key")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New(100)

	// Set with short TTL
	c.Set("key1", "value1", 50*time.Millisecond)

	// Should be available immediately
	_, ok := c.Get("key1")
	if !ok {
		t.Error("expected to find key immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = c.Get("key1")
	if ok {
		t.Error("expected key to be expired")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New(100)

	c.Set("key1", "value1", time.Minute)
	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestCache_DeleteByPrefix(t *testing.T) {
	c := New(100)

	c.Set("prefix:key1", "value1", time.Minute)
	c.Set("prefix:key2", "value2", time.Minute)
	c.Set("other:key3", "value3", time.Minute)

	c.DeleteByPrefix("prefix:")

	_, ok := c.Get("prefix:key1")
	if ok {
		t.Error("expected prefix:key1 to be deleted")
	}
	_, ok = c.Get("prefix:key2")
	if ok {
		t.Error("expected prefix:key2 to be deleted")
	}

	// Other keys should remain
	val, ok := c.Get("other:key3")
	if !ok {
		t.Error("expected other:key3 to exist")
	}
	if val != "value3" {
		t.Errorf("expected value3, got %v", val)
	}
}

func TestCache_LRU(t *testing.T) {
	c := New(3)

	// Add 3 entries
	c.Set("key1", "value1", time.Minute)
	c.Set("key2", "value2", time.Minute)
	c.Set("key3", "value3", time.Minute)

	if c.Len() != 3 {
		t.Errorf("expected 3 entries, got %d", c.Len())
	}

	// Access key1 to make it recently used
	c.Get("key1")

	// Add 4th entry, should evict key2 (least recently used)
	c.Set("key4", "value4", time.Minute)

	if c.Len() != 3 {
		t.Errorf("expected 3 entries, got %d", c.Len())
	}

	_, ok := c.Get("key2")
	if ok {
		t.Error("expected key2 to be evicted (LRU)")
	}

	// key1 should still exist
	_, ok = c.Get("key1")
	if !ok {
		t.Error("expected key1 to exist (recently used)")
	}
}

func TestCache_Clear(t *testing.T) {
	c := New(100)

	c.Set("key1", "value1", time.Minute)
	c.Set("key2", "value2", time.Minute)
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("expected empty cache, got %d entries", c.Len())
	}

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be cleared")
	}
}

func TestCache_Stats(t *testing.T) {
	c := New(100)

	// Miss
	c.Get("key1")

	// Hit
	c.Set("key1", "value1", time.Minute)
	c.Get("key1")

	// Another hit
	c.Get("key1")

	stats := c.Stats()

	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.Entries != 1 {
		t.Errorf("expected 1 entry, got %d", stats.Entries)
	}
	if stats.MaxEntries != 100 {
		t.Errorf("expected max entries 100, got %d", stats.MaxEntries)
	}
}

func TestStats_HitRatio(t *testing.T) {
	tests := []struct {
		name     string
		hits     int64
		misses   int64
		expected float64
	}{
		{"all hits", 100, 0, 1.0},
		{"all misses", 0, 100, 0.0},
		{"50/50", 50, 50, 0.5},
		{"empty", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{Hits: tt.hits, Misses: tt.misses}
			got := s.HitRatio()
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestStats_MissRatio(t *testing.T) {
	tests := []struct {
		name     string
		hits     int64
		misses   int64
		expected float64
	}{
		{"all hits", 100, 0, 0.0},
		{"all misses", 0, 100, 1.0},
		{"50/50", 50, 50, 0.5},
		{"empty", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{Hits: tt.hits, Misses: tt.misses}
			got := s.MissRatio()
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestReactionCache(t *testing.T) {
	t.Run("enabled cache", func(t *testing.T) {
		config := DefaultReactionCacheConfig()
		c := NewReactionCache(config)

		if !c.IsEnabled() {
			t.Error("expected cache to be enabled")
		}

		// Set and get user reaction
		reaction := UserReaction{ReactionType: "LIKE", Timestamp: time.Now()}
		c.SetUserReaction("user1", "photo", "1", reaction)

		got, ok := c.GetUserReaction("user1", "photo", "1")
		if !ok {
			t.Error("expected to find user reaction")
		}
		if got.ReactionType != "LIKE" {
			t.Errorf("expected LIKE, got %s", got.ReactionType)
		}

		// Set and get entity counts
		counts := EntityCounts{
			CountsByType:   map[string]int64{"LIKE": 5},
			TotalReactions: 5,
			Timestamp:      time.Now(),
		}
		c.SetEntityCounts("photo", "1", counts)

		gotCounts, ok := c.GetEntityCounts("photo", "1")
		if !ok {
			t.Error("expected to find entity counts")
		}
		if gotCounts.TotalReactions != 5 {
			t.Errorf("expected total 5, got %d", gotCounts.TotalReactions)
		}
	})

	t.Run("disabled cache", func(t *testing.T) {
		config := DefaultReactionCacheConfig()
		config.Enabled = false
		c := NewReactionCache(config)

		if c.IsEnabled() {
			t.Error("expected cache to be disabled")
		}

		// Set should be no-op
		reaction := UserReaction{ReactionType: "LIKE", Timestamp: time.Now()}
		c.SetUserReaction("user1", "photo", "1", reaction)

		// Get should return not found
		_, ok := c.GetUserReaction("user1", "photo", "1")
		if ok {
			t.Error("expected not to find reaction in disabled cache")
		}
	})

	t.Run("entity invalidation", func(t *testing.T) {
		config := DefaultReactionCacheConfig()
		c := NewReactionCache(config)

		// Set multiple user reactions for an entity
		c.SetUserReaction("user1", "photo", "1", UserReaction{ReactionType: "LIKE", Timestamp: time.Now()})
		c.SetUserReaction("user2", "photo", "1", UserReaction{ReactionType: "LOVE", Timestamp: time.Now()})
		c.SetEntityCounts("photo", "1", EntityCounts{CountsByType: map[string]int64{"LIKE": 1, "LOVE": 1}, TotalReactions: 2})

		// Invalidate by entity
		c.InvalidateByEntity("photo", "1")

		// All entries for the entity should be gone
		_, ok := c.GetUserReaction("user1", "photo", "1")
		if ok {
			t.Error("expected user1 reaction to be invalidated")
		}
		_, ok = c.GetUserReaction("user2", "photo", "1")
		if ok {
			t.Error("expected user2 reaction to be invalidated")
		}
		_, ok = c.GetEntityCounts("photo", "1")
		if ok {
			t.Error("expected entity counts to be invalidated")
		}
	})

	t.Run("cache stats", func(t *testing.T) {
		config := DefaultReactionCacheConfig()
		c := NewReactionCache(config)

		// Generate some hits and misses
		c.SetUserReaction("user1", "photo", "1", UserReaction{ReactionType: "LIKE", Timestamp: time.Now()})
		c.GetUserReaction("user1", "photo", "1") // hit
		c.GetUserReaction("user2", "photo", "1") // miss

		c.SetEntityCounts("photo", "1", EntityCounts{CountsByType: map[string]int64{}})
		c.GetEntityCounts("photo", "1") // hit
		c.GetEntityCounts("video", "1") // miss

		stats := c.Stats()
		if stats.Hits != 2 {
			t.Errorf("expected 2 hits, got %d", stats.Hits)
		}
		if stats.Misses != 2 {
			t.Errorf("expected 2 misses, got %d", stats.Misses)
		}
	})
}

func TestUserReactionKey(t *testing.T) {
	key := userReactionKey("user1", "photo", "123")
	expected := "entity:photo:123:user:user1"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestEntityCountsKey(t *testing.T) {
	key := entityCountsKey("photo", "123")
	expected := "entity:photo:123:counts"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := New(100)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a' + n))
				c.Set(key, j, time.Minute)
				c.Get(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not have panicked, verify cache is still functional
	if c.Len() > 100 {
		t.Errorf("expected at most 100 entries (max), got %d", c.Len())
	}

	// Verify stats
	stats := c.Stats()
	if stats.Hits == 0 {
		t.Error("expected some hits")
	}
}

func BenchmarkCache_Get(b *testing.B) {
	c := New(1000)
	c.Set("key", "value", time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("key")
	}
}

func BenchmarkCache_Set(b *testing.B) {
	c := New(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(string(rune('a'+i%26)), i, time.Minute)
	}
}
