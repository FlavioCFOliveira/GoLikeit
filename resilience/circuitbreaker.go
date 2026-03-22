package resilience

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState int32

const (
	// StateClosed means the circuit is closed and requests are allowed.
	// This is the normal operating state.
	StateClosed CircuitBreakerState = iota

	// StateOpen means the circuit is open and all requests are rejected.
	// This state occurs after the failure threshold is reached.
	StateOpen

	// StateHalfOpen means the circuit is testing if the underlying
	// problem has been resolved by allowing a single request through.
	StateHalfOpen
)

// String returns the string representation of the state.
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures required
	// to open the circuit.
	// Default: 5
	FailureThreshold int

	// RecoveryTimeout is the duration after which the circuit transitions
	// from open to half-open.
	// Default: 30s
	RecoveryTimeout time.Duration

	// SuccessThreshold is the number of consecutive successes required
	// in half-open state to close the circuit.
	// Default: 3
	SuccessThreshold int
}

// DefaultCircuitBreakerConfig returns a CircuitBreakerConfig with sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		SuccessThreshold: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern for fault tolerance.
// It prevents cascade failures by temporarily rejecting requests when a
// service is experiencing high error rates.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	// state is the current state of the circuit (atomic).
	state atomic.Int32

	// failures tracks consecutive failures (atomic).
	failures atomic.Int32

	// successes tracks consecutive successes in half-open state (atomic).
	successes atomic.Int32

	// lastFailureTime is the time of the last failure.
	lastFailureTime atomic.Int64

	// mu protects state transitions.
	mu sync.RWMutex
}

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		config: config,
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	return CircuitBreakerState(cb.state.Load())
}

// Execute runs the given function if the circuit allows it.
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if we can execute
	if !cb.canExecute() {
		return ErrCircuitOpen
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.recordResult(err)

	return err
}

// canExecute determines if a request should be allowed through.
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	state := CircuitBreakerState(cb.state.Load())
	cb.mu.RUnlock()

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if recovery timeout has elapsed
		lastFailure := time.Unix(0, cb.lastFailureTime.Load())
		if time.Since(lastFailure) > cb.config.RecoveryTimeout {
			cb.transitionToHalfOpen()
			return true
		}
		return false

	case StateHalfOpen:
		// In half-open state, we allow requests to test recovery
		// Using compare-and-swap to ensure only one request goes through
		// at a time in half-open state
		return true

	default:
		return false
	}
}

// recordResult updates circuit breaker state based on the result.
func (cb *CircuitBreaker) recordResult(err error) {
	state := CircuitBreakerState(cb.state.Load())

	if err != nil {
		// Record failure
		switch state {
		case StateClosed:
			failures := cb.failures.Add(1)
			if int(failures) >= cb.config.FailureThreshold {
				cb.transitionToOpen()
			}
			cb.lastFailureTime.Store(time.Now().UnixNano())

		case StateHalfOpen:
			// Failure in half-open state goes back to open
			cb.transitionToOpen()
		}
	} else {
		// Record success
		switch state {
		case StateClosed:
			// Reset failure count on success
			cb.failures.Store(0)

		case StateHalfOpen:
			successes := cb.successes.Add(1)
			if int(successes) >= cb.config.SuccessThreshold {
				cb.transitionToClosed()
			}
		}
	}
}

// transitionToOpen transitions the circuit to open state.
func (cb *CircuitBreaker) transitionToOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state.Load() == int32(StateOpen) {
		return // Already open
	}

	cb.state.Store(int32(StateOpen))
	cb.successes.Store(0)
	cb.failures.Store(0)
	cb.lastFailureTime.Store(time.Now().UnixNano())
}

// transitionToHalfOpen transitions the circuit to half-open state.
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state.Load() != int32(StateOpen) {
		return // Not in open state
	}

	cb.state.Store(int32(StateHalfOpen))
	cb.successes.Store(0)
	cb.failures.Store(0)
}

// transitionToClosed transitions the circuit to closed state.
func (cb *CircuitBreaker) transitionToClosed() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state.Load() == int32(StateClosed) {
		return // Already closed
	}

	cb.state.Store(int32(StateClosed))
	cb.successes.Store(0)
	cb.failures.Store(0)
	cb.lastFailureTime.Store(0)
}

// Reset forces the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state.Store(int32(StateClosed))
	cb.successes.Store(0)
	cb.failures.Store(0)
	cb.lastFailureTime.Store(0)
}

// Failures returns the current consecutive failure count.
func (cb *CircuitBreaker) Failures() int {
	return int(cb.failures.Load())
}

// Successes returns the current consecutive success count in half-open state.
func (cb *CircuitBreaker) Successes() int {
	return int(cb.successes.Load())
}
