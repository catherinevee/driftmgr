package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// ResourceDiscoverer represents a resource discoverer
type ResourceDiscoverer struct {
	providers map[string]Provider
	config    *DiscoveryConfig
}

// DiscoveryConfig represents configuration for resource discovery
type DiscoveryConfig struct {
	Timeout       time.Duration          `json:"timeout"`
	MaxRetries    int                    `json:"max_retries"`
	Regions       []string               `json:"regions"`
	ResourceTypes []string               `json:"resource_types"`
	Filters       map[string]interface{} `json:"filters"`
}

// Provider represents a cloud provider interface
type Provider interface {
	Name() string
	DiscoverResources(ctx context.Context, region string) ([]models.Resource, error)
}

// NewResourceDiscoverer creates a new resource discoverer
func NewResourceDiscoverer(config *DiscoveryConfig) *ResourceDiscoverer {
	if config == nil {
		config = &DiscoveryConfig{
			Timeout:       30 * time.Second,
			MaxRetries:    3,
			Regions:       []string{"us-east-1", "us-west-2"},
			ResourceTypes: []string{"aws_instance", "aws_security_group"},
		}
	}

	return &ResourceDiscoverer{
		providers: make(map[string]Provider),
		config:    config,
	}
}

// DiscoverResources discovers resources across all configured providers and regions
func (rd *ResourceDiscoverer) DiscoverResources(ctx context.Context) ([]models.Resource, error) {
	var allResources []models.Resource

	for _, provider := range rd.providers {
		for _, region := range rd.config.Regions {
			resources, err := provider.DiscoverResources(ctx, region)
			if err != nil {
				return nil, fmt.Errorf("failed to discover resources in %s for %s: %w", region, provider.Name(), err)
			}
			allResources = append(allResources, resources...)
		}
	}

	return allResources, nil
}

// discoverResourcesForRegion discovers resources for a specific region
func (rd *ResourceDiscoverer) discoverResourcesForRegion(ctx context.Context, provider, region string) ([]models.Resource, error) {
	providerImpl, exists := rd.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	return providerImpl.DiscoverResources(ctx, region)
}

// DiscoverResourcesByType discovers resources of a specific type
func (rd *ResourceDiscoverer) DiscoverResourcesByType(ctx context.Context, resourceType string) ([]models.Resource, error) {
	allResources, err := rd.DiscoverResources(ctx)
	if err != nil {
		return nil, err
	}

	var filteredResources []models.Resource
	for _, resource := range allResources {
		if resource.Type == resourceType {
			filteredResources = append(filteredResources, resource)
		}
	}

	return filteredResources, nil
}

// GetResourceCount returns the total count of discovered resources
func (rd *ResourceDiscoverer) GetResourceCount(ctx context.Context) (int, error) {
	resources, err := rd.DiscoverResources(ctx)
	if err != nil {
		return 0, err
	}

	return len(resources), nil
}
