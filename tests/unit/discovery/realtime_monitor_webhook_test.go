package discovery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealTimeMonitor_WebhookConfiguration(t *testing.T) {
	monitor := discovery.NewRealTimeMonitor()
	defer monitor.Stop()

	// Test adding webhook
	webhookConfig := discovery.WebhookConfig{
		URL:        "https://example.com/webhook",
		Events:     []string{"error", "warning"},
		Headers:    map[string]string{"X-Custom": "value"},
		RetryCount: 3,
		Timeout:    30 * time.Second,
	}

	monitor.AddWebhook(webhookConfig)

	// Test removing webhook
	monitor.RemoveWebhook("https://example.com/webhook")

	// Test adding multiple webhooks
	webhook1 := discovery.WebhookConfig{
		URL:     "https://example.com/webhook1",
		Events:  []string{"error"},
		Headers: map[string]string{"X-Custom1": "value1"},
	}

	webhook2 := discovery.WebhookConfig{
		URL:     "https://example.com/webhook2",
		Events:  []string{"warning"},
		Headers: map[string]string{"X-Custom2": "value2"},
	}

	monitor.AddWebhook(webhook1)
	monitor.AddWebhook(webhook2)

	// Verify webhooks are added
	dashboardData := monitor.GetDashboardData()
	assert.Equal(t, 2, dashboardData["webhooks"])
}

func TestRealTimeMonitor_WebhookTriggering(t *testing.T) {
	monitor := discovery.NewRealTimeMonitor()
	defer monitor.Stop()

	// Create a test server to receive webhooks
	webhookReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookReceived = true
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "DriftMgr-Webhook/1.0", r.Header.Get("User-Agent"))
		assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Add webhook configuration
	webhookConfig := discovery.WebhookConfig{
		URL:        server.URL,
		Events:     []string{"error", "critical"},
		Headers:    map[string]string{"X-Custom-Header": "test-value"},
		RetryCount: 1,
		Timeout:    5 * time.Second,
	}
	monitor.AddWebhook(webhookConfig)

	// Start the monitor
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	require.NoError(t, err)

	// Create an error alert that should trigger the webhook
	monitor.CreateAlert(
		discovery.AlertTypeError,
		discovery.AlertSeverityCritical,
		"Test error alert",
		map[string]interface{}{
			"provider": "aws",
			"region":   "us-east-1",
		},
	)

	// Wait for webhook to be triggered
	time.Sleep(2 * time.Second)

	// Verify webhook was received
	assert.True(t, webhookReceived, "Webhook should have been triggered")
}

func TestRealTimeMonitor_WebhookEventFiltering(t *testing.T) {
	monitor := discovery.NewRealTimeMonitor()
	defer monitor.Stop()

	// Create test servers for different event types
	errorWebhookReceived := false
	warningWebhookReceived := false

	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errorWebhookReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer errorServer.Close()

	warningServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		warningWebhookReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer warningServer.Close()

	// Add webhook configurations for different event types
	errorWebhook := discovery.WebhookConfig{
		URL:        errorServer.URL,
		Events:     []string{"error"},
		RetryCount: 1,
		Timeout:    5 * time.Second,
	}

	warningWebhook := discovery.WebhookConfig{
		URL:        warningServer.URL,
		Events:     []string{"warning"},
		RetryCount: 1,
		Timeout:    5 * time.Second,
	}

	monitor.AddWebhook(errorWebhook)
	monitor.AddWebhook(warningWebhook)

	// Start the monitor
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	require.NoError(t, err)

	// Create error alert - should trigger error webhook only
	monitor.CreateAlert(
		discovery.AlertTypeError,
		discovery.AlertSeverityHigh,
		"Test error alert",
		map[string]interface{}{},
	)

	// Wait for webhook processing
	time.Sleep(2 * time.Second)

	// Verify only error webhook was triggered
	assert.True(t, errorWebhookReceived, "Error webhook should have been triggered")
	assert.False(t, warningWebhookReceived, "Warning webhook should not have been triggered")

	// Reset flags
	errorWebhookReceived = false
	warningWebhookReceived = false

	// Create warning alert - should trigger warning webhook only
	monitor.CreateAlert(
		discovery.AlertTypeWarning,
		discovery.AlertSeverityMedium,
		"Test warning alert",
		map[string]interface{}{},
	)

	// Wait for webhook processing
	time.Sleep(2 * time.Second)

	// Verify only warning webhook was triggered
	assert.False(t, errorWebhookReceived, "Error webhook should not have been triggered")
	assert.True(t, warningWebhookReceived, "Warning webhook should have been triggered")
}

func TestRealTimeMonitor_WebhookRetryLogic(t *testing.T) {
	monitor := discovery.NewRealTimeMonitor()
	defer monitor.Stop()

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

	// Add webhook configuration with retry
	webhookConfig := discovery.WebhookConfig{
		URL:        server.URL,
		Events:     []string{"error"},
		RetryCount: 3,
		Timeout:    5 * time.Second,
	}
	monitor.AddWebhook(webhookConfig)

	// Start the monitor
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	require.NoError(t, err)

	// Create an error alert
	monitor.CreateAlert(
		discovery.AlertTypeError,
		discovery.AlertSeverityHigh,
		"Test error alert",
		map[string]interface{}{},
	)

	// Wait for retries to complete
	time.Sleep(5 * time.Second)

	// Verify webhook eventually succeeded after retries
	assert.Equal(t, 3, attemptCount, "Webhook should have made 3 attempts total")
}

func TestRealTimeMonitor_WebhookPayload(t *testing.T) {
	monitor := discovery.NewRealTimeMonitor()
	defer monitor.Stop()

	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and parse the payload
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		// Parse JSON payload
		err := json.Unmarshal(body, &receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Add webhook configuration
	webhookConfig := discovery.WebhookConfig{
		URL:        server.URL,
		Events:     []string{"error"},
		RetryCount: 1,
		Timeout:    5 * time.Second,
	}
	monitor.AddWebhook(webhookConfig)

	// Start the monitor
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	require.NoError(t, err)

	// Create an error alert
	monitor.CreateAlert(
		discovery.AlertTypeError,
		discovery.AlertSeverityCritical,
		"Test error alert",
		map[string]interface{}{
			"provider": "aws",
			"region":   "us-east-1",
			"resource": "test-resource",
		},
	)

	// Wait for webhook processing
	time.Sleep(2 * time.Second)

	// Verify payload structure
	assert.NotNil(t, receivedPayload)
	assert.Equal(t, "driftmgr", receivedPayload["source"])
	assert.Equal(t, "alert", receivedPayload["type"])
	assert.Contains(t, receivedPayload, "timestamp")
	assert.Contains(t, receivedPayload, "data")

	// Verify data section
	data, ok := receivedPayload["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data, "metrics")
	assert.Contains(t, data, "alerts")
	assert.Contains(t, data, "status")
}

func TestRealTimeMonitor_WebhookWithMultipleAlerts(t *testing.T) {
	monitor := discovery.NewRealTimeMonitor()
	defer monitor.Stop()

	webhookCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCallCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Add webhook configuration
	webhookConfig := discovery.WebhookConfig{
		URL:        server.URL,
		Events:     []string{"error", "warning"},
		RetryCount: 1,
		Timeout:    5 * time.Second,
	}
	monitor.AddWebhook(webhookConfig)

	// Start the monitor
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	require.NoError(t, err)

	// Create multiple alerts
	monitor.CreateAlert(
		discovery.AlertTypeError,
		discovery.AlertSeverityHigh,
		"Error alert 1",
		map[string]interface{}{},
	)

	monitor.CreateAlert(
		discovery.AlertTypeWarning,
		discovery.AlertSeverityMedium,
		"Warning alert 1",
		map[string]interface{}{},
	)

	monitor.CreateAlert(
		discovery.AlertTypeError,
		discovery.AlertSeverityCritical,
		"Error alert 2",
		map[string]interface{}{},
	)

	// Wait for webhook processing
	time.Sleep(3 * time.Second)

	// Verify webhook was called (should be called once per processing cycle, not per alert)
	assert.GreaterOrEqual(t, webhookCallCount, 1, "Webhook should have been called at least once")
}
