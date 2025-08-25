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

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          bool
	RetryableErrors []error
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// CloudProviderRetryConfig returns config optimized for cloud providers
func CloudProviderRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 2 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func(ctx context.Context) error

// RetryResult contains the outcome of a retry operation
type RetryResult struct {
	Attempts      int
	LastError     error
	Success       bool
	TotalDuration time.Duration
}

// Retry executes a function with exponential backoff
func Retry(ctx context.Context, config *RetryConfig, fn RetryableFunc) (*RetryResult, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}

	logger := logging.GetLogger()
	startTime := time.Now()
	result := &RetryResult{}

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result.Attempts = attempt

		// Execute the function
		err := fn(ctx)

		if err == nil {
			// Success!
			result.Success = true
			result.TotalDuration = time.Since(startTime)

			if attempt > 1 {
				logger.Info("Operation succeeded after retry", map[string]interface{}{
					"attempt":  attempt,
					"duration": result.TotalDuration.String(),
				})
			}

			return result, nil
		}

		result.LastError = err

		// Check if error is retryable
		if !isRetryable(err, config.RetryableErrors) {
			logger.Warn("Non-retryable error encountered", map[string]interface{}{
				"error":   err.Error(),
				"attempt": attempt,
			})
			result.TotalDuration = time.Since(startTime)
			return result, err
		}

		// Check if we've exhausted attempts
		if attempt >= config.MaxAttempts {
			logger.Error("Max retry attempts reached", err, map[string]interface{}{
				"attempts": attempt,
				"duration": time.Since(startTime).String(),
			})
			result.TotalDuration = time.Since(startTime)
			return result, fmt.Errorf("operation failed after %d attempts: %w", attempt, err)
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(attempt, config)

		logger.Debug("Retrying operation", map[string]interface{}{
			"attempt":    attempt,
			"next_delay": delay.String(),
			"error":      err.Error(),
		})

		// Wait before retrying
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			// Context cancelled
			result.TotalDuration = time.Since(startTime)
			return result, ctx.Err()
		}
	}

	result.TotalDuration = time.Since(startTime)
	return result, result.LastError
}

// RetryWithTimeout executes a function with retry and overall timeout
func RetryWithTimeout(timeout time.Duration, config *RetryConfig, fn RetryableFunc) (*RetryResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return Retry(ctx, config, fn)
}

// calculateDelay computes the delay for the next retry attempt
func calculateDelay(attempt int, config *RetryConfig) time.Duration {
	// Calculate exponential backoff
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt-1))

	// Apply max delay cap
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	if config.Jitter {
		jitter := rand.Float64() * 0.3 * delay // Up to 30% jitter
		delay = delay + jitter
	}

	return time.Duration(delay)
}

// isRetryable determines if an error should trigger a retry
func isRetryable(err error, retryableErrors []error) bool {
	// If no specific errors configured, retry all errors
	if len(retryableErrors) == 0 {
		return true
	}

	// Check if error matches any retryable errors
	for _, retryableErr := range retryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	// Check for common retryable conditions
	errStr := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"TooManyRequests",
		"rate limit",
		"throttled",
		"temporary",
		"503",
		"429",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > 0 && len(substr) > 0 &&
				contains(toLowerCase(s), toLowerCase(substr)))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// RetryOperation wraps common cloud operations with retry logic
type RetryOperation struct {
	Name     string
	Provider string
	Config   *RetryConfig
	Logger   *logging.Logger
}

// NewRetryOperation creates a new retry operation
func NewRetryOperation(name, provider string) *RetryOperation {
	return &RetryOperation{
		Name:     name,
		Provider: provider,
		Config:   CloudProviderRetryConfig(),
		Logger:   logging.GetLogger(),
	}
}

// Execute runs the operation with retry logic and logging
func (r *RetryOperation) Execute(ctx context.Context, fn RetryableFunc) error {
	r.Logger.Info("Starting operation", map[string]interface{}{
		"operation": r.Name,
		"provider":  r.Provider,
	})

	startTime := time.Now()

	result, err := Retry(ctx, r.Config, func(ctx context.Context) error {
		opErr := fn(ctx)
		if opErr != nil {
			r.Logger.Debug("Operation attempt failed", map[string]interface{}{
				"operation": r.Name,
				"provider":  r.Provider,
				"error":     opErr.Error(),
			})
		}
		return opErr
	})

	duration := time.Since(startTime)

	if err != nil {
		r.Logger.Error("Operation failed", err, map[string]interface{}{
			"operation": r.Name,
			"provider":  r.Provider,
			"attempts":  result.Attempts,
			"duration":  duration.String(),
		})

		// Record metric
		logging.Metric("operation.failure", float64(result.Attempts), "attempts", map[string]string{
			"operation": r.Name,
			"provider":  r.Provider,
		})

		return err
	}

	r.Logger.Info("Operation completed successfully", map[string]interface{}{
		"operation": r.Name,
		"provider":  r.Provider,
		"attempts":  result.Attempts,
		"duration":  duration.String(),
	})

	// Record success metric
	logging.Metric("operation.success", duration.Seconds(), "seconds", map[string]string{
		"operation": r.Name,
		"provider":  r.Provider,
		"attempts":  fmt.Sprintf("%d", result.Attempts),
	})

	return nil
}
