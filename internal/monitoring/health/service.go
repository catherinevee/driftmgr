package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// HealthService provides comprehensive health monitoring
type HealthService struct {
	monitor  *HealthMonitor
	checkers map[string]HealthChecker
	eventBus EventBus
	mu       sync.RWMutex
	interval time.Duration
	stopChan chan struct{}
	running  bool
}

// NewHealthService creates a new health monitoring service
func NewHealthService(eventBus EventBus) *HealthService {
	monitor := NewHealthMonitor(eventBus)
	service := &HealthService{
		monitor:  monitor,
		checkers: make(map[string]HealthChecker),
		eventBus: eventBus,
		interval: 5 * time.Minute, // Default 5-minute interval
		stopChan: make(chan struct{}),
	}

	// Register default health checkers
	service.registerDefaultCheckers()

	return service
}

// registerDefaultCheckers registers default health checkers
func (hs *HealthService) registerDefaultCheckers() {
	// Note: Default checkers would be registered here
	// For now, we'll skip this to avoid import cycles
	// In a real implementation, you would register checkers from external packages
}

// RegisterChecker registers a health checker
func (hs *HealthService) RegisterChecker(checker HealthChecker) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.checkers[checker.GetType()] = checker
	hs.monitor.RegisterHealthChecker(checker)
}

// SetMonitoringInterval sets the monitoring interval
func (hs *HealthService) SetMonitoringInterval(interval time.Duration) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.interval = interval
}

// Start starts the health monitoring service
func (hs *HealthService) Start(ctx context.Context) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.running {
		return fmt.Errorf("health service is already running")
	}

	hs.running = true
	go hs.monitoringLoop(ctx)

	return nil
}

// Stop stops the health monitoring service
func (hs *HealthService) Stop() {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if !hs.running {
		return
	}

	hs.running = false
	close(hs.stopChan)
}

// monitoringLoop runs the continuous monitoring loop
func (hs *HealthService) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(hs.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hs.stopChan:
			return
		case <-ticker.C:
			// Perform periodic health checks
			hs.performPeriodicChecks(ctx)
		}
	}
}

// performPeriodicChecks performs periodic health checks
func (hs *HealthService) performPeriodicChecks(ctx context.Context) {
	// This would typically fetch all resources and perform health checks
	// For now, we'll just log that periodic checks are running
	fmt.Println("Performing periodic health checks...")
}

// CheckResourceHealth checks the health of a specific resource
func (hs *HealthService) CheckResourceHealth(ctx context.Context, resource *models.Resource) (*HealthReport, error) {
	return hs.monitor.CheckResourceHealth(ctx, resource)
}

// GetResourceHealth returns the health report for a resource
func (hs *HealthService) GetResourceHealth(resourceID string) (*HealthReport, error) {
	return hs.monitor.GetResourceHealth(resourceID)
}

// ListUnhealthyResources returns resources with unhealthy status
func (hs *HealthService) ListUnhealthyResources() []*HealthReport {
	return hs.monitor.ListUnhealthyResources()
}

// GetHealthMetrics returns health metrics for a resource
func (hs *HealthService) GetHealthMetrics(resourceID string, timeRange time.Duration) ([]HealthMetric, error) {
	return hs.monitor.GetHealthMetrics(resourceID, timeRange)
}

// RecordHealthMetric records a health metric
func (hs *HealthService) RecordHealthMetric(resourceID string, metric HealthMetric) {
	hs.monitor.RecordHealthMetric(resourceID, metric)
}

// GetHealthSummary returns a summary of all resource health
func (hs *HealthService) GetHealthSummary() map[string]interface{} {
	return hs.monitor.GetHealthSummary()
}

// SetAlertThreshold sets alerting thresholds for a metric
func (hs *HealthService) SetAlertThreshold(metricName string, threshold AlertThreshold) {
	hs.monitor.SetAlertThreshold(metricName, threshold)
}

// BulkHealthCheck performs health checks on multiple resources
func (hs *HealthService) BulkHealthCheck(ctx context.Context, resources []*models.Resource) ([]*HealthReport, error) {
	var reports []*HealthReport
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Limit concurrent health checks
	semaphore := make(chan struct{}, 10) // Max 10 concurrent checks

	for _, resource := range resources {
		wg.Add(1)
		go func(resource *models.Resource) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			report, err := hs.CheckResourceHealth(ctx, resource)
			if err != nil {
				// Create error report
				report = &HealthReport{
					ResourceID:    resource.ID,
					ResourceType:  resource.Type,
					Provider:      resource.Provider,
					Region:        resource.Region,
					OverallStatus: HealthStatusUnknown,
					HealthScore:   0.0,
					LastUpdated:   time.Now(),
					Checks: []HealthCheck{
						{
							ID:          fmt.Sprintf("error-%s", resource.ID),
							Name:        "Health Check Error",
							Type:        "error",
							ResourceID:  resource.ID,
							Status:      HealthStatusUnknown,
							Message:     fmt.Sprintf("Health check failed: %v", err),
							LastChecked: time.Now(),
						},
					},
					Recommendations: []string{fmt.Sprintf("Investigate health check failure: %v", err)},
				}
			}

			mu.Lock()
			reports = append(reports, report)
			mu.Unlock()
		}(resource)
	}

	wg.Wait()
	return reports, nil
}

// GetHealthTrends returns health trends over time
func (hs *HealthService) GetHealthTrends(resourceID string, timeRange time.Duration) ([]HealthTrend, error) {
	// This would typically query a time-series database
	// For now, return mock data
	trends := []HealthTrend{
		{
			Timestamp:   time.Now().Add(-24 * time.Hour),
			HealthScore: 95.0,
			Status:      HealthStatusHealthy,
		},
		{
			Timestamp:   time.Now().Add(-12 * time.Hour),
			HealthScore: 92.0,
			Status:      HealthStatusHealthy,
		},
		{
			Timestamp:   time.Now().Add(-6 * time.Hour),
			HealthScore: 88.0,
			Status:      HealthStatusWarning,
		},
		{
			Timestamp:   time.Now(),
			HealthScore: 85.0,
			Status:      HealthStatusWarning,
		},
	}

	return trends, nil
}

// GetHealthAlerts returns active health alerts
func (hs *HealthService) GetHealthAlerts() ([]HealthAlert, error) {
	// This would typically query an alerting system
	// For now, return mock data
	alerts := []HealthAlert{
		{
			ID:         "alert-1",
			ResourceID: "resource-1",
			Type:       "critical",
			Message:    "Resource health score below 50%",
			Timestamp:  time.Now().Add(-1 * time.Hour),
			Status:     "active",
		},
		{
			ID:         "alert-2",
			ResourceID: "resource-2",
			Type:       "warning",
			Message:    "Resource has no tags",
			Timestamp:  time.Now().Add(-2 * time.Hour),
			Status:     "active",
		},
	}

	return alerts, nil
}

// IsRunning returns whether the health service is running
func (hs *HealthService) IsRunning() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.running
}

// GetMonitoringInterval returns the current monitoring interval
func (hs *HealthService) GetMonitoringInterval() time.Duration {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.interval
}

// GetRegisteredCheckers returns the list of registered health checkers
func (hs *HealthService) GetRegisteredCheckers() []string {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	var checkers []string
	for checkerType := range hs.checkers {
		checkers = append(checkers, checkerType)
	}

	return checkers
}
