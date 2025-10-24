package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/integrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookSystem_Integration(t *testing.T) {
	// Create integration manager
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create a test server to receive webhooks
	webhookReceived := false
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookReceived = true

		// Verify request method and headers
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "DriftMgr-Webhook/1.0", r.Header.Get("User-Agent"))

		// Read and parse payload
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		receivedPayload = payload

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Create webhook integration
	integration := &integrations.Integration{
		Name:     "Test Webhook Integration",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": server.URL,
			"headers": map[string]interface{}{
				"X-Custom-Header": "test-value",
			},
			"auth": map[string]interface{}{
				"type":      "hmac",
				"secret":    "test-secret",
				"algorithm": "sha256",
			},
		},
		Enabled: true,
	}

	// Create integration
	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)
	assert.NotEmpty(t, integration.ID)

	// Create test event
	event := &integrations.IntegrationEvent{
		ID:            "test-event-1",
		IntegrationID: integration.ID,
		Type:          "alert",
		Data: map[string]interface{}{
			"message":  "Test webhook integration",
			"severity": "high",
			"provider": "aws",
			"region":   "us-east-1",
		},
		Status:    "pending",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	// Send event
	err = manager.SendEvent(ctx, integration.ID, event)
	require.NoError(t, err)

	// Wait for webhook delivery
	time.Sleep(2 * time.Second)

	// Verify webhook was received
	assert.True(t, webhookReceived, "Webhook should have been delivered")
	assert.NotNil(t, receivedPayload)

	// Verify payload structure
	assert.Equal(t, "test-event-1", receivedPayload["id"])
	assert.Equal(t, integration.ID, receivedPayload["integration_id"])
	assert.Equal(t, "alert", receivedPayload["type"])
	assert.Equal(t, "pending", receivedPayload["status"])
	assert.Contains(t, receivedPayload, "timestamp")
	assert.Contains(t, receivedPayload, "data")

	// Verify data section
	data, ok := receivedPayload["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test webhook integration", data["message"])
	assert.Equal(t, "high", data["severity"])
	assert.Equal(t, "aws", data["provider"])
	assert.Equal(t, "us-east-1", data["region"])

	// Verify integration status was updated
	updatedIntegration, err := manager.GetIntegration(ctx, integration.ID)
	require.NoError(t, err)
	assert.False(t, updatedIntegration.LastSync.IsZero())
}

func TestWebhookSystem_Authentication(t *testing.T) {
	// Create integration manager
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create a test server that verifies HMAC signature
	hmacVerified := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HMAC signature header
		signature := r.Header.Get("X-Hub-Signature-256")
		assert.NotEmpty(t, signature)
		assert.Contains(t, signature, "sha256=")
		hmacVerified = true

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Create webhook integration with HMAC authentication
	integration := &integrations.Integration{
		Name:     "Test HMAC Webhook",
		Type:     "webhook",
		Provider: "github",
		Config: map[string]interface{}{
			"url": server.URL,
			"auth": map[string]interface{}{
				"type":             "hmac",
				"secret":           "test-secret-key",
				"algorithm":        "sha256",
				"signature_header": "X-Hub-Signature-256",
			},
		},
		Enabled: true,
	}

	// Create integration
	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	// Create test event
	event := &integrations.IntegrationEvent{
		ID:            "test-hmac-event",
		IntegrationID: integration.ID,
		Type:          "webhook",
		Data: map[string]interface{}{
			"message": "Test HMAC authentication",
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	// Send event
	err = manager.SendEvent(ctx, integration.ID, event)
	require.NoError(t, err)

	// Wait for webhook delivery
	time.Sleep(2 * time.Second)

	// Verify HMAC authentication was used
	assert.True(t, hmacVerified, "HMAC signature should have been verified")
}

func TestWebhookSystem_RetryLogic(t *testing.T) {
	// Create integration manager
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	attemptCount := 0
	// Create a test server that fails first two attempts, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "service unavailable"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success"}`))
		}
	}))
	defer server.Close()

	// Create webhook integration
	integration := &integrations.Integration{
		Name:     "Test Retry Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": server.URL,
		},
		Enabled: true,
	}

	// Create integration
	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	// Create test event
	event := &integrations.IntegrationEvent{
		ID:            "test-retry-event",
		IntegrationID: integration.ID,
		Type:          "alert",
		Data: map[string]interface{}{
			"message": "Test retry logic",
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	// Send event
	err = manager.SendEvent(ctx, integration.ID, event)
	require.NoError(t, err)

	// Wait for retries to complete
	time.Sleep(5 * time.Second)

	// Verify webhook eventually succeeded after retries
	assert.Equal(t, 3, attemptCount, "Webhook should have made 3 attempts total")
}

func TestWebhookSystem_ErrorHandling(t *testing.T) {
	// Create integration manager
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Test sending event to non-existent integration
	event := &integrations.IntegrationEvent{
		ID:            "test-error-event",
		IntegrationID: "non-existent-id",
		Type:          "alert",
		Data: map[string]interface{}{
			"message": "Test error handling",
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	// Send event to non-existent integration
	err := manager.SendEvent(ctx, "non-existent-id", event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integration non-existent-id not found")

	// Test sending event to disabled integration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	integration := &integrations.Integration{
		Name:     "Disabled Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": server.URL,
		},
		Enabled: false, // Disabled
	}

	err = manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	err = manager.SendEvent(ctx, integration.ID, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integration "+integration.ID+" is disabled")
}

func TestWebhookSystem_Configuration(t *testing.T) {
	// Create integration manager
	manager := integrations.NewIntegrationManager()

	// Get current configuration
	config := manager.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 100, config.MaxIntegrations)
	assert.Equal(t, 5*time.Minute, config.SyncInterval)
	assert.Equal(t, 3, config.RetryAttempts)

	// Update configuration
	newConfig := &integrations.IntegrationConfig{
		MaxIntegrations:     200,
		SyncInterval:        10 * time.Minute,
		RetryAttempts:       5,
		RetryDelay:          60 * time.Second,
		Timeout:             60 * time.Second,
		AutoSync:            false,
		NotificationEnabled: false,
	}
	manager.SetConfig(newConfig)

	// Verify configuration was updated
	updatedConfig := manager.GetConfig()
	assert.Equal(t, 200, updatedConfig.MaxIntegrations)
	assert.Equal(t, 10*time.Minute, updatedConfig.SyncInterval)
	assert.Equal(t, 5, updatedConfig.RetryAttempts)
	assert.Equal(t, 60*time.Second, updatedConfig.RetryDelay)
	assert.Equal(t, 60*time.Second, updatedConfig.Timeout)
	assert.False(t, updatedConfig.AutoSync)
	assert.False(t, updatedConfig.NotificationEnabled)
}


