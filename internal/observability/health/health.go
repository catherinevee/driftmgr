package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logger"
	"github.com/catherinevee/driftmgr/internal/telemetry"
	"go.opentelemetry.io/otel/trace"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Check represents a health check
type Check struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Timeout     time.Duration `json:"timeout"`
	Critical    bool          `json:"critical"`
	CheckFunc   func(context.Context) error
}

// Result represents the result of a health check
type Result struct {
	Name        string        `json:"name"`
	Status      Status        `json:"status"`
	Message     string        `json:"message,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	LastChecked time.Time     `json:"last_checked"`
}

// Report represents a complete health report
type Report struct {
	Status      Status    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	Version     string    `json:"version"`
	Uptime      time.Duration `json:"uptime"`
	Results     []Result  `json:"results"`
	TotalChecks int       `json:"total_checks"`
	Healthy     int       `json:"healthy"`
	Unhealthy   int       `json:"unhealthy"`
	Degraded    int       `json:"degraded"`
}

// Service manages health checks
type Service struct {
	checks      map[string]Check
	results     map[string]Result
	mu          sync.RWMutex
	startTime   time.Time
	version     string
	log         logger.Logger
}

// NewService creates a new health service
func NewService(version string) *Service {
	return &Service{
		checks:    make(map[string]Check),
		results:   make(map[string]Result),
		startTime: time.Now(),
		version:   version,
		log:       logger.New("health_service"),
	}
}

// RegisterCheck registers a health check
func (s *Service) RegisterCheck(check Check) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if check.Timeout <= 0 {
		check.Timeout = 5 * time.Second
	}
	
	s.checks[check.Name] = check
	
	s.log.Info("Registered health check",
		logger.String("name", check.Name),
		logger.Bool("critical", check.Critical),
		logger.Duration("timeout", check.Timeout),
	)
}

// UnregisterCheck removes a health check
func (s *Service) UnregisterCheck(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.checks, name)
	delete(s.results, name)
	
	s.log.Info("Unregistered health check",
		logger.String("name", name),
	)
}

// RunCheck runs a specific health check
func (s *Service) RunCheck(ctx context.Context, name string) (Result, error) {
	s.mu.RLock()
	check, exists := s.checks[name]
	s.mu.RUnlock()
	
	if !exists {
		return Result{}, nil
	}
	
	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()
	
	// Record telemetry
	if telemetry.Get() != nil {
		var span trace.Span
		checkCtx, span = telemetry.Get().StartSpan(checkCtx, "health.check")
		defer span.End()
	}
	
	start := time.Now()
	err := check.CheckFunc(checkCtx)
	duration := time.Since(start)
	
	result := Result{
		Name:        check.Name,
		Duration:    duration,
		LastChecked: time.Now(),
	}
	
	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
		s.log.Warn("Health check failed",
			logger.String("name", check.Name),
			logger.Error(err),
			logger.Duration("duration", duration),
		)
	} else {
		result.Status = StatusHealthy
		result.Message = "Check passed"
		s.log.Debug("Health check passed",
			logger.String("name", check.Name),
			logger.Duration("duration", duration),
		)
	}
	
	// Store result
	s.mu.Lock()
	s.results[name] = result
	s.mu.Unlock()
	
	return result, nil
}

// RunAllChecks runs all registered health checks
func (s *Service) RunAllChecks(ctx context.Context) Report {
	s.mu.RLock()
	checksCopy := make(map[string]Check)
	for k, v := range s.checks {
		checksCopy[k] = v
	}
	s.mu.RUnlock()
	
	report := Report{
		Timestamp:   time.Now(),
		Version:     s.version,
		Uptime:      time.Since(s.startTime),
		Results:     make([]Result, 0, len(checksCopy)),
		TotalChecks: len(checksCopy),
	}
	
	// Run checks concurrently
	var wg sync.WaitGroup
	resultsCh := make(chan Result, len(checksCopy))
	
	for name := range checksCopy {
		wg.Add(1)
		go func(checkName string) {
			defer wg.Done()
			result, _ := s.RunCheck(ctx, checkName)
			resultsCh <- result
		}(name)
	}
	
	wg.Wait()
	close(resultsCh)
	
	// Collect results
	overallStatus := StatusHealthy
	for result := range resultsCh {
		report.Results = append(report.Results, result)
		
		switch result.Status {
		case StatusHealthy:
			report.Healthy++
		case StatusUnhealthy:
			report.Unhealthy++
			if s.checks[result.Name].Critical {
				overallStatus = StatusUnhealthy
			} else if overallStatus != StatusUnhealthy {
				overallStatus = StatusDegraded
			}
		case StatusDegraded:
			report.Degraded++
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
	}
	
	report.Status = overallStatus
	
	s.log.Info("Health check report generated",
		logger.String("status", string(report.Status)),
		logger.Int("total", report.TotalChecks),
		logger.Int("healthy", report.Healthy),
		logger.Int("unhealthy", report.Unhealthy),
	)
	
	return report
}

// GetReport returns the latest health report
func (s *Service) GetReport() Report {
	ctx := context.Background()
	return s.RunAllChecks(ctx)
}

// GetCachedResults returns cached health check results
func (s *Service) GetCachedResults() map[string]Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	results := make(map[string]Result)
	for k, v := range s.results {
		results[k] = v
	}
	
	return results
}

// HTTPHandler returns an HTTP handler for health checks
func (s *Service) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Check for specific check name
		checkName := r.URL.Query().Get("check")
		
		if checkName != "" {
			// Run specific check
			result, err := s.RunCheck(ctx, checkName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			if result.Status != StatusHealthy {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			
			json.NewEncoder(w).Encode(result)
			return
		}
		
		// Run all checks
		report := s.RunAllChecks(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		
		// Set appropriate HTTP status
		switch report.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK) // Still operational
		case StatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		
		json.NewEncoder(w).Encode(report)
	}
}

// LivenessHandler returns a simple liveness check handler
func (s *Service) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "alive",
			"timestamp": time.Now(),
			"uptime":    time.Since(s.startTime).String(),
		})
	}
}

// ReadinessHandler returns a readiness check handler
func (s *Service) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		report := s.RunAllChecks(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		
		if report.Status == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ready":     report.Status != StatusUnhealthy,
			"status":    report.Status,
			"timestamp": report.Timestamp,
			"checks":    report.TotalChecks,
			"healthy":   report.Healthy,
		})
	}
}

// StartPeriodicChecks starts periodic health checks
func (s *Service) StartPeriodicChecks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.RunAllChecks(ctx)
			}
		}
	}()
	
	s.log.Info("Started periodic health checks",
		logger.Duration("interval", interval),
	)
}

// CheckType represents the type of health check
type CheckType string

const (
	CheckTypeLiveness  CheckType = "liveness"
	CheckTypeReadiness CheckType = "readiness"
)

// HealthChecker provides backward compatibility
type HealthChecker struct {
	service *Service
}

// NewHealthChecker creates a new health checker for backward compatibility
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	service := NewService("1.0.0")
	return &HealthChecker{
		service: service,
	}
}

// RegisterCheck registers a health check (backward compatibility)
func (h *HealthChecker) RegisterCheck(check interface{}) {
	// Convert old check format to new format
	// This is a stub for compatibility
}

// Start starts periodic checks (backward compatibility)
func (h *HealthChecker) Start(ctx context.Context) {
	h.service.StartPeriodicChecks(ctx, 30*time.Second)
}

// Check performs a health check (backward compatibility)
func (h *HealthChecker) Check(ctx context.Context, checkType CheckType) (map[string]interface{}, error) {
	report := h.service.RunAllChecks(ctx)
	
	result := map[string]interface{}{
		"status":    string(report.Status),
		"timestamp": report.Timestamp,
		"checks":    len(report.Results),
	}
	
	if checkType == CheckTypeReadiness {
		result["ready"] = report.Status != StatusUnhealthy
	} else if checkType == CheckTypeLiveness {
		result["alive"] = true
	}
	
	return result, nil
}

// GetStatus returns the overall health status (backward compatibility)
func (h *HealthChecker) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	return h.Check(ctx, "")
}

// GetLivenessStatus returns liveness status (backward compatibility)
func (h *HealthChecker) GetLivenessStatus(ctx context.Context) (map[string]interface{}, error) {
	return h.Check(ctx, CheckTypeLiveness)
}

// GetReadinessStatus returns readiness status (backward compatibility)
func (h *HealthChecker) GetReadinessStatus(ctx context.Context) (map[string]interface{}, error) {
	return h.Check(ctx, CheckTypeReadiness)
}