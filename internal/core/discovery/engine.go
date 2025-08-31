package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/providers/cloud"
)

// Discoverer interface for resource discovery
type Discoverer interface {
	Discover(ctx context.Context) ([]cloud.Resource, error)
}

// Engine provides discovery engine functionality
type Engine struct {
	service     *Service
	discoverers map[string]Discoverer
	cache       *Cache
	mu          sync.RWMutex
}

// NewEngine creates a new discovery engine
func NewEngine() *Engine {
	return &Engine{
		service:     NewService(),
		discoverers: make(map[string]Discoverer),
		cache:       NewCache(5 * time.Minute),
	}
}

// DiscoverResources discovers resources for a provider
func (e *Engine) DiscoverResources(ctx context.Context, provider string, options *DiscoveryOptions) ([]cloud.Resource, error) {
	e.mu.RLock()
	discoverer, exists := e.discoverers[provider]
	e.mu.RUnlock()
	
	if !exists {
		// Return empty list if provider not found
		return []cloud.Resource{}, nil
	}
	
	return discoverer.Discover(ctx)
}

// DiscoverResourcesInRegion discovers resources for a provider in a specific region
func (e *Engine) DiscoverResourcesInRegion(ctx context.Context, provider string, region string, resourceTypes []string) ([]cloud.Resource, error) {
	e.mu.RLock()
	discoverer, exists := e.discoverers[provider]
	e.mu.RUnlock()
	
	if !exists {
		// Return empty list if provider not found
		return []cloud.Resource{}, nil
	}
	
	// Check if discoverer supports regional discovery
	if regionalDiscoverer, ok := discoverer.(interface {
		DiscoverInRegion(ctx context.Context, region string, resourceTypes []string) ([]cloud.Resource, error)
	}); ok {
		return regionalDiscoverer.DiscoverInRegion(ctx, region, resourceTypes)
	}
	
	// Fallback to general discovery and filter by region
	resources, err := discoverer.Discover(ctx)
	if err != nil {
		return nil, err
	}
	
	// Filter by region if provided
	if region != "" {
		var filtered []cloud.Resource
		for _, r := range resources {
			if r.Region == region {
				filtered = append(filtered, r)
			}
		}
		resources = filtered
	}
	
	// Filter by resource types if provided
	if len(resourceTypes) > 0 {
		typeMap := make(map[string]bool)
		for _, t := range resourceTypes {
			typeMap[t] = true
		}
		
		var filtered []cloud.Resource
		for _, r := range resources {
			if typeMap[r.Type] {
				filtered = append(filtered, r)
			}
		}
		resources = filtered
	}
	
	return resources, nil
}

// RegisterProvider registers a cloud provider discoverer
func (e *Engine) RegisterProvider(name string, discoverer Discoverer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.discoverers[name] = discoverer
}

// GetProvider returns a registered provider discoverer
func (e *Engine) GetProvider(name string) (Discoverer, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	d, exists := e.discoverers[name]
	return d, exists
}