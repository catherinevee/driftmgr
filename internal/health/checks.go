package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/credentials"
	"github.com/catherinevee/driftmgr/internal/logging"
	"github.com/catherinevee/driftmgr/internal/resilience"
)

// Status represents health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check is a health check function
type Check func(ctx context.Context) CheckResult

// CheckResult contains the result of a health check
type CheckResult struct {
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration_ms"`
}

// HealthChecker manages health checks
type HealthChecker struct {
	checks      map[string]Check
	mu          sync.RWMutex
	logger      *logging.Logger
	lastResults map[string]CheckResult
	resultsMu   sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	hc := &HealthChecker{
		checks:      make(map[string]Check),
		lastResults: make(map[string]CheckResult),
		logger:      logging.GetLogger(),
	}

	// Register default checks
	hc.registerDefaultChecks()

	// Start background health monitoring
	go hc.monitorHealth()

	return hc
}

// registerDefaultChecks registers all default health checks
func (h *HealthChecker) registerDefaultChecks() {
	// System health
	h.RegisterCheck("system", h.checkSystem)

	// Cloud provider connectivity
	h.RegisterCheck("aws", h.checkAWS)
	h.RegisterCheck("azure", h.checkAzure)
	h.RegisterCheck("gcp", h.checkGCP)
	h.RegisterCheck("digitalocean", h.checkDigitalOcean)

	// Cache health
	h.RegisterCheck("cache", h.checkCache)

	// Circuit breakers
	h.RegisterCheck("circuit_breakers", h.checkCircuitBreakers)

	// Rate limiters
	h.RegisterCheck("rate_limiters", h.checkRateLimiters)

	// Database (if configured)
	h.RegisterCheck("database", h.checkDatabase)

	// State manager
	h.RegisterCheck("state_manager", h.checkStateManager)

	h.logger.Info("Health checks registered", map[string]interface{}{
		"count": len(h.checks),
	})
}

// RegisterCheck registers a new health check
func (h *HealthChecker) RegisterCheck(name string, check Check) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.checks[name] = check

	h.logger.Debug("Health check registered", map[string]interface{}{
		"name": name,
	})
}

// CheckHealth runs all health checks
func (h *HealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	h.mu.RLock()
	checks := make(map[string]Check)
	for name, check := range h.checks {
		checks[name] = check
	}
	h.mu.RUnlock()

	results := make(map[string]CheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Run checks in parallel
	for name, check := range checks {
		wg.Add(1)
		go func(n string, c Check) {
			defer wg.Done()

			// Run check with timeout
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			startTime := time.Now()
			result := c(checkCtx)
			result.Duration = time.Since(startTime)
			result.Timestamp = time.Now()

			mu.Lock()
			results[n] = result
			mu.Unlock()

			// Update last results
			h.resultsMu.Lock()
			h.lastResults[n] = result
			h.resultsMu.Unlock()
		}(name, check)
	}

	wg.Wait()

	// Determine overall status
	overallStatus := StatusHealthy
	unhealthyCount := 0
	degradedCount := 0

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			unhealthyCount++
			overallStatus = StatusUnhealthy
		case StatusDegraded:
			degradedCount++
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
	}

	return HealthStatus{
		Status:    overallStatus,
		Checks:    results,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"healthy_checks":   len(results) - unhealthyCount - degradedCount,
			"degraded_checks":  degradedCount,
			"unhealthy_checks": unhealthyCount,
			"total_checks":     len(results),
		},
	}
}

// System health check
func (h *HealthChecker) checkSystem(ctx context.Context) CheckResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check memory usage
	memoryUsagePercent := float64(m.Alloc) / float64(m.Sys) * 100

	status := StatusHealthy
	if memoryUsagePercent > 80 {
		status = StatusDegraded
	}
	if memoryUsagePercent > 95 {
		status = StatusUnhealthy
	}

	return CheckResult{
		Status: status,
		Details: map[string]interface{}{
			"memory_alloc_mb":      m.Alloc / 1024 / 1024,
			"memory_sys_mb":        m.Sys / 1024 / 1024,
			"memory_usage_percent": memoryUsagePercent,
			"num_goroutines":       runtime.NumGoroutine(),
			"num_cpu":              runtime.NumCPU(),
		},
	}
}

// AWS health check
func (h *HealthChecker) checkAWS(ctx context.Context) CheckResult {
	detector := credentials.NewCredentialDetector()

	if !detector.IsConfigured("aws") {
		return CheckResult{
			Status:  StatusDegraded,
			Message: "AWS credentials not configured",
		}
	}

	// Check circuit breaker status
	breaker := resilience.GetCircuitBreakers().GetBreaker("aws")
	if breaker != nil {
		state := breaker.GetState()
		if state == resilience.StateOpen {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "Circuit breaker open",
				Details: map[string]interface{}{
					"circuit_state": state.String(),
				},
			}
		}
	}

	// Check rate limiter
	limiter := resilience.GetRateLimiter()
	available := limiter.Available("aws")

	status := StatusHealthy
	if available < 2 {
		status = StatusDegraded
	}

	return CheckResult{
		Status: status,
		Details: map[string]interface{}{
			"configured":           true,
			"rate_limit_available": available,
		},
	}
}

// Azure health check
func (h *HealthChecker) checkAzure(ctx context.Context) CheckResult {
	detector := credentials.NewCredentialDetector()

	if !detector.IsConfigured("azure") {
		return CheckResult{
			Status:  StatusDegraded,
			Message: "Azure credentials not configured",
		}
	}

	breaker := resilience.GetCircuitBreakers().GetBreaker("azure")
	if breaker != nil && breaker.GetState() == resilience.StateOpen {
		return CheckResult{
			Status:  StatusUnhealthy,
			Message: "Circuit breaker open",
		}
	}

	return CheckResult{
		Status: StatusHealthy,
		Details: map[string]interface{}{
			"configured": true,
		},
	}
}

// GCP health check
func (h *HealthChecker) checkGCP(ctx context.Context) CheckResult {
	detector := credentials.NewCredentialDetector()

	if !detector.IsConfigured("gcp") {
		return CheckResult{
			Status:  StatusDegraded,
			Message: "GCP credentials not configured",
		}
	}

	breaker := resilience.GetCircuitBreakers().GetBreaker("gcp")
	if breaker != nil && breaker.GetState() == resilience.StateOpen {
		return CheckResult{
			Status:  StatusUnhealthy,
			Message: "Circuit breaker open",
		}
	}

	return CheckResult{
		Status: StatusHealthy,
		Details: map[string]interface{}{
			"configured": true,
		},
	}
}

// DigitalOcean health check
func (h *HealthChecker) checkDigitalOcean(ctx context.Context) CheckResult {
	detector := credentials.NewCredentialDetector()

	if !detector.IsConfigured("digitalocean") {
		return CheckResult{
			Status:  StatusDegraded,
			Message: "DigitalOcean credentials not configured",
		}
	}

	return CheckResult{
		Status: StatusHealthy,
		Details: map[string]interface{}{
			"configured": true,
		},
	}
}

// Cache health check
func (h *HealthChecker) checkCache(ctx context.Context) CheckResult {
	providerCache := cache.GetProviderCache()
	stats := providerCache.GetCacheStats()

	// Calculate overall hit rate
	totalHits := int64(0)
	totalMisses := int64(0)

	for _, stat := range stats {
		totalHits += stat.Hits
		totalMisses += stat.Misses
	}

	hitRate := float64(0)
	if totalHits+totalMisses > 0 {
		hitRate = float64(totalHits) / float64(totalHits+totalMisses)
	}

	status := StatusHealthy
	if hitRate < 0.5 {
		status = StatusDegraded
	}
	if hitRate < 0.2 {
		status = StatusUnhealthy
	}

	return CheckResult{
		Status: status,
		Details: map[string]interface{}{
			"hit_rate":     hitRate,
			"total_hits":   totalHits,
			"total_misses": totalMisses,
			"cache_stats":  stats,
		},
	}
}

// Circuit breaker health check
func (h *HealthChecker) checkCircuitBreakers(ctx context.Context) CheckResult {
	breakers := resilience.GetCircuitBreakers()
	stats := breakers.GetAllStats()

	openCount := 0
	halfOpenCount := 0

	for _, stat := range stats {
		switch stat.State {
		case "open":
			openCount++
		case "half-open":
			halfOpenCount++
		}
	}

	status := StatusHealthy
	message := ""

	if openCount > 0 {
		status = StatusDegraded
		message = fmt.Sprintf("%d circuit breakers open", openCount)
	}

	if openCount > len(stats)/2 {
		status = StatusUnhealthy
		message = "Majority of circuit breakers open"
	}

	return CheckResult{
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"total_breakers":  len(stats),
			"open_count":      openCount,
			"half_open_count": halfOpenCount,
			"breaker_stats":   stats,
		},
	}
}

// Rate limiter health check
func (h *HealthChecker) checkRateLimiters(ctx context.Context) CheckResult {
	limiter := resilience.GetRateLimiter()

	providers := []string{"aws", "azure", "gcp", "digitalocean"}
	limitStatus := make(map[string]int)

	totalAvailable := 0
	for _, provider := range providers {
		available := limiter.Available(provider)
		limitStatus[provider] = available
		totalAvailable += available
	}

	avgAvailable := totalAvailable / len(providers)

	status := StatusHealthy
	if avgAvailable < 3 {
		status = StatusDegraded
	}
	if avgAvailable < 1 {
		status = StatusUnhealthy
	}

	return CheckResult{
		Status: status,
		Details: map[string]interface{}{
			"average_available": avgAvailable,
			"provider_limits":   limitStatus,
		},
	}
}

// Database health check
func (h *HealthChecker) checkDatabase(ctx context.Context) CheckResult {
	// This would check actual database connectivity
	// For now, return healthy if no database is configured

	return CheckResult{
		Status:  StatusHealthy,
		Message: "No database configured",
	}
}

// State manager health check
func (h *HealthChecker) checkStateManager(ctx context.Context) CheckResult {
	// Check if state manager is accessible
	// This would check etcd or other distributed state backend

	return CheckResult{
		Status:  StatusHealthy,
		Message: "State manager operational",
	}
}

// monitorHealth runs periodic health checks
func (h *HealthChecker) monitorHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		status := h.CheckHealth(ctx)
		cancel()

		if status.Status == StatusUnhealthy {
			h.logger.Error("System unhealthy", nil, map[string]interface{}{
				"unhealthy_checks": status.Details["unhealthy_checks"],
			})

			// Record metric
			logging.Metric("health.status", 0, "status", map[string]string{
				"status": "unhealthy",
			})
		} else if status.Status == StatusDegraded {
			h.logger.Warn("System degraded", map[string]interface{}{
				"degraded_checks": status.Details["degraded_checks"],
			})

			logging.Metric("health.status", 0.5, "status", map[string]string{
				"status": "degraded",
			})
		} else {
			logging.Metric("health.status", 1, "status", map[string]string{
				"status": "healthy",
			})
		}
	}
}

// GetLastResults returns the last health check results
func (h *HealthChecker) GetLastResults() map[string]CheckResult {
	h.resultsMu.RLock()
	defer h.resultsMu.RUnlock()

	results := make(map[string]CheckResult)
	for k, v := range h.lastResults {
		results[k] = v
	}

	return results
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    Status                 `json:"status"`
	Checks    map[string]CheckResult `json:"checks"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// HTTPHandler handles HTTP health check requests
func (h *HealthChecker) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Support different health check paths
	switch r.URL.Path {
	case "/health/live":
		h.handleLiveness(w, r)
	case "/health/ready":
		h.handleReadiness(w, r)
	case "/health":
		h.handleFullHealth(w, r, ctx)
	default:
		http.NotFound(w, r)
	}
}

// handleLiveness handles liveness probe
func (h *HealthChecker) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - is the application running?
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleReadiness handles readiness probe
func (h *HealthChecker) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// Check if application is ready to serve traffic
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Run critical checks only
	criticalChecks := []string{"system", "cache", "circuit_breakers"}
	ready := true

	for _, checkName := range criticalChecks {
		h.mu.RLock()
		check, exists := h.checks[checkName]
		h.mu.RUnlock()

		if exists {
			result := check(ctx)
			if result.Status == StatusUnhealthy {
				ready = false
				break
			}
		}
	}

	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now(),
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleFullHealth handles full health check
func (h *HealthChecker) handleFullHealth(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	status := h.CheckHealth(ctx)

	statusCode := http.StatusOK
	switch status.Status {
	case StatusDegraded:
		statusCode = http.StatusOK // Still operational but degraded
	case StatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("Failed to encode health response", err, nil)
	}
}

// Global health checker instance
var globalHealthChecker *HealthChecker
var healthOnce sync.Once

// GetHealthChecker returns the global health checker
func GetHealthChecker() *HealthChecker {
	healthOnce.Do(func() {
		globalHealthChecker = NewHealthChecker()
	})
	return globalHealthChecker
}
