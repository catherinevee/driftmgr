package types

import (
	"time"
)

// ResourceChange represents a change to a Terraform resource
type ResourceChange struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Module    string                 `json:"module"`
	Provider  string                 `json:"provider"`
	Action    ChangeAction           `json:"action"`
	Before    map[string]interface{} `json:"before"`
	After     map[string]interface{} `json:"after"`
	Changes   map[string]Change      `json:"changes"`
	Metadata  ResourceMetadata       `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
}

// ChangeAction represents the type of change
type ChangeAction string

const (
	ActionCreate ChangeAction = "create"
	ActionUpdate ChangeAction = "update"
	ActionDelete ChangeAction = "delete"
	ActionNoOp   ChangeAction = "no-op"
)

// Change represents a specific field change
type Change struct {
	Before interface{}  `json:"before"`
	After  interface{}  `json:"after"`
	Action ChangeAction `json:"action"`
}

// ResourceMetadata contains additional resource information
type ResourceMetadata struct {
	Tags       map[string]string `json:"tags"`
	Cost       *CostInfo         `json:"cost,omitempty"`
	Security   *SecurityInfo     `json:"security,omitempty"`
	Compliance *ComplianceInfo   `json:"compliance,omitempty"`
}

// CostInfo contains cost-related information
type CostInfo struct {
	MonthlyCost float64 `json:"monthly_cost"`
	Currency    string  `json:"currency"`
	Provider    string  `json:"provider"`
}

// SecurityInfo contains security-related information
type SecurityInfo struct {
	RiskLevel       string   `json:"risk_level"`
	Vulnerabilities []string `json:"vulnerabilities"`
	Compliance      []string `json:"compliance"`
}

// ComplianceInfo contains compliance-related information
type ComplianceInfo struct {
	Standards  []string  `json:"standards"`
	Violations []string  `json:"violations"`
	LastAudit  time.Time `json:"last_audit"`
}

// NewResourceChange creates a new ResourceChange instance with validation
func NewResourceChange(id, resourceType, name, module, provider string, action ChangeAction) (*ResourceChange, error) {
	if id == "" {
		return nil, &ValidationError{Field: "id", Message: "resource ID cannot be empty"}
	}
	if resourceType == "" {
		return nil, &ValidationError{Field: "type", Message: "resource type cannot be empty"}
	}
	if name == "" {
		return nil, &ValidationError{Field: "name", Message: "resource name cannot be empty"}
	}
	if provider == "" {
		return nil, &ValidationError{Field: "provider", Message: "provider cannot be empty"}
	}

	return &ResourceChange{
		ID:        id,
		Type:      resourceType,
		Name:      name,
		Module:    module,
		Provider:  provider,
		Action:    action,
		Before:    make(map[string]interface{}),
		After:     make(map[string]interface{}),
		Changes:   make(map[string]Change),
		Metadata:  ResourceMetadata{Tags: make(map[string]string)},
		Timestamp: time.Now(),
	}, nil
}

// AddChange adds a field change to the resource change
func (rc *ResourceChange) AddChange(field string, before, after interface{}, action ChangeAction) {
	rc.Changes[field] = Change{
		Before: before,
		After:  after,
		Action: action,
	}
}

// SetBeforeState sets the before state of the resource
func (rc *ResourceChange) SetBeforeState(state map[string]interface{}) {
	rc.Before = sanitizeState(state)
}

// SetAfterState sets the after state of the resource
func (rc *ResourceChange) SetAfterState(state map[string]interface{}) {
	rc.After = sanitizeState(state)
}

// SetCostInfo sets cost information for the resource
func (rc *ResourceChange) SetCostInfo(cost *CostInfo) {
	rc.Metadata.Cost = cost
}

// SetSecurityInfo sets security information for the resource
func (rc *ResourceChange) SetSecurityInfo(security *SecurityInfo) {
	rc.Metadata.Security = security
}

// SetComplianceInfo sets compliance information for the resource
func (rc *ResourceChange) SetComplianceInfo(compliance *ComplianceInfo) {
	rc.Metadata.Compliance = compliance
}

// AddTag adds a tag to the resource metadata
func (rc *ResourceChange) AddTag(key, value string) {
	if rc.Metadata.Tags == nil {
		rc.Metadata.Tags = make(map[string]string)
	}
	rc.Metadata.Tags[key] = value
}

// GetResourceIdentifier returns a unique identifier for the resource
func (rc *ResourceChange) GetResourceIdentifier() string {
	if rc.Module != "" {
		return rc.Module + "." + rc.Type + "." + rc.Name
	}
	return rc.Type + "." + rc.Name
}

// IsCreate returns true if this is a create action
func (rc *ResourceChange) IsCreate() bool {
	return rc.Action == ActionCreate
}

// IsUpdate returns true if this is an update action
func (rc *ResourceChange) IsUpdate() bool {
	return rc.Action == ActionUpdate
}

// IsDelete returns true if this is a delete action
func (rc *ResourceChange) IsDelete() bool {
	return rc.Action == ActionDelete
}

// IsNoOp returns true if this is a no-op action
func (rc *ResourceChange) IsNoOp() bool {
	return rc.Action == ActionNoOp
}

// HasChanges returns true if the resource has field changes
func (rc *ResourceChange) HasChanges() bool {
	return len(rc.Changes) > 0
}

// GetChangeCount returns the number of field changes
func (rc *ResourceChange) GetChangeCount() int {
	return len(rc.Changes)
}

// sanitizeState removes sensitive information from state data
func sanitizeState(state map[string]interface{}) map[string]interface{} {
	if state == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	sensitiveFields := []string{"password", "secret", "key", "token", "credential"}

	for key, value := range state {
		// Check if field contains sensitive information
		isSensitive := false
		for _, sensitive := range sensitiveFields {
			if containsIgnoreCase(key, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[key] = "[REDACTED]"
		} else {
			sanitized[key] = value
		}
	}

	return sanitized
}

// containsIgnoreCase checks if a string contains a substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (ve *ValidationError) Error() string {
	return "validation error in field '" + ve.Field + "': " + ve.Message
}
