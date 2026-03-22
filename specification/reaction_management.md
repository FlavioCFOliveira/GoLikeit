# Reaction Management Specification

## Overview

The system shall provide abstract reaction management capabilities, allowing users to express sentiment toward any entity in the system through configurable reaction types. The module is reaction-type agnostic - no reaction types are predefined. Instead, reaction types are defined by the consuming application during module initialization through configuration.

Each user may have exactly one active reaction per Reaction Target. When a user adds a reaction to a target that already has a reaction from that user, the previous reaction is replaced (upsert behavior). There is no concept of "undo" or "revert" to a previous state - reactions can only be deleted or replaced.

## Module Scope and Boundaries

### Exclusive Responsibility: User Reactions

This module handles **EXCLUSIVELY** user reactions. It is responsible for recording, managing, and querying which reactions users have expressed toward entities.

### Out of Scope

The following concerns are **NOT** the responsibility of this module:

1. **Authentication:** User identity verification is handled by the consuming application. This module receives user identifiers as opaque strings and does not validate their authenticity.

2. **Reaction Target Management:** The creation, validation, or lifecycle of reaction targets (entities being reacted to) is outside this module's scope. The module accepts entity_type and entity_id as opaque identifiers without validating their existence or managing their state.

3. **Authorization:** Access control policies (e.g., "user A cannot react to entity B") are enforced by the consuming application before calling this module.

4. **Entity Metadata:** Any metadata about the entities being reacted to (titles, descriptions, ownership) is not stored or managed by this module.

5. **Offline Operations:** The module assumes all operations are performed synchronously against available storage. Offline operation queuing, conflict resolution, and synchronization are the responsibility of the consuming application.

6. **Reaction Metadata:** Additional metadata associated with reactions (context, location, device info, timestamps, custom fields) is not stored by this module. The consuming application is responsible for storing any contextual information about reactions if needed.

7. **Circuit Breaker Pattern:** Failure handling strategies such as circuit breakers, retries with backoff, and graceful degradation are the responsibility of the consuming application. This module returns errors immediately when storage is unavailable and does not implement circuit breaker patterns.

8. **Observability and Monitoring:** Metrics collection (latency, throughput, operation counts), distributed tracing, structured logging, and health checks are the responsibility of the consuming application. This module does not expose observability endpoints or integrate with monitoring systems.

9. **Sharding and Horizontal Partitioning:** Data sharding, horizontal partitioning, and distributed database strategies are not supported by this module. For massive scale deployments (billions of reactions), horizontal scaling must be implemented by the consuming application using multiple module instances or external sharding solutions.

10. **Multi-Tenancy:** Native multi-tenancy support (tenant isolation, schema-per-tenant, row-level tenant filtering) is not provided by this module. The module operates in single-tenant mode. SaaS applications requiring multi-tenancy must implement tenant isolation at the application layer (e.g., by prefixing entity_id with tenant_id) or use separate module instances per tenant.

### Module Responsibilities

This module provides:
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
- Different Reaction Targets are completely isolated (reactions on one do not affect another)

### User Reaction

A **User Reaction** is the unique combination of a user identifier and a Reaction Target, representing a specific user's reaction to a specific target. It is represented as a tuple `(user_id, entity_type, entity_id)` or equivalently `(user_id, ReactionTarget)`.

**Properties:**
- Represents the binding between a user and a reaction target
- At most **ONE** active reaction can exist per User Reaction at any time
- Uniquely identifies a reaction record in the system
- The reaction type is stored as the value of this unique key

**Constraints:**
- Only **ONE** reaction can exist per User Reaction at any time (not multiple)
- Absence of reaction is not represented as a state; it is the absence of a record
- User Reactions are immutable in terms of identity; the reaction type can be replaced

### Reaction Type

A **Reaction Type** is a string identifier representing a specific type of reaction. The module does not define any built-in reaction types - all types are provided through configuration.

**Format:**
- Reaction types shall match the pattern: `^[A-Z0-9_-]+$`
- Must be uppercase letters, digits, hyphens, and underscores only
- Minimum length: 1 character
- Maximum length: 64 characters

**Valid Examples:** `LIKE`, `DISLIKE`, `LOVE`, `ANGRY`, `THUMBS_UP`, `STAR_5`

**Invalid Examples:** `like` (lowercase), `Love` (mixed case), `angry!!` (special chars), `very-long-reaction-type-name-exceeding-limit` (too long)

**Configuration:**
- Reaction types must be defined during module initialization
- Validation occurs at startup - if any reaction type fails validation, the module cannot be initialized
- No reaction types can be added during runtime
- The module is agnostic to the semantic meaning of reaction types

## Functional Requirements

### Requirement 1: Reaction Type Configuration

**Description:** The system shall accept reaction type definitions during module initialization and validate them before the module becomes operational.

**Inputs:**
- List of reaction type strings to be supported by the module

**Validation:**
- Each reaction type must match pattern `^[A-Z0-9_-]+$`
- Each reaction type must be between 1 and 64 characters
- Reaction type list must not be empty (at least one type required)
- Duplicate reaction types are not allowed

**Outputs:**
- Validated reaction type registry
- Initialization error if validation fails

**Error Cases:**
- The system shall reject reaction types with invalid format
- The system shall reject empty reaction type lists
- The system shall reject duplicate reaction types
- The system shall reject reaction types exceeding 64 characters
- The module shall fail to initialize if validation fails

**Behavior:**
- Validation occurs during module initialization
- All reaction types are validated before any operation
- If any reaction type fails validation, the module cannot be initialized
- Validated reaction types are immutable after initialization

### Requirement 2: Add or Replace Reaction

**Description:** The system shall allow a user to add a reaction to a Reaction Target. If the user already has a reaction on that target, the previous reaction is replaced with the new one.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)
- Reaction type (must be in the configured reaction type registry)

**Outputs:**
- Confirmation of successful reaction operation
- Indication of whether a previous reaction was replaced
- Current reaction type for the User Reaction
- Current count per reaction type for the Reaction Target

**Error Cases:**
- The system shall reject unregistered reaction types
- The system shall reject invalid user identifiers
- The system shall reject invalid Reaction Target identifiers
- The system shall reject attempts when storage is unavailable

**Behavior:**
- When a user adds a reaction to a Reaction Target:
  - If no previous reaction exists: create new reaction record
  - If a previous reaction exists: replace it with the new reaction type
- The previous reaction is permanently replaced (not revertible)
- Reaction counts per type are updated atomically
- The operation is atomic - either fully succeeds or fully fails

**Replacement Semantics:**
- Adding reaction type A when reaction type B exists: B is deleted, A is created
- Adding the same reaction type that already exists: no state change, returns success (idempotent)
- Replacement is not "undoable" - the previous reaction type is lost

### Requirement 3: Remove Reaction

**Description:** The system shall allow a user to remove their current reaction from a Reaction Target.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Confirmation of successful removal
- Updated reaction counts per type for the Reaction Target
- Current reaction state for the User Reaction (empty/nil after removal)

**Error Cases:**
- The system shall reject removal attempts when no reaction exists for the User Reaction
- The system shall reject invalid user identifiers
- The system shall reject invalid Reaction Target identifiers

**Behavior:**
- When a user removes a reaction from a Reaction Target, the system shall delete the reaction record
- The system shall update Reaction Target counts accordingly
- Removal shall only succeed if a reaction exists for the User Reaction
- Attempting to remove when no reaction exists shall result in an error
- **Hard Delete:** Removal permanently deletes the reaction record from storage
- **History Preservation:** Historical record of the reaction is maintained in the audit log

### Requirement 4: Query User Reaction

**Description:** The system shall support querying the current reaction type for a specific User Reaction.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Current reaction type (if any)
- Error if no reaction exists

**Behavior:**
- Returns the current reaction type for the User Reaction
- Returns "not found" error if no reaction exists
- Query is read-only and does not modify state

### Requirement 5: Query Entity Reaction Counts

**Description:** The system shall provide aggregated counts of reactions per type for a Reaction Target.

**Inputs:**
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Map of reaction type to count
- Total number of reactions (sum of all types)
- Timestamp of last reaction (optional)

**Requirements:**
- Counts shall reflect current active reactions only
- Counts shall be accurate and consistent with the reaction data
- Results include counts for all configured reaction types (zero for types with no reactions)

## Constraints and Limitations

1. **Scope Limitation:** This module exclusively handles user reactions. Authentication, authorization, and reaction target management are the responsibility of the consuming application.

2. **Single Reaction Per User:** A user may have at most **ONE** active reaction per Reaction Target at any given time. Adding a new reaction automatically replaces any existing reaction.

3. **No Revert:** When a reaction is replaced, the previous reaction type is permanently lost. There is no mechanism to "undo" a replacement or revert to a previous state.

4. **Configuration-Time Only:** Reaction types can only be defined during module initialization. No reaction types can be added, removed, or modified at runtime.

5. **Format Constraint:** Reaction types must match pattern `^[A-Z0-9_-]+$`. No lowercase letters, no special characters, no spaces.

6. **Strict Validation:** Module initialization fails if any reaction type fails validation. The module cannot operate with invalid configuration.

7. **Hard Delete:** Removal operations perform hard deletes on reaction records. The reaction data is permanently removed from the reactions table. Historical records of removed reactions are available exclusively through the audit log.

8. **Atomicity:** Operations that modify User Reactions and Reaction Target counts shall be atomic. Partial failures shall not leave the system in an inconsistent state.

9. **Validation:** All inputs shall be validated before processing. Invalid inputs shall result in immediate rejection with clear error indicators.

10. **Reaction Target Isolation:** Reactions on different Reaction Targets are completely independent. Operations on one Reaction Target never affect another.

## Relationships with Other Functional Blocks

- **[entity_association.md](entity_association.md):** Defines how reactions associate with entity types and identifiers
- **[user_interactions.md](user_interactions.md):** Defines user-centric behaviors
- **[data_persistence.md](data_persistence.md):** Defines how reaction states are stored and retrieved
- **[api_interface.md](api_interface.md):** Defines the public interface for invoking these operations
- **[audit_logging.md](audit_logging.md):** Defines the audit logging of all reaction operations

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version with LIKE/UNLIKE/DISLIKE/UNDISLIKE model |
| 2026-03-22 | Major | Refactored to abstract reaction model - removed fixed reaction types, added configuration-driven types, single reaction per user, replacement semantics |

## Acceptance Criteria

1. **AC1:** The system accepts reaction type configuration during initialization and validates format `[A-Z0-9_-]+`
2. **AC2:** Module initialization fails if any reaction type has invalid format
3. **AC3:** Module initialization fails if reaction type list is empty
4. **AC4:** A user can successfully add a reaction to a Reaction Target
5. **AC5:** Adding a reaction when none exists creates a new reaction record
6. **AC6:** Adding a reaction when one exists replaces the previous reaction
7. **AC7:** The replaced reaction is permanently removed (not revertible)
8. **AC8:** Adding the same reaction type that already exists returns success with no state change (idempotent)
9. **AC9:** A user can successfully remove their current reaction
10. **AC10:** Attempting to remove a reaction when none exists returns an error
11. **AC11:** Reaction Target counts accurately reflect the number of reactions per type
12. **AC12:** Invalid inputs (unregistered reaction types, invalid user/target) are rejected with clear error indicators
13. **AC13:** Replacement operations are atomic - either fully succeed or fully fail
14. **AC14:** Querying a User Reaction returns the current reaction type or "not found"
15. **AC15:** Querying Entity Reaction Counts returns accurate counts per configured reaction type
