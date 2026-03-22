# Event System Specification

## Overview

The system provides an event system that publishes notifications for all reaction-related state changes. Events enable consuming applications to integrate with external systems, implement distributed cache invalidation, trigger business logic, and maintain real-time audit trails.

Events are emitted synchronously during reaction operations and may be consumed synchronously or asynchronously by registered listeners.

## Functional Requirements

### Requirement 1: Event Types

The system defines event types covering all reaction operations.

**Reaction Event Types:**

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `ReactionAdded` | New reaction created | AddReaction when no previous reaction exists |
| `ReactionReplaced` | Reaction replaced with different type | AddReaction when previous reaction exists |
| `ReactionRemoved` | Reaction deleted | RemoveReaction |

**System Event Types:**

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `EntityCountsUpdated` | Aggregate counts changed | Any reaction operation |
| `BulkReactionsProcessed` | Batch operation completed | Bulk operation completed |
| `CacheInvalidated` | Cache entry removed | Cache invalidation triggered |
| `AuditEntryCreated` | Audit record persisted | Audit log entry created |

### Requirement 2: Event Payload Schema

Each event carries a standardized payload.

**Base Event Structure:**
```go
type Event struct {
    Type      string
    Timestamp time.Time
    Version   int
}
```

**ReactionEvent Structure:**
```go
type ReactionEvent struct {
    Event
    UserID           string
    EntityType       string
    EntityID         string
    ReactionType     string
    PreviousReaction *string
}
```

**EntityCountsUpdated Structure:**
```go
type EntityCountsUpdated struct {
    Event
    EntityType     string
    EntityID       string
    CountsByType   map[string]int64
    TotalReactions int64
    Deltas         map[string]int64
}
```

### Requirement 3: Event Subscription

The system provides subscription mechanisms.

**Subscription Types:**
- **Synchronous:** Handlers execute in the same goroutine
- **Asynchronous:** Events are queued and processed by separate goroutines
- **Filtered:** Subscribe to specific event types

**Subscription Interface:**
```go
type EventSubscriber interface {
    HandleEvent(ctx context.Context, event Event) error
}

type EventFilter struct {
    EventTypes  []string
    EntityTypes []string
    UserIDs     []string
}
```

### Requirement 4: Event Delivery Guarantees

The system provides configurable delivery guarantees.

- **At-Least-Once (Default):** Events guaranteed to be delivered at least once
- **At-Most-Once:** Events delivered zero or one times
- **Exactly-Once:** Events delivered exactly once (requires deduplication)

### Requirement 5: Event Ordering

The system maintains event ordering within a context.

- **Per-User Ordering:** Events for a specific user delivered in order
- **Per-Entity Ordering:** Events affecting a specific entity delivered in order
- **Causal Ordering:** If Event A causes Event B, A is delivered before B

### Requirement 6: Event Performance

The event system operates with minimal performance impact.

- Event emission latency: <1ms synchronous, <0.1ms asynchronous
- Event throughput: 10,000+ events/second
- Memory overhead: <100 bytes per queued event

### Requirement 7: Event Configuration

The system provides configuration options.

```go
type EventConfig struct {
    Enabled          bool
    SyncTimeout      time.Duration
    AsyncQueueSize   int
    AsyncWorkers     int
    DeliveryGuarantee DeliveryGuarantee
    DisabledEvents   []string
}
```

## Constraints and Limitations

1. **No Persistence:** Events are ephemeral and not persisted.
2. **In-Process Only:** Event system is in-process only.
3. **No Replay:** Events cannot be replayed from history.
4. **No Webhooks:** HTTP webhooks are not provided.

## Acceptance Criteria

1. **AC1:** All event types are defined
2. **AC2:** Event payload schemas include required fields
3. **AC3:** Synchronous subscription executes handlers in calling goroutine
4. **AC4:** Asynchronous subscription executes handlers in background goroutines
5. **AC5:** Event filtering works by event type, entity type, and user ID
6. **AC6:** At-least-once delivery guarantee is default
7. **AC7:** Events are emitted in causal order
8. **AC8:** Event emission adds <1ms latency
9. **AC9:** Event system handles 10,000+ events/second throughput
10. **AC10:** Event system can be disabled via configuration
