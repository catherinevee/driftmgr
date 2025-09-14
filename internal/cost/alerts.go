package cost

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CostAlertManager manages cost alerts and notifications
type CostAlertManager struct {
	alerts        map[string]*CostAlert
	alertRules    map[string]*AlertRule
	notifications []AlertNotification
	mu            sync.RWMutex
	eventBus      EventBus
	config        *AlertConfig
}

// CostAlert represents a cost alert
type CostAlert struct {
	ID           string                 `json:"id"`
	RuleID       string                 `json:"rule_id"`
	Type         string                 `json:"type"`
	Severity     string                 `json:"severity"`
	Message      string                 `json:"message"`
	CurrentValue float64                `json:"current_value"`
	Threshold    float64                `json:"threshold"`
	Currency     string                 `json:"currency"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AlertRule represents a rule for triggering cost alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Condition   AlertCondition         `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Currency    string                 `json:"currency"`
	Severity    string                 `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertCondition represents the condition for an alert rule
type AlertCondition struct {
	Field      string        `json:"field"`
	Operator   string        `json:"operator"`
	Value      interface{}   `json:"value"`
	TimeWindow time.Duration `json:"time_window"`
}

// AlertNotification represents a notification sent for an alert
type AlertNotification struct {
	ID        string                 `json:"id"`
	AlertID   string                 `json:"alert_id"`
	Type      string                 `json:"type"`
	Recipient string                 `json:"recipient"`
	Message   string                 `json:"message"`
	Status    string                 `json:"status"`
	SentAt    time.Time              `json:"sent_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertConfig contains configuration for the alert manager
type AlertConfig struct {
	DefaultCurrency      string        `json:"default_currency"`
	CheckInterval        time.Duration `json:"check_interval"`
	MaxAlertsPerHour     int           `json:"max_alerts_per_hour"`
	NotificationChannels []string      `json:"notification_channels"`
	Enabled              bool          `json:"enabled"`
}

// NewCostAlertManager creates a new cost alert manager
func NewCostAlertManager(eventBus EventBus) *CostAlertManager {
	config := &AlertConfig{
		DefaultCurrency:      "USD",
		CheckInterval:        5 * time.Minute,
		MaxAlertsPerHour:     10,
		NotificationChannels: []string{"email", "slack"},
		Enabled:              true,
	}

	return &CostAlertManager{
		alerts:        make(map[string]*CostAlert),
		alertRules:    make(map[string]*AlertRule),
		notifications: []AlertNotification{},
		eventBus:      eventBus,
		config:        config,
	}
}

// CreateAlertRule creates a new alert rule
func (cam *CostAlertManager) CreateAlertRule(rule *AlertRule) error {
	cam.mu.Lock()
	defer cam.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().Unix())
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	cam.alertRules[rule.ID] = rule

	// Publish event
	if cam.eventBus != nil {
		event := CostEvent{
			Type:      "alert_rule_created",
			Message:   fmt.Sprintf("Alert rule '%s' created", rule.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
				"threshold": rule.Threshold,
			},
		}
		_ = cam.eventBus.PublishCostEvent(event)
	}

	return nil
}

// UpdateAlertRule updates an existing alert rule
func (cam *CostAlertManager) UpdateAlertRule(ruleID string, updates *AlertRule) error {
	cam.mu.Lock()
	defer cam.mu.Unlock()

	rule, exists := cam.alertRules[ruleID]
	if !exists {
		return fmt.Errorf("alert rule %s not found", ruleID)
	}

	// Update fields
	if updates.Name != "" {
		rule.Name = updates.Name
	}
	if updates.Description != "" {
		rule.Description = updates.Description
	}
	if updates.Type != "" {
		rule.Type = updates.Type
	}
	if updates.Threshold != 0 {
		rule.Threshold = updates.Threshold
	}
	if updates.Currency != "" {
		rule.Currency = updates.Currency
	}
	if updates.Severity != "" {
		rule.Severity = updates.Severity
	}
	rule.Enabled = updates.Enabled
	rule.UpdatedAt = time.Now()

	// Publish event
	if cam.eventBus != nil {
		event := CostEvent{
			Type:      "alert_rule_updated",
			Message:   fmt.Sprintf("Alert rule '%s' updated", rule.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
			},
		}
		_ = cam.eventBus.PublishCostEvent(event)
	}

	return nil
}

// DeleteAlertRule deletes an alert rule
func (cam *CostAlertManager) DeleteAlertRule(ruleID string) error {
	cam.mu.Lock()
	defer cam.mu.Unlock()

	rule, exists := cam.alertRules[ruleID]
	if !exists {
		return fmt.Errorf("alert rule %s not found", ruleID)
	}

	delete(cam.alertRules, ruleID)

	// Publish event
	if cam.eventBus != nil {
		event := CostEvent{
			Type:      "alert_rule_deleted",
			Message:   fmt.Sprintf("Alert rule '%s' deleted", rule.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
			},
		}
		_ = cam.eventBus.PublishCostEvent(event)
	}

	return nil
}

// GetAlertRules returns all alert rules
func (cam *CostAlertManager) GetAlertRules() map[string]*AlertRule {
	cam.mu.RLock()
	defer cam.mu.RUnlock()

	rules := make(map[string]*AlertRule)
	for id, rule := range cam.alertRules {
		rules[id] = rule
	}
	return rules
}

// GetAlertRule returns a specific alert rule
func (cam *CostAlertManager) GetAlertRule(ruleID string) (*AlertRule, error) {
	cam.mu.RLock()
	defer cam.mu.RUnlock()

	rule, exists := cam.alertRules[ruleID]
	if !exists {
		return nil, fmt.Errorf("alert rule %s not found", ruleID)
	}

	return rule, nil
}

// CheckAlerts checks all enabled alert rules against current cost data
func (cam *CostAlertManager) CheckAlerts(ctx context.Context, costData *CostAnalysis) error {
	if !cam.config.Enabled {
		return nil
	}

	cam.mu.RLock()
	rules := make(map[string]*AlertRule)
	for id, rule := range cam.alertRules {
		if rule.Enabled {
			rules[id] = rule
		}
	}
	cam.mu.RUnlock()

	for ruleID, rule := range rules {
		if err := cam.checkRule(ctx, rule, costData); err != nil {
			// Log error but continue with other rules
			fmt.Printf("Error checking rule %s: %v\n", ruleID, err)
		}
	}

	return nil
}

// checkRule checks a single alert rule against cost data
func (cam *CostAlertManager) checkRule(ctx context.Context, rule *AlertRule, costData *CostAnalysis) error {
	var currentValue float64
	var shouldAlert bool

	// Get current value based on rule type
	switch rule.Type {
	case "total_cost":
		currentValue = costData.TotalCost
		shouldAlert = cam.evaluateCondition(currentValue, rule.Condition, rule.Threshold)
	case "cost_by_type":
		if typeValue, ok := costData.CostByType[rule.Condition.Field]; ok {
			currentValue = typeValue
			shouldAlert = cam.evaluateCondition(currentValue, rule.Condition, rule.Threshold)
		}
	case "cost_by_region":
		if regionValue, ok := costData.CostByRegion[rule.Condition.Field]; ok {
			currentValue = regionValue
			shouldAlert = cam.evaluateCondition(currentValue, rule.Condition, rule.Threshold)
		}
	case "cost_growth_rate":
		// Calculate growth rate (simplified)
		currentValue = 0.0 // Would calculate actual growth rate
		shouldAlert = cam.evaluateCondition(currentValue, rule.Condition, rule.Threshold)
	case "resource_count":
		currentValue = float64(costData.ResourceCount)
		shouldAlert = cam.evaluateCondition(currentValue, rule.Condition, rule.Threshold)
	default:
		return fmt.Errorf("unknown alert rule type: %s", rule.Type)
	}

	if shouldAlert {
		return cam.createAlert(rule, currentValue)
	}

	return nil
}

// evaluateCondition evaluates an alert condition
func (cam *CostAlertManager) evaluateCondition(currentValue float64, condition AlertCondition, threshold float64) bool {
	switch condition.Operator {
	case "greater_than":
		return currentValue > threshold
	case "greater_than_or_equal":
		return currentValue >= threshold
	case "less_than":
		return currentValue < threshold
	case "less_than_or_equal":
		return currentValue <= threshold
	case "equals":
		return currentValue == threshold
	case "not_equals":
		return currentValue != threshold
	default:
		return false
	}
}

// createAlert creates a new alert
func (cam *CostAlertManager) createAlert(rule *AlertRule, currentValue float64) error {
	cam.mu.Lock()
	defer cam.mu.Unlock()

	// Check if alert already exists for this rule
	for _, alert := range cam.alerts {
		if alert.RuleID == rule.ID && alert.Status == "active" {
			// Update existing alert
			alert.CurrentValue = currentValue
			alert.UpdatedAt = time.Now()
			return nil
		}
	}

	// Create new alert
	alert := &CostAlert{
		ID:           fmt.Sprintf("alert_%d", time.Now().Unix()),
		RuleID:       rule.ID,
		Type:         rule.Type,
		Severity:     rule.Severity,
		Message:      cam.generateAlertMessage(rule, currentValue),
		CurrentValue: currentValue,
		Threshold:    rule.Threshold,
		Currency:     rule.Currency,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"rule_name": rule.Name,
		},
	}

	cam.alerts[alert.ID] = alert

	// Send notifications
	cam.sendNotifications(alert)

	// Publish event
	if cam.eventBus != nil {
		event := CostEvent{
			Type:      "cost_alert_triggered",
			Amount:    currentValue,
			Currency:  rule.Currency,
			Message:   alert.Message,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"alert_id":  alert.ID,
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
				"severity":  rule.Severity,
				"threshold": rule.Threshold,
			},
		}
		_ = cam.eventBus.PublishCostEvent(event)
	}

	return nil
}

// generateAlertMessage generates an alert message
func (cam *CostAlertManager) generateAlertMessage(rule *AlertRule, currentValue float64) string {
	return fmt.Sprintf("Cost alert triggered: %s is %.2f %s (threshold: %.2f %s)",
		rule.Name, currentValue, rule.Currency, rule.Threshold, rule.Currency)
}

// sendNotifications sends notifications for an alert
func (cam *CostAlertManager) sendNotifications(alert *CostAlert) {
	for _, channel := range cam.config.NotificationChannels {
		notification := AlertNotification{
			ID:        fmt.Sprintf("notif_%d", time.Now().Unix()),
			AlertID:   alert.ID,
			Type:      channel,
			Recipient: "admin@example.com", // Would be configurable
			Message:   alert.Message,
			Status:    "sent",
			SentAt:    time.Now(),
			Metadata: map[string]interface{}{
				"severity": alert.Severity,
			},
		}

		cam.notifications = append(cam.notifications, notification)
	}
}

// ResolveAlert resolves an active alert
func (cam *CostAlertManager) ResolveAlert(alertID string) error {
	cam.mu.Lock()
	defer cam.mu.Unlock()

	alert, exists := cam.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	if alert.Status != "active" {
		return fmt.Errorf("alert %s is not active", alertID)
	}

	now := time.Now()
	alert.Status = "resolved"
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	// Publish event
	if cam.eventBus != nil {
		event := CostEvent{
			Type:      "cost_alert_resolved",
			Message:   fmt.Sprintf("Alert %s resolved", alertID),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"alert_id": alert.ID,
				"rule_id":  alert.RuleID,
			},
		}
		_ = cam.eventBus.PublishCostEvent(event)
	}

	return nil
}

// GetActiveAlerts returns all active alerts
func (cam *CostAlertManager) GetActiveAlerts() []*CostAlert {
	cam.mu.RLock()
	defer cam.mu.RUnlock()

	var activeAlerts []*CostAlert
	for _, alert := range cam.alerts {
		if alert.Status == "active" {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

// GetAllAlerts returns all alerts
func (cam *CostAlertManager) GetAllAlerts() []*CostAlert {
	cam.mu.RLock()
	defer cam.mu.RUnlock()

	var allAlerts []*CostAlert
	for _, alert := range cam.alerts {
		allAlerts = append(allAlerts, alert)
	}

	return allAlerts
}

// GetAlert returns a specific alert
func (cam *CostAlertManager) GetAlert(alertID string) (*CostAlert, error) {
	cam.mu.RLock()
	defer cam.mu.RUnlock()

	alert, exists := cam.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert %s not found", alertID)
	}

	return alert, nil
}

// GetNotifications returns all notifications
func (cam *CostAlertManager) GetNotifications() []AlertNotification {
	cam.mu.RLock()
	defer cam.mu.RUnlock()

	return cam.notifications
}

// SetConfig updates the alert manager configuration
func (cam *CostAlertManager) SetConfig(config *AlertConfig) {
	cam.mu.Lock()
	defer cam.mu.Unlock()
	cam.config = config
}

// GetConfig returns the current alert manager configuration
func (cam *CostAlertManager) GetConfig() *AlertConfig {
	cam.mu.RLock()
	defer cam.mu.RUnlock()
	return cam.config
}

// CreateDefaultRules creates default alert rules
func (cam *CostAlertManager) CreateDefaultRules() error {
	defaultRules := []*AlertRule{
		{
			Name:        "High Total Cost",
			Description: "Alert when total cost exceeds $10,000",
			Type:        "total_cost",
			Condition: AlertCondition{
				Field:    "total_cost",
				Operator: "greater_than",
				Value:    10000.0,
			},
			Threshold: 10000.0,
			Currency:  "USD",
			Severity:  "high",
			Enabled:   true,
		},
		{
			Name:        "Compute Cost Spike",
			Description: "Alert when compute costs exceed $5,000",
			Type:        "cost_by_type",
			Condition: AlertCondition{
				Field:    "compute",
				Operator: "greater_than",
				Value:    5000.0,
			},
			Threshold: 5000.0,
			Currency:  "USD",
			Severity:  "medium",
			Enabled:   true,
		},
		{
			Name:        "Storage Cost Alert",
			Description: "Alert when storage costs exceed $1,000",
			Type:        "cost_by_type",
			Condition: AlertCondition{
				Field:    "storage",
				Operator: "greater_than",
				Value:    1000.0,
			},
			Threshold: 1000.0,
			Currency:  "USD",
			Severity:  "medium",
			Enabled:   true,
		},
		{
			Name:        "Resource Count Alert",
			Description: "Alert when resource count exceeds 500",
			Type:        "resource_count",
			Condition: AlertCondition{
				Field:    "resource_count",
				Operator: "greater_than",
				Value:    500.0,
			},
			Threshold: 500.0,
			Currency:  "USD",
			Severity:  "low",
			Enabled:   true,
		},
	}

	for _, rule := range defaultRules {
		if err := cam.CreateAlertRule(rule); err != nil {
			return fmt.Errorf("failed to create default rule %s: %w", rule.Name, err)
		}
	}

	return nil
}
