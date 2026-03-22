# Event System Specification

## Overview

The system shall provide a comprehensive event system that publishes notifications for all reaction-related state changes. Events enable consuming applications to integrate with external systems, implement distributed cache invalidation, trigger business logic, and maintain real-time audit trails.

Events are emitted synchronously during reaction operations and may be consumed synchronously or asynchronously by registered listeners. The event system is designed for high-throughput scenarios with minimal latency impact.

## Functional Requirements

### Requirement 1: Event Types

**Description:** The system shall define a comprehensive set of event types covering all reaction operations and system-level changes.

**Reaction Event Types:**

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `ReactionAdded` | New reaction was created | AddReaction when no previous reaction exists |
| `ReactionReplaced` | Reaction was replaced with different type | AddReaction when previous reaction exists |
| `ReactionRemoved` | Reaction was deleted | RemoveReaction |

**System Event Types:**

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `EntityCountsUpdated` | Aggregate counts changed | Any reaction operation that modifies counts |
| `BulkReactionsProcessed` | Batch operation completed | Bulk operation completed |
| `CacheInvalidated` | Cache entry removed | Cache invalidation triggered by write operation |
| `AuditEntryCreated` | Audit record persisted | Audit log entry created |

**Event Type Hierarchy:**
```
Event (base)
├── ReactionEvent
│   ├── ReactionAdded
│   ├── ReactionReplaced
│   └── ReactionRemoved
├── EntityEvent
│   └── EntityCountsUpdated
└── SystemEvent
    ├── BulkReactionsProcessed
    ├── CacheInvalidated
    └── AuditEntryCreated
```

### Requirement 2: Event Payload Schema

**Description:** Each event shall carry a standardized payload containing all relevant information about the event.

**Base Event Structure:**
```go
type Event struct {
    Type      string    // Event type identifier
    Timestamp time.Time // UTC timestamp of event occurrence
    Version   int       // Schema version for evolution
}
```

**ReactionEvent Structure:**
```go
type ReactionEvent struct {
    Event
    UserID           string  // User who performed the action
    EntityType       string  // Type of entity reacted to
    EntityID         string  // Entity instance identifier
    ReactionType     string  // Current reaction type (e.g., "LIKE", "LOVE")
    PreviousReaction *string // Pointer to previous reaction type, nil if no previous reaction
    Metadata         map[string]interface{} // Optional metadata
}
```

**EntityCountsUpdated Structure:**
```go
type EntityCountsUpdated struct {
    Event
    EntityType    string            // Type of entity
    EntityID      string            // Entity instance identifier
    CountsByType  map[string]int64  // Current count per reaction type
    TotalReactions int64            // Total reactions across all types
    Deltas        map[string]int64   // Change in counts from operation (+1, -1, 0 per type)
}
```

**BulkReactionsProcessed Structure:**
```go
type BulkReactionsProcessed struct {
    Event
    OperationType string   // Type of bulk operation
    UserID        string   // User who initiated (if applicable)
    EntityCount   int      // Number of entities affected
    DurationMs    int64    // Processing duration in milliseconds
}
```

**CacheInvalidated Structure:**
```go
type CacheInvalidated struct {
    Event
    CacheKey         string   // Key that was invalidated
    InvalidationType string   // "user", "entity", "bulk"
    Reason           string   // Why cache was invalidated
}
```

**AuditEntryCreated Structure:**
```go
type AuditEntryCreated struct {
    Event
    AuditEntryID string    // Reference to audit entry
    Operation    string      // Operation type (ADD, REPLACE, REMOVE)
    UserID       string      // User who performed action
    Success      bool        // Whether audit write succeeded
}
```

### Requirement 3: Event Subscription Mechanism

**Description:** The system shall provide flexible subscription mechanisms for consuming events.

**Subscription Types:**

1. **Synchronous Subscription:**
   - Event handlers execute in the same goroutine as the triggering operation
   - Blocking handlers delay the operation
   - Suitable for immediate cache invalidation or validation

2. **Asynchronous Subscription:**
   - Events are queued and processed by separate goroutines
   - Non-blocking for the triggering operation
   - Suitable for external notifications or long-running processing

3. **Filtered Subscription:**
   - Subscribe to specific event types only
   - Filter by entity type or user ID patterns
   - Reduce noise for specific use cases

**Subscription Interface:**
```go
type EventSubscriber interface {
    HandleEvent(ctx context.Context, event Event) error
}

type EventFilter struct {
    EventTypes   []string // Filter by event type (empty = all)
    EntityTypes  []string // Filter by entity type (empty = all)
    UserIDs      []string // Filter by user ID (empty = all)
}
```

**Subscription Registration:**
```go
// Subscribe to all events synchronously
client.Subscribe(subscriber)

// Subscribe to specific events asynchronously
client.SubscribeAsync(subscriber, EventFilter{
    EventTypes: []string{"ReactionAdded", "ReactionRemoved"},
})

// Subscribe with custom filter
client.SubscribeAsync(subscriber, EventFilter{
    EventTypes:  []string{"EntityCountsUpdated"},
    EntityTypes: []string{"photo", "video"},
})
```

### Requirement 4: Event Delivery Guarantees

**Description:** The system shall provide configurable delivery guarantees for events.

**Delivery Guarantees:**

1. **At-Least-Once (Default):**
   - Events are guaranteed to be delivered at least once
   - Duplicate events possible in failure scenarios
   - Suitable for idempotent operations

2. **At-Most-Once:**
   - Events are delivered zero or one times
   - No duplicates, but possible loss on failure
   - Suitable for best-effort notifications

3. **Exactly-Once:**
   - Events are delivered exactly once
   - Requires deduplication mechanism
   - Higher overhead, suitable for critical operations

**Failure Handling:**
- Synchronous subscribers: Failure stops operation, retry configurable
- Asynchronous subscribers: Failure logged, retry with backoff
- Dead letter queue for failed async deliveries

### Requirement 5: Event Ordering and Sequencing

**Description:** The system shall maintain event ordering within a context.

**Ordering Guarantees:**

1. **Per-User Ordering:**
   - Events for a specific user are delivered in order of occurrence
   - Important for tracking user action sequences

2. **Per-Entity Ordering:**
   - Events affecting a specific entity are delivered in order
   - Ensures count consistency in listeners

3. **Causal Ordering:**
   - If Event A causes Event B, A is delivered before B
   - Example: ReactionAdded → EntityCountsUpdated → AuditEntryCreated

**No Global Ordering:**
- Events from different users/entities may be delivered out of order
- Subscribers must handle out-of-order events idempotently

### Requirement 6: Event Performance

**Description:** The event system shall operate with minimal performance impact.

**Performance Requirements:**
- Event emission latency: <1ms synchronous, <0.1ms asynchronous
- Event throughput: 10,000+ events/second
- Memory overhead: <100 bytes per queued event
- Subscriber timeout: Configurable (default 5 seconds)

**Resource Management:**
- Bounded event queue for async subscribers
- Queue size configurable (default 10,000 events)
- Backpressure when queue is full
- Automatic cleanup of completed async event processing

### Requirement 7: Event Configuration

**Description:** The system shall provide configuration options for the event system.

**Configuration Options:**
```go
type EventConfig struct {
    // Enable/disable event system
    Enabled bool // Default: true

    // Synchronous subscriber timeout
    SyncTimeout time.Duration // Default: 5s

    // Asynchronous queue size
    AsyncQueueSize int // Default: 10000

    // Number of async worker goroutines
    AsyncWorkers int // Default: 10

    // Default delivery guarantee
    DeliveryGuarantee DeliveryGuarantee // Default: AtLeastOnce

    // Enable/disable specific event types
    DisabledEvents []string // Default: none

    // Event buffer size (pre-allocated)
    EventBufferSize int // Default: 1000
}
```

### Requirement 8: Event Security

**Description:** The event system shall respect security boundaries.

**Security Requirements:**
- Event payloads do not contain sensitive data (passwords, tokens)
- Event subscribers cannot modify reaction data
- Audit events contain only non-sensitive metadata
- Event filtering prevents information leakage between tenants (in multi-tenant scenarios)
- Events respect module boundaries (business layer events only)

## Event Examples

### Example 1: Add Reaction (New)

```
1. User calls AddReaction(user_123, "photo", "photo_456", "LIKE")
2. No previous reaction exists
3. Events emitted (in order):
   a. ReactionAdded {UserID: "user_123", EntityType: "photo", EntityID: "photo_456", ReactionType: "LIKE", PreviousReaction: nil}
   b. EntityCountsUpdated {EntityType: "photo", EntityID: "photo_456", CountsByType: {"LIKE": 1}, TotalReactions: 1, Deltas: {"LIKE": +1}}
   c. AuditEntryCreated {AuditEntryID: "...", Operation: "ADD", UserID: "user_123", Success: true}
   d. CacheInvalidated {CacheKey: "user_123:photo:photo_456", InvalidationType: "user", Reason: "reaction_added"}
```

### Example 2: Replace Reaction

```
1. User already has "DISLIKE" on photo_456
2. User calls AddReaction(user_123, "photo", "photo_456", "LIKE")
3. Previous reaction "DISLIKE" is replaced with "LIKE"
4. Events emitted:
   a. ReactionReplaced {UserID: "user_123", EntityType: "photo", EntityID: "photo_456", ReactionType: "LIKE", PreviousReaction: ptr("DISLIKE")}
   b. EntityCountsUpdated {EntityType: "photo", EntityID: "photo_456", CountsByType: {"LIKE": 1, "DISLIKE": 0}, TotalReactions: 1, Deltas: {"LIKE": +1, "DISLIKE": -1}}
   c. AuditEntryCreated {AuditEntryID: "...", Operation: "REPLACE", UserID: "user_123", Success: true}
   d. CacheInvalidated {CacheKey: "user_123:photo:photo_456", InvalidationType: "user", Reason: "reaction_replaced"}
```

### Example 3: Remove Reaction

```
1. User has "LIKE" on photo_456
2. User calls RemoveReaction(user_123, "photo", "photo_456")
3. Events emitted:
   a. ReactionRemoved {UserID: "user_123", EntityType: "photo", EntityID: "photo_456", ReactionType: "LIKE"}
   b. EntityCountsUpdated {EntityType: "photo", EntityID: "photo_456", CountsByType: {"LIKE": 0}, TotalReactions: 0, Deltas: {"LIKE": -1}}
   c. AuditEntryCreated {AuditEntryID: "...", Operation: "REMOVE", UserID: "user_123", Success: true}
   d. CacheInvalidated {CacheKey: "user_123:photo:photo_456", InvalidationType: "user", Reason: "reaction_removed"}
```

### Example 4: Batch Operation

```
1. User calls GetUserReactionsBulk(user_123, [entity1, entity2, entity3])
2. Events emitted:
   a. BulkReactionsProcessed {OperationType: "GetUserReactionsBulk", UserID: "user_123", EntityCount: 3, DurationMs: 15}
```

## Constraints and Limitations

1. **No Persistence:** Events are ephemeral and not persisted. If no subscriber is registered, events are dropped.

2. **In-Process Only:** Event system is in-process only. Distributed event streaming (Kafka, RabbitMQ) must be implemented by the consuming application.

3. **No Replay:** Events cannot be replayed from history. Subscribers only receive events from the time of subscription.

4. **Ordering Limits:** Ordering is best-effort under high concurrency. Applications requiring strict ordering must implement sequencing.

5. **Resource Limits:** Async event queue is bounded. If full, events may be dropped (configurable behavior).

6. **No Webhooks:** HTTP webhooks for external system notifications are not provided by the module. Applications requiring webhook functionality must implement it using the Event System as the trigger mechanism. The consuming application is responsible for HTTP delivery, retry policies, endpoint configuration, HMAC signing, and circuit breaker patterns for webhook failures.

## Relationships with Other Functional Blocks

- **[reaction_management.md](reaction_management.md):** Events are emitted during reaction operations
- **[api_interface.md](api_interface.md):** Event subscription is part of the public API
- **[cache_layer.md](cache_layer.md):** CacheInvalidated events enable distributed cache invalidation
- **[audit_logging.md](audit_logging.md):** AuditEntryCreated events provide real-time audit notifications

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version with 14 event types including Like/Dislike specific events |
| 2026-03-22 | Major | Simplified to 7 core event types; ReactionAdded/ReactionReplaced/ReactionRemoved replace Like/Dislike specific events |

## Acceptance Criteria

1. **AC1:** All 7 event types are defined and documented
2. **AC2:** Event payload schemas include all required fields
3. **AC3:** Synchronous subscription executes handlers in calling goroutine
4. **AC4:** Asynchronous subscription executes handlers in background goroutines
5. **AC5:** Event filtering works by event type, entity type, and user ID
6. **AC6:** At-least-once delivery guarantee is default
7. **AC7:** Events are emitted in causal order (causes before effects)
8. **AC8:** Event emission adds <1ms latency for synchronous handlers
9. **AC9:** Event system handles 10,000+ events/second throughput
10. **AC10:** Event queue is bounded with configurable size
11. **AC11:** Disabled event types do not emit events
12. **AC12:** Event security prevents sensitive data leakage
13. **AC13:** Examples demonstrate all major event scenarios
14. **AC14:** Event system can be disabled via configuration
