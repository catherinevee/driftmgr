package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// EventProcessor processes cloud events
type EventProcessor struct {
	eventChan chan CloudEvent
	handlers  map[string]EventHandler
	buffer    *EventBuffer
	metrics   *EventMetrics
	mu        sync.RWMutex
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// EventHandler processes specific event types
type EventHandler func(event CloudEvent) error

// EventBuffer stores events for batch processing
type EventBuffer struct {
	events  []CloudEvent
	maxSize int
	mu      sync.Mutex
}

// EventMetrics tracks event processing metrics
type EventMetrics struct {
	TotalEvents     int64
	ProcessedEvents int64
	FailedEvents    int64
	EventsByType    map[string]int64
	LastEventTime   time.Time
	mu              sync.RWMutex
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(bufferSize int) *EventProcessor {
	return &EventProcessor{
		eventChan: make(chan CloudEvent, bufferSize),
		handlers:  make(map[string]EventHandler),
		buffer:    &EventBuffer{maxSize: bufferSize},
		metrics:   &EventMetrics{EventsByType: make(map[string]int64)},
		stopChan:  make(chan struct{}),
	}
}

// Start starts the event processor
func (p *EventProcessor) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.processEvents(ctx)
}

// Stop stops the event processor
func (p *EventProcessor) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}

// ProcessEvent processes a single event
func (p *EventProcessor) ProcessEvent(event CloudEvent) {
	select {
	case p.eventChan <- event:
		p.updateMetrics(event, false)
	default:
		// Buffer is full, log and drop
		fmt.Printf("Event buffer full, dropping event: %s\n", event.ID)
		p.updateMetrics(event, true)
	}
}

// RegisterHandler registers an event handler
func (p *EventProcessor) RegisterHandler(eventType string, handler EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[eventType] = handler
}

// processEvents processes events from the channel
func (p *EventProcessor) processEvents(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.flushBuffer()
			return
		case <-p.stopChan:
			p.flushBuffer()
			return
		case event := <-p.eventChan:
			p.handleEvent(event)
			p.buffer.Add(event)
		case <-ticker.C:
			p.processBatch()
		}
	}
}

// handleEvent handles a single event
func (p *EventProcessor) handleEvent(event CloudEvent) {
	p.mu.RLock()
	handler, exists := p.handlers[event.Type]
	p.mu.RUnlock()

	if !exists {
		// Try generic handler
		p.mu.RLock()
		handler, exists = p.handlers["*"]
		p.mu.RUnlock()
	}

	if exists {
		if err := handler(event); err != nil {
			fmt.Printf("Error handling event %s: %v\n", event.ID, err)
			p.metrics.mu.Lock()
			p.metrics.FailedEvents++
			p.metrics.mu.Unlock()
		} else {
			p.metrics.mu.Lock()
			p.metrics.ProcessedEvents++
			p.metrics.mu.Unlock()
		}
	}
}

// processBatch processes buffered events in batch
func (p *EventProcessor) processBatch() {
	events := p.buffer.Flush()
	if len(events) == 0 {
		return
	}

	// Group events by type for batch processing
	eventsByType := make(map[string][]CloudEvent)
	for _, event := range events {
		eventsByType[event.Type] = append(eventsByType[event.Type], event)
	}

	// Process each group
	for eventType, group := range eventsByType {
		fmt.Printf("Processing batch of %d %s events\n", len(group), eventType)
		// Here you would implement batch processing logic
		// For example, aggregate metrics, generate reports, etc.
	}
}

// flushBuffer processes all remaining buffered events
func (p *EventProcessor) flushBuffer() {
	p.processBatch()
}

// updateMetrics updates event metrics
func (p *EventProcessor) updateMetrics(event CloudEvent, dropped bool) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.TotalEvents++
	p.metrics.EventsByType[event.Type]++
	p.metrics.LastEventTime = event.Time

	if dropped {
		p.metrics.FailedEvents++
	}
}

// GetMetrics returns current metrics
func (p *EventProcessor) GetMetrics() EventMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	// Return a copy
	return EventMetrics{
		TotalEvents:     p.metrics.TotalEvents,
		ProcessedEvents: p.metrics.ProcessedEvents,
		FailedEvents:    p.metrics.FailedEvents,
		EventsByType:    copyMap(p.metrics.EventsByType),
		LastEventTime:   p.metrics.LastEventTime,
	}
}

// EventBuffer methods

// Add adds an event to the buffer
func (b *EventBuffer) Add(event CloudEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) >= b.maxSize {
		// Remove oldest event
		b.events = b.events[1:]
	}
	b.events = append(b.events, event)
}

// Flush returns all events and clears the buffer
func (b *EventBuffer) Flush() []CloudEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	events := b.events
	b.events = nil
	return events
}

// ChangeDetector detects changes in resources
type ChangeDetector struct {
	previousStates map[string]map[string]interface{}
	mu             sync.RWMutex
}

// NewChangeDetector creates a new change detector
func NewChangeDetector() *ChangeDetector {
	return &ChangeDetector{
		previousStates: make(map[string]map[string]interface{}),
	}
}

// Change represents a detected change
type Change struct {
	ResourceID string
	Type       ChangeType
	Details    map[string]interface{}
	Timestamp  time.Time
}

// DetectChanges detects changes in resources
func (d *ChangeDetector) DetectChanges(provider string, currentResources []interface{}) []Change {
	d.mu.Lock()
	defer d.mu.Unlock()

	var changes []Change
	currentState := make(map[string]interface{})

	// Build current state map
	for _, resource := range currentResources {
		// Extract resource ID (this would be provider-specific)
		if resMap, ok := resource.(map[string]interface{}); ok {
			if id, ok := resMap["id"].(string); ok {
				currentState[id] = resource
			}
		}
	}

	// Get previous state
	previousState, exists := d.previousStates[provider]
	if !exists {
		previousState = make(map[string]interface{})
	}

	// Detect deletions
	for id := range previousState {
		if _, exists := currentState[id]; !exists {
			changes = append(changes, Change{
				ResourceID: id,
				Type:       ChangeTypeDelete,
				Timestamp:  time.Now(),
			})
		}
	}

	// Detect creations and updates
	for id, resource := range currentState {
		previous, existed := previousState[id]
		if !existed {
			changes = append(changes, Change{
				ResourceID: id,
				Type:       ChangeTypeCreate,
				Details:    map[string]interface{}{"resource": resource},
				Timestamp:  time.Now(),
			})
		} else if !resourcesEqual(previous, resource) {
			changes = append(changes, Change{
				ResourceID: id,
				Type:       ChangeTypeUpdate,
				Details: map[string]interface{}{
					"before": previous,
					"after":  resource,
				},
				Timestamp: time.Now(),
			})
		}
	}

	// Update previous state
	d.previousStates[provider] = currentState

	return changes
}

// resourcesEqual compares two resources for equality
func resourcesEqual(a, b interface{}) bool {
	// Simple JSON comparison
	// In production, this would be more sophisticated
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// copyMap creates a copy of a string->int64 map
func copyMap(m map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range m {
		copy[k] = v
	}
	return copy
}
