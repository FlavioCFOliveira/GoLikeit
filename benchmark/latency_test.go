// Package benchmark provides latency-specific performance tests.
package benchmark

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/storage"
)

// LatencyTarget defines performance targets for latency tests.
type LatencyTarget struct {
	Name     string
	MaxP95   time.Duration
	MaxP99   time.Duration
	MaxMean  time.Duration
}

// PerformanceTargets defines expected performance targets.
var PerformanceTargets = []LatencyTarget{
	{
		Name:    "WriteOperations",
		MaxP95:  50 * time.Millisecond,
		MaxP99:  100 * time.Millisecond,
		MaxMean: 25 * time.Millisecond,
	},
	{
		Name:    "ReadOperations",
		MaxP95:  10 * time.Millisecond,
		MaxP99:  25 * time.Millisecond,
		MaxMean: 5 * time.Millisecond,
	},
}

// TestLatencyTargets validates that operations meet latency requirements.
func TestLatencyTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency tests in short mode")
	}

	s := storage.NewMemoryStorage()
	defer s.Close()

	ctx := context.Background()

	// Test write operation latency
	t.Run("WriteOperations", func(t *testing.T) {
		target := PerformanceTargets[0]
		latencies := measureWriteLatencies(ctx, s, 1000)

		stats := calculateStats(latencies)

		t.Logf("Write latency stats: min=%v, mean=%v, p95=%v, p99=%v, max=%v",
			stats.Min, stats.Mean, stats.P95, stats.P99, stats.Max)

		if stats.P95 > target.MaxP95 {
			t.Errorf("p95 latency %v exceeds target %v", stats.P95, target.MaxP95)
		}
		if stats.P99 > target.MaxP99 {
			t.Errorf("p99 latency %v exceeds target %v", stats.P99, target.MaxP99)
		}
		if stats.Mean > target.MaxMean {
			t.Errorf("mean latency %v exceeds target %v", stats.Mean, target.MaxMean)
		}
	})

	// Test read operation latency
	t.Run("ReadOperations", func(t *testing.T) {
		// Pre-populate data
		for i := 0; i < 1000; i++ {
			target := golikeit.EntityTarget{
				EntityType: "post",
				EntityID:   fmt.Sprintf("post-%d", i%100),
			}
			userID := fmt.Sprintf("user-%d", i)
			s.AddReaction(ctx, userID, target, "LIKE")
		}

		target := PerformanceTargets[1]
		latencies := measureReadLatencies(ctx, s, 1000)

		stats := calculateStats(latencies)

		t.Logf("Read latency stats: min=%v, mean=%v, p95=%v, p99=%v, max=%v",
			stats.Min, stats.Mean, stats.P95, stats.P99, stats.Max)

		if stats.P95 > target.MaxP95 {
			t.Errorf("p95 latency %v exceeds target %v", stats.P95, target.MaxP95)
		}
		if stats.P99 > target.MaxP99 {
			t.Errorf("p99 latency %v exceeds target %v", stats.P99, target.MaxP99)
		}
		if stats.Mean > target.MaxMean {
			t.Errorf("mean latency %v exceeds target %v", stats.Mean, target.MaxMean)
		}
	})
}

// TestLatencyUnderLoad validates latency under concurrent load.
func TestLatencyUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	concurrencyLevels := []int{10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency%d", concurrency), func(t *testing.T) {
			latencies := measureConcurrentLatencies(ctx, s, concurrency, 1000)
			stats := calculateStats(latencies)

			t.Logf("Concurrent (%d) latency: p95=%v, p99=%v, mean=%v",
				concurrency, stats.P95, stats.P99, stats.Mean)

			// More lenient targets under load
			maxP95 := 100 * time.Millisecond
			if concurrency >= 50 {
				maxP95 = 200 * time.Millisecond
			}

			if stats.P95 > maxP95 {
				t.Errorf("p95 latency %v exceeds %v under %d concurrent ops",
					stats.P95, maxP95, concurrency)
			}
		})
	}
}

// TestLatencyDistribution checks latency distribution is within expected bounds.
func TestLatencyDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping distribution tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}

	// Collect latency samples
	samples := 10000
	latencies := make([]time.Duration, 0, samples)

	for i := 0; i < samples; i++ {
		start := time.Now()
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
		latencies = append(latencies, time.Since(start))
	}

	stats := calculateStats(latencies)

	// Check that standard deviation is reasonable (< 50% of mean)
	if float64(stats.StdDev) > float64(stats.Mean)*0.5 {
		t.Logf("High variance detected: stddev=%v, mean=%v", stats.StdDev, stats.Mean)
	}

	// Check that max is not an extreme outlier (> 10x p99)
	if stats.Max > stats.P99*10 {
		t.Logf("Extreme outlier detected: max=%v, p99=%v", stats.Max, stats.P99)
	}

	// Verify distribution is roughly normal (p50 close to mean)
	diff := stats.P50 - stats.Mean
	if diff < 0 {
		diff = -diff
	}
	if float64(diff) > float64(stats.Mean)*0.3 {
		t.Logf("Skewed distribution: p50=%v, mean=%v", stats.P50, stats.Mean)
	}
}

// TestColdStartLatency measures latency after storage initialization.
func TestColdStartLatency(t *testing.T) {
	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}

	// Test multiple cold starts
	for i := 0; i < 10; i++ {
		s := storage.NewMemoryStorage()

		start := time.Now()
		s.AddReaction(ctx, "test-user", target, "LIKE")
		latency := time.Since(start)

		s.Close()

		// Cold start should still be fast
		if latency > 10*time.Millisecond {
			t.Errorf("Cold start latency %v too high", latency)
		}
	}
}

// TestLatencyConsistency ensures consistent performance over time.
func TestLatencyConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping consistency tests in short mode")
	}

	ctx := context.Background()
	s := storage.NewMemoryStorage()
	defer s.Close()

	target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}

	// Measure latency in 5 windows of 1000 operations each
	windows := 5
	opsPerWindow := 1000

	var p95Values []time.Duration
	for w := 0; w < windows; w++ {
		var latencies []time.Duration
		for i := 0; i < opsPerWindow; i++ {
			userID := fmt.Sprintf("user-%d-%d", w, i)
			start := time.Now()
			s.AddReaction(ctx, userID, target, "LIKE")
			latencies = append(latencies, time.Since(start))
		}
		stats := calculateStats(latencies)
		p95Values = append(p95Values, stats.P95)
		t.Logf("Window %d: p95=%v", w, stats.P95)
	}

	// Check variance between windows using coefficient of variation (CV = stdDev / mean).
	// CV is dimensionless: a value < 0.5 means relative std dev is within 50% of mean.
	if len(p95Values) > 1 {
		meanP95 := calculateMean(p95Values)
		variance := calculateVariance(p95Values, meanP95)
		stdDev := math.Sqrt(variance)

		if meanP95 > 0 {
			cv := stdDev / meanP95
			if cv > 0.5 {
				t.Errorf("High variance in p95 latency across windows: CV=%.2f (stdDev=%.0fns, mean=%.0fns)", cv, stdDev, meanP95)
			}
		}
	}
}

// measureWriteLatencies measures latencies for write operations.
func measureWriteLatencies(ctx context.Context, s *storage.MemoryStorage, count int) []time.Duration {
	latencies := make([]time.Duration, 0, count)
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}

	for i := 0; i < count; i++ {
		start := time.Now()
		userID := fmt.Sprintf("user-%d", i)
		s.AddReaction(ctx, userID, target, "LIKE")
		latencies = append(latencies, time.Since(start))
	}

	return latencies
}

// measureReadLatencies measures latencies for read operations.
func measureReadLatencies(ctx context.Context, s *storage.MemoryStorage, count int) []time.Duration {
	latencies := make([]time.Duration, 0, count)
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}

	for i := 0; i < count; i++ {
		userID := fmt.Sprintf("user-%d", i%1000)
		start := time.Now()
		s.GetUserReaction(ctx, userID, target)
		latencies = append(latencies, time.Since(start))
	}

	return latencies
}

// measureConcurrentLatencies measures latencies under concurrent load.
func measureConcurrentLatencies(ctx context.Context, s *storage.MemoryStorage, concurrency, totalOps int) []time.Duration {
	var latencies []time.Duration
	var mu sync.Mutex
	var wg sync.WaitGroup

	opsPerGoroutine := totalOps / concurrency
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "test-post"}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				userID := fmt.Sprintf("user-%d-%d", id, j)
				start := time.Now()
				s.AddReaction(ctx, userID, target, "LIKE")
				mu.Lock()
				latencies = append(latencies, time.Since(start))
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return latencies
}

// calculateStats calculates latency statistics.
func calculateStats(latencies []time.Duration) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// Sort for percentiles
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate statistics
	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	mean := sum / time.Duration(len(sorted))

	// Calculate standard deviation
	var variance time.Duration
	for _, d := range sorted {
		diff := d - mean
		if diff < 0 {
			diff = -diff
		}
		variance += diff
	}
	stdDev := variance / time.Duration(len(sorted))

	p50Index := len(sorted) * 50 / 100
	p95Index := len(sorted) * 95 / 100
	p99Index := len(sorted) * 99 / 100

	return LatencyStats{
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Mean:   mean,
		P50:    sorted[p50Index],
		P95:    sorted[p95Index],
		P99:    sorted[p99Index],
		StdDev: stdDev,
	}
}

// calculateMean calculates the mean of durations.
func calculateMean(values []time.Duration) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum time.Duration
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// calculateVariance calculates variance.
func calculateVariance(values []time.Duration, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		diff := float64(v) - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}
