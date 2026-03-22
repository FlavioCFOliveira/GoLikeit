# Architecture Specification

## Overview

The system shall be organized into distinct architectural layers with clear separation of concerns. Each layer has specific responsibilities and communicates with adjacent layers through well-defined interfaces. This layered architecture enables testability, maintainability, and flexibility in implementation choices.

## Functional Requirements

### Requirement 1: Layer Organization

**Description:** The system shall be organized into two primary layers: Business Layer and Data Layer.

**Layer Responsibilities:**

**Business Layer (Service Layer):**
- Contains business logic and rules
- Validates input parameters including reaction types against configured registry
- Orchestrates data layer operations
- Defines the public API surface
- Handles transaction coordination
- Enforces single-reaction-per-user constraint
- Manages reaction type configuration at initialization

**Data Layer (Repository Layer):**
- Handles all database interactions
- Abstracts database-specific implementations
- Provides CRUD operations for reaction entities
- Manages database connections and transactions
- Implements storage-specific optimizations
- Supports reaction type-agnostic storage (reaction type stored as string)

**Communication Rules:**
- The Business Layer may call the Data Layer
- The Data Layer shall not call the Business Layer
- External consumers interact only with the Business Layer
- Cross-layer communication occurs through interfaces

### Requirement 2: Interface Contracts

**Description:** Each layer shall expose its capabilities through well-defined interfaces.

**Business Layer Interface:**
- Define AddReaction(entity_type, entity_id, user_id, reaction_type) (isReplacement bool, error)
- Define RemoveReaction(entity_type, entity_id, user_id) error
- Define GetUserReaction(entity_type, entity_id, user_id) (reaction_type string, error)
- Define GetEntityReactionCounts(entity_type, entity_id) (counts map[string]int64, total int64, error)
- Define GetUserReactions(user_id, filters) ([]Reaction, error)
- Define HasUserReaction(entity_type, entity_id, user_id) (bool, error)
- Define HasUserReactionType(entity_type, entity_id, user_id, reaction_type) (bool, error)

**Data Layer Interface:**
- Define CreateReaction(user_id, entity_type, entity_id, reaction_type) error
- Define ReplaceReaction(user_id, entity_type, entity_id, new_reaction_type, previous_reaction_type) error
- Define DeleteReaction(user_id, entity_type, entity_id) error
- Define GetReaction(user_id, entity_type, entity_id) (reaction_type string, error)
- Define GetReactionsByUser(user_id, filters) ([]Reaction, error)
- Define GetReactionsByEntity(entity_type, entity_id) ([]Reaction, error)
- Define GetCountsByEntity(entity_type, entity_id) (map[string]int64, error)
- Define IncrementCount(entity_type, entity_id, reaction_type) error
- Define DecrementCount(entity_type, entity_id, reaction_type) error
- Define UpdateCountForReplacement(entity_type, entity_id, old_type, new_type) error

**Requirements:**
- Interfaces shall be language-idiomatic (Go interfaces)
- Interface methods shall have clear input and output contracts
- Error types shall be consistent and informative
- Reaction types are passed as strings (not enums) to support configuration

### Requirement 3: Dependency Direction

**Description:** Dependencies shall flow inward, with inner layers having no knowledge of outer layers.

**Dependency Rules:**
- Business Layer depends on Data Layer interfaces
- Data Layer has no dependencies on Business Layer
- Data Layer implementations depend on database drivers
- External consumers depend on Business Layer interfaces
- Reaction type configuration is passed downward from initialization

**Benefits:**
- Business logic is isolated from storage concerns
- Data Layer implementations can be swapped without affecting business logic
- Testing is simplified through interface mocking
- Reaction types are configurable without code changes

### Requirement 4: Configuration Management

**Description:** The system shall support configuration for both layers including reaction type definitions.

**Business Layer Configuration:**
- **Reaction Types (Required):** List of reaction types supported by the module (minimum 1)
- Validation rules (identifier lengths, allowed characters)
- Behavior flags (e.g., strict mode vs. lenient mode)

**Data Layer Configuration:**
- Database type (PostgreSQL, MariaDB, SQLite, MongoDB, Cassandra, Redis, In-Memory)
- Connection parameters (host, port, credentials, database name)
- Connection pool settings
- Migration configuration

**Reaction Type Configuration:**
- Reaction types must be provided during module initialization
- Validation occurs at startup - module fails if invalid
- Format: `^[A-Z0-9_-]+$` (uppercase letters, digits, hyphens, underscores)
- No reaction types can be added after initialization
- Configuration is immutable after startup

**Configuration Delivery:**
- Configuration shall be passed during initialization
- Environment variables or configuration files may be used by the consuming application
- The module shall accept configuration through code (not read files directly)
- Reaction type list is validated before module becomes operational

### Requirement 5: Error Handling

**Description:** The system shall define clear error types and handling patterns.

**Error Categories:**
- **Validation Errors:** Invalid input parameters, invalid reaction types
- **Business Logic Errors:** Constraint violations (e.g., removing non-existent reaction)
- **Storage Errors:** Database connectivity or query failures
- **System Errors:** Unexpected internal failures
- **Configuration Errors:** Invalid reaction types, empty type list

**Requirements:**
- Errors shall be typed to allow programmatic handling
- Error messages shall be clear and actionable
- Storage errors shall not expose sensitive information (credentials, internal paths)
- Business logic errors shall include context (which entity, which user)
- Configuration errors occur at initialization and prevent module startup

### Requirement 6: Extensibility Points

**Description:** The architecture shall provide extension points for future capabilities.

**Extension Points:**
- New storage backends may be added by implementing the Data Layer interface
- Middleware/interceptors may be added for cross-cutting concerns (logging, metrics)
- Custom validators may be plugged into the Business Layer
- Reaction types are defined by configuration (not code)

**Requirements:**
- Extension points shall use interfaces or function types
- Default implementations shall be provided for common cases
- Extensions shall not require modification of existing layer code
- Reaction types are configured, not hardcoded

### Requirement 7: High Concurrency and Load Support

**Description:** The entire module, in all its components, shall be designed to operate without failures in high-load, high-concurrency environments. This is a critical requirement for all technical decisions.

### Requirement 8: Caching Layer

**Description:** The system shall include an optional caching layer between Business and Data layers.

**Cache Layer Responsibilities:**
- Cache user reaction states for fast lookups (key includes reaction type)
- Cache entity reaction counts per type to reduce database load
- Implement cache invalidation on write operations
- Provide thread-safe cache operations

**Cache Configuration:**
- Cache is optional (can be disabled)
- TTL (time-to-live) configurable per entry type
- Maximum size with LRU eviction
- Metrics for hit/miss rates

**Integration:**
- Business Layer checks cache before Data Layer
- Cache misses trigger Data Layer queries
- Write operations invalidate relevant cache entries
- Cache is transparent to API consumers

**Rationale:**
- Improves read performance for frequently accessed data
- Reduces database load in high-traffic scenarios
- Supports high concurrency requirements

**Concurrency Requirements:**
- **Lock-Free Operations:** Prefer lock-free algorithms over mutex-based synchronization where possible
- **Minimal Lock Scope:** When locks are necessary, they must be held for the shortest possible duration
- **No Global Locks:** No operation shall require a global lock that blocks all concurrent operations
- **Connection Pooling:** Database connections must be pooled and efficiently shared
- **Goroutine Safety:** All components must be safe for concurrent access from multiple goroutines

**Load Requirements:**
- **Horizontal Scalability:** Design must support horizontal scaling (multiple application instances)
- **Resource Limits:** Clear limits and backpressure mechanisms to prevent resource exhaustion
- **Graceful Degradation:** Under extreme load, the system degrades gracefully rather than failing
- **No Memory Leaks:** All goroutines, connections, and resources are properly managed and released

**Performance Critical Design:**
- Database queries must be optimized with proper indexes
- N+1 queries are strictly prohibited
- Batch operations are preferred over individual operations where applicable
- Caching strategies must consider cache invalidation in concurrent scenarios

**Critical Decision Criteria:**
- Every design decision must consider its impact on concurrency
- Every implementation must be reviewed for race conditions
- Every database operation must consider query performance under load
- Every resource allocation must have a corresponding release mechanism

**Rationale:**
- The module is intended for production systems with high user concurrency
- Reaction operations are frequent and must not become bottlenecks
- Race conditions in reaction counts are unacceptable (data integrity)
- Performance under load is a primary quality attribute

## Constraints and Limitations

1. **No Direct Database Access:** External consumers shall not access the Data Layer directly; all access goes through the Business Layer.

2. **No Business Logic in Data Layer:** The Data Layer shall contain only storage logic, not business rules or validation.

3. **Synchronous Operations:** All layer interactions are synchronous; asynchronous patterns are the responsibility of the consuming application.

4. **No Distributed Transactions:** The system assumes a single database instance; distributed transaction coordination is not provided.

5. **No Automatic Retry:** Failed operations are reported to the caller; retry logic is the responsibility of the consuming application.

6. **High Concurrency Required:** All design decisions must prioritize concurrent safety; global locks and long-held locks are prohibited.

7. **Caching Layer:** Optional cache layer improves read performance but adds complexity to consistency management.

8. **Reaction Type Immutability:** Reaction types cannot be modified after initialization. The module must be restarted to change supported reaction types.

9. **Single Reaction Per User:** The architecture enforces that each user can have only one reaction per target. This is a fundamental constraint of the data model.

## Layer Communication Flow

```
External Consumer
       |
       v
+------------------+
| Business Layer   | (Validates, orchestrates, enforces rules)
| - Input validation
| - Reaction type validation against configured registry
| - Business logic (single reaction per user)
| - Transaction coordination
+------------------+
       |
       v
+------------------+
| Cache Layer      | (Optional - improves read performance)
| - Reaction state cache
| - Entity counts cache (per reaction type)
| - Cache invalidation
+------------------+
       |
       v
+------------------+
| Data Layer       | (Abstracts storage, executes queries)
| - CRUD operations
| - Database-specific code
| - Connection management
| - Atomic replacement operations
+------------------+
       |
       v
   Database
```

## Relationships with Other Functional Blocks

- **[reaction_management.md](reaction_management.md):** Defines the business logic implemented in the Business Layer
- **[data_persistence.md](data_persistence.md):** Defines the storage capabilities implemented in the Data Layer
- **[api_interface.md](api_interface.md):** Defines how external consumers interact with the Business Layer

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version of architecture specification |
| 2026-03-21 | Update | Added Requirement 7 (High Concurrency and Load Support) as critical design constraint |
| 2026-03-21 | Update | Added Requirement 8 (Caching Layer) to architecture |
| 2026-03-22 | Major | Updated for abstract reaction model - reaction types are configured, not hardcoded; added single-reaction-per-user constraint; updated interfaces to use string reaction types |

## Acceptance Criteria

1. **AC1:** The Business Layer and Data Layer are distinct with clear responsibilities
2. **AC2:** Each layer exposes capabilities through well-defined interfaces
3. **AC3:** Dependencies flow inward (Business → Data, not Data → Business)
4. **AC4:** The Data Layer can be swapped without modifying Business Layer code
5. **AC5:** Business logic resides only in the Business Layer
6. **AC6:** Storage-specific code resides only in the Data Layer
7. **AC7:** Error types allow programmatic handling of different error categories
8. **AC8:** Configuration is passed during initialization, not read from files
9. **AC9:** Extension points exist for storage backends and validators
10. **AC10:** External consumers interact only with the Business Layer
11. **AC11:** No global locks exist that would block all concurrent operations
12. **AC12:** Lock-free or minimal-lock patterns are used for high-concurrency paths
13. **AC13:** Connection pooling is properly implemented and configured
14. **AC14:** Race condition testing passes under high concurrent load
15. **AC15:** Cache layer is optional and can be disabled
16. **AC16:** Cache invalidation occurs on write operations
17. **AC17:** Cache is thread-safe for concurrent access
18. **AC18:** Reaction type configuration is validated at initialization
19. **AC19:** Module fails to initialize if reaction type configuration is invalid
20. **AC20:** Interfaces support any reaction type defined by configuration
