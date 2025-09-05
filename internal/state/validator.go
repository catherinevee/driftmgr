package state

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Validator validates Terraform state files
type Validator struct {
	rules      []ValidationRule
	strictMode bool
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Name        string
	Description string
	Validate    func(*TerraformState) error
	Severity    Severity
}

// Severity defines the severity of a validation issue
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
	SeverityCritical
)

// ValidationResult contains the results of validation
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []ValidationError
}

// ValidationError represents a validation error
type ValidationError struct {
	Rule     string   `json:"rule"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Resource string   `json:"resource,omitempty"`
	Field    string   `json:"field,omitempty"`
}

// NewValidator creates a new state validator
func NewValidator() *Validator {
	v := &Validator{
		rules:      make([]ValidationRule, 0),
		strictMode: false,
	}

	// Add default validation rules
	v.addDefaultRules()

	return v
}

// Validate validates a Terraform state
func (v *Validator) Validate(state *TerraformState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
	}

	// Run all validation rules
	for _, rule := range v.rules {
		if err := rule.Validate(state); err != nil {
			valErr := ValidationError{
				Rule:     rule.Name,
				Message:  err.Error(),
				Severity: rule.Severity,
			}

			switch rule.Severity {
			case SeverityWarning:
				result.Warnings = append(result.Warnings, valErr)
			case SeverityError, SeverityCritical:
				result.Errors = append(result.Errors, valErr)
				result.Valid = false
			}
		}
	}

	// In strict mode, warnings are treated as errors
	if v.strictMode && len(result.Warnings) > 0 {
		result.Valid = false
	}

	if !result.Valid {
		return v.formatValidationError(result)
	}

	return nil
}

// addDefaultRules adds the default validation rules
func (v *Validator) addDefaultRules() {
	// Version validation
	v.AddRule(ValidationRule{
		Name:        "version",
		Description: "Validates state version",
		Severity:    SeverityCritical,
		Validate: func(state *TerraformState) error {
			if state.Version < 3 || state.Version > 4 {
				return fmt.Errorf("unsupported state version: %d", state.Version)
			}
			return nil
		},
	})

	// Lineage validation
	v.AddRule(ValidationRule{
		Name:        "lineage",
		Description: "Validates state lineage",
		Severity:    SeverityError,
		Validate: func(state *TerraformState) error {
			if state.Lineage == "" {
				return fmt.Errorf("state lineage is empty")
			}
			if len(state.Lineage) < 8 {
				return fmt.Errorf("state lineage is too short: %s", state.Lineage)
			}
			return nil
		},
	})

	// Serial validation
	v.AddRule(ValidationRule{
		Name:        "serial",
		Description: "Validates state serial",
		Severity:    SeverityWarning,
		Validate: func(state *TerraformState) error {
			if state.Serial < 0 {
				return fmt.Errorf("state serial is negative: %d", state.Serial)
			}
			return nil
		},
	})

	// Resource validation
	v.AddRule(ValidationRule{
		Name:        "resources",
		Description: "Validates resources",
		Severity:    SeverityError,
		Validate: func(state *TerraformState) error {
			addresses := make(map[string]bool)

			for _, resource := range state.Resources {
				// Validate resource structure
				if err := v.validateResource(resource); err != nil {
					return err
				}

				// Check for duplicate addresses
				for i := range resource.Instances {
					address := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
					if len(resource.Instances) > 1 {
						address = fmt.Sprintf("%s[%d]", address, i)
					}

					if addresses[address] {
						return fmt.Errorf("duplicate resource address: %s", address)
					}
					addresses[address] = true
				}
			}

			return nil
		},
	})

	// Provider validation
	v.AddRule(ValidationRule{
		Name:        "providers",
		Description: "Validates provider configurations",
		Severity:    SeverityWarning,
		Validate: func(state *TerraformState) error {
			for _, resource := range state.Resources {
				if resource.Provider == "" {
					return fmt.Errorf("resource %s.%s has no provider", resource.Type, resource.Name)
				}
			}
			return nil
		},
	})

	// Output validation
	v.AddRule(ValidationRule{
		Name:        "outputs",
		Description: "Validates output values",
		Severity:    SeverityWarning,
		Validate: func(state *TerraformState) error {
			for name, output := range state.Outputs {
				if output.Value == nil {
					return fmt.Errorf("output %s has nil value", name)
				}
			}
			return nil
		},
	})

	// Dependency validation
	v.AddRule(ValidationRule{
		Name:        "dependencies",
		Description: "Validates resource dependencies",
		Severity:    SeverityWarning,
		Validate: func(state *TerraformState) error {
			// Build resource map
			resourceMap := make(map[string]bool)
			for _, resource := range state.Resources {
				key := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
				resourceMap[key] = true
			}

			// Check dependencies
			for _, resource := range state.Resources {
				for _, dep := range resource.DependsOn {
					if !resourceMap[dep] {
						return fmt.Errorf("resource %s.%s depends on non-existent resource: %s",
							resource.Type, resource.Name, dep)
					}
				}
			}

			return nil
		},
	})
}

// validateResource validates a single resource
func (v *Validator) validateResource(resource Resource) error {
	// Validate resource type
	if resource.Type == "" {
		return fmt.Errorf("resource has empty type")
	}

	// Validate resource name
	if resource.Name == "" {
		return fmt.Errorf("resource %s has empty name", resource.Type)
	}

	// Validate resource name format
	if !isValidResourceName(resource.Name) {
		return fmt.Errorf("resource %s has invalid name: %s", resource.Type, resource.Name)
	}

	// Validate mode
	if resource.Mode != "managed" && resource.Mode != "data" {
		return fmt.Errorf("resource %s.%s has invalid mode: %s", resource.Type, resource.Name, resource.Mode)
	}

	// Validate instances
	if len(resource.Instances) == 0 {
		return fmt.Errorf("resource %s.%s has no instances", resource.Type, resource.Name)
	}

	for i, instance := range resource.Instances {
		if err := v.validateInstance(resource, i, instance); err != nil {
			return err
		}
	}

	return nil
}

// validateInstance validates a resource instance
func (v *Validator) validateInstance(resource Resource, index int, instance Instance) error {
	// Validate schema version
	if instance.SchemaVersion < 0 {
		return fmt.Errorf("resource %s.%s[%d] has negative schema version: %d",
			resource.Type, resource.Name, index, instance.SchemaVersion)
	}

	// Validate attributes
	if instance.Attributes == nil && instance.AttributesFlat == nil {
		return fmt.Errorf("resource %s.%s[%d] has no attributes",
			resource.Type, resource.Name, index)
	}

	// Validate status if present
	if instance.Status != "" {
		validStatuses := []string{"", "tainted", "deposed"}
		valid := false
		for _, s := range validStatuses {
			if instance.Status == s {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("resource %s.%s[%d] has invalid status: %s",
				resource.Type, resource.Name, index, instance.Status)
		}
	}

	return nil
}

// isValidResourceName checks if a resource name is valid
func isValidResourceName(name string) bool {
	// Resource names must start with a letter or underscore
	// and can contain letters, digits, underscores, and hyphens
	match, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_-]*$`, name)
	return match
}

// AddRule adds a custom validation rule
func (v *Validator) AddRule(rule ValidationRule) {
	v.rules = append(v.rules, rule)
}

// RemoveRule removes a validation rule by name
func (v *Validator) RemoveRule(name string) {
	newRules := make([]ValidationRule, 0, len(v.rules)-1)
	for _, rule := range v.rules {
		if rule.Name != name {
			newRules = append(newRules, rule)
		}
	}
	v.rules = newRules
}

// SetStrictMode enables or disables strict validation mode
func (v *Validator) SetStrictMode(strict bool) {
	v.strictMode = strict
}

// formatValidationError formats validation errors into a single error message
func (v *Validator) formatValidationError(result *ValidationResult) error {
	var messages []string

	for _, err := range result.Errors {
		messages = append(messages, fmt.Sprintf("[%s] %s", err.Rule, err.Message))
	}

	if v.strictMode {
		for _, warn := range result.Warnings {
			messages = append(messages, fmt.Sprintf("[%s] %s (warning)", warn.Rule, warn.Message))
		}
	}

	return fmt.Errorf("state validation failed:\n%s", strings.Join(messages, "\n"))
}

// ValidateResourceAddress validates a resource address format
func (v *Validator) ValidateResourceAddress(address string) error {
	// Basic format: type.name or type.name[index]
	pattern := `^[a-zA-Z][a-zA-Z0-9_-]*\.[a-zA-Z_][a-zA-Z0-9_-]*(\[\d+\])?$`
	match, _ := regexp.MatchString(pattern, address)

	if !match {
		return fmt.Errorf("invalid resource address format: %s", address)
	}

	return nil
}

// ValidateJSON validates that state data is valid JSON
func (v *Validator) ValidateJSON(data []byte) error {
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// GetRules returns all validation rules
func (v *Validator) GetRules() []ValidationRule {
	return v.rules
}

// ClearRules removes all validation rules
func (v *Validator) ClearRules() {
	v.rules = make([]ValidationRule, 0)
}
