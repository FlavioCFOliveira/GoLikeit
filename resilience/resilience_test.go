package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryPolicy_Default(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", policy.MaxAttempts)
	}
	if policy.InitialBackoff != 100*time.Millisecond {
		t.Errorf("expected InitialBackoff=100ms, got %v", policy.InitialBackoff)
	}
	if policy.MaxBackoff != 10*time.Second {
		t.Errorf("expected MaxBackoff=10s, got %v", policy.MaxBackoff)
	}
	if policy.Jitter != 0.1 {
		t.Errorf("expected Jitter=0.1, got %f", policy.Jitter)
	}
}

func TestRetryPolicy_ExponentialBackoffWithJitter(t *testing.T) {
	policy := RetryPolicy{
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Jitter:         0.0, // Disable jitter for predictable tests
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{10, 10 * time.Second}, // Should cap at MaxBackoff
	}

	for _, tt := range tests {
		backoff := policy.ExponentialBackoffWithJitter(tt.attempt)
		// Allow small tolerance for floating point math
		if backoff < tt.expected-time.Millisecond || backoff > tt.expected+time.Millisecond {
			t.Errorf("attempt %d: expected backoff ~%v, got %v", tt.attempt, tt.expected, backoff)
		}
	}
}

func TestRetryPolicy_ExponentialBackoffWithJitter_Caps(t *testing.T) {
	policy := RetryPolicy{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     5 * time.Second,
		Jitter:         0.0,
	}

	// After several attempts, should cap at MaxBackoff
	backoff := policy.ExponentialBackoffWithJitter(10)
	if backoff > policy.MaxBackoff {
		t.Errorf("backoff should not exceed MaxBackoff, got %v", backoff)
	}
}

func TestRetryPolicy_Retry_SuccessFirstAttempt(t *testing.T) {
	policy := DefaultRetryPolicy()

	callCount := 0
	err := policy.Retry(func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryPolicy_Retry_EventualSuccess(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}

	callCount := 0
	err := policy.Retry(func() error {
		callCount++
		if callCount < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestRetryPolicy_Retry_Exhaustion(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}

	callCount := 0
	expectedErr := errors.New("persistent error")
	err := policy.Retry(func() error {
		callCount++
		return expectedErr
	})

	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("expected ErrMaxRetriesExceeded, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryPolicy_RetryWithContext_Cancellation(t *testing.T) {
	policy := DefaultRetryPolicy()
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	errCh := make(chan error, 1)

	go func() {
		errCh <- policy.RetryWithContext(ctx, func(ctx context.Context) error {
			callCount++
			if callCount >= 2 {
				cancel() // Cancel context during second attempt
			}
			return errors.New("error")
		})
	}()

	// Wait for the goroutine to finish
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for retry")
	}
}

func TestRetryPolicy_Retry_NonRetryable(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
		ShouldRetry: func(err error) bool {
			// Don't retry non-retryable errors
			return !IsNonRetryableError(err)
		},
	}

	callCount := 0
	nonRetryableErr := MarkNonRetryable(errors.New("fatal error"))

	err := policy.Retry(func() error {
		callCount++
		return nonRetryableErr
	})

	if callCount != 1 {
		t.Errorf("expected 1 call (no retries), got %d", callCount)
	}
	if err == nil || !errors.Is(err, nonRetryableErr) {
		t.Errorf("expected nonRetryableErr, got %v", err)
	}
}

func TestCircuitBreakerConfig_Default(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	if config.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold=5, got %d", config.FailureThreshold)
	}
	if config.RecoveryTimeout != 30*time.Second {
		t.Errorf("expected RecoveryTimeout=30s, got %v", config.RecoveryTimeout)
	}
	if config.SuccessThreshold != 3 {
		t.Errorf("expected SuccessThreshold=3, got %d", config.SuccessThreshold)
	}
}

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	if cb.State() != StateClosed {
		t.Errorf("expected initial state Closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_Success(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state Closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  1 * time.Minute,
	}
	cb := NewCircuitBreaker(config)

	// Trigger failures to reach threshold
	for i := 0; i < config.FailureThreshold; i++ {
		err := cb.Execute(func() error {
			return errors.New("failure")
		})
		if err == nil {
			t.Errorf("expected error on attempt %d", i+1)
		}
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state Open after threshold, got %v", cb.State())
	}

	// Next request should be rejected
	err := cb.Execute(func() error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Execute(func() error { return errors.New("failure") })
	if cb.State() != StateOpen {
		t.Fatal("expected circuit to be open")
	}

	// Wait for recovery timeout
	time.Sleep(100 * time.Millisecond)

	// Next request should transition to half-open and succeed
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error after recovery, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state Closed after successful recovery, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpen_Failure(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Execute(func() error { return errors.New("failure") })

	// Wait for recovery timeout
	time.Sleep(100 * time.Millisecond)

	// Fail in half-open state
	err := cb.Execute(func() error {
		return errors.New("still failing")
	})

	if err == nil {
		t.Error("expected error")
	}
	if cb.State() != StateOpen {
		t.Errorf("expected state Open after half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreaker_MultipleSuccessesToClose(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		RecoveryTimeout:  1 * time.Millisecond,
		SuccessThreshold: 3,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Execute(func() error { return errors.New("failure") })

	// Wait for recovery
	time.Sleep(10 * time.Millisecond)

	// Need multiple successes to close
	for i := 0; i < config.SuccessThreshold; i++ {
		err := cb.Execute(func() error { return nil })
		if err != nil {
			t.Errorf("unexpected error on success %d: %v", i+1, err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state Closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		RecoveryTimeout:  1 * time.Minute,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Execute(func() error { return errors.New("failure") })
	if cb.State() != StateOpen {
		t.Fatal("expected circuit to be open")
	}

	// Reset manually
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected state Closed after reset, got %v", cb.State())
	}
	if cb.Failures() != 0 {
		t.Errorf("expected 0 failures after reset, got %d", cb.Failures())
	}
}

func TestCircuitBreakerState_String(t *testing.T) {
	tests := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitBreakerState(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("state %d: expected %q, got %q", tt.state, tt.expected, got)
		}
	}
}

func TestMarkRetryable(t *testing.T) {
	originalErr := errors.New("some error")
	markedErr := MarkRetryable(originalErr)

	if !IsRetryableError(markedErr) {
		t.Error("expected marked error to be retryable")
	}
	if IsNonRetryableError(markedErr) {
		t.Error("expected marked error not to be non-retryable")
	}
	if !errors.Is(markedErr, originalErr) {
		t.Error("expected marked error to wrap original")
	}
}

func TestMarkNonRetryable(t *testing.T) {
	originalErr := errors.New("some error")
	markedErr := MarkNonRetryable(originalErr)

	if !IsNonRetryableError(markedErr) {
		t.Error("expected marked error to be non-retryable")
	}
	if IsRetryableError(markedErr) {
		t.Error("expected marked error not to be retryable")
	}
	if !errors.Is(markedErr, originalErr) {
		t.Error("expected marked error to wrap original")
	}
}

func TestMarkNil(t *testing.T) {
	if MarkRetryable(nil) != nil {
		t.Error("MarkRetryable(nil) should return nil")
	}
	if MarkNonRetryable(nil) != nil {
		t.Error("MarkNonRetryable(nil) should return nil")
	}
}

func BenchmarkRetryPolicy_Retry(b *testing.B) {
	policy := RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy.Retry(func() error {
			return nil
		})
	}
}

func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(func() error {
			return nil
		})
	}
}
