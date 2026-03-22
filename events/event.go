// Package events provides an event system for reaction state changes.
// Events enable integration with external systems, cache invalidation,
// and real-time business logic triggers.
package events

import (
	"time"
)

// Type represents the type of event.
type Type string

// Event type constants.
const (
	// Reaction Event Types
	TypeReactionAdded    Type = "ReactionAdded"
	TypeReactionReplaced Type = "ReactionReplaced"
	TypeReactionRemoved  Type = "ReactionRemoved"

	// System Event Types
	TypeEntityCountsUpdated  Type = "EntityCountsUpdated"
	TypeBulkReactionsProcessed Type = "BulkReactionsProcessed"
	TypeCacheInvalidated     Type = "CacheInvalidated"
	TypeAuditEntryCreated    Type = "AuditEntryCreated"
)

// Event is the base event structure.
// All event types embed this struct.
type Event struct {
	// Type is the event type identifier.
	Type Type

	// Timestamp is when the event was emitted, in UTC.
	Timestamp time.Time

	// Version is the event schema version.
	Version int
}

// NewEvent creates a new base event with the given type.
func NewEvent(eventType Type) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Version:   1,
	}
}

// BaseEvent returns the base Event (implements EventProvider).
func (e Event) BaseEvent() Event {
	return e
}

// GetEntityType returns empty string for base Event (implements EventProvider).
func (e Event) GetEntityType() string {
	return ""
}

// GetUserID returns empty string for base Event (implements EventProvider).
func (e Event) GetUserID() string {
	return ""
}

// ReactionEvent is emitted when a reaction is added, replaced, or removed.
type ReactionEvent struct {
	Event

	// UserID identifies the user who performed the action.
	UserID string

	// EntityType is the type of entity affected.
	EntityType string

	// EntityID is the identifier of the entity affected.
	EntityID string

	// ReactionType is the reaction type after the operation.
	ReactionType string

	// PreviousReaction is the reaction type before the operation.
	// Nil if no previous reaction existed.
	PreviousReaction *string
}

// NewReactionEvent creates a new ReactionEvent.
func NewReactionEvent(eventType Type, userID, entityType, entityID, reactionType string, previous *string) ReactionEvent {
	return ReactionEvent{
		Event:            NewEvent(eventType),
		UserID:           userID,
		EntityType:       entityType,
		EntityID:         entityID,
		ReactionType:     reactionType,
		PreviousReaction: previous,
	}
}

// GetEntityType returns the entity type (implements EventProvider).
func (r ReactionEvent) GetEntityType() string {
	return r.EntityType
}

// GetUserID returns the user ID (implements EventProvider).
func (r ReactionEvent) GetUserID() string {
	return r.UserID
}

// CountDelta represents the change in count for a specific reaction type.
type CountDelta struct {
	// ReactionType is the type of reaction.
	ReactionType string

	// OldCount is the count before the operation.
	OldCount int64

	// NewCount is the count after the operation.
	NewCount int64
}

// Delta returns the change in count (NewCount - OldCount).
func (d CountDelta) Delta() int64 {
	return d.NewCount - d.OldCount
}

// EntityCountsUpdated is emitted when aggregate counts change for an entity.
type EntityCountsUpdated struct {
	Event

	// EntityType is the type of entity affected.
	EntityType string

	// EntityID is the identifier of the entity affected.
	EntityID string

	// CountsByType maps reaction types to their current counts.
	CountsByType map[string]int64

	// TotalReactions is the total number of reactions for the entity.
	TotalReactions int64

	// Deltas contains the count changes for each reaction type.
	Deltas []CountDelta
}

// NewEntityCountsUpdated creates a new EntityCountsUpdated event.
func NewEntityCountsUpdated(entityType, entityID string, counts map[string]int64, deltas []CountDelta) EntityCountsUpdated {
	total := int64(0)
	for _, count := range counts {
		total += count
	}

	return EntityCountsUpdated{
		Event:          NewEvent(TypeEntityCountsUpdated),
		EntityType:     entityType,
		EntityID:       entityID,
		CountsByType:   counts,
		TotalReactions: total,
		Deltas:         deltas,
	}
}

// GetEntityType returns the entity type (implements EventProvider).
func (e EntityCountsUpdated) GetEntityType() string {
	return e.EntityType
}

// GetUserID returns empty string as this event is not user-specific.
func (e EntityCountsUpdated) GetUserID() string {
	return ""
}

// CacheInvalidated is emitted when a cache entry is invalidated.
type CacheInvalidated struct {
	Event

	// EntityType is the type of entity affected.
	EntityType string

	// EntityID is the identifier of the entity affected.
	EntityID string

	// Keys contains the cache keys that were invalidated.
	Keys []string
}

// NewCacheInvalidated creates a new CacheInvalidated event.
func NewCacheInvalidated(entityType, entityID string, keys []string) CacheInvalidated {
	return CacheInvalidated{
		Event:      NewEvent(TypeCacheInvalidated),
		EntityType: entityType,
		EntityID:   entityID,
		Keys:       keys,
	}
}

// GetEntityType returns the entity type (implements EventProvider).
func (c CacheInvalidated) GetEntityType() string {
	return c.EntityType
}

// GetUserID returns empty string as this event is not user-specific.
func (c CacheInvalidated) GetUserID() string {
	return ""
}

// AuditEntryCreated is emitted when an audit entry is created.
type AuditEntryCreated struct {
	Event

	// AuditID is the identifier of the audit entry.
	AuditID string

	// Operation is the type of operation audited.
	Operation string

	// UserID identifies the user who performed the operation.
	UserID string

	// EntityType is the type of entity affected.
	EntityType string

	// EntityID is the identifier of the entity affected.
	EntityID string
}

// NewAuditEntryCreated creates a new AuditEntryCreated event.
func NewAuditEntryCreated(auditID, operation, userID, entityType, entityID string) AuditEntryCreated {
	return AuditEntryCreated{
		Event:      NewEvent(TypeAuditEntryCreated),
		AuditID:    auditID,
		Operation:  operation,
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
	}
}

// GetEntityType returns the entity type (implements EventProvider).
func (a AuditEntryCreated) GetEntityType() string {
	return a.EntityType
}

// GetUserID returns the user ID (implements EventProvider).
func (a AuditEntryCreated) GetUserID() string {
	return a.UserID
}

// BulkReactionsProcessed is emitted when a batch operation completes.
type BulkReactionsProcessed struct {
	Event

	// Operation is the bulk operation type.
	Operation string

	// SuccessCount is the number of successful operations.
	SuccessCount int

	// FailureCount is the number of failed operations.
	FailureCount int

	// Errors contains error messages for failed operations.
	Errors []string
}

// NewBulkReactionsProcessed creates a new BulkReactionsProcessed event.
func NewBulkReactionsProcessed(operation string, success, failure int, errors []string) BulkReactionsProcessed {
	return BulkReactionsProcessed{
		Event:        NewEvent(TypeBulkReactionsProcessed),
		Operation:    operation,
		SuccessCount: success,
		FailureCount: failure,
		Errors:       errors,
	}
}

// GetEntityType returns empty string as this event is not entity-specific.
func (b BulkReactionsProcessed) GetEntityType() string {
	return ""
}

// GetUserID returns empty string as this event is not user-specific.
func (b BulkReactionsProcessed) GetUserID() string {
	return ""
}
