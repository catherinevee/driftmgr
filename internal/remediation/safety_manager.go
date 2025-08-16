package remediation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// SafetyManager provides production environment protection and compliance policy enforcement
type SafetyManager struct {
	policies map[string]SafetyPolicy
	mu       sync.RWMutex
}

// SafetyPolicy defines safety rules for remediation
type SafetyPolicy struct {
	Name           string
	Description    string
	Rules          []SafetyRule
	Enforcement    EnforcementLevel
	LastUpdated    time.Time
}

// SafetyRule defines a single safety rule
type SafetyRule struct {
	RuleID       string
	Description  string
	Condition    string // "production_tag", "business_hours", "critical_resource"
	Parameters   map[string]interface{}
	Action       string // "block", "warn", "require_approval"
	Message      string
}

// EnforcementLevel represents how strictly a policy is enforced
type EnforcementLevel string

const (
	EnforcementLevelAdvisory EnforcementLevel = "advisory"
	EnforcementLevelWarning  EnforcementLevel = "warning"
	EnforcementLevelBlocking EnforcementLevel = "blocking"
)

// NewSafetyManager creates a new safety manager
func NewSafetyManager() *SafetyManager {
	sm := &SafetyManager{
		policies: make(map[string]SafetyPolicy),
	}
	
	// Register default policies
	sm.registerDefaultPolicies()
	
	return sm
}

// RegisterPolicy registers a safety policy
func (sm *SafetyManager) RegisterPolicy(name string, policy SafetyPolicy) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.policies[name] = policy
}

// ValidateRemediation validates if a remediation is safe to proceed
func (sm *SafetyManager) ValidateRemediation(
	ctx context.Context,
	drift DriftAnalysis,
	resource models.Resource,
	options RemediationOptions,
) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var violations []string

	// Check each policy
	for _, policy := range sm.policies {
		policyViolations := sm.checkPolicy(policy, drift, resource, options)
		violations = append(violations, policyViolations...)
	}

	// Handle violations based on enforcement level
	if len(violations) > 0 {
		if options.Force {
			// Log violations but allow with force flag
			fmt.Printf("Safety violations detected but proceeding with force flag: %v\n", violations)
			return nil
		}

		return fmt.Errorf("safety validation failed: %s", strings.Join(violations, "; "))
	}

	return nil
}

// checkPolicy checks a single policy against the remediation
func (sm *SafetyManager) checkPolicy(
	policy SafetyPolicy,
	drift DriftAnalysis,
	resource models.Resource,
	options RemediationOptions,
) []string {
	var violations []string

	for _, rule := range policy.Rules {
		if sm.evaluateRule(rule, drift, resource, options) {
			violations = append(violations, rule.Message)
		}
	}

	return violations
}

// evaluateRule evaluates a single safety rule
func (sm *SafetyManager) evaluateRule(
	rule SafetyRule,
	drift DriftAnalysis,
	resource models.Resource,
	options RemediationOptions,
) bool {
	switch rule.Condition {
	case "production_tag":
		return sm.checkProductionTag(rule, resource)
	case "business_hours":
		return sm.checkBusinessHours(rule)
	case "critical_resource":
		return sm.checkCriticalResource(rule, resource)
	case "high_severity_drift":
		return sm.checkHighSeverityDrift(rule, drift)
	case "cost_threshold":
		return sm.checkCostThreshold(rule, resource)
	default:
		return false
	}
}

// checkProductionTag checks if resource has production tags
func (sm *SafetyManager) checkProductionTag(rule SafetyRule, resource models.Resource) bool {
	productionTags := []string{"production", "prod", "live", "critical"}
	
	for _, tag := range productionTags {
		for key, value := range resource.Tags {
			if strings.Contains(strings.ToLower(key), tag) || 
			   strings.Contains(strings.ToLower(value), tag) {
				return true
			}
		}
	}
	
	return false
}

// checkBusinessHours checks if current time is within business hours
func (sm *SafetyManager) checkBusinessHours(rule SafetyRule) bool {
	now := time.Now()
	
	// Default business hours: 9 AM - 5 PM, Monday-Friday
	startHour := 9
	endHour := 17
	
	if startHourParam, exists := rule.Parameters["start_hour"]; exists {
		if start, ok := startHourParam.(int); ok {
			startHour = start
		}
	}
	
	if endHourParam, exists := rule.Parameters["end_hour"]; exists {
		if end, ok := endHourParam.(int); ok {
			endHour = end
		}
	}
	
	// Check if it's a weekday
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return true // Outside business hours
	}
	
	// Check if it's within business hours
	if now.Hour() < startHour || now.Hour() >= endHour {
		return true // Outside business hours
	}
	
	return false
}

// checkCriticalResource checks if resource is marked as critical
func (sm *SafetyManager) checkCriticalResource(rule SafetyRule, resource models.Resource) bool {
	criticalIndicators := []string{"critical", "essential", "core", "primary"}
	
	for _, indicator := range criticalIndicators {
		for key, value := range resource.Tags {
			if strings.Contains(strings.ToLower(key), indicator) || 
			   strings.Contains(strings.ToLower(value), indicator) {
				return true
			}
		}
		
		// Check resource name
		if strings.Contains(strings.ToLower(resource.Name), indicator) {
			return true
		}
	}
	
	return false
}

// checkHighSeverityDrift checks if drift is high severity
func (sm *SafetyManager) checkHighSeverityDrift(rule SafetyRule, drift DriftAnalysis) bool {
	highSeverityLevels := []string{"high", "critical"}
	
	for _, level := range highSeverityLevels {
		if strings.ToLower(drift.Severity) == level {
			return true
		}
	}
	
	return false
}

// checkCostThreshold checks if resource cost exceeds threshold
func (sm *SafetyManager) checkCostThreshold(rule SafetyRule, resource models.Resource) bool {
	_, exists := rule.Parameters["threshold"]
	if !exists {
		return false
	}
	
	// This would integrate with cost analysis
	// For now, return false (no cost threshold violation)
	return false
}

// registerDefaultPolicies registers default safety policies
func (sm *SafetyManager) registerDefaultPolicies() {
	// Production Environment Protection Policy
	productionPolicy := SafetyPolicy{
		Name:        "production_protection",
		Description: "Protects production environments from accidental changes",
		Enforcement: EnforcementLevelBlocking,
		Rules: []SafetyRule{
			{
				RuleID:      "prod_tag_check",
				Description: "Block changes to resources with production tags",
				Condition:   "production_tag",
				Action:      "block",
				Message:     "Resource has production tags - manual approval required",
			},
			{
				RuleID:      "business_hours_check",
				Description: "Warn about changes outside business hours",
				Condition:   "business_hours",
				Action:      "warn",
				Message:     "Changes outside business hours detected",
			},
		},
		LastUpdated: time.Now(),
	}
	
	// Critical Resource Protection Policy
	criticalPolicy := SafetyPolicy{
		Name:        "critical_resource_protection",
		Description: "Protects critical resources from changes",
		Enforcement: EnforcementLevelBlocking,
		Rules: []SafetyRule{
			{
				RuleID:      "critical_resource_check",
				Description: "Block changes to critical resources",
				Condition:   "critical_resource",
				Action:      "block",
				Message:     "Resource is marked as critical - manual approval required",
			},
		},
		LastUpdated: time.Now(),
	}
	
	// High Severity Drift Policy
	severityPolicy := SafetyPolicy{
		Name:        "high_severity_protection",
		Description: "Requires approval for high severity drift remediation",
		Enforcement: EnforcementLevelWarning,
		Rules: []SafetyRule{
			{
				RuleID:      "high_severity_check",
				Description: "Require approval for high severity drift",
				Condition:   "high_severity_drift",
				Action:      "require_approval",
				Message:     "High severity drift detected - approval required",
			},
		},
		LastUpdated: time.Now(),
	}
	
	sm.RegisterPolicy("production_protection", productionPolicy)
	sm.RegisterPolicy("critical_resource_protection", criticalPolicy)
	sm.RegisterPolicy("high_severity_protection", severityPolicy)
}
