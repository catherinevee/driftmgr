package discovery

import (
	"context"
	"fmt"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// DigitalOceanProvider implements the Provider interface for DigitalOcean
type DigitalOceanProvider struct {
	discoverer *DigitalOceanDiscoverer
}

// NewDigitalOceanProvider creates a new DigitalOcean provider
func NewDigitalOceanProvider() (*DigitalOceanProvider, error) {
	discoverer, err := NewDigitalOceanDiscoverer("")
	if err != nil {
		return nil, fmt.Errorf("failed to create DigitalOcean discoverer: %w", err)
	}

	return &DigitalOceanProvider{
		discoverer: discoverer,
	}, nil
}

// Discover discovers DigitalOcean resources based on the options
func (p *DigitalOceanProvider) Discover(ctx context.Context, options DiscoveryOptions) (*Result, error) {
	var allResources []models.Resource

	// If specific regions are requested, we need to handle that
	if len(options.Regions) > 0 {
		for _, region := range options.Regions {
			discoverer, err := NewDigitalOceanDiscoverer(region)
			if err != nil {
				return nil, fmt.Errorf("failed to create discoverer for region %s: %w", region, err)
			}
			resources, err := discoverer.Discover()
			if err != nil {
				return nil, fmt.Errorf("failed to discover resources in region %s: %w", region, err)
			}
			allResources = append(allResources, resources...)
		}
	} else {
		// Otherwise discover all regions
		resources, err := p.discoverer.Discover()
		if err != nil {
			return nil, err
		}
		allResources = resources
	}

	return &Result{
		Resources: allResources,
		Metadata: map[string]interface{}{
			"provider":       "digitalocean",
			"resource_count": len(allResources),
			"regions":        options.Regions,
		},
	}, nil
}

// Name returns the provider name
func (p *DigitalOceanProvider) Name() string {
	return "DigitalOcean"
}

// Regions returns the list of available DigitalOcean regions
func (p *DigitalOceanProvider) Regions() []string {
	return p.SupportedRegions()
}

// Services returns the list of available DigitalOcean services
func (p *DigitalOceanProvider) Services() []string {
	return []string{
		"Droplets", "Kubernetes", "Databases", "Spaces",
		"Load Balancers", "Volumes", "VPC", "App Platform",
		"Monitoring", "CDN", "Container Registry", "Functions",
	}
}

// GetAccountInfo returns DigitalOcean account information
func (p *DigitalOceanProvider) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	if p.discoverer == nil || p.discoverer.client == nil {
		return nil, fmt.Errorf("DigitalOcean client not initialized")
	}
	
	account, err := p.discoverer.GetAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}
	
	return &AccountInfo{
		ID:       account.UUID,
		Name:     account.Email,
		Type:     "digitalocean",
		Provider: "digitalocean",
		Regions:  p.SupportedRegions(),
		Metadata: map[string]interface{}{
			"email":         account.Email,
			"status":        account.Status,
			"droplet_limit": account.DropletLimit,
			"team":          account.Team.Name,
		},
	}, nil
}

// SupportedRegions returns the list of supported regions
func (p *DigitalOceanProvider) SupportedRegions() []string {
	return []string{
		"nyc1", "nyc2", "nyc3", // New York
		"sfo1", "sfo2", "sfo3", // San Francisco
		"ams2", "ams3", // Amsterdam
		"sgp1", // Singapore
		"lon1", // London
		"fra1", // Frankfurt
		"tor1", // Toronto
		"blr1", // Bangalore
		"syd1", // Sydney
	}
}

// SupportedResourceTypes returns the list of supported resource types
func (p *DigitalOceanProvider) SupportedResourceTypes() []string {
	return []string{
		"digitalocean_droplet",
		"digitalocean_kubernetes_cluster",
		"digitalocean_database_cluster",
		"digitalocean_loadbalancer",
		"digitalocean_volume",
		"digitalocean_spaces_bucket",
		"digitalocean_domain",
		"digitalocean_project",
		"digitalocean_vpc",
		"digitalocean_firewall",
		"digitalocean_floating_ip",
		"digitalocean_ssh_key",
		"digitalocean_tag",
		"digitalocean_certificate",
		"digitalocean_cdn",
		"digitalocean_container_registry",
		"digitalocean_app",
		"digitalocean_monitor_alert",
		"digitalocean_record",
	}
}

// ValidateCredentials validates DigitalOcean credentials
func (p *DigitalOceanProvider) ValidateCredentials(ctx context.Context) error {
	if p.discoverer == nil || p.discoverer.client == nil {
		return fmt.Errorf("DigitalOcean client not initialized")
	}
	
	// Try to get account info to validate credentials
	_, err := p.discoverer.GetAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate DigitalOcean credentials: %w", err)
	}
	
	return nil
}

// DiscoverRegion discovers resources in a specific region
func (p *DigitalOceanProvider) DiscoverRegion(ctx context.Context, region string) ([]models.Resource, error) {
	options := DiscoveryOptions{
		Regions: []string{region},
	}
	result, err := p.Discover(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Resources, nil
}
