// Package business defines the business layer interface for the GoLikeit reaction system.
// The business layer contains all business logic, validation, and orchestration
// of data layer operations.
package business

import (
	"context"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
)

// Service defines the interface for the business layer of the reaction system.
// Implementations must be safe for concurrent use by multiple goroutines.
//
// The business layer is responsible for:
//   - Input validation and sanitization
//   - Reaction type validation against configured types
//   - Enforcing single-reaction-per-user constraint
//   - Cache invalidation
//   - Event emission
//   - Orchestrating data layer operations
//
// All methods accept a context.Context for cancellation and timeout control.
// The error return value should be checked on every call.
type Service interface {
	// AddReaction adds or replaces a user's reaction on an entity.
	//
	// If the user already has a reaction on this entity, it is replaced with the
	// new reaction type. Returns true if a previous reaction was replaced.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user making the reaction
	//   - entityType: The type of entity being reacted to (e.g., "photo")
	//   - entityID: The unique identifier of the entity
	//   - reactionType: The type of reaction (e.g., "LIKE", "LOVE")
	//
	// Returns:
	//   - isReplacement: true if a previous reaction was replaced, false otherwise
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrInvalidReactionType: if reactionType is not configured
	//       * ErrStorageUnavailable: if the storage backend fails
	AddReaction(ctx context.Context, userID, entityType, entityID, reactionType string) (isReplacement bool, err error)

	// RemoveReaction removes a user's reaction from an entity.
	//
	// Returns an error if no reaction exists for this user-entity pair.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user whose reaction should be removed
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//
	// Returns:
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrReactionNotFound: if no reaction exists for this user-entity pair
	//       * ErrStorageUnavailable: if the storage backend fails
	RemoveReaction(ctx context.Context, userID, entityType, entityID string) error

	// GetUserReaction retrieves a user's current reaction type on an entity.
	//
	// Returns an empty string if no reaction exists (not an error).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user to query
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//
	// Returns:
	//   - reactionType: The current reaction type, or empty string if none
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetUserReaction(ctx context.Context, userID, entityType, entityID string) (reactionType string, err error)

	// GetEntityReactionCounts retrieves the counts per reaction type for an entity.
	//
	// Returns counts for all configured reaction types (zero for types with no reactions).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//
	// Returns:
	//   - counts: Map of reaction type to count for all configured types
	//   - total: The sum of all reaction counts
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetEntityReactionCounts(ctx context.Context, entityType, entityID string) (counts map[string]int64, total int64, err error)

	// HasUserReaction checks if a user has any reaction on an entity.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user to query
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//
	// Returns:
	//   - hasReaction: true if the user has reacted, false otherwise
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	HasUserReaction(ctx context.Context, userID, entityType, entityID string) (hasReaction bool, err error)

	// HasUserReactionType checks if a user has a specific reaction type on an entity.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user to query
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//   - reactionType: The reaction type to check for
	//
	// Returns:
	//   - hasReactionType: true if the user has this reaction type, false otherwise
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrInvalidReactionType: if reactionType is not configured
	//       * ErrStorageUnavailable: if the storage backend fails
	HasUserReactionType(ctx context.Context, userID, entityType, entityID, reactionType string) (hasReactionType bool, err error)

	// GetUserReactions retrieves all reactions for a user with pagination.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user to query
	//   - pagination: Pagination parameters (limit, offset)
	//
	// Returns:
	//   - result: Paginated list of user reactions
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetUserReactions(ctx context.Context, userID string, pagination golikeit.Pagination) (golikeit.PaginatedResult[golikeit.UserReaction], error)

	// GetUserReactionsByType retrieves reactions of a specific type for a user.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user to query
	//   - reactionType: The type of reaction to filter by
	//   - pagination: Pagination parameters
	//
	// Returns:
	//   - result: Paginated list of user reactions
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrInvalidReactionType: if reactionType is not configured
	//       * ErrStorageUnavailable: if the storage backend fails
	GetUserReactionsByType(ctx context.Context, userID, reactionType string, pagination golikeit.Pagination) (golikeit.PaginatedResult[golikeit.UserReaction], error)

	// GetEntityReactions retrieves all reactions on an entity with pagination.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//   - pagination: Pagination parameters
	//
	// Returns:
	//   - result: Paginated list of entity reactions
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetEntityReactions(ctx context.Context, entityType, entityID string, pagination golikeit.Pagination) (golikeit.PaginatedResult[golikeit.EntityReaction], error)

	// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//   - maxRecentUsers: Maximum number of recent users to include per reaction type
	//
	// Returns:
	//   - detail: Comprehensive reaction details including counts and recent users
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetEntityReactionDetail(ctx context.Context, entityType, entityID string, maxRecentUsers int) (golikeit.EntityReactionDetail, error)

	// GetUserReactionsBulk retrieves reaction states for multiple targets.
	//
	// This is a bulk operation for efficiently checking a user's reactions
	// on multiple entities in a single call.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user to query
	//   - targets: List of entity targets to check
	//
	// Returns:
	//   - reactions: Map from EntityTarget to reaction type (empty string if no reaction)
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetUserReactionsBulk(ctx context.Context, userID string, targets []golikeit.EntityTarget) (map[golikeit.EntityTarget]string, error)

	// GetEntityCountsBulk retrieves counts for multiple targets.
	//
	// This is a bulk operation for efficiently getting counts
	// for multiple entities in a single call.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - targets: List of entity targets to query
	//
	// Returns:
	//   - counts: Map from EntityTarget to EntityCounts
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetEntityCountsBulk(ctx context.Context, targets []golikeit.EntityTarget) (map[golikeit.EntityTarget]golikeit.EntityCounts, error)

	// GetMultipleUserReactions retrieves reactions from multiple users on a single entity.
	//
	// This is a bulk operation for efficiently getting reactions
	// from multiple users on the same entity.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userIDs: List of users to query
	//   - entityType: The type of entity
	//   - entityID: The unique identifier of the entity
	//
	// Returns:
	//   - reactions: Map from user ID to reaction type (empty string if no reaction)
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetMultipleUserReactions(ctx context.Context, userIDs []string, entityType, entityID string) (map[string]string, error)

	// GetUserReactionCounts retrieves aggregated counts per reaction type for a user.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - userID: The user to query
	//   - entityTypeFilter: Optional filter by entity type (empty for all types)
	//
	// Returns:
	//   - counts: Map of reaction type to count
	//   - error: nil on success, or one of:
	//       * ErrInvalidInput: if any parameter is invalid
	//       * ErrStorageUnavailable: if the storage backend fails
	GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error)

	// Close releases all resources held by the service.
	//
	// It is safe to call Close multiple times. Subsequent calls will return nil.
	// After Close is called, no other methods should be called.
	//
	// Returns:
	//   - error: nil on success, or an error if cleanup fails
	Close() error
}

// Config holds the configuration for creating a new Service.
type Config struct {
	// ReactionTypes is the list of allowed reaction types.
	// At least one type must be provided.
	ReactionTypes []string

	// EnableCaching enables the caching layer.
	EnableCaching bool

	// EnableEvents enables the event system.
	EnableEvents bool

	// EnableRateLimiting enables rate limiting.
	EnableRateLimiting bool
}

// Validate validates the business layer configuration.
func (c Config) Validate() error {
	if len(c.ReactionTypes) == 0 {
		return ErrNoReactionTypes
	}

	seen := make(map[string]struct{}, len(c.ReactionTypes))
	for _, rt := range c.ReactionTypes {
		if rt == "" {
			return ErrInvalidReactionType
		}
		if _, exists := seen[rt]; exists {
			return ErrDuplicateReactionType
		}
		seen[rt] = struct{}{}
	}

	return nil
}

// ServiceError is the common error type for business layer errors.
type ServiceError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	return e.Op + ": " + e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// Common business layer errors.
// These should be used with errors.Is() for checking.
var (
	// ErrNoReactionTypes indicates no reaction types were configured.
	ErrNoReactionTypes = &ServiceError{Op: "config", Err: errNoReactionTypes}

	// ErrInvalidReactionType indicates an invalid or unknown reaction type.
	ErrInvalidReactionType = &ServiceError{Op: "validate", Err: errInvalidReactionType}

	// ErrDuplicateReactionType indicates a duplicate reaction type in config.
	ErrDuplicateReactionType = &ServiceError{Op: "config", Err: errDuplicateReactionType}

	// ErrInvalidInput indicates invalid input parameters.
	ErrInvalidInput = &ServiceError{Op: "validate", Err: errInvalidInput}

	// ErrReactionNotFound indicates a reaction was not found.
	ErrReactionNotFound = &ServiceError{Op: "query", Err: errReactionNotFound}

	// ErrStorageUnavailable indicates the storage backend is unavailable.
	ErrStorageUnavailable = &ServiceError{Op: "storage", Err: errStorageUnavailable}

	// ErrNotImplemented indicates a method is not yet implemented.
	ErrNotImplemented = &ServiceError{Op: "method", Err: errNotImplemented}
)

// Internal error values for wrapping.
var (
	errNoReactionTypes       = NewBusinessError("no reaction types configured")
	errInvalidReactionType   = NewBusinessError("invalid reaction type")
	errDuplicateReactionType = NewBusinessError("duplicate reaction type")
	errInvalidInput          = NewBusinessError("invalid input")
	errReactionNotFound      = NewBusinessError("reaction not found")
	errStorageUnavailable    = NewBusinessError("storage unavailable")
	errNotImplemented        = NewBusinessError("not implemented")
)

// BusinessError represents a domain-specific error.
type BusinessError struct {
	msg string
}

// Error implements the error interface.
func (e *BusinessError) Error() string {
	return e.msg
}

// NewBusinessError creates a new BusinessError.
func NewBusinessError(msg string) error {
	return &BusinessError{msg: msg}
}
