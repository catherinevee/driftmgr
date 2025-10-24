package drift

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
)

// Service defines the business logic interface for drift management
type Service interface {
	// CreateDriftResult creates a new drift detection result
	CreateDriftResult(ctx context.Context, req *models.DriftResultRequest) (*models.DriftResult, error)

	// GetDriftResult retrieves a drift result by ID
	GetDriftResult(ctx context.Context, id string) (*models.DriftResult, error)

	// ListDriftResults retrieves drift results with filtering and pagination
	ListDriftResults(ctx context.Context, query *models.DriftResultQuery) (*models.PaginatedDriftResults, error)

	// GetDriftHistory retrieves drift detection history
	GetDriftHistory(ctx context.Context, req *models.DriftHistoryRequest) (*models.DriftHistoryResponse, error)

	// GetDriftSummary retrieves drift summary statistics
	GetDriftSummary(ctx context.Context, provider string) (*models.DriftSummaryResponse, error)

	// UpdateDriftResult updates an existing drift result
	UpdateDriftResult(ctx context.Context, result *models.DriftResult) error

	// DeleteDriftResult deletes a drift result by ID
	DeleteDriftResult(ctx context.Context, id string) error

	// DeleteDriftResultsByProvider deletes drift results by provider
	DeleteDriftResultsByProvider(ctx context.Context, provider string) error

	// DeleteDriftResultsByDateRange deletes drift results within a date range
	DeleteDriftResultsByDateRange(ctx context.Context, startDate, endDate time.Time) error

	// GetDriftTrend retrieves drift trend data
	GetDriftTrend(ctx context.Context, provider string, days int) ([]*models.DriftResult, error)

	// GetTopDriftedResources retrieves the most frequently drifted resources
	GetTopDriftedResources(ctx context.Context, limit int) ([]string, error)

	// GetDriftBySeverity retrieves drift results grouped by severity
	GetDriftBySeverity(ctx context.Context, provider string) (map[string]int, error)

	// Health check
	Health(ctx context.Context) error
}

// DriftService implements the Service interface
type DriftService struct {
	repo   Repository
	config *ServiceConfig
}

// Repository defines the repository interface for drift data access
type Repository interface {
	Create(ctx context.Context, result *models.DriftResult) error
	GetByID(ctx context.Context, id string) (*models.DriftResult, error)
	List(ctx context.Context, query *models.DriftResultQuery) (*models.PaginatedDriftResults, error)
	GetHistory(ctx context.Context, req *models.DriftHistoryRequest) (*models.DriftHistoryResponse, error)
	GetSummary(ctx context.Context, provider string) (*models.DriftSummaryResponse, error)
	Update(ctx context.Context, result *models.DriftResult) error
	Delete(ctx context.Context, id string) error
	DeleteByProvider(ctx context.Context, provider string) error
	DeleteByDateRange(ctx context.Context, startDate, endDate time.Time) error
	GetDriftTrend(ctx context.Context, provider string, days int) ([]*models.DriftResult, error)
	GetTopDriftedResources(ctx context.Context, limit int) ([]string, error)
	GetDriftBySeverity(ctx context.Context, provider string) (map[string]int, error)
	Health(ctx context.Context) error
}

// ServiceConfig contains configuration for the drift service
type ServiceConfig struct {
	MaxDriftResults   int
	MaxHistoryDays    int
	DefaultLimit      int
	MaxLimit          int
	RetentionDays     int
	EnableAutoCleanup bool
	CleanupInterval   time.Duration
}

// NewDriftService creates a new drift service
func NewDriftService(repo Repository, config *ServiceConfig) *DriftService {
	if config == nil {
		config = &ServiceConfig{
			MaxDriftResults:   10000,
			MaxHistoryDays:    90,
			DefaultLimit:      50,
			MaxLimit:          1000,
			RetentionDays:     30,
			EnableAutoCleanup: true,
			CleanupInterval:   time.Hour * 24,
		}
	}

	return &DriftService{
		repo:   repo,
		config: config,
	}
}

// CreateDriftResult creates a new drift detection result
func (s *DriftService) CreateDriftResult(ctx context.Context, req *models.DriftResultRequest) (*models.DriftResult, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Generate ID
	id := uuid.New().String()

	// Create drift result
	result := &models.DriftResult{
		ID:        id,
		Timestamp: time.Now(),
		Provider:  req.Provider,
		Status:    "running",
		Resources: []models.DriftedResource{},
		Summary:   models.DriftSummary{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to repository
	if err := s.repo.Create(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to create drift result: %w", err)
	}

	return result, nil
}

// GetDriftResult retrieves a drift result by ID
func (s *DriftService) GetDriftResult(ctx context.Context, id string) (*models.DriftResult, error) {
	if id == "" {
		return nil, models.ErrDriftResultNotFound
	}

	result, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift result: %w", err)
	}

	return result, nil
}

// ListDriftResults retrieves drift results with filtering and pagination
func (s *DriftService) ListDriftResults(ctx context.Context, query *models.DriftResultQuery) (*models.PaginatedDriftResults, error) {
	// Validate query
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// Set default values
	if query.Limit <= 0 {
		query.Limit = s.config.DefaultLimit
	}
	if query.Limit > s.config.MaxLimit {
		query.Limit = s.config.MaxLimit
	}

	// Get results from repository
	results, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list drift results: %w", err)
	}

	return results, nil
}

// GetDriftHistory retrieves drift detection history
func (s *DriftService) GetDriftHistory(ctx context.Context, req *models.DriftHistoryRequest) (*models.DriftHistoryResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set default values
	if req.Limit <= 0 {
		req.Limit = s.config.DefaultLimit
	}
	if req.Limit > s.config.MaxLimit {
		req.Limit = s.config.MaxLimit
	}

	// Set default date range if not provided
	if req.StartDate.IsZero() {
		req.StartDate = time.Now().AddDate(0, 0, -s.config.MaxHistoryDays)
	}

	// Get history from repository
	history, err := s.repo.GetHistory(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift history: %w", err)
	}

	return history, nil
}

// GetDriftSummary retrieves drift summary statistics
func (s *DriftService) GetDriftSummary(ctx context.Context, provider string) (*models.DriftSummaryResponse, error) {
	summary, err := s.repo.GetSummary(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift summary: %w", err)
	}

	return summary, nil
}

// UpdateDriftResult updates an existing drift result
func (s *DriftService) UpdateDriftResult(ctx context.Context, result *models.DriftResult) error {
	if result == nil {
		return models.ErrDriftResultNotFound
	}

	if result.ID == "" {
		return models.ErrDriftResultNotFound
	}

	// Update timestamp
	result.UpdatedAt = time.Now()

	// Recalculate summary if resources changed
	if len(result.Resources) > 0 {
		result.CalculateSummary()
	}

	// Update in repository
	if err := s.repo.Update(ctx, result); err != nil {
		return fmt.Errorf("failed to update drift result: %w", err)
	}

	return nil
}

// DeleteDriftResult deletes a drift result by ID
func (s *DriftService) DeleteDriftResult(ctx context.Context, id string) error {
	if id == "" {
		return models.ErrDriftResultNotFound
	}

	// Check if drift result exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get drift result: %w", err)
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete drift result: %w", err)
	}

	return nil
}

// DeleteDriftResultsByProvider deletes drift results by provider
func (s *DriftService) DeleteDriftResultsByProvider(ctx context.Context, provider string) error {
	if provider == "" {
		return models.ErrInvalidProvider
	}

	// Validate provider
	validProviders := map[string]bool{
		"aws":          true,
		"azure":        true,
		"gcp":          true,
		"digitalocean": true,
	}

	if !validProviders[provider] {
		return models.ErrInvalidProvider
	}

	// Delete from repository
	if err := s.repo.DeleteByProvider(ctx, provider); err != nil {
		return fmt.Errorf("failed to delete drift results by provider: %w", err)
	}

	return nil
}

// DeleteDriftResultsByDateRange deletes drift results within a date range
func (s *DriftService) DeleteDriftResultsByDateRange(ctx context.Context, startDate, endDate time.Time) error {
	if startDate.IsZero() || endDate.IsZero() {
		return models.ErrInvalidDateRange
	}

	if startDate.After(endDate) {
		return models.ErrInvalidDateRange
	}

	// Check if date range is too large (more than retention period)
	maxRange := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	if startDate.Before(maxRange) {
		return fmt.Errorf("date range exceeds retention period of %d days", s.config.RetentionDays)
	}

	// Delete from repository
	if err := s.repo.DeleteByDateRange(ctx, startDate, endDate); err != nil {
		return fmt.Errorf("failed to delete drift results by date range: %w", err)
	}

	return nil
}

// GetDriftTrend retrieves drift trend data
func (s *DriftService) GetDriftTrend(ctx context.Context, provider string, days int) ([]*models.DriftResult, error) {
	// Validate parameters
	if days <= 0 {
		days = 30 // Default to 30 days
	}
	if days > s.config.MaxHistoryDays {
		days = s.config.MaxHistoryDays
	}

	// Get trend from repository
	trend, err := s.repo.GetDriftTrend(ctx, provider, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift trend: %w", err)
	}

	return trend, nil
}

// GetTopDriftedResources retrieves the most frequently drifted resources
func (s *DriftService) GetTopDriftedResources(ctx context.Context, limit int) ([]string, error) {
	// Validate limit
	if limit <= 0 {
		limit = 10 // Default to top 10
	}
	if limit > 100 {
		limit = 100 // Maximum limit
	}

	// Get top drifted resources from repository
	resources, err := s.repo.GetTopDriftedResources(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top drifted resources: %w", err)
	}

	return resources, nil
}

// GetDriftBySeverity retrieves drift results grouped by severity
func (s *DriftService) GetDriftBySeverity(ctx context.Context, provider string) (map[string]int, error) {
	// Validate provider if provided
	if provider != "" {
		validProviders := map[string]bool{
			"aws":          true,
			"azure":        true,
			"gcp":          true,
			"digitalocean": true,
		}

		if !validProviders[provider] {
			return nil, models.ErrInvalidProvider
		}
	}

	// Get drift by severity from repository
	severity, err := s.repo.GetDriftBySeverity(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift by severity: %w", err)
	}

	return severity, nil
}

// Health check
func (s *DriftService) Health(ctx context.Context) error {
	return s.repo.Health(ctx)
}

// Helper methods

// CompleteDriftResult marks a drift result as completed with resources and summary
func (s *DriftService) CompleteDriftResult(ctx context.Context, id string, resources []models.DriftedResource, duration time.Duration) error {
	// Get existing drift result
	result, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get drift result: %w", err)
	}

	// Update result
	result.Status = "completed"
	result.Resources = resources
	result.Duration = duration
	result.UpdatedAt = time.Now()

	// Calculate summary
	result.CalculateSummary()

	// Update in repository
	if err := s.repo.Update(ctx, result); err != nil {
		return fmt.Errorf("failed to update drift result: %w", err)
	}

	return nil
}

// FailDriftResult marks a drift result as failed with error message
func (s *DriftService) FailDriftResult(ctx context.Context, id string, errorMsg string) error {
	// Get existing drift result
	result, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get drift result: %w", err)
	}

	// Update result
	result.Status = "failed"
	result.Error = &errorMsg
	result.UpdatedAt = time.Now()

	// Update in repository
	if err := s.repo.Update(ctx, result); err != nil {
		return fmt.Errorf("failed to update drift result: %w", err)
	}

	return nil
}

// GetRunningDriftResults retrieves all running drift results
func (s *DriftService) GetRunningDriftResults(ctx context.Context) ([]*models.DriftResult, error) {
	query := &models.DriftResultQuery{
		Filter: models.DriftResultFilter{
			Status: "running",
		},
		Sort: models.DriftResultSort{
			Field: "timestamp",
			Order: "desc",
		},
		Limit:  100,
		Offset: 0,
	}

	results, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get running drift results: %w", err)
	}

	var driftResults []*models.DriftResult
	for i := range results.Results {
		driftResults = append(driftResults, &results.Results[i])
	}

	return driftResults, nil
}

// GetLatestDriftResult retrieves the latest drift result for a provider
func (s *DriftService) GetLatestDriftResult(ctx context.Context, provider string) (*models.DriftResult, error) {
	query := &models.DriftResultQuery{
		Filter: models.DriftResultFilter{
			Provider: provider,
		},
		Sort: models.DriftResultSort{
			Field: "timestamp",
			Order: "desc",
		},
		Limit:  1,
		Offset: 0,
	}

	results, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest drift result: %w", err)
	}

	if len(results.Results) == 0 {
		return nil, models.ErrDriftResultNotFound
	}

	return &results.Results[0], nil
}

// CleanupOldDriftResults removes old drift results based on retention policy
func (s *DriftService) CleanupOldDriftResults(ctx context.Context) error {
	if !s.config.EnableAutoCleanup {
		return nil
	}

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	// Delete old drift results
	if err := s.repo.DeleteByDateRange(ctx, time.Time{}, cutoffDate); err != nil {
		return fmt.Errorf("failed to cleanup old drift results: %w", err)
	}

	return nil
}
