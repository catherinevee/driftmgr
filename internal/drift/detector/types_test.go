package detector

import (
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/comparator"
	"github.com/stretchr/testify/assert"
)

func TestDriftTypes(t *testing.T) {
	tests := []struct {
		name     string
		drift    DriftType
		expected int
	}{
		{"NoDrift", NoDrift, 0},
		{"ResourceMissing", ResourceMissing, 1},
		{"ResourceUnmanaged", ResourceUnmanaged, 2},
		{"ConfigurationDrift", ConfigurationDrift, 3},
		{"ResourceOrphaned", ResourceOrphaned, 4},
		{"DriftTypeMissing alias", DriftTypeMissing, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.drift))
		})
	}

	// Test that alias works correctly
	assert.Equal(t, ResourceMissing, DriftTypeMissing)
}

func TestDriftSeverity(t *testing.T) {
	severities := []DriftSeverity{
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	for i, severity := range severities {
		assert.Equal(t, DriftSeverity(i), severity)
	}

	// Test severity ordering
	assert.Less(t, SeverityLow, SeverityMedium)
	assert.Less(t, SeverityMedium, SeverityHigh)
	assert.Less(t, SeverityHigh, SeverityCritical)
}

func TestDetectorConfig(t *testing.T) {
	tests := []struct {
		name   string
		config DetectorConfig
	}{
		{
			name: "default config",
			config: DetectorConfig{
				MaxWorkers:        5,
				Timeout:           30 * time.Second,
				CheckUnmanaged:    true,
				DeepComparison:    true,
				ParallelDiscovery: true,
				RetryAttempts:     3,
				RetryDelay:        5 * time.Second,
			},
		},
		{
			name: "minimal config",
			config: DetectorConfig{
				MaxWorkers: 1,
				Timeout:    10 * time.Second,
			},
		},
		{
			name: "config with ignored attributes",
			config: DetectorConfig{
				MaxWorkers:       5,
				Timeout:          30 * time.Second,
				IgnoreAttributes: []string{"tags", "metadata", "last_modified"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.config)
			assert.GreaterOrEqual(t, tt.config.MaxWorkers, 1)
			assert.Greater(t, tt.config.Timeout, time.Duration(0))
		})
	}
}

func TestDriftResult(t *testing.T) {
	tests := []struct {
		name   string
		result DriftResult
	}{
		{
			name: "missing resource",
			result: DriftResult{
				Resource:     "aws_instance.web",
				ResourceType: "aws_instance",
				Provider:     "aws",
				DriftType:    ResourceMissing,
				Severity:     SeverityHigh,
				DesiredState: map[string]interface{}{
					"instance_type": "t2.micro",
					"ami":           "ami-12345",
				},
				ActualState:    nil,
				Impact:         []string{"Service unavailable", "Data loss risk"},
				Recommendation: "Recreate the missing instance",
				Timestamp:      time.Now(),
			},
		},
		{
			name: "configuration drift",
			result: DriftResult{
				Resource:     "aws_s3_bucket.data",
				ResourceType: "aws_s3_bucket",
				Provider:     "aws",
				DriftType:    ConfigurationDrift,
				Severity:     SeverityMedium,
				Differences: []comparator.Difference{
					{
						Path:     "versioning.enabled",
						Expected: true,
						Actual:   false,
					},
				},
				DesiredState: map[string]interface{}{
					"versioning": map[string]interface{}{"enabled": true},
				},
				ActualState: map[string]interface{}{
					"versioning": map[string]interface{}{"enabled": false},
				},
				Impact:         []string{"No version history", "Cannot recover deleted objects"},
				Recommendation: "Enable versioning on the bucket",
				Timestamp:      time.Now(),
			},
		},
		{
			name: "unmanaged resource",
			result: DriftResult{
				Resource:     "aws_security_group.unknown",
				ResourceType: "aws_security_group",
				Provider:     "aws",
				DriftType:    ResourceUnmanaged,
				Severity:     SeverityLow,
				ActualState: map[string]interface{}{
					"name":        "unknown-sg",
					"description": "Manually created",
				},
				Impact:         []string{"Resource not tracked in state"},
				Recommendation: "Import resource or delete if unnecessary",
				Timestamp:      time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.result.Resource)
			assert.NotEmpty(t, tt.result.ResourceType)
			assert.NotEmpty(t, tt.result.Provider)
			assert.NotEmpty(t, tt.result.Recommendation)
			assert.NotZero(t, tt.result.Timestamp)

			if tt.result.DriftType == ConfigurationDrift {
				assert.NotEmpty(t, tt.result.Differences)
			}

			if tt.result.DriftType == ResourceMissing {
				assert.Nil(t, tt.result.ActualState)
				assert.NotNil(t, tt.result.DesiredState)
			}

			if tt.result.DriftType == ResourceUnmanaged {
				assert.NotNil(t, tt.result.ActualState)
				assert.Empty(t, tt.result.DesiredState)
			}
		})
	}
}

func TestDriftReport(t *testing.T) {
	report := DriftReport{
		Timestamp:          time.Now(),
		TotalResources:     100,
		DriftedResources:   15,
		MissingResources:   3,
		UnmanagedResources: 5,
		DriftResults: []DriftResult{
			{
				Resource:  "aws_instance.web",
				DriftType: ConfigurationDrift,
				Severity:  SeverityMedium,
			},
			{
				Resource:  "aws_s3_bucket.logs",
				DriftType: ResourceMissing,
				Severity:  SeverityHigh,
			},
		},
		Summary: &DriftSummary{
			ByProvider: map[string]*ProviderDriftSummary{
				"aws": {
					Provider:         "aws",
					TotalResources:   80,
					DriftedResources: 12,
					DriftPercentage:  15.0,
				},
				"azure": {
					Provider:         "azure",
					TotalResources:   20,
					DriftedResources: 3,
					DriftPercentage:  15.0,
				},
			},
			BySeverity: map[DriftSeverity]int{
				SeverityLow:      5,
				SeverityMedium:   7,
				SeverityHigh:     3,
				SeverityCritical: 0,
			},
			DriftScore: 15.0,
		},
		Recommendations: []string{
			"Review and apply missing resources",
			"Update configuration drift items",
			"Import or remove unmanaged resources",
		},
	}

	assert.NotZero(t, report.Timestamp)
	assert.Equal(t, 100, report.TotalResources)
	assert.Equal(t, 15, report.DriftedResources)
	assert.Equal(t, 3, report.MissingResources)
	assert.Equal(t, 5, report.UnmanagedResources)
	assert.Len(t, report.DriftResults, 2)
	assert.NotNil(t, report.Summary)
	assert.NotEmpty(t, report.Recommendations)

	// Test drift percentage calculation
	driftPercentage := float64(report.DriftedResources) / float64(report.TotalResources) * 100
	assert.Equal(t, 15.0, driftPercentage)
}

func TestDriftSummary(t *testing.T) {
	summary := &DriftSummary{
		ByProvider: map[string]*ProviderDriftSummary{
			"aws": {
				Provider:         "aws",
				TotalResources:   100,
				DriftedResources: 10,
				DriftPercentage:  10.0,
			},
		},
		ByType: map[string]*TypeDriftSummary{
			"aws_instance": {
				ResourceType:     "aws_instance",
				TotalResources:   50,
				DriftedResources: 5,
				CommonIssues:     []string{"missing tags"},
			},
		},
		BySeverity: map[DriftSeverity]int{
			SeverityLow:    2,
			SeverityMedium: 5,
			SeverityHigh:   3,
		},
		DriftScore: 10.0,
	}

	assert.NotNil(t, summary.ByProvider)
	assert.NotNil(t, summary.ByType)
	assert.NotNil(t, summary.BySeverity)
	assert.Equal(t, 10.0, summary.DriftScore)

	// Test provider summary
	awsSummary := summary.ByProvider["aws"]
	assert.Equal(t, "aws", awsSummary.Provider)
	assert.Equal(t, 10.0, awsSummary.DriftPercentage)

	// Test severity counts
	assert.Equal(t, 2, summary.BySeverity[SeverityLow])
	assert.Equal(t, 5, summary.BySeverity[SeverityMedium])
	assert.Equal(t, 3, summary.BySeverity[SeverityHigh])
}

func TestProviderDriftSummary(t *testing.T) {
	summary := &ProviderDriftSummary{
		Provider:         "aws",
		TotalResources:   100,
		DriftedResources: 15,
	}

	// Calculate drift percentage
	summary.DriftPercentage = float64(summary.DriftedResources) / float64(summary.TotalResources) * 100

	assert.Equal(t, "aws", summary.Provider)
	assert.Equal(t, 100, summary.TotalResources)
	assert.Equal(t, 15, summary.DriftedResources)
	assert.Equal(t, 15.0, summary.DriftPercentage)
}

func TestTypeDriftSummary(t *testing.T) {
	summary := &TypeDriftSummary{
		ResourceType:     "aws_instance",
		TotalResources:   50,
		DriftedResources: 5,
		CommonIssues:     []string{"missing tags", "wrong instance type"},
	}

	assert.Equal(t, "aws_instance", summary.ResourceType)
	assert.Equal(t, 50, summary.TotalResources)
	assert.Equal(t, 5, summary.DriftedResources)
	assert.Len(t, summary.CommonIssues, 2)

	// Calculate drift percentage manually
	driftPercentage := float64(summary.DriftedResources) / float64(summary.TotalResources) * 100
	assert.Equal(t, 10.0, driftPercentage)
}

func BenchmarkDriftResult(b *testing.B) {
	for i := 0; i < b.N; i++ {
		result := DriftResult{
			Resource:     "aws_instance.web",
			ResourceType: "aws_instance",
			Provider:     "aws",
			DriftType:    ConfigurationDrift,
			Severity:     SeverityMedium,
			Timestamp:    time.Now(),
			Differences: []comparator.Difference{
				{
					Path:     "instance_type",
					Expected: "t2.micro",
					Actual:   "t2.small",
				},
			},
		}
		_ = result.Severity
	}
}

func BenchmarkDriftReport(b *testing.B) {
	for i := 0; i < b.N; i++ {
		report := DriftReport{
			Timestamp:        time.Now(),
			TotalResources:   1000,
			DriftedResources: 150,
			DriftResults:     make([]DriftResult, 150),
		}
		_ = float64(report.DriftedResources) / float64(report.TotalResources)
	}
}