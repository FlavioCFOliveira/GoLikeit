# GoLikeit Specification Index

This directory contains functional specifications for the GoLikeit module - a Go library for managing user reactions on entities through a configurable, abstract reaction system.

## Specification-First Policy

**CRITICAL:** No code may be written without a corresponding functional specification. The specification defines WHAT the system does; implementation defines HOW it is done.

```
User Request → spec-orchestrator → specification/ → go-elite-developer → Implementation
```

## Functional Blocks

| Specification | Purpose |
|---------------|---------|
| [architecture.md](architecture.md) | System design with high concurrency support, caching layer, and component boundaries |
| [reaction_management.md](reaction_management.md) | Core reaction operations with configurable reaction types and replacement semantics |
| [entity_association.md](entity_association.md) | Reaction Target definition (entity_type + entity_id) |
| [user_interactions.md](user_interactions.md) | User Reaction definition (user_id + Reaction Target) |
| [data_persistence.md](data_persistence.md) | Data storage with PostgreSQL, MariaDB, SQLite, MongoDB, Cassandra, Redis, and In-Memory support |
| [api_interface.md](api_interface.md) | Public API with reaction type configuration, caching, bulk operations |
| [security_policies.md](security_policies.md) | Security-first policy with mandatory immutable audit logging |
| [performance_requirements.md](performance_requirements.md) | Performance expectations and scalability |
| [audit_logging.md](audit_logging.md) | Configurable audit logging with NullAuditor default |
| [event_system.md](event_system.md) | Event system for reaction state changes and system events |
| [rate_limiting.md](rate_limiting.md) | Configurable per-user rate limiting using sliding window algorithm |
| [test_strategy.md](test_strategy.md) | Functional strategy for testing and validating implementations |

## Core Concepts

### Reaction Target
The unique combination of `(entity_type, entity_id)` that identifies what is being reacted to. Examples: `("photo", "123")`, `("article", "abc-456")`.

### User Reaction
The unique combination of `(user_id, entity_type, entity_id)` representing a user's reaction to a specific Reaction Target. Only **one** reaction can exist per User Reaction - adding a new reaction replaces any existing reaction.

### Reaction Type
A string identifier representing a type of reaction. Format: `^[A-Z0-9_-]+$` (uppercase letters, digits, hyphens, underscores only). The module has no predefined reaction types - all types are configured at initialization. Examples: `LIKE`, `DISLIKE`, `LOVE`, `ANGRY`.

### Replacement Semantics
When a user adds a reaction to a target where they already have a reaction, the previous reaction is **replaced** (not converted). The previous reaction type is permanently lost.

### Audit Storage Independence
The audit package operates independently from reaction storage, supporting separate database configurations. Audit entries are append-only with only Insert and Get operations exposed.

### Caching Layer
Optional in-process caching layer improves read performance. Configurable TTL, LRU eviction, automatic invalidation on writes. Cache invalidation occurs at Reaction Target granularity.

### Security-First Policy
Security is a fundamental design principle. Secure defaults, defense in depth, fail secure. All state-changing operations create immutable audit entries. No soft delete - RemoveReaction performs hard deletes; audit log maintains history.

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

---

**Language Convention:** All specifications are written in clear, simple English (functional focus - WHAT not HOW).
