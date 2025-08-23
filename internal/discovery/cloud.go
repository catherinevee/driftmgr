package discovery

import (
	"context"
	"fmt"
	"sync"

	"github.com/catherinevee/driftmgr/internal/models"
)

// CloudDiscoverer provides cloud-agnostic discovery capabilities
type CloudDiscoverer struct {
	providers map[string]Provider
}

// NewCloudDiscoverer creates a new cloud discoverer
func NewCloudDiscoverer() *CloudDiscoverer {
	return &CloudDiscoverer{
		providers: make(map[string]Provider),
	}
}

// AddProvider adds a provider to the discoverer
func (cd *CloudDiscoverer) AddProvider(name string, provider Provider) {
	cd.providers[name] = provider
}

// DiscoverAll discovers resources from all providers
func (cd *CloudDiscoverer) DiscoverAll(ctx context.Context) (map[string][]models.Resource, error) {
	results := make(map[string][]models.Resource)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, provider := range cd.providers {
		wg.Add(1)
		go func(providerName string, p Provider) {
			defer wg.Done()

			resources, err := p.Discover(Config{})
			if err != nil {
				fmt.Printf("Error discovering from %s: %v\n", providerName, err)
				return
			}

			mu.Lock()
			results[providerName] = resources
			mu.Unlock()
		}(name, provider)
	}

	wg.Wait()
	return results, nil
}

// DiscoverProvider discovers resources from a specific provider
func (cd *CloudDiscoverer) DiscoverProvider(ctx context.Context, providerName string, config Config) ([]models.Resource, error) {
	provider, exists := cd.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	return provider.Discover(config)
}
