package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Warmer pre-fetches commonly accessed resources
type Warmer struct {
	cache           *GlobalCache
	discoveryFunc   func(ctx context.Context, provider string, region string) ([]models.Resource, error)
	warmupSchedule  map[string]time.Duration
	providers       []string
	regions         []string
	running         bool
	stopChan        chan struct{}
	mu              sync.RWMutex
}

// NewWarmer creates a new cache warmer
func NewWarmer(cache *GlobalCache, discoveryFunc func(context.Context, string, string) ([]models.Resource, error)) *Warmer {
	return &Warmer{
		cache:         cache,
		discoveryFunc: discoveryFunc,
		warmupSchedule: map[string]time.Duration{
			"aws":          5 * time.Minute,
			"azure":        10 * time.Minute,
			"gcp":          10 * time.Minute,
			"digitalocean": 15 * time.Minute,
		},
		providers: []string{"aws", "azure", "gcp", "digitalocean"},
		regions: []string{
			"us-east-1", "us-west-2", // AWS
			"eastus", "westus2",       // Azure
			"us-central1", "us-east1", // GCP
			"nyc1", "sfo2",            // DigitalOcean
		},
		stopChan: make(chan struct{}),
	}
}

// Start begins the cache warming process
func (w *Warmer) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("cache warmer already running")
	}
	w.running = true
	w.mu.Unlock()

	// Start warmup goroutines for each provider
	for _, provider := range w.providers {
		go w.warmProvider(provider)
	}

	// Start priority resource warming
	go w.warmPriorityResources()

	return nil
}

// Stop stops the cache warming process
func (w *Warmer) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	close(w.stopChan)
	w.mu.Unlock()
}

// warmProvider continuously warms cache for a specific provider
func (w *Warmer) warmProvider(provider string) {
	interval := w.warmupSchedule[provider]
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial warm
	w.warmProviderOnce(provider)

	for {
		select {
		case <-ticker.C:
			w.warmProviderOnce(provider)
		case <-w.stopChan:
			return
		}
	}
}

// warmProviderOnce performs a single cache warm for a provider
func (w *Warmer) warmProviderOnce(provider string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Determine regions based on provider
	regions := w.getProviderRegions(provider)

	for _, region := range regions {
		// Check if cache already has recent data
		cacheKey := fmt.Sprintf("discovery:%s:%s", provider, region)
		if _, found, age := w.cache.GetWithAge(cacheKey); found && age < 3*time.Minute {
			// Skip if cache is fresh enough
			continue
		}

		// Fetch resources
		if w.discoveryFunc != nil {
			resources, err := w.discoveryFunc(ctx, provider, region)
			if err == nil && len(resources) > 0 {
				// Cache the results
				w.cache.Set(cacheKey, resources, 5*time.Minute)
			}
		}
	}
}

// warmPriorityResources warms cache for high-priority resources
func (w *Warmer) warmPriorityResources() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	priorityResources := []struct {
		provider string
		region   string
		types    []string
	}{
		{"aws", "us-east-1", []string{"ec2_instance", "rds_instance", "s3_bucket"}},
		{"aws", "us-west-2", []string{"ec2_instance", "eks_cluster"}},
		{"azure", "eastus", []string{"virtual_machine", "storage_account"}},
		{"gcp", "us-central1", []string{"compute_instance", "gke_cluster"}},
	}

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			for _, pr := range priorityResources {
				cacheKey := fmt.Sprintf("discovery:priority:%s:%s", pr.provider, pr.region)
				
				// Check if already cached
				if _, found, age := w.cache.GetWithAge(cacheKey); found && age < 1*time.Minute {
					continue
				}

				// Warm this priority resource
				if w.discoveryFunc != nil {
					resources, err := w.discoveryFunc(ctx, pr.provider, pr.region)
					if err == nil {
						// Filter by resource types if needed
						filtered := w.filterByTypes(resources, pr.types)
						if len(filtered) > 0 {
							w.cache.Set(cacheKey, filtered, 3*time.Minute)
						}
					}
				}
			}
		case <-w.stopChan:
			return
		}
	}
}

// getProviderRegions returns regions for a provider
func (w *Warmer) getProviderRegions(provider string) []string {
	switch provider {
	case "aws":
		return []string{"us-east-1", "us-west-2", "eu-west-1"}
	case "azure":
		return []string{"eastus", "westus2", "northeurope"}
	case "gcp":
		return []string{"us-central1", "us-east1", "europe-west1"}
	case "digitalocean":
		return []string{"nyc1", "sfo2", "lon1"}
	default:
		return []string{}
	}
}

// filterByTypes filters resources by type
func (w *Warmer) filterByTypes(resources []models.Resource, types []string) []models.Resource {
	if len(types) == 0 {
		return resources
	}

	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	var filtered []models.Resource
	for _, r := range resources {
		if typeMap[r.Type] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// WarmSpecific warms cache for specific provider/region combination
func (w *Warmer) WarmSpecific(provider, region string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if w.discoveryFunc == nil {
		return fmt.Errorf("discovery function not set")
	}

	resources, err := w.discoveryFunc(ctx, provider, region)
	if err != nil {
		return fmt.Errorf("failed to warm cache: %w", err)
	}

	cacheKey := fmt.Sprintf("discovery:%s:%s", provider, region)
	w.cache.Set(cacheKey, resources, 5*time.Minute)

	return nil
}

// GetWarmupStatus returns the status of cache warming
func (w *Warmer) GetWarmupStatus() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	status := map[string]interface{}{
		"running":   w.running,
		"providers": w.providers,
		"schedule":  w.warmupSchedule,
	}

	// Check cache freshness for each provider
	freshness := make(map[string]string)
	for _, provider := range w.providers {
		for _, region := range w.getProviderRegions(provider) {
			key := fmt.Sprintf("%s-%s", provider, region)
			cacheKey := fmt.Sprintf("discovery:%s:%s", provider, region)
			
			if _, found, age := w.cache.GetWithAge(cacheKey); found {
				if age < 1*time.Minute {
					freshness[key] = "hot"
				} else if age < 5*time.Minute {
					freshness[key] = "warm"
				} else {
					freshness[key] = "cold"
				}
			} else {
				freshness[key] = "empty"
			}
		}
	}
	status["freshness"] = freshness

	return status
}