package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logger"
	"github.com/catherinevee/driftmgr/internal/telemetry"
	"golang.org/x/time/rate"
)

var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrInvalidConfig     = errors.New("invalid rate limit configuration")
)

// Limiter represents a rate limiter
type Limiter struct {
	limiter  *rate.Limiter
	name     string
	requests int
	window   time.Duration
	burst    int
	log      logger.Logger
}

// Config represents rate limiter configuration
type Config struct {
	Name     string
	Requests int           // Requests per window
	Window   time.Duration // Time window
	Burst    int           // Burst capacity
}

// NewLimiter creates a new rate limiter
func NewLimiter(config Config) (*Limiter, error) {
	if config.Requests <= 0 || config.Window <= 0 {
		return nil, ErrInvalidConfig
	}

	if config.Burst <= 0 {
		config.Burst = config.Requests
	}

	// Calculate rate per second
	ratePerSecond := float64(config.Requests) / config.Window.Seconds()

	return &Limiter{
		limiter:  rate.NewLimiter(rate.Limit(ratePerSecond), config.Burst),
		name:     config.Name,
		requests: config.Requests,
		window:   config.Window,
		burst:    config.Burst,
		log:      logger.New("rate_limiter"),
	}, nil
}

// Allow checks if a request is allowed
func (l *Limiter) Allow() bool {
	allowed := l.limiter.Allow()

	if !allowed {
		l.log.Debug("Rate limit exceeded",
			logger.String("name", l.name),
			logger.Int("requests", l.requests),
			logger.Duration("window", l.window),
		)
	}

	return allowed
}

// AllowN checks if n requests are allowed
func (l *Limiter) AllowN(n int) bool {
	allowed := l.limiter.AllowN(time.Now(), n)

	if !allowed {
		l.log.Debug("Rate limit exceeded for N requests",
			logger.String("name", l.name),
			logger.Int("n", n),
			logger.Int("requests", l.requests),
			logger.Duration("window", l.window),
		)
	}

	return allowed
}

// Wait blocks until a request is allowed
func (l *Limiter) Wait(ctx context.Context) error {
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "rate_limiter.wait")
		defer span.End()
	}

	err := l.limiter.Wait(ctx)
	if err != nil {
		l.log.Debug("Rate limiter wait failed",
			logger.String("name", l.name),
			logger.Error(err),
		)
	}

	return err
}

// WaitN blocks until n requests are allowed
func (l *Limiter) WaitN(ctx context.Context, n int) error {
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "rate_limiter.wait_n")
		defer span.End()
	}

	err := l.limiter.WaitN(ctx, n)
	if err != nil {
		l.log.Debug("Rate limiter wait N failed",
			logger.String("name", l.name),
			logger.Int("n", n),
			logger.Error(err),
		)
	}

	return err
}

// Reserve reserves n tokens
func (l *Limiter) Reserve(n int) *rate.Reservation {
	return l.limiter.ReserveN(time.Now(), n)
}

// Tokens returns the number of available tokens
func (l *Limiter) Tokens() float64 {
	return l.limiter.Tokens()
}

// Manager manages multiple rate limiters
type Manager struct {
	limiters map[string]*Limiter
	mu       sync.RWMutex
	log      logger.Logger
}

// NewManager creates a new rate limiter manager
func NewManager() *Manager {
	return &Manager{
		limiters: make(map[string]*Limiter),
		log:      logger.New("rate_limiter_manager"),
	}
}

// AddLimiter adds a rate limiter
func (m *Manager) AddLimiter(name string, config Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.Name = name
	limiter, err := NewLimiter(config)
	if err != nil {
		return err
	}

	m.limiters[name] = limiter

	m.log.Info("Added rate limiter",
		logger.String("name", name),
		logger.Int("requests", config.Requests),
		logger.Duration("window", config.Window),
		logger.Int("burst", config.Burst),
	)

	return nil
}

// GetLimiter returns a rate limiter by name
func (m *Manager) GetLimiter(name string) (*Limiter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	limiter, exists := m.limiters[name]
	return limiter, exists
}

// RemoveLimiter removes a rate limiter
func (m *Manager) RemoveLimiter(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.limiters, name)

	m.log.Info("Removed rate limiter",
		logger.String("name", name),
	)
}

// Allow checks if a request is allowed for a specific limiter
func (m *Manager) Allow(name string) bool {
	limiter, exists := m.GetLimiter(name)
	if !exists {
		return true // No limiter means no limit
	}

	return limiter.Allow()
}

// Wait blocks until a request is allowed for a specific limiter
func (m *Manager) Wait(ctx context.Context, name string) error {
	limiter, exists := m.GetLimiter(name)
	if !exists {
		return nil // No limiter means no limit
	}

	return limiter.Wait(ctx)
}

// Middleware creates HTTP middleware for rate limiting
func (m *Manager) Middleware(limiterName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.Allow(limiterName) {
				m.log.Warn("Rate limit exceeded for HTTP request",
					logger.String("limiter", limiterName),
					logger.String("method", r.Method),
					logger.String("path", r.URL.Path),
					logger.String("remote", r.RemoteAddr),
				)

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPBasedLimiter provides IP-based rate limiting
type IPBasedLimiter struct {
	limiters map[string]*Limiter
	config   Config
	mu       sync.RWMutex
	cleanup  *time.Ticker
	log      logger.Logger
}

// NewIPBasedLimiter creates a new IP-based rate limiter
func NewIPBasedLimiter(config Config) *IPBasedLimiter {
	ipl := &IPBasedLimiter{
		limiters: make(map[string]*Limiter),
		config:   config,
		cleanup:  time.NewTicker(5 * time.Minute),
		log:      logger.New("ip_rate_limiter"),
	}

	// Start cleanup routine
	go ipl.cleanupRoutine()

	return ipl
}

// Allow checks if a request from an IP is allowed
func (ipl *IPBasedLimiter) Allow(ip string) bool {
	ipl.mu.RLock()
	limiter, exists := ipl.limiters[ip]
	ipl.mu.RUnlock()

	if !exists {
		// Create new limiter for this IP
		ipl.mu.Lock()
		// Double-check after acquiring write lock
		if limiter, exists = ipl.limiters[ip]; !exists {
			config := ipl.config
			config.Name = ip
			limiter, _ = NewLimiter(config)
			ipl.limiters[ip] = limiter
		}
		ipl.mu.Unlock()
	}

	return limiter.Allow()
}

// Wait blocks until a request from an IP is allowed
func (ipl *IPBasedLimiter) Wait(ctx context.Context, ip string) error {
	ipl.mu.RLock()
	limiter, exists := ipl.limiters[ip]
	ipl.mu.RUnlock()

	if !exists {
		// Create new limiter for this IP
		ipl.mu.Lock()
		// Double-check after acquiring write lock
		if limiter, exists = ipl.limiters[ip]; !exists {
			config := ipl.config
			config.Name = ip
			limiter, _ = NewLimiter(config)
			ipl.limiters[ip] = limiter
		}
		ipl.mu.Unlock()
	}

	return limiter.Wait(ctx)
}

// cleanupRoutine removes unused IP limiters
func (ipl *IPBasedLimiter) cleanupRoutine() {
	for range ipl.cleanup.C {
		ipl.mu.Lock()

		// Remove limiters with no recent activity
		for ip, limiter := range ipl.limiters {
			if limiter.Tokens() >= float64(limiter.burst) {
				delete(ipl.limiters, ip)
				ipl.log.Debug("Removed inactive IP limiter",
					logger.String("ip", ip),
				)
			}
		}

		ipl.mu.Unlock()
	}
}

// Stop stops the cleanup routine
func (ipl *IPBasedLimiter) Stop() {
	ipl.cleanup.Stop()
}

// Middleware creates HTTP middleware for IP-based rate limiting
func (ipl *IPBasedLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !ipl.Allow(ip) {
				ipl.log.Warn("IP rate limit exceeded",
					logger.String("ip", ip),
					logger.String("method", r.Method),
					logger.String("path", r.URL.Path),
				)

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if there are multiple
		if idx := len(xff) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if xff[i] == ',' {
					return xff[i+1:]
				}
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
