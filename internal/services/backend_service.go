package services

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/providers/aws"
)

// BackendService handles backend discovery and management business logic
type BackendService struct {
	repository BackendRepository
	awsClient  *aws.Client
}

// BackendRepository defines the interface for backend data persistence
type BackendRepository interface {
	Create(ctx context.Context, backend *models.ProviderConfiguration) error
	GetByID(ctx context.Context, id string) (*models.ProviderConfiguration, error)
	GetAll(ctx context.Context, filters BackendFilters) ([]*models.ProviderConfiguration, error)
	Update(ctx context.Context, backend *models.ProviderConfiguration) error
	Delete(ctx context.Context, id string) error
	TestConnection(ctx context.Context, backend *models.ProviderConfiguration) (*models.ProviderTestConnectionResponse, error)
}

// BackendFilters represents filters for backend queries
type BackendFilters struct {
	Provider  string `json:"provider,omitempty"`
	Region    string `json:"region,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
	IsDefault *bool  `json:"is_default,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// DiscoveryConfig represents configuration for backend discovery
type DiscoveryConfig struct {
	Provider    string                      `json:"provider" validate:"required"`
	Region      string                      `json:"region" validate:"required"`
	Credentials *models.ProviderCredentials `json:"credentials" validate:"required"`
	Settings    *models.ProviderSettings    `json:"settings,omitempty"`
	Options     DiscoveryOptions            `json:"options,omitempty"`
}

// DiscoveryOptions represents additional options for discovery
type DiscoveryOptions struct {
	IncludeResources []string `json:"include_resources,omitempty"`
	ExcludeResources []string `json:"exclude_resources,omitempty"`
	MaxConcurrency   int      `json:"max_concurrency,omitempty"`
	Timeout          int      `json:"timeout,omitempty"`
}

// DiscoveryResults represents the results of a discovery operation
type DiscoveryResults struct {
	TotalDiscovered   int                     `json:"total_discovered"`
	ResourcesByType   map[string]int          `json:"resources_by_type"`
	ResourcesByRegion map[string]int          `json:"resources_by_region"`
	NewResources      []string                `json:"new_resources"`
	UpdatedResources  []string                `json:"updated_resources"`
	DeletedResources  []string                `json:"deleted_resources"`
	Errors            []models.DiscoveryError `json:"errors"`
	Summary           map[string]interface{}  `json:"summary"`
	DiscoveredAt      time.Time               `json:"discovered_at"`
}

// ConnectionTest represents the result of a connection test
type ConnectionTest struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
	TestedAt time.Time              `json:"tested_at"`
	Duration time.Duration          `json:"duration"`
}

// NewBackendService creates a new backend service
func NewBackendService(repository BackendRepository) *BackendService {
	return &BackendService{
		repository: repository,
	}
}

// ListBackends retrieves all backends with optional filtering
func (s *BackendService) ListBackends(ctx context.Context, filters BackendFilters) ([]*models.ProviderConfiguration, error) {
	if filters.Limit <= 0 {
		filters.Limit = 100 // Default limit
	}

	backends, err := s.repository.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list backends: %w", err)
	}

	return backends, nil
}

// GetBackend retrieves a specific backend by ID
func (s *BackendService) GetBackend(ctx context.Context, id string) (*models.ProviderConfiguration, error) {
	backend, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend %s: %w", id, err)
	}

	return backend, nil
}

// CreateBackend creates a new backend configuration
func (s *BackendService) CreateBackend(ctx context.Context, req *models.ProviderConfigurationRequest) (*models.ProviderConfiguration, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid backend configuration: %w", err)
	}

	// Create the backend configuration
	backend := &models.ProviderConfiguration{
		ID:               generateBackendID(),
		Provider:         req.Provider,
		Name:             req.Name,
		Description:      req.Description,
		AccountID:        req.AccountID,
		Region:           req.Region,
		Credentials:      req.Credentials,
		Settings:         *req.Settings,
		IsActive:         true,
		IsDefault:        req.IsDefault,
		ConnectionStatus: models.ConnectionStatusUnknown,
		CreatedBy:        getCurrentUserID(ctx),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Save to repository
	if err := s.repository.Create(ctx, backend); err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	return backend, nil
}

// UpdateBackend updates an existing backend configuration
func (s *BackendService) UpdateBackend(ctx context.Context, id string, req *models.ProviderConfigurationUpdateRequest) (*models.ProviderConfiguration, error) {
	// Get existing backend
	backend, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend %s: %w", id, err)
	}

	// Update fields if provided
	if req.Name != nil {
		backend.Name = *req.Name
	}
	if req.Description != nil {
		backend.Description = *req.Description
	}
	if req.Credentials != nil {
		backend.Credentials = *req.Credentials
	}
	if req.Settings != nil {
		backend.Settings = *req.Settings
	}
	if req.IsActive != nil {
		backend.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		backend.IsDefault = *req.IsDefault
	}

	backend.UpdatedAt = time.Now()

	// Save to repository
	if err := s.repository.Update(ctx, backend); err != nil {
		return nil, fmt.Errorf("failed to update backend: %w", err)
	}

	return backend, nil
}

// DeleteBackend deletes a backend configuration
func (s *BackendService) DeleteBackend(ctx context.Context, id string) error {
	// Check if backend exists
	_, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get backend %s: %w", id, err)
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete backend: %w", err)
	}

	return nil
}

// TestBackendConnection tests the connection to a backend
func (s *BackendService) TestBackendConnection(ctx context.Context, id string) (*ConnectionTest, error) {
	start := time.Now()

	// Get backend configuration
	backend, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend %s: %w", id, err)
	}

	// Test connection based on provider
	var result *ConnectionTest
	switch backend.Provider {
	case models.ProviderAWS:
		result, err = s.testAWSConnection(ctx, backend)
	case models.ProviderAzure:
		result, err = s.testAzureConnection(ctx, backend)
	case models.ProviderGCP:
		result, err = s.testGCPConnection(ctx, backend)
	case models.ProviderDigitalOcean:
		result, err = s.testDigitalOceanConnection(ctx, backend)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", backend.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	result.Duration = time.Since(start)
	result.TestedAt = time.Now()

	// Update connection status in repository
	backend.ConnectionStatus = getConnectionStatus(result.Success)
	backend.LastConnected = &result.TestedAt
	s.repository.Update(ctx, backend)

	return result, nil
}

// DiscoverBackends performs resource discovery for a backend
func (s *BackendService) DiscoverBackends(ctx context.Context, config DiscoveryConfig) (*DiscoveryResults, error) {
	// Create AWS client for discovery
	awsClient, err := aws.NewClient(config.Region, config.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create discovery engine
	discoveryEngine := aws.NewDiscoveryEngine(awsClient)

	// Create discovery job
	job := &models.DiscoveryJob{
		ID:        generateDiscoveryJobID(),
		Provider:  models.CloudProvider(config.Provider),
		Region:    config.Region,
		Status:    models.JobStatusRunning,
		StartedAt: time.Now(),
		CreatedBy: getCurrentUserID(ctx),
		CreatedAt: time.Now(),
	}

	// Perform discovery
	results, err := discoveryEngine.DiscoverResources(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Convert to service results
	serviceResults := &DiscoveryResults{
		TotalDiscovered:   results.TotalDiscovered,
		ResourcesByType:   results.ResourcesByType,
		ResourcesByRegion: results.ResourcesByRegion,
		NewResources:      results.NewResources,
		UpdatedResources:  results.UpdatedResources,
		DeletedResources:  results.DeletedResources,
		Errors:            results.Errors,
		Summary:           results.Summary,
		DiscoveredAt:      time.Now(),
	}

	return serviceResults, nil
}

// testAWSConnection tests connection to AWS
func (s *BackendService) testAWSConnection(ctx context.Context, backend *models.ProviderConfiguration) (*ConnectionTest, error) {
	// Create AWS client
	awsClient, err := aws.NewClient(backend.Region, &backend.Credentials)
	if err != nil {
		return &ConnectionTest{
			Success: false,
			Message: fmt.Sprintf("Failed to create AWS client: %v", err),
		}, nil
	}

	// Test connection
	if err := awsClient.TestConnection(ctx); err != nil {
		return &ConnectionTest{
			Success: false,
			Message: fmt.Sprintf("AWS connection test failed: %v", err),
		}, nil
	}

	// Get account information
	accountID, err := awsClient.GetAccountID(ctx)
	if err != nil {
		return &ConnectionTest{
			Success: false,
			Message: fmt.Sprintf("Failed to get AWS account ID: %v", err),
		}, nil
	}

	// Get available regions
	regions, err := awsClient.GetAvailableRegions(ctx)
	if err != nil {
		return &ConnectionTest{
			Success: false,
			Message: fmt.Sprintf("Failed to get AWS regions: %v", err),
		}, nil
	}

	return &ConnectionTest{
		Success: true,
		Message: "AWS connection successful",
		Details: map[string]interface{}{
			"account_id":        accountID,
			"region":            backend.Region,
			"available_regions": regions,
			"provider":          "aws",
		},
	}, nil
}

// testAzureConnection tests connection to Azure (placeholder)
func (s *BackendService) testAzureConnection(ctx context.Context, backend *models.ProviderConfiguration) (*ConnectionTest, error) {
	return &ConnectionTest{
		Success: false,
		Message: "Azure connection testing not yet implemented",
	}, nil
}

// testGCPConnection tests connection to GCP (placeholder)
func (s *BackendService) testGCPConnection(ctx context.Context, backend *models.ProviderConfiguration) (*ConnectionTest, error) {
	return &ConnectionTest{
		Success: false,
		Message: "GCP connection testing not yet implemented",
	}, nil
}

// testDigitalOceanConnection tests connection to DigitalOcean (placeholder)
func (s *BackendService) testDigitalOceanConnection(ctx context.Context, backend *models.ProviderConfiguration) (*ConnectionTest, error) {
	return &ConnectionTest{
		Success: false,
		Message: "DigitalOcean connection testing not yet implemented",
	}, nil
}

// Helper functions

func generateBackendID() string {
	// In a real implementation, this would generate a proper UUID
	return fmt.Sprintf("backend_%d", time.Now().UnixNano())
}

func getConnectionStatus(success bool) models.ConnectionStatus {
	if success {
		return models.ConnectionStatusConnected
	}
	return models.ConnectionStatusError
}
