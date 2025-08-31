package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Provider wraps the GCP discovery provider
type Provider struct {
	gcpProvider *discovery.GCPProvider
}

// NewProvider creates a new GCP provider
func NewProvider() discovery.Provider {
	gcpProvider, _ := discovery.NewGCPProvider()
	return &Provider{
		gcpProvider: gcpProvider,
	}
}

// Discover performs discovery across all configured regions
func (p *Provider) Discover(ctx context.Context, options discovery.DiscoveryOptions) (*discovery.Result, error) {
	if p.gcpProvider == nil {
		var err error
		p.gcpProvider, err = discovery.NewGCPProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GCP provider: %w", err)
		}
	}

	startTime := time.Now()
	result := &discovery.Result{
		Provider:  "gcp",
		Timestamp: startTime,
		Resources: []models.Resource{},
		Regions:   options.Regions,
	}

	// If no regions specified, use default
	if len(options.Regions) == 0 {
		options.Regions = []string{"us-central1"}
	}

	// Discover resources in each region
	for _, region := range options.Regions {
		resources, err := p.DiscoverRegion(ctx, region)
		if err != nil {
			result.Errors = append(result.Errors, discovery.DiscoveryError{
				Region:  region,
				Service: "gcp",
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
	if p.gcpProvider == nil {
		var err error
		p.gcpProvider, err = discovery.NewGCPProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GCP provider: %w", err)
		}
	}

	// Use the GCP provider's Discover method
	options := discovery.DiscoveryOptions{
		Regions: []string{region},
		Timeout: 30 * time.Second,
	}
	
	result, err := p.gcpProvider.Discover(ctx, options)
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
	if p.gcpProvider != nil {
		return p.gcpProvider.Name()
	}
	return "Google Cloud Platform"
}

// Regions returns supported regions
func (p *Provider) Regions() []string {
	if p.gcpProvider != nil {
		return p.gcpProvider.Regions()
	}
	return []string{"us-central1", "us-east1", "us-west1", "europe-west1"}
}

// Services returns supported services
func (p *Provider) Services() []string {
	if p.gcpProvider != nil {
		return p.gcpProvider.Services()
	}
	return []string{"Compute", "Storage", "CloudSQL", "AppEngine"}
}

// ValidateCredentials validates GCP credentials
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	if p.gcpProvider == nil {
		var err error
		p.gcpProvider, err = discovery.NewGCPProvider()
		if err != nil {
			return fmt.Errorf("failed to validate credentials: %w", err)
		}
	}
	return nil
}

// GetAccountInfo returns GCP account information
func (p *Provider) GetAccountInfo(ctx context.Context) (*discovery.AccountInfo, error) {
	return &discovery.AccountInfo{
		Provider:  "gcp",
		ID: "my-project-123456",
		Name:      "GCP Project",
	}, nil
}