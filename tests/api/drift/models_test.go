package drift

import (
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDriftResult_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request *models.DriftResultRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &models.DriftResultRequest{
				Provider: "aws",
				Options: models.DriftOptions{
					Recursive:   true,
					IncludeTags: true,
					MaxDepth:    10,
					Timeout:     time.Minute * 5,
					Parallel:    true,
					DryRun:      false,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			request: &models.DriftResultRequest{
				Provider: "invalid",
			},
			wantErr: true,
		},
		{
			name: "empty provider",
			request: &models.DriftResultRequest{
				Provider: "",
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

func TestDriftHistoryRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request *models.DriftHistoryRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &models.DriftHistoryRequest{
				Provider:  "aws",
				StartDate: time.Now().Add(-time.Hour * 24),
				EndDate:   time.Now(),
				Limit:     50,
				Offset:    0,
				Status:    "completed",
			},
			wantErr: false,
		},
		{
			name: "invalid limit",
			request: &models.DriftHistoryRequest{
				Limit: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid offset",
			request: &models.DriftHistoryRequest{
				Offset: -1,
			},
			wantErr: true,
		},
		{
			name: "limit too high",
			request: &models.DriftHistoryRequest{
				Limit: 2000,
			},
			wantErr: true,
		},
		{
			name: "invalid date range",
			request: &models.DriftHistoryRequest{
				StartDate: time.Now(),
				EndDate:   time.Now().Add(-time.Hour),
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

func TestDriftResultQuery_Validation(t *testing.T) {
	tests := []struct {
		name    string
		query   *models.DriftResultQuery
		wantErr bool
	}{
		{
			name: "valid query",
			query: &models.DriftResultQuery{
				Filter: models.DriftResultFilter{
					Provider: "aws",
					Status:   "completed",
				},
				Sort: models.DriftResultSort{
					Field: "timestamp",
					Order: "desc",
				},
				Limit:  50,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid limit",
			query: &models.DriftResultQuery{
				Limit: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid offset",
			query: &models.DriftResultQuery{
				Offset: -1,
			},
			wantErr: true,
		},
		{
			name: "limit too high",
			query: &models.DriftResultQuery{
				Limit: 2000,
			},
			wantErr: true,
		},
		{
			name: "invalid sort field",
			query: &models.DriftResultQuery{
				Sort: models.DriftResultSort{
					Field: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid sort order",
			query: &models.DriftResultQuery{
				Sort: models.DriftResultSort{
					Order: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDriftResult_HelperMethods(t *testing.T) {
	result := &models.DriftResult{
		ID:         "test-id",
		Status:     "completed",
		DriftCount: 2,
		Resources: []models.DriftedResource{
			{
				Address:   "aws_instance.test",
				DriftType: "modified",
				Severity:  "critical",
			},
			{
				Address:   "aws_s3_bucket.test",
				DriftType: "added",
				Severity:  "high",
			},
		},
	}

	// Test IsCompleted
	assert.True(t, result.IsCompleted())
	assert.False(t, result.IsFailed())
	assert.False(t, result.IsRunning())

	// Test HasDrift
	assert.True(t, result.HasDrift())

	// Test GetCriticalResources
	critical := result.GetCriticalResources()
	assert.Len(t, critical, 1)
	assert.Equal(t, "critical", critical[0].Severity)

	// Test GetHighSeverityResources
	high := result.GetHighSeverityResources()
	assert.Len(t, high, 2) // critical + high

	// Test CalculateSummary
	result.CalculateSummary()
	assert.Equal(t, 2, result.Summary.TotalResources)
	assert.Equal(t, 2, result.Summary.DriftedResources)
	assert.Equal(t, 1, result.Summary.CriticalDrift)
	assert.Equal(t, 1, result.Summary.HighDrift)
}

func TestDriftedResource_Validation(t *testing.T) {
	resource := models.DriftedResource{
		Address:    "aws_instance.test",
		Type:       "aws_instance",
		Provider:   "aws",
		Region:     "us-west-2",
		DriftType:  "modified",
		Severity:   "high",
		DetectedAt: time.Now(),
		Changes: []models.ResourceChange{
			{
				Field:      "instance_type",
				OldValue:   "t2.micro",
				NewValue:   "t2.small",
				ChangeType: "modified",
			},
		},
	}

	// Test basic validation
	assert.NotEmpty(t, resource.Address)
	assert.NotEmpty(t, resource.Type)
	assert.NotEmpty(t, resource.Provider)
	assert.NotEmpty(t, resource.DriftType)
	assert.NotEmpty(t, resource.Severity)
	assert.False(t, resource.DetectedAt.IsZero())
	assert.Len(t, resource.Changes, 1)
}

func TestResourceChange_Validation(t *testing.T) {
	change := models.ResourceChange{
		Field:      "instance_type",
		OldValue:   "t2.micro",
		NewValue:   "t2.small",
		ChangeType: "modified",
	}

	// Test basic validation
	assert.NotEmpty(t, change.Field)
	assert.NotNil(t, change.OldValue)
	assert.NotNil(t, change.NewValue)
	assert.NotEmpty(t, change.ChangeType)
	assert.NotEqual(t, change.OldValue, change.NewValue)
}

func TestDriftSummary_Validation(t *testing.T) {
	summary := models.DriftSummary{
		TotalResources:    10,
		DriftedResources:  3,
		AddedResources:    1,
		RemovedResources:  1,
		ModifiedResources: 1,
		CriticalDrift:     1,
		HighDrift:         1,
		MediumDrift:       1,
		LowDrift:          0,
	}

	// Test basic validation
	assert.Equal(t, 10, summary.TotalResources)
	assert.Equal(t, 3, summary.DriftedResources)
	assert.Equal(t, 1, summary.AddedResources)
	assert.Equal(t, 1, summary.RemovedResources)
	assert.Equal(t, 1, summary.ModifiedResources)
	assert.Equal(t, 1, summary.CriticalDrift)
	assert.Equal(t, 1, summary.HighDrift)
	assert.Equal(t, 1, summary.MediumDrift)
	assert.Equal(t, 0, summary.LowDrift)

	// Test consistency
	assert.Equal(t, summary.DriftedResources, summary.AddedResources+summary.RemovedResources+summary.ModifiedResources)
}

func TestDriftOptions_Validation(t *testing.T) {
	options := models.DriftOptions{
		Recursive:   true,
		IncludeTags: true,
		MaxDepth:    10,
		Timeout:     time.Minute * 5,
		Parallel:    true,
		DryRun:      false,
	}

	// Test basic validation
	assert.True(t, options.Recursive)
	assert.True(t, options.IncludeTags)
	assert.Equal(t, 10, options.MaxDepth)
	assert.Equal(t, time.Minute*5, options.Timeout)
	assert.True(t, options.Parallel)
	assert.False(t, options.DryRun)
}

func TestPaginatedDriftResults_Validation(t *testing.T) {
	results := models.PaginatedDriftResults{
		Results: []models.DriftResult{
			{ID: "1", Provider: "aws"},
			{ID: "2", Provider: "azure"},
		},
		Total:   2,
		Page:    1,
		PerPage: 50,
		Pages:   1,
	}

	// Test basic validation
	assert.Len(t, results.Results, 2)
	assert.Equal(t, 2, results.Total)
	assert.Equal(t, 1, results.Page)
	assert.Equal(t, 50, results.PerPage)
	assert.Equal(t, 1, results.Pages)
}

func TestDriftHistoryResponse_Validation(t *testing.T) {
	response := models.DriftHistoryResponse{
		Results: []models.DriftResult{
			{ID: "1", Provider: "aws"},
		},
		Total:  1,
		Limit:  50,
		Offset: 0,
	}

	// Test basic validation
	assert.Len(t, response.Results, 1)
	assert.Equal(t, 1, response.Total)
	assert.Equal(t, 50, response.Limit)
	assert.Equal(t, 0, response.Offset)
}

func TestDriftSummaryResponse_Validation(t *testing.T) {
	response := models.DriftSummaryResponse{
		Summary: models.DriftSummary{
			TotalResources:   10,
			DriftedResources: 3,
		},
		LastUpdated: time.Now(),
		Provider:    "aws",
	}

	// Test basic validation
	assert.Equal(t, 10, response.Summary.TotalResources)
	assert.Equal(t, 3, response.Summary.DriftedResources)
	assert.False(t, response.LastUpdated.IsZero())
	assert.Equal(t, "aws", response.Provider)
}

func TestDriftResultResponse_Validation(t *testing.T) {
	response := models.DriftResultResponse{
		ID:        "test-id",
		Status:    "completed",
		Message:   "Drift detection completed successfully",
		Timestamp: time.Now(),
	}

	// Test basic validation
	assert.Equal(t, "test-id", response.ID)
	assert.Equal(t, "completed", response.Status)
	assert.Equal(t, "Drift detection completed successfully", response.Message)
	assert.False(t, response.Timestamp.IsZero())
}
