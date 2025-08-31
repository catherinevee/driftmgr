package resilience

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
)

// RetryWithBreaker combines retry logic with circuit breaker
type RetryWithBreaker struct {
	retryPolicy    *RetryPolicy
	circuitBreaker *CircuitBreaker
	logger         *logging.Logger
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          bool
	RetryableErrors func(error) bool
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableErrors: func(err error) bool {
			// Default: retry on any error except context cancellation
			return !errors.Is(err, context.Canceled)
		},
	}
}

// NewRetryWithBreaker creates a new retry with circuit breaker
func NewRetryWithBreaker(name string, retryPolicy *RetryPolicy, breakerConfig *CircuitBreakerConfig) *RetryWithBreaker {
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}

	return &RetryWithBreaker{
		retryPolicy:    retryPolicy,
		circuitBreaker: NewCircuitBreaker(breakerConfig),
		logger:         logging.GetLogger(),
	}
}

// Execute executes a function with retry and circuit breaker protection
func (r *RetryWithBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	return r.ExecuteWithMetadata(ctx, fn, nil)
}

// ExecuteWithMetadata executes with additional metadata for logging
func (r *RetryWithBreaker) ExecuteWithMetadata(ctx context.Context, fn func(context.Context) error, metadata map[string]interface{}) error {
	var lastErr error
	
	for attempt := 1; attempt <= r.retryPolicy.MaxAttempts; attempt++ {
		// Check circuit breaker
		err := r.circuitBreaker.ExecuteContext(ctx, func(execCtx context.Context) error {
			return fn(execCtx)
		})

		if err == nil {
			// Success
			if attempt > 1 {
				r.logger.Info("Operation succeeded after retry", map[string]interface{}{
					"attempt":  attempt,
					"metadata": metadata,
				})
			}
			return nil
		}

		lastErr = err

		// Check if error is circuit breaker open
		if errors.Is(err, ErrCircuitOpen) {
			r.logger.Warn("Circuit breaker is open", map[string]interface{}{
				"error":    err.Error(),
				"metadata": metadata,
			})
			return err
		}

		// Check if we should retry
		if !r.retryPolicy.RetryableErrors(err) {
			r.logger.Debug("Error is not retryable", map[string]interface{}{
				"error":    err.Error(),
				"metadata": metadata,
			})
			return err
		}

		// Check if we've exhausted retries
		if attempt >= r.retryPolicy.MaxAttempts {
			r.logger.Error("Max retries exhausted", map[string]interface{}{
				"attempts": attempt,
				"error":    err.Error(),
				"metadata": metadata,
			})
			break
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		r.logger.Warn("Operation failed, retrying", map[string]interface{}{
			"attempt":     attempt,
			"next_attempt": attempt + 1,
			"delay":       delay.String(),
			"error":       err.Error(),
			"metadata":    metadata,
		})

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", r.retryPolicy.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay before next retry
func (r *RetryWithBreaker) calculateDelay(attempt int) time.Duration {
	delay := float64(r.retryPolicy.InitialDelay) * math.Pow(r.retryPolicy.Multiplier, float64(attempt-1))
	
	// Cap at max delay
	if delay > float64(r.retryPolicy.MaxDelay) {
		delay = float64(r.retryPolicy.MaxDelay)
	}

	// Add jitter if configured
	if r.retryPolicy.Jitter {
		jitter := rand.Float64() * 0.3 * delay // Up to 30% jitter
		delay = delay + jitter
	}

	return time.Duration(delay)
}

// GetCircuitBreaker returns the underlying circuit breaker
func (r *RetryWithBreaker) GetCircuitBreaker() *CircuitBreaker {
	return r.circuitBreaker
}

// GetMetrics returns current metrics
func (r *RetryWithBreaker) GetMetrics() map[string]interface{} {
	cbMetrics := r.circuitBreaker.GetMetrics()
	return map[string]interface{}{
		"circuit_breaker": cbMetrics,
		"retry_policy": map[string]interface{}{
			"max_attempts":  r.retryPolicy.MaxAttempts,
			"initial_delay": r.retryPolicy.InitialDelay.String(),
			"max_delay":     r.retryPolicy.MaxDelay.String(),
			"multiplier":    r.retryPolicy.Multiplier,
		},
	}
}

// BulkheadExecutor provides bulkhead pattern for limiting concurrent executions
type BulkheadExecutor struct {
	name       string
	maxConcurrent int
	timeout    time.Duration
	semaphore  chan struct{}
	logger     *logging.Logger
}

// NewBulkheadExecutor creates a new bulkhead executor
func NewBulkheadExecutor(name string, maxConcurrent int, timeout time.Duration) *BulkheadExecutor {
	return &BulkheadExecutor{
		name:          name,
		maxConcurrent: maxConcurrent,
		timeout:       timeout,
		semaphore:     make(chan struct{}, maxConcurrent),
		logger:        logging.GetLogger(),
	}
}

// Execute executes a function with bulkhead protection
func (b *BulkheadExecutor) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Try to acquire semaphore
	select {
	case b.semaphore <- struct{}{}:
		// Acquired
		defer func() { <-b.semaphore }()
		
		// Execute with timeout
		execCtx, cancel := context.WithTimeout(ctx, b.timeout)
		defer cancel()
		
		return fn(execCtx)
		
	case <-ctx.Done():
		return ctx.Err()
		
	case <-time.After(100 * time.Millisecond):
		// Quick timeout for acquiring semaphore
		return fmt.Errorf("bulkhead full: max concurrent executions (%d) reached", b.maxConcurrent)
	}
}

// ResilientExecutor combines all resilience patterns
type ResilientExecutor struct {
	name         string
	retry        *RetryWithBreaker
	bulkhead     *BulkheadExecutor
	rateLimiter  *RateLimiter
	logger       *logging.Logger
}

// ResilientExecutorConfig configures a resilient executor
type ResilientExecutorConfig struct {
	Name             string
	RetryPolicy      *RetryPolicy
	CircuitBreaker   *CircuitBreakerConfig
	MaxConcurrent    int
	ExecutionTimeout time.Duration
	RateLimit        int // requests per second
}

// NewResilientExecutor creates a new resilient executor with all patterns
func NewResilientExecutor(config *ResilientExecutorConfig) *ResilientExecutor {
	return &ResilientExecutor{
		name: config.Name,
		retry: NewRetryWithBreaker(
			config.Name,
			config.RetryPolicy,
			config.CircuitBreaker,
		),
		bulkhead: NewBulkheadExecutor(
			config.Name,
			config.MaxConcurrent,
			config.ExecutionTimeout,
		),
		rateLimiter: NewRateLimiter(config.RateLimit, config.RateLimit*2),
		logger:      logging.GetLogger(),
	}
}

// Execute executes with all resilience patterns
func (r *ResilientExecutor) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Apply rate limiting
	if err := r.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	}

	// Apply bulkhead pattern
	return r.bulkhead.Execute(ctx, func(bulkheadCtx context.Context) error {
		// Apply retry with circuit breaker
		return r.retry.Execute(bulkheadCtx, fn)
	})
}

// GetMetrics returns metrics from all components
func (r *ResilientExecutor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"name":         r.name,
		"retry":        r.retry.GetMetrics(),
		"rate_limiter": r.rateLimiter.GetMetrics(),
	}
}

// Common error types for retry decisions
var (
	ErrTemporary = errors.New("temporary error")
	ErrThrottled = errors.New("request throttled")
	ErrTimeout   = errors.New("operation timeout")
)

// IsRetryable determines if an error should trigger a retry
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	if errors.Is(err, ErrTemporary) ||
		errors.Is(err, ErrThrottled) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check error string for common retryable patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary",
		"throttled",
		"rate limit",
		"too many requests",
		"service unavailable",
		"gateway timeout",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > len(substr) && 
		 containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}