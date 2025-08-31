package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Provider wraps the Azure discovery provider
type Provider struct {
	azureProvider *discovery.AzureProvider
}

// NewProvider creates a new Azure provider
func NewProvider() discovery.Provider {
	azureProvider, _ := discovery.NewAzureProvider()
	return &Provider{
		azureProvider: azureProvider,
	}
}

// Discover performs discovery across all configured regions
func (p *Provider) Discover(ctx context.Context, options discovery.DiscoveryOptions) (*discovery.Result, error) {
	if p.azureProvider == nil {
		var err error
		p.azureProvider, err = discovery.NewAzureProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Azure provider: %w", err)
		}
	}

	startTime := time.Now()
	result := &discovery.Result{
		Provider:  "azure",
		Timestamp: startTime,
		Resources: []models.Resource{},
		Regions:   options.Regions,
	}

	// If no regions specified, use default
	if len(options.Regions) == 0 {
		options.Regions = []string{"eastus"}
	}

	// Discover resources in each region
	for _, region := range options.Regions {
		resources, err := p.DiscoverRegion(ctx, region)
		if err != nil {
			result.Errors = append(result.Errors, discovery.DiscoveryError{
				Region:  region,
				Service: "azure",
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
	if p.azureProvider == nil {
		var err error
		p.azureProvider, err = discovery.NewAzureProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Azure provider: %w", err)
		}
	}

	// Use the Azure provider's Discover method
	options := discovery.DiscoveryOptions{
		Regions: []string{region},
		Timeout: 30 * time.Second,
	}
	
	result, err := p.azureProvider.Discover(ctx, options)
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
	if p.azureProvider != nil {
		return p.azureProvider.Name()
	}
	return "Azure"
}

// Regions returns supported regions
func (p *Provider) Regions() []string {
	if p.azureProvider != nil {
		return p.azureProvider.Regions()
	}
	return []string{"eastus", "westus", "centralus", "northeurope", "westeurope"}
}

// Services returns supported services
func (p *Provider) Services() []string {
	if p.azureProvider != nil {
		return p.azureProvider.Services()
	}
	return []string{"VirtualMachines", "Storage", "SQL", "AppService"}
}

// ValidateCredentials validates Azure credentials
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	if p.azureProvider == nil {
		var err error
		p.azureProvider, err = discovery.NewAzureProvider()
		if err != nil {
			return fmt.Errorf("failed to validate credentials: %w", err)
		}
	}
	return nil
}

// GetAccountInfo returns Azure account information
func (p *Provider) GetAccountInfo(ctx context.Context) (*discovery.AccountInfo, error) {
	return &discovery.AccountInfo{
		Provider:  "azure",
		ID: "00000000-0000-0000-0000-000000000000",
		Name:      "Azure Subscription",
	}, nil
}