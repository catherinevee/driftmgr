package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting and DDoS protection
type RateLimiter struct {
	limiters map[string]*userLimiter
	mu       sync.RWMutex
	config   *Config
	logger   *zerolog.Logger

	// DDoS protection
	globalLimiter *rate.Limiter
	ipBanList     map[string]time.Time
	banMutex      sync.RWMutex
}

// userLimiter tracks rate limits per user/IP
type userLimiter struct {
	limiter      *rate.Limiter
	lastAccess   time.Time
	requestCount int64
	violations   int
}

// Config for rate limiting
type Config struct {
	// RequestsPerSecond is the rate limit per user/IP
	RequestsPerSecond int `json:"requests_per_second"`

	// BurstSize is the maximum burst size
	BurstSize int `json:"burst_size"`

	// GlobalRequestsPerSecond is the global rate limit
	GlobalRequestsPerSecond int `json:"global_requests_per_second"`

	// GlobalBurstSize is the global burst size
	GlobalBurstSize int `json:"global_burst_size"`

	// CleanupInterval is how often to clean up old limiters
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// LimiterTTL is how long to keep inactive limiters
	LimiterTTL time.Duration `json:"limiter_ttl"`

	// BanDuration is how long to ban IPs that violate limits
	BanDuration time.Duration `json:"ban_duration"`

	// MaxViolations before banning
	MaxViolations int `json:"max_violations"`

	// EnableDDoSProtection enables advanced DDoS protection
	EnableDDoSProtection bool `json:"enable_ddos_protection"`

	// SuspiciousRequestThreshold for DDoS detection
	SuspiciousRequestThreshold int64 `json:"suspicious_request_threshold"`

	// WindowSize for tracking request patterns
	WindowSize time.Duration `json:"window_size"`
}

// DefaultConfig returns default rate limiting configuration
func DefaultConfig() *Config {
	return &Config{
		RequestsPerSecond:          10,
		BurstSize:                  20,
		GlobalRequestsPerSecond:    1000,
		GlobalBurstSize:            2000,
		CleanupInterval:            1 * time.Minute,
		LimiterTTL:                 10 * time.Minute,
		BanDuration:                30 * time.Minute,
		MaxViolations:              5,
		EnableDDoSProtection:       true,
		SuspiciousRequestThreshold: 100,
		WindowSize:                 1 * time.Minute,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *Config) *RateLimiter {
	if config == nil {
		config = DefaultConfig()
	}

	logger := logging.WithComponent("rate-limiter")

	rl := &RateLimiter{
		limiters:      make(map[string]*userLimiter),
		config:        config,
		logger:        &logger,
		globalLimiter: rate.NewLimiter(rate.Limit(config.GlobalRequestsPerSecond), config.GlobalBurstSize),
		ipBanList:     make(map[string]time.Time),
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	// Start ban list cleanup
	go rl.banCleanupRoutine()

	return rl
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	// Check if IP is banned
	if rl.isBanned(identifier) {
		rl.logger.Warn().
			Str("identifier", identifier).
			Msg("request from banned identifier")
		return false
	}

	// Check global rate limit first
	if !rl.globalLimiter.Allow() {
		rl.logger.Warn().
			Str("identifier", identifier).
			Msg("global rate limit exceeded")
		return false
	}

	// Get or create limiter for this identifier
	limiter := rl.getLimiter(identifier)

	// Check if allowed
	allowed := limiter.limiter.Allow()

	// Update stats
	limiter.lastAccess = time.Now()
	limiter.requestCount++

	if !allowed {
		limiter.violations++

		// Check if should ban
		if limiter.violations >= rl.config.MaxViolations {
			rl.ban(identifier)
		}

		rl.logger.Warn().
			Str("identifier", identifier).
			Int("violations", limiter.violations).
			Msg("rate limit exceeded")
	}

	// DDoS detection
	if rl.config.EnableDDoSProtection {
		rl.checkDDoSPattern(identifier, limiter)
	}

	return allowed
}

// AllowN checks if n requests are allowed
func (rl *RateLimiter) AllowN(identifier string, n int) bool {
	if rl.isBanned(identifier) {
		return false
	}

	if !rl.globalLimiter.AllowN(time.Now(), n) {
		return false
	}

	limiter := rl.getLimiter(identifier)
	return limiter.limiter.AllowN(time.Now(), n)
}

// Wait blocks until the request can proceed
func (rl *RateLimiter) Wait(ctx context.Context, identifier string) error {
	if rl.isBanned(identifier) {
		return fmt.Errorf("identifier is banned")
	}

	// Wait for global limiter
	if err := rl.globalLimiter.Wait(ctx); err != nil {
		return err
	}

	// Wait for user limiter
	limiter := rl.getLimiter(identifier)
	return limiter.limiter.Wait(ctx)
}

// getLimiter gets or creates a limiter for an identifier
func (rl *RateLimiter) getLimiter(identifier string) *userLimiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[identifier]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check
	if limiter, exists := rl.limiters[identifier]; exists {
		return limiter
	}

	limiter = &userLimiter{
		limiter:      rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.BurstSize),
		lastAccess:   time.Now(),
		requestCount: 0,
		violations:   0,
	}

	rl.limiters[identifier] = limiter
	return limiter
}

// isBanned checks if an identifier is banned
func (rl *RateLimiter) isBanned(identifier string) bool {
	rl.banMutex.RLock()
	defer rl.banMutex.RUnlock()

	banTime, exists := rl.ipBanList[identifier]
	if !exists {
		return false
	}

	// Check if ban has expired
	if time.Now().After(banTime) {
		return false
	}

	return true
}

// ban adds an identifier to the ban list
func (rl *RateLimiter) ban(identifier string) {
	rl.banMutex.Lock()
	defer rl.banMutex.Unlock()

	banUntil := time.Now().Add(rl.config.BanDuration)
	rl.ipBanList[identifier] = banUntil

	rl.logger.Warn().
		Str("identifier", identifier).
		Time("ban_until", banUntil).
		Msg("identifier banned for rate limit violations")
}

// checkDDoSPattern checks for DDoS attack patterns
func (rl *RateLimiter) checkDDoSPattern(identifier string, limiter *userLimiter) {
	// Check if request count is suspicious
	if limiter.requestCount > rl.config.SuspiciousRequestThreshold {
		// Calculate request rate
		duration := time.Since(limiter.lastAccess)
		if duration < rl.config.WindowSize {
			requestRate := float64(limiter.requestCount) / duration.Seconds()

			// If rate is extremely high, it might be DDoS
			if requestRate > float64(rl.config.RequestsPerSecond*10) {
				rl.logger.Error().
					Str("identifier", identifier).
					Float64("request_rate", requestRate).
					Int64("request_count", limiter.requestCount).
					Msg("potential DDoS attack detected")

				// Immediately ban the identifier
				rl.ban(identifier)
			}
		}
	}
}

// cleanupRoutine periodically cleans up old limiters
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes old limiters
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for identifier, limiter := range rl.limiters {
		if now.Sub(limiter.lastAccess) > rl.config.LimiterTTL {
			delete(rl.limiters, identifier)
		}
	}
}

// banCleanupRoutine periodically cleans up expired bans
func (rl *RateLimiter) banCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanupBans()
	}
}

// cleanupBans removes expired bans
func (rl *RateLimiter) cleanupBans() {
	rl.banMutex.Lock()
	defer rl.banMutex.Unlock()

	now := time.Now()
	for identifier, banTime := range rl.ipBanList {
		if now.After(banTime) {
			delete(rl.ipBanList, identifier)
		}
	}
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	activeCount := len(rl.limiters)
	rl.mu.RUnlock()

	rl.banMutex.RLock()
	bannedCount := len(rl.ipBanList)
	rl.banMutex.RUnlock()

	return map[string]interface{}{
		"active_limiters": activeCount,
		"banned_count":    bannedCount,
		"config":          rl.config,
	}
}

// Reset resets the rate limiter for an identifier
func (rl *RateLimiter) Reset(identifier string) {
	rl.mu.Lock()
	delete(rl.limiters, identifier)
	rl.mu.Unlock()

	rl.banMutex.Lock()
	delete(rl.ipBanList, identifier)
	rl.banMutex.Unlock()
}

// Middleware returns HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get identifier (IP address or user ID)
		identifier := rl.getIdentifier(r)

		// Check rate limit
		if !rl.Allow(identifier) {
			rl.sendRateLimitError(w)
			return
		}

		// Add rate limit headers
		rl.addRateLimitHeaders(w, identifier)

		// Continue to next handler
		next(w, r)
	}
}

// getIdentifier extracts the identifier from the request
func (rl *RateLimiter) getIdentifier(r *http.Request) string {
	// Try to get user ID from context (if authenticated)
	// For now, use IP address
	return r.RemoteAddr
}

// sendRateLimitError sends a rate limit error response
func (rl *RateLimiter) sendRateLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.config.BanDuration.Seconds())))
	w.WriteHeader(http.StatusTooManyRequests)

	w.Write([]byte(`{"error":"rate limit exceeded","status":429}`))
}

// addRateLimitHeaders adds rate limit headers to the response
func (rl *RateLimiter) addRateLimitHeaders(w http.ResponseWriter, identifier string) {
	limiter := rl.getLimiter(identifier)

	// Add standard rate limit headers
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerSecond))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", int(limiter.limiter.Tokens())))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
}
