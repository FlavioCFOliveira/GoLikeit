# Reaction Management Specification

## Overview

The system shall provide core reaction management capabilities, allowing users to express sentiment (positive or negative) toward any entity in the system. The module supports four primary operations: LIKE, UNLIKE, DISLIKE, and UNDISLIKE, with an extensible design for future reaction types.

## Module Scope and Boundaries

### Exclusive Responsibility: User Reactions

This module handles **EXCLUSIVELY** user reactions. It is responsible for recording, managing, and querying which reactions (LIKE/DISLIKE) users have expressed toward entities.

### Out of Scope

The following concerns are **NOT** the responsibility of this module:

1. **Authentication:** User identity verification is handled by the consuming application. This module receives user identifiers as opaque strings and does not validate their authenticity.

2. **Reaction Target Management:** The creation, validation, or lifecycle of reaction targets (entities being reacted to) is outside this module's scope. The module accepts entity_type and entity_id as opaque identifiers without validating their existence or managing their state.

3. **Authorization:** Access control policies (e.g., "user A cannot like entity B") are enforced by the consuming application before calling this module.

4. **Entity Metadata:** Any metadata about the entities being reacted to (titles, descriptions, ownership) is not stored or managed by this module.

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
- At most one active reaction (LIKE or DISLIKE) can exist per User Reaction
- Uniquely identifies a reaction record in the system

**Constraints:**
- Only one reaction state (LIKE, DISLIKE, or NONE) can exist per User Reaction
- User Reactions are immutable in terms of identity; only the reaction type can change

## Functional Requirements

### Requirement 1: Like an Entity

**Description:** The system shall allow a user to register a positive reaction (LIKE) toward a Reaction Target.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Confirmation of successful like registration
- Current like count for the Reaction Target
- Current reaction state for the User Reaction

**Error Cases:**
- The system shall reject duplicate LIKE attempts for the same User Reaction (idempotency violation)
- The system shall reject invalid user identifiers
- The system shall reject invalid Reaction Target identifiers
- The system shall reject attempts when storage is unavailable

**Behavior:**
- When a user LIKES a Reaction Target, the system shall record the association
- If the user previously DISLIKED the Reaction Target, the DISLIKE shall be removed and replaced with LIKE
- The system shall maintain accurate counts of all reactions per Reaction Target
- **Idempotency:** If a LIKE already exists for the User Reaction, the operation shall return an error indicating a duplicate reaction attempt, and no state change shall occur

### Requirement 2: Unlike an Entity

**Description:** The system shall allow a user to remove a previously registered LIKE from a Reaction Target.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Confirmation of successful unlike operation
- Updated like count for the Reaction Target
- Current reaction state for the User Reaction (none if no reactions remain)

**Error Cases:**
- The system shall reject UNLIKE attempts when no LIKE exists for the User Reaction
- The system shall reject invalid user identifiers
- The system shall reject invalid Reaction Target identifiers

**Behavior:**
- When a user UNLIKES a Reaction Target, the system shall remove the LIKE association
- The system shall update Reaction Target counts accordingly
- UNLIKE shall only succeed if a LIKE exists for the User Reaction
- Attempting to UNLIKE when no LIKE exists shall result in an error

### Requirement 3: Dislike an Entity

**Description:** The system shall allow a user to register a negative reaction (DISLIKE) toward a Reaction Target.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Confirmation of successful dislike registration
- Current dislike count for the Reaction Target
- Current reaction state for the User Reaction

**Error Cases:**
- The system shall reject duplicate DISLIKE attempts for the same User Reaction (idempotency violation)
- The system shall reject invalid user identifiers
- The system shall reject invalid Reaction Target identifiers
- The system shall reject attempts when storage is unavailable

**Behavior:**
- When a user DISLIKES a Reaction Target, the system shall record the association
- If the user previously LIKED the Reaction Target, the LIKE shall be removed and replaced with DISLIKE
- The system shall maintain accurate counts of all reactions per Reaction Target
- A user cannot simultaneously LIKE and DISLIKE the same Reaction Target
- **Idempotency:** If a DISLIKE already exists for the User Reaction, the operation shall return an error indicating a duplicate reaction attempt, and no state change shall occur

### Requirement 4: Undislike an Entity

**Description:** The system shall allow a user to remove a previously registered DISLIKE from a Reaction Target.

**Inputs:**
- User identifier (user_id)
- Reaction Target (entity_type, entity_id)

**Outputs:**
- Confirmation of successful undislike operation
- Updated dislike count for the Reaction Target
- Current reaction state for the User Reaction (none if no reactions remain)

**Error Cases:**
- The system shall reject UNDISLIKE attempts when no DISLIKE exists for the User Reaction
- The system shall reject invalid user identifiers
- The system shall reject invalid Reaction Target identifiers

**Behavior:**
- When a user UNDISLIKES a Reaction Target, the system shall remove the DISLIKE association
- The system shall update Reaction Target counts accordingly
- UNDISLIKE shall only succeed if a DISLIKE exists for the User Reaction
- Attempting to UNDISLIKE when no DISLIKE exists shall result in an error

### Requirement 5: Extensible Reaction Types

**Description:** The system shall provide a framework for adding new reaction types beyond LIKE and DISLIKE.

**Requirements:**
- The system shall define a clear interface for registering new reaction types
- New reaction types shall follow the same semantic pattern: ACTION and UN-ACTION
- The system shall ensure mutual exclusivity between conflicting reaction types
- The system shall maintain separate counts per reaction type per entity

## Constraints and Limitations

1. **Scope Limitation:** This module exclusively handles user reactions. Authentication, authorization, and reaction target management are the responsibility of the consuming application.

2. **Mutual Exclusivity:** A user may have at most one active reaction (LIKE or DISLIKE) per Reaction Target at any given time. Registering a new reaction of a different type automatically removes any existing reaction.

3. **Idempotency of LIKE/DISLIKE:** LIKE and DISLIKE operations are idempotent. If a LIKE already exists for a User Reaction, attempting another LIKE shall return an error indicating a duplicate reaction. Similarly for DISLIKE. No state change shall occur on duplicate attempts.

4. **Strict Removal:** UNLIKE shall only succeed if a LIKE exists for the User Reaction; UNDISLIKE shall only succeed if a DISLIKE exists. Attempting to remove a non-existent reaction shall result in an error.

5. **Atomicity:** Operations that modify both User Reactions and Reaction Target counts shall be atomic. Partial failures shall not leave the system in an inconsistent state.

6. **Validation:** All inputs shall be validated before processing. Invalid inputs shall result in immediate rejection with clear error indicators.

7. **Reaction Target Isolation:** Reactions on different Reaction Targets are completely independent. Operations on one Reaction Target never affect another.

## Relationships with Other Functional Blocks

- **[entity_association.md](entity_association.md):** Defines how reactions associate with entity types and identifiers
- **[user_interactions.md](user_interactions.md):** Defines user-centric behaviors and permissions
- **[data_persistence.md](data_persistence.md):** Defines how reaction states are stored and retrieved
- **[api_interface.md](api_interface.md):** Defines the public interface for invoking these operations
- **[audit_logging.md](audit_logging.md):** Defines the audit logging of all reaction operations

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version of reaction management specification |
| 2026-03-21 | Update | Clarified UNLIKE/UNDISLIKE require existing reactions; removed idempotency for removals |
| 2026-03-21 | Update | Introduced Reaction Target and User Reaction concepts; added idempotency for LIKE/DISLIKE operations |
| 2026-03-21 | Update | Added Module Scope and Boundaries section clarifying exclusive responsibility for user reactions and out-of-scope concerns |

## Acceptance Criteria

1. **AC1:** A user can successfully LIKE a Reaction Target and receive confirmation with updated counts
2. **AC2:** Attempting to LIKE a Reaction Target that the user already LIKES shall return an error indicating duplicate reaction; no state change shall occur (idempotency)
3. **AC3:** A user can successfully UNLIKE a Reaction Target they previously LIKED
4. **AC4:** UNLIKE on a Reaction Target without a LIKE shall result in an error
5. **AC5:** LIKE on a DISLIKED Reaction Target removes the DISLIKE and creates a LIKE (conversion)
6. **AC6:** A user can successfully DISLIKE a Reaction Target and receive confirmation with updated counts
7. **AC7:** Attempting to DISLIKE a Reaction Target that the user already DISLIKES shall return an error indicating duplicate reaction; no state change shall occur (idempotency)
8. **AC8:** DISLIKE on a LIKED Reaction Target removes the LIKE and creates a DISLIKE (conversion)
9. **AC9:** A user can successfully UNDISLIKE a previously DISLIKED Reaction Target
10. **AC10:** UNDISLIKE on a Reaction Target without a DISLIKE shall result in an error
11. **AC11:** Reaction Target counts are always consistent with the sum of User Reactions
12. **AC12:** Invalid inputs are rejected with clear error indicators
13. **AC13:** Duplicate LIKE attempts return a specific error indicating the reaction already exists
14. **AC14:** Duplicate DISLIKE attempts return a specific error indicating the reaction already exists
