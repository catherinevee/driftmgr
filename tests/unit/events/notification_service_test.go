package events

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	sharedEvents "github.com/catherinevee/driftmgr/internal/shared/events"
	"github.com/stretchr/testify/assert"
)

func TestNotificationService_NewNotificationService(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	config := &events.NotificationConfig{
		Enabled:         true,
		MaxSubscribers:  100,
		CleanupInterval: 1 * time.Minute,
		InactiveTimeout: 5 * time.Minute,
	}

	service := events.NewNotificationService(eventBus, config)

	if service == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestNotificationService_Subscribe(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	service := events.NewNotificationService(eventBus, nil)

	channels := []events.NotificationChannel{
		{
			Type:    events.ChannelTypeWebSocket,
			Enabled: true,
			Config:  map[string]interface{}{"endpoint": "ws://localhost:8080"},
		},
	}

	filters := []events.EventFilter{
		{
			EventTypes: []sharedEvents.EventType{sharedEvents.EventDriftDetected},
			Severity:   []string{"warning", "error"},
		},
	}

	subscriber, err := service.Subscribe("user123", channels, filters)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if subscriber == nil {
		t.Fatal("Expected subscriber to be created")
	}

	if subscriber.UserID != "user123" {
		t.Errorf("Expected UserID to be 'user123', got %s", subscriber.UserID)
	}
}

func TestNotificationService_Unsubscribe(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	service := events.NewNotificationService(eventBus, nil)

	channels := []events.NotificationChannel{
		{Type: events.ChannelTypeWebSocket, Enabled: true},
	}

	subscriber, err := service.Subscribe("user123", channels, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = service.Unsubscribe(subscriber.ID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Try to unsubscribe again
	err = service.Unsubscribe(subscriber.ID)
	if err == nil {
		t.Error("Expected error when unsubscribing non-existent subscriber")
	}
}

func TestNotificationService_SendNotification(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	service := events.NewNotificationService(eventBus, nil)

	channels := []events.NotificationChannel{
		{Type: events.ChannelTypeWebSocket, Enabled: true},
	}

	subscriber, err := service.Subscribe("user123", channels, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	message := &events.NotificationMessage{
		ID:        "msg123",
		Type:      "test",
		Title:     "Test Message",
		Message:   "This is a test message",
		Severity:  "info",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": true},
	}

	err = service.SendNotification(subscriber.ID, message)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNotificationService_GetSubscriberCount(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	config := &events.NotificationConfig{
		Enabled:         true,
		MaxSubscribers:  100,
		CleanupInterval: 1 * time.Hour, // Disable cleanup for this test
		InactiveTimeout: 1 * time.Hour, // Disable cleanup for this test
	}
	service := events.NewNotificationService(eventBus, config)

	channels := []events.NotificationChannel{
		{Type: events.ChannelTypeWebSocket, Enabled: true},
	}

	// Initially no subscribers
	count := service.GetSubscriberCount()
	if count != 0 {
		t.Errorf("Expected 0 subscribers, got %d", count)
	}

	// Add first subscriber
	sub1, err := service.Subscribe("user1", channels, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check count after first subscriber
	count = service.GetSubscriberCount()
	if count != 1 {
		t.Errorf("Expected 1 subscriber, got %d", count)
	}

	// Add second subscriber
	sub2, err := service.Subscribe("user2", channels, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check count after second subscriber
	count = service.GetSubscriberCount()
	if count != 2 {
		t.Errorf("Expected 2 subscribers, got %d", count)
	}

	// Verify subscribers exist
	allSubs := service.GetSubscribers()
	if len(allSubs) != 2 {
		t.Errorf("Expected 2 total subscribers, got %d", len(allSubs))
	}

	// Check that our subscribers are in the list
	found1, found2 := false, false
	for _, sub := range allSubs {
		if sub.ID == sub1.ID {
			found1 = true
		}
		if sub.ID == sub2.ID {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("Expected both subscribers to be found in the list")
	}
}

func TestNotificationService_StartStop(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	service := events.NewNotificationService(eventBus, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start service
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Stop service
	service.Stop()
}

func TestNotificationService_Integration(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	config := &events.NotificationConfig{
		Enabled:         true,
		MaxSubscribers:  10,
		CleanupInterval: 30 * time.Second,
		InactiveTimeout: 2 * time.Minute,
	}

	service := events.NewNotificationService(eventBus, config)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the service
	err := service.Start(ctx)
	assert.NoError(t, err)

	// Subscribe to events
	channels := []events.NotificationChannel{
		{Type: events.ChannelTypeWebSocket, Enabled: true},
	}

	filters := []events.EventFilter{
		{
			EventTypes: []sharedEvents.EventType{sharedEvents.EventDriftDetected},
			Severity:   []string{"warning", "error"},
		},
	}

	subscriber, err := service.Subscribe("test-user", channels, filters)
	assert.NoError(t, err)
	assert.NotNil(t, subscriber)

	// Publish a test event
	testEvent := sharedEvents.Event{
		Type:      sharedEvents.EventDriftDetected,
		Timestamp: time.Now(),
		Source:    "test-source",
		Data: map[string]interface{}{
			"drift_count": 5,
			"severity":    "warning",
		},
	}

	err = eventBus.Publish(testEvent)
	assert.NoError(t, err)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop the service
	service.Stop()
}
