# User Interactions Specification

## Overview

The system provides user-agnostic reaction capabilities, treating user identifiers as opaque values passed through from the consuming application. The module does not maintain user information, authentication state, or user profiles.

Each user may have exactly **one** reaction per Reaction Target. When a user adds a reaction to a target that already has a reaction from that user, the previous reaction is replaced.

## Core Concepts

### User Reaction

A **User Reaction** is the unique combination of a user identifier and a Reaction Target. It is represented as a tuple `(user_id, entity_type, entity_id)`.

**Properties:**
- **User Component:** The identifier of the user performing the reaction (opaque string)
- **Reaction Target Component:** The target being reacted to
- **Uniqueness:** The combination `(user_id, entity_type, entity_id)` is unique
- **Single Reaction:** Each User Reaction has exactly **one** reaction type at any time
- **Reaction Type:** The current reaction type is stored as the value

**Constraints:**
- Only **one** reaction can exist per User Reaction at any time
- Absence of a User Reaction is represented by the absence of a record

## Functional Requirements

### Requirement 1: User Identification

The system accepts and stores user identifiers provided by the consuming application.

**Inputs:**
- User identifier (user_id)

**Outputs:**
- Validation result for the user identifier format

**Constraints:**
- User identifiers shall be non-empty strings
- User identifiers shall have a maximum length of 256 characters
- User identifiers are treated as opaque values

### Requirement 2: User Reaction State

The system maintains the current reaction type for each User Reaction.

- For any User Reaction, the system tracks the current reaction type (if any)
- The reaction type must be one of the configured reaction types
- Only **one** reaction can exist per User Reaction at any time

### Requirement 3: User Reaction History

The system supports retrieving all User Reactions performed by a specific user.

**Inputs:**
- User identifier (user_id)
- Optional filters: entity_type, reaction_type, pagination parameters

**Outputs:**
- List of User Reactions performed by the user
- Each entry includes: Reaction Target, reaction_type, timestamp

**Requirements:**
- Results ordered by timestamp (most recent first)
- Pagination supports limit and offset parameters
- Filtering by entity_type restricts results to that entity type
- Filtering by reaction_type restricts results to that reaction type

### Requirement 4: User Reaction Counts

The system provides aggregation of User Reactions performed by a user.

**Inputs:**
- User identifier (user_id)
- Optional filter: entity_type

**Outputs:**
- Total count of User Reactions per reaction type
- Breakdown by entity type (if requested)

**Requirements:**
- Counts reflect current active User Reactions only
- Counts are accurate and consistent with stored data

### Requirement 5: Anonymous Users

The system supports anonymous or guest user identifiers.

- The system does not distinguish between authenticated and anonymous user identifiers
- Special user identifier values are treated the same as any other identifier
- The consuming application is responsible for generating and managing anonymous identifiers

## Constraints and Limitations

1. **No User Authentication:** User authentication is the responsibility of the consuming application.

2. **No User Metadata:** The system does not store user names, profiles, or any other user metadata.

3. **No User Sessions:** The system does not maintain session state or login/logout tracking.

4. **Opaque Identifiers:** The system does not interpret user identifier format.

5. **No Cross-User Operations:** The system does not provide operations that compare users or aggregate across multiple users.

6. **Single Reaction Per Target:** Users cannot have multiple simultaneous reactions on the same target.

## Relationships with Other Functional Blocks

- **[reaction_management.md](reaction_management.md):** Reaction operations
- **[entity_association.md](entity_association.md):** Entities users react to
- **[data_persistence.md](data_persistence.md):** Storage of user identifiers
- **[security_policies.md](security_policies.md):** Security considerations

## Acceptance Criteria

1. **AC1:** The system accepts any non-empty user identifier (max 256 chars)
2. **AC2:** The system rejects empty user identifiers
3. **AC3:** The system rejects user identifiers exceeding 256 characters
4. **AC4:** User identifiers are treated as opaque
5. **AC5:** Querying User Reaction state returns reaction type or empty string
6. **AC6:** User Reaction history includes all current reactions for the user
7. **AC7:** User Reaction history supports pagination
8. **AC8:** User Reaction history supports filtering by entity_type
9. **AC9:** User Reaction counts reflect current active reactions only
10. **AC10:** Anonymous user identifiers are processed identically to authenticated identifiers
11. **AC11:** Each User Reaction can have at most one active reaction
12. **AC12:** Adding a new reaction replaces any existing reaction
