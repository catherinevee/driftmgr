package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

// DigitalOceanDiscoverer handles DigitalOcean resource discovery
type DigitalOceanDiscoverer struct {
	client *godo.Client
	region string
}

// TokenSource for OAuth2
type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

// NewDigitalOceanDiscoverer creates a new DigitalOcean discoverer
func NewDigitalOceanDiscoverer(region string) (*DigitalOceanDiscoverer, error) {
	// Check various common DigitalOcean token environment variables
	tokenVars := []string{
		"DIGITALOCEAN_ACCESS_TOKEN",
		"DIGITALOCEAN_TOKEN",
		"DO_TOKEN",
		"DO_ACCESS_TOKEN",
		"DIGITAL_OCEAN_TOKEN",
	}

	var token string
	for _, envVar := range tokenVars {
		if t := os.Getenv(envVar); t != "" {
			token = t
			break
		}
	}

	// If no environment variable, try to get token from doctl config file
	if token == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			doctlConfigPath := filepath.Join(homeDir, "AppData", "Roaming", "doctl", "config.yaml")
			data, err := os.ReadFile(doctlConfigPath)
			if err == nil {
				var config map[string]interface{}
				if err := yaml.Unmarshal(data, &config); err == nil {
					if accessToken, ok := config["access-token"].(string); ok {
						token = accessToken
					}
				}
			}
		}
	}

	if token == "" {
		return nil, fmt.Errorf("DigitalOcean token not found. Set one of: %v or configure doctl", tokenVars)
	}

	tokenSource := &TokenSource{
		AccessToken: token,
	}

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)

	return &DigitalOceanDiscoverer{
		client: client,
		region: region,
	}, nil
}

// Discover discovers all DigitalOcean resources
func (d *DigitalOceanDiscoverer) Discover() ([]models.Resource, error) {
	var resources []models.Resource
	ctx := context.Background()

	// Discover Droplets
	droplets, err := d.discoverDroplets(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover droplets: %v\n", err)
	} else {
		resources = append(resources, droplets...)
	}

	// Discover Kubernetes Clusters
	clusters, err := d.discoverKubernetesClusters(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover Kubernetes clusters: %v\n", err)
	} else {
		resources = append(resources, clusters...)
	}

	// Discover Databases
	databases, err := d.discoverDatabases(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover databases: %v\n", err)
	} else {
		resources = append(resources, databases...)
	}

	// Discover Load Balancers
	loadBalancers, err := d.discoverLoadBalancers(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover load balancers: %v\n", err)
	} else {
		resources = append(resources, loadBalancers...)
	}

	// Discover Volumes
	volumes, err := d.discoverVolumes(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover volumes: %v\n", err)
	} else {
		resources = append(resources, volumes...)
	}

	// Discover Spaces (S3-compatible storage)
	spaces, err := d.discoverSpaces(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover spaces: %v\n", err)
	} else {
		resources = append(resources, spaces...)
	}

	// Discover Domains
	domains, err := d.discoverDomains(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover domains: %v\n", err)
	} else {
		resources = append(resources, domains...)
	}

	// Discover Projects
	projects, err := d.discoverProjects(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover projects: %v\n", err)
	} else {
		resources = append(resources, projects...)
	}

	// Discover VPCs
	vpcs, err := d.discoverVPCs(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover VPCs: %v\n", err)
	} else {
		resources = append(resources, vpcs...)
	}

	// Discover Firewalls
	firewalls, err := d.discoverFirewalls(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to discover firewalls: %v\n", err)
	} else {
		resources = append(resources, firewalls...)
	}

	return resources, nil
}

// discoverDroplets discovers all droplets
func (d *DigitalOceanDiscoverer) discoverDroplets(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		droplets, resp, err := d.client.Droplets.List(ctx, opt)
		if err != nil {
			return resources, fmt.Errorf("failed to list droplets: %w", err)
		}

		for _, droplet := range droplets {
			// Filter by region if specified
			if d.region != "" && droplet.Region.Slug != d.region {
				continue
			}

			resource := models.Resource{
				ID:       fmt.Sprintf("%d", droplet.ID),
				Name:     droplet.Name,
				Type:     "digitalocean_droplet",
				Provider: "digitalocean",
				Region:   droplet.Region.Slug,
				State:    droplet.Status,
				Tags:     droplet.Tags, // []string type
				Attributes: map[string]interface{}{
					"size":       droplet.Size.Slug,
					"image":      droplet.Image.Slug,
					"vpc_uuid":   droplet.VPCUUID,
					"created_at": droplet.Created,
					"ip_address": droplet.Networks.V4[0].IPAddress,
				},
			}
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return resources, nil
}

// discoverKubernetesClusters discovers Kubernetes clusters
func (d *DigitalOceanDiscoverer) discoverKubernetesClusters(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	clusters, _, err := d.client.Kubernetes.List(ctx, nil)
	if err != nil {
		return resources, fmt.Errorf("failed to list Kubernetes clusters: %w", err)
	}

	for _, cluster := range clusters {
		// Filter by region if specified
		if d.region != "" && cluster.RegionSlug != d.region {
			continue
		}

		resource := models.Resource{
			ID:       cluster.ID,
			Name:     cluster.Name,
			Type:     "digitalocean_kubernetes_cluster",
			Provider: "digitalocean",
			Region:   cluster.RegionSlug,
			State:    string(cluster.Status.State),
			Tags:     cluster.Tags,
			Attributes: map[string]interface{}{
				"version":       cluster.VersionSlug,
				"node_count":    len(cluster.NodePools),
				"vpc_uuid":      cluster.VPCUUID,
				"created_at":    cluster.CreatedAt,
				"auto_upgrade":  cluster.AutoUpgrade,
				"surge_upgrade": cluster.SurgeUpgrade,
			},
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverDatabases discovers managed databases
func (d *DigitalOceanDiscoverer) discoverDatabases(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		databases, resp, err := d.client.Databases.List(ctx, opt)
		if err != nil {
			return resources, fmt.Errorf("failed to list databases: %w", err)
		}

		for _, db := range databases {
			// Filter by region if specified
			if d.region != "" && db.RegionSlug != d.region {
				continue
			}

			resource := models.Resource{
				ID:       db.ID,
				Name:     db.Name,
				Type:     "digitalocean_database_cluster",
				Provider: "digitalocean",
				Region:   db.RegionSlug,
				State:    db.Status,
				Tags:     db.Tags,
				Attributes: map[string]interface{}{
					"engine":               db.EngineSlug,
					"version":              db.VersionSlug,
					"size":                 db.SizeSlug,
					"node_count":           db.NumNodes,
					"private_network_uuid": db.PrivateNetworkUUID,
					"created_at":           db.CreatedAt,
				},
			}
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return resources, nil
}

// discoverLoadBalancers discovers load balancers
func (d *DigitalOceanDiscoverer) discoverLoadBalancers(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		loadBalancers, resp, err := d.client.LoadBalancers.List(ctx, opt)
		if err != nil {
			return resources, fmt.Errorf("failed to list load balancers: %w", err)
		}

		for _, lb := range loadBalancers {
			// Filter by region if specified
			if d.region != "" && lb.Region.Slug != d.region {
				continue
			}

			resource := models.Resource{
				ID:       lb.ID,
				Name:     lb.Name,
				Type:     "digitalocean_loadbalancer",
				Provider: "digitalocean",
				Region:   lb.Region.Slug,
				State:    lb.Status,
				Tags:     lb.Tags,
				Attributes: map[string]interface{}{
					"ip":               lb.IP,
					"algorithm":        lb.Algorithm,
					"size":             lb.SizeSlug,
					"vpc_uuid":         lb.VPCUUID,
					"droplet_ids":      lb.DropletIDs,
					"created_at":       lb.Created,
					"forwarding_rules": len(lb.ForwardingRules),
				},
			}
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return resources, nil
}

// discoverVolumes discovers block storage volumes
func (d *DigitalOceanDiscoverer) discoverVolumes(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		volumes, resp, err := d.client.Storage.ListVolumes(ctx, &godo.ListVolumeParams{
			ListOptions: opt,
		})
		if err != nil {
			return resources, fmt.Errorf("failed to list volumes: %w", err)
		}

		for _, volume := range volumes {
			// Filter by region if specified
			if d.region != "" && volume.Region.Slug != d.region {
				continue
			}

			resource := models.Resource{
				ID:       volume.ID,
				Name:     volume.Name,
				Type:     "digitalocean_volume",
				Provider: "digitalocean",
				Region:   volume.Region.Slug,
				Tags:     volume.Tags,
				Attributes: map[string]interface{}{
					"size_gigabytes":  volume.SizeGigaBytes,
					"description":     volume.Description,
					"droplet_ids":     volume.DropletIDs,
					"filesystem_type": volume.FilesystemType,
					"created_at":      volume.CreatedAt,
				},
			}
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return resources, nil
}

// discoverSpaces discovers Spaces (S3-compatible storage)
func (d *DigitalOceanDiscoverer) discoverSpaces(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Use DigitalOcean API to list Spaces buckets
	// Note: DigitalOcean API doesn't directly support Spaces listing,
	// but we can discover them through the CDN endpoint API
	
	// Get CDN endpoints which are associated with Spaces
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		cdnEndpoints, resp, err := d.client.CDNs.List(ctx, opt)
		if err != nil {
			// CDN API might not be available in all accounts
			// Return empty list without error
			return resources, nil
		}

		for _, cdn := range cdnEndpoints {
			// Each CDN endpoint represents a Space
			spaceName := extractSpaceNameFromOrigin(cdn.Origin)
			if spaceName == "" {
				continue
			}

			resource := models.Resource{
				ID:       cdn.ID,
				Name:     spaceName,
				Type:     "digitalocean_spaces_bucket",
				Provider: "digitalocean",
				Attributes: map[string]interface{}{
					"origin":      cdn.Origin,
					"endpoint":    cdn.Endpoint,
					"created_at":  cdn.CreatedAt,
					"ttl":         cdn.TTL,
					"custom_domain": cdn.CustomDomain,
					"certificate_id": cdn.CertificateID,
				},
			}
			
			// Extract region from origin URL if possible
			if region := extractRegionFromSpacesURL(cdn.Origin); region != "" {
				resource.Region = region
			}
			
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	// Also check for Spaces via the Apps API (some Spaces might be used by Apps)
	apps, _, err := d.client.Apps.List(ctx, nil)
	if err == nil {
		for _, app := range apps {
			// Check static sites which might use Spaces
			for _, spec := range app.Spec.StaticSites {
				if spec.SourceDir != "" && strings.Contains(spec.SourceDir, "spaces") {
					resource := models.Resource{
						ID:       fmt.Sprintf("spaces-app-%s", app.ID),
						Name:     fmt.Sprintf("%s-spaces", app.Spec.Name),
						Type:     "digitalocean_spaces_bucket",
						Provider: "digitalocean",
						Region:   app.Region.Slug,
						Attributes: map[string]interface{}{
							"app_id":     app.ID,
							"app_name":   app.Spec.Name,
							"source_dir": spec.SourceDir,
						},
					}
					resources = append(resources, resource)
				}
			}
		}
	}

	return resources, nil
}

// extractSpaceNameFromOrigin extracts the Space name from a CDN origin URL
func extractSpaceNameFromOrigin(origin string) string {
	// Origin format: spacename.region.digitaloceanspaces.com
	if strings.Contains(origin, ".digitaloceanspaces.com") {
		parts := strings.Split(origin, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

// extractRegionFromSpacesURL extracts the region from a Spaces URL
func extractRegionFromSpacesURL(url string) string {
	// URL format: spacename.region.digitaloceanspaces.com
	if strings.Contains(url, ".digitaloceanspaces.com") {
		parts := strings.Split(url, ".")
		if len(parts) >= 3 {
			return parts[1]
		}
	}
	return ""
}

// discoverDomains discovers DNS domains
func (d *DigitalOceanDiscoverer) discoverDomains(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	domains, _, err := d.client.Domains.List(ctx, nil)
	if err != nil {
		return resources, fmt.Errorf("failed to list domains: %w", err)
	}

	for _, domain := range domains {
		resource := models.Resource{
			ID:       domain.Name,
			Name:     domain.Name,
			Type:     "digitalocean_domain",
			Provider: "digitalocean",
			Attributes: map[string]interface{}{
				"ttl": domain.TTL,
			},
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverProjects discovers projects
func (d *DigitalOceanDiscoverer) discoverProjects(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		projects, resp, err := d.client.Projects.List(ctx, opt)
		if err != nil {
			return resources, fmt.Errorf("failed to list projects: %w", err)
		}

		for _, project := range projects {
			resource := models.Resource{
				ID:       project.ID,
				Name:     project.Name,
				Type:     "digitalocean_project",
				Provider: "digitalocean",
				Attributes: map[string]interface{}{
					"description": project.Description,
					"purpose":     project.Purpose,
					"environment": project.Environment,
					"is_default":  project.IsDefault,
					"created_at":  project.CreatedAt,
				},
			}
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return resources, nil
}

// discoverVPCs discovers VPCs
func (d *DigitalOceanDiscoverer) discoverVPCs(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		vpcs, resp, err := d.client.VPCs.List(ctx, opt)
		if err != nil {
			return resources, fmt.Errorf("failed to list VPCs: %w", err)
		}

		for _, vpc := range vpcs {
			// Filter by region if specified
			if d.region != "" && vpc.RegionSlug != d.region {
				continue
			}

			resource := models.Resource{
				ID:       vpc.ID,
				Name:     vpc.Name,
				Type:     "digitalocean_vpc",
				Provider: "digitalocean",
				Region:   vpc.RegionSlug,
				Attributes: map[string]interface{}{
					"ip_range":    vpc.IPRange,
					"description": vpc.Description,
					"default":     vpc.Default,
					"created_at":  vpc.CreatedAt,
				},
			}
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return resources, nil
}

// discoverFirewalls discovers firewalls
func (d *DigitalOceanDiscoverer) discoverFirewalls(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		firewalls, resp, err := d.client.Firewalls.List(ctx, opt)
		if err != nil {
			return resources, fmt.Errorf("failed to list firewalls: %w", err)
		}

		for _, firewall := range firewalls {
			resource := models.Resource{
				ID:       firewall.ID,
				Name:     firewall.Name,
				Type:     "digitalocean_firewall",
				Provider: "digitalocean",
				State:    firewall.Status,
				Tags:     firewall.Tags,
				Attributes: map[string]interface{}{
					"inbound_rules":  len(firewall.InboundRules),
					"outbound_rules": len(firewall.OutboundRules),
					"droplet_ids":    firewall.DropletIDs,
					"created_at":     firewall.Created,
				},
			}
			resources = append(resources, resource)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return resources, nil
}

// IsAvailable checks if DigitalOcean credentials are available
func (d *DigitalOceanDiscoverer) IsAvailable() bool {
	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if token == "" {
		token = os.Getenv("DO_TOKEN")
	}
	if token == "" {
		token = os.Getenv("DIGITALOCEAN_ACCESS_TOKEN")
	}
	return token != ""
}

// GetRegions returns available DigitalOcean regions
func (d *DigitalOceanDiscoverer) GetRegions(ctx context.Context) ([]string, error) {
	regions, _, err := d.client.Regions.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	var regionSlugs []string
	for _, region := range regions {
		if region.Available {
			regionSlugs = append(regionSlugs, region.Slug)
		}
	}

	return regionSlugs, nil
}

// GetAccount returns account information
func (d *DigitalOceanDiscoverer) GetAccount(ctx context.Context) (*godo.Account, error) {
	account, _, err := d.client.Account.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return account, nil
}

// CheckCredentials verifies DigitalOcean API credentials
func CheckDigitalOceanCredentials() bool {
	// Check various common DigitalOcean token environment variables
	tokenVars := []string{
		"DIGITALOCEAN_ACCESS_TOKEN",
		"DIGITALOCEAN_TOKEN",
		"DO_TOKEN",
		"DO_ACCESS_TOKEN",
		"DIGITAL_OCEAN_TOKEN",
	}

	for _, envVar := range tokenVars {
		if t := os.Getenv(envVar); t != "" {
			return true
		}
	}

	// Check for doctl config file
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		doctlConfigPath := filepath.Join(homeDir, "AppData", "Roaming", "doctl", "config.yaml")
		if _, err := os.Stat(doctlConfigPath); err == nil {
			// Config file exists, check if it has an access token
			data, err := os.ReadFile(doctlConfigPath)
			if err == nil && strings.Contains(string(data), "access-token:") {
				return true
			}
		}
	}

	return false
}

// GetDigitalOceanAccountInfo returns account details
func GetDigitalOceanAccountInfo() (string, error) {
	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if token == "" {
		token = os.Getenv("DO_TOKEN")
	}
	if token == "" {
		token = os.Getenv("DIGITALOCEAN_ACCESS_TOKEN")
	}

	if token == "" {
		return "", fmt.Errorf("DigitalOcean token not found")
	}

	tokenSource := &TokenSource{AccessToken: token}
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)

	account, _, err := client.Account.Get(context.Background())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Email: %s, Team: %s, Status: %s",
		account.Email,
		account.Team.Name,
		account.Status), nil
}
