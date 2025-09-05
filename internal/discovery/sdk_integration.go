package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// SDKIntegration provides SDK integration capabilities for various cloud providers
type SDKIntegration struct {
	mu            sync.RWMutex
	providers     map[string]CloudSDK
	rateLimiters  map[string]*RateLimiter
	retryPolicies map[string]*RetryPolicy
	credentials   map[string]Credentials
	clientCache   map[string]interface{}
	metrics       *SDKMetrics
}

// CloudSDK interface for cloud provider SDKs
type CloudSDK interface {
	Name() string
	Initialize(credentials Credentials) error
	ListResources(ctx context.Context, resourceType string, params map[string]interface{}) ([]models.Resource, error)
	GetResource(ctx context.Context, resourceID string) (*models.Resource, error)
	TagResource(ctx context.Context, resourceID string, tags map[string]string) error
	GetAPICallCount() int64
	GetLastError() error
}

// Credentials represents cloud provider credentials
type Credentials struct {
	Provider       string
	AccessKey      string
	SecretKey      string
	Token          string
	Region         string
	ProjectID      string
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	ServiceAccount string
	KeyFile        string
	Extra          map[string]string
}

// RateLimiter implements rate limiting for API calls
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
	waitQueue  []chan struct{}
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []string
}

// SDKMetrics tracks SDK usage metrics
type SDKMetrics struct {
	APICallsTotal     map[string]int64
	APICallsPerSecond map[string]float64
	ErrorCount        map[string]int64
	RetryCount        map[string]int64
	RateLimitHits     map[string]int64
	AverageLatency    map[string]time.Duration
	LastAPICall       map[string]time.Time
}

// NewSDKIntegration creates a new SDK integration manager
func NewSDKIntegration() *SDKIntegration {
	return &SDKIntegration{
		providers:     make(map[string]CloudSDK),
		rateLimiters:  make(map[string]*RateLimiter),
		retryPolicies: make(map[string]*RetryPolicy),
		credentials:   make(map[string]Credentials),
		clientCache:   make(map[string]interface{}),
		metrics: &SDKMetrics{
			APICallsTotal:     make(map[string]int64),
			APICallsPerSecond: make(map[string]float64),
			ErrorCount:        make(map[string]int64),
			RetryCount:        make(map[string]int64),
			RateLimitHits:     make(map[string]int64),
			AverageLatency:    make(map[string]time.Duration),
			LastAPICall:       make(map[string]time.Time),
		},
	}
}

// RegisterProvider registers a cloud provider SDK
func (si *SDKIntegration) RegisterProvider(provider string, sdk CloudSDK) {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.providers[provider] = sdk

	// Set default rate limiter
	si.rateLimiters[provider] = &RateLimiter{
		maxTokens:  100,
		tokens:     100,
		refillRate: 10,
		lastRefill: time.Now(),
		waitQueue:  make([]chan struct{}, 0),
	}

	// Set default retry policy
	si.retryPolicies[provider] = &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"Throttling",
			"ServiceUnavailable",
			"RequestTimeout",
			"TooManyRequests",
		},
	}
}

// SetCredentials sets credentials for a provider
func (si *SDKIntegration) SetCredentials(provider string, creds Credentials) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	if _, exists := si.providers[provider]; !exists {
		return fmt.Errorf("provider %s not registered", provider)
	}

	si.credentials[provider] = creds

	// Initialize the SDK with credentials
	if err := si.providers[provider].Initialize(creds); err != nil {
		return fmt.Errorf("failed to initialize %s SDK: %w", provider, err)
	}

	return nil
}

// SetRateLimiter sets a custom rate limiter for a provider
func (si *SDKIntegration) SetRateLimiter(provider string, maxTokens, refillRate int) {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.rateLimiters[provider] = &RateLimiter{
		maxTokens:  maxTokens,
		tokens:     maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
		waitQueue:  make([]chan struct{}, 0),
	}
}

// SetRetryPolicy sets a custom retry policy for a provider
func (si *SDKIntegration) SetRetryPolicy(provider string, policy *RetryPolicy) {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.retryPolicies[provider] = policy
}

// CallAPI makes an API call with rate limiting and retry logic
func (si *SDKIntegration) CallAPI(ctx context.Context, provider string, operation func() (interface{}, error)) (interface{}, error) {
	// Check if provider exists
	si.mu.RLock()
	rateLimiter, hasLimiter := si.rateLimiters[provider]
	retryPolicy, hasPolicy := si.retryPolicies[provider]
	si.mu.RUnlock()

	if !hasLimiter || !hasPolicy {
		return nil, fmt.Errorf("provider %s not configured", provider)
	}

	// Apply rate limiting
	if err := si.applyRateLimit(provider, rateLimiter); err != nil {
		return nil, err
	}

	// Execute with retry logic
	var result interface{}
	var lastErr error
	delay := retryPolicy.InitialDelay

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		startTime := time.Now()

		result, lastErr = operation()

		// Update metrics
		si.updateMetrics(provider, time.Since(startTime), lastErr)

		if lastErr == nil {
			return result, nil
		}

		// Check if error is retryable
		if !si.isRetryableError(lastErr, retryPolicy) {
			break
		}

		if attempt < retryPolicy.MaxRetries {
			si.mu.Lock()
			si.metrics.RetryCount[provider]++
			si.mu.Unlock()

			// Apply backoff
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * retryPolicy.BackoffFactor)
				if delay > retryPolicy.MaxDelay {
					delay = retryPolicy.MaxDelay
				}
			}
		}
	}

	return nil, fmt.Errorf("API call failed after %d retries: %w", retryPolicy.MaxRetries, lastErr)
}

// ListResources lists resources using the appropriate SDK
func (si *SDKIntegration) ListResources(ctx context.Context, provider, resourceType string, params map[string]interface{}) ([]models.Resource, error) {
	si.mu.RLock()
	sdk, exists := si.providers[provider]
	si.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not registered", provider)
	}

	result, err := si.CallAPI(ctx, provider, func() (interface{}, error) {
		return sdk.ListResources(ctx, resourceType, params)
	})

	if err != nil {
		return nil, err
	}

	return result.([]models.Resource), nil
}

// GetResource gets a single resource using the SDK
func (si *SDKIntegration) GetResource(ctx context.Context, provider, resourceID string) (*models.Resource, error) {
	si.mu.RLock()
	sdk, exists := si.providers[provider]
	si.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not registered", provider)
	}

	result, err := si.CallAPI(ctx, provider, func() (interface{}, error) {
		return sdk.GetResource(ctx, resourceID)
	})

	if err != nil {
		return nil, err
	}

	return result.(*models.Resource), nil
}

// TagResource tags a resource using the SDK
func (si *SDKIntegration) TagResource(ctx context.Context, provider, resourceID string, tags map[string]string) error {
	si.mu.RLock()
	sdk, exists := si.providers[provider]
	si.mu.RUnlock()

	if !exists {
		return fmt.Errorf("provider %s not registered", provider)
	}

	_, err := si.CallAPI(ctx, provider, func() (interface{}, error) {
		return nil, sdk.TagResource(ctx, resourceID, tags)
	})

	return err
}

// GetMetrics returns SDK usage metrics
func (si *SDKIntegration) GetMetrics() *SDKMetrics {
	si.mu.RLock()
	defer si.mu.RUnlock()

	// Create a copy to avoid race conditions
	metricsCopy := &SDKMetrics{
		APICallsTotal:     make(map[string]int64),
		APICallsPerSecond: make(map[string]float64),
		ErrorCount:        make(map[string]int64),
		RetryCount:        make(map[string]int64),
		RateLimitHits:     make(map[string]int64),
		AverageLatency:    make(map[string]time.Duration),
		LastAPICall:       make(map[string]time.Time),
	}

	for k, v := range si.metrics.APICallsTotal {
		metricsCopy.APICallsTotal[k] = v
	}
	for k, v := range si.metrics.APICallsPerSecond {
		metricsCopy.APICallsPerSecond[k] = v
	}
	for k, v := range si.metrics.ErrorCount {
		metricsCopy.ErrorCount[k] = v
	}
	for k, v := range si.metrics.RetryCount {
		metricsCopy.RetryCount[k] = v
	}
	for k, v := range si.metrics.RateLimitHits {
		metricsCopy.RateLimitHits[k] = v
	}
	for k, v := range si.metrics.AverageLatency {
		metricsCopy.AverageLatency[k] = v
	}
	for k, v := range si.metrics.LastAPICall {
		metricsCopy.LastAPICall[k] = v
	}

	return metricsCopy
}

// GetProviderStatus returns the status of a provider
func (si *SDKIntegration) GetProviderStatus(provider string) map[string]interface{} {
	si.mu.RLock()
	defer si.mu.RUnlock()

	sdk, exists := si.providers[provider]
	if !exists {
		return map[string]interface{}{
			"status": "not_registered",
		}
	}

	status := map[string]interface{}{
		"status":          "active",
		"api_calls_total": si.metrics.APICallsTotal[provider],
		"error_count":     si.metrics.ErrorCount[provider],
		"retry_count":     si.metrics.RetryCount[provider],
		"rate_limit_hits": si.metrics.RateLimitHits[provider],
		"average_latency": si.metrics.AverageLatency[provider].String(),
		"last_api_call":   si.metrics.LastAPICall[provider].Format(time.RFC3339),
		"has_credentials": si.credentials[provider].Provider != "",
	}

	if sdk != nil {
		status["sdk_api_calls"] = sdk.GetAPICallCount()
		if lastErr := sdk.GetLastError(); lastErr != nil {
			status["last_error"] = lastErr.Error()
		}
	}

	return status
}

// ClearCache clears the client cache
func (si *SDKIntegration) ClearCache() {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.clientCache = make(map[string]interface{})
}

// Helper functions

func (si *SDKIntegration) applyRateLimit(provider string, limiter *RateLimiter) error {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(limiter.lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(limiter.refillRate))

	if tokensToAdd > 0 {
		limiter.tokens += tokensToAdd
		if limiter.tokens > limiter.maxTokens {
			limiter.tokens = limiter.maxTokens
		}
		limiter.lastRefill = now
	}

	// Check if we have tokens available
	if limiter.tokens > 0 {
		limiter.tokens--
		return nil
	}

	// No tokens available, increment rate limit hit counter
	si.mu.Lock()
	si.metrics.RateLimitHits[provider]++
	si.mu.Unlock()

	// Wait for token
	waitCh := make(chan struct{})
	limiter.waitQueue = append(limiter.waitQueue, waitCh)

	go func() {
		time.Sleep(time.Second / time.Duration(limiter.refillRate))
		close(waitCh)
	}()

	<-waitCh
	return nil
}

func (si *SDKIntegration) isRetryableError(err error, policy *RetryPolicy) bool {
	errStr := err.Error()
	for _, retryableErr := range policy.RetryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}
	return false
}

func (si *SDKIntegration) updateMetrics(provider string, latency time.Duration, err error) {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.metrics.APICallsTotal[provider]++
	si.metrics.LastAPICall[provider] = time.Now()

	// Update average latency (simple moving average)
	if currentAvg, exists := si.metrics.AverageLatency[provider]; exists {
		callCount := si.metrics.APICallsTotal[provider]
		newAvg := (currentAvg*time.Duration(callCount-1) + latency) / time.Duration(callCount)
		si.metrics.AverageLatency[provider] = newAvg
	} else {
		si.metrics.AverageLatency[provider] = latency
	}

	// Calculate calls per second
	if lastCall, exists := si.metrics.LastAPICall[provider]; exists {
		timeDiff := time.Now().Sub(lastCall).Seconds()
		if timeDiff > 0 {
			si.metrics.APICallsPerSecond[provider] = 1.0 / timeDiff
		}
	}

	if err != nil {
		si.metrics.ErrorCount[provider]++
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Cleanup cleans up resources
func (si *SDKIntegration) Cleanup() {
	si.mu.Lock()
	defer si.mu.Unlock()

	// Clear all caches and connections
	si.clientCache = make(map[string]interface{})
	si.providers = make(map[string]CloudSDK)
}

// Default SDK implementations would be added here for each provider
// These would wrap the actual cloud provider SDKs
