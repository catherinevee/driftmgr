package events

import (
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
	EventResourceFound      EventType = "resource.found"
	DiscoveryStarted        EventType = "discovery.started"   // Alias for compatibility
	DiscoveryProgress       EventType = "discovery.progress"  // Alias for compatibility
	DiscoveryCompleted      EventType = "discovery.completed" // Alias for compatibility
	DiscoveryFailed         EventType = "discovery.failed"    // Alias for compatibility

	// Drift events
	EventDriftDetected      EventType = "drift.detected"
	EventDriftAnalyzed      EventType = "drift.analyzed"
	EventDriftRemediated    EventType = "drift.remediated"
	DriftDetectionStarted   EventType = "drift.detection.started"
	DriftDetectionCompleted EventType = "drift.detection.completed"
	DriftDetectionFailed    EventType = "drift.detection.failed"

	// Remediation events
	EventRemediationStarted   EventType = "remediation.started"
	EventRemediationProgress  EventType = "remediation.progress"
	EventRemediationCompleted EventType = "remediation.completed"
	EventRemediationFailed    EventType = "remediation.failed"
	RemediationStarted        EventType = "remediation.started"   // Alias
	RemediationCompleted      EventType = "remediation.completed" // Alias
	RemediationFailed         EventType = "remediation.failed"    // Alias

	// System events
	EventSystemStartup  EventType = "system.startup"
	EventSystemShutdown EventType = "system.shutdown"
	EventSystemError    EventType = "system.error"
	EventSystemWarning  EventType = "system.warning"
	EventSystemInfo     EventType = "system.info"

	// State events
	EventStateChanged  EventType = "state.changed"
	EventStateBackup   EventType = "state.backup"
	EventStateRestored EventType = "state.restored"
	EventStateLocked   EventType = "state.locked"
	EventStateUnlocked EventType = "state.unlocked"

	// Job events
	EventJobQueued    EventType = "job.queued"
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
	EventJobFailed    EventType = "job.failed"
	EventJobCancelled EventType = "job.cancelled"
	JobCreated        EventType = "job.created"   // Alias for compatibility
	JobStarted        EventType = "job.started"   // Alias for compatibility
	JobCompleted      EventType = "job.completed" // Alias for compatibility
	JobFailed         EventType = "job.failed"    // Alias for compatibility

	// Resource events
	EventResourceCreated EventType = "resource.created"
	EventResourceUpdated EventType = "resource.updated"
	EventResourceDeleted EventType = "resource.deleted"
	ResourceCreated      EventType = "resource.created" // Alias for compatibility
	ResourceUpdated      EventType = "resource.updated" // Alias for compatibility
	ResourceDeleted      EventType = "resource.deleted" // Alias for compatibility

	// Additional state events
	EventStateImported EventType = "state.imported"
	EventStateAnalyzed EventType = "state.analyzed"
	EventStateDeleted  EventType = "state.deleted"
	StateImported      EventType = "state.imported" // Alias for compatibility
	StateAnalyzed      EventType = "state.analyzed" // Alias for compatibility
	StateDeleted       EventType = "state.deleted"  // Alias for compatibility
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
}

// EventHandler is a function that handles events
type EventHandler func(event Event)

// Subscription represents an event subscription
type Subscription struct {
	ID      string
	Handler EventHandler
	Types   []EventType
}

// EventBus manages event publishing and subscriptions
type EventBus struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription
	handlers      map[EventType][]EventHandler
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscriptions: make(map[string]*Subscription),
		handlers:      make(map[EventType][]EventHandler),
	}
}

// Subscribe registers a handler for specific event types
func (eb *EventBus) Subscribe(types []EventType, handler EventHandler) *Subscription {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	sub := &Subscription{
		ID:      generateID(),
		Handler: handler,
		Types:   types,
	}

	eb.subscriptions[sub.ID] = sub

	for _, t := range types {
		eb.handlers[t] = append(eb.handlers[t], handler)
	}

	return sub
}

// SubscribeToTypes is an alias for Subscribe for compatibility
func (eb *EventBus) SubscribeToTypes(types []EventType, handler EventHandler) *Subscription {
	return eb.Subscribe(types, handler)
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(sub *Subscription) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.subscriptions, sub.ID)

	// Remove handlers
	for _, t := range sub.Types {
		handlers := eb.handlers[t]
		for i, h := range handlers {
			// Compare function pointers
			if &h == &sub.Handler {
				eb.handlers[t] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// Publish sends an event to all registered handlers
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		// Call handler in goroutine to prevent blocking
		go handler(event)
	}
}

// generateID generates a unique ID
func generateID() string {
	return time.Now().Format("20060102150405.999999999")
}
