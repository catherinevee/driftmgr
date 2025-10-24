package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/compliance"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
)

func BenchmarkOPAEngine_Evaluate(b *testing.B) {
	// Setup
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}
	engine := compliance.NewOPAEngine(config)

	ctx := context.Background()
	err := engine.LoadPolicies(ctx)
	if err != nil {
		b.Fatalf("Failed to load policies: %v", err)
	}

	// Test input
	input := compliance.PolicyInput{
		Resource: map[string]interface{}{
			"type": "s3_bucket",
			"name": "test-bucket",
			"encryption": map[string]interface{}{
				"enabled": true,
			},
		},
		Action: "read",
		Tags: map[string]string{
			"Owner":       "test-user",
			"Environment": "production",
			"CostCenter":  "engineering",
		},
		Provider: "aws",
		Region:   "us-east-1",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.Evaluate(ctx, "aws.security", input)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

func BenchmarkOPAEngine_EvaluateWithCache(b *testing.B) {
	// Setup
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}
	engine := compliance.NewOPAEngine(config)

	ctx := context.Background()
	err := engine.LoadPolicies(ctx)
	if err != nil {
		b.Fatalf("Failed to load policies: %v", err)
	}

	// Test input
	input := compliance.PolicyInput{
		Resource: map[string]interface{}{
			"type": "s3_bucket",
			"name": "test-bucket",
		},
		Action: "read",
		Tags: map[string]string{
			"Owner": "test-user",
		},
	}

	// Warm up cache
	_, err = engine.Evaluate(ctx, "aws.security", input)
	if err != nil {
		b.Fatalf("Warm up evaluation failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.Evaluate(ctx, "aws.security", input)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

func BenchmarkComplianceService_BatchEvaluatePolicies(b *testing.B) {
	// Setup
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}
	policyEngine := compliance.NewOPAEngine(config)

	ctx := context.Background()
	err := policyEngine.LoadPolicies(ctx)
	if err != nil {
		b.Fatalf("Failed to load policies: %v", err)
	}

	// Create mock compliance reporter
	dataSource := &mockDataSource{}
	reporter := compliance.NewComplianceReporter(dataSource, policyEngine)
	service := compliance.NewComplianceService(policyEngine, reporter)

	// Test input
	input := compliance.PolicyInput{
		Resource: map[string]interface{}{
			"type": "s3_bucket",
			"name": "test-bucket",
		},
		Action: "read",
		Tags: map[string]string{
			"Owner": "test-user",
		},
	}

	policyPackages := []string{"aws.security", "hipaa.compliance"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.BatchEvaluatePolicies(ctx, policyPackages, input)
		if err != nil {
			b.Fatalf("Batch evaluation failed: %v", err)
		}
	}
}

func BenchmarkPDFFormatter_Format(b *testing.B) {
	// Create test report
	report := &compliance.ComplianceReport{
		ID:          "benchmark-report",
		Type:        compliance.ComplianceSOC2,
		Title:       "SOC 2 Compliance Report",
		GeneratedAt: time.Now(),
		Period: compliance.ReportPeriod{
			Start: time.Now().AddDate(0, -1, 0),
			End:   time.Now(),
		},
		Summary: compliance.ReportSummary{
			TotalControls:    100,
			PassedControls:   85,
			FailedControls:   15,
			ComplianceScore:  85.0,
			CriticalFindings: 5,
			HighFindings:     10,
			MediumFindings:   20,
			LowFindings:      15,
		},
		Sections: []compliance.ReportSection{
			{
				Title:       "Security",
				Description: "Security controls and measures",
				Controls: []compliance.Control{
					{
						ID:          "CC6.1",
						Title:       "Logical Access Controls",
						Description: "The entity implements logical access security software",
						Category:    "Security",
						Status:      compliance.ControlStatusPassed,
						Findings:    []compliance.Finding{},
					},
					{
						ID:          "CC6.2",
						Title:       "Encryption",
						Description: "The entity uses encryption to supplement other access controls",
						Category:    "Security",
						Status:      compliance.ControlStatusFailed,
						Findings: []compliance.Finding{
							{
								ID:          "finding-1",
								Severity:    "high",
								Title:       "Unencrypted S3 Bucket",
								Description: "S3 bucket is not encrypted",
								Resource:    "s3://test-bucket",
								Remediation: "Enable encryption on the S3 bucket",
							},
						},
					},
				},
			},
		},
	}

	formatter := &compliance.PDFFormatter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(report)
		if err != nil {
			b.Fatalf("PDF generation failed: %v", err)
		}
	}
}

func BenchmarkHTMLFormatter_Format(b *testing.B) {
	// Create test report (same as PDF benchmark)
	report := &compliance.ComplianceReport{
		ID:          "benchmark-report",
		Type:        compliance.ComplianceSOC2,
		Title:       "SOC 2 Compliance Report",
		GeneratedAt: time.Now(),
		Period: compliance.ReportPeriod{
			Start: time.Now().AddDate(0, -1, 0),
			End:   time.Now(),
		},
		Summary: compliance.ReportSummary{
			TotalControls:    100,
			PassedControls:   85,
			FailedControls:   15,
			ComplianceScore:  85.0,
			CriticalFindings: 5,
			HighFindings:     10,
			MediumFindings:   20,
			LowFindings:      15,
		},
		Sections: []compliance.ReportSection{
			{
				Title:       "Security",
				Description: "Security controls and measures",
				Controls: []compliance.Control{
					{
						ID:          "CC6.1",
						Title:       "Logical Access Controls",
						Description: "The entity implements logical access security software",
						Category:    "Security",
						Status:      compliance.ControlStatusPassed,
						Findings:    []compliance.Finding{},
					},
				},
			},
		},
	}

	formatter := &compliance.HTMLFormatter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(report)
		if err != nil {
			b.Fatalf("HTML generation failed: %v", err)
		}
	}
}

func BenchmarkJSONFormatter_Format(b *testing.B) {
	// Create test report (same as PDF benchmark)
	report := &compliance.ComplianceReport{
		ID:          "benchmark-report",
		Type:        compliance.ComplianceSOC2,
		Title:       "SOC 2 Compliance Report",
		GeneratedAt: time.Now(),
		Period: compliance.ReportPeriod{
			Start: time.Now().AddDate(0, -1, 0),
			End:   time.Now(),
		},
		Summary: compliance.ReportSummary{
			TotalControls:    100,
			PassedControls:   85,
			FailedControls:   15,
			ComplianceScore:  85.0,
			CriticalFindings: 5,
			HighFindings:     10,
			MediumFindings:   20,
			LowFindings:      15,
		},
	}

	formatter := &compliance.JSONFormatter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(report)
		if err != nil {
			b.Fatalf("JSON generation failed: %v", err)
		}
	}
}

func BenchmarkComplianceService_GenerateComplianceReport(b *testing.B) {
	// Setup
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}
	policyEngine := compliance.NewOPAEngine(config)

	ctx := context.Background()
	err := policyEngine.LoadPolicies(ctx)
	if err != nil {
		b.Fatalf("Failed to load policies: %v", err)
	}

	// Create mock compliance reporter
	dataSource := &mockDataSource{}
	reporter := compliance.NewComplianceReporter(dataSource, policyEngine)
	service := compliance.NewComplianceService(policyEngine, reporter)

	period := compliance.ReportPeriod{
		Start: time.Now().AddDate(0, -1, 0),
		End:   time.Now(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.GenerateComplianceReport(ctx, compliance.ComplianceSOC2, period)
		if err != nil {
			b.Fatalf("Report generation failed: %v", err)
		}
	}
}

// Memory benchmarks

func BenchmarkOPAEngine_MemoryUsage(b *testing.B) {
	// Setup
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}
	engine := compliance.NewOPAEngine(config)

	ctx := context.Background()
	err := engine.LoadPolicies(ctx)
	if err != nil {
		b.Fatalf("Failed to load policies: %v", err)
	}

	// Test input
	input := compliance.PolicyInput{
		Resource: map[string]interface{}{
			"type": "s3_bucket",
			"name": "test-bucket",
		},
		Action: "read",
		Tags: map[string]string{
			"Owner": "test-user",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.Evaluate(ctx, "aws.security", input)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
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
