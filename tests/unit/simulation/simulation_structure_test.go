package simulation_test

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/simulation"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestSimulationSystem_Structure(t *testing.T) {
	// Test that all simulators can be created
	awsSimulator := simulation.NewAWSSimulator()
	azureSimulator := simulation.NewAzureSimulator()
	gcpSimulator := simulation.NewGCPSimulator()

	assert.NotNil(t, awsSimulator, "AWS simulator should be created")
	assert.NotNil(t, azureSimulator, "Azure simulator should be created")
	assert.NotNil(t, gcpSimulator, "GCP simulator should be created")
}

func TestSimulationSystem_DriftTypes(t *testing.T) {
	// Test that all drift types are defined
	expectedDriftTypes := []simulation.DriftType{
		simulation.DriftTypeTagChange,
		simulation.DriftTypeRuleAddition,
		simulation.DriftTypeResourceCreation,
		simulation.DriftTypeAttributeChange,
		simulation.DriftTypeResourceDeletion,
		simulation.DriftTypeRandom,
	}

	for _, driftType := range expectedDriftTypes {
		assert.NotEmpty(t, string(driftType), "Drift type should not be empty: %s", driftType)
	}
}

func TestSimulationSystem_ResourceDeletionSupport(t *testing.T) {
	// Test that all simulators support resource deletion drift type
	awsSimulator := simulation.NewAWSSimulator()
	azureSimulator := simulation.NewAzureSimulator()
	gcpSimulator := simulation.NewGCPSimulator()

	simulators := map[string]interface{}{
		"aws":   awsSimulator,
		"azure": azureSimulator,
		"gcp":   gcpSimulator,
	}

	ctx := context.Background()

	for providerName, simulator := range simulators {
		t.Run(providerName, func(t *testing.T) {
			// Create a minimal test state
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

			// Test that the simulator can handle resource deletion drift type
			// This should not panic even if it fails due to missing credentials
			var result *simulation.SimulationResult
			var err error

			switch s := simulator.(type) {
			case *simulation.AWSSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeResourceDeletion, "test-resource", mockState)
			case *simulation.AzureSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeResourceDeletion, "test-resource", mockState)
			case *simulation.GCPSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeResourceDeletion, "test-resource", mockState)
			}

			// The result should either be successful or fail gracefully
			if err != nil {
				// If it fails, it should be a known error, not a panic
				assert.NotContains(t, err.Error(), "panic", "Should not panic")
				assert.NotContains(t, err.Error(), "nil pointer", "Should not have nil pointer dereference")
			} else {
				assert.NotNil(t, result, "Result should not be nil if no error")
				assert.Equal(t, simulation.DriftTypeResourceDeletion, result.DriftType, "Drift type should match")
			}
		})
	}
}

func TestSimulationSystem_AllDriftTypesSupported(t *testing.T) {
	// Test that all simulators support all drift types
	awsSimulator := simulation.NewAWSSimulator()
	azureSimulator := simulation.NewAzureSimulator()
	gcpSimulator := simulation.NewGCPSimulator()

	simulators := map[string]interface{}{
		"aws":   awsSimulator,
		"azure": azureSimulator,
		"gcp":   gcpSimulator,
	}

	driftTypes := []simulation.DriftType{
		simulation.DriftTypeTagChange,
		simulation.DriftTypeRuleAddition,
		simulation.DriftTypeResourceCreation,
		simulation.DriftTypeAttributeChange,
		simulation.DriftTypeResourceDeletion,
	}

	ctx := context.Background()

	for providerName, simulator := range simulators {
		t.Run(providerName, func(t *testing.T) {
			for _, driftType := range driftTypes {
				t.Run(string(driftType), func(t *testing.T) {
					// Create a minimal test state
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

					// Test that the simulator can handle this drift type
					var result *simulation.SimulationResult
					var err error

					switch s := simulator.(type) {
					case *simulation.AWSSimulator:
						result, err = s.SimulateDrift(ctx, driftType, "test-resource", mockState)
					case *simulation.AzureSimulator:
						result, err = s.SimulateDrift(ctx, driftType, "test-resource", mockState)
					case *simulation.GCPSimulator:
						result, err = s.SimulateDrift(ctx, driftType, "test-resource", mockState)
					}

					// The result should either be successful or fail gracefully
					if err != nil {
						// If it fails, it should be a known error, not a panic
						assert.NotContains(t, err.Error(), "panic", "Should not panic for drift type %s", driftType)
						assert.NotContains(t, err.Error(), "nil pointer", "Should not have nil pointer dereference for drift type %s", driftType)
						// Should not be "not implemented" error anymore
						assert.NotContains(t, err.Error(), "not implemented", "Drift type %s should be implemented", driftType)
					} else {
						assert.NotNil(t, result, "Result should not be nil if no error for drift type %s", driftType)
						assert.Equal(t, driftType, result.DriftType, "Drift type should match for %s", driftType)
					}
				})
			}
		})
	}
}

func TestSimulationSystem_ErrorHandling(t *testing.T) {
	// Test error handling for invalid inputs
	awsSimulator := simulation.NewAWSSimulator()
	azureSimulator := simulation.NewAzureSimulator()
	gcpSimulator := simulation.NewGCPSimulator()

	simulators := map[string]interface{}{
		"aws":   awsSimulator,
		"azure": azureSimulator,
		"gcp":   gcpSimulator,
	}

	ctx := context.Background()

	for providerName, simulator := range simulators {
		t.Run(providerName, func(t *testing.T) {
			// Test with nil state
			var result *simulation.SimulationResult
			var err error

			switch s := simulator.(type) {
			case *simulation.AWSSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", nil)
			case *simulation.AzureSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", nil)
			case *simulation.GCPSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", nil)
			}
			assert.Error(t, err, "Should return error for nil state")
			assert.Nil(t, result, "Result should be nil for nil state")

			// Test with empty state
			emptyState := &state.TerraformState{
				Resources: []state.Resource{},
			}
			switch s := simulator.(type) {
			case *simulation.AWSSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", emptyState)
			case *simulation.AzureSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", emptyState)
			case *simulation.GCPSimulator:
				result, err = s.SimulateDrift(ctx, simulation.DriftTypeTagChange, "test-resource", emptyState)
			}
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
			switch s := simulator.(type) {
			case *simulation.AWSSimulator:
				result, err = s.SimulateDrift(ctx, "unsupported-drift-type", "test-resource", mockState)
			case *simulation.AzureSimulator:
				result, err = s.SimulateDrift(ctx, "unsupported-drift-type", "test-resource", mockState)
			case *simulation.GCPSimulator:
				result, err = s.SimulateDrift(ctx, "unsupported-drift-type", "test-resource", mockState)
			}
			assert.Error(t, err, "Should return error for unsupported drift type")
			assert.Nil(t, result, "Result should be nil for unsupported drift type")
			assert.Contains(t, err.Error(), "not implemented", "Error should mention not implemented")
		})
	}
}
