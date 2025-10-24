package remediation

import (
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRemediationJob_Validation(t *testing.T) {
	tests := []struct {
		name    string
		job     models.RemediationJob
		wantErr bool
	}{
		{
			name: "valid job",
			job: models.RemediationJob{
				ID:            "550e8400-e29b-41d4-a716-446655440000",
				DriftResultID: "550e8400-e29b-41d4-a716-446655440001",
				Strategy: models.RemediationStrategy{
					Type: models.StrategyTypeTerraformApply,
					Name: "Test Strategy",
				},
				Status:    models.JobStatusPending,
				Priority:  models.JobPriorityMedium,
				CreatedBy: "user-123",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid job status",
			job: models.RemediationJob{
				ID:            "550e8400-e29b-41d4-a716-446655440000",
				DriftResultID: "550e8400-e29b-41d4-a716-446655440001",
				Strategy: models.RemediationStrategy{
					Type: models.StrategyTypeTerraformApply,
					Name: "Test Strategy",
				},
				Status:    "invalid_status",
				Priority:  models.JobPriorityMedium,
				CreatedBy: "user-123",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid job priority",
			job: models.RemediationJob{
				ID:            "550e8400-e29b-41d4-a716-446655440000",
				DriftResultID: "550e8400-e29b-41d4-a716-446655440001",
				Strategy: models.RemediationStrategy{
					Type: models.StrategyTypeTerraformApply,
					Name: "Test Strategy",
				},
				Status:    models.JobStatusPending,
				Priority:  "invalid_priority",
				CreatedBy: "user-123",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing required fields",
			job: models.RemediationJob{
				ID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationStrategy_Validation(t *testing.T) {
	tests := []struct {
		name     string
		strategy models.RemediationStrategy
		wantErr  bool
	}{
		{
			name: "valid strategy",
			strategy: models.RemediationStrategy{
				Type:        models.StrategyTypeTerraformApply,
				Name:        "Test Strategy",
				Description: "A test strategy",
				RetryCount:  3,
			},
			wantErr: false,
		},
		{
			name: "invalid strategy type",
			strategy: models.RemediationStrategy{
				Type:        "invalid_type",
				Name:        "Test Strategy",
				Description: "A test strategy",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			strategy: models.RemediationStrategy{
				Type:        models.StrategyTypeTerraformApply,
				Name:        "",
				Description: "A test strategy",
			},
			wantErr: true,
		},
		{
			name: "invalid retry count",
			strategy: models.RemediationStrategy{
				Type:       models.StrategyTypeTerraformApply,
				Name:       "Test Strategy",
				RetryCount: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.strategy.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationJobRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request models.RemediationJobRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.RemediationJobRequest{
				DriftResultID: "550e8400-e29b-41d4-a716-446655440001",
				Strategy: models.RemediationStrategy{
					Type: models.StrategyTypeTerraformApply,
					Name: "Test Strategy",
				},
				Priority:         models.JobPriorityMedium,
				DryRun:           false,
				RequiresApproval: false,
			},
			wantErr: false,
		},
		{
			name: "invalid priority",
			request: models.RemediationJobRequest{
				DriftResultID: "550e8400-e29b-41d4-a716-446655440001",
				Strategy: models.RemediationStrategy{
					Type: models.StrategyTypeTerraformApply,
					Name: "Test Strategy",
				},
				Priority:         "invalid_priority",
				DryRun:           false,
				RequiresApproval: false,
			},
			wantErr: true,
		},
		{
			name: "missing drift result ID",
			request: models.RemediationJobRequest{
				DriftResultID: "",
				Strategy: models.RemediationStrategy{
					Type: models.StrategyTypeTerraformApply,
					Name: "Test Strategy",
				},
				Priority:         models.JobPriorityMedium,
				DryRun:           false,
				RequiresApproval: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationJobListRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request models.RemediationJobListRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.RemediationJobListRequest{
				Limit:     50,
				Offset:    0,
				SortBy:    "created_at",
				SortOrder: "desc",
			},
			wantErr: false,
		},
		{
			name: "invalid limit",
			request: models.RemediationJobListRequest{
				Limit:  -1,
				Offset: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid offset",
			request: models.RemediationJobListRequest{
				Limit:  50,
				Offset: -1,
			},
			wantErr: true,
		},
		{
			name: "limit too high",
			request: models.RemediationJobListRequest{
				Limit:  2000,
				Offset: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid sort field",
			request: models.RemediationJobListRequest{
				Limit:  50,
				Offset: 0,
				SortBy: "invalid_field",
			},
			wantErr: true,
		},
		{
			name: "invalid sort order",
			request: models.RemediationJobListRequest{
				Limit:     50,
				Offset:    0,
				SortBy:    "created_at",
				SortOrder: "invalid_order",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationStrategyRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request models.RemediationStrategyRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.RemediationStrategyRequest{
				Type:        models.StrategyTypeTerraformApply,
				Name:        "Test Strategy",
				Description: "A test strategy",
				RetryCount:  3,
			},
			wantErr: false,
		},
		{
			name: "invalid strategy type",
			request: models.RemediationStrategyRequest{
				Type:        "invalid_type",
				Name:        "Test Strategy",
				Description: "A test strategy",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			request: models.RemediationStrategyRequest{
				Type:        models.StrategyTypeTerraformApply,
				Name:        "",
				Description: "A test strategy",
			},
			wantErr: true,
		},
		{
			name: "invalid retry count",
			request: models.RemediationStrategyRequest{
				Type:       models.StrategyTypeTerraformApply,
				Name:       "Test Strategy",
				RetryCount: 15, // Too high
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationHistoryRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request models.RemediationHistoryRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.RemediationHistoryRequest{
				Limit:  50,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid limit",
			request: models.RemediationHistoryRequest{
				Limit:  -1,
				Offset: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid offset",
			request: models.RemediationHistoryRequest{
				Limit:  50,
				Offset: -1,
			},
			wantErr: true,
		},
		{
			name: "limit too high",
			request: models.RemediationHistoryRequest{
				Limit:  2000,
				Offset: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJobProgressUpdate_Validation(t *testing.T) {
	tests := []struct {
		name    string
		update  models.JobProgressUpdate
		wantErr bool
	}{
		{
			name: "valid update",
			update: models.JobProgressUpdate{
				JobID:               "550e8400-e29b-41d4-a716-446655440000",
				TotalResources:      10,
				ProcessedResources:  5,
				SuccessfulResources: 4,
				FailedResources:     1,
				CurrentStep:         "Processing",
			},
			wantErr: false,
		},
		{
			name: "invalid job ID",
			update: models.JobProgressUpdate{
				JobID:              "",
				TotalResources:     10,
				ProcessedResources: 5,
				CurrentStep:        "Processing",
			},
			wantErr: true,
		},
		{
			name: "negative total resources",
			update: models.JobProgressUpdate{
				JobID:              "job-123",
				TotalResources:     -1,
				ProcessedResources: 5,
				CurrentStep:        "Processing",
			},
			wantErr: true,
		},
		{
			name: "negative processed resources",
			update: models.JobProgressUpdate{
				JobID:              "job-123",
				TotalResources:     10,
				ProcessedResources: -1,
				CurrentStep:        "Processing",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.update.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJobCancelRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request models.JobCancelRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.JobCancelRequest{
				Reason: "User requested cancellation",
			},
			wantErr: false,
		},
		{
			name: "reason too long",
			request: models.JobCancelRequest{
				Reason: string(make([]byte, 501)), // 501 characters
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApprovalRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request models.ApprovalRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.ApprovalRequest{
				JobID:    "550e8400-e29b-41d4-a716-446655440000",
				Approved: true,
				Comments: "Approved for execution",
			},
			wantErr: false,
		},
		{
			name: "invalid job ID",
			request: models.ApprovalRequest{
				JobID:    "",
				Approved: true,
				Comments: "Approved for execution",
			},
			wantErr: true,
		},
		{
			name: "comments too long",
			request: models.ApprovalRequest{
				JobID:    "job-123",
				Approved: true,
				Comments: string(make([]byte, 1001)), // 1001 characters
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationJob_HelperMethods(t *testing.T) {
	job := &models.RemediationJob{
		Status:           models.JobStatusRunning,
		RequiresApproval: true,
	}

	// Test IsCompleted
	assert.False(t, job.IsCompleted())

	job.Status = models.JobStatusCompleted
	assert.True(t, job.IsCompleted())

	job.Status = models.JobStatusFailed
	assert.True(t, job.IsCompleted())

	job.Status = models.JobStatusCancelled
	assert.True(t, job.IsCompleted())

	// Test IsRunning
	job.Status = models.JobStatusRunning
	assert.True(t, job.IsRunning())

	job.Status = models.JobStatusPending
	assert.False(t, job.IsRunning())

	// Test CanBeCancelled
	job.Status = models.JobStatusRunning
	assert.True(t, job.CanBeCancelled())

	job.Status = models.JobStatusPending
	assert.True(t, job.CanBeCancelled())

	job.Status = models.JobStatusCompleted
	assert.False(t, job.CanBeCancelled())

	// Test NeedsApproval
	job.RequiresApproval = true
	job.ApprovedBy = nil
	assert.True(t, job.NeedsApproval())

	job.ApprovedBy = stringPtr("user-123")
	assert.False(t, job.NeedsApproval())
}

func TestJobProgress_HelperMethods(t *testing.T) {
	progress := &models.JobProgress{
		TotalResources:     10,
		ProcessedResources: 0,
	}

	// Test CalculateProgressPercentage
	assert.Equal(t, 0.0, progress.CalculateProgressPercentage())

	progress.ProcessedResources = 5
	assert.Equal(t, 50.0, progress.CalculateProgressPercentage())

	progress.ProcessedResources = 10
	assert.Equal(t, 100.0, progress.CalculateProgressPercentage())

	// Test UpdateProgress
	progress.UpdateProgress(8, 7, 1)
	assert.Equal(t, 8, progress.ProcessedResources)
	assert.Equal(t, 7, progress.SuccessfulResources)
	assert.Equal(t, 1, progress.FailedResources)
	assert.Equal(t, 80.0, progress.Percentage)
}

func TestJobLog_Validation(t *testing.T) {
	tests := []struct {
		name    string
		log     models.JobLog
		wantErr bool
	}{
		{
			name: "valid log",
			log: models.JobLog{
				ID:        "log-123",
				JobID:     "job-123",
				Level:     models.LogLevelInfo,
				Message:   "Test log message",
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			log: models.JobLog{
				ID:        "log-123",
				JobID:     "job-123",
				Level:     "invalid_level",
				Message:   "Test log message",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "empty message",
			log: models.JobLog{
				ID:        "log-123",
				JobID:     "job-123",
				Level:     models.LogLevelInfo,
				Message:   "",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
