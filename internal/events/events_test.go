package events

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		event    EventType
		expected string
	}{
		// Discovery events
		{"discovery started", EventDiscoveryStarted, "discovery.started"},
		{"discovery progress", EventDiscoveryProgress, "discovery.progress"},
		{"discovery completed", EventDiscoveryCompleted, "discovery.completed"},
		{"discovery failed", EventDiscoveryFailed, "discovery.failed"},
		{"resource found", EventResourceFound, "resource.found"},

		// Test aliases
		{"discovery started alias", DiscoveryStarted, "discovery.started"},
		{"discovery progress alias", DiscoveryProgress, "discovery.progress"},
		{"discovery completed alias", DiscoveryCompleted, "discovery.completed"},
		{"discovery failed alias", DiscoveryFailed, "discovery.failed"},

		// Drift events
		{"drift detected", EventDriftDetected, "drift.detected"},
		{"drift analyzed", EventDriftAnalyzed, "drift.analyzed"},
		{"drift remediated", EventDriftRemediated, "drift.remediated"},
		{"drift detection started", DriftDetectionStarted, "drift.detection.started"},
		{"drift detection completed", DriftDetectionCompleted, "drift.detection.completed"},
		{"drift detection failed", DriftDetectionFailed, "drift.detection.failed"},

		// Remediation events
		{"remediation started", EventRemediationStarted, "remediation.started"},
		{"remediation progress", EventRemediationProgress, "remediation.progress"},
		{"remediation completed", EventRemediationCompleted, "remediation.completed"},
		{"remediation failed", EventRemediationFailed, "remediation.failed"},

		// Test remediation aliases
		{"remediation started alias", RemediationStarted, "remediation.started"},
		{"remediation completed alias", RemediationCompleted, "remediation.completed"},
		{"remediation failed alias", RemediationFailed, "remediation.failed"},

		// System events
		{"system startup", EventSystemStartup, "system.startup"},
		{"system shutdown", EventSystemShutdown, "system.shutdown"},
		{"system error", EventSystemError, "system.error"},
		{"system warning", EventSystemWarning, "system.warning"},
		{"system info", EventSystemInfo, "system.info"},

		// State events
		{"state changed", EventStateChanged, "state.changed"},
		{"state backup", EventStateBackup, "state.backup"},
		{"state restored", EventStateRestored, "state.restored"},
		{"state locked", EventStateLocked, "state.locked"},
		{"state unlocked", EventStateUnlocked, "state.unlocked"},

		// Job events
		{"job queued", EventJobQueued, "job.queued"},
		{"job started", EventJobStarted, "job.started"},
		{"job completed", EventJobCompleted, "job.completed"},
		{"job failed", EventJobFailed, "job.failed"},

		// Resource events
		{"resource created", EventResourceCreated, "resource.created"},
		{"resource updated", EventResourceUpdated, "resource.updated"},
		{"resource deleted", EventResourceDeleted, "resource.deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, EventType(tt.expected), tt.event)
			assert.Equal(t, tt.expected, string(tt.event))
		})
	}
}

func TestEvent(t *testing.T) {
	event := Event{
		ID:        "event-123",
		Type:      EventDiscoveryStarted,
		Timestamp: time.Now(),
		Source:    "discovery-engine",
		Data: map[string]interface{}{
			"provider":  "aws",
			"region":    "us-east-1",
			"resources": 100,
		},
	}

	assert.Equal(t, "event-123", event.ID)
	assert.Equal(t, EventDiscoveryStarted, event.Type)
	assert.NotZero(t, event.Timestamp)
	assert.Equal(t, "discovery-engine", event.Source)
	assert.NotNil(t, event.Data)
	assert.Equal(t, "aws", event.Data["provider"])
	assert.Equal(t, "us-east-1", event.Data["region"])
	assert.Equal(t, 100, event.Data["resources"])
}

func TestEventHandler(t *testing.T) {
	handled := false
	var receivedEvent Event

	handler := EventHandler(func(event Event) {
		handled = true
		receivedEvent = event
	})

	event := Event{
		ID:        "test-123",
		Type:      EventSystemInfo,
		Timestamp: time.Now(),
		Source:    "test",
	}

	handler(event)

	assert.True(t, handled)
	assert.Equal(t, "test-123", receivedEvent.ID)
	assert.Equal(t, EventSystemInfo, receivedEvent.Type)
}

func TestSubscription(t *testing.T) {
	handler := EventHandler(func(event Event) {})

	sub := Subscription{
		ID:      "sub-123",
		Handler: handler,
		Types: []EventType{
			EventDiscoveryStarted,
			EventDiscoveryCompleted,
			EventDriftDetected,
		},
	}

	assert.Equal(t, "sub-123", sub.ID)
	assert.NotNil(t, sub.Handler)
	assert.Len(t, sub.Types, 3)
	assert.Contains(t, sub.Types, EventDiscoveryStarted)
	assert.Contains(t, sub.Types, EventDiscoveryCompleted)
	assert.Contains(t, sub.Types, EventDriftDetected)
}

func TestEventAliases(t *testing.T) {
	// Test that aliases have the same value as the main event types
	assert.Equal(t, EventDiscoveryStarted, DiscoveryStarted)
	assert.Equal(t, EventDiscoveryProgress, DiscoveryProgress)
	assert.Equal(t, EventDiscoveryCompleted, DiscoveryCompleted)
	assert.Equal(t, EventDiscoveryFailed, DiscoveryFailed)

	assert.Equal(t, EventRemediationStarted, RemediationStarted)
	assert.Equal(t, EventRemediationCompleted, RemediationCompleted)
	assert.Equal(t, EventRemediationFailed, RemediationFailed)

	assert.Equal(t, EventJobStarted, JobStarted)
	assert.Equal(t, EventJobCompleted, JobCompleted)
	assert.Equal(t, EventJobFailed, JobFailed)

	assert.Equal(t, EventResourceCreated, ResourceCreated)
	assert.Equal(t, EventResourceUpdated, ResourceUpdated)
	assert.Equal(t, EventResourceDeleted, ResourceDeleted)
}

func TestEventCreation(t *testing.T) {
	now := time.Now()
	event := Event{
		ID:        "evt-001",
		Type:      EventSystemStartup,
		Timestamp: now,
		Source:    "system",
		Data: map[string]interface{}{
			"version": "1.0.0",
			"pid":     12345,
		},
	}

	assert.Equal(t, "evt-001", event.ID)
	assert.Equal(t, EventSystemStartup, event.Type)
	assert.Equal(t, now, event.Timestamp)
	assert.Equal(t, "system", event.Source)
	assert.Equal(t, "1.0.0", event.Data["version"])
	assert.Equal(t, 12345, event.Data["pid"])
}

func TestMultipleEventTypes(t *testing.T) {
	// Test that different event types can be created
	events := []Event{
		{ID: "1", Type: EventDiscoveryStarted, Source: "discovery"},
		{ID: "2", Type: EventDriftDetected, Source: "drift"},
		{ID: "3", Type: EventRemediationStarted, Source: "remediation"},
		{ID: "4", Type: EventSystemError, Source: "system"},
		{ID: "5", Type: EventStateChanged, Source: "state"},
		{ID: "6", Type: EventJobQueued, Source: "job"},
		{ID: "7", Type: EventResourceCreated, Source: "resource"},
	}

	for _, event := range events {
		assert.NotEmpty(t, event.ID)
		assert.NotEmpty(t, event.Type)
		assert.NotEmpty(t, event.Source)
	}
}

func TestEventDataManipulation(t *testing.T) {
	event := Event{
		ID:        "test",
		Type:      EventSystemInfo,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      make(map[string]interface{}),
	}

	// Add data
	event.Data["key1"] = "value1"
	event.Data["key2"] = 42
	event.Data["key3"] = true

	assert.Equal(t, "value1", event.Data["key1"])
	assert.Equal(t, 42, event.Data["key2"])
	assert.Equal(t, true, event.Data["key3"])

	// Update data
	event.Data["key1"] = "updated"
	assert.Equal(t, "updated", event.Data["key1"])

	// Delete data
	delete(event.Data, "key2")
	_, exists := event.Data["key2"]
	assert.False(t, exists)
}

func BenchmarkEventCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Event{
			ID:        fmt.Sprintf("evt-%d", i),
			Type:      EventSystemInfo,
			Timestamp: time.Now(),
			Source:    "benchmark",
			Data: map[string]interface{}{
				"index": i,
			},
		}
	}
}

func BenchmarkEventHandler(b *testing.B) {
	handler := EventHandler(func(event Event) {
		// Simulate some work
		_ = event.ID
	})

	event := Event{
		ID:        "bench",
		Type:      EventSystemInfo,
		Timestamp: time.Now(),
		Source:    "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(event)
	}
}