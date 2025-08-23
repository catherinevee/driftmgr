package discovery

import (
	"fmt"
	"github.com/catherinevee/driftmgr/internal/models"
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

// Discover discovers DigitalOcean resources based on the config
func (p *DigitalOceanProvider) Discover(config Config) ([]models.Resource, error) {
	// If specific regions are requested, we need to handle that
	if len(config.Regions) > 0 {
		var allResources []models.Resource
		for _, region := range config.Regions {
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
		return allResources, nil
	}

	// Otherwise discover all regions
	return p.discoverer.Discover()
}

// Name returns the provider name
func (p *DigitalOceanProvider) Name() string {
	return "digitalocean"
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
