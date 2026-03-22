# API Interface Specification

## Overview

The system shall expose a public API that allows consuming applications to integrate reaction functionality. The API is library-based (Go package) rather than network-based, providing programmatic access to all reaction capabilities through idiomatic Go interfaces.

The module is reaction-type agnostic. All reaction types are defined by the consuming application during initialization through configuration.

## Functional Requirements

### Requirement 1: Reaction Type Configuration

**Description:** The system shall accept reaction type configuration during client initialization.

**Configuration Interface:**
```go
// ReactionConfig defines the reaction types supported by the module
type ReactionConfig struct {
    // ReactionTypes is the list of allowed reaction types
    // Each type must match pattern: ^[A-Z0-9_-]+$
    // Minimum 1 type required
    ReactionTypes []string
}
```

**Validation Requirements:**
- Each reaction type must match pattern `^[A-Z0-9_-]+$`
- Each reaction type must be 1-64 characters
- Duplicate reaction types are rejected
- Empty reaction type list is rejected
- Validation occurs during `New()` - module fails to initialize if invalid

**Configuration Pattern:**
```go
// Configuration with custom reaction types
client, err := golikeit.New(golikeit.Config{
    DatabaseURL: "postgres://user:pass@localhost/reactions",
    ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE", "ANGRY"},
})
```

### Requirement 2: Core Reaction Operations

**Description:** The system shall provide operations for managing user reactions.

**Operations:**

```go
// AddReaction adds or replaces a user's reaction on a Reaction Target
// If user already has a reaction, it is replaced with the new type
// Returns: isReplacement (true if previous reaction existed), error
func (c *Client) AddReaction(ctx context.Context, userID, entityType, entityID, reactionType string) (isReplacement bool, err error)

// RemoveReaction removes the user's current reaction from a Reaction Target
// Returns error if no reaction exists
func (c *Client) RemoveReaction(ctx context.Context, userID, entityType, entityID string) error

// GetUserReaction retrieves the current reaction for a user on a Reaction Target
// Returns: reactionType (empty string if none), error
func (c *Client) GetUserReaction(ctx context.Context, userID, entityType, entityID string) (reactionType string, err error)

// HasUserReaction checks if user has any reaction on a Reaction Target
// Returns: true if user has a reaction, false otherwise
func (c *Client) HasUserReaction(ctx context.Context, userID, entityType, entityID string) (bool, error)

// HasUserReactionType checks if user has a specific reaction type on a Reaction Target
// Returns: true if user has the specified reaction type, false otherwise
func (c *Client) HasUserReactionType(ctx context.Context, userID, entityType, entityID, reactionType string) (bool, error)
```

### Requirement 3: Query Operations

**Description:** The system shall provide query operations for retrieving reaction data.

**Query Operations:**

```go
// GetEntityReactionCounts retrieves the count of each reaction type for a Reaction Target
// Returns: map[reactionType]count, totalCount, error
func (c *Client) GetEntityReactionCounts(ctx context.Context, entityType, entityID string) (counts map[string]int64, total int64, err error)

// GetEntityReactionDetail retrieves consolidated reaction data for a Reaction Target
// Includes counts and recent users per reaction type
func (c *Client) GetEntityReactionDetail(ctx context.Context, entityType, entityID string, options ReactionDetailOptions) (EntityReactionDetail, error)

// GetUserReactions retrieves all reactions for a user
// Returns paginated list of reactions with reaction type
func (c *Client) GetUserReactions(ctx context.Context, userID string, pagination Pagination) (PaginatedResult[UserReaction], error)

// GetUserReactionsByType retrieves reactions of a specific type for a user
func (c *Client) GetUserReactionsByType(ctx context.Context, userID, reactionType string, pagination Pagination) (PaginatedResult[UserReaction], error)

// GetEntityReactions retrieves all reactions on a Reaction Target
func (c *Client) GetEntityReactions(ctx context.Context, entityType, entityID string, pagination Pagination) (PaginatedResult[EntityReaction], error)

// GetEntityReactionsByType retrieves reactions of a specific type on a Reaction Target
func (c *Client) GetEntityReactionsByType(ctx context.Context, entityType, entityID, reactionType string, pagination Pagination) (PaginatedResult[EntityReaction], error)
```

**Reaction Detail Options:**
```go
type ReactionDetailOptions struct {
    // MaxRecentUsers is the number of recent users to return per reaction type
    // Default: 10, Maximum: 100
    MaxRecentUsers int
}

// EntityReactionDetail provides consolidated reaction data
type EntityReactionDetail struct {
    EntityType    string                          // Type of entity
    EntityID      string                          // Entity identifier
    TotalReactions int64                          // Total reactions across all types
    CountsByType  map[string]int64                // Count per reaction type
    RecentUsers   map[string][]RecentUserReaction // Recent users per reaction type
    LastReaction  *time.Time                      // Timestamp of last reaction (if any)
}

type RecentUserReaction struct {
    UserID    string    // User who reacted
    CreatedAt time.Time // When the reaction was created
}
```

### Requirement 4: Bulk Operations

**Description:** The API shall support bulk operations for efficient batch processing.

**Bulk Operations:**

```go
// GetUserReactionsBulk retrieves reaction states for multiple Reaction Targets for a single user
// Input: user_id, slice of (entity_type, entity_id) tuples
// Output: Map of entity target to reaction type (empty string if no reaction)
func (c *Client) GetUserReactionsBulk(ctx context.Context, userID string, targets []EntityTarget) (map[EntityTarget]string, error)

// GetEntityCountsBulk retrieves counts for multiple Reaction Targets
// Input: Slice of (entity_type, entity_id) tuples
// Output: Map of entity target to reaction counts
func (c *Client) GetEntityCountsBulk(ctx context.Context, targets []EntityTarget) (map[EntityTarget]EntityCounts, error)

// GetMultipleUserReactions retrieves reactions from multiple users for a single Reaction Target
// Input: Slice of user_ids, entity_type, entity_id
// Output: Map of user_id to reaction type
func (c *Client) GetMultipleUserReactions(ctx context.Context, userIDs []string, entityType, entityID string) (map[string]string, error)
```

**Entity Counts Structure:**
```go
type EntityCounts struct {
    EntityType     string
    EntityID       string
    CountsByType   map[string]int64 // Count per reaction type
    TotalReactions int64
}
```

### Requirement 5: Client Initialization

**Description:** The system shall provide a client constructor that accepts configuration including reaction types.

**Configuration:**
```go
type Config struct {
    DatabaseURL   string
    ReactionTypes []string        // Required: at least one reaction type
    CacheConfig   *CacheConfig    // Optional
    AuditConfig   *AuditConfig    // Optional
    RateLimitConfig *RateLimitConfig // Optional
}
```

**Requirements:**
- Client construction shall validate reaction type configuration
- Invalid reaction types shall return descriptive errors
- Empty reaction type list shall return error
- The client shall be safe for concurrent use
- Resources shall be properly initialized during construction

### Requirement 6: Error Types

**Description:** The system shall export typed errors for programmatic handling.

**Error Types:**
```go
var (
    // ErrInvalidInput: Input validation failed
    ErrInvalidInput = errors.New("invalid input")

    // ErrReactionNotFound: Requested reaction does not exist
    ErrReactionNotFound = errors.New("reaction not found")

    // ErrStorageUnavailable: Database connection or query failed
    ErrStorageUnavailable = errors.New("storage unavailable")

    // ErrInvalidReactionType: Reaction type not in configured registry
    ErrInvalidReactionType = errors.New("invalid reaction type")

    // ErrInvalidReactionFormat: Reaction type has invalid format
    ErrInvalidReactionFormat = errors.New("reaction type must match [A-Z0-9_-]+")

    // ErrNoReactionTypes: Reaction type list is empty
    ErrNoReactionTypes = errors.New("at least one reaction type required")

    // ErrDuplicateReactionType: Duplicate reaction type in configuration
    ErrDuplicateReactionType = errors.New("duplicate reaction type")
)
```

**Requirements:**
- Errors shall be comparable using errors.Is()
- Error messages shall be descriptive and actionable
- Sensitive information shall not be exposed in error messages

### Requirement 7: Context Support

**Description:** All API methods shall accept context.Context for operation control.

**Requirements:**
- Context cancellation shall be respected and terminate operations promptly
- Context timeouts shall be respected and return timeout errors
- Context values may be used for tracing and logging (optional)
- Operations shall not outlive the provided context

**Behavior:**
- Cancelled contexts result in immediate termination with context.Canceled error
- Expired timeouts result in termination with context.DeadlineExceeded error
- Resources acquired before cancellation shall be released

### Requirement 8: Shutdown and Cleanup

**Description:** The system shall provide graceful shutdown capabilities.

**Requirements:**
- A Close() method shall release all resources
- Close() shall complete pending operations or wait for them
- Close() shall be safe to call multiple times (idempotent)
- Close() shall return an error if cleanup fails

**Behavior:**
- Database connections are closed
- Goroutines are terminated
- Resources are released
- Subsequent API calls return errors after Close()

### Requirement 9: Thread Safety

**Description:** The client shall be safe for concurrent use.

**Requirements:**
- Multiple goroutines may call API methods simultaneously
- No external synchronization required by callers
- Internal state shall be protected from race conditions
- Concurrent operations shall not corrupt data

### Requirement 10: Simple Configuration API

**Description:** The module shall expose a simple, intuitive API for users to configure and use.

**Requirements:**
- **Minimal Setup:** Basic configuration requires database connection and at least one reaction type
- **Sensible Defaults:** All optional parameters have sensible defaults
- **Fluent Interface:** Optional configuration uses functional options for readability
- **Validation on Build:** Configuration errors are caught at initialization time with clear messages

**Configuration Pattern:**
```go
// Simple - minimal configuration
client, err := golikeit.New(golikeit.Config{
    DatabaseURL: "postgres://user:pass@localhost/reactions",
    ReactionTypes: []string{"LIKE", "DISLIKE"},
})

// Advanced - with options
client, err := golikeit.New(
    golikeit.WithDatabaseURL("postgres://user:pass@localhost/reactions"),
    golikeit.WithReactionTypes("LIKE", "DISLIKE", "LOVE", "ANGRY"),
    golikeit.WithCache(golikeit.CacheConfig{Enabled: true, TTL: 5 * time.Minute}),
    golikeit.WithAuditDatabaseURL("postgres://user:pass@localhost/audit"),
)
```

### Requirement 11: Caching Layer

**Description:** The system shall provide an optional caching layer to avoid database access and improve response times.

**Cache Configuration:**
- **Enabled:** Cache can be enabled/disabled via configuration
- **TTL:** Time-to-live for cached entries (configurable, default 5 minutes)
- **Max Size:** Maximum number of entries in cache (configurable, default 10,000)
- **Eviction Policy:** LRU (Least Recently Used) eviction when max size reached

**Cached Data:**
- **User Reactions:** Individual user reaction states are cached
- **Entity Counts:** Aggregated reaction counts per entity are cached
- **Cache Keys:** Composite keys including user_id, entity_type, entity_id

**Cache Invalidation:**
- Cache invalidation occurs at **Reaction Target granularity** (user_id + entity_type + entity_id)
- When user_123 calls AddReaction("photo", "photo_456", "LIKE"), only cache entries for (user_123, "photo", "photo_456") are invalidated
- **Invalidation Scope:**
  - User reaction cache entry for the specific user and entity
  - Entity counts cache entry for the specific entity
  - User reaction lists are NOT invalidated (rebuild on next query)
- **Bulk Invalidation:** Operations affecting multiple entities trigger individual invalidations for each affected Reaction Target
- TTL-based expiration for automatic stale data removal

**Requirements:**
- Cache must be thread-safe for concurrent access
- Cache misses shall transparently fall back to database
- Cache hit/miss metrics shall be exposed

### Requirement 12: Pagination Support

**Description:** All query operations shall use a consistent pagination mechanism with configurable parameters.

**Pagination Configuration:**
```go
type PaginationConfig struct {
    // Default number of records per page
    DefaultLimit int // Default: 20, configurable

    // Maximum allowed records per page (hard upper bound)
    MaxLimit int // Default: 100, configurable

    // Maximum offset allowed (prevents excessive pagination)
    MaxOffset int // Default: 10000, configurable
}
```

**Pagination Model (Limit-Offset):**
```go
type Pagination struct {
    Limit  int // Number of records requested (max from config)
    Offset int // Starting position (0-based)
}

type PaginatedResult[T] struct {
    Items       []T   // Current page items
    Total       int64 // Total records matching query
    TotalPages  int   // Total number of pages (calculated from Total/Limit)
    CurrentPage int   // Current page number (1-based, calculated from Offset/Limit)
    Limit       int   // Records per page (as requested, capped at MaxLimit)
    Offset      int   // Current offset position
    HasNext     bool  // Whether there are more records after this page
    HasPrev     bool  // Whether there are records before this page
}
```

**Limit-Offset Principle:**
- **Limit:** Indicates how many records are requested (e.g., 20), capped at MaxLimit
- **Offset:** Indicates the starting position (0-based, e.g., 0 for first page, 20 for second page), capped at MaxOffset
- **Page Calculation:** Page = (Offset / Limit) + 1
- **Total Pages:** Calculated as ceil(Total / Limit)
- **Default Behavior:** If no pagination specified, uses DefaultLimit

**Requirements:**
- **Mandatory Pagination:** All list queries return paginated results (no unbounded queries)
- **Configurable Defaults:** DefaultLimit is externally configurable
- **Enforced Maximum:** MaxLimit is externally configurable and enforced
- **Offset Protection:** MaxOffset prevents excessive pagination depth
- **Consistent Ordering:** Results ordered by timestamp descending (newest first, based on reaction created_at)

## Constraints and Limitations

1. **Library API Only:** The system provides a Go library API, not HTTP/gRPC endpoints. Network APIs are the responsibility of the consuming application.

2. **Go Version Compatibility:** The API requires Go 1.21 or later (generics, slog, etc.).

3. **No Callbacks:** The API does not support callback patterns; all operations are synchronous with context support.

4. **No Streaming:** Query operations return complete result sets; streaming/pagination is handled through limit/offset parameters.

5. **No Event Hooks:** The API does not provide event callbacks; observing changes is the responsibility of the consuming application (see Event System specification).

6. **Simple Configuration Focus:** The API prioritizes simplicity over exhaustive configurability; edge cases may require custom wrappers.

7. **Cache Limitations:** Cache is optional and in-process only; distributed caching is not provided.

8. **Reaction Type Immutability:** Reaction types cannot be modified after initialization. The module must be restarted to change supported reaction types.

## API Usage Example

```go
// Configuration
config := golikeit.Config{
    Database: golikeit.DatabaseConfig{
        Type:     "postgresql",
        Host:     "localhost",
        Port:     5432,
        Database: "reactions",
        User:     "app",
        Password: "secret",
    },
    ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE", "ANGRY"},
}

// Client construction
client, err := golikeit.New(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Operations
ctx := context.Background()

// User adds a LIKE reaction
isReplaced, err := client.AddReaction(ctx, "user_123", "photo", "photo_456", "LIKE")
// isReplaced = false (no previous reaction)

// User replaces LIKE with LOVE
isReplaced, err = client.AddReaction(ctx, "user_123", "photo", "photo_456", "LOVE")
// isReplaced = true (LIKE was replaced with LOVE)

// Check reaction state
reactionType, err := client.GetUserReaction(ctx, "user_123", "photo", "photo_456")
// reactionType = "LOVE"

// Get counts for Reaction Target
counts, total, err := client.GetEntityReactionCounts(ctx, "photo", "photo_456")
// counts = {"LIKE": 0, "LOVE": 1, "DISLIKE": 0, "ANGRY": 0}
// total = 1

// Remove reaction
err = client.RemoveReaction(ctx, "user_123", "photo", "photo_456")

// Check if reaction exists
hasReaction, err := client.HasUserReaction(ctx, "user_123", "photo", "photo_456")
// hasReaction = false
```

## Relationships with Other Functional Blocks

- **[architecture.md](architecture.md):** Defines the layered architecture exposing the public API
- **[reaction_management.md](reaction_management.md):** Defines the operations available through the API
- **[data_persistence.md](data_persistence.md):** Defines the configuration for storage

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version with Like/Unlike/Dislike/Undislike operations |
| 2026-03-22 | Major | Refactored to abstract reaction model - replaced specific operations with generic AddReaction/RemoveReaction, added ReactionTypes configuration |

## Acceptance Criteria

1. **AC1:** All API methods accept context.Context as the first parameter
2. **AC2:** All API methods return appropriate results with error tuples
3. **AC3:** Exported error types allow comparison with errors.Is()
4. **AC4:** Client construction validates reaction type configuration and returns descriptive errors
5. **AC5:** Client is safe for concurrent use without external synchronization
6. **AC6:** Context cancellation terminates operations promptly
7. **AC7:** Context timeouts are respected and return timeout errors
8. **AC8:** Close() releases resources and is safe to call multiple times
9. **AC9:** Configuration types use struct tags and provide defaults
10. **AC10:** API is idiomatic Go (no unnecessary abstraction, clear naming)
11. **AC11:** Basic configuration requires database URL and at least one reaction type
12. **AC12:** AddReaction returns isReplacement flag indicating if previous reaction existed
13. **AC13:** RemoveReaction returns error if no reaction exists
14. **AC14:** GetUserReaction returns reaction type or empty string if none
15. **AC15:** GetEntityReactionCounts returns counts for all configured reaction types
16. **AC16:** Cache can be enabled/disabled via configuration with configurable TTL
17. **AC17:** Cache invalidation occurs on reaction state changes
18. **AC18:** Bulk operations minimize database round trips
19. **AC19:** Pagination is used for queries potentially returning >50 records
20. **AC20:** Reaction type validation occurs at initialization with pattern [A-Z0-9_-]+
