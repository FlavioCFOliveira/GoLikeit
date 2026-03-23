// Package e2e provides end-to-end testing utilities.
//
//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	"github.com/FlavioCFOliveira/GoLikeit/storage"
)

// e2eStorage wraps storage.MemoryStorage to implement golikeit.ReactionStorage.
type e2eStorage struct {
	inner *storage.MemoryStorage
}

// newE2EStorage creates a new E2E storage wrapper.
func newE2EStorage() *e2eStorage {
	return &e2eStorage{
		inner: storage.NewMemoryStorage(),
	}
}

// AddReaction adds or replaces a reaction.
func (e *e2eStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	return e.inner.AddReaction(ctx, userID, target, reactionType)
}

// RemoveReaction removes a reaction.
func (e *e2eStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	return e.inner.RemoveReaction(ctx, userID, target)
}

// GetUserReaction retrieves a user's reaction.
func (e *e2eStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	return e.inner.GetUserReaction(ctx, userID, target)
}

// GetEntityCounts retrieves counts for an entity.
func (e *e2eStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	return e.inner.GetEntityCounts(ctx, target)
}

// GetUserReactions retrieves all reactions for a user with pagination.
func (e *e2eStorage) GetUserReactions(ctx context.Context, userID string, pg golikeit.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Convert golikeit.Pagination to storage.Filters and pagination.Pagination
	filters := storage.Filters{}
	return e.inner.GetUserReactions(ctx, userID, filters, pagination.Pagination{
		Limit:  pg.Limit,
		Offset: pg.Offset,
	})
}

// GetUserReactionCounts retrieves aggregated counts per reaction type.
func (e *e2eStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	return e.inner.GetUserReactionCounts(ctx, userID, entityTypeFilter)
}

// GetUserReactionsByType retrieves reactions of a specific type.
func (e *e2eStorage) GetUserReactionsByType(ctx context.Context, userID, reactionType string, pg golikeit.Pagination) ([]golikeit.UserReaction, int64, error) {
	return e.inner.GetUserReactionsByType(ctx, userID, reactionType, pagination.Pagination{
		Limit:  pg.Limit,
		Offset: pg.Offset,
	})
}

// GetEntityReactions retrieves all reactions on an entity.
func (e *e2eStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pg golikeit.Pagination) ([]golikeit.EntityReaction, int64, error) {
	return e.inner.GetEntityReactions(ctx, target, pagination.Pagination{
		Limit:  pg.Limit,
		Offset: pg.Offset,
	})
}

// GetRecentReactions retrieves recent reactions.
func (e *e2eStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	return e.inner.GetRecentReactions(ctx, target, limit)
}

// GetLastReactionTime retrieves the timestamp of the most recent reaction.
func (e *e2eStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	return e.inner.GetLastReactionTime(ctx, target)
}

// Close releases resources.
func (e *e2eStorage) Close() error {
	return e.inner.Close()
}
