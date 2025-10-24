package integrations_test

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

func TestWebhookDeliveryService_DeliverWebhook(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, "POST", r.Method)

		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "DriftMgr-Webhook/1.0", r.Header.Get("User-Agent"))

		// Read and verify payload
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "test", payload["message"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)
	headers := map[string]string{
		"X-Custom-Header": "test-value",
	}

	// Test successful delivery
	delivery, err := service.DeliverWebhook(ctx, server.URL, payload, headers, nil)
	require.NoError(t, err)
	assert.NotNil(t, delivery)
	assert.Equal(t, integrations.DeliveryStatusDelivered, delivery.Status)
	assert.Equal(t, server.URL, delivery.URL)
	assert.Equal(t, payload, delivery.Payload)
	assert.Equal(t, 200, delivery.ResponseCode)
}

func TestWebhookDeliveryService_DeliverWebhookWithAuth(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	// Create a test server that verifies HMAC signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HMAC signature header
		signature := r.Header.Get("X-Hub-Signature-256")
		assert.NotEmpty(t, signature)
		assert.Contains(t, signature, "sha256=")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)
	auth := &integrations.WebhookAuth{
		Type:            "hmac",
		Secret:          "test-secret",
		Algorithm:       "sha256",
		SignatureHeader: "X-Hub-Signature-256",
	}

	// Test delivery with authentication
	delivery, err := service.DeliverWebhook(ctx, server.URL, payload, nil, auth)
	require.NoError(t, err)
	assert.NotNil(t, delivery)
	assert.Equal(t, integrations.DeliveryStatusDelivered, delivery.Status)
}

func TestWebhookDeliveryService_DeliverWebhookFailure(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)

	// Test failed delivery
	delivery, err := service.DeliverWebhook(ctx, server.URL, payload, nil, nil)
	require.NoError(t, err) // Service doesn't return error, delivery does
	assert.NotNil(t, delivery)
	// The delivery might be in retrying status initially, so we check for either failed or retrying
	assert.True(t, delivery.Status == integrations.DeliveryStatusFailed || delivery.Status == integrations.DeliveryStatusRetrying)
	assert.Equal(t, 500, delivery.ResponseCode)
	if delivery.Status == integrations.DeliveryStatusFailed {
		assert.Contains(t, delivery.Error, "webhook delivery failed with status 500")
	}
}

func TestWebhookDeliveryService_RetryLogic(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

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

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)

	// Test delivery with retry
	delivery, err := service.DeliverWebhook(ctx, server.URL, payload, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, delivery)

	// Wait for retries to complete
	time.Sleep(5 * time.Second)

	// Get updated delivery status
	updatedDelivery, err := service.GetDelivery(delivery.ID)
	require.NoError(t, err)

	// Should eventually succeed after retries
	assert.Equal(t, integrations.DeliveryStatusDelivered, updatedDelivery.Status)
	assert.Equal(t, 3, attemptCount) // Should have made 3 attempts total
}

func TestWebhookDeliveryService_GetDelivery(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)

	// Create delivery
	delivery, err := service.DeliverWebhook(ctx, server.URL, payload, nil, nil)
	require.NoError(t, err)

	// Retrieve delivery
	retrievedDelivery, err := service.GetDelivery(delivery.ID)
	require.NoError(t, err)
	assert.Equal(t, delivery.ID, retrievedDelivery.ID)
	assert.Equal(t, delivery.URL, retrievedDelivery.URL)
	assert.Equal(t, delivery.Status, retrievedDelivery.Status)
}

func TestWebhookDeliveryService_ListDeliveries(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)

	// Create multiple deliveries
	delivery1, err := service.DeliverWebhook(ctx, server.URL, payload, nil, nil)
	require.NoError(t, err)

	delivery2, err := service.DeliverWebhook(ctx, server.URL, payload, nil, nil)
	require.NoError(t, err)

	// List all deliveries
	deliveries, err := service.ListDeliveries("", 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(deliveries), 2)

	// List only delivered deliveries
	deliveredDeliveries, err := service.ListDeliveries(integrations.DeliveryStatusDelivered, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(deliveredDeliveries), 2)

	// Verify specific deliveries are in the list
	deliveryIDs := make(map[string]bool)
	for _, d := range deliveries {
		deliveryIDs[d.ID] = true
	}
	assert.True(t, deliveryIDs[delivery1.ID])
	assert.True(t, deliveryIDs[delivery2.ID])
}

func TestWebhookDeliveryService_GetDeliveryStats(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)

	// Create a delivery
	_, err := service.DeliverWebhook(ctx, server.URL, payload, nil, nil)
	require.NoError(t, err)

	// Get stats
	stats := service.GetDeliveryStats()
	assert.Contains(t, stats, "total_deliveries")
	assert.Contains(t, stats, "delivered_deliveries")
	assert.Contains(t, stats, "failed_deliveries")
	assert.Contains(t, stats, "queue_size")

	assert.GreaterOrEqual(t, stats["total_deliveries"].(int), 1)
	assert.GreaterOrEqual(t, stats["delivered_deliveries"].(int), 1)
}

func TestWebhookDeliveryService_RetryDelivery(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	attemptCount := 0
	// Create a test server that fails first attempt, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "service unavailable"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success"}`))
		}
	}))
	defer server.Close()

	ctx := context.Background()
	payload := []byte(`{"message": "test"}`)

	// Create delivery that will fail
	delivery, err := service.DeliverWebhook(ctx, server.URL, payload, nil, nil)
	require.NoError(t, err)

	// Wait a bit for initial delivery attempt
	time.Sleep(100 * time.Millisecond)

	// Manually retry
	err = service.RetryDelivery(ctx, delivery.ID)
	require.NoError(t, err)

	// Get updated delivery
	updatedDelivery, err := service.GetDelivery(delivery.ID)
	require.NoError(t, err)
	assert.Equal(t, integrations.DeliveryStatusDelivered, updatedDelivery.Status)
	assert.Equal(t, 2, attemptCount) // Should have made 2 attempts total
}

func TestWebhookDeliveryService_Configuration(t *testing.T) {
	service := integrations.NewWebhookDeliveryService()
	defer service.Stop()

	// Get current config
	config := service.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 5*time.Second, config.RetryDelay)

	// Update config
	newConfig := &integrations.WebhookDeliveryConfig{
		MaxRetries:   5,
		RetryDelay:   10 * time.Second,
		RetryBackoff: 1.5,
		Timeout:      60 * time.Second,
		WorkerCount:  10,
	}
	service.SetConfig(newConfig)

	// Verify config was updated
	updatedConfig := service.GetConfig()
	assert.Equal(t, 5, updatedConfig.MaxRetries)
	assert.Equal(t, 10*time.Second, updatedConfig.RetryDelay)
	assert.Equal(t, 1.5, updatedConfig.RetryBackoff)
	assert.Equal(t, 60*time.Second, updatedConfig.Timeout)
	assert.Equal(t, 10, updatedConfig.WorkerCount)
}
