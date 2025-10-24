package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
)

// Engine represents the alerting engine
type Engine struct {
	ruleRepo         RuleRepository
	alertRepo        AlertRepository
	notificationRepo NotificationRepository
	eventBus         EventBus
	config           EngineConfig
	activeRules      map[uuid.UUID]*RuleContext
	mu               sync.RWMutex
	stopChan         chan struct{}
	isRunning        bool
}

// RuleRepository defines the interface for alert rule persistence
type RuleRepository interface {
	CreateRule(ctx context.Context, rule *models.AlertRule) error
	GetRule(ctx context.Context, id uuid.UUID) (*models.AlertRule, error)
	UpdateRule(ctx context.Context, rule *models.AlertRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
	ListRules(ctx context.Context, filter RuleFilter) ([]*models.AlertRule, error)
	GetRuleStats(ctx context.Context, id uuid.UUID) (*RuleStats, error)
}

// AlertRepository defines the interface for alert persistence
type AlertRepository interface {
	CreateAlert(ctx context.Context, alert *models.Alert) error
	UpdateAlert(ctx context.Context, alert *models.Alert) error
	GetAlert(ctx context.Context, id uuid.UUID) (*models.Alert, error)
	ListAlerts(ctx context.Context, filter AlertFilter) ([]*models.Alert, error)
	GetAlertHistory(ctx context.Context, ruleID uuid.UUID, limit int) ([]*models.Alert, error)
	GetAlertStats(ctx context.Context, ruleID uuid.UUID) (*AlertStats, error)
}

// NotificationRepository defines the interface for notification persistence
type NotificationRepository interface {
	CreateNotificationChannel(ctx context.Context, channel *models.NotificationChannel) error
	GetNotificationChannel(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error)
	UpdateNotificationChannel(ctx context.Context, channel *models.NotificationChannel) error
	DeleteNotificationChannel(ctx context.Context, id uuid.UUID) error
	ListNotificationChannels(ctx context.Context, filter NotificationFilter) ([]*models.NotificationChannel, error)
}

// EventBus defines the interface for event communication
type EventBus interface {
	PublishEvent(ctx context.Context, event *models.AlertEvent) error
	SubscribeToEvents(ctx context.Context, eventType string, handler EventHandler) error
	UnsubscribeFromEvents(ctx context.Context, eventType string) error
}

// EventHandler defines the interface for handling alert events
type EventHandler interface {
	HandleEvent(ctx context.Context, event *models.AlertEvent) error
}

// EngineConfig holds configuration for the alerting engine
type EngineConfig struct {
	MaxRulesPerUser    int           `json:"max_rules_per_user"`
	MaxAlertsPerHour   int           `json:"max_alerts_per_hour"`
	EvaluationInterval time.Duration `json:"evaluation_interval"`
	AlertTimeout       time.Duration `json:"alert_timeout"`
	EnableEventLogging bool          `json:"enable_event_logging"`
	EnableMetrics      bool          `json:"enable_metrics"`
	EnableAuditLogging bool          `json:"enable_audit_logging"`
	EnableCorrelation  bool          `json:"enable_correlation"`
	EnableSuppression  bool          `json:"enable_suppression"`
}

// RuleContext holds the context for an alert rule
type RuleContext struct {
	Rule        *models.AlertRule
	LastTrigger time.Time
	IsActive    bool
	CancelFunc  context.CancelFunc
}

// RuleFilter defines filters for rule queries
type RuleFilter struct {
	UserID   *uuid.UUID            `json:"user_id,omitempty"`
	Status   *models.RuleStatus    `json:"status,omitempty"`
	Severity *models.AlertSeverity `json:"severity,omitempty"`
	Tags     []string              `json:"tags,omitempty"`
	Search   string                `json:"search,omitempty"`
	Limit    int                   `json:"limit,omitempty"`
	Offset   int                   `json:"offset,omitempty"`
}

// AlertFilter defines filters for alert queries
type AlertFilter struct {
	RuleID    *uuid.UUID            `json:"rule_id,omitempty"`
	UserID    *uuid.UUID            `json:"user_id,omitempty"`
	Status    *models.AlertStatus   `json:"status,omitempty"`
	Severity  *models.AlertSeverity `json:"severity,omitempty"`
	StartTime *time.Time            `json:"start_time,omitempty"`
	EndTime   *time.Time            `json:"end_time,omitempty"`
	Limit     int                   `json:"limit,omitempty"`
	Offset    int                   `json:"offset,omitempty"`
}

// NotificationFilter defines filters for notification channel queries
type NotificationFilter struct {
	UserID *uuid.UUID            `json:"user_id,omitempty"`
	Type   *models.ChannelType   `json:"type,omitempty"`
	Status *models.ChannelStatus `json:"status,omitempty"`
	Tags   []string              `json:"tags,omitempty"`
	Search string                `json:"search,omitempty"`
	Limit  int                   `json:"limit,omitempty"`
	Offset int                   `json:"offset,omitempty"`
}

// RuleStats represents statistics for an alert rule
type RuleStats struct {
	RuleID              uuid.UUID     `json:"rule_id"`
	TotalAlerts         int           `json:"total_alerts"`
	ActiveAlerts        int           `json:"active_alerts"`
	ResolvedAlerts      int           `json:"resolved_alerts"`
	SuppressedAlerts    int           `json:"suppressed_alerts"`
	LastTrigger         *time.Time    `json:"last_trigger"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	SuccessRate         float64       `json:"success_rate"`
}

// AlertStats represents alert statistics
type AlertStats struct {
	RuleID              uuid.UUID          `json:"rule_id"`
	TotalAlerts         int                `json:"total_alerts"`
	ActiveAlerts        int                `json:"active_alerts"`
	ResolvedAlerts      int                `json:"resolved_alerts"`
	SuppressedAlerts    int                `json:"suppressed_alerts"`
	LastAlert           *time.Time         `json:"last_alert"`
	AverageResponseTime time.Duration      `json:"average_response_time"`
	SuccessRate         float64            `json:"success_rate"`
	AlertsByDay         []DailyAlertCount  `json:"alerts_by_day"`
	AlertsByHour        []HourlyAlertCount `json:"alerts_by_hour"`
}

// DailyAlertCount represents alert count for a day
type DailyAlertCount struct {
	Date   time.Time `json:"date"`
	Count  int       `json:"count"`
	Status string    `json:"status"`
}

// HourlyAlertCount represents alert count for an hour
type HourlyAlertCount struct {
	Hour   int    `json:"hour"`
	Count  int    `json:"count"`
	Status string `json:"status"`
}

// NewEngine creates a new alerting engine
func NewEngine(
	ruleRepo RuleRepository,
	alertRepo AlertRepository,
	notificationRepo NotificationRepository,
	eventBus EventBus,
	config EngineConfig,
) *Engine {
	return &Engine{
		ruleRepo:         ruleRepo,
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
		eventBus:         eventBus,
		config:           config,
		activeRules:      make(map[uuid.UUID]*RuleContext),
		stopChan:         make(chan struct{}),
	}
}

// CreateRule creates a new alert rule
func (e *Engine) CreateRule(ctx context.Context, userID uuid.UUID, req *models.AlertRuleRequest) (*models.AlertRule, error) {
	// Check rule limit
	if err := e.checkRuleLimit(ctx, userID); err != nil {
		return nil, fmt.Errorf("rule limit exceeded: %w", err)
	}

	// Validate the rule
	if err := e.validateRule(req); err != nil {
		return nil, fmt.Errorf("rule validation failed: %w", err)
	}

	// Create the rule
	rule := &models.AlertRule{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Conditions:  req.Conditions,
		Severity:    req.Severity,
		Channels:    req.Channels,
		Settings:    req.Settings,
		Status:      models.RuleStatusDraft,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the rule
	if err := e.ruleRepo.CreateRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "rule_created", userID, rule.ID, map[string]interface{}{
			"rule_name":        rule.Name,
			"severity":         rule.Severity,
			"conditions_count": len(rule.Conditions),
		})
	}

	return rule, nil
}

// GetRule retrieves a rule by ID
func (e *Engine) GetRule(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.AlertRule, error) {
	rule, err := e.ruleRepo.GetRule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	// Check ownership
	if rule.UserID != userID {
		return nil, fmt.Errorf("rule not found or access denied")
	}

	return rule, nil
}

// UpdateRule updates an existing rule
func (e *Engine) UpdateRule(ctx context.Context, userID uuid.UUID, id uuid.UUID, req *models.AlertRuleRequest) (*models.AlertRule, error) {
	// Get existing rule
	rule, err := e.ruleRepo.GetRule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	// Check ownership
	if rule.UserID != userID {
		return nil, fmt.Errorf("rule not found or access denied")
	}

	// Validate the rule
	if err := e.validateRule(req); err != nil {
		return nil, fmt.Errorf("rule validation failed: %w", err)
	}

	// Update rule fields
	rule.Name = req.Name
	rule.Description = req.Description
	rule.Conditions = req.Conditions
	rule.Severity = req.Severity
	rule.Channels = req.Channels
	rule.Settings = req.Settings
	rule.Tags = req.Tags
	rule.UpdatedAt = time.Now()

	// Save the updated rule
	if err := e.ruleRepo.UpdateRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update rule: %w", err)
	}

	// Re-register rule if active
	if rule.Status == models.RuleStatusActive {
		if err := e.registerRule(ctx, rule); err != nil {
			log.Printf("Failed to re-register rule %s: %v", id, err)
		}
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "rule_updated", userID, rule.ID, map[string]interface{}{
			"rule_name": rule.Name,
			"severity":  rule.Severity,
		})
	}

	return rule, nil
}

// DeleteRule deletes a rule
func (e *Engine) DeleteRule(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	// Get rule to check ownership
	rule, err := e.ruleRepo.GetRule(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	// Check ownership
	if rule.UserID != userID {
		return fmt.Errorf("rule not found or access denied")
	}

	// Unregister rule if active
	if rule.Status == models.RuleStatusActive {
		if err := e.unregisterRule(ctx, id); err != nil {
			log.Printf("Failed to unregister rule %s: %v", id, err)
		}
	}

	// Delete the rule
	if err := e.ruleRepo.DeleteRule(ctx, id); err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "rule_deleted", userID, rule.ID, map[string]interface{}{
			"rule_name": rule.Name,
		})
	}

	return nil
}

// ListRules lists rules with optional filtering
func (e *Engine) ListRules(ctx context.Context, userID uuid.UUID, filter RuleFilter) ([]*models.AlertRule, error) {
	// Set user ID filter
	filter.UserID = &userID

	rules, err := e.ruleRepo.ListRules(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	return rules, nil
}

// ActivateRule activates a rule
func (e *Engine) ActivateRule(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	rule, err := e.ruleRepo.GetRule(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	// Check ownership
	if rule.UserID != userID {
		return fmt.Errorf("rule not found or access denied")
	}

	if rule.Status == models.RuleStatusActive {
		return fmt.Errorf("rule is already active")
	}

	// Validate rule before activation
	if err := e.validateRuleForActivation(rule); err != nil {
		return fmt.Errorf("rule validation failed: %w", err)
	}

	// Update rule status
	rule.Status = models.RuleStatusActive
	rule.UpdatedAt = time.Now()

	if err := e.ruleRepo.UpdateRule(ctx, rule); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	// Register rule
	if err := e.registerRule(ctx, rule); err != nil {
		// Rollback status change
		rule.Status = models.RuleStatusDraft
		e.ruleRepo.UpdateRule(ctx, rule)
		return fmt.Errorf("failed to register rule: %w", err)
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "rule_activated", userID, rule.ID, map[string]interface{}{
			"rule_name": rule.Name,
		})
	}

	return nil
}

// DeactivateRule deactivates a rule
func (e *Engine) DeactivateRule(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	rule, err := e.ruleRepo.GetRule(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	// Check ownership
	if rule.UserID != userID {
		return fmt.Errorf("rule not found or access denied")
	}

	if rule.Status != models.RuleStatusActive {
		return fmt.Errorf("rule is not active")
	}

	// Unregister rule
	if err := e.unregisterRule(ctx, id); err != nil {
		log.Printf("Failed to unregister rule %s: %v", id, err)
	}

	// Update rule status
	rule.Status = models.RuleStatusDraft
	rule.UpdatedAt = time.Now()

	if err := e.ruleRepo.UpdateRule(ctx, rule); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "rule_deactivated", userID, rule.ID, map[string]interface{}{
			"rule_name": rule.Name,
		})
	}

	return nil
}

// TestRule tests an alert rule
func (e *Engine) TestRule(ctx context.Context, userID uuid.UUID, id uuid.UUID, testData map[string]interface{}) (*models.Alert, error) {
	rule, err := e.ruleRepo.GetRule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	// Check ownership
	if rule.UserID != userID {
		return nil, fmt.Errorf("rule not found or access denied")
	}

	// Evaluate rule conditions
	triggered, err := e.evaluateRuleConditions(rule, testData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate rule conditions: %w", err)
	}

	if !triggered {
		return nil, fmt.Errorf("rule conditions not met")
	}

	// Create test alert
	alert := &models.Alert{
		ID:        uuid.New(),
		RuleID:    rule.ID,
		UserID:    userID,
		Status:    models.AlertStatusActive,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Test alert for rule: %s", rule.Name),
		Data:      models.JSONB(testData),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "rule_tested", userID, rule.ID, map[string]interface{}{
			"rule_name": rule.Name,
			"triggered": triggered,
		})
	}

	return alert, nil
}

// GetAlert retrieves an alert by ID
func (e *Engine) GetAlert(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.Alert, error) {
	alert, err := e.alertRepo.GetAlert(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	// Check ownership
	if alert.UserID != userID {
		return nil, fmt.Errorf("alert not found or access denied")
	}

	return alert, nil
}

// ListAlerts lists alerts with optional filtering
func (e *Engine) ListAlerts(ctx context.Context, userID uuid.UUID, filter AlertFilter) ([]*models.Alert, error) {
	// Set user ID filter
	filter.UserID = &userID

	alerts, err := e.alertRepo.ListAlerts(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	return alerts, nil
}

// ResolveAlert resolves an alert
func (e *Engine) ResolveAlert(ctx context.Context, userID uuid.UUID, id uuid.UUID, resolution string) error {
	alert, err := e.alertRepo.GetAlert(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	// Check ownership
	if alert.UserID != userID {
		return fmt.Errorf("alert not found or access denied")
	}

	// Check if alert can be resolved
	if alert.Status != models.AlertStatusActive {
		return fmt.Errorf("alert is not active")
	}

	// Update alert status
	alert.Status = models.AlertStatusResolved
	alert.Resolution = resolution
	alert.ResolvedAt = &time.Time{}
	*alert.ResolvedAt = time.Now()
	alert.UpdatedAt = time.Now()

	if err := e.alertRepo.UpdateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "alert_resolved", userID, alert.RuleID, map[string]interface{}{
			"alert_id":   alert.ID,
			"resolution": resolution,
		})
	}

	return nil
}

// SuppressAlert suppresses an alert
func (e *Engine) SuppressAlert(ctx context.Context, userID uuid.UUID, id uuid.UUID, reason string) error {
	alert, err := e.alertRepo.GetAlert(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	// Check ownership
	if alert.UserID != userID {
		return fmt.Errorf("alert not found or access denied")
	}

	// Check if alert can be suppressed
	if alert.Status != models.AlertStatusActive {
		return fmt.Errorf("alert is not active")
	}

	// Update alert status
	alert.Status = models.AlertStatusSuppressed
	alert.SuppressionReason = reason
	alert.SuppressedAt = &time.Time{}
	*alert.SuppressedAt = time.Now()
	alert.UpdatedAt = time.Now()

	if err := e.alertRepo.UpdateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}

	// Log audit event
	if e.config.EnableAuditLogging {
		e.logAuditEvent(ctx, "alert_suppressed", userID, alert.RuleID, map[string]interface{}{
			"alert_id": alert.ID,
			"reason":   reason,
		})
	}

	return nil
}

// GetAlertHistory retrieves alert history for a rule
func (e *Engine) GetAlertHistory(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID, limit int) ([]*models.Alert, error) {
	// Check rule ownership
	rule, err := e.ruleRepo.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	if rule.UserID != userID {
		return nil, fmt.Errorf("rule not found or access denied")
	}

	history, err := e.alertRepo.GetAlertHistory(ctx, ruleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert history: %w", err)
	}
	return history, nil
}

// GetRuleStats retrieves statistics for a rule
func (e *Engine) GetRuleStats(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (*RuleStats, error) {
	// Check rule ownership
	rule, err := e.ruleRepo.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	if rule.UserID != userID {
		return nil, fmt.Errorf("rule not found or access denied")
	}

	stats, err := e.ruleRepo.GetRuleStats(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule stats: %w", err)
	}
	return stats, nil
}

// GetAlertStats retrieves alert statistics for a rule
func (e *Engine) GetAlertStats(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (*AlertStats, error) {
	// Check rule ownership
	rule, err := e.ruleRepo.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	if rule.UserID != userID {
		return nil, fmt.Errorf("rule not found or access denied")
	}

	stats, err := e.alertRepo.GetAlertStats(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats: %w", err)
	}
	return stats, nil
}

// Start starts the alerting engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return fmt.Errorf("alerting engine is already running")
	}

	e.isRunning = true
	e.stopChan = make(chan struct{})

	// Start rule evaluation
	go e.evaluateRules(ctx)

	log.Println("Alerting engine started successfully")
	return nil
}

// Stop stops the alerting engine
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return fmt.Errorf("alerting engine is not running")
	}

	// Signal stop
	close(e.stopChan)

	// Cancel all active rules
	for _, ruleContext := range e.activeRules {
		ruleContext.CancelFunc()
	}

	e.isRunning = false
	log.Println("Alerting engine stopped successfully")
	return nil
}

// checkRuleLimit checks if the user has reached the rule limit
func (e *Engine) checkRuleLimit(ctx context.Context, userID uuid.UUID) error {
	if e.config.MaxRulesPerUser <= 0 {
		return nil // No limit
	}

	filter := RuleFilter{
		UserID: &userID,
		Limit:  1,
	}

	rules, err := e.ruleRepo.ListRules(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check rule limit: %w", err)
	}

	if len(rules) >= e.config.MaxRulesPerUser {
		return fmt.Errorf("rule limit exceeded: %d rules", e.config.MaxRulesPerUser)
	}

	return nil
}

// validateRule validates a rule request
func (e *Engine) validateRule(req *models.AlertRuleRequest) error {
	if req.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if len(req.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}

	if len(req.Channels) == 0 {
		return fmt.Errorf("rule must have at least one notification channel")
	}

	return nil
}

// validateRuleForActivation validates a rule before activation
func (e *Engine) validateRuleForActivation(rule *models.AlertRule) error {
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}

	if len(rule.Channels) == 0 {
		return fmt.Errorf("rule must have at least one notification channel")
	}

	return nil
}

// registerRule registers a rule for evaluation
func (e *Engine) registerRule(ctx context.Context, rule *models.AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if rule is already registered
	if _, exists := e.activeRules[rule.ID]; exists {
		return fmt.Errorf("rule already registered")
	}

	// Create rule context
	ruleCtx, cancel := context.WithCancel(ctx)
	ruleContext := &RuleContext{
		Rule:       rule,
		IsActive:   true,
		CancelFunc: cancel,
	}

	// Add to active rules
	e.activeRules[rule.ID] = ruleContext

	log.Printf("Registered rule %s for evaluation", rule.ID)
	return nil
}

// unregisterRule unregisters a rule from evaluation
func (e *Engine) unregisterRule(ctx context.Context, ruleID uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ruleContext, exists := e.activeRules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found")
	}

	// Cancel the rule context
	ruleContext.CancelFunc()

	// Remove from active rules
	delete(e.activeRules, ruleID)

	log.Printf("Unregistered rule %s from evaluation", ruleID)
	return nil
}

// evaluateRules evaluates all active rules
func (e *Engine) evaluateRules(ctx context.Context) {
	ticker := time.NewTicker(e.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.evaluateAllRules(ctx)
		}
	}
}

// evaluateAllRules evaluates all active rules
func (e *Engine) evaluateAllRules(ctx context.Context) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for ruleID, ruleContext := range e.activeRules {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		default:
			// Evaluate rule
			go e.evaluateRule(ctx, ruleContext)
		}
	}
}

// evaluateRule evaluates a single rule
func (e *Engine) evaluateRule(ctx context.Context, ruleContext *RuleContext) {
	rule := ruleContext.Rule

	// Get data for evaluation
	data, err := e.getDataForEvaluation(ctx, rule)
	if err != nil {
		log.Printf("Failed to get data for rule %s: %v", rule.ID, err)
		return
	}

	// Evaluate rule conditions
	triggered, err := e.evaluateRuleConditions(rule, data)
	if err != nil {
		log.Printf("Failed to evaluate rule %s: %v", rule.ID, err)
		return
	}

	if triggered {
		// Create alert
		alert, err := e.createAlert(ctx, rule, data)
		if err != nil {
			log.Printf("Failed to create alert for rule %s: %v", rule.ID, err)
			return
		}

		// Send notifications
		go e.sendNotifications(ctx, alert, rule.Channels)

		// Update last trigger time
		ruleContext.LastTrigger = time.Now()
	}
}

// evaluateRuleConditions evaluates the conditions of a rule
func (e *Engine) evaluateRuleConditions(rule *models.AlertRule, data map[string]interface{}) (bool, error) {
	// Simple implementation - in production, you'd use a proper rule engine
	for _, condition := range rule.Conditions {
		if !e.evaluateCondition(condition, data) {
			return false, nil
		}
	}
	return true, nil
}

// evaluateCondition evaluates a single condition
func (e *Engine) evaluateCondition(condition models.AlertCondition, data map[string]interface{}) bool {
	// Simple implementation - in production, you'd use a proper condition evaluator
	value, exists := data[condition.Field]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "equals":
		return value == condition.Value
	case "not_equals":
		return value != condition.Value
	case "greater_than":
		if num, ok := value.(float64); ok {
			if target, ok := condition.Value.(float64); ok {
				return num > target
			}
		}
		return false
	case "less_than":
		if num, ok := value.(float64); ok {
			if target, ok := condition.Value.(float64); ok {
				return num < target
			}
		}
		return false
	case "contains":
		if str, ok := value.(string); ok {
			if target, ok := condition.Value.(string); ok {
				return contains(str, target)
			}
		}
		return false
	default:
		return false
	}
}

// getDataForEvaluation gets data for rule evaluation
func (e *Engine) getDataForEvaluation(ctx context.Context, rule *models.AlertRule) (map[string]interface{}, error) {
	// Simple implementation - in production, you'd get data from various sources
	return map[string]interface{}{
		"cpu_usage":     75.5,
		"memory_usage":  60.2,
		"disk_usage":    45.8,
		"error_count":   5,
		"response_time": 250.0,
	}, nil
}

// createAlert creates an alert
func (e *Engine) createAlert(ctx context.Context, rule *models.AlertRule, data map[string]interface{}) (*models.Alert, error) {
	alert := &models.Alert{
		ID:        uuid.New(),
		RuleID:    rule.ID,
		UserID:    rule.UserID,
		Status:    models.AlertStatusActive,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Alert triggered: %s", rule.Name),
		Data:      models.JSONB(data),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save alert
	if err := e.alertRepo.CreateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	return alert, nil
}

// sendNotifications sends notifications for an alert
func (e *Engine) sendNotifications(ctx context.Context, alert *models.Alert, channels []uuid.UUID) {
	for _, channelID := range channels {
		go func(chID uuid.UUID) {
			// Get notification channel
			channel, err := e.notificationRepo.GetNotificationChannel(ctx, chID)
			if err != nil {
				log.Printf("Failed to get notification channel %s: %v", chID, err)
				return
			}

			// Send notification
			if err := e.sendNotification(ctx, alert, channel); err != nil {
				log.Printf("Failed to send notification via channel %s: %v", chID, err)
			}
		}(channelID)
	}
}

// sendNotification sends a notification via a channel
func (e *Engine) sendNotification(ctx context.Context, alert *models.Alert, channel *models.NotificationChannel) error {
	// Simple implementation - in production, you'd use actual notification services
	log.Printf("Sending notification via %s channel %s for alert %s", channel.Type, channel.ID, alert.ID)
	return nil
}

// logAuditEvent logs an audit event
func (e *Engine) logAuditEvent(ctx context.Context, action string, userID uuid.UUID, ruleID uuid.UUID, data map[string]interface{}) {
	// In a real implementation, this would log to an audit system
	log.Printf("AUDIT: %s by user %s for rule %s: %+v", action, userID, ruleID, data)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
