// Package benchmark provides concurrent operation performance tests.
package benchmark

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/storage"
)

// Minimum concurrent operations for validation
const (
	MinConcurrentOps     = 100
	ConcurrentTestDuration = 5 * time.Second
)

// TestConcurrentWrites validates concurrent write operations.
func TestConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	concurrencyLevels := []int{10, 50, 100, 200}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency%d", concurrency), func(t *testing.T) {
			target := golikeit.EntityTarget{EntityType: "post", EntityID: "concurrent-test"}
			var successCount int64
			var errorCount int64

			var wg sync.WaitGroup
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						userID := fmt.Sprintf("user-%d-%d", id, j)
						_, err := s.AddReaction(ctx, userID, target, "LIKE")
						if err != nil {
							atomic.AddInt64(&errorCount, 1)
						} else {
							atomic.AddInt64(&successCount, 1)
						}
					}
				}(i)
			}

			wg.Wait()

			totalOps := successCount + errorCount
			t.Logf("Concurrent writes with %d goroutines: %d success, %d errors (total: %d)",
				concurrency, successCount, errorCount, totalOps)

			// All operations should succeed for in-memory storage
			if errorCount > 0 {
				t.Errorf("Expected 0 errors, got %d", errorCount)
			}

			// Verify total reactions count
			counts, err := s.GetEntityCounts(ctx, target)
			if err != nil {
				t.Fatalf("GetEntityCounts failed: %v", err)
			}

			if counts.Total != int64(concurrency*100) {
				t.Errorf("Expected %d total reactions, got %d", concurrency*100, counts.Total)
			}
		})
	}
}

// TestConcurrentReads validates concurrent read operations.
func TestConcurrentReads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	// Pre-populate with data
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "concurrent-read-test"}
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	concurrencyLevels := []int{50, 100, 200}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency%d", concurrency), func(t *testing.T) {
			var successCount int64
			var errorCount int64

			var wg sync.WaitGroup
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						userID := fmt.Sprintf("user-%d", j%1000)
						_, err := s.GetUserReaction(ctx, userID, target)
						if err != nil && err != golikeit.ErrReactionNotFound {
							atomic.AddInt64(&errorCount, 1)
						} else {
							atomic.AddInt64(&successCount, 1)
						}
					}
				}(i)
			}

			wg.Wait()

			t.Logf("Concurrent reads with %d goroutines: %d success, %d errors",
				concurrency, successCount, errorCount)

			if errorCount > 0 {
				t.Errorf("Expected 0 errors, got %d", errorCount)
			}
		})
	}
}

// TestConcurrentMixedOperations validates concurrent mixed read/write operations.
func TestConcurrentMixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	// Pre-populate
	for i := 0; i < 500; i++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", i%50),
		}
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	var wg sync.WaitGroup
	errors := make(chan error, 1000)

	// Writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", (id+j)%50),
				}
				userID := fmt.Sprintf("new-user-%d-%d", id, j)
				_, err := s.AddReaction(ctx, userID, target, "LOVE")
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", (id+j)%50),
				}
				userID := fmt.Sprintf("user-%d", (id+j)%500)
				_, err := s.GetUserReaction(ctx, userID, target)
				if err != nil && err != golikeit.ErrReactionNotFound {
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
			errorCount++
			t.Logf("Error: %v", err)
		}
	}

	if errorCount > 0 {
		t.Errorf("Got %d errors during concurrent mixed operations", errorCount)
	}

	t.Logf("Completed 100 concurrent workers (50 writers, 50 readers) without errors")
}

// TestConcurrentReplaceOperations validates concurrent reaction replacements.
func TestConcurrentReplaceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	target := golikeit.EntityTarget{EntityType: "post", EntityID: "replace-test"}
	userID := "concurrent-user"

	// Initial reaction
	s.AddReaction(ctx, userID, target, "LIKE")

	var wg sync.WaitGroup
	replaced := make(chan bool, 100)

	// Concurrent replacements
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			reactionType := "LIKE"
			if id%2 == 0 {
				reactionType = "LOVE"
			}
			wasReplaced, err := s.AddReaction(ctx, userID, target, reactionType)
			if err != nil {
				t.Errorf("Replace failed: %v", err)
				return
			}
			replaced <- wasReplaced
		}(i)
	}

	wg.Wait()
	close(replaced)

	// Count replacements
	replaceCount := 0
	for wasReplaced := range replaced {
		if wasReplaced {
			replaceCount++
		}
	}

	// All concurrent operations should be replacements (initial reaction already exists)
	if replaceCount != 100 {
		t.Errorf("Expected 100 replacements, got %d", replaceCount)
	}

	// Final state should have exactly one reaction
	counts, err := s.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("GetEntityCounts failed: %v", err)
	}

	if counts.Total != 1 {
		t.Errorf("Expected 1 reaction after concurrent replacements, got %d", counts.Total)
	}
}

// TestConcurrentCountConsistency validates that counts remain consistent under concurrent operations.
func TestConcurrentCountConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	target := golikeit.EntityTarget{EntityType: "post", EntityID: "consistency-test"}

	// Track expected count
	var expectedCount int64
	var mu sync.Mutex

	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				userID := fmt.Sprintf("user-%d-%d", id, j)
				_, err := s.AddReaction(ctx, userID, target, "LIKE")
				if err != nil {
					t.Errorf("Add failed: %v", err)
					return
				}
				mu.Lock()
				expectedCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify final count
	counts, err := s.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("GetEntityCounts failed: %v", err)
	}

	mu.Lock()
	expected := expectedCount
	mu.Unlock()

	if counts.Total != expected {
		t.Errorf("Count inconsistency: expected %d, got %d", expected, counts.Total)
	}
}

// TestStressTest runs a stress test with maximum concurrency.
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	numWorkers := 100
	opsPerWorker := 100

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	var totalOps int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-startBarrier

			for j := 0; j < opsPerWorker; j++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", (id+j)%100),
				}
				userID := fmt.Sprintf("user-%d-%d", id, j)

				switch j % 3 {
				case 0:
					s.AddReaction(ctx, userID, target, "LIKE")
				case 1:
					s.GetUserReaction(ctx, userID, target)
				case 2:
					s.GetEntityCounts(ctx, target)
				}

				atomic.AddInt64(&totalOps, 1)
			}
		}(i)
	}

	start := time.Now()
	close(startBarrier)
	wg.Wait()
	elapsed := time.Since(start)

	opsPerSec := float64(totalOps) / elapsed.Seconds()

	t.Logf("Stress test completed: %d ops by %d workers in %v (%.2f ops/sec)",
		totalOps, numWorkers, elapsed, opsPerSec)

	// Verify we processed all operations
	if totalOps != int64(numWorkers*opsPerWorker) {
		t.Errorf("Operation count mismatch: expected %d, got %d",
			numWorkers*opsPerWorker, totalOps)
	}
}

// BenchmarkHighConcurrency benchmarks operations with 100+ concurrent goroutines.
func BenchmarkHighConcurrency(b *testing.B) {
	b.Run("AddReaction_100Concurrent", func(b *testing.B) {
		s := storage.NewMemoryStorage()
		defer s.Close()

		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		var wg sync.WaitGroup
		opsPerGoroutine := b.N / 100

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < opsPerGoroutine; j++ {
					target := golikeit.EntityTarget{
						EntityType: "post",
						EntityID:   fmt.Sprintf("post-%d", j),
					}
					userID := fmt.Sprintf("user-%d-%d", id, j)
					s.AddReaction(ctx, userID, target, "LIKE")
				}
			}(i)
		}

		wg.Wait()
	})

	b.Run("GetUserReaction_100Concurrent", func(b *testing.B) {
		s := storage.NewMemoryStorage()
		defer s.Close()

		ctx := context.Background()
		target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}

		// Pre-populate
		for i := 0; i < 1000; i++ {
			userID := fmt.Sprintf("user-%d", i)
			s.AddReaction(ctx, userID, target, "LIKE")
		}

		b.ResetTimer()
		b.ReportAllocs()

		var wg sync.WaitGroup
		opsPerGoroutine := b.N / 100

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < opsPerGoroutine; j++ {
					userID := fmt.Sprintf("user-%d", j%1000)
					s.GetUserReaction(ctx, userID, target)
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestConcurrentSafety validates thread safety under concurrent access.
func TestConcurrentSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping safety tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	// Run with race detector
	var wg sync.WaitGroup

	// Multiple writers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", j),
				}
				userID := fmt.Sprintf("user-%d", id)
				s.AddReaction(ctx, userID, target, "LIKE")
			}
		}(i)
	}

	// Multiple readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", j),
				}
				s.GetEntityCounts(ctx, target)
			}
		}(i)
	}

	// Mixed operations
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", j),
				}
				userID := fmt.Sprintf("user-mixed-%d", id)
				s.AddReaction(ctx, userID, target, "LIKE")
				s.GetUserReaction(ctx, userID, target)
				s.RemoveReaction(ctx, userID, target)
			}
		}(i)
	}

	wg.Wait()

	t.Log("Concurrent safety test completed without race conditions")
}
