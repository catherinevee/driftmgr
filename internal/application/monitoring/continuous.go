package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
	disc "github.com/catherinevee/driftmgr/internal/discovery"
)

// MonitorConfig defines configuration for continuous monitoring
type MonitorConfig struct {
	Provider          string
	Region            string
	Interval          time.Duration
	WebhookURL        string
	AlertThresholds   AlertThresholds
	NotificationRules NotificationRules
	EnableWebhooks    bool
	EnableLogging     bool
	LogPath           string
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	NewUnmanagedResources int
	ShadowITResources     int
	ComplianceViolations  int
	CostThreshold         float64
	DriftPercentage       float64
}

// NotificationRules defines notification settings
type NotificationRules struct {
	AlertOnNewResources     bool
	AlertOnShadowIT         bool
	AlertOnHighCost         bool
	AlertOnComplianceIssues bool
	SummaryInterval         time.Duration
	Recipients              []string
}

// ResourceSnapshot represents a point-in-time resource state
type ResourceSnapshot struct {
	Timestamp         time.Time
	TotalResources    int
	ManagedResources  int
	UnmanagedResources int
	Categories        map[disc.ResourceCategory]int
	NewResources      []models.Resource
	RemovedResources  []models.Resource
	ModifiedResources []models.Resource
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string
	Type        AlertType
	Severity    AlertSeverity
	Resource    models.Resource
	Message     string
	Details     map[string]interface{}
	Timestamp   time.Time
	Acknowledged bool
}

type AlertType string

const (
	AlertTypeNewResource      AlertType = "NEW_RESOURCE"
	AlertTypeShadowIT         AlertType = "SHADOW_IT"
	AlertTypeCompliance       AlertType = "COMPLIANCE"
	AlertTypeCost             AlertType = "HIGH_COST"
	AlertTypeDrift            AlertType = "DRIFT"
	AlertTypeOrphaned         AlertType = "ORPHANED"
	AlertTypeSecurityRisk     AlertType = "SECURITY_RISK"
)

type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "CRITICAL"
	SeverityHigh     AlertSeverity = "HIGH"
	SeverityMedium   AlertSeverity = "MEDIUM"
	SeverityLow      AlertSeverity = "LOW"
	SeverityInfo     AlertSeverity = "INFO"
)

// ContinuousMonitor provides real-time resource monitoring
type ContinuousMonitor struct {
	config           MonitorConfig
	discoveryService *discovery.Service
	categorizer      *disc.ResourceCategorizer
	lastSnapshot     *ResourceSnapshot
	alerts           []Alert
	alertHandlers    []AlertHandler
	mu               sync.RWMutex
	running          bool
	stopChan         chan struct{}
	resourceCache    map[string]models.Resource
	stateTracker     *StateTracker
}

// AlertHandler interface for handling alerts
type AlertHandler interface {
	HandleAlert(alert Alert) error
}

// StateTracker tracks resource state changes over time
type StateTracker struct {
	snapshots       []ResourceSnapshot
	resourceHistory map[string][]ResourceEvent
	trends          TrendAnalysis
	mu              sync.RWMutex
}

// ResourceEvent represents a change event for a resource
type ResourceEvent struct {
	EventType   string
	Resource    models.Resource
	OldState    *models.Resource
	NewState    *models.Resource
	Timestamp   time.Time
	ChangedBy   string
	ChangeSource string
}

// TrendAnalysis contains trend information
type TrendAnalysis struct {
	UnmanagedGrowthRate     float64
	ShadowITGrowthRate      float64
	CostTrend               string // "increasing", "decreasing", "stable"
	ComplianceTrend         string
	MostCreatedResourceType string
	PeakCreationTime        time.Time
}

// NewContinuousMonitor creates a new continuous monitor
func NewContinuousMonitor(config MonitorConfig) (*ContinuousMonitor, error) {
	discoveryService, err := discovery.InitializeServiceSilent(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discovery service: %w", err)
	}
	
	return &ContinuousMonitor{
		config:           config,
		discoveryService: discoveryService,
		categorizer:      disc.NewResourceCategorizer(),
		resourceCache:    make(map[string]models.Resource),
		stateTracker:     NewStateTracker(),
		stopChan:         make(chan struct{}),
		alertHandlers:    []AlertHandler{},
	}, nil
}

// NewStateTracker creates a new state tracker
func NewStateTracker() *StateTracker {
	return &StateTracker{
		snapshots:       []ResourceSnapshot{},
		resourceHistory: make(map[string][]ResourceEvent),
	}
}

// Start begins continuous monitoring
func (cm *ContinuousMonitor) Start(ctx context.Context) error {
	cm.mu.Lock()
	if cm.running {
		cm.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	cm.running = true
	cm.mu.Unlock()
	
	// Initial discovery
	if err := cm.runDiscoveryIteration(ctx); err != nil {
		return fmt.Errorf("initial discovery failed: %w", err)
	}
	
	// Start monitoring loop
	go cm.monitorLoop(ctx)
	
	// Start alert processor
	go cm.processAlerts(ctx)
	
	// Start trend analyzer
	go cm.analyzeTrends(ctx)
	
	return nil
}

// Stop stops continuous monitoring
func (cm *ContinuousMonitor) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.running {
		cm.running = false
		close(cm.stopChan)
	}
}

// monitorLoop runs the continuous monitoring loop
func (cm *ContinuousMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.config.Interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := cm.runDiscoveryIteration(ctx); err != nil {
				cm.createAlert(Alert{
					Type:     AlertTypeDrift,
					Severity: SeverityMedium,
					Message:  fmt.Sprintf("Discovery iteration failed: %v", err),
				})
			}
		case <-cm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// runDiscoveryIteration runs a single discovery iteration
func (cm *ContinuousMonitor) runDiscoveryIteration(ctx context.Context) error {
	// Discover resources
	opts := discovery.DiscoveryOptions{}
	if cm.config.Region != "" {
		opts.Regions = []string{cm.config.Region}
	}
	result, err := cm.discoveryService.DiscoverProvider(ctx, cm.config.Provider, opts)
	if err != nil {
		return err
	}
	
	// Create snapshot
	snapshot := cm.createSnapshot(result.Resources)
	
	// Compare with last snapshot
	if cm.lastSnapshot != nil {
		cm.compareSnapshots(cm.lastSnapshot, snapshot)
	}
	
	// Update state tracker
	cm.stateTracker.AddSnapshot(*snapshot)
	
	// Update last snapshot
	cm.mu.Lock()
	cm.lastSnapshot = snapshot
	cm.mu.Unlock()
	
	// Check thresholds
	cm.checkThresholds(snapshot)
	
	return nil
}

// createSnapshot creates a resource snapshot
func (cm *ContinuousMonitor) createSnapshot(resources []models.Resource) *ResourceSnapshot {
	snapshot := &ResourceSnapshot{
		Timestamp:        time.Now(),
		TotalResources:   len(resources),
		Categories:       make(map[disc.ResourceCategory]int),
		NewResources:     []models.Resource{},
		RemovedResources: []models.Resource{},
	}
	
	managedCount := 0
	unmanagedCount := 0
	
	// Process each resource
	for _, resource := range resources {
		// Check if in state (simplified - would check actual state files)
		inState := false // Would check against state files
		
		category := cm.categorizer.CategorizeResource(resource, inState)
		snapshot.Categories[category]++
		
		if inState {
			managedCount++
		} else {
			unmanagedCount++
		}
		
		// Check if new resource
		if _, exists := cm.resourceCache[resource.ID]; !exists {
			snapshot.NewResources = append(snapshot.NewResources, resource)
		}
		
		// Update cache
		cm.resourceCache[resource.ID] = resource
	}
	
	snapshot.ManagedResources = managedCount
	snapshot.UnmanagedResources = unmanagedCount
	
	return snapshot
}

// compareSnapshots compares two snapshots and generates alerts
func (cm *ContinuousMonitor) compareSnapshots(old, new *ResourceSnapshot) {
	// Check for new unmanaged resources
	newUnmanaged := new.UnmanagedResources - old.UnmanagedResources
	if newUnmanaged > 0 {
		for _, resource := range new.NewResources {
			category := cm.categorizer.CategorizeResource(resource, false)
			
			// Alert based on category
			switch category {
			case disc.CategoryShadowIT:
				cm.createAlert(Alert{
					Type:     AlertTypeShadowIT,
					Severity: SeverityHigh,
					Resource: resource,
					Message:  fmt.Sprintf("Shadow IT resource detected: %s", resource.Name),
					Details: map[string]interface{}{
						"category": category,
						"type":     resource.Type,
					},
				})
			case disc.CategoryManageable:
				if cm.config.NotificationRules.AlertOnNewResources {
					cm.createAlert(Alert{
						Type:     AlertTypeNewResource,
						Severity: SeverityMedium,
						Resource: resource,
						Message:  fmt.Sprintf("New unmanaged resource: %s", resource.Name),
						Details: map[string]interface{}{
							"import_score": cm.categorizer.ScoreImportCandidate(resource).Score,
						},
					})
				}
			case disc.CategoryOrphaned:
				cm.createAlert(Alert{
					Type:     AlertTypeOrphaned,
					Severity: SeverityMedium,
					Resource: resource,
					Message:  fmt.Sprintf("Orphaned resource detected: %s", resource.Name),
				})
			}
			
			// Check for security implications
			if cm.hasSecurityRisk(resource) {
				cm.createAlert(Alert{
					Type:     AlertTypeSecurityRisk,
					Severity: SeverityCritical,
					Resource: resource,
					Message:  fmt.Sprintf("Security risk: Unmanaged %s resource", resource.Type),
				})
			}
			
			// Check cost
			candidate := cm.categorizer.ScoreImportCandidate(resource)
			if candidate.Cost > cm.config.AlertThresholds.CostThreshold {
				cm.createAlert(Alert{
					Type:     AlertTypeCost,
					Severity: SeverityHigh,
					Resource: resource,
					Message:  fmt.Sprintf("High-cost unmanaged resource: $%.2f/month", candidate.Cost),
				})
			}
			
			// Check compliance
			if len(candidate.ComplianceIssues) > 0 {
				cm.createAlert(Alert{
					Type:     AlertTypeCompliance,
					Severity: SeverityMedium,
					Resource: resource,
					Message:  fmt.Sprintf("Compliance issues: %s", strings.Join(candidate.ComplianceIssues, ", ")),
				})
			}
		}
	}
	
	// Track resource events
	for _, resource := range new.NewResources {
		cm.stateTracker.AddEvent(ResourceEvent{
			EventType:    "created",
			Resource:     resource,
			Timestamp:    time.Now(),
			ChangeSource: "cloud_api",
		})
	}
	
	for _, resource := range new.RemovedResources {
		cm.stateTracker.AddEvent(ResourceEvent{
			EventType:    "deleted",
			Resource:     resource,
			Timestamp:    time.Now(),
			ChangeSource: "cloud_api",
		})
	}
}

// checkThresholds checks configured thresholds
func (cm *ContinuousMonitor) checkThresholds(snapshot *ResourceSnapshot) {
	thresholds := cm.config.AlertThresholds
	
	// Check shadow IT threshold
	shadowITCount := snapshot.Categories[disc.CategoryShadowIT]
	if shadowITCount >= thresholds.ShadowITResources {
		cm.createAlert(Alert{
			Type:     AlertTypeShadowIT,
			Severity: SeverityCritical,
			Message:  fmt.Sprintf("Shadow IT threshold exceeded: %d resources", shadowITCount),
		})
	}
	
	// Check drift percentage
	if snapshot.TotalResources > 0 {
		driftPercentage := float64(snapshot.UnmanagedResources) / float64(snapshot.TotalResources) * 100
		if driftPercentage >= thresholds.DriftPercentage {
			cm.createAlert(Alert{
				Type:     AlertTypeDrift,
				Severity: SeverityHigh,
				Message:  fmt.Sprintf("High drift detected: %.1f%% resources unmanaged", driftPercentage),
			})
		}
	}
}

// createAlert creates and queues an alert
func (cm *ContinuousMonitor) createAlert(alert Alert) {
	alert.ID = fmt.Sprintf("%s-%d", alert.Type, time.Now().UnixNano())
	alert.Timestamp = time.Now()
	
	cm.mu.Lock()
	cm.alerts = append(cm.alerts, alert)
	cm.mu.Unlock()
	
	// Send to handlers
	for _, handler := range cm.alertHandlers {
		go handler.HandleAlert(alert)
	}
}

// processAlerts processes queued alerts
func (cm *ContinuousMonitor) processAlerts(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.processAlertBatch()
		case <-cm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processAlertBatch processes a batch of alerts
func (cm *ContinuousMonitor) processAlertBatch() {
	cm.mu.Lock()
	alerts := cm.alerts
	cm.alerts = []Alert{}
	cm.mu.Unlock()
	
	if len(alerts) == 0 {
		return
	}
	
	// Group alerts by type
	alertGroups := make(map[AlertType][]Alert)
	for _, alert := range alerts {
		alertGroups[alert.Type] = append(alertGroups[alert.Type], alert)
	}
	
	// Send notifications
	if cm.config.EnableWebhooks && cm.config.WebhookURL != "" {
		cm.sendWebhookNotification(alertGroups)
	}
	
	// Log alerts
	if cm.config.EnableLogging {
		cm.logAlerts(alerts)
	}
}

// sendWebhookNotification sends alerts to webhook
func (cm *ContinuousMonitor) sendWebhookNotification(alertGroups map[AlertType][]Alert) {
	payload := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"provider":  cm.config.Provider,
		"region":    cm.config.Region,
		"alerts":    alertGroups,
		"summary": map[string]int{
			"total":     len(cm.alerts),
			"critical":  cm.countAlertsBySeverity(SeverityCritical),
			"high":      cm.countAlertsBySeverity(SeverityHigh),
			"medium":    cm.countAlertsBySeverity(SeverityMedium),
		},
	}
	
	// Send webhook (simplified - would use proper HTTP client)
	data, _ := json.Marshal(payload)
	_ = data // Would send via HTTP POST
}

// logAlerts logs alerts to file
func (cm *ContinuousMonitor) logAlerts(alerts []Alert) {
	if cm.config.LogPath == "" {
		return
	}
	
	for _, alert := range alerts {
		logEntry := map[string]interface{}{
			"timestamp": alert.Timestamp,
			"type":      alert.Type,
			"severity":  alert.Severity,
			"message":   alert.Message,
			"resource":  alert.Resource.Name,
			"details":   alert.Details,
		}
		
		data, _ := json.Marshal(logEntry)
		// Would append to log file
		_ = data
	}
}

// analyzeTrends analyzes resource trends
func (cm *ContinuousMonitor) analyzeTrends(ctx context.Context) {
	ticker := time.NewTicker(cm.config.NotificationRules.SummaryInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			trends := cm.stateTracker.AnalyzeTrends()
			cm.sendTrendReport(trends)
		case <-cm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// sendTrendReport sends trend analysis report
func (cm *ContinuousMonitor) sendTrendReport(trends TrendAnalysis) {
	report := fmt.Sprintf(
		"Resource Trend Report:\n"+
		"- Unmanaged Growth Rate: %.1f%%\n"+
		"- Shadow IT Growth Rate: %.1f%%\n"+
		"- Cost Trend: %s\n"+
		"- Compliance Trend: %s\n"+
		"- Most Created Type: %s\n",
		trends.UnmanagedGrowthRate,
		trends.ShadowITGrowthRate,
		trends.CostTrend,
		trends.ComplianceTrend,
		trends.MostCreatedResourceType,
	)
	
	// Send report (would send via email/webhook)
	_ = report
}

// Helper methods

func (cm *ContinuousMonitor) hasSecurityRisk(resource models.Resource) bool {
	securityTypes := []string{
		"SecurityGroup", "NetworkACL", "Firewall",
		"IAMRole", "IAMPolicy", "AccessKey",
	}
	
	for _, secType := range securityTypes {
		if strings.Contains(resource.Type, secType) {
			return true
		}
	}
	
	return false
}

func (cm *ContinuousMonitor) countAlertsBySeverity(severity AlertSeverity) int {
	count := 0
	for _, alert := range cm.alerts {
		if alert.Severity == severity {
			count++
		}
	}
	return count
}

// StateTracker methods

func (st *StateTracker) AddSnapshot(snapshot ResourceSnapshot) {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	st.snapshots = append(st.snapshots, snapshot)
	
	// Keep only last 100 snapshots
	if len(st.snapshots) > 100 {
		st.snapshots = st.snapshots[1:]
	}
}

func (st *StateTracker) AddEvent(event ResourceEvent) {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	st.resourceHistory[event.Resource.ID] = append(
		st.resourceHistory[event.Resource.ID],
		event,
	)
}

func (st *StateTracker) AnalyzeTrends() TrendAnalysis {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	if len(st.snapshots) < 2 {
		return TrendAnalysis{}
	}
	
	// Calculate growth rates
	firstSnapshot := st.snapshots[0]
	lastSnapshot := st.snapshots[len(st.snapshots)-1]
	
	unmanagedGrowth := float64(lastSnapshot.UnmanagedResources-firstSnapshot.UnmanagedResources) /
		float64(firstSnapshot.UnmanagedResources+1) * 100
	
	shadowITGrowth := float64(lastSnapshot.Categories[disc.CategoryShadowIT]-
		firstSnapshot.Categories[disc.CategoryShadowIT]) /
		float64(firstSnapshot.Categories[disc.CategoryShadowIT]+1) * 100
	
	// Determine trends
	costTrend := "stable"
	if unmanagedGrowth > 10 {
		costTrend = "increasing"
	} else if unmanagedGrowth < -10 {
		costTrend = "decreasing"
	}
	
	complianceTrend := "stable"
	if shadowITGrowth > 5 {
		complianceTrend = "deteriorating"
	} else if shadowITGrowth < -5 {
		complianceTrend = "improving"
	}
	
	return TrendAnalysis{
		UnmanagedGrowthRate:     unmanagedGrowth,
		ShadowITGrowthRate:      shadowITGrowth,
		CostTrend:               costTrend,
		ComplianceTrend:         complianceTrend,
		MostCreatedResourceType: st.findMostCreatedType(),
		PeakCreationTime:        st.findPeakCreationTime(),
	}
}

func (st *StateTracker) findMostCreatedType() string {
	typeCounts := make(map[string]int)
	
	for _, events := range st.resourceHistory {
		for _, event := range events {
			if event.EventType == "created" {
				typeCounts[event.Resource.Type]++
			}
		}
	}
	
	maxType := ""
	maxCount := 0
	for typ, count := range typeCounts {
		if count > maxCount {
			maxType = typ
			maxCount = count
		}
	}
	
	return maxType
}

func (st *StateTracker) findPeakCreationTime() time.Time {
	hourCounts := make(map[int]int)
	
	for _, events := range st.resourceHistory {
		for _, event := range events {
			if event.EventType == "created" {
				hour := event.Timestamp.Hour()
				hourCounts[hour]++
			}
		}
	}
	
	maxHour := 0
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxHour = hour
			maxCount = count
		}
	}
	
	return time.Now().Truncate(24*time.Hour).Add(time.Duration(maxHour) * time.Hour)
}

// GetSnapshot returns the latest snapshot
func (cm *ContinuousMonitor) GetSnapshot() *ResourceSnapshot {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.lastSnapshot
}

// GetAlerts returns current alerts
func (cm *ContinuousMonitor) GetAlerts() []Alert {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.alerts
}

// AddAlertHandler adds a custom alert handler
func (cm *ContinuousMonitor) AddAlertHandler(handler AlertHandler) {
	cm.alertHandlers = append(cm.alertHandlers, handler)
}