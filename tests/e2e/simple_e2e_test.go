package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/config"
	"github.com/catherinevee/driftmgr/internal/discovery"
	awsprovider "github.com/catherinevee/driftmgr/internal/providers/aws"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndWorkflow tests the complete workflow
func TestEndToEndWorkflow(t *testing.T) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create temp directory for test artifacts
	tempDir := t.TempDir()

	// Initialize configuration
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			EnableCaching:        true,
			CacheTTL:             5 * time.Minute,
			ConcurrencyLimit:     5,
			MaxConcurrentRegions: 3,
			Regions:              []string{"us-east-1", "us-west-2"},
			AWSProfile:           "default",
		},
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Host: filepath.Join(tempDir, "test.db"),
		},
	}

	t.Run("Complete Discovery and Drift Detection Workflow", func(t *testing.T) {
		// Step 1: Initialize cloud discoverer
		discoverer := discovery.NewCloudDiscoverer()

		// Add AWS provider
		awsProvider, err := awsprovider.NewAWSProvider()
		if err != nil {
			t.Skipf("AWS provider not available: %v", err)
			return
		}
		discoverer.AddProvider("aws", awsProvider)

		// Step 2: Perform discovery
		t.Log("Step 2: Performing resource discovery...")
		discoveryConfig := discovery.Config{
			Regions: cfg.Discovery.Regions,
		}

		resources, err := discoverer.DiscoverProvider(ctx, "aws", discoveryConfig)
		if err != nil {
			t.Skipf("AWS discovery failed (likely missing credentials): %v", err)
			return
		}

		require.NotEmpty(t, resources, "Should discover at least some resources")
		t.Logf("Discovered %d resources", len(resources))

		// Step 3: Save state file
		t.Log("Step 3: Saving state file...")
		stateFile := filepath.Join(tempDir, "terraform.tfstate")
		state := createMockStateFile(resources)
		err = saveStateFile(stateFile, state)
		require.NoError(t, err)

		// Step 4: Detect drift
		t.Log("Step 4: Detecting drift...")
		// Simulate drift detection by comparing resources
		driftItems := detectDriftSimple(state.Resources, resources)

		t.Logf("Detected %d drift items", len(driftItems))

		// Step 6: Analyze drift patterns
		if len(driftItems) > 0 {
			t.Log("Step 6: Analyzing drift patterns...")

			// Group by type
			driftByType := make(map[string]int)
			for _, item := range driftItems {
				driftByType[string(item.DriftType)]++
			}

			t.Log("Drift summary by type:")
			for driftType, count := range driftByType {
				t.Logf("  %s: %d items", driftType, count)
			}
		}

		// Step 7: Generate remediation plan (simulated)
		t.Log("Step 7: Generating remediation plan...")
		if len(driftItems) > 0 {
			plan := generateRemediationPlan(driftItems)
			assert.NotNil(t, plan)
			t.Logf("Remediation plan includes %d actions", len(plan.Actions))
		}

		// Step 8: Export results
		t.Log("Step 8: Exporting results...")
		exportFile := filepath.Join(tempDir, "drift-report.json")
		err = exportDriftReport(exportFile, driftItems)
		require.NoError(t, err)

		// Verify export file exists
		_, err = os.Stat(exportFile)
		require.NoError(t, err)

		t.Log("End-to-end workflow completed successfully!")
	})
}

// Helper functions

func detectDriftSimple(stateResources []StateResource, actualResources []models.Resource) []models.DriftItem {
	var driftItems []models.DriftItem

	// Simple comparison - if counts don't match, there's drift
	if len(stateResources) != len(actualResources) {
		driftItems = append(driftItems, models.DriftItem{
			ResourceID:   "summary",
			ResourceType: "count_mismatch",
			DriftType:    "added",
			Severity:     "medium",
			Description:  "Resource count mismatch between state and actual",
		})
	}

	// Create a map of actual resources for comparison
	actualMap := make(map[string]models.Resource)
	for _, resource := range actualResources {
		actualMap[resource.ID] = resource
	}

	// Check for missing resources
	for _, stateResource := range stateResources {
		for _, instance := range stateResource.Instances {
			if _, exists := actualMap[instance.ID]; !exists {
				driftItems = append(driftItems, models.DriftItem{
					ResourceID:   instance.ID,
					ResourceType: stateResource.Type,
					DriftType:    "deleted",
					Severity:     "high",
					Description:  "Resource exists in state but not in actual infrastructure",
				})
			}
		}
	}

	return driftItems
}

func createMockStateFile(resources []models.Resource) *StateFile {
	state := &StateFile{
		Version:   4,
		Resources: make([]StateResource, 0, len(resources)),
	}

	for _, resource := range resources {
		stateResource := StateResource{
			Type:     resource.Type,
			Name:     resource.Name,
			Provider: resource.Provider,
			Instances: []StateInstance{
				{
					ID:         resource.ID,
					Attributes: resource.Attributes,
				},
			},
		}
		state.Resources = append(state.Resources, stateResource)
	}

	return state
}

func saveStateFile(path string, state *StateFile) error {
	// In a real implementation, this would serialize to JSON
	// For testing, we just create the file
	return os.WriteFile(path, []byte("{}"), 0644)
}

func generateRemediationPlan(driftItems []models.DriftItem) *RemediationPlan {
	plan := &RemediationPlan{
		Actions:   make([]RemediationAction, 0, len(driftItems)),
		CreatedAt: time.Now(),
	}

	for _, item := range driftItems {
		action := RemediationAction{
			Type:        string(item.DriftType),
			ResourceID:  item.ResourceID,
			Description: "Fix drift for " + item.ResourceID,
		}
		plan.Actions = append(plan.Actions, action)
	}

	return plan
}

func exportDriftReport(path string, driftItems []models.DriftItem) error {
	// In a real implementation, this would serialize to JSON
	// For testing, we just create the file
	return os.WriteFile(path, []byte("{}"), 0644)
}

// Test structures

type StateFile struct {
	Version   int             `json:"version"`
	Resources []StateResource `json:"resources"`
}

type StateResource struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Provider  string          `json:"provider"`
	Instances []StateInstance `json:"instances"`
}

type StateInstance struct {
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
}

type RemediationPlan struct {
	Actions   []RemediationAction `json:"actions"`
	CreatedAt time.Time           `json:"created_at"`
}

type RemediationAction struct {
	Type        string `json:"type"`
	ResourceID  string `json:"resource_id"`
	Description string `json:"description"`
}
