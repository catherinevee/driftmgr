package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
)

// RateLimiter provides rate limiting for API calls
type RateLimiter struct {
	mu           sync.Mutex
	tokens       int
	maxTokens    int
	refillRate   int
	refillPeriod time.Duration
	lastRefill   time.Time
	waiting      []chan struct{}
	logger       *logging.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate int, refillPeriod time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:       maxTokens,
		maxTokens:    maxTokens,
		refillRate:   refillRate,
		refillPeriod: refillPeriod,
		lastRefill:   time.Now(),
		waiting:      make([]chan struct{}, 0),
		logger:       logging.GetLogger(),
	}

	// Start refill goroutine
	go rl.refillRoutine()

	return rl
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.WaitN(ctx, 1)
}

// WaitN blocks until n tokens are available
func (rl *RateLimiter) WaitN(ctx context.Context, n int) error {
	if n > rl.maxTokens {
		return fmt.Errorf("requested %d tokens, but max is %d", n, rl.maxTokens)
	}

	// Try to acquire immediately
	if rl.tryAcquire(n) {
		return nil
	}

	// Create wait channel
	wait := make(chan struct{})

	rl.mu.Lock()
	rl.waiting = append(rl.waiting, wait)
	rl.mu.Unlock()

	// Wait for token or context cancellation
	select {
	case <-wait:
		return rl.WaitN(ctx, n) // Retry after being notified
	case <-ctx.Done():
		// Remove from waiting list
		rl.mu.Lock()
		for i, w := range rl.waiting {
			if w == wait {
				rl.waiting = append(rl.waiting[:i], rl.waiting[i+1:]...)
				break
			}
		}
		rl.mu.Unlock()
		return ctx.Err()
	}
}

// tryAcquire attempts to acquire n tokens without blocking
func (rl *RateLimiter) tryAcquire(n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on elapsed time
	rl.refill()

	if rl.tokens >= n {
		rl.tokens -= n

		rl.logger.Debug("Rate limiter tokens acquired", map[string]interface{}{
			"acquired":  n,
			"remaining": rl.tokens,
		})

		return true
	}

	return false
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	if elapsed >= rl.refillPeriod {
		// Calculate tokens to add
		periods := int(elapsed / rl.refillPeriod)
		tokensToAdd := periods * rl.refillRate

		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}

		rl.lastRefill = now

		// Notify waiting goroutines
		for _, wait := range rl.waiting {
			select {
			case wait <- struct{}{}:
			default:
			}
		}
		rl.waiting = rl.waiting[:0]
	}
}

// refillRoutine periodically refills tokens
func (rl *RateLimiter) refillRoutine() {
	ticker := time.NewTicker(rl.refillPeriod)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		rl.refill()
		rl.mu.Unlock()
	}
}

// Available returns the current number of available tokens
func (rl *RateLimiter) Available() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()
	return rl.tokens
}

// ProviderRateLimits defines rate limits for each cloud provider
type ProviderRateLimits struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	logger   *logging.Logger
}

// NewProviderRateLimits creates rate limiters for all providers
func NewProviderRateLimits() *ProviderRateLimits {
	prl := &ProviderRateLimits{
		limiters: make(map[string]*RateLimiter),
		logger:   logging.GetLogger(),
	}

	// Configure provider-specific limits
	// AWS: 10 requests per second
	prl.limiters["aws"] = NewRateLimiter(10, 10, 1*time.Second)

	// Azure: 12 requests per second
	prl.limiters["azure"] = NewRateLimiter(12, 12, 1*time.Second)

	// GCP: 10 requests per second
	prl.limiters["gcp"] = NewRateLimiter(10, 10, 1*time.Second)

	// DigitalOcean: 5 requests per second
	prl.limiters["digitalocean"] = NewRateLimiter(5, 5, 1*time.Second)

	// Default limiter for unknown providers
	prl.limiters["default"] = NewRateLimiter(5, 5, 1*time.Second)

	return prl
}

// Wait waits for rate limit clearance for a provider
func (prl *ProviderRateLimits) Wait(ctx context.Context, provider string) error {
	prl.mu.RLock()
	limiter, exists := prl.limiters[provider]
	if !exists {
		limiter = prl.limiters["default"]
	}
	prl.mu.RUnlock()

	start := time.Now()
	err := limiter.Wait(ctx)

	if err == nil {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			prl.logger.Warn("Rate limit delay", map[string]interface{}{
				"provider": provider,
				"delay":    duration.String(),
			})
		}

		// Record metric
		logging.Metric("rate_limit.wait", duration.Seconds(), "seconds", map[string]string{
			"provider": provider,
		})
	}

	return err
}

// WaitN waits for n tokens for a provider
func (prl *ProviderRateLimits) WaitN(ctx context.Context, provider string, n int) error {
	prl.mu.RLock()
	limiter, exists := prl.limiters[provider]
	if !exists {
		limiter = prl.limiters["default"]
	}
	prl.mu.RUnlock()

	return limiter.WaitN(ctx, n)
}

// Available returns available tokens for a provider
func (prl *ProviderRateLimits) Available(provider string) int {
	prl.mu.RLock()
	limiter, exists := prl.limiters[provider]
	if !exists {
		limiter = prl.limiters["default"]
	}
	prl.mu.RUnlock()

	return limiter.Available()
}

// SetLimit updates the rate limit for a provider
func (prl *ProviderRateLimits) SetLimit(provider string, maxTokens int, refillRate int, refillPeriod time.Duration) {
	prl.mu.Lock()
	defer prl.mu.Unlock()

	prl.limiters[provider] = NewRateLimiter(maxTokens, refillRate, refillPeriod)

	prl.logger.Info("Rate limit updated", map[string]interface{}{
		"provider":      provider,
		"max_tokens":    maxTokens,
		"refill_rate":   refillRate,
		"refill_period": refillPeriod.String(),
	})
}

// AdaptiveRateLimiter adjusts rate limits based on response times
type AdaptiveRateLimiter struct {
	base           *RateLimiter
	measurements   []time.Duration
	mu             sync.Mutex
	adjustInterval time.Duration
	minRate        int
	maxRate        int
	currentRate    int
	logger         *logging.Logger
}

// NewAdaptiveRateLimiter creates a rate limiter that adapts to API response times
func NewAdaptiveRateLimiter(initialRate int, minRate int, maxRate int) *AdaptiveRateLimiter {
	arl := &AdaptiveRateLimiter{
		base:           NewRateLimiter(initialRate, initialRate, 1*time.Second),
		measurements:   make([]time.Duration, 0, 100),
		adjustInterval: 30 * time.Second,
		minRate:        minRate,
		maxRate:        maxRate,
		currentRate:    initialRate,
		logger:         logging.GetLogger(),
	}

	// Start adjustment routine
	go arl.adjustRoutine()

	return arl
}

// Wait waits for rate limit and records response time
func (arl *AdaptiveRateLimiter) Wait(ctx context.Context) error {
	return arl.base.Wait(ctx)
}

// RecordLatency records API response latency for adaptation
func (arl *AdaptiveRateLimiter) RecordLatency(latency time.Duration) {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	arl.measurements = append(arl.measurements, latency)

	// Keep only last 100 measurements
	if len(arl.measurements) > 100 {
		arl.measurements = arl.measurements[1:]
	}
}

// adjustRoutine periodically adjusts rate limits based on measurements
func (arl *AdaptiveRateLimiter) adjustRoutine() {
	ticker := time.NewTicker(arl.adjustInterval)
	defer ticker.Stop()

	for range ticker.C {
		arl.adjust()
	}
}

// adjust calculates and applies new rate limit
func (arl *AdaptiveRateLimiter) adjust() {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	if len(arl.measurements) < 10 {
		return // Not enough data
	}

	// Calculate average latency
	var total time.Duration
	for _, m := range arl.measurements {
		total += m
	}
	avgLatency := total / time.Duration(len(arl.measurements))

	// Adjust rate based on latency
	newRate := arl.currentRate

	if avgLatency < 200*time.Millisecond {
		// Fast responses, can increase rate
		newRate = arl.currentRate + 1
	} else if avgLatency > 500*time.Millisecond {
		// Slow responses, decrease rate
		newRate = arl.currentRate - 1
	}

	// Apply bounds
	if newRate < arl.minRate {
		newRate = arl.minRate
	} else if newRate > arl.maxRate {
		newRate = arl.maxRate
	}

	// Update if changed
	if newRate != arl.currentRate {
		arl.currentRate = newRate
		arl.base = NewRateLimiter(newRate, newRate, 1*time.Second)

		arl.logger.Info("Adaptive rate limit adjusted", map[string]interface{}{
			"old_rate":    arl.currentRate,
			"new_rate":    newRate,
			"avg_latency": avgLatency.String(),
		})
	}

	// Clear old measurements
	arl.measurements = arl.measurements[:0]
}

// Global rate limiter instance
var globalRateLimiter *ProviderRateLimits
var rateLimiterOnce sync.Once

// GetRateLimiter returns the global rate limiter instance
func GetRateLimiter() *ProviderRateLimits {
	rateLimiterOnce.Do(func() {
		globalRateLimiter = NewProviderRateLimits()
	})
	return globalRateLimiter
}
