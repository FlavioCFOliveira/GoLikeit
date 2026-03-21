# Data Persistence Specification

## Overview

The system shall provide a storage-agnostic data layer capable of persisting reaction data across multiple database backends including PostgreSQL, MariaDB, and SQLite. The data layer abstracts database-specific implementations while providing consistent behavior and data integrity guarantees.

## Functional Requirements

### Requirement 1: Multi-Database Support

**Description:** The system shall support PostgreSQL, MariaDB, SQLite, MongoDB, Cassandra, Redis, and In-Memory as storage backends.

**Supported Backends:**
- **PostgreSQL:** Versions 12 and above
- **MariaDB:** Versions 10.5 and above
- **SQLite:** Version 3.35.0 and above (with WAL mode support)
- **MongoDB:** Versions 4.4 and above
- **Cassandra:** Versions 3.11 and above (CQL compatible)
- **Redis:** Versions 6.0 and above (with RedisJSON module recommended)
- **In-Memory:** Pure in-memory storage (no persistence, data lost on restart)

**Requirements:**
- The system shall detect or be configured with the target database type
- Database-specific optimizations shall be applied for maximum performance
- Behavior shall remain consistent across all supported backends
- Connection pooling shall be supported where applicable
- In-Memory storage is suitable for development, testing, or ephemeral data

**Database-Specific Optimizations:**
- **PostgreSQL:** Use prepared statements, proper index types, connection pooling with pgx
- **MariaDB:** Use InnoDB engine, connection pooling, query caching where appropriate
- **SQLite:** Use WAL mode, appropriate cache size, single-writer optimizations
- **MongoDB:** Use compound indexes, aggregation pipelines for counts, connection pooling
- **Cassandra:** Use appropriate replication factor, partition key design, prepared statements, batch operations where applicable
- **Redis:** Use hash data structures for reactions, sets for entity counts, pipelining for batch operations
- **In-Memory:** Use Go maps with RWMutex for thread-safe access; periodic snapshots optional

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

**Description:** The system shall support efficient querying of User Reaction data with consolidated results.

**Required Queries:**
- Retrieve User Reaction by (user_id, entity_type, entity_id)
- Retrieve all User Reactions for a user with optional filters (liked entities, disliked entities)
- Retrieve all User Reactions for a Reaction Target
- Retrieve aggregated counts by Reaction Target (total likes, total dislikes)
- Retrieve aggregated counts by user
- **Consolidated Entity Query:** Retrieve counts AND recent users who reacted in single operation
  - Returns: total likes, total dislikes, list of recent users who liked, list of recent users who disliked
  - Single database invocation to get complete reaction picture

**Efficiency Requirements:**
- **Minimize Round Trips:** All query operations shall fetch required data in minimum database calls
- **Single Invocation:** Consolidated queries (counts + users) shall use single database invocation where possible
  - SQL: Use JOINs or subqueries to fetch counts and users in one query
  - MongoDB: Use aggregation pipelines with $facet
  - Redis: Use Lua scripts or pipelining for multi-key operations
  - Cassandra: Design tables to support query patterns; denormalize where necessary
- **Batch Operations:** Support fetching multiple entities/users in single call
- **Projection:** Only request fields needed for the specific query
- **No N+1:** Query implementations shall avoid N+1 query patterns

**Pagination Requirements:**
- **Consistent Pagination:** All data layer implementations use same pagination model
- **Page Size:** Default 20, maximum 100 items per page
- **Offset/Limit:** Use OFFSET/LIMIT for SQL; skip/limit for MongoDB; range queries for Cassandra
- **Cursor Support:** Optional cursor-based pagination for Redis and high-volume scenarios
- **Total Count:** Return total item count for pagination UI
- **Threshold:** Automatic pagination for queries returning >50 records

**Fast Check Operations:**
- **HasUserLiked:** Ultra-fast boolean check (single key lookup)
  - SQL: SELECT EXISTS(SELECT 1 FROM reactions WHERE ...)
  - Redis: EXISTS reaction:{user_id}:{entity_type}:{entity_id}
  - MongoDB: Count with limit 1
  - In-Memory: Map lookup with O(1)
- **HasUserDisliked:** Same optimizations as HasUserLiked
- **Performance Target:** <10ms p95 latency
- **Cache Priority:** Check cache before database for fastest response

**Performance Requirements:**
- Queries by (user_id, entity_type, entity_id) shall use indexed lookups
- Reaction Target count queries shall be optimized (materialized or cached)
- Pagination shall be supported for large result sets
- Consolidated queries shall complete within same time budget as simple count queries

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
- Connection pooling shall be supported for PostgreSQL, MariaDB, MongoDB, and Cassandra
- Connection timeouts shall be configurable
- Failed connections shall be reported with clear error messages
- Resources shall be properly released on shutdown
- Connection strings shall be validated at startup

### Requirement 8: Performance Optimization

**Description:** All database interactions shall be optimized for maximum performance.

**Query Optimization Requirements:**
- **Index Usage:** All queries must use appropriate indexes; full table scans are prohibited for hot paths
- **Prepared Statements:** Use prepared statements for all repeated queries
- **Batch Operations:** Support batch inserts/updates where applicable
- **Projection:** Queries shall request only necessary fields, not use SELECT *
- **Connection Reuse:** Connections must be returned to the pool promptly
- **Query Plan Review:** Query patterns shall be reviewed for optimal execution plans

**Database-Specific Performance:**
- **PostgreSQL:** Use EXPLAIN ANALYZE for query review; implement partial indexes where beneficial
- **MariaDB:** Use EXPLAIN for query review; consider covering indexes
- **SQLite:** Use EXPLAIN QUERY PLAN; optimize for single-writer scenarios
- **MongoDB:** Use explain() for query review; design indexes for query patterns
- **Cassandra:** Design tables for query patterns; avoid ALLOW FILTERING

**Monitoring:**
- Slow query logging (configurable threshold)
- Query execution time metrics
- Connection pool utilization metrics

### Requirement 9: Redis Storage Support

**Description:** The system shall support Redis as a storage backend for high-performance, low-latency reaction storage.

**Redis Data Model:**
- **User Reactions:** Stored as hash entries with key pattern `reaction:{user_id}:{entity_type}:{entity_id}`
- **Entity Counts:** Stored as hash with key `counts:{entity_type}:{entity_id}` containing like/dislike counters
- **User Reaction Index:** Set per user containing all entities reacted to for quick lookup
- **Entity Reaction Index:** Set per entity containing all users who reacted for aggregation

**Redis Operations:**
- **Pipelining:** Batch operations use pipelining to minimize round trips
- **Transactions:** Multi/Exec for atomic counter updates
- **Lua Scripts:** Server-side scripts for complex atomic operations (e.g., swap LIKE to DISLIKE)
- **TTL Support:** Optional expiration for temporary reactions

**Requirements:**
- Redis connection pooling via go-redis or similar client
- Support for Redis Cluster and Redis Sentinel configurations
- Automatic reconnection with exponential backoff
- Configurable key prefix for namespace isolation

**Redis Limitations:**
- Memory-based storage requires sufficient RAM for dataset
- Durability depends on Redis persistence configuration (RDB/AOF)
- Eventual consistency in Redis Cluster mode for some operations

### Requirement 10: In-Memory Storage Support

**Description:** The system shall provide a pure in-memory storage implementation for development, testing, and ephemeral use cases.

**In-Memory Storage Characteristics:**
- **No Persistence:** Data exists only in process memory; all data is lost on application restart
- **No External Dependencies:** No database server required; ideal for development and testing
- **Thread-Safe:** All operations are safe for concurrent access from multiple goroutines
- **Performance:** Fastest possible read/write performance (no network or disk I/O)

**Use Cases:**
- Development and local testing without database setup
- Unit tests requiring isolated, fast storage
- Ephemeral data that does not require persistence
- High-performance scenarios where durability is not required

**Data Structures:**
- **User Reactions:** Stored in Go maps with composite key (user_id + entity_type + entity_id)
- **Entity Counts:** Maintained in separate counters map with atomic operations
- **Concurrency:** RWMutex for read-heavy operations; sync.Map optional for high concurrency

**Limitations:**
- Data is not persisted across application restarts
- Memory usage is limited by available RAM
- No replication or backup capabilities
- Single-node only (no distributed in-memory option)

**Requirements:**
- In-Memory storage implements the same Data Layer interface as persistent backends
- Switching between In-Memory and persistent storage requires no code changes (only configuration)
- Atomic operations are simulated with mutex locks
- Query capabilities match persistent backends (filtered by user, entity, etc.)

## Constraints and Limitations

1. **No Automatic Schema Creation in Production:** Production deployments should create schemas through explicit migration commands, not automatic initialization.

2. **Database-Specific Types:** While behavior is consistent, some field types may vary by database (e.g., UUID storage, timestamp precision).

3. **SQLite Limitations:** SQLite has reduced concurrency compared to PostgreSQL and MariaDB. Write operations are serialized at the database level.

4. **MongoDB Considerations:** MongoDB uses eventual consistency for some operations; transaction support requires replica sets.

5. **Cassandra Considerations:** Cassandra prioritizes availability over consistency; design requires careful partition key selection.

6. **Redis Considerations:** Redis is memory-based; durability depends on persistence settings. Cluster mode has eventual consistency for multi-key operations.

7. **In-Memory Limitations:** Data is lost on application restart; limited by available RAM; not suitable for production use cases requiring durability.

8. **No Cross-Database Replication:** The system does not provide built-in replication between different database types.

9. **No Embedded Cache:** The data persistence layer does not include application-level caching; this is the responsibility of higher layers.

10. **Independent Audit Storage:** While reaction data uses the configured primary storage, audit logging may be configured to use a separate database. See [audit_logging.md](audit_logging.md) for details.

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
| 2026-03-21 | Update | Added MongoDB and Cassandra support; added Requirement 8 (Performance Optimization) |
| 2026-03-21 | Update | Added Requirement 9 (Redis Storage Support) and Requirement 10 (In-Memory Storage Support) |
| 2026-03-21 | Update | Added Requirement 5 efficiency requirements (minimize round trips, single invocation for consolidated queries) |
| 2026-03-21 | Update | Added pagination requirements and fast check operation requirements to Requirement 5 |

## Acceptance Criteria

1. **AC1:** The system supports PostgreSQL 12+ as a storage backend
2. **AC2:** The system supports MariaDB 10.5+ as a storage backend
3. **AC3:** The system supports SQLite 3.35.0+ as a storage backend
4. **AC4:** The system supports MongoDB 4.4+ as a storage backend
5. **AC5:** The system supports Cassandra 3.11+ as a storage backend
6. **AC6:** The system supports Redis 6.0+ as a storage backend
7. **AC7:** The system supports In-Memory storage for development and testing
8. **AC8:** All User Reaction data includes required fields (id, user_id, entity_type, entity_id, reaction_type, created_at)
9. **AC9:** The combination of (user_id, entity_type, entity_id) is unique across the system (User Reaction uniqueness)
10. **AC10:** Creating a User Reaction and updating Reaction Target counts is atomic
11. **AC11:** Removing a User Reaction and updating Reaction Target counts is atomic
12. **AC12:** Failed operations do not leave partial data in the database
13. **AC13:** Queries by (user_id, entity_type, entity_id) use indexed lookups
14. **AC14:** Schema migrations are versioned and tracked
15. **AC15:** Migration scripts are provided for all supported databases
16. **AC16:** Connection pooling is supported for PostgreSQL, MariaDB, MongoDB, Cassandra, and Redis
17. **AC17:** Duplicate LIKE/DISLIKE attempts are detected and rejected with no database changes
18. **AC18:** All queries use appropriate indexes; no full table scans on hot paths
19. **AC19:** Prepared statements are used for all repeated queries
20. **AC20:** Query execution times are logged and monitored
21. **AC21:** In-Memory storage implements the same Data Layer interface as persistent backends
22. **AC22:** Redis supports pipelining for batch operations
23. **AC23:** Consolidated queries (counts + users) use single database invocation where possible
24. **AC24:** No N+1 query patterns in query implementations
25. **AC25:** Queries for user likes/dislikes use pagination (default 20, max 100)
26. **AC26:** All data layer implementations use consistent pagination model
27. **AC27:** HasUserLiked and HasUserDisliked use single key lookup
28. **AC28:** Fast check operations complete in <10ms p95 latency
