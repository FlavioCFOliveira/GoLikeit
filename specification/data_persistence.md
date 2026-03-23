# Data Persistence Specification

## Overview

The system provides a storage-agnostic data layer capable of persisting reaction data across multiple database backends.

## Functional Requirements

### Requirement 1: Multi-Database Support

The system supports multiple storage backends.

**Supported Backends:**
- **PostgreSQL:** Versions 12+
- **MariaDB:** Versions 10.5+
- **SQLite:** Version 3.35.0+
- **MongoDB:** Versions 4.4+
- **Cassandra:** Versions 3.11+
- **Redis:** Versions 6.0+
- **In-Memory:** Pure in-memory storage

### Requirement 2: Data Integrity

The system maintains data integrity.

- Each User Reaction has a unique identifier
- Composite key (user_id, entity_type, entity_id) is unique
- Timestamps stored in UTC
- Reaction Target counts remain consistent with User Reaction data

### Requirement 3: User Reaction Data Model

**Required Fields:**
- **id:** Unique identifier
- **user_id:** User who performed the reaction
- **entity_type:** Type of entity
- **entity_id:** Entity instance identifier
- **reaction_type:** Type of reaction
- **created_at:** Timestamp when reaction was created or replaced

### Requirement 4: Atomic Operations

Operations that modify multiple records are atomic.

- Creating/replacing User Reaction and updating counts is atomic
- Removing User Reaction and updating counts is atomic
- Failed operations leave database in consistent state

### Requirement 5: Query Capabilities

The system supports efficient querying.

**Required Queries:**
- Retrieve User Reaction by composite key
- Retrieve all reactions for a user
- Retrieve all reactions for a Reaction Target
- Retrieve aggregated counts per reaction type
- Consolidated query (counts + recent users)

**Pagination:**
- Limit-offset model
- Default limit: 25
- Maximum limit: 100
- Maximum offset: 100

**Fast Check Operations:**
- HasUserReaction: Single key lookup
- HasUserReactionType: Single key lookup
- Target: <10ms p95 latency

### Requirement 6: Migration Support

Schema migrations are supported.

- Schema changes are versioned
- Migration scripts provided for each database
- Migration state tracked in database

### Requirement 7: Connection Management

Database connections are managed efficiently.

- Connection pooling supported
- Configurable timeouts
- Resources released on shutdown

### Requirement 8: Performance Optimization

Database interactions are optimized.

- Proper index usage
- Prepared statements for repeated queries
- Batch operations supported
- Projection (only request necessary fields)

### Requirement 9: Redis Storage

Redis is supported as a storage backend.

- User Reactions stored as hash entries
- Entity Counts stored as hash per type
- Pipelining for batch operations
- Support for Redis Cluster and Sentinel

### Requirement 10: In-Memory Storage

Pure in-memory storage is provided.

- No persistence (data lost on restart)
- Thread-safe operations
- Suitable for development and testing

### Requirement 11: Logical Data Model

The system defines a logical data model for persistence, regardless of the underlying storage technology.

**Entities and Attributes:**

- **Reaction:**
    - `id`: Unique internal identifier (implementation-defined, e.g. auto-increment or UUID)
    - `user_id`: Identifier of the user (non-empty opaque string, max 256 characters)
    - `entity_type`: Type of the entity (alphanumeric snake_case with optional hyphens)
    - `entity_id`: Identifier of the specific entity instance (non-empty opaque string, max 256 characters)
    - `reaction_type`: Type of the reaction (uppercase alphanumeric)
    - `created_at`: Timestamp (ISO 8601 UTC)

- **Reaction Target (Aggregate View):**
    - `entity_type`: Type of the entity
    - `entity_id`: Identifier of the entity instance
    - `counts`: Map of reaction types to their respective totals
    - `last_reaction_at`: Timestamp of the most recent reaction

**Persistence Guarantees:**

1.  **Uniqueness**: For any combination of `user_id`, `entity_type`, and `entity_id`, there MUST be at most one `Reaction` record.
2.  **Atomicity**: Creating or updating a `Reaction` MUST be atomic with respect to the `Reaction Target` counts.
3.  **Consistency**: The sum of individual `Reaction` records for a given `entity_type` and `entity_id` MUST match the totals stored in the `Reaction Target`.
4.  **Durability**: Once a reaction operation is confirmed, it MUST be persistent in the underlying storage.

### Requirement 12: Connection and Resource Management

The system MUST provide mechanisms for efficient management of database connections and system resources.

**Capabilities:**

1.  **Connection Configuration**: The system MUST accept parameters for locating, identifying, and authenticating with the target data source.
2.  **Concurrency Control (Pooling)**: The system MUST allow configuring the maximum number of simultaneous active connections to prevent resource exhaustion.
3.  **Resource Recycling**: The system MUST allow defining maximum lifetime and idle times for connections to ensure resource health and rotation.
4.  **Operational Timeouts**: The system MUST support configurable time limits for both establishing connections and executing operations to maintain system responsiveness.
5.  **Graceful Termination**: All resources and connections MUST be properly released during system shutdown.

### Requirement 13: Distributed Storage and Eventual Consistency

The system supports distributed database backends (MongoDB, Cassandra) that may operate under eventual consistency models.

**Consistency Rules:**

1.  **Database-Specific Configuration**: The system MUST allow configuring read and write consistency levels per database adapter (e.g., Read Preference in MongoDB, Consistency Level in Cassandra).
2.  **State Transparency**: The module MUST provide the best available data provided by the storage backend at the time of the request. No additional logic is implemented to mitigate temporary inconsistencies in paginated results or aggregate counts.
3.  **Conflict Resolution**: In the event of concurrent write operations, the system delegates conflict resolution to the underlying database mechanism, adhering to the **Last Write Wins (LWW)** principle.
4.  **Operational Awareness**: Applications using eventually consistent backends MUST be aware that counts and user states may have a synchronization delay across nodes.

### Requirement 14: Redis Logical Data Organization

When using Redis as a data source, the system MUST organize data logically to ensure efficient access and support for distributed environments (Redis Cluster).

**Key Naming Convention:**
- Use a hierarchical colon-separated format: `prefix:{hash_tag}:suffix`.
- **Hash Tags**: Use curly braces `{}` to ensure related keys are colocated on the same cluster shard.

**Logical Key Mapping:**

1.  **User Reaction State**:
    - **Key Pattern**: `reaction:{user_id}:{entity_type}:{entity_id}`
    - **Structure**: `HASH` containing reaction details (`type`, `created_at`).
    - **Note**: This provides O(1) access to a specific user's reaction.

2.  **Entity Reaction Counts (Aggregate)**:
    - **Key Pattern**: `counts:{{entity_type}:{entity_id}}`
    - **Structure**: `HASH` where fields are `reaction_type` and values are the respective `counts`.
    - **Note**: Using the entity identifier as the hash tag ensures all counts for a specific target are colocated.

3.  **Recent Reactions (Optional Buffer)**:
    - **Key Pattern**: `recent:{{entity_type}:{entity_id}}`
    - **Structure**: `ZSET` (Sorted Set) scored by timestamp for efficient retrieval of recent activity.

**Cluster Support:**
- The system MUST support Redis Cluster hash tags to enable atomic operations across related keys (e.g., updating a reaction and its target counts) when supported by the underlying driver's pipelining/transaction capabilities.

## Constraints and Limitations

1. **No Automatic Schema Creation:** Production deployments use explicit migrations.
2. **SQLite Limitations:** Reduced concurrency compared to PostgreSQL/MariaDB.
3. **MongoDB Considerations:** Eventual consistency for some operations.
4. **Redis Considerations:** Memory-based; durability depends on persistence settings.
5. **In-Memory Limitations:** Data lost on restart; limited by RAM.
6. **Independent Audit Storage:** Audit may use separate database.

## Acceptance Criteria

1. **AC1:** Supports PostgreSQL 12+
2. **AC2:** Supports MariaDB 10.5+
3. **AC3:** Supports SQLite 3.35.0+
4. **AC4:** Supports MongoDB 4.4+
5. **AC5:** Supports Cassandra 3.11+
6. **AC6:** Supports Redis 6.0+
7. **AC7:** Supports In-Memory storage
8. **AC8:** All User Reactions include required fields
9. **AC9:** Composite key (user_id, entity_type, entity_id) is unique
10. **AC10:** Creating/replacing reaction and updating counts is atomic
11. **AC11:** Removing reaction and updating counts is atomic
12. **AC12:** Queries use indexed lookups
13. **AC13:** Schema migrations are versioned
14. **AC14:** Connection pooling is supported
15. **AC15:** Fast check operations complete in <10ms p95
