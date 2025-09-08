package security

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// ComplianceReport represents a comprehensive compliance report
type ComplianceReport struct {
	ID              string                 `json:"id"`
	Standard        string                 `json:"standard"`
	GeneratedAt     time.Time              `json:"generated_at"`
	ValidUntil      time.Time              `json:"valid_until"`
	Policies        []*CompliancePolicy    `json:"policies"`
	Results         []*ComplianceResult    `json:"results"`
	Summary         map[string]interface{} `json:"summary"`
	Recommendations []Recommendation       `json:"recommendations"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Recommendation represents a compliance recommendation
type Recommendation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    string                 `json:"priority"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource,omitempty"`
	Policy      string                 `json:"policy,omitempty"`
	Rule        string                 `json:"rule,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceReportGenerator generates compliance reports
type ComplianceReportGenerator struct {
	complianceManager *ComplianceManager
}

// NewComplianceReportGenerator creates a new compliance report generator
func NewComplianceReportGenerator(complianceManager *ComplianceManager) *ComplianceReportGenerator {
	return &ComplianceReportGenerator{
		complianceManager: complianceManager,
	}
}

// GenerateReport generates a comprehensive compliance report
func (crg *ComplianceReportGenerator) GenerateReport(ctx context.Context, standard string) (*ComplianceReport, error) {
	// Get base report from compliance manager
	baseReport, err := crg.complianceManager.GetComplianceReport(ctx, standard)
	if err != nil {
		return nil, fmt.Errorf("failed to get base compliance report: %w", err)
	}

	// Create comprehensive report
	report := &ComplianceReport{
		ID:              fmt.Sprintf("report_%s_%d", standard, time.Now().Unix()),
		Standard:        standard,
		GeneratedAt:     time.Now(),
		ValidUntil:      time.Now().Add(30 * 24 * time.Hour), // Valid for 30 days
		Policies:        baseReport.Policies,
		Results:         baseReport.Results,
		Summary:         baseReport.Summary,
		Recommendations: []Recommendation{},
		Metadata:        make(map[string]interface{}),
	}

	// Generate recommendations
	recommendations, err := crg.generateRecommendations(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}
	report.Recommendations = recommendations

	// Enhance summary with additional metrics
	crg.enhanceSummary(report)

	// Add metadata
	report.Metadata["generator_version"] = "1.0.0"
	report.Metadata["report_type"] = "compliance_assessment"
	report.Metadata["standards_version"] = crg.getStandardVersion(standard)

	return report, nil
}

// GenerateExecutiveSummary generates an executive summary of the compliance report
func (crg *ComplianceReportGenerator) GenerateExecutiveSummary(ctx context.Context, report *ComplianceReport) (*ExecutiveSummary, error) {
	summary := &ExecutiveSummary{
		ReportID:        report.ID,
		Standard:        report.Standard,
		GeneratedAt:     report.GeneratedAt,
		ComplianceScore: 0.0,
		Status:          "Unknown",
		KeyFindings:     []string{},
		CriticalIssues:  []string{},
		Recommendations: []string{},
		NextSteps:       []string{},
		Metadata:        make(map[string]interface{}),
	}

	// Calculate compliance score
	if score, ok := report.Summary["compliance_score"].(float64); ok {
		summary.ComplianceScore = score
	}

	// Determine overall status
	if summary.ComplianceScore >= 95 {
		summary.Status = "Excellent"
	} else if summary.ComplianceScore >= 85 {
		summary.Status = "Good"
	} else if summary.ComplianceScore >= 70 {
		summary.Status = "Fair"
	} else if summary.ComplianceScore >= 50 {
		summary.Status = "Poor"
	} else {
		summary.Status = "Critical"
	}

	// Generate key findings
	summary.KeyFindings = crg.generateKeyFindings(report)

	// Generate critical issues
	summary.CriticalIssues = crg.generateCriticalIssues(report)

	// Generate recommendations
	summary.Recommendations = crg.generateExecutiveRecommendations(report)

	// Generate next steps
	summary.NextSteps = crg.generateNextSteps(report)

	return summary, nil
}

// ExecutiveSummary represents an executive summary of compliance
type ExecutiveSummary struct {
	ReportID        string                 `json:"report_id"`
	Standard        string                 `json:"standard"`
	GeneratedAt     time.Time              `json:"generated_at"`
	ComplianceScore float64                `json:"compliance_score"`
	Status          string                 `json:"status"`
	KeyFindings     []string               `json:"key_findings"`
	CriticalIssues  []string               `json:"critical_issues"`
	Recommendations []string               `json:"recommendations"`
	NextSteps       []string               `json:"next_steps"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// generateRecommendations generates compliance recommendations
func (crg *ComplianceReportGenerator) generateRecommendations(ctx context.Context, report *ComplianceReport) ([]Recommendation, error) {
	var recommendations []Recommendation

	// Analyze failed checks
	for _, result := range report.Results {
		if result.Status == "FAIL" {
			for _, violation := range result.Violations {
				recommendation := Recommendation{
					ID:          fmt.Sprintf("rec_%d", time.Now().Unix()),
					Type:        "compliance_violation",
					Priority:    crg.determinePriority(violation.Severity),
					Title:       fmt.Sprintf("Fix %s violation", violation.Type),
					Description: violation.Description,
					Action:      violation.Remediation,
					Resource:    violation.Resource,
					Policy:      result.PolicyID,
					Rule:        result.RuleID,
					Metadata: map[string]interface{}{
						"violation_type": violation.Type,
						"severity":       violation.Severity,
						"field":          violation.Field,
					},
				}
				recommendations = append(recommendations, recommendation)
			}
		}
	}

	// Add general recommendations based on standard
	standardRecommendations := crg.getStandardRecommendations(report.Standard)
	recommendations = append(recommendations, standardRecommendations...)

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		priorityOrder := map[string]int{
			"critical": 1,
			"high":     2,
			"medium":   3,
			"low":      4,
		}
		return priorityOrder[recommendations[i].Priority] < priorityOrder[recommendations[j].Priority]
	})

	return recommendations, nil
}

// enhanceSummary enhances the report summary with additional metrics
func (crg *ComplianceReportGenerator) enhanceSummary(report *ComplianceReport) {
	// Add policy coverage
	report.Summary["policy_coverage"] = len(report.Policies)

	// Add resource coverage
	resourceCount := make(map[string]int)
	for _, result := range report.Results {
		resourceCount[result.ResourceID]++
	}
	report.Summary["resources_checked"] = len(resourceCount)

	// Add violation breakdown
	violationTypes := make(map[string]int)
	for _, result := range report.Results {
		for _, violation := range result.Violations {
			violationTypes[violation.Type]++
		}
	}
	report.Summary["violation_types"] = violationTypes

	// Add severity breakdown
	severityCount := make(map[string]int)
	for _, result := range report.Results {
		severityCount[result.Severity]++
	}
	report.Summary["severity_breakdown"] = severityCount
}

// generateKeyFindings generates key findings for the executive summary
func (crg *ComplianceReportGenerator) generateKeyFindings(report *ComplianceReport) []string {
	var findings []string

	// Compliance score finding
	if score, ok := report.Summary["compliance_score"].(float64); ok {
		findings = append(findings, fmt.Sprintf("Overall compliance score: %.1f%%", score))
	}

	// Policy coverage finding
	if coverage, ok := report.Summary["policy_coverage"].(int); ok {
		findings = append(findings, fmt.Sprintf("%d compliance policies evaluated", coverage))
	}

	// Resource coverage finding
	if resources, ok := report.Summary["resources_checked"].(int); ok {
		findings = append(findings, fmt.Sprintf("%d resources assessed for compliance", resources))
	}

	// Violation summary
	if violations, ok := report.Summary["violation_types"].(map[string]int); ok {
		if len(violations) > 0 {
			findings = append(findings, fmt.Sprintf("Found violations in %d different categories", len(violations)))
		} else {
			findings = append(findings, "No compliance violations detected")
		}
	}

	return findings
}

// generateCriticalIssues generates critical issues for the executive summary
func (crg *ComplianceReportGenerator) generateCriticalIssues(report *ComplianceReport) []string {
	var issues []string

	// Check for critical violations
	for _, result := range report.Results {
		if result.Severity == "critical" && result.Status == "FAIL" {
			issues = append(issues, fmt.Sprintf("Critical violation in resource %s: %s", result.ResourceID, result.Message))
		}
	}

	// Check for high-severity violations
	highSeverityCount := 0
	for _, result := range report.Results {
		if result.Severity == "high" && result.Status == "FAIL" {
			highSeverityCount++
		}
	}
	if highSeverityCount > 0 {
		issues = append(issues, fmt.Sprintf("%d high-severity violations require immediate attention", highSeverityCount))
	}

	// Check compliance score
	if score, ok := report.Summary["compliance_score"].(float64); ok {
		if score < 70 {
			issues = append(issues, fmt.Sprintf("Compliance score below acceptable threshold (%.1f%%)", score))
		}
	}

	return issues
}

// generateExecutiveRecommendations generates executive-level recommendations
func (crg *ComplianceReportGenerator) generateExecutiveRecommendations(report *ComplianceReport) []string {
	var recommendations []string

	// Compliance score recommendations
	if score, ok := report.Summary["compliance_score"].(float64); ok {
		if score < 85 {
			recommendations = append(recommendations, "Implement comprehensive compliance monitoring and remediation program")
		}
	}

	// Violation-based recommendations
	if violations, ok := report.Summary["violation_types"].(map[string]int); ok {
		if len(violations) > 0 {
			recommendations = append(recommendations, "Establish regular compliance reviews and automated remediation")
		}
	}

	// Policy-based recommendations
	if coverage, ok := report.Summary["policy_coverage"].(int); ok {
		if coverage < 5 {
			recommendations = append(recommendations, "Expand compliance policy coverage to include additional standards")
		}
	}

	// General recommendations
	recommendations = append(recommendations, "Schedule regular compliance assessments and audits")
	recommendations = append(recommendations, "Implement continuous compliance monitoring")

	return recommendations
}

// generateNextSteps generates next steps for the executive summary
func (crg *ComplianceReportGenerator) generateNextSteps(report *ComplianceReport) []string {
	var steps []string

	// Immediate actions
	steps = append(steps, "Review and prioritize critical and high-severity violations")
	steps = append(steps, "Develop remediation plan for identified issues")

	// Short-term actions
	steps = append(steps, "Implement automated compliance monitoring")
	steps = append(steps, "Establish compliance review process")

	// Long-term actions
	steps = append(steps, "Schedule next compliance assessment")
	steps = append(steps, "Consider additional compliance standards")

	return steps
}

// determinePriority determines the priority of a recommendation based on severity
func (crg *ComplianceReportGenerator) determinePriority(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

// getStandardRecommendations gets standard-specific recommendations
func (crg *ComplianceReportGenerator) getStandardRecommendations(standard string) []Recommendation {
	var recommendations []Recommendation

	switch standard {
	case "SOC2":
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("soc2_rec_%d", time.Now().Unix()),
			Type:        "standard_recommendation",
			Priority:    "medium",
			Title:       "Implement SOC2 Type II Controls",
			Description: "Ensure all SOC2 Type II control requirements are met",
			Action:      "Review and implement SOC2 Type II control framework",
			Metadata:    map[string]interface{}{"standard": "SOC2"},
		})
	case "HIPAA":
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("hipaa_rec_%d", time.Now().Unix()),
			Type:        "standard_recommendation",
			Priority:    "high",
			Title:       "Implement HIPAA Security Rule",
			Description: "Ensure compliance with HIPAA Security Rule requirements",
			Action:      "Review and implement HIPAA Security Rule controls",
			Metadata:    map[string]interface{}{"standard": "HIPAA"},
		})
	case "PCI-DSS":
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("pci_rec_%d", time.Now().Unix()),
			Type:        "standard_recommendation",
			Priority:    "high",
			Title:       "Implement PCI-DSS Requirements",
			Description: "Ensure compliance with PCI-DSS requirements",
			Action:      "Review and implement PCI-DSS control framework",
			Metadata:    map[string]interface{}{"standard": "PCI-DSS"},
		})
	}

	return recommendations
}

// getStandardVersion gets the version of a compliance standard
func (crg *ComplianceReportGenerator) getStandardVersion(standard string) string {
	versions := map[string]string{
		"SOC2":     "2017",
		"HIPAA":    "2013",
		"PCI-DSS":  "3.2.1",
		"ISO27001": "2013",
		"NIST":     "800-53",
	}

	if version, exists := versions[standard]; exists {
		return version
	}
	return "Unknown"
}
