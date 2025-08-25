package resilience

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting capabilities
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	config   *RateLimiterConfig
	mu       sync.RWMutex
	metrics  *RateLimiterMetrics
}

// RateLimiterConfig configures rate limiting
type RateLimiterConfig struct {
	DefaultRate     int                 `yaml:"default_rate"`  // Requests per second
	DefaultBurst    int                 `yaml:"default_burst"` // Burst size
	PerKeyLimits    map[string]KeyLimit `yaml:"per_key_limits"`
	GlobalLimit     int                 `yaml:"global_limit"`     // Global rate limit
	WindowSize      time.Duration       `yaml:"window_size"`      // Time window for limits
	CleanupInterval time.Duration       `yaml:"cleanup_interval"` // Cleanup old limiters
	EnableMetrics   bool                `yaml:"enable_metrics"`
}

// KeyLimit represents per-key rate limits
type KeyLimit struct {
	Rate  int `yaml:"rate"`
	Burst int `yaml:"burst"`
}

// RateLimiterMetrics tracks rate limiter metrics
type RateLimiterMetrics struct {
	mu              sync.RWMutex
	totalRequests   int64
	allowedRequests int64
	deniedRequests  int64
	activeKeys      int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	if config == nil {
		config = &RateLimiterConfig{
			DefaultRate:     100,
			DefaultBurst:    10,
			GlobalLimit:     1000,
			WindowSize:      time.Minute,
			CleanupInterval: 5 * time.Minute,
			EnableMetrics:   true,
		}
	}

	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
		metrics:  &RateLimiterMetrics{},
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get or create limiter for key
	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rl.createLimiter(key)
		rl.limiters[key] = limiter
	}

	// Check if allowed
	allowed := limiter.Allow()

	// Update metrics
	if rl.config.EnableMetrics {
		rl.updateMetrics(allowed)
	}

	return allowed
}

// AllowN checks if n requests are allowed
func (rl *RateLimiter) AllowN(key string, n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rl.createLimiter(key)
		rl.limiters[key] = limiter
	}

	allowed := limiter.AllowN(time.Now(), n)

	if rl.config.EnableMetrics {
		rl.updateMetrics(allowed)
	}

	return allowed
}

// Wait waits until a request is allowed
func (rl *RateLimiter) Wait(ctx context.Context, key string) error {
	rl.mu.Lock()
	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rl.createLimiter(key)
		rl.limiters[key] = limiter
	}
	rl.mu.Unlock()

	err := limiter.Wait(ctx)

	if rl.config.EnableMetrics {
		rl.updateMetrics(err == nil)
	}

	return err
}

// Reserve reserves a request
func (rl *RateLimiter) Reserve(key string) *rate.Reservation {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rl.createLimiter(key)
		rl.limiters[key] = limiter
	}

	return limiter.Reserve()
}

// createLimiter creates a new limiter for a key
func (rl *RateLimiter) createLimiter(key string) *rate.Limiter {
	// Check for per-key limits
	if keyLimit, exists := rl.config.PerKeyLimits[key]; exists {
		return rate.NewLimiter(rate.Limit(keyLimit.Rate), keyLimit.Burst)
	}

	// Use default limits
	return rate.NewLimiter(rate.Limit(rl.config.DefaultRate), rl.config.DefaultBurst)
}

// cleanup removes old limiters
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// In production, you'd track last access time and remove old limiters
		// For now, we'll keep all limiters
		rl.metrics.activeKeys = len(rl.limiters)
		rl.mu.Unlock()
	}
}

// updateMetrics updates rate limiter metrics
func (rl *RateLimiter) updateMetrics(allowed bool) {
	rl.metrics.mu.Lock()
	defer rl.metrics.mu.Unlock()

	rl.metrics.totalRequests++
	if allowed {
		rl.metrics.allowedRequests++
	} else {
		rl.metrics.deniedRequests++
	}
}

// GetMetrics returns current metrics
func (rl *RateLimiter) GetMetrics() RateLimiterMetrics {
	rl.metrics.mu.RLock()
	defer rl.metrics.mu.RUnlock()
	return *rl.metrics
}

// Reset resets the rate limiter for a key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, key)
}

// ResetAll resets all rate limiters
func (rl *RateLimiter) ResetAll() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limiters = make(map[string]*rate.Limiter)
}

// TokenBucketLimiter implements token bucket algorithm
type TokenBucketLimiter struct {
	capacity   int
	tokens     int
	refillRate time.Duration
	mu         sync.Mutex
	lastRefill time.Time
}

// NewTokenBucketLimiter creates a new token bucket limiter
func NewTokenBucketLimiter(capacity int, refillRate time.Duration) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (tbl *TokenBucketLimiter) Allow() bool {
	return tbl.AllowN(1)
}

// AllowN checks if n requests are allowed
func (tbl *TokenBucketLimiter) AllowN(n int) bool {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	tbl.refill()

	if tbl.tokens >= n {
		tbl.tokens -= n
		return true
	}

	return false
}

// refill refills tokens based on elapsed time
func (tbl *TokenBucketLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(tbl.lastRefill)

	// Calculate tokens to add
	tokensToAdd := int(elapsed / tbl.refillRate)
	if tokensToAdd > 0 {
		tbl.tokens = min(tbl.capacity, tbl.tokens+tokensToAdd)
		tbl.lastRefill = now
	}
}

// SlidingWindowLimiter implements sliding window algorithm
type SlidingWindowLimiter struct {
	windowSize  time.Duration
	maxRequests int
	requests    []time.Time
	mu          sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window limiter
func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0),
	}
}

// Allow checks if a request is allowed
func (swl *SlidingWindowLimiter) Allow() bool {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-swl.windowSize)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, reqTime := range swl.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	swl.requests = validRequests

	// Check if we can allow the request
	if len(swl.requests) < swl.maxRequests {
		swl.requests = append(swl.requests, now)
		return true
	}

	return false
}

// AdaptiveRateLimiter adjusts rate limits based on system load
type AdaptiveRateLimiter struct {
	baseLimiter     *RateLimiter
	loadMonitor     LoadMonitor
	adjustmentRatio float64
	mu              sync.RWMutex
}

// LoadMonitor monitors system load
type LoadMonitor interface {
	GetLoad() float64
	GetErrorRate() float64
	GetLatency() time.Duration
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(config *RateLimiterConfig, monitor LoadMonitor) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseLimiter:     NewRateLimiter(config),
		loadMonitor:     monitor,
		adjustmentRatio: 1.0,
	}
}

// Allow checks if a request is allowed with adaptive limits
func (arl *AdaptiveRateLimiter) Allow(key string) bool {
	arl.adjustLimits()
	return arl.baseLimiter.Allow(key)
}

// adjustLimits adjusts rate limits based on system load
func (arl *AdaptiveRateLimiter) adjustLimits() {
	load := arl.loadMonitor.GetLoad()
	errorRate := arl.loadMonitor.GetErrorRate()

	arl.mu.Lock()
	defer arl.mu.Unlock()

	// Adjust based on load and error rate
	if load > 0.8 || errorRate > 0.1 {
		// Reduce rate limits
		arl.adjustmentRatio = math.Max(0.5, arl.adjustmentRatio-0.1)
	} else if load < 0.5 && errorRate < 0.01 {
		// Increase rate limits
		arl.adjustmentRatio = math.Min(2.0, arl.adjustmentRatio+0.1)
	}

	// Apply adjustment
	newRate := int(float64(arl.baseLimiter.config.DefaultRate) * arl.adjustmentRatio)
	arl.baseLimiter.config.DefaultRate = newRate
}

// PriorityRateLimiter provides priority-based rate limiting
type PriorityRateLimiter struct {
	limiters map[Priority]*RateLimiter
	config   *PriorityRateLimiterConfig
	mu       sync.RWMutex
}

// Priority represents request priority
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// PriorityRateLimiterConfig configures priority-based rate limiting
type PriorityRateLimiterConfig struct {
	LowRate      int `yaml:"low_rate"`
	NormalRate   int `yaml:"normal_rate"`
	HighRate     int `yaml:"high_rate"`
	CriticalRate int `yaml:"critical_rate"`
}

// NewPriorityRateLimiter creates a new priority rate limiter
func NewPriorityRateLimiter(config *PriorityRateLimiterConfig) *PriorityRateLimiter {
	prl := &PriorityRateLimiter{
		limiters: make(map[Priority]*RateLimiter),
		config:   config,
	}

	// Create limiters for each priority
	prl.limiters[PriorityLow] = NewRateLimiter(&RateLimiterConfig{
		DefaultRate:  config.LowRate,
		DefaultBurst: config.LowRate / 10,
	})
	prl.limiters[PriorityNormal] = NewRateLimiter(&RateLimiterConfig{
		DefaultRate:  config.NormalRate,
		DefaultBurst: config.NormalRate / 10,
	})
	prl.limiters[PriorityHigh] = NewRateLimiter(&RateLimiterConfig{
		DefaultRate:  config.HighRate,
		DefaultBurst: config.HighRate / 10,
	})
	prl.limiters[PriorityCritical] = NewRateLimiter(&RateLimiterConfig{
		DefaultRate:  config.CriticalRate,
		DefaultBurst: config.CriticalRate / 10,
	})

	return prl
}

// Allow checks if a request is allowed based on priority
func (prl *PriorityRateLimiter) Allow(key string, priority Priority) bool {
	prl.mu.RLock()
	limiter, exists := prl.limiters[priority]
	prl.mu.RUnlock()

	if !exists {
		return false
	}

	return limiter.Allow(key)
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	Key        string
	RetryAfter time.Duration
	Limit      int
	Remaining  int
	ResetAt    time.Time
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for key %s, retry after %v", e.Key, e.RetryAfter)
}
