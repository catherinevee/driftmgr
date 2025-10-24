package compliance

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/compliance"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplianceService_CreatePolicy(t *testing.T) {
	// Setup
	policyEngine := createMockOPAEngine(t)
	reporter := createMockComplianceReporter(t)
	service := compliance.NewComplianceService(policyEngine, reporter)

	tests := []struct {
		name        string
		policy      *compliance.Policy
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid policy",
			policy: &compliance.Policy{
				ID:          "test-policy-1",
				Name:        "Test Policy",
				Description: "A test policy",
				Package:     "test.policy",
				Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
}`,
			},
			expectError: false,
		},
		{
			name: "missing policy ID",
			policy: &compliance.Policy{
				Name:        "Test Policy",
				Description: "A test policy",
				Package:     "test.policy",
				Rules: `package test.policy

default allow = false`,
			},
			expectError: true,
			errorMsg:    "policy ID is required",
		},
		{
			name: "missing policy rules",
			policy: &compliance.Policy{
				ID:          "test-policy-2",
				Name:        "Test Policy",
				Description: "A test policy",
				Package:     "test.policy",
			},
			expectError: true,
			errorMsg:    "policy rules are required",
		},
		{
			name: "missing policy package",
			policy: &compliance.Policy{
				ID:          "test-policy-3",
				Name:        "Test Policy",
				Description: "A test policy",
				Rules: `package test.policy

default allow = false`,
			},
			expectError: true,
			errorMsg:    "policy package is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := service.CreatePolicy(ctx, tt.policy)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.policy.CreatedAt)
				assert.NotZero(t, tt.policy.UpdatedAt)
			}
		})
	}
}

func TestComplianceService_EvaluatePolicy(t *testing.T) {
	// Setup
	policyEngine := createMockOPAEngine(t)
	reporter := createMockComplianceReporter(t)
	service := compliance.NewComplianceService(policyEngine, reporter)

	// Create a test policy first
	policy := &compliance.Policy{
		ID:          "test-policy",
		Name:        "Test Policy",
		Description: "A test policy",
		Package:     "test.policy",
		Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
}

violations["missing_owner"] {
    not input.tags.Owner
    violation := {
        "rule": "required_tags",
        "message": "Missing required tag: Owner",
        "severity": "medium",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Add Owner tag to the resource"
    }
}`,
	}

	ctx := context.Background()
	err := service.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	tests := []struct {
		name               string
		input              compliance.PolicyInput
		expectedAllow      bool
		expectedViolations int
	}{
		{
			name: "allowed action",
			input: compliance.PolicyInput{
				Resource: map[string]interface{}{
					"type": "s3_bucket",
					"name": "test-bucket",
				},
				Action: "read",
				Tags: map[string]string{
					"Owner": "test-user",
				},
			},
			expectedAllow:      true,
			expectedViolations: 0,
		},
		{
			name: "denied action with violation",
			input: compliance.PolicyInput{
				Resource: map[string]interface{}{
					"type": "s3_bucket",
					"name": "test-bucket",
				},
				Action: "write",
				Tags:   map[string]string{},
			},
			expectedAllow:      false,
			expectedViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := service.EvaluatePolicy(ctx, policy.Package, tt.input)

			assert.NoError(t, err)
			assert.NotNil(t, decision)
			assert.Equal(t, tt.expectedAllow, decision.Allow)
			assert.Len(t, decision.Violations, tt.expectedViolations)
			assert.NotZero(t, decision.EvaluatedAt)
		})
	}
}

func TestComplianceService_BatchEvaluatePolicies(t *testing.T) {
	// Setup
	policyEngine := createMockOPAEngine(t)
	reporter := createMockComplianceReporter(t)
	service := compliance.NewComplianceService(policyEngine, reporter)

	// Create test policies
	policies := []*compliance.Policy{
		{
			ID:      "policy-1",
			Name:    "Policy 1",
			Package: "test.policy1",
			Rules: `package test.policy1

default allow = false

allow {
    input.action == "read"
}`,
		},
		{
			ID:      "policy-2",
			Name:    "Policy 2",
			Package: "test.policy2",
			Rules: `package test.policy2

default allow = false

allow {
    input.action == "write"
}`,
		},
	}

	ctx := context.Background()
	for _, policy := range policies {
		err := service.CreatePolicy(ctx, policy)
		require.NoError(t, err)
	}

	input := compliance.PolicyInput{
		Resource: map[string]interface{}{
			"type": "s3_bucket",
			"name": "test-bucket",
		},
		Action: "read",
	}

	policyPackages := []string{"test.policy1", "test.policy2"}
	results, err := service.BatchEvaluatePolicies(ctx, policyPackages, input)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results["test.policy1"].Allow)
	assert.False(t, results["test.policy2"].Allow)
}

func TestComplianceService_ValidatePolicy(t *testing.T) {
	// Setup
	policyEngine := createMockOPAEngine(t)
	reporter := createMockComplianceReporter(t)
	service := compliance.NewComplianceService(policyEngine, reporter)

	tests := []struct {
		name        string
		policy      *compliance.Policy
		expectError bool
	}{
		{
			name: "valid policy",
			policy: &compliance.Policy{
				ID:      "valid-policy",
				Package: "test.policy",
				Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
}`,
			},
			expectError: false,
		},
		{
			name: "invalid policy syntax",
			policy: &compliance.Policy{
				ID:      "invalid-policy",
				Package: "test.policy",
				Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
    // Missing closing brace
}`,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := service.ValidatePolicy(ctx, tt.policy)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComplianceService_GenerateComplianceReport(t *testing.T) {
	// Setup
	policyEngine := createMockOPAEngine(t)
	reporter := createMockComplianceReporter(t)
	service := compliance.NewComplianceService(policyEngine, reporter)

	ctx := context.Background()
	period := compliance.ReportPeriod{
		Start: time.Now().AddDate(0, -1, 0),
		End:   time.Now(),
	}

	report, err := service.GenerateComplianceReport(ctx, compliance.ComplianceSOC2, period)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, compliance.ComplianceSOC2, report.Type)
	assert.NotEmpty(t, report.ID)
	assert.NotEmpty(t, report.Title)
	assert.Equal(t, period, report.Period)
	assert.NotZero(t, report.GeneratedAt)
}

func TestComplianceService_ExportReport(t *testing.T) {
	// Setup
	policyEngine := createMockOPAEngine(t)
	reporter := createMockComplianceReporter(t)
	service := compliance.NewComplianceService(policyEngine, reporter)

	report := &compliance.ComplianceReport{
		ID:          "test-report",
		Type:        compliance.ComplianceSOC2,
		Title:       "Test Report",
		GeneratedAt: time.Now(),
		Period: compliance.ReportPeriod{
			Start: time.Now().AddDate(0, -1, 0),
			End:   time.Now(),
		},
		Summary: compliance.ReportSummary{
			TotalControls:    10,
			PassedControls:   8,
			FailedControls:   2,
			ComplianceScore:  80.0,
			CriticalFindings: 1,
			HighFindings:     2,
			MediumFindings:   3,
			LowFindings:      1,
		},
	}

	tests := []struct {
		name     string
		format   string
		expected bool
	}{
		{"JSON format", "json", true},
		{"PDF format", "pdf", true},
		{"HTML format", "html", true},
		{"YAML format", "yaml", true},
		{"Unsupported format", "xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			data, err := service.ExportReport(ctx, report, tt.format)

			if tt.expected {
				assert.NoError(t, err)
				assert.NotEmpty(t, data)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported format")
			}
		})
	}
}

// Helper functions

func createMockOPAEngine(t *testing.T) *compliance.OPAEngine {
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)

	// Load policies
	ctx := context.Background()
	err := engine.LoadPolicies(ctx)
	if err != nil {
		t.Logf("Warning: Could not load policies: %v", err)
	}

	return engine
}

func createMockComplianceReporter(t *testing.T) *compliance.ComplianceReporter {
	// Create a mock data source
	dataSource := &mockDataSource{}

	// Create a mock policy engine for the reporter
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}
	policyEngine := compliance.NewOPAEngine(config)

	return compliance.NewComplianceReporter(dataSource, policyEngine)
}

// Mock data source for testing
type mockDataSource struct{}

func (m *mockDataSource) GetDriftResults(ctx context.Context) ([]*detector.DriftResult, error) {
	return []*detector.DriftResult{}, nil
}

func (m *mockDataSource) GetPolicyViolations(ctx context.Context) ([]compliance.PolicyViolation, error) {
	return []compliance.PolicyViolation{}, nil
}

func (m *mockDataSource) GetResourceInventory(ctx context.Context) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m *mockDataSource) GetAuditLogs(ctx context.Context, since time.Time) ([]interface{}, error) {
	return []interface{}{}, nil
}
