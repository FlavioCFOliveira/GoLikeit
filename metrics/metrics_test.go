package metrics

import (
	"sync"
	"testing"
)

func TestNoopCounter(t *testing.T) {
	counter := &NoopCounter{}

	// These should not panic
	counter.Inc()
	counter.IncBy(5)

	// Value always returns 0
	if v := counter.Value(); v != 0 {
		t.Errorf("expected Value() = 0, got %d", v)
	}
}

func TestNoopHistogram(t *testing.T) {
	histogram := &NoopHistogram{}

	// These should not panic
	histogram.Record(100.0)
	histogram.Observe([]float64{1.0, 2.0, 3.0})

	// Values always return 0
	if c := histogram.Count(); c != 0 {
		t.Errorf("expected Count() = 0, got %d", c)
	}
	if s := histogram.Sum(); s != 0 {
		t.Errorf("expected Sum() = 0, got %f", s)
	}
}

func TestNoopMetrics(t *testing.T) {
	metrics := &NoopMetrics{}

	// Counter should return valid Counter
	counter := metrics.Counter("test_counter", Labels{"key": "value"})
	if counter == nil {
		t.Error("expected non-nil Counter")
	}
	counter.Inc() // Should not panic

	// Histogram should return valid Histogram
	histogram := metrics.Histogram("test_histogram", []float64{1, 10, 100}, Labels{"key": "value"})
	if histogram == nil {
		t.Error("expected non-nil Histogram")
	}
	histogram.Record(50.0) // Should not panic

	// Close should not panic
	if err := metrics.Close(); err != nil {
		t.Errorf("expected Close() = nil, got %v", err)
	}
}

func TestDefaultMetrics(t *testing.T) {
	metrics := DefaultMetrics()
	if metrics == nil {
		t.Error("expected non-nil MetricsCollector from DefaultMetrics()")
	}

	// Should return noop implementations
	counter := metrics.Counter("test", nil)
	counter.Inc()
	if counter.Value() != 0 {
		t.Error("expected noop counter value to be 0")
	}

	histogram := metrics.Histogram("test", nil, nil)
	histogram.Record(100.0)
	if histogram.Count() != 0 {
		t.Error("expected noop histogram count to be 0")
	}
}

func TestAtomicCounter(t *testing.T) {
	counter := NewAtomicCounter()

	// Test Inc
	counter.Inc()
	if v := counter.Value(); v != 1 {
		t.Errorf("expected Value() = 1, got %d", v)
	}

	// Test IncBy
	counter.IncBy(5)
	if v := counter.Value(); v != 6 {
		t.Errorf("expected Value() = 6, got %d", v)
	}

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Inc()
		}()
	}
	wg.Wait()

	if v := counter.Value(); v != 106 {
		t.Errorf("expected Value() = 106, got %d", v)
	}
}

func TestAtomicHistogram(t *testing.T) {
	histogram := NewAtomicHistogram()

	// Test Record
	histogram.Record(100.0)
	if c := histogram.Count(); c != 1 {
		t.Errorf("expected Count() = 1, got %d", c)
	}

	// Test Observe
	histogram.Observe([]float64{50.0, 75.0, 25.0})
	if c := histogram.Count(); c != 4 {
		t.Errorf("expected Count() = 4, got %d", c)
	}

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			histogram.Record(val)
		}(float64(i))
	}
	wg.Wait()

	if c := histogram.Count(); c != 104 {
		t.Errorf("expected Count() = 104, got %d", c)
	}

	// Test Values
	values := histogram.Values()
	if len(values) < 4 {
		t.Errorf("expected at least 4 values, got %d", len(values))
	}
}

func TestAtomicHistogramSum(t *testing.T) {
	histogram := NewAtomicHistogram()

	histogram.Record(1.5)
	histogram.Record(2.7)
	histogram.Record(0.001)

	got := histogram.Sum()
	want := 1.5 + 2.7 + 0.001
	// Allow small floating-point tolerance.
	const epsilon = 1e-9
	diff := got - want
	if diff < -epsilon || diff > epsilon {
		t.Errorf("Sum() = %v, want %v (diff %v)", got, want, diff)
	}

	if histogram.Count() != 3 {
		t.Errorf("Count() = %d, want 3", histogram.Count())
	}
}

func TestAtomicHistogramSum_IntegerValues(t *testing.T) {
	histogram := NewAtomicHistogram()

	histogram.Record(10.0)
	histogram.Record(20.0)
	histogram.Record(30.0)

	got := histogram.Sum()
	want := 60.0
	if got != want {
		t.Errorf("Sum() = %v, want %v", got, want)
	}
}

func BenchmarkNoopCounter_Inc(b *testing.B) {
	counter := &NoopCounter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Inc()
	}
}

func BenchmarkAtomicCounter_Inc(b *testing.B) {
	counter := NewAtomicCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Inc()
	}
}

func BenchmarkNoopHistogram_Record(b *testing.B) {
	histogram := &NoopHistogram{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		histogram.Record(float64(i))
	}
}

func BenchmarkAtomicHistogram_Record(b *testing.B) {
	histogram := NewAtomicHistogram()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		histogram.Record(float64(i))
	}
}

func TestMetricConstants(t *testing.T) {
	// Verify constants are defined
	if OperationLatency != "operation_latency_ms" {
		t.Error("OperationLatency constant incorrect")
	}
	if OperationErrors != "operation_errors_total" {
		t.Error("OperationErrors constant incorrect")
	}
	if OperationThroughput != "operation_throughput" {
		t.Error("OperationThroughput constant incorrect")
	}
	if CacheHits != "cache_hits_total" {
		t.Error("CacheHits constant incorrect")
	}
	if CacheMisses != "cache_misses_total" {
		t.Error("CacheMisses constant incorrect")
	}
	if ConnectionPoolActive != "connection_pool_active" {
		t.Error("ConnectionPoolActive constant incorrect")
	}
	if ConnectionPoolIdle != "connection_pool_idle" {
		t.Error("ConnectionPoolIdle constant incorrect")
	}
	if EventEmitLatency != "event_emit_latency_ms" {
		t.Error("EventEmitLatency constant incorrect")
	}
}
