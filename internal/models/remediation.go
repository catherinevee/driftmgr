package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// RemediationJob represents a remediation job in the system
type RemediationJob struct {
	ID               string                 `json:"id" db:"id" validate:"required,uuid"`
	DriftResultID    string                 `json:"drift_result_id" db:"drift_result_id" validate:"required,uuid"`
	Strategy         RemediationStrategy    `json:"strategy" db:"strategy" validate:"required"`
	Status           JobStatus              `json:"status" db:"status" validate:"required,oneof=pending queued running completed failed cancelled"`
	Priority         JobPriority            `json:"priority" db:"priority" validate:"required,oneof=low medium high critical"`
	CreatedBy        string                 `json:"created_by" db:"created_by" validate:"required"`
	ApprovedBy       *string                `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt       *time.Time             `json:"approved_at,omitempty" db:"approved_at"`
	StartedAt        *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Progress         RemediationJobProgress `json:"progress" db:"progress"`
	Configuration    map[string]interface{} `json:"configuration" db:"configuration"`
	DryRun           bool                   `json:"dry_run" db:"dry_run"`
	RequiresApproval bool                   `json:"requires_approval" db:"requires_approval"`
	Error            *string                `json:"error,omitempty" db:"error"`
	Logs             []JobLog               `json:"logs,omitempty" db:"logs"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// RemediationStrategy defines the type of remediation to perform
type RemediationStrategy struct {
	ID          string                 `json:"id,omitempty" db:"id"`
	Type        StrategyType           `json:"type" validate:"required,oneof=terraform_apply terraform_destroy terraform_import state_manipulation resource_creation resource_deletion"`
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description" validate:"max=1000"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty" validate:"min=0,max=10"`
	IsCustom    bool                   `json:"is_custom"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
}

// JobStatus is defined in common.go

// JobPriority represents the priority level of a remediation job
type JobPriority string

const (
	JobPriorityLow      JobPriority = "low"
	JobPriorityMedium   JobPriority = "medium"
	JobPriorityHigh     JobPriority = "high"
	JobPriorityCritical JobPriority = "critical"
)

// StrategyType represents the type of remediation strategy
type StrategyType string

const (
	StrategyTypeTerraformApply    StrategyType = "terraform_apply"
	StrategyTypeTerraformDestroy  StrategyType = "terraform_destroy"
	StrategyTypeTerraformImport   StrategyType = "terraform_import"
	StrategyTypeStateManipulation StrategyType = "state_manipulation"
	StrategyTypeResourceCreation  StrategyType = "resource_creation"
	StrategyTypeResourceDeletion  StrategyType = "resource_deletion"
)

// JobProgress tracks the progress of a remediation job
// RemediationJobProgress represents the progress of a remediation job
type RemediationJobProgress struct {
	TotalResources      int            `json:"total_resources"`
	ProcessedResources  int            `json:"processed_resources"`
	SuccessfulResources int            `json:"successful_resources"`
	FailedResources     int            `json:"failed_resources"`
	Percentage          float64        `json:"percentage"`
	CurrentStep         string         `json:"current_step"`
	EstimatedTime       *time.Duration `json:"estimated_time,omitempty"`
	StartTime           *time.Time     `json:"start_time,omitempty"`
	LastUpdate          time.Time      `json:"last_update"`
}

// JobLog represents a log entry for a remediation job
type JobLog struct {
	ID        string                 `json:"id" db:"id"`
	JobID     string                 `json:"job_id" db:"job_id"`
	Level     LogLevel               `json:"level" db:"level" validate:"required,oneof=debug info warn error"`
	Message   string                 `json:"message" db:"message" validate:"required"`
	Details   map[string]interface{} `json:"details,omitempty" db:"details"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
}

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// RemediationJobRequest represents a request to create a remediation job
type RemediationJobRequest struct {
	DriftResultID    string                 `json:"drift_result_id" validate:"required,uuid"`
	Strategy         RemediationStrategy    `json:"strategy" validate:"required"`
	Priority         JobPriority            `json:"priority" validate:"required,oneof=low medium high critical"`
	DryRun           bool                   `json:"dry_run"`
	RequiresApproval bool                   `json:"requires_approval"`
	Configuration    map[string]interface{} `json:"configuration,omitempty"`
}

// RemediationJobResponse represents the response for a remediation job
type RemediationJobResponse struct {
	ID            string                 `json:"id"`
	Status        JobStatus              `json:"status"`
	Progress      RemediationJobProgress `json:"progress"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	EstimatedTime *time.Duration         `json:"estimated_time,omitempty"`
	Error         *string                `json:"error,omitempty"`
}

// RemediationJobListRequest represents a request to list remediation jobs
type RemediationJobListRequest struct {
	Status       *JobStatus    `json:"status,omitempty" validate:"omitempty,oneof=pending queued running completed failed cancelled"`
	Priority     *JobPriority  `json:"priority,omitempty" validate:"omitempty,oneof=low medium high critical"`
	CreatedBy    *string       `json:"created_by,omitempty"`
	StrategyType *StrategyType `json:"strategy_type,omitempty" validate:"omitempty,oneof=terraform_apply terraform_destroy terraform_import state_manipulation resource_creation resource_deletion"`
	StartDate    *time.Time    `json:"start_date,omitempty"`
	EndDate      *time.Time    `json:"end_date,omitempty"`
	Limit        int           `json:"limit" validate:"min=1,max=1000"`
	Offset       int           `json:"offset" validate:"min=0"`
	SortBy       string        `json:"sort_by" validate:"omitempty,oneof=created_at updated_at status priority"`
	SortOrder    string        `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// RemediationJobListResponse represents the response for listing remediation jobs
type RemediationJobListResponse struct {
	Jobs   []RemediationJob `json:"jobs"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// RemediationStrategyRequest represents a request to create a remediation strategy
type RemediationStrategyRequest struct {
	Type        StrategyType           `json:"type" validate:"required,oneof=terraform_apply terraform_destroy terraform_import state_manipulation resource_creation resource_deletion"`
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description" validate:"max=1000"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty" validate:"min=0,max=10"`
}

// RemediationStrategyResponse represents the response for a remediation strategy
type RemediationStrategyResponse struct {
	ID          string                 `json:"id"`
	Type        StrategyType           `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
	IsCustom    bool                   `json:"is_custom"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
}

// RemediationStrategyListResponse represents the response for listing remediation strategies
type RemediationStrategyListResponse struct {
	Strategies []RemediationStrategyResponse `json:"strategies"`
	Total      int                           `json:"total"`
}

// RemediationHistoryRequest represents a request to get remediation history
type RemediationHistoryRequest struct {
	StartDate *time.Time    `json:"start_date,omitempty"`
	EndDate   *time.Time    `json:"end_date,omitempty"`
	Status    *JobStatus    `json:"status,omitempty" validate:"omitempty,oneof=pending queued running completed failed cancelled"`
	Strategy  *StrategyType `json:"strategy,omitempty" validate:"omitempty,oneof=terraform_apply terraform_destroy terraform_import state_manipulation resource_creation resource_deletion"`
	Limit     int           `json:"limit" validate:"min=1,max=1000"`
	Offset    int           `json:"offset" validate:"min=0"`
}

// RemediationHistoryResponse represents the response for remediation history
type RemediationHistoryResponse struct {
	Jobs   []RemediationJob `json:"jobs"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// JobProgressUpdate represents a progress update for a remediation job
type JobProgressUpdate struct {
	JobID               string         `json:"job_id" validate:"required,uuid"`
	TotalResources      int            `json:"total_resources" validate:"min=0"`
	ProcessedResources  int            `json:"processed_resources" validate:"min=0"`
	SuccessfulResources int            `json:"successful_resources" validate:"min=0"`
	FailedResources     int            `json:"failed_resources" validate:"min=0"`
	CurrentStep         string         `json:"current_step"`
	EstimatedTime       *time.Duration `json:"estimated_time,omitempty"`
}

// JobCancelRequest represents a request to cancel a remediation job
type JobCancelRequest struct {
	Reason string `json:"reason" validate:"max=500"`
}

// JobCancelResponse represents the response for cancelling a remediation job
type JobCancelResponse struct {
	JobID       string    `json:"job_id"`
	Status      JobStatus `json:"status"`
	CancelledAt time.Time `json:"cancelled_at"`
	Reason      string    `json:"reason"`
}

// ApprovalRequest represents a request for job approval
type ApprovalRequest struct {
	JobID    string `json:"job_id" validate:"required,uuid"`
	Approved bool   `json:"approved" validate:"required"`
	Comments string `json:"comments" validate:"max=1000"`
}

// ApprovalResponse represents the response for job approval
type ApprovalResponse struct {
	JobID      string    `json:"job_id"`
	Approved   bool      `json:"approved"`
	ApprovedBy string    `json:"approved_by"`
	ApprovedAt time.Time `json:"approved_at"`
	Comments   string    `json:"comments"`
}

// Validation methods

// Validate validates the RemediationJob struct
func (rj *RemediationJob) Validate() error {
	validate := validator.New()
	return validate.Struct(rj)
}

// Validate validates the RemediationStrategy struct
func (rs *RemediationStrategy) Validate() error {
	validate := validator.New()
	return validate.Struct(rs)
}

// Validate validates the RemediationJobRequest struct
func (rjr *RemediationJobRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rjr)
}

// Validate validates the RemediationJobListRequest struct
func (rjlr *RemediationJobListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rjlr)
}

// Validate validates the RemediationStrategyRequest struct
func (rsr *RemediationStrategyRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rsr)
}

// Validate validates the RemediationHistoryRequest struct
func (rhr *RemediationHistoryRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rhr)
}

// Validate validates the JobProgressUpdate struct
func (jpu *JobProgressUpdate) Validate() error {
	validate := validator.New()
	return validate.Struct(jpu)
}

// Validate validates the JobCancelRequest struct
func (jcr *JobCancelRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(jcr)
}

// Validate validates the ApprovalRequest struct
func (ar *ApprovalRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(ar)
}

// Validate validates the JobLog struct
func (jl *JobLog) Validate() error {
	validate := validator.New()
	return validate.Struct(jl)
}

// Helper methods

// IsCompleted returns true if the job is in a completed state
func (rj *RemediationJob) IsCompleted() bool {
	return rj.Status == JobStatusCompleted || rj.Status == JobStatusFailed || rj.Status == JobStatusCancelled
}

// IsRunning returns true if the job is currently running
func (rj *RemediationJob) IsRunning() bool {
	return rj.Status == JobStatusRunning
}

// CanBeCancelled returns true if the job can be cancelled
func (rj *RemediationJob) CanBeCancelled() bool {
	return rj.Status == JobStatusPending || rj.Status == JobStatusQueued || rj.Status == JobStatusRunning
}

// NeedsApproval returns true if the job requires approval
func (rj *RemediationJob) NeedsApproval() bool {
	return rj.RequiresApproval && rj.ApprovedBy == nil
}

// UpdateProgress updates the job progress
func (rj *RemediationJob) UpdateProgress(progress RemediationJobProgress) {
	rj.Progress = progress
	rj.UpdatedAt = time.Now()
}

// AddLog adds a log entry to the job
func (rj *RemediationJob) AddLog(level LogLevel, message string, details map[string]interface{}) {
	log := JobLog{
		ID:        generateID(),
		JobID:     rj.ID,
		Level:     level,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}
	rj.Logs = append(rj.Logs, log)
}

// CalculateProgressPercentage calculates the progress percentage
func (jp *RemediationJobProgress) CalculateProgressPercentage() float64 {
	if jp.TotalResources == 0 {
		return 0
	}
	return float64(jp.ProcessedResources) / float64(jp.TotalResources) * 100
}

// UpdateProgress updates the progress with new values
func (jp *RemediationJobProgress) UpdateProgress(processed, successful, failed int) {
	jp.ProcessedResources = processed
	jp.SuccessfulResources = successful
	jp.FailedResources = failed
	jp.Percentage = jp.CalculateProgressPercentage()
	jp.LastUpdate = time.Now()
}

// Helper function to generate unique IDs
func generateID() string {
	// This would typically use a UUID generator
	// For now, we'll use a simple timestamp-based ID
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// JobStatistics represents statistics about remediation jobs
type JobStatistics struct {
	TotalJobs       int                `json:"total_jobs"`
	CompletedJobs   int                `json:"completed_jobs"`
	FailedJobs      int                `json:"failed_jobs"`
	CancelledJobs   int                `json:"cancelled_jobs"`
	PendingJobs     int                `json:"pending_jobs"`
	RunningJobs     int                `json:"running_jobs"`
	SuccessRate     float64            `json:"success_rate"`
	AverageDuration time.Duration      `json:"average_duration"`
	JobsByStatus    map[string]int     `json:"jobs_by_status"`
	JobsByPriority  map[string]int     `json:"jobs_by_priority"`
	JobsByStrategy  map[string]int     `json:"jobs_by_strategy"`
	TopUsers        []UserJobCount     `json:"top_users"`
	TopStrategies   []StrategyJobCount `json:"top_strategies"`
	DailyJobCounts  []DailyJobCount    `json:"daily_job_counts"`
	HourlyJobCounts []HourlyJobCount   `json:"hourly_job_counts"`
}

// UserJobCount represents job count by user
type UserJobCount struct {
	UserID      string  `json:"user_id"`
	JobCount    int     `json:"job_count"`
	SuccessRate float64 `json:"success_rate"`
}

// StrategyJobCount represents job count by strategy
type StrategyJobCount struct {
	StrategyType string  `json:"strategy_type"`
	JobCount     int     `json:"job_count"`
	SuccessRate  float64 `json:"success_rate"`
}

// DailyJobCount represents job count by day
type DailyJobCount struct {
	Date         time.Time `json:"date"`
	JobCount     int       `json:"job_count"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
}

// HourlyJobCount represents job count by hour
type HourlyJobCount struct {
	Hour         int `json:"hour"`
	JobCount     int `json:"job_count"`
	SuccessCount int `json:"success_count"`
	FailureCount int `json:"failure_count"`
}

// Helper function to generate random strings
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
