package drift

import (
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// AttributeDriftDetector provides enhanced drift detection with configurable sensitivity
type AttributeDriftDetector struct {
	SensitiveFields   map[string]bool
	IgnoreFields      map[string]bool
	Thresholds        DriftThresholds
	CustomComparators map[string]AttributeComparator
	SeverityRules     []SeverityRule
}

// DriftThresholds defines sensitivity thresholds for drift detection
type DriftThresholds struct {
	CriticalPercentage float64
	HighPercentage     float64
	MediumPercentage   float64
	LowPercentage      float64
}

// AttributeComparator defines custom comparison logic for complex attributes
type AttributeComparator func(oldValue, newValue interface{}) (bool, []string, error)

// SeverityRule defines rules for determining drift severity
type SeverityRule struct {
	ResourceType  string
	AttributePath string
	Condition     string
	Severity      string
	Description   string
}

// NewAttributeDriftDetector creates a new enhanced drift detector
func NewAttributeDriftDetector() *AttributeDriftDetector {
	return &AttributeDriftDetector{
		SensitiveFields: make(map[string]bool),
		IgnoreFields:    make(map[string]bool),
		Thresholds: DriftThresholds{
			CriticalPercentage: 0.8,
			HighPercentage:     0.6,
			MediumPercentage:   0.4,
			LowPercentage:      0.2,
		},
		CustomComparators: make(map[string]AttributeComparator),
		SeverityRules:     []SeverityRule{},
	}
}

// AddSensitiveField marks a field as sensitive for drift detection
func (d *AttributeDriftDetector) AddSensitiveField(fieldPath string) {
	d.SensitiveFields[fieldPath] = true
}

// AddIgnoreField marks a field to be ignored during drift detection
func (d *AttributeDriftDetector) AddIgnoreField(fieldPath string) {
	d.IgnoreFields[fieldPath] = true
}

// AddCustomComparator adds custom comparison logic for specific attributes
func (d *AttributeDriftDetector) AddCustomComparator(attributePath string, comparator AttributeComparator) {
	d.CustomComparators[attributePath] = comparator
}

// AddSeverityRule adds a rule for determining drift severity
func (d *AttributeDriftDetector) AddSeverityRule(rule SeverityRule) {
	d.SeverityRules = append(d.SeverityRules, rule)
}

// DetectDrift performs enhanced drift detection between state and live resources
func (d *AttributeDriftDetector) DetectDrift(stateResources, liveResources []models.Resource) models.AnalysisResult {
	var driftResults []models.DriftResult
	var summary models.AnalysisSummary

	// Create maps for efficient lookup
	stateMap := make(map[string]models.Resource)
	liveMap := make(map[string]models.Resource)

	for _, resource := range stateResources {
		stateMap[resource.ID] = resource
	}

	for _, resource := range liveResources {
		liveMap[resource.ID] = resource
	}

	// Detect missing resources (in state but not in live)
	for id, stateResource := range stateMap {
		if _, exists := liveMap[id]; !exists {
			driftResult := d.createDriftResult(stateResource, "missing", "high",
				"Resource exists in Terraform state but not in live infrastructure")
			driftResults = append(driftResults, driftResult)
		}
	}

	// Detect extra resources (in live but not in state)
	for id, liveResource := range liveMap {
		if _, exists := stateMap[id]; !exists {
			driftResult := d.createDriftResult(liveResource, "extra", "medium",
				"Resource exists in live infrastructure but not in Terraform state")
			driftResults = append(driftResults, driftResult)
		}
	}

	// Detect modified resources (attribute-level drift)
	for id, stateResource := range stateMap {
		if liveResource, exists := liveMap[id]; exists {
			attributeDrifts := d.detectAttributeDrift(stateResource, liveResource)
			if len(attributeDrifts) > 0 {
				driftResult := d.createAttributeDriftResult(stateResource, attributeDrifts)
				driftResults = append(driftResults, driftResult)
			}
		}
	}

	// Calculate summary
	summary = d.calculateSummary(driftResults, len(stateResources), len(liveResources))

	return models.AnalysisResult{
		DriftResults: driftResults,
		Summary:      summary,
		Timestamp:    time.Now(),
	}
}

// detectAttributeDrift performs deep attribute comparison
func (d *AttributeDriftDetector) detectAttributeDrift(stateResource, liveResource models.Resource) []models.DriftChange {
	var changes []models.DriftChange

	// Compare basic attributes
	changes = append(changes, d.compareBasicAttributes(stateResource, liveResource)...)

	// Compare tags if they exist
	if len(stateResource.Tags) > 0 || len(liveResource.Tags) > 0 {
		tagChanges := d.compareTags(stateResource.Tags, liveResource.Tags)
		changes = append(changes, tagChanges...)
	}

	// Apply custom comparators
	for attributePath, comparator := range d.CustomComparators {
		if change := d.applyCustomComparator(attributePath, comparator, stateResource, liveResource); change != nil {
			changes = append(changes, *change)
		}
	}

	return changes
}

// compareBasicAttributes compares basic resource attributes
func (d *AttributeDriftDetector) compareBasicAttributes(stateResource, liveResource models.Resource) []models.DriftChange {
	var changes []models.DriftChange

	// Compare name
	if stateResource.Name != liveResource.Name {
		changes = append(changes, models.DriftChange{
			Field:      "name",
			OldValue:   stateResource.Name,
			NewValue:   liveResource.Name,
			ChangeType: "modified",
		})
	}

	// Compare region
	if stateResource.Region != liveResource.Region {
		changes = append(changes, models.DriftChange{
			Field:      "region",
			OldValue:   stateResource.Region,
			NewValue:   liveResource.Region,
			ChangeType: "modified",
		})
	}

	// Compare state
	if stateResource.State != liveResource.State {
		changes = append(changes, models.DriftChange{
			Field:      "state",
			OldValue:   stateResource.State,
			NewValue:   liveResource.State,
			ChangeType: "modified",
		})
	}

	return changes
}

// compareTags compares resource tags with configurable sensitivity
func (d *AttributeDriftDetector) compareTags(stateTags, liveTags map[string]string) []models.DriftChange {
	var changes []models.DriftChange

	// Check for missing tags in live resource
	for key, value := range stateTags {
		if d.shouldIgnoreField("tags." + key) {
			continue
		}

		if liveValue, exists := liveTags[key]; !exists {
			changes = append(changes, models.DriftChange{
				Field:      fmt.Sprintf("tags.%s", key),
				OldValue:   value,
				NewValue:   nil,
				ChangeType: "missing",
			})
		} else if value != liveValue {
			// Only report tag changes for sensitive fields
			if d.isSensitiveField("tags." + key) {
				changes = append(changes, models.DriftChange{
					Field:      fmt.Sprintf("tags.%s", key),
					OldValue:   value,
					NewValue:   liveValue,
					ChangeType: "modified",
				})
			}
		}
	}

	// Check for extra tags in live resource
	for key, value := range liveTags {
		if d.shouldIgnoreField("tags." + key) {
			continue
		}

		if _, exists := stateTags[key]; !exists {
			changes = append(changes, models.DriftChange{
				Field:      fmt.Sprintf("tags.%s", key),
				OldValue:   nil,
				NewValue:   value,
				ChangeType: "extra",
			})
		}
	}

	return changes
}

// applyCustomComparator applies custom comparison logic
func (d *AttributeDriftDetector) applyCustomComparator(attributePath string, comparator AttributeComparator, stateResource, liveResource models.Resource) *models.DriftChange {
	// Extract values from resources based on attribute path
	stateValue := d.extractValue(stateResource, attributePath)
	liveValue := d.extractValue(liveResource, attributePath)

	// Apply custom comparison
	hasDrift, _, err := comparator(stateValue, liveValue)
	if err != nil {
		// Log error but continue
		return nil
	}

	if hasDrift {
		return &models.DriftChange{
			Field:      attributePath,
			OldValue:   stateValue,
			NewValue:   liveValue,
			ChangeType: "custom",
		}
	}

	return nil
}

// extractValue extracts a value from a resource based on attribute path
func (d *AttributeDriftDetector) extractValue(resource models.Resource, attributePath string) interface{} {
	// Simple implementation - can be enhanced with reflection for nested paths
	switch attributePath {
	case "id":
		return resource.ID
	case "name":
		return resource.Name
	case "type":
		return resource.Type
	case "provider":
		return resource.Provider
	case "region":
		return resource.Region
	case "state":
		return resource.State
	default:
		// Handle nested paths like "tags.environment"
		if strings.HasPrefix(attributePath, "tags.") {
			tagKey := strings.TrimPrefix(attributePath, "tags.")
			if value, exists := resource.Tags[tagKey]; exists {
				return value
			}
		}
		return nil
	}
}

// shouldIgnoreField checks if a field should be ignored
func (d *AttributeDriftDetector) shouldIgnoreField(fieldPath string) bool {
	return d.IgnoreFields[fieldPath]
}

// isSensitiveField checks if a field is marked as sensitive
func (d *AttributeDriftDetector) isSensitiveField(fieldPath string) bool {
	return d.SensitiveFields[fieldPath]
}

// createDriftResult creates a basic drift result with risk reasoning
func (d *AttributeDriftDetector) createDriftResult(resource models.Resource, driftType, severity, description string) models.DriftResult {
	riskReasoning := d.generateRiskReasoning(resource, driftType, severity, nil)

	return models.DriftResult{
		ResourceID:    resource.ID,
		ResourceName:  resource.Name,
		ResourceType:  resource.Type,
		Provider:      resource.Provider,
		Region:        resource.Region,
		DriftType:     driftType,
		Severity:      severity,
		Description:   description,
		RiskReasoning: riskReasoning,
		DetectedAt:    time.Now(),
	}
}

// createAttributeDriftResult creates a drift result for attribute-level changes
func (d *AttributeDriftDetector) createAttributeDriftResult(resource models.Resource, changes []models.DriftChange) models.DriftResult {
	severity := d.determineSeverity(resource, changes)
	description := d.generateDescription(changes)
	riskReasoning := d.generateRiskReasoning(resource, "modified", severity, changes)

	return models.DriftResult{
		ResourceID:    resource.ID,
		ResourceName:  resource.Name,
		ResourceType:  resource.Type,
		Provider:      resource.Provider,
		Region:        resource.Region,
		DriftType:     "modified",
		Severity:      severity,
		Description:   description,
		RiskReasoning: riskReasoning,
		Changes:       changes,
		DetectedAt:    time.Now(),
	}
}

// determineSeverity determines the severity based on changes and rules
func (d *AttributeDriftDetector) determineSeverity(resource models.Resource, changes []models.DriftChange) string {
	// Apply severity rules
	for _, rule := range d.SeverityRules {
		if rule.ResourceType == resource.Type {
			for _, change := range changes {
				if change.Field == rule.AttributePath {
					return rule.Severity
				}
			}
		}
	}

	// Default severity based on number of changes
	switch len(changes) {
	case 0:
		return "low"
	case 1:
		return "medium"
	case 2, 3:
		return "high"
	default:
		return "critical"
	}
}

// generateDescription generates a human-readable description of changes
func (d *AttributeDriftDetector) generateDescription(changes []models.DriftChange) string {
	if len(changes) == 0 {
		return "No changes detected"
	}

	var descriptions []string
	for _, change := range changes {
		switch change.ChangeType {
		case "modified":
			descriptions = append(descriptions, fmt.Sprintf("Field '%s' changed from '%v' to '%v'", change.Field, change.OldValue, change.NewValue))
		case "missing":
			descriptions = append(descriptions, fmt.Sprintf("Field '%s' is missing (expected: '%v')", change.Field, change.OldValue))
		case "extra":
			descriptions = append(descriptions, fmt.Sprintf("Field '%s' has unexpected value '%v'", change.Field, change.NewValue))
		}
	}

	return strings.Join(descriptions, "; ")
}

// calculateSummary calculates comprehensive drift summary
func (d *AttributeDriftDetector) calculateSummary(driftResults []models.DriftResult, stateCount, liveCount int) models.AnalysisSummary {
	summary := models.AnalysisSummary{
		TotalDrifts:         len(driftResults),
		DriftsFound:         len(driftResults),
		TotalStateResources: stateCount,
		TotalLiveResources:  liveCount,
		BySeverity:          make(map[string]int),
		ByProvider:          make(map[string]int),
		ByResourceType:      make(map[string]int),
	}

	// Count by severity and type
	for _, drift := range driftResults {
		summary.BySeverity[drift.Severity]++
		summary.ByProvider[drift.Provider]++
		summary.ByResourceType[drift.ResourceType]++

		switch drift.Severity {
		case "critical":
			summary.CriticalDrifts++
		case "high":
			summary.HighDrifts++
		case "medium":
			summary.MediumDrifts++
		case "low":
			summary.LowDrifts++
		}

		switch drift.DriftType {
		case "missing":
			summary.Missing++
		case "extra":
			summary.Extra++
		case "modified":
			summary.Modified++
		}
	}

	// Calculate percentages
	if summary.TotalStateResources > 0 {
		summary.CoveragePercentage = float64(summary.TotalStateResources-summary.Missing) / float64(summary.TotalStateResources) * 100
	}
	if summary.TotalLiveResources > 0 {
		summary.DriftPercentage = float64(summary.DriftsFound) / float64(summary.TotalLiveResources) * 100
	}

	// Calculate perspective percentage
	totalResources := summary.TotalStateResources + summary.Extra
	if totalResources > 0 {
		summary.PerspectivePercentage = float64(totalResources-summary.DriftsFound) / float64(totalResources) * 100
	}

	return summary
}

// Predefined comparators for common use cases

// SecurityGroupComparator compares security group rules
func SecurityGroupComparator(oldValue, newValue interface{}) (bool, []string, error) {
	// Implementation for security group rule comparison
	// This would compare ingress/egress rules, ports, protocols, etc.
	return false, nil, nil
}

// IAMPolicyComparator compares IAM policies
func IAMPolicyComparator(oldValue, newValue interface{}) (bool, []string, error) {
	// Implementation for IAM policy comparison
	// This would compare permissions, principals, conditions, etc.
	return false, nil, nil
}

// TagComparator compares resource tags with business logic
func TagComparator(oldValue, newValue interface{}) (bool, []string, error) {
	// Implementation for tag comparison with business rules
	// This would check required tags, tag formats, etc.
	return false, nil, nil
}

// generateRiskReasoning generates detailed explanation for risk level assignment
func (d *AttributeDriftDetector) generateRiskReasoning(resource models.Resource, driftType, severity string, changes []models.DriftChange) string {
	var reasons []string

	// Base reasoning for drift type
	switch driftType {
	case "missing":
		reasons = append(reasons, "Resource exists in Terraform state but not in live infrastructure")
		reasons = append(reasons, "This indicates potential resource deletion or misconfiguration")
	case "extra":
		reasons = append(reasons, "Resource exists in live infrastructure but not in Terraform state")
		reasons = append(reasons, "This indicates unmanaged resources that should be under Terraform control")
	case "modified":
		reasons = append(reasons, "Resource attributes have changed from expected state")
	}

	// Severity-specific reasoning
	switch severity {
	case "critical":
		reasons = append(reasons, "CRITICAL: This change affects production environment, security, or compliance")
		if resource.Type == "aws_instance" {
			reasons = append(reasons, "Production instances require immediate attention")
		}
		if resource.Type == "aws_security_group" {
			reasons = append(reasons, "Security group changes can impact network security")
		}
	case "high":
		reasons = append(reasons, "HIGH: This change affects important configuration or could impact operations")
		if driftType == "missing" {
			reasons = append(reasons, "Missing resources can cause service outages")
		}
	case "medium":
		reasons = append(reasons, "MEDIUM: This change should be reviewed but is not immediately critical")
		if driftType == "extra" {
			reasons = append(reasons, "Unmanaged resources may incur unexpected costs")
		}
	case "low":
		reasons = append(reasons, "LOW: This change is minor and may be auto-generated or temporary")
	}

	// Resource type specific reasoning
	switch resource.Type {
	case "aws_instance":
		reasons = append(reasons, "EC2 instances are compute resources that can affect application availability")
	case "aws_security_group":
		reasons = append(reasons, "Security groups control network access and are critical for security")
	case "aws_s3_bucket":
		reasons = append(reasons, "S3 buckets store data and changes can affect data access or compliance")
	case "aws_rds_instance":
		reasons = append(reasons, "RDS instances are databases that can affect data availability")
	}

	// Change-specific reasoning for modified resources
	if len(changes) > 0 {
		for _, change := range changes {
			if d.isSensitiveField(change.Field) {
				reasons = append(reasons, fmt.Sprintf("Field '%s' is marked as sensitive and requires attention", change.Field))
			}

			// Specific field reasoning
			switch change.Field {
			case "tags.environment":
				reasons = append(reasons, "Environment tag changes can affect resource classification and cost allocation")
			case "tags.owner":
				reasons = append(reasons, "Owner tag changes can affect responsibility assignment and access control")
			case "tags.cost-center":
				reasons = append(reasons, "Cost center tag changes can affect billing and budget tracking")
			case "security_groups":
				reasons = append(reasons, "Security group changes can affect network access and security posture")
			case "iam_policies":
				reasons = append(reasons, "IAM policy changes can affect access permissions and security")
			}
		}
	}

	// Severity rule matching
	for _, rule := range d.SeverityRules {
		if rule.ResourceType == resource.Type {
			if len(changes) > 0 {
				for _, change := range changes {
					if change.Field == rule.AttributePath {
						reasons = append(reasons, fmt.Sprintf("Matches severity rule: %s", rule.Description))
						break
					}
				}
			}
		}
	}

	return strings.Join(reasons, "; ")
}
