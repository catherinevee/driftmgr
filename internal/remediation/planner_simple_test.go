package remediation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPlannerConfig(t *testing.T) {
	config := PlannerConfig{
		AutoApprove:        false,
		MaxParallelActions: 5,
		SafeMode:           true,
		DryRun:             false,
		BackupBeforeAction: true,
		MaxRetries:         3,
		ActionTimeout:      30 * time.Second,
	}

	assert.Equal(t, false, config.AutoApprove)
	assert.Equal(t, 5, config.MaxParallelActions)
	assert.Equal(t, true, config.SafeMode)
	assert.Equal(t, false, config.DryRun)
	assert.Equal(t, true, config.BackupBeforeAction)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 30*time.Second, config.ActionTimeout)
}

func TestRemediationPlan(t *testing.T) {
	plan := RemediationPlan{
		ID:               "plan-1",
		Name:             "Test Plan",
		Description:      "Test remediation plan",
		CreatedAt:        time.Now(),
		RiskLevel:        RiskLevelLow,
		RequiresApproval: false,
	}

	assert.Equal(t, "plan-1", plan.ID)
	assert.Equal(t, "Test Plan", plan.Name)
	assert.NotEmpty(t, plan.Description)
	assert.NotZero(t, plan.CreatedAt)
	assert.Equal(t, RiskLevelLow, plan.RiskLevel)
	assert.False(t, plan.RequiresApproval)
}

func TestRiskLevels(t *testing.T) {
	assert.Equal(t, RiskLevel(0), RiskLevelLow)
	assert.Equal(t, RiskLevel(1), RiskLevelMedium)
	assert.Equal(t, RiskLevel(2), RiskLevelHigh)
	assert.Equal(t, RiskLevel(3), RiskLevelCritical)
}

func TestActionTypes(t *testing.T) {
	types := []ActionType{
		ActionType("create"),
		ActionType("update"),
		ActionType("delete"),
		ActionType("import"),
		ActionType("refresh"),
	}

	for _, at := range types {
		assert.NotEmpty(t, string(at))
	}
}

func TestRemediationPlanner(t *testing.T) {
	config := &PlannerConfig{
		MaxParallelActions: 5,
		SafeMode:           true,
	}

	planner := &RemediationPlanner{
		config: config,
	}

	assert.NotNil(t, planner)
	assert.NotNil(t, planner.config)
	assert.Equal(t, 5, planner.config.MaxParallelActions)
	assert.True(t, planner.config.SafeMode)
}