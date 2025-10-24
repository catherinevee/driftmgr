package automation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/automation/actions"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/models"
)

// RuleEngine manages automation rules
type RuleEngine struct {
	rules         map[string]*AutomationRule
	actionManager *actions.ActionManager
	eventBus      *events.EventBus
	mu            sync.RWMutex
	config        *RuleConfig
}

// AutomationRule represents an automation rule
type AutomationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Priority    int                    `json:"priority"`
	Conditions  []RuleCondition        `json:"conditions"`
	Actions     []RuleAction           `json:"actions"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RuleCondition represents a condition for a rule
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type"`
}

// RuleAction represents an action to take when a rule is triggered
type RuleAction struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Parameters    map[string]interface{} `json:"parameters"`
	Configuration models.JSONB           `json:"configuration"`
	Description   string                 `json:"description"`
}

// RuleConfig represents configuration for the rule engine
type RuleConfig struct {
	MaxRules            int           `json:"max_rules"`
	EvaluationInterval  time.Duration `json:"evaluation_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	AutoCleanup         bool          `json:"auto_cleanup"`
	NotificationEnabled bool          `json:"notification_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(eventBus *events.EventBus, notificationService *events.NotificationService) *RuleEngine {
	config := &RuleConfig{
		MaxRules:            1000,
		EvaluationInterval:  1 * time.Minute,
		RetentionPeriod:     30 * 24 * time.Hour,
		AutoCleanup:         true,
		NotificationEnabled: true,
		AuditLogging:        true,
	}

	actionManager := actions.NewActionManager(eventBus, notificationService, "logs")

	return &RuleEngine{
		rules:         make(map[string]*AutomationRule),
		actionManager: actionManager,
		eventBus:      eventBus,
		config:        config,
	}
}

// CreateRule creates a new automation rule
func (re *RuleEngine) CreateRule(ctx context.Context, rule *AutomationRule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	// Check rule limit
	if len(re.rules) >= re.config.MaxRules {
		return fmt.Errorf("maximum number of rules reached (%d)", re.config.MaxRules)
	}

	// Validate rule
	if err := re.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Set defaults
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().Unix())
	}
	if rule.Priority == 0 {
		rule.Priority = 100
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	// Store rule
	re.rules[rule.ID] = rule

	// Publish rule created event
	if re.eventBus != nil {
		re.eventBus.Publish(events.Event{
			ID:        fmt.Sprintf("rule_%d", time.Now().UnixNano()),
			Type:      events.EventSystemInfo,
			Timestamp: time.Now(),
			Source:    "automation_rule_engine",
			Data: map[string]interface{}{
				"action":    "rule_created",
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
				"category":  rule.Category,
				"enabled":   rule.Enabled,
			},
		})
	}

	return nil
}

// UpdateRule updates an existing automation rule
func (re *RuleEngine) UpdateRule(ctx context.Context, ruleID string, updates *AutomationRule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	// Update fields
	if updates.Name != "" {
		rule.Name = updates.Name
	}
	if updates.Description != "" {
		rule.Description = updates.Description
	}
	if updates.Category != "" {
		rule.Category = updates.Category
	}
	if updates.Priority != 0 {
		rule.Priority = updates.Priority
	}
	if len(updates.Conditions) > 0 {
		rule.Conditions = updates.Conditions
	}
	if len(updates.Actions) > 0 {
		rule.Actions = updates.Actions
	}
	rule.UpdatedAt = time.Now()

	// Publish rule updated event
	if re.eventBus != nil {
		re.eventBus.Publish(events.Event{
			ID:        fmt.Sprintf("rule_%d", time.Now().UnixNano()),
			Type:      events.EventSystemInfo,
			Timestamp: time.Now(),
			Source:    "automation_rule_engine",
			Data: map[string]interface{}{
				"action":    "rule_updated",
				"rule_id":   ruleID,
				"rule_name": rule.Name,
				"category":  rule.Category,
				"enabled":   rule.Enabled,
			},
		})
	}

	return nil
}

// DeleteRule deletes an automation rule
func (re *RuleEngine) DeleteRule(ctx context.Context, ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	_, exists := re.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	// Delete rule
	delete(re.rules, ruleID)

	// Publish rule deleted event
	if re.eventBus != nil {
		re.eventBus.Publish(events.Event{
			ID:        fmt.Sprintf("rule_%d", time.Now().UnixNano()),
			Type:      events.EventSystemInfo,
			Timestamp: time.Now(),
			Source:    "automation_rule_engine",
			Data: map[string]interface{}{
				"action":  "rule_deleted",
				"rule_id": ruleID,
			},
		})
	}

	return nil
}

// GetRule retrieves an automation rule
func (re *RuleEngine) GetRule(ctx context.Context, ruleID string) (*AutomationRule, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule %s not found", ruleID)
	}

	return rule, nil
}

// ListRules lists all automation rules
func (re *RuleEngine) ListRules(ctx context.Context) ([]*AutomationRule, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rules := make([]*AutomationRule, 0, len(re.rules))
	for _, rule := range re.rules {
		rules = append(rules, rule)
	}

	return rules, nil
}

// EvaluateRules evaluates all enabled rules against given data
func (re *RuleEngine) EvaluateRules(ctx context.Context, data map[string]interface{}) ([]*RuleEvaluation, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	var evaluations []*RuleEvaluation

	for _, rule := range re.rules {
		if !rule.Enabled {
			continue
		}

		evaluation := &RuleEvaluation{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Matched:   false,
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		}

		// Evaluate conditions
		matched := true
		for _, condition := range rule.Conditions {
			if !re.evaluateCondition(condition, data) {
				matched = false
				break
			}
		}

		evaluation.Matched = matched
		evaluations = append(evaluations, evaluation)

		// If rule matched, execute actions
		if matched {
			for _, action := range rule.Actions {
				if err := re.executeAction(ctx, action, data); err != nil {
					// Log error but continue with other actions
					fmt.Printf("Warning: failed to execute rule action: %v\n", err)
				}
			}

			// Event publishing for matched rules handled in EvaluateRule method
		}
	}

	return evaluations, nil
}

// RuleEvaluation represents the result of rule evaluation
type RuleEvaluation struct {
	RuleID    string                 `json:"rule_id"`
	RuleName  string                 `json:"rule_name"`
	Matched   bool                   `json:"matched"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Helper methods

// validateRule validates an automation rule
func (re *RuleEngine) validateRule(rule *AutomationRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.Category == "" {
		return fmt.Errorf("rule category is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}
	if len(rule.Actions) == 0 {
		return fmt.Errorf("rule must have at least one action")
	}
	return nil
}

// evaluateCondition evaluates a single condition
func (re *RuleEngine) evaluateCondition(condition RuleCondition, data map[string]interface{}) bool {
	actualValue := data[condition.Field]

	switch condition.Operator {
	case "equals":
		return actualValue == condition.Value
	case "not_equals":
		return actualValue != condition.Value
	case "greater_than":
		return re.compareValues(actualValue, condition.Value) > 0
	case "less_than":
		return re.compareValues(actualValue, condition.Value) < 0
	case "greater_than_or_equal":
		return re.compareValues(actualValue, condition.Value) >= 0
	case "less_than_or_equal":
		return re.compareValues(actualValue, condition.Value) <= 0
	case "contains":
		if str, ok := actualValue.(string); ok {
			if val, ok := condition.Value.(string); ok {
				return contains(str, val)
			}
		}
		return false
	case "not_contains":
		if str, ok := actualValue.(string); ok {
			if val, ok := condition.Value.(string); ok {
				return !contains(str, val)
			}
		}
		return false
	case "in":
		if arr, ok := condition.Value.([]interface{}); ok {
			for _, val := range arr {
				if actualValue == val {
					return true
				}
			}
		}
		return false
	case "not_in":
		if arr, ok := condition.Value.([]interface{}); ok {
			for _, val := range arr {
				if actualValue == val {
					return false
				}
			}
		}
		return true
	case "exists":
		return actualValue != nil
	case "not_exists":
		return actualValue == nil
	default:
		return false
	}
}

// compareValues compares two values
func (re *RuleEngine) compareValues(a, b interface{}) int {
	// Simplified comparison - in reality, you'd handle different types
	if a == b {
		return 0
	}
	// This is a placeholder - real implementation would be more sophisticated
	return 1
}

// executeAction executes a rule action
func (re *RuleEngine) executeAction(ctx context.Context, action RuleAction, data map[string]interface{}) error {
	switch action.Type {
	case "execute_workflow":
		// This would be handled by the automation service
		fmt.Printf("Executing workflow action: %s\n", action.Description)
		return nil

	case "send_notification":
		// Execute notification action using action manager
		notificationAction := &models.AutomationAction{
			ID:            action.ID,
			Name:          action.Name,
			Description:   action.Description,
			Type:          models.ActionTypeNotification,
			Configuration: action.Configuration,
		}
		_, err := re.actionManager.ExecuteAction(ctx, notificationAction, data)
		return err

	case "log_event":
		// Execute logging action using action manager
		loggingAction := &models.AutomationAction{
			ID:          action.ID,
			Name:        action.Name,
			Description: action.Description,
			Type:        models.ActionTypeCustom,
			Configuration: models.JSONB(map[string]interface{}{
				"subtype": "logging",
				"level":   "info",
				"message": action.Description,
				"output":  "stdout",
				"format":  "json",
			}),
		}
		_, err := re.actionManager.ExecuteAction(ctx, loggingAction, data)
		return err

	case "update_resource":
		// Execute resource update action using action manager
		resourceUpdateAction := &models.AutomationAction{
			ID:          action.ID,
			Name:        action.Name,
			Description: action.Description,
			Type:        models.ActionTypeCustom,
			Configuration: models.JSONB(map[string]interface{}{
				"subtype": "resource_update",
			}),
		}
		_, err := re.actionManager.ExecuteAction(ctx, resourceUpdateAction, data)
		return err

	case "create_alert":
		// Execute alert creation action using action manager
		alertCreationAction := &models.AutomationAction{
			ID:          action.ID,
			Name:        action.Name,
			Description: action.Description,
			Type:        models.ActionTypeCustom,
			Configuration: models.JSONB(map[string]interface{}{
				"subtype": "alert_creation",
			}),
		}
		_, err := re.actionManager.ExecuteAction(ctx, alertCreationAction, data)
		return err

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// SetConfig updates the rule engine configuration
func (re *RuleEngine) SetConfig(config *RuleConfig) {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.config = config
}

// GetConfig returns the current rule engine configuration
func (re *RuleEngine) GetConfig() *RuleConfig {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return re.config
}
