# Architecture Specification

## Overview

The system is organized into distinct architectural layers with clear separation of concerns.

## Functional Requirements

### Requirement 1: Layer Organization

The system is organized into two primary layers.

**Business Layer:**
- Contains business logic and rules
- Validates input parameters including reaction types
- Orchestrates data layer operations
- Defines the public API
- Enforces single-reaction-per-user constraint

**Data Layer:**
- Handles all database interactions
- Abstracts database-specific implementations
- Provides CRUD operations
- Manages connections and transactions

**Communication Rules:**
- Business Layer may call Data Layer
- Data Layer does not call Business Layer
- External consumers interact only with Business Layer

### Requirement 2: Interface Contracts

Each layer exposes capabilities through interfaces.

**Business Layer Interface:**
- AddReaction(entity_type, entity_id, user_id, reaction_type) (isReplacement bool, error)
- RemoveReaction(entity_type, entity_id, user_id) error
- GetUserReaction(entity_type, entity_id, user_id) (reaction_type string, error)
- GetEntityReactionCounts(entity_type, entity_id) (counts map[string]int64, total int64, error)

**Data Layer Interface:**
- CreateReaction(user_id, entity_type, entity_id, reaction_type) error
- ReplaceReaction(user_id, entity_type, entity_id, new_type, previous_type) error
- DeleteReaction(user_id, entity_type, entity_id) error
- GetReaction(user_id, entity_type, entity_id) (reaction_type string, error)
- GetCountsByEntity(entity_type, entity_id) (map[string]int64, error)

### Requirement 3: Dependency Direction

Dependencies flow inward.

- Business Layer depends on Data Layer interfaces
- Data Layer has no dependencies on Business Layer
- External consumers depend on Business Layer

### Requirement 4: Configuration Management

Configuration is supported for both layers.

**Business Layer:**
- Reaction Types (required, minimum 1)
- Validation rules
- Behavior flags

**Data Layer:**
- Database type and connection parameters
- Connection pool settings

**Reaction Type Configuration:**
- Provided during initialization
- Format: `^[A-Z0-9_-]+$`
- Validated at startup
- Immutable after startup

### Requirement 5: Error Handling

Clear error types are defined.

**Error Categories:**
- Validation Errors
- Business Logic Errors
- Storage Errors
- Configuration Errors

### Requirement 6: Extensibility Points

Extension points are provided.

- New storage backends by implementing Data Layer interface
- Middleware/interceptors for cross-cutting concerns
- Custom validators in Business Layer

### Requirement 7: High Concurrency

The module is designed for high-load, high-concurrency environments.

**Concurrency Requirements:**
- Lock-free operations preferred
- Minimal lock scope when necessary
- No global locks
- Goroutine-safe components

**Load Requirements:**
- Horizontal scalability support
- Resource limits and backpressure
- Graceful degradation
- No memory leaks

### Requirement 8: Caching Layer

An optional caching layer is included.

**Responsibilities:**
- Cache user reaction states
- Cache entity counts per type
- Cache invalidation on writes
- Thread-safe operations

## Constraints and Limitations

1. **No Direct Database Access:** All access through Business Layer.
2. **No Business Logic in Data Layer:** Only storage logic.
3. **Synchronous Operations:** Layer interactions are synchronous.
4. **No Distributed Transactions:** Single database instance assumed.
5. **Reaction Type Immutability:** Cannot modify after initialization.
6. **Single Reaction Per User:** Fundamental constraint.

## Acceptance Criteria

1. **AC1:** Business Layer and Data Layer are distinct
2. **AC2:** Each layer exposes capabilities through interfaces
3. **AC3:** Dependencies flow inward
4. **AC4:** Data Layer can be swapped without modifying Business Layer
5. **AC5:** Business logic resides only in Business Layer
6. **AC6:** Error types allow programmatic handling
7. **AC7:** Configuration is passed during initialization
8. **AC8:** No global locks exist
9. **AC9:** Lock-free or minimal-lock patterns used
10. **AC10:** Connection pooling is properly implemented
11. **AC11:** Cache layer is optional
12. **AC12:** Reaction type configuration is validated at initialization
