package audit

import (
	"context"
	"time"
)

// Auditor defines the interface for audit logging operations.
// Implementations must be safe for concurrent use.
type Auditor interface {
	// LogOperation records an audit entry.
	// Implementations should use fire-and-forget semantics:
	// failures should not block or impact the primary operation.
	LogOperation(ctx context.Context, entry Entry) error

	// GetByUser retrieves audit entries for a specific user.
	// Results are ordered by timestamp (most recent first).
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]Entry, error)

	// GetByEntity retrieves audit entries for a specific entity.
	// Results are ordered by timestamp (most recent first).
	GetByEntity(ctx context.Context, entityType, entityID string, limit, offset int) ([]Entry, error)

	// GetByOperation retrieves audit entries of a specific operation type.
	// Results are ordered by timestamp (most recent first).
	GetByOperation(ctx context.Context, operation Operation, limit, offset int) ([]Entry, error)

	// GetByDateRange retrieves audit entries within a time range.
	// Results are ordered by timestamp (most recent first).
	GetByDateRange(ctx context.Context, start, end time.Time, limit, offset int) ([]Entry, error)
}

// NullAuditor is a no-op implementation of Auditor.
// It is the default auditor when no persistent storage is configured.
// Safe for concurrent use.
type NullAuditor struct{}

// NewNullAuditor creates a new NullAuditor instance.
func NewNullAuditor() *NullAuditor {
	return &NullAuditor{}
}

// LogOperation is a no-op that always returns nil.
// It never blocks and executes instantly.
func (n *NullAuditor) LogOperation(_ context.Context, _ Entry) error {
	return nil
}

// GetByUser always returns an empty slice and nil error.
func (n *NullAuditor) GetByUser(_ context.Context, _ string, _, _ int) ([]Entry, error) {
	return []Entry{}, nil
}

// GetByEntity always returns an empty slice and nil error.
func (n *NullAuditor) GetByEntity(_ context.Context, _, _ string, _, _ int) ([]Entry, error) {
	return []Entry{}, nil
}

// GetByOperation always returns an empty slice and nil error.
func (n *NullAuditor) GetByOperation(_ context.Context, _ Operation, _, _ int) ([]Entry, error) {
	return []Entry{}, nil
}

// GetByDateRange always returns an empty slice and nil error.
func (n *NullAuditor) GetByDateRange(_ context.Context, _, _ time.Time, _, _ int) ([]Entry, error) {
	return []Entry{}, nil
}

// Ensure NullAuditor implements Auditor.
var _ Auditor = (*NullAuditor)(nil)
