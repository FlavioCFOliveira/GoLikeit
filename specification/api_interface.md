# API Interface Specification

## Overview

The system exposes a public API that allows consuming applications to integrate reaction functionality. The API is library-based (Go package) providing programmatic access to all reaction capabilities.

The module is reaction-type agnostic. All reaction types are defined by the consuming application during initialization.

## Functional Requirements

### Requirement 1: Reaction Type Configuration

The system accepts reaction type configuration during client initialization.

**Configuration Interface:**
```go
type ReactionConfig struct {
    ReactionTypes []string // Must match pattern: ^[A-Z0-9_-]+$
}
```

**Validation:**
- Each reaction type must match pattern `^[A-Z0-9_-]+$`
- Each reaction type must be 1-64 characters
- Duplicate reaction types are rejected
- Empty reaction type list is rejected
- Validation occurs during `New()` - module fails to initialize if invalid

**Configuration Pattern:**
```go
client, err := golikeit.New(golikeit.Config{
    DatabaseURL: "postgres://user:pass@localhost/reactions",
    ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE", "ANGRY"},
})
```

### Requirement 2: Core Reaction Operations

The system provides operations for managing user reactions.

**Operations:**

```go
// AddReaction adds or replaces a user's reaction
func (c *Client) AddReaction(ctx context.Context, userID, entityType, entityID, reactionType string) (isReplacement bool, err error)

// RemoveReaction removes the user's current reaction
func (c *Client) RemoveReaction(ctx context.Context, userID, entityType, entityID string) error

// GetUserReaction retrieves the current reaction for a user
func (c *Client) GetUserReaction(ctx context.Context, userID, entityType, entityID string) (reactionType string, err error)

// HasUserReaction checks if user has any reaction
func (c *Client) HasUserReaction(ctx context.Context, userID, entityType, entityID string) (bool, error)

// HasUserReactionType checks if user has specific reaction type
func (c *Client) HasUserReactionType(ctx context.Context, userID, entityType, entityID, reactionType string) (bool, error)
```

### Requirement 3: Query Operations

The system provides query operations for retrieving reaction data.

**Operations:**

```go
// GetEntityReactionCounts retrieves counts per reaction type
func (c *Client) GetEntityReactionCounts(ctx context.Context, entityType, entityID string) (counts map[string]int64, total int64, err error)

// GetEntityReactionDetail retrieves consolidated reaction data
func (c *Client) GetEntityReactionDetail(ctx context.Context, entityType, entityID string, options ReactionDetailOptions) (EntityReactionDetail, error)

// GetUserReactions retrieves all reactions for a user
func (c *Client) GetUserReactions(ctx context.Context, userID string, pagination Pagination) (PaginatedResult[UserReaction], error)

// GetUserReactionsByType retrieves reactions of specific type for user
func (c *Client) GetUserReactionsByType(ctx context.Context, userID, reactionType string, pagination Pagination) (PaginatedResult[UserReaction], error)

// GetEntityReactions retrieves all reactions on a Reaction Target
func (c *Client) GetEntityReactions(ctx context.Context, entityType, entityID string, pagination Pagination) (PaginatedResult[EntityReaction], error)
```

**Reaction Detail Options:**
```go
type ReactionDetailOptions struct {
    MaxRecentUsers int
}

type EntityReactionDetail struct {
    EntityType     string
    EntityID       string
    TotalReactions int64
    CountsByType   map[string]int64
    RecentUsers    map[string][]RecentUserReaction
    LastReaction   *time.Time
}
```

### Requirement 4: Bulk Operations

The API supports bulk operations for efficient batch processing.

**Operations:**

```go
// GetUserReactionsBulk retrieves reaction states for multiple Reaction Targets
func (c *Client) GetUserReactionsBulk(ctx context.Context, userID string, targets []EntityTarget) (map[EntityTarget]string, error)

// GetEntityCountsBulk retrieves counts for multiple Reaction Targets
func (c *Client) GetEntityCountsBulk(ctx context.Context, targets []EntityTarget) (map[EntityTarget]EntityCounts, error)

// GetMultipleUserReactions retrieves reactions from multiple users
func (c *Client) GetMultipleUserReactions(ctx context.Context, userIDs []string, entityType, entityID string) (map[string]string, error)
```

### Requirement 5: Error Types

The system exports typed errors for programmatic handling.

**Error Types:**
```go
var (
    ErrInvalidInput          = errors.New("invalid input")
    ErrReactionNotFound      = errors.New("reaction not found")
    ErrStorageUnavailable    = errors.New("storage unavailable")
    ErrInvalidReactionType   = errors.New("invalid reaction type")
    ErrInvalidReactionFormat = errors.New("reaction type must match [A-Z0-9_-]+")
    ErrNoReactionTypes       = errors.New("at least one reaction type required")
    ErrDuplicateReactionType = errors.New("duplicate reaction type")
)
```

### Requirement 6: Context Support

All API methods accept context.Context for operation control.

- Context cancellation terminates operations promptly
- Context timeouts return timeout errors
- Operations do not outlive the provided context

### Requirement 7: Shutdown and Cleanup

The system provides graceful shutdown capabilities.

- A `Close()` method releases all resources
- `Close()` is safe to call multiple times (idempotent)
- Subsequent API calls return errors after `Close()`

### Requirement 8: Thread Safety

The client is safe for concurrent use.

- Multiple goroutines may call API methods simultaneously
- No external synchronization required by callers
- Internal state is protected from race conditions

### Requirement 9: Simple Configuration API

The module exposes a simple, intuitive API.

- Basic configuration requires database connection and at least one reaction type
- Optional parameters have sensible defaults
- Configuration errors are caught at initialization

### Requirement 10: Caching Layer

The system provides an optional caching layer.

- Enabled/disabled via configuration
- Configurable TTL and max size
- LRU eviction when max size reached
- Cache invalidation occurs at Reaction Target granularity

### Requirement 11: Pagination

All query operations use consistent limit-offset pagination.

**Configuration:**
```go
type PaginationConfig struct {
    DefaultLimit int // Default: 20
    MaxLimit     int // Default: 100
    MaxOffset    int // Default: 10000
}
```

**Pagination Model:**
```go
type Pagination struct {
    Limit  int
    Offset int
}

type PaginatedResult[T] struct {
    Items       []T
    Total       int64
    TotalPages  int
    CurrentPage int
    Limit       int
    Offset      int
    HasNext     bool
    HasPrev     bool
}
```

## Constraints and Limitations

1. **Library API Only:** Provides a Go library API, not HTTP/gRPC endpoints.
2. **Go Version Compatibility:** Requires Go 1.21 or later.
3. **No Callbacks:** All operations are synchronous with context support.
4. **No Streaming:** Query operations return complete result sets.
5. **Cache Limitations:** Cache is optional and in-process only.
6. **Reaction Type Immutability:** Reaction types cannot be modified after initialization.

## API Usage Example

```go
config := golikeit.Config{
    Database: golikeit.DatabaseConfig{
        Type:     "postgresql",
        Host:     "localhost",
        Port:     5432,
        Database: "reactions",
    },
    ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE", "ANGRY"},
}

client, err := golikeit.New(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

ctx := context.Background()

// User adds a LIKE reaction
isReplaced, err := client.AddReaction(ctx, "user_123", "photo", "photo_456", "LIKE")

// User replaces LIKE with LOVE
isReplaced, err = client.AddReaction(ctx, "user_123", "photo", "photo_456", "LOVE")

// Get counts
counts, total, err := client.GetEntityReactionCounts(ctx, "photo", "photo_456")

// Remove reaction
err = client.RemoveReaction(ctx, "user_123", "photo", "photo_456")
```

## Acceptance Criteria

1. **AC1:** All API methods accept context.Context
2. **AC2:** AddReaction returns isReplacement flag
3. **AC3:** RemoveReaction returns error if no reaction exists
4. **AC4:** GetUserReaction returns reaction type or empty string
5. **AC5:** GetEntityReactionCounts returns counts for all configured reaction types
6. **AC6:** Client is safe for concurrent use
7. **AC7:** Context cancellation terminates operations
8. **AC8:** Close() releases resources and is idempotent
9. **AC9:** Configuration errors are caught at initialization
10. **AC10:** Cache can be enabled/disabled via configuration
11. **AC11:** Bulk operations minimize database round trips
12. **AC12:** Pagination is consistent across all list queries
13. **AC13:** Reaction type validation occurs at initialization
