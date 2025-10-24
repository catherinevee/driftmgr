package simulation_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/simulation"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimulationSystem_Integration(t *testing.T) {
	ctx := context.Background()

	// Create mock state with resources from all providers
	mockState := &state.TerraformState{
		Resources: []state.Resource{
			// AWS resources
			{
				ID:   "aws-instance",
				Type: "aws_instance",
				Name: "aws-instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
			{
				ID:   "aws-bucket",
				Type: "aws_s3_bucket",
				Name: "aws-bucket",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"bucket": "test-bucket-name",
						},
					},
				},
			},
			// Azure resources
			{
				ID:   "azure-vm",
				Type: "azurerm_virtual_machine",
				Name: "azure-vm",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"name": "azure-vm",
						},
					},
				},
			},
			{
				ID:   "azure-storage",
				Type: "azurerm_storage_account",
				Name: "azure-storage",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"name": "azure-storage",
						},
					},
				},
			},
			// GCP resources
			{
				ID:   "gcp-instance",
				Type: "google_compute_instance",
				Name: "gcp-instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"name": "gcp-instance",
						},
					},
				},
			},
			{
				ID:   "gcp-bucket",
				Type: "google_storage_bucket",
				Name: "gcp-bucket",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"name": "gcp-bucket",
						},
					},
				},
			},
		},
	}

	// Test all simulators with all drift types
	simulators := map[string]simulation.Simulator{
		"aws":   simulation.NewAWSSimulator(),
		"azure": simulation.NewAzureSimulator(),
		"gcp":   simulation.NewGCPSimulator(),
	}

	driftTypes := []simulation.DriftType{
		simulation.DriftTypeTagChange,
		simulation.DriftTypeRuleAddition,
		simulation.DriftTypeResourceCreation,
		simulation.DriftTypeAttributeChange,
		simulation.DriftTypeResourceDeletion,
	}

	// Test each simulator with each drift type
	for providerName, simulator := range simulators {
		t.Run(providerName, func(t *testing.T) {
			for _, driftType := range driftTypes {
				t.Run(string(driftType), func(t *testing.T) {
					// Find a resource for this provider
					var testResourceID string
					for _, resource := range mockState.Resources {
						if (providerName == "aws" && (resource.Type == "aws_instance" || resource.Type == "aws_s3_bucket")) ||
							(providerName == "azure" && (resource.Type == "azurerm_virtual_machine" || resource.Type == "azurerm_storage_account")) ||
							(providerName == "gcp" && (resource.Type == "google_compute_instance" || resource.Type == "google_storage_bucket")) {
							testResourceID = resource.ID
							break
						}
					}

					require.NotEmpty(t, testResourceID, "No test resource found for provider %s", providerName)

					result, err := simulator.SimulateDrift(ctx, driftType, testResourceID, mockState)

					assert.NoError(t, err, "Simulation should succeed for %s with drift type %s", providerName, driftType)
					assert.NotNil(t, result, "Result should not be nil")
					assert.Equal(t, providerName, result.Provider, "Provider should match")
					assert.Equal(t, driftType, result.DriftType, "Drift type should match")
					assert.Equal(t, testResourceID, result.ResourceID, "Resource ID should match")
					assert.True(t, result.Success, "Simulation should be successful")
					assert.NotNil(t, result.Changes, "Changes should not be nil")
					assert.NotEmpty(t, result.CostEstimate, "Cost estimate should not be empty")
				})
			}
		})
	}
}

func TestSimulationSystem_ResourceDeletionIntegration(t *testing.T) {
	ctx := context.Background()

	// Test resource deletion across all providers
	testCases := []struct {
		provider     string
		resourceType string
		resourceID   string
	}{
		{
			provider:     "aws",
			resourceType: "aws_instance",
			resourceID:   "aws-instance",
		},
		{
			provider:     "aws",
			resourceType: "aws_s3_bucket",
			resourceID:   "aws-bucket",
		},
		{
			provider:     "azure",
			resourceType: "azurerm_virtual_machine",
			resourceID:   "azure-vm",
		},
		{
			provider:     "azure",
			resourceType: "azurerm_storage_account",
			resourceID:   "azure-storage",
		},
		{
			provider:     "gcp",
			resourceType: "google_compute_instance",
			resourceID:   "gcp-instance",
		},
		{
			provider:     "gcp",
			resourceType: "google_storage_bucket",
			resourceID:   "gcp-bucket",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.provider+"_"+tc.resourceType, func(t *testing.T) {
			// Create simulator for this provider
			var simulator simulation.Simulator
			switch tc.provider {
			case "aws":
				simulator = simulation.NewAWSSimulator()
			case "azure":
				simulator = simulation.NewAzureSimulator()
			case "gcp":
				simulator = simulation.NewGCPSimulator()
			default:
				t.Fatalf("Unknown provider: %s", tc.provider)
			}

			// Create state with the test resource
			mockState := &state.TerraformState{
				Resources: []state.Resource{
					{
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
					},
				},
			}

			// Test resource deletion
			result, err := simulator.SimulateDrift(ctx, simulation.DriftTypeResourceDeletion, tc.resourceID, mockState)

			assert.NoError(t, err, "Resource deletion simulation should succeed")
			assert.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, tc.provider, result.Provider, "Provider should match")
			assert.Equal(t, simulation.DriftTypeResourceDeletion, result.DriftType, "Drift type should be resource deletion")
			assert.Equal(t, tc.resourceID, result.ResourceID, "Resource ID should match")
			assert.True(t, result.Success, "Simulation should be successful")
			assert.Contains(t, result.Changes, "deletion_simulated", "Should contain deletion simulation flag")
			assert.True(t, result.Changes["deletion_simulated"].(bool), "Deletion should be simulated")
			assert.NotNil(t, result.RollbackData, "Rollback data should be present")
			assert.Len(t, result.DetectedDrift, 1, "Should detect one drift item")
			assert.Equal(t, "resource_deletion", result.DetectedDrift[0].DriftType, "Detected drift should be resource deletion")
		})
	}
}

func TestSimulationSystem_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	simulators := map[string]simulation.Simulator{
		"aws":   simulation.NewAWSSimulator(),
		"azure": simulation.NewAzureSimulator(),
		"gcp":   simulation.NewGCPSimulator(),
	}

	for providerName, simulator := range simulators {
		t.Run(providerName+"_error_handling", func(t *testing.T) {
			// Test with nil state
			result, err := simulator.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", nil)
			assert.Error(t, err, "Should return error for nil state")
			assert.Nil(t, result, "Result should be nil for nil state")

			// Test with empty state
			emptyState := &state.TerraformState{
				Resources: []state.Resource{},
			}
			result, err = simulator.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", emptyState)
			assert.Error(t, err, "Should return error for empty state")
			assert.Nil(t, result, "Result should be nil for empty state")
			assert.Contains(t, err.Error(), "not found in state", "Error should mention resource not found")

			// Test with unsupported drift type
			mockState := &state.TerraformState{
				Resources: []state.Resource{
					{
						ID:   "test-resource",
						Type: "test_type",
						Name: "test-resource",
						Instances: []state.Instance{
							{
								Attributes: map[string]interface{}{
									"name": "test-resource",
								},
							},
						},
					},
				},
			}
			result, err = simulator.SimulateDrift(ctx, "unsupported-drift-type", "test-resource", mockState)
			assert.Error(t, err, "Should return error for unsupported drift type")
			assert.Nil(t, result, "Result should be nil for unsupported drift type")
		})
	}
}

func TestSimulationSystem_Performance(t *testing.T) {
	ctx := context.Background()

	// Create a large state with many resources
	resources := make([]state.Resource, 100)
	for i := 0; i < 100; i++ {
		resources[i] = state.Resource{
			ID:   fmt.Sprintf("resource-%d", i),
			Type: "aws_instance",
			Name: fmt.Sprintf("resource-%d", i),
			Instances: []state.Instance{
				{
					Attributes: map[string]interface{}{
						"id": fmt.Sprintf("i-%d", i),
					},
				},
			},
		}
	}

	mockState := &state.TerraformState{
		Resources: resources,
	}

	simulator := simulation.NewAWSSimulator()

	// Test performance with multiple simulations
	start := time.Now()
	for i := 0; i < 10; i++ {
		result, err := simulator.SimulateDrift(ctx, simulation.DriftTypeTagChange, fmt.Sprintf("resource-%d", i), mockState)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
	duration := time.Since(start)

	// Performance should be reasonable (less than 1 second for 10 simulations)
	assert.Less(t, duration, time.Second, "Simulations should complete within 1 second")
}
