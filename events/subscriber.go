package events

import (
	"context"
	"time"
)

// EventProvider is an interface for types that can provide a base Event.
// This allows specific event types (like ReactionEvent) to be matched by filters.
type EventProvider interface {
	// BaseEvent returns the base Event.
	BaseEvent() Event

	// GetEntityType returns the entity type if available, empty string otherwise.
	GetEntityType() string

	// GetUserID returns the user ID if available, empty string otherwise.
	GetUserID() string
}

// HandlerFunc is an adapter to allow the use of ordinary functions as subscribers.
type HandlerFunc func(ctx context.Context, event Event) error

// HandleEvent calls f(ctx, event).
func (f HandlerFunc) HandleEvent(ctx context.Context, event Event) error {
	return f(ctx, event)
}

// Subscriber is the interface for event subscribers.
type Subscriber interface {
	// HandleEvent processes the given event.
	// Implementations should respect context cancellation.
	// Errors are logged but do not affect event delivery to other subscribers.
	HandleEvent(ctx context.Context, event Event) error
}

// Filter defines criteria for filtering events.
type Filter struct {
	// EventTypes limits subscription to specific event types.
	// Empty slice means all event types.
	EventTypes []Type

	// EntityTypes limits subscription to specific entity types.
	// Empty slice means all entity types.
	EntityTypes []string

	// UserIDs limits subscription to events from specific users.
	// Empty slice means all users.
	UserIDs []string
}

// Matches checks if an event provider matches the filter criteria.
func (f Filter) Matches(provider EventProvider) bool {
	// Check event type
	if len(f.EventTypes) > 0 {
		found := false
		baseEvent := provider.BaseEvent()
		for _, et := range f.EventTypes {
			if et == baseEvent.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check entity type
	if len(f.EntityTypes) > 0 {
		found := false
		entityType := provider.GetEntityType()
		for _, et := range f.EntityTypes {
			if et == entityType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check user ID
	if len(f.UserIDs) > 0 {
		found := false
		userID := provider.GetUserID()
		for _, uid := range f.UserIDs {
			if uid == userID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Config configures the event system behavior.
type Config struct {
	// Enabled controls whether the event system is active.
	// Default: true
	Enabled bool

	// SyncTimeout is the maximum time allowed for synchronous handlers.
	// Default: 1 second
	SyncTimeout time.Duration

	// AsyncQueueSize is the size of the async event queue.
	// Default: 1000
	AsyncQueueSize int

	// AsyncWorkers is the number of worker goroutines for async processing.
	// Default: 5
	AsyncWorkers int

	// DisabledEvents contains event types that should not be emitted.
	DisabledEvents []Type
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		SyncTimeout:    time.Second,
		AsyncQueueSize: 1000,
		AsyncWorkers:   5,
		DisabledEvents: nil,
	}
}

// IsEventDisabled checks if an event type is disabled.
func (c Config) IsEventDisabled(eventType Type) bool {
	for _, et := range c.DisabledEvents {
		if et == eventType {
			return true
		}
	}
	return false
}
