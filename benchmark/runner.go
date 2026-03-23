// Package benchmark provides performance testing utilities and custom benchmark runners.
// It measures latencies, throughput, and validates performance targets.
package benchmark

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Metrics holds benchmark results for a single operation type.
type Metrics struct {
	Name           string
	Iterations     int64
	TotalDuration  time.Duration
	Latencies      []time.Duration
	Errors         int64
	ConcurrentOps  int
}

// LatencyStats holds calculated latency statistics.
type LatencyStats struct {
	Min       time.Duration
	Max       time.Duration
	Mean      time.Duration
	P50       time.Duration
	P95       time.Duration
	P99       time.Duration
	StdDev    time.Duration
}

// ThroughputStats holds calculated throughput statistics.
type ThroughputStats struct {
	OpsPerSecond float64
	TotalOps     int64
	Duration     time.Duration
}

// Result holds comprehensive benchmark results.
type Result struct {
	Name      string
	Latencies LatencyStats
	Throughput ThroughputStats
	Errors    int64
}

// Runner executes benchmarks and collects metrics.
type Runner struct {
	mu      sync.Mutex
	metrics map[string]*Metrics
}

// NewRunner creates a new benchmark runner.
func NewRunner() *Runner {
	return &Runner{
		metrics: make(map[string]*Metrics),
	}
}

// Run executes a benchmark function and collects metrics.
func (r *Runner) Run(name string, concurrentOps int, duration time.Duration, fn func(context.Context) error) *Result {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	m := &Metrics{
		Name:          name,
		Latencies:     make([]time.Duration, 0, 10000),
		ConcurrentOps: concurrentOps,
	}

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// Start concurrent workers
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Wait for all workers to be ready
			<-startBarrier

			for {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()
					err := fn(ctx)
					elapsed := time.Since(start)

					atomic.AddInt64(&m.Iterations, 1)

					if err != nil {
						atomic.AddInt64(&m.Errors, 1)
					} else {
						r.mu.Lock()
						m.Latencies = append(m.Latencies, elapsed)
						r.mu.Unlock()
					}
				}
			}
		}(i)
	}

	// Start all workers simultaneously
	start := time.Now()
	close(startBarrier)

	// Wait for context to complete
	<-ctx.Done()
	wg.Wait()

	m.TotalDuration = time.Since(start)

	r.mu.Lock()
	r.metrics[name] = m
	r.mu.Unlock()

	return r.calculateResult(m)
}

// calculateResult computes statistics from collected metrics.
func (r *Runner) calculateResult(m *Metrics) *Result {
	if len(m.Latencies) == 0 {
		return &Result{Name: m.Name}
	}

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(m.Latencies))
	copy(sorted, m.Latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate statistics
	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}

	mean := sum / time.Duration(len(sorted))

	// Calculate standard deviation
	var variance float64
	for _, d := range sorted {
		diff := float64(d - mean)
		variance += diff * diff
	}
	stdDev := time.Duration(math.Sqrt(variance / float64(len(sorted))))

	// Calculate percentiles
	p50Index := int(float64(len(sorted)) * 0.50)
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	// Bounds check
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	opsPerSecond := float64(m.Iterations) / m.TotalDuration.Seconds()

	return &Result{
		Name: m.Name,
		Latencies: LatencyStats{
			Min:    sorted[0],
			Max:    sorted[len(sorted)-1],
			Mean:   mean,
			P50:    sorted[p50Index],
			P95:    sorted[p95Index],
			P99:    sorted[p99Index],
			StdDev: stdDev,
		},
		Throughput: ThroughputStats{
			OpsPerSecond: opsPerSecond,
			TotalOps:     m.Iterations,
			Duration:     m.TotalDuration,
		},
		Errors: m.Errors,
	}
}

// GetResult retrieves a benchmark result by name.
func (r *Runner) GetResult(name string) *Result {
	r.mu.Lock()
	defer r.mu.Unlock()

	if m, ok := r.metrics[name]; ok {
		return r.calculateResult(m)
	}
	return nil
}

// PrintResults outputs benchmark results in a formatted table.
func (r *Runner) PrintResults() {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("BENCHMARK RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	for _, m := range r.metrics {
		result := r.calculateResult(m)
		fmt.Printf("\n%s\n", result.Name)
		fmt.Println(strings.Repeat("-", len(result.Name)))
		fmt.Printf("  Concurrent Operations: %d\n", m.ConcurrentOps)
		fmt.Printf("  Total Operations:      %d\n", result.Throughput.TotalOps)
		fmt.Printf("  Total Duration:        %v\n", result.Throughput.Duration)
		fmt.Printf("  Errors:                %d\n", result.Errors)
		fmt.Println()
		fmt.Println("  Latency Statistics:")
		fmt.Printf("    Min:  %12v\n", result.Latencies.Min)
		fmt.Printf("    Mean: %12v\n", result.Latencies.Mean)
		fmt.Printf("    P50:  %12v\n", result.Latencies.P50)
		fmt.Printf("    P95:  %12v\n", result.Latencies.P95)
		fmt.Printf("    P99:  %12v\n", result.Latencies.P99)
		fmt.Printf("    Max:  %12v\n", result.Latencies.Max)
		fmt.Printf("    StdDev: %10v\n", result.Latencies.StdDev)
		fmt.Println()
		fmt.Println("  Throughput:")
		fmt.Printf("    %.2f ops/sec\n", result.Throughput.OpsPerSecond)
	}

	fmt.Println(strings.Repeat("=", 80))
}

// ValidateLatency checks if p95 latency meets the target.
func (r *Result) ValidateLatency(maxP95 time.Duration) bool {
	return r.Latencies.P95 <= maxP95
}

// ValidateThroughput checks if throughput meets the target.
func (r *Result) ValidateThroughput(minOpsPerSecond float64) bool {
	return r.Throughput.OpsPerSecond >= minOpsPerSecond
}

// Report generates a detailed benchmark report.
func (r *Result) Report() string {
	return fmt.Sprintf(
		"%s:\n"+
		"  Latency: min=%v, mean=%v, p50=%v, p95=%v, p99=%v, max=%v\n"+
		"  Throughput: %.2f ops/sec (%d total ops)\n"+
		"  Errors: %d",
		r.Name,
		r.Latencies.Min, r.Latencies.Mean, r.Latencies.P50,
		r.Latencies.P95, r.Latencies.P99, r.Latencies.Max,
		r.Throughput.OpsPerSecond, r.Throughput.TotalOps,
		r.Errors,
	)
}


// FormatDuration formats a duration for display.
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/1000)
	}
	return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
}

// TargetPerformance defines expected performance targets.
type TargetPerformance struct {
	WriteP95Latency    time.Duration
	ReadP95Latency     time.Duration
	MinWriteThroughput float64
	MinReadThroughput  float64
	MinConcurrentOps   int
}

// DefaultTargets returns default performance targets.
func DefaultTargets() TargetPerformance {
	return TargetPerformance{
		WriteP95Latency:    50 * time.Millisecond,
		ReadP95Latency:     10 * time.Millisecond,
		MinWriteThroughput: 1000,
		MinReadThroughput:  5000,
		MinConcurrentOps:   100,
	}
}

// Validate checks if results meet all targets.
func (t TargetPerformance) Validate(results []*Result) []string {
	var failures []string

	for _, r := range results {
		isWrite := r.Name == "AddReaction" || r.Name == "RemoveReaction"
		isRead := r.Name == "GetUserReaction" || r.Name == "GetEntityCounts"

		if isWrite {
			if !r.ValidateLatency(t.WriteP95Latency) {
				failures = append(failures, fmt.Sprintf(
					"%s: p95 latency %v exceeds target %v",
					r.Name, r.Latencies.P95, t.WriteP95Latency,
				))
			}
			if !r.ValidateThroughput(t.MinWriteThroughput) {
				failures = append(failures, fmt.Sprintf(
					"%s: throughput %.2f ops/sec below target %.2f",
					r.Name, r.Throughput.OpsPerSecond, t.MinWriteThroughput,
				))
			}
		}

		if isRead {
			if !r.ValidateLatency(t.ReadP95Latency) {
				failures = append(failures, fmt.Sprintf(
					"%s: p95 latency %v exceeds target %v",
					r.Name, r.Latencies.P95, t.ReadP95Latency,
				))
			}
			if !r.ValidateThroughput(t.MinReadThroughput) {
				failures = append(failures, fmt.Sprintf(
					"%s: throughput %.2f ops/sec below target %.2f",
					r.Name, r.Throughput.OpsPerSecond, t.MinReadThroughput,
				))
			}
		}
	}

	return failures
}

// BenchmarkFunc is a function type for standard Go benchmarks.
type BenchmarkFunc func(b *testing.B)

// RunStandardBenchmark runs a standard Go benchmark with memory profiling.
func RunStandardBenchmark(b *testing.B, fn func()) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		fn()
	}
}

// ParallelBenchmark runs a benchmark with parallel workers.
func ParallelBenchmark(b *testing.B, fn func(*testing.PB)) {
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(fn)
}
