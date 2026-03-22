package events

import (
	"context"
	"sync"
)

// Bus is the central event bus that manages subscriptions and event dispatching.
// It supports synchronous and asynchronous subscriptions with filtering.
type Bus struct {
	config Config

	// syncSubscribers holds synchronous subscriptions
	syncSubscribers []syncSubscription

	// asyncSubscribers holds asynchronous subscriptions
	asyncSubscribers []asyncSubscription

	// queue is the async event queue
	queue chan Event

	// wg tracks active goroutines for graceful shutdown
	wg sync.WaitGroup

	// mu protects subscriber slices
	mu sync.RWMutex

	// closed tracks if the bus is shut down
	closed bool
}

// syncSubscription represents a synchronous event subscription.
type syncSubscription struct {
	subscriber Subscriber
	filter     Filter
}

// asyncSubscription represents an asynchronous event subscription.
type asyncSubscription struct {
	subscriber Subscriber
	filter     Filter
}

// NewBus creates a new event bus with the given configuration.
func NewBus(config Config) *Bus {
	bus := &Bus{
		config:           config,
		syncSubscribers:  make([]syncSubscription, 0),
		asyncSubscribers: make([]asyncSubscription, 0),
		queue:            make(chan Event, config.AsyncQueueSize),
	}

	// Start async workers
	if config.Enabled {
		for i := 0; i < config.AsyncWorkers; i++ {
			bus.wg.Add(1)
			go bus.asyncWorker()
		}
	}

	return bus
}

// SubscribeSync adds a synchronous subscriber.
// Synchronous subscribers execute in the same goroutine as the emitter.
func (b *Bus) SubscribeSync(subscriber Subscriber, filter Filter) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.syncSubscribers = append(b.syncSubscribers, syncSubscription{
		subscriber: subscriber,
		filter:     filter,
	})
}

// SubscribeAsync adds an asynchronous subscriber.
// Asynchronous subscribers execute in a background goroutine.
func (b *Bus) SubscribeAsync(subscriber Subscriber, filter Filter) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.asyncSubscribers = append(b.asyncSubscribers, asyncSubscription{
		subscriber: subscriber,
		filter:     filter,
	})
}

// SubscribeSyncFunc adds a synchronous handler function.
func (b *Bus) SubscribeSyncFunc(filter Filter, fn HandlerFunc) {
	b.SubscribeSync(fn, filter)
}

// SubscribeAsyncFunc adds an asynchronous handler function.
func (b *Bus) SubscribeAsyncFunc(filter Filter, fn HandlerFunc) {
	b.SubscribeAsync(fn, filter)
}

// Emit emits an event to all matching subscribers.
// This method is fire-and-forget: errors in handlers do not propagate.
func (b *Bus) Emit(event EventProvider) {
	if !b.config.Enabled {
		return
	}

	if b.config.IsEventDisabled(event.BaseEvent().Type) {
		return
	}

	// Emit to synchronous subscribers
	b.emitSync(event)

	// Emit to asynchronous subscribers
	b.emitAsync(event)
}

// emitSync sends the event to all synchronous subscribers.
func (b *Bus) emitSync(event EventProvider) {
	b.mu.RLock()
	subscribers := make([]syncSubscription, len(b.syncSubscribers))
	copy(subscribers, b.syncSubscribers)
	b.mu.RUnlock()

	for _, sub := range subscribers {
		if !sub.filter.Matches(event) {
			continue
		}

		// Create timeout context for sync handler
		ctx, cancel := context.WithTimeout(context.Background(), b.config.SyncTimeout)

		// Fire-and-forget: ignore errors
		_ = sub.subscriber.HandleEvent(ctx, event.BaseEvent())

		cancel()
	}
}

// emitAsync sends the event to the async queue.
func (b *Bus) emitAsync(event EventProvider) {
	b.mu.RLock()
	hasAsync := len(b.asyncSubscribers) > 0
	b.mu.RUnlock()

	if !hasAsync {
		return
	}

	// Non-blocking queue insert
	select {
	case b.queue <- event.BaseEvent():
	default:
		// Queue is full, drop the event
		// This is fire-and-forget semantics
	}
}

// asyncWorker processes events from the async queue.
func (b *Bus) asyncWorker() {
	defer b.wg.Done()

	for event := range b.queue {
		b.processAsyncEvent(event)
	}
}

// processAsyncEvent processes an event for all async subscribers.
func (b *Bus) processAsyncEvent(event Event) {
	b.mu.RLock()
	subscribers := make([]asyncSubscription, len(b.asyncSubscribers))
	copy(subscribers, b.asyncSubscribers)
	b.mu.RUnlock()

	for _, sub := range subscribers {
		if !sub.filter.Matches(event) {
			continue
		}

		// Fire-and-forget: errors are ignored
		_ = sub.subscriber.HandleEvent(context.Background(), event)
	}
}

// Close shuts down the event bus gracefully.
// It stops accepting new events and waits for async workers to finish.
func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	close(b.queue)
	b.wg.Wait()

	return nil
}

// IsClosed returns true if the bus has been closed.
func (b *Bus) IsClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.closed
}

// SubscriberCount returns the total number of subscribers.
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.syncSubscribers) + len(b.asyncSubscribers)
}
