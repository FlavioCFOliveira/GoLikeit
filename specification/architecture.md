# Architecture Specification

## Overview

The system shall be organized into distinct architectural layers with clear separation of concerns. Each layer has specific responsibilities and communicates with adjacent layers through well-defined interfaces. This layered architecture enables testability, maintainability, and flexibility in implementation choices.

## Functional Requirements

### Requirement 1: Layer Organization

**Description:** The system shall be organized into two primary layers: Business Layer and Data Layer.

**Layer Responsibilities:**

**Business Layer (Service Layer):**
- Contains business logic and rules
- Validates input parameters
- Orchestrates data layer operations
- Defines the public API surface
- Handles transaction coordination
- Enforces business constraints (e.g., mutual exclusivity of LIKE/DISLIKE)

**Data Layer (Repository Layer):**
- Handles all database interactions
- Abstracts database-specific implementations
- Provides CRUD operations for reaction entities
- Manages database connections and transactions
- Implements storage-specific optimizations

**Communication Rules:**
- The Business Layer may call the Data Layer
- The Data Layer shall not call the Business Layer
- External consumers interact only with the Business Layer
- Cross-layer communication occurs through interfaces

### Requirement 2: Interface Contracts

**Description:** Each layer shall expose its capabilities through well-defined interfaces.

**Business Layer Interface:**
- Define Like(entity_type, entity_id, user_id) error
- Define Unlike(entity_type, entity_id, user_id) error
- Define Dislike(entity_type, entity_id, user_id) error
- Define Undislike(entity_type, entity_id, user_id) error
- Define GetReaction(entity_type, entity_id, user_id) (ReactionType, error)
- Define GetEntityCounts(entity_type, entity_id) (Counts, error)
- Define GetUserReactions(user_id, filters) ([]Reaction, error)

**Data Layer Interface:**
- Define CreateReaction(reaction) error
- Define DeleteReaction(user_id, entity_type, entity_id) error
- Define UpdateReaction(reaction) error
- Define GetReaction(user_id, entity_type, entity_id) (Reaction, error)
- Define GetReactionsByUser(user_id, filters) ([]Reaction, error)
- Define GetReactionsByEntity(entity_type, entity_id) ([]Reaction, error)
- Define GetCountsByEntity(entity_type, entity_id) (Counts, error)
- Define IncrementCount(entity_type, entity_id, reaction_type) error
- Define DecrementCount(entity_type, entity_id, reaction_type) error

**Requirements:**
- Interfaces shall be language-idiomatic (Go interfaces)
- Interface methods shall have clear input and output contracts
- Error types shall be consistent and informative

### Requirement 3: Dependency Direction

**Description:** Dependencies shall flow inward, with inner layers having no knowledge of outer layers.

**Dependency Rules:**
- Business Layer depends on Data Layer interfaces
- Data Layer has no dependencies on Business Layer
- Data Layer implementations depend on database drivers
- External consumers depend on Business Layer interfaces

**Benefits:**
- Business logic is isolated from storage concerns
- Data Layer implementations can be swapped without affecting business logic
- Testing is simplified through interface mocking

### Requirement 4: Configuration Management

**Description:** The system shall support configuration for both layers.

**Business Layer Configuration:**
- Default reaction types (extensible list)
- Validation rules (identifier lengths, allowed characters)
- Behavior flags (e.g., strict mode vs. lenient mode)

**Data Layer Configuration:**
- Database type (PostgreSQL, MariaDB, SQLite)
- Connection parameters (host, port, credentials, database name)
- Connection pool settings
- Migration configuration

**Configuration Delivery:**
- Configuration shall be passed during initialization
- Environment variables or configuration files may be used by the consuming application
- The module shall accept configuration through code (not read files directly)

### Requirement 5: Error Handling

**Description:** The system shall define clear error types and handling patterns.

**Error Categories:**
- **Validation Errors:** Invalid input parameters
- **Business Logic Errors:** Constraint violations (e.g., duplicate LIKE)
- **Storage Errors:** Database connectivity or query failures
- **System Errors:** Unexpected internal failures

**Requirements:**
- Errors shall be typed to allow programmatic handling
- Error messages shall be clear and actionable
- Storage errors shall not expose sensitive information (credentials, internal paths)
- Business logic errors shall include context (which entity, which user)

### Requirement 6: Extensibility Points

**Description:** The architecture shall provide extension points for future capabilities.

**Extension Points:**
- New reaction types may be added without modifying core logic
- New storage backends may be added by implementing the Data Layer interface
- Middleware/interceptors may be added for cross-cutting concerns (logging, metrics)
- Custom validators may be plugged into the Business Layer

**Requirements:**
- Extension points shall use interfaces or function types
- Default implementations shall be provided for common cases
- Extensions shall not require modification of existing layer code

### Requirement 7: High Concurrency and Load Support

**Description:** The entire module, in all its components, shall be designed to operate without failures in high-load, high-concurrency environments. This is a critical requirement for all technical decisions.

### Requirement 8: Caching Layer

**Description:** The system shall include an optional caching layer between Business and Data layers.

**Cache Layer Responsibilities:**
- Cache user reaction states for fast lookups
- Cache entity reaction counts to reduce database load
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

## Layer Communication Flow

```
External Consumer
       |
       v
+------------------+
| Business Layer   | (Validates, orchestrates, enforces rules)
| - Input validation
| - Business logic
| - Transaction coordination
+------------------+
       |
       v
+------------------+
| Cache Layer      | (Optional - improves read performance)
| - Reaction state cache
| - Entity counts cache
| - Cache invalidation
+------------------+
       |
       v
+------------------+
| Data Layer       | (Abstracts storage, executes queries)
| - CRUD operations
| - Database-specific code
| - Connection management
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

## Acceptance Criteria

1. **AC1:** The Business Layer and Data Layer are distinct with clear responsibilities
2. **AC2:** Each layer exposes capabilities through well-defined interfaces
3. **AC3:** Dependencies flow inward (Business → Data, not Data → Business)
4. **AC4:** The Data Layer can be swapped without modifying Business Layer code
5. **AC5:** Business logic resides only in the Business Layer
6. **AC6:** Storage-specific code resides only in the Data Layer
7. **AC7:** Error types allow programmatic handling of different error categories
8. **AC8:** Configuration is passed during initialization, not read from files
9. **AC9:** Extension points exist for new reaction types and storage backends
10. **AC10:** External consumers interact only with the Business Layer
11. **AC11:** No global locks exist that would block all concurrent operations
12. **AC12:** Lock-free or minimal-lock patterns are used for high-concurrency paths
13. **AC13:** Connection pooling is properly implemented and configured
14. **AC14:** Race condition testing passes under high concurrent load
15. **AC15:** Cache layer is optional and can be disabled
16. **AC16:** Cache invalidation occurs on write operations
17. **AC17:** Cache is thread-safe for concurrent access
