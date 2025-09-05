package journeys

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/catherinevee/driftmgr/internal/remediation/strategies"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDevOpsEngineerJourney tests the complete daily workflow of a DevOps engineer
func TestDevOpsEngineerJourney(t *testing.T) {
	t.Run("MorningDriftCheck", func(t *testing.T) {
		// User Story: As a DevOps engineer, I want to check for drift
		// across all environments first thing in the morning

		// Given: Multiple environments with Terraform stacks
		workspace := setupTestEnvironments(t)
		defer cleanupTestEnvironments(workspace)

		// When: Running quick drift scan across all environments
		start := time.Now()
		results := runQuickScan(t, workspace, []string{"dev", "staging", "prod"})
		duration := time.Since(start)

		// Then: Results returned within 30 seconds
		assert.Less(t, duration, 30*time.Second,
			"Quick scan took %v, expected < 30s", duration)

		// And: Summary clearly shows drift by environment
		assert.NotNil(t, results.EnvironmentSummary)
		assert.Contains(t, results.EnvironmentSummary, "dev")
		assert.Contains(t, results.EnvironmentSummary, "staging")
		assert.Contains(t, results.EnvironmentSummary, "prod")

		// And: Critical drift is highlighted
		if results.HasCriticalDrift {
			assert.Greater(t, len(results.CriticalItems), 0)
			assert.NotEmpty(t, results.CriticalItems[0].SuggestedAction)
		}
	})

	t.Run("InvestigateSpecificDrift", func(t *testing.T) {
		// User Story: As a DevOps engineer, I want to deep-dive into
		// specific drift items to understand what changed

		workspace := setupTestEnvironments(t)
		defer cleanupTestEnvironments(workspace)

		// Given: Morning scan found drift in production
		introduceDrift(t, filepath.Join(workspace, "prod", "database"))
		quickResults := runQuickScan(t, workspace, []string{"prod/database"})
		require.True(t, quickResults.HasDrift, "Test requires drift to be present")

		// When: Investigating specific drift with deep mode
		driftItem := quickResults.DriftItems[0]
		detailedReport := investigateDrift(t, workspace, driftItem.ResourceID)

		// Then: Exact attribute changes are shown
		assert.NotNil(t, detailedReport.AttributeChanges)
		assert.Greater(t, len(detailedReport.AttributeChanges), 0)

		// And: Drift timeline is available
		assert.NotNil(t, detailedReport.DriftTimeline)
		assert.True(t, detailedReport.EstimatedDriftTime.Before(time.Now()))

		// And: Actionable remediation options provided
		assert.GreaterOrEqual(t, len(detailedReport.RemediationOptions), 2)
		hasApplyOption := false
		for _, opt := range detailedReport.RemediationOptions {
			if opt.Type == "terraform_apply" {
				hasApplyOption = true
				break
			}
		}
		assert.True(t, hasApplyOption, "Should have terraform apply option")
	})

	t.Run("RemediateNonCriticalDrift", func(t *testing.T) {
		// User Story: As a DevOps engineer, I want to safely remediate
		// non-critical drift in lower environments

		workspace := setupTestEnvironments(t)
		defer cleanupTestEnvironments(workspace)

		// Given: Non-critical drift in dev environment
		introduceNonCriticalDrift(t, filepath.Join(workspace, "dev", "compute"))
		driftReport := runQuickScan(t, workspace, []string{"dev/compute"})
		require.True(t, driftReport.HasDrift)

		// When: Running remediation with dry-run first
		dryRunResult := remediateDrift(t, workspace, driftReport.DriftItems, true)

		// Then: Dry-run shows what would change
		assert.True(t, dryRunResult.IsDryRun)
		assert.NotNil(t, dryRunResult.PlannedChanges)
		assert.Less(t, dryRunResult.EstimatedDuration, 5*time.Minute)

		// When: Applying actual remediation
		actualResult := remediateDrift(t, workspace, driftReport.DriftItems, false)

		// Then: Remediation completes successfully
		assert.True(t, actualResult.Success)
		assert.Equal(t, len(driftReport.DriftItems), actualResult.RemediatedCount)

		// And: Drift is resolved
		verifyReport := runQuickScan(t, workspace, []string{"dev/compute"})
		assert.False(t, verifyReport.HasDrift)
	})

	t.Run("HandleProductionDrift", func(t *testing.T) {
		// User Story: As a DevOps engineer, I need extra safety
		// when remediating production drift

		workspace := setupTestEnvironments(t)
		defer cleanupTestEnvironments(workspace)

		// Given: Critical drift in production
		introduceCriticalDrift(t, filepath.Join(workspace, "prod", "compute"))
		driftReport := runQuickScan(t, workspace, []string{"prod/compute"})

		// When: Attempting remediation without approval
		result := attemptRemediation(t, workspace, driftReport.DriftItems, false, false)

		// Then: Should require approval for critical changes
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "approval required")

		// When: Providing approval
		approvedResult := attemptRemediation(t, workspace, driftReport.DriftItems, false, true)

		// Then: Remediation proceeds with approval
		assert.True(t, approvedResult.Success)
		assert.NotNil(t, approvedResult.BackupCreated)
		assert.NotNil(t, approvedResult.RollbackPlan)
	})
}

// Helper functions for test setup and execution

func setupTestEnvironments(t *testing.T) string {
	workspace := t.TempDir()
	environments := []string{"dev", "staging", "prod"}
	stacks := []string{"networking", "compute", "database"}

	for _, env := range environments {
		for _, stack := range stacks {
			stackDir := filepath.Join(workspace, env, stack)
			require.NoError(t, os.MkdirAll(stackDir, 0755))
			createTerraformStack(t, stackDir, env, stack)
			initializeStack(t, stackDir)
		}
	}

	return workspace
}

func createTerraformStack(t *testing.T, dir, env, stack string) {
	// Create realistic Terraform configuration
	mainTf := fmt.Sprintf(`
terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "%s_%s_server" {
  ami           = "ami-12345678"
  instance_type = "%s"
  
  tags = {
    Name        = "%s-%s-server"
    Environment = "%s"
    Stack       = "%s"
    ManagedBy   = "terraform"
  }
}
`, env, stack, getInstanceType(env), env, stack, env, stack)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "main.tf"),
		[]byte(mainTf),
		0644,
	))
}

func getInstanceType(env string) string {
	switch env {
	case "prod":
		return "t3.large"
	case "staging":
		return "t3.medium"
	default:
		return "t3.micro"
	}
}

func runQuickScan(t *testing.T, workspace string, paths []string) *ScanResult {
	ctx := context.Background()

	scanner := &Scanner{
		Workspace: workspace,
		Mode:      detector.QuickMode,
	}

	result, err := scanner.Scan(ctx, paths)
	require.NoError(t, err)

	return result
}

func investigateDrift(t *testing.T, workspace, resourceID string) *DetailedDriftReport {
	ctx := context.Background()

	investigator := &DriftInvestigator{
		Workspace: workspace,
		Mode:      detector.DeepMode,
	}

	report, err := investigator.Investigate(ctx, resourceID)
	require.NoError(t, err)

	return report
}

func remediateDrift(t *testing.T, workspace string, driftItems []DriftItem, dryRun bool) *RemediationResult {
	ctx := context.Background()

	config := &strategies.StrategyConfig{
		DryRun:      dryRun,
		AutoApprove: !dryRun && true, // Auto-approve for dev environment tests
		WorkingDir:  workspace,
	}

	strategy := strategies.NewCodeAsTruthStrategy(config)

	// Convert drift items to drift result
	driftResult := &detector.DriftResult{
		HasDrift:    len(driftItems) > 0,
		Differences: convertToDifferences(driftItems),
	}

	plan, err := strategy.Plan(ctx, driftResult)
	require.NoError(t, err)

	if dryRun {
		return &RemediationResult{
			IsDryRun:          true,
			PlannedChanges:    plan.Actions,
			EstimatedDuration: plan.EstimatedTime,
		}
	}

	result, err := strategy.Execute(ctx, plan)
	require.NoError(t, err)

	return &RemediationResult{
		Success:         result.Success,
		RemediatedCount: len(result.ActionsExecuted),
	}
}

func introduceDrift(t *testing.T, stackDir string) {
	// Simulate drift by modifying actual resources
	// In real tests, this would interact with cloud provider
	stateFile := filepath.Join(stackDir, "terraform.tfstate")

	// Modify state to simulate drift
	parser := state.NewStateParser()
	tfState, err := parser.ParseFile(stateFile)
	require.NoError(t, err)

	// Change an attribute to simulate drift
	if len(tfState.Resources) > 0 {
		tfState.Resources[0].Attributes["instance_type"] = "t3.small"
	}

	// Save modified state
	require.NoError(t, state.SaveState(tfState, stateFile))
}

func introduceNonCriticalDrift(t *testing.T, stackDir string) {
	// Introduce minor drift like tag changes
	introduceDrift(t, stackDir)
}

func introduceCriticalDrift(t *testing.T, stackDir string) {
	// Introduce critical drift like security group changes
	stateFile := filepath.Join(stackDir, "terraform.tfstate")

	parser := state.NewStateParser()
	tfState, err := parser.ParseFile(stateFile)
	require.NoError(t, err)

	if len(tfState.Resources) > 0 {
		// Simulate critical security group change
		tfState.Resources[0].Attributes["security_groups"] = []string{"sg-public"}
	}

	require.NoError(t, state.SaveState(tfState, stateFile))
}
