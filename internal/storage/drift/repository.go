package drift

import (
	"context"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Repository defines the interface for drift result data access
type Repository interface {
	// Create creates a new drift result
	Create(ctx context.Context, result *models.DriftResult) error

	// GetByID retrieves a drift result by ID
	GetByID(ctx context.Context, id string) (*models.DriftResult, error)

	// List retrieves drift results with filtering and pagination
	List(ctx context.Context, query *models.DriftResultQuery) (*models.PaginatedDriftResults, error)

	// GetHistory retrieves drift detection history
	GetHistory(ctx context.Context, req *models.DriftHistoryRequest) (*models.DriftHistoryResponse, error)

	// GetSummary retrieves drift summary statistics
	GetSummary(ctx context.Context, provider string) (*models.DriftSummaryResponse, error)

	// Update updates an existing drift result
	Update(ctx context.Context, result *models.DriftResult) error

	// Delete deletes a drift result by ID
	Delete(ctx context.Context, id string) error

	// DeleteByProvider deletes drift results by provider
	DeleteByProvider(ctx context.Context, provider string) error

	// DeleteByDateRange deletes drift results within a date range
	DeleteByDateRange(ctx context.Context, startDate, endDate time.Time) error

	// Count returns the total count of drift results
	Count(ctx context.Context, filter *models.DriftResultFilter) (int, error)

	// GetByStatus retrieves drift results by status
	GetByStatus(ctx context.Context, status string) ([]*models.DriftResult, error)

	// GetByProvider retrieves drift results by provider
	GetByProvider(ctx context.Context, provider string, limit int) ([]*models.DriftResult, error)

	// GetLatestByProvider retrieves the latest drift result for a provider
	GetLatestByProvider(ctx context.Context, provider string) (*models.DriftResult, error)

	// GetDriftTrend retrieves drift trend data
	GetDriftTrend(ctx context.Context, provider string, days int) ([]*models.DriftResult, error)

	// GetTopDriftedResources retrieves the most frequently drifted resources
	GetTopDriftedResources(ctx context.Context, limit int) ([]string, error)

	// GetDriftBySeverity retrieves drift results grouped by severity
	GetDriftBySeverity(ctx context.Context, provider string) (map[string]int, error)

	// Health check
	Health(ctx context.Context) error
}

// RepositoryConfig contains configuration for the repository
type RepositoryConfig struct {
	DatabaseURL string
	MaxConns    int
	MinConns    int
	MaxLifetime time.Duration
	MaxIdleTime time.Duration
}

// RepositoryMetrics defines metrics for the repository
type RepositoryMetrics struct {
	QueryDuration     time.Duration
	QueryCount        int64
	ErrorCount        int64
	ConnectionCount   int64
	ActiveConnections int64
}

// RepositoryStats provides statistics about the repository
type RepositoryStats struct {
	TotalDriftResults     int64
	TotalDriftedResources int64
	AverageDriftCount     float64
	MostActiveProvider    string
	LastDriftDetection    time.Time
	Metrics               RepositoryMetrics
}
