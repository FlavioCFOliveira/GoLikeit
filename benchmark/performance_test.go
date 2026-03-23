// Package benchmark provides performance tests for the GoLikeit storage layer.
package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/storage"
)

// setupStorage creates a fresh in-memory storage for benchmarks.
func setupStorage(b *testing.B) *storage.MemoryStorage {
	b.Helper()
	s := storage.NewMemoryStorage()
	return s
}

// BenchmarkAddReaction measures performance of adding reactions.
func BenchmarkAddReaction(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post-1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		_, err := s.AddReaction(ctx, userID, target, "LIKE")
		if err != nil {
			b.Fatalf("AddReaction failed: %v", err)
		}
	}
}

// BenchmarkAddReactionParallel measures concurrent add performance.
func BenchmarkAddReactionParallel(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			target := golikeit.EntityTarget{
				EntityType: "post",
				EntityID:   fmt.Sprintf("post-%d", i%1000),
			}
			userID := fmt.Sprintf("user-%d", i)
			_, err := s.AddReaction(ctx, userID, target, "LIKE")
			if err != nil {
				b.Fatalf("AddReaction failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkRemoveReaction measures performance of removing reactions.
func BenchmarkRemoveReaction(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post-1"}

	// Pre-populate with reactions
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		err := s.RemoveReaction(ctx, userID, target)
		if err != nil {
			b.Fatalf("RemoveReaction failed: %v", err)
		}
	}
}

// BenchmarkRemoveReactionParallel measures concurrent remove performance.
func BenchmarkRemoveReactionParallel(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()

	// Pre-populate with reactions
	for i := 0; i < b.N; i++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", i%1000),
		}
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	b.ResetTimer()
	b.ReportAllocs()

	var i int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			target := golikeit.EntityTarget{
				EntityType: "post",
				EntityID:   fmt.Sprintf("post-%d", i%1000),
			}
			userID := fmt.Sprintf("user-%d", i)
			s.RemoveReaction(ctx, userID, target)
			i++
		}
	})
}

// BenchmarkGetUserReaction measures performance of retrieving a user's reaction.
func BenchmarkGetUserReaction(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post-1"}

	// Pre-populate
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i%1000)
		_, err := s.GetUserReaction(ctx, userID, target)
		if err != nil {
			b.Fatalf("GetUserReaction failed: %v", err)
		}
	}
}

// BenchmarkGetUserReactionParallel measures concurrent read performance.
func BenchmarkGetUserReactionParallel(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post-1"}

	// Pre-populate
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user-%d", i%1000)
			_, err := s.GetUserReaction(ctx, userID, target)
			if err != nil {
				b.Fatalf("GetUserReaction failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkGetEntityCounts measures performance of retrieving reaction counts.
func BenchmarkGetEntityCounts(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post-1"}

	// Pre-populate with 1000 reactions
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := s.GetEntityCounts(ctx, target)
		if err != nil {
			b.Fatalf("GetEntityCounts failed: %v", err)
		}
	}
}

// BenchmarkGetEntityCountsParallel measures concurrent count retrieval.
func BenchmarkGetEntityCountsParallel(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()

	// Pre-populate multiple entities
	for j := 0; j < 100; j++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", j),
		}
		for i := 0; i < 100; i++ {
			userID := fmt.Sprintf("user-%d", i)
			s.AddReaction(ctx, userID, target, "LIKE")
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			target := golikeit.EntityTarget{
				EntityType: "post",
				EntityID:   fmt.Sprintf("post-%d", i%100),
			}
			_, err := s.GetEntityCounts(ctx, target)
			if err != nil {
				b.Fatalf("GetEntityCounts failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkReplaceReaction measures performance of replacing an existing reaction.
func BenchmarkReplaceReaction(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post-1"}
	userID := "benchmark-user"

	// Add initial reaction
	s.AddReaction(ctx, userID, target, "LIKE")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reactionType := "LOVE"
		if i%2 == 0 {
			reactionType = "LIKE"
		}
		_, err := s.AddReaction(ctx, userID, target, reactionType)
		if err != nil {
			b.Fatalf("Replace reaction failed: %v", err)
		}
	}
}

// BenchmarkMixedWorkload simulates a realistic mixed read/write workload.
func BenchmarkMixedWorkload(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", i%100),
		}
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", i%100),
		}
		userID := fmt.Sprintf("user-%d", i%1000)

		switch i % 10 {
		case 0, 1: // 20% writes
			_, err := s.AddReaction(ctx, userID, target, "LIKE")
			if err != nil {
				b.Fatalf("AddReaction failed: %v", err)
			}
		case 2: // 10% deletes
			s.RemoveReaction(ctx, userID, target)
		default: // 70% reads
			_, err := s.GetUserReaction(ctx, userID, target)
			if err != nil && err != golikeit.ErrReactionNotFound {
				b.Fatalf("GetUserReaction failed: %v", err)
			}
		}
	}
}

// BenchmarkMemoryAllocation tracks memory allocations for key operations.
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("AddReaction", func(b *testing.B) {
		s := setupStorage(b)
		defer s.Close()

		ctx := context.Background()
		target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("user-%d", i)
			s.AddReaction(ctx, userID, target, "LIKE")
		}
	})

	b.Run("GetUserReaction", func(b *testing.B) {
		s := setupStorage(b)
		defer s.Close()

		ctx := context.Background()
		target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post"}

		// Pre-populate
		for i := 0; i < 1000; i++ {
			userID := fmt.Sprintf("user-%d", i)
			s.AddReaction(ctx, userID, target, "LIKE")
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("user-%d", i%1000)
			s.GetUserReaction(ctx, userID, target)
		}
	})

	b.Run("GetEntityCounts", func(b *testing.B) {
		s := setupStorage(b)
		defer s.Close()

		ctx := context.Background()
		target := golikeit.EntityTarget{EntityType: "post", EntityID: "benchmark-post"}

		// Pre-populate
		for i := 0; i < 1000; i++ {
			userID := fmt.Sprintf("user-%d", i)
			s.AddReaction(ctx, userID, target, "LIKE")
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			s.GetEntityCounts(ctx, target)
		}
	})
}

// BenchmarkScalability measures performance with increasing data size.
func BenchmarkScalability(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := setupStorage(b)
			defer s.Close()

			ctx := context.Background()

			// Pre-populate with size reactions
			for i := 0; i < size; i++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", i%100),
				}
				userID := fmt.Sprintf("user-%d", i)
				s.AddReaction(ctx, userID, target, "LIKE")
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				target := golikeit.EntityTarget{
					EntityType: "post",
					EntityID:   fmt.Sprintf("post-%d", i%100),
				}
				userID := fmt.Sprintf("user-%d", i%size)
				s.GetUserReaction(ctx, userID, target)
			}
		})
	}
}

// BenchmarkLatencyDistribution measures latency distribution across operations.
func BenchmarkLatencyDistribution(b *testing.B) {
	s := setupStorage(b)
	defer s.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 10000; i++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", i%1000),
		}
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			target := golikeit.EntityTarget{
				EntityType: "post",
				EntityID:   fmt.Sprintf("post-%d", i%1000),
			}
			userID := fmt.Sprintf("new-user-%d", i)
			s.AddReaction(ctx, userID, target, "LIKE")
		}
	})

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			target := golikeit.EntityTarget{
				EntityType: "post",
				EntityID:   fmt.Sprintf("post-%d", i%1000),
			}
			userID := fmt.Sprintf("user-%d", i%10000)
			s.GetUserReaction(ctx, userID, target)
		}
	})

	b.Run("Counts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			target := golikeit.EntityTarget{
				EntityType: "post",
				EntityID:   fmt.Sprintf("post-%d", i%1000),
			}
			s.GetEntityCounts(ctx, target)
		}
	})
}

// RunBenchmarks runs all benchmarks and returns results.
func RunBenchmarks() map[string]*Result {
	results := make(map[string]*Result)

	// Define benchmark functions
	benchmarks := []struct {
		name     string
		fn       func(context.Context) error
		isWrite  bool
		isRead   bool
	}{
		{
			name: "AddReaction",
			fn: func(ctx context.Context) error {
				s := storage.NewMemoryStorage()
				defer s.Close()
				target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}
				_, err := s.AddReaction(ctx, "test-user", target, "LIKE")
				return err
			},
			isWrite: true,
		},
		{
			name: "GetUserReaction",
			fn: func(ctx context.Context) error {
				s := storage.NewMemoryStorage()
				defer s.Close()
				target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}
				s.AddReaction(ctx, "test-user", target, "LIKE")
				_, err := s.GetUserReaction(ctx, "test-user", target)
				return err
			},
			isRead: true,
		},
		{
			name: "GetEntityCounts",
			fn: func(ctx context.Context) error {
				s := storage.NewMemoryStorage()
				defer s.Close()
				target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}
				s.AddReaction(ctx, "test-user", target, "LIKE")
				_, err := s.GetEntityCounts(ctx, target)
				return err
			},
			isRead: true,
		},
	}

	runner := NewRunner()
	targets := DefaultTargets()

	for _, bm := range benchmarks {
		// Run with different concurrency levels
		for _, concurrency := range []int{1, 10, 50, 100} {
			name := fmt.Sprintf("%s-Concurrency%d", bm.name, concurrency)
			result := runner.Run(name, concurrency, 5*time.Second, bm.fn)
			results[name] = result

			// Validate against targets
			if bm.isWrite {
				if !result.ValidateLatency(targets.WriteP95Latency) {
					fmt.Printf("WARNING: %s p95 latency %v exceeds target %v\n",
						name, result.Latencies.P95, targets.WriteP95Latency)
				}
				if !result.ValidateThroughput(targets.MinWriteThroughput) {
					fmt.Printf("WARNING: %s throughput %.2f below target %.2f\n",
						name, result.Throughput.OpsPerSecond, targets.MinWriteThroughput)
				}
			}
			if bm.isRead {
				if !result.ValidateLatency(targets.ReadP95Latency) {
					fmt.Printf("WARNING: %s p95 latency %v exceeds target %v\n",
						name, result.Latencies.P95, targets.ReadP95Latency)
				}
				if !result.ValidateThroughput(targets.MinReadThroughput) {
					fmt.Printf("WARNING: %s throughput %.2f below target %.2f\n",
						name, result.Throughput.OpsPerSecond, targets.MinReadThroughput)
				}
			}
		}
	}

	return results
}
