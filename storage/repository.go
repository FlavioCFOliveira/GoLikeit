// Package storage defines the repository interface and provides storage implementations
// for persisting reaction data across multiple backends.
package storage

import (
	"context"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
)

// Filters defines filter criteria for querying reactions.
type Filters struct {
	// EntityType filters by entity type (optional).
	EntityType string

	// ReactionType filters by reaction type (optional).
	ReactionType string

	// Since filters reactions created after this timestamp (optional).
	Since *time.Time

	// Until filters reactions created before this timestamp (optional).
	Until *time.Time
}

// IsEmpty returns true if no filters are set.
func (f Filters) IsEmpty() bool {
	return f.EntityType == "" && f.ReactionType == "" && f.Since == nil && f.Until == nil
}

// Repository defines the interface for reaction storage operations.
// Implementations must be safe for concurrent use.
type Repository interface {
	// AddReaction adds or replaces a reaction for a user.
	// Returns true if a previous reaction was replaced.
	AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error)

	// RemoveReaction removes a user's reaction.
	// Returns golikeit.ErrReactionNotFound if no reaction exists.
	RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error

	// GetUserReaction retrieves a user's current reaction type for a target.
	// Returns ("", golikeit.ErrReactionNotFound) if no reaction exists.
	GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error)

	// HasUserReaction checks if a user has any reaction on a target.
	HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error)

	// GetEntityCounts retrieves the reaction counts for an entity.
	GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error)

	// GetUserReactions retrieves all reactions for a user with optional filters and pagination.
	// Results are ordered by timestamp (most recent first).
	GetUserReactions(ctx context.Context, userID string, filters Filters, pagination pagination.Pagination) ([]golikeit.UserReaction, int64, error)

	// GetUserReactionCounts returns aggregated counts per reaction type for a user.
	// If entityTypeFilter is non-empty, counts are filtered to that entity type.
	GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error)

	// GetUserReactionsByType retrieves reactions of a specific type for a user.
	// Results are ordered by timestamp (most recent first).
	GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pagination pagination.Pagination) ([]golikeit.UserReaction, int64, error)

	// GetEntityReactions retrieves all reactions on an entity with pagination.
	// Results are ordered by timestamp (most recent first).
	GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pagination pagination.Pagination) ([]golikeit.EntityReaction, int64, error)

	// GetRecentReactions retrieves recent reactions on an entity.
	// Results are ordered by timestamp (most recent first).
	GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error)

	// GetLastReactionTime retrieves the timestamp of the most recent reaction on an entity.
	GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error)

	// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
	GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error)

	// Close releases any resources held by the repository.
	Close() error
}
