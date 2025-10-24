package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/models"
	sharedEvents "github.com/catherinevee/driftmgr/internal/shared/events"
)

// NotificationAction handles sending notifications through various channels
type NotificationAction struct {
	notificationService *events.NotificationService
	eventBus            *events.EventBus
}

// NotificationConfig represents the configuration for a notification action
type NotificationConfig struct {
	Type       string                 `json:"type"`       // notification type (email, slack, webhook, etc.)
	Recipients []string               `json:"recipients"` // list of recipients
	Subject    string                 `json:"subject"`    // notification subject
	Message    string                 `json:"message"`    // notification message
	Priority   string                 `json:"priority"`   // priority level (low, medium, high, critical)
	Channels   []string               `json:"channels"`   // notification channels to use
	Data       map[string]interface{} `json:"data"`       // additional data
	Template   string                 `json:"template"`   // template to use for formatting
}

// NewNotificationAction creates a new notification action handler
func NewNotificationAction(notificationService *events.NotificationService, eventBus *events.EventBus) *NotificationAction {
	return &NotificationAction{
		notificationService: notificationService,
		eventBus:            eventBus,
	}
}

// Execute executes a notification action
func (na *NotificationAction) Execute(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	startTime := time.Now()

	// Parse notification configuration
	config, err := na.parseConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse notification configuration: %w", err)
	}

	// Process template if specified
	message := config.Message
	if config.Template != "" {
		message, err = na.processTemplate(config.Template, context)
		if err != nil {
			return nil, fmt.Errorf("failed to process template: %w", err)
		}
	}

	// Create notification channels
	channels := na.createNotificationChannels(config)

	// Create event filters for the notification
	filters := na.createEventFilters(config)

	// Subscribe to notifications (if not already subscribed)
	subscriber, err := na.notificationService.Subscribe("automation", channels, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to notifications: %w", err)
	}

	// Create notification message
	notificationMessage := &events.NotificationMessage{
		ID:        fmt.Sprintf("automation_%d", time.Now().UnixNano()),
		Type:      config.Type,
		Title:     config.Subject,
		Message:   message,
		Severity:  na.mapPriorityToSeverity(config.Priority),
		Timestamp: time.Now(),
		Data:      config.Data,
		Metadata: map[string]string{
			"action_id":   action.ID,
			"action_name": action.Name,
			"source":      "automation",
		},
	}

	// Send notification
	err = na.notificationService.SendNotification(subscriber.ID, notificationMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}

	// Publish automation event
	automationEvent := events.Event{
		Type:      events.EventType("automation.notification_sent"),
		Timestamp: time.Now(),
		Source:    "automation_service",
		Data: map[string]interface{}{
			"action_id":         action.ID,
			"action_name":       action.Name,
			"notification_type": config.Type,
			"recipients":        config.Recipients,
			"subject":           config.Subject,
			"priority":          config.Priority,
			"action_type":       "notification",
		},
	}

	na.eventBus.Publish(automationEvent)

	// Create action result
	result := &models.ActionResult{
		ActionID:      action.ID,
		Status:        models.ActionStatusCompleted,
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
		Output: models.JSONB(map[string]interface{}{
			"notification_type": config.Type,
			"recipients":        config.Recipients,
			"subject":           config.Subject,
			"message":           message,
			"priority":          config.Priority,
			"channels":          config.Channels,
			"sent_at":           time.Now(),
			"subscriber_id":     subscriber.ID,
		}),
	}

	return result, nil
}

// Validate validates a notification action
func (na *NotificationAction) Validate(action *models.AutomationAction) error {
	if action.Type != models.ActionTypeNotification {
		return fmt.Errorf("invalid action type: expected %s, got %s", models.ActionTypeNotification, action.Type)
	}

	config, err := na.parseConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if config.Type == "" {
		return fmt.Errorf("notification type is required")
	}

	if len(config.Recipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if config.Subject == "" {
		return fmt.Errorf("notification subject is required")
	}

	if config.Message == "" && config.Template == "" {
		return fmt.Errorf("either message or template is required")
	}

	return nil
}

// parseConfig parses the notification configuration from JSONB
func (na *NotificationAction) parseConfig(config models.JSONB) (*NotificationConfig, error) {
	var notificationConfig NotificationConfig

	// JSONB is already a map[string]interface{}
	configMap := config

	// Parse type
	if typeVal, ok := configMap["type"].(string); ok {
		notificationConfig.Type = typeVal
	}

	// Parse recipients
	if recipientsVal, ok := configMap["recipients"].([]interface{}); ok {
		notificationConfig.Recipients = make([]string, len(recipientsVal))
		for i, recipient := range recipientsVal {
			if recipientStr, ok := recipient.(string); ok {
				notificationConfig.Recipients[i] = recipientStr
			}
		}
	}

	// Parse subject
	if subjectVal, ok := configMap["subject"].(string); ok {
		notificationConfig.Subject = subjectVal
	}

	// Parse message
	if messageVal, ok := configMap["message"].(string); ok {
		notificationConfig.Message = messageVal
	}

	// Parse priority
	if priorityVal, ok := configMap["priority"].(string); ok {
		notificationConfig.Priority = priorityVal
	}

	// Parse channels
	if channelsVal, ok := configMap["channels"].([]interface{}); ok {
		notificationConfig.Channels = make([]string, len(channelsVal))
		for i, channel := range channelsVal {
			if channelStr, ok := channel.(string); ok {
				notificationConfig.Channels[i] = channelStr
			}
		}
	}

	// Parse data
	if dataVal, ok := configMap["data"].(map[string]interface{}); ok {
		notificationConfig.Data = dataVal
	}

	// Parse template
	if templateVal, ok := configMap["template"].(string); ok {
		notificationConfig.Template = templateVal
	}

	return &notificationConfig, nil
}

// processTemplate processes a notification template with context data
func (na *NotificationAction) processTemplate(template string, context map[string]interface{}) (string, error) {
	// Simple template processing - replace {{key}} with values from context
	// In a real implementation, you might use a more sophisticated templating engine
	result := template
	for key, value := range context {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return result, nil
}

// createNotificationChannels creates notification channels from configuration
func (na *NotificationAction) createNotificationChannels(config *NotificationConfig) []events.NotificationChannel {
	channels := make([]events.NotificationChannel, 0)

	// If no specific channels are configured, use default channels
	if len(config.Channels) == 0 {
		config.Channels = []string{"websocket", "email"}
	}

	for _, channelType := range config.Channels {
		channel := events.NotificationChannel{
			Type:    events.ChannelType(channelType),
			Enabled: true,
			Config: map[string]interface{}{
				"recipients": config.Recipients,
				"priority":   config.Priority,
			},
		}
		channels = append(channels, channel)
	}

	return channels
}

// createEventFilters creates event filters for the notification
func (na *NotificationAction) createEventFilters(config *NotificationConfig) []events.EventFilter {
	// Create a filter that matches the notification type
	filter := events.EventFilter{
		EventTypes: []sharedEvents.EventType{sharedEvents.EventType(config.Type)},
		Severity:   []string{na.mapPriorityToSeverity(config.Priority)},
	}

	return []events.EventFilter{filter}
}

// mapPriorityToSeverity maps priority levels to severity levels
func (na *NotificationAction) mapPriorityToSeverity(priority string) string {
	switch priority {
	case "critical":
		return "error"
	case "high":
		return "warning"
	case "medium":
		return "info"
	case "low":
		return "info"
	default:
		return "info"
	}
}
