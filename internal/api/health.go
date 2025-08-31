package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/credentials"
	"github.com/gin-gonic/gin"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Status      HealthStatus           `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Uptime      string                 `json:"uptime"`
	Checks      map[string]CheckResult `json:"checks"`
	System      SystemInfo             `json:"system"`
	Dependencies map[string]DependencyHealth `json:"dependencies"`
}

// CheckResult represents an individual check result
type CheckResult struct {
	Status  HealthStatus           `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
	Latency time.Duration          `json:"latency_ms"`
}

// SystemInfo contains system information
type SystemInfo struct {
	CPUUsage       float64 `json:"cpu_usage_percent"`
	MemoryUsage    float64 `json:"memory_usage_percent"`
	MemoryAllocMB  uint64  `json:"memory_alloc_mb"`
	MemoryTotalMB  uint64  `json:"memory_total_mb"`
	Goroutines     int     `json:"goroutines"`
	CPUCores       int     `json:"cpu_cores"`
	OpenFiles      int     `json:"open_files,omitempty"`
	Connections    int     `json:"connections,omitempty"`
}

// DependencyHealth represents external dependency health
type DependencyHealth struct {
	Status       HealthStatus `json:"status"`
	ResponseTime time.Duration `json:"response_time_ms"`
	LastChecked  time.Time    `json:"last_checked"`
	Error        string       `json:"error,omitempty"`
}

// HealthService manages health checks
type HealthService struct {
	startTime      time.Time
	version        string
	db             *sql.DB
	cache          *cache.GlobalCache
	checks         map[string]HealthChecker
	dependencies   map[string]DependencyChecker
	mu             sync.RWMutex
	lastCheck      *HealthCheck
	checkInterval  time.Duration
}

// HealthChecker interface for custom health checks
type HealthChecker interface {
	Check(ctx context.Context) CheckResult
	Name() string
}

// DependencyChecker interface for external dependency checks
type DependencyChecker interface {
	Check(ctx context.Context) DependencyHealth
	Name() string
}

// NewHealthService creates a new health service
func NewHealthService(version string, db *sql.DB, cache *cache.GlobalCache) *HealthService {
	hs := &HealthService{
		startTime:     time.Now(),
		version:       version,
		db:            db,
		cache:         cache,
		checks:        make(map[string]HealthChecker),
		dependencies:  make(map[string]DependencyChecker),
		checkInterval: 30 * time.Second,
	}

	// Register default checks
	hs.registerDefaultChecks()

	// Start background health check
	go hs.backgroundHealthCheck()

	return hs
}

// registerDefaultChecks registers default health checks
func (hs *HealthService) registerDefaultChecks() {
	// Database check
	hs.RegisterCheck("database", &DatabaseHealthCheck{db: hs.db})
	
	// Cache check
	hs.RegisterCheck("cache", &CacheHealthCheck{cache: hs.cache})
	
	// Disk space check
	hs.RegisterCheck("disk", &DiskHealthCheck{})
	
	// Memory check
	hs.RegisterCheck("memory", &MemoryHealthCheck{})
}

// RegisterCheck registers a custom health check
func (hs *HealthService) RegisterCheck(name string, checker HealthChecker) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.checks[name] = checker
}

// RegisterDependency registers an external dependency check
func (hs *HealthService) RegisterDependency(name string, checker DependencyChecker) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.dependencies[name] = checker
}

// GetHealth performs a comprehensive health check
func (hs *HealthService) GetHealth(ctx context.Context) *HealthCheck {
	hs.mu.RLock()
	// Return cached result if recent
	if hs.lastCheck != nil && time.Since(hs.lastCheck.Timestamp) < 5*time.Second {
		cached := *hs.lastCheck
		hs.mu.RUnlock()
		return &cached
	}
	hs.mu.RUnlock()

	// Perform new health check
	health := &HealthCheck{
		Timestamp:    time.Now(),
		Version:      hs.version,
		Uptime:       time.Since(hs.startTime).String(),
		Checks:       make(map[string]CheckResult),
		Dependencies: make(map[string]DependencyHealth),
		System:       hs.getSystemInfo(),
	}

	// Run all checks concurrently
	var wg sync.WaitGroup
	checkResults := make(chan struct {
		name   string
		result CheckResult
	}, len(hs.checks))

	hs.mu.RLock()
	for name, checker := range hs.checks {
		wg.Add(1)
		go func(n string, c HealthChecker) {
			defer wg.Done()
			start := time.Now()
			result := c.Check(ctx)
			result.Latency = time.Since(start)
			checkResults <- struct {
				name   string
				result CheckResult
			}{name: n, result: result}
		}(name, checker)
	}
	hs.mu.RUnlock()

	// Wait for all checks
	go func() {
		wg.Wait()
		close(checkResults)
	}()

	// Collect results
	overallStatus := HealthStatusHealthy
	for res := range checkResults {
		health.Checks[res.name] = res.result
		if res.result.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if res.result.Status == HealthStatusDegraded && overallStatus != HealthStatusUnhealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	// Check dependencies
	hs.mu.RLock()
	for name, checker := range hs.dependencies {
		depHealth := checker.Check(ctx)
		health.Dependencies[name] = depHealth
		if depHealth.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusDegraded
		}
	}
	hs.mu.RUnlock()

	health.Status = overallStatus

	// Cache the result
	hs.mu.Lock()
	hs.lastCheck = health
	hs.mu.Unlock()

	return health
}

// GetLiveness returns a simple liveness check
func (hs *HealthService) GetLiveness() map[string]interface{} {
	return map[string]interface{}{
		"status": "alive",
		"timestamp": time.Now(),
	}
}

// GetReadiness returns readiness status
func (hs *HealthService) GetReadiness(ctx context.Context) map[string]interface{} {
	// Check critical dependencies
	ready := true
	message := "ready"

	// Check database
	if hs.db != nil {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		if err := hs.db.PingContext(ctx); err != nil {
			ready = false
			message = "database not ready"
		}
	}

	return map[string]interface{}{
		"ready":     ready,
		"message":   message,
		"timestamp": time.Now(),
	}
}

// getSystemInfo collects system information
func (hs *HealthService) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		MemoryAllocMB: m.Alloc / 1024 / 1024,
		MemoryTotalMB: m.TotalAlloc / 1024 / 1024,
		Goroutines:    runtime.NumGoroutine(),
		CPUCores:      runtime.NumCPU(),
		CPUUsage:      getCPUUsage(),
		MemoryUsage:   float64(m.Alloc) / float64(m.Sys) * 100,
	}
}

// backgroundHealthCheck runs periodic health checks
func (hs *HealthService) backgroundHealthCheck() {
	ticker := time.NewTicker(hs.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		hs.GetHealth(ctx)
		cancel()
	}
}

// HTTP Handlers

// HealthHandler returns comprehensive health status
func (hs *HealthService) HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		health := hs.GetHealth(ctx)
		
		statusCode := http.StatusOK
		if health.Status == HealthStatusDegraded {
			statusCode = http.StatusOK // Still return 200 for degraded
		} else if health.Status == HealthStatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, health)
	}
}

// LivenessHandler returns liveness status
func (hs *HealthService) LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, hs.GetLiveness())
	}
}

// ReadinessHandler returns readiness status
func (hs *HealthService) ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		readiness := hs.GetReadiness(ctx)
		
		statusCode := http.StatusOK
		if !readiness["ready"].(bool) {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, readiness)
	}
}

// Default health checkers

// DatabaseHealthCheck checks database health
type DatabaseHealthCheck struct {
	db *sql.DB
}

func (d *DatabaseHealthCheck) Name() string { return "database" }

func (d *DatabaseHealthCheck) Check(ctx context.Context) CheckResult {
	if d.db == nil {
		return CheckResult{
			Status:  HealthStatusUnhealthy,
			Message: "Database not configured",
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := d.db.PingContext(ctx)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("Database ping failed: %v", err),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Check connection pool stats
	stats := d.db.Stats()
	
	status := HealthStatusHealthy
	if stats.OpenConnections > stats.MaxOpenConnections*90/100 {
		status = HealthStatusDegraded
	}

	return CheckResult{
		Status: status,
		Details: map[string]interface{}{
			"ping_latency_ms":    latency.Milliseconds(),
			"open_connections":   stats.OpenConnections,
			"in_use":            stats.InUse,
			"idle":              stats.Idle,
			"max_open":          stats.MaxOpenConnections,
			"wait_count":        stats.WaitCount,
			"wait_duration_ms":  stats.WaitDuration.Milliseconds(),
		},
	}
}

// CacheHealthCheck checks cache health
type CacheHealthCheck struct {
	cache *cache.GlobalCache
}

func (c *CacheHealthCheck) Name() string { return "cache" }

func (c *CacheHealthCheck) Check(ctx context.Context) CheckResult {
	if c.cache == nil {
		return CheckResult{
			Status:  HealthStatusUnhealthy,
			Message: "Cache not configured",
		}
	}

	// Test cache operations
	testKey := "health_check_test"
	testValue := time.Now().Unix()

	// Test set
	c.cache.Set(testKey, testValue, 10*time.Second)

	// Test get
	retrieved, found := c.cache.Get(testKey)
	if !found || retrieved != testValue {
		return CheckResult{
			Status:  HealthStatusUnhealthy,
			Message: "Cache operations failed",
		}
	}

	// Get cache stats
	stats := c.cache.GetStats()

	return CheckResult{
		Status: HealthStatusHealthy,
		Details: stats,
	}
}

// DiskHealthCheck checks disk space
type DiskHealthCheck struct{}

func (d *DiskHealthCheck) Name() string { return "disk" }

func (d *DiskHealthCheck) Check(ctx context.Context) CheckResult {
	// This would check actual disk usage
	// Simplified for demonstration
	return CheckResult{
		Status: HealthStatusHealthy,
		Details: map[string]interface{}{
			"available_gb": 100,
			"used_gb":      50,
			"total_gb":     150,
			"usage_percent": 33.3,
		},
	}
}

// MemoryHealthCheck checks memory usage
type MemoryHealthCheck struct{}

func (m *MemoryHealthCheck) Name() string { return "memory" }

func (m *MemoryHealthCheck) Check(ctx context.Context) CheckResult {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	usagePercent := float64(memStats.Alloc) / float64(memStats.Sys) * 100
	
	status := HealthStatusHealthy
	if usagePercent > 90 {
		status = HealthStatusUnhealthy
	} else if usagePercent > 75 {
		status = HealthStatusDegraded
	}

	return CheckResult{
		Status: status,
		Details: map[string]interface{}{
			"alloc_mb":       memStats.Alloc / 1024 / 1024,
			"total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
			"sys_mb":         memStats.Sys / 1024 / 1024,
			"gc_count":       memStats.NumGC,
			"usage_percent":  usagePercent,
		},
	}
}

// CloudProviderHealthCheck checks cloud provider connectivity
type CloudProviderHealthCheck struct {
	provider string
	detector *credentials.CredentialDetector
}

func (c *CloudProviderHealthCheck) Name() string { 
	return fmt.Sprintf("%s_provider", c.provider) 
}

func (c *CloudProviderHealthCheck) Check(ctx context.Context) DependencyHealth {
	start := time.Now()
	
	// Check if credentials are configured
	creds := c.detector.Detect(c.provider)
	
	if creds.Status != "configured" {
		return DependencyHealth{
			Status:       HealthStatusUnhealthy,
			ResponseTime: time.Since(start),
			LastChecked:  time.Now(),
			Error:        "Provider credentials not configured",
		}
	}

	// Would perform actual API call to verify connectivity
	// Simplified for demonstration
	
	return DependencyHealth{
		Status:       HealthStatusHealthy,
		ResponseTime: time.Since(start),
		LastChecked:  time.Now(),
	}
}

// Helper function to get CPU usage (simplified)
func getCPUUsage() float64 {
	// This would use actual CPU monitoring
	// Simplified for demonstration
	return 25.5
}