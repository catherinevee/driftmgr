package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/shared/events"
)

// DashboardService provides real-time event dashboard functionality
type DashboardService struct {
	eventBus        *events.EventBus
	notificationSvc *NotificationService
	aggregator      *EventAggregator
	metrics         *DashboardMetrics
	mu              sync.RWMutex
	config          *DashboardConfig
	active          bool
	stopChan        chan struct{}
	wg              sync.WaitGroup
	subscription    *events.Subscription
}

// DashboardMetrics tracks dashboard metrics
type DashboardMetrics struct {
	TotalEvents      int64
	EventsByType     map[events.EventType]int64
	EventsBySeverity map[string]int64
	EventsBySource   map[string]int64
	RecentEvents     []events.Event
	ActiveAlerts     []Alert
	SystemHealth     SystemHealth
	LastUpdated      time.Time
	mu               sync.RWMutex
}

// Alert represents a system alert
type Alert struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Severity   string                 `json:"severity"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
	Data       map[string]interface{} `json:"data"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// SystemHealth represents system health status
type SystemHealth struct {
	Status     string            `json:"status"`
	Score      int               `json:"score"`
	Components map[string]string `json:"components"`
	LastCheck  time.Time         `json:"last_check"`
	Issues     []string          `json:"issues"`
}

// DashboardConfig contains configuration for the dashboard service
type DashboardConfig struct {
	Enabled             bool            `json:"enabled"`
	UpdateInterval      time.Duration   `json:"update_interval"`
	MaxRecentEvents     int             `json:"max_recent_events"`
	AlertThresholds     AlertThresholds `json:"alert_thresholds"`
	HealthCheckInterval time.Duration   `json:"health_check_interval"`
}

// AlertThresholds defines thresholds for generating alerts
type AlertThresholds struct {
	ErrorRate    float64 `json:"error_rate"`
	DriftCount   int     `json:"drift_count"`
	FailedJobs   int     `json:"failed_jobs"`
	ResponseTime int     `json:"response_time_ms"`
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(eventBus *events.EventBus, notificationSvc *NotificationService, aggregator *EventAggregator, config *DashboardConfig) *DashboardService {
	if config == nil {
		config = &DashboardConfig{
			Enabled:             true,
			UpdateInterval:      5 * time.Second,
			MaxRecentEvents:     1000,
			HealthCheckInterval: 30 * time.Second,
			AlertThresholds: AlertThresholds{
				ErrorRate:    0.05, // 5%
				DriftCount:   10,
				FailedJobs:   5,
				ResponseTime: 5000, // 5 seconds
			},
		}
	}

	return &DashboardService{
		eventBus:        eventBus,
		notificationSvc: notificationSvc,
		aggregator:      aggregator,
		metrics: &DashboardMetrics{
			EventsByType:     make(map[events.EventType]int64),
			EventsBySeverity: make(map[string]int64),
			EventsBySource:   make(map[string]int64),
			RecentEvents:     make([]events.Event, 0),
			ActiveAlerts:     make([]Alert, 0),
			SystemHealth: SystemHealth{
				Status:     "healthy",
				Score:      100,
				Components: make(map[string]string),
				LastCheck:  time.Now(),
				Issues:     make([]string, 0),
			},
		},
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start starts the dashboard service
func (ds *DashboardService) Start(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.active {
		return fmt.Errorf("dashboard service already active")
	}

	ds.active = true

	// Subscribe to all events
	filter := events.EventFilter{
		Types: []events.EventType{
			events.EventDiscoveryStarted,
			events.EventDiscoveryProgress,
			events.EventDiscoveryCompleted,
			events.EventDiscoveryFailed,
			events.EventDriftDetected,
			events.EventRemediationStarted,
			events.EventRemediationProgress,
			events.EventRemediationCompleted,
			events.EventRemediationFailed,
			events.EventHealthCheck,
			events.EventConfigChanged,
			events.EventAuditLog,
			events.EventJobQueued,
			events.EventJobStarted,
			events.EventJobCompleted,
			events.EventJobFailed,
			events.EventResourceDeleted,
			events.EventResourceImported,
		},
	}

	ds.subscription = ds.eventBus.Subscribe(ctx, filter, 1000)

	// Start background tasks
	ds.wg.Add(3)
	go ds.eventProcessor(ctx)
	go ds.metricsUpdater(ctx)
	go ds.healthChecker(ctx)

	log.Println("Dashboard service started")
	return nil
}

// Stop stops the dashboard service
func (ds *DashboardService) Stop() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.active {
		return
	}

	ds.active = false
	close(ds.stopChan)
	ds.wg.Wait()

	if ds.subscription != nil {
		ds.eventBus.Unsubscribe(ds.subscription)
	}

	log.Println("Dashboard service stopped")
}

// eventProcessor processes events for the dashboard
func (ds *DashboardService) eventProcessor(ctx context.Context) {
	defer ds.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopChan:
			return
		case event := <-ds.subscription.Channel:
			ds.processEvent(event)
		}
	}
}

// processEvent processes a single event for dashboard metrics
func (ds *DashboardService) processEvent(event events.Event) {
	ds.metrics.mu.Lock()
	defer ds.metrics.mu.Unlock()

	// Update metrics
	ds.metrics.TotalEvents++
	ds.metrics.EventsByType[event.Type]++
	ds.metrics.EventsBySource[event.Source]++

	// Determine severity
	severity := ds.determineSeverity(event)
	ds.metrics.EventsBySeverity[severity]++

	// Add to recent events
	ds.metrics.RecentEvents = append(ds.metrics.RecentEvents, event)
	if len(ds.metrics.RecentEvents) > ds.config.MaxRecentEvents {
		ds.metrics.RecentEvents = ds.metrics.RecentEvents[1:]
	}

	// Check for alerts
	ds.checkForAlerts(event)

	ds.metrics.LastUpdated = time.Now()
}

// determineSeverity determines the severity of an event
func (ds *DashboardService) determineSeverity(event events.Event) string {
	switch event.Type {
	case events.EventDiscoveryFailed, events.EventRemediationFailed, events.EventJobFailed:
		return "error"
	case events.EventDriftDetected:
		return "warning"
	case events.EventDiscoveryStarted, events.EventRemediationStarted, events.EventHealthCheck, events.EventConfigChanged, events.EventAuditLog:
		return "info"
	case events.EventDiscoveryCompleted, events.EventRemediationCompleted, events.EventJobCompleted:
		return "success"
	default:
		return "info"
	}
}

// checkForAlerts checks if an event should generate an alert
func (ds *DashboardService) checkForAlerts(event events.Event) {
	// Check for high drift count
	if event.Type == events.EventDriftDetected {
		if driftCount, ok := event.Data["drift_count"].(int); ok && driftCount >= ds.config.AlertThresholds.DriftCount {
			ds.createAlert("high_drift_count", "warning", "High Drift Count Detected",
				fmt.Sprintf("Detected %d drifts, exceeding threshold of %d", driftCount, ds.config.AlertThresholds.DriftCount),
				event)
		}
	}

	// Check for failed jobs
	if event.Type == events.EventJobFailed {
		ds.createAlert("job_failed", "error", "Job Failed",
			fmt.Sprintf("Job failed: %s", event.Data["job_id"]), event)
	}

	// Check for health check failures
	if event.Type == events.EventHealthCheck {
		if status, ok := event.Data["status"].(string); ok && status != "healthy" {
			ds.createAlert("health_check_failed", "error", "Health Check Failed",
				fmt.Sprintf("Health check failed: %s", event.Data["error"]), event)
		}
	}

	// Check for remediation failures
	if event.Type == events.EventRemediationFailed {
		ds.createAlert("remediation_failed", "error", "Remediation Failed",
			fmt.Sprintf("Remediation failed for resource: %s", event.Data["resource_id"]), event)
	}
}

// createAlert creates a new alert
func (ds *DashboardService) createAlert(alertType, severity, title, message string, event events.Event) {
	alert := Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Type:      alertType,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Timestamp: event.Timestamp,
		Source:    event.Source,
		Data:      event.Data,
		Resolved:  false,
	}

	ds.metrics.ActiveAlerts = append(ds.metrics.ActiveAlerts, alert)

	// Send notification if notification service is available
	if ds.notificationSvc != nil {
		notification := &NotificationMessage{
			ID:        alert.ID,
			Type:      "alert",
			Title:     alert.Title,
			Message:   alert.Message,
			Severity:  alert.Severity,
			Timestamp: alert.Timestamp,
			Data:      alert.Data,
		}

		ds.notificationSvc.BroadcastNotification(notification)
	}

	log.Printf("Alert created: %s - %s", alert.Title, alert.Message)
}

// metricsUpdater updates dashboard metrics periodically
func (ds *DashboardService) metricsUpdater(ctx context.Context) {
	defer ds.wg.Done()

	ticker := time.NewTicker(ds.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopChan:
			return
		case <-ticker.C:
			ds.updateMetrics()
		}
	}
}

// updateMetrics updates dashboard metrics
func (ds *DashboardService) updateMetrics() {
	ds.metrics.mu.Lock()
	defer ds.metrics.mu.Unlock()

	// Update system health
	ds.updateSystemHealth()

	// Clean up old alerts (older than 24 hours)
	now := time.Now()
	activeAlerts := make([]Alert, 0)
	for _, alert := range ds.metrics.ActiveAlerts {
		if now.Sub(alert.Timestamp) < 24*time.Hour {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	ds.metrics.ActiveAlerts = activeAlerts

	ds.metrics.LastUpdated = time.Now()
}

// updateSystemHealth updates system health status
func (ds *DashboardService) updateSystemHealth() {
	health := &ds.metrics.SystemHealth
	health.LastCheck = time.Now()
	health.Issues = make([]string, 0)

	// Calculate error rate
	totalEvents := ds.metrics.TotalEvents
	errorEvents := ds.metrics.EventsBySeverity["error"]
	errorRate := float64(0)
	if totalEvents > 0 {
		errorRate = float64(errorEvents) / float64(totalEvents)
	}

	// Check error rate threshold
	if errorRate > ds.config.AlertThresholds.ErrorRate {
		health.Issues = append(health.Issues, fmt.Sprintf("High error rate: %.2f%%", errorRate*100))
	}

	// Check active alerts
	if len(ds.metrics.ActiveAlerts) > 0 {
		health.Issues = append(health.Issues, fmt.Sprintf("%d active alerts", len(ds.metrics.ActiveAlerts)))
	}

	// Check failed jobs
	failedJobs := ds.metrics.EventsByType[events.EventJobFailed]
	if failedJobs >= int64(ds.config.AlertThresholds.FailedJobs) {
		health.Issues = append(health.Issues, fmt.Sprintf("%d failed jobs", failedJobs))
	}

	// Update health status and score
	if len(health.Issues) == 0 {
		health.Status = "healthy"
		health.Score = 100
	} else if len(health.Issues) <= 2 {
		health.Status = "warning"
		health.Score = 75
	} else {
		health.Status = "critical"
		health.Score = 25
	}

	// Update component health
	health.Components["event_bus"] = "healthy"
	health.Components["notification_service"] = "healthy"
	health.Components["aggregator"] = "healthy"
	if ds.notificationSvc != nil {
		subscriberCount := ds.notificationSvc.GetSubscriberCount()
		if subscriberCount > 0 {
			health.Components["notification_service"] = "active"
		}
	}
}

// healthChecker performs periodic health checks
func (ds *DashboardService) healthChecker(ctx context.Context) {
	defer ds.wg.Done()

	ticker := time.NewTicker(ds.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopChan:
			return
		case <-ticker.C:
			ds.performHealthCheck()
		}
	}
}

// performHealthCheck performs a comprehensive health check
func (ds *DashboardService) performHealthCheck() {
	// Check event bus health
	if ds.eventBus == nil {
		ds.createHealthAlert("event_bus_unavailable", "critical", "Event Bus Unavailable", "Event bus is not available")
	}

	// Check notification service health
	if ds.notificationSvc == nil {
		ds.createHealthAlert("notification_service_unavailable", "warning", "Notification Service Unavailable", "Notification service is not available")
	}

	// Check aggregator health
	if ds.aggregator == nil {
		ds.createHealthAlert("aggregator_unavailable", "warning", "Aggregator Unavailable", "Event aggregator is not available")
	}
}

// createHealthAlert creates a health-related alert
func (ds *DashboardService) createHealthAlert(alertType, severity, title, message string) {
	alert := Alert{
		ID:        fmt.Sprintf("health_%d", time.Now().UnixNano()),
		Type:      alertType,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Source:    "health_checker",
		Data:      make(map[string]interface{}),
		Resolved:  false,
	}

	ds.metrics.mu.Lock()
	ds.metrics.ActiveAlerts = append(ds.metrics.ActiveAlerts, alert)
	ds.metrics.mu.Unlock()

	log.Printf("Health alert created: %s - %s", alert.Title, alert.Message)
}

// GetDashboardData returns comprehensive dashboard data
func (ds *DashboardService) GetDashboardData() map[string]interface{} {
	ds.metrics.mu.RLock()
	defer ds.metrics.mu.RUnlock()

	// Get aggregator metrics if available
	aggregatorMetrics := make(map[string]interface{})
	if ds.aggregator != nil {
		aggregatorMetrics = ds.aggregator.GetMetrics()
	}

	// Get notification service metrics if available
	notificationMetrics := make(map[string]interface{})
	if ds.notificationSvc != nil {
		notificationMetrics = map[string]interface{}{
			"subscriber_count": ds.notificationSvc.GetSubscriberCount(),
		}
	}

	return map[string]interface{}{
		"metrics": map[string]interface{}{
			"total_events":       ds.metrics.TotalEvents,
			"events_by_type":     ds.metrics.EventsByType,
			"events_by_severity": ds.metrics.EventsBySeverity,
			"events_by_source":   ds.metrics.EventsBySource,
			"last_updated":       ds.metrics.LastUpdated,
		},
		"recent_events": ds.metrics.RecentEvents,
		"active_alerts": ds.metrics.ActiveAlerts,
		"system_health": ds.metrics.SystemHealth,
		"aggregator":    aggregatorMetrics,
		"notifications": notificationMetrics,
		"config":        ds.config,
	}
}

// GetRecentEvents returns recent events
func (ds *DashboardService) GetRecentEvents(limit int) []events.Event {
	ds.metrics.mu.RLock()
	defer ds.metrics.mu.RUnlock()

	if limit <= 0 || limit > len(ds.metrics.RecentEvents) {
		limit = len(ds.metrics.RecentEvents)
	}

	// Return most recent events
	start := len(ds.metrics.RecentEvents) - limit
	return ds.metrics.RecentEvents[start:]
}

// GetActiveAlerts returns active alerts
func (ds *DashboardService) GetActiveAlerts() []Alert {
	ds.metrics.mu.RLock()
	defer ds.metrics.mu.RUnlock()

	alerts := make([]Alert, len(ds.metrics.ActiveAlerts))
	copy(alerts, ds.metrics.ActiveAlerts)
	return alerts
}

// ResolveAlert resolves an alert by ID
func (ds *DashboardService) ResolveAlert(alertID string) error {
	ds.metrics.mu.Lock()
	defer ds.metrics.mu.Unlock()

	for i, alert := range ds.metrics.ActiveAlerts {
		if alert.ID == alertID {
			now := time.Now()
			ds.metrics.ActiveAlerts[i].Resolved = true
			ds.metrics.ActiveAlerts[i].ResolvedAt = &now
			log.Printf("Alert resolved: %s", alertID)
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// GetSystemHealth returns system health status
func (ds *DashboardService) GetSystemHealth() SystemHealth {
	ds.metrics.mu.RLock()
	defer ds.metrics.mu.RUnlock()

	return ds.metrics.SystemHealth
}

// ExportMetrics exports metrics in JSON format
func (ds *DashboardService) ExportMetrics() ([]byte, error) {
	data := ds.GetDashboardData()
	return json.MarshalIndent(data, "", "  ")
}
