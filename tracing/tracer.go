// Package tracing provides a minimal distributed tracing abstraction for GoLikeit.
//
// The Tracer interface is intentionally thin so it can bridge to any tracing
// backend without importing backend-specific packages. Consumers running
// OpenTelemetry, Jaeger, or Datadog can write a thin adapter that satisfies
// this interface without modifying the library.
//
// Usage:
//
//	// Implement Tracer using your backend:
//	type OtelAdapter struct{ t trace.Tracer }
//	func (a *OtelAdapter) Start(ctx context.Context, name string, attrs tracing.Attributes) (context.Context, tracing.Span) {
//	    ctx, span := a.t.Start(ctx, name)
//	    for k, v := range attrs { span.SetAttributes(attribute.String(k, v)) }
//	    return ctx, &OtelSpanAdapter{span}
//	}
//
//	// Inject into client:
//	client, _ := golikeit.New(golikeit.Config{
//	    ReactionTypes: []string{"LIKE"},
//	    Tracer: &OtelAdapter{tracer: otelTracer},
//	})
package tracing

import "context"

// Attributes is a map of string key-value pairs attached to a span.
type Attributes map[string]string

// Tracer creates spans for tracing distributed operations.
// Implementations must be safe for concurrent use by multiple goroutines.
type Tracer interface {
	// Start creates a new span for the named operation.
	// The returned context carries the span for downstream propagation.
	// Callers must call Span.End() to finalize the span.
	Start(ctx context.Context, operationName string, attrs Attributes) (context.Context, Span)
}

// Span represents a single unit of traced work.
type Span interface {
	// End marks the span as complete. Must be called exactly once.
	End()
	// RecordError records an error event on the span without ending it.
	RecordError(err error)
	// SetAttribute adds or updates a key-value attribute on the span.
	SetAttribute(key, value string)
}

// NoopTracer is a Tracer that does nothing. It is used when no tracer is
// configured and has zero overhead (no allocations, no function calls that
// cross package boundaries at runtime).
type NoopTracer struct{}

// Start returns a no-op span and the original context unchanged.
func (NoopTracer) Start(ctx context.Context, _ string, _ Attributes) (context.Context, Span) {
	return ctx, NoopSpan{}
}

// NoopSpan is a Span that does nothing.
type NoopSpan struct{}

// End is a no-op.
func (NoopSpan) End() {}

// RecordError is a no-op.
func (NoopSpan) RecordError(_ error) {}

// SetAttribute is a no-op.
func (NoopSpan) SetAttribute(_, _ string) {}
