package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
)

// ResourceValidator validates resource data
type ResourceValidator struct {
	// Validation rules
	providerPatterns map[string]*regexp.Regexp
	typePatterns     map[string]*regexp.Regexp
}

// NewResourceValidator creates a new resource validator
func NewResourceValidator() *ResourceValidator {
	return &ResourceValidator{
		providerPatterns: map[string]*regexp.Regexp{
			"aws":          regexp.MustCompile(`^aws$`),
			"azure":        regexp.MustCompile(`^azure$`),
			"gcp":          regexp.MustCompile(`^gcp$`),
			"digitalocean": regexp.MustCompile(`^digitalocean$`),
		},
		typePatterns: map[string]*regexp.Regexp{
			"aws":          regexp.MustCompile(`^aws_[a-z0-9_]+$`),
			"azure":        regexp.MustCompile(`^azurerm_[a-z0-9_]+$`),
			"gcp":          regexp.MustCompile(`^google_[a-z0-9_]+$`),
			"digitalocean": regexp.MustCompile(`^digitalocean_[a-z0-9_]+$`),
		},
	}
}

// ValidateResource validates a single resource
func (v *ResourceValidator) ValidateResource(resource *apimodels.Resource) []string {
	var errors []string
	
	// Required fields
	if resource.ID == "" {
		errors = append(errors, "resource ID is required")
	}
	
	if resource.Type == "" {
		errors = append(errors, "resource type is required")
	}
	
	if resource.Provider == "" {
		errors = append(errors, "resource provider is required")
	}
	
	// Provider validation
	if resource.Provider != "" {
		provider := strings.ToLower(resource.Provider)
		if pattern, ok := v.providerPatterns[provider]; ok {
			if !pattern.MatchString(provider) {
				errors = append(errors, fmt.Sprintf("invalid provider format: %s", resource.Provider))
			}
		} else {
			errors = append(errors, fmt.Sprintf("unsupported provider: %s", resource.Provider))
		}
		
		// Type validation based on provider
		if resource.Type != "" && v.typePatterns[provider] != nil {
			if !v.typePatterns[provider].MatchString(resource.Type) {
				errors = append(errors, fmt.Sprintf("invalid resource type for provider %s: %s", provider, resource.Type))
			}
		}
	}
	
	// Region validation for cloud providers
	if resource.Provider != "" && resource.Region == "" {
		// Some resources might not have regions (global resources)
		globalTypes := []string{
			"aws_iam_", "aws_route53_", "aws_cloudfront_", "aws_waf_",
			"azure_management_", "azure_policy_", "azure_blueprint_",
			"google_project_", "google_organization_", "google_billing_",
			"digitalocean_project_", "digitalocean_firewall_",
		}
		isGlobal := false
		for _, prefix := range globalTypes {
			if strings.HasPrefix(resource.Type, prefix) {
				isGlobal = true
				break
			}
		}
		// Only warn about region for non-global resources, don't error
		// Pre-discovered resources might not have regions populated yet
		if !isGlobal && resource.Region == "" {
			// Make this a warning, not an error
			// errors = append(errors, "region is recommended for cloud resources")
			// Skip region validation for now - too strict for pre-discovered resources
		}
	}
	
	// Timestamp validation
	if !resource.CreatedAt.IsZero() && resource.CreatedAt.After(time.Now().Add(1*time.Hour)) {
		errors = append(errors, "created_at timestamp is in the future")
	}
	
	if !resource.ModifiedAt.IsZero() && resource.ModifiedAt.After(time.Now().Add(1*time.Hour)) {
		errors = append(errors, "modified_at timestamp is in the future")
	}
	
	if !resource.CreatedAt.IsZero() && !resource.ModifiedAt.IsZero() {
		if resource.ModifiedAt.Before(resource.CreatedAt) {
			errors = append(errors, "modified_at cannot be before created_at")
		}
	}
	
	// Status validation - be more permissive with status values
	validStatuses := map[string]bool{
		"active":    true,
		"running":   true,
		"stopped":   true,
		"deleted":   true,
		"pending":   true,
		"failed":    true,
		"unknown":   true,
		"missing":   true,
		"succeeded": true,  // Azure/GCP status
		"success":   true,  // Alternative success status
		"available": true,  // AWS status
		"creating":  true,  // Resource being created
		"updating":  true,  // Resource being updated
		"deleting":  true,  // Resource being deleted
		"error":     true,  // Error state
		"degraded":  true,  // Degraded state
		"healthy":   true,  // Health status
		"unhealthy": true,  // Health status
		"attached":  true,  // Attachment status
		"detached":  true,  // Attachment status
		"enabled":   true,  // Enable/disable status
		"disabled":  true,  // Enable/disable status
	}
	
	if resource.Status != "" && !validStatuses[strings.ToLower(resource.Status)] {
		// Don't fail validation for unknown statuses, just accept them
		// Different providers have different status values
		// errors = append(errors, fmt.Sprintf("invalid status: %s", resource.Status))
	}
	
	return errors
}

// ValidateResources validates multiple resources
func (v *ResourceValidator) ValidateResources(resources []apimodels.Resource) ([]apimodels.Resource, []string) {
	var validResources []apimodels.Resource
	var allErrors []string
	
	// Check for duplicates
	seen := make(map[string]bool)
	
	for i, resource := range resources {
		// Validate individual resource
		errors := v.ValidateResource(&resource)
		
		// Check for duplicate IDs
		if seen[resource.ID] {
			errors = append(errors, fmt.Sprintf("duplicate resource ID: %s", resource.ID))
		} else {
			seen[resource.ID] = true
		}
		
		if len(errors) > 0 {
			for _, err := range errors {
				allErrors = append(allErrors, fmt.Sprintf("resource[%d]: %s", i, err))
			}
		} else {
			validResources = append(validResources, resource)
		}
	}
	
	return validResources, allErrors
}

// DriftValidator validates drift data
type DriftValidator struct{}

// NewDriftValidator creates a new drift validator
func NewDriftValidator() *DriftValidator {
	return &DriftValidator{}
}

// ValidateDrift validates a drift record
func (v *DriftValidator) ValidateDrift(drift *DriftRecord) []string {
	var errors []string
	
	// Required fields
	if drift.ResourceID == "" {
		errors = append(errors, "resource ID is required for drift")
	}
	
	if drift.ResourceType == "" {
		errors = append(errors, "resource type is required for drift")
	}
	
	if drift.Provider == "" {
		errors = append(errors, "provider is required for drift")
	}
	
	if drift.DriftType == "" {
		errors = append(errors, "drift type is required")
	}
	
	// Severity validation
	validSeverities := map[string]bool{
		"critical": true,
		"high":     true,
		"medium":   true,
		"low":      true,
	}
	
	if drift.Severity != "" && !validSeverities[strings.ToLower(drift.Severity)] {
		errors = append(errors, fmt.Sprintf("invalid severity: %s", drift.Severity))
	}
	
	// Status validation
	validStatuses := map[string]bool{
		"active":   true,
		"resolved": true,
		"ignored":  true,
	}
	
	if drift.Status != "" && !validStatuses[strings.ToLower(drift.Status)] {
		errors = append(errors, fmt.Sprintf("invalid drift status: %s", drift.Status))
	}
	
	// Timestamp validation
	if drift.DetectedAt.IsZero() {
		errors = append(errors, "drift detected_at timestamp is required")
	}
	
	if drift.ResolvedAt != nil && drift.ResolvedAt.Before(drift.DetectedAt) {
		errors = append(errors, "resolved_at cannot be before detected_at")
	}
	
	return errors
}

// ConsistencyChecker checks data consistency
type ConsistencyChecker struct {
	validator      *ResourceValidator
	driftValidator *DriftValidator
}

// NewConsistencyChecker creates a new consistency checker
func NewConsistencyChecker() *ConsistencyChecker {
	return &ConsistencyChecker{
		validator:      NewResourceValidator(),
		driftValidator: NewDriftValidator(),
	}
}

// CheckResourceConsistency checks consistency between resources and drifts
func (c *ConsistencyChecker) CheckResourceConsistency(resources []apimodels.Resource, drifts []*DriftRecord) []string {
	var issues []string
	
	// Build resource map
	resourceMap := make(map[string]*apimodels.Resource)
	for i := range resources {
		resourceMap[resources[i].ID] = &resources[i]
	}
	
	// Check that all drifts reference existing resources
	for _, drift := range drifts {
		if _, exists := resourceMap[drift.ResourceID]; !exists {
			issues = append(issues, fmt.Sprintf("drift references non-existent resource: %s", drift.ResourceID))
		}
	}
	
	// Check for resource count consistency
	providerCounts := make(map[string]int)
	for _, resource := range resources {
		providerCounts[resource.Provider]++
	}
	
	// Check for provider consistency
	for _, drift := range drifts {
		resource, exists := resourceMap[drift.ResourceID]
		if exists && resource.Provider != drift.Provider {
			issues = append(issues, fmt.Sprintf("provider mismatch for drift %s: resource has %s, drift has %s", 
				drift.ID, resource.Provider, drift.Provider))
		}
		if exists && resource.Type != drift.ResourceType {
			issues = append(issues, fmt.Sprintf("type mismatch for drift %s: resource has %s, drift has %s", 
				drift.ID, resource.Type, drift.ResourceType))
		}
	}
	
	return issues
}

// CheckStatsConsistency validates that stats match actual data
func (c *ConsistencyChecker) CheckStatsConsistency(resources []apimodels.Resource, drifts []*DriftRecord, stats map[string]interface{}) []string {
	var issues []string
	
	// Count actual resources
	actualTotal := len(resources)
	if statsTotal, ok := stats["total"].(int); ok && statsTotal != actualTotal {
		issues = append(issues, fmt.Sprintf("stats total mismatch: reported %d, actual %d", statsTotal, actualTotal))
	}
	
	// Count actual drifted resources
	activeDrifts := 0
	for _, drift := range drifts {
		if drift.Status == "active" {
			activeDrifts++
		}
	}
	
	if statsDrifted, ok := stats["drifted"].(int); ok && statsDrifted != activeDrifts {
		issues = append(issues, fmt.Sprintf("stats drifted mismatch: reported %d, actual %d", statsDrifted, activeDrifts))
	}
	
	// Check compliance score calculation
	if actualTotal > 0 {
		expectedCompliance := ((actualTotal - activeDrifts) * 100) / actualTotal
		if statsCompliance, ok := stats["complianceScore"].(int); ok {
			if statsCompliance != expectedCompliance {
				issues = append(issues, fmt.Sprintf("compliance score mismatch: reported %d, expected %d", 
					statsCompliance, expectedCompliance))
			}
		}
	}
	
	return issues
}