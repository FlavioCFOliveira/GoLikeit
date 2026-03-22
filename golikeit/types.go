package golikeit

import (
	"fmt"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/pagination"
)

// Pagination represents pagination parameters for list queries.
// This is an alias to pagination.Pagination for backward compatibility.
type Pagination = pagination.Pagination

// PaginatedResult is a generic container for paginated query results.
// This is an alias to pagination.Result for backward compatibility.
type PaginatedResult[T any] = pagination.Result[T]

// NewPaginatedResult creates a new paginated result with calculated pagination info.
// This is an alias to pagination.NewResult for backward compatibility.
func NewPaginatedResult[T any](items []T, total int64, limit, offset int) PaginatedResult[T] {
	return pagination.NewResult(items, total, limit, offset)
}

// PaginationConfig holds configuration for pagination behavior.
// This is an alias to pagination.Config for backward compatibility.
type PaginationConfig = pagination.Config

// DefaultPaginationConfig returns a PaginationConfig with sensible defaults.
// This is an alias to pagination.DefaultConfig for backward compatibility.
func DefaultPaginationConfig() PaginationConfig {
	return pagination.DefaultConfig()
}

// EntityTarget identifies a unique entity that can receive reactions.
// It is a tuple of (entity_type, entity_id).
type EntityTarget struct {
	// EntityType is the type of entity (e.g., "photo", "article", "comment").
	EntityType string

	// EntityID is the unique identifier of the entity instance.
	EntityID string
}

// String returns a string representation of the EntityTarget.
func (e EntityTarget) String() string {
	return fmt.Sprintf("%s:%s", e.EntityType, e.EntityID)
}

// IsValid checks if the EntityTarget has valid non-empty fields.
func (e EntityTarget) IsValid() bool {
	return e.EntityType != "" && e.EntityID != ""
}

// UserReaction represents a user's reaction to a specific target.
type UserReaction struct {
	// UserID is the identifier of the user who made the reaction.
	UserID string

	// EntityType is the type of entity reacted to.
	EntityType string

	// EntityID is the identifier of the entity reacted to.
	EntityID string

	// ReactionType is the type of reaction (e.g., "LIKE", "LOVE").
	ReactionType string

	// CreatedAt is when the reaction was first created.
	CreatedAt time.Time

	// UpdatedAt is when the reaction was last modified.
	UpdatedAt time.Time
}

// EntityReaction represents a reaction on a specific entity.
type EntityReaction struct {
	// UserID is the identifier of the user who made the reaction.
	UserID string

	// ReactionType is the type of reaction.
	ReactionType string

	// CreatedAt is when the reaction was created.
	CreatedAt time.Time
}

// EntityCounts holds the aggregated reaction counts for an entity.
type EntityCounts struct {
	// Counts is a map of reaction type to count.
	Counts map[string]int64

	// Total is the sum of all reaction counts.
	Total int64
}

// RecentUserReaction represents a recent user's reaction with timestamp.
type RecentUserReaction struct {
	// UserID is the identifier of the user.
	UserID string

	// ReactionType is the type of reaction.
	ReactionType string

	// Timestamp is when the reaction was made.
	Timestamp time.Time
}

// EntityReactionDetail provides comprehensive information about reactions on an entity.
type EntityReactionDetail struct {
	// EntityType is the type of entity.
	EntityType string

	// EntityID is the identifier of the entity.
	EntityID string

	// TotalReactions is the total number of reactions on this entity.
	TotalReactions int64

	// CountsByType is a map of reaction type to count.
	CountsByType map[string]int64

	// RecentUsers is a map of reaction type to list of recent users.
	RecentUsers map[string][]RecentUserReaction

	// LastReaction is the timestamp of the most recent reaction, if any.
	LastReaction *time.Time
}

// ReactionConfig holds the configuration for reaction types.
type ReactionConfig struct {
	// ReactionTypes is the list of allowed reaction types.
	// Each type must match the pattern ^[A-Z0-9_-]+$.
	ReactionTypes []string
}
