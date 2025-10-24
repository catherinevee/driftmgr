package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/shared/events"
)

// NotificationService handles real-time event notifications
type NotificationService struct {
	eventBus    *events.EventBus
	subscribers map[string]*NotificationSubscriber
	mu          sync.RWMutex
	config      *NotificationConfig
	active      bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NotificationSubscriber represents a notification subscriber
type NotificationSubscriber struct {
	ID           string
	UserID       string
	Channels     []NotificationChannel
	Filters      []EventFilter
	LastActivity time.Time
	Active       bool
}

// NotificationChannel represents a notification delivery channel
type NotificationChannel struct {
	Type    ChannelType
	Config  map[string]interface{}
	Enabled bool
}

// ChannelType represents the type of notification channel
type ChannelType string

const (
	ChannelTypeWebSocket ChannelType = "websocket"
	ChannelTypeEmail     ChannelType = "email"
	ChannelTypeSlack     ChannelType = "slack"
	ChannelTypeWebhook   ChannelType = "webhook"
	ChannelTypeSMS       ChannelType = "sms"
	ChannelTypePush      ChannelType = "push"
)

// EventFilter represents a filter for events
type EventFilter struct {
	EventTypes []events.EventType
	Sources    []string
	Severity   []string
	Tags       map[string]string
}

// NotificationConfig contains configuration for the notification service
type NotificationConfig struct {
	Enabled         bool          `json:"enabled"`
	MaxSubscribers  int           `json:"max_subscribers"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	InactiveTimeout time.Duration `json:"inactive_timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	BatchSize       int           `json:"batch_size"`
	BatchTimeout    time.Duration `json:"batch_timeout"`
}

// NotificationMessage represents a notification message
type NotificationMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
	UserID    string                 `json:"user_id,omitempty"`
}

// NewNotificationService creates a new notification service
func NewNotificationService(eventBus *events.EventBus, config *NotificationConfig) *NotificationService {
	if config == nil {
		config = &NotificationConfig{
			Enabled:         true,
			MaxSubscribers:  1000,
			CleanupInterval: 5 * time.Minute,
			InactiveTimeout: 30 * time.Minute,
			RetryAttempts:   3,
			RetryDelay:      time.Second,
			BatchSize:       100,
			BatchTimeout:    10 * time.Second,
		}
	}

	return &NotificationService{
		eventBus:    eventBus,
		subscribers: make(map[string]*NotificationSubscriber),
		config:      config,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the notification service
func (ns *NotificationService) Start(ctx context.Context) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if ns.active {
		return fmt.Errorf("notification service already active")
	}

	ns.active = true
	ns.stopChan = make(chan struct{}) // Recreate the channel

	// Start background tasks
	ns.wg.Add(3)
	go ns.eventProcessor(ctx)
	go ns.cleanupInactiveSubscribers(ctx)
	go ns.metricsCollector(ctx)

	log.Println("Notification service started")
	return nil
}

// Stop stops the notification service
func (ns *NotificationService) Stop() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.active {
		return
	}

	ns.active = false
	close(ns.stopChan)
	ns.wg.Wait()

	log.Println("Notification service stopped")
}

// Subscribe creates a new notification subscription
func (ns *NotificationService) Subscribe(userID string, channels []NotificationChannel, filters []EventFilter) (*NotificationSubscriber, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if len(ns.subscribers) >= ns.config.MaxSubscribers {
		return nil, fmt.Errorf("maximum number of subscribers reached")
	}

	subscriber := &NotificationSubscriber{
		ID:           generateSubscriberID(),
		UserID:       userID,
		Channels:     channels,
		Filters:      filters,
		LastActivity: time.Now(),
		Active:       true,
	}

	ns.subscribers[subscriber.ID] = subscriber

	log.Printf("New notification subscriber created: %s for user: %s", subscriber.ID, userID)
	return subscriber, nil
}

// Unsubscribe removes a notification subscription
func (ns *NotificationService) Unsubscribe(subscriberID string) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if _, exists := ns.subscribers[subscriberID]; exists {
		delete(ns.subscribers, subscriberID)
		log.Printf("Notification subscriber removed: %s", subscriberID)
		return nil
	}

	return fmt.Errorf("subscriber not found: %s", subscriberID)
}

// SendNotification sends a notification to a specific subscriber
func (ns *NotificationService) SendNotification(subscriberID string, message *NotificationMessage) error {
	ns.mu.RLock()
	subscriber, exists := ns.subscribers[subscriberID]
	ns.mu.RUnlock()

	if !exists {
		return fmt.Errorf("subscriber not found: %s", subscriberID)
	}

	if !subscriber.Active {
		return fmt.Errorf("subscriber is inactive: %s", subscriberID)
	}

	// Update last activity
	ns.mu.Lock()
	subscriber.LastActivity = time.Now()
	ns.mu.Unlock()

	// Send to all enabled channels
	for _, channel := range subscriber.Channels {
		if !channel.Enabled {
			continue
		}

		if err := ns.sendToChannel(channel, message); err != nil {
			log.Printf("Failed to send notification to channel %s: %v", channel.Type, err)
			// Continue with other channels
		}
	}

	return nil
}

// BroadcastNotification broadcasts a notification to all matching subscribers
func (ns *NotificationService) BroadcastNotification(message *NotificationMessage) error {
	ns.mu.RLock()
	subscribers := make([]*NotificationSubscriber, 0, len(ns.subscribers))
	for _, sub := range ns.subscribers {
		if sub.Active && ns.matchesFilters(message, sub.Filters) {
			subscribers = append(subscribers, sub)
		}
	}
	ns.mu.RUnlock()

	// Send to all matching subscribers
	for _, subscriber := range subscribers {
		message.UserID = subscriber.UserID
		if err := ns.SendNotification(subscriber.ID, message); err != nil {
			log.Printf("Failed to send broadcast notification to subscriber %s: %v", subscriber.ID, err)
		}
	}

	log.Printf("Broadcast notification sent to %d subscribers", len(subscribers))
	return nil
}

// eventProcessor processes events from the event bus
func (ns *NotificationService) eventProcessor(ctx context.Context) {
	defer ns.wg.Done()

	// Subscribe to all events
	filter := events.EventFilter{
		Types: []events.EventType{
			events.EventDiscoveryStarted,
			events.EventDiscoveryCompleted,
			events.EventDiscoveryFailed,
			events.EventDriftDetected,
			events.EventRemediationStarted,
			events.EventRemediationCompleted,
			events.EventRemediationFailed,
			events.EventHealthCheck,
			events.EventConfigChanged,
			events.EventJobFailed,
			events.EventResourceDeleted,
			events.EventResourceImported,
		},
	}

	subscription := ns.eventBus.Subscribe(ctx, filter, ns.config.BatchSize)
	defer ns.eventBus.Unsubscribe(subscription)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ns.stopChan:
			return
		case event := <-subscription.Channel:
			ns.processEvent(event)
		}
	}
}

// processEvent processes a single event and creates notifications
func (ns *NotificationService) processEvent(event events.Event) {
	// Create notification message from event
	message := &NotificationMessage{
		ID:        generateMessageID(),
		Type:      string(event.Type),
		Timestamp: event.Timestamp,
		Data:      event.Data,
		Metadata:  event.Metadata,
	}

	// Set title and message based on event type
	switch event.Type {
	case events.EventDiscoveryStarted:
		message.Title = "Discovery Started"
		message.Message = fmt.Sprintf("Resource discovery started for provider: %s", event.Data["provider"])
		message.Severity = "info"
	case events.EventDiscoveryCompleted:
		message.Title = "Discovery Completed"
		message.Message = fmt.Sprintf("Resource discovery completed. Found %d resources", event.Data["resource_count"])
		message.Severity = "success"
	case events.EventDiscoveryFailed:
		message.Title = "Discovery Failed"
		message.Message = fmt.Sprintf("Resource discovery failed: %s", event.Data["error"])
		message.Severity = "error"
	case events.EventDriftDetected:
		message.Title = "Drift Detected"
		message.Message = fmt.Sprintf("Drift detected in %d resources", event.Data["drift_count"])
		message.Severity = "warning"
	case events.EventRemediationStarted:
		message.Title = "Remediation Started"
		message.Message = fmt.Sprintf("Remediation started for resource: %s", event.Data["resource_id"])
		message.Severity = "info"
	case events.EventRemediationCompleted:
		message.Title = "Remediation Completed"
		message.Message = fmt.Sprintf("Remediation completed successfully for resource: %s", event.Data["resource_id"])
		message.Severity = "success"
	case events.EventRemediationFailed:
		message.Title = "Remediation Failed"
		message.Message = fmt.Sprintf("Remediation failed for resource: %s", event.Data["resource_id"])
		message.Severity = "error"
	case events.EventHealthCheck:
		message.Title = "Health Check"
		message.Message = fmt.Sprintf("Health check: %s", event.Data["status"])
		message.Severity = "info"
	case events.EventConfigChanged:
		message.Title = "Configuration Changed"
		message.Message = fmt.Sprintf("Configuration changed: %s", event.Data["config"])
		message.Severity = "info"
	case events.EventJobFailed:
		message.Title = "Job Failed"
		message.Message = fmt.Sprintf("Job failed: %s", event.Data["job_id"])
		message.Severity = "error"
	case events.EventResourceDeleted:
		message.Title = "Resource Deleted"
		message.Message = fmt.Sprintf("Resource deleted: %s", event.Data["resource_id"])
		message.Severity = "warning"
	case events.EventResourceImported:
		message.Title = "Resource Imported"
		message.Message = fmt.Sprintf("Resource imported: %s", event.Data["resource_id"])
		message.Severity = "info"
	default:
		message.Title = "System Event"
		message.Message = fmt.Sprintf("Event: %s", event.Type)
		message.Severity = "info"
	}

	// Broadcast notification to all matching subscribers
	if err := ns.BroadcastNotification(message); err != nil {
		log.Printf("Failed to broadcast notification: %v", err)
	}
}

// sendToChannel sends a notification to a specific channel
func (ns *NotificationService) sendToChannel(channel NotificationChannel, message *NotificationMessage) error {
	switch channel.Type {
	case ChannelTypeWebSocket:
		return ns.sendWebSocketNotification(channel, message)
	case ChannelTypeEmail:
		return ns.sendEmailNotification(channel, message)
	case ChannelTypeSlack:
		return ns.sendSlackNotification(channel, message)
	case ChannelTypeWebhook:
		return ns.sendWebhookNotification(channel, message)
	case ChannelTypeSMS:
		return ns.sendSMSNotification(channel, message)
	case ChannelTypePush:
		return ns.sendPushNotification(channel, message)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// sendWebSocketNotification sends a WebSocket notification
func (ns *NotificationService) sendWebSocketNotification(channel NotificationChannel, message *NotificationMessage) error {
	// This would integrate with the WebSocket service
	// For now, we'll log the notification
	log.Printf("WebSocket notification: %s - %s", message.Title, message.Message)
	return nil
}

// sendEmailNotification sends an email notification
func (ns *NotificationService) sendEmailNotification(channel NotificationChannel, message *NotificationMessage) error {
	// This would integrate with an email service
	// For now, we'll log the notification
	log.Printf("Email notification: %s - %s", message.Title, message.Message)
	return nil
}

// sendSlackNotification sends a Slack notification
func (ns *NotificationService) sendSlackNotification(channel NotificationChannel, message *NotificationMessage) error {
	// This would integrate with Slack API
	// For now, we'll log the notification
	log.Printf("Slack notification: %s - %s", message.Title, message.Message)
	return nil
}

// sendWebhookNotification sends a webhook notification
func (ns *NotificationService) sendWebhookNotification(channel NotificationChannel, message *NotificationMessage) error {
	// This would send HTTP POST to webhook URL
	// For now, we'll log the notification
	log.Printf("Webhook notification: %s - %s", message.Title, message.Message)
	return nil
}

// sendSMSNotification sends an SMS notification
func (ns *NotificationService) sendSMSNotification(channel NotificationChannel, message *NotificationMessage) error {
	// This would integrate with SMS service
	// For now, we'll log the notification
	log.Printf("SMS notification: %s - %s", message.Title, message.Message)
	return nil
}

// sendPushNotification sends a push notification
func (ns *NotificationService) sendPushNotification(channel NotificationChannel, message *NotificationMessage) error {
	// This would integrate with push notification service
	// For now, we'll log the notification
	log.Printf("Push notification: %s - %s", message.Title, message.Message)
	return nil
}

// matchesFilters checks if a message matches the given filters
func (ns *NotificationService) matchesFilters(message *NotificationMessage, filters []EventFilter) bool {
	if len(filters) == 0 {
		return true // No filters means match all
	}

	for _, filter := range filters {
		if ns.matchesFilter(message, filter) {
			return true
		}
	}

	return false
}

// matchesFilter checks if a message matches a specific filter
func (ns *NotificationService) matchesFilter(message *NotificationMessage, filter EventFilter) bool {
	// Check event types
	if len(filter.EventTypes) > 0 {
		matched := false
		for _, eventType := range filter.EventTypes {
			if events.EventType(message.Type) == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check severity
	if len(filter.Severity) > 0 {
		matched := false
		for _, severity := range filter.Severity {
			if message.Severity == severity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check tags
	if len(filter.Tags) > 0 {
		for key, value := range filter.Tags {
			if message.Metadata[key] != value {
				return false
			}
		}
	}

	return true
}

// cleanupInactiveSubscribers removes inactive subscribers
func (ns *NotificationService) cleanupInactiveSubscribers(ctx context.Context) {
	defer ns.wg.Done()

	ticker := time.NewTicker(ns.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ns.stopChan:
			return
		case <-ticker.C:
			ns.mu.Lock()
			now := time.Now()
			for id, subscriber := range ns.subscribers {
				if now.Sub(subscriber.LastActivity) > ns.config.InactiveTimeout {
					delete(ns.subscribers, id)
					log.Printf("Removed inactive subscriber: %s", id)
				}
			}
			ns.mu.Unlock()
		}
	}
}

// metricsCollector collects and logs metrics
func (ns *NotificationService) metricsCollector(ctx context.Context) {
	defer ns.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ns.stopChan:
			return
		case <-ticker.C:
			ns.mu.RLock()
			subscriberCount := len(ns.subscribers)
			activeCount := 0
			for _, sub := range ns.subscribers {
				if sub.Active {
					activeCount++
				}
			}
			ns.mu.RUnlock()

			log.Printf("Notification service metrics - Total subscribers: %d, Active: %d", subscriberCount, activeCount)
		}
	}
}

// GetSubscriberCount returns the number of active subscribers
func (ns *NotificationService) GetSubscriberCount() int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	count := 0
	for _, sub := range ns.subscribers {
		if sub.Active {
			count++
		}
	}
	return count
}

// GetSubscribers returns all subscribers
func (ns *NotificationService) GetSubscribers() []*NotificationSubscriber {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	subscribers := make([]*NotificationSubscriber, 0, len(ns.subscribers))
	for _, sub := range ns.subscribers {
		subscribers = append(subscribers, sub)
	}
	return subscribers
}

// Helper functions

var subscriberCounter int64

func generateSubscriberID() string {
	subscriberCounter++
	return fmt.Sprintf("sub_%d_%d", time.Now().UnixNano(), subscriberCounter)
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
