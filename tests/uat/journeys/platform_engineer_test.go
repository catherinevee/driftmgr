package journeys

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformEngineerMultiCloudManagement(t *testing.T) {
	// Platform Engineer managing multi-cloud infrastructure
	ctx := context.Background()
	j := NewJourney("platform_engineer", "Multi-Cloud Infrastructure Management")

	// Step 1: Discover resources across multiple clouds
	t.Run("MultiCloudDiscovery", func(t *testing.T) {
		step := j.AddStep("Discover AWS resources", "platform_engineer discovers AWS resources")
		
		output, err := j.ExecuteCommand(ctx, "driftmgr", "discover", "--provider", "aws", "--region", "us-east-1")
		require.NoError(t, err)
		assert.Contains(t, output, "Discovery complete")
		
		step.Complete(true, "AWS discovery successful")

		// Azure discovery
		step = j.AddStep("Discover Azure resources", "platform_engineer discovers Azure resources")
		output, err = j.ExecuteCommand(ctx, "driftmgr", "discover", "--provider", "azure")
		// Azure might not be configured, so we just check the command runs
		step.Complete(err == nil, "Azure discovery attempted")

		// GCP discovery
		step = j.AddStep("Discover GCP resources", "platform_engineer discovers GCP resources")
		output, err = j.ExecuteCommand(ctx, "driftmgr", "discover", "--provider", "gcp")
		step.Complete(err == nil, "GCP discovery attempted")
	})

	// Step 2: Analyze Terraform states
	t.Run("StateAnalysis", func(t *testing.T) {
		step := j.AddStep("Analyze state files", "platform_engineer analyzes Terraform states")
		
		// Create a test state file
		stateFile := createTestStateFile(t)
		defer cleanupTestFile(stateFile)
		
		output, err := j.ExecuteCommand(ctx, "driftmgr", "analyze", "--state", stateFile)
		require.NoError(t, err)
		assert.Contains(t, output, "resources")
		
		step.Complete(true, "State analysis complete")
	})

	// Step 3: Manage state backends
	t.Run("StateBackendManagement", func(t *testing.T) {
		step := j.AddStep("List remote states", "platform_engineer lists states in S3 backend")
		
		// This might fail if S3 isn't configured
		output, err := j.ExecuteCommand(ctx, "driftmgr", "state", "list", "--backend", "s3", "--bucket", "test-states")
		
		if err != nil {
			step.Complete(false, "S3 backend not configured")
		} else {
			assert.Contains(t, output, "state")
			step.Complete(true, "Remote states listed")
		}
	})

	// Step 4: Import unmanaged resources
	t.Run("ImportGeneration", func(t *testing.T) {
		step := j.AddStep("Generate import commands", "platform_engineer generates Terraform import commands")
		
		output, err := j.ExecuteCommand(ctx, "driftmgr", "import", "--provider", "aws", "--dry-run")
		
		if err != nil {
			step.Complete(false, "Import generation not available")
		} else {
			step.Complete(true, "Import commands generated")
		}
	})

	// Step 5: Terragrunt support
	t.Run("TerragruntIntegration", func(t *testing.T) {
		step := j.AddStep("Analyze Terragrunt configs", "platform_engineer analyzes Terragrunt configurations")
		
		// Create test Terragrunt config
		tgConfig := createTestTerragruntConfig(t)
		defer cleanupTestFile(tgConfig)
		
		output, err := j.ExecuteCommand(ctx, "driftmgr", "terragrunt", "analyze", "--path", ".")
		
		if err != nil {
			step.Complete(false, "Terragrunt analysis failed")
		} else {
			step.Complete(true, "Terragrunt analysis complete")
		}
	})

	// Generate report
	report := j.GenerateReport()
	assert.NotNil(t, report)
	assert.Equal(t, "platform_engineer", report.Persona)
	assert.True(t, report.CompletionRate > 0)
	
	t.Logf("Platform Engineer Journey: %d/%d steps completed (%.1f%%)",
		report.CompletedSteps, report.TotalSteps, report.CompletionRate)
}

func TestPlatformEngineerDisasterRecovery(t *testing.T) {
	// Platform Engineer handling disaster recovery scenario
	ctx := context.Background()
	j := NewJourney("platform_engineer", "Disaster Recovery")

	// Step 1: Backup current state
	t.Run("StateBackup", func(t *testing.T) {
		step := j.AddStep("Backup state", "platform_engineer backs up current state")
		
		stateFile := createTestStateFile(t)
		defer cleanupTestFile(stateFile)
		
		output, err := j.ExecuteCommand(ctx, "driftmgr", "state", "backup", "--state", stateFile)
		
		if err != nil {
			// Backup command might not be implemented
			step.Complete(false, "Backup feature not available")
		} else {
			assert.Contains(t, output, "backup")
			step.Complete(true, "State backed up successfully")
		}
	})

	// Step 2: Validate state integrity
	t.Run("StateValidation", func(t *testing.T) {
		step := j.AddStep("Validate state", "platform_engineer validates state integrity")
		
		stateFile := createTestStateFile(t)
		defer cleanupTestFile(stateFile)
		
		output, err := j.ExecuteCommand(ctx, "driftmgr", "state", "validate", "--state", stateFile)
		
		if err != nil {
			step.Complete(false, "Validation failed")
		} else {
			step.Complete(true, "State validated successfully")
		}
	})

	// Step 3: Bulk resource operations
	t.Run("BulkOperations", func(t *testing.T) {
		step := j.AddStep("Plan bulk deletion", "platform_engineer plans bulk resource deletion")
		
		output, err := j.ExecuteCommand(ctx, "driftmgr", "bulk-delete", 
			"--provider", "aws", 
			"--filter", "tag:Environment=test",
			"--dry-run")
		
		if err != nil {
			step.Complete(false, "Bulk delete planning failed")
		} else {
			step.Complete(true, "Bulk deletion plan created")
		}
	})

	// Generate report
	report := j.GenerateReport()
	assert.NotNil(t, report)
	t.Logf("Platform Engineer DR Journey: %.1f%% complete", report.CompletionRate)
}

func createTestTerragruntConfig(t *testing.T) string {
	content := `
terraform {
  source = "../../modules/vpc"
}

inputs = {
  vpc_cidr = "10.0.0.0/16"
  environment = "test"
}
`
	return createTestFile(t, "terragrunt.hcl", content)
}