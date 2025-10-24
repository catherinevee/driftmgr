package services

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// DriftService handles drift detection and management business logic
type DriftService struct {
	repository      DriftRepository
	stateService    *StateService
	resourceService *ResourceService
}

// DriftRepository defines the interface for drift data persistence
type DriftRepository interface {
	Create(ctx context.Context, result *models.DriftResult) error
	GetByID(ctx context.Context, id string) (*models.DriftResult, error)
	GetAll(ctx context.Context, filters DriftFilters) ([]*models.DriftResult, error)
	Update(ctx context.Context, result *models.DriftResult) error
	Delete(ctx context.Context, id string) error
	GetDriftHistory(ctx context.Context, resourceID string) ([]*models.DriftEvent, error)
	GetDriftSummary(ctx context.Context, filters DriftFilters) (*DriftSummary, error)
}

// DriftFilters represents filters for drift queries
type DriftFilters struct {
	ResourceID   string    `json:"resource_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	Region       string    `json:"region,omitempty"`
	Severity     string    `json:"severity,omitempty"`
	Status       string    `json:"status,omitempty"`
	StartDate    time.Time `json:"start_date,omitempty"`
	EndDate      time.Time `json:"end_date,omitempty"`
	CreatedBy    string    `json:"created_by,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	Offset       int       `json:"offset,omitempty"`
}

// DriftConfig represents configuration for drift detection
type DriftConfig struct {
	ResourceID   string       `json:"resource_id,omitempty"`
	ResourceType string       `json:"resource_type,omitempty"`
	Provider     string       `json:"provider,omitempty"`
	Region       string       `json:"region,omitempty"`
	StateID      string       `json:"state_id,omitempty"`
	Options      DriftOptions `json:"options,omitempty"`
}

// DriftOptions represents additional options for drift detection
type DriftOptions struct {
	DeepScan        bool     `json:"deep_scan,omitempty"`
	IncludeMetadata bool     `json:"include_metadata,omitempty"`
	IncludeTags     bool     `json:"include_tags,omitempty"`
	IgnoreFields    []string `json:"ignore_fields,omitempty"`
	Timeout         int      `json:"timeout,omitempty"`
}

// DriftResults represents the results of a drift detection operation
type DriftResults struct {
	ID            string                 `json:"id"`
	ResourceID    string                 `json:"resource_id"`
	ResourceType  string                 `json:"resource_type"`
	Provider      string                 `json:"provider"`
	Region        string                 `json:"region"`
	StateID       string                 `json:"state_id"`
	Status        string                 `json:"status"`
	DriftDetected bool                   `json:"drift_detected"`
	DriftCount    int                    `json:"drift_count"`
	DriftDetails  []DriftDetail          `json:"drift_details"`
	Summary       map[string]interface{} `json:"summary"`
	DetectedAt    time.Time              `json:"detected_at"`
	Duration      time.Duration          `json:"duration"`
	CreatedBy     string                 `json:"created_by"`
}

// DriftDetail represents a specific drift finding
type DriftDetail struct {
	Field         string      `json:"field"`
	ExpectedValue interface{} `json:"expected_value"`
	ActualValue   interface{} `json:"actual_value"`
	Severity      string      `json:"severity"`
	Description   string      `json:"description"`
	Remediation   string      `json:"remediation"`
}

// DriftSummary represents a summary of drift detection results
type DriftSummary struct {
	TotalResources   int                    `json:"total_resources"`
	DriftedResources int                    `json:"drifted_resources"`
	DriftPercentage  float64                `json:"drift_percentage"`
	DriftsByType     map[string]int         `json:"drifts_by_type"`
	DriftsByProvider map[string]int         `json:"drifts_by_provider"`
	DriftsByRegion   map[string]int         `json:"drifts_by_region"`
	DriftsBySeverity map[string]int         `json:"drifts_by_severity"`
	RecentDrifts     []*models.DriftResult  `json:"recent_drifts"`
	Summary          map[string]interface{} `json:"summary"`
	GeneratedAt      time.Time              `json:"generated_at"`
}

// NewDriftService creates a new drift service
func NewDriftService(repository DriftRepository, stateService *StateService, resourceService *ResourceService) *DriftService {
	return &DriftService{
		repository:      repository,
		stateService:    stateService,
		resourceService: resourceService,
	}
}

// DetectDrift performs drift detection for resources
func (s *DriftService) DetectDrift(ctx context.Context, config DriftConfig) (*DriftResults, error) {
	start := time.Now()

	// Generate drift detection ID
	driftID := generateDriftID()

	// Create drift result
	result := &DriftResults{
		ID:           driftID,
		ResourceID:   config.ResourceID,
		ResourceType: config.ResourceType,
		Provider:     config.Provider,
		Region:       config.Region,
		StateID:      config.StateID,
		Status:       "running",
		DetectedAt:   time.Now(),
		CreatedBy:    getCurrentUserID(ctx),
	}

	// Perform drift detection based on configuration
	var err error
	if config.ResourceID != "" {
		// Single resource drift detection
		err = s.detectResourceDrift(ctx, result, config)
	} else if config.ResourceType != "" {
		// Resource type drift detection
		err = s.detectResourceTypeDrift(ctx, result, config)
	} else if config.Provider != "" {
		// Provider-wide drift detection
		err = s.detectProviderDrift(ctx, result, config)
	} else {
		return nil, fmt.Errorf("invalid drift detection configuration: no target specified")
	}

	if err != nil {
		result.Status = "failed"
		result.Summary = map[string]interface{}{
			"error": err.Error(),
		}
	} else {
		result.Status = "completed"
	}

	result.Duration = time.Since(start)

	// Save drift result to repository
	driftResult := &models.DriftResult{
		ID:         result.ID,
		Timestamp:  result.DetectedAt,
		Provider:   result.Provider,
		Status:     result.Status,
		DriftCount: result.DriftCount,
		Resources:  []models.DriftedResource{}, // Convert from DriftDetail to DriftedResource
		Summary:    models.DriftSummary{},      // Convert from service summary
		Duration:   result.Duration,
		Error:      nil, // Set if there was an error
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repository.Create(ctx, driftResult); err != nil {
		return nil, fmt.Errorf("failed to save drift result: %w", err)
	}

	return result, nil
}

// GetDriftResults retrieves drift detection results with optional filtering
func (s *DriftService) GetDriftResults(ctx context.Context, filters DriftFilters) ([]*models.DriftResult, error) {
	if filters.Limit <= 0 {
		filters.Limit = 100 // Default limit
	}

	results, err := s.repository.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift results: %w", err)
	}

	return results, nil
}

// GetDriftResult retrieves a specific drift result by ID
func (s *DriftService) GetDriftResult(ctx context.Context, id string) (*models.DriftResult, error) {
	result, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift result %s: %w", id, err)
	}

	return result, nil
}

// DeleteDriftResult deletes a drift result
func (s *DriftService) DeleteDriftResult(ctx context.Context, id string) error {
	// Check if drift result exists
	_, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("drift result %s not found: %w", id, err)
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete drift result: %w", err)
	}

	return nil
}

// GetDriftHistory retrieves drift history for a resource
func (s *DriftService) GetDriftHistory(ctx context.Context, resourceID string) ([]*models.DriftEvent, error) {
	events, err := s.repository.GetDriftHistory(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift history for resource %s: %w", resourceID, err)
	}

	return events, nil
}

// GetDriftSummary retrieves a summary of drift detection results
func (s *DriftService) GetDriftSummary(ctx context.Context, filters DriftFilters) (*DriftSummary, error) {
	summary, err := s.repository.GetDriftSummary(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get drift summary: %w", err)
	}

	return summary, nil
}

// detectResourceDrift performs drift detection for a single resource
func (s *DriftService) detectResourceDrift(ctx context.Context, result *DriftResults, config DriftConfig) error {
	// Get current resource state
	resource, err := s.resourceService.GetResource(ctx, config.ResourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Get expected state from Terraform state
	var expectedState map[string]interface{}
	if config.StateID != "" {
		stateDetails, err := s.stateService.GetStateDetails(ctx, config.StateID)
		if err != nil {
			return fmt.Errorf("failed to get state details: %w", err)
		}
		// Find the resource in the state
		expectedState = s.findResourceInState(stateDetails, resource.Type, resource.Name)
	}

	// Compare current vs expected state
	driftDetails := s.compareResourceStates(resource, expectedState, config.Options)

	result.DriftDetails = driftDetails
	result.DriftDetected = len(driftDetails) > 0
	result.DriftCount = len(driftDetails)

	// Update result fields
	result.ResourceID = resource.ID
	result.ResourceType = resource.Type
	result.Provider = string(resource.Provider)
	result.Region = resource.Region

	return nil
}

// detectResourceTypeDrift performs drift detection for a resource type
func (s *DriftService) detectResourceTypeDrift(ctx context.Context, result *DriftResults, config DriftConfig) error {
	// Get all resources of the specified type
	filters := ResourceFilters{
		Type:     config.ResourceType,
		Provider: config.Provider,
		Region:   config.Region,
		Limit:    1000, // Large limit for type-wide detection
	}

	resources, err := s.resourceService.ListResources(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	// Perform drift detection for each resource
	var allDriftDetails []DriftDetail
	driftedCount := 0

	for _, resource := range resources {
		// Get expected state from Terraform state
		var expectedState map[string]interface{}
		if config.StateID != "" {
			stateDetails, err := s.stateService.GetStateDetails(ctx, config.StateID)
			if err != nil {
				continue // Skip this resource if state not available
			}
			expectedState = s.findResourceInState(stateDetails, resource.Type, resource.Name)
		}

		// Compare states
		driftDetails := s.compareResourceStates(resource, expectedState, config.Options)
		if len(driftDetails) > 0 {
			driftedCount++
			allDriftDetails = append(allDriftDetails, driftDetails...)
		}
	}

	result.DriftDetails = allDriftDetails
	result.DriftDetected = len(allDriftDetails) > 0
	result.DriftCount = len(allDriftDetails)
	result.ResourceType = config.ResourceType
	result.Provider = config.Provider
	result.Region = config.Region

	return nil
}

// detectProviderDrift performs drift detection for a provider
func (s *DriftService) detectProviderDrift(ctx context.Context, result *DriftResults, config DriftConfig) error {
	// Get all resources for the provider
	filters := ResourceFilters{
		Provider: config.Provider,
		Region:   config.Region,
		Limit:    1000, // Large limit for provider-wide detection
	}

	resources, err := s.resourceService.ListResources(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	// Perform drift detection for each resource
	var allDriftDetails []DriftDetail
	driftedCount := 0

	for _, resource := range resources {
		// Get expected state from Terraform state
		var expectedState map[string]interface{}
		if config.StateID != "" {
			stateDetails, err := s.stateService.GetStateDetails(ctx, config.StateID)
			if err != nil {
				continue // Skip this resource if state not available
			}
			expectedState = s.findResourceInState(stateDetails, resource.Type, resource.Name)
		}

		// Compare states
		driftDetails := s.compareResourceStates(resource, expectedState, config.Options)
		if len(driftDetails) > 0 {
			driftedCount++
			allDriftDetails = append(allDriftDetails, driftDetails...)
		}
	}

	result.DriftDetails = allDriftDetails
	result.DriftDetected = len(allDriftDetails) > 0
	result.DriftCount = len(allDriftDetails)
	result.Provider = config.Provider
	result.Region = config.Region

	return nil
}

// findResourceInState finds a resource in the Terraform state
func (s *DriftService) findResourceInState(stateDetails *StateDetails, resourceType, resourceName string) map[string]interface{} {
	for _, resource := range stateDetails.Resources {
		if resource.Type == resourceType && resource.Name == resourceName {
			return resource.Config
		}
	}
	return nil
}

// compareResourceStates compares current and expected resource states
func (s *DriftService) compareResourceStates(resource *models.CloudResource, expectedState map[string]interface{}, options DriftOptions) []DriftDetail {
	var driftDetails []DriftDetail

	if expectedState == nil {
		// No expected state found - this could be a new resource or missing from state
		driftDetails = append(driftDetails, DriftDetail{
			Field:         "state_presence",
			ExpectedValue: "present",
			ActualValue:   "missing",
			Severity:      "warning",
			Description:   "Resource not found in Terraform state",
			Remediation:   "Consider importing the resource into Terraform state",
		})
		return driftDetails
	}

	// Compare configuration fields
	if options.IncludeMetadata {
		driftDetails = append(driftDetails, s.compareMaps(resource.Metadata, expectedState, "metadata")...)
	}

	if options.IncludeTags {
		// Convert map[string]string to map[string]interface{}
		tagsInterface := make(map[string]interface{})
		for k, v := range resource.Tags {
			tagsInterface[k] = v
		}
		driftDetails = append(driftDetails, s.compareMaps(tagsInterface, expectedState, "tags")...)
	}

	// Compare other configuration fields
	driftDetails = append(driftDetails, s.compareMaps(resource.Configuration, expectedState, "configuration")...)

	return driftDetails
}

// compareMaps compares two maps and returns drift details
func (s *DriftService) compareMaps(actual, expected map[string]interface{}, prefix string) []DriftDetail {
	var driftDetails []DriftDetail

	if actual == nil {
		actual = make(map[string]interface{})
	}
	if expected == nil {
		expected = make(map[string]interface{})
	}

	// Check for missing fields in actual
	for key, expectedValue := range expected {
		if actualValue, exists := actual[key]; !exists {
			driftDetails = append(driftDetails, DriftDetail{
				Field:         fmt.Sprintf("%s.%s", prefix, key),
				ExpectedValue: expectedValue,
				ActualValue:   nil,
				Severity:      "warning",
				Description:   fmt.Sprintf("Field %s.%s is missing", prefix, key),
				Remediation:   "Add the missing field to the resource configuration",
			})
		} else if !s.valuesEqual(actualValue, expectedValue) {
			driftDetails = append(driftDetails, DriftDetail{
				Field:         fmt.Sprintf("%s.%s", prefix, key),
				ExpectedValue: expectedValue,
				ActualValue:   actualValue,
				Severity:      "error",
				Description:   fmt.Sprintf("Field %s.%s has drifted", prefix, key),
				Remediation:   "Update the resource configuration to match expected state",
			})
		}
	}

	// Check for extra fields in actual
	for key, actualValue := range actual {
		if _, exists := expected[key]; !exists {
			driftDetails = append(driftDetails, DriftDetail{
				Field:         fmt.Sprintf("%s.%s", prefix, key),
				ExpectedValue: nil,
				ActualValue:   actualValue,
				Severity:      "info",
				Description:   fmt.Sprintf("Field %s.%s is not in expected state", prefix, key),
				Remediation:   "Consider adding this field to the expected configuration",
			})
		}
	}

	return driftDetails
}

// valuesEqual compares two values for equality
func (s *DriftService) valuesEqual(a, b interface{}) bool {
	// Simple equality check - in a real implementation, this would be more sophisticated
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// Helper functions
