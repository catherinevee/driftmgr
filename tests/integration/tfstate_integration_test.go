package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	awsprovider "github.com/catherinevee/driftmgr/internal/cloud/aws"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerraformStateIntegration tests the complete integration with Terraform state files
func TestTerraformStateIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create temp directory for test
	tempDir := t.TempDir()

	t.Run("Parse and Compare Real Terraform State", func(t *testing.T) {
		// Step 1: Create a realistic Terraform state file
		stateFile := filepath.Join(tempDir, "terraform.tfstate")
		tfState := createRealisticTerraformState()

		data, err := json.MarshalIndent(tfState, "", "  ")
		require.NoError(t, err)

		err = os.WriteFile(stateFile, data, 0644)
		require.NoError(t, err)

		// Step 2: Load and parse the state file
		t.Log("Loading Terraform state file...")
		loader := state.NewStateLoader(stateFile)
		loadedState, err := loader.Load()
		require.NoError(t, err)
		assert.NotNil(t, loadedState)

		t.Logf("Loaded state with %d resources", len(loadedState.Resources))
		assert.Equal(t, 4, loadedState.Version)
		assert.Equal(t, "1.5.0", loadedState.TerraformVersion)
		assert.Len(t, loadedState.Resources, 5)

		// Step 3: Convert state resources to models.Resource
		stateResources := convertStateToResources(loadedState)
		assert.Len(t, stateResources, 5)

		// Step 4: Discover actual cloud resources
		t.Log("Discovering actual cloud resources...")
		discoverer := discovery.NewCloudDiscoverer()

		awsProvider, err := awsprovider.NewAWSProvider()
		if err == nil {
			discoverer.AddProvider("aws", awsProvider)

			config := discovery.Config{
				Regions: []string{"us-east-1"},
			}

			actualResources, err := discoverer.DiscoverProvider(ctx, "aws", config)
			if err == nil {
				t.Logf("Discovered %d actual AWS resources", len(actualResources))

				// Step 5: Detect drift between state and actual
				t.Log("Detecting drift between state and actual resources...")
				driftItems := detectDrift(stateResources, actualResources)

				t.Logf("Found %d drift items", len(driftItems))

				// Analyze drift by type
				driftByType := make(map[string]int)
				for _, item := range driftItems {
					driftByType[string(item.DriftType)]++
				}

				t.Log("Drift summary:")
				for driftType, count := range driftByType {
					t.Logf("  %s: %d", driftType, count)
				}
			} else {
				t.Skipf("AWS discovery skipped: %v", err)
			}
		} else {
			t.Skipf("AWS provider not available: %v", err)
		}

		// Step 6: Test state file analysis
		t.Log("Analyzing state file structure...")
		analysis := analyzeStateFile(loadedState)
		assert.NotNil(t, analysis)

		// Verify resource type distribution
		assert.Equal(t, 2, analysis.ResourceTypes["aws_instance"])
		assert.Equal(t, 1, analysis.ResourceTypes["aws_s3_bucket"])
		assert.Equal(t, 1, analysis.ResourceTypes["aws_vpc"])
		assert.Equal(t, 1, analysis.ResourceTypes["aws_security_group"])

		// Verify provider distribution
		assert.Equal(t, 5, analysis.ProviderCounts["aws"])

		t.Log("State file analysis completed successfully")
	})

	t.Run("Test State File with Multiple Providers", func(t *testing.T) {
		// Create state with multiple providers
		multiProviderState := createMultiProviderState()
		stateFile := filepath.Join(tempDir, "multi-provider.tfstate")

		data, err := json.MarshalIndent(multiProviderState, "", "  ")
		require.NoError(t, err)

		err = os.WriteFile(stateFile, data, 0644)
		require.NoError(t, err)

		// Load and verify
		loader := state.NewStateLoader(stateFile)
		loadedState, err := loader.Load()
		require.NoError(t, err)

		// Count resources by provider
		providerCounts := make(map[string]int)
		for _, resource := range loadedState.Resources {
			provider := extractProviderFromResource(resource.Provider)
			providerCounts[provider]++
		}

		assert.Equal(t, 2, providerCounts["aws"])
		assert.Equal(t, 2, providerCounts["azurerm"])
		assert.Equal(t, 1, providerCounts["google"])

		t.Log("Multi-provider state file parsed successfully")
	})

	t.Run("Test Drift Detection with State File", func(t *testing.T) {
		// Create a state file with known resources
		stateFile := filepath.Join(tempDir, "drift-test.tfstate")
		tfState := createDriftTestState()

		data, err := json.MarshalIndent(tfState, "", "  ")
		require.NoError(t, err)

		err = os.WriteFile(stateFile, data, 0644)
		require.NoError(t, err)

		// Load state
		loader := state.NewStateLoader(stateFile)
		loadedState, err := loader.Load()
		require.NoError(t, err)

		// Convert to resources
		stateResources := convertStateToResources(loadedState)

		// Create mock actual resources with differences
		actualResources := createMockActualResources()

		// Detect drift
		driftAnalysis := performDriftAnalysis(stateResources, actualResources)

		assert.NotNil(t, driftAnalysis)
		assert.Greater(t, driftAnalysis.TotalDrift, 0)

		t.Logf("Drift analysis: %d added, %d modified, %d deleted",
			driftAnalysis.AddedResources,
			driftAnalysis.ModifiedResources,
			driftAnalysis.DeletedResources)
	})
}

// Helper functions

func createRealisticTerraformState() *state.TerraformState {
	return &state.TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           42,
		Lineage:          "d7c4b6a1-5b1e-4b7f-8c3d-9e2f1a3b4c5d",
		Resources: []state.StateResource{
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "web_server",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":            "i-1234567890abcdef0",
							"ami":           "ami-0c55b159cbfafe1f0",
							"instance_type": "t3.medium",
							"tags": map[string]interface{}{
								"Name":        "WebServer",
								"Environment": "production",
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "app_server",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":            "i-0987654321fedcba0",
							"ami":           "ami-0c55b159cbfafe1f0",
							"instance_type": "t3.large",
							"tags": map[string]interface{}{
								"Name":        "AppServer",
								"Environment": "production",
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_s3_bucket",
				Name:     "data",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id":     "my-data-bucket-12345",
							"bucket": "my-data-bucket-12345",
							"region": "us-east-1",
							"tags": map[string]interface{}{
								"Environment": "production",
								"Purpose":     "data-storage",
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_vpc",
				Name:     "main",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":         "vpc-0a1b2c3d4e5f67890",
							"cidr_block": "10.0.0.0/16",
							"tags": map[string]interface{}{
								"Name":        "main-vpc",
								"Environment": "production",
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_security_group",
				Name:     "web",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":          "sg-0123456789abcdef0",
							"name":        "web-security-group",
							"description": "Security group for web servers",
							"vpc_id":      "vpc-0a1b2c3d4e5f67890",
						},
					},
				},
			},
		},
	}
}

func createMultiProviderState() *state.TerraformState {
	return &state.TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           10,
		Lineage:          "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Resources: []state.StateResource{
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "web",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id": "i-aws123",
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_s3_bucket",
				Name:     "storage",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id": "my-aws-bucket",
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "azurerm_resource_group",
				Name:     "main",
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id": "/subscriptions/12345/resourceGroups/main-rg",
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "azurerm_virtual_machine",
				Name:     "vm",
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id": "/subscriptions/12345/resourceGroups/main-rg/providers/Microsoft.Compute/virtualMachines/vm1",
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "google_compute_instance",
				Name:     "gcp_vm",
				Provider: "provider[\"registry.terraform.io/hashicorp/google\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 6,
						Attributes: map[string]interface{}{
							"id": "projects/my-project/zones/us-central1-a/instances/gcp-vm",
						},
					},
				},
			},
		},
	}
}

func createDriftTestState() *state.TerraformState {
	return &state.TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           5,
		Lineage:          "test-drift-detection",
		Resources: []state.StateResource{
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "test",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":            "i-drift-test",
							"instance_type": "t2.micro", // Will be different in actual
							"tags": map[string]interface{}{
								"Name": "DriftTest",
							},
						},
					},
				},
			},
		},
	}
}

func convertStateToResources(tfState *state.TerraformState) []models.Resource {
	var resources []models.Resource

	for _, stateResource := range tfState.Resources {
		for _, instance := range stateResource.Instances {
			resource := models.Resource{
				Type:       stateResource.Type,
				Name:       stateResource.Name,
				Provider:   extractProviderFromResource(stateResource.Provider),
				Attributes: instance.Attributes,
				Metadata: map[string]string{
					"mode":           stateResource.Mode,
					"schema_version": fmt.Sprintf("%d", instance.SchemaVersion),
				},
			}

			// Extract ID if present
			if id, ok := instance.Attributes["id"].(string); ok {
				resource.ID = id
			}

			// Extract tags if present
			if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
				tagMap := make(map[string]string)
				for k, v := range tags {
					if str, ok := v.(string); ok {
						tagMap[k] = str
					}
				}
				resource.Tags = tagMap
			}

			resources = append(resources, resource)
		}
	}

	return resources
}

func extractProviderFromResource(provider string) string {
	// Extract provider name from format like "provider[\"registry.terraform.io/hashicorp/aws\"]"
	if len(provider) > 0 {
		start := strings.LastIndex(provider, "/")
		end := strings.LastIndex(provider, "\"")
		if start > 0 && end > start {
			return provider[start+1 : end]
		}
	}
	return "unknown"
}

func detectDrift(stateResources, actualResources []models.Resource) []models.DriftItem {
	var driftItems []models.DriftItem

	// Create maps for comparison
	stateMap := make(map[string]models.Resource)
	for _, r := range stateResources {
		stateMap[r.ID] = r
	}

	actualMap := make(map[string]models.Resource)
	for _, r := range actualResources {
		actualMap[r.ID] = r
	}

	// Check for deleted resources (in state but not actual)
	for id, stateResource := range stateMap {
		if _, exists := actualMap[id]; !exists {
			driftItems = append(driftItems, models.DriftItem{
				ResourceID:   id,
				ResourceType: stateResource.Type,
				DriftType:    "deleted",
				Severity:     "high",
				Description:  "Resource exists in state but not in actual infrastructure",
			})
		}
	}

	// Check for unmanaged resources (in actual but not state)
	for id, actualResource := range actualMap {
		if _, exists := stateMap[id]; !exists {
			driftItems = append(driftItems, models.DriftItem{
				ResourceID:   id,
				ResourceType: actualResource.Type,
				DriftType:    "unmanaged",
				Severity:     "medium",
				Description:  "Resource exists in infrastructure but not in state",
			})
		}
	}

	return driftItems
}

type StateAnalysis struct {
	ResourceTypes  map[string]int
	ProviderCounts map[string]int
	TotalResources int
	HasDataSources bool
}

func analyzeStateFile(tfState *state.TerraformState) *StateAnalysis {
	analysis := &StateAnalysis{
		ResourceTypes:  make(map[string]int),
		ProviderCounts: make(map[string]int),
		TotalResources: len(tfState.Resources),
	}

	for _, resource := range tfState.Resources {
		analysis.ResourceTypes[resource.Type]++

		provider := extractProviderFromResource(resource.Provider)
		analysis.ProviderCounts[provider]++

		if resource.Mode == "data" {
			analysis.HasDataSources = true
		}
	}

	return analysis
}

func createMockActualResources() []models.Resource {
	return []models.Resource{
		{
			ID:       "i-drift-test",
			Type:     "aws_instance",
			Name:     "test",
			Provider: "aws",
			Attributes: map[string]interface{}{
				"instance_type": "t3.micro", // Changed from t2.micro
				"tags": map[string]interface{}{
					"Name":        "DriftTest",
					"Environment": "staging", // Added tag
				},
			},
		},
		{
			ID:       "i-unmanaged",
			Type:     "aws_instance",
			Name:     "unmanaged",
			Provider: "aws",
			Attributes: map[string]interface{}{
				"instance_type": "t2.nano",
			},
		},
	}
}

type DriftAnalysis struct {
	TotalDrift        int
	AddedResources    int
	ModifiedResources int
	DeletedResources  int
}

func performDriftAnalysis(stateResources, actualResources []models.Resource) *DriftAnalysis {
	analysis := &DriftAnalysis{}

	driftItems := detectDrift(stateResources, actualResources)
	analysis.TotalDrift = len(driftItems)

	for _, item := range driftItems {
		switch item.DriftType {
		case "unmanaged":
			analysis.AddedResources++
		case "modified":
			analysis.ModifiedResources++
		case "deleted":
			analysis.DeletedResources++
		}
	}

	return analysis
}
