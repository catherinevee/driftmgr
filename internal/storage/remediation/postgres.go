package remediation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// PostgresRepository implements the Repository interface using PostgreSQL
type PostgresRepository struct {
	db     *sqlx.DB
	config *RepositoryConfig
}

// NewPostgresRepository creates a new PostgreSQL repository instance
func NewPostgresRepository(config *RepositoryConfig) (*PostgresRepository, error) {
	db, err := sqlx.Connect("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxConns)
	db.SetMaxIdleConns(config.MinConns)
	db.SetConnMaxLifetime(config.MaxLifetime)
	db.SetConnMaxIdleTime(config.MaxIdleTime)

	repo := &PostgresRepository{
		db:     db,
		config: config,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	if err := repo.Health(ctx); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	return repo, nil
}

// CreateJob creates a new remediation job
func (r *PostgresRepository) CreateJob(ctx context.Context, job *models.RemediationJob) error {
	query := `
		INSERT INTO remediation_jobs (
			id, drift_result_id, strategy, status, priority, created_by, 
			approved_by, approved_at, started_at, completed_at, progress,
			configuration, dry_run, requires_approval, error, created_at, updated_at
		) VALUES (
			:id, :drift_result_id, :strategy, :status, :priority, :created_by,
			:approved_by, :approved_at, :started_at, :completed_at, :progress,
			:configuration, :dry_run, :requires_approval, :error, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, job)
	return err
}

// GetJobByID retrieves a remediation job by ID
func (r *PostgresRepository) GetJobByID(ctx context.Context, id string) (*models.RemediationJob, error) {
	query := `
		SELECT id, drift_result_id, strategy, status, priority, created_by,
			   approved_by, approved_at, started_at, completed_at, progress,
			   configuration, dry_run, requires_approval, error, created_at, updated_at
		FROM remediation_jobs
		WHERE id = $1`

	var job models.RemediationJob
	err := r.db.GetContext(ctx, &job, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrRemediationJobNotFound
		}
		return nil, err
	}

	// Load job logs
	logs, err := r.GetJobLogs(ctx, id)
	if err != nil {
		return nil, err
	}
	job.Logs = logs

	return &job, nil
}

// UpdateJob updates an existing remediation job
func (r *PostgresRepository) UpdateJob(ctx context.Context, job *models.RemediationJob) error {
	query := `
		UPDATE remediation_jobs SET
			drift_result_id = :drift_result_id,
			strategy = :strategy,
			status = :status,
			priority = :priority,
			created_by = :created_by,
			approved_by = :approved_by,
			approved_at = :approved_at,
			started_at = :started_at,
			completed_at = :completed_at,
			progress = :progress,
			configuration = :configuration,
			dry_run = :dry_run,
			requires_approval = :requires_approval,
			error = :error,
			updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, job)
	return err
}

// DeleteJob deletes a remediation job
func (r *PostgresRepository) DeleteJob(ctx context.Context, id string) error {
	// Delete job logs first
	_, err := r.db.ExecContext(ctx, "DELETE FROM remediation_job_logs WHERE job_id = $1", id)
	if err != nil {
		return err
	}

	// Delete the job
	_, err = r.db.ExecContext(ctx, "DELETE FROM remediation_jobs WHERE id = $1", id)
	return err
}

// ListJobs lists remediation jobs with filtering and pagination
func (r *PostgresRepository) ListJobs(ctx context.Context, req *models.RemediationJobListRequest) (*models.RemediationJobListResponse, error) {
	// Build query with filters
	query := "SELECT * FROM remediation_jobs WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if req.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *req.Status)
		argIndex++
	}

	if req.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, *req.Priority)
		argIndex++
	}

	if req.CreatedBy != nil {
		query += fmt.Sprintf(" AND created_by = $%d", argIndex)
		args = append(args, *req.CreatedBy)
		argIndex++
	}

	if req.StrategyType != nil {
		query += fmt.Sprintf(" AND strategy->>'type' = $%d", argIndex)
		args = append(args, *req.StrategyType)
		argIndex++
	}

	if req.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *req.StartDate)
		argIndex++
	}

	if req.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *req.EndDate)
		argIndex++
	}

	// Add sorting
	if req.SortBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", req.SortBy)
		if req.SortOrder == "desc" {
			query += " DESC"
		}
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.Limit, req.Offset)

	var jobs []models.RemediationJob
	err := r.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM remediation_jobs WHERE 1=1"
	countArgs := args[:len(args)-2] // Remove limit and offset

	if req.Status != nil {
		countQuery += " AND status = $1"
	}
	if req.Priority != nil {
		countQuery += " AND priority = $2"
	}
	// ... add other filters

	var total int
	err = r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, err
	}

	return &models.RemediationJobListResponse{
		Jobs:   jobs,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

// UpdateJobStatus updates the status of a remediation job
func (r *PostgresRepository) UpdateJobStatus(ctx context.Context, id string, status models.JobStatus) error {
	query := "UPDATE remediation_jobs SET status = $1, updated_at = $2 WHERE id = $3"
	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

// UpdateJobProgress updates the progress of a remediation job
func (r *PostgresRepository) UpdateJobProgress(ctx context.Context, id string, progress models.JobProgress) error {
	query := "UPDATE remediation_jobs SET progress = $1, updated_at = $2 WHERE id = $3"
	_, err := r.db.ExecContext(ctx, query, progress, time.Now(), id)
	return err
}

// AddJobLog adds a log entry to a remediation job
func (r *PostgresRepository) AddJobLog(ctx context.Context, log *models.JobLog) error {
	query := `
		INSERT INTO remediation_job_logs (id, job_id, level, message, details, timestamp)
		VALUES (:id, :job_id, :level, :message, :details, :timestamp)`

	_, err := r.db.NamedExecContext(ctx, query, log)
	return err
}

// GetJobLogs retrieves all logs for a remediation job
func (r *PostgresRepository) GetJobLogs(ctx context.Context, jobID string) ([]models.JobLog, error) {
	query := `
		SELECT id, job_id, level, message, details, timestamp
		FROM remediation_job_logs
		WHERE job_id = $1
		ORDER BY timestamp ASC`

	var logs []models.JobLog
	err := r.db.SelectContext(ctx, &logs, query, jobID)
	return logs, err
}

// GetJobsByStatus retrieves jobs by status
func (r *PostgresRepository) GetJobsByStatus(ctx context.Context, status models.JobStatus) ([]models.RemediationJob, error) {
	query := "SELECT * FROM remediation_jobs WHERE status = $1 ORDER BY created_at DESC"
	var jobs []models.RemediationJob
	err := r.db.SelectContext(ctx, &jobs, query, status)
	return jobs, err
}

// GetJobsByPriority retrieves jobs by priority
func (r *PostgresRepository) GetJobsByPriority(ctx context.Context, priority models.JobPriority) ([]models.RemediationJob, error) {
	query := "SELECT * FROM remediation_jobs WHERE priority = $1 ORDER BY created_at DESC"
	var jobs []models.RemediationJob
	err := r.db.SelectContext(ctx, &jobs, query, priority)
	return jobs, err
}

// GetJobsByUser retrieves jobs by user
func (r *PostgresRepository) GetJobsByUser(ctx context.Context, userID string) ([]models.RemediationJob, error) {
	query := "SELECT * FROM remediation_jobs WHERE created_by = $1 ORDER BY created_at DESC"
	var jobs []models.RemediationJob
	err := r.db.SelectContext(ctx, &jobs, query, userID)
	return jobs, err
}

// GetJobsByStrategy retrieves jobs by strategy type
func (r *PostgresRepository) GetJobsByStrategy(ctx context.Context, strategyType models.StrategyType) ([]models.RemediationJob, error) {
	query := "SELECT * FROM remediation_jobs WHERE strategy->>'type' = $1 ORDER BY created_at DESC"
	var jobs []models.RemediationJob
	err := r.db.SelectContext(ctx, &jobs, query, strategyType)
	return jobs, err
}

// GetJobsByDateRange retrieves jobs within a date range
func (r *PostgresRepository) GetJobsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.RemediationJob, error) {
	query := "SELECT * FROM remediation_jobs WHERE created_at BETWEEN $1 AND $2 ORDER BY created_at DESC"
	var jobs []models.RemediationJob
	err := r.db.SelectContext(ctx, &jobs, query, startDate, endDate)
	return jobs, err
}

// CreateStrategy creates a new remediation strategy
func (r *PostgresRepository) CreateStrategy(ctx context.Context, strategy *models.RemediationStrategy) error {
	query := `
		INSERT INTO remediation_strategies (
			id, type, name, description, parameters, timeout, retry_count,
			is_custom, created_by, created_at
		) VALUES (
			:id, :type, :name, :description, :parameters, :timeout, :retry_count,
			:is_custom, :created_by, :created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, strategy)
	return err
}

// GetStrategyByID retrieves a remediation strategy by ID
func (r *PostgresRepository) GetStrategyByID(ctx context.Context, id string) (*models.RemediationStrategy, error) {
	query := `
		SELECT id, type, name, description, parameters, timeout, retry_count,
			   is_custom, created_by, created_at
		FROM remediation_strategies
		WHERE id = $1`

	var strategy models.RemediationStrategy
	err := r.db.GetContext(ctx, &strategy, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrRemediationStrategyNotFound
		}
		return nil, err
	}

	return &strategy, nil
}

// GetStrategyByName retrieves a remediation strategy by name
func (r *PostgresRepository) GetStrategyByName(ctx context.Context, name string) (*models.RemediationStrategy, error) {
	query := `
		SELECT id, type, name, description, parameters, timeout, retry_count,
			   is_custom, created_by, created_at
		FROM remediation_strategies
		WHERE name = $1`

	var strategy models.RemediationStrategy
	err := r.db.GetContext(ctx, &strategy, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrRemediationStrategyNotFound
		}
		return nil, err
	}

	return &strategy, nil
}

// UpdateStrategy updates an existing remediation strategy
func (r *PostgresRepository) UpdateStrategy(ctx context.Context, strategy *models.RemediationStrategy) error {
	query := `
		UPDATE remediation_strategies SET
			type = :type,
			name = :name,
			description = :description,
			parameters = :parameters,
			timeout = :timeout,
			retry_count = :retry_count,
			is_custom = :is_custom,
			created_by = :created_by
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, strategy)
	return err
}

// DeleteStrategy deletes a remediation strategy
func (r *PostgresRepository) DeleteStrategy(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM remediation_strategies WHERE id = $1", id)
	return err
}

// ListStrategies lists all remediation strategies
func (r *PostgresRepository) ListStrategies(ctx context.Context) ([]models.RemediationStrategy, error) {
	query := "SELECT * FROM remediation_strategies ORDER BY name ASC"
	var strategies []models.RemediationStrategy
	err := r.db.SelectContext(ctx, &strategies, query)
	return strategies, err
}

// GetStrategiesByType retrieves strategies by type
func (r *PostgresRepository) GetStrategiesByType(ctx context.Context, strategyType models.StrategyType) ([]models.RemediationStrategy, error) {
	query := "SELECT * FROM remediation_strategies WHERE type = $1 ORDER BY name ASC"
	var strategies []models.RemediationStrategy
	err := r.db.SelectContext(ctx, &strategies, query, strategyType)
	return strategies, err
}

// GetRemediationHistory retrieves remediation history
func (r *PostgresRepository) GetRemediationHistory(ctx context.Context, req *models.RemediationHistoryRequest) (*models.RemediationHistoryResponse, error) {
	// Build query with filters
	query := "SELECT * FROM remediation_jobs WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if req.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *req.Status)
		argIndex++
	}

	if req.Strategy != nil {
		query += fmt.Sprintf(" AND strategy->>'type' = $%d", argIndex)
		args = append(args, *req.Strategy)
		argIndex++
	}

	if req.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *req.StartDate)
		argIndex++
	}

	if req.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *req.EndDate)
		argIndex++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.Limit, req.Offset)

	var jobs []models.RemediationJob
	err := r.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM remediation_jobs WHERE 1=1"
	countArgs := args[:len(args)-2] // Remove limit and offset

	var total int
	err = r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, err
	}

	return &models.RemediationHistoryResponse{
		Jobs:   jobs,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

// GetJobStatistics retrieves job statistics
func (r *PostgresRepository) GetJobStatistics(ctx context.Context, startDate, endDate time.Time) (*models.JobStatistics, error) {
	// This would contain complex queries to calculate statistics
	// For now, returning a basic implementation
	stats := &models.JobStatistics{
		JobsByStatus:   make(map[string]int),
		JobsByPriority: make(map[string]int),
		JobsByStrategy: make(map[string]int),
	}

	// Get total jobs
	err := r.db.GetContext(ctx, &stats.TotalJobs,
		"SELECT COUNT(*) FROM remediation_jobs WHERE created_at BETWEEN $1 AND $2",
		startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get jobs by status
	rows, err := r.db.QueryContext(ctx,
		"SELECT status, COUNT(*) FROM remediation_jobs WHERE created_at BETWEEN $1 AND $2 GROUP BY status",
		startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.JobsByStatus[status] = count
	}

	return stats, nil
}

// GetSuccessRate calculates the success rate
func (r *PostgresRepository) GetSuccessRate(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	var successRate float64
	err := r.db.GetContext(ctx, &successRate, `
		SELECT 
			CASE 
				WHEN COUNT(*) = 0 THEN 0
				ELSE (COUNT(*) FILTER (WHERE status = 'completed'))::float / COUNT(*)::float * 100
			END
		FROM remediation_jobs 
		WHERE created_at BETWEEN $1 AND $2`, startDate, endDate)
	return successRate, err
}

// GetAverageJobDuration calculates the average job duration
func (r *PostgresRepository) GetAverageJobDuration(ctx context.Context, startDate, endDate time.Time) (time.Duration, error) {
	var avgDuration time.Duration
	err := r.db.GetContext(ctx, &avgDuration, `
		SELECT AVG(completed_at - started_at)
		FROM remediation_jobs 
		WHERE created_at BETWEEN $1 AND $2 
		AND completed_at IS NOT NULL 
		AND started_at IS NOT NULL`, startDate, endDate)
	return avgDuration, err
}

// DeleteOldJobs deletes old jobs
func (r *PostgresRepository) DeleteOldJobs(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM remediation_jobs WHERE created_at < $1", olderThan)
	return err
}

// DeleteOldLogs deletes old logs
func (r *PostgresRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM remediation_job_logs WHERE timestamp < $1", olderThan)
	return err
}

// CleanupCompletedJobs cleans up old completed jobs
func (r *PostgresRepository) CleanupCompletedJobs(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM remediation_jobs WHERE status IN ('completed', 'failed', 'cancelled') AND completed_at < $1",
		olderThan)
	return err
}

// Health checks the database connection
func (r *PostgresRepository) Health(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// GetQueueDepth returns the number of pending jobs
func (r *PostgresRepository) GetQueueDepth(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM remediation_jobs WHERE status IN ('pending', 'queued')")
	return count, err
}

// GetActiveJobsCount returns the number of active jobs
func (r *PostgresRepository) GetActiveJobsCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM remediation_jobs WHERE status = 'running'")
	return count, err
}

// GetFailedJobsCount returns the number of failed jobs
func (r *PostgresRepository) GetFailedJobsCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM remediation_jobs WHERE status = 'failed'")
	return count, err
}

// WithTransaction executes a function within a database transaction
func (r *PostgresRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create a new context with the transaction
	txCtx := context.WithValue(ctx, "tx", tx)

	if err := fn(txCtx); err != nil {
		return err
	}

	return tx.Commit()
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}
