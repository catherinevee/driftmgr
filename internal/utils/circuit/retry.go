package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/catherinevee/driftmgr/internal/utils/errors"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts     int                `json:"max_attempts" yaml:"max_attempts"`
	InitialDelay    time.Duration      `json:"initial_delay" yaml:"initial_delay"`
	MaxDelay        time.Duration      `json:"max_delay" yaml:"max_delay"`
	Multiplier      float64            `json:"multiplier" yaml:"multiplier"`
	Jitter          bool               `json:"jitter" yaml:"jitter"`
	RetryableErrors []errors.ErrorType `json:"retryable_errors" yaml:"retryable_errors"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableErrors: []errors.ErrorType{
			errors.ErrorTypeTimeout,
			errors.ErrorTypeRateLimit,
			errors.ErrorTypeNetwork,
		},
	}
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, cfg *RetryConfig, fn func() error) error {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}

	logger := logging.FromContext(ctx).With().
		Str("component", "retry").
		Int("max_attempts", cfg.MaxAttempts).
		Logger()

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), errors.ErrorTypeTimeout, "context cancelled during retry")
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info().
					Int("attempt", attempt).
					Msg("retry successful")
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err, cfg.RetryableErrors) {
			logger.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("non-retryable error encountered")
			return err
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts {
			// Apply jitter if configured
			actualDelay := delay
			if cfg.Jitter {
				jitter := time.Duration(rand.Float64() * float64(delay) * 0.3)
				actualDelay = delay + jitter
			}

			logger.Warn().
				Err(err).
				Int("attempt", attempt).
				Dur("delay", actualDelay).
				Msg("retrying after delay")

			// Sleep with context cancellation support
			select {
			case <-time.After(actualDelay):
			case <-ctx.Done():
				return errors.Wrap(ctx.Err(), errors.ErrorTypeTimeout, "context cancelled during retry delay")
			}

			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return errors.Wrapf(lastErr, errors.ErrorTypeInternal,
		"operation failed after %d attempts", cfg.MaxAttempts)
}

// RetryWithBackoff is a convenience function for exponential backoff retry
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return Retry(ctx, DefaultRetryConfig(), fn)
}

// isRetryable checks if an error is retryable
func isRetryable(err error, retryableTypes []errors.ErrorType) bool {
	// Check if error implements retryable interface
	if errors.IsRetryable(err) {
		return true
	}

	// Check against configured retryable error types
	errType := errors.GetType(err)
	for _, t := range retryableTypes {
		if errType == t {
			return true
		}
	}

	return false
}

// ExponentialBackoff calculates exponential backoff delay
func ExponentialBackoff(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}

	delay := time.Duration(math.Pow(2, float64(attempt-1))) * baseDelay
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

// LinearBackoff calculates linear backoff delay
func LinearBackoff(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	delay := time.Duration(attempt) * baseDelay
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}
