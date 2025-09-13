package health

import (
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/graph"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestHealthStatus(t *testing.T) {
	statuses := []HealthStatus{
		HealthStatusHealthy,
		HealthStatusWarning,
		HealthStatusCritical,
		HealthStatusDegraded,
		HealthStatusUnknown,
	}

	expectedStrings := []string{
		"healthy",
		"warning",
		"critical",
		"degraded",
		"unknown",
	}

	for i, status := range statuses {
		assert.Equal(t, HealthStatus(expectedStrings[i]), status)
		assert.NotEmpty(t, string(status))
	}
}

func TestSeverity(t *testing.T) {
	severities := []Severity{
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	expectedStrings := []string{
		"low",
		"medium",
		"high",
		"critical",
	}

	for i, severity := range severities {
		assert.Equal(t, Severity(expectedStrings[i]), severity)
		assert.NotEmpty(t, string(severity))
	}
}

func TestImpactLevel(t *testing.T) {
	impacts := []ImpactLevel{
		ImpactNone,
		ImpactLow,
		ImpactMedium,
		ImpactHigh,
		ImpactCritical,
	}

	expectedStrings := []string{
		"none",
		"low",
		"medium",
		"high",
		"critical",
	}

	for i, impact := range impacts {
		assert.Equal(t, ImpactLevel(expectedStrings[i]), impact)
		assert.NotEmpty(t, string(impact))
	}
}

func TestIssueType(t *testing.T) {
	types := []IssueType{
		IssueTypeMisconfiguration,
		IssueTypeDeprecation,
		IssueTypeSecurity,
		IssueTypePerformance,
		IssueTypeCost,
		IssueTypeCompliance,
		IssueTypeBestPractice,
	}

	expectedStrings := []string{
		"misconfiguration",
		"deprecation",
		"security",
		"performance",
		"cost",
		"compliance",
		"best_practice",
	}

	for i, issueType := range types {
		assert.Equal(t, IssueType(expectedStrings[i]), issueType)
		assert.NotEmpty(t, string(issueType))
	}
}

func TestHealthReport(t *testing.T) {
	tests := []struct {
		name   string
		report HealthReport
	}{
		{
			name: "healthy resource",
			report: HealthReport{
				Resource:    "aws_instance.web",
				Status:      HealthStatusHealthy,
				Score:       95,
				Issues:      []HealthIssue{},
				Suggestions: []string{},
				Impact:      ImpactNone,
				LastChecked: time.Now(),
			},
		},
		{
			name: "resource with warnings",
			report: HealthReport{
				Resource: "aws_s3_bucket.data",
				Status:   HealthStatusWarning,
				Score:    75,
				Issues: []HealthIssue{
					{
						Type:     IssueTypeSecurity,
						Severity: SeverityMedium,
						Message:  "Bucket versioning is not enabled",
						Field:    "versioning",
					},
				},
				Suggestions: []string{
					"Enable versioning for data protection",
					"Consider enabling MFA delete",
				},
				Impact:      ImpactLow,
				LastChecked: time.Now(),
			},
		},
		{
			name: "critical health issues",
			report: HealthReport{
				Resource: "aws_rds_instance.main",
				Status:   HealthStatusCritical,
				Score:    25,
				Issues: []HealthIssue{
					{
						Type:          IssueTypeSecurity,
						Severity:      SeverityCritical,
						Message:       "Database is publicly accessible",
						Field:         "publicly_accessible",
						CurrentValue:  true,
						ExpectedValue: false,
					},
					{
						Type:          IssueTypeCompliance,
						Severity:      SeverityHigh,
						Message:       "Encryption at rest is not enabled",
						Field:         "storage_encrypted",
						CurrentValue:  false,
						ExpectedValue: true,
					},
				},
				Suggestions: []string{
					"Disable public accessibility immediately",
					"Enable encryption at rest",
					"Review security group rules",
				},
				Impact:      ImpactCritical,
				LastChecked: time.Now(),
				Metadata: map[string]interface{}{
					"compliance_frameworks": []string{"HIPAA", "PCI-DSS"},
					"risk_score":            95,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.report.Resource)
			assert.NotEmpty(t, tt.report.Status)
			assert.GreaterOrEqual(t, tt.report.Score, 0)
			assert.LessOrEqual(t, tt.report.Score, 100)
			assert.NotZero(t, tt.report.LastChecked)
			assert.NotEmpty(t, tt.report.Impact)

			// Check status correlates with score
			if tt.report.Status == HealthStatusHealthy {
				assert.Greater(t, tt.report.Score, 80)
			}
			if tt.report.Status == HealthStatusCritical {
				assert.Less(t, tt.report.Score, 40)
			}

			// Check issues have required fields
			for _, issue := range tt.report.Issues {
				assert.NotEmpty(t, issue.Type)
				assert.NotEmpty(t, issue.Severity)
				assert.NotEmpty(t, issue.Message)
			}
		})
	}
}

func TestHealthIssue(t *testing.T) {
	issue := HealthIssue{
		Type:          IssueTypeSecurity,
		Severity:      SeverityHigh,
		Message:       "Security group allows unrestricted access",
		Field:         "ingress_rules",
		CurrentValue:  "0.0.0.0/0",
		ExpectedValue: "10.0.0.0/8",
		Documentation: "https://docs.aws.amazon.com/security",
		Category:      "Network Security",
		ResourceID:    "sg-12345",
	}

	assert.Equal(t, IssueTypeSecurity, issue.Type)
	assert.Equal(t, SeverityHigh, issue.Severity)
	assert.NotEmpty(t, issue.Message)
	assert.Equal(t, "ingress_rules", issue.Field)
	assert.Equal(t, "0.0.0.0/0", issue.CurrentValue)
	assert.Equal(t, "10.0.0.0/8", issue.ExpectedValue)
	assert.NotEmpty(t, issue.Documentation)
	assert.Equal(t, "Network Security", issue.Category)
	assert.Equal(t, "sg-12345", issue.ResourceID)
}

func TestSecurityRule(t *testing.T) {
	rules := []SecurityRule{
		{
			ID:          "rule-001",
			Name:        "No public S3 buckets",
			Description: "S3 buckets should not be publicly accessible",
			ResourceTypes: []string{"aws_s3_bucket"},
			Severity:    SeverityHigh,
			Category:    "Storage Security",
		},
		{
			ID:          "rule-002",
			Name:        "RDS encryption required",
			Description: "RDS instances must have encryption enabled",
			ResourceTypes: []string{"aws_rds_instance", "aws_rds_cluster"},
			Severity:    SeverityCritical,
			Category:    "Data Protection",
		},
	}

	for _, rule := range rules {
		assert.NotEmpty(t, rule.ID)
		assert.NotEmpty(t, rule.Name)
		assert.NotEmpty(t, rule.Description)
		assert.NotEmpty(t, rule.ResourceTypes)
		assert.NotEmpty(t, rule.Severity)
		assert.NotEmpty(t, rule.Category)
	}
}

func TestHealthCheck(t *testing.T) {
	check := HealthCheck{
		ID:          "check-001",
		Name:        "Instance health check",
		Type:        "availability",
		Enabled:     true,
		Interval:    5 * time.Minute,
		Timeout:     30 * time.Second,
		RetryCount:  3,
		Parameters: map[string]interface{}{
			"endpoint": "http://example.com/health",
			"method":   "GET",
		},
	}

	assert.NotEmpty(t, check.ID)
	assert.NotEmpty(t, check.Name)
	assert.NotEmpty(t, check.Type)
	assert.True(t, check.Enabled)
	assert.Equal(t, 5*time.Minute, check.Interval)
	assert.Equal(t, 30*time.Second, check.Timeout)
	assert.Equal(t, 3, check.RetryCount)
	assert.NotNil(t, check.Parameters)
}

func TestHealthAnalyzer(t *testing.T) {
	analyzer := &HealthAnalyzer{
		graph:          graph.NewDependencyGraph(),
		providers:      make(map[string]ProviderHealthChecker),
		customChecks:   []HealthCheck{},
		severityLevels: map[string]Severity{
			"low":      SeverityLow,
			"medium":   SeverityMedium,
			"high":     SeverityHigh,
			"critical": SeverityCritical,
		},
	}

	assert.NotNil(t, analyzer.graph)
	assert.NotNil(t, analyzer.providers)
	assert.NotNil(t, analyzer.customChecks)
	assert.NotNil(t, analyzer.severityLevels)
	assert.Len(t, analyzer.severityLevels, 4)
}

// Mock provider health checker
type mockProviderHealthChecker struct {
	requiredAttrs   []string
	deprecatedAttrs []string
	securityRules   []SecurityRule
}

func (m *mockProviderHealthChecker) CheckResource(resource *state.Resource, instance *state.Instance) *HealthReport {
	return &HealthReport{
		Resource: resource.Address,
		Status:   HealthStatusHealthy,
		Score:    90,
	}
}

func (m *mockProviderHealthChecker) GetRequiredAttributes(resourceType string) []string {
	return m.requiredAttrs
}

func (m *mockProviderHealthChecker) GetDeprecatedAttributes(resourceType string) []string {
	return m.deprecatedAttrs
}

func (m *mockProviderHealthChecker) GetSecurityRules(resourceType string) []SecurityRule {
	return m.securityRules
}

func TestProviderHealthChecker(t *testing.T) {
	checker := &mockProviderHealthChecker{
		requiredAttrs:   []string{"name", "type", "region"},
		deprecatedAttrs: []string{"old_field", "legacy_option"},
		securityRules: []SecurityRule{
			{
				ID:       "sec-001",
				Name:     "Test security rule",
				Severity: SeverityMedium,
			},
		},
	}

	// Test required attributes
	attrs := checker.GetRequiredAttributes("aws_instance")
	assert.Len(t, attrs, 3)
	assert.Contains(t, attrs, "name")

	// Test deprecated attributes
	deprecated := checker.GetDeprecatedAttributes("aws_instance")
	assert.Len(t, deprecated, 2)
	assert.Contains(t, deprecated, "old_field")

	// Test security rules
	rules := checker.GetSecurityRules("aws_instance")
	assert.Len(t, rules, 1)
	assert.Equal(t, "sec-001", rules[0].ID)

	// Test resource check
	resource := &state.Resource{
		Address: "aws_instance.test",
	}
	report := checker.CheckResource(resource, nil)
	assert.Equal(t, HealthStatusHealthy, report.Status)
	assert.Equal(t, 90, report.Score)
}

func TestCalculateHealthScore(t *testing.T) {
	tests := []struct {
		name          string
		issues        []HealthIssue
		expectedScore int
	}{
		{
			name:          "no issues",
			issues:        []HealthIssue{},
			expectedScore: 100,
		},
		{
			name: "minor issues",
			issues: []HealthIssue{
				{Severity: SeverityLow},
				{Severity: SeverityLow},
			},
			expectedScore: 90,
		},
		{
			name: "mixed issues",
			issues: []HealthIssue{
				{Severity: SeverityLow},
				{Severity: SeverityMedium},
				{Severity: SeverityHigh},
			},
			expectedScore: 65,
		},
		{
			name: "critical issues",
			issues: []HealthIssue{
				{Severity: SeverityCritical},
				{Severity: SeverityCritical},
			},
			expectedScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateHealthScore(tt.issues)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

// Helper function for testing
func calculateHealthScore(issues []HealthIssue) int {
	if len(issues) == 0 {
		return 100
	}

	score := 100
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityLow:
			score -= 5
		case SeverityMedium:
			score -= 10
		case SeverityHigh:
			score -= 20
		case SeverityCritical:
			score -= 50
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}

func BenchmarkHealthReport(b *testing.B) {
	for i := 0; i < b.N; i++ {
		report := HealthReport{
			Resource:    "aws_instance.bench",
			Status:      HealthStatusHealthy,
			Score:       95,
			LastChecked: time.Now(),
		}
		_ = report.Score
	}
}