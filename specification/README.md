# GoLikeit Specification Index

This directory contains functional specifications for the GoLikeit module - a Go library for managing user reactions (Like/Dislike) on entities.

## Specification-First Policy

**CRITICAL:** No code may be written without a corresponding functional specification. The specification defines WHAT the system does; implementation defines HOW it is done.

```
User Request → spec-orchestrator → specification/ → go-elite-developer → Implementation
```

## Functional Blocks

| Specification | Purpose | Last Modified |
|---------------|---------|---------------|
| [architecture.md](architecture.md) | System design with high concurrency support, caching layer, and component boundaries | 2026-03-21 |
| [reaction_management.md](reaction_management.md) | Core reaction operations with scope boundaries and idempotency | 2026-03-21 |
| [entity_association.md](entity_association.md) | Reaction Target definition (entity_type + entity_id) | 2026-03-21 |
| [user_interactions.md](user_interactions.md) | User Reaction definition (user_id + Reaction Target) | 2026-03-21 |
| [data_persistence.md](data_persistence.md) | Data storage with PostgreSQL, MariaDB, SQLite, MongoDB, Cassandra, Redis, and In-Memory support | 2026-03-21 |
| [api_interface.md](api_interface.md) | Public API with caching, bulk operations, and simple configuration | 2026-03-21 |
| [security_policies.md](security_policies.md) | Security-first policy with mandatory immutable audit logging | 2026-03-21 |
| [performance_requirements.md](performance_requirements.md) | Performance expectations and scalability | 2026-03-21 |
| [audit_logging.md](audit_logging.md) | Configurable audit logging with NullAuditor default, independent storage, insert/get only | 2026-03-21 |

## Core Concepts Glossary

### Reaction Target
The unique combination of `(entity_type, entity_id)` that identifies what is being reacted to. Examples: `("photo", "123")`, `("article", "abc-456")`. See [entity_association.md](entity_association.md) for details.

### User Reaction
The unique combination of `(user_id, entity_type, entity_id)` representing a specific user's reaction to a specific Reaction Target. At most one active reaction (LIKE or DISLIKE) can exist per User Reaction. See [user_interactions.md](user_interactions.md) for details.

### Idempotency of LIKE/DISLIKE
LIKE and DISLIKE operations are idempotent. If a LIKE already exists for a User Reaction, attempting another LIKE returns an error (ErrDuplicateReaction) with no state change. Same for DISLIKE. See [reaction_management.md](reaction_management.md) for details.

### Audit Storage Independence
The audit package operates independently from reaction storage, supporting separate database configurations. Audit entries are append-only with only Insert and Get operations exposed. See [audit_logging.md](audit_logging.md) for details.

### High Concurrency Design
All module components are designed for high-load, high-concurrency environments. Lock-free patterns preferred, minimal lock scope when necessary, no global locks. See [architecture.md](architecture.md) for details.

### Simple Configuration API
The module exposes a simple, intuitive API with sensible defaults, fluent interface options, and minimal required configuration for quick adoption. See [api_interface.md](api_interface.md) for details.

### Module Scope
This module exclusively handles user reactions. Authentication, authorization, and reaction target management are the responsibility of the consuming application. See [reaction_management.md](reaction_management.md) for details.

### Caching Layer
Optional in-process caching layer improves read performance and reduces database load. Configurable TTL, LRU eviction, automatic invalidation on writes. See [api_interface.md](api_interface.md) and [architecture.md](architecture.md) for details.

### Bulk Operations
API supports bulk operations for efficient batch processing: GetUserReactionsBulk, GetEntityCountsBulk, GetMultipleUserReactions. See [api_interface.md](api_interface.md) for details.

### Multi-Database Support
Supports PostgreSQL, MariaDB, SQLite, MongoDB, Cassandra, Redis, and In-Memory with database-specific optimizations. See [data_persistence.md](data_persistence.md) for details.

### NullAuditor Default
Audit logging defaults to NullAuditor (no-op) implementation. Persistent auditing is opt-in via configuration. See [audit_logging.md](audit_logging.md) for details.

### Security-First Policy
Security is a fundamental design principle. Secure defaults, defense in depth, fail secure. See [security_policies.md](security_policies.md) for details.

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

---

**Language Convention:** All specifications are written in clear, simple English (functional focus - WHAT not HOW).
