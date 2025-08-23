package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	Jitter         bool
	RetryableError func(error) bool
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableError: func(err error) bool {
			// Default: retry on any error
			return err != nil
		},
	}
}

// CloudAPIConfig returns retry config optimized for cloud APIs
func CloudAPIConfig() *Config {
	return &Config{
		MaxRetries:   5,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     60 * time.Second,
		Multiplier:   1.5,
		Jitter:       true,
		RetryableError: func(err error) bool {
			// Retry on rate limiting, timeouts, and temporary errors
			if err == nil {
				return false
			}
			errStr := err.Error()
			return contains(errStr, "rate limit") ||
				contains(errStr, "timeout") ||
				contains(errStr, "temporary") ||
				contains(errStr, "throttl") ||
				contains(errStr, "429") ||
				contains(errStr, "503") ||
				contains(errStr, "504")
		},
	}
}

// Do executes a function with retry logic
func Do(ctx context.Context, config *Config, fn func() error) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !config.RetryableError(err) {
			return err
		}

		// Don't retry if we've exhausted attempts
		if attempt == config.MaxRetries {
			break
		}

		// Calculate next delay with exponential backoff
		if attempt > 0 {
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}

		// Add jitter if configured
		actualDelay := delay
		if config.Jitter {
			jitter := time.Duration(rand.Float64() * float64(delay) * 0.3)
			actualDelay = delay + jitter
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(actualDelay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, lastErr)
}

// DoWithResult executes a function that returns a value with retry logic
func DoWithResult[T any](ctx context.Context, config *Config, fn func() (T, error)) (T, error) {
	var result T
	err := Do(ctx, config, func() error {
		var fnErr error
		result, fnErr = fn()
		return fnErr
	})
	return result, err
}

// CircuitBreaker provides circuit breaker pattern
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	failureCount    int
	lastFailureTime time.Time
	state           string // "closed", "open", "half-open"
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	// Check if circuit is open
	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = "half-open"
			cb.failureCount = 0
		} else {
			return errors.New("circuit breaker is open")
		}
	}

	// Execute the function
	err := fn()

	if err == nil {
		// Success - reset failure count
		if cb.state == "half-open" {
			cb.state = "closed"
		}
		cb.failureCount = 0
		return nil
	}

	// Failure - increment count
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.maxFailures {
		cb.state = "open"
		return fmt.Errorf("circuit breaker opened after %d failures: %w", cb.maxFailures, err)
	}

	return err
}

// Helper function for string contains
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Backoff calculates the backoff duration for a given attempt
func Backoff(attempt int, config *Config) time.Duration {
	if config == nil {
		config = DefaultConfig()
	}

	delay := config.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
			break
		}
	}

	if config.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.3)
		delay += jitter
	}

	return delay
}
