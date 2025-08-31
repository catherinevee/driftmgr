package events

import (
	"context"
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	// Discovery events
	DiscoveryStarted   EventType = "discovery.started"
	DiscoveryProgress  EventType = "discovery.progress"
	DiscoveryCompleted EventType = "discovery.completed"
	DiscoveryFailed    EventType = "discovery.failed"

	// State events
	StateImported EventType = "state.imported"
	StateAnalyzed EventType = "state.analyzed"
	StateDeleted  EventType = "state.deleted"

	// Drift events
	DriftDetectionStarted   EventType = "drift.detection.started"
	DriftDetectionCompleted EventType = "drift.detection.completed"
	DriftDetectionFailed    EventType = "drift.detection.failed"

	// Remediation events
	RemediationStarted   EventType = "remediation.started"
	RemediationCompleted EventType = "remediation.completed"
	RemediationFailed    EventType = "remediation.failed"

	// Resource events
	ResourceCreated EventType = "resource.created"
	ResourceUpdated EventType = "resource.updated"
	ResourceDeleted EventType = "resource.deleted"

	// Cache events
	CacheCleared    EventType = "cache.cleared"
	CacheInvalidated EventType = "cache.invalidated"

	// Job events
	JobCreated   EventType = "job.created"
	JobStarted   EventType = "job.started"
	JobCompleted EventType = "job.completed"
	JobFailed    EventType = "job.failed"
)

// Event represents an event in the system
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// Handler is a function that handles events
type Handler func(event Event)

// Subscription represents a subscription to events
type Subscription struct {
	ID       string
	Filter   func(Event) bool
	Handler  Handler
	Channel  chan Event
	cancel   context.CancelFunc
}

// EventBus manages event publishing and subscriptions
type EventBus struct {
	subscribers map[string]*Subscription
	mu          sync.RWMutex
	buffer      []Event
	bufferSize  int
	metrics     *EventMetrics
}

// EventMetrics tracks event bus metrics
type EventMetrics struct {
	EventsPublished   int64
	EventsDelivered   int64
	SubscriberCount   int
	DroppedEvents     int64
	ProcessingTimeMs  int64
}

// NewEventBus creates a new event bus
func NewEventBus(bufferSize int) *EventBus {
	return &EventBus{
		subscribers: make(map[string]*Subscription),
		buffer:      make([]Event, 0, bufferSize),
		bufferSize:  bufferSize,
		metrics:     &EventMetrics{},
	}
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event Event) {
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Add to buffer for replay
	eb.addToBuffer(event)

	// Send to all matching subscribers
	for _, sub := range eb.subscribers {
		if sub.Filter == nil || sub.Filter(event) {
			select {
			case sub.Channel <- event:
				eb.metrics.EventsDelivered++
			default:
				// Channel full, drop event
				eb.metrics.DroppedEvents++
			}
		}
	}

	eb.metrics.EventsPublished++
}

// Subscribe creates a new subscription for events
func (eb *EventBus) Subscribe(filter func(Event) bool, handler Handler) *Subscription {
	ctx, cancel := context.WithCancel(context.Background())
	
	sub := &Subscription{
		ID:      generateSubscriptionID(),
		Filter:  filter,
		Handler: handler,
		Channel: make(chan Event, 100),
		cancel:  cancel,
	}

	eb.mu.Lock()
	eb.subscribers[sub.ID] = sub
	eb.metrics.SubscriberCount = len(eb.subscribers)
	eb.mu.Unlock()

	// Start handler goroutine
	go eb.handleSubscription(ctx, sub)

	return sub
}

// SubscribeToType subscribes to a specific event type
func (eb *EventBus) SubscribeToType(eventType EventType, handler Handler) *Subscription {
	filter := func(e Event) bool {
		return e.Type == eventType
	}
	return eb.Subscribe(filter, handler)
}

// SubscribeToTypes subscribes to multiple event types
func (eb *EventBus) SubscribeToTypes(eventTypes []EventType, handler Handler) *Subscription {
	typeMap := make(map[EventType]bool)
	for _, t := range eventTypes {
		typeMap[t] = true
	}
	
	filter := func(e Event) bool {
		return typeMap[e.Type]
	}
	return eb.Subscribe(filter, handler)
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(sub *Subscription) {
	if sub == nil {
		return
	}

	eb.mu.Lock()
	delete(eb.subscribers, sub.ID)
	eb.metrics.SubscriberCount = len(eb.subscribers)
	eb.mu.Unlock()

	sub.cancel()
	close(sub.Channel)
}

// handleSubscription processes events for a subscription
func (eb *EventBus) handleSubscription(ctx context.Context, sub *Subscription) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-sub.Channel:
			start := time.Now()
			sub.Handler(event)
			eb.metrics.ProcessingTimeMs += time.Since(start).Milliseconds()
		}
	}
}

// addToBuffer adds an event to the replay buffer
func (eb *EventBus) addToBuffer(event Event) {
	if len(eb.buffer) >= eb.bufferSize {
		// Remove oldest event
		eb.buffer = eb.buffer[1:]
	}
	eb.buffer = append(eb.buffer, event)
}

// GetRecentEvents returns recent events from the buffer
func (eb *EventBus) GetRecentEvents(count int) []Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if count > len(eb.buffer) {
		count = len(eb.buffer)
	}

	start := len(eb.buffer) - count
	if start < 0 {
		start = 0
	}

	result := make([]Event, count)
	copy(result, eb.buffer[start:])
	return result
}

// GetMetrics returns event bus metrics
func (eb *EventBus) GetMetrics() EventMetrics {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return *eb.metrics
}

// Clear clears all subscriptions and buffer
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for id, sub := range eb.subscribers {
		sub.cancel()
		close(sub.Channel)
		delete(eb.subscribers, id)
	}

	eb.buffer = make([]Event, 0, eb.bufferSize)
	eb.metrics = &EventMetrics{}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(8)
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	return "sub-" + generateRandomString(12)
}

// generateRandomString generates a random string of given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}