package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Manager represents a resource manager
type Manager struct {
	catalog    *Catalog
	metadata   *MetadataManager
	tagging    *TaggingManager
	compliance *ComplianceManager
	cost       *CostManager
	mu         sync.RWMutex
}

// NewManager creates a new resource manager
func NewManager() *Manager {
	return &Manager{
		catalog:    NewCatalog(),
		metadata:   NewMetadataManager(),
		tagging:    NewTaggingManager(),
		compliance: NewComplianceManager(),
		cost:       NewCostManager(),
	}
}

// AddResource adds a resource to the catalog
func (m *Manager) AddResource(ctx context.Context, resource *models.CloudResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to catalog
	if err := m.catalog.Add(resource); err != nil {
		return fmt.Errorf("failed to add resource to catalog: %w", err)
	}

	// Update metadata
	if err := m.metadata.Update(ctx, resource); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Update compliance status
	if err := m.compliance.CheckCompliance(ctx, resource); err != nil {
		return fmt.Errorf("failed to check compliance: %w", err)
	}

	// Update cost information
	if err := m.cost.UpdateCost(ctx, resource); err != nil {
		return fmt.Errorf("failed to update cost: %w", err)
	}

	return nil
}

// UpdateResource updates an existing resource
func (m *Manager) UpdateResource(ctx context.Context, resource *models.CloudResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update in catalog
	if err := m.catalog.Update(resource); err != nil {
		return fmt.Errorf("failed to update resource in catalog: %w", err)
	}

	// Update metadata
	if err := m.metadata.Update(ctx, resource); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Re-check compliance
	if err := m.compliance.CheckCompliance(ctx, resource); err != nil {
		return fmt.Errorf("failed to check compliance: %w", err)
	}

	// Update cost information
	if err := m.cost.UpdateCost(ctx, resource); err != nil {
		return fmt.Errorf("failed to update cost: %w", err)
	}

	return nil
}

// RemoveResource removes a resource from the catalog
func (m *Manager) RemoveResource(ctx context.Context, resourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove from catalog
	if err := m.catalog.Remove(resourceID); err != nil {
		return fmt.Errorf("failed to remove resource from catalog: %w", err)
	}

	// Clean up metadata
	if err := m.metadata.Delete(ctx, resourceID); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	// Clean up compliance data
	if err := m.compliance.DeleteCompliance(ctx, resourceID); err != nil {
		return fmt.Errorf("failed to delete compliance data: %w", err)
	}

	// Clean up cost data
	if err := m.cost.DeleteCost(ctx, resourceID); err != nil {
		return fmt.Errorf("failed to delete cost data: %w", err)
	}

	return nil
}

// GetResource retrieves a resource by ID
func (m *Manager) GetResource(ctx context.Context, resourceID string) (*models.CloudResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resource, err := m.catalog.Get(resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	// Enrich with metadata
	if err := m.metadata.Enrich(ctx, resource); err != nil {
		return nil, fmt.Errorf("failed to enrich with metadata: %w", err)
	}

	// Enrich with compliance data
	if err := m.compliance.Enrich(ctx, resource); err != nil {
		return nil, fmt.Errorf("failed to enrich with compliance data: %w", err)
	}

	// Enrich with cost data
	if err := m.cost.Enrich(ctx, resource); err != nil {
		return nil, fmt.Errorf("failed to enrich with cost data: %w", err)
	}

	return resource, nil
}

// ListResources lists resources with optional filtering
func (m *Manager) ListResources(ctx context.Context, req *models.ResourceListRequest) (*models.ResourceListResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resources, total, err := m.catalog.List(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Enrich resources with additional data
	for _, resource := range resources {
		if err := m.metadata.Enrich(ctx, &resource); err != nil {
			// Log error but continue
			continue
		}
		if err := m.compliance.Enrich(ctx, &resource); err != nil {
			// Log error but continue
			continue
		}
		if err := m.cost.Enrich(ctx, &resource); err != nil {
			// Log error but continue
			continue
		}
	}

	return &models.ResourceListResponse{
		Resources: resources,
		Total:     total,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}, nil
}

// SearchResources searches for resources
func (m *Manager) SearchResources(ctx context.Context, req *models.ResourceSearchRequest) (*models.ResourceSearchResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resources, total, err := m.catalog.Search(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search resources: %w", err)
	}

	// Enrich resources with additional data
	for _, resource := range resources {
		if err := m.metadata.Enrich(ctx, &resource); err != nil {
			// Log error but continue
			continue
		}
		if err := m.compliance.Enrich(ctx, &resource); err != nil {
			// Log error but continue
			continue
		}
		if err := m.cost.Enrich(ctx, &resource); err != nil {
			// Log error but continue
			continue
		}
	}

	return &models.ResourceSearchResponse{
		Resources: resources,
		Total:     total,
		Limit:     req.Limit,
		Offset:    req.Offset,
		Query:     req.Query,
	}, nil
}

// GetResourceRelationships returns relationships for a resource
func (m *Manager) GetResourceRelationships(ctx context.Context, resourceID string) ([]models.ResourceRelationship, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.catalog.GetRelationships(resourceID)
}

// UpdateResourceTags updates tags for a resource
func (m *Manager) UpdateResourceTags(ctx context.Context, resourceID string, tags map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update tags
	if err := m.tagging.UpdateTags(ctx, resourceID, tags); err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}

	// Update resource in catalog
	resource, err := m.catalog.Get(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	resource.Tags = tags
	resource.UpdatedAt = time.Now()

	if err := m.catalog.Update(resource); err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	return nil
}

// GetResourceCompliance returns compliance status for a resource
func (m *Manager) GetResourceCompliance(ctx context.Context, resourceID string) (*models.ComplianceStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.compliance.GetCompliance(ctx, resourceID)
}

// GetResourceCost returns cost information for a resource
func (m *Manager) GetResourceCost(ctx context.Context, resourceID string) (*models.CostInformation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cost.GetCost(ctx, resourceID)
}

// GetResourceStatistics returns statistics about resources
func (m *Manager) GetResourceStatistics(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	// Get catalog statistics
	catalogStats := m.catalog.GetStatistics()
	stats["catalog"] = catalogStats

	// Get metadata statistics
	metadataStats := m.metadata.GetStatistics(ctx)
	stats["metadata"] = metadataStats

	// Get compliance statistics
	complianceStats := m.compliance.GetStatistics(ctx)
	stats["compliance"] = complianceStats

	// Get cost statistics
	costStats := m.cost.GetStatistics(ctx)
	stats["cost"] = costStats

	return stats, nil
}

// GetResourcesByProvider returns resources grouped by provider
func (m *Manager) GetResourcesByProvider(ctx context.Context) (map[models.CloudProvider][]models.CloudResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.catalog.GetByProvider()
}

// GetResourcesByType returns resources grouped by type
func (m *Manager) GetResourcesByType(ctx context.Context) (map[string][]models.CloudResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.catalog.GetByType()
}

// GetResourcesByRegion returns resources grouped by region
func (m *Manager) GetResourcesByRegion(ctx context.Context) (map[string][]models.CloudResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.catalog.GetByRegion()
}

// GetResourcesByAccount returns resources grouped by account
func (m *Manager) GetResourcesByAccount(ctx context.Context) (map[string][]models.CloudResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.catalog.GetByAccount()
}

// CompareResources compares two resources
func (m *Manager) CompareResources(ctx context.Context, resourceID1, resourceID2 string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resource1, err := m.catalog.Get(resourceID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource 1: %w", err)
	}

	resource2, err := m.catalog.Get(resourceID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource 2: %w", err)
	}

	comparison := make(map[string]interface{})
	comparison["resource1"] = resource1
	comparison["resource2"] = resource2
	comparison["differences"] = m.findDifferences(resource1, resource2)

	return comparison, nil
}

// findDifferences finds differences between two resources
func (m *Manager) findDifferences(resource1, resource2 *models.CloudResource) []string {
	var differences []string

	// Compare basic fields
	if resource1.Provider != resource2.Provider {
		differences = append(differences, "provider")
	}
	if resource1.Type != resource2.Type {
		differences = append(differences, "type")
	}
	if resource1.Name != resource2.Name {
		differences = append(differences, "name")
	}
	if resource1.Region != resource2.Region {
		differences = append(differences, "region")
	}

	// Compare tags
	if len(resource1.Tags) != len(resource2.Tags) {
		differences = append(differences, "tags")
	} else {
		for key, value := range resource1.Tags {
			if resource2.Tags[key] != value {
				differences = append(differences, "tags")
				break
			}
		}
	}

	return differences
}

// Health checks the health of the resource manager
func (m *Manager) Health(ctx context.Context) error {
	// Check catalog health
	if err := m.catalog.Health(); err != nil {
		return fmt.Errorf("catalog health check failed: %w", err)
	}

	// Check metadata manager health
	if err := m.metadata.Health(ctx); err != nil {
		return fmt.Errorf("metadata manager health check failed: %w", err)
	}

	// Check compliance manager health
	if err := m.compliance.Health(ctx); err != nil {
		return fmt.Errorf("compliance manager health check failed: %w", err)
	}

	// Check cost manager health
	if err := m.cost.Health(ctx); err != nil {
		return fmt.Errorf("cost manager health check failed: %w", err)
	}

	return nil
}
