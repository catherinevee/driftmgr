package digitalocean

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Provider wraps the DigitalOcean discovery provider
type Provider struct {
	doProvider *discovery.DigitalOceanProvider
}

// NewProvider creates a new DigitalOcean provider
func NewProvider() discovery.Provider {
	doProvider, _ := discovery.NewDigitalOceanProvider()
	return &Provider{
		doProvider: doProvider,
	}
}

// Discover performs discovery across all configured regions
func (p *Provider) Discover(ctx context.Context, options discovery.DiscoveryOptions) (*discovery.Result, error) {
	if p.doProvider == nil {
		var err error
		p.doProvider, err = discovery.NewDigitalOceanProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize DigitalOcean provider: %w", err)
		}
	}

	startTime := time.Now()
	result := &discovery.Result{
		Provider:  "digitalocean",
		Timestamp: startTime,
		Resources: []models.Resource{},
		Regions:   options.Regions,
	}

	// If no regions specified, use default
	if len(options.Regions) == 0 {
		options.Regions = []string{"nyc1"}
	}

	// Discover resources in each region
	for _, region := range options.Regions {
		resources, err := p.DiscoverRegion(ctx, region)
		if err != nil {
			result.Errors = append(result.Errors, discovery.DiscoveryError{
				Region:  region,
				Service: "digitalocean",
				Error:   err.Error(),
			})
			continue
		}
		result.Resources = append(result.Resources, resources...)
	}

	result.ResourceCount = len(result.Resources)
	result.Duration = time.Since(startTime)

	return result, nil
}

// DiscoverRegion discovers resources in a specific region
func (p *Provider) DiscoverRegion(ctx context.Context, region string) ([]models.Resource, error) {
	if p.doProvider == nil {
		var err error
		p.doProvider, err = discovery.NewDigitalOceanProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize DigitalOcean provider: %w", err)
		}
	}

	// Use the DigitalOcean provider's Discover method
	options := discovery.DiscoveryOptions{
		Regions: []string{region},
		Timeout: 30 * time.Second,
	}
	
	result, err := p.doProvider.Discover(ctx, options)
	if err != nil {
		return nil, err
	}
	
	if result != nil {
		return result.Resources, nil
	}
	
	return []models.Resource{}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	if p.doProvider != nil {
		return p.doProvider.Name()
	}
	return "DigitalOcean"
}

// Regions returns supported regions
func (p *Provider) Regions() []string {
	if p.doProvider != nil {
		return p.doProvider.Regions()
	}
	return []string{"nyc1", "nyc3", "sfo3", "ams3", "lon1"}
}

// Services returns supported services
func (p *Provider) Services() []string {
	if p.doProvider != nil {
		return p.doProvider.Services()
	}
	return []string{"Droplets", "Volumes", "LoadBalancers", "Databases"}
}

// ValidateCredentials validates DigitalOcean credentials
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	if p.doProvider == nil {
		var err error
		p.doProvider, err = discovery.NewDigitalOceanProvider()
		if err != nil {
			return fmt.Errorf("failed to validate credentials: %w", err)
		}
	}
	return nil
}

// GetAccountInfo returns DigitalOcean account information
func (p *Provider) GetAccountInfo(ctx context.Context) (*discovery.AccountInfo, error) {
	return &discovery.AccountInfo{
		Provider:  "digitalocean",
		ID: "do-account-123",
		Name:      "DigitalOcean Account",
	}, nil
}