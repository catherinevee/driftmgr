package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// HealthMonitor manages health monitoring for resources
type HealthMonitor struct {
	mu              sync.RWMutex
	healthChecks    map[string][]HealthCheck
	healthMetrics   map[string][]HealthMetric
	healthReports   map[string]*HealthReport
	checkers        map[string]HealthChecker
	alertThresholds map[string]AlertThreshold
	eventBus        EventBus
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(eventBus EventBus) *HealthMonitor {
	return &HealthMonitor{
		healthChecks:    make(map[string][]HealthCheck),
		healthMetrics:   make(map[string][]HealthMetric),
		healthReports:   make(map[string]*HealthReport),
		checkers:        make(map[string]HealthChecker),
		alertThresholds: make(map[string]AlertThreshold),
		eventBus:        eventBus,
	}
}

// RegisterHealthChecker registers a health checker
func (hm *HealthMonitor) RegisterHealthChecker(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checkers[checker.GetType()] = checker
}

// SetAlertThreshold sets alerting thresholds for a metric
func (hm *HealthMonitor) SetAlertThreshold(metricName string, threshold AlertThreshold) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.alertThresholds[metricName] = threshold
}

// CheckResourceHealth performs health checks on a resource
func (hm *HealthMonitor) CheckResourceHealth(ctx context.Context, resource *models.Resource) (*HealthReport, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	report := &HealthReport{
		ResourceID:      resource.ID,
		ResourceType:    resource.Type,
		Provider:        resource.Provider,
		Region:          resource.Region,
		LastUpdated:     time.Now(),
		Checks:          []HealthCheck{},
		Metrics:         []HealthMetric{},
		Recommendations: []string{},
		Metadata:        make(map[string]interface{}),
	}

	// Run all applicable health checks
	for _, checker := range hm.checkers {
		check, err := checker.Check(ctx, resource)
		if err != nil {
			// Log error but continue with other checks
			check = &HealthCheck{
				ID:          fmt.Sprintf("%s-%s", checker.GetType(), resource.ID),
				Name:        checker.GetType(),
				Type:        checker.GetType(),
				ResourceID:  resource.ID,
				Status:      HealthStatusUnknown,
				Message:     fmt.Sprintf("Health check failed: %v", err),
				LastChecked: time.Now(),
			}
		}
		report.Checks = append(report.Checks, *check)
	}

	// Calculate overall health status and score
	report.OverallStatus = hm.calculateOverallStatus(report.Checks)
	report.HealthScore = hm.calculateHealthScore(report.Checks)

	// Generate recommendations
	report.Recommendations = hm.generateRecommendations(report)

	// Store the report
	hm.healthReports[resource.ID] = report

	// Publish health event if status changed
	hm.publishHealthEvent(report)

	return report, nil
}

// GetResourceHealth returns the health report for a resource
func (hm *HealthMonitor) GetResourceHealth(resourceID string) (*HealthReport, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	report, exists := hm.healthReports[resourceID]
	if !exists {
		return nil, fmt.Errorf("health report not found for resource %s", resourceID)
	}

	return report, nil
}

// ListUnhealthyResources returns resources with unhealthy status
func (hm *HealthMonitor) ListUnhealthyResources() []*HealthReport {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var unhealthy []*HealthReport
	for _, report := range hm.healthReports {
		if report.OverallStatus == HealthStatusCritical ||
			report.OverallStatus == HealthStatusWarning ||
			report.OverallStatus == HealthStatusDegraded {
			unhealthy = append(unhealthy, report)
		}
	}

	return unhealthy
}

// GetHealthMetrics returns health metrics for a resource
func (hm *HealthMonitor) GetHealthMetrics(resourceID string, timeRange time.Duration) ([]HealthMetric, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	metrics, exists := hm.healthMetrics[resourceID]
	if !exists {
		return nil, fmt.Errorf("health metrics not found for resource %s", resourceID)
	}

	// Filter metrics by time range
	cutoff := time.Now().Add(-timeRange)
	var filtered []HealthMetric
	for _, metric := range metrics {
		if metric.Timestamp.After(cutoff) {
			filtered = append(filtered, metric)
		}
	}

	return filtered, nil
}

// RecordHealthMetric records a health metric
func (hm *HealthMonitor) RecordHealthMetric(resourceID string, metric HealthMetric) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	metric.Timestamp = time.Now()

	// Determine status based on thresholds
	if threshold, exists := hm.alertThresholds[metric.Name]; exists {
		if metric.Value >= threshold.Critical {
			metric.Status = HealthStatusCritical
		} else if metric.Value >= threshold.Warning {
			metric.Status = HealthStatusWarning
		} else {
			metric.Status = HealthStatusHealthy
		}
	} else {
		metric.Status = HealthStatusUnknown
	}

	hm.healthMetrics[resourceID] = append(hm.healthMetrics[resourceID], metric)

	// Check if we need to trigger an alert
	hm.checkMetricAlert(resourceID, metric)
}

// calculateOverallStatus calculates the overall health status from checks
func (hm *HealthMonitor) calculateOverallStatus(checks []HealthCheck) HealthStatus {
	if len(checks) == 0 {
		return HealthStatusUnknown
	}

	criticalCount := 0
	warningCount := 0
	unknownCount := 0

	for _, check := range checks {
		switch check.Status {
		case HealthStatusCritical:
			criticalCount++
		case HealthStatusWarning, HealthStatusDegraded:
			warningCount++
		case HealthStatusUnknown:
			unknownCount++
		}
	}

	if criticalCount > 0 {
		return HealthStatusCritical
	}
	if warningCount > 0 {
		return HealthStatusWarning
	}
	if unknownCount == len(checks) {
		return HealthStatusUnknown
	}

	return HealthStatusHealthy
}

// calculateHealthScore calculates a health score (0-100)
func (hm *HealthMonitor) calculateHealthScore(checks []HealthCheck) float64 {
	if len(checks) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, check := range checks {
		switch check.Status {
		case HealthStatusHealthy:
			totalScore += 100.0
		case HealthStatusWarning:
			totalScore += 70.0
		case HealthStatusDegraded:
			totalScore += 50.0
		case HealthStatusCritical:
			totalScore += 0.0
		case HealthStatusUnknown:
			totalScore += 30.0
		}
	}

	return totalScore / float64(len(checks))
}

// generateRecommendations generates health recommendations
func (hm *HealthMonitor) generateRecommendations(report *HealthReport) []string {
	var recommendations []string

	for _, check := range report.Checks {
		switch check.Status {
		case HealthStatusCritical:
			recommendations = append(recommendations,
				fmt.Sprintf("CRITICAL: %s - %s", check.Name, check.Message))
		case HealthStatusWarning:
			recommendations = append(recommendations,
				fmt.Sprintf("WARNING: %s - %s", check.Name, check.Message))
		case HealthStatusDegraded:
			recommendations = append(recommendations,
				fmt.Sprintf("DEGRADED: %s - %s", check.Name, check.Message))
		}
	}

	// Add general recommendations based on health score
	if report.HealthScore < 50 {
		recommendations = append(recommendations,
			"Resource health is poor - immediate attention required")
	} else if report.HealthScore < 80 {
		recommendations = append(recommendations,
			"Resource health is below optimal - consider optimization")
	}

	return recommendations
}

// checkMetricAlert checks if a metric should trigger an alert
func (hm *HealthMonitor) checkMetricAlert(resourceID string, metric HealthMetric) {
	if threshold, exists := hm.alertThresholds[metric.Name]; exists {
		if metric.Value >= threshold.Critical || metric.Value >= threshold.Warning {
			event := HealthEvent{
				Type:       "metric_alert",
				ResourceID: resourceID,
				Status:     metric.Status,
				Message:    fmt.Sprintf("Metric %s exceeded threshold: %.2f", metric.Name, metric.Value),
				Timestamp:  time.Now(),
				Metadata: map[string]interface{}{
					"metric_name":  metric.Name,
					"metric_value": metric.Value,
					"threshold":    threshold,
				},
			}

			if hm.eventBus != nil {
				hm.eventBus.PublishHealthEvent(event)
			}
		}
	}
}

// publishHealthEvent publishes a health event
func (hm *HealthMonitor) publishHealthEvent(report *HealthReport) {
	if hm.eventBus == nil {
		return
	}

	event := HealthEvent{
		Type:       "health_status_change",
		ResourceID: report.ResourceID,
		Status:     report.OverallStatus,
		Message:    fmt.Sprintf("Health status: %s (Score: %.1f)", report.OverallStatus, report.HealthScore),
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"health_score":  report.HealthScore,
			"resource_type": report.ResourceType,
			"provider":      report.Provider,
			"region":        report.Region,
		},
	}

	hm.eventBus.PublishHealthEvent(event)
}

// GetHealthSummary returns a summary of all resource health
func (hm *HealthMonitor) GetHealthSummary() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	summary := map[string]interface{}{
		"total_resources": len(hm.healthReports),
		"healthy":         0,
		"warning":         0,
		"critical":        0,
		"unknown":         0,
		"degraded":        0,
		"average_score":   0.0,
	}

	totalScore := 0.0
	for _, report := range hm.healthReports {
		switch report.OverallStatus {
		case HealthStatusHealthy:
			summary["healthy"] = summary["healthy"].(int) + 1
		case HealthStatusWarning:
			summary["warning"] = summary["warning"].(int) + 1
		case HealthStatusCritical:
			summary["critical"] = summary["critical"].(int) + 1
		case HealthStatusUnknown:
			summary["unknown"] = summary["unknown"].(int) + 1
		case HealthStatusDegraded:
			summary["degraded"] = summary["degraded"].(int) + 1
		}
		totalScore += report.HealthScore
	}

	if len(hm.healthReports) > 0 {
		summary["average_score"] = totalScore / float64(len(hm.healthReports))
	}

	return summary
}
