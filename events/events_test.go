package events

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// mockSubscriber is a test subscriber that counts events.
type mockSubscriber struct {
	count      atomic.Int32
	lastEvent  Event
	shouldError bool
	filter      Filter
}

func (m *mockSubscriber) HandleEvent(ctx context.Context, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.count.Add(1)
	m.lastEvent = event
	if m.shouldError {
		return context.DeadlineExceeded
	}
	return nil
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(TypeReactionAdded)

	if event.Type != TypeReactionAdded {
		t.Errorf("expected type ReactionAdded, got %s", event.Type)
	}
	if event.Version != 1 {
		t.Errorf("expected version 1, got %d", event.Version)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestNewReactionEvent(t *testing.T) {
	prev := "LIKE"
	event := NewReactionEvent(TypeReactionReplaced, "user1", "photo", "123", "LOVE", &prev)

	if event.Type != TypeReactionReplaced {
		t.Errorf("expected type ReactionReplaced, got %s", event.Type)
	}
	if event.UserID != "user1" {
		t.Errorf("expected userID user1, got %s", event.UserID)
	}
	if event.EntityType != "photo" {
		t.Errorf("expected entityType photo, got %s", event.EntityType)
	}
	if event.EntityID != "123" {
		t.Errorf("expected entityID 123, got %s", event.EntityID)
	}
	if event.ReactionType != "LOVE" {
		t.Errorf("expected reactionType LOVE, got %s", event.ReactionType)
	}
	if event.PreviousReaction == nil || *event.PreviousReaction != "LIKE" {
		t.Error("expected previousReaction to be LIKE")
	}
}

func TestNewEntityCountsUpdated(t *testing.T) {
	counts := map[string]int64{
		"LIKE": 5,
		"LOVE": 3,
	}
	deltas := []CountDelta{
		{ReactionType: "LIKE", OldCount: 4, NewCount: 5},
		{ReactionType: "LOVE", OldCount: 2, NewCount: 3},
	}

	event := NewEntityCountsUpdated("photo", "123", counts, deltas)

	if event.Type != TypeEntityCountsUpdated {
		t.Errorf("expected type EntityCountsUpdated, got %s", event.Type)
	}
	if event.EntityType != "photo" {
		t.Errorf("expected entityType photo, got %s", event.EntityType)
	}
	if event.EntityID != "123" {
		t.Errorf("expected entityID 123, got %s", event.EntityID)
	}
	if event.TotalReactions != 8 {
		t.Errorf("expected total 8, got %d", event.TotalReactions)
	}
	if len(event.CountsByType) != 2 {
		t.Errorf("expected 2 count types, got %d", len(event.CountsByType))
	}
}

func TestCountDelta_Delta(t *testing.T) {
	d := CountDelta{ReactionType: "LIKE", OldCount: 5, NewCount: 10}
	if d.Delta() != 5 {
		t.Errorf("expected delta 5, got %d", d.Delta())
	}

	d = CountDelta{ReactionType: "LIKE", OldCount: 10, NewCount: 5}
	if d.Delta() != -5 {
		t.Errorf("expected delta -5, got %d", d.Delta())
	}
}

func TestFilter_Matches(t *testing.T) {
	tests := []struct {
		name     string
		filter   Filter
		event    EventProvider
		expected bool
	}{
		{
			name:     "empty filter matches all",
			filter:   Filter{},
			event:    NewEvent(TypeReactionAdded),
			expected: true,
		},
		{
			name:     "matches specific type",
			filter:   Filter{EventTypes: []Type{TypeReactionAdded}},
			event:    NewEvent(TypeReactionAdded),
			expected: true,
		},
		{
			name:     "does not match different type",
			filter:   Filter{EventTypes: []Type{TypeReactionAdded}},
			event:    NewEvent(TypeReactionRemoved),
			expected: false,
		},
		{
			name:     "matches one of multiple types",
			filter:   Filter{EventTypes: []Type{TypeReactionAdded, TypeReactionReplaced}},
			event:    NewEvent(TypeReactionReplaced),
			expected: true,
		},
		{
			name:     "matches by entity type",
			filter:   Filter{EntityTypes: []string{"photo"}},
			event:    NewReactionEvent(TypeReactionAdded, "user1", "photo", "123", "LIKE", nil),
			expected: true,
		},
		{
			name:     "does not match different entity type",
			filter:   Filter{EntityTypes: []string{"video"}},
			event:    NewReactionEvent(TypeReactionAdded, "user1", "photo", "123", "LIKE", nil),
			expected: false,
		},
		{
			name:     "matches by user ID",
			filter:   Filter{UserIDs: []string{"user1"}},
			event:    NewReactionEvent(TypeReactionAdded, "user1", "photo", "123", "LIKE", nil),
			expected: true,
		},
		{
			name:     "does not match different user",
			filter:   Filter{UserIDs: []string{"user2"}},
			event:    NewReactionEvent(TypeReactionAdded, "user1", "photo", "123", "LIKE", nil),
			expected: false,
		},
		{
			name:     "combined filters match",
			filter:   Filter{EventTypes: []Type{TypeReactionAdded}, EntityTypes: []string{"photo"}, UserIDs: []string{"user1"}},
			event:    NewReactionEvent(TypeReactionAdded, "user1", "photo", "123", "LIKE", nil),
			expected: true,
		},
		{
			name:     "combined filters mismatch",
			filter:   Filter{EventTypes: []Type{TypeReactionAdded}, EntityTypes: []string{"photo"}, UserIDs: []string{"user1"}},
			event:    NewReactionEvent(TypeReactionRemoved, "user1", "photo", "123", "LIKE", nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Matches(tt.event)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}


func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true")
	}
	if config.SyncTimeout != time.Second {
		t.Errorf("expected SyncTimeout 1s, got %v", config.SyncTimeout)
	}
	if config.AsyncQueueSize != 1000 {
		t.Errorf("expected AsyncQueueSize 1000, got %d", config.AsyncQueueSize)
	}
	if config.AsyncWorkers != 5 {
		t.Errorf("expected AsyncWorkers 5, got %d", config.AsyncWorkers)
	}
}

func TestConfig_IsEventDisabled(t *testing.T) {
	config := Config{
		DisabledEvents: []Type{TypeReactionAdded, TypeReactionRemoved},
	}

	if !config.IsEventDisabled(TypeReactionAdded) {
		t.Error("expected ReactionAdded to be disabled")
	}
	if !config.IsEventDisabled(TypeReactionRemoved) {
		t.Error("expected ReactionRemoved to be disabled")
	}
	if config.IsEventDisabled(TypeReactionReplaced) {
		t.Error("expected ReactionReplaced to be enabled")
	}
}

func TestBus(t *testing.T) {
	t.Run("synchronous subscription", func(t *testing.T) {
		bus := NewBus(DefaultConfig())
		defer bus.Close()

		sub := &mockSubscriber{}
		bus.SubscribeSync(sub, Filter{})

		event := NewEvent(TypeReactionAdded)
		bus.Emit(event)

		// Give a moment for sync processing
		time.Sleep(10 * time.Millisecond)

		if sub.count.Load() != 1 {
			t.Errorf("expected 1 event, got %d", sub.count.Load())
		}
	})

	t.Run("asynchronous subscription", func(t *testing.T) {
		bus := NewBus(DefaultConfig())
		defer bus.Close()

		sub := &mockSubscriber{}
		bus.SubscribeAsync(sub, Filter{})

		event := NewEvent(TypeReactionAdded)
		bus.Emit(event)

		// Wait for async processing
		time.Sleep(100 * time.Millisecond)

		if sub.count.Load() != 1 {
			t.Errorf("expected 1 event, got %d", sub.count.Load())
		}
	})

	t.Run("filtering works", func(t *testing.T) {
		bus := NewBus(DefaultConfig())
		defer bus.Close()

		photoSub := &mockSubscriber{}
		videoSub := &mockSubscriber{}

		bus.SubscribeSync(photoSub, Filter{EntityTypes: []string{"photo"}})
		bus.SubscribeSync(videoSub, Filter{EntityTypes: []string{"video"}})

		photoEvent := NewReactionEvent(TypeReactionAdded, "user1", "photo", "1", "LIKE", nil)
		videoEvent := NewReactionEvent(TypeReactionAdded, "user2", "video", "2", "LOVE", nil)

		bus.Emit(photoEvent)
		bus.Emit(videoEvent)

		time.Sleep(10 * time.Millisecond)

		if photoSub.count.Load() != 1 {
			t.Errorf("expected photoSub to receive 1 event, got %d", photoSub.count.Load())
		}
		if videoSub.count.Load() != 1 {
			t.Errorf("expected videoSub to receive 1 event, got %d", videoSub.count.Load())
		}
	})

	t.Run("disabled events are not emitted", func(t *testing.T) {
		config := DefaultConfig()
		config.DisabledEvents = []Type{TypeReactionAdded}

		bus := NewBus(config)
		defer bus.Close()

		sub := &mockSubscriber{}
		bus.SubscribeSync(sub, Filter{})

		bus.Emit(NewEvent(TypeReactionAdded))
		bus.Emit(NewEvent(TypeReactionRemoved))

		time.Sleep(10 * time.Millisecond)

		if sub.count.Load() != 1 {
			t.Errorf("expected 1 event (only ReactionRemoved), got %d", sub.count.Load())
		}
	})

	t.Run("disabled bus does not emit", func(t *testing.T) {
		config := DefaultConfig()
		config.Enabled = false

		bus := NewBus(config)
		defer bus.Close()

		sub := &mockSubscriber{}
		bus.SubscribeSync(sub, Filter{})

		bus.Emit(NewEvent(TypeReactionAdded))

		time.Sleep(10 * time.Millisecond)

		if sub.count.Load() != 0 {
			t.Errorf("expected 0 events when disabled, got %d", sub.count.Load())
		}
	})

	t.Run("errors do not block other subscribers", func(t *testing.T) {
		bus := NewBus(DefaultConfig())
		defer bus.Close()

		errorSub := &mockSubscriber{shouldError: true}
		okSub := &mockSubscriber{}

		bus.SubscribeSync(errorSub, Filter{})
		bus.SubscribeSync(okSub, Filter{})

		bus.Emit(NewEvent(TypeReactionAdded))

		time.Sleep(10 * time.Millisecond)

		if okSub.count.Load() != 1 {
			t.Errorf("expected okSub to receive 1 event, got %d", okSub.count.Load())
		}
	})

	t.Run("subscriber count", func(t *testing.T) {
		bus := NewBus(DefaultConfig())
		defer bus.Close()

		if bus.SubscriberCount() != 0 {
			t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
		}

		bus.SubscribeSync(&mockSubscriber{}, Filter{})
		if bus.SubscriberCount() != 1 {
			t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
		}

		bus.SubscribeAsync(&mockSubscriber{}, Filter{})
		if bus.SubscriberCount() != 2 {
			t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
		}
	})

	t.Run("close is idempotent", func(t *testing.T) {
		bus := NewBus(DefaultConfig())

		if err := bus.Close(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := bus.Close(); err != nil {
			t.Errorf("unexpected error on second close: %v", err)
		}

		if !bus.IsClosed() {
			t.Error("expected bus to be closed")
		}
	})

	t.Run("handler function subscription", func(t *testing.T) {
		bus := NewBus(DefaultConfig())
		defer bus.Close()

		var count atomic.Int32
		fn := HandlerFunc(func(ctx context.Context, event Event) error {
			count.Add(1)
			return nil
		})

		bus.SubscribeSyncFunc(Filter{}, fn)
		bus.Emit(NewEvent(TypeReactionAdded))

		time.Sleep(10 * time.Millisecond)

		if count.Load() != 1 {
			t.Errorf("expected handler to be called once, got %d", count.Load())
		}
	})

	t.Run("multiple events received", func(t *testing.T) {
		bus := NewBus(DefaultConfig())
		defer bus.Close()

		sub := &mockSubscriber{}
		bus.SubscribeSync(sub, Filter{})

		for i := 0; i < 10; i++ {
			bus.Emit(NewEvent(TypeReactionAdded))
		}

		time.Sleep(50 * time.Millisecond)

		if sub.count.Load() != 10 {
			t.Errorf("expected 10 events, got %d", sub.count.Load())
		}
	})
}

func TestBus_AsyncTimeout(t *testing.T) {
	config := DefaultConfig()
	config.SyncTimeout = 50 * time.Millisecond

	bus := NewBus(config)
	defer bus.Close()

	// Create a subscriber that takes longer than the timeout
	slowSub := HandlerFunc(func(ctx context.Context, event Event) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	bus.SubscribeSync(slowSub, Filter{})

	// This should not block indefinitely
	done := make(chan bool)
	go func() {
		bus.Emit(NewEvent(TypeReactionAdded))
		done <- true
	}()

	select {
	case <-done:
		// Success - emit completed within timeout
	case <-time.After(500 * time.Millisecond):
		t.Error("emit took too long, should have timed out")
	}
}

func BenchmarkBus_Emit(b *testing.B) {
	bus := NewBus(DefaultConfig())
	defer bus.Close()

	sub := &mockSubscriber{}
	bus.SubscribeSync(sub, Filter{})

	event := NewEvent(TypeReactionAdded)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Emit(event)
	}
}
