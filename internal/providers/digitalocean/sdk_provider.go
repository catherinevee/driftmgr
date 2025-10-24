package digitalocean

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

// DigitalOceanSDKProvider implements CloudProvider for DigitalOcean using DigitalOcean SDK
type DigitalOceanSDKProvider struct {
	client *godo.Client
	region string
}

// NewDigitalOceanSDKProvider creates a new DigitalOcean provider using DigitalOcean SDK
func NewDigitalOceanSDKProvider(region string) (*DigitalOceanSDKProvider, error) {
	// Get API token from environment
	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DIGITALOCEAN_TOKEN environment variable is required")
	}

	// Create OAuth2 token source
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	// Create OAuth2 client
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)

	// Create DigitalOcean client
	client := godo.NewClient(oauthClient)

	if region == "" {
		region = "nyc1"
	}

	return &DigitalOceanSDKProvider{
		client: client,
		region: region,
	}, nil
}

// Name returns the provider name
func (p *DigitalOceanSDKProvider) Name() string {
	return "digitalocean"
}

// DiscoverResources discovers resources in the specified region
func (p *DigitalOceanSDKProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	var resources []models.Resource

	// If region is specified, use it
	if region != "" {
		p.region = region
	}

	// Discover droplets
	droplets, err := p.discoverDroplets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover droplets: %w", err)
	}
	resources = append(resources, droplets...)

	// Discover volumes
	volumes, err := p.discoverVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover volumes: %w", err)
	}
	resources = append(resources, volumes...)

	// Discover load balancers
	loadBalancers, err := p.discoverLoadBalancers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover load balancers: %w", err)
	}
	resources = append(resources, loadBalancers...)

	// Discover databases
	databases, err := p.discoverDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover databases: %w", err)
	}
	resources = append(resources, databases...)

	return resources, nil
}

// discoverDroplets discovers DigitalOcean droplets
func (p *DigitalOceanSDKProvider) discoverDroplets(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// List all droplets
	droplets, _, err := p.client.Droplets.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list droplets: %w", err)
	}

	for _, droplet := range droplets {
		// Convert tags
		tags := make(map[string]string)
		for _, tag := range droplet.Tags {
			tags[tag] = ""
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = droplet.Name
		attributes["memory"] = droplet.Memory
		attributes["vcpus"] = droplet.Vcpus
		attributes["disk"] = droplet.Disk
		attributes["locked"] = droplet.Locked
		attributes["status"] = droplet.Status
		attributes["size_slug"] = droplet.SizeSlug
		attributes["created_at"] = droplet.Created
		attributes["features"] = droplet.Features
		attributes["backup_ids"] = droplet.BackupIDs
		attributes["snapshot_ids"] = droplet.SnapshotIDs
		attributes["image"] = droplet.Image
		attributes["size"] = droplet.Size
		attributes["networks"] = droplet.Networks
		attributes["region"] = droplet.Region
		attributes["tags"] = droplet.Tags
		attributes["vpc_uuid"] = droplet.VPCUUID

		resource := models.Resource{
			ID:           fmt.Sprintf("%d", droplet.ID),
			Type:         "digitalocean_droplet",
			Provider:     "digitalocean",
			Region:       droplet.Region.Slug,
			Attributes:   attributes,
			Tags:         tags,
			CreatedAt:    time.Now(), // DigitalOcean API returns string, using current time as fallback
			LastModified: time.Now(),
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverVolumes discovers DigitalOcean volumes
func (p *DigitalOceanSDKProvider) discoverVolumes(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// List all volumes
	volumes, _, err := p.client.Storage.ListVolumes(ctx, &godo.ListVolumeParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	for _, volume := range volumes {
		// Convert tags
		tags := make(map[string]string)
		for _, tag := range volume.Tags {
			tags[tag] = ""
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = volume.Name
		attributes["size_gigabytes"] = volume.SizeGigaBytes
		attributes["description"] = volume.Description
		attributes["droplet_ids"] = volume.DropletIDs
		attributes["created_at"] = volume.CreatedAt
		attributes["tags"] = volume.Tags
		attributes["filesystem_type"] = volume.FilesystemType
		attributes["filesystem_label"] = volume.FilesystemLabel
		attributes["region"] = volume.Region

		resource := models.Resource{
			ID:           volume.ID,
			Type:         "digitalocean_volume",
			Provider:     "digitalocean",
			Region:       volume.Region.Slug,
			Attributes:   attributes,
			Tags:         tags,
			CreatedAt:    volume.CreatedAt,
			LastModified: volume.CreatedAt,
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverLoadBalancers discovers DigitalOcean load balancers
func (p *DigitalOceanSDKProvider) discoverLoadBalancers(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// List all load balancers
	loadBalancers, _, err := p.client.LoadBalancers.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list load balancers: %w", err)
	}

	for _, lb := range loadBalancers {
		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = lb.Name
		attributes["ip"] = lb.IP
		attributes["algorithm"] = lb.Algorithm
		attributes["status"] = lb.Status
		attributes["created_at"] = lb.Created
		attributes["forwarding_rules"] = lb.ForwardingRules
		attributes["health_check"] = lb.HealthCheck
		attributes["sticky_sessions"] = lb.StickySessions
		attributes["tag"] = lb.Tag
		attributes["droplet_ids"] = lb.DropletIDs
		attributes["redirect_http_to_https"] = lb.RedirectHttpToHttps
		attributes["enable_proxy_protocol"] = lb.EnableProxyProtocol
		attributes["vpc_uuid"] = lb.VPCUUID
		attributes["region"] = lb.Region

		resource := models.Resource{
			ID:           lb.ID,
			Type:         "digitalocean_loadbalancer",
			Provider:     "digitalocean",
			Region:       lb.Region.Slug,
			Attributes:   attributes,
			Tags:         make(map[string]string), // Load balancers don't have tags in the same way
			CreatedAt:    time.Now(),              // DigitalOcean API returns string, using current time as fallback
			LastModified: time.Now(),
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverDatabases discovers DigitalOcean databases
func (p *DigitalOceanSDKProvider) discoverDatabases(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// List all databases
	databases, _, err := p.client.Databases.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	for _, db := range databases {
		// Convert tags
		tags := make(map[string]string)
		for _, tag := range db.Tags {
			tags[tag] = ""
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = db.Name
		attributes["engine"] = db.EngineSlug
		attributes["version"] = db.VersionSlug
		attributes["num_nodes"] = db.NumNodes
		attributes["size"] = db.SizeSlug
		attributes["db_names"] = db.DBNames
		attributes["users"] = db.Users
		attributes["status"] = db.Status
		attributes["created_at"] = db.CreatedAt
		attributes["maintenance_window"] = db.MaintenanceWindow
		attributes["tags"] = db.Tags
		attributes["private_network_uuid"] = db.PrivateNetworkUUID
		attributes["region"] = db.RegionSlug

		resource := models.Resource{
			ID:           db.ID,
			Type:         "digitalocean_database_cluster",
			Provider:     "digitalocean",
			Region:       db.RegionSlug,
			Attributes:   attributes,
			Tags:         tags,
			CreatedAt:    db.CreatedAt,
			LastModified: db.CreatedAt,
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// GetResource retrieves a specific resource by ID
func (p *DigitalOceanSDKProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	// Try to determine resource type from ID format
	var resourceType string

	switch {
	case strings.HasPrefix(resourceID, "droplet-") || isNumeric(resourceID):
		resourceType = "digitalocean_droplet"
	case strings.HasPrefix(resourceID, "vol-"):
		resourceType = "digitalocean_volume"
	case strings.HasPrefix(resourceID, "lb-"):
		resourceType = "digitalocean_loadbalancer"
	case strings.HasPrefix(resourceID, "db-"):
		resourceType = "digitalocean_database_cluster"
	default:
		// Try different resource types
		for _, rt := range []string{"digitalocean_droplet", "digitalocean_volume", "digitalocean_loadbalancer", "digitalocean_database_cluster"} {
			if resource, err := p.GetResourceByType(ctx, rt, resourceID); err == nil {
				return resource, nil
			}
		}
		return nil, fmt.Errorf("unable to determine resource type for ID: %s", resourceID)
	}

	return p.GetResourceByType(ctx, resourceType, resourceID)
}

// GetResourceByType retrieves a specific resource by type and name
func (p *DigitalOceanSDKProvider) GetResourceByType(ctx context.Context, resourceType, resourceID string) (*models.Resource, error) {
	switch resourceType {
	case "digitalocean_droplet":
		// Convert string ID to int for droplet
		var dropletID int
		if _, err := fmt.Sscanf(resourceID, "%d", &dropletID); err != nil {
			return nil, fmt.Errorf("invalid droplet ID format: %s", resourceID)
		}

		droplet, _, err := p.client.Droplets.Get(ctx, dropletID)
		if err != nil {
			return nil, fmt.Errorf("failed to get droplet: %w", err)
		}

		// Convert tags
		tags := make(map[string]string)
		for _, tag := range droplet.Tags {
			tags[tag] = ""
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = droplet.Name
		attributes["memory"] = droplet.Memory
		attributes["vcpus"] = droplet.Vcpus
		attributes["disk"] = droplet.Disk
		attributes["locked"] = droplet.Locked
		attributes["status"] = droplet.Status
		attributes["size_slug"] = droplet.SizeSlug
		attributes["created_at"] = droplet.Created
		attributes["features"] = droplet.Features
		attributes["backup_ids"] = droplet.BackupIDs
		attributes["snapshot_ids"] = droplet.SnapshotIDs
		attributes["image"] = droplet.Image
		attributes["size"] = droplet.Size
		attributes["networks"] = droplet.Networks
		attributes["region"] = droplet.Region
		attributes["tags"] = droplet.Tags
		attributes["vpc_uuid"] = droplet.VPCUUID

		return &models.Resource{
			ID:           fmt.Sprintf("%d", droplet.ID),
			Type:         "digitalocean_droplet",
			Provider:     "digitalocean",
			Region:       droplet.Region.Slug,
			Attributes:   attributes,
			Tags:         tags,
			CreatedAt:    time.Now(), // DigitalOcean API returns string, using current time as fallback
			LastModified: time.Now(),
		}, nil

	case "digitalocean_volume":
		volume, _, err := p.client.Storage.GetVolume(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get volume: %w", err)
		}

		// Convert tags
		tags := make(map[string]string)
		for _, tag := range volume.Tags {
			tags[tag] = ""
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = volume.Name
		attributes["size_gigabytes"] = volume.SizeGigaBytes
		attributes["description"] = volume.Description
		attributes["droplet_ids"] = volume.DropletIDs
		attributes["created_at"] = volume.CreatedAt
		attributes["tags"] = volume.Tags
		attributes["filesystem_type"] = volume.FilesystemType
		attributes["filesystem_label"] = volume.FilesystemLabel
		attributes["region"] = volume.Region

		return &models.Resource{
			ID:           volume.ID,
			Type:         "digitalocean_volume",
			Provider:     "digitalocean",
			Region:       volume.Region.Slug,
			Attributes:   attributes,
			Tags:         tags,
			CreatedAt:    volume.CreatedAt,
			LastModified: volume.CreatedAt,
		}, nil

	case "digitalocean_loadbalancer":
		lb, _, err := p.client.LoadBalancers.Get(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get load balancer: %w", err)
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = lb.Name
		attributes["ip"] = lb.IP
		attributes["algorithm"] = lb.Algorithm
		attributes["status"] = lb.Status
		attributes["created_at"] = lb.Created
		attributes["forwarding_rules"] = lb.ForwardingRules
		attributes["health_check"] = lb.HealthCheck
		attributes["sticky_sessions"] = lb.StickySessions
		attributes["tag"] = lb.Tag
		attributes["droplet_ids"] = lb.DropletIDs
		attributes["redirect_http_to_https"] = lb.RedirectHttpToHttps
		attributes["enable_proxy_protocol"] = lb.EnableProxyProtocol
		attributes["vpc_uuid"] = lb.VPCUUID
		attributes["region"] = lb.Region

		return &models.Resource{
			ID:           lb.ID,
			Type:         "digitalocean_loadbalancer",
			Provider:     "digitalocean",
			Region:       lb.Region.Slug,
			Attributes:   attributes,
			Tags:         make(map[string]string),
			CreatedAt:    time.Now(), // DigitalOcean API returns string, using current time as fallback
			LastModified: time.Now(),
		}, nil

	case "digitalocean_database_cluster":
		db, _, err := p.client.Databases.Get(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get database: %w", err)
		}

		// Convert tags
		tags := make(map[string]string)
		for _, tag := range db.Tags {
			tags[tag] = ""
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = db.Name
		attributes["engine"] = db.EngineSlug
		attributes["version"] = db.VersionSlug
		attributes["num_nodes"] = db.NumNodes
		attributes["size"] = db.SizeSlug
		attributes["db_names"] = db.DBNames
		attributes["users"] = db.Users
		attributes["status"] = db.Status
		attributes["created_at"] = db.CreatedAt
		attributes["maintenance_window"] = db.MaintenanceWindow
		attributes["tags"] = db.Tags
		attributes["private_network_uuid"] = db.PrivateNetworkUUID
		attributes["region"] = db.RegionSlug

		return &models.Resource{
			ID:           db.ID,
			Type:         "digitalocean_database_cluster",
			Provider:     "digitalocean",
			Region:       db.RegionSlug,
			Attributes:   attributes,
			Tags:         tags,
			CreatedAt:    db.CreatedAt,
			LastModified: db.CreatedAt,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// ValidateCredentials checks if the provider credentials are valid
func (p *DigitalOceanSDKProvider) ValidateCredentials(ctx context.Context) error {
	// Test connection by getting account info
	_, _, err := p.client.Account.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate DigitalOcean credentials: %w", err)
	}
	return nil
}

// ListRegions returns available regions for the provider
func (p *DigitalOceanSDKProvider) ListRegions(ctx context.Context) ([]string, error) {
	// Return common DigitalOcean regions
	return []string{
		"nyc1", "nyc2", "nyc3", "sfo1", "sfo2", "sfo3",
		"ams2", "ams3", "sgp1", "lon1", "fra1", "tor1",
		"blr1", "syd1", "sfo3", "nyc3", "ams3", "sgp1",
	}, nil
}

// SupportedResourceTypes returns the list of supported resource types
func (p *DigitalOceanSDKProvider) SupportedResourceTypes() []string {
	return []string{
		"digitalocean_droplet",
		"digitalocean_volume",
		"digitalocean_load_balancer",
		"digitalocean_database_cluster",
		"digitalocean_kubernetes_cluster",
		"digitalocean_vpc",
		"digitalocean_firewall",
		"digitalocean_domain",
		"digitalocean_record",
		"digitalocean_spaces_bucket",
		"digitalocean_certificate",
		"digitalocean_snapshot",
		"digitalocean_image",
		"digitalocean_ssh_key",
		"digitalocean_tag",
		"digitalocean_project",
		"digitalocean_team",
		"digitalocean_alert_policy",
		"digitalocean_monitoring_alert",
		"digitalocean_cdn",
	}
}

// TestConnection tests the connection to DigitalOcean
func (p *DigitalOceanSDKProvider) TestConnection(ctx context.Context) error {
	return p.ValidateCredentials(ctx)
}

// GetRegion returns the region
func (p *DigitalOceanSDKProvider) GetRegion() string {
	return p.region
}

// ListResourcesByType lists all resources of a specific type
func (p *DigitalOceanSDKProvider) ListResourcesByType(ctx context.Context, resourceType string) ([]models.Resource, error) {
	var resources []models.Resource

	// Get all resources and filter by type
	allResources, err := p.DiscoverResources(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	for _, resource := range allResources {
		if resource.Type == resourceType {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}
