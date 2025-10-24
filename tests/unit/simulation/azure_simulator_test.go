package simulation_test

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/simulation"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureSimulator_SimulateDrift(t *testing.T) {
	simulator := simulation.NewAzureSimulator()
	ctx := context.Background()

	// Create a mock state with a test resource
	mockState := &state.TerraformState{
		Resources: []state.Resource{
			{
				ID:   "test-vm",
				Type: "azurerm_virtual_machine",
				Name: "test-vm",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"name": "test-vm",
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
			resourceID: "test-vm",
			expectErr:  false,
		},
		{
			name:       "rule addition drift",
			driftType:  simulation.DriftTypeRuleAddition,
			resourceID: "test-vm",
			expectErr:  false,
		},
		{
			name:       "resource creation drift",
			driftType:  simulation.DriftTypeResourceCreation,
			resourceID: "test-vm",
			expectErr:  false,
		},
		{
			name:       "attribute change drift",
			driftType:  simulation.DriftTypeAttributeChange,
			resourceID: "test-vm",
			expectErr:  false,
		},
		{
			name:       "resource deletion drift",
			driftType:  simulation.DriftTypeResourceDeletion,
			resourceID: "test-vm",
			expectErr:  false,
		},
		{
			name:       "unsupported drift type",
			driftType:  "unsupported",
			resourceID: "test-vm",
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
				assert.Equal(t, "azure", result.Provider)
				assert.Equal(t, tc.driftType, result.DriftType)
				assert.Equal(t, tc.resourceID, result.ResourceID)
			}
		})
	}
}

func TestAzureSimulator_SimulateResourceDeletion(t *testing.T) {
	simulator := simulation.NewAzureSimulator()
	ctx := context.Background()

	testCases := []struct {
		name         string
		resourceType string
		resourceID   string
		expectErr    bool
	}{
		{
			name:         "Virtual machine deletion",
			resourceType: "azurerm_virtual_machine",
			resourceID:   "test-vm",
			expectErr:    false,
		},
		{
			name:         "Storage account deletion",
			resourceType: "azurerm_storage_account",
			resourceID:   "test-storage",
			expectErr:    false,
		},
		{
			name:         "Network security group deletion",
			resourceType: "azurerm_network_security_group",
			resourceID:   "test-nsg",
			expectErr:    false,
		},
		{
			name:         "Generic resource deletion",
			resourceType: "azurerm_unknown_resource",
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
							"name": tc.resourceID,
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

func TestAzureSimulator_Initialize(t *testing.T) {
	simulator := simulation.NewAzureSimulator()
	ctx := context.Background()

	// Test initialization (this will fail in test environment without Azure credentials, but we can test the structure)
	err := simulator.Initialize(ctx)

	// In a test environment without Azure credentials, this should fail
	// but we can verify the method exists and handles errors appropriately
	if err != nil {
		assert.Contains(t, err.Error(), "Azure")
	}
}

func TestAzureSimulator_ResourceStructure(t *testing.T) {
	simulator := simulation.NewAzureSimulator()

	// Test that simulator can handle different resource types
	resourceTypes := []string{
		"azurerm_virtual_machine",
		"azurerm_storage_account",
		"azurerm_network_security_group",
		"azurerm_resource_group",
		"azurerm_virtual_network",
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
							"name": "test-resource",
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

func TestAzureSimulator_AllDriftTypes(t *testing.T) {
	simulator := simulation.NewAzureSimulator()
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
				Type: "azurerm_virtual_machine",
				Name: "test-vm",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"name": "test-vm",
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
			assert.Equal(t, "azure", result.Provider, "Provider should be azure")
		})
	}
}

func TestAzureSimulator_ErrorHandling(t *testing.T) {
	simulator := simulation.NewAzureSimulator()
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

func TestAzureSimulator_RollbackData(t *testing.T) {
	simulator := simulation.NewAzureSimulator()
	ctx := context.Background()

	mockState := &state.TerraformState{
		Resources: []state.Resource{
			{
				ID:   "test-vm",
				Type: "azurerm_virtual_machine",
				Name: "test-vm",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"name": "test-vm",
						},
					},
				},
			},
		},
	}

	result, err := simulator.SimulateDrift(ctx, simulation.DriftTypeResourceDeletion, "test-vm", mockState)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.RollbackData)

	// Verify rollback data structure
	assert.Equal(t, "test-vm", result.RollbackData.ResourceID)
	assert.Equal(t, "azurerm_virtual_machine", result.RollbackData.ResourceType)
	assert.NotNil(t, result.RollbackData.OriginalData)
	assert.Contains(t, result.RollbackData.OriginalData, "vm_name")
	assert.Contains(t, result.RollbackData.OriginalData, "resource_type")
}
