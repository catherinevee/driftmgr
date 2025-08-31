package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RemediationJob represents a remediation job
type RemediationJob struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	Status       string                 `json:"status"` // pending, in_progress, completed, failed
	Action       string                 `json:"action"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	DriftType    string                 `json:"drift_type,omitempty"`
	StateValue   interface{}            `json:"state_value,omitempty"`
	ActualValue  interface{}            `json:"actual_value,omitempty"`
	RemediatedBy string                 `json:"remediated_by,omitempty"`
}

// RemediationStore manages remediation jobs
type RemediationStore struct {
	mu   sync.RWMutex
	jobs map[string]*RemediationJob
}

// NewRemediationStore creates a new remediation store
func NewRemediationStore() *RemediationStore {
	return &RemediationStore{
		jobs: make(map[string]*RemediationJob),
	}
}

// CreateJob creates a new remediation job
func (rs *RemediationStore) CreateJob(ctx context.Context, resourceID, resourceType, provider, region, action, driftType string) (*RemediationJob, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	job := &RemediationJob{
		ID:           fmt.Sprintf("rem-%s", uuid.New().String()[:8]),
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Provider:     provider,
		Region:       region,
		Status:       "pending",
		Action:       action,
		CreatedAt:    time.Now(),
		DriftType:    driftType,
		Details:      make(map[string]interface{}),
	}

	rs.jobs[job.ID] = job
	return job, nil
}

// GetJob retrieves a job by ID
func (rs *RemediationStore) GetJob(jobID string) (*RemediationJob, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	job, exists := rs.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

// GetAllJobs returns all remediation jobs
func (rs *RemediationStore) GetAllJobs() []*RemediationJob {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	jobs := make([]*RemediationJob, 0, len(rs.jobs))
	for _, job := range rs.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetJobsByStatus returns jobs with a specific status
func (rs *RemediationStore) GetJobsByStatus(status string) []*RemediationJob {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var jobs []*RemediationJob
	for _, job := range rs.jobs {
		if job.Status == status {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// GetJobsByResource returns jobs for a specific resource
func (rs *RemediationStore) GetJobsByResource(resourceID string) []*RemediationJob {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var jobs []*RemediationJob
	for _, job := range rs.jobs {
		if job.ResourceID == resourceID {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// UpdateJobStatus updates the status of a job
func (rs *RemediationStore) UpdateJobStatus(jobID, status string, error string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	job, exists := rs.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Status = status
	now := time.Now()

	switch status {
	case "in_progress":
		job.StartedAt = &now
	case "completed", "failed":
		job.CompletedAt = &now
		if error != "" {
			job.Error = error
		}
	}

	return nil
}

// AddJobDetails adds details to a job
func (rs *RemediationStore) AddJobDetails(jobID string, key string, value interface{}) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	job, exists := rs.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Details == nil {
		job.Details = make(map[string]interface{})
	}
	job.Details[key] = value

	return nil
}

// GetJobsSummary returns a summary of job statuses
func (rs *RemediationStore) GetJobsSummary() map[string]int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	summary := map[string]int{
		"pending":     0,
		"in_progress": 0,
		"completed":   0,
		"failed":      0,
		"total":       0,
	}

	for _, job := range rs.jobs {
		summary[job.Status]++
		summary["total"]++
	}

	return summary
}

// CleanupOldJobs removes jobs older than the specified duration
func (rs *RemediationStore) CleanupOldJobs(maxAge time.Duration) int {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, job := range rs.jobs {
		if job.CreatedAt.Before(cutoff) && (job.Status == "completed" || job.Status == "failed") {
			delete(rs.jobs, id)
			removed++
		}
	}

	return removed
}

// GetRecentJobs returns the most recent jobs
func (rs *RemediationStore) GetRecentJobs(limit int) []*RemediationJob {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	// Convert map to slice for sorting
	jobs := make([]*RemediationJob, 0, len(rs.jobs))
	for _, job := range rs.jobs {
		jobs = append(jobs, job)
	}

	// Sort by created_at descending (most recent first)
	for i := 0; i < len(jobs)-1; i++ {
		for j := i + 1; j < len(jobs); j++ {
			if jobs[j].CreatedAt.After(jobs[i].CreatedAt) {
				jobs[i], jobs[j] = jobs[j], jobs[i]
			}
		}
	}

	// Return limited results
	if limit > 0 && limit < len(jobs) {
		return jobs[:limit]
	}
	return jobs
}