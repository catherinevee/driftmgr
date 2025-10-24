package simulation_test

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/simulation"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSSimulator_SimulateDrift(t *testing.T) {
	simulator := simulation.NewAWSSimulator()
	ctx := context.Background()

	// Create a mock state with a test resource
	mockState := &state.TerraformState{
		Resources: []state.Resource{
			{
				ID:   "test-instance",
				Type: "aws_instance",
				Name: "test-instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name       string
		driftType  simulation.DriftType
		resourceID string
		expectErr  bool
	}{
		{
			name:       "tag change drift",
			driftType:  simulation.DriftTypeTagChange,
			resourceID: "test-instance",
			expectErr:  false,
		},
		{
			name:       "rule addition drift",
			driftType:  simulation.DriftTypeRuleAddition,
			resourceID: "test-instance",
			expectErr:  false,
		},
		{
			name:       "resource creation drift",
			driftType:  simulation.DriftTypeResourceCreation,
			resourceID: "test-instance",
			expectErr:  false,
		},
		{
			name:       "attribute change drift",
			driftType:  simulation.DriftTypeAttributeChange,
			resourceID: "test-instance",
			expectErr:  false,
		},
		{
			name:       "resource deletion drift",
			driftType:  simulation.DriftTypeResourceDeletion,
			resourceID: "test-instance",
			expectErr:  false,
		},
		{
			name:       "unsupported drift type",
			driftType:  "unsupported",
			resourceID: "test-instance",
			expectErr:  true,
		},
		{
			name:       "resource not found",
			driftType:  simulation.DriftTypeTagChange,
			resourceID: "nonexistent-resource",
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := simulator.SimulateDrift(ctx, tc.driftType, tc.resourceID, mockState)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "aws", result.Provider)
				assert.Equal(t, tc.driftType, result.DriftType)
				assert.Equal(t, tc.resourceID, result.ResourceID)
			}
		})
	}
}

func TestAWSSimulator_SimulateResourceDeletion(t *testing.T) {
	simulator := simulation.NewAWSSimulator()
	ctx := context.Background()

	testCases := []struct {
		name         string
		resourceType string
		resourceID   string
		expectErr    bool
	}{
		{
			name:         "EC2 instance deletion",
			resourceType: "aws_instance",
			resourceID:   "test-instance",
			expectErr:    false,
		},
		{
			name:         "S3 bucket deletion",
			resourceType: "aws_s3_bucket",
			resourceID:   "test-bucket",
			expectErr:    false,
		},
		{
			name:         "Security group deletion",
			resourceType: "aws_security_group",
			resourceID:   "test-sg",
			expectErr:    false,
		},
		{
			name:         "Generic resource deletion",
			resourceType: "aws_unknown_resource",
			resourceID:   "test-resource",
			expectErr:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource := state.Resource{
				ID:   tc.resourceID,
				Type: tc.resourceType,
				Name: tc.resourceID,
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": tc.resourceID,
						},
					},
				},
			}

			result, err := simulator.SimulateDrift(ctx, simulation.DriftTypeResourceDeletion, tc.resourceID, &state.TerraformState{
				Resources: []state.Resource{resource},
			})

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, simulation.DriftTypeResourceDeletion, result.DriftType)
				assert.Equal(t, tc.resourceID, result.ResourceID)
				assert.True(t, result.Success)
				assert.Contains(t, result.Changes, "deletion_simulated")
				assert.True(t, result.Changes["deletion_simulated"].(bool))
				assert.NotNil(t, result.RollbackData)
				assert.Len(t, result.DetectedDrift, 1)
				assert.Equal(t, "resource_deletion", result.DetectedDrift[0].DriftType)
			}
		})
	}
}

func TestAWSSimulator_Initialize(t *testing.T) {
	simulator := simulation.NewAWSSimulator()
	ctx := context.Background()

	// Test initialization (this will fail in test environment without AWS credentials, but we can test the structure)
	err := simulator.Initialize(ctx)

	// In a test environment without AWS credentials, this should fail
	// but we can verify the method exists and handles errors appropriately
	if err != nil {
		assert.Contains(t, err.Error(), "failed to load AWS config")
	}
}

func TestAWSSimulator_ResourceStructure(t *testing.T) {
	simulator := simulation.NewAWSSimulator()

	// Test that simulator can handle different resource types
	resourceTypes := []string{
		"aws_instance",
		"aws_s3_bucket",
		"aws_security_group",
		"aws_vpc",
		"aws_subnet",
	}

	for _, resourceType := range resourceTypes {
		t.Run(resourceType, func(t *testing.T) {
			resource := state.Resource{
				ID:   "test-resource",
				Type: resourceType,
				Name: "test-resource",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "test-resource-id",
						},
					},
				},
			}

			mockState := &state.TerraformState{
				Resources: []state.Resource{resource},
			}

			// Test that the simulator can handle this resource type
			result, err := simulator.SimulateDrift(context.Background(), simulation.DriftTypeTagChange, "test-resource", mockState)

			// The simulation should either succeed or fail gracefully
			if err != nil {
				// If it fails, it should be a known error, not a panic
				assert.NotContains(t, err.Error(), "panic", "Should not panic")
			} else {
				assert.NotNil(t, result, "Result should not be nil if no error")
			}
		})
	}
}

func TestAWSSimulator_AllDriftTypes(t *testing.T) {
	simulator := simulation.NewAWSSimulator()
	ctx := context.Background()

	// Test all supported drift types
	driftTypes := []simulation.DriftType{
		simulation.DriftTypeTagChange,
		simulation.DriftTypeRuleAddition,
		simulation.DriftTypeResourceCreation,
		simulation.DriftTypeAttributeChange,
		simulation.DriftTypeResourceDeletion,
	}

	mockState := &state.TerraformState{
		Resources: []state.Resource{
			{
				ID:   "test-resource",
				Type: "aws_instance",
				Name: "test-resource",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
		},
	}

	for _, driftType := range driftTypes {
		t.Run(string(driftType), func(t *testing.T) {
			result, err := simulator.SimulateDrift(ctx, driftType, "test-resource", mockState)

			assert.NoError(t, err, "Drift type %s should be supported", driftType)
			assert.NotNil(t, result, "Result should not be nil for drift type %s", driftType)
			assert.Equal(t, driftType, result.DriftType, "Result drift type should match input")
			assert.Equal(t, "aws", result.Provider, "Provider should be aws")
		})
	}
}

func TestAWSSimulator_ErrorHandling(t *testing.T) {
	simulator := simulation.NewAWSSimulator()
	ctx := context.Background()

	// Test with nil state
	result, err := simulator.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", nil)
	assert.Error(t, err)
	assert.Nil(t, result)

	// Test with empty state
	emptyState := &state.TerraformState{
		Resources: []state.Resource{},
	}
	result, err = simulator.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", emptyState)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found in state")
}

func TestAWSSimulator_RollbackData(t *testing.T) {
	simulator := simulation.NewAWSSimulator()
	ctx := context.Background()

	mockState := &state.TerraformState{
		Resources: []state.Resource{
			{
				ID:   "test-instance",
				Type: "aws_instance",
				Name: "test-instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
		},
	}

	result, err := simulator.SimulateDrift(ctx, simulation.DriftTypeResourceDeletion, "test-instance", mockState)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.RollbackData)

	// Verify rollback data structure
	assert.Equal(t, "test-instance", result.RollbackData.ResourceID)
	assert.Equal(t, "aws_instance", result.RollbackData.ResourceType)
	assert.NotNil(t, result.RollbackData.OriginalData)
	assert.Contains(t, result.RollbackData.OriginalData, "instance_id")
	assert.Contains(t, result.RollbackData.OriginalData, "resource_type")
}
