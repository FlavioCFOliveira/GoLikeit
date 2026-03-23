package tracing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/FlavioCFOliveira/GoLikeit/tracing"
)

func TestNoopTracer_Start(t *testing.T) {
	tr := tracing.NoopTracer{}
	ctx := context.Background()

	retCtx, span := tr.Start(ctx, "test_op", tracing.Attributes{"key": "value"})
	if retCtx != ctx {
		t.Error("NoopTracer.Start must return the original context unchanged")
	}
	if span == nil {
		t.Fatal("NoopTracer.Start must not return nil span")
	}
}

func TestNoopSpan_AllMethodsRunWithoutPanic(t *testing.T) {
	span := tracing.NoopSpan{}
	span.SetAttribute("k", "v")
	span.RecordError(errors.New("some error"))
	span.RecordError(nil)
	span.End()
}

// TestNoopTracer_ZeroAllocation verifies the noop tracer does not allocate.
func TestNoopTracer_ZeroAllocation(t *testing.T) {
	tr := tracing.NoopTracer{}
	ctx := context.Background()

	allocs := testing.AllocsPerRun(1000, func() {
		_, span := tr.Start(ctx, "op", nil)
		span.End()
	})

	if allocs > 0 {
		t.Errorf("NoopTracer.Start allocated %v times, expected 0", allocs)
	}
}
