package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthChecker provides health checking capabilities
type HealthChecker struct {
	checks   map[string]Check
	mu       sync.RWMutex
	status   *HealthStatus
	interval time.Duration
	cancel   context.CancelFunc
}

// Check represents a health check
type Check interface {
	Name() string
	Check(ctx context.Context) error
	Type() CheckType
	Critical() bool
}

// CheckType represents the type of health check
type CheckType string

const (
	CheckTypeLiveness  CheckType = "liveness"
	CheckTypeReadiness CheckType = "readiness"
	CheckTypeStartup   CheckType = "startup"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    Status                   `json:"status"`
	Timestamp time.Time                `json:"timestamp"`
	Checks    map[string]*CheckResult  `json:"checks"`
	Version   string                   `json:"version"`
	Uptime    time.Duration            `json:"uptime"`
	Metadata  map[string]interface{}   `json:"metadata,omitempty"`
}

// Status represents health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name        string        `json:"name"`
	Status      Status        `json:"status"`
	Type        CheckType     `json:"type"`
	Critical    bool          `json:"critical"`
	Message     string        `json:"message,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	LastChecked time.Time     `json:"last_checked"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration) *HealthChecker {
	return &HealthChecker{
		checks:   make(map[string]Check),
		interval: interval,
		status: &HealthStatus{
			Status:    StatusHealthy,
			Timestamp: time.Now(),
			Checks:    make(map[string]*CheckResult),
			Metadata:  make(map[string]interface{}),
		},
	}
}

// RegisterCheck registers a health check
func (hc *HealthChecker) RegisterCheck(check Check) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[check.Name()] = check
}

// Start starts the health checker
func (hc *HealthChecker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	hc.cancel = cancel

	go hc.runChecks(ctx)
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	if hc.cancel != nil {
		hc.cancel()
	}
}

// runChecks runs health checks periodically
func (hc *HealthChecker) runChecks(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// Run initial checks
	hc.performChecks(ctx)

	for {
		select {
		case <-ticker.C:
			hc.performChecks(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// performChecks performs all registered health checks
func (hc *HealthChecker) performChecks(ctx context.Context) {
	hc.mu.RLock()
	checks := make(map[string]Check, len(hc.checks))
	for k, v := range hc.checks {
		checks[k] = v
	}
	hc.mu.RUnlock()

	results := make(map[string]*CheckResult)
	overallStatus := StatusHealthy
	
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, check := range checks {
		wg.Add(1)
		go func(n string, c Check) {
			defer wg.Done()
			
			start := time.Now()
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			err := c.Check(checkCtx)
			duration := time.Since(start)

			result := &CheckResult{
				Name:        c.Name(),
				Type:        c.Type(),
				Critical:    c.Critical(),
				Duration:    duration,
				LastChecked: time.Now(),
			}

			if err != nil {
				result.Status = StatusUnhealthy
				result.Error = err.Error()
				
				mu.Lock()
				if c.Critical() {
					overallStatus = StatusUnhealthy
				} else if overallStatus != StatusUnhealthy {
					overallStatus = StatusDegraded
				}
				mu.Unlock()
			} else {
				result.Status = StatusHealthy
				result.Message = "Check passed"
			}

			mu.Lock()
			results[n] = result
			mu.Unlock()
		}(name, check)
	}

	wg.Wait()

	hc.mu.Lock()
	hc.status.Checks = results
	hc.status.Status = overallStatus
	hc.status.Timestamp = time.Now()
	hc.mu.Unlock()
}

// GetStatus returns the current health status
func (hc *HealthChecker) GetStatus() *HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	status := &HealthStatus{
		Status:    hc.status.Status,
		Timestamp: hc.status.Timestamp,
		Checks:    make(map[string]*CheckResult),
		Version:   hc.status.Version,
		Uptime:    hc.status.Uptime,
		Metadata:  hc.status.Metadata,
	}
	
	for k, v := range hc.status.Checks {
		status.Checks[k] = v
	}
	
	return status
}

// IsHealthy returns true if the service is healthy
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status.Status == StatusHealthy
}

// IsReady returns true if the service is ready
func (hc *HealthChecker) IsReady() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	for _, check := range hc.status.Checks {
		if check.Type == CheckTypeReadiness && check.Status != StatusHealthy {
			return false
		}
	}
	
	return true
}

// GetLivenessStatus returns liveness status
func (hc *HealthChecker) GetLivenessStatus() *HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	status := &HealthStatus{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Checks:    make(map[string]*CheckResult),
		Version:   hc.status.Version,
		Uptime:    hc.status.Uptime,
	}
	
	// Only check liveness checks
	for name, check := range hc.checks {
		if check.Type() == CheckTypeLiveness {
			if result, ok := hc.status.Checks[name]; ok {
				status.Checks[name] = result
				if result.Status == StatusUnhealthy && check.Critical() {
					status.Status = StatusUnhealthy
				}
			}
		}
	}
	
	return status
}

// GetReadinessStatus returns readiness status
func (hc *HealthChecker) GetReadinessStatus() *HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	status := &HealthStatus{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Checks:    make(map[string]*CheckResult),
		Version:   hc.status.Version,
		Uptime:    hc.status.Uptime,
	}
	
	// Check readiness checks
	for name, check := range hc.checks {
		if check.Type() == CheckTypeReadiness {
			if result, ok := hc.status.Checks[name]; ok {
				status.Checks[name] = result
				if result.Status == StatusUnhealthy {
					status.Status = StatusUnhealthy
				}
			}
		}
	}
	
	return status
}

// HTTPHandler returns an HTTP handler for health checks
func (hc *HealthChecker) HTTPHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := hc.GetStatus()
		
		httpStatus := http.StatusOK
		if status.Status == StatusUnhealthy {
			httpStatus = http.StatusServiceUnavailable
		} else if status.Status == StatusDegraded {
			httpStatus = http.StatusOK // Still return 200 for degraded
		}
		
		c.JSON(httpStatus, status)
	}
}

// LivenessHandler returns an HTTP handler for liveness checks
func (hc *HealthChecker) LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := hc.GetStatus()
		
		// Check only liveness checks
		isAlive := true
		for _, check := range status.Checks {
			if check.Type == CheckTypeLiveness && check.Status == StatusUnhealthy {
				isAlive = false
				break
			}
		}
		
		if isAlive {
			c.JSON(http.StatusOK, gin.H{
				"status": "alive",
				"timestamp": time.Now(),
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "dead",
				"timestamp": time.Now(),
			})
		}
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks
func (hc *HealthChecker) ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if hc.IsReady() {
			c.JSON(http.StatusOK, gin.H{
				"status": "ready",
				"timestamp": time.Now(),
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"timestamp": time.Now(),
			})
		}
	}
}

// Built-in health checks

// DatabaseCheck checks database connectivity
type DatabaseCheck struct {
	name     string
	db       interface{} // Your database connection
	critical bool
}

func NewDatabaseCheck(db interface{}) *DatabaseCheck {
	return &DatabaseCheck{
		name:     "database",
		db:       db,
		critical: true,
	}
}

func (dc *DatabaseCheck) Name() string { return dc.name }
func (dc *DatabaseCheck) Type() CheckType { return CheckTypeReadiness }
func (dc *DatabaseCheck) Critical() bool { return dc.critical }

func (dc *DatabaseCheck) Check(ctx context.Context) error {
	// Implement database ping
	// Example: return dc.db.PingContext(ctx)
	return nil
}

// CacheCheck checks cache connectivity
type CacheCheck struct {
	name     string
	cache    interface{} // Your cache connection
	critical bool
}

func NewCacheCheck(cache interface{}) *CacheCheck {
	return &CacheCheck{
		name:     "cache",
		cache:    cache,
		critical: false,
	}
}

func (cc *CacheCheck) Name() string { return cc.name }
func (cc *CacheCheck) Type() CheckType { return CheckTypeReadiness }
func (cc *CacheCheck) Critical() bool { return cc.critical }

func (cc *CacheCheck) Check(ctx context.Context) error {
	// Implement cache ping
	return nil
}

// DiskSpaceCheck checks available disk space
type DiskSpaceCheck struct {
	name      string
	path      string
	threshold float64 // Minimum free space percentage
}

func NewDiskSpaceCheck(path string, threshold float64) *DiskSpaceCheck {
	return &DiskSpaceCheck{
		name:      "disk_space",
		path:      path,
		threshold: threshold,
	}
}

func (dsc *DiskSpaceCheck) Name() string { return dsc.name }
func (dsc *DiskSpaceCheck) Type() CheckType { return CheckTypeLiveness }
func (dsc *DiskSpaceCheck) Critical() bool { return false }

func (dsc *DiskSpaceCheck) Check(ctx context.Context) error {
	// Implement disk space check
	// This would use syscall or a library to get disk usage
	return nil
}

// MemoryCheck checks memory usage
type MemoryCheck struct {
	name      string
	threshold float64 // Maximum memory usage percentage
}

func NewMemoryCheck(threshold float64) *MemoryCheck {
	return &MemoryCheck{
		name:      "memory",
		threshold: threshold,
	}
}

func (mc *MemoryCheck) Name() string { return mc.name }
func (mc *MemoryCheck) Type() CheckType { return CheckTypeLiveness }
func (mc *MemoryCheck) Critical() bool { return false }

func (mc *MemoryCheck) Check(ctx context.Context) error {
	// Implement memory check
	// This would use runtime.MemStats or similar
	return nil
}

// CloudProviderCheck checks cloud provider connectivity
type CloudProviderCheck struct {
	name     string
	provider string
	client   interface{}
}

func NewCloudProviderCheck(provider string, client interface{}) *CloudProviderCheck {
	return &CloudProviderCheck{
		name:     fmt.Sprintf("%s_provider", provider),
		provider: provider,
		client:   client,
	}
}

func (cpc *CloudProviderCheck) Name() string { return cpc.name }
func (cpc *CloudProviderCheck) Type() CheckType { return CheckTypeReadiness }
func (cpc *CloudProviderCheck) Critical() bool { return true }

func (cpc *CloudProviderCheck) Check(ctx context.Context) error {
	// Implement provider-specific health check
	switch cpc.provider {
	case "aws":
		// Check AWS connectivity
		return nil
	case "azure":
		// Check Azure connectivity
		return nil
	case "gcp":
		// Check GCP connectivity
		return nil
	default:
		return fmt.Errorf("unknown provider: %s", cpc.provider)
	}
}

// HTTPEndpointCheck checks HTTP endpoint availability
type HTTPEndpointCheck struct {
	name     string
	url      string
	timeout  time.Duration
	critical bool
}

func NewHTTPEndpointCheck(name, url string, critical bool) *HTTPEndpointCheck {
	return &HTTPEndpointCheck{
		name:     name,
		url:      url,
		timeout:  5 * time.Second,
		critical: critical,
	}
}

func (hec *HTTPEndpointCheck) Name() string { return hec.name }
func (hec *HTTPEndpointCheck) Type() CheckType { return CheckTypeReadiness }
func (hec *HTTPEndpointCheck) Critical() bool { return hec.critical }

func (hec *HTTPEndpointCheck) Check(ctx context.Context) error {
	client := &http.Client{
		Timeout: hec.timeout,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", hec.url, nil)
	if err != nil {
		return err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}
	
	return nil
}

// CustomCheck allows for custom health check implementations
type CustomCheck struct {
	name     string
	checkFn  func(context.Context) error
	checkType CheckType
	critical bool
}

func NewCustomCheck(name string, checkType CheckType, critical bool, checkFn func(context.Context) error) *CustomCheck {
	return &CustomCheck{
		name:      name,
		checkFn:   checkFn,
		checkType: checkType,
		critical:  critical,
	}
}

func (cc *CustomCheck) Name() string { return cc.name }
func (cc *CustomCheck) Type() CheckType { return cc.checkType }
func (cc *CustomCheck) Critical() bool { return cc.critical }
func (cc *CustomCheck) Check(ctx context.Context) error { return cc.checkFn(ctx) }

// MetricsExporter exports health metrics
type MetricsExporter struct {
	checker *HealthChecker
}

func NewMetricsExporter(checker *HealthChecker) *MetricsExporter {
	return &MetricsExporter{
		checker: checker,
	}
}

// Export exports metrics in Prometheus format
func (me *MetricsExporter) Export() string {
	status := me.checker.GetStatus()
	
	metrics := fmt.Sprintf("# HELP health_status Overall health status (1=healthy, 0=unhealthy)\n")
	metrics += fmt.Sprintf("# TYPE health_status gauge\n")
	
	healthValue := 0
	if status.Status == StatusHealthy {
		healthValue = 1
	}
	metrics += fmt.Sprintf("health_status %d\n", healthValue)
	
	metrics += fmt.Sprintf("# HELP health_check_status Individual health check status\n")
	metrics += fmt.Sprintf("# TYPE health_check_status gauge\n")
	
	for name, check := range status.Checks {
		checkValue := 0
		if check.Status == StatusHealthy {
			checkValue = 1
		}
		metrics += fmt.Sprintf("health_check_status{name=\"%s\",type=\"%s\"} %d\n", 
			name, check.Type, checkValue)
	}
	
	metrics += fmt.Sprintf("# HELP health_check_duration_seconds Health check duration\n")
	metrics += fmt.Sprintf("# TYPE health_check_duration_seconds gauge\n")
	
	for name, check := range status.Checks {
		metrics += fmt.Sprintf("health_check_duration_seconds{name=\"%s\"} %f\n", 
			name, check.Duration.Seconds())
	}
	
	return metrics
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled           bool          `yaml:"enabled"`
	Interval          time.Duration `yaml:"interval"`
	Timeout           time.Duration `yaml:"timeout"`
	StartupDelay      time.Duration `yaml:"startup_delay"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
	Checks            []CheckConfig `yaml:"checks"`
}

// CheckConfig represents individual check configuration
type CheckConfig struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Enabled  bool                   `yaml:"enabled"`
	Critical bool                   `yaml:"critical"`
	Config   map[string]interface{} `yaml:"config"`
}

// LoadHealthChecks loads health checks from configuration
func LoadHealthChecks(config *HealthCheckConfig) (*HealthChecker, error) {
	if !config.Enabled {
		return nil, nil
	}
	
	checker := NewHealthChecker(config.Interval)
	
	for _, checkConfig := range config.Checks {
		if !checkConfig.Enabled {
			continue
		}
		
		// Create appropriate check based on type
		var check Check
		switch checkConfig.Type {
		case "http":
			url := checkConfig.Config["url"].(string)
			check = NewHTTPEndpointCheck(checkConfig.Name, url, checkConfig.Critical)
		case "custom":
			// Would need to be registered programmatically
			continue
		default:
			return nil, fmt.Errorf("unknown check type: %s", checkConfig.Type)
		}
		
		checker.RegisterCheck(check)
	}
	
	return checker, nil
}

// HealthCheckMiddleware creates middleware for health check endpoints
func HealthCheckMiddleware(checker *HealthChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip health check endpoints themselves
		if c.Request.URL.Path == "/health" || 
		   c.Request.URL.Path == "/health/live" || 
		   c.Request.URL.Path == "/health/ready" {
			c.Next()
			return
		}
		
		// Check if service is healthy before processing request
		if !checker.IsHealthy() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "service unhealthy",
			})
			return
		}
		
		c.Next()
	}
}