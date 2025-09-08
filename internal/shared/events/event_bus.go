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
	EventDiscoveryStarted   EventType = "discovery.started"
	EventDiscoveryProgress  EventType = "discovery.progress"
	EventDiscoveryCompleted EventType = "discovery.completed"
	EventDiscoveryFailed    EventType = "discovery.failed"
	EventResourceDiscovered EventType = "discovery.resource"

	// Drift detection events
	EventDriftDetectionStarted   EventType = "drift.started"
	EventDriftDetectionProgress  EventType = "drift.progress"
	EventDriftDetectionCompleted EventType = "drift.completed"
	EventDriftDetected           EventType = "drift.detected"
	EventDriftResolved           EventType = "drift.resolved"

	// Remediation events
	EventRemediationStarted   EventType = "remediation.started"
	EventRemediationProgress  EventType = "remediation.progress"
	EventRemediationCompleted EventType = "remediation.completed"
	EventRemediationFailed    EventType = "remediation.failed"
	EventResourceDeleted      EventType = "remediation.deleted"
	EventResourceImported     EventType = "remediation.imported"

	// State management events
	EventStateBackupCreated EventType = "state.backup.created"
	EventStatePulled        EventType = "state.pulled"
	EventStatePushed        EventType = "state.pushed"
	EventStateValidated     EventType = "state.validated"
	EventStateModified      EventType = "state.modified"

	// System events
	EventCacheCleared   EventType = "cache.cleared"
	EventCacheRefreshed EventType = "cache.refreshed"
	EventHealthCheck    EventType = "health.check"
	EventConfigChanged  EventType = "config.changed"
	EventAuditLog       EventType = "audit.log"

	// Job events
	EventJobQueued    EventType = "job.queued"
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
	EventJobFailed    EventType = "job.failed"
	EventJobRetrying  EventType = "job.retrying"

	// WebSocket events
	EventWSClientConnected    EventType = "ws.connected"
	EventWSClientDisconnected EventType = "ws.disconnected"
	EventWSMessage            EventType = "ws.message"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Error     error                  `json:"error,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// Subscription represents a subscription to events
type Subscription struct {
	ID      string
	Filter  EventFilter
	Channel chan Event
	cancel  context.CancelFunc
	ctx     context.Context
}

// EventFilter defines criteria for event filtering
type EventFilter struct {
	Types    []EventType
	Sources  []string
	MinTime  *time.Time
	MaxTime  *time.Time
	Metadata map[string]string
}

// EventHandler is a function that handles events
type EventHandler func(event Event)

// EventBusInterface defines the contract for event bus implementations
type EventBusInterface interface {
	Publish(event Event) error
	Subscribe(ctx context.Context, filter EventFilter, bufferSize int) *Subscription
	Unsubscribe(sub *Subscription)
	RegisterHandler(eventType EventType, handler EventHandler)
	GetMetrics() *EventMetrics
	Close()
}

// EventBus manages event publishing and subscriptions
type EventBus struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription
	handlers      map[EventType][]EventHandler
	buffer        []Event
	bufferSize    int
	metrics       *EventMetrics
	closed        bool
}

// EventMetrics tracks event bus metrics
type EventMetrics struct {
	mu                sync.RWMutex
	EventsPublished   map[EventType]int64
	EventsDelivered   map[EventType]int64
	SubscriptionCount int
	ActiveHandlers    int
}

// NewEventBus creates a new event bus
func NewEventBus(bufferSize int) *EventBus {
	return &EventBus{
		subscriptions: make(map[string]*Subscription),
		handlers:      make(map[EventType][]EventHandler),
		buffer:        make([]Event, 0, bufferSize),
		bufferSize:    bufferSize,
		metrics: &EventMetrics{
			EventsPublished: make(map[EventType]int64),
			EventsDelivered: make(map[EventType]int64),
		},
	}
}

// Publish publishes an event to all matching subscribers
func (eb *EventBus) Publish(event Event) error {
	eb.mu.RLock()
	if eb.closed {
		eb.mu.RUnlock()
		return nil
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Update metrics
	eb.metrics.mu.Lock()
	eb.metrics.EventsPublished[event.Type]++
	eb.metrics.mu.Unlock()

	// Add to buffer
	if len(eb.buffer) >= eb.bufferSize {
		eb.buffer = eb.buffer[1:]
	}
	eb.buffer = append(eb.buffer, event)

	// Get subscriptions and handlers
	subs := make([]*Subscription, 0, len(eb.subscriptions))
	for _, sub := range eb.subscriptions {
		if eb.matchesFilter(event, sub.Filter) {
			subs = append(subs, sub)
		}
	}

	handlers := make([]EventHandler, len(eb.handlers[event.Type]))
	copy(handlers, eb.handlers[event.Type])
	eb.mu.RUnlock()

	// Deliver to subscriptions
	for _, sub := range subs {
		select {
		case sub.Channel <- event:
			eb.metrics.mu.Lock()
			eb.metrics.EventsDelivered[event.Type]++
			eb.metrics.mu.Unlock()
		case <-sub.ctx.Done():
			// Subscription cancelled
		default:
			// Channel full, skip
		}
	}

	// Execute handlers
	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// Handler panicked, ignore
				}
			}()
			h(event)
		}(handler)
	}

	return nil
}

// Subscribe creates a new subscription with a filter
func (eb *EventBus) Subscribe(ctx context.Context, filter EventFilter, bufferSize int) *Subscription {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return nil
	}

	subCtx, cancel := context.WithCancel(ctx)
	sub := &Subscription{
		ID:      generateSubscriptionID(),
		Filter:  filter,
		Channel: make(chan Event, bufferSize),
		cancel:  cancel,
		ctx:     subCtx,
	}

	eb.subscriptions[sub.ID] = sub
	eb.metrics.mu.Lock()
	eb.metrics.SubscriptionCount++
	eb.metrics.mu.Unlock()

	// Send buffered events that match the filter
	for _, event := range eb.buffer {
		if eb.matchesFilter(event, filter) {
			select {
			case sub.Channel <- event:
			default:
				// Channel full
			}
		}
	}

	return sub
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(sub *Subscription) {
	if sub == nil {
		return
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	if _, exists := eb.subscriptions[sub.ID]; exists {
		delete(eb.subscriptions, sub.ID)
		sub.cancel()
		close(sub.Channel)

		eb.metrics.mu.Lock()
		eb.metrics.SubscriptionCount--
		eb.metrics.mu.Unlock()
	}
}

// RegisterHandler registers a handler for specific event types
func (eb *EventBus) RegisterHandler(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)

	eb.metrics.mu.Lock()
	eb.metrics.ActiveHandlers++
	eb.metrics.mu.Unlock()
}

// GetMetrics returns current event bus metrics
func (eb *EventBus) GetMetrics() *EventMetrics {
	eb.metrics.mu.RLock()
	defer eb.metrics.mu.RUnlock()

	// Create a copy of metrics
	metrics := EventMetrics{
		EventsPublished:   make(map[EventType]int64),
		EventsDelivered:   make(map[EventType]int64),
		SubscriptionCount: eb.metrics.SubscriptionCount,
		ActiveHandlers:    eb.metrics.ActiveHandlers,
	}

	for k, v := range eb.metrics.EventsPublished {
		metrics.EventsPublished[k] = v
	}
	for k, v := range eb.metrics.EventsDelivered {
		metrics.EventsDelivered[k] = v
	}

	return &metrics
}

// GetBuffer returns the current event buffer
func (eb *EventBus) GetBuffer() []Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	buffer := make([]Event, len(eb.buffer))
	copy(buffer, eb.buffer)
	return buffer
}

// Close closes the event bus and all subscriptions
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return
	}

	eb.closed = true

	// Cancel all subscriptions
	for _, sub := range eb.subscriptions {
		sub.cancel()
		close(sub.Channel)
	}

	eb.subscriptions = make(map[string]*Subscription)
	eb.handlers = make(map[EventType][]EventHandler)
	eb.buffer = nil
}

// matchesFilter checks if an event matches a filter
func (eb *EventBus) matchesFilter(event Event, filter EventFilter) bool {
	// Check event types
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if event.Type == t {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check sources
	if len(filter.Sources) > 0 {
		matched := false
		for _, s := range filter.Sources {
			if event.Source == s {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check time range
	if filter.MinTime != nil && event.Timestamp.Before(*filter.MinTime) {
		return false
	}
	if filter.MaxTime != nil && event.Timestamp.After(*filter.MaxTime) {
		return false
	}

	// Check metadata
	if len(filter.Metadata) > 0 {
		for key, value := range filter.Metadata {
			if event.Metadata[key] != value {
				return false
			}
		}
	}

	return true
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102-150405.000000")
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	return "sub-" + time.Now().Format("20060102-150405.000000")
}
