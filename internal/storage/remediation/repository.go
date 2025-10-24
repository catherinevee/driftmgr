package remediation

import (
	"context"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Repository defines the interface for remediation data operations
type Repository interface {
	// Job management
	CreateJob(ctx context.Context, job *models.RemediationJob) error
	GetJobByID(ctx context.Context, id string) (*models.RemediationJob, error)
	UpdateJob(ctx context.Context, job *models.RemediationJob) error
	DeleteJob(ctx context.Context, id string) error
	ListJobs(ctx context.Context, req *models.RemediationJobListRequest) (*models.RemediationJobListResponse, error)

	// Job status management
	UpdateJobStatus(ctx context.Context, id string, status models.JobStatus) error
	UpdateJobProgress(ctx context.Context, id string, progress models.JobProgress) error
	AddJobLog(ctx context.Context, log *models.JobLog) error
	GetJobLogs(ctx context.Context, jobID string) ([]models.JobLog, error)

	// Job filtering and search
	GetJobsByStatus(ctx context.Context, status models.JobStatus) ([]models.RemediationJob, error)
	GetJobsByPriority(ctx context.Context, priority models.JobPriority) ([]models.RemediationJob, error)
	GetJobsByUser(ctx context.Context, userID string) ([]models.RemediationJob, error)
	GetJobsByStrategy(ctx context.Context, strategyType models.StrategyType) ([]models.RemediationJob, error)
	GetJobsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.RemediationJob, error)

	// Strategy management
	CreateStrategy(ctx context.Context, strategy *models.RemediationStrategy) error
	GetStrategyByID(ctx context.Context, id string) (*models.RemediationStrategy, error)
	GetStrategyByName(ctx context.Context, name string) (*models.RemediationStrategy, error)
	UpdateStrategy(ctx context.Context, strategy *models.RemediationStrategy) error
	DeleteStrategy(ctx context.Context, id string) error
	ListStrategies(ctx context.Context) ([]models.RemediationStrategy, error)
	GetStrategiesByType(ctx context.Context, strategyType models.StrategyType) ([]models.RemediationStrategy, error)

	// History and analytics
	GetRemediationHistory(ctx context.Context, req *models.RemediationHistoryRequest) (*models.RemediationHistoryResponse, error)
	GetJobStatistics(ctx context.Context, startDate, endDate time.Time) (*models.JobStatistics, error)
	GetSuccessRate(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetAverageJobDuration(ctx context.Context, startDate, endDate time.Time) (time.Duration, error)

	// Cleanup and maintenance
	DeleteOldJobs(ctx context.Context, olderThan time.Time) error
	DeleteOldLogs(ctx context.Context, olderThan time.Time) error
	CleanupCompletedJobs(ctx context.Context, olderThan time.Time) error

	// Health and monitoring
	Health(ctx context.Context) error
	GetQueueDepth(ctx context.Context) (int, error)
	GetActiveJobsCount(ctx context.Context) (int, error)
	GetFailedJobsCount(ctx context.Context) (int, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(context.Context) error) error

	// Close connection
	Close() error
}

// RepositoryConfig holds configuration for the repository
type RepositoryConfig struct {
	DatabaseURL    string        `json:"database_url"`
	MaxConns       int           `json:"max_conns"`
	MinConns       int           `json:"min_conns"`
	MaxLifetime    time.Duration `json:"max_lifetime"`
	MaxIdleTime    time.Duration `json:"max_idle_time"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
	QueryTimeout   time.Duration `json:"query_timeout"`
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

// RepositoryFactory creates repository instances
type RepositoryFactory interface {
	CreateRepository(config *RepositoryConfig) (Repository, error)
}

// DefaultRepositoryFactory is the default implementation of RepositoryFactory
type DefaultRepositoryFactory struct{}

// CreateRepository creates a new repository instance
func (f *DefaultRepositoryFactory) CreateRepository(config *RepositoryConfig) (Repository, error) {
	return NewPostgresRepository(config)
}
