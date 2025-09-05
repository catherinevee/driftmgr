package errors

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RecoveryExecutor handles error recovery strategies
type RecoveryExecutor struct {
	strategies map[string]RecoveryFunc
	metrics    *RecoveryMetrics
}

// RecoveryFunc is a function that attempts to recover from an error
type RecoveryFunc func(ctx context.Context, err *DriftError) error

// RecoveryMetrics tracks recovery attempts and success rates
type RecoveryMetrics struct {
	Attempts    int64
	Successes   int64
	Failures    int64
	LastAttempt time.Time
}

// NewRecoveryExecutor creates a new recovery executor
func NewRecoveryExecutor() *RecoveryExecutor {
	executor := &RecoveryExecutor{
		strategies: make(map[string]RecoveryFunc),
		metrics:    &RecoveryMetrics{},
	}

	// Register default strategies
	executor.RegisterStrategy("retry", executor.retryStrategy)
	executor.RegisterStrategy("retry_with_backoff", executor.retryWithBackoffStrategy)
	executor.RegisterStrategy("circuit_breaker", executor.circuitBreakerStrategy)
	executor.RegisterStrategy("fallback", executor.fallbackStrategy)
	executor.RegisterStrategy("compensate", executor.compensateStrategy)

	return executor
}

// RegisterStrategy registers a custom recovery strategy
func (r *RecoveryExecutor) RegisterStrategy(name string, fn RecoveryFunc) {
	r.strategies[name] = fn
}

// Execute attempts to recover from an error
func (r *RecoveryExecutor) Execute(ctx context.Context, err *DriftError) error {
	r.metrics.Attempts++
	r.metrics.LastAttempt = time.Now()

	// No recovery strategy defined
	if err.Recovery.Strategy == "" {
		return err
	}

	// Find and execute strategy
	strategy, exists := r.strategies[err.Recovery.Strategy]
	if !exists {
		return fmt.Errorf("unknown recovery strategy: %s", err.Recovery.Strategy)
	}

	// Execute recovery
	if recoveryErr := strategy(ctx, err); recoveryErr != nil {
		r.metrics.Failures++
		return recoveryErr
	}

	r.metrics.Successes++
	return nil
}

// retryStrategy implements simple retry logic
func (r *RecoveryExecutor) retryStrategy(ctx context.Context, err *DriftError) error {
	if !err.Retryable {
		return err
	}

	// Wait for retry delay
	if err.RetryAfter > 0 {
		select {
		case <-time.After(err.RetryAfter):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// In real implementation, this would retry the original operation
	// For now, we just return nil to indicate recovery succeeded
	return nil
}

// retryWithBackoffStrategy implements exponential backoff retry
func (r *RecoveryExecutor) retryWithBackoffStrategy(ctx context.Context, err *DriftError) error {
	maxRetries := 3
	if val, ok := err.Recovery.Params["max_retries"].(int); ok {
		maxRetries = val
	}

	baseDelay := 1 * time.Second
	if val, ok := err.Recovery.Params["base_delay"].(string); ok {
		if d, err := time.ParseDuration(val); err == nil {
			baseDelay = d
		}
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Calculate backoff delay
		delay := time.Duration(math.Pow(2, float64(attempt-1))) * baseDelay

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}

		// In real implementation, retry the operation here
		// For demonstration, we simulate success on last attempt
		if attempt == maxRetries {
			return nil
		}
	}

	return err
}

// circuitBreakerStrategy implements circuit breaker pattern
func (r *RecoveryExecutor) circuitBreakerStrategy(ctx context.Context, err *DriftError) error {
	// Check circuit breaker state
	threshold := 5
	if val, ok := err.Recovery.Params["threshold"].(int); ok {
		threshold = val
	}

	// If too many recent failures, circuit is open
	if r.metrics.Failures > int64(threshold) {
		return fmt.Errorf("circuit breaker open: too many failures")
	}

	// Try operation
	// In real implementation, this would attempt the operation
	return nil
}

// fallbackStrategy implements fallback to alternative
func (r *RecoveryExecutor) fallbackStrategy(ctx context.Context, err *DriftError) error {
	fallbackFn, ok := err.Recovery.Params["fallback_function"].(func() error)
	if !ok {
		return fmt.Errorf("no fallback function provided")
	}

	return fallbackFn()
}

// compensateStrategy implements compensation logic
func (r *RecoveryExecutor) compensateStrategy(ctx context.Context, err *DriftError) error {
	// Execute compensation steps
	for _, step := range err.Recovery.Steps {
		fmt.Printf("Executing compensation step: %s\n", step)
		// In real implementation, execute actual compensation logic
	}

	return nil
}

// RetryOperation wraps an operation with retry logic
func RetryOperation(ctx context.Context, operation func() error, maxRetries int, delay time.Duration) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if driftErr, ok := err.(*DriftError); ok && !driftErr.Retryable {
			return err
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// RetryWithExponentialBackoff retries with exponential backoff
func RetryWithExponentialBackoff(ctx context.Context, operation func() error, config BackoffConfig) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * config.BaseDelay
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if driftErr, ok := err.(*DriftError); ok && !driftErr.Retryable {
			return err
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, lastErr)
}

// BackoffConfig configures exponential backoff
type BackoffConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// DefaultBackoffConfig returns default backoff configuration
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}
