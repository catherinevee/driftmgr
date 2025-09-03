package health

import (
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/analysis/graph"
	"github.com/catherinevee/driftmgr/internal/state/parser"
)

// HealthAnalyzer performs health analysis on resources
type HealthAnalyzer struct {
	graph          *graph.DependencyGraph
	providers      map[string]ProviderHealthChecker
	customChecks   []HealthCheck
	severityLevels map[string]Severity
}

// ProviderHealthChecker defines provider-specific health checks
type ProviderHealthChecker interface {
	CheckResource(resource *parser.Resource, instance *parser.Instance) *HealthReport
	GetRequiredAttributes(resourceType string) []string
	GetDeprecatedAttributes(resourceType string) []string
	GetSecurityRules(resourceType string) []SecurityRule
}

// HealthReport contains the health analysis results for a resource
type HealthReport struct {
	Resource       string          `json:"resource"`
	Status         HealthStatus    `json:"status"`
	Score          int             `json:"score"` // 0-100
	Issues         []HealthIssue   `json:"issues"`
	Suggestions    []string        `json:"suggestions"`
	Impact         ImpactLevel     `json:"impact"`
	LastChecked    time.Time       `json:"last_checked"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// HealthIssue represents a specific health problem
type HealthIssue struct {
	Type        IssueType    `json:"type"`
	Severity    Severity     `json:"severity"`
	Message     string       `json:"message"`
	Field       string       `json:"field,omitempty"`
	CurrentValue interface{} `json:"current_value,omitempty"`
	ExpectedValue interface{} `json:"expected_value,omitempty"`
	Documentation string      `json:"documentation,omitempty"`
}

// HealthStatus represents the overall health status
type HealthStatus int

const (
	HealthStatusHealthy HealthStatus = iota
	HealthStatusWarning
	HealthStatusCritical
	HealthStatusUnknown
)

// IssueType categorizes health issues
type IssueType string

const (
	IssueTypeMissingAttribute   IssueType = "missing_attribute"
	IssueTypeDeprecated         IssueType = "deprecated"
	IssueTypeSecurity           IssueType = "security"
	IssueTypePerformance        IssueType = "performance"
	IssueTypeConfiguration      IssueType = "configuration"
	IssueTypeCost               IssueType = "cost"
	IssueTypeCompliance         IssueType = "compliance"
	IssueTypeDependency         IssueType = "dependency"
)

// Severity levels for issues
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// ImpactLevel represents the impact of an unhealthy resource
type ImpactLevel int

const (
	ImpactLevelLow ImpactLevel = iota
	ImpactLevelMedium
	ImpactLevelHigh
	ImpactLevelCritical
)

// SecurityRule defines a security check
type SecurityRule struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Check       func(attributes map[string]interface{}) bool
	Severity    Severity    `json:"severity"`
	Remediation string      `json:"remediation"`
}

// HealthCheck defines a custom health check
type HealthCheck struct {
	Name        string
	Description string
	Check       func(resource *parser.Resource, instance *parser.Instance) *HealthIssue
	Applies     func(resourceType string) bool
}

// NewHealthAnalyzer creates a new health analyzer
func NewHealthAnalyzer(depGraph *graph.DependencyGraph) *HealthAnalyzer {
	analyzer := &HealthAnalyzer{
		graph:          depGraph,
		providers:      make(map[string]ProviderHealthChecker),
		customChecks:   make([]HealthCheck, 0),
		severityLevels: make(map[string]Severity),
	}

	// Register default providers
	analyzer.registerDefaultProviders()
	
	// Add default custom checks
	analyzer.addDefaultChecks()

	return analyzer
}

// AnalyzeState performs health analysis on entire state
func (ha *HealthAnalyzer) AnalyzeState(state *parser.TerraformState) (*StateHealthReport, error) {
	report := &StateHealthReport{
		Timestamp:       time.Now(),
		TotalResources:  0,
		HealthyResources: 0,
		Issues:          make([]HealthReport, 0),
		OverallScore:    100,
		ResourceReports: make(map[string]*HealthReport),
	}

	for _, resource := range state.Resources {
		for i, instance := range resource.Instances {
			address := ha.formatResourceAddress(resource, i)
			resourceReport := ha.AnalyzeResource(&resource, &instance)
			
			report.TotalResources++
			report.ResourceReports[address] = resourceReport
			
			if resourceReport.Status == HealthStatusHealthy {
				report.HealthyResources++
			} else {
				report.Issues = append(report.Issues, *resourceReport)
			}
			
			// Update overall score
			report.OverallScore = (report.OverallScore + resourceReport.Score) / 2
		}
	}

	// Calculate summary statistics
	report.CriticalIssues = ha.countIssuesBySeverity(report.Issues, SeverityCritical)
	report.HighIssues = ha.countIssuesBySeverity(report.Issues, SeverityHigh)
	report.MediumIssues = ha.countIssuesBySeverity(report.Issues, SeverityMedium)
	report.LowIssues = ha.countIssuesBySeverity(report.Issues, SeverityLow)

	return report, nil
}

// AnalyzeResource performs health analysis on a single resource
func (ha *HealthAnalyzer) AnalyzeResource(resource *parser.Resource, instance *parser.Instance) *HealthReport {
	report := &HealthReport{
		Resource:    ha.formatResourceAddress(*resource, 0),
		Status:      HealthStatusHealthy,
		Score:       100,
		Issues:      make([]HealthIssue, 0),
		Suggestions: make([]string, 0),
		LastChecked: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Get provider-specific checker
	providerName := ha.extractProviderName(resource.Provider)
	if checker, exists := ha.providers[providerName]; exists {
		providerReport := checker.CheckResource(resource, instance)
		if providerReport != nil {
			report.Issues = append(report.Issues, providerReport.Issues...)
			report.Suggestions = append(report.Suggestions, providerReport.Suggestions...)
		}
		
		// Check required attributes
		ha.checkRequiredAttributes(resource, instance, checker, report)
		
		// Check deprecated attributes
		ha.checkDeprecatedAttributes(resource, instance, checker, report)
		
		// Check security rules
		ha.checkSecurityRules(resource, instance, checker, report)
	}

	// Run custom checks
	for _, check := range ha.customChecks {
		if check.Applies(resource.Type) {
			if issue := check.Check(resource, instance); issue != nil {
				report.Issues = append(report.Issues, *issue)
			}
		}
	}

	// Calculate impact based on dependencies
	report.Impact = ha.calculateImpact(resource)

	// Update status and score based on issues
	ha.updateReportStatus(report)

	return report
}

// checkRequiredAttributes checks for missing required attributes
func (ha *HealthAnalyzer) checkRequiredAttributes(resource *parser.Resource, instance *parser.Instance, 
	checker ProviderHealthChecker, report *HealthReport) {
	
	required := checker.GetRequiredAttributes(resource.Type)
	for _, attr := range required {
		if instance.Attributes == nil || instance.Attributes[attr] == nil {
			report.Issues = append(report.Issues, HealthIssue{
				Type:          IssueTypeMissingAttribute,
				Severity:      SeverityHigh,
				Message:       fmt.Sprintf("Missing required attribute: %s", attr),
				Field:         attr,
				ExpectedValue: "non-null value",
			})
		}
	}
}

// checkDeprecatedAttributes checks for usage of deprecated attributes
func (ha *HealthAnalyzer) checkDeprecatedAttributes(resource *parser.Resource, instance *parser.Instance,
	checker ProviderHealthChecker, report *HealthReport) {
	
	deprecated := checker.GetDeprecatedAttributes(resource.Type)
	for _, attr := range deprecated {
		if instance.Attributes != nil && instance.Attributes[attr] != nil {
			report.Issues = append(report.Issues, HealthIssue{
				Type:         IssueTypeDeprecated,
				Severity:     SeverityMedium,
				Message:      fmt.Sprintf("Using deprecated attribute: %s", attr),
				Field:        attr,
				CurrentValue: instance.Attributes[attr],
				Documentation: fmt.Sprintf("https://registry.terraform.io/providers/%s/latest/docs/resources/%s",
					ha.extractProviderName(resource.Provider), 
					strings.TrimPrefix(resource.Type, ha.extractProviderName(resource.Provider)+"_")),
			})
			
			report.Suggestions = append(report.Suggestions, 
				fmt.Sprintf("Consider removing or replacing deprecated attribute '%s'", attr))
		}
	}
}

// checkSecurityRules checks security compliance
func (ha *HealthAnalyzer) checkSecurityRules(resource *parser.Resource, instance *parser.Instance,
	checker ProviderHealthChecker, report *HealthReport) {
	
	rules := checker.GetSecurityRules(resource.Type)
	for _, rule := range rules {
		if instance.Attributes != nil && !rule.Check(instance.Attributes) {
			report.Issues = append(report.Issues, HealthIssue{
				Type:          IssueTypeSecurity,
				Severity:      rule.Severity,
				Message:       rule.Description,
				Documentation: rule.Remediation,
			})
			
			if rule.Remediation != "" {
				report.Suggestions = append(report.Suggestions, rule.Remediation)
			}
		}
	}
}

// calculateImpact calculates the impact level based on dependencies
func (ha *HealthAnalyzer) calculateImpact(resource *parser.Resource) ImpactLevel {
	if ha.graph == nil {
		return ImpactLevelLow
	}

	address := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
	dependents := ha.graph.GetTransitiveDependents(address)
	
	switch {
	case len(dependents) == 0:
		return ImpactLevelLow
	case len(dependents) < 5:
		return ImpactLevelMedium
	case len(dependents) < 10:
		return ImpactLevelHigh
	default:
		return ImpactLevelCritical
	}
}

// updateReportStatus updates the report status and score based on issues
func (ha *HealthAnalyzer) updateReportStatus(report *HealthReport) {
	if len(report.Issues) == 0 {
		report.Status = HealthStatusHealthy
		report.Score = 100
		return
	}

	// Calculate score based on issues
	scoreDeduction := 0
	hasCritical := false
	hasHigh := false
	
	for _, issue := range report.Issues {
		switch issue.Severity {
		case SeverityCritical:
			scoreDeduction += 25
			hasCritical = true
		case SeverityHigh:
			scoreDeduction += 15
			hasHigh = true
		case SeverityMedium:
			scoreDeduction += 10
		case SeverityLow:
			scoreDeduction += 5
		}
	}

	report.Score = max(0, 100-scoreDeduction)
	
	// Set status
	if hasCritical {
		report.Status = HealthStatusCritical
	} else if hasHigh {
		report.Status = HealthStatusWarning
	} else {
		report.Status = HealthStatusWarning
	}
}

// addDefaultChecks adds default custom health checks
func (ha *HealthAnalyzer) addDefaultChecks() {
	// Check for hardcoded credentials
	ha.customChecks = append(ha.customChecks, HealthCheck{
		Name:        "hardcoded_credentials",
		Description: "Check for hardcoded credentials",
		Applies:     func(resourceType string) bool { return true },
		Check: func(resource *parser.Resource, instance *parser.Instance) *HealthIssue {
			if instance.Attributes == nil {
				return nil
			}
			
			sensitiveFields := []string{"password", "secret", "token", "api_key", "access_key"}
			for _, field := range sensitiveFields {
				if val, exists := instance.Attributes[field]; exists {
					if str, ok := val.(string); ok && str != "" && !strings.Contains(str, "${") {
						return &HealthIssue{
							Type:     IssueTypeSecurity,
							Severity: SeverityCritical,
							Message:  fmt.Sprintf("Hardcoded credential found in field '%s'", field),
							Field:    field,
						}
					}
				}
			}
			return nil
		},
	})

	// Check for missing tags
	ha.customChecks = append(ha.customChecks, HealthCheck{
		Name:        "missing_tags",
		Description: "Check for missing required tags",
		Applies:     func(resourceType string) bool { 
			return strings.Contains(resourceType, "instance") || strings.Contains(resourceType, "bucket")
		},
		Check: func(resource *parser.Resource, instance *parser.Instance) *HealthIssue {
			if instance.Attributes == nil {
				return nil
			}
			
			tags, exists := instance.Attributes["tags"]
			if !exists || tags == nil {
				return &HealthIssue{
					Type:     IssueTypeCompliance,
					Severity: SeverityMedium,
					Message:  "Resource is missing tags",
				}
			}
			
			// Check for required tags
			requiredTags := []string{"Environment", "Owner", "Project"}
			if tagMap, ok := tags.(map[string]interface{}); ok {
				for _, required := range requiredTags {
					if _, exists := tagMap[required]; !exists {
						return &HealthIssue{
							Type:     IssueTypeCompliance,
							Severity: SeverityLow,
							Message:  fmt.Sprintf("Missing required tag: %s", required),
							Field:    "tags",
						}
					}
				}
			}
			
			return nil
		},
	})
}

// registerDefaultProviders registers default provider health checkers
func (ha *HealthAnalyzer) registerDefaultProviders() {
	ha.providers["aws"] = NewAWSHealthChecker()
	ha.providers["azurerm"] = NewAzureHealthChecker()
	ha.providers["google"] = NewGCPHealthChecker()
}

// formatResourceAddress formats a resource address
func (ha *HealthAnalyzer) formatResourceAddress(resource parser.Resource, index int) string {
	if len(resource.Instances) == 1 {
		return fmt.Sprintf("%s.%s", resource.Type, resource.Name)
	}
	return fmt.Sprintf("%s.%s[%d]", resource.Type, resource.Name, index)
}

// extractProviderName extracts the provider name from a provider string
func (ha *HealthAnalyzer) extractProviderName(provider string) string {
	parts := strings.Split(provider, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return provider
}

// countIssuesBySeverity counts issues by severity level
func (ha *HealthAnalyzer) countIssuesBySeverity(reports []HealthReport, severity Severity) int {
	count := 0
	for _, report := range reports {
		for _, issue := range report.Issues {
			if issue.Severity == severity {
				count++
			}
		}
	}
	return count
}

// StateHealthReport contains the overall health report for a state
type StateHealthReport struct {
	Timestamp        time.Time                `json:"timestamp"`
	TotalResources   int                      `json:"total_resources"`
	HealthyResources int                      `json:"healthy_resources"`
	Issues           []HealthReport           `json:"issues"`
	OverallScore     int                      `json:"overall_score"`
	CriticalIssues   int                      `json:"critical_issues"`
	HighIssues       int                      `json:"high_issues"`
	MediumIssues     int                      `json:"medium_issues"`
	LowIssues        int                      `json:"low_issues"`
	ResourceReports  map[string]*HealthReport `json:"resource_reports"`
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}