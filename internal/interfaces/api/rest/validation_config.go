package api

import (
	"fmt"
	"strings"

	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
)

// ValidationLevel represents the strictness of validation
type ValidationLevel int

const (
	// ValidationLevelStrict enforces all validation rules
	ValidationLevelStrict ValidationLevel = iota
	// ValidationLevelNormal enforces important rules but is lenient on optional fields
	ValidationLevelNormal
	// ValidationLevelLenient only enforces critical rules
	ValidationLevelLenient
	// ValidationLevelNone disables validation
	ValidationLevelNone
)

// ValidationConfig configures validation behavior
type ValidationConfig struct {
	Level              ValidationLevel
	RequireRegion      bool
	RequireTimestamps  bool
	ValidateStatus     bool
	ValidateProvider   bool
	AllowUnknownFields bool
	LogWarnings        bool
}

// DefaultValidationConfig returns the default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		Level:              ValidationLevelNormal,
		RequireRegion:      false, // Don't require region by default
		RequireTimestamps:  false, // Timestamps are optional
		ValidateStatus:     false, // Accept any status value
		ValidateProvider:   true,  // Provider should be valid
		AllowUnknownFields: true,  // Allow additional fields
		LogWarnings:        true,  // Log validation warnings
	}
}

// StrictValidationConfig returns a strict validation configuration
func StrictValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		Level:              ValidationLevelStrict,
		RequireRegion:      true,
		RequireTimestamps:  true,
		ValidateStatus:     true,
		ValidateProvider:   true,
		AllowUnknownFields: false,
		LogWarnings:        true,
	}
}

// LenientValidationConfig returns a lenient validation configuration
func LenientValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		Level:              ValidationLevelLenient,
		RequireRegion:      false,
		RequireTimestamps:  false,
		ValidateStatus:     false,
		ValidateProvider:   false,
		AllowUnknownFields: true,
		LogWarnings:        false,
	}
}

// ConfigurableResourceValidator validates resources with configurable rules
type ConfigurableResourceValidator struct {
	config *ValidationConfig
	base   *ResourceValidator
}

// NewConfigurableResourceValidator creates a new configurable validator
func NewConfigurableResourceValidator(config *ValidationConfig) *ConfigurableResourceValidator {
	if config == nil {
		config = DefaultValidationConfig()
	}
	return &ConfigurableResourceValidator{
		config: config,
		base:   NewResourceValidator(),
	}
}

// ValidateResource validates a resource based on configuration
func (v *ConfigurableResourceValidator) ValidateResource(resource *apimodels.Resource) ([]string, []string) {
	var errors []string
	var warnings []string
	
	// Skip validation if disabled
	if v.config.Level == ValidationLevelNone {
		return errors, warnings
	}
	
	// Required fields (always validated except in None level)
	if resource.ID == "" {
		errors = append(errors, "resource ID is required")
	}
	
	if resource.Type == "" {
		errors = append(errors, "resource type is required")
	}
	
	if resource.Provider == "" {
		errors = append(errors, "resource provider is required")
	}
	
	// Provider validation (if enabled)
	if v.config.ValidateProvider && resource.Provider != "" {
		if err := v.validateProvider(resource); err != nil {
			if v.config.Level == ValidationLevelStrict {
				errors = append(errors, err.Error())
			} else {
				warnings = append(warnings, err.Error())
			}
		}
	}
	
	// Region validation (if required)
	if v.config.RequireRegion && resource.Region == "" {
		if v.isGlobalResource(resource.Type) {
			// Global resources don't need regions
		} else {
			if v.config.Level == ValidationLevelStrict {
				errors = append(errors, "region is required for cloud resources")
			} else {
				warnings = append(warnings, "region is recommended for cloud resources")
			}
		}
	}
	
	// Status validation (if enabled)
	if v.config.ValidateStatus && resource.Status != "" {
		if err := v.validateStatus(resource.Status); err != nil {
			if v.config.Level == ValidationLevelStrict {
				errors = append(errors, err.Error())
			} else {
				warnings = append(warnings, err.Error())
			}
		}
	}
	
	// Timestamp validation (if required)
	if v.config.RequireTimestamps {
		if resource.CreatedAt.IsZero() {
			warnings = append(warnings, "created_at timestamp is missing")
		}
		if resource.ModifiedAt.IsZero() {
			warnings = append(warnings, "modified_at timestamp is missing")
		}
	}
	
	return errors, warnings
}

// ValidateResources validates multiple resources
func (v *ConfigurableResourceValidator) ValidateResources(resources []apimodels.Resource) ([]apimodels.Resource, []string, []string) {
	var validResources []apimodels.Resource
	var allErrors []string
	var allWarnings []string
	
	seen := make(map[string]bool)
	
	for i, resource := range resources {
		errors, warnings := v.ValidateResource(&resource)
		
		// Check for duplicate IDs
		if seen[resource.ID] {
			errors = append(errors, fmt.Sprintf("duplicate resource ID: %s", resource.ID))
		} else {
			seen[resource.ID] = true
		}
		
		// Format errors with resource index
		for _, err := range errors {
			allErrors = append(allErrors, fmt.Sprintf("resource[%d]: %s", i, err))
		}
		
		// Format warnings with resource index
		for _, warn := range warnings {
			allWarnings = append(allWarnings, fmt.Sprintf("resource[%d]: %s", i, warn))
		}
		
		// Add resource if no critical errors
		if len(errors) == 0 || v.config.Level == ValidationLevelLenient {
			validResources = append(validResources, resource)
		}
	}
	
	return validResources, allErrors, allWarnings
}

// validateProvider checks if the provider is valid
func (v *ConfigurableResourceValidator) validateProvider(resource *apimodels.Resource) error {
	validProviders := []string{"aws", "azure", "gcp", "digitalocean"}
	provider := strings.ToLower(resource.Provider)
	
	for _, valid := range validProviders {
		if provider == valid {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported provider: %s", resource.Provider)
}

// validateStatus checks if the status is valid
func (v *ConfigurableResourceValidator) validateStatus(status string) error {
	// Accept all status values in non-strict mode
	if v.config.Level != ValidationLevelStrict {
		return nil
	}
	
	validStatuses := []string{
		"active", "running", "stopped", "deleted", "pending", "failed",
		"unknown", "missing", "succeeded", "success", "available",
		"creating", "updating", "deleting", "error", "degraded",
		"healthy", "unhealthy", "attached", "detached", "enabled", "disabled",
	}
	
	statusLower := strings.ToLower(status)
	for _, valid := range validStatuses {
		if statusLower == valid {
			return nil
		}
	}
	
	return fmt.Errorf("unrecognized status: %s", status)
}

// isGlobalResource checks if a resource type is global (doesn't require region)
func (v *ConfigurableResourceValidator) isGlobalResource(resourceType string) bool {
	globalPrefixes := []string{
		// AWS global resources
		"aws_iam_", "aws_route53_", "aws_cloudfront_", "aws_waf_",
		"aws_organizations_", "aws_cloudtrail_",
		// Azure global resources
		"azure_management_", "azure_policy_", "azure_blueprint_",
		"azurerm_management_", "azurerm_policy_",
		// GCP global resources
		"google_project_", "google_organization_", "google_billing_",
		"google_folder_", "google_logging_",
		// DigitalOcean global resources
		"digitalocean_project_", "digitalocean_firewall_",
		"digitalocean_certificate_", "digitalocean_domain_",
	}
	
	typeLower := strings.ToLower(resourceType)
	for _, prefix := range globalPrefixes {
		if strings.HasPrefix(typeLower, prefix) {
			return true
		}
	}
	
	return false
}

// ValidationLogger logs validation issues
type ValidationLogger struct {
	config *ValidationConfig
}

// NewValidationLogger creates a new validation logger
func NewValidationLogger(config *ValidationConfig) *ValidationLogger {
	return &ValidationLogger{config: config}
}

// LogValidation logs validation results
func (l *ValidationLogger) LogValidation(errors []string, warnings []string) {
	if !l.config.LogWarnings {
		return
	}
	
	// In production, these would go to a proper logging system
	for _, err := range errors {
		fmt.Printf("[Validation Error] %s\n", err)
	}
	
	for _, warn := range warnings {
		fmt.Printf("[Validation Warning] %s\n", warn)
	}
}

// ValidationMetrics tracks validation statistics
type ValidationMetrics struct {
	TotalValidations   int
	TotalErrors        int
	TotalWarnings      int
	ErrorsByField      map[string]int
	WarningsByField    map[string]int
	ValidationDuration int64 // microseconds
}

// NewValidationMetrics creates new validation metrics
func NewValidationMetrics() *ValidationMetrics {
	return &ValidationMetrics{
		ErrorsByField:   make(map[string]int),
		WarningsByField: make(map[string]int),
	}
}

// RecordValidation records validation results in metrics
func (m *ValidationMetrics) RecordValidation(errors []string, warnings []string, durationMicros int64) {
	m.TotalValidations++
	m.TotalErrors += len(errors)
	m.TotalWarnings += len(warnings)
	m.ValidationDuration += durationMicros
	
	// Track which fields have the most errors
	for _, err := range errors {
		if strings.Contains(err, "region") {
			m.ErrorsByField["region"]++
		} else if strings.Contains(err, "status") {
			m.ErrorsByField["status"]++
		} else if strings.Contains(err, "provider") {
			m.ErrorsByField["provider"]++
		} else if strings.Contains(err, "ID") {
			m.ErrorsByField["id"]++
		} else {
			m.ErrorsByField["other"]++
		}
	}
	
	// Track which fields have the most warnings
	for _, warn := range warnings {
		if strings.Contains(warn, "region") {
			m.WarningsByField["region"]++
		} else if strings.Contains(warn, "timestamp") {
			m.WarningsByField["timestamp"]++
		} else {
			m.WarningsByField["other"]++
		}
	}
}