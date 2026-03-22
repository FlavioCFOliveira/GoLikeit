# GoLikeit Specification Index

This directory contains functional specifications for the GoLikeit module - a Go library for managing user reactions on entities through a configurable, abstract reaction system.

## Specification-First Policy

**CRITICAL:** No code may be written without a corresponding functional specification. The specification defines WHAT the system does; implementation defines HOW it is done.

```
User Request → spec-orchestrator → specification/ → go-elite-developer → Implementation
```

## Functional Blocks

| Specification | Purpose | Last Modified |
|---------------|---------|---------------|
| [architecture.md](architecture.md) | System design with high concurrency support, caching layer, and component boundaries | 2026-03-21 |
| [reaction_management.md](reaction_management.md) | Core reaction operations with configurable reaction types, scope boundaries, and replacement semantics | 2026-03-22 |
| [entity_association.md](entity_association.md) | Reaction Target definition (entity_type + entity_id) | 2026-03-21 |
| [user_interactions.md](user_interactions.md) | User Reaction definition (user_id + Reaction Target) with single reaction per target constraint | 2026-03-22 |
| [data_persistence.md](data_persistence.md) | Data storage with PostgreSQL, MariaDB, SQLite, MongoDB, Cassandra, Redis, and In-Memory support | 2026-03-22 |
| [api_interface.md](api_interface.md) | Public API with reaction type configuration, caching, bulk operations, and simple configuration | 2026-03-22 |
| [security_policies.md](security_policies.md) | Security-first policy with mandatory immutable audit logging | 2026-03-21 |
| [performance_requirements.md](performance_requirements.md) | Performance expectations and scalability | 2026-03-21 |
| [audit_logging.md](audit_logging.md) | Configurable audit logging with NullAuditor default, independent storage, insert/get only | 2026-03-22 |
| [event_system.md](event_system.md) | Event system with 7 event types for reaction state changes and system events | 2026-03-22 |
| [rate_limiting.md](rate_limiting.md) | Configurable per-user rate limiting using sliding window algorithm | 2026-03-22 |

## Core Concepts Glossary

### Reaction Target
The unique combination of `(entity_type, entity_id)` that identifies what is being reacted to. Examples: `("photo", "123")`, `("article", "abc-456")`. See [entity_association.md](entity_association.md) for details.

### User Reaction
The unique combination of `(user_id, entity_type, entity_id)` representing a specific user's reaction to a specific Reaction Target. **IMPORTANT:** At most **ONE** reaction can exist per User Reaction. Adding a new reaction replaces any existing reaction. See [user_interactions.md](user_interactions.md) for details.

### Reaction Type
A string identifier representing a specific type of reaction. Format: `^[A-Z0-9_-]+$` (uppercase letters, digits, hyphens, underscores only). The module has **NO** predefined reaction types - all types are configured by the consuming application at initialization time. Examples: `LIKE`, `DISLIKE`, `LOVE`, `ANGRY`, `THUMBS_UP`. See [api_interface.md](api_interface.md) for configuration details.

### Reaction Replacement (Upsert)
When a user adds a reaction to a target where they already have a reaction, the previous reaction is **replaced** (not converted or undone). The previous reaction type is permanently lost and cannot be recovered. This is a core constraint: users can have only one reaction per target. See [reaction_management.md](reaction_management.md) for details.

### Audit Storage Independence
The audit package operates independently from reaction storage, supporting separate database configurations. Audit entries are append-only with only Insert and Get operations exposed. See [audit_logging.md](audit_logging.md) for details.

### High Concurrency Design
All module components are designed for high-load, high-concurrency environments. Lock-free patterns preferred, minimal lock scope when necessary, no global locks. See [architecture.md](architecture.md) for details.

### Simple Configuration API
The module exposes a simple, intuitive API with sensible defaults, fluent interface options, and minimal required configuration for quick adoption. See [api_interface.md](api_interface.md) for details.

### Module Scope
This module exclusively handles user reactions. Authentication, authorization, and reaction target management are the responsibility of the consuming application. See [reaction_management.md](reaction_management.md) for details.

### Caching Layer
Optional in-process caching layer improves read performance and reduces database load. Configurable TTL (time-to-live) for cache entries, LRU eviction, automatic invalidation on writes. TTL applies exclusively to cache entries, not to storage backends. Cache invalidation occurs at Reaction Target granularity (user_id + entity_type + entity_id) - only the specific cached entry is invalidated, not global cache. See [api_interface.md](api_interface.md) and [architecture.md](architecture.md) for details.

### Bulk Operations
API supports bulk operations for efficient batch processing: GetUserReactionsBulk, GetEntityCountsBulk, GetMultipleUserReactions. See [api_interface.md](api_interface.md) for details.

### Multi-Database Support
Supports PostgreSQL, MariaDB, SQLite, MongoDB, Cassandra, Redis, and In-Memory with database-specific optimizations. See [data_persistence.md](data_persistence.md) for details.

### NullAuditor Fallback
Audit logging is architecturally mandatory. When no audit storage is configured, the system uses NullAuditor (no-op) as a fallback. Persistent auditing is activated by configuring audit storage. See [audit_logging.md](audit_logging.md) for details.

### Security-First Policy
### No Soft Delete
RemoveReaction operations perform hard deletes on reaction records, permanently removing them from the reactions table. Historical records of removed reactions are available exclusively through the audit log. This design prioritizes storage efficiency and query performance while maintaining full historical accountability. See [reaction_management.md](reaction_management.md) for details.
Security is a fundamental design principle. Secure defaults, defense in depth, fail secure. See [security_policies.md](security_policies.md) for details.

### Consolidated Queries
Queries that return complete reaction data including counts and recent users in a single operation. Minimizes database round trips by fetching all required data in one invocation. See [api_interface.md](api_interface.md) and [data_persistence.md](data_persistence.md) for details.

### Pagination
Consistent limit-offset pagination using PaginatedResult[T] for queries potentially returning more than 50 records. Limit indicates records requested, offset indicates starting position. Response includes total records and total pages. Default limit 20, maximum 100. See [api_interface.md](api_interface.md) for details.

### Fast Reaction Check
Ultra-fast boolean operations (HasUserReaction, HasUserReactionType) optimized for quick UI feedback. Single key lookup, cache-first, <10ms response time. See [api_interface.md](api_interface.md) for details.

### Event System
Comprehensive event system that publishes notifications for all reaction-related state changes. Includes 7 event types: ReactionAdded, ReactionReplaced, ReactionRemoved, EntityCountsUpdated, BulkReactionsProcessed, CacheInvalidated, and AuditEntryCreated. Supports synchronous and asynchronous subscriptions with configurable delivery guarantees. See [event_system.md](event_system.md) for details.

### Rate Limiting
Configurable per-user rate limiting using sliding window algorithm to prevent API abuse. Supports different limits per operation (AddReaction, RemoveReaction) and global limits. All limits are externally configurable with no hardcoded values. Supports in-memory and Redis backends. See [rate_limiting.md](rate_limiting.md) for details.

### Reaction Type Validation
All reaction types must be configured at module initialization and match pattern `^[A-Z0-9_-]+$`. Validation occurs at startup - if any type fails validation, the module cannot be initialized. No reaction types can be added at runtime. See [api_interface.md](api_interface.md) for details.

## Navigation

### For New Features
1. Identify the functional block affected
2. Update or create the specification file
3. Ensure acceptance criteria are defined
4. Signal readiness to go-elite-developer

### For Bug Fixes
1. Review relevant functional specification
2. Update specification if behavior needs clarification
3. Proceed with implementation

### For Refactoring
1. Verify current implementation against specification
2. Update architecture.md if structural changes are needed
3. Ensure all functional blocks remain consistent

## Change History

| Date | Specification | Change | Description |
|------|---------------|--------|-------------|
| 2026-03-21 | All | Initial | Created all functional block specifications for GoLikeit module |
| 2026-03-21 | reaction_management, data_persistence | Update | Clarified UNLIKE/UNDISLIKE strict validation, reaction timestamp behavior, added audit_logging spec |
| 2026-03-21 | reaction_management, entity_association, user_interactions, api_interface, data_persistence | Update | Introduced Reaction Target and User Reaction concepts; added idempotency for LIKE/DISLIKE operations |
| 2026-03-21 | audit_logging | Update | Added Requirement 7 (Audit Operations Restriction - insert/get only) and Requirement 8 (Independent Storage Layer) |
| 2026-03-21 | api_interface | Update | Added Requirement 8 (Simple Configuration API) with fluent interface pattern |
| 2026-03-21 | architecture | Update | Added Requirement 7 (High Concurrency and Load Support) as critical design constraint |
| 2026-03-21 | security_policies | Update | Updated Requirement 7 to reflect mandatory, immutable audit logging |
| 2026-03-21 | data_persistence | Update | Added note about independent audit storage capability |
| 2026-03-21 | reaction_management | Update | Added Module Scope and Boundaries section |
| 2026-03-21 | data_persistence | Update | Added MongoDB and Cassandra support; added Requirement 8 (Performance Optimization) |
| 2026-03-21 | api_interface | Update | Added Requirement 9 (Caching Layer) and Requirement 10 (Bulk Operations) |
| 2026-03-21 | security_policies | Update | Added Security-First Policy section |
| 2026-03-21 | architecture | Update | Added Requirement 8 (Caching Layer) |
| 2026-03-21 | audit_logging | Update | Modified Requirement 1 to Configurable Audit Logging with NullAuditor as default |
| 2026-03-21 | data_persistence | Update | Added Requirement 9 (Redis Storage Support) and Requirement 10 (In-Memory Storage Support) |
| 2026-03-21 | api_interface | Update | Added GetUserLikes, GetUserDislikes, GetEntityReactionsWithUsers operations; added consolidated queries |
| 2026-03-21 | data_persistence | Update | Added efficiency requirements to Requirement 5 (minimize round trips, single invocation) |
| 2026-03-21 | api_interface | Update | Added Requirement 11 (Pagination Support) and Requirement 12 (Fast Reaction Check Operations) |
| 2026-03-21 | data_persistence | Update | Added pagination requirements and fast check operation requirements to Requirement 5 |
| 2026-03-21 | api_interface | Update | Updated pagination model to limit-offset; includes total records and total pages |
| 2026-03-21 | data_persistence | Update | Updated pagination to limit-offset principle with response metadata |
| 2026-03-21 | README | Update | Updated Pagination glossary entry to reflect limit-offset model |
| 2026-03-21 | audit_logging | Clarification | Requirement 1 clarified: Audit logging is architecturally mandatory; NullAuditor is fallback |
| 2026-03-21 | security_policies | Clarification | Requirement 7 clarified: Audit logging is architecturally mandatory with NullAuditor fallback |
| 2026-03-21 | README | Update | Updated NullAuditor glossary entry to reflect fallback (not default) |
| 2026-03-21 | event_system | Create | New specification added for Event System with 14 event types |
| 2026-03-21 | README | Update | Added event_system.md to functional blocks and glossary |
| 2026-03-21 | reaction_management | Update | Documented that UNLIKE/UNDISLIKE are hard deletes; audit log is source of historical data |
| 2026-03-21 | audit_logging | Update | Added constraint that audit log is source of historical truth for removed reactions |
| 2026-03-21 | rate_limiting | Create | New specification added for per-user rate limiting with sliding window |
| 2026-03-21 | README | Update | Added rate_limiting.md to functional blocks and glossary |
| 2026-03-21 | reaction_management | Update | Documented that offline operations are out of scope |
| 2026-03-21 | data_persistence | Update | Clarified that TTL applies to cache layer, not Redis storage backend |
| 2026-03-21 | README | Update | Clarified that TTL applies exclusively to cache entries, not storage |
| 2026-03-21 | api_interface | Update | Clarified cache invalidation granularity: Reaction Target level (user + entity) |
| 2026-03-21 | README | Update | Updated Caching Layer glossary entry to reflect Reaction Target invalidation |
| 2026-03-22 | data_persistence | Update | Documented eventual consistency characteristics for MongoDB/Cassandra pagination |
| 2026-03-22 | api_interface | Update | Clarified GetUserReactions semantics: active reactions only, ordered by created_at desc |
| 2026-03-22 | audit_logging | Update | Clarified audit failure behavior: reaction continues, audit failure is logged only |
| 2026-03-22 | api_interface | Update | Normalized pagination: removed threshold, created PaginationConfig with configurable parameters |
| 2026-03-22 | user_interactions | Update | Normalized nomenclature: removed NONE state, absence of reaction is nil/not found |
| 2026-03-22 | reaction_management | Update | Normalized nomenclature: removed NONE from reaction states |
| 2026-03-22 | audit_logging | Update | Normalized nomenclature: NONE replaced with empty/nil for previous_reaction |
| 2026-03-22 | event_system | Update | Normalized nomenclature: PreviousType changed to pointer, nil if no previous reaction |
| 2026-03-22 | extended_reaction_types | Create | New specification for custom reaction types (LOVE, ANGRY, etc.) beyond LIKE/DISLIKE |
| 2026-03-22 | README | Update | Added extended_reaction_types.md to functional blocks and glossary |
| 2026-03-22 | reaction_management | Update | Documented that reaction metadata is out of scope |
| 2026-03-22 | reaction_management | Update | Documented that circuit breaker pattern is out of scope |
| 2026-03-22 | reaction_management | Update | Documented that observability and monitoring is out of scope |
| 2026-03-22 | reaction_management | Update | Documented that sharding is out of scope |
| 2026-03-22 | data_persistence | Update | Documented that backup and restore is out of scope |
| 2026-03-22 | reaction_management | Update | Documented that multi-tenancy is out of scope |
| 2026-03-22 | security_policies | Update | Documented that GDPR compliance is out of scope |
| 2026-03-22 | event_system | Update | Documented that webhooks are out of scope; Event System is sufficient |
| 2026-03-22 | All | Major | Refactored to abstract reaction model: removed fixed LIKE/DISLIKE operations, replaced with configurable reaction types, single reaction per user, replacement semantics |

---

**Language Convention:** All specifications are written in clear, simple English (functional focus - WHAT not HOW).
