package golikeit

import (
	"context"
	"time"
)

// ComponentStatus represents the health status of a single component.
type ComponentStatus string

const (
	// StatusUp means the component is healthy and fully operational.
	StatusUp ComponentStatus = "UP"
	// StatusDown means the component is unavailable.
	StatusDown ComponentStatus = "DOWN"
	// StatusDegraded means the component is available but not fully healthy.
	StatusDegraded ComponentStatus = "DEGRADED"
)

// ComponentHealth holds the health information for one system component.
type ComponentHealth struct {
	// Status is the current health status.
	Status ComponentStatus
	// Latency is the round-trip time measured during the health probe.
	// Zero if the component was not probed (e.g. optional and not configured).
	Latency time.Duration
	// Error is a human-readable description of the failure, if any.
	Error string
}

// HealthStatus holds the aggregate health of the Client and its components.
type HealthStatus struct {
	// Overall is the aggregate status: UP only if all required components are UP.
	Overall ComponentStatus
	// Storage is the health of the storage backend.
	Storage ComponentHealth
	// Cache is the health of the cache layer.
	Cache ComponentHealth
	// CircuitBreaker describes the current circuit-breaker state.
	CircuitBreaker ComponentHealth
}

// Pinger is implemented by storage backends that support connectivity probing.
type Pinger interface {
	// Ping verifies the backend is reachable and returns the round-trip latency.
	Ping(ctx context.Context) error
}

// Health probes each component and returns the aggregate health status.
// The provided context controls the maximum time for all probes combined.
func (c *Client) Health(ctx context.Context) HealthStatus {
	h := HealthStatus{
		Overall: StatusUp,
	}

	// --- Storage ---
	if c.storage == nil {
		h.Storage = ComponentHealth{Status: StatusDown, Error: "no storage configured"}
		h.Overall = StatusDown
	} else if pinger, ok := c.storage.(Pinger); ok {
		start := time.Now()
		if err := pinger.Ping(ctx); err != nil {
			h.Storage = ComponentHealth{
				Status:  StatusDown,
				Latency: time.Since(start),
				Error:   err.Error(),
			}
			h.Overall = StatusDown
		} else {
			h.Storage = ComponentHealth{
				Status:  StatusUp,
				Latency: time.Since(start),
			}
		}
	} else {
		// Storage present but does not implement Pinger — report as UP without latency.
		h.Storage = ComponentHealth{Status: StatusUp}
	}

	// --- Cache ---
	if c.cache == nil {
		h.Cache = ComponentHealth{Status: StatusUp} // Cache is optional; absent means disabled, not down.
	} else {
		h.Cache = ComponentHealth{Status: StatusUp}
	}

	// --- Circuit Breaker ---
	cbState := c.circuitBreaker.State()
	cbStatus := StatusUp
	if cbState.String() == "open" {
		cbStatus = StatusDown
		if h.Overall == StatusUp {
			h.Overall = StatusDegraded
		}
	} else if cbState.String() == "half-open" {
		cbStatus = StatusDegraded
		if h.Overall == StatusUp {
			h.Overall = StatusDegraded
		}
	}
	h.CircuitBreaker = ComponentHealth{
		Status: cbStatus,
		Error:  func() string {
			if cbStatus != StatusUp {
				return "circuit breaker is " + cbState.String()
			}
			return ""
		}(),
	}

	return h
}
