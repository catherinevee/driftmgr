package drift

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// NewPostgresRepository creates a new PostgreSQL repository
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

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresRepository{
		db:     db,
		config: config,
	}, nil
}

// Create creates a new drift result
func (r *PostgresRepository) Create(ctx context.Context, result *models.DriftResult) error {
	query := `
		INSERT INTO drift.drift_results (
			id, timestamp, provider, status, drift_count, 
			resources, summary, duration, error, created_at, updated_at
		) VALUES (
			:id, :timestamp, :provider, :status, :drift_count,
			:resources, :summary, :duration, :error, :created_at, :updated_at
		)`

	// Serialize JSON fields
	resourcesJSON, err := json.Marshal(result.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources: %w", err)
	}

	summaryJSON, err := json.Marshal(result.Summary)
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	args := map[string]interface{}{
		"id":          result.ID,
		"timestamp":   result.Timestamp,
		"provider":    result.Provider,
		"status":      result.Status,
		"drift_count": result.DriftCount,
		"resources":   string(resourcesJSON),
		"summary":     string(summaryJSON),
		"duration":    result.Duration,
		"error":       result.Error,
		"created_at":  result.CreatedAt,
		"updated_at":  result.UpdatedAt,
	}

	_, err = r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("failed to create drift result: %w", err)
	}

	return nil
}

// GetByID retrieves a drift result by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*models.DriftResult, error) {
	query := `
		SELECT id, timestamp, provider, status, drift_count,
		       resources, summary, duration, error, created_at, updated_at
		FROM drift.drift_results
		WHERE id = $1`

	var result models.DriftResult
	var resourcesJSON, summaryJSON string

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&result.ID,
		&result.Timestamp,
		&result.Provider,
		&result.Status,
		&result.DriftCount,
		&resourcesJSON,
		&summaryJSON,
		&result.Duration,
		&result.Error,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrDriftResultNotFound
		}
		return nil, fmt.Errorf("failed to get drift result: %w", err)
	}

	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(resourcesJSON), &result.Resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
	}

	return &result, nil
}

// List retrieves drift results with filtering and pagination
func (r *PostgresRepository) List(ctx context.Context, query *models.DriftResultQuery) (*models.PaginatedDriftResults, error) {
	// Build WHERE clause
	whereClause, args := r.buildWhereClause(&query.Filter)

	// Build ORDER BY clause
	orderClause := r.buildOrderClause(&query.Sort)

	// Build LIMIT and OFFSET
	limitClause := "LIMIT $1 OFFSET $2"
	args = append(args, query.Limit, query.Offset)

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM drift.drift_results %s", whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:len(args)-2]...); err != nil {
		return nil, fmt.Errorf("failed to count drift results: %w", err)
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, timestamp, provider, status, drift_count,
		       resources, summary, duration, error, created_at, updated_at
		FROM drift.drift_results
		%s %s %s`, whereClause, orderClause, limitClause)

	rows, err := r.db.QueryxContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query drift results: %w", err)
	}
	defer rows.Close()

	var results []models.DriftResult
	for rows.Next() {
		var result models.DriftResult
		var resourcesJSON, summaryJSON string

		err := rows.Scan(
			&result.ID,
			&result.Timestamp,
			&result.Provider,
			&result.Status,
			&result.DriftCount,
			&resourcesJSON,
			&summaryJSON,
			&result.Duration,
			&result.Error,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan drift result: %w", err)
		}

		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(resourcesJSON), &result.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate drift results: %w", err)
	}

	// Calculate pagination
	pages := (total + query.Limit - 1) / query.Limit
	if pages == 0 {
		pages = 1
	}

	return &models.PaginatedDriftResults{
		Results: results,
		Total:   total,
		Page:    (query.Offset / query.Limit) + 1,
		PerPage: query.Limit,
		Pages:   pages,
	}, nil
}

// GetHistory retrieves drift detection history
func (r *PostgresRepository) GetHistory(ctx context.Context, req *models.DriftHistoryRequest) (*models.DriftHistoryResponse, error) {
	whereClause, args := r.buildHistoryWhereClause(req)

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM drift.drift_results %s", whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, fmt.Errorf("failed to count drift history: %w", err)
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, timestamp, provider, status, drift_count,
		       resources, summary, duration, error, created_at, updated_at
		FROM drift.drift_results
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)

	args = append(args, req.Limit, req.Offset)

	rows, err := r.db.QueryxContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query drift history: %w", err)
	}
	defer rows.Close()

	var results []models.DriftResult
	for rows.Next() {
		var result models.DriftResult
		var resourcesJSON, summaryJSON string

		err := rows.Scan(
			&result.ID,
			&result.Timestamp,
			&result.Provider,
			&result.Status,
			&result.DriftCount,
			&resourcesJSON,
			&summaryJSON,
			&result.Duration,
			&result.Error,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan drift result: %w", err)
		}

		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(resourcesJSON), &result.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate drift history: %w", err)
	}

	return &models.DriftHistoryResponse{
		Results: results,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// GetSummary retrieves drift summary statistics
func (r *PostgresRepository) GetSummary(ctx context.Context, provider string) (*models.DriftSummaryResponse, error) {
	whereClause := ""
	args := []interface{}{}

	if provider != "" {
		whereClause = "WHERE provider = $1"
		args = append(args, provider)
	}

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_detections,
			COUNT(*) FILTER (WHERE status = 'completed') as successful_detections,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_detections,
			AVG(drift_count) as avg_drift_count,
			MAX(timestamp) as last_detection,
			SUM((summary->>'drifted_resources')::int) as total_drifted_resources,
			SUM((summary->>'critical_drift')::int) as total_critical_drift,
			SUM((summary->>'high_drift')::int) as total_high_drift,
			SUM((summary->>'medium_drift')::int) as total_medium_drift,
			SUM((summary->>'low_drift')::int) as total_low_drift
		FROM drift.drift_results
		%s`, whereClause)

	var result struct {
		TotalDetections       int       `db:"total_detections"`
		SuccessfulDetections  int       `db:"successful_detections"`
		FailedDetections      int       `db:"failed_detections"`
		AvgDriftCount         float64   `db:"avg_drift_count"`
		LastDetection         time.Time `db:"last_detection"`
		TotalDriftedResources int       `db:"total_drifted_resources"`
		TotalCriticalDrift    int       `db:"total_critical_drift"`
		TotalHighDrift        int       `db:"total_high_drift"`
		TotalMediumDrift      int       `db:"total_medium_drift"`
		TotalLowDrift         int       `db:"total_low_drift"`
	}

	err := r.db.GetContext(ctx, &result, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift summary: %w", err)
	}

	summary := models.DriftSummary{
		TotalResources:   result.TotalDetections,
		DriftedResources: result.TotalDriftedResources,
		CriticalDrift:    result.TotalCriticalDrift,
		HighDrift:        result.TotalHighDrift,
		MediumDrift:      result.TotalMediumDrift,
		LowDrift:         result.TotalLowDrift,
	}

	return &models.DriftSummaryResponse{
		Summary:     summary,
		LastUpdated: result.LastDetection,
		Provider:    provider,
	}, nil
}

// Update updates an existing drift result
func (r *PostgresRepository) Update(ctx context.Context, result *models.DriftResult) error {
	query := `
		UPDATE drift.drift_results SET
			timestamp = :timestamp,
			provider = :provider,
			status = :status,
			drift_count = :drift_count,
			resources = :resources,
			summary = :summary,
			duration = :duration,
			error = :error,
			updated_at = :updated_at
		WHERE id = :id`

	// Serialize JSON fields
	resourcesJSON, err := json.Marshal(result.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources: %w", err)
	}

	summaryJSON, err := json.Marshal(result.Summary)
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	args := map[string]interface{}{
		"id":          result.ID,
		"timestamp":   result.Timestamp,
		"provider":    result.Provider,
		"status":      result.Status,
		"drift_count": result.DriftCount,
		"resources":   string(resourcesJSON),
		"summary":     string(summaryJSON),
		"duration":    result.Duration,
		"error":       result.Error,
		"updated_at":  result.UpdatedAt,
	}

	result_, err := r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("failed to update drift result: %w", err)
	}

	rowsAffected, err := result_.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return models.ErrDriftResultNotFound
	}

	return nil
}

// Delete deletes a drift result by ID
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM drift.drift_results WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete drift result: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return models.ErrDriftResultNotFound
	}

	return nil
}

// DeleteByProvider deletes drift results by provider
func (r *PostgresRepository) DeleteByProvider(ctx context.Context, provider string) error {
	query := "DELETE FROM drift.drift_results WHERE provider = $1"

	_, err := r.db.ExecContext(ctx, query, provider)
	if err != nil {
		return fmt.Errorf("failed to delete drift results by provider: %w", err)
	}

	return nil
}

// DeleteByDateRange deletes drift results within a date range
func (r *PostgresRepository) DeleteByDateRange(ctx context.Context, startDate, endDate time.Time) error {
	query := "DELETE FROM drift.drift_results WHERE timestamp BETWEEN $1 AND $2"

	_, err := r.db.ExecContext(ctx, query, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to delete drift results by date range: %w", err)
	}

	return nil
}

// Count returns the total count of drift results
func (r *PostgresRepository) Count(ctx context.Context, filter *models.DriftResultFilter) (int, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf("SELECT COUNT(*) FROM drift.drift_results %s", whereClause)

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count drift results: %w", err)
	}

	return count, nil
}

// GetByStatus retrieves drift results by status
func (r *PostgresRepository) GetByStatus(ctx context.Context, status string) ([]*models.DriftResult, error) {
	query := `
		SELECT id, timestamp, provider, status, drift_count,
		       resources, summary, duration, error, created_at, updated_at
		FROM drift.drift_results
		WHERE status = $1
		ORDER BY timestamp DESC`

	rows, err := r.db.QueryxContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query drift results by status: %w", err)
	}
	defer rows.Close()

	var results []*models.DriftResult
	for rows.Next() {
		var result models.DriftResult
		var resourcesJSON, summaryJSON string

		err := rows.Scan(
			&result.ID,
			&result.Timestamp,
			&result.Provider,
			&result.Status,
			&result.DriftCount,
			&resourcesJSON,
			&summaryJSON,
			&result.Duration,
			&result.Error,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan drift result: %w", err)
		}

		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(resourcesJSON), &result.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
		}

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate drift results: %w", err)
	}

	return results, nil
}

// GetByProvider retrieves drift results by provider
func (r *PostgresRepository) GetByProvider(ctx context.Context, provider string, limit int) ([]*models.DriftResult, error) {
	query := `
		SELECT id, timestamp, provider, status, drift_count,
		       resources, summary, duration, error, created_at, updated_at
		FROM drift.drift_results
		WHERE provider = $1
		ORDER BY timestamp DESC
		LIMIT $2`

	rows, err := r.db.QueryxContext(ctx, query, provider, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query drift results by provider: %w", err)
	}
	defer rows.Close()

	var results []*models.DriftResult
	for rows.Next() {
		var result models.DriftResult
		var resourcesJSON, summaryJSON string

		err := rows.Scan(
			&result.ID,
			&result.Timestamp,
			&result.Provider,
			&result.Status,
			&result.DriftCount,
			&resourcesJSON,
			&summaryJSON,
			&result.Duration,
			&result.Error,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan drift result: %w", err)
		}

		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(resourcesJSON), &result.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
		}

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate drift results: %w", err)
	}

	return results, nil
}

// GetLatestByProvider retrieves the latest drift result for a provider
func (r *PostgresRepository) GetLatestByProvider(ctx context.Context, provider string) (*models.DriftResult, error) {
	query := `
		SELECT id, timestamp, provider, status, drift_count,
		       resources, summary, duration, error, created_at, updated_at
		FROM drift.drift_results
		WHERE provider = $1
		ORDER BY timestamp DESC
		LIMIT 1`

	var result models.DriftResult
	var resourcesJSON, summaryJSON string

	err := r.db.QueryRowxContext(ctx, query, provider).Scan(
		&result.ID,
		&result.Timestamp,
		&result.Provider,
		&result.Status,
		&result.DriftCount,
		&resourcesJSON,
		&summaryJSON,
		&result.Duration,
		&result.Error,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrDriftResultNotFound
		}
		return nil, fmt.Errorf("failed to get latest drift result: %w", err)
	}

	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(resourcesJSON), &result.Resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
	}

	return &result, nil
}

// GetDriftTrend retrieves drift trend data
func (r *PostgresRepository) GetDriftTrend(ctx context.Context, provider string, days int) ([]*models.DriftResult, error) {
	query := `
		SELECT id, timestamp, provider, status, drift_count,
		       resources, summary, duration, error, created_at, updated_at
		FROM drift.drift_results
		WHERE provider = $1 AND timestamp >= NOW() - INTERVAL '%d days'
		ORDER BY timestamp ASC`

	rows, err := r.db.QueryxContext(ctx, fmt.Sprintf(query, days), provider)
	if err != nil {
		return nil, fmt.Errorf("failed to query drift trend: %w", err)
	}
	defer rows.Close()

	var results []*models.DriftResult
	for rows.Next() {
		var result models.DriftResult
		var resourcesJSON, summaryJSON string

		err := rows.Scan(
			&result.ID,
			&result.Timestamp,
			&result.Provider,
			&result.Status,
			&result.DriftCount,
			&resourcesJSON,
			&summaryJSON,
			&result.Duration,
			&result.Error,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan drift result: %w", err)
		}

		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(resourcesJSON), &result.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		if err := json.Unmarshal([]byte(summaryJSON), &result.Summary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
		}

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate drift trend: %w", err)
	}

	return results, nil
}

// GetTopDriftedResources retrieves the most frequently drifted resources
func (r *PostgresRepository) GetTopDriftedResources(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT resource_address, COUNT(*) as drift_count
		FROM (
			SELECT jsonb_array_elements(resources)->>'address' as resource_address
			FROM drift.drift_results
			WHERE status = 'completed'
		) as drifted_resources
		WHERE resource_address IS NOT NULL
		GROUP BY resource_address
		ORDER BY drift_count DESC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top drifted resources: %w", err)
	}
	defer rows.Close()

	var resources []string
	for rows.Next() {
		var resource string
		var count int
		if err := rows.Scan(&resource, &count); err != nil {
			return nil, fmt.Errorf("failed to scan top drifted resource: %w", err)
		}
		resources = append(resources, resource)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate top drifted resources: %w", err)
	}

	return resources, nil
}

// GetDriftBySeverity retrieves drift results grouped by severity
func (r *PostgresRepository) GetDriftBySeverity(ctx context.Context, provider string) (map[string]int, error) {
	whereClause := ""
	args := []interface{}{}

	if provider != "" {
		whereClause = "WHERE provider = $1"
		args = append(args, provider)
	}

	query := fmt.Sprintf(`
		SELECT 
			SUM((summary->>'critical_drift')::int) as critical,
			SUM((summary->>'high_drift')::int) as high,
			SUM((summary->>'medium_drift')::int) as medium,
			SUM((summary->>'low_drift')::int) as low
		FROM drift.drift_results
		%s`, whereClause)

	var result struct {
		Critical int `db:"critical"`
		High     int `db:"high"`
		Medium   int `db:"medium"`
		Low      int `db:"low"`
	}

	err := r.db.GetContext(ctx, &result, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift by severity: %w", err)
	}

	return map[string]int{
		"critical": result.Critical,
		"high":     result.High,
		"medium":   result.Medium,
		"low":      result.Low,
	}, nil
}

// Health check
func (r *PostgresRepository) Health(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// Helper methods

// buildWhereClause builds a WHERE clause from filter
func (r *PostgresRepository) buildWhereClause(filter *models.DriftResultFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Provider != "" {
		conditions = append(conditions, fmt.Sprintf("provider = $%d", argIndex))
		args = append(args, filter.Provider)
		argIndex++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if !filter.StartDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, filter.StartDate)
		argIndex++
	}

	if !filter.EndDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, filter.EndDate)
		argIndex++
	}

	if filter.MinDrift > 0 {
		conditions = append(conditions, fmt.Sprintf("drift_count >= $%d", argIndex))
		args = append(args, filter.MinDrift)
		argIndex++
	}

	if filter.MaxDrift > 0 {
		conditions = append(conditions, fmt.Sprintf("drift_count <= $%d", argIndex))
		args = append(args, filter.MaxDrift)
		argIndex++
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// buildHistoryWhereClause builds a WHERE clause for history requests
func (r *PostgresRepository) buildHistoryWhereClause(req *models.DriftHistoryRequest) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if req.Provider != "" {
		conditions = append(conditions, fmt.Sprintf("provider = $%d", argIndex))
		args = append(args, req.Provider)
		argIndex++
	}

	if req.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, req.Status)
		argIndex++
	}

	if !req.StartDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, req.StartDate)
		argIndex++
	}

	if !req.EndDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, req.EndDate)
		argIndex++
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// buildOrderClause builds an ORDER BY clause from sort
func (r *PostgresRepository) buildOrderClause(sort *models.DriftResultSort) string {
	if sort.Field == "" {
		return "ORDER BY timestamp DESC"
	}

	order := "ASC"
	if sort.Order == "desc" {
		order = "DESC"
	}

	return fmt.Sprintf("ORDER BY %s %s", sort.Field, order)
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}
