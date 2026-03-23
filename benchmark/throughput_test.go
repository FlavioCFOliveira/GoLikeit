// Package benchmark provides throughput performance tests.
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

// ThroughputTarget defines expected throughput targets.
type ThroughputTarget struct {
	Name        string
	MinOpsPerSec float64
	Duration    time.Duration
}

// ThroughputTargets defines expected performance targets.
var ThroughputTargets = []ThroughputTarget{
	{
		Name:         "WriteThroughput",
		MinOpsPerSec: 1000,
		Duration:     5 * time.Second,
	},
	{
		Name:         "ReadThroughput",
		MinOpsPerSec: 5000,
		Duration:     5 * time.Second,
	},
}

// TestWriteThroughput validates write throughput meets targets.
func TestWriteThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	target := ThroughputTargets[0]

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency%d", concurrency), func(t *testing.T) {
			result := measureWriteThroughput(ctx, s, concurrency, target.Duration)

			t.Logf("Write throughput with %d goroutines: %.2f ops/sec (total: %d ops)",
				concurrency, result.OpsPerSec, result.TotalOps)

			if result.OpsPerSec < target.MinOpsPerSec {
				t.Errorf("Throughput %.2f ops/sec below target %.2f ops/sec",
					result.OpsPerSec, target.MinOpsPerSec)
			}
		})
	}
}

// TestReadThroughput validates read throughput meets targets.
func TestReadThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", i%100),
		}
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	target := ThroughputTargets[1]

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency%d", concurrency), func(t *testing.T) {
			result := measureReadThroughput(ctx, s, concurrency, target.Duration)

			t.Logf("Read throughput with %d goroutines: %.2f ops/sec (total: %d ops)",
				concurrency, result.OpsPerSec, result.TotalOps)

			if result.OpsPerSec < target.MinOpsPerSec {
				t.Errorf("Throughput %.2f ops/sec below target %.2f ops/sec",
					result.OpsPerSec, target.MinOpsPerSec)
			}
		})
	}
}

// TestMixedThroughput validates throughput with mixed read/write operations.
func TestMixedThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		target := golikeit.EntityTarget{
			EntityType: "post",
			EntityID:   fmt.Sprintf("post-%d", i%100),
		}
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
	}

	readRatio := 0.7 // 70% reads, 30% writes
	duration := 5 * time.Second

	result := measureMixedThroughput(ctx, s, 50, duration, readRatio)

	t.Logf("Mixed throughput (%.0f%% reads): %.2f ops/sec (total: %d ops)",
		readRatio*100, result.OpsPerSec, result.TotalOps)

	// Mixed throughput target (conservative)
	minThroughput := 2000.0
	if result.OpsPerSec < minThroughput {
		t.Errorf("Mixed throughput %.2f ops/sec below target %.2f ops/sec",
			result.OpsPerSec, minThroughput)
	}
}

// TestSustainedThroughput validates throughput over extended period.
func TestSustainedThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained throughput tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	duration := 30 * time.Second
	concurrency := 50

	result := measureWriteThroughput(ctx, s, concurrency, duration)

	t.Logf("Sustained throughput over %v: %.2f ops/sec (total: %d ops)",
		duration, result.OpsPerSec, result.TotalOps)

	// Sustained throughput should be close to short-term
	minThroughput := 800.0 // Slightly lower for sustained
	if result.OpsPerSec < minThroughput {
		t.Errorf("Sustained throughput %.2f ops/sec below target %.2f ops/sec",
			result.OpsPerSec, minThroughput)
	}
}

// TestThroughputScalability checks if throughput scales with concurrency.
func TestThroughputScalability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	duration := 5 * time.Second
	concurrencyLevels := []int{1, 10, 25, 50, 100}

	var results []ThroughputResult
	for _, concurrency := range concurrencyLevels {
		result := measureWriteThroughput(ctx, s, concurrency, duration)
		results = append(results, result)

		t.Logf("Concurrency %d: %.2f ops/sec", concurrency, result.OpsPerSec)
	}

	// Check scalability - throughput should generally increase with concurrency
	// (though not linearly due to lock contention)
	if len(results) >= 2 {
		first := results[0].OpsPerSec
		last := results[len(results)-1].OpsPerSec

		// Throughput with 100 goroutines should be at least 5x single goroutine
		if last < first*5 {
			t.Logf("Limited scalability: 1 goroutine = %.2f ops/sec, 100 goroutines = %.2f ops/sec",
				first, last)
		}
	}
}

// TestPeakThroughput finds maximum achievable throughput.
func TestPeakThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping peak throughput tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	duration := 3 * time.Second

	// Test with high concurrency
	concurrency := 200
	result := measureWriteThroughput(ctx, s, concurrency, duration)

	t.Logf("Peak throughput with %d goroutines: %.2f ops/sec", concurrency, result.OpsPerSec)

	// Record peak for analysis
	if result.OpsPerSec < 500 {
		t.Errorf("Peak throughput %.2f ops/sec unacceptably low", result.OpsPerSec)
	}
}

// ThroughputResult holds throughput measurement results.
type ThroughputResult struct {
	OpsPerSec float64
	TotalOps  int64
	Duration  time.Duration
}

// measureWriteThroughput measures write throughput.
func measureWriteThroughput(ctx context.Context, s *storage.MemoryStorage, concurrency int, duration time.Duration) ThroughputResult {
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "throughput-test"}

	var totalOps int64
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-startBarrier

			for {
				select {
				case <-ctx.Done():
					return
				default:
					userID := fmt.Sprintf("user-%d-%d", id, atomic.LoadInt64(&totalOps))
					s.AddReaction(ctx, userID, target, "LIKE")
					atomic.AddInt64(&totalOps, 1)
				}
			}
		}(i)
	}

	start := time.Now()
	close(startBarrier)

	<-ctx.Done()
	wg.Wait()

	elapsed := time.Since(start)
	opsPerSec := float64(totalOps) / elapsed.Seconds()

	return ThroughputResult{
		OpsPerSec: opsPerSec,
		TotalOps:  totalOps,
		Duration:  elapsed,
	}
}

// measureReadThroughput measures read throughput.
func measureReadThroughput(ctx context.Context, s *storage.MemoryStorage, concurrency int, duration time.Duration) ThroughputResult {
	var totalOps int64
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-startBarrier

			for {
				select {
				case <-ctx.Done():
					return
				default:
					target := golikeit.EntityTarget{
						EntityType: "post",
						EntityID:   fmt.Sprintf("post-%d", id%100),
					}
					userID := fmt.Sprintf("user-%d", id%1000)
					s.GetUserReaction(ctx, userID, target)
					atomic.AddInt64(&totalOps, 1)
				}
			}
		}(i)
	}

	start := time.Now()
	close(startBarrier)

	<-ctx.Done()
	wg.Wait()

	elapsed := time.Since(start)
	opsPerSec := float64(totalOps) / elapsed.Seconds()

	return ThroughputResult{
		OpsPerSec: opsPerSec,
		TotalOps:  totalOps,
		Duration:  elapsed,
	}
}

// measureMixedThroughput measures throughput with mixed operations.
func measureMixedThroughput(ctx context.Context, s *storage.MemoryStorage, concurrency int, duration time.Duration, readRatio float64) ThroughputResult {
	var totalOps int64
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-startBarrier

			opCount := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					target := golikeit.EntityTarget{
						EntityType: "post",
						EntityID:   fmt.Sprintf("post-%d", opCount%100),
					}
					userID := fmt.Sprintf("user-%d-%d", id, opCount)

					if float64(opCount%100)/100.0 < readRatio {
						// Read operation
						s.GetUserReaction(ctx, userID, target)
					} else {
						// Write operation
						s.AddReaction(ctx, userID, target, "LIKE")
					}
					atomic.AddInt64(&totalOps, 1)
					opCount++
				}
			}
		}(i)
	}

	start := time.Now()
	close(startBarrier)

	<-ctx.Done()
	wg.Wait()

	elapsed := time.Since(start)
	opsPerSec := float64(totalOps) / elapsed.Seconds()

	return ThroughputResult{
		OpsPerSec: opsPerSec,
		TotalOps:  totalOps,
		Duration:  elapsed,
	}
}
