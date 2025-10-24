package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionService_Integration(t *testing.T) {
	// Skip if no credentials are available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create event bus
	eventBus := events.NewEventBus(100)
	defer eventBus.Close()

	// Create provider factory
	factory := providers.NewProviderFactory()

	// Create connection service
	connectionService := providers.NewConnectionService(factory, eventBus, 30*time.Second)

	t.Run("TestProviderConnection_AWS", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		result, err := connectionService.TestProviderConnection(context.Background(), "aws", "us-east-1")

		// Should not fail even if credentials are invalid
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "aws", result.Provider)
		assert.Equal(t, "us-east-1", result.Region)
		assert.Greater(t, result.Latency, time.Duration(0))
	})

	t.Run("TestProviderConnection_Azure", func(t *testing.T) {
		// Skip if Azure credentials are not available
		if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
			t.Skip("Azure credentials not available")
		}

		result, err := connectionService.TestProviderConnection(context.Background(), "azure", "eastus")

		// Should not fail even if credentials are invalid
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "azure", result.Provider)
		assert.Equal(t, "eastus", result.Region)
		assert.Greater(t, result.Latency, time.Duration(0))
	})

	t.Run("TestProviderConnection_GCP", func(t *testing.T) {
		// Skip if GCP credentials are not available
		if os.Getenv("GCP_PROJECT_ID") == "" {
			t.Skip("GCP credentials not available")
		}

		result, err := connectionService.TestProviderConnection(context.Background(), "gcp", "us-central1")

		// Should not fail even if credentials are invalid
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "gcp", result.Provider)
		assert.Equal(t, "us-central1", result.Region)
		assert.Greater(t, result.Latency, time.Duration(0))
	})

	t.Run("TestProviderConnection_DigitalOcean", func(t *testing.T) {
		// Skip if DigitalOcean credentials are not available
		if os.Getenv("DIGITALOCEAN_TOKEN") == "" {
			t.Skip("DigitalOcean credentials not available")
		}

		result, err := connectionService.TestProviderConnection(context.Background(), "digitalocean", "nyc1")

		// Should not fail even if credentials are invalid
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "digitalocean", result.Provider)
		assert.Equal(t, "nyc1", result.Region)
		assert.Greater(t, result.Latency, time.Duration(0))
	})

	t.Run("TestAllProviders", func(t *testing.T) {
		results, err := connectionService.TestAllProviders(context.Background(), "us-east-1")

		require.NoError(t, err)
		require.NotNil(t, results)

		// Should have results for all providers
		expectedProviders := []string{"aws", "azure", "gcp", "digitalocean"}
		for _, provider := range expectedProviders {
			assert.Contains(t, results, provider)
			assert.NotNil(t, results[provider])
		}
	})

	t.Run("TestProviderAllRegions_AWS", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		results, err := connectionService.TestProviderAllRegions(context.Background(), "aws")

		require.NoError(t, err)
		require.NotNil(t, results)
		assert.Greater(t, len(results), 0)

		// All results should be for AWS
		for _, result := range results {
			assert.Equal(t, "aws", result.Provider)
			assert.NotEmpty(t, result.Region)
		}
	})

	t.Run("TestProviderAllServices_AWS", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		results, err := connectionService.TestProviderAllServices(context.Background(), "aws", "us-east-1")

		require.NoError(t, err)
		require.NotNil(t, results)
		assert.Greater(t, len(results), 0)

		// All results should be for AWS
		for _, result := range results {
			assert.Equal(t, "aws", result.Provider)
			assert.Equal(t, "us-east-1", result.Region)
			assert.NotEmpty(t, result.Service)
		}
	})

	t.Run("GetConnectionResults", func(t *testing.T) {
		// Test getting results for a specific provider
		results := connectionService.GetConnectionResults("aws")
		assert.NotNil(t, results)

		// Test getting all results
		allResults := connectionService.GetAllConnectionResults()
		assert.NotNil(t, allResults)
	})

	t.Run("GetConnectionSummary", func(t *testing.T) {
		summary := connectionService.GetConnectionSummary()
		assert.NotNil(t, summary)
	})

	t.Run("RunHealthCheck", func(t *testing.T) {
		result, err := connectionService.RunHealthCheck(context.Background(), "us-east-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "us-east-1", result.Region)
		assert.Greater(t, result.Duration, time.Duration(0))
		assert.NotNil(t, result.Results)
		assert.NotNil(t, result.Summary)
	})

	t.Run("ClearConnectionResults", func(t *testing.T) {
		// Clear results for a specific provider
		connectionService.ClearConnectionResults("aws")

		// Clear all results
		connectionService.ClearConnectionResults("")
	})
}

func TestConnectionService_EventPublishing(t *testing.T) {
	// Create event bus
	eventBus := events.NewEventBus(100)
	defer eventBus.Close()

	// Create provider factory
	factory := providers.NewProviderFactory()

	// Create connection service
	connectionService := providers.NewConnectionService(factory, eventBus, 30*time.Second)

	// Subscribe to connection events
	eventFilter := events.EventFilter{
		Types: []events.EventType{
			events.EventType("connection.test.completed"),
			events.EventType("connection.all.providers.tested"),
			events.EventType("connection.health_check.completed"),
		},
	}

	subscription := eventBus.Subscribe(context.Background(), eventFilter, 10)
	defer eventBus.Unsubscribe(subscription)

	// Test provider connection to trigger event
	result, err := connectionService.TestProviderConnection(context.Background(), "aws", "us-east-1")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Wait for event
	select {
	case event := <-subscription.Channel:
		assert.Equal(t, "connection.test.completed", string(event.Type))
		assert.Equal(t, "connection_service", event.Source)
		assert.Contains(t, event.Data, "provider")
		assert.Contains(t, event.Data, "region")
		assert.Contains(t, event.Data, "success")
	case <-time.After(5 * time.Second):
		t.Fatal("Expected connection test event not received")
	}

	// Test all providers to trigger summary event
	results, err := connectionService.TestAllProviders(context.Background(), "us-east-1")
	require.NoError(t, err)
	require.NotNil(t, results)

	// Wait for summary event
	select {
	case event := <-subscription.Channel:
		assert.Equal(t, "connection.all.providers.tested", string(event.Type))
		assert.Equal(t, "connection_service", event.Source)
		assert.Contains(t, event.Data, "region")
		assert.Contains(t, event.Data, "summary")
		assert.Contains(t, event.Data, "results")
	case <-time.After(5 * time.Second):
		t.Fatal("Expected all providers test event not received")
	}

	// Test health check to trigger health check event
	healthResult, err := connectionService.RunHealthCheck(context.Background(), "us-east-1")
	require.NoError(t, err)
	require.NotNil(t, healthResult)

	// Wait for health check event
	select {
	case event := <-subscription.Channel:
		assert.Equal(t, "connection.health_check.completed", string(event.Type))
		assert.Equal(t, "connection_service", event.Source)
		assert.Contains(t, event.Data, "region")
		assert.Contains(t, event.Data, "duration")
		assert.Contains(t, event.Data, "summary")
		assert.Contains(t, event.Data, "results")
	case <-time.After(5 * time.Second):
		t.Fatal("Expected health check event not received")
	}
}
