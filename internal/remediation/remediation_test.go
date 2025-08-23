package remediation

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDriftDetector mocks drift detection
type MockDriftDetector struct {
	mock.Mock
}

func (m *MockDriftDetector) DetectDrift(ctx context.Context, resource models.Resource) (*models.DriftItem, error) {
	args := m.Called(ctx, resource)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DriftItem), args.Error(1)
}

// MockCloudProvider mocks cloud provider operations
type MockCloudProvider struct {
	mock.Mock
}

func (m *MockCloudProvider) UpdateResource(ctx context.Context, resourceID string, updates map[string]interface{}) error {
	args := m.Called(ctx, resourceID, updates)
	return args.Error(0)
}

func (m *MockCloudProvider) CreateResource(ctx context.Context, resourceType string, config map[string]interface{}) (string, error) {
	args := m.Called(ctx, resourceType, config)
	return args.String(0), args.Error(1)
}

func (m *MockCloudProvider) DeleteResource(ctx context.Context, resourceID string) error {
	args := m.Called(ctx, resourceID)
	return args.Error(0)
}

func (m *MockCloudProvider) GetResourceState(ctx context.Context, resourceID string) (map[string]interface{}, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func TestRemediationEngine_NewEngine(t *testing.T) {
	engine := NewRemediationEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.strategies)
	assert.NotNil(t, engine.history)
	assert.NotNil(t, engine.rollbackManager)
}

func TestRemediationEngine_CreateRemediationPlan(t *testing.T) {
	tests := []struct {
		name        string
		drift       models.DriftItem
		strategy    string
		expectError bool
	}{
		{
			name: "create plan for modified resource",
			drift: models.DriftItem{
				ResourceID:   "i-123",
				ResourceType: "aws_instance",
				DriftType:    "modified",
				Changes: []models.DriftChange{
					{Field: "instance_type", OldValue: "t2.micro", NewValue: "t2.small"},
				},
			},
			strategy:    "auto",
			expectError: false,
		},
		{
			name: "create plan for deleted resource",
			drift: models.DriftItem{
				ResourceID:   "sg-456",
				ResourceType: "aws_security_group",
				DriftType:    "deleted",
			},
			strategy:    "manual",
			expectError: false,
		},
		{
			name: "create plan for added resource",
			drift: models.DriftItem{
				ResourceID:   "vpc-789",
				ResourceType: "aws_vpc",
				DriftType:    "added",
			},
			strategy:    "auto",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRemediationEngine()
			
			plan, err := engine.CreateRemediationPlan(context.Background(), tt.drift, tt.strategy)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, plan)
				assert.Equal(t, tt.drift.ResourceID, plan.ResourceID)
				assert.NotEmpty(t, plan.Actions)
			}
		})
	}
}

func TestRemediationEngine_ExecuteRemediation(t *testing.T) {
	tests := []struct {
		name        string
		plan        RemediationPlan
		dryRun      bool
		expectError bool
	}{
		{
			name: "execute update remediation",
			plan: RemediationPlan{
				ID:         "plan-1",
				ResourceID: "i-123",
				Actions: []RemediationAction{
					{
						Type:       "update",
						ResourceID: "i-123",
						Changes: map[string]interface{}{
							"instance_type": "t2.micro",
						},
					},
				},
			},
			dryRun:      false,
			expectError: false,
		},
		{
			name: "dry run remediation",
			plan: RemediationPlan{
				ID:         "plan-2",
				ResourceID: "sg-456",
				Actions: []RemediationAction{
					{
						Type:       "create",
						ResourceID: "sg-456",
						Changes: map[string]interface{}{
							"name": "test-sg",
						},
					},
				},
			},
			dryRun:      true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRemediationEngine()
			
			mockProvider := new(MockCloudProvider)
			if !tt.dryRun {
				mockProvider.On("UpdateResource", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockProvider.On("CreateResource", mock.Anything, mock.Anything, mock.Anything).Return("sg-456", nil)
			}
			engine.provider = mockProvider

			result, err := engine.ExecuteRemediation(context.Background(), tt.plan, tt.dryRun)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.dryRun {
					assert.False(t, result.Executed)
				}
			}
		})
	}
}

func TestRemediationEngine_SafetyChecks(t *testing.T) {
	tests := []struct {
		name       string
		plan       RemediationPlan
		expectPass bool
	}{
		{
			name: "pass safety check for test resource",
			plan: RemediationPlan{
				ResourceID:   "test-instance",
				ResourceType: "aws_instance",
				ResourceTags: map[string]string{"Environment": "test"},
			},
			expectPass: true,
		},
		{
			name: "fail safety check for production resource",
			plan: RemediationPlan{
				ResourceID:   "prod-db",
				ResourceType: "aws_rds_instance",
				ResourceTags: map[string]string{"Environment": "production"},
			},
			expectPass: false,
		},
		{
			name: "fail safety check for high-risk action",
			plan: RemediationPlan{
				ResourceID: "vpc-main",
				Actions: []RemediationAction{
					{Type: "delete", Risk: "high"},
				},
			},
			expectPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRemediationEngine()
			
			passed := engine.performSafetyChecks(tt.plan)
			assert.Equal(t, tt.expectPass, passed)
		})
	}
}

func TestRemediationEngine_Rollback(t *testing.T) {
	engine := NewRemediationEngine()
	
	// Create a snapshot before remediation
	snapshot := ResourceSnapshot{
		ResourceID: "i-123",
		State: map[string]interface{}{
			"instance_type": "t2.micro",
			"tags":          map[string]string{"Name": "test"},
		},
		Timestamp: time.Now(),
	}
	
	engine.rollbackManager.SaveSnapshot(snapshot)

	// Simulate failed remediation
	plan := RemediationPlan{
		ID:         "plan-rollback",
		ResourceID: "i-123",
		Actions: []RemediationAction{
			{
				Type:       "update",
				ResourceID: "i-123",
				Changes: map[string]interface{}{
					"instance_type": "t2.large",
				},
			},
		},
	}

	mockProvider := new(MockCloudProvider)
	mockProvider.On("UpdateResource", mock.Anything, "i-123", snapshot.State).Return(nil)
	engine.provider = mockProvider

	err := engine.Rollback(context.Background(), plan.ID, snapshot.ResourceID)
	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

func TestRemediationEngine_ImpactAnalysis(t *testing.T) {
	tests := []struct {
		name           string
		plan           RemediationPlan
		expectedImpact string
	}{
		{
			name: "low impact for tag changes",
			plan: RemediationPlan{
				Actions: []RemediationAction{
					{
						Type: "update",
						Changes: map[string]interface{}{
							"tags": map[string]string{"env": "test"},
						},
					},
				},
			},
			expectedImpact: "low",
		},
		{
			name: "high impact for deletion",
			plan: RemediationPlan{
				Actions: []RemediationAction{
					{Type: "delete"},
				},
			},
			expectedImpact: "high",
		},
		{
			name: "medium impact for configuration changes",
			plan: RemediationPlan{
				Actions: []RemediationAction{
					{
						Type: "update",
						Changes: map[string]interface{}{
							"instance_type": "t2.large",
						},
					},
				},
			},
			expectedImpact: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRemediationEngine()
			
			impact := engine.AnalyzeImpact(tt.plan)
			assert.Equal(t, tt.expectedImpact, impact.Level)
		})
	}
}

func TestRemediationEngine_BatchRemediation(t *testing.T) {
	engine := NewRemediationEngine()
	
	drifts := []models.DriftItem{
		{
			ResourceID:   "i-1",
			ResourceType: "aws_instance",
			DriftType:    "modified",
		},
		{
			ResourceID:   "i-2",
			ResourceType: "aws_instance",
			DriftType:    "modified",
		},
		{
			ResourceID:   "i-3",
			ResourceType: "aws_instance",
			DriftType:    "modified",
		},
	}

	options := BatchRemediationOptions{
		Strategy:    "auto",
		Parallel:    true,
		MaxWorkers:  2,
		DryRun:      true,
	}

	results, err := engine.BatchRemediate(context.Background(), drifts, options)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))
}

func TestRemediationEngine_ValidationSteps(t *testing.T) {
	engine := NewRemediationEngine()
	
	plan := RemediationPlan{
		ResourceID:   "i-123",
		ResourceType: "aws_instance",
		Actions: []RemediationAction{
			{
				Type:       "update",
				ResourceID: "i-123",
				Changes: map[string]interface{}{
					"instance_type": "t2.micro",
				},
			},
		},
	}

	mockProvider := new(MockCloudProvider)
	mockProvider.On("GetResourceState", mock.Anything, "i-123").Return(
		map[string]interface{}{
			"instance_type": "t2.micro",
			"state":         "running",
		}, nil,
	)
	engine.provider = mockProvider

	// Add validation steps
	plan.ValidationSteps = []ValidationStep{
		{
			Type:        "state_check",
			Expected:    "running",
			Field:       "state",
			Description: "Verify instance is running",
		},
		{
			Type:        "value_check",
			Expected:    "t2.micro",
			Field:       "instance_type",
			Description: "Verify instance type updated",
		},
	}

	valid, results := engine.ValidateRemediation(context.Background(), plan)
	assert.True(t, valid)
	assert.Equal(t, 2, len(results))
	for _, result := range results {
		assert.True(t, result.Passed)
	}
}

func TestRemediationEngine_TerraformGeneration(t *testing.T) {
	engine := NewRemediationEngine()
	
	drift := models.DriftItem{
		ResourceID:   "i-123",
		ResourceType: "aws_instance",
		DriftType:    "modified",
		Changes: []models.DriftChange{
			{
				Field:    "instance_type",
				OldValue: "t2.small",
				NewValue: "t2.micro",
			},
		},
	}

	terraform, err := engine.GenerateTerraformPlan(drift)
	assert.NoError(t, err)
	assert.Contains(t, terraform, "resource \"aws_instance\"")
	assert.Contains(t, terraform, "instance_type = \"t2.micro\"")
}

func TestRemediationEngine_ScheduledRemediation(t *testing.T) {
	engine := NewRemediationEngine()
	
	plan := RemediationPlan{
		ID:         "scheduled-plan",
		ResourceID: "i-123",
		Schedule: &RemediationSchedule{
			Type:      "scheduled",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	err := engine.ScheduleRemediation(context.Background(), plan)
	assert.NoError(t, err)
	
	// Verify plan is scheduled
	scheduled := engine.GetScheduledRemediations()
	assert.Equal(t, 1, len(scheduled))
	assert.Equal(t, plan.ID, scheduled[0].ID)
}

func TestRemediationEngine_ApprovalWorkflow(t *testing.T) {
	engine := NewRemediationEngine()
	
	plan := RemediationPlan{
		ID:         "approval-plan",
		ResourceID: "prod-db",
		RequiresApproval: true,
		ApprovalConfig: &ApprovalConfig{
			MinApprovers: 2,
			Approvers:    []string{"admin1", "admin2", "admin3"},
			Timeout:      30 * time.Minute,
		},
	}

	// Submit for approval
	err := engine.SubmitForApproval(context.Background(), plan)
	assert.NoError(t, err)
	assert.Equal(t, "pending_approval", plan.Status)

	// Add approvals
	err = engine.AddApproval(plan.ID, "admin1", true, "Looks good")
	assert.NoError(t, err)
	
	err = engine.AddApproval(plan.ID, "admin2", true, "Approved")
	assert.NoError(t, err)

	// Check if approved
	approved := engine.IsApproved(plan.ID)
	assert.True(t, approved)
}