package engine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
)

// TriggerManagerImpl implements the TriggerManager interface
type TriggerManagerImpl struct {
	workflowRepo   WorkflowRepository
	eventBus       EventBus
	engine         *Engine
	activeTriggers map[uuid.UUID]*TriggerContext
	mu             sync.RWMutex
	stopChan       chan struct{}
	isRunning      bool
}

// TriggerContext holds the context for a trigger
type TriggerContext struct {
	Workflow    *models.AutomationWorkflow
	Trigger     *models.AutomationTrigger
	LastTrigger time.Time
	IsActive    bool
	CancelFunc  context.CancelFunc
}

// NewTriggerManager creates a new trigger manager
func NewTriggerManager(workflowRepo WorkflowRepository, eventBus EventBus, engine *Engine) *TriggerManagerImpl {
	return &TriggerManagerImpl{
		workflowRepo:   workflowRepo,
		eventBus:       eventBus,
		engine:         engine,
		activeTriggers: make(map[uuid.UUID]*TriggerContext),
		stopChan:       make(chan struct{}),
	}
}

// RegisterTrigger registers a workflow trigger
func (tm *TriggerManagerImpl) RegisterTrigger(ctx context.Context, workflow *models.AutomationWorkflow) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if trigger is already registered
	if _, exists := tm.activeTriggers[workflow.ID]; exists {
		return fmt.Errorf("trigger already registered for workflow %s", workflow.ID)
	}

	// Create trigger context
	triggerCtx, cancel := context.WithCancel(ctx)
	triggerContext := &TriggerContext{
		Workflow:   workflow,
		Trigger:    &workflow.Trigger,
		IsActive:   true,
		CancelFunc: cancel,
	}

	// Register trigger based on type
	switch workflow.Trigger.Type {
	case models.TriggerTypeManual:
		// Manual triggers don't need active monitoring
		triggerContext.IsActive = false
	case models.TriggerTypeScheduled:
		if err := tm.registerScheduledTrigger(triggerCtx, triggerContext); err != nil {
			cancel()
			return fmt.Errorf("failed to register scheduled trigger: %w", err)
		}
	case models.TriggerTypeEvent:
		if err := tm.registerEventTrigger(triggerCtx, triggerContext); err != nil {
			cancel()
			return fmt.Errorf("failed to register event trigger: %w", err)
		}
	case models.TriggerTypeWebhook:
		if err := tm.registerWebhookTrigger(triggerCtx, triggerContext); err != nil {
			cancel()
			return fmt.Errorf("failed to register webhook trigger: %w", err)
		}
	default:
		cancel()
		return fmt.Errorf("unsupported trigger type: %s", workflow.Trigger.Type)
	}

	// Add to active triggers
	tm.activeTriggers[workflow.ID] = triggerContext

	log.Printf("Registered trigger for workflow %s (type: %s)", workflow.ID, workflow.Trigger.Type)
	return nil
}

// UnregisterTrigger unregisters a workflow trigger
func (tm *TriggerManagerImpl) UnregisterTrigger(ctx context.Context, workflowID uuid.UUID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	triggerContext, exists := tm.activeTriggers[workflowID]
	if !exists {
		return fmt.Errorf("trigger not found for workflow %s", workflowID)
	}

	// Cancel the trigger context
	triggerContext.CancelFunc()

	// Remove from active triggers
	delete(tm.activeTriggers, workflowID)

	log.Printf("Unregistered trigger for workflow %s", workflowID)
	return nil
}

// StartTriggerMonitoring starts monitoring for triggers
func (tm *TriggerManagerImpl) StartTriggerMonitoring(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.isRunning {
		return fmt.Errorf("trigger monitoring is already running")
	}

	tm.isRunning = true
	tm.stopChan = make(chan struct{})

	// Start monitoring goroutines
	go tm.monitorScheduledTriggers(ctx)
	go tm.monitorEventTriggers(ctx)
	go tm.monitorWebhookTriggers(ctx)

	log.Println("Trigger monitoring started")
	return nil
}

// StopTriggerMonitoring stops monitoring for triggers
func (tm *TriggerManagerImpl) StopTriggerMonitoring(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.isRunning {
		return fmt.Errorf("trigger monitoring is not running")
	}

	// Signal stop
	close(tm.stopChan)

	// Cancel all active triggers
	for _, triggerContext := range tm.activeTriggers {
		triggerContext.CancelFunc()
	}

	tm.isRunning = false
	log.Println("Trigger monitoring stopped")
	return nil
}

// registerScheduledTrigger registers a scheduled trigger
func (tm *TriggerManagerImpl) registerScheduledTrigger(ctx context.Context, triggerContext *TriggerContext) error {
	// Parse schedule configuration
	schedule, err := tm.parseScheduleConfig(triggerContext.Trigger.Configuration)
	if err != nil {
		return fmt.Errorf("failed to parse schedule configuration: %w", err)
	}

	// Validate schedule
	if err := tm.validateSchedule(schedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	log.Printf("Registered scheduled trigger for workflow %s with schedule: %s",
		triggerContext.Workflow.ID, schedule.Expression)
	return nil
}

// registerEventTrigger registers an event trigger
func (tm *TriggerManagerImpl) registerEventTrigger(ctx context.Context, triggerContext *TriggerContext) error {
	// Parse event configuration
	eventConfig, err := tm.parseEventConfig(triggerContext.Trigger.Configuration)
	if err != nil {
		return fmt.Errorf("failed to parse event configuration: %w", err)
	}

	// Validate event configuration
	if err := tm.validateEventConfig(eventConfig); err != nil {
		return fmt.Errorf("invalid event configuration: %w", err)
	}

	// Subscribe to events
	handler := &EventTriggerHandler{
		triggerManager: tm,
		workflowID:     triggerContext.Workflow.ID,
		eventConfig:    eventConfig,
	}

	if err := tm.eventBus.SubscribeToEvents(ctx, eventConfig.EventType, handler); err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	log.Printf("Registered event trigger for workflow %s with event type: %s",
		triggerContext.Workflow.ID, eventConfig.EventType)
	return nil
}

// registerWebhookTrigger registers a webhook trigger
func (tm *TriggerManagerImpl) registerWebhookTrigger(ctx context.Context, triggerContext *TriggerContext) error {
	// Parse webhook configuration
	webhookConfig, err := tm.parseWebhookConfig(triggerContext.Trigger.Configuration)
	if err != nil {
		return fmt.Errorf("failed to parse webhook configuration: %w", err)
	}

	// Validate webhook configuration
	if err := tm.validateWebhookConfig(webhookConfig); err != nil {
		return fmt.Errorf("invalid webhook configuration: %w", err)
	}

	// Register webhook endpoint (this would typically be done with an HTTP server)
	log.Printf("Registered webhook trigger for workflow %s with endpoint: %s",
		triggerContext.Workflow.ID, webhookConfig.Endpoint)
	return nil
}

// monitorScheduledTriggers monitors scheduled triggers
func (tm *TriggerManagerImpl) monitorScheduledTriggers(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tm.stopChan:
			return
		case <-ticker.C:
			tm.checkScheduledTriggers(ctx)
		}
	}
}

// monitorEventTriggers monitors event triggers
func (tm *TriggerManagerImpl) monitorEventTriggers(ctx context.Context) {
	// Event triggers are handled by the event bus subscription
	// This goroutine just keeps the monitoring alive
	select {
	case <-ctx.Done():
		return
	case <-tm.stopChan:
		return
	}
}

// monitorWebhookTriggers monitors webhook triggers
func (tm *TriggerManagerImpl) monitorWebhookTriggers(ctx context.Context) {
	// Webhook triggers are handled by the HTTP server
	// This goroutine just keeps the monitoring alive
	select {
	case <-ctx.Done():
		return
	case <-tm.stopChan:
		return
	}
}

// checkScheduledTriggers checks all scheduled triggers
func (tm *TriggerManagerImpl) checkScheduledTriggers(ctx context.Context) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	now := time.Now()

	for workflowID, triggerContext := range tm.activeTriggers {
		if triggerContext.Trigger.Type != models.TriggerTypeScheduled {
			continue
		}

		// Parse schedule configuration
		schedule, err := tm.parseScheduleConfig(triggerContext.Trigger.Configuration)
		if err != nil {
			log.Printf("Failed to parse schedule for workflow %s: %v", workflowID, err)
			continue
		}

		// Check if it's time to trigger
		if tm.shouldTrigger(now, schedule, triggerContext.LastTrigger) {
			// Trigger the workflow
			go tm.triggerWorkflow(ctx, workflowID, map[string]interface{}{
				"trigger_type": "scheduled",
				"triggered_at": now,
				"schedule":     schedule.Expression,
			})

			// Update last trigger time
			triggerContext.LastTrigger = now
		}
	}
}

// shouldTrigger determines if a scheduled trigger should fire
func (tm *TriggerManagerImpl) shouldTrigger(now time.Time, schedule *ScheduleConfig, lastTrigger time.Time) bool {
	// Simple implementation - in production, you'd use a proper cron parser
	switch schedule.Type {
	case "interval":
		interval, err := time.ParseDuration(schedule.Expression)
		if err != nil {
			return false
		}
		return now.Sub(lastTrigger) >= interval
	case "cron":
		// For now, just trigger every minute for cron expressions
		// In production, use a proper cron library
		return now.Sub(lastTrigger) >= time.Minute
	default:
		return false
	}
}

// triggerWorkflow triggers a workflow execution
func (tm *TriggerManagerImpl) triggerWorkflow(ctx context.Context, workflowID uuid.UUID, input map[string]interface{}) {
	// Execute the workflow
	_, err := tm.engine.ExecuteWorkflow(ctx, workflowID, input)
	if err != nil {
		log.Printf("Failed to execute workflow %s: %v", workflowID, err)
		return
	}

	log.Printf("Triggered workflow %s", workflowID)
}

// Configuration parsing structures
type ScheduleConfig struct {
	Type       string `json:"type"`       // "interval" or "cron"
	Expression string `json:"expression"` // e.g., "5m", "0 0 * * *"
	Timezone   string `json:"timezone"`
}

type EventConfig struct {
	EventType  string                 `json:"event_type"`
	Filters    map[string]interface{} `json:"filters"`
	Conditions []EventCondition       `json:"conditions"`
}

type EventCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type WebhookConfig struct {
	Endpoint string            `json:"endpoint"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Secret   string            `json:"secret"`
	Timeout  time.Duration     `json:"timeout"`
}

// parseScheduleConfig parses schedule configuration
func (tm *TriggerManagerImpl) parseScheduleConfig(config models.JSONB) (*ScheduleConfig, error) {
	var schedule ScheduleConfig
	if err := config.Unmarshal(&schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedule config: %w", err)
	}
	return &schedule, nil
}

// parseEventConfig parses event configuration
func (tm *TriggerManagerImpl) parseEventConfig(config models.JSONB) (*EventConfig, error) {
	var eventConfig EventConfig
	if err := config.Unmarshal(&eventConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event config: %w", err)
	}
	return &eventConfig, nil
}

// parseWebhookConfig parses webhook configuration
func (tm *TriggerManagerImpl) parseWebhookConfig(config models.JSONB) (*WebhookConfig, error) {
	var webhookConfig WebhookConfig
	if err := config.Unmarshal(&webhookConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook config: %w", err)
	}
	return &webhookConfig, nil
}

// validateSchedule validates a schedule configuration
func (tm *TriggerManagerImpl) validateSchedule(schedule *ScheduleConfig) error {
	if schedule.Type == "" {
		return fmt.Errorf("schedule type is required")
	}

	if schedule.Expression == "" {
		return fmt.Errorf("schedule expression is required")
	}

	switch schedule.Type {
	case "interval":
		_, err := time.ParseDuration(schedule.Expression)
		if err != nil {
			return fmt.Errorf("invalid interval expression: %w", err)
		}
	case "cron":
		// In production, validate cron expression with a proper library
		if schedule.Expression == "" {
			return fmt.Errorf("cron expression is required")
		}
	default:
		return fmt.Errorf("unsupported schedule type: %s", schedule.Type)
	}

	return nil
}

// validateEventConfig validates an event configuration
func (tm *TriggerManagerImpl) validateEventConfig(eventConfig *EventConfig) error {
	if eventConfig.EventType == "" {
		return fmt.Errorf("event type is required")
	}

	return nil
}

// validateWebhookConfig validates a webhook configuration
func (tm *TriggerManagerImpl) validateWebhookConfig(webhookConfig *WebhookConfig) error {
	if webhookConfig.Endpoint == "" {
		return fmt.Errorf("webhook endpoint is required")
	}

	if webhookConfig.Method == "" {
		webhookConfig.Method = "POST" // Default to POST
	}

	return nil
}

// EventTriggerHandler handles event triggers
type EventTriggerHandler struct {
	triggerManager *TriggerManagerImpl
	workflowID     uuid.UUID
	eventConfig    *EventConfig
}

// HandleEvent handles an automation event
func (h *EventTriggerHandler) HandleEvent(ctx context.Context, event *models.AutomationEvent) error {
	// Check if this event matches our trigger criteria
	if !h.matchesEvent(event) {
		return nil
	}

	// Trigger the workflow
	input := map[string]interface{}{
		"trigger_type": "event",
		"event":        event,
		"triggered_at": time.Now(),
	}

	_, err := h.triggerManager.engine.ExecuteWorkflow(ctx, h.workflowID, input)
	if err != nil {
		log.Printf("Failed to execute workflow %s for event %s: %v", h.workflowID, event.ID, err)
		return err
	}

	log.Printf("Triggered workflow %s for event %s", h.workflowID, event.ID)
	return nil
}

// matchesEvent checks if an event matches the trigger criteria
func (h *EventTriggerHandler) matchesEvent(event *models.AutomationEvent) bool {
	// Check event type
	if event.Type != h.eventConfig.EventType {
		return false
	}

	// Check filters
	for field, expectedValue := range h.eventConfig.Filters {
		actualValue := h.getEventFieldValue(event, field)
		if actualValue != expectedValue {
			return false
		}
	}

	// Check conditions
	for _, condition := range h.eventConfig.Conditions {
		actualValue := h.getEventFieldValue(event, condition.Field)
		if !h.evaluateCondition(actualValue, condition.Operator, condition.Value) {
			return false
		}
	}

	return true
}

// getEventFieldValue gets a field value from an event
func (h *EventTriggerHandler) getEventFieldValue(event *models.AutomationEvent, field string) interface{} {
	switch field {
	case "type":
		return event.Type
	case "workflow_id":
		return event.WorkflowID
	case "execution_id":
		return event.ExecutionID
	case "message":
		return event.Message
	case "timestamp":
		return event.Timestamp
	default:
		// Try to get from data
		if event.Data != nil {
			var data map[string]interface{}
			if err := event.Data.Unmarshal(&data); err == nil {
				return data[field]
			}
		}
		return nil
	}
}

// evaluateCondition evaluates a condition
func (h *EventTriggerHandler) evaluateCondition(actualValue interface{}, operator string, expectedValue interface{}) bool {
	switch operator {
	case "equals":
		return actualValue == expectedValue
	case "not_equals":
		return actualValue != expectedValue
	case "contains":
		if actualStr, ok := actualValue.(string); ok {
			if expectedStr, ok := expectedValue.(string); ok {
				return strings.Contains(actualStr, expectedStr)
			}
		}
		return false
	case "greater_than":
		if actualNum, ok := actualValue.(float64); ok {
			if expectedNum, ok := expectedValue.(float64); ok {
				return actualNum > expectedNum
			}
		}
		return false
	case "less_than":
		if actualNum, ok := actualValue.(float64); ok {
			if expectedNum, ok := expectedValue.(float64); ok {
				return actualNum < expectedNum
			}
		}
		return false
	default:
		return false
	}
}
