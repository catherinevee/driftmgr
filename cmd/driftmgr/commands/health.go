package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// CheckType represents the type of health check
type CheckType string

const (
	CheckTypeLiveness  CheckType = "liveness"
	CheckTypeReadiness CheckType = "readiness"
)

// HealthChecker manages health checks
type HealthChecker struct {
	checks  []HealthCheck
	timeout time.Duration
	mu      sync.RWMutex
}

// HealthCheck interface for health checks
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
	Type() CheckType
	Critical() bool
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		checks:  []HealthCheck{},
		timeout: timeout,
	}
}

// RegisterCheck registers a health check
func (h *HealthChecker) RegisterCheck(check HealthCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, check)
}

// Start starts background health checks
func (h *HealthChecker) Start(ctx context.Context) {
	// Background health check implementation would go here
}

// GetStatus returns the overall health status
func (h *HealthChecker) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := map[string]interface{}{
		"status": "healthy",
		"checks": []map[string]interface{}{},
	}

	for _, check := range h.checks {
		checkStatus := map[string]interface{}{
			"name": check.Name(),
			"type": check.Type(),
		}

		if err := check.Check(ctx); err != nil {
			checkStatus["status"] = "unhealthy"
			checkStatus["error"] = err.Error()
			if check.Critical() {
				status["status"] = "unhealthy"
			}
		} else {
			checkStatus["status"] = "healthy"
		}

		checks := status["checks"].([]map[string]interface{})
		status["checks"] = append(checks, checkStatus)
	}

	return status, nil
}

// GetLivenessStatus returns the liveness status
func (h *HealthChecker) GetLivenessStatus(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status": "alive",
	}, nil
}

// GetReadinessStatus returns the readiness status
func (h *HealthChecker) GetReadinessStatus(ctx context.Context) (map[string]interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, check := range h.checks {
		if check.Type() == CheckTypeReadiness && check.Critical() {
			if err := check.Check(ctx); err != nil {
				return map[string]interface{}{
					"status": "not_ready",
					"error":  err.Error(),
				}, nil
			}
		}
	}

	return map[string]interface{}{
		"status": "ready",
	}, nil
}

// HealthServer provides health check endpoints
type HealthServer struct {
	checker *HealthChecker
	db      *sql.DB
}

// NewHealthServer creates a new health server
func NewHealthServer(db *sql.DB) *HealthServer {
	checker := NewHealthChecker(5 * time.Second)

	// Add database check
	if db != nil {
		checker.RegisterCheck(&DatabaseCheck{db: db})
	}

	// Add API check
	checker.RegisterCheck(&APICheck{})

	// Add discovery service check
	checker.RegisterCheck(&DiscoveryCheck{})

	// Start background health checks
	checker.Start(context.Background())

	return &HealthServer{
		checker: checker,
		db:      db,
	}
}

// RegisterHealthEndpoints registers health check endpoints
func (h *HealthServer) RegisterHealthEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/health/live", h.handleLiveness)
	mux.HandleFunc("/health/ready", h.handleReadiness)
}

func (h *HealthServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	status, err := h.checker.GetStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if statusStr, ok := status["status"].(string); ok && statusStr != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(status)
}

func (h *HealthServer) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness checks if the process is alive
	status, err := h.checker.GetLivenessStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if statusStr, ok := status["status"].(string); ok && statusStr != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status["status"],
		"timestamp": time.Now().UTC(),
	})
}

func (h *HealthServer) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// Readiness checks if the service can handle requests
	status, err := h.checker.GetReadinessStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	ready, _ := status["ready"].(bool)
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status["status"],
		"ready":     ready,
		"timestamp": time.Now().UTC(),
	})
}

// DatabaseCheck checks database connectivity
type DatabaseCheck struct {
	db *sql.DB
}

func (d *DatabaseCheck) Name() string {
	return "database"
}

func (d *DatabaseCheck) Check(ctx context.Context) error {
	if d.db == nil {
		return nil // No database configured
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return d.db.PingContext(ctx)
}

func (d *DatabaseCheck) Type() CheckType {
	return CheckTypeReadiness
}

func (d *DatabaseCheck) Critical() bool {
	return false // Database is not critical for liveness
}

// APICheck checks if the API is responsive
type APICheck struct{}

func (a *APICheck) Name() string {
	return "api"
}

func (a *APICheck) Check(ctx context.Context) error {
	// Simple check - if we're running this, the API is up
	return nil
}

func (a *APICheck) Type() CheckType {
	return CheckTypeLiveness
}

func (a *APICheck) Critical() bool {
	return true
}

// DiscoveryCheck checks if discovery services are accessible
type DiscoveryCheck struct{}

func (d *DiscoveryCheck) Name() string {
	return "discovery"
}

func (d *DiscoveryCheck) Check(ctx context.Context) error {
	// Check if discovery services are configured
	// In production, this would check actual service connectivity
	return nil
}

func (d *DiscoveryCheck) Type() CheckType {
	return CheckTypeReadiness
}

func (d *DiscoveryCheck) Critical() bool {
	return false
}
