# Reaction Management Specification

## Overview

The system provides abstract reaction management capabilities. The module is reaction-type agnostic - no reaction types are predefined. Instead, reaction types are defined by the consuming application during module initialization through configuration.

Each user may have exactly one active reaction per Reaction Target. When a user adds a reaction to a target that already has a reaction from that user, the previous reaction is replaced.

## Module Scope and Boundaries

### Exclusive Responsibility: User Reactions

This module handles **exclusively** user reactions. It is responsible for recording, managing, and querying which reactions users have expressed toward entities.

### Out of Scope

- **Authentication:** User identity verification is handled by the consuming application.
- **Reaction Target Management:** The creation, validation, or lifecycle of reaction targets is outside this module's scope.
- **Authorization:** Access control policies are enforced by the consuming application before calling this module.
- **Entity Metadata:** Any metadata about the entities being reacted to is not stored or managed by this module.
- **Offline Operations:** The module assumes all operations are performed synchronously against available storage.
- **Reaction Metadata:** Additional metadata associated with reactions is not stored by this module.
- **Circuit Breaker Pattern:** Failure handling strategies are the responsibility of the consuming application.
- **Observability and Monitoring:** Metrics collection, distributed tracing, and health checks are the responsibility of the consuming application.
- **Sharding and Horizontal Partitioning:** Data sharding is not supported by this module.
- **Multi-Tenancy:** Native multi-tenancy support is not provided by this module.

### Module Responsibilities

- Mechanisms to persist user reactions to storage
- Mechanisms to query reactions (individual and aggregated)
- Atomic operations ensuring data consistency
- Audit logging of all reaction operations
- Caching capabilities for improved read performance

## Core Concepts

### Reaction Target

A **Reaction Target** is the unique combination of an entity type and an entity instance identifier that identifies what is being reacted to. It is represented as a tuple `(entity_type, entity_id)`.

**Properties:**
- Uniquely identifies a specific piece of content within the system
- Serves as the target for all reaction operations
- Examples: `("photo", "123")`, `("article", "abc-456")`, `("comment", "789")`

**Constraints:**
- Both `entity_type` and `entity_id` are required components
- The combination must be unique within the reaction system
- Different Reaction Targets are completely isolated

### User Reaction

A **User Reaction** is the unique combination of a user identifier and a Reaction Target. It is represented as a tuple `(user_id, entity_type, entity_id)`.

**Properties:**
- Represents the binding between a user and a reaction target
- At most **one** active reaction can exist per User Reaction at any time
- Uniquely identifies a reaction record in the system

**Constraints:**
- Only **one** reaction can exist per User Reaction at any time
- Absence of reaction is represented by the absence of a record

### Reaction Type

A **Reaction Type** is a string identifier representing a specific type of reaction.

**Format:**
- Pattern: `^[A-Z0-9_-]+$`
- Uppercase letters, digits, hyphens, and underscores only
- Length: 1-64 characters

**Examples:** `LIKE`, `DISLIKE`, `LOVE`, `ANGRY`, `THUMBS_UP`, `STAR_5`

**Configuration:**
- Reaction types must be defined during module initialization
- Validation occurs at startup - if any reaction type fails validation, the module cannot be initialized
- No reaction types can be added during runtime

## Functional Requirements

### Requirement 1: Reaction Type Configuration

The system accepts reaction type definitions during module initialization and validates them before the module becomes operational.

**Inputs:**
- List of reaction type strings to be supported by the module

**Validation:**
- Each reaction type must match pattern `^[A-Z0-9_-]+$`
- Each reaction type must be 1-64 characters
- Reaction type list must not be empty
- Duplicate reaction types are not allowed

**Outputs:**
- Validated reaction type registry
- Initialization error if validation fails

**Error Cases:**
- Reject reaction types with invalid format
- Reject empty reaction type lists
- Reject duplicate reaction types
- Module fails to initialize if validation fails

### Requirement 2: Add or Replace Reaction

The system allows a user to add a reaction to a Reaction Target. If the user already has a reaction on that target, the previous reaction is replaced.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)
- Reaction type (must be in the configured registry)

**Outputs:**
- Confirmation of successful reaction operation
- Indication of whether a previous reaction was replaced
- Current count per reaction type for the Reaction Target

**Error Cases:**
- Reject unregistered reaction types
- Reject invalid user identifiers
- Reject invalid Reaction Target identifiers
- Reject attempts when storage is unavailable

**Behavior:**
- When a user adds a reaction:
  - If no previous reaction exists: create new reaction record
  - If a previous reaction exists: replace it with the new reaction type
- The previous reaction is permanently replaced
- Reaction counts per type are updated atomically

### Requirement 3: Remove Reaction

The system allows a user to remove their current reaction from a Reaction Target.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Confirmation of successful removal
- Updated reaction counts per type for the Reaction Target

**Error Cases:**
- Reject removal attempts when no reaction exists
- Reject invalid user identifiers
- Reject invalid Reaction Target identifiers

**Behavior:**
- RemoveReaction permanently deletes the reaction record (hard delete)
- Historical record is maintained in the audit log

### Requirement 4: Query User Reaction

The system supports querying the current reaction type for a specific User Reaction.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Current reaction type (if any)
- Error if no reaction exists

### Requirement 5: Query Entity Reaction Counts

The system provides aggregated counts of reactions per type for a Reaction Target.

**Inputs:**
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Map of reaction type to count
- Total number of reactions
- Timestamp of last reaction (optional)

## Constraints and Limitations

1. **Single Reaction Per User:** A user may have at most **one** active reaction per Reaction Target. Adding a new reaction replaces any existing reaction.

2. **No Revert:** When a reaction is replaced, the previous reaction type is permanently lost.

3. **Configuration-Time Only:** Reaction types can only be defined during module initialization.

4. **Format Constraint:** Reaction types must match pattern `^[A-Z0-9_-]+$`.

5. **Strict Validation:** Module initialization fails if any reaction type fails validation.

6. **Hard Delete:** RemoveReaction permanently deletes reaction records.

7. **Atomicity:** Operations that modify User Reactions and Reaction Target counts are atomic.

8. **Reaction Target Isolation:** Reactions on different Reaction Targets are completely independent.

## Relationships with Other Functional Blocks

- **[entity_association.md](entity_association.md):** Reaction Target definition
- **[user_interactions.md](user_interactions.md):** User-centric behaviors
- **[data_persistence.md](data_persistence.md):** Storage and retrieval
- **[api_interface.md](api_interface.md):** Public interface
- **[audit_logging.md](audit_logging.md):** Audit logging

## Acceptance Criteria

1. **AC1:** Reaction type configuration is validated at startup with pattern `[A-Z0-9_-]+`
2. **AC2:** Module initialization fails if any reaction type has invalid format
3. **AC3:** Module initialization fails if reaction type list is empty
4. **AC4:** A user can successfully add a reaction to a Reaction Target
5. **AC5:** Adding a reaction when none exists creates a new reaction record
6. **AC6:** Adding a reaction when one exists replaces the previous reaction
7. **AC7:** Adding the same reaction type that already exists returns success (idempotent)
8. **AC8:** A user can successfully remove their current reaction
9. **AC9:** Attempting to remove a reaction when none exists returns an error
10. **AC10:** Reaction Target counts accurately reflect the number of reactions per type
11. **AC11:** Invalid inputs are rejected with clear error indicators
12. **AC12:** Replacement operations are atomic
13. **AC13:** Querying a User Reaction returns the current reaction type or "not found"
14. **AC14:** Querying Entity Reaction Counts returns accurate counts per configured reaction type
