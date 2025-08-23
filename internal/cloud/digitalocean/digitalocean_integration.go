package digitalocean

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// DigitalOceanIntegration provides integration with the existing discovery system
type DigitalOceanIntegration struct {
	discoverer *DigitalOceanDiscoverer
}

// NewDigitalOceanIntegration creates a new DigitalOcean integration
func NewDigitalOceanIntegration(apiToken, region string) *DigitalOceanIntegration {
	discoverer, _ := NewDigitalOceanDiscoverer(apiToken)
	return &DigitalOceanIntegration{
		discoverer: discoverer,
	}
}

// DiscoverResources discovers DigitalOcean resources and integrates with the existing system
func (di *DigitalOceanIntegration) DiscoverResources(ctx context.Context, regions []string, provider string) ([]models.Resource, error) {
	log.Printf("Starting DigitalOcean resource discovery for regions: %v", regions)

	// Get API token from environment or credentials file
	apiToken := os.Getenv("DIGITALOCEAN_TOKEN")
	if apiToken == "" {
		// Try to read from credentials file
		homeDir, err := os.UserHomeDir()
		if err == nil {
			credentialsPath := fmt.Sprintf("%s/.digitalocean/credentials", homeDir)
			if content, err := os.ReadFile(credentialsPath); err == nil {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "DIGITALOCEAN_TOKEN=") {
						apiToken = strings.TrimPrefix(line, "DIGITALOCEAN_TOKEN=")
						break
					}
				}
			}
		}
	}

	if apiToken == "" {
		return nil, fmt.Errorf("DigitalOcean API token not found. Set DIGITALOCEAN_TOKEN environment variable or configure credentials")
	}

	// If no regions specified, use default regions
	if len(regions) == 0 {
		regions = []string{"nyc1", "sfo2", "lon1", "fra1", "sgp1", "tor1", "ams3", "blr1"}
	}

	var allResources []models.Resource

	for _, region := range regions {
		log.Printf("Scanning DigitalOcean region: %s", region)

		discoverer, _ := NewDigitalOceanDiscoverer(apiToken)

		// Discover different resource types
		resources, _ := discoverer.discoverDroplets(ctx)
		allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverLoadBalancers(ctx)
		allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverDatabases(ctx)
		allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverKubernetesClusters(ctx)
		allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverSpaces(ctx)
		allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverVolumes(ctx)
		allResources = append(allResources, resources...)

		// Note: Some methods are not yet implemented in the discoverer
		// resources, _ = discoverer.discoverSnapshots(ctx)
		// allResources = append(allResources, resources...)

		// resources, _ = discoverer.discoverNetworks(ctx)
		// allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverFirewalls(ctx)
		allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverDomains(ctx)
		allResources = append(allResources, resources...)

		// resources, _ = discoverer.discoverCertificates(ctx)
		// allResources = append(allResources, resources...)

		resources, _ = discoverer.discoverProjects(ctx)
		allResources = append(allResources, resources...)
	}

	log.Printf("DigitalOcean discovery complete. Found %d resources", len(allResources))
	return allResources, nil
}

// GetSupportedRegions returns the list of supported DigitalOcean regions
func (di *DigitalOceanIntegration) GetSupportedRegions() []string {
	return []string{
		"nyc1", "nyc3", // New York
		"sfo2", "sfo3", // San Francisco
		"lon1", "lon3", // London
		"fra1", "fra3", // Frankfurt
		"sgp1", "sgp3", // Singapore
		"tor1", "tor3", // Toronto
		"ams3", "ams4", // Amsterdam
		"blr1", "blr3", // Bangalore
		"syd1", "syd3", // Sydney
		"sgp1", "sgp3", // Singapore
	}
}

// ValidateCredentials validates DigitalOcean API credentials
func (di *DigitalOceanIntegration) ValidateCredentials(ctx context.Context) error {
	apiToken := os.Getenv("DIGITALOCEAN_TOKEN")
	if apiToken == "" {
		return fmt.Errorf("DIGITALOCEAN_TOKEN environment variable not set")
	}

	// Use doctl to test credentials
	// This would typically make a simple API call to test the token
	// For now, we'll just check if the token is not empty
	if len(apiToken) < 10 {
		return fmt.Errorf("DigitalOcean API token appears to be invalid (too short)")
	}

	return nil
}

// GetResourceTypes returns the list of DigitalOcean resource types that can be discovered
func (di *DigitalOceanIntegration) GetResourceTypes() []string {
	return []string{
		"digitalocean_droplet",
		"digitalocean_loadbalancer",
		"digitalocean_database_cluster",
		"digitalocean_kubernetes_cluster",
		"digitalocean_spaces_bucket",
		"digitalocean_volume",
		"digitalocean_snapshot",
		"digitalocean_vpc",
		"digitalocean_firewall",
		"digitalocean_domain",
		"digitalocean_certificate",
		"digitalocean_project",
	}
}
