package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logger"
	"github.com/catherinevee/driftmgr/internal/utils/circuit"
	"github.com/catherinevee/driftmgr/internal/telemetry"
)

// EventType represents the type of webhook event
type EventType string

const (
	EventDiscoveryStarted    EventType = "discovery.started"
	EventDiscoveryCompleted  EventType = "discovery.completed"
	EventDiscoveryFailed     EventType = "discovery.failed"
	EventDriftDetected       EventType = "drift.detected"
	EventDriftAnalyzed       EventType = "drift.analyzed"
	EventRemediationStarted  EventType = "remediation.started"
	EventRemediationCompleted EventType = "remediation.completed"
	EventRemediationFailed   EventType = "remediation.failed"
	EventAlertTriggered      EventType = "alert.triggered"
	EventAlertResolved       EventType = "alert.resolved"
)

// Event represents a webhook event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// Webhook represents a webhook endpoint configuration
type Webhook struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Secret      string            `json:"secret,omitempty"`
	Events      []EventType       `json:"events"`
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     bool              `json:"enabled"`
	RetryPolicy RetryPolicy       `json:"retry_policy"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// RetryPolicy defines webhook retry behavior
type RetryPolicy struct {
	MaxRetries     int           `json:"max_retries"`
	InitialDelay   time.Duration `json:"initial_delay"`
	MaxDelay       time.Duration `json:"max_delay"`
	BackoffFactor  float64       `json:"backoff_factor"`
	RetryOnStatus  []int         `json:"retry_on_status"`
}

// Manager manages webhooks and event dispatching
type Manager struct {
	webhooks       map[string]*Webhook
	client         *http.Client
	circuitBreaker *resilience.Manager
	eventQueue     chan *eventDispatch
	workers        int
	mu             sync.RWMutex
	wg             sync.WaitGroup
	shutdownCh     chan struct{}
	log            logger.Logger
}

// eventDispatch represents an event to be dispatched
type eventDispatch struct {
	webhook *Webhook
	event   *Event
	retries int
}

// Config represents webhook manager configuration
type Config struct {
	Workers        int
	QueueSize      int
	DefaultTimeout time.Duration
	CircuitBreaker *resilience.Config
}

// NewManager creates a new webhook manager
func NewManager(config Config) *Manager {
	if config.Workers <= 0 {
		config.Workers = 5
	}
	
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	
	m := &Manager{
		webhooks: make(map[string]*Webhook),
		client: &http.Client{
			Timeout: config.DefaultTimeout,
		},
		circuitBreaker: resilience.NewManager(),
		eventQueue:     make(chan *eventDispatch, config.QueueSize),
		workers:        config.Workers,
		shutdownCh:     make(chan struct{}),
		log:            logger.New("webhook_manager"),
	}
	
	// Start workers
	m.startWorkers()
	
	m.log.Info("Webhook manager initialized",
		logger.Int("workers", config.Workers),
		logger.Int("queue_size", config.QueueSize),
	)
	
	return m
}

// RegisterWebhook registers a new webhook
func (m *Manager) RegisterWebhook(webhook *Webhook) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if webhook.ID == "" {
		webhook.ID = generateID()
	}
	
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = time.Now()
	}
	
	webhook.UpdatedAt = time.Now()
	
	// Set default retry policy if not provided
	if webhook.RetryPolicy.MaxRetries == 0 {
		webhook.RetryPolicy = RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			RetryOnStatus: []int{500, 502, 503, 504},
		}
	}
	
	m.webhooks[webhook.ID] = webhook
	
	m.log.Info("Webhook registered",
		logger.String("id", webhook.ID),
		logger.String("name", webhook.Name),
		logger.String("url", webhook.URL),
		logger.Int("events", len(webhook.Events)),
	)
	
	return nil
}

// UnregisterWebhook removes a webhook
func (m *Manager) UnregisterWebhook(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.webhooks[id]; !exists {
		return fmt.Errorf("webhook not found: %s", id)
	}
	
	delete(m.webhooks, id)
	
	m.log.Info("Webhook unregistered",
		logger.String("id", id),
	)
	
	return nil
}

// GetWebhook returns a webhook by ID
func (m *Manager) GetWebhook(id string) (*Webhook, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	webhook, exists := m.webhooks[id]
	return webhook, exists
}

// ListWebhooks returns all registered webhooks
func (m *Manager) ListWebhooks() []*Webhook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	webhooks := make([]*Webhook, 0, len(m.webhooks))
	for _, webhook := range m.webhooks {
		webhooks = append(webhooks, webhook)
	}
	
	return webhooks
}

// Emit emits an event to all matching webhooks
func (m *Manager) Emit(event *Event) {
	if event.ID == "" {
		event.ID = generateID()
	}
	
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	dispatched := 0
	for _, webhook := range m.webhooks {
		if !webhook.Enabled {
			continue
		}
		
		// Check if webhook is subscribed to this event type
		if !m.isSubscribed(webhook, event.Type) {
			continue
		}
		
		// Queue event for dispatch
		select {
		case m.eventQueue <- &eventDispatch{
			webhook: webhook,
			event:   event,
			retries: 0,
		}:
			dispatched++
		default:
			m.log.Warn("Event queue full, dropping event",
				logger.String("event_id", event.ID),
				logger.String("event_type", string(event.Type)),
				logger.String("webhook_id", webhook.ID),
			)
		}
	}
	
	if dispatched > 0 {
		m.log.Debug("Event queued for dispatch",
			logger.String("event_id", event.ID),
			logger.String("event_type", string(event.Type)),
			logger.Int("webhooks", dispatched),
		)
	}
}

// startWorkers starts the webhook dispatch workers
func (m *Manager) startWorkers() {
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
}

// worker processes webhook dispatch queue
func (m *Manager) worker(id int) {
	defer m.wg.Done()
	
	m.log.Debug("Webhook worker started",
		logger.Int("worker_id", id),
	)
	
	for {
		select {
		case <-m.shutdownCh:
			m.log.Debug("Webhook worker shutting down",
				logger.Int("worker_id", id),
			)
			return
			
		case dispatch := <-m.eventQueue:
			m.dispatchEvent(dispatch)
		}
	}
}

// dispatchEvent dispatches an event to a webhook
func (m *Manager) dispatchEvent(dispatch *eventDispatch) {
	ctx := context.Background()
	
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "webhook.dispatch")
		defer span.End()
	}
	
	// Use circuit breaker for the webhook URL
	cb := m.circuitBreaker.GetOrCreate(dispatch.webhook.URL, resilience.Config{
		Name:         dispatch.webhook.URL,
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	})
	
	// Execute with circuit breaker
	_, err := cb.Call(ctx, func() (interface{}, error) {
		return nil, m.sendWebhook(ctx, dispatch.webhook, dispatch.event)
	})
	
	if err != nil {
		// Check if we should retry
		if m.shouldRetry(dispatch, err) {
			m.retryDispatch(dispatch)
		} else {
			m.log.Error("Failed to dispatch webhook",
				logger.String("webhook_id", dispatch.webhook.ID),
				logger.String("event_id", dispatch.event.ID),
				logger.Error(err),
				logger.Int("retries", dispatch.retries),
			)
		}
	} else {
		m.log.Debug("Webhook dispatched successfully",
			logger.String("webhook_id", dispatch.webhook.ID),
			logger.String("event_id", dispatch.event.ID),
		)
	}
}

// sendWebhook sends the webhook HTTP request
func (m *Manager) sendWebhook(ctx context.Context, webhook *Webhook, event *Event) error {
	// Prepare payload
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", string(event.Type))
	req.Header.Set("X-Webhook-ID", webhook.ID)
	req.Header.Set("X-Event-ID", event.ID)
	
	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}
	
	// Add signature if secret is configured
	if webhook.Secret != "" {
		signature := m.generateSignature(webhook.Secret, payload)
		req.Header.Set("X-Webhook-Signature", signature)
	}
	
	// Send request
	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	
	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

// generateSignature generates HMAC-SHA256 signature
func (m *Manager) generateSignature(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// isSubscribed checks if webhook is subscribed to event type
func (m *Manager) isSubscribed(webhook *Webhook, eventType EventType) bool {
	if len(webhook.Events) == 0 {
		return true // Subscribe to all events if none specified
	}
	
	for _, et := range webhook.Events {
		if et == eventType {
			return true
		}
	}
	
	return false
}

// shouldRetry determines if dispatch should be retried
func (m *Manager) shouldRetry(dispatch *eventDispatch, err error) bool {
	if dispatch.retries >= dispatch.webhook.RetryPolicy.MaxRetries {
		return false
	}
	
	// Check if error is retryable
	// This is simplified - in production you'd check specific error types
	return true
}

// retryDispatch retries a webhook dispatch
func (m *Manager) retryDispatch(dispatch *eventDispatch) {
	dispatch.retries++
	
	// Calculate delay with exponential backoff
	delay := dispatch.webhook.RetryPolicy.InitialDelay
	for i := 1; i < dispatch.retries; i++ {
		delay = time.Duration(float64(delay) * dispatch.webhook.RetryPolicy.BackoffFactor)
		if delay > dispatch.webhook.RetryPolicy.MaxDelay {
			delay = dispatch.webhook.RetryPolicy.MaxDelay
			break
		}
	}
	
	m.log.Debug("Retrying webhook dispatch",
		logger.String("webhook_id", dispatch.webhook.ID),
		logger.String("event_id", dispatch.event.ID),
		logger.Int("retry", dispatch.retries),
		logger.Duration("delay", delay),
	)
	
	// Schedule retry
	time.AfterFunc(delay, func() {
		select {
		case m.eventQueue <- dispatch:
		default:
			m.log.Warn("Failed to requeue webhook for retry",
				logger.String("webhook_id", dispatch.webhook.ID),
				logger.String("event_id", dispatch.event.ID),
			)
		}
	})
}

// Shutdown gracefully shuts down the webhook manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.log.Info("Shutting down webhook manager")
	
	close(m.shutdownCh)
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		m.log.Info("Webhook manager shutdown complete")
		return nil
	case <-ctx.Done():
		m.log.Warn("Webhook manager shutdown timed out")
		return ctx.Err()
	}
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}