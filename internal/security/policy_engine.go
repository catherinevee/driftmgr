package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// PolicyEngine manages security policies and enforcement
type PolicyEngine struct {
	policies     map[string]*SecurityPolicy
	rules        map[string]*SecurityRule
	enforcements map[string]*PolicyEnforcement
	mu           sync.RWMutex
	eventBus     EventBus
	config       *PolicyConfig
}

// SecurityPolicy represents a security policy
type SecurityPolicy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Priority    string                 `json:"priority"`
	Rules       []string               `json:"rules"` // Rule IDs
	Scope       PolicyScope            `json:"scope"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityRule represents a security rule
type SecurityRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Category    string                 `json:"category"`
	Conditions  []RuleCondition        `json:"conditions"`
	Actions     []RuleAction           `json:"actions"`
	Severity    string                 `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyScope represents the scope of a policy
type PolicyScope struct {
	Tenants       []string          `json:"tenants,omitempty"`
	Accounts      []string          `json:"accounts,omitempty"`
	Regions       []string          `json:"regions,omitempty"`
	Providers     []string          `json:"providers,omitempty"`
	ResourceTypes []string          `json:"resource_types,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

// PolicyEnforcement represents the enforcement of a policy
type PolicyEnforcement struct {
	ID         string                 `json:"id"`
	PolicyID   string                 `json:"policy_id"`
	RuleID     string                 `json:"rule_id"`
	ResourceID string                 `json:"resource_id"`
	Status     string                 `json:"status"` // ENFORCED, VIOLATED, PENDING
	Action     string                 `json:"action"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyConfig represents configuration for the policy engine
type PolicyConfig struct {
	DefaultEnforcement  string        `json:"default_enforcement"`
	AutoRemediation     bool          `json:"auto_remediation"`
	NotificationEnabled bool          `json:"notification_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
	RetentionPeriod     time.Duration `json:"retention_period"`
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(eventBus EventBus) *PolicyEngine {
	config := &PolicyConfig{
		DefaultEnforcement:  "warn",
		AutoRemediation:     false,
		NotificationEnabled: true,
		AuditLogging:        true,
		RetentionPeriod:     90 * 24 * time.Hour,
	}

	return &PolicyEngine{
		policies:     make(map[string]*SecurityPolicy),
		rules:        make(map[string]*SecurityRule),
		enforcements: make(map[string]*PolicyEnforcement),
		eventBus:     eventBus,
		config:       config,
	}
}

// CreatePolicy creates a new security policy
func (pe *PolicyEngine) CreatePolicy(ctx context.Context, policy *SecurityPolicy) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	// Validate policy
	if err := pe.validatePolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	// Set defaults
	if policy.ID == "" {
		policy.ID = fmt.Sprintf("policy_%d", time.Now().Unix())
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	// Store policy
	pe.policies[policy.ID] = policy

	// Publish event
	if pe.eventBus != nil {
		event := ComplianceEvent{
			Type:      "security_policy_created",
			PolicyID:  policy.ID,
			Message:   fmt.Sprintf("Security policy '%s' created", policy.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"policy_name": policy.Name,
				"category":    policy.Category,
				"priority":    policy.Priority,
			},
		}
		_ = pe.eventBus.PublishComplianceEvent(event)
	}

	return nil
}

// CreateRule creates a new security rule
func (pe *PolicyEngine) CreateRule(ctx context.Context, rule *SecurityRule) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	// Validate rule
	if err := pe.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Set defaults
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().Unix())
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	// Store rule
	pe.rules[rule.ID] = rule

	// Publish event
	if pe.eventBus != nil {
		event := ComplianceEvent{
			Type:      "security_rule_created",
			Message:   fmt.Sprintf("Security rule '%s' created", rule.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"rule_name": rule.Name,
				"rule_type": rule.Type,
				"category":  rule.Category,
			},
		}
		_ = pe.eventBus.PublishComplianceEvent(event)
	}

	return nil
}

// EvaluatePolicy evaluates a policy against a resource
func (pe *PolicyEngine) EvaluatePolicy(ctx context.Context, policyID string, resource *models.Resource) (*PolicyEvaluation, error) {
	pe.mu.RLock()
	policy, exists := pe.policies[policyID]
	if !exists {
		pe.mu.RUnlock()
		return nil, fmt.Errorf("policy %s not found", policyID)
	}
	pe.mu.RUnlock()

	// Check if policy applies to this resource
	if !pe.policyApplies(policy, resource) {
		return &PolicyEvaluation{
			PolicyID:   policyID,
			ResourceID: resource.ID,
			Status:     "NOT_APPLICABLE",
			Message:    "Policy does not apply to this resource",
			Timestamp:  time.Now(),
		}, nil
	}

	// Evaluate each rule in the policy
	evaluation := &PolicyEvaluation{
		PolicyID:    policyID,
		ResourceID:  resource.ID,
		Status:      "COMPLIANT",
		Message:     "Resource complies with policy",
		Timestamp:   time.Now(),
		RuleResults: []RuleEvaluation{},
		Violations:  []PolicyViolation{},
	}

	for _, ruleID := range policy.Rules {
		pe.mu.RLock()
		rule, exists := pe.rules[ruleID]
		pe.mu.RUnlock()

		if !exists {
			continue
		}

		ruleResult := pe.evaluateRule(rule, resource)
		evaluation.RuleResults = append(evaluation.RuleResults, ruleResult)

		if ruleResult.Status == "VIOLATED" {
			evaluation.Status = "NON_COMPLIANT"
			evaluation.Message = "Resource violates policy rules"

			violation := PolicyViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().Unix()),
				RuleID:      ruleID,
				Type:        rule.Type,
				Severity:    rule.Severity,
				Description: ruleResult.Message,
				Resource:    resource.ID,
				Field:       ruleResult.Field,
				Expected:    ruleResult.Expected,
				Actual:      ruleResult.Actual,
				Remediation: pe.generateRemediation(rule),
				Timestamp:   time.Now(),
			}
			evaluation.Violations = append(evaluation.Violations, violation)
		}
	}

	// Store enforcement record
	enforcement := &PolicyEnforcement{
		ID:         fmt.Sprintf("enforcement_%d", time.Now().Unix()),
		PolicyID:   policyID,
		ResourceID: resource.ID,
		Status:     evaluation.Status,
		Action:     pe.determineAction(evaluation),
		Message:    evaluation.Message,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	pe.mu.Lock()
	pe.enforcements[enforcement.ID] = enforcement
	pe.mu.Unlock()

	// Publish event
	if pe.eventBus != nil {
		event := ComplianceEvent{
			Type:       "policy_evaluated",
			PolicyID:   policyID,
			ResourceID: resource.ID,
			Message:    evaluation.Message,
			Severity:   evaluation.Status,
			Timestamp:  time.Now(),
			Metadata: map[string]interface{}{
				"evaluation_status": evaluation.Status,
				"violation_count":   len(evaluation.Violations),
			},
		}
		_ = pe.eventBus.PublishComplianceEvent(event)
	}

	return evaluation, nil
}

// PolicyEvaluation represents the result of policy evaluation
type PolicyEvaluation struct {
	PolicyID    string                 `json:"policy_id"`
	ResourceID  string                 `json:"resource_id"`
	Status      string                 `json:"status"` // COMPLIANT, NON_COMPLIANT, NOT_APPLICABLE
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	RuleResults []RuleEvaluation       `json:"rule_results"`
	Violations  []PolicyViolation      `json:"violations"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RuleEvaluation represents the result of rule evaluation
type RuleEvaluation struct {
	RuleID    string      `json:"rule_id"`
	Status    string      `json:"status"` // COMPLIANT, VIOLATED
	Message   string      `json:"message"`
	Field     string      `json:"field,omitempty"`
	Expected  interface{} `json:"expected,omitempty"`
	Actual    interface{} `json:"actual,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`
	Field       string                 `json:"field"`
	Expected    interface{}            `json:"expected"`
	Actual      interface{}            `json:"actual"`
	Remediation string                 `json:"remediation"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// validatePolicy validates a security policy
func (pe *PolicyEngine) validatePolicy(policy *SecurityPolicy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if policy.Category == "" {
		return fmt.Errorf("policy category is required")
	}
	if len(policy.Rules) == 0 {
		return fmt.Errorf("policy must have at least one rule")
	}
	return nil
}

// validateRule validates a security rule
func (pe *PolicyEngine) validateRule(rule *SecurityRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}
	return nil
}

// policyApplies checks if a policy applies to a resource
func (pe *PolicyEngine) policyApplies(policy *SecurityPolicy, resource *models.Resource) bool {
	scope := policy.Scope

	// Check tenant scope
	if len(scope.Tenants) > 0 {
		// This would check if resource belongs to any of the specified tenants
		// For now, assume it applies
	}

	// Check account scope
	if len(scope.Accounts) > 0 {
		// This would check if resource belongs to any of the specified accounts
		// For now, assume it applies
	}

	// Check region scope
	if len(scope.Regions) > 0 {
		found := false
		for _, region := range scope.Regions {
			if resource.Region == region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check provider scope
	if len(scope.Providers) > 0 {
		found := false
		for _, provider := range scope.Providers {
			if resource.Provider == provider {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check resource type scope
	if len(scope.ResourceTypes) > 0 {
		found := false
		for _, resourceType := range scope.ResourceTypes {
			if resource.Type == resourceType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check tag scope
	if len(scope.Tags) > 0 {
		if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
			for key, value := range scope.Tags {
				if tagValue, exists := tags[key]; !exists || tagValue != value {
					return false
				}
			}
		} else {
			return false
		}
	}

	return true
}

// evaluateRule evaluates a security rule against a resource
func (pe *PolicyEngine) evaluateRule(rule *SecurityRule, resource *models.Resource) RuleEvaluation {
	result := RuleEvaluation{
		RuleID:    rule.ID,
		Status:    "COMPLIANT",
		Message:   "Rule compliance check passed",
		Timestamp: time.Now(),
	}

	// Evaluate each condition
	for _, condition := range rule.Conditions {
		if !pe.evaluateCondition(condition, resource) {
			result.Status = "VIOLATED"
			result.Message = fmt.Sprintf("Rule '%s' violated", rule.Name)
			result.Field = condition.Field
			result.Expected = condition.Value
			result.Actual = pe.getResourceValue(resource, condition.Field)
			break
		}
	}

	return result
}

// evaluateCondition evaluates a single condition
func (pe *PolicyEngine) evaluateCondition(condition RuleCondition, resource *models.Resource) bool {
	actualValue := pe.getResourceValue(resource, condition.Field)

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
		return pe.compareValues(actualValue, condition.Value) > 0
	case "less_than":
		return pe.compareValues(actualValue, condition.Value) < 0
	default:
		return false
	}
}

// getResourceValue gets a value from a resource
func (pe *PolicyEngine) getResourceValue(resource *models.Resource, field string) interface{} {
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
func (pe *PolicyEngine) compareValues(a, b interface{}) int {
	// Simplified comparison - in reality, you'd handle different types
	if a == b {
		return 0
	}
	// This is a placeholder - real implementation would be more sophisticated
	return 1
}

// generateRemediation generates remediation advice
func (pe *PolicyEngine) generateRemediation(rule *SecurityRule) string {
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
	case "network_security":
		return "Review network security configuration"
	case "data_protection":
		return "Implement data protection measures"
	default:
		return "Review resource configuration"
	}
}

// determineAction determines the action to take based on evaluation
func (pe *PolicyEngine) determineAction(evaluation *PolicyEvaluation) string {
	if evaluation.Status == "COMPLIANT" {
		return "none"
	}

	// Check if any violations are critical
	for _, violation := range evaluation.Violations {
		if violation.Severity == "critical" {
			return "block"
		}
	}

	// Check if any violations are high severity
	for _, violation := range evaluation.Violations {
		if violation.Severity == "high" {
			return "warn"
		}
	}

	return "warn"
}

// SetConfig updates the policy engine configuration
func (pe *PolicyEngine) SetConfig(config *PolicyConfig) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.config = config
}

// GetConfig returns the current policy engine configuration
func (pe *PolicyEngine) GetConfig() *PolicyConfig {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.config
}
