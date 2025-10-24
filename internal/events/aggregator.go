package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/shared/events"
)

// EventAggregator aggregates and processes events in real-time
type EventAggregator struct {
	eventBus     *events.EventBus
	aggregations map[string]*EventAggregation
	mu           sync.RWMutex
	config       *AggregatorConfig
	active       bool
	stopChan     chan struct{}
	wg           sync.WaitGroup
	subscription *events.Subscription
}

// EventAggregation represents an aggregated view of events
type EventAggregation struct {
	ID        string
	Type      string
	Count     int64
	FirstSeen time.Time
	LastSeen  time.Time
	Severity  string
	Sources   map[string]int64
	Data      map[string]interface{}
	Metadata  map[string]string
	SubEvents []events.Event
	mu        sync.RWMutex
}

// AggregatorConfig contains configuration for the event aggregator
type AggregatorConfig struct {
	Enabled           bool          `json:"enabled"`
	AggregationWindow time.Duration `json:"aggregation_window"`
	MaxEvents         int           `json:"max_events"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	BatchSize         int           `json:"batch_size"`
	FlushInterval     time.Duration `json:"flush_interval"`
}

// AggregationRule defines how events should be aggregated
type AggregationRule struct {
	ID          string
	Name        string
	EventTypes  []events.EventType
	GroupBy     []string
	AggregateBy string
	Window      time.Duration
	Threshold   int64
	Enabled     bool
}

// NewEventAggregator creates a new event aggregator
func NewEventAggregator(eventBus *events.EventBus, config *AggregatorConfig) *EventAggregator {
	if config == nil {
		config = &AggregatorConfig{
			Enabled:           true,
			AggregationWindow: 5 * time.Minute,
			MaxEvents:         10000,
			CleanupInterval:   1 * time.Minute,
			BatchSize:         100,
			FlushInterval:     30 * time.Second,
		}
	}

	return &EventAggregator{
		eventBus:     eventBus,
		aggregations: make(map[string]*EventAggregation),
		config:       config,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the event aggregator
func (ea *EventAggregator) Start(ctx context.Context) error {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	if ea.active {
		return fmt.Errorf("event aggregator already active")
	}

	ea.active = true

	// Subscribe to all events
	filter := events.EventFilter{
		Types: []events.EventType{
			events.EventDiscoveryStarted,
			events.EventDiscoveryProgress,
			events.EventDiscoveryCompleted,
			events.EventDiscoveryFailed,
			events.EventDriftDetected,
			events.EventRemediationStarted,
			events.EventRemediationProgress,
			events.EventRemediationCompleted,
			events.EventRemediationFailed,
			events.EventHealthCheck,
			events.EventConfigChanged,
			events.EventAuditLog,
			events.EventJobQueued,
			events.EventJobStarted,
			events.EventJobCompleted,
			events.EventJobFailed,
			events.EventResourceDeleted,
			events.EventResourceImported,
		},
	}

	ea.subscription = ea.eventBus.Subscribe(ctx, filter, ea.config.BatchSize)

	// Start background tasks
	ea.wg.Add(3)
	go ea.eventProcessor(ctx)
	go ea.aggregationProcessor(ctx)
	go ea.cleanupProcessor(ctx)

	log.Println("Event aggregator started")
	return nil
}

// Stop stops the event aggregator
func (ea *EventAggregator) Stop() {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	if !ea.active {
		return
	}

	ea.active = false
	close(ea.stopChan)
	ea.wg.Wait()

	if ea.subscription != nil {
		ea.eventBus.Unsubscribe(ea.subscription)
	}

	log.Println("Event aggregator stopped")
}

// eventProcessor processes events from the event bus
func (ea *EventAggregator) eventProcessor(ctx context.Context) {
	defer ea.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ea.stopChan:
			return
		case event := <-ea.subscription.Channel:
			ea.processEvent(event)
		}
	}
}

// processEvent processes a single event and adds it to aggregations
func (ea *EventAggregator) processEvent(event events.Event) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	// Generate aggregation key based on event type and source
	key := ea.generateAggregationKey(event)

	// Get or create aggregation
	aggregation, exists := ea.aggregations[key]
	if !exists {
		aggregation = &EventAggregation{
			ID:        key,
			Type:      string(event.Type),
			Count:     0,
			FirstSeen: event.Timestamp,
			LastSeen:  event.Timestamp,
			Severity:  ea.determineSeverity(event),
			Sources:   make(map[string]int64),
			Data:      make(map[string]interface{}),
			Metadata:  make(map[string]string),
			SubEvents: make([]events.Event, 0),
		}
		ea.aggregations[key] = aggregation
	}

	// Update aggregation
	aggregation.mu.Lock()
	aggregation.Count++
	aggregation.LastSeen = event.Timestamp
	aggregation.Sources[event.Source]++

	// Add sub-event (keep only recent ones)
	aggregation.SubEvents = append(aggregation.SubEvents, event)
	if len(aggregation.SubEvents) > 100 {
		aggregation.SubEvents = aggregation.SubEvents[1:]
	}

	// Update data based on event type
	ea.updateAggregationData(aggregation, event)
	aggregation.mu.Unlock()

	// Check if we need to flush this aggregation
	if ea.shouldFlushAggregation(aggregation) {
		ea.flushAggregation(aggregation)
	}
}

// generateAggregationKey generates a key for event aggregation
func (ea *EventAggregator) generateAggregationKey(event events.Event) string {
	// Group by event type and source
	return fmt.Sprintf("%s:%s", event.Type, event.Source)
}

// determineSeverity determines the severity of an event
func (ea *EventAggregator) determineSeverity(event events.Event) string {
	switch event.Type {
	case events.EventDiscoveryFailed, events.EventRemediationFailed, events.EventJobFailed:
		return "error"
	case events.EventDriftDetected:
		return "warning"
	case events.EventDiscoveryStarted, events.EventRemediationStarted, events.EventHealthCheck, events.EventConfigChanged, events.EventAuditLog:
		return "info"
	case events.EventDiscoveryCompleted, events.EventRemediationCompleted, events.EventJobCompleted:
		return "success"
	default:
		return "info"
	}
}

// updateAggregationData updates aggregation data based on event type
func (ea *EventAggregator) updateAggregationData(aggregation *EventAggregation, event events.Event) {
	switch event.Type {
	case events.EventDiscoveryProgress:
		if progress, ok := event.Data["progress"].(int); ok {
			aggregation.Data["progress"] = progress
		}
		if provider, ok := event.Data["provider"].(string); ok {
			aggregation.Data["provider"] = provider
		}
	case events.EventDriftDetected:
		if driftCount, ok := event.Data["drift_count"].(int); ok {
			if current, exists := aggregation.Data["total_drifts"]; exists {
				aggregation.Data["total_drifts"] = current.(int) + driftCount
			} else {
				aggregation.Data["total_drifts"] = driftCount
			}
		}
	case events.EventRemediationProgress:
		if progress, ok := event.Data["progress"].(int); ok {
			aggregation.Data["progress"] = progress
		}
		if resourceID, ok := event.Data["resource_id"].(string); ok {
			aggregation.Data["resource_id"] = resourceID
		}
	case events.EventJobCompleted:
		if jobType, ok := event.Data["job_type"].(string); ok {
			aggregation.Data["job_type"] = jobType
		}
		if duration, ok := event.Data["duration"].(time.Duration); ok {
			aggregation.Data["duration"] = duration
		}
	}
}

// shouldFlushAggregation determines if an aggregation should be flushed
func (ea *EventAggregator) shouldFlushAggregation(aggregation *EventAggregation) bool {
	// Flush if we've reached the threshold
	if aggregation.Count >= 100 {
		return true
	}

	// Flush if the aggregation window has passed
	if time.Since(aggregation.FirstSeen) >= ea.config.AggregationWindow {
		return true
	}

	// Flush if it's a high-severity event
	if aggregation.Severity == "error" && aggregation.Count >= 5 {
		return true
	}

	return false
}

// flushAggregation flushes an aggregation (publishes aggregated event)
func (ea *EventAggregator) flushAggregation(aggregation *EventAggregation) {
	aggregation.mu.RLock()
	defer aggregation.mu.RUnlock()

	// Create aggregated event
	aggregatedEvent := events.Event{
		ID:        fmt.Sprintf("agg_%s_%d", aggregation.ID, time.Now().UnixNano()),
		Type:      events.EventType(fmt.Sprintf("aggregated.%s", aggregation.Type)),
		Timestamp: time.Now(),
		Source:    "event_aggregator",
		Data: map[string]interface{}{
			"aggregation_id":   aggregation.ID,
			"event_type":       aggregation.Type,
			"count":            aggregation.Count,
			"first_seen":       aggregation.FirstSeen,
			"last_seen":        aggregation.LastSeen,
			"severity":         aggregation.Severity,
			"sources":          aggregation.Sources,
			"aggregated_data":  aggregation.Data,
			"sub_events_count": len(aggregation.SubEvents),
		},
		Metadata: aggregation.Metadata,
	}

	// Publish aggregated event
	if err := ea.eventBus.Publish(aggregatedEvent); err != nil {
		log.Printf("Failed to publish aggregated event: %v", err)
	}

	log.Printf("Flushed aggregation %s with %d events", aggregation.ID, aggregation.Count)
}

// aggregationProcessor processes aggregations periodically
func (ea *EventAggregator) aggregationProcessor(ctx context.Context) {
	defer ea.wg.Done()

	ticker := time.NewTicker(ea.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ea.stopChan:
			return
		case <-ticker.C:
			ea.flushExpiredAggregations()
		}
	}
}

// flushExpiredAggregations flushes all expired aggregations
func (ea *EventAggregator) flushExpiredAggregations() {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, aggregation := range ea.aggregations {
		if now.Sub(aggregation.FirstSeen) >= ea.config.AggregationWindow {
			ea.flushAggregation(aggregation)
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired aggregations
	for _, key := range expiredKeys {
		delete(ea.aggregations, key)
	}

	if len(expiredKeys) > 0 {
		log.Printf("Flushed %d expired aggregations", len(expiredKeys))
	}
}

// cleanupProcessor cleans up old aggregations
func (ea *EventAggregator) cleanupProcessor(ctx context.Context) {
	defer ea.wg.Done()

	ticker := time.NewTicker(ea.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ea.stopChan:
			return
		case <-ticker.C:
			ea.cleanupOldAggregations()
		}
	}
}

// cleanupOldAggregations removes old aggregations
func (ea *EventAggregator) cleanupOldAggregations() {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	now := time.Now()
	cleanupThreshold := 2 * ea.config.AggregationWindow
	removedCount := 0

	for key, aggregation := range ea.aggregations {
		if now.Sub(aggregation.LastSeen) > cleanupThreshold {
			delete(ea.aggregations, key)
			removedCount++
		}
	}

	if removedCount > 0 {
		log.Printf("Cleaned up %d old aggregations", removedCount)
	}
}

// GetAggregations returns all current aggregations
func (ea *EventAggregator) GetAggregations() map[string]*EventAggregation {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	// Return a copy of aggregations
	result := make(map[string]*EventAggregation)
	for key, aggregation := range ea.aggregations {
		aggregation.mu.RLock()
		aggCopy := &EventAggregation{
			ID:        aggregation.ID,
			Type:      aggregation.Type,
			Count:     aggregation.Count,
			FirstSeen: aggregation.FirstSeen,
			LastSeen:  aggregation.LastSeen,
			Severity:  aggregation.Severity,
			Sources:   make(map[string]int64),
			Data:      make(map[string]interface{}),
			Metadata:  make(map[string]string),
			SubEvents: make([]events.Event, len(aggregation.SubEvents)),
		}

		// Copy maps
		for k, v := range aggregation.Sources {
			aggCopy.Sources[k] = v
		}
		for k, v := range aggregation.Data {
			aggCopy.Data[k] = v
		}
		for k, v := range aggregation.Metadata {
			aggCopy.Metadata[k] = v
		}
		copy(aggCopy.SubEvents, aggregation.SubEvents)

		result[key] = aggCopy
		aggregation.mu.RUnlock()
	}

	return result
}

// GetAggregation returns a specific aggregation
func (ea *EventAggregator) GetAggregation(id string) (*EventAggregation, bool) {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	aggregation, exists := ea.aggregations[id]
	if !exists {
		return nil, false
	}

	// Return a copy
	aggregation.mu.RLock()
	aggCopy := &EventAggregation{
		ID:        aggregation.ID,
		Type:      aggregation.Type,
		Count:     aggregation.Count,
		FirstSeen: aggregation.FirstSeen,
		LastSeen:  aggregation.LastSeen,
		Severity:  aggregation.Severity,
		Sources:   make(map[string]int64),
		Data:      make(map[string]interface{}),
		Metadata:  make(map[string]string),
		SubEvents: make([]events.Event, len(aggregation.SubEvents)),
	}

	// Copy maps
	for k, v := range aggregation.Sources {
		aggCopy.Sources[k] = v
	}
	for k, v := range aggregation.Data {
		aggCopy.Data[k] = v
	}
	for k, v := range aggregation.Metadata {
		aggCopy.Metadata[k] = v
	}
	copy(aggCopy.SubEvents, aggregation.SubEvents)

	aggregation.mu.RUnlock()
	return aggCopy, true
}

// GetMetrics returns aggregator metrics
func (ea *EventAggregator) GetMetrics() map[string]interface{} {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	totalEvents := int64(0)
	severityCounts := make(map[string]int64)
	sourceCounts := make(map[string]int64)

	for _, aggregation := range ea.aggregations {
		aggregation.mu.RLock()
		totalEvents += aggregation.Count
		severityCounts[aggregation.Severity]++
		for source := range aggregation.Sources {
			sourceCounts[source]++
		}
		aggregation.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_aggregations": len(ea.aggregations),
		"total_events":       totalEvents,
		"severity_counts":    severityCounts,
		"source_counts":      sourceCounts,
		"active":             ea.active,
	}
}
