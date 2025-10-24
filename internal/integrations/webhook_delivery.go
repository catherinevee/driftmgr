package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// WebhookDeliveryService handles webhook delivery with retry logic and authentication
type WebhookDeliveryService struct {
	client      *http.Client
	deliveries  map[string]*WebhookDelivery
	mu          sync.RWMutex
	config      *WebhookDeliveryConfig
	retryQueue  chan *WebhookDelivery
	stopCh      chan struct{}
	workerCount int
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID           string                 `json:"id"`
	URL          string                 `json:"url"`
	Payload      []byte                 `json:"payload"`
	Headers      map[string]string      `json:"headers"`
	Method       string                 `json:"method"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	Status       DeliveryStatus         `json:"status"`
	LastAttempt  time.Time              `json:"last_attempt"`
	NextRetry    time.Time              `json:"next_retry"`
	Error        string                 `json:"error,omitempty"`
	ResponseCode int                    `json:"response_code,omitempty"`
	ResponseBody string                 `json:"response_body,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// DeliveryStatus represents the status of a webhook delivery
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
	DeliveryStatusExpired   DeliveryStatus = "expired"
)

// WebhookDeliveryConfig represents configuration for webhook delivery
type WebhookDeliveryConfig struct {
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	RetryBackoff      float64       `json:"retry_backoff"`
	Timeout           time.Duration `json:"timeout"`
	MaxDeliveryAge    time.Duration `json:"max_delivery_age"`
	WorkerCount       int           `json:"worker_count"`
	QueueSize         int           `json:"queue_size"`
	EnableAuth        bool          `json:"enable_auth"`
	DefaultAuthMethod string        `json:"default_auth_method"`
}

// WebhookAuth represents webhook authentication configuration
type WebhookAuth struct {
	Type            string            `json:"type"` // hmac, bearer, basic, custom
	Secret          string            `json:"secret,omitempty"`
	Token           string            `json:"token,omitempty"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"password,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	SignatureHeader string            `json:"signature_header,omitempty"`
	Algorithm       string            `json:"algorithm,omitempty"` // sha256, sha1, etc.
}

// NewWebhookDeliveryService creates a new webhook delivery service
func NewWebhookDeliveryService() *WebhookDeliveryService {
	config := &WebhookDeliveryConfig{
		MaxRetries:        3,
		RetryDelay:        5 * time.Second,
		RetryBackoff:      2.0,
		Timeout:           30 * time.Second,
		MaxDeliveryAge:    24 * time.Hour,
		WorkerCount:       5,
		QueueSize:         1000,
		EnableAuth:        true,
		DefaultAuthMethod: "hmac",
	}

	service := &WebhookDeliveryService{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		deliveries:  make(map[string]*WebhookDelivery),
		config:      config,
		retryQueue:  make(chan *WebhookDelivery, config.QueueSize),
		stopCh:      make(chan struct{}),
		workerCount: config.WorkerCount,
	}

	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		go service.worker(i)
	}

	// Start cleanup goroutine
	go service.cleanup()

	return service
}

// DeliverWebhook delivers a webhook with the given configuration
func (wds *WebhookDeliveryService) DeliverWebhook(ctx context.Context, url string, payload []byte, headers map[string]string, auth *WebhookAuth) (*WebhookDelivery, error) {
	delivery := &WebhookDelivery{
		ID:          fmt.Sprintf("webhook_%d", time.Now().UnixNano()),
		URL:         url,
		Payload:     payload,
		Headers:     make(map[string]string),
		Method:      "POST",
		RetryCount:  0,
		MaxRetries:  wds.config.MaxRetries,
		Status:      DeliveryStatusPending,
		LastAttempt: time.Now(),
		NextRetry:   time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Copy headers
	for k, v := range headers {
		delivery.Headers[k] = v
	}

	// Add authentication headers
	if auth != nil && wds.config.EnableAuth {
		if err := wds.addAuthHeaders(delivery, auth); err != nil {
			return nil, fmt.Errorf("failed to add auth headers: %w", err)
		}
	}

	// Set default headers
	if delivery.Headers["Content-Type"] == "" {
		delivery.Headers["Content-Type"] = "application/json"
	}
	if delivery.Headers["User-Agent"] == "" {
		delivery.Headers["User-Agent"] = "DriftMgr-Webhook/1.0"
	}

	// Store delivery
	wds.mu.Lock()
	wds.deliveries[delivery.ID] = delivery
	wds.mu.Unlock()

	// Attempt delivery
	if err := wds.attemptDelivery(ctx, delivery); err != nil {
		delivery.Error = err.Error()
		delivery.Status = DeliveryStatusFailed
		delivery.UpdatedAt = time.Now()

		// Schedule retry if not exceeded max retries
		if delivery.RetryCount < delivery.MaxRetries {
			delivery.Status = DeliveryStatusRetrying
			delivery.RetryCount++
			delivery.NextRetry = time.Now().Add(wds.calculateRetryDelay(delivery.RetryCount))
			delivery.UpdatedAt = time.Now()

			// Add to retry queue
			select {
			case wds.retryQueue <- delivery:
			default:
				// Queue is full, mark as expired
				delivery.Status = DeliveryStatusExpired
				delivery.Error = "retry queue full"
			}
		}
	} else {
		delivery.Status = DeliveryStatusDelivered
		delivery.UpdatedAt = time.Now()
	}

	return delivery, nil
}

// GetDelivery retrieves a webhook delivery by ID
func (wds *WebhookDeliveryService) GetDelivery(deliveryID string) (*WebhookDelivery, error) {
	wds.mu.RLock()
	defer wds.mu.RUnlock()

	delivery, exists := wds.deliveries[deliveryID]
	if !exists {
		return nil, fmt.Errorf("delivery %s not found", deliveryID)
	}

	return delivery, nil
}

// ListDeliveries lists all webhook deliveries with optional filtering
func (wds *WebhookDeliveryService) ListDeliveries(status DeliveryStatus, limit int) ([]*WebhookDelivery, error) {
	wds.mu.RLock()
	defer wds.mu.RUnlock()

	deliveries := make([]*WebhookDelivery, 0)
	count := 0

	for _, delivery := range wds.deliveries {
		if status == "" || delivery.Status == status {
			deliveries = append(deliveries, delivery)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return deliveries, nil
}

// RetryDelivery manually retries a failed delivery
func (wds *WebhookDeliveryService) RetryDelivery(ctx context.Context, deliveryID string) error {
	wds.mu.Lock()
	delivery, exists := wds.deliveries[deliveryID]
	wds.mu.Unlock()

	if !exists {
		return fmt.Errorf("delivery %s not found", deliveryID)
	}

	if delivery.Status == DeliveryStatusDelivered {
		return fmt.Errorf("delivery %s already delivered", deliveryID)
	}

	if delivery.RetryCount >= delivery.MaxRetries {
		return fmt.Errorf("delivery %s exceeded max retries", deliveryID)
	}

	// Reset for retry
	delivery.Status = DeliveryStatusRetrying
	delivery.RetryCount++
	delivery.NextRetry = time.Now()
	delivery.UpdatedAt = time.Now()

	// Attempt delivery
	if err := wds.attemptDelivery(ctx, delivery); err != nil {
		delivery.Error = err.Error()
		delivery.Status = DeliveryStatusFailed
		delivery.UpdatedAt = time.Now()
		return err
	}

	delivery.Status = DeliveryStatusDelivered
	delivery.UpdatedAt = time.Now()
	return nil
}

// GetDeliveryStats returns delivery statistics
func (wds *WebhookDeliveryService) GetDeliveryStats() map[string]interface{} {
	wds.mu.RLock()
	defer wds.mu.RUnlock()

	stats := map[string]interface{}{
		"total_deliveries":     len(wds.deliveries),
		"pending_deliveries":   0,
		"delivered_deliveries": 0,
		"failed_deliveries":    0,
		"retrying_deliveries":  0,
		"expired_deliveries":   0,
		"queue_size":           len(wds.retryQueue),
	}

	for _, delivery := range wds.deliveries {
		switch delivery.Status {
		case DeliveryStatusPending:
			stats["pending_deliveries"] = stats["pending_deliveries"].(int) + 1
		case DeliveryStatusDelivered:
			stats["delivered_deliveries"] = stats["delivered_deliveries"].(int) + 1
		case DeliveryStatusFailed:
			stats["failed_deliveries"] = stats["failed_deliveries"].(int) + 1
		case DeliveryStatusRetrying:
			stats["retrying_deliveries"] = stats["retrying_deliveries"].(int) + 1
		case DeliveryStatusExpired:
			stats["expired_deliveries"] = stats["expired_deliveries"].(int) + 1
		}
	}

	return stats
}

// Stop stops the webhook delivery service
func (wds *WebhookDeliveryService) Stop() {
	close(wds.stopCh)
}

// SetConfig updates the webhook delivery service configuration
func (wds *WebhookDeliveryService) SetConfig(config *WebhookDeliveryConfig) {
	wds.mu.Lock()
	defer wds.mu.Unlock()
	wds.config = config
}

// GetConfig returns the current webhook delivery service configuration
func (wds *WebhookDeliveryService) GetConfig() *WebhookDeliveryConfig {
	wds.mu.RLock()
	defer wds.mu.RUnlock()
	return wds.config
}

// Private methods

func (wds *WebhookDeliveryService) attemptDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, delivery.Method, delivery.URL, bytes.NewReader(delivery.Payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for k, v := range delivery.Headers {
		req.Header.Set(k, v)
	}

	// Make request
	resp, err := wds.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Update delivery with response
	delivery.ResponseCode = resp.StatusCode
	delivery.ResponseBody = string(body)

	// Check for success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook delivery failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (wds *WebhookDeliveryService) addAuthHeaders(delivery *WebhookDelivery, auth *WebhookAuth) error {
	switch auth.Type {
	case "hmac":
		return wds.addHMACAuth(delivery, auth)
	case "bearer":
		delivery.Headers["Authorization"] = "Bearer " + auth.Token
	case "basic":
		// Basic auth would be handled by http.Client with Transport
		delivery.Headers["Authorization"] = "Basic " + auth.Token
	case "custom":
		for k, v := range auth.Headers {
			delivery.Headers[k] = v
		}
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}
	return nil
}

func (wds *WebhookDeliveryService) addHMACAuth(delivery *WebhookDelivery, auth *WebhookAuth) error {
	if auth.Secret == "" {
		return fmt.Errorf("HMAC secret is required")
	}

	algorithm := auth.Algorithm
	if algorithm == "" {
		algorithm = "sha256"
	}

	var hashFunc func() hash.Hash
	switch strings.ToLower(algorithm) {
	case "sha256":
		hashFunc = sha256.New
	case "sha1":
		hashFunc = sha1.New
	default:
		return fmt.Errorf("unsupported HMAC algorithm: %s", algorithm)
	}

	// Create HMAC signature
	mac := hmac.New(hashFunc, []byte(auth.Secret))
	mac.Write(delivery.Payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	// Add signature header
	signatureHeader := auth.SignatureHeader
	if signatureHeader == "" {
		signatureHeader = "X-Hub-Signature-256"
	}
	delivery.Headers[signatureHeader] = algorithm + "=" + signature

	return nil
}

func (wds *WebhookDeliveryService) calculateRetryDelay(retryCount int) time.Duration {
	delay := wds.config.RetryDelay
	for i := 0; i < retryCount; i++ {
		delay = time.Duration(float64(delay) * wds.config.RetryBackoff)
	}
	return delay
}

func (wds *WebhookDeliveryService) worker(workerID int) {
	for {
		select {
		case <-wds.stopCh:
			return
		case delivery := <-wds.retryQueue:
			// Check if it's time to retry
			if time.Now().Before(delivery.NextRetry) {
				// Put it back in the queue with a delay
				go func() {
					time.Sleep(time.Until(delivery.NextRetry))
					select {
					case wds.retryQueue <- delivery:
					case <-wds.stopCh:
					}
				}()
				continue
			}

			// Attempt delivery
			ctx, cancel := context.WithTimeout(context.Background(), wds.config.Timeout)
			err := wds.attemptDelivery(ctx, delivery)
			cancel()

			wds.mu.Lock()
			if err != nil {
				delivery.Error = err.Error()
				delivery.Status = DeliveryStatusFailed
				delivery.UpdatedAt = time.Now()

				// Schedule another retry if not exceeded max retries
				if delivery.RetryCount < delivery.MaxRetries {
					delivery.Status = DeliveryStatusRetrying
					delivery.RetryCount++
					delivery.NextRetry = time.Now().Add(wds.calculateRetryDelay(delivery.RetryCount))
					delivery.UpdatedAt = time.Now()

					// Add back to retry queue
					select {
					case wds.retryQueue <- delivery:
					default:
						delivery.Status = DeliveryStatusExpired
						delivery.Error = "retry queue full"
					}
				}
			} else {
				delivery.Status = DeliveryStatusDelivered
				delivery.UpdatedAt = time.Now()
			}
			wds.mu.Unlock()
		}
	}
}

func (wds *WebhookDeliveryService) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-wds.stopCh:
			return
		case <-ticker.C:
			wds.mu.Lock()
			now := time.Now()
			for id, delivery := range wds.deliveries {
				// Remove expired deliveries
				if now.Sub(delivery.CreatedAt) > wds.config.MaxDeliveryAge {
					delete(wds.deliveries, id)
				}
			}
			wds.mu.Unlock()
		}
	}
}
