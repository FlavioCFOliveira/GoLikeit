// Package metrics provides instrumentation interfaces for operational metrics.
package metrics

import (
	"math"
	"sync"
	"sync/atomic"
)

// Standard metric names for operational metrics.
const (
	// OperationLatency tracks operation latency in milliseconds.
	OperationLatency = "operation_latency_ms"

	// OperationErrors tracks total number of operation errors.
	OperationErrors = "operation_errors_total"

	// OperationThroughput tracks operation throughput (ops/sec).
	OperationThroughput = "operation_throughput"

	// CacheHits tracks total cache hits.
	CacheHits = "cache_hits_total"

	// CacheMisses tracks total cache misses.
	CacheMisses = "cache_misses_total"

	// ConnectionPoolActive tracks active connections in pool.
	ConnectionPoolActive = "connection_pool_active"

	// ConnectionPoolIdle tracks idle connections in pool.
	ConnectionPoolIdle = "connection_pool_idle"

	// EventEmitLatency tracks event emission latency in milliseconds.
	EventEmitLatency = "event_emit_latency_ms"
)

// Labels represents metric labels as key-value pairs.
type Labels map[string]string

// Counter is a cumulative metric that represents a single monotonically
// increasing counter whose value can only increase or be reset to zero.
type Counter interface {
	// Inc increments the counter by 1.
	Inc()

	// IncBy increments the counter by the given delta.
	IncBy(delta int64)

	// Value returns the current counter value.
	Value() int64
}

// Histogram samples observations (usually things like request durations
// or response sizes) and counts them in configurable buckets.
type Histogram interface {
	// Record records a single observation with the given value.
	Record(value float64)

	// Count returns the number of recorded observations.
	Count() int64

	// Sum returns the sum of all recorded values.
	Sum() float64

	// Observe records multiple observations at once.
	Observe(values []float64)
}

// MetricsCollector is the main interface for creating and managing metrics.
type MetricsCollector interface {
	// Counter creates or retrieves a counter metric with the given name and labels.
	Counter(name string, labels Labels) Counter

	// Histogram creates or retrieves a histogram metric with the given name,
	// bucket boundaries, and labels.
	Histogram(name string, buckets []float64, labels Labels) Histogram

	// Close releases resources held by the metrics collector.
	Close() error
}

// NoopCounter is a Counter implementation that does nothing.
type NoopCounter struct{}

// Inc implements Counter.
func (n *NoopCounter) Inc() {}

// IncBy implements Counter.
func (n *NoopCounter) IncBy(delta int64) {}

// Value implements Counter.
func (n *NoopCounter) Value() int64 { return 0 }

// NoopHistogram is a Histogram implementation that does nothing.
type NoopHistogram struct{}

// Record implements Histogram.
func (n *NoopHistogram) Record(value float64) {}

// Count implements Histogram.
func (n *NoopHistogram) Count() int64 { return 0 }

// Sum implements Histogram.
func (n *NoopHistogram) Sum() float64 { return 0 }

// Observe implements Histogram.
func (n *NoopHistogram) Observe(values []float64) {}

// NoopMetrics is a MetricsCollector implementation that does nothing.
type NoopMetrics struct{}

// Counter implements MetricsCollector.
func (n *NoopMetrics) Counter(name string, labels Labels) Counter {
	return &NoopCounter{}
}

// Histogram implements MetricsCollector.
func (n *NoopMetrics) Histogram(name string, buckets []float64, labels Labels) Histogram {
	return &NoopHistogram{}
}

// Close implements MetricsCollector.
func (n *NoopMetrics) Close() error { return nil }

// DefaultMetrics returns a MetricsCollector that discards all metrics.
// This is the default when no metrics collection is configured.
func DefaultMetrics() MetricsCollector {
	return &NoopMetrics{}
}

// AtomicCounter is a thread-safe Counter implementation using atomic operations.
type AtomicCounter struct {
	value atomic.Int64
}

// NewAtomicCounter creates a new AtomicCounter with initial value 0.
func NewAtomicCounter() *AtomicCounter {
	return &AtomicCounter{}
}

// Inc implements Counter.
func (c *AtomicCounter) Inc() {
	c.value.Add(1)
}

// IncBy implements Counter.
func (c *AtomicCounter) IncBy(delta int64) {
	c.value.Add(delta)
}

// Value implements Counter.
func (c *AtomicCounter) Value() int64 {
	return c.value.Load()
}

// AtomicHistogram is a thread-safe Histogram implementation using atomic operations.
type AtomicHistogram struct {
	count atomic.Int64
	sum   atomic.Uint64 // Store as uint64 bits of float64 for atomic operations
	mu    sync.RWMutex
	// values stores individual observations for percentile calculation
	values []float64
}

// NewAtomicHistogram creates a new AtomicHistogram.
func NewAtomicHistogram() *AtomicHistogram {
	return &AtomicHistogram{
		values: make([]float64, 0, 1024),
	}
}

// Record implements Histogram.
func (h *AtomicHistogram) Record(value float64) {
	h.count.Add(1)
	// Atomically add float64 value using CAS loop with IEEE 754 bit representation.
	for {
		old := h.sum.Load()
		newBits := math.Float64bits(math.Float64frombits(old) + value)
		if h.sum.CompareAndSwap(old, newBits) {
			break
		}
	}

	h.mu.Lock()
	h.values = append(h.values, value)
	h.mu.Unlock()
}

// Count implements Histogram.
func (h *AtomicHistogram) Count() int64 {
	return h.count.Load()
}

// Sum implements Histogram.
func (h *AtomicHistogram) Sum() float64 {
	return math.Float64frombits(h.sum.Load())
}

// Observe implements Histogram.
func (h *AtomicHistogram) Observe(values []float64) {
	for _, v := range values {
		h.Record(v)
	}
}

// Values returns a copy of observed values (for testing and debugging).
func (h *AtomicHistogram) Values() []float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]float64, len(h.values))
	copy(result, h.values)
	return result
}
