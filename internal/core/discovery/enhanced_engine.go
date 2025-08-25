package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/credentials"
	"github.com/catherinevee/driftmgr/internal/utils/cache"
	"github.com/catherinevee/driftmgr/internal/utils/retry"
)

// EnhancedEngine provides discovery with caching and retry logic
type EnhancedEngine struct {
	providers  map[string]Provider
	cache      *cache.DiscoveryCache
	credDetect *credentials.CredentialDetector
	retryConf  *retry.Config
}

// NewEnhancedEngine creates an enhanced discovery engine
func NewEnhancedEngine() (*EnhancedEngine, error) {
	engine := &EnhancedEngine{
		providers:  make(map[string]Provider),
		cache:      cache.NewDiscoveryCache(),
		credDetect: credentials.NewCredentialDetector(),
		retryConf:  retry.CloudAPIConfig(),
	}

	// Initialize providers based on detected credentials
	if err := engine.initializeProviders(); err != nil {
		return nil, err
	}

	// Start cache cleanup task
	engine.cache.StartCleanupTask(5 * time.Minute)

	return engine, nil
}

// initializeProviders initializes providers based on available credentials
func (e *EnhancedEngine) initializeProviders() error {
	ctx := context.Background()

	// Check AWS
	if e.credDetect.IsConfigured("aws") {
		err := retry.Do(ctx, e.retryConf, func() error {
			provider, err := NewAWSProvider()
			if err == nil {
				e.providers["aws"] = provider
			}
			return err
		})
		if err != nil {
			fmt.Printf("[Warning] Failed to initialize AWS provider: %v\n", err)
		}
	}

	// Check Azure
	if e.credDetect.IsConfigured("azure") {
		err := retry.Do(ctx, e.retryConf, func() error {
			provider, err := NewAzureProvider()
			if err == nil {
				e.providers["azure"] = provider
			}
			return err
		})
		if err != nil {
			fmt.Printf("[Warning] Failed to initialize Azure provider: %v\n", err)
		}
	}

	// Check GCP
	if e.credDetect.IsConfigured("gcp") {
		err := retry.Do(ctx, e.retryConf, func() error {
			provider, err := NewGCPProvider()
			if err == nil {
				e.providers["gcp"] = provider
			}
			return err
		})
		if err != nil {
			fmt.Printf("[Warning] Failed to initialize GCP provider: %v\n", err)
		}
	}

	// Check DigitalOcean
	if e.credDetect.IsConfigured("digitalocean") {
		err := retry.Do(ctx, e.retryConf, func() error {
			provider, err := NewDigitalOceanProvider()
			if err == nil {
				e.providers["digitalocean"] = provider
			}
			return err
		})
		if err != nil {
			fmt.Printf("[Warning] Failed to initialize DigitalOcean provider: %v\n", err)
		}
	}

	if len(e.providers) == 0 {
		return fmt.Errorf("no cloud providers could be initialized")
	}

	return nil
}

// Discover performs discovery with caching and retry
func (e *EnhancedEngine) Discover(config Config) ([]models.Resource, error) {
	// Generate cache key
	cacheKey := cache.GetDiscoveryKey(config.Provider,
		fmt.Sprintf("%v", config.Regions), config.ResourceType)

	// Try to get from cache first
	if cachedData, ok := e.cache.Get(cacheKey); ok {
		if resources, ok := cachedData.([]models.Resource); ok {
			fmt.Printf("  [Cache] Using cached results for %s\n", config.Provider)
			return resources, nil
		}
	}

	// Get provider
	provider, exists := e.providers[config.Provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	fmt.Printf("🔍 Discovering resources with %s provider...\n", provider.Name())

	// Convert config to options
	options := DiscoveryOptions{
		Regions: config.Regions,
	}
	if config.ResourceType != "" {
		options.ResourceTypes = []string{config.ResourceType}
	}

	// Perform discovery with retry
	ctx := context.Background()
	resources, err := retry.DoWithResult(ctx, e.retryConf, func() ([]models.Resource, error) {
		result, err := provider.Discover(ctx, options)
		if err != nil {
			return nil, err
		}
		return result.Resources, nil
	})

	if err != nil {
		return nil, fmt.Errorf("discovery failed after retries: %w", err)
	}

	// Apply post-discovery filtering
	filtered := e.applyFilters(resources, config)

	// Cache the results
	e.cache.SetWithTTL(cacheKey, filtered, 15*time.Minute)

	return filtered, nil
}

// DiscoverAllProviders discovers resources across all configured providers
func (e *EnhancedEngine) DiscoverAllProviders(config Config) ([]models.Resource, error) {
	var allResources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Discover from each provider in parallel
	for providerName := range e.providers {
		wg.Add(1)
		go func(pName string) {
			defer wg.Done()

			providerConfig := config
			providerConfig.Provider = pName

			resources, err := e.Discover(providerConfig)
			if err != nil {
				fmt.Printf("Warning: Failed to discover %s resources: %v\n", pName, err)
				return
			}

			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(providerName)
	}

	wg.Wait()
	return allResources, nil
}

// DiscoverWithProgress performs discovery with progress reporting
func (e *EnhancedEngine) DiscoverWithProgress(config Config, progress chan<- string) ([]models.Resource, error) {
	defer close(progress)

	// Report initialization
	progress <- fmt.Sprintf("Initializing %s provider...", config.Provider)

	// Check cache
	cacheKey := cache.GetDiscoveryKey(config.Provider,
		fmt.Sprintf("%v", config.Regions), config.ResourceType)

	if cachedData, ok := e.cache.Get(cacheKey); ok {
		if resources, ok := cachedData.([]models.Resource); ok {
			progress <- fmt.Sprintf("Found %d cached resources", len(resources))
			return resources, nil
		}
	}

	// Get provider
	provider, exists := e.providers[config.Provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	progress <- fmt.Sprintf("Discovering %s resources...", provider.Name())

	// Perform discovery with retry
	attempts := 0
	ctx := context.Background()

	// Convert config to options
	options := DiscoveryOptions{
		Regions: config.Regions,
	}
	if config.ResourceType != "" {
		options.ResourceTypes = []string{config.ResourceType}
	}

	resources, err := retry.DoWithResult(ctx, e.retryConf, func() ([]models.Resource, error) {
		attempts++
		if attempts > 1 {
			progress <- fmt.Sprintf("Retry attempt %d...", attempts)
		}
		result, err := provider.Discover(ctx, options)
		if err != nil {
			return nil, err
		}
		return result.Resources, nil
	})

	if err != nil {
		return nil, err
	}

	progress <- fmt.Sprintf("Found %d resources", len(resources))

	// Apply filters
	filtered := e.applyFilters(resources, config)
	if len(filtered) != len(resources) {
		progress <- fmt.Sprintf("Filtered to %d resources", len(filtered))
	}

	// Cache results
	e.cache.SetWithTTL(cacheKey, filtered, 15*time.Minute)
	progress <- "Results cached"

	return filtered, nil
}

// applyFilters applies filtering to discovered resources
func (e *EnhancedEngine) applyFilters(resources []models.Resource, config Config) []models.Resource {
	var filtered []models.Resource

	for _, resource := range resources {
		// Filter by resource type if specified
		if config.ResourceType != "" && resource.Type != config.ResourceType {
			continue
		}

		// Filter by tags if specified
		if len(config.Tags) > 0 {
			// Convert map to slice of key:value strings
			var tagFilters []string
			for k, v := range config.Tags {
				tagFilters = append(tagFilters, fmt.Sprintf("%s:%s", k, v))
			}
			if !e.matchesTags(resource, tagFilters) {
				continue
			}
		}

		filtered = append(filtered, resource)
	}

	return filtered
}

// matchesTags checks if a resource matches tag filters
func (e *EnhancedEngine) matchesTags(resource models.Resource, tagFilters []string) bool {
	for _, filter := range tagFilters {
		parts := strings.Split(filter, ":")
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		tags := resource.GetTagsAsMap()
		resourceValue, exists := tags[key]
		if !exists || resourceValue != value {
			return false
		}
	}
	return true
}

// GetProviders returns list of initialized providers
func (e *EnhancedEngine) GetProviders() []string {
	var providers []string
	for name := range e.providers {
		providers = append(providers, name)
	}
	return providers
}

// ClearCache clears the discovery cache
func (e *EnhancedEngine) ClearCache() {
	e.cache.Clear()
}

// GetCacheStats returns cache statistics
func (e *EnhancedEngine) GetCacheStats() map[string]interface{} {
	// This would be enhanced with actual stats
	return map[string]interface{}{
		"providers": len(e.providers),
		"cached":    "active",
	}
}
