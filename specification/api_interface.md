# API Interface Specification

## Overview

The system shall expose a public API that allows consuming applications to integrate reaction functionality. The API is library-based (Go package) rather than network-based, providing programmatic access to all reaction capabilities through idiomatic Go interfaces.

## Functional Requirements

### Requirement 1: Public API Surface

**Description:** The system shall expose a clean, idiomatic Go API for all reaction operations.

**Core Operations:**
- Like(ctx, user_id, entity_type, entity_id) error
- Unlike(ctx, user_id, entity_type, entity_id) error
- Dislike(ctx, user_id, entity_type, entity_id) error
- Undislike(ctx, user_id, entity_type, entity_id) error

**Query Operations:**
- GetUserReaction(ctx, user_id, entity_type, entity_id) (ReactionState, error)
- GetEntityCounts(ctx, entity_type, entity_id) (ReactionCounts, error)
- GetUserReactions(ctx, user_id, pagination) (PaginatedResult[Reaction], error)
- GetEntityReactions(ctx, entity_type, entity_id, pagination) (PaginatedResult[Reaction], error)
- GetUserLikes(ctx, user_id, pagination) (PaginatedResult[EntityTarget], error) - Returns paginated entities liked by user
- GetUserDislikes(ctx, user_id, pagination) (PaginatedResult[EntityTarget], error) - Returns paginated entities disliked by user
- GetEntityReactionsWithUsers(ctx, entity_type, entity_id, options) (EntityReactionDetail, error) - Consolidated view with counts and recent users
- **HasUserLiked(ctx, user_id, entity_type, entity_id) (bool, error)** - Ultra-fast check if user liked target
- **HasUserDisliked(ctx, user_id, entity_type, entity_id) (bool, error)** - Ultra-fast check if user disliked target

**Consolidated Query Operations:**
- **EntityReactionDetail** provides:
  - Total likes count
  - Total dislikes count
  - Last N users who liked (configurable N, default 10)
  - Last N users who disliked (configurable N, default 10)
  - Timestamps of most recent reactions

**Requirements:**
- All query operations return complete, consolidated data in a single call
- No separate calls required to get counts and user lists
- **Pagination required** for queries that may return more than 50 records
- Options parameter allows configuring N (number of recent users to return)

**Bulk Operations:**
- GetUserReactionsBulk(ctx, user_id, entityTargets) (map[EntityTarget]ReactionState, error)
- GetEntityCountsBulk(ctx, entityTargets) (map[EntityTarget]ReactionCounts, error)
- GetMultipleUserReactions(ctx, userIDs, entity_type, entity_id) (map[string]ReactionState, error)

**Requirements:**
- All operations accept a context.Context for cancellation and timeouts
- All operations return (result, error) tuples
- Error types are exported for programmatic error handling
- Types are exported with clear documentation

### Requirement 2: Client Initialization

**Description:** The system shall provide a client constructor that accepts configuration.

**Inputs:**
- Database configuration (type, connection string, pool settings)
- Optional: Cache configuration (enabled, TTL, max size)
- Optional: Logger interface
- Optional: Metrics collector interface
- Optional: Custom validator functions

**Outputs:**
- Client instance implementing the public API interface
- Initialization error if configuration is invalid

**Requirements:**
- Client construction shall validate configuration
- Invalid configurations shall return descriptive errors
- The client shall be safe for concurrent use
- Resources shall be properly initialized during construction
- Cache layer shall be initialized if enabled

### Requirement 3: Context Support

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

### Requirement 4: Error Types

**Description:** The system shall export typed errors for programmatic handling.

**Error Types:**
- ErrInvalidInput: Input validation failed
- ErrDuplicateReaction: User already has an active LIKE or DISLIKE on the Reaction Target (idempotency violation)
- ErrReactionNotFound: Requested reaction does not exist
- ErrStorageUnavailable: Database connection or query failed
- ErrInvalidReactionType: Unknown reaction type specified

**Duplicate Reaction Error Details:**
- **ErrDuplicateLike:** Returned when attempting to LIKE a Reaction Target that the user already LIKES
- **ErrDuplicateDislike:** Returned when attempting to DISLIKE a Reaction Target that the user already DISLIKES
- Both errors indicate that no state change occurred (idempotency preserved)
- The error shall include the Reaction Target identifier and current reaction state

**Requirements:**
- Errors shall be comparable using errors.Is()
- Error messages shall be descriptive and actionable
- Sensitive information shall not be exposed in error messages
- Duplicate reaction errors shall clearly indicate the existing reaction state

### Requirement 5: Configuration Types

**Description:** The system shall export configuration types for setup.

**Configuration Types:**
- DatabaseConfig: Database connection parameters
- PoolConfig: Connection pool settings
- ReactionConfig: Reaction type definitions and behaviors

**Requirements:**
- Configuration types shall use struct tags for validation
- Default values shall be provided where appropriate
- Configuration shall be immutable after client construction

### Requirement 6: Shutdown and Cleanup

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

### Requirement 7: Thread Safety

**Description:** The client shall be safe for concurrent use.

**Requirements:**
- Multiple goroutines may call API methods simultaneously
- No external synchronization required by callers
- Internal state shall be protected from race conditions
- Concurrent operations shall not corrupt data

### Requirement 8: Simple Configuration API

**Description:** The module shall expose a simple, intuitive API for users to configure and use.

**Requirements:**
- **Minimal Setup:** Basic configuration requires only essential parameters (storage connection)
- **Sensible Defaults:** All optional parameters have sensible defaults; zero-config for common cases
- **Fluent Interface:** Optional configuration uses builder pattern or functional options for readability
- **Single Entry Point:** One constructor/function to create a fully configured client
- **Clear Documentation:** Every configuration option is documented with examples
- **Validation on Build:** Configuration errors are caught at initialization time with clear messages

**Configuration Pattern:**
```go
// Simple - minimal configuration
client, err := golikeit.New(golikeit.Config{
    DatabaseURL: "postgres://user:pass@localhost/reactions",
})

// Advanced - with options
client, err := golikeit.New(
    golikeit.WithDatabaseURL("postgres://user:pass@localhost/reactions"),
    golikeit.WithAuditDatabaseURL("postgres://user:pass@localhost/audit"),
    golikeit.WithMaxConnections(100),
    golikeit.WithLogger(logger),
)
```

**Rationale:**
- Reduces cognitive load for new users
- Enables quick adoption with minimal boilerplate
- Supports advanced use cases through progressive disclosure
- Follows Go idioms for library design

### Requirement 9: Caching Layer

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
- Cache entries are invalidated on reaction state changes
- Write operations (Like, Unlike, Dislike, Undislike) trigger invalidation
- Bulk invalidation supported for entity-level cache clearing
- TTL-based expiration for automatic stale data removal

**Requirements:**
- Cache must be thread-safe for concurrent access
- Cache misses shall transparently fall back to database
- Cache hit/miss metrics shall be exposed
- Cache warming not required; lazy loading acceptable

**Rationale:**
- Reduces database load for read-heavy workloads
- Improves response times for frequently accessed data
- Critical for high-concurrency scenarios

### Requirement 10: Bulk Operations

**Description:** The API shall support bulk operations for efficient batch processing.

**Bulk Operations:**
- **GetUserReactionsBulk:** Retrieve reaction states for multiple entity targets for a single user
  - Input: user_id, slice of (entity_type, entity_id) tuples
  - Output: Map of entity target to reaction state

- **GetEntityCountsBulk:** Retrieve counts for multiple entity targets
  - Input: Slice of (entity_type, entity_id) tuples
  - Output: Map of entity target to reaction counts

- **GetMultipleUserReactions:** Retrieve reactions from multiple users for a single entity
  - Input: Slice of user_ids, entity_type, entity_id
  - Output: Map of user_id to reaction state

**Requirements:**
- Bulk operations shall minimize database round trips
- Bulk operations shall respect context cancellation
- Maximum batch size shall be enforced (configurable, default 100)
- Partial failures shall return successful results with error indicators
- Bulk operations shall leverage cache when available

### Requirement 11: Pagination Support

**Description:** All query operations that may return more than 50 records shall use a consistent pagination mechanism.

**Pagination Model:**
```go
type Pagination struct {
    Page    int // 1-based page number
    PerPage int // Items per page (max 100)
}

type PaginatedResult[T] struct {
    Items      []T   // Current page items
    Total      int64 // Total items across all pages
    Page       int   // Current page number
    PerPage    int   // Items per page
    TotalPages int   // Total number of pages
    HasNext    bool  // Whether there is a next page
    HasPrev    bool  // Whether there is a previous page
}
```

**Requirements:**
- **Threshold:** Pagination is mandatory for queries potentially returning >50 records
- **Default Page Size:** 20 items per page (configurable)
- **Maximum Page Size:** 100 items per page (enforced)
- **Page Numbering:** 1-based (first page is 1, not 0)
- **Consistent Ordering:** Results ordered by timestamp descending (newest first)
- **Total Count:** Include total item count for UI pagination controls
- **Cursor Support:** Optional cursor-based pagination for very large datasets (>10,000 items)

**Affected Operations:**
- GetUserReactions (user's reaction history)
- GetEntityReactions (all reactions on an entity)
- GetUserLikes (user's liked entities)
- GetUserDislikes (user's disliked entities)

**Rationale:**
- Prevents memory issues with large result sets
- Provides predictable performance
- Enables efficient UI pagination
- Consistent experience across all list queries

### Requirement 12: Fast Reaction Check Operations

**Description:** The API shall provide ultra-fast, efficient methods to check if a user has a specific reaction on a target.

**Fast Check Operations:**
- **HasUserLiked(ctx, user_id, entity_type, entity_id) (bool, error)**
  - Returns: true if user has LIKED the target, false otherwise
  - Optimized for speed; minimal database overhead
  - Uses cache when available
  - Single key lookup in most backends

- **HasUserDisliked(ctx, user_id, entity_type, entity_id) (bool, error)**
  - Returns: true if user has DISLIKED the target, false otherwise
  - Same optimization as HasUserLiked

**Performance Requirements:**
- Response time: <5ms (p95) when cached
- Response time: <10ms (p95) when not cached
- Single database lookup (no joins, no aggregations)
- Returns boolean only (no additional data fetched)

**Implementation Strategy:**
- **Cache-First:** Check in-memory cache before database
- **Key Design:** Use composite key (user_id:entity_type:entity_id) for O(1) lookups
- **Database:** Use primary key or covering index lookup
- **No Object Mapping:** Raw database value to boolean, no struct instantiation

**Use Cases:**
- UI button state ("already liked" indicator)
- Permission checks ("can like" validation)
- Batch processing requiring quick state verification
- Real-time features requiring fast feedback

**Rationale:**
- UI often needs to check if current user liked content (for heart icon state)
- GetUserReaction returns full state; HasUserLiked is optimized boolean check
- Critical for high-traffic scenarios with many concurrent users
- Reduces unnecessary data transfer

## Constraints and Limitations

1. **Library API Only:** The system provides a Go library API, not HTTP/gRPC endpoints. Network APIs are the responsibility of the consuming application.

2. **Go Version Compatibility:** The API requires Go 1.21 or later (generics, slog, etc.).

3. **No Callbacks:** The API does not support callback patterns; all operations are synchronous with context support.

4. **No Streaming:** Query operations return complete result sets; streaming/pagination is handled through limit/offset parameters.

5. **No Event Hooks:** The API does not provide event callbacks; observing changes is the responsibility of the consuming application.

6. **Simple Configuration Focus:** The API prioritizes simplicity over exhaustive configurability; edge cases may require custom wrappers.

7. **Cache Limitations:** Cache is optional and in-process only; distributed caching is not provided. Cache invalidation is the responsibility of the module when configured.

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
}

// Client construction
client, err := golikeit.New(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Operations
ctx := context.Background()

// User likes a photo (Reaction Target: type="photo", id="photo_456")
err = client.Like(ctx, "user_123", "photo", "photo_456")

// Attempting to like again returns ErrDuplicateReaction (idempotent behavior)
err = client.Like(ctx, "user_123", "photo", "photo_456")
// err == golikeit.ErrDuplicateReaction (or ErrDuplicateLike)

// Check reaction state for User Reaction (user_123 + photo:photo_456)
state, err := client.GetUserReaction(ctx, "user_123", "photo", "photo_456")
// state == golikeit.ReactionLike

// Get counts for Reaction Target
 counts, err := client.GetEntityCounts(ctx, "photo", "photo_456")
// counts.Likes, counts.Dislikes
```

## Relationships with Other Functional Blocks

- **[architecture.md](architecture.md):** Defines the layered architecture exposing the public API
- **[reaction_management.md](reaction_management.md):** Defines the operations available through the API
- **[data_persistence.md](data_persistence.md):** Defines the configuration for storage

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version of API interface specification |
| 2026-03-21 | Update | Added duplicate reaction error types for idempotency; updated examples to reflect Reaction Target and User Reaction concepts |
| 2026-03-21 | Update | Added Requirement 8 (Simple Configuration API) with fluent interface pattern |
| 2026-03-21 | Update | Added Requirement 9 (Caching Layer) and Requirement 10 (Bulk Operations) |
| 2026-03-21 | Update | Added GetUserLikes, GetUserDislikes, GetEntityReactionsWithUsers operations; added consolidated query operations with counts and recent users |
| 2026-03-21 | Update | Added Requirement 11 (Pagination Support) and Requirement 12 (Fast Reaction Check Operations) |

## Acceptance Criteria

1. **AC1:** All API methods accept context.Context as the first parameter
2. **AC2:** All API methods return (result, error) tuples
3. **AC3:** Exported error types allow comparison with errors.Is()
4. **AC4:** Client construction validates configuration and returns descriptive errors
5. **AC5:** Client is safe for concurrent use without external synchronization
6. **AC6:** Context cancellation terminates operations promptly
7. **AC7:** Context timeouts are respected and return timeout errors
8. **AC8:** Close() releases resources and is safe to call multiple times
9. **AC9:** Configuration types use struct tags and provide defaults
10. **AC10:** API is idiomatic Go (no unnecessary abstraction, clear naming)
11. **AC11:** Basic configuration requires only a database URL (one-liner setup)
12. **AC12:** Configuration uses functional options pattern or builder for extensibility
13. **AC13:** Configuration errors are caught at initialization with clear messages
14. **AC14:** All exported types have clear documentation with usage examples
15. **AC15:** Cache can be enabled/disabled via configuration with configurable TTL
16. **AC16:** Cache invalidation occurs on reaction state changes
17. **AC17:** Bulk operations minimize database round trips
18. **AC18:** Bulk operations enforce maximum batch size limits
19. **AC19:** GetUserLikes and GetUserDislikes return paginated entities with user reactions
20. **AC20:** GetEntityReactionsWithUsers returns consolidated data (counts + recent users) in single call
21. **AC21:** Recent users list is configurable (N users, default 10)
22. **AC22:** Pagination is used for queries potentially returning >50 records
23. **AC23:** Pagination uses consistent model (PaginatedResult[T]) across all operations
24. **AC24:** HasUserLiked and HasUserDisliked return boolean in <10ms
25. **AC25:** Fast check operations use single key lookup optimization
