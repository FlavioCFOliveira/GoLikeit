# Data Persistence Specification

## Overview

The system shall provide a storage-agnostic data layer capable of persisting reaction data across multiple database backends including PostgreSQL, MariaDB, and SQLite. The data layer abstracts database-specific implementations while providing consistent behavior and data integrity guarantees.

## Functional Requirements

### Requirement 1: Multi-Database Support

**Description:** The system shall support PostgreSQL, MariaDB, and SQLite as storage backends.

**Supported Backends:**
- **PostgreSQL:** Versions 12 and above
- **MariaDB:** Versions 10.5 and above
- **SQLite:** Version 3.35.0 and above (with WAL mode support)

**Requirements:**
- The system shall detect or be configured with the target database type
- Database-specific optimizations may be used, but behavior shall remain consistent
- Connection pooling shall be supported where applicable

### Requirement 2: Data Integrity

**Description:** The system shall maintain data integrity for all User Reaction records.

**Requirements:**
- Each User Reaction record shall have a unique identifier
- The combination of (user_id, entity_type, entity_id) shall be unique (one reaction per User Reaction)
- Reaction timestamps shall be stored with timezone information or in UTC
- Reaction Target counts shall remain consistent with the underlying User Reaction data

**User Reaction Uniqueness:**
- The composite key (user_id, entity_type, entity_id) represents a single User Reaction
- Duplicate User Reactions are prevented at the database level via unique constraints
- Attempting to create a duplicate User Reaction with the same reaction type shall be rejected

**Constraints:**
- Data corruption shall be detected and reported
- Partial writes shall be prevented or automatically rolled back
- Concurrent modifications shall not leave the database in an inconsistent state

### Requirement 3: User Reaction Data Model

**Description:** The system shall store User Reaction data with the following minimum attributes.

**Required Fields:**
- **id:** Unique identifier for the User Reaction record
- **user_id:** Identifier of the user who performed the reaction (User Reaction user component)
- **entity_type:** Type of entity being reacted to (Reaction Target type component)
- **entity_id:** Identifier of the specific entity instance (Reaction Target ID component)
- **reaction_type:** Type of reaction (LIKE, DISLIKE, etc.)
- **created_at:** Timestamp when the reaction was created (represents the moment of the LIKE or DISLIKE action)

**Composite Key:**
- The combination of (user_id, entity_type, entity_id) represents a User Reaction
- This composite key shall be unique at the database level
- Together, these fields identify a user's reaction to a specific Reaction Target

**Behavior:**
- Creating a new LIKE or DISLIKE sets created_at to the current time
- When a LIKE replaces a DISLIKE (or vice versa), the created_at timestamp is updated to reflect the moment of the new reaction
- Deleting a reaction (UNLIKE or UNDISLIKE) removes the record entirely
- The created_at timestamp always represents when the current reaction (LIKE or DISLIKE) was established
- Duplicate LIKE/DISLIKE attempts (same User Reaction, same reaction_type) are rejected at the application level

### Requirement 4: Atomic Operations

**Description:** Operations that modify multiple related records shall be atomic.

**Requirements:**
- Creating a User Reaction and updating Reaction Target counts shall be atomic
- Removing a User Reaction and updating Reaction Target counts shall be atomic
- Switching reaction types (LIKE to DISLIKE) shall be atomic
- Failed operations shall leave the database in its previous consistent state

**Idempotency Enforcement:**
- Duplicate LIKE/DISLIKE detection occurs within the transaction
- If a duplicate is detected, the transaction rolls back with no changes
- This ensures atomicity of idempotency checks

**Constraints:**
- Atomicity shall be enforced at the database transaction level
- Transactions shall use appropriate isolation levels
- Deadlocks shall be detected and handled appropriately

### Requirement 5: Query Capabilities

**Description:** The system shall support efficient querying of User Reaction data.

**Required Queries:**
- Retrieve User Reaction by (user_id, entity_type, entity_id)
- Retrieve all User Reactions for a user with optional filters
- Retrieve all User Reactions for a Reaction Target
- Retrieve aggregated counts by Reaction Target (total likes, total dislikes)
- Retrieve aggregated counts by user

**Performance Requirements:**
- Queries by (user_id, entity_type, entity_id) shall use indexed lookups
- Reaction Target count queries shall be optimized (materialized or cached)
- Pagination shall be supported for large result sets

### Requirement 6: Migration Support

**Description:** The system shall provide schema migration capabilities.

**Requirements:**
- Schema changes shall be versioned
- Migration scripts shall be provided for each supported database
- Up and down migrations shall be available
- Migration state shall be tracked in the database

**Constraints:**
- Migrations shall be reversible where possible
- Data loss during migration shall be minimized
- Migration failures shall be detected and reported

### Requirement 7: Connection Management

**Description:** The system shall manage database connections efficiently.

**Requirements:**
- Connection pooling shall be supported for PostgreSQL and MariaDB
- Connection timeouts shall be configurable
- Failed connections shall be reported with clear error messages
- Resources shall be properly released on shutdown

## Constraints and Limitations

1. **No Automatic Schema Creation in Production:** Production deployments should create schemas through explicit migration commands, not automatic initialization.

2. **Database-Specific Types:** While behavior is consistent, some field types may vary by database (e.g., UUID storage, timestamp precision).

3. **SQLite Limitations:** SQLite has reduced concurrency compared to PostgreSQL and MariaDB. Write operations are serialized at the database level.

4. **No Cross-Database Replication:** The system does not provide built-in replication between different database types.

5. **No Embedded Cache:** The data persistence layer does not include application-level caching; this is the responsibility of higher layers.

6. **Independent Audit Storage:** While reaction data uses the configured primary storage, audit logging may be configured to use a separate database. See [audit_logging.md](audit_logging.md) for details.

## Relationships with Other Functional Blocks

- **[reaction_management.md](reaction_management.md):** Defines the data operations for reactions
- **[entity_association.md](entity_association.md):** Defines how entity references are stored
- **[user_interactions.md](user_interactions.md):** Defines how user identifiers are stored
- **[performance_requirements.md](performance_requirements.md):** Defines storage performance expectations
- **[audit_logging.md](audit_logging.md):** Defines the audit logging storage requirements

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version of data persistence specification |
| 2026-03-21 | Update | Clarified reaction timestamp represents moment of LIKE/DISLIKE action; removed updated_at field |
| 2026-03-21 | Update | Updated data model to reflect User Reaction and Reaction Target concepts; added duplicate detection requirements |
| 2026-03-21 | Update | Added note about independent audit storage capability |

## Acceptance Criteria

1. **AC1:** The system supports PostgreSQL 12+ as a storage backend
2. **AC2:** The system supports MariaDB 10.5+ as a storage backend
3. **AC3:** The system supports SQLite 3.35.0+ as a storage backend
4. **AC4:** All User Reaction data includes required fields (id, user_id, entity_type, entity_id, reaction_type, created_at)
5. **AC5:** The combination of (user_id, entity_type, entity_id) is unique across the system (User Reaction uniqueness)
6. **AC6:** Creating a User Reaction and updating Reaction Target counts is atomic
7. **AC7:** Removing a User Reaction and updating Reaction Target counts is atomic
8. **AC8:** Failed operations do not leave partial data in the database
9. **AC9:** Queries by (user_id, entity_type, entity_id) use indexed lookups
10. **AC10:** Schema migrations are versioned and tracked
11. **AC11:** Migration scripts are provided for all supported databases
12. **AC12:** Connection pooling is supported for PostgreSQL and MariaDB
13. **AC13:** Duplicate LIKE/DISLIKE attempts are detected and rejected with no database changes
