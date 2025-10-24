package events

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	sharedEvents "github.com/catherinevee/driftmgr/internal/shared/events"
	"github.com/stretchr/testify/assert"
)

func TestEventAggregator_NewEventAggregator(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	config := &events.AggregatorConfig{
		Enabled:           true,
		AggregationWindow: 5 * time.Minute,
		MaxEvents:         1000,
		CleanupInterval:   1 * time.Minute,
		BatchSize:         100,
		FlushInterval:     30 * time.Second,
	}

	aggregator := events.NewEventAggregator(eventBus, config)

	if aggregator == nil {
		t.Fatal("Expected aggregator to be created")
	}
}

func TestEventAggregator_StartStop(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	aggregator := events.NewEventAggregator(eventBus, nil)

	ctx := context.Background()

	// Test start
	err := aggregator.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test stop
	aggregator.Stop()
}

func TestEventAggregator_GetAggregations(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	aggregator := events.NewEventAggregator(eventBus, nil)

	aggregations := aggregator.GetAggregations()
	if aggregations == nil {
		t.Error("Expected aggregations map to be initialized")
	}
}

func TestEventAggregator_GetAggregation(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	aggregator := events.NewEventAggregator(eventBus, nil)

	// Test getting non-existent aggregation
	_, exists := aggregator.GetAggregation("non-existent")
	if exists {
		t.Error("Expected aggregation to not exist")
	}
}

func TestEventAggregator_GetMetrics(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	aggregator := events.NewEventAggregator(eventBus, nil)

	metrics := aggregator.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be returned")
	}
}

func TestEventAggregator_Integration(t *testing.T) {
	eventBus := sharedEvents.NewEventBus(100)
	config := &events.AggregatorConfig{
		Enabled:           true,
		AggregationWindow: 1 * time.Minute,
		MaxEvents:         100,
		CleanupInterval:   30 * time.Second,
		BatchSize:         10,
		FlushInterval:     10 * time.Second,
	}

	aggregator := events.NewEventAggregator(eventBus, config)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the aggregator
	err := aggregator.Start(ctx)
	assert.NoError(t, err)

	// Publish some test events
	testEvent := sharedEvents.Event{
		Type:      sharedEvents.EventDiscoveryStarted,
		Timestamp: time.Now(),
		Source:    "test-source",
		Data: map[string]interface{}{
			"provider": "aws",
			"region":   "us-east-1",
		},
	}

	err = eventBus.Publish(testEvent)
	assert.NoError(t, err)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Check aggregations
	aggregations := aggregator.GetAggregations()
	assert.NotNil(t, aggregations)

	// Stop the aggregator
	aggregator.Stop()
}
