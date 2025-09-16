package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventBus(t *testing.T) {
	bufferSize := 100
	eventBus := NewEventBus(bufferSize)

	assert.NotNil(t, eventBus)
	assert.NotNil(t, eventBus.subscriptions)
	assert.NotNil(t, eventBus.handlers)
	assert.NotNil(t, eventBus.buffer)
	assert.Equal(t, bufferSize, eventBus.bufferSize)
	assert.NotNil(t, eventBus.metrics)
	assert.False(t, eventBus.closed)
}

func TestEventBus_Publish(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	event := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
		Data: map[string]interface{}{
			"message": "Test event",
		},
	}

	err := eventBus.Publish(event)
	assert.NoError(t, err)

	// Verify event was added to buffer
	buffer := eventBus.GetBuffer()
	assert.Len(t, buffer, 1)
	assert.Equal(t, EventDiscoveryStarted, buffer[0].Type)
	assert.Equal(t, "test-source", buffer[0].Source)
	assert.Equal(t, "Test event", buffer[0].Data["message"])
	assert.False(t, buffer[0].Timestamp.IsZero())
	assert.NotEmpty(t, buffer[0].ID)
}

func TestEventBus_Publish_WithExistingID(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	event := Event{
		ID:     "custom-id",
		Type:   EventDiscoveryStarted,
		Source: "test-source",
	}

	err := eventBus.Publish(event)
	assert.NoError(t, err)

	buffer := eventBus.GetBuffer()
	assert.Len(t, buffer, 1)
	assert.Equal(t, "custom-id", buffer[0].ID)
}

func TestEventBus_Publish_WithExistingTimestamp(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	timestamp := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	event := Event{
		Type:      EventDiscoveryStarted,
		Source:    "test-source",
		Timestamp: timestamp,
	}

	err := eventBus.Publish(event)
	assert.NoError(t, err)

	buffer := eventBus.GetBuffer()
	assert.Len(t, buffer, 1)
	assert.Equal(t, timestamp, buffer[0].Timestamp)
}

func TestEventBus_Publish_Closed(t *testing.T) {
	eventBus := NewEventBus(10)
	eventBus.Close()

	event := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
	}

	err := eventBus.Publish(event)
	assert.NoError(t, err) // Should not error when closed

	buffer := eventBus.GetBuffer()
	assert.Len(t, buffer, 0) // Should not add to buffer when closed
}

func TestEventBus_Subscribe(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	ctx := context.Background()
	filter := EventFilter{
		Types: []EventType{EventDiscoveryStarted},
	}

	sub := eventBus.Subscribe(ctx, filter, 5)
	assert.NotNil(t, sub)
	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, filter, sub.Filter)
	assert.NotNil(t, sub.Channel)
	assert.NotNil(t, sub.ctx)
	assert.NotNil(t, sub.cancel)

	// Verify subscription was added
	metrics := eventBus.GetMetrics()
	assert.Equal(t, 1, metrics.SubscriptionCount)
}

func TestEventBus_Subscribe_Closed(t *testing.T) {
	eventBus := NewEventBus(10)
	eventBus.Close()

	ctx := context.Background()
	filter := EventFilter{
		Types: []EventType{EventDiscoveryStarted},
	}

	sub := eventBus.Subscribe(ctx, filter, 5)
	assert.Nil(t, sub)
}

func TestEventBus_Subscribe_WithBufferedEvents(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	// Publish some events before subscription
	event1 := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
		Data:   map[string]interface{}{"message": "Event 1"},
	}
	event2 := Event{
		Type:   EventDiscoveryCompleted,
		Source: "test-source",
		Data:   map[string]interface{}{"message": "Event 2"},
	}

	eventBus.Publish(event1)
	eventBus.Publish(event2)

	// Subscribe to discovery events
	ctx := context.Background()
	filter := EventFilter{
		Types: []EventType{EventDiscoveryStarted, EventDiscoveryCompleted},
	}

	sub := eventBus.Subscribe(ctx, filter, 5)
	require.NotNil(t, sub)

	// Should receive buffered events
	select {
	case receivedEvent := <-sub.Channel:
		assert.Contains(t, []EventType{EventDiscoveryStarted, EventDiscoveryCompleted}, receivedEvent.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive buffered event")
	}

	select {
	case receivedEvent := <-sub.Channel:
		assert.Contains(t, []EventType{EventDiscoveryStarted, EventDiscoveryCompleted}, receivedEvent.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive second buffered event")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	ctx := context.Background()
	filter := EventFilter{
		Types: []EventType{EventDiscoveryStarted},
	}

	sub := eventBus.Subscribe(ctx, filter, 5)
	require.NotNil(t, sub)

	// Verify subscription exists
	metrics := eventBus.GetMetrics()
	assert.Equal(t, 1, metrics.SubscriptionCount)

	// Unsubscribe
	eventBus.Unsubscribe(sub)

	// Verify subscription was removed
	metrics = eventBus.GetMetrics()
	assert.Equal(t, 0, metrics.SubscriptionCount)
}

func TestEventBus_Unsubscribe_Nil(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	// Should not panic
	assert.NotPanics(t, func() {
		eventBus.Unsubscribe(nil)
	})
}

func TestEventBus_RegisterHandler(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	handlerCalled := false
	handler := func(event Event) {
		handlerCalled = true
		assert.Equal(t, EventDiscoveryStarted, event.Type)
	}

	eventBus.RegisterHandler(EventDiscoveryStarted, handler)

	// Publish event
	event := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
	}

	eventBus.Publish(event)

	// Wait for handler to be called
	time.Sleep(10 * time.Millisecond)
	assert.True(t, handlerCalled)

	// Verify handler was registered
	metrics := eventBus.GetMetrics()
	assert.Equal(t, 1, metrics.ActiveHandlers)
}

func TestEventBus_RegisterHandler_Closed(t *testing.T) {
	eventBus := NewEventBus(10)
	eventBus.Close()

	handler := func(event Event) {}

	eventBus.RegisterHandler(EventDiscoveryStarted, handler)

	// Should not register handler when closed
	metrics := eventBus.GetMetrics()
	assert.Equal(t, 0, metrics.ActiveHandlers)
}

func TestEventBus_GetMetrics(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	// Initial metrics
	metrics := eventBus.GetMetrics()
	assert.Equal(t, 0, metrics.SubscriptionCount)
	assert.Equal(t, 0, metrics.ActiveHandlers)
	assert.NotNil(t, metrics.EventsPublished)
	assert.NotNil(t, metrics.EventsDelivered)

	// Register handler
	eventBus.RegisterHandler(EventDiscoveryStarted, func(event Event) {})

	// Subscribe
	ctx := context.Background()
	filter := EventFilter{Types: []EventType{EventDiscoveryStarted}}
	sub := eventBus.Subscribe(ctx, filter, 5)
	require.NotNil(t, sub)

	// Publish event
	event := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
	}

	eventBus.Publish(event)

	// Check updated metrics
	metrics = eventBus.GetMetrics()
	assert.Equal(t, 1, metrics.SubscriptionCount)
	assert.Equal(t, 1, metrics.ActiveHandlers)
	assert.Equal(t, int64(1), metrics.EventsPublished[EventDiscoveryStarted])
	assert.Equal(t, int64(1), metrics.EventsDelivered[EventDiscoveryStarted])
}

func TestEventBus_GetBuffer(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	// Empty buffer
	buffer := eventBus.GetBuffer()
	assert.Len(t, buffer, 0)

	// Add events
	event1 := Event{Type: EventDiscoveryStarted, Source: "test"}
	event2 := Event{Type: EventDiscoveryCompleted, Source: "test"}

	eventBus.Publish(event1)
	eventBus.Publish(event2)

	// Get buffer
	buffer = eventBus.GetBuffer()
	assert.Len(t, buffer, 2)
	assert.Equal(t, EventDiscoveryStarted, buffer[0].Type)
	assert.Equal(t, EventDiscoveryCompleted, buffer[1].Type)
}

func TestEventBus_Close(t *testing.T) {
	eventBus := NewEventBus(10)

	// Add subscription
	ctx := context.Background()
	filter := EventFilter{Types: []EventType{EventDiscoveryStarted}}
	sub := eventBus.Subscribe(ctx, filter, 5)
	require.NotNil(t, sub)

	// Register handler
	eventBus.RegisterHandler(EventDiscoveryStarted, func(event Event) {})

	// Close
	eventBus.Close()

	// Verify closed state
	assert.True(t, eventBus.closed)

	// Verify subscriptions were cleaned up
	metrics := eventBus.GetMetrics()
	// Note: Close() doesn't reset metrics, just cleans up subscriptions
	assert.NotNil(t, metrics)

	// Verify buffer was cleared
	buffer := eventBus.GetBuffer()
	assert.Len(t, buffer, 0)
}

func TestEventBus_Close_AlreadyClosed(t *testing.T) {
	eventBus := NewEventBus(10)
	eventBus.Close()

	// Should not panic when closing already closed bus
	assert.NotPanics(t, func() {
		eventBus.Close()
	})
}

func TestEventBus_matchesFilter(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	event := Event{
		Type:      EventDiscoveryStarted,
		Source:    "test-source",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Metadata: map[string]string{
			"env": "test",
		},
	}

	tests := []struct {
		name     string
		filter   EventFilter
		expected bool
	}{
		{
			name:     "no filter",
			filter:   EventFilter{},
			expected: true,
		},
		{
			name: "matching type",
			filter: EventFilter{
				Types: []EventType{EventDiscoveryStarted},
			},
			expected: true,
		},
		{
			name: "non-matching type",
			filter: EventFilter{
				Types: []EventType{EventDiscoveryCompleted},
			},
			expected: false,
		},
		{
			name: "matching source",
			filter: EventFilter{
				Sources: []string{"test-source"},
			},
			expected: true,
		},
		{
			name: "non-matching source",
			filter: EventFilter{
				Sources: []string{"other-source"},
			},
			expected: false,
		},
		{
			name: "matching time range",
			filter: EventFilter{
				MinTime: &[]time.Time{time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC)}[0],
				MaxTime: &[]time.Time{time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)}[0],
			},
			expected: true,
		},
		{
			name: "before min time",
			filter: EventFilter{
				MinTime: &[]time.Time{time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)}[0],
			},
			expected: false,
		},
		{
			name: "after max time",
			filter: EventFilter{
				MaxTime: &[]time.Time{time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC)}[0],
			},
			expected: false,
		},
		{
			name: "matching metadata",
			filter: EventFilter{
				Metadata: map[string]string{"env": "test"},
			},
			expected: true,
		},
		{
			name: "non-matching metadata",
			filter: EventFilter{
				Metadata: map[string]string{"env": "prod"},
			},
			expected: false,
		},
		{
			name: "multiple criteria all matching",
			filter: EventFilter{
				Types:    []EventType{EventDiscoveryStarted},
				Sources:  []string{"test-source"},
				Metadata: map[string]string{"env": "test"},
			},
			expected: true,
		},
		{
			name: "multiple criteria one non-matching",
			filter: EventFilter{
				Types:    []EventType{EventDiscoveryStarted},
				Sources:  []string{"test-source"},
				Metadata: map[string]string{"env": "prod"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eventBus.matchesFilter(event, tt.filter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEventBus_EventDelivery(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	// Subscribe to events
	ctx := context.Background()
	filter := EventFilter{Types: []EventType{EventDiscoveryStarted}}
	sub := eventBus.Subscribe(ctx, filter, 5)
	require.NotNil(t, sub)

	// Publish event
	event := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
		Data:   map[string]interface{}{"message": "Test event"},
	}

	err := eventBus.Publish(event)
	assert.NoError(t, err)

	// Receive event
	select {
	case receivedEvent := <-sub.Channel:
		assert.Equal(t, EventDiscoveryStarted, receivedEvent.Type)
		assert.Equal(t, "test-source", receivedEvent.Source)
		assert.Equal(t, "Test event", receivedEvent.Data["message"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive event")
	}
}

func TestEventBus_HandlerExecution(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	// Register handler
	handlerCalled := false
	var receivedEvent Event
	handler := func(event Event) {
		handlerCalled = true
		receivedEvent = event
	}

	eventBus.RegisterHandler(EventDiscoveryStarted, handler)

	// Publish event
	event := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
		Data:   map[string]interface{}{"message": "Test event"},
	}

	eventBus.Publish(event)

	// Wait for handler
	time.Sleep(10 * time.Millisecond)
	assert.True(t, handlerCalled)
	assert.Equal(t, EventDiscoveryStarted, receivedEvent.Type)
	assert.Equal(t, "test-source", receivedEvent.Source)
	assert.Equal(t, "Test event", receivedEvent.Data["message"])
}

func TestEventBus_HandlerPanic(t *testing.T) {
	eventBus := NewEventBus(10)
	defer eventBus.Close()

	// Register handler that panics
	handler := func(event Event) {
		panic("handler panic")
	}

	eventBus.RegisterHandler(EventDiscoveryStarted, handler)

	// Publish event - should not cause test to fail
	event := Event{
		Type:   EventDiscoveryStarted,
		Source: "test-source",
	}

	assert.NotPanics(t, func() {
		eventBus.Publish(event)
		time.Sleep(10 * time.Millisecond)
	})
}

func TestEventBus_BufferOverflow(t *testing.T) {
	eventBus := NewEventBus(3) // Small buffer
	defer eventBus.Close()

	// Publish more events than buffer size
	for i := 0; i < 5; i++ {
		event := Event{
			Type:   EventDiscoveryStarted,
			Source: "test-source",
			Data:   map[string]interface{}{"index": i},
		}
		eventBus.Publish(event)
	}

	// Buffer should only contain last 3 events
	buffer := eventBus.GetBuffer()
	assert.Len(t, buffer, 3)
	assert.Equal(t, 2, buffer[0].Data["index"]) // First event should be index 2
	assert.Equal(t, 3, buffer[1].Data["index"]) // Second event should be index 3
	assert.Equal(t, 4, buffer[2].Data["index"]) // Third event should be index 4
}

func TestEventBus_ConcurrentAccess(t *testing.T) {
	eventBus := NewEventBus(100)
	defer eventBus.Close()

	// Test concurrent publishing
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			event := Event{
				Type:   EventDiscoveryStarted,
				Source: "test-source",
				Data:   map[string]interface{}{"index": i},
			}
			eventBus.Publish(event)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all events were published
	metrics := eventBus.GetMetrics()
	assert.Equal(t, int64(10), metrics.EventsPublished[EventDiscoveryStarted])
}

func TestEventType_Constants(t *testing.T) {
	// Discovery events
	assert.Equal(t, string(EventDiscoveryStarted), "discovery.started")
	assert.Equal(t, string(EventDiscoveryProgress), "discovery.progress")
	assert.Equal(t, string(EventDiscoveryCompleted), "discovery.completed")
	assert.Equal(t, string(EventDiscoveryFailed), "discovery.failed")
	assert.Equal(t, string(EventResourceDiscovered), "discovery.resource")

	// Drift detection events
	assert.Equal(t, string(EventDriftDetectionStarted), "drift.started")
	assert.Equal(t, string(EventDriftDetectionProgress), "drift.progress")
	assert.Equal(t, string(EventDriftDetectionCompleted), "drift.completed")
	assert.Equal(t, string(EventDriftDetected), "drift.detected")
	assert.Equal(t, string(EventDriftResolved), "drift.resolved")

	// Remediation events
	assert.Equal(t, string(EventRemediationStarted), "remediation.started")
	assert.Equal(t, string(EventRemediationProgress), "remediation.progress")
	assert.Equal(t, string(EventRemediationCompleted), "remediation.completed")
	assert.Equal(t, string(EventRemediationFailed), "remediation.failed")
	assert.Equal(t, string(EventResourceDeleted), "remediation.deleted")
	assert.Equal(t, string(EventResourceImported), "remediation.imported")

	// State management events
	assert.Equal(t, string(EventStateBackupCreated), "state.backup.created")
	assert.Equal(t, string(EventStatePulled), "state.pulled")
	assert.Equal(t, string(EventStatePushed), "state.pushed")
	assert.Equal(t, string(EventStateValidated), "state.validated")
	assert.Equal(t, string(EventStateModified), "state.modified")

	// System events
	assert.Equal(t, string(EventCacheCleared), "cache.cleared")
	assert.Equal(t, string(EventCacheRefreshed), "cache.refreshed")
	assert.Equal(t, string(EventHealthCheck), "health.check")
	assert.Equal(t, string(EventConfigChanged), "config.changed")
	assert.Equal(t, string(EventAuditLog), "audit.log")

	// Job events
	assert.Equal(t, string(EventJobQueued), "job.queued")
	assert.Equal(t, string(EventJobStarted), "job.started")
	assert.Equal(t, string(EventJobCompleted), "job.completed")
	assert.Equal(t, string(EventJobFailed), "job.failed")
	assert.Equal(t, string(EventJobRetrying), "job.retrying")

	// WebSocket events
	assert.Equal(t, string(EventWSClientConnected), "ws.connected")
	assert.Equal(t, string(EventWSClientDisconnected), "ws.disconnected")
	assert.Equal(t, string(EventWSMessage), "ws.message")
}

func TestEvent_Struct(t *testing.T) {
	event := Event{
		ID:        "event-123",
		Type:      EventDiscoveryStarted,
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Source:    "test-source",
		Data: map[string]interface{}{
			"message": "Test event",
		},
		Metadata: map[string]string{
			"env": "test",
		},
		UserID:    "user-123",
		SessionID: "session-456",
	}

	assert.Equal(t, "event-123", event.ID)
	assert.Equal(t, EventDiscoveryStarted, event.Type)
	assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), event.Timestamp)
	assert.Equal(t, "test-source", event.Source)
	assert.Equal(t, "Test event", event.Data["message"])
	assert.Equal(t, "test", event.Metadata["env"])
	assert.Equal(t, "user-123", event.UserID)
	assert.Equal(t, "session-456", event.SessionID)
}

func TestSubscription_Struct(t *testing.T) {
	ctx := context.Background()
	filter := EventFilter{Types: []EventType{EventDiscoveryStarted}}
	channel := make(chan Event, 5)
	cancel := func() {}

	sub := &Subscription{
		ID:      "sub-123",
		Filter:  filter,
		Channel: channel,
		cancel:  cancel,
		ctx:     ctx,
	}

	assert.Equal(t, "sub-123", sub.ID)
	assert.Equal(t, filter, sub.Filter)
	assert.Equal(t, channel, sub.Channel)
	// Can't compare functions directly, just verify cancel is not nil
	assert.NotNil(t, sub.cancel)
	assert.Equal(t, ctx, sub.ctx)
}

func TestEventFilter_Struct(t *testing.T) {
	minTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	maxTime := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

	filter := EventFilter{
		Types:    []EventType{EventDiscoveryStarted, EventDiscoveryCompleted},
		Sources:  []string{"source1", "source2"},
		MinTime:  &minTime,
		MaxTime:  &maxTime,
		Metadata: map[string]string{"env": "test"},
	}

	assert.Len(t, filter.Types, 2)
	assert.Contains(t, filter.Types, EventDiscoveryStarted)
	assert.Contains(t, filter.Types, EventDiscoveryCompleted)
	assert.Len(t, filter.Sources, 2)
	assert.Contains(t, filter.Sources, "source1")
	assert.Contains(t, filter.Sources, "source2")
	assert.Equal(t, minTime, *filter.MinTime)
	assert.Equal(t, maxTime, *filter.MaxTime)
	assert.Equal(t, "test", filter.Metadata["env"])
}

func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := generateEventID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should be unique
	assert.Contains(t, id1, "-") // Should contain timestamp separator
}

func TestGenerateSubscriptionID(t *testing.T) {
	id1 := generateSubscriptionID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := generateSubscriptionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should be unique
	assert.Contains(t, id1, "sub-")
	assert.Contains(t, id2, "sub-")
}
