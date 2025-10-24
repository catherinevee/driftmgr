package resources

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Catalog represents a resource catalog
type Catalog struct {
	resources     map[string]*models.CloudResource
	byProvider    map[models.CloudProvider][]*models.CloudResource
	byType        map[string][]*models.CloudResource
	byRegion      map[string][]*models.CloudResource
	byAccount     map[string][]*models.CloudResource
	relationships map[string][]models.ResourceRelationship
	mu            sync.RWMutex
}

// NewCatalog creates a new resource catalog
func NewCatalog() *Catalog {
	return &Catalog{
		resources:     make(map[string]*models.CloudResource),
		byProvider:    make(map[models.CloudProvider][]*models.CloudResource),
		byType:        make(map[string][]*models.CloudResource),
		byRegion:      make(map[string][]*models.CloudResource),
		byAccount:     make(map[string][]*models.CloudResource),
		relationships: make(map[string][]models.ResourceRelationship),
	}
}

// Add adds a resource to the catalog
func (c *Catalog) Add(resource *models.CloudResource) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.resources[resource.ID] != nil {
		return fmt.Errorf("resource %s already exists", resource.ID)
	}

	c.resources[resource.ID] = resource
	c.addToIndexes(resource)

	return nil
}

// Update updates a resource in the catalog
func (c *Catalog) Update(resource *models.CloudResource) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.resources[resource.ID] == nil {
		return fmt.Errorf("resource %s not found", resource.ID)
	}

	// Remove from indexes
	c.removeFromIndexes(c.resources[resource.ID])

	// Update resource
	c.resources[resource.ID] = resource
	c.addToIndexes(resource)

	return nil
}

// Remove removes a resource from the catalog
func (c *Catalog) Remove(resourceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	resource := c.resources[resourceID]
	if resource == nil {
		return fmt.Errorf("resource %s not found", resourceID)
	}

	// Remove from indexes
	c.removeFromIndexes(resource)

	// Remove resource
	delete(c.resources, resourceID)
	delete(c.relationships, resourceID)

	return nil
}

// Get retrieves a resource by ID
func (c *Catalog) Get(resourceID string) (*models.CloudResource, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resource := c.resources[resourceID]
	if resource == nil {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}

	return resource, nil
}

// List lists resources with optional filtering
func (c *Catalog) List(req *models.ResourceListRequest) ([]models.CloudResource, int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var resources []*models.CloudResource

	// Apply filters
	if req.Provider != nil {
		resources = c.byProvider[*req.Provider]
	} else if req.ResourceType != nil {
		resources = c.byType[*req.ResourceType]
	} else if req.Region != nil {
		resources = c.byRegion[*req.Region]
	} else if req.AccountID != nil {
		resources = c.byAccount[*req.AccountID]
	} else {
		// Get all resources
		for _, resource := range c.resources {
			resources = append(resources, resource)
		}
	}

	// Apply additional filters
	resources = c.applyFilters(resources, req)

	// Sort resources
	c.sortResources(resources, req.SortBy, req.SortOrder)

	// Apply pagination
	total := len(resources)
	start := req.Offset
	end := start + req.Limit

	if start >= total {
		return []models.CloudResource{}, total, nil
	}

	if end > total {
		end = total
	}

	// Convert to slice of values
	result := make([]models.CloudResource, end-start)
	for i, resource := range resources[start:end] {
		result[i] = *resource
	}

	return result, total, nil
}

// Search searches for resources
func (c *Catalog) Search(req *models.ResourceSearchRequest) ([]models.CloudResource, int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var resources []*models.CloudResource

	// Start with all resources or filtered by provider/type/region
	if req.Provider != nil {
		resources = c.byProvider[*req.Provider]
	} else if req.ResourceType != nil {
		resources = c.byType[*req.ResourceType]
	} else if req.Region != nil {
		resources = c.byRegion[*req.Region]
	} else {
		for _, resource := range c.resources {
			resources = append(resources, resource)
		}
	}

	// Apply search query
	if req.Query != "" {
		resources = c.searchByQuery(resources, req.Query)
	}

	// Apply additional filters
	resources = c.applyFilters(resources, &models.ResourceListRequest{
		AccountID:     req.AccountID,
		ResourceGroup: req.ResourceGroup,
		Tags:          req.Tags,
		Compliance:    req.Compliance,
	})

	// Sort resources
	c.sortResources(resources, "name", "asc")

	// Apply pagination
	total := len(resources)
	start := req.Offset
	end := start + req.Limit

	if start >= total {
		return []models.CloudResource{}, total, nil
	}

	if end > total {
		end = total
	}

	// Convert to slice of values
	result := make([]models.CloudResource, end-start)
	for i, resource := range resources[start:end] {
		result[i] = *resource
	}

	return result, total, nil
}

// GetRelationships returns relationships for a resource
func (c *Catalog) GetRelationships(resourceID string) ([]models.ResourceRelationship, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	relationships := c.relationships[resourceID]
	if relationships == nil {
		return []models.ResourceRelationship{}, nil
	}

	return relationships, nil
}

// AddRelationship adds a relationship between resources
func (c *Catalog) AddRelationship(relationship *models.ResourceRelationship) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.relationships[relationship.SourceID] = append(c.relationships[relationship.SourceID], *relationship)
	return nil
}

// GetByProvider returns resources grouped by provider
func (c *Catalog) GetByProvider() (map[models.CloudProvider][]models.CloudResource, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[models.CloudProvider][]models.CloudResource)
	for provider, resources := range c.byProvider {
		result[provider] = make([]models.CloudResource, len(resources))
		for i, resource := range resources {
			result[provider][i] = *resource
		}
	}

	return result, nil
}

// GetByType returns resources grouped by type
func (c *Catalog) GetByType() (map[string][]models.CloudResource, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]models.CloudResource)
	for resourceType, resources := range c.byType {
		result[resourceType] = make([]models.CloudResource, len(resources))
		for i, resource := range resources {
			result[resourceType][i] = *resource
		}
	}

	return result, nil
}

// GetByRegion returns resources grouped by region
func (c *Catalog) GetByRegion() (map[string][]models.CloudResource, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]models.CloudResource)
	for region, resources := range c.byRegion {
		result[region] = make([]models.CloudResource, len(resources))
		for i, resource := range resources {
			result[region][i] = *resource
		}
	}

	return result, nil
}

// GetByAccount returns resources grouped by account
func (c *Catalog) GetByAccount() (map[string][]models.CloudResource, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]models.CloudResource)
	for account, resources := range c.byAccount {
		result[account] = make([]models.CloudResource, len(resources))
		for i, resource := range resources {
			result[account][i] = *resource
		}
	}

	return result, nil
}

// GetStatistics returns catalog statistics
func (c *Catalog) GetStatistics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"total_resources": len(c.resources),
		"by_provider":     make(map[string]int),
		"by_type":         make(map[string]int),
		"by_region":       make(map[string]int),
		"by_account":      make(map[string]int),
		"relationships":   len(c.relationships),
	}

	// Count by provider
	for provider, resources := range c.byProvider {
		stats["by_provider"].(map[string]int)[provider.String()] = len(resources)
	}

	// Count by type
	for resourceType, resources := range c.byType {
		stats["by_type"].(map[string]int)[resourceType] = len(resources)
	}

	// Count by region
	for region, resources := range c.byRegion {
		stats["by_region"].(map[string]int)[region] = len(resources)
	}

	// Count by account
	for account, resources := range c.byAccount {
		stats["by_account"].(map[string]int)[account] = len(resources)
	}

	return stats
}

// Health checks the health of the catalog
func (c *Catalog) Health() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Basic health check - ensure indexes are consistent
	if len(c.resources) != len(c.byProvider) {
		return fmt.Errorf("catalog index inconsistency detected")
	}

	return nil
}

// Helper methods

// addToIndexes adds a resource to all indexes
func (c *Catalog) addToIndexes(resource *models.CloudResource) {
	// Add to provider index
	c.byProvider[resource.Provider] = append(c.byProvider[resource.Provider], resource)

	// Add to type index
	c.byType[resource.Type] = append(c.byType[resource.Type], resource)

	// Add to region index
	c.byRegion[resource.Region] = append(c.byRegion[resource.Region], resource)

	// Add to account index
	c.byAccount[resource.AccountID] = append(c.byAccount[resource.AccountID], resource)
}

// removeFromIndexes removes a resource from all indexes
func (c *Catalog) removeFromIndexes(resource *models.CloudResource) {
	// Remove from provider index
	c.removeFromSlice(&c.byProvider[resource.Provider], resource)

	// Remove from type index
	c.removeFromSlice(&c.byType[resource.Type], resource)

	// Remove from region index
	c.removeFromSlice(&c.byRegion[resource.Region], resource)

	// Remove from account index
	c.removeFromSlice(&c.byAccount[resource.AccountID], resource)
}

// removeFromSlice removes a resource from a slice
func (c *Catalog) removeFromSlice(slice *[]*models.CloudResource, resource *models.CloudResource) {
	for i, r := range *slice {
		if r.ID == resource.ID {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			break
		}
	}
}

// applyFilters applies filters to resources
func (c *Catalog) applyFilters(resources []*models.CloudResource, req *models.ResourceListRequest) []*models.CloudResource {
	var filtered []*models.CloudResource

	for _, resource := range resources {
		// Apply account filter
		if req.AccountID != nil && resource.AccountID != *req.AccountID {
			continue
		}

		// Apply region filter
		if req.Region != nil && resource.Region != *req.Region {
			continue
		}

		// Apply resource group filter
		if req.ResourceGroup != nil && resource.ResourceGroup != *req.ResourceGroup {
			continue
		}

		// Apply tags filter
		if req.Tags != nil {
			matches := true
			for key, value := range req.Tags {
				if resource.Tags[key] != value {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}

		// Apply compliance filter
		if req.Compliance != nil && resource.Compliance.Status != *req.Compliance {
			continue
		}

		filtered = append(filtered, resource)
	}

	return filtered
}

// searchByQuery searches resources by query string
func (c *Catalog) searchByQuery(resources []*models.CloudResource, query string) []*models.CloudResource {
	var results []*models.CloudResource
	query = strings.ToLower(query)

	for _, resource := range resources {
		// Search in name
		if strings.Contains(strings.ToLower(resource.Name), query) {
			results = append(results, resource)
			continue
		}

		// Search in type
		if strings.Contains(strings.ToLower(resource.Type), query) {
			results = append(results, resource)
			continue
		}

		// Search in tags
		for key, value := range resource.Tags {
			if strings.Contains(strings.ToLower(key), query) || strings.Contains(strings.ToLower(value), query) {
				results = append(results, resource)
				break
			}
		}
	}

	return results
}

// sortResources sorts resources by the specified field and order
func (c *Catalog) sortResources(resources []*models.CloudResource, sortBy, sortOrder string) {
	sort.Slice(resources, func(i, j int) bool {
		var result bool

		switch sortBy {
		case "name":
			result = resources[i].Name < resources[j].Name
		case "last_discovered":
			result = resources[i].LastDiscovered.Before(resources[j].LastDiscovered)
		case "created_at":
			result = resources[i].CreatedAt.Before(resources[j].CreatedAt)
		case "updated_at":
			result = resources[i].UpdatedAt.Before(resources[j].UpdatedAt)
		default:
			result = resources[i].Name < resources[j].Name
		}

		if sortOrder == "desc" {
			result = !result
		}

		return result
	})
}
