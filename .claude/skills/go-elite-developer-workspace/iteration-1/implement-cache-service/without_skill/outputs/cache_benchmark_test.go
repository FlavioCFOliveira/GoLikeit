package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// BenchmarkCache_Set benchmarks setting values in the cache
func BenchmarkCache_Set(b *testing.B) {
	cache := NewLRUCache(10000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
			i++
		}
	})
}

// BenchmarkCache_Get benchmarks getting values from the cache
func BenchmarkCache_Get(b *testing.B) {
	cache := NewLRUCache(10000)
	// Pre-populate cache
	for i := 0; i < 10000; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Get(fmt.Sprintf("key%d", i%10000))
			i++
		}
	})
}

// BenchmarkCache_SetAndGet benchmarks mixed operations
func BenchmarkCache_SetAndGet(b *testing.B) {
	cache := NewLRUCache(10000)
	// Pre-populate cache
	for i := 0; i < 5000; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				cache.Set(fmt.Sprintf("key%d", i%10000), i, time.Minute)
			} else {
				cache.Get(fmt.Sprintf("key%d", i%10000))
			}
			i++
		}
	})
}

// BenchmarkCache_SetWithTTL benchmarks setting values with TTL
func BenchmarkCache_SetWithTTL(b *testing.B) {
	cache := NewLRUCache(10000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Set(fmt.Sprintf("key%d", i), i, time.Duration(i%60)*time.Second)
			i++
		}
	})
}

// BenchmarkCache_ConcurrentMixed benchmarks concurrent mixed operations
func BenchmarkCache_ConcurrentMixed(b *testing.B) {
	cache := NewLRUCache(10000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			op := rng.Intn(4)
			switch op {
			case 0: // Set
				cache.Set(fmt.Sprintf("key%d", i%10000), i, time.Minute)
			case 1: // Get
				cache.Get(fmt.Sprintf("key%d", i%10000))
			case 2: // Delete
				cache.Delete(fmt.Sprintf("key%d", i%10000))
			case 3: // Set with TTL
				cache.Set(fmt.Sprintf("key%d", i%10000), i, time.Duration(rng.Intn(60))*time.Second)
			}
			i++
		}
	})
}

// BenchmarkCache_Eviction benchmarks cache eviction under load
func BenchmarkCache_Eviction(b *testing.B) {
	cache := NewLRUCache(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
	}
}

// BenchmarkCache_ConcurrentWrites benchmarks concurrent write operations
func BenchmarkCache_ConcurrentWrites(b *testing.B) {
	cache := NewLRUCache(100000)
	b.ResetTimer()

	var wg sync.WaitGroup
	numGoroutines := 10
	opsPerGoroutine := b.N / numGoroutines

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := fmt.Sprintf("goroutine%d-key%d", id, i)
				cache.Set(key, i, time.Minute)
			}
		}(g)
	}
	wg.Wait()
}

// BenchmarkCache_RandomAccess benchmarks random access patterns
func BenchmarkCache_RandomAccess(b *testing.B) {
	cache := NewLRUCache(10000)
	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
	}
	b.ResetTimer()

	rng := rand.New(rand.NewSource(42))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("key%d", rng.Intn(10000))
			if rng.Float32() < 0.5 {
				cache.Get(key)
			} else {
				cache.Set(key, rng.Int(), time.Minute)
			}
		}
	})
}

// BenchmarkCache_Delete benchmarks delete operations
func BenchmarkCache_Delete(b *testing.B) {
	cache := NewLRUCache(10000)
	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Delete(fmt.Sprintf("key%d", i%10000))
	}
}

// BenchmarkCache_Clear benchmarks clearing the cache
func BenchmarkCache_Clear(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		cache := NewLRUCache(10000)
		for j := 0; j < 10000; j++ {
			cache.Set(fmt.Sprintf("key%d", j), j, time.Minute)
		}
		b.StartTimer()
		cache.Clear()
	}
}

// BenchmarkCache_DifferentSizes benchmarks cache with different capacities
func BenchmarkCache_DifferentSizes(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			cache := NewLRUCache(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
				if i%2 == 0 {
					cache.Get(fmt.Sprintf("key%d", i/2))
				}
			}
		})
	}
}

// BenchmarkCache_MemoryUsage helps understand memory overhead
func BenchmarkCache_MemoryUsage(b *testing.B) {
	cache := NewLRUCache(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i, time.Minute)
	}
	b.Logf("Cache size: %d entries", cache.Len())
}
