# User Interactions Specification

## Overview

The system shall provide user-agnostic reaction capabilities, treating user identifiers as opaque values passed through from the consuming application. The module does not maintain user information, authentication state, or user profiles; it solely records which user identifier (user_id) performed which reaction on which Reaction Target.

Each user may have exactly **one** reaction per Reaction Target. When a user adds a reaction to a target that already has a reaction from that user, the previous reaction is replaced with the new one.

## Core Concepts

### User Reaction

A **User Reaction** is the unique combination of a user identifier and a Reaction Target, representing a specific user's reaction to a specific target. It is represented as a tuple `(user_id, entity_type, entity_id)`.

**Properties:**
- **User Component:** The identifier of the user performing the reaction (opaque string)
- **Reaction Target Component:** The target being reacted to (combination of entity_type and entity_id)
- **Uniqueness:** The combination `(user_id, entity_type, entity_id)` is unique within the system
- **Single Reaction:** Each User Reaction has exactly **one** reaction type at any time (or none if no reaction exists)
- **Reaction Type:** The current reaction type is stored as the value of the User Reaction

**Constraints:**
- Only **ONE** reaction can exist per User Reaction at any time
- User Reactions are the atomic unit of user-to-target reaction tracking
- The system maintains the current reaction type for active User Reactions
- Absence of a User Reaction (no reaction) is represented by the absence of a record

**Replacement Behavior:**
- When a user adds reaction type A to a target where they already have reaction type B:
  - Reaction type B is permanently removed
  - Reaction type A is created
  - This is a replacement (upsert), not a conversion or undo
- The previous reaction type cannot be recovered

## Functional Requirements

### Requirement 1: User Identification

**Description:** The system shall accept and store user identifiers provided by the consuming application.

**Inputs:**
- User identifier (user_id)

**Outputs:**
- Validation result for the user identifier format

**Constraints:**
- User identifiers shall be non-empty strings
- User identifiers shall have a maximum length of 256 characters
- The system shall treat user identifiers as opaque values

**Behavior:**
- The system shall accept any non-empty user identifier within length limits
- The system shall not validate user existence against external systems
- User identifiers are treated as immutable references

### Requirement 2: User Reaction State

**Description:** The system shall maintain the current reaction type for each User Reaction (user_id + Reaction Target).

**Requirements:**
- For any User Reaction (user_id, entity_type, entity_id), the system shall track the current reaction type (if any)
- The reaction type must be one of the configured reaction types
- Only **ONE** reaction can exist per User Reaction at any time

**Behavior:**
- Querying the reaction state for a User Reaction returns the current reaction type, or empty string if no reaction exists
- State changes follow the rules defined in reaction_management.md (replacement semantics)
- The system shall provide efficient lookup of reaction state by user and Reaction Target

### Requirement 3: User Reaction History

**Description:** The system shall support retrieving all User Reactions performed by a specific user.

**Inputs:**
- User identifier (user_id)
- Optional filters: entity_type, reaction_type, pagination parameters

**Outputs:**
- List of User Reactions performed by the user
- Each entry includes: Reaction Target (entity_type, entity_id), reaction_type, timestamp

**Requirements:**
- Results shall be ordered by timestamp (most recent first by default)
- Pagination shall support limit and offset parameters
- Filtering by entity_type restricts results to User Reactions on that entity type
- Filtering by reaction_type restricts results to User Reactions of that reaction type

### Requirement 4: User Reaction Counts

**Description:** The system shall provide aggregation of User Reactions performed by a user.

**Inputs:**
- User identifier (user_id)
- Optional filter: entity_type

**Outputs:**
- Total count of User Reactions per reaction type
- Breakdown by entity type (if requested)

**Requirements:**
- Counts shall reflect current active User Reactions only (removed reactions are excluded)
- Counts shall be accurate and consistent with the User Reaction data

### Requirement 5: Anonymous Users

**Description:** The system shall support anonymous or guest user identifiers.

**Requirements:**
- The system does not distinguish between authenticated and anonymous user identifiers
- Special user identifier values (e.g., "anonymous", "guest") are treated the same as any other identifier
- The consuming application is responsible for generating and managing anonymous identifiers

**Behavior:**
- Reactions from "user_123" and "anonymous_456" are processed identically
- No additional validation is performed based on user identifier patterns

## Constraints and Limitations

1. **No User Authentication:** The system does not authenticate users or validate credentials. User authentication is the responsibility of the consuming application.

2. **No User Metadata:** The system does not store user names, profiles, email addresses, or any other user metadata. Only the user identifier component of User Reactions is stored.

3. **No User Sessions:** The system does not maintain session state or login/logout tracking.

4. **Opaque Identifiers:** The system does not interpret user identifier format. UUIDs, integers-as-strings, email addresses, and custom formats are all treated uniformly as the user component of User Reactions.

5. **No Cross-User Operations:** The system does not provide operations that compare users or aggregate across multiple users (except for Reaction Target-level counts).

6. **User Identifier Immutability:** Once a User Reaction is recorded with a user identifier, that identifier cannot be changed in the reaction record. User identifier migration must be handled by the consuming application.

7. **User Reaction Uniqueness:** Each combination of (user_id, entity_type, entity_id) represents a single User Reaction. Duplicate User Reactions cannot exist simultaneously. Only one reaction type is stored per User Reaction.

8. **Single Reaction Per Target:** Users cannot have multiple simultaneous reactions on the same target. Adding a new reaction replaces any existing reaction.

## Relationships with Other Functional Blocks

- **[reaction_management.md](reaction_management.md):** Defines the reaction operations performed by users
- **[entity_association.md](entity_association.md):** Defines the entities users react to
- **[data_persistence.md](data_persistence.md):** Defines how user identifiers are stored
- **[security_policies.md](security_policies.md):** Defines security considerations for user data

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version of user interactions specification |
| 2026-03-21 | Update | Introduced User Reaction concept; updated all requirements to use User Reaction terminology |
| 2026-03-22 | Major | Updated to reflect single reaction per user policy; documented replacement behavior |

## Acceptance Criteria

1. **AC1:** The system accepts any non-empty user identifier (max 256 chars)
2. **AC2:** The system rejects empty user identifiers
3. **AC3:** The system rejects user identifiers exceeding 256 characters
4. **AC4:** User identifiers are treated as opaque (no format validation beyond length)
5. **AC5:** Querying User Reaction state returns reaction type or empty string if no reaction exists
6. **AC6:** User Reaction history includes all current User Reactions for the user
7. **AC7:** User Reaction history supports pagination (limit/offset)
8. **AC8:** User Reaction history supports filtering by entity_type component of Reaction Target
9. **AC9:** User Reaction counts reflect current active User Reactions only
10. **AC10:** User counts by type are accurate and consistent with stored User Reaction data
11. **AC11:** Anonymous user identifiers are processed identically to authenticated user identifiers
12. **AC12:** No authentication or session validation is performed on user identifiers
13. **AC13:** Each User Reaction (user_id + Reaction Target) can have at most one active reaction
14. **AC14:** Adding a new reaction to a target with an existing reaction replaces the previous reaction
