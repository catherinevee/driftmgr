package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// ComplianceManager manages security compliance and policy enforcement
type ComplianceManager struct {
	policies map[string]*CompliancePolicy
	checks   map[string]*ComplianceCheck
	results  map[string]*ComplianceResult
	mu       sync.RWMutex
	eventBus EventBus
	config   *ComplianceConfig
}

// CompliancePolicy represents a compliance policy
type CompliancePolicy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Standard    string                 `json:"standard"` // SOC2, HIPAA, PCI-DSS, etc.
	Version     string                 `json:"version"`
	Category    string                 `json:"category"`
	Severity    string                 `json:"severity"`
	Rules       []ComplianceRule       `json:"rules"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceRule represents a rule within a compliance policy
type ComplianceRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Conditions  []RuleCondition        `json:"conditions"`
	Actions     []RuleAction           `json:"actions"`
	Severity    string                 `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RuleCondition represents a condition for a compliance rule
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type"`
}

// RuleAction represents an action to take when a rule is violated
type RuleAction struct {
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
}

// ComplianceCheck represents a compliance check
type ComplianceCheck struct {
	ID          string                 `json:"id"`
	PolicyID    string                 `json:"policy_id"`
	RuleID      string                 `json:"rule_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Resource    string                 `json:"resource"`
	Status      string                 `json:"status"`
	LastRun     time.Time              `json:"last_run"`
	NextRun     time.Time              `json:"next_run"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceResult represents the result of a compliance check
type ComplianceResult struct {
	ID         string                 `json:"id"`
	CheckID    string                 `json:"check_id"`
	PolicyID   string                 `json:"policy_id"`
	RuleID     string                 `json:"rule_id"`
	ResourceID string                 `json:"resource_id"`
	Status     string                 `json:"status"` // PASS, FAIL, WARN, ERROR
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Violations []ComplianceViolation  `json:"violations,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`
	Field       string                 `json:"field"`
	Expected    interface{}            `json:"expected"`
	Actual      interface{}            `json:"actual"`
	Remediation string                 `json:"remediation"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceConfig represents configuration for the compliance manager
type ComplianceConfig struct {
	DefaultStandards    []string      `json:"default_standards"`
	CheckInterval       time.Duration `json:"check_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	AutoRemediation     bool          `json:"auto_remediation"`
	NotificationEnabled bool          `json:"notification_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
}

// EventBus interface for compliance events
type EventBus interface {
	PublishComplianceEvent(event ComplianceEvent) error
}

// ComplianceEvent represents a compliance-related event
type ComplianceEvent struct {
	Type       string                 `json:"type"`
	PolicyID   string                 `json:"policy_id,omitempty"`
	CheckID    string                 `json:"check_id,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Message    string                 `json:"message"`
	Severity   string                 `json:"severity"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewComplianceManager creates a new compliance manager
func NewComplianceManager(eventBus EventBus) *ComplianceManager {
	config := &ComplianceConfig{
		DefaultStandards:    []string{"SOC2", "HIPAA", "PCI-DSS"},
		CheckInterval:       24 * time.Hour,
		RetentionPeriod:     90 * 24 * time.Hour,
		AutoRemediation:     false,
		NotificationEnabled: true,
		AuditLogging:        true,
	}

	return &ComplianceManager{
		policies: make(map[string]*CompliancePolicy),
		checks:   make(map[string]*ComplianceCheck),
		results:  make(map[string]*ComplianceResult),
		eventBus: eventBus,
		config:   config,
	}
}

// CreatePolicy creates a new compliance policy
func (cm *ComplianceManager) CreatePolicy(ctx context.Context, policy *CompliancePolicy) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate policy
	if err := cm.validatePolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	// Set defaults
	if policy.ID == "" {
		policy.ID = fmt.Sprintf("policy_%d", time.Now().Unix())
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	// Store policy
	cm.policies[policy.ID] = policy

	// Create compliance checks for each rule
	for _, rule := range policy.Rules {
		if rule.Enabled {
			check := &ComplianceCheck{
				ID:          fmt.Sprintf("check_%s_%s", policy.ID, rule.ID),
				PolicyID:    policy.ID,
				RuleID:      rule.ID,
				Name:        rule.Name,
				Description: rule.Description,
				Type:        rule.Type,
				Status:      "pending",
				LastRun:     time.Time{},
				NextRun:     time.Now().Add(cm.config.CheckInterval),
				Enabled:     true,
				Metadata:    make(map[string]interface{}),
			}
			cm.checks[check.ID] = check
		}
	}

	// Publish event
	if cm.eventBus != nil {
		event := ComplianceEvent{
			Type:      "policy_created",
			PolicyID:  policy.ID,
			Message:   fmt.Sprintf("Compliance policy '%s' created", policy.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"policy_name": policy.Name,
				"standard":    policy.Standard,
				"rule_count":  len(policy.Rules),
			},
		}
		cm.eventBus.PublishComplianceEvent(event)
	}

	return nil
}

// RunComplianceCheck runs a compliance check for a specific resource
func (cm *ComplianceManager) RunComplianceCheck(ctx context.Context, checkID string, resource *models.Resource) (*ComplianceResult, error) {
	cm.mu.RLock()
	check, exists := cm.checks[checkID]
	if !exists {
		cm.mu.RUnlock()
		return nil, fmt.Errorf("compliance check %s not found", checkID)
	}
	cm.mu.RUnlock()

	// Get policy and rule
	policy := cm.policies[check.PolicyID]
	if policy == nil {
		return nil, fmt.Errorf("policy %s not found", check.PolicyID)
	}

	var rule *ComplianceRule
	for _, r := range policy.Rules {
		if r.ID == check.RuleID {
			rule = &r
			break
		}
	}
	if rule == nil {
		return nil, fmt.Errorf("rule %s not found", check.RuleID)
	}

	// Run the check
	result := &ComplianceResult{
		ID:         fmt.Sprintf("result_%d", time.Now().Unix()),
		CheckID:    checkID,
		PolicyID:   check.PolicyID,
		RuleID:     check.RuleID,
		ResourceID: resource.ID,
		Status:     "PASS",
		Severity:   rule.Severity,
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
		Violations: []ComplianceViolation{},
		Metadata:   make(map[string]interface{}),
	}

	// Evaluate rule conditions
	violations := cm.evaluateRule(rule, resource)
	if len(violations) > 0 {
		result.Status = "FAIL"
		result.Violations = violations
		result.Message = fmt.Sprintf("Compliance check failed with %d violations", len(violations))
	} else {
		result.Message = "Compliance check passed"
	}

	// Store result
	cm.mu.Lock()
	cm.results[result.ID] = result
	cm.mu.Unlock()

	// Update check status
	cm.mu.Lock()
	check.Status = result.Status
	check.LastRun = time.Now()
	check.NextRun = time.Now().Add(cm.config.CheckInterval)
	cm.mu.Unlock()

	// Publish event
	if cm.eventBus != nil {
		event := ComplianceEvent{
			Type:       "compliance_check_completed",
			CheckID:    checkID,
			ResourceID: resource.ID,
			Message:    result.Message,
			Severity:   result.Status,
			Timestamp:  time.Now(),
			Metadata: map[string]interface{}{
				"policy_id":    policy.ID,
				"rule_id":      rule.ID,
				"violations":   len(violations),
				"check_status": result.Status,
			},
		}
		cm.eventBus.PublishComplianceEvent(event)
	}

	return result, nil
}

// RunAllComplianceChecks runs all enabled compliance checks
func (cm *ComplianceManager) RunAllComplianceChecks(ctx context.Context, resources []*models.Resource) ([]*ComplianceResult, error) {
	var results []*ComplianceResult

	cm.mu.RLock()
	checks := make([]*ComplianceCheck, 0, len(cm.checks))
	for _, check := range cm.checks {
		if check.Enabled {
			checks = append(checks, check)
		}
	}
	cm.mu.RUnlock()

	for _, check := range checks {
		for _, resource := range resources {
			result, err := cm.RunComplianceCheck(ctx, check.ID, resource)
			if err != nil {
				// Log error but continue with other checks
				fmt.Printf("Warning: failed to run compliance check %s: %v\n", check.ID, err)
				continue
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// GetComplianceReport generates a compliance report
func (cm *ComplianceManager) GetComplianceReport(ctx context.Context, standard string) (*ComplianceReport, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	report := &ComplianceReport{
		Standard:    standard,
		GeneratedAt: time.Now(),
		Policies:    []*CompliancePolicy{},
		Results:     []*ComplianceResult{},
		Summary:     make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	// Filter policies by standard
	for _, policy := range cm.policies {
		if policy.Standard == standard && policy.Enabled {
			report.Policies = append(report.Policies, policy)
		}
	}

	// Filter results by standard
	for _, result := range cm.results {
		policy := cm.policies[result.PolicyID]
		if policy != nil && policy.Standard == standard {
			report.Results = append(report.Results, result)
		}
	}

	// Generate summary
	cm.generateReportSummary(report)

	return report, nil
}

// Helper methods

// validatePolicy validates a compliance policy
func (cm *ComplianceManager) validatePolicy(policy *CompliancePolicy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if policy.Standard == "" {
		return fmt.Errorf("policy standard is required")
	}
	if len(policy.Rules) == 0 {
		return fmt.Errorf("policy must have at least one rule")
	}
	return nil
}

// evaluateRule evaluates a compliance rule against a resource
func (cm *ComplianceManager) evaluateRule(rule *ComplianceRule, resource *models.Resource) []ComplianceViolation {
	var violations []ComplianceViolation

	// Evaluate each condition
	for _, condition := range rule.Conditions {
		if !cm.evaluateCondition(condition, resource) {
			violation := ComplianceViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().Unix()),
				Type:        rule.Type,
				Severity:    rule.Severity,
				Description: fmt.Sprintf("Rule '%s' violated", rule.Name),
				Resource:    resource.ID,
				Field:       condition.Field,
				Expected:    condition.Value,
				Actual:      cm.getResourceValue(resource, condition.Field),
				Remediation: cm.generateRemediation(rule, condition),
				Metadata:    make(map[string]interface{}),
			}
			violations = append(violations, violation)
		}
	}

	return violations
}

// evaluateCondition evaluates a single condition
func (cm *ComplianceManager) evaluateCondition(condition RuleCondition, resource *models.Resource) bool {
	actualValue := cm.getResourceValue(resource, condition.Field)

	switch condition.Operator {
	case "equals":
		return actualValue == condition.Value
	case "not_equals":
		return actualValue != condition.Value
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
	case "greater_than":
		return cm.compareValues(actualValue, condition.Value) > 0
	case "less_than":
		return cm.compareValues(actualValue, condition.Value) < 0
	default:
		return false
	}
}

// getResourceValue gets a value from a resource
func (cm *ComplianceManager) getResourceValue(resource *models.Resource, field string) interface{} {
	switch field {
	case "type":
		return resource.Type
	case "provider":
		return resource.Provider
	case "region":
		return resource.Region
	case "state":
		return resource.State
	default:
		// Check resource attributes
		if val, ok := resource.Attributes[field]; ok {
			return val
		}
		return nil
	}
}

// compareValues compares two values
func (cm *ComplianceManager) compareValues(a, b interface{}) int {
	// Simplified comparison - in reality, you'd handle different types
	if a == b {
		return 0
	}
	// This is a placeholder - real implementation would be more sophisticated
	return 1
}

// generateRemediation generates remediation advice
func (cm *ComplianceManager) generateRemediation(rule *ComplianceRule, condition RuleCondition) string {
	switch rule.Type {
	case "encryption":
		return "Enable encryption for this resource"
	case "access_control":
		return "Review and update access controls"
	case "logging":
		return "Enable audit logging"
	case "backup":
		return "Configure automated backups"
	case "monitoring":
		return "Enable monitoring and alerting"
	default:
		return "Review resource configuration"
	}
}

// generateReportSummary generates a summary for the compliance report
func (cm *ComplianceManager) generateReportSummary(report *ComplianceReport) {
	totalChecks := len(report.Results)
	passedChecks := 0
	failedChecks := 0
	warningChecks := 0

	for _, result := range report.Results {
		switch result.Status {
		case "PASS":
			passedChecks++
		case "FAIL":
			failedChecks++
		case "WARN":
			warningChecks++
		}
	}

	report.Summary["total_checks"] = totalChecks
	report.Summary["passed_checks"] = passedChecks
	report.Summary["failed_checks"] = failedChecks
	report.Summary["warning_checks"] = warningChecks
	report.Summary["compliance_score"] = float64(passedChecks) / float64(totalChecks) * 100
}

// SetConfig updates the compliance manager configuration
func (cm *ComplianceManager) SetConfig(config *ComplianceConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config = config
}

// GetConfig returns the current compliance manager configuration
func (cm *ComplianceManager) GetConfig() *ComplianceConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
