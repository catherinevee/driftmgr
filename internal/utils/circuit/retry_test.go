package resilience

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/utils/errors"
)

func TestRetry(t *testing.T) {
	t.Run("successful on first attempt", func(t *testing.T) {
		attempts := 0
		err := Retry(context.Background(), nil, func() error {
			attempts++
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("successful after retries", func(t *testing.T) {
		attempts := 0
		err := Retry(context.Background(), &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       false,
		}, func() error {
			attempts++
			if attempts < 3 {
				return errors.New(errors.ErrorTypeNetwork, "network error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("fails after max attempts", func(t *testing.T) {
		attempts := 0
		err := Retry(context.Background(), &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       false,
		}, func() error {
			attempts++
			return errors.New(errors.ErrorTypeNetwork, "network error")
		})

		if err == nil {
			t.Error("expected error, got nil")
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}

		if !errors.Is(err, errors.ErrorTypeInternal) {
			t.Errorf("expected internal error type, got %v", errors.GetType(err))
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		attempts := 0
		err := Retry(context.Background(), nil, func() error {
			attempts++
			return errors.New(errors.ErrorTypeValidation, "validation error")
		})

		if err == nil {
			t.Error("expected error, got nil")
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
		}

		if !errors.Is(err, errors.ErrorTypeValidation) {
			t.Errorf("expected validation error type, got %v", errors.GetType(err))
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := Retry(ctx, &RetryConfig{
			MaxAttempts:  5,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Jitter:       false,
		}, func() error {
			attempts++
			return errors.New(errors.ErrorTypeNetwork, "network error")
		})

		if err == nil {
			t.Error("expected error due to context cancellation")
		}

		if attempts >= 5 {
			t.Errorf("expected fewer than 5 attempts due to cancellation, got %d", attempts)
		}

		if !errors.Is(err, errors.ErrorTypeTimeout) {
			t.Errorf("expected timeout error type, got %v", errors.GetType(err))
		}
	})

	t.Run("exponential backoff", func(t *testing.T) {
		attempts := 0
		start := time.Now()

		err := Retry(context.Background(), &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       false,
		}, func() error {
			attempts++
			if attempts < 3 {
				return errors.New(errors.ErrorTypeNetwork, "network error")
			}
			return nil
		})

		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// With initial delay of 10ms and multiplier of 2:
		// Attempt 1: immediate
		// Attempt 2: 10ms delay
		// Attempt 3: 20ms delay
		// Total expected delay: 30ms (plus some execution time)
		if elapsed < 25*time.Millisecond {
			t.Errorf("expected at least 25ms elapsed time, got %v", elapsed)
		}
	})

	t.Run("jitter", func(t *testing.T) {
		// Run multiple times to test jitter randomness
		for i := 0; i < 5; i++ {
			attempts := 0
			start := time.Now()

			_ = Retry(context.Background(), &RetryConfig{
				MaxAttempts:  2,
				InitialDelay: 10 * time.Millisecond,
				MaxDelay:     100 * time.Millisecond,
				Multiplier:   1.0,
				Jitter:       true,
			}, func() error {
				attempts++
				if attempts < 2 {
					return errors.New(errors.ErrorTypeNetwork, "network error")
				}
				return nil
			})

			elapsed := time.Since(start)

			// With jitter, the delay should be between 10ms and 13ms (10ms + 30% jitter)
			if elapsed < 8*time.Millisecond || elapsed > 15*time.Millisecond {
				// Allow some variance for execution time
				t.Logf("Iteration %d: elapsed time %v may be outside expected jitter range", i, elapsed)
			}
		}
	})

	t.Run("custom retryable errors", func(t *testing.T) {
		attempts := 0
		err := Retry(context.Background(), &RetryConfig{
			MaxAttempts:     3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			Multiplier:      2.0,
			Jitter:          false,
			RetryableErrors: []errors.ErrorType{errors.ErrorTypeProvider},
		}, func() error {
			attempts++
			if attempts < 2 {
				return errors.New(errors.ErrorTypeProvider, "provider error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}

func TestRetryWithBackoff(t *testing.T) {
	attempts := 0
	err := RetryWithBackoff(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New(errors.ErrorTypeNetwork, "network error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestExponentialBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		base     time.Duration
		max      time.Duration
		expected time.Duration
	}{
		{0, 100 * time.Millisecond, 10 * time.Second, 100 * time.Millisecond},
		{1, 100 * time.Millisecond, 10 * time.Second, 100 * time.Millisecond},
		{2, 100 * time.Millisecond, 10 * time.Second, 200 * time.Millisecond},
		{3, 100 * time.Millisecond, 10 * time.Second, 400 * time.Millisecond},
		{4, 100 * time.Millisecond, 10 * time.Second, 800 * time.Millisecond},
		{10, 100 * time.Millisecond, 1 * time.Second, 1 * time.Second}, // Max delay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			result := ExponentialBackoff(tt.attempt, tt.base, tt.max)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLinearBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		base     time.Duration
		max      time.Duration
		expected time.Duration
	}{
		{1, 100 * time.Millisecond, 10 * time.Second, 100 * time.Millisecond},
		{2, 100 * time.Millisecond, 10 * time.Second, 200 * time.Millisecond},
		{3, 100 * time.Millisecond, 10 * time.Second, 300 * time.Millisecond},
		{10, 100 * time.Millisecond, 500 * time.Millisecond, 500 * time.Millisecond}, // Max delay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			result := LinearBackoff(tt.attempt, tt.base, tt.max)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Benchmarks

func BenchmarkRetry(b *testing.B) {
	ctx := context.Background()
	cfg := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		_ = Retry(ctx, cfg, func() error {
			attempts++
			if attempts < 2 {
				return errors.New(errors.ErrorTypeNetwork, "network error")
			}
			return nil
		})
	}
}

func BenchmarkExponentialBackoff(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ExponentialBackoff(5, 100*time.Millisecond, 10*time.Second)
	}
}