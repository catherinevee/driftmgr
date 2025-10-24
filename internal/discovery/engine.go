package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/providers/aws"
	"github.com/catherinevee/driftmgr/internal/providers/azure"
	"github.com/catherinevee/driftmgr/internal/providers/gcp"
	"github.com/catherinevee/driftmgr/internal/providers/digitalocean"
)

// Engine represents the main discovery engine
type Engine struct {
	providers map[models.CloudProvider]ProviderDiscovery
	cache     *Cache
	scheduler *Scheduler
	mu        sync.RWMutex
}

// ProviderDiscovery represents a provider-specific discovery interface
type ProviderDiscovery interface {
	DiscoverResources(ctx context.Context, job *models.DiscoveryJob) (*models.DiscoveryResults, error)
	GetResourceCount(ctx context.Context, resourceType string) (int, error)
	GetResourceTypes() []string
	ValidateConfiguration(ctx context.Context) error
	GetDiscoveryCapabilities() map[string]interface{}
}

// NewEngine creates a new discovery engine
func NewEngine() *Engine {
	engine := &Engine{
		providers: make(map[models.CloudProvider]ProviderDiscovery),
		cache:     NewCache(),
		scheduler: NewScheduler(),
	}

	// Initialize providers
	engine.initializeProviders()

	return engine
}

// initializeProviders initializes all available providers
func (e *Engine) initializeProviders() {
	// AWS provider will be initialized when credentials are provided
	// For now, we'll register the provider type
	e.providers[models.ProviderAWS] = nil
	e.providers[models.ProviderAzure] = nil
	e.providers[models.ProviderGCP] = nil
	e.providers[models.ProviderDigitalOcean] = nil
}

// RegisterProvider registers a provider discovery implementation
func (e *Engine) RegisterProvider(provider models.CloudProvider, discovery ProviderDiscovery) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.providers[provider] = discovery
}

// DiscoverResources discovers resources across all providers
func (e *Engine) DiscoverResources(ctx context.Context, job *models.DiscoveryJob) (*models.DiscoveryResults, error) {
	e.mu.RLock()
	providerDiscovery, exists := e.providers[job.Provider]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not supported", job.Provider)
	}

	if providerDiscovery == nil {
		return nil, fmt.Errorf("provider %s not initialized", job.Provider)
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s:%s", job.Provider, job.AccountID, job.Region)
	if cached, found := e.cache.Get(cacheKey); found {
		// Return cached results if still valid
		if time.Since(cached.Timestamp) < time.Duration(30)*time.Minute {
			return cached.Results, nil
		}
	}

	// Perform discovery
	results, err := providerDiscovery.DiscoverResources(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("discovery failed for provider %s: %w", job.Provider, err)
	}

	// Cache results
	e.cache.Set(cacheKey, &CacheEntry{
		Results:   results,
		Timestamp: time.Now(),
	})

	return results, nil
}

// DiscoverResourcesParallel discovers resources across multiple providers in parallel
func (e *Engine) DiscoverResourcesParallel(ctx context.Context, jobs []*models.DiscoveryJob) (map[string]*models.DiscoveryResults, error) {
	results := make(map[string]*models.DiscoveryResults)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, job := range jobs {
		wg.Add(1)
		go func(job *models.DiscoveryJob) {
			defer wg.Done()

			jobResults, err := e.DiscoverResources(ctx, job)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors = append(errors, fmt.Errorf("job %s failed: %w", job.ID, err))
			} else {
				results[job.ID] = jobResults
			}
		}(job)
	}

	wg.Wait()

	if len(errors) > 0 {
		return results, fmt.Errorf("some discovery jobs failed: %v", errors)
	}

	return results, nil
}

// GetResourceCount returns the count of resources for a specific provider and type
func (e *Engine) GetResourceCount(ctx context.Context, provider models.CloudProvider, resourceType string) (int, error) {
	e.mu.RLock()
	providerDiscovery, exists := e.providers[provider]
	e.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("provider %s not supported", provider)
	}

	if providerDiscovery == nil {
		return 0, fmt.Errorf("provider %s not initialized", provider)
	}

	return providerDiscovery.GetResourceCount(ctx, resourceType)
}

// GetSupportedProviders returns the list of supported providers
func (e *Engine) GetSupportedProviders() []models.CloudProvider {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var providers []models.CloudProvider
	for provider := range e.providers {
		providers = append(providers, provider)
	}
	return providers
}

// GetProviderCapabilities returns the capabilities of a specific provider
func (e *Engine) GetProviderCapabilities(ctx context.Context, provider models.CloudProvider) (map[string]interface{}, error) {
	e.mu.RLock()
	providerDiscovery, exists := e.providers[provider]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not supported", provider)
	}

	if providerDiscovery == nil {
		return nil, fmt.Errorf("provider %s not initialized", provider)
	}

	return providerDiscovery.GetDiscoveryCapabilities(), nil
}

// ValidateProviderConfiguration validates the configuration of a specific provider
func (e *Engine) ValidateProviderConfiguration(ctx context.Context, provider models.CloudProvider) error {
	e.mu.RLock()
	providerDiscovery, exists := e.providers[provider]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("provider %s not supported", provider)
	}

	if providerDiscovery == nil {
		return fmt.Errorf("provider %s not initialized", provider)
	}

	return providerDiscovery.ValidateConfiguration(ctx)
}

// InitializeAWSProvider initializes the AWS provider with credentials
func (e *Engine) InitializeAWSProvider(ctx context.Context, credentials models.ProviderCredentials, region string, settings models.ProviderSettings) error {
	// Create AWS client
	client, err := aws.NewClient(region, &credentials)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create AWS discovery engine
	discovery := aws.NewDiscoveryEngine(client)

	// Register the provider
	e.RegisterProvider(models.ProviderAWS, discovery)

	return nil
}

// InitializeAzureProvider initializes the Azure provider with credentials
func (e *Engine) InitializeAzureProvider(ctx context.Context, credentials models.ProviderCredentials, region string, settings models.ProviderSettings) error {
	// Create Azure SDK provider
	// Extract Azure-specific credentials from custom field
	subscription := ""
	tenantID := ""
	if credentials.Custom != nil {
		if sub, ok := credentials.Custom["subscription"].(string); ok {
			subscription = sub
		}
		if tenant, ok := credentials.Custom["tenant_id"].(string); ok {
			tenantID = tenant
		}
	}
	client, err := azure.NewAzureSDKProviderSimple(subscription, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create Azure client: %w", err)
	}

	// Create Azure discovery engine
	discovery := azure.NewDiscoveryEngine(client)

	// Register the provider
	e.RegisterProvider(models.ProviderAzure, discovery)

	return nil
}

// InitializeGCPProvider initializes the GCP provider with credentials
func (e *Engine) InitializeGCPProvider(ctx context.Context, credentials models.ProviderCredentials, region string, settings models.ProviderSettings) error {
	// Create GCP SDK provider
	// Extract GCP-specific credentials from custom field
	projectID := ""
	if credentials.Custom != nil {
		if proj, ok := credentials.Custom["project_id"].(string); ok {
			projectID = proj
		}
	}
	client, err := gcp.NewGCPSDKProvider(projectID, region)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	// Create GCP discovery engine
	discovery := gcp.NewDiscoveryEngine(client)

	// Register the provider
	e.RegisterProvider(models.ProviderGCP, discovery)

	return nil
}

// InitializeDigitalOceanProvider initializes the DigitalOcean provider with credentials
func (e *Engine) InitializeDigitalOceanProvider(ctx context.Context, credentials models.ProviderCredentials, region string, settings models.ProviderSettings) error {
	// Create DigitalOcean SDK provider
	// Extract DigitalOcean-specific credentials from custom field
	apiKey := ""
	if credentials.Custom != nil {
		if key, ok := credentials.Custom["api_key"].(string); ok {
			apiKey = key
		}
	}
	client, err := digitalocean.NewDigitalOceanSDKProvider(apiKey)
	if err != nil {
		return fmt.Errorf("failed to create DigitalOcean client: %w", err)
	}

	// Create DigitalOcean discovery engine
	discovery := digitalocean.NewDiscoveryEngine(client)

	// Register the provider
	e.RegisterProvider(models.ProviderDigitalOcean, discovery)

	return nil
}

// GetDiscoveryStatistics returns statistics about discovery operations
func (e *Engine) GetDiscoveryStatistics() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]interface{}{
		"total_providers":       len(e.providers),
		"initialized_providers": 0,
		"cache_entries":         e.cache.Size(),
		"cache_hit_rate":        e.cache.HitRate(),
		"supported_providers":   e.GetSupportedProviders(),
	}

	// Count initialized providers
	for _, provider := range e.providers {
		if provider != nil {
			stats["initialized_providers"] = stats["initialized_providers"].(int) + 1
		}
	}

	return stats
}

// ClearCache clears the discovery cache
func (e *Engine) ClearCache() {
	e.cache.Clear()
}

// GetCacheStatistics returns cache statistics
func (e *Engine) GetCacheStatistics() map[string]interface{} {
	return map[string]interface{}{
		"size":      e.cache.Size(),
		"hit_rate":  e.cache.HitRate(),
		"miss_rate": e.cache.MissRate(),
	}
}

// ScheduleDiscovery schedules a discovery job
func (e *Engine) ScheduleDiscovery(job *models.DiscoveryJob) error {
	return e.scheduler.Schedule(job)
}

// CancelScheduledDiscovery cancels a scheduled discovery job
func (e *Engine) CancelScheduledDiscovery(jobID string) error {
	return e.scheduler.Cancel(jobID)
}

// GetScheduledJobs returns all scheduled discovery jobs
func (e *Engine) GetScheduledJobs() []*models.DiscoveryJob {
	return e.scheduler.GetJobs()
}

// StartScheduler starts the discovery scheduler
func (e *Engine) StartScheduler(ctx context.Context) {
	e.scheduler.Start(ctx)
}

// StopScheduler stops the discovery scheduler
func (e *Engine) StopScheduler() {
	e.scheduler.Stop()
}
