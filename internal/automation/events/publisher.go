package events

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/shared/events"
)

// EventPublisher handles publishing automation events
type EventPublisher struct {
	eventBus   *events.EventBus
	config     *PublisherConfig
	middleware []EventMiddleware
}

// PublisherConfig contains configuration for event publishing
type PublisherConfig struct {
	Enabled    bool          `json:"enabled"`
	BufferSize int           `json:"buffer_size"`
	Timeout    time.Duration `json:"timeout"`
	RetryCount int           `json:"retry_count"`
	Topics     []string      `json:"topics"`
}

// AutomationEvent represents an automation-related event
type AutomationEvent struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
	Metadata      map[string]interface{} `json:"metadata"`
	CorrelationID string                 `json:"correlation_id"`
}

// EventMiddleware defines middleware for event processing
type EventMiddleware interface {
	Process(ctx context.Context, event *AutomationEvent) error
}

// convertMetadata converts map[string]interface{} to map[string]string
func convertMetadata(metadata map[string]interface{}) map[string]string {
	if metadata == nil {
		return nil
	}

	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// NewEventPublisher creates a new event publisher with configuration
func NewEventPublisher(eventBus *events.EventBus, config *PublisherConfig) *EventPublisher {
	if config == nil {
		config = &PublisherConfig{
			Enabled:    true,
			BufferSize: 1000,
			Timeout:    30 * time.Second,
			RetryCount: 3,
		}
	}

	return &EventPublisher{
		eventBus:   eventBus,
		config:     config,
		middleware: make([]EventMiddleware, 0),
	}
}

// AddMiddleware adds middleware to the event publisher
func (ep *EventPublisher) AddMiddleware(middleware EventMiddleware) {
	ep.middleware = append(ep.middleware, middleware)
}

// PublishEvent publishes an automation event
func (ep *EventPublisher) PublishEvent(ctx context.Context, event *AutomationEvent) error {
	if !ep.config.Enabled {
		return nil // Silently skip if disabled
	}

	if event == nil {
		return &ValidationError{Field: "event", Message: "event cannot be nil"}
	}

	if event.Type == "" {
		return &ValidationError{Field: "type", Message: "event type cannot be empty"}
	}

	// Set default values
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Data == nil {
		event.Data = make(map[string]interface{})
	}
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}

	// Apply middleware
	for _, middleware := range ep.middleware {
		if err := middleware.Process(ctx, event); err != nil {
			return fmt.Errorf("middleware processing failed: %w", err)
		}
	}

	// Convert AutomationEvent to Event
	systemEvent := events.Event{
		ID:        event.ID,
		Type:      events.EventType(event.Type),
		Timestamp: event.Timestamp,
		Source:    event.Source,
		Data:      event.Data,
		Metadata:  convertMetadata(event.Metadata),
	}

	// Publish to event bus with timeout
	ctx, cancel := context.WithTimeout(ctx, ep.config.Timeout)
	defer cancel()

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= ep.config.RetryCount; attempt++ {
		if err := ep.eventBus.Publish(systemEvent); err != nil {
			lastErr = err
			if attempt < ep.config.RetryCount {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
		} else {
			return nil // Success
		}
	}

	return fmt.Errorf("failed to publish event after %d attempts: %w", ep.config.RetryCount+1, lastErr)
}

// PublishRuleExecutionEvent publishes a rule execution event
func (ep *EventPublisher) PublishRuleExecutionEvent(ctx context.Context, ruleID string, status string, details map[string]interface{}) error {
	event := &AutomationEvent{
		Type:   "rule.execution",
		Source: "rule_engine",
		Data: map[string]interface{}{
			"rule_id": ruleID,
			"status":  status,
			"details": details,
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   "1.0",
		},
	}

	return ep.PublishEvent(ctx, event)
}

// PublishSchedulerEvent publishes a scheduler event
func (ep *EventPublisher) PublishSchedulerEvent(ctx context.Context, eventType string, jobID string, details map[string]interface{}) error {
	event := &AutomationEvent{
		Type:   "scheduler." + eventType,
		Source: "scheduler",
		Data: map[string]interface{}{
			"job_id":  jobID,
			"details": details,
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   "1.0",
		},
	}

	return ep.PublishEvent(ctx, event)
}

// PublishServiceEvent publishes a service event
func (ep *EventPublisher) PublishServiceEvent(ctx context.Context, eventType string, serviceName string, details map[string]interface{}) error {
	event := &AutomationEvent{
		Type:   "service." + eventType,
		Source: serviceName,
		Data: map[string]interface{}{
			"service": serviceName,
			"details": details,
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   "1.0",
		},
	}

	return ep.PublishEvent(ctx, event)
}

// PublishRemediationEvent publishes a remediation event
func (ep *EventPublisher) PublishRemediationEvent(ctx context.Context, eventType string, resourceID string, details map[string]interface{}) error {
	event := &AutomationEvent{
		Type:   "remediation." + eventType,
		Source: "remediation_engine",
		Data: map[string]interface{}{
			"resource_id": resourceID,
			"details":     details,
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   "1.0",
		},
	}

	return ep.PublishEvent(ctx, event)
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (ve *ValidationError) Error() string {
	return "validation error in field '" + ve.Field + "': " + ve.Message
}
