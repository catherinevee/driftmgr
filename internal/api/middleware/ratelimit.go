package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware provides rate limiting for API endpoints
type RateLimitMiddleware struct {
	limiters map[string]*userLimiter
	mu       sync.RWMutex
	config   RateLimitConfig
	cleanup  *time.Ticker
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	// Global limits
	GlobalRPS     int // Requests per second globally
	GlobalBurst   int // Burst size globally
	
	// Per-user limits
	UserRPS       int // Requests per second per user
	UserBurst     int // Burst size per user
	
	// Per-IP limits
	IPRPS         int // Requests per second per IP
	IPBurst       int // Burst size per IP
	
	// API key limits (higher for service accounts)
	APIKeyRPS     int // Requests per second for API keys
	APIKeyBurst   int // Burst size for API keys
	
	// Endpoint-specific limits
	EndpointLimits map[string]EndpointLimit
	
	// Cleanup interval
	CleanupInterval time.Duration
	
	// TTL for inactive limiters
	InactiveTTL     time.Duration
}

// EndpointLimit defines limits for specific endpoints
type EndpointLimit struct {
	RPS   int
	Burst int
}

// userLimiter tracks rate limiting for a specific user/IP
type userLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRPS:   1000,
		GlobalBurst: 2000,
		
		UserRPS:     100,
		UserBurst:   200,
		
		IPRPS:       50,
		IPBurst:     100,
		
		APIKeyRPS:   500,
		APIKeyBurst: 1000,
		
		EndpointLimits: map[string]EndpointLimit{
			"/api/discovery":     {RPS: 10, Burst: 20},
			"/api/remediation":   {RPS: 5, Burst: 10},
			"/api/drift/detect":  {RPS: 10, Burst: 20},
			"/api/state/import":  {RPS: 5, Burst: 10},
			"/api/auth/login":    {RPS: 5, Burst: 10},
		},
		
		CleanupInterval: 5 * time.Minute,
		InactiveTTL:     30 * time.Minute,
	}
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(config RateLimitConfig) *RateLimitMiddleware {
	m := &RateLimitMiddleware{
		limiters: make(map[string]*userLimiter),
		config:   config,
	}

	// Start cleanup routine
	m.cleanup = time.NewTicker(config.CleanupInterval)
	go m.cleanupInactive()

	return m
}

// RateLimit enforces rate limiting
func (m *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	// Create global limiter
	globalLimiter := rate.NewLimiter(rate.Limit(m.config.GlobalRPS), m.config.GlobalBurst)

	return func(c *gin.Context) {
		// Check global rate limit first
		if !globalLimiter.Allow() {
			m.rateLimitExceeded(c, "global")
			return
		}

		// Get identifier for rate limiting
		identifier := m.getIdentifier(c)
		
		// Get or create limiter for this identifier
		limiter := m.getLimiter(identifier, c)
		
		// Check endpoint-specific limits if configured
		if endpointLimit, exists := m.config.EndpointLimits[c.Request.URL.Path]; exists {
			endpointLimiter := m.getEndpointLimiter(identifier+":"+c.Request.URL.Path, endpointLimit)
			if !endpointLimiter.Allow() {
				m.rateLimitExceeded(c, "endpoint")
				return
			}
		}

		// Check user/IP rate limit
		if !limiter.Allow() {
			m.rateLimitExceeded(c, "user")
			return
		}

		// Add rate limit headers
		m.addRateLimitHeaders(c, limiter)

		c.Next()
	}
}

// getIdentifier returns the identifier for rate limiting (user ID, API key, or IP)
func (m *RateLimitMiddleware) getIdentifier(c *gin.Context) string {
	// Check for API key first
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return "apikey:" + apiKey
	}

	// Check for authenticated user
	if userID, exists := c.Get("user_id"); exists {
		return "user:" + userID.(string)
	}

	// Fall back to IP address
	ip := c.ClientIP()
	return "ip:" + ip
}

// getLimiter returns the rate limiter for an identifier
func (m *RateLimitMiddleware) getLimiter(identifier string, c *gin.Context) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if limiter exists
	if ul, exists := m.limiters[identifier]; exists {
		ul.lastSeen = time.Now()
		return ul.limiter
	}

	// Create new limiter based on identifier type
	var limiter *rate.Limiter
	
	if isAPIKey(identifier) {
		// Higher limits for API keys
		limiter = rate.NewLimiter(rate.Limit(m.config.APIKeyRPS), m.config.APIKeyBurst)
	} else if isUser(identifier) {
		// Standard limits for authenticated users
		limiter = rate.NewLimiter(rate.Limit(m.config.UserRPS), m.config.UserBurst)
	} else {
		// Lower limits for anonymous IPs
		limiter = rate.NewLimiter(rate.Limit(m.config.IPRPS), m.config.IPBurst)
	}

	m.limiters[identifier] = &userLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

// getEndpointLimiter returns a rate limiter for a specific endpoint
func (m *RateLimitMiddleware) getEndpointLimiter(identifier string, limit EndpointLimit) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ul, exists := m.limiters[identifier]; exists {
		ul.lastSeen = time.Now()
		return ul.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(limit.RPS), limit.Burst)
	m.limiters[identifier] = &userLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

// rateLimitExceeded handles rate limit exceeded response
func (m *RateLimitMiddleware) rateLimitExceeded(c *gin.Context, limitType string) {
	c.Header("Retry-After", "60")
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":      "Rate limit exceeded",
		"limit_type": limitType,
		"retry_after": 60,
		"message":    "Too many requests. Please slow down and try again later.",
	})
	c.Abort()
}

// addRateLimitHeaders adds rate limit information headers
func (m *RateLimitMiddleware) addRateLimitHeaders(c *gin.Context, limiter *rate.Limiter) {
	// Get current state
	tokens := int(limiter.Tokens())
	limit := int(limiter.Limit())
	
	// Add headers
	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(tokens))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))
}

// cleanupInactive removes inactive limiters
func (m *RateLimitMiddleware) cleanupInactive() {
	for range m.cleanup.C {
		m.mu.Lock()
		now := time.Now()
		for id, ul := range m.limiters {
			if now.Sub(ul.lastSeen) > m.config.InactiveTTL {
				delete(m.limiters, id)
			}
		}
		m.mu.Unlock()
	}
}

// Stop stops the cleanup routine
func (m *RateLimitMiddleware) Stop() {
	if m.cleanup != nil {
		m.cleanup.Stop()
	}
}

// Helper functions to identify identifier types
func isAPIKey(identifier string) bool {
	return len(identifier) > 7 && identifier[:7] == "apikey:"
}

func isUser(identifier string) bool {
	return len(identifier) > 5 && identifier[:5] == "user:"
}

func isIP(identifier string) bool {
	return len(identifier) > 3 && identifier[:3] == "ip:"
}

// AdaptiveRateLimit provides adaptive rate limiting based on system load
type AdaptiveRateLimit struct {
	base         *RateLimitMiddleware
	loadMonitor  func() float64 // Returns system load 0.0 to 1.0
	adjustFactor float64
}

// NewAdaptiveRateLimit creates an adaptive rate limiter
func NewAdaptiveRateLimit(config RateLimitConfig, loadMonitor func() float64) *AdaptiveRateLimit {
	return &AdaptiveRateLimit{
		base:         NewRateLimitMiddleware(config),
		loadMonitor:  loadMonitor,
		adjustFactor: 0.5, // Reduce limits by up to 50% under load
	}
}

// RateLimit applies adaptive rate limiting
func (a *AdaptiveRateLimit) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get current system load
		load := a.loadMonitor()
		
		// Adjust rate limits based on load
		if load > 0.8 {
			// High load - reduce limits
			adjustedConfig := a.base.config
			factor := 1.0 - (load-0.8)*a.adjustFactor*2.5 // More aggressive reduction
			
			adjustedConfig.UserRPS = int(float64(adjustedConfig.UserRPS) * factor)
			adjustedConfig.IPRPS = int(float64(adjustedConfig.IPRPS) * factor)
			
			// Create temporary middleware with adjusted limits
			tempMiddleware := NewRateLimitMiddleware(adjustedConfig)
			tempMiddleware.RateLimit()(c)
			tempMiddleware.Stop()
		} else {
			// Normal load - use standard limits
			a.base.RateLimit()(c)
		}
	}
}

// IPWhitelist provides IP whitelisting to bypass rate limits
type IPWhitelist struct {
	whitelist map[string]bool
	mu        sync.RWMutex
}

// NewIPWhitelist creates a new IP whitelist
func NewIPWhitelist(ips []string) *IPWhitelist {
	whitelist := make(map[string]bool)
	for _, ip := range ips {
		whitelist[ip] = true
	}
	return &IPWhitelist{
		whitelist: whitelist,
	}
}

// IsWhitelisted checks if an IP is whitelisted
func (w *IPWhitelist) IsWhitelisted(ip string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.whitelist[ip]
}

// Add adds an IP to the whitelist
func (w *IPWhitelist) Add(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.whitelist[ip] = true
}

// Remove removes an IP from the whitelist
func (w *IPWhitelist) Remove(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.whitelist, ip)
}

// RateLimitWithWhitelist applies rate limiting with IP whitelist bypass
func RateLimitWithWhitelist(rateLimiter *RateLimitMiddleware, whitelist *IPWhitelist) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if IP is whitelisted
		if whitelist.IsWhitelisted(c.ClientIP()) {
			c.Next()
			return
		}
		
		// Apply rate limiting
		rateLimiter.RateLimit()(c)
	}
}