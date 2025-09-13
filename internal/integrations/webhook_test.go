package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebhookConfig(t *testing.T) {
	config := &WebhookConfig{
		MaxHandlers:       50,
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Second,
		ValidationEnabled: true,
		LoggingEnabled:    true,
	}

	assert.Equal(t, 50, config.MaxHandlers)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 5*time.Second, config.RetryDelay)
	assert.True(t, config.ValidationEnabled)
	assert.True(t, config.LoggingEnabled)
}

func TestWebhookResult(t *testing.T) {
	result := &WebhookResult{
		ID:        "webhook-123",
		Status:    "success",
		Message:   "Webhook processed successfully",
		Data: map[string]interface{}{
			"resources": 10,
			"severity":  "high",
		},
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"version": "1.0",
		},
	}

	assert.Equal(t, "webhook-123", result.ID)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "Webhook processed successfully", result.Message)
	assert.Equal(t, 10, result.Data["resources"])
	assert.NotZero(t, result.Timestamp)

	// Test JSON marshaling
	data, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "webhook-123")
}

func TestNewWebhookHandler(t *testing.T) {
	handler := NewWebhookHandler()

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.handlers)
	assert.NotNil(t, handler.config)
	assert.Equal(t, 50, handler.config.MaxHandlers)
	assert.Equal(t, 30*time.Second, handler.config.Timeout)
}

func TestWebhookHandler_Register(t *testing.T) {
	handler := NewWebhookHandler()

	// Create mock processor
	mockProcessor := &mockWebhookProcessor{
		processFunc: func(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
			return &WebhookResult{
				ID:      "test-123",
				Status:  "success",
				Message: "Processed",
			}, nil
		},
	}

	// Register processor
	err := handler.Register("test-webhook", mockProcessor)
	assert.NoError(t, err)

	// Verify registration
	handler.mu.RLock()
	processor, exists := handler.handlers["test-webhook"]
	handler.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, mockProcessor, processor)
}

func TestWebhookHandler_Process(t *testing.T) {
	handler := NewWebhookHandler()

	// Register processor
	mockProcessor := &mockWebhookProcessor{
		processFunc: func(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
			var data map[string]interface{}
			json.Unmarshal(payload, &data)
			return &WebhookResult{
				ID:      "processed-123",
				Status:  "success",
				Message: fmt.Sprintf("Processed event: %s", data["event"]),
			}, nil
		},
	}
	handler.Register("test", mockProcessor)

	// Process webhook
	payload := []byte(`{"event":"test.event","data":"test"}`)
	headers := map[string]string{"Content-Type": "application/json"}

	result, err := handler.Process(context.Background(), "test", payload, headers)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Status)
	assert.Contains(t, result.Message, "test.event")
}

func TestWebhookHandler_Unregister(t *testing.T) {
	handler := NewWebhookHandler()

	// Register processor
	mockProcessor := &mockWebhookProcessor{}
	handler.Register("test", mockProcessor)

	// Verify it exists
	handler.mu.RLock()
	_, exists := handler.handlers["test"]
	handler.mu.RUnlock()
	assert.True(t, exists)

	// Unregister
	handler.Unregister("test")

	// Verify it's gone
	handler.mu.RLock()
	_, exists = handler.handlers["test"]
	handler.mu.RUnlock()
	assert.False(t, exists)
}

func TestWebhookHandler_ProcessWithTimeout(t *testing.T) {
	handler := NewWebhookHandler()
	handler.config.Timeout = 100 * time.Millisecond

	// Register slow processor
	mockProcessor := &mockWebhookProcessor{
		processFunc: func(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return &WebhookResult{Status: "success"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	handler.Register("slow", mockProcessor)

	// Process should timeout
	ctx := context.Background()
	_, err := handler.Process(ctx, "slow", []byte(`{}`), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestWebhookHandler_ConcurrentProcessing(t *testing.T) {
	handler := NewWebhookHandler()
	processedCount := 0

	// Register processor
	mockProcessor := &mockWebhookProcessor{
		processFunc: func(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
			processedCount++
			return &WebhookResult{
				ID:     fmt.Sprintf("result-%d", processedCount),
				Status: "success",
			}, nil
		},
	}
	handler.Register("concurrent", mockProcessor)

	// Process multiple webhooks concurrently
	const numWebhooks = 10
	results := make(chan *WebhookResult, numWebhooks)
	errors := make(chan error, numWebhooks)

	for i := 0; i < numWebhooks; i++ {
		go func(n int) {
			payload := []byte(fmt.Sprintf(`{"id":%d}`, n))
			result, err := handler.Process(context.Background(), "concurrent", payload, nil)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numWebhooks; i++ {
		select {
		case result := <-results:
			assert.Equal(t, "success", result.Status)
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for results")
		}
	}

	assert.Equal(t, numWebhooks, processedCount)
}

// Mock webhook processor for testing
type mockWebhookProcessor struct {
	processFunc func(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error)
}

func (m *mockWebhookProcessor) ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, payload, headers)
	}
	return &WebhookResult{
		ID:      "default-123",
		Status:  "success",
		Message: "Default response",
	}, nil
}

func BenchmarkWebhookHandler_Process(b *testing.B) {
	handler := NewWebhookHandler()

	// Register fast processor
	mockProcessor := &mockWebhookProcessor{
		processFunc: func(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
			return &WebhookResult{
				ID:     "bench-123",
				Status: "success",
			}, nil
		},
	}
	handler.Register("benchmark", mockProcessor)

	payload := []byte(`{"event":"benchmark"}`)
	headers := map[string]string{"Content-Type": "application/json"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Process(context.Background(), "benchmark", payload, headers)
	}
}