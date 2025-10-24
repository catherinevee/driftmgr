package remediation

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/storage/remediation"
)

// ServiceConfig holds configuration for the remediation service
type ServiceConfig struct {
	MaxConcurrentJobs      int           `json:"max_concurrent_jobs"`
	DefaultTimeout         time.Duration `json:"default_timeout"`
	MaxRetryCount          int           `json:"max_retry_count"`
	RetentionDays          int           `json:"retention_days"`
	EnableAutoCleanup      bool          `json:"enable_auto_cleanup"`
	CleanupInterval        time.Duration `json:"cleanup_interval"`
	RequireApproval        bool          `json:"require_approval"`
	DefaultPriority        string        `json:"default_priority"`
	MaxJobHistory          int           `json:"max_job_history"`
	EnableProgressTracking bool          `json:"enable_progress_tracking"`
}

// Service implements the business logic for remediation management
type Service struct {
	repo   remediation.Repository
	config *ServiceConfig
}

// NewService creates a new remediation service
func NewService(repo remediation.Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = &ServiceConfig{
			MaxConcurrentJobs:      10,
			DefaultTimeout:         time.Hour,
			MaxRetryCount:          3,
			RetentionDays:          30,
			EnableAutoCleanup:      true,
			CleanupInterval:        time.Hour * 24,
			RequireApproval:        false,
			DefaultPriority:        "medium",
			MaxJobHistory:          1000,
			EnableProgressTracking: true,
		}
	}

	return &Service{
		repo:   repo,
		config: config,
	}
}

// CreateJob creates a new remediation job
func (s *Service) CreateJob(ctx context.Context, req *models.RemediationJobRequest) (*models.RemediationJob, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if drift result exists
	// This would typically check against the drift results repository
	// For now, we'll assume it exists

	// Create job
	job := &models.RemediationJob{
		ID:               generateJobID(),
		DriftResultID:    req.DriftResultID,
		Strategy:         req.Strategy,
		Status:           models.JobStatusPending,
		Priority:         req.Priority,
		CreatedBy:        getCurrentUser(ctx), // This would get from context
		DryRun:           req.DryRun,
		RequiresApproval: req.RequiresApproval || s.config.RequireApproval,
		Configuration:    req.Configuration,
		Progress: models.JobProgress{
			TotalResources:      0,
			ProcessedResources:  0,
			SuccessfulResources: 0,
			FailedResources:     0,
			Percentage:          0,
			CurrentStep:         "Initializing",
			LastUpdate:          time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Validate job
	if err := job.Validate(); err != nil {
		return nil, fmt.Errorf("invalid job: %w", err)
	}

	// Save to repository
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Add initial log
	job.AddLog(models.LogLevelInfo, "Remediation job created", map[string]interface{}{
		"strategy": job.Strategy.Type,
		"priority": job.Priority,
		"dry_run":  job.DryRun,
	})

	// Save log
	if err := s.repo.AddJobLog(ctx, &job.Logs[len(job.Logs)-1]); err != nil {
		// Log error but don't fail the job creation
		fmt.Printf("Failed to save initial log: %v\n", err)
	}

	return job, nil
}

// GetJob retrieves a remediation job by ID
func (s *Service) GetJob(ctx context.Context, id string) (*models.RemediationJob, error) {
	if id == "" {
		return nil, models.ErrBadRequest
	}

	job, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// ListJobs lists remediation jobs with filtering
func (s *Service) ListJobs(ctx context.Context, req *models.RemediationJobListRequest) (*models.RemediationJobListResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > s.config.MaxJobHistory {
		req.Limit = s.config.MaxJobHistory
	}
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Get jobs from repository
	response, err := s.repo.ListJobs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return response, nil
}

// UpdateJobStatus updates the status of a remediation job
func (s *Service) UpdateJobStatus(ctx context.Context, id string, status models.JobStatus) error {
	if id == "" {
		return models.ErrBadRequest
	}

	// Validate status
	if !isValidJobStatus(status) {
		return models.ErrInvalidJobStatus
	}

	// Get current job
	job, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if status transition is valid
	if !isValidStatusTransition(job.Status, status) {
		return fmt.Errorf("invalid status transition from %s to %s", job.Status, status)
	}

	// Update status
	if err := s.repo.UpdateJobStatus(ctx, id, status); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update timestamps based on status
	now := time.Now()
	switch status {
	case models.JobStatusRunning:
		job.StartedAt = &now
	case models.JobStatusCompleted, models.JobStatusFailed, models.JobStatusCancelled:
		job.CompletedAt = &now
	}

	// Update job
	job.Status = status
	job.UpdatedAt = now
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Add log
	job.AddLog(models.LogLevelInfo, fmt.Sprintf("Job status changed to %s", status), nil)
	if err := s.repo.AddJobLog(ctx, &job.Logs[len(job.Logs)-1]); err != nil {
		fmt.Printf("Failed to save status change log: %v\n", err)
	}

	return nil
}

// UpdateJobProgress updates the progress of a remediation job
func (s *Service) UpdateJobProgress(ctx context.Context, id string, progress models.JobProgress) error {
	if id == "" {
		return models.ErrBadRequest
	}

	// Validate progress
	if progress.TotalResources < 0 || progress.ProcessedResources < 0 {
		return models.ErrBadRequest
	}

	// Update progress
	if err := s.repo.UpdateJobProgress(ctx, id, progress); err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// CancelJob cancels a remediation job
func (s *Service) CancelJob(ctx context.Context, id string, reason string) error {
	if id == "" {
		return models.ErrBadRequest
	}

	// Get current job
	job, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if job can be cancelled
	if !job.CanBeCancelled() {
		return models.ErrJobCannotBeCancelled
	}

	// Update status to cancelled
	if err := s.UpdateJobStatus(ctx, id, models.JobStatusCancelled); err != nil {
		return err
	}

	// Add cancellation log
	job.AddLog(models.LogLevelInfo, "Job cancelled", map[string]interface{}{
		"reason": reason,
	})
	if err := s.repo.AddJobLog(ctx, &job.Logs[len(job.Logs)-1]); err != nil {
		fmt.Printf("Failed to save cancellation log: %v\n", err)
	}

	return nil
}

// ApproveJob approves a remediation job
func (s *Service) ApproveJob(ctx context.Context, id string, approved bool, comments string) error {
	if id == "" {
		return models.ErrBadRequest
	}

	// Get current job
	job, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if job requires approval
	if !job.NeedsApproval() {
		return models.ErrJobNotApproved
	}

	// Check if already approved
	if job.ApprovedBy != nil {
		return models.ErrJobAlreadyApproved
	}

	// Update approval
	now := time.Now()
	user := getCurrentUser(ctx)

	job.ApprovedBy = &user
	job.ApprovedAt = &now
	job.UpdatedAt = now

	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job approval: %w", err)
	}

	// Add approval log
	status := "rejected"
	if approved {
		status = "approved"
	}

	job.AddLog(models.LogLevelInfo, fmt.Sprintf("Job %s", status), map[string]interface{}{
		"approved_by": user,
		"comments":    comments,
	})
	if err := s.repo.AddJobLog(ctx, &job.Logs[len(job.Logs)-1]); err != nil {
		fmt.Printf("Failed to save approval log: %v\n", err)
	}

	return nil
}

// GetRemediationHistory retrieves remediation history
func (s *Service) GetRemediationHistory(ctx context.Context, req *models.RemediationHistoryRequest) (*models.RemediationHistoryResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > s.config.MaxJobHistory {
		req.Limit = s.config.MaxJobHistory
	}

	// Get history from repository
	response, err := s.repo.GetRemediationHistory(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get remediation history: %w", err)
	}

	return response, nil
}

// GetJobStatistics retrieves job statistics
func (s *Service) GetJobStatistics(ctx context.Context, startDate, endDate time.Time) (*models.JobStatistics, error) {
	stats, err := s.repo.GetJobStatistics(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get job statistics: %w", err)
	}

	return stats, nil
}

// CreateStrategy creates a new remediation strategy
func (s *Service) CreateStrategy(ctx context.Context, req *models.RemediationStrategyRequest) (*models.RemediationStrategy, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if strategy with same name exists
	existing, err := s.repo.GetStrategyByName(ctx, req.Name)
	if err == nil && existing != nil {
		return nil, models.ErrRemediationStrategyExists
	}

	// Create strategy
	strategy := &models.RemediationStrategy{
		ID:          generateStrategyID(),
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		Parameters:  req.Parameters,
		Timeout:     req.Timeout,
		RetryCount:  req.RetryCount,
		IsCustom:    true,
		CreatedBy:   getCurrentUser(ctx),
		CreatedAt:   time.Now(),
	}

	// Validate strategy
	if err := strategy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid strategy: %w", err)
	}

	// Save to repository
	if err := s.repo.CreateStrategy(ctx, strategy); err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}

	return strategy, nil
}

// ListStrategies lists all remediation strategies
func (s *Service) ListStrategies(ctx context.Context) ([]models.RemediationStrategy, error) {
	strategies, err := s.repo.ListStrategies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list strategies: %w", err)
	}

	return strategies, nil
}

// GetStrategy retrieves a remediation strategy by ID
func (s *Service) GetStrategy(ctx context.Context, id string) (*models.RemediationStrategy, error) {
	if id == "" {
		return nil, models.ErrBadRequest
	}

	strategy, err := s.repo.GetStrategyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return strategy, nil
}

// Health checks the health of the remediation service
func (s *Service) Health(ctx context.Context) error {
	// Check repository health
	if err := s.repo.Health(ctx); err != nil {
		return fmt.Errorf("repository health check failed: %w", err)
	}

	// Check queue depth
	queueDepth, err := s.repo.GetQueueDepth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get queue depth: %w", err)
	}

	// Check if queue is too deep
	if queueDepth > s.config.MaxConcurrentJobs*2 {
		return fmt.Errorf("queue depth too high: %d", queueDepth)
	}

	return nil
}

// Helper functions

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job-%d-%s", time.Now().UnixNano(), randomString(8))
}

// generateStrategyID generates a unique strategy ID
func generateStrategyID() string {
	return fmt.Sprintf("strategy-%d-%s", time.Now().UnixNano(), randomString(8))
}

// getCurrentUser gets the current user from context
func getCurrentUser(ctx context.Context) string {
	// This would typically extract user from JWT token or session
	// For now, return a default user
	if user := ctx.Value("user_id"); user != nil {
		return user.(string)
	}
	return "system"
}

// isValidJobStatus checks if a job status is valid
func isValidJobStatus(status models.JobStatus) bool {
	validStatuses := []models.JobStatus{
		models.JobStatusPending,
		models.JobStatusQueued,
		models.JobStatusRunning,
		models.JobStatusCompleted,
		models.JobStatusFailed,
		models.JobStatusCancelled,
	}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

// isValidStatusTransition checks if a status transition is valid
func isValidStatusTransition(from, to models.JobStatus) bool {
	validTransitions := map[models.JobStatus][]models.JobStatus{
		models.JobStatusPending:   {models.JobStatusQueued, models.JobStatusCancelled},
		models.JobStatusQueued:    {models.JobStatusRunning, models.JobStatusCancelled},
		models.JobStatusRunning:   {models.JobStatusCompleted, models.JobStatusFailed, models.JobStatusCancelled},
		models.JobStatusCompleted: {},
		models.JobStatusFailed:    {models.JobStatusQueued}, // Allow retry
		models.JobStatusCancelled: {},
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedTransition := range allowedTransitions {
		if to == allowedTransition {
			return true
		}
	}
	return false
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
