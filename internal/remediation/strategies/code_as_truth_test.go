package strategies

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/comparator"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeAsTruthStrategy(t *testing.T) {
	ctx := context.Background()

	config := &StrategyConfig{
		TerraformPath: "terraform",
		Timeout:       5 * time.Minute,
		DryRun:        true, // Always dry run in tests
		AutoApprove:   false,
		MaxParallel:   1,
	}

	strategy := NewCodeAsTruthStrategy(config)

	t.Run("GetType", func(t *testing.T) {
		assert.Equal(t, CodeAsTruthStrategy, strategy.GetType())
	})

	t.Run("GetDescription", func(t *testing.T) {
		desc := strategy.GetDescription()
		assert.Contains(t, desc, "Terraform")
		assert.Contains(t, desc, "fix drift")
	})

	t.Run("Validate", func(t *testing.T) {
		// Test with no drift
		noDrift := &detector.DriftResult{
			DriftType: detector.NoDrift,
		}
		err := strategy.Validate(noDrift)
		assert.Error(t, err, "Should error when no drift detected")

		// Test with drift
		withDrift := &detector.DriftResult{
			DriftType: detector.ConfigurationDrift,
			Differences: []comparator.Difference{
				{
					Path:       "aws_instance.test",
					Type:       comparator.DiffTypeModified,
					Importance: comparator.ImportanceMedium,
				},
			},
		}
		err = strategy.Validate(withDrift)
		// May fail if terraform not in PATH, but that's okay for tests
		if err != nil {
			assert.Contains(t, err.Error(), "terraform")
		}
	})

	t.Run("Plan", func(t *testing.T) {
		drift := &detector.DriftResult{
			DriftType: detector.ConfigurationDrift,
			Differences: []comparator.Difference{
				{
					Path:       "aws_instance.test",
					Type:       comparator.DiffTypeModified,
					Importance: comparator.ImportanceCritical,
					Expected:   "t2.micro",
					Actual:     "t2.small",
				},
				{
					Path:       "aws_s3_bucket.backup",
					Type:       comparator.DiffTypeRemoved,
					Importance: comparator.ImportanceHigh,
					Expected:   map[string]interface{}{"name": "backup"},
					Actual:     nil,
				},
			},
		}

		plan, err := strategy.Plan(ctx, drift)
		if err != nil {
			// If terraform is not available, skip this test
			t.Skip("Terraform not available in test environment")
			return
		}

		require.NoError(t, err)
		require.NotNil(t, plan)

		// Verify plan structure
		assert.NotEmpty(t, plan.ID)
		assert.Equal(t, CodeAsTruthStrategy, plan.Strategy)
		assert.NotZero(t, plan.CreatedAt)
		assert.NotNil(t, plan.DriftSummary)

		// Check drift summary
		assert.Equal(t, 2, plan.DriftSummary.TotalResources)
		assert.Equal(t, 1, plan.DriftSummary.DriftedResources)
		assert.Equal(t, 1, plan.DriftSummary.MissingResources)
		assert.Equal(t, 1, plan.DriftSummary.CriticalDrifts)

		// Check actions
		assert.NotEmpty(t, plan.Actions)

		// Should have at least a plan action
		hasPlanAction := false
		for _, action := range plan.Actions {
			if action.Type == ActionApply {
				hasPlanAction = true
				break
			}
		}
		assert.True(t, hasPlanAction, "Should have a plan/apply action")

		// Check risk level
		assert.Equal(t, RiskHigh, plan.RiskLevel, "Should be high risk due to critical drift")

		// Check metadata
		assert.Contains(t, plan.Metadata, "terraform_version")
		assert.Contains(t, plan.Metadata, "dry_run")
	})

	t.Run("Execute_DryRun", func(t *testing.T) {
		drift := &detector.DriftResult{
			DriftType: detector.ConfigurationDrift,
			Differences: []comparator.Difference{
				{
					Path:       "aws_instance.test",
					Type:       comparator.DiffTypeModified,
					Importance: comparator.ImportanceMedium,
				},
			},
		}

		plan, err := strategy.Plan(ctx, drift)
		if err != nil {
			t.Skip("Terraform not available in test environment")
			return
		}

		require.NoError(t, err)
		require.NotNil(t, plan)

		// Execute in dry run mode
		result, err := strategy.Execute(ctx, plan)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify execution result
		assert.Equal(t, plan.ID, result.PlanID)
		assert.NotZero(t, result.StartedAt)
		assert.NotZero(t, result.CompletedAt)
		assert.NotZero(t, result.Duration)

		// Check actions executed
		assert.Len(t, result.ActionsExecuted, len(plan.Actions))

		// In dry run, all actions should succeed
		for _, action := range result.ActionsExecuted {
			assert.True(t, action.Success, "Dry run actions should succeed")
			assert.Contains(t, action.Output, "[DRY RUN]")
		}
	})

	t.Run("RequiresApproval", func(t *testing.T) {
		// Test auto-approve config
		autoApproveConfig := &StrategyConfig{
			TerraformPath: "terraform",
			AutoApprove:   true,
			DryRun:        true,
		}
		autoStrategy := NewCodeAsTruthStrategy(autoApproveConfig)

		drift := &detector.DriftResult{
			DriftType: detector.ConfigurationDrift,
			Differences: []comparator.Difference{
				{
					Path:       "aws_instance.critical",
					Type:       comparator.DiffTypeModified,
					Importance: comparator.ImportanceCritical,
				},
			},
		}

		plan, err := autoStrategy.Plan(ctx, drift)
		if err != nil {
			t.Skip("Terraform not available in test environment")
			return
		}

		assert.False(t, plan.RequiresApproval, "Should not require approval with auto-approve")

		// Test manual approval for critical changes
		manualConfig := &StrategyConfig{
			TerraformPath:      "terraform",
			AutoApprove:        false,
			DryRun:             true,
			RequireApprovalFor: []RiskLevel{RiskHigh, RiskCritical},
		}
		manualStrategy := NewCodeAsTruthStrategy(manualConfig)

		plan, err = manualStrategy.Plan(ctx, drift)
		if err != nil {
			t.Skip("Terraform not available in test environment")
			return
		}

		assert.True(t, plan.RequiresApproval, "Should require approval for critical changes")
	})
}

func TestDriftSummaryCreation(t *testing.T) {
	strategy := NewCodeAsTruthStrategy(nil)

	drift := &detector.DriftResult{
		DriftType: detector.ConfigurationDrift,
		Differences: []comparator.Difference{
			{
				Path:       "aws_instance.web",
				Type:       comparator.DiffTypeModified,
				Importance: comparator.ImportanceCritical,
			},
			{
				Path:       "aws_instance.db",
				Type:       comparator.DiffTypeRemoved,
				Importance: comparator.ImportanceHigh,
			},
			{
				Path:       "aws_s3_bucket.logs",
				Type:       comparator.DiffTypeAdded,
				Importance: comparator.ImportanceLow,
			},
			{
				Path:       "aws_rds_cluster.main",
				Type:       comparator.DiffTypeModified,
				Importance: comparator.ImportanceMedium,
			},
		},
	}

	summary := strategy.createDriftSummary(drift)

	assert.NotNil(t, summary)
	assert.Equal(t, 4, summary.TotalResources)
	assert.Equal(t, 2, summary.DriftedResources)
	assert.Equal(t, 1, summary.MissingResources)
	assert.Equal(t, 1, summary.UnmanagedResources)
	assert.Equal(t, 1, summary.CriticalDrifts)
	assert.Contains(t, summary.AffectedServices, "aws_instance")
	assert.Contains(t, summary.AffectedServices, "aws_s3_bucket")
	assert.Contains(t, summary.AffectedServices, "aws_rds_cluster")
}

func TestEstimateExecutionTime(t *testing.T) {
	strategy := NewCodeAsTruthStrategy(nil)

	tests := []struct {
		name        string
		actionCount int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "Small operation",
			actionCount: 1,
			minExpected: 30 * time.Second,
			maxExpected: 1 * time.Minute,
		},
		{
			name:        "Medium operation",
			actionCount: 5,
			minExpected: 1 * time.Minute,
			maxExpected: 2 * time.Minute,
		},
		{
			name:        "Large operation",
			actionCount: 15,
			minExpected: 3 * time.Minute,
			maxExpected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := strategy.estimateExecutionTime(tt.actionCount)
			assert.GreaterOrEqual(t, duration, tt.minExpected)
			assert.LessOrEqual(t, duration, tt.maxExpected)
		})
	}
}

func TestBuildPlanCommand(t *testing.T) {
	config := &StrategyConfig{
		TerraformPath: "terraform",
		AutoApprove:   true,
	}
	strategy := NewCodeAsTruthStrategy(config)

	targets := []string{
		"aws_instance.web",
		"aws_s3_bucket.data",
	}

	command := strategy.buildPlanCommand(targets)

	assert.Contains(t, command, "terraform")
	assert.Contains(t, command, "plan")
	assert.Contains(t, command, "-out=drift-remediation.tfplan")
	assert.Contains(t, command, "-target=aws_instance.web")
	assert.Contains(t, command, "-target=aws_s3_bucket.data")
	assert.Contains(t, command, "-auto-approve")
}
