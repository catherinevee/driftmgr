package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// ErrorType defines the type of error
type ErrorType string

const (
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeAuth       ErrorType = "authentication"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeUnknown    ErrorType = "unknown"
)

// DiscoveryError represents a discovery-specific error
type DiscoveryError struct {
	Type        ErrorType              `json:"type"`
	Provider    string                 `json:"provider"`
	Region      string                 `json:"region"`
	Service     string                 `json:"service"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Retryable   bool                   `json:"retryable"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     string                 `json:"context"`
	Suggestions []string               `json:"suggestions"`
}

// Error returns the error message
func (de *DiscoveryError) Error() string {
	return fmt.Sprintf("[%s] %s - %s (%s/%s): %s",
		de.Type, de.Provider, de.Service, de.Region, de.Context, de.Message)
}

// IsRetryable returns whether the error is retryable
func (de *DiscoveryError) IsRetryable() bool {
	return de.Retryable && de.RetryCount < de.MaxRetries
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries        int           `json:"max_retries"`
	InitialDelay      time.Duration `json:"initial_delay"`
	MaxDelay          time.Duration `json:"max_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	Jitter            bool          `json:"jitter"`
	RetryableErrors   []ErrorType   `json:"retryable_errors"`
}

// CircuitBreakerConfig defines circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	HalfOpenLimit    int           `json:"half_open_limit"`
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config          CircuitBreakerConfig
	state           CircuitBreakerState
	failureCount    int
	lastFailureTime time.Time
	mu              sync.RWMutex
}

// CircuitBreakerState represents the circuit breaker state
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	cb.mu.Lock()

	// Check if we can execute
	if !cb.canExecuteLocked() {
		state := cb.state
		cb.mu.Unlock()
		return &DiscoveryError{
			Type:      ErrorTypeInternal,
			Message:   "Circuit breaker is open",
			Retryable: false,
			Details: map[string]interface{}{
				"state": state,
			},
		}
	}

	// Execute the function
	cb.mu.Unlock()
	err := fn()

	// Record result atomically
	cb.recordResultAtomic(err)
	return err
}

// canExecuteLocked checks if the circuit breaker allows execution (assumes lock is held)
func (cb *CircuitBreaker) canExecuteLocked() bool {
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		return time.Since(cb.lastFailureTime) > cb.config.RecoveryTimeout
	case CircuitBreakerHalfOpen:
		return cb.failureCount < cb.config.HalfOpenLimit
	default:
		return false
	}
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.canExecuteLocked()
}

// recordResultAtomic records the result of an execution atomically
func (cb *CircuitBreaker) recordResultAtomic(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.state == CircuitBreakerClosed && cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitBreakerOpen
		} else if cb.state == CircuitBreakerHalfOpen {
			cb.state = CircuitBreakerOpen
		}
	} else {
		if cb.state == CircuitBreakerHalfOpen {
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
		}
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	cb.recordResultAtomic(err)
}

// getState returns the current circuit breaker state
func (cb *CircuitBreaker) getState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// RetryManager manages retry operations
type RetryManager struct {
	config *RetryConfig
}

// NewRetryManager creates a new retry manager
func NewRetryManager(config *RetryConfig) *RetryManager {
	if config == nil {
		config = &RetryConfig{
			MaxRetries:        3,
			InitialDelay:      1 * time.Second,
			MaxDelay:          30 * time.Second,
			BackoffMultiplier: 2.0,
			Jitter:            true,
			RetryableErrors: []ErrorType{
				ErrorTypeNetwork,
				ErrorTypeRateLimit,
				ErrorTypeTimeout,
			},
		}
	}
	return &RetryManager{config: config}
}

// ExecuteWithRetry executes a function with retry logic
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, fn func() ([]models.Resource, error)) ([]models.Resource, error) {
	var lastErr error
	delay := rm.config.InitialDelay

	for attempt := 0; attempt <= rm.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resources, err := fn()
		if err == nil {
			return resources, nil
		}

		lastErr = err

		// Check if error is retryable
		if discoveryErr, ok := err.(*DiscoveryError); ok {
			if !discoveryErr.IsRetryable() || !rm.isRetryableError(discoveryErr.Type) {
				return nil, err
			}
			discoveryErr.RetryCount = attempt
		} else {
			// For non-discovery errors, check if we should retry
			if !rm.shouldRetryGenericError(err) {
				return nil, err
			}
		}

		if attempt == rm.config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay = rm.calculateDelay(delay, attempt)

		log.Printf("Retry attempt %d/%d after %v: %v",
			attempt+1, rm.config.MaxRetries+1, delay, err)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryableError checks if an error type is retryable
func (rm *RetryManager) isRetryableError(errorType ErrorType) bool {
	for _, retryableType := range rm.config.RetryableErrors {
		if errorType == retryableType {
			return true
		}
	}
	return false
}

// shouldRetryGenericError checks if a generic error should be retried
func (rm *RetryManager) shouldRetryGenericError(err error) bool {
	errStr := strings.ToLower(err.Error())

	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"network is unreachable",
		"rate limit",
		"throttling",
		"temporary",
		"retry",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// calculateDelay calculates the delay for the next retry
func (rm *RetryManager) calculateDelay(currentDelay time.Duration, attempt int) time.Duration {
	delay := time.Duration(float64(currentDelay) * rm.config.BackoffMultiplier)

	if delay > rm.config.MaxDelay {
		delay = rm.config.MaxDelay
	}

	if rm.config.Jitter {
		// Add jitter to prevent thundering herd
		jitter := time.Duration(float64(delay) * 0.1) // 10% jitter
		delay += time.Duration(float64(jitter) * (0.5 + float64(attempt%10)/10.0))
	}

	return delay
}

// ErrorHandler provides comprehensive error handling
type ErrorHandler struct {
	retryManager    *RetryManager
	circuitBreakers map[string]*CircuitBreaker
	errorStats      map[ErrorType]int
	mu              sync.RWMutex
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(retryConfig *RetryConfig) *ErrorHandler {
	return &ErrorHandler{
		retryManager:    NewRetryManager(retryConfig),
		circuitBreakers: make(map[string]*CircuitBreaker),
		errorStats:      make(map[ErrorType]int),
	}
}

// HandleDiscoveryError handles a discovery error
func (eh *ErrorHandler) HandleDiscoveryError(err *DiscoveryError) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.errorStats[err.Type]++

	// Log the error with context
	log.Printf("[ERROR] Discovery failed: %s", err.Error())

	// Provide suggestions based on error type
	eh.provideSuggestions(err)
}

// provideSuggestions provides helpful suggestions for errors
func (eh *ErrorHandler) provideSuggestions(err *DiscoveryError) {
	switch err.Type {
	case ErrorTypeAuth:
		err.Suggestions = []string{
			"Check your cloud provider credentials",
			"Verify that your access keys are valid and not expired",
			"Ensure you have the necessary permissions for the requested resources",
			"Try running 'aws configure' or equivalent for your provider",
		}
	case ErrorTypePermission:
		err.Suggestions = []string{
			"Verify your IAM roles and permissions",
			"Check if your account has access to the requested region",
			"Ensure your service account has the necessary API permissions",
			"Contact your cloud administrator for access",
		}
	case ErrorTypeRateLimit:
		err.Suggestions = []string{
			"Reduce the number of concurrent requests",
			"Increase the delay between API calls",
			"Use exponential backoff for retries",
			"Consider using SDK instead of CLI for better rate limiting",
		}
	case ErrorTypeNetwork:
		err.Suggestions = []string{
			"Check your internet connection",
			"Verify your VPN connection if using one",
			"Check firewall settings",
			"Try again in a few minutes",
		}
	case ErrorTypeTimeout:
		err.Suggestions = []string{
			"Increase the timeout configuration",
			"Reduce the scope of discovery",
			"Check network latency to the cloud provider",
			"Try discovering resources in smaller batches",
		}
	}
}

// GetCircuitBreaker gets or creates a circuit breaker for a service
func (eh *ErrorHandler) GetCircuitBreaker(provider, region, service string) *CircuitBreaker {
	key := fmt.Sprintf("%s:%s:%s", provider, region, service)

	eh.mu.Lock()
	defer eh.mu.Unlock()

	if cb, exists := eh.circuitBreakers[key]; exists {
		return cb
	}

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  60 * time.Second,
		HalfOpenLimit:    3,
	})

	eh.circuitBreakers[key] = cb
	return cb
}

// GetErrorStats returns error statistics
func (eh *ErrorHandler) GetErrorStats() map[ErrorType]int {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	stats := make(map[ErrorType]int)
	for errorType, count := range eh.errorStats {
		stats[errorType] = count
	}
	return stats
}

// ResetErrorStats resets error statistics
func (eh *ErrorHandler) ResetErrorStats() {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	eh.errorStats = make(map[ErrorType]int)
}

// CreateDiscoveryError creates a new discovery error
func CreateDiscoveryError(errorType ErrorType, provider, region, service, message, context string) *DiscoveryError {
	return &DiscoveryError{
		Type:       errorType,
		Provider:   provider,
		Region:     region,
		Service:    service,
		Message:    message,
		Context:    context,
		Retryable:  isRetryableErrorType(errorType),
		MaxRetries: 3,
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
}

// isRetryableErrorType determines if an error type is retryable
func isRetryableErrorType(errorType ErrorType) bool {
	retryableTypes := []ErrorType{
		ErrorTypeNetwork,
		ErrorTypeRateLimit,
		ErrorTypeTimeout,
	}

	for _, retryableType := range retryableTypes {
		if errorType == retryableType {
			return true
		}
	}
	return false
}

// EnhancedErrorReporting provides detailed error reporting
type EnhancedErrorReporting struct {
	errors     []*DiscoveryError
	errorStats map[ErrorType]*ErrorStats
	mu         sync.RWMutex
}

// ErrorStats contains statistics for a specific error type
type ErrorStats struct {
	Count       int                    `json:"count"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	Providers   map[string]int         `json:"providers"`
	Regions     map[string]int         `json:"regions"`
	Services    map[string]int         `json:"services"`
	Suggestions []string               `json:"suggestions"`
	Details     map[string]interface{} `json:"details"`
}

// NewEnhancedErrorReporting creates a new error reporting system
func NewEnhancedErrorReporting() *EnhancedErrorReporting {
	return &EnhancedErrorReporting{
		errors:     make([]*DiscoveryError, 0),
		errorStats: make(map[ErrorType]*ErrorStats),
	}
}

// ReportError reports an error for analysis
func (eer *EnhancedErrorReporting) ReportError(err *DiscoveryError) {
	eer.mu.Lock()
	defer eer.mu.Unlock()

	eer.errors = append(eer.errors, err)

	// Update statistics
	if stats, exists := eer.errorStats[err.Type]; exists {
		stats.Count++
		stats.LastSeen = err.Timestamp
	} else {
		eer.errorStats[err.Type] = &ErrorStats{
			Count:       1,
			FirstSeen:   err.Timestamp,
			LastSeen:    err.Timestamp,
			Providers:   make(map[string]int),
			Regions:     make(map[string]int),
			Services:    make(map[string]int),
			Suggestions: err.Suggestions,
			Details:     err.Details,
		}
	}

	stats := eer.errorStats[err.Type]
	stats.Providers[err.Provider]++
	stats.Regions[err.Region]++
	stats.Services[err.Service]++
}

// GetErrorReport returns a comprehensive error report
func (eer *EnhancedErrorReporting) GetErrorReport() map[string]interface{} {
	eer.mu.RLock()
	defer eer.mu.RUnlock()

	report := map[string]interface{}{
		"total_errors": len(eer.errors),
		"error_stats":  eer.errorStats,
		"recent_errors": func() []*DiscoveryError {
			if len(eer.errors) > 10 {
				return eer.errors[len(eer.errors)-10:]
			}
			return eer.errors
		}(),
	}

	return report
}

// GetErrorSuggestions returns suggestions for resolving errors
func (eer *EnhancedErrorReporting) GetErrorSuggestions() []string {
	eer.mu.RLock()
	defer eer.mu.RUnlock()

	suggestions := make([]string, 0)
	seen := make(map[string]bool)

	for _, stats := range eer.errorStats {
		for _, suggestion := range stats.Suggestions {
			if !seen[suggestion] {
				suggestions = append(suggestions, suggestion)
				seen[suggestion] = true
			}
		}
	}

	return suggestions
}

// ClearErrors clears all reported errors
func (eer *EnhancedErrorReporting) ClearErrors() {
	eer.mu.Lock()
	defer eer.mu.Unlock()

	eer.errors = make([]*DiscoveryError, 0)
	eer.errorStats = make(map[ErrorType]*ErrorStats)
}
