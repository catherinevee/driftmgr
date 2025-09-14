package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// WebhookHandler handles incoming webhooks
type WebhookHandler struct {
	handlers map[string]WebhookProcessor
	mu       sync.RWMutex
	config   *WebhookConfig
}

// WebhookProcessor represents a webhook processor
type WebhookProcessor interface {
	ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error)
}

// WebhookResult represents the result of webhook processing
type WebhookResult struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"` // success, error, ignored
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WebhookConfig represents configuration for webhook handling
type WebhookConfig struct {
	MaxHandlers       int           `json:"max_handlers"`
	Timeout           time.Duration `json:"timeout"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	ValidationEnabled bool          `json:"validation_enabled"`
	LoggingEnabled    bool          `json:"logging_enabled"`
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler() *WebhookHandler {
	config := &WebhookConfig{
		MaxHandlers:       50,
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Second,
		ValidationEnabled: true,
		LoggingEnabled:    true,
	}

	return &WebhookHandler{
		handlers: make(map[string]WebhookProcessor),
		config:   config,
	}
}

// RegisterHandler registers a webhook processor
func (wh *WebhookHandler) RegisterHandler(webhookType string, processor WebhookProcessor) error {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	// Check handler limit
	if len(wh.handlers) >= wh.config.MaxHandlers {
		return fmt.Errorf("maximum number of handlers reached (%d)", wh.config.MaxHandlers)
	}

	wh.handlers[webhookType] = processor
	return nil
}

// ProcessWebhook processes an incoming webhook
func (wh *WebhookHandler) ProcessWebhook(ctx context.Context, webhookType string, payload []byte, headers map[string]string) (*WebhookResult, error) {
	wh.mu.RLock()
	processor, exists := wh.handlers[webhookType]
	wh.mu.RUnlock()

	if !exists {
		return &WebhookResult{
			ID:        fmt.Sprintf("webhook_%d", time.Now().Unix()),
			Status:    "ignored",
			Message:   fmt.Sprintf("No handler for webhook type: %s", webhookType),
			Timestamp: time.Now(),
			Metadata:  make(map[string]interface{}),
		}, nil
	}

	// Create context with timeout
	processCtx, cancel := context.WithTimeout(ctx, wh.config.Timeout)
	defer cancel()

	// Process webhook
	result, err := processor.ProcessWebhook(processCtx, payload, headers)
	if err != nil {
		return &WebhookResult{
			ID:        fmt.Sprintf("webhook_%d", time.Now().Unix()),
			Status:    "error",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Metadata:  make(map[string]interface{}),
		}, err
	}

	return result, nil
}

// HandleHTTP handles HTTP webhook requests
func (wh *WebhookHandler) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract webhook type from URL path
	webhookType := r.URL.Path[1:] // Remove leading slash

	// Read payload
	payload := make([]byte, r.ContentLength)
	if r.ContentLength > 0 {
		_, err := r.Body.Read(payload)
		if err != nil {
			http.Error(w, "Failed to read payload", http.StatusBadRequest)
			return
		}
	}

	// Extract headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Process webhook
	result, err := wh.ProcessWebhook(r.Context(), webhookType, payload, headers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Built-in webhook processors

// SlackWebhookProcessor processes Slack webhooks
type SlackWebhookProcessor struct{}

// ProcessWebhook processes a Slack webhook
func (swp *SlackWebhookProcessor) ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	var slackData map[string]interface{}
	if err := json.Unmarshal(payload, &slackData); err != nil {
		return nil, fmt.Errorf("failed to parse Slack payload: %w", err)
	}

	// Process Slack webhook data
	result := &WebhookResult{
		ID:        fmt.Sprintf("slack_%d", time.Now().Unix()),
		Status:    "success",
		Message:   "Slack webhook processed successfully",
		Data:      slackData,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "slack",
			"type":   "webhook",
		},
	}

	return result, nil
}

// TeamsWebhookProcessor processes Microsoft Teams webhooks
type TeamsWebhookProcessor struct{}

// ProcessWebhook processes a Teams webhook
func (twp *TeamsWebhookProcessor) ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	var teamsData map[string]interface{}
	if err := json.Unmarshal(payload, &teamsData); err != nil {
		return nil, fmt.Errorf("failed to parse Teams payload: %w", err)
	}

	// Process Teams webhook data
	result := &WebhookResult{
		ID:        fmt.Sprintf("teams_%d", time.Now().Unix()),
		Status:    "success",
		Message:   "Teams webhook processed successfully",
		Data:      teamsData,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "teams",
			"type":   "webhook",
		},
	}

	return result, nil
}

// PagerDutyWebhookProcessor processes PagerDuty webhooks
type PagerDutyWebhookProcessor struct{}

// ProcessWebhook processes a PagerDuty webhook
func (pdwp *PagerDutyWebhookProcessor) ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	var pagerDutyData map[string]interface{}
	if err := json.Unmarshal(payload, &pagerDutyData); err != nil {
		return nil, fmt.Errorf("failed to parse PagerDuty payload: %w", err)
	}

	// Process PagerDuty webhook data
	result := &WebhookResult{
		ID:        fmt.Sprintf("pagerduty_%d", time.Now().Unix()),
		Status:    "success",
		Message:   "PagerDuty webhook processed successfully",
		Data:      pagerDutyData,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "pagerduty",
			"type":   "webhook",
		},
	}

	return result, nil
}

// GitHubWebhookProcessor processes GitHub webhooks
type GitHubWebhookProcessor struct{}

// ProcessWebhook processes a GitHub webhook
func (ghwp *GitHubWebhookProcessor) ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	var githubData map[string]interface{}
	if err := json.Unmarshal(payload, &githubData); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub payload: %w", err)
	}

	// Process GitHub webhook data
	result := &WebhookResult{
		ID:        fmt.Sprintf("github_%d", time.Now().Unix()),
		Status:    "success",
		Message:   "GitHub webhook processed successfully",
		Data:      githubData,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "github",
			"type":   "webhook",
		},
	}

	return result, nil
}

// SetConfig updates the webhook handler configuration
func (wh *WebhookHandler) SetConfig(config *WebhookConfig) {
	wh.mu.Lock()
	defer wh.mu.Unlock()
	wh.config = config
}

// GetConfig returns the current webhook handler configuration
func (wh *WebhookHandler) GetConfig() *WebhookConfig {
	wh.mu.RLock()
	defer wh.mu.RUnlock()
	return wh.config
}

// Register is an alias for RegisterHandler for compatibility
func (wh *WebhookHandler) Register(webhookType string, processor WebhookProcessor) error {
	return wh.RegisterHandler(webhookType, processor)
}

// Process is an alias for ProcessWebhook for compatibility
func (wh *WebhookHandler) Process(ctx context.Context, webhookType string, payload []byte, headers map[string]string) (*WebhookResult, error) {
	return wh.ProcessWebhook(ctx, webhookType, payload, headers)
}

// Unregister removes a webhook processor
func (wh *WebhookHandler) Unregister(webhookType string) {
	wh.mu.Lock()
	defer wh.mu.Unlock()
	delete(wh.handlers, webhookType)
}
