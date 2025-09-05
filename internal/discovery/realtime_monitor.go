package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// RealTimeMonitor provides real-time monitoring of discovery operations
type RealTimeMonitor struct {
	mu              sync.RWMutex
	eventBus        *events.EventBus
	subscriptions   map[string]*events.Subscription
	metrics         *MonitoringMetrics
	alerts          []Alert
	webhooks        []WebhookConfig
	streamClients   map[string]chan Event
	running         bool
	stopCh          chan struct{}
	refreshInterval time.Duration
}

// MonitoringMetrics tracks real-time metrics
type MonitoringMetrics struct {
	ResourcesDiscovered  int64
	ResourcesPerSecond   float64
	ErrorCount           int64
	WarningCount         int64
	LastUpdateTime       time.Time
	AverageDiscoveryTime time.Duration
	ActiveProviders      map[string]bool
	ActiveRegions        map[string]bool
	DiscoveryRate        map[string]float64 // provider -> rate
	ErrorRate            float64
	SuccessRate          float64
}

// Alert represents a monitoring alert
type Alert struct {
	ID           string
	Type         AlertType
	Severity     AlertSeverity
	Message      string
	Details      map[string]interface{}
	Timestamp    time.Time
	Acknowledged bool
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeError         AlertType = "error"
	AlertTypeWarning       AlertType = "warning"
	AlertTypeInfo          AlertType = "info"
	AlertTypeRateLimit     AlertType = "rate_limit"
	AlertTypeQuotaExceeded AlertType = "quota_exceeded"
	AlertTypeSlowDiscovery AlertType = "slow_discovery"
)

// AlertSeverity defines the severity of an alert
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
)

// WebhookConfig defines webhook configuration
type WebhookConfig struct {
	URL        string
	Events     []string
	Headers    map[string]string
	RetryCount int
	Timeout    time.Duration
}

// Event represents a monitoring event
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Provider  string                 `json:"provider"`
	Region    string                 `json:"region"`
	Data      map[string]interface{} `json:"data"`
}

// NewRealTimeMonitor creates a new real-time monitor
func NewRealTimeMonitor() *RealTimeMonitor {
	return &RealTimeMonitor{
		eventBus:      events.NewEventBus(),
		subscriptions: make(map[string]*events.Subscription),
		metrics: &MonitoringMetrics{
			ActiveProviders: make(map[string]bool),
			ActiveRegions:   make(map[string]bool),
			DiscoveryRate:   make(map[string]float64),
			LastUpdateTime:  time.Now(),
		},
		alerts:          make([]Alert, 0),
		webhooks:        make([]WebhookConfig, 0),
		streamClients:   make(map[string]chan Event),
		stopCh:          make(chan struct{}),
		refreshInterval: 1 * time.Second,
	}
}

// Start begins real-time monitoring
func (rtm *RealTimeMonitor) Start(ctx context.Context) error {
	rtm.mu.Lock()
	if rtm.running {
		rtm.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	rtm.running = true
	rtm.mu.Unlock()

	// Subscribe to discovery events
	rtm.subscribeToEvents()

	// Start metrics collection
	go rtm.collectMetrics(ctx)

	// Start alert processor
	go rtm.processAlerts(ctx)

	// Start webhook processor
	go rtm.processWebhooks(ctx)

	return nil
}

// Stop stops the real-time monitor
func (rtm *RealTimeMonitor) Stop() {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	if !rtm.running {
		return
	}

	rtm.running = false
	close(rtm.stopCh)

	// Unsubscribe from events
	for _, sub := range rtm.subscriptions {
		rtm.eventBus.Unsubscribe(sub)
	}

	// Close stream clients
	for _, ch := range rtm.streamClients {
		close(ch)
	}
}

// AddWebhook adds a webhook configuration
func (rtm *RealTimeMonitor) AddWebhook(config WebhookConfig) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()
	rtm.webhooks = append(rtm.webhooks, config)
}

// RemoveWebhook removes a webhook configuration
func (rtm *RealTimeMonitor) RemoveWebhook(url string) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	filtered := make([]WebhookConfig, 0)
	for _, webhook := range rtm.webhooks {
		if webhook.URL != url {
			filtered = append(filtered, webhook)
		}
	}
	rtm.webhooks = filtered
}

// SubscribeToStream subscribes to the event stream
func (rtm *RealTimeMonitor) SubscribeToStream(clientID string) <-chan Event {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	ch := make(chan Event, 100)
	rtm.streamClients[clientID] = ch
	return ch
}

// UnsubscribeFromStream unsubscribes from the event stream
func (rtm *RealTimeMonitor) UnsubscribeFromStream(clientID string) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	if ch, exists := rtm.streamClients[clientID]; exists {
		close(ch)
		delete(rtm.streamClients, clientID)
	}
}

// GetMetrics returns current monitoring metrics
func (rtm *RealTimeMonitor) GetMetrics() *MonitoringMetrics {
	rtm.mu.RLock()
	defer rtm.mu.RUnlock()

	// Create a copy to avoid race conditions
	metricsCopy := &MonitoringMetrics{
		ResourcesDiscovered:  rtm.metrics.ResourcesDiscovered,
		ResourcesPerSecond:   rtm.metrics.ResourcesPerSecond,
		ErrorCount:           rtm.metrics.ErrorCount,
		WarningCount:         rtm.metrics.WarningCount,
		LastUpdateTime:       rtm.metrics.LastUpdateTime,
		AverageDiscoveryTime: rtm.metrics.AverageDiscoveryTime,
		ErrorRate:            rtm.metrics.ErrorRate,
		SuccessRate:          rtm.metrics.SuccessRate,
		ActiveProviders:      make(map[string]bool),
		ActiveRegions:        make(map[string]bool),
		DiscoveryRate:        make(map[string]float64),
	}

	for k, v := range rtm.metrics.ActiveProviders {
		metricsCopy.ActiveProviders[k] = v
	}

	for k, v := range rtm.metrics.ActiveRegions {
		metricsCopy.ActiveRegions[k] = v
	}

	for k, v := range rtm.metrics.DiscoveryRate {
		metricsCopy.DiscoveryRate[k] = v
	}

	return metricsCopy
}

// GetAlerts returns current alerts
func (rtm *RealTimeMonitor) GetAlerts(unacknowledgedOnly bool) []Alert {
	rtm.mu.RLock()
	defer rtm.mu.RUnlock()

	if !unacknowledgedOnly {
		alertsCopy := make([]Alert, len(rtm.alerts))
		copy(alertsCopy, rtm.alerts)
		return alertsCopy
	}

	unacknowledged := make([]Alert, 0)
	for _, alert := range rtm.alerts {
		if !alert.Acknowledged {
			unacknowledged = append(unacknowledged, alert)
		}
	}
	return unacknowledged
}

// AcknowledgeAlert acknowledges an alert
func (rtm *RealTimeMonitor) AcknowledgeAlert(alertID string) error {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	for i, alert := range rtm.alerts {
		if alert.ID == alertID {
			rtm.alerts[i].Acknowledged = true
			return nil
		}
	}
	return fmt.Errorf("alert not found: %s", alertID)
}

// CreateAlert creates a new alert
func (rtm *RealTimeMonitor) CreateAlert(alertType AlertType, severity AlertSeverity, message string, details map[string]interface{}) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	alert := Alert{
		ID:           fmt.Sprintf("alert-%d", time.Now().UnixNano()),
		Type:         alertType,
		Severity:     severity,
		Message:      message,
		Details:      details,
		Timestamp:    time.Now(),
		Acknowledged: false,
	}

	rtm.alerts = append(rtm.alerts, alert)

	// Broadcast to stream clients
	event := Event{
		Type:      "alert",
		Timestamp: alert.Timestamp,
		Data: map[string]interface{}{
			"alert": alert,
		},
	}
	rtm.broadcastEvent(event)
}

// GetDashboardData returns data for dashboard display
func (rtm *RealTimeMonitor) GetDashboardData() map[string]interface{} {
	rtm.mu.RLock()
	defer rtm.mu.RUnlock()

	return map[string]interface{}{
		"metrics": rtm.metrics,
		"alerts": map[string]interface{}{
			"total":          len(rtm.alerts),
			"unacknowledged": rtm.countUnacknowledgedAlerts(),
			"critical":       rtm.countAlertsBySeverity(AlertSeverityCritical),
			"high":           rtm.countAlertsBySeverity(AlertSeverityHigh),
		},
		"active_providers": len(rtm.metrics.ActiveProviders),
		"active_regions":   len(rtm.metrics.ActiveRegions),
		"stream_clients":   len(rtm.streamClients),
		"webhooks":         len(rtm.webhooks),
		"status":           rtm.getSystemStatus(),
	}
}

// Helper functions

func (rtm *RealTimeMonitor) subscribeToEvents() {
	// Subscribe to discovery events
	discoverySub := rtm.eventBus.Subscribe([]events.EventType{
		events.EventDiscoveryStarted,
		events.EventDiscoveryProgress,
		events.EventDiscoveryCompleted,
		events.EventDiscoveryFailed,
		events.EventResourceFound,
	}, rtm.handleDiscoveryEvent)
	rtm.subscriptions["discovery"] = discoverySub

	// Subscribe to system events
	systemSub := rtm.eventBus.Subscribe([]events.EventType{
		events.EventSystemError,
		events.EventSystemWarning,
	}, rtm.handleSystemEvent)
	rtm.subscriptions["system"] = systemSub
}

func (rtm *RealTimeMonitor) handleDiscoveryEvent(event events.Event) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	switch event.Type {
	case events.EventResourceFound:
		rtm.metrics.ResourcesDiscovered++
		if provider, ok := event.Data["provider"].(string); ok {
			rtm.metrics.ActiveProviders[provider] = true
		}
		if region, ok := event.Data["region"].(string); ok {
			rtm.metrics.ActiveRegions[region] = true
		}
	case events.EventDiscoveryFailed:
		rtm.metrics.ErrorCount++
		rtm.CreateAlert(AlertTypeError, AlertSeverityHigh, "Discovery failed", event.Data)
	}

	// Broadcast to stream clients
	rtm.broadcastEvent(Event{
		Type:      string(event.Type),
		Timestamp: event.Timestamp,
		Provider:  event.Data["provider"].(string),
		Region:    event.Data["region"].(string),
		Data:      event.Data,
	})
}

func (rtm *RealTimeMonitor) handleSystemEvent(event events.Event) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	switch event.Type {
	case events.EventSystemError:
		rtm.metrics.ErrorCount++
		rtm.CreateAlert(AlertTypeError, AlertSeverityCritical, "System error", event.Data)
	case events.EventSystemWarning:
		rtm.metrics.WarningCount++
		rtm.CreateAlert(AlertTypeWarning, AlertSeverityMedium, "System warning", event.Data)
	}
}

func (rtm *RealTimeMonitor) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(rtm.refreshInterval)
	defer ticker.Stop()

	lastResourceCount := int64(0)
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rtm.stopCh:
			return
		case <-ticker.C:
			rtm.mu.Lock()

			// Calculate resources per second
			currentTime := time.Now()
			timeDiff := currentTime.Sub(lastTime).Seconds()
			resourceDiff := rtm.metrics.ResourcesDiscovered - lastResourceCount

			if timeDiff > 0 {
				rtm.metrics.ResourcesPerSecond = float64(resourceDiff) / timeDiff
			}

			// Calculate success rate
			total := rtm.metrics.ResourcesDiscovered + rtm.metrics.ErrorCount
			if total > 0 {
				rtm.metrics.SuccessRate = float64(rtm.metrics.ResourcesDiscovered) / float64(total)
				rtm.metrics.ErrorRate = float64(rtm.metrics.ErrorCount) / float64(total)
			}

			rtm.metrics.LastUpdateTime = currentTime
			lastResourceCount = rtm.metrics.ResourcesDiscovered
			lastTime = currentTime

			rtm.mu.Unlock()
		}
	}
}

func (rtm *RealTimeMonitor) processAlerts(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rtm.stopCh:
			return
		case <-ticker.C:
			rtm.checkAlertConditions()
		}
	}
}

func (rtm *RealTimeMonitor) checkAlertConditions() {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	// Check for slow discovery
	if rtm.metrics.ResourcesPerSecond < 1.0 && rtm.metrics.ResourcesPerSecond > 0 {
		rtm.CreateAlert(AlertTypeSlowDiscovery, AlertSeverityLow,
			fmt.Sprintf("Discovery rate is slow: %.2f resources/sec", rtm.metrics.ResourcesPerSecond),
			map[string]interface{}{"rate": rtm.metrics.ResourcesPerSecond})
	}

	// Check error rate
	if rtm.metrics.ErrorRate > 0.1 { // More than 10% errors
		rtm.CreateAlert(AlertTypeError, AlertSeverityHigh,
			fmt.Sprintf("High error rate detected: %.1f%%", rtm.metrics.ErrorRate*100),
			map[string]interface{}{"error_rate": rtm.metrics.ErrorRate})
	}
}

func (rtm *RealTimeMonitor) processWebhooks(ctx context.Context) {
	// Webhook processing would be implemented here
	// This is a placeholder for webhook delivery logic
}

func (rtm *RealTimeMonitor) broadcastEvent(event Event) {
	for _, ch := range rtm.streamClients {
		select {
		case ch <- event:
		default:
			// Channel is full, skip this client
		}
	}
}

func (rtm *RealTimeMonitor) countUnacknowledgedAlerts() int {
	count := 0
	for _, alert := range rtm.alerts {
		if !alert.Acknowledged {
			count++
		}
	}
	return count
}

func (rtm *RealTimeMonitor) countAlertsBySeverity(severity AlertSeverity) int {
	count := 0
	for _, alert := range rtm.alerts {
		if alert.Severity == severity {
			count++
		}
	}
	return count
}

func (rtm *RealTimeMonitor) getSystemStatus() string {
	if rtm.metrics.ErrorRate > 0.5 {
		return "critical"
	} else if rtm.metrics.ErrorRate > 0.1 {
		return "warning"
	} else if rtm.running {
		return "healthy"
	}
	return "stopped"
}

// ProcessResource processes a discovered resource
func (rtm *RealTimeMonitor) ProcessResource(resource models.Resource) {
	// Emit event for resource discovery
	rtm.eventBus.Publish(events.Event{
		ID:        fmt.Sprintf("resource-%d", time.Now().UnixNano()),
		Type:      events.EventResourceFound,
		Timestamp: time.Now(),
		Source:    "realtime_monitor",
		Data: map[string]interface{}{
			"provider": resource.Provider,
			"region":   resource.Region,
			"type":     resource.Type,
			"id":       resource.ID,
			"name":     resource.Name,
		},
	})
}
