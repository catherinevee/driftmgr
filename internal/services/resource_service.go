package services

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/providers/aws"
)

// ResourceService handles cloud resource management business logic
type ResourceService struct {
	repository      ResourceRepository
	discoveryEngine *aws.DiscoveryEngine
	awsClient       *aws.Client
}

// ResourceRepository defines the interface for resource data persistence
type ResourceRepository interface {
	Create(ctx context.Context, resource *models.CloudResource) error
	GetByID(ctx context.Context, id string) (*models.CloudResource, error)
	GetAll(ctx context.Context, filters ResourceFilters) ([]*models.CloudResource, error)
	Update(ctx context.Context, resource *models.CloudResource) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query ResourceSearchQuery) ([]*models.CloudResource, error)
	UpdateTags(ctx context.Context, resourceID string, tags map[string]string) error
	GetResourceCost(ctx context.Context, resourceID string) (*models.CostInformation, error)
	GetResourceCompliance(ctx context.Context, resourceID string) (*models.ComplianceStatus, error)
}

// ResourceFilters represents filters for resource queries
type ResourceFilters struct {
	Provider  string            `json:"provider,omitempty"`
	Type      string            `json:"type,omitempty"`
	Region    string            `json:"region,omitempty"`
	AccountID string            `json:"account_id,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	State     string            `json:"state,omitempty"`
	CreatedBy string            `json:"created_by,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
}

// ResourceSearchQuery represents a search query for resources
type ResourceSearchQuery struct {
	Query     string          `json:"query"`
	Filters   ResourceFilters `json:"filters,omitempty"`
	SortBy    string          `json:"sort_by,omitempty"`
	SortOrder string          `json:"sort_order,omitempty"`
	Limit     int             `json:"limit,omitempty"`
	Offset    int             `json:"offset,omitempty"`
}

// ResourceCost represents cost information for a resource
type ResourceCost struct {
	ResourceID   string                 `json:"resource_id"`
	Provider     string                 `json:"provider"`
	Type         string                 `json:"type"`
	Region       string                 `json:"region"`
	CostPerHour  float64                `json:"cost_per_hour"`
	CostPerMonth float64                `json:"cost_per_month"`
	Currency     string                 `json:"currency"`
	LastUpdated  time.Time              `json:"last_updated"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// ResourceCompliance represents compliance status for a resource
type ResourceCompliance struct {
	ResourceID  string                 `json:"resource_id"`
	Provider    string                 `json:"provider"`
	Type        string                 `json:"type"`
	Compliant   bool                   `json:"compliant"`
	Score       float64                `json:"score"`
	Violations  []ComplianceViolation  `json:"violations"`
	LastChecked time.Time              `json:"last_checked"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// NewResourceService creates a new resource service
func NewResourceService(repository ResourceRepository) *ResourceService {
	return &ResourceService{
		repository: repository,
	}
}

// SetAWSClient sets the AWS client for the service
func (s *ResourceService) SetAWSClient(client *aws.Client) {
	s.awsClient = client
	if client != nil {
		s.discoveryEngine = aws.NewDiscoveryEngine(client)
	}
}

// ListResources retrieves all resources with optional filtering
func (s *ResourceService) ListResources(ctx context.Context, filters ResourceFilters) ([]*models.CloudResource, error) {
	if filters.Limit <= 0 {
		filters.Limit = 100 // Default limit
	}

	resources, err := s.repository.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return resources, nil
}

// GetResource retrieves a specific resource by ID
func (s *ResourceService) GetResource(ctx context.Context, id string) (*models.CloudResource, error) {
	resource, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", id, err)
	}

	return resource, nil
}

// SearchResources searches for resources based on query and filters
func (s *ResourceService) SearchResources(ctx context.Context, query ResourceSearchQuery) ([]*models.CloudResource, error) {
	if query.Limit <= 0 {
		query.Limit = 100 // Default limit
	}

	resources, err := s.repository.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search resources: %w", err)
	}

	return resources, nil
}

// CreateResource creates a new resource record
func (s *ResourceService) CreateResource(ctx context.Context, resource *models.CloudResource) (*models.CloudResource, error) {
	// Validate the resource
	if err := s.validateResource(resource); err != nil {
		return nil, fmt.Errorf("invalid resource: %w", err)
	}

	// Set timestamps
	now := time.Now()
	resource.LastDiscovered = now
	resource.CreatedAt = now
	resource.UpdatedAt = now

	// Save to repository
	if err := s.repository.Create(ctx, resource); err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	return resource, nil
}

// UpdateResource updates an existing resource
func (s *ResourceService) UpdateResource(ctx context.Context, id string, updates map[string]interface{}) (*models.CloudResource, error) {
	// Get existing resource
	resource, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", id, err)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		resource.Name = name
	}
	if tags, ok := updates["tags"].(map[string]string); ok {
		resource.Tags = tags
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		resource.Metadata = metadata
	}
	if configuration, ok := updates["configuration"].(map[string]interface{}); ok {
		resource.Configuration = configuration
	}

	resource.UpdatedAt = time.Now()

	// Save to repository
	if err := s.repository.Update(ctx, resource); err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	return resource, nil
}

// DeleteResource deletes a resource record
func (s *ResourceService) DeleteResource(ctx context.Context, id string) error {
	// Check if resource exists
	_, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get resource %s: %w", id, err)
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return nil
}

// UpdateResourceTags updates tags for a resource
func (s *ResourceService) UpdateResourceTags(ctx context.Context, resourceID string, tags map[string]string) error {
	// Validate resource exists
	_, err := s.repository.GetByID(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("resource %s not found: %w", resourceID, err)
	}

	// Update tags
	if err := s.repository.UpdateTags(ctx, resourceID, tags); err != nil {
		return fmt.Errorf("failed to update resource tags: %w", err)
	}

	return nil
}

// GetResourceCost retrieves cost information for a resource
func (s *ResourceService) GetResourceCost(ctx context.Context, resourceID string) (*ResourceCost, error) {
	// Get resource
	resource, err := s.repository.GetByID(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("resource %s not found: %w", resourceID, err)
	}

	// Get cost information from repository
	costInfo, err := s.repository.GetResourceCost(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost information: %w", err)
	}

	// Convert to service response
	cost := &ResourceCost{
		ResourceID:   resourceID,
		Provider:     string(resource.Provider),
		Type:         resource.Type,
		Region:       resource.Region,
		CostPerHour:  costInfo.HourlyCost,
		CostPerMonth: costInfo.MonthlyCost,
		Currency:     costInfo.Currency,
		LastUpdated:  costInfo.LastUpdated,
		Details:      convertCostBreakdown(costInfo.CostBreakdown),
	}

	return cost, nil
}

// GetResourceCompliance retrieves compliance status for a resource
func (s *ResourceService) GetResourceCompliance(ctx context.Context, resourceID string) (*ResourceCompliance, error) {
	// Get resource
	resource, err := s.repository.GetByID(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("resource %s not found: %w", resourceID, err)
	}

	// Get compliance information from repository
	complianceInfo, err := s.repository.GetResourceCompliance(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance information: %w", err)
	}

	// Convert violations
	var violations []ComplianceViolation
	for _, v := range complianceInfo.Violations {
		violations = append(violations, ComplianceViolation{
			RuleID:      v.RuleID,
			RuleName:    v.RuleName,
			Severity:    v.Severity,
			Description: v.Description,
			Remediation: v.Remediation,
		})
	}

	// Convert to service response
	compliance := &ResourceCompliance{
		ResourceID:  resourceID,
		Provider:    string(resource.Provider),
		Type:        resource.Type,
		Compliant:   complianceInfo.Status == models.ComplianceLevelCompliant,
		Score:       0.0, // Calculate from violations
		Violations:  violations,
		LastChecked: complianceInfo.LastChecked,
		Details:     map[string]interface{}{"policy_id": complianceInfo.PolicyID, "policy_name": complianceInfo.PolicyName},
	}

	return compliance, nil
}

// DiscoverResources performs resource discovery using the AWS discovery engine
func (s *ResourceService) DiscoverResources(ctx context.Context, provider string, region string) ([]*models.CloudResource, error) {
	if s.discoveryEngine == nil {
		return nil, fmt.Errorf("AWS discovery engine not initialized")
	}

	// Create discovery job
	job := &models.DiscoveryJob{
		ID:        generateDiscoveryJobID(),
		Provider:  models.CloudProvider(provider),
		Region:    region,
		Status:    models.JobStatusRunning,
		StartedAt: time.Now(),
		CreatedBy: getCurrentUserID(ctx),
		CreatedAt: time.Now(),
	}

	// Perform discovery
	_, err := s.discoveryEngine.DiscoverResources(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Convert results to CloudResource models
	var resources []*models.CloudResource
	// Note: This would need to be implemented based on the actual discovery results
	// For now, return empty slice as the discovery engine returns DiscoveryResults
	// which would need to be converted to individual CloudResource objects

	return resources, nil
}

// RefreshResource refreshes a resource by re-discovering it
func (s *ResourceService) RefreshResource(ctx context.Context, resourceID string) (*models.CloudResource, error) {
	// Get existing resource
	resource, err := s.repository.GetByID(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("resource %s not found: %w", resourceID, err)
	}

	// Re-discover the resource
	if s.discoveryEngine != nil {
		// This would involve calling the discovery engine for a specific resource
		// For now, just update the last discovered timestamp
		resource.LastDiscovered = time.Now()
		resource.UpdatedAt = time.Now()

		// Save updated resource
		if err := s.repository.Update(ctx, resource); err != nil {
			return nil, fmt.Errorf("failed to update resource: %w", err)
		}
	}

	return resource, nil
}

// validateResource validates a resource before creation/update
func (s *ResourceService) validateResource(resource *models.CloudResource) error {
	if resource.ID == "" {
		return fmt.Errorf("resource ID is required")
	}
	if resource.Type == "" {
		return fmt.Errorf("resource type is required")
	}
	if resource.Provider == "" {
		return fmt.Errorf("resource provider is required")
	}
	if resource.Region == "" {
		return fmt.Errorf("resource region is required")
	}
	if resource.Name == "" {
		return fmt.Errorf("resource name is required")
	}

	return nil
}

// Helper functions

// convertCostBreakdown converts map[string]float64 to map[string]interface{}
func convertCostBreakdown(breakdown map[string]float64) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range breakdown {
		result[k] = v
	}
	return result
}
