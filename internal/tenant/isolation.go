package tenant

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// ResourceIsolationManager manages resource isolation between tenants
type ResourceIsolationManager struct {
	isolationRules map[string]*IsolationRule
	resourceTags   map[string]*ResourceTag
	mu             sync.RWMutex
	eventBus       EventBus
	config         *IsolationConfig
}

// IsolationRule represents a rule for resource isolation
type IsolationRule struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Conditions  []IsolationCondition   `json:"conditions"`
	Actions     []IsolationAction      `json:"actions"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// IsolationCondition represents a condition for isolation
type IsolationCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type"`
}

// IsolationAction represents an action to take for isolation
type IsolationAction struct {
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
}

// ResourceTag represents a tag for resource identification
type ResourceTag struct {
	ResourceID string                 `json:"resource_id"`
	TenantID   string                 `json:"tenant_id"`
	AccountID  string                 `json:"account_id"`
	Tags       map[string]string      `json:"tags"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// IsolationConfig represents configuration for isolation
type IsolationConfig struct {
	EnforceIsolation   bool     `json:"enforce_isolation"`
	DefaultIsolation   string   `json:"default_isolation"`
	AllowedCrossTenant []string `json:"allowed_cross_tenant"`
	TagBasedIsolation  bool     `json:"tag_based_isolation"`
	NetworkIsolation   bool     `json:"network_isolation"`
	StorageIsolation   bool     `json:"storage_isolation"`
	ComputeIsolation   bool     `json:"compute_isolation"`
}

// IsolationViolation represents a violation of isolation rules
type IsolationViolation struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ResourceID  string                 `json:"resource_id"`
	RuleID      string                 `json:"rule_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewResourceIsolationManager creates a new resource isolation manager
func NewResourceIsolationManager(eventBus EventBus) *ResourceIsolationManager {
	config := &IsolationConfig{
		EnforceIsolation:   true,
		DefaultIsolation:   "strict",
		AllowedCrossTenant: []string{},
		TagBasedIsolation:  true,
		NetworkIsolation:   true,
		StorageIsolation:   true,
		ComputeIsolation:   true,
	}

	return &ResourceIsolationManager{
		isolationRules: make(map[string]*IsolationRule),
		resourceTags:   make(map[string]*ResourceTag),
		eventBus:       eventBus,
		config:         config,
	}
}

// CreateIsolationRule creates a new isolation rule
func (rim *ResourceIsolationManager) CreateIsolationRule(ctx context.Context, rule *IsolationRule) error {
	rim.mu.Lock()
	defer rim.mu.Unlock()

	// Validate rule
	if err := rim.validateIsolationRule(rule); err != nil {
		return fmt.Errorf("invalid isolation rule: %w", err)
	}

	// Set defaults
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("isolation_rule_%d", time.Now().Unix())
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	// Store rule
	rim.isolationRules[rule.ID] = rule

	// Publish event
	if rim.eventBus != nil {
		event := TenantEvent{
			Type:      "isolation_rule_created",
			TenantID:  rule.TenantID,
			Message:   fmt.Sprintf("Isolation rule '%s' created", rule.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
				"rule_type": rule.Type,
			},
		}
		rim.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// TagResource tags a resource for isolation
func (rim *ResourceIsolationManager) TagResource(ctx context.Context, resourceID, tenantID, accountID string, tags map[string]string) error {
	rim.mu.Lock()
	defer rim.mu.Unlock()

	// Create or update resource tag
	resourceTag := &ResourceTag{
		ResourceID: resourceID,
		TenantID:   tenantID,
		AccountID:  accountID,
		Tags:       tags,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	rim.resourceTags[resourceID] = resourceTag

	// Publish event
	if rim.eventBus != nil {
		event := TenantEvent{
			Type:      "resource_tagged",
			TenantID:  tenantID,
			Message:   fmt.Sprintf("Resource %s tagged for tenant %s", resourceID, tenantID),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"resource_id": resourceID,
				"account_id":  accountID,
				"tags":        tags,
			},
		}
		rim.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// CheckIsolation checks if a resource violates isolation rules
func (rim *ResourceIsolationManager) CheckIsolation(ctx context.Context, resource *models.Resource, tenantID string) ([]IsolationViolation, error) {
	rim.mu.RLock()
	defer rim.mu.RUnlock()

	var violations []IsolationViolation

	// Check if isolation is enforced
	if !rim.config.EnforceIsolation {
		return violations, nil
	}

	// Get resource tags
	resourceTag, exists := rim.resourceTags[resource.ID]
	if !exists {
		// Resource not tagged - create default tag
		resourceTag = &ResourceTag{
			ResourceID: resource.ID,
			TenantID:   tenantID,
			Tags:       make(map[string]string),
		}
	}

	// Check isolation rules
	for _, rule := range rim.isolationRules {
		if !rule.Enabled {
			continue
		}

		// Check if rule applies to this tenant
		if rule.TenantID != "" && rule.TenantID != tenantID {
			continue
		}

		// Check conditions
		if rim.conditionsMatch(rule.Conditions, resource, resourceTag) {
			// Check for violations
			violation := rim.checkRuleViolation(rule, resource, resourceTag, tenantID)
			if violation != nil {
				violations = append(violations, *violation)
			}
		}
	}

	return violations, nil
}

// EnforceIsolation enforces isolation rules for a resource
func (rim *ResourceIsolationManager) EnforceIsolation(ctx context.Context, resource *models.Resource, tenantID string) error {
	rim.mu.Lock()
	defer rim.mu.Unlock()

	// Check for violations
	violations, err := rim.CheckIsolation(ctx, resource, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check isolation: %w", err)
	}

	// Handle violations
	for _, violation := range violations {
		if err := rim.handleViolation(ctx, violation); err != nil {
			// Log error but continue with other violations
			fmt.Printf("Warning: failed to handle violation %s: %v\n", violation.ID, err)
		}
	}

	return nil
}

// GetResourceTags retrieves tags for a resource
func (rim *ResourceIsolationManager) GetResourceTags(ctx context.Context, resourceID string) (*ResourceTag, error) {
	rim.mu.RLock()
	defer rim.mu.RUnlock()

	resourceTag, exists := rim.resourceTags[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource tags for %s not found", resourceID)
	}

	return resourceTag, nil
}

// ListIsolationRules lists all isolation rules
func (rim *ResourceIsolationManager) ListIsolationRules(ctx context.Context) ([]*IsolationRule, error) {
	rim.mu.RLock()
	defer rim.mu.RUnlock()

	rules := make([]*IsolationRule, 0, len(rim.isolationRules))
	for _, rule := range rim.isolationRules {
		rules = append(rules, rule)
	}

	return rules, nil
}

// CreateDefaultIsolationRules creates default isolation rules
func (rim *ResourceIsolationManager) CreateDefaultIsolationRules(ctx context.Context) error {
	defaultRules := []*IsolationRule{
		{
			Name:        "Tenant Resource Isolation",
			Description: "Ensures resources are properly isolated by tenant",
			Type:        "tenant_isolation",
			Conditions: []IsolationCondition{
				{
					Field:    "tenant_id",
					Operator: "not_equals",
					Value:    "",
					Type:     "string",
				},
			},
			Actions: []IsolationAction{
				{
					Type:        "tag_resource",
					Description: "Tag resource with tenant ID",
					Parameters: map[string]interface{}{
						"tag_key":   "tenant_id",
						"tag_value": "{{tenant_id}}",
					},
				},
			},
			Priority: 100,
			Enabled:  true,
		},
		{
			Name:        "Network Isolation",
			Description: "Ensures network resources are isolated by tenant",
			Type:        "network_isolation",
			Conditions: []IsolationCondition{
				{
					Field:    "resource_type",
					Operator: "in",
					Value:    []string{"aws_vpc", "azurerm_virtual_network", "google_compute_network"},
					Type:     "array",
				},
			},
			Actions: []IsolationAction{
				{
					Type:        "isolate_network",
					Description: "Isolate network resources",
					Parameters: map[string]interface{}{
						"isolation_type": "vpc",
					},
				},
			},
			Priority: 90,
			Enabled:  true,
		},
		{
			Name:        "Storage Isolation",
			Description: "Ensures storage resources are isolated by tenant",
			Type:        "storage_isolation",
			Conditions: []IsolationCondition{
				{
					Field:    "resource_type",
					Operator: "in",
					Value:    []string{"aws_s3_bucket", "azurerm_storage_account", "google_storage_bucket"},
					Type:     "array",
				},
			},
			Actions: []IsolationAction{
				{
					Type:        "isolate_storage",
					Description: "Isolate storage resources",
					Parameters: map[string]interface{}{
						"isolation_type": "bucket",
					},
				},
			},
			Priority: 80,
			Enabled:  true,
		},
	}

	for _, rule := range defaultRules {
		if err := rim.CreateIsolationRule(ctx, rule); err != nil {
			return fmt.Errorf("failed to create default rule %s: %w", rule.Name, err)
		}
	}

	return nil
}

// Helper methods

// validateIsolationRule validates an isolation rule
func (rim *ResourceIsolationManager) validateIsolationRule(rule *IsolationRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("at least one condition is required")
	}
	if len(rule.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}
	return nil
}

// conditionsMatch checks if conditions match for a resource
func (rim *ResourceIsolationManager) conditionsMatch(conditions []IsolationCondition, resource *models.Resource, resourceTag *ResourceTag) bool {
	for _, condition := range conditions {
		if !rim.conditionMatches(condition, resource, resourceTag) {
			return false
		}
	}
	return true
}

// conditionMatches checks if a single condition matches
func (rim *ResourceIsolationManager) conditionMatches(condition IsolationCondition, resource *models.Resource, resourceTag *ResourceTag) bool {
	var actualValue interface{}

	// Get value based on field
	switch condition.Field {
	case "tenant_id":
		actualValue = resourceTag.TenantID
	case "account_id":
		actualValue = resourceTag.AccountID
	case "resource_type":
		actualValue = resource.Type
	case "resource_region":
		actualValue = resource.Region
	case "resource_provider":
		actualValue = resource.Provider
	default:
		// Check resource attributes
		if val, ok := resource.Attributes[condition.Field]; ok {
			actualValue = val
		} else {
			return false
		}
	}

	// Apply operator
	switch condition.Operator {
	case "equals":
		return actualValue == condition.Value
	case "not_equals":
		return actualValue != condition.Value
	case "in":
		if arr, ok := condition.Value.([]string); ok {
			if str, ok := actualValue.(string); ok {
				for _, v := range arr {
					if v == str {
						return true
					}
				}
			}
		}
		return false
	case "not_in":
		if arr, ok := condition.Value.([]string); ok {
			if str, ok := actualValue.(string); ok {
				for _, v := range arr {
					if v == str {
						return false
					}
				}
			}
		}
		return true
	default:
		return false
	}
}

// checkRuleViolation checks if a rule is violated
func (rim *ResourceIsolationManager) checkRuleViolation(rule *IsolationRule, resource *models.Resource, resourceTag *ResourceTag, tenantID string) *IsolationViolation {
	// Check for common violations
	if rule.Type == "tenant_isolation" {
		if resourceTag.TenantID != tenantID {
			return &IsolationViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().Unix()),
				TenantID:    tenantID,
				ResourceID:  resource.ID,
				RuleID:      rule.ID,
				Type:        "tenant_mismatch",
				Severity:    "high",
				Description: fmt.Sprintf("Resource %s belongs to tenant %s but accessed by tenant %s", resource.ID, resourceTag.TenantID, tenantID),
				DetectedAt:  time.Now(),
				Status:      "active",
				Metadata: map[string]interface{}{
					"expected_tenant": tenantID,
					"actual_tenant":   resourceTag.TenantID,
				},
			}
		}
	}

	return nil
}

// handleViolation handles an isolation violation
func (rim *ResourceIsolationManager) handleViolation(ctx context.Context, violation IsolationViolation) error {
	// In a real implementation, you would take appropriate actions
	// such as blocking access, logging, alerting, etc.

	// For now, just log the violation
	fmt.Printf("Isolation violation detected: %s - %s\n", violation.Type, violation.Description)

	// Publish event
	if rim.eventBus != nil {
		event := TenantEvent{
			Type:      "isolation_violation",
			TenantID:  violation.TenantID,
			Message:   fmt.Sprintf("Isolation violation: %s", violation.Description),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"violation_id":   violation.ID,
				"resource_id":    violation.ResourceID,
				"rule_id":        violation.RuleID,
				"violation_type": violation.Type,
				"severity":       violation.Severity,
			},
		}
		rim.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// SetConfig updates the isolation manager configuration
func (rim *ResourceIsolationManager) SetConfig(config *IsolationConfig) {
	rim.mu.Lock()
	defer rim.mu.Unlock()
	rim.config = config
}

// GetConfig returns the current isolation manager configuration
func (rim *ResourceIsolationManager) GetConfig() *IsolationConfig {
	rim.mu.RLock()
	defer rim.mu.RUnlock()
	return rim.config
}
