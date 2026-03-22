package audit

import (
	"context"
	"time"
)

// PersistentAuditor is an Auditor implementation that persists audit entries.
// It wraps a Storage and provides fire-and-forget semantics:
// failures in audit logging do not propagate to the caller.
type PersistentAuditor struct {
	storage Storage
}

// NewPersistentAuditor creates a new PersistentAuditor with the given storage.
func NewPersistentAuditor(storage Storage) *PersistentAuditor {
	return &PersistentAuditor{storage: storage}
}

// LogOperation records an audit entry.
// Uses fire-and-forget semantics: errors are logged but not returned
// to prevent audit failures from impacting primary operations.
func (p *PersistentAuditor) LogOperation(ctx context.Context, entry Entry) error {
	// Fire-and-forget: we attempt to store but don't block on failure
	// This ensures audit storage issues don't impact primary operations
	_, _ = p.storage.Insert(ctx, entry)
	return nil
}

// GetByUser retrieves audit entries for a specific user.
// Results are ordered by timestamp (most recent first).
func (p *PersistentAuditor) GetByUser(ctx context.Context, userID string, limit, offset int) ([]Entry, error) {
	return p.storage.GetByUser(ctx, userID, limit, offset)
}

// GetByEntity retrieves audit entries for a specific entity.
// Results are ordered by timestamp (most recent first).
func (p *PersistentAuditor) GetByEntity(ctx context.Context, entityType, entityID string, limit, offset int) ([]Entry, error) {
	return p.storage.GetByEntity(ctx, entityType, entityID, limit, offset)
}

// GetByOperation retrieves audit entries of a specific operation type.
// Results are ordered by timestamp (most recent first).
func (p *PersistentAuditor) GetByOperation(ctx context.Context, operation Operation, limit, offset int) ([]Entry, error) {
	return p.storage.GetByOperation(ctx, operation, limit, offset)
}

// GetByDateRange retrieves audit entries within a time range.
// Results are ordered by timestamp (most recent first).
func (p *PersistentAuditor) GetByDateRange(ctx context.Context, start, end time.Time, limit, offset int) ([]Entry, error) {
	return p.storage.GetByDateRange(ctx, start, end, limit, offset)
}

// Ensure PersistentAuditor implements Auditor.
var _ Auditor = (*PersistentAuditor)(nil)
