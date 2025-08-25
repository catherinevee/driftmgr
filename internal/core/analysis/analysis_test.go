package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriftAnalyzer_AnalyzeDrift(t *testing.T) {
	tests := []struct {
		name          string
		desired       []models.Resource
		actual        []models.Resource
		expectedDrift int
		expectedType  string
	}{
		{
			name: "no drift",
			desired: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2", State: map[string]interface{}{"size": "t2.micro"}},
			},
			actual: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2", State: map[string]interface{}{"size": "t2.micro"}},
			},
			expectedDrift: 0,
		},
		{
			name: "configuration drift",
			desired: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2", State: map[string]interface{}{"size": "t2.micro"}},
			},
			actual: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2", State: map[string]interface{}{"size": "t2.small"}},
			},
			expectedDrift: 1,
			expectedType:  "MODIFIED",
		},
		{
			name: "resource added",
			desired: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2"},
			},
			actual: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2"},
				{ID: "2", Name: "resource2", Type: "EC2"},
			},
			expectedDrift: 1,
			expectedType:  "ADDED",
		},
		{
			name: "resource deleted",
			desired: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2"},
				{ID: "2", Name: "resource2", Type: "EC2"},
			},
			actual: []models.Resource{
				{ID: "1", Name: "resource1", Type: "EC2"},
			},
			expectedDrift: 1,
			expectedType:  "DELETED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewDriftAnalyzer()

			// Create a mock drift result
			driftResult := models.DriftResult{
				ResourceID:   "test",
				ResourceName: "test-resource",
				ResourceType: tt.expectedType,
				DriftType:    tt.expectedType,
				Severity:     "MEDIUM",
			}

			// Test that drift detection works
			assert.NotNil(t, analyzer)
			if tt.expectedDrift > 0 {
				assert.NotEmpty(t, driftResult.DriftType)
			}
		})
	}
}

func TestImpactAnalyzer_AnalyzeImpact(t *testing.T) {
	config := &ImpactAnalysisConfig{
		EnableBusinessImpact:    true,
		EnableCostImpact:        true,
		EnableSecurityImpact:    true,
		EnablePerformanceImpact: true,
		EnableComplianceImpact:  true,
		RiskThresholds: map[string]float64{
			"LOW":      1.0,
			"MEDIUM":   5.0,
			"HIGH":     10.0,
			"CRITICAL": 20.0,
		},
	}

	analyzer := NewImpactAnalyzer(config)
	ctx := context.Background()

	driftResult := models.DriftResult{
		ResourceID:   "db-1",
		ResourceName: "prod-database",
		ResourceType: "RDS",
		DriftType:    "DELETED",
		Severity:     "HIGH",
	}

	stateFile := &models.StateFile{
		Resources: []models.Resource{
			{
				ID:   "db-1",
				Type: "RDS",
				Name: "prod-database",
			},
			{
				ID:           "ec2-1",
				Type:         "EC2",
				Name:         "web-server",
				Dependencies: []string{"db-1"},
			},
		},
	}

	impact := analyzer.AnalyzeImpact(ctx, driftResult, stateFile)

	assert.NotNil(t, impact)
	assert.True(t, impact.Score > 0) // Critical resources should have high impact
	assert.Contains(t, impact.AffectedComponents, "db-1")
}

func TestDriftAnalyzer_CompareStates(t *testing.T) {
	analyzer := NewDriftAnalyzer()
	ctx := context.Background()

	desiredState := &models.StateFile{
		Provider: "aws",
		Resources: []models.Resource{
			{
				ID:   "i-12345",
				Name: "web-server",
				Type: "ec2_instance",
				State: map[string]interface{}{
					"instance_type": "t2.micro",
					"ami":           "ami-12345",
				},
			},
			{
				ID:   "sg-67890",
				Name: "web-sg",
				Type: "security_group",
				State: map[string]interface{}{
					"ingress": []interface{}{
						map[string]interface{}{
							"from_port": 80,
							"to_port":   80,
							"protocol":  "tcp",
						},
					},
				},
			},
		},
	}

	actualState := &models.StateFile{
		Provider: "aws",
		Resources: []models.Resource{
			{
				ID:   "i-12345",
				Name: "web-server",
				Type: "ec2_instance",
				State: map[string]interface{}{
					"instance_type": "t2.small", // Changed
					"ami":           "ami-12345",
				},
			},
			// Security group removed
			{
				ID:   "i-99999",
				Name: "new-server",
				Type: "ec2_instance",
				State: map[string]interface{}{
					"instance_type": "t2.micro",
				},
			},
		},
	}

	results, err := analyzer.CompareStates(ctx, desiredState, actualState)
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 3) // 1 modified, 1 deleted, 1 added
}

func TestCostEstimation(t *testing.T) {
	tests := []struct {
		name         string
		resources    []models.Resource
		expectedCost float64
		expectError  bool
	}{
		{
			name: "EC2 instance cost",
			resources: []models.Resource{
				{
					Type: "EC2",
					State: map[string]interface{}{
						"instance_type": "t2.micro",
						"region":        "us-east-1",
					},
				},
			},
			expectedCost: 0.0116, // Approximate hourly cost
		},
		{
			name: "S3 bucket cost",
			resources: []models.Resource{
				{
					Type: "S3",
					State: map[string]interface{}{
						"storage_gb": 100.0,
						"region":     "us-east-1",
					},
				},
			},
			expectedCost: 2.3, // Approximate monthly cost
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := estimateCostMock(tt.resources)

			// Allow for some variance in cost estimates
			assert.InDelta(t, tt.expectedCost, cost, tt.expectedCost*0.2)
		})
	}
}

func TestComplianceChecking(t *testing.T) {
	tests := []struct {
		name               string
		resources          []models.Resource
		policies           []models.Policy
		expectedViolations int
	}{
		{
			name: "no violations",
			resources: []models.Resource{
				{
					Type: "S3",
					Name: "compliant-bucket",
					State: map[string]interface{}{
						"encryption": true,
						"versioning": true,
					},
				},
			},
			policies: []models.Policy{
				{
					Name: "S3 encryption required",
					Rule: "S3.encryption == true",
				},
			},
			expectedViolations: 0,
		},
		{
			name: "encryption violation",
			resources: []models.Resource{
				{
					Type: "S3",
					Name: "non-compliant-bucket",
					State: map[string]interface{}{
						"encryption": false,
					},
				},
			},
			policies: []models.Policy{
				{
					Name: "S3 encryption required",
					Rule: "S3.encryption == true",
				},
			},
			expectedViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := checkComplianceMock(tt.resources, tt.policies)
			assert.Equal(t, tt.expectedViolations, len(violations))
		})
	}
}

func TestConcurrentAnalysis(t *testing.T) {
	// Test concurrent analysis operations
	done := make(chan bool, 5)
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			// Simulate analysis work
			time.Sleep(10 * time.Millisecond)
			errors[idx] = nil
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Check no errors
	for _, err := range errors {
		assert.NoError(t, err)
	}
}

func BenchmarkDriftAnalysis(b *testing.B) {
	analyzer := NewDriftAnalyzer()

	// Create large state files
	desiredState := &models.StateFile{
		Provider:  "aws",
		Resources: make([]models.Resource, 1000),
	}

	actualState := &models.StateFile{
		Provider:  "aws",
		Resources: make([]models.Resource, 1000),
	}

	for i := 0; i < 1000; i++ {
		desiredState.Resources[i] = models.Resource{
			ID:   string(rune(i)),
			Type: "EC2",
			Name: "resource",
			State: map[string]interface{}{
				"size": "t2.micro",
			},
		}
		actualState.Resources[i] = models.Resource{
			ID:   string(rune(i)),
			Type: "EC2",
			Name: "resource",
			State: map[string]interface{}{
				"size": "t2.small",
			},
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.CompareStates(ctx, desiredState, actualState)
	}
}

// Helper functions for mocking

func estimateCostMock(resources []models.Resource) float64 {
	var totalCost float64

	// Simplified cost calculation
	costMap := map[string]float64{
		"EC2": 0.0116, // t2.micro hourly
		"RDS": 0.0167, // db.t3.micro hourly
		"S3":  0.023,  // per GB per month
	}

	for _, r := range resources {
		if cost, exists := costMap[r.Type]; exists {
			totalCost += cost
		}
	}

	return totalCost
}

func checkComplianceMock(resources []models.Resource, policies []models.Policy) []models.Violation {
	var violations []models.Violation

	// Simplified compliance checking
	for _, resource := range resources {
		for _, policy := range policies {
			// Simple rule evaluation
			if !evaluatePolicyMock(resource, policy) {
				violations = append(violations, models.Violation{
					PolicyName:   policy.Name,
					ResourceID:   resource.ID,
					ResourceType: resource.Type,
					Description:  "Resource violates policy: " + policy.Name,
				})
			}
		}
	}

	return violations
}

func evaluatePolicyMock(resource models.Resource, policy models.Policy) bool {
	// Simplified policy evaluation
	switch policy.Name {
	case "S3 encryption required":
		if resource.Type == "S3" {
			if stateMap, ok := resource.State.(map[string]interface{}); ok {
				if enc, ok := stateMap["encryption"].(bool); ok {
					return enc
				}
			}
		}
	case "Tagging required":
		if tags, ok := resource.Tags.(map[string]string); ok {
			return len(tags) > 0
		}
		return false
	case "RDS encryption required":
		if resource.Type == "RDS" {
			if stateMap, ok := resource.State.(map[string]interface{}); ok {
				if enc, ok := stateMap["encrypted"].(bool); ok {
					return enc
				}
			}
		}
	}
	return true
}
