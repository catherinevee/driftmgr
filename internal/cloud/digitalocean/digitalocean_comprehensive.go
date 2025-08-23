package digitalocean

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/digitalocean/godo"
)

// ComprehensiveDigitalOceanDiscoverer discovers ALL DigitalOcean resources
type ComprehensiveDigitalOceanDiscoverer struct {
	client   *godo.Client
	progress chan DODiscoveryProgress
}

// DODiscoveryProgress tracks discovery progress for DigitalOcean
type DODiscoveryProgress struct {
	Service      string
	ResourceType string
	Count        int
	Message      string
}

// NewComprehensiveDigitalOceanDiscoverer creates a new comprehensive DigitalOcean discoverer
func NewComprehensiveDigitalOceanDiscoverer() (*ComprehensiveDigitalOceanDiscoverer, error) {
	// Get token from environment or configuration
	token := getDigitalOceanToken()
	if token == "" {
		return nil, fmt.Errorf("DigitalOcean API token not configured")
	}

	client := godo.NewFromToken(token)

	return &ComprehensiveDigitalOceanDiscoverer{
		client:   client,
		progress: make(chan DODiscoveryProgress, 100),
	}, nil
}

// DiscoverAllDigitalOceanResources discovers all DigitalOcean resources
func (d *ComprehensiveDigitalOceanDiscoverer) DiscoverAllDigitalOceanResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	// Start progress reporter
	go d.reportProgress()

	var allResources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// DigitalOcean resources discovery functions
	resourceTypes := []func(context.Context) []models.Resource{
		d.discoverDroplets,
		d.discoverKubernetesClusters,
		d.discoverDatabases,
		d.discoverLoadBalancers,
		d.discoverVolumes,
		d.discoverSnapshots,
		d.discoverFloatingIPs,
		d.discoverDomains,
		d.discoverFirewalls,
		d.discoverVPCs,
		d.discoverSSHKeys,
		d.discoverProjects,
		d.discoverSpaces,
		d.discoverAppPlatformApps,
		d.discoverCDNEndpoints,
		d.discoverContainerRegistry,
		d.discoverDatabaseReplicas,
		d.discoverDatabasePools,
	}

	for _, discoveryFunc := range resourceTypes {
		wg.Add(1)
		go func(fn func(context.Context) []models.Resource) {
			defer wg.Done()
			resources := fn(ctx)
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(discoveryFunc)
	}

	wg.Wait()
	close(d.progress)

	log.Printf("Comprehensive DigitalOcean discovery completed: %d total resources found", len(allResources))
	return allResources, nil
}

// discoverDroplets discovers all droplets
func (d *ComprehensiveDigitalOceanDiscoverer) discoverDroplets(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		droplets, resp, err := d.client.Droplets.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list droplets: %v", err)
			break
		}

		for _, droplet := range droplets {
			properties := make(map[string]interface{})
			properties["size"] = droplet.Size.Slug
			properties["image"] = droplet.Image.Slug
			properties["vcpus"] = droplet.Vcpus
			properties["memory"] = droplet.Memory
			properties["disk"] = droplet.Disk
			properties["status"] = droplet.Status

			// Get primary IP
			if len(droplet.Networks.V4) > 0 {
				properties["ip_address"] = droplet.Networks.V4[0].IPAddress
			}

			tags := make(map[string]string)
			for _, tag := range droplet.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("%d", droplet.ID),
				Name:       droplet.Name,
				Type:       "digitalocean_droplet",
				Provider:   "digitalocean",
				Region:     droplet.Region.Slug,
				State:      droplet.Status,
				Tags:       tags,
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Compute", ResourceType: "Droplets", Count: len(resources)}
	return resources
}

// discoverKubernetesClusters discovers all Kubernetes clusters
func (d *ComprehensiveDigitalOceanDiscoverer) discoverKubernetesClusters(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		clusters, resp, err := d.client.Kubernetes.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list Kubernetes clusters: %v", err)
			break
		}

		for _, cluster := range clusters {
			properties := make(map[string]interface{})
			properties["version"] = cluster.VersionSlug
			properties["status"] = cluster.Status.State
			properties["node_pools"] = len(cluster.NodePools)

			var nodeCount int
			for _, pool := range cluster.NodePools {
				nodeCount += pool.Count
			}
			properties["total_nodes"] = nodeCount

			tags := make(map[string]string)
			for _, tag := range cluster.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         cluster.ID,
				Name:       cluster.Name,
				Type:       "digitalocean_kubernetes_cluster",
				Provider:   "digitalocean",
				Region:     cluster.RegionSlug,
				State:      string(cluster.Status.State),
				Tags:       tags,
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Kubernetes", ResourceType: "Clusters", Count: len(resources)}
	return resources
}

// discoverDatabases discovers all managed databases
func (d *ComprehensiveDigitalOceanDiscoverer) discoverDatabases(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		databases, resp, err := d.client.Databases.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list databases: %v", err)
			break
		}

		for _, db := range databases {
			properties := make(map[string]interface{})
			properties["engine"] = db.EngineSlug
			properties["version"] = db.VersionSlug
			properties["size"] = db.SizeSlug
			properties["status"] = db.Status
			properties["nodes"] = db.NumNodes

			if db.Connection != nil {
				properties["host"] = db.Connection.Host
				properties["port"] = db.Connection.Port
			}

			tags := make(map[string]string)
			for _, tag := range db.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         db.ID,
				Name:       db.Name,
				Type:       "digitalocean_database",
				Provider:   "digitalocean",
				Region:     db.RegionSlug,
				State:      db.Status,
				Tags:       tags,
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Database", ResourceType: "Managed Databases", Count: len(resources)}
	return resources
}

// discoverLoadBalancers discovers all load balancers
func (d *ComprehensiveDigitalOceanDiscoverer) discoverLoadBalancers(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		lbs, resp, err := d.client.LoadBalancers.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list load balancers: %v", err)
			break
		}

		for _, lb := range lbs {
			properties := make(map[string]interface{})
			properties["algorithm"] = lb.Algorithm
			properties["status"] = lb.Status
			properties["ip"] = lb.IP
			properties["size"] = lb.SizeSlug
			properties["forwarding_rules"] = len(lb.ForwardingRules)

			tags := make(map[string]string)
			for _, tag := range lb.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         lb.ID,
				Name:       lb.Name,
				Type:       "digitalocean_load_balancer",
				Provider:   "digitalocean",
				Region:     lb.Region.Slug,
				State:      lb.Status,
				Tags:       tags,
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Network", ResourceType: "Load Balancers", Count: len(resources)}
	return resources
}

// discoverVolumes discovers all block storage volumes
func (d *ComprehensiveDigitalOceanDiscoverer) discoverVolumes(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListVolumeParams{
		ListOptions: &godo.ListOptions{
			Page:    1,
			PerPage: 200,
		},
	}

	for {
		volumes, resp, err := d.client.Storage.ListVolumes(ctx, opt)
		if err != nil {
			log.Printf("Failed to list volumes: %v", err)
			break
		}

		for _, volume := range volumes {
			properties := make(map[string]interface{})
			properties["size_gb"] = volume.SizeGigaBytes
			properties["filesystem_type"] = volume.FilesystemType
			properties["filesystem_label"] = volume.FilesystemLabel

			if len(volume.DropletIDs) > 0 {
				properties["attached_to"] = volume.DropletIDs
			}

			tags := make(map[string]string)
			for _, tag := range volume.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         volume.ID,
				Name:       volume.Name,
				Type:       "digitalocean_volume",
				Provider:   "digitalocean",
				Region:     volume.Region.Slug,
				State:      "active",
				Tags:       tags,
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.ListOptions.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Storage", ResourceType: "Volumes", Count: len(resources)}
	return resources
}

// discoverSnapshots discovers all snapshots
func (d *ComprehensiveDigitalOceanDiscoverer) discoverSnapshots(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		snapshots, resp, err := d.client.Snapshots.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list snapshots: %v", err)
			break
		}

		for _, snapshot := range snapshots {
			properties := make(map[string]interface{})
			properties["size_gb"] = snapshot.SizeGigaBytes
			properties["resource_type"] = snapshot.ResourceType
			properties["resource_id"] = snapshot.ResourceID
			properties["min_disk_size"] = snapshot.MinDiskSize

			tags := make(map[string]string)
			for _, tag := range snapshot.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         snapshot.ID,
				Name:       snapshot.Name,
				Type:       "digitalocean_snapshot",
				Provider:   "digitalocean",
				Region:     "global", // Snapshots are global
				State:      "active",
				Tags:       tags,
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Storage", ResourceType: "Snapshots", Count: len(resources)}
	return resources
}

// discoverFloatingIPs discovers all floating IPs
func (d *ComprehensiveDigitalOceanDiscoverer) discoverFloatingIPs(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		ips, resp, err := d.client.FloatingIPs.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list floating IPs: %v", err)
			break
		}

		for _, ip := range ips {
			properties := make(map[string]interface{})
			properties["ip"] = ip.IP

			if ip.Droplet != nil {
				properties["droplet_id"] = ip.Droplet.ID
			}

			resources = append(resources, models.Resource{
				ID:         ip.IP,
				Name:       ip.IP,
				Type:       "digitalocean_floating_ip",
				Provider:   "digitalocean",
				Region:     ip.Region.Slug,
				State:      "active",
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Network", ResourceType: "Floating IPs", Count: len(resources)}
	return resources
}

// discoverDomains discovers all domains
func (d *ComprehensiveDigitalOceanDiscoverer) discoverDomains(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		domains, resp, err := d.client.Domains.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list domains: %v", err)
			break
		}

		for _, domain := range domains {
			properties := make(map[string]interface{})
			properties["ttl"] = domain.TTL
			properties["zone_file"] = domain.ZoneFile

			resources = append(resources, models.Resource{
				ID:         domain.Name,
				Name:       domain.Name,
				Type:       "digitalocean_domain",
				Provider:   "digitalocean",
				Region:     "global",
				State:      "active",
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "DNS", ResourceType: "Domains", Count: len(resources)}
	return resources
}

// discoverFirewalls discovers all firewalls
func (d *ComprehensiveDigitalOceanDiscoverer) discoverFirewalls(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		firewalls, resp, err := d.client.Firewalls.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list firewalls: %v", err)
			break
		}

		for _, fw := range firewalls {
			properties := make(map[string]interface{})
			properties["status"] = fw.Status
			properties["inbound_rules"] = len(fw.InboundRules)
			properties["outbound_rules"] = len(fw.OutboundRules)
			properties["droplet_ids"] = fw.DropletIDs

			tags := make(map[string]string)
			for _, tag := range fw.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         fw.ID,
				Name:       fw.Name,
				Type:       "digitalocean_firewall",
				Provider:   "digitalocean",
				Region:     "global",
				State:      fw.Status,
				Tags:       tags,
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Security", ResourceType: "Firewalls", Count: len(resources)}
	return resources
}

// discoverVPCs discovers all VPCs
func (d *ComprehensiveDigitalOceanDiscoverer) discoverVPCs(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		vpcs, resp, err := d.client.VPCs.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list VPCs: %v", err)
			break
		}

		for _, vpc := range vpcs {
			properties := make(map[string]interface{})
			properties["ip_range"] = vpc.IPRange
			properties["description"] = vpc.Description
			properties["default"] = vpc.Default

			resources = append(resources, models.Resource{
				ID:         vpc.ID,
				Name:       vpc.Name,
				Type:       "digitalocean_vpc",
				Provider:   "digitalocean",
				Region:     vpc.RegionSlug,
				State:      "active",
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Network", ResourceType: "VPCs", Count: len(resources)}
	return resources
}

// discoverSSHKeys discovers all SSH keys
func (d *ComprehensiveDigitalOceanDiscoverer) discoverSSHKeys(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		keys, resp, err := d.client.Keys.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list SSH keys: %v", err)
			break
		}

		for _, key := range keys {
			properties := make(map[string]interface{})
			properties["fingerprint"] = key.Fingerprint
			properties["public_key"] = key.PublicKey

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("%d", key.ID),
				Name:       key.Name,
				Type:       "digitalocean_ssh_key",
				Provider:   "digitalocean",
				Region:     "global",
				State:      "active",
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Security", ResourceType: "SSH Keys", Count: len(resources)}
	return resources
}

// discoverProjects discovers all projects
func (d *ComprehensiveDigitalOceanDiscoverer) discoverProjects(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		projects, resp, err := d.client.Projects.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list projects: %v", err)
			break
		}

		for _, project := range projects {
			properties := make(map[string]interface{})
			properties["description"] = project.Description
			properties["purpose"] = project.Purpose
			properties["environment"] = project.Environment
			properties["is_default"] = project.IsDefault

			resources = append(resources, models.Resource{
				ID:         project.ID,
				Name:       project.Name,
				Type:       "digitalocean_project",
				Provider:   "digitalocean",
				Region:     "global",
				State:      "active",
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Organization", ResourceType: "Projects", Count: len(resources)}
	return resources
}

// discoverSpaces discovers all Spaces (S3-compatible object storage)
func (d *ComprehensiveDigitalOceanDiscoverer) discoverSpaces(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// Note: Spaces uses S3-compatible API, so this is a simplified version
	// In a full implementation, you'd use the S3 SDK with DigitalOcean endpoints

	d.progress <- DODiscoveryProgress{Service: "Storage", ResourceType: "Spaces", Count: len(resources)}
	return resources
}

// discoverAppPlatformApps discovers all App Platform apps
func (d *ComprehensiveDigitalOceanDiscoverer) discoverAppPlatformApps(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		apps, resp, err := d.client.Apps.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list apps: %v", err)
			break
		}

		for _, app := range apps {
			properties := make(map[string]interface{})
			if app.Spec != nil {
				properties["services"] = len(app.Spec.Services)
				properties["workers"] = len(app.Spec.Workers)
				properties["static_sites"] = len(app.Spec.StaticSites)
				properties["databases"] = len(app.Spec.Databases)
			}

			resources = append(resources, models.Resource{
				ID:         app.ID,
				Name:       app.Spec.Name,
				Type:       "digitalocean_app",
				Provider:   "digitalocean",
				Region:     app.Region.Slug,
				State:      "active",
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "Apps", ResourceType: "App Platform Apps", Count: len(resources)}
	return resources
}

// discoverCDNEndpoints discovers all CDN endpoints
func (d *ComprehensiveDigitalOceanDiscoverer) discoverCDNEndpoints(ctx context.Context) []models.Resource {
	var resources []models.Resource

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		endpoints, resp, err := d.client.CDNs.List(ctx, opt)
		if err != nil {
			log.Printf("Failed to list CDN endpoints: %v", err)
			break
		}

		for _, cdn := range endpoints {
			properties := make(map[string]interface{})
			properties["origin"] = cdn.Origin
			properties["endpoint"] = cdn.Endpoint
			properties["ttl"] = cdn.TTL
			properties["custom_domain"] = cdn.CustomDomain
			properties["certificate_id"] = cdn.CertificateID

			resources = append(resources, models.Resource{
				ID:         cdn.ID,
				Name:       cdn.Endpoint,
				Type:       "digitalocean_cdn",
				Provider:   "digitalocean",
				Region:     "global",
				State:      "active",
				Properties: properties,
			})
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	d.progress <- DODiscoveryProgress{Service: "CDN", ResourceType: "CDN Endpoints", Count: len(resources)}
	return resources
}

// discoverContainerRegistry discovers container registry repositories
func (d *ComprehensiveDigitalOceanDiscoverer) discoverContainerRegistry(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// Get registry information
	registry, _, err := d.client.Registry.Get(ctx)
	if err != nil {
		log.Printf("Failed to get container registry: %v", err)
		return resources
	}

	if registry != nil {
		properties := make(map[string]interface{})
		properties["storage_usage_bytes"] = registry.StorageUsageBytes
		properties["storage_usage_updated_at"] = registry.StorageUsageBytesUpdatedAt

		resources = append(resources, models.Resource{
			ID:         registry.Name,
			Name:       registry.Name,
			Type:       "digitalocean_container_registry",
			Provider:   "digitalocean",
			Region:     registry.Region,
			State:      "active",
			Properties: properties,
		})

		// List repositories
		opt := &godo.ListOptions{
			Page:    1,
			PerPage: 200,
		}

		for {
			repos, resp, err := d.client.Registry.ListRepositories(ctx, registry.Name, opt)
			if err != nil {
				log.Printf("Failed to list repositories: %v", err)
				break
			}

			for _, repo := range repos {
				repoProps := make(map[string]interface{})
				repoProps["tag_count"] = repo.TagCount
				repoProps["latest_tag"] = repo.LatestTag

				resources = append(resources, models.Resource{
					ID:         fmt.Sprintf("%s/%s", registry.Name, repo.Name),
					Name:       repo.Name,
					Type:       "digitalocean_container_repository",
					Provider:   "digitalocean",
					Region:     registry.Region,
					State:      "active",
					Properties: repoProps,
				})
			}

			if resp.Links == nil || resp.Links.IsLastPage() {
				break
			}
			opt.Page++
		}
	}

	d.progress <- DODiscoveryProgress{Service: "Registry", ResourceType: "Container Registry", Count: len(resources)}
	return resources
}

// discoverDatabaseReplicas discovers all database replicas
func (d *ComprehensiveDigitalOceanDiscoverer) discoverDatabaseReplicas(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// First get all databases
	databases, _, err := d.client.Databases.List(ctx, &godo.ListOptions{})
	if err != nil {
		log.Printf("Failed to list databases for replicas: %v", err)
		return resources
	}

	for _, db := range databases {
		// List replicas for each database
		replicas, _, err := d.client.Databases.ListReplicas(ctx, db.ID, &godo.ListOptions{})
		if err != nil {
			continue
		}

		for _, replica := range replicas {
			properties := make(map[string]interface{})
			properties["size"] = replica.Size
			properties["status"] = replica.Status
			properties["primary_db"] = db.ID

			if replica.Connection != nil {
				properties["host"] = replica.Connection.Host
				properties["port"] = replica.Connection.Port
			}

			tags := make(map[string]string)
			for _, tag := range replica.Tags {
				tags[tag] = "true"
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("%s-replica-%s", db.ID, replica.Name),
				Name:       replica.Name,
				Type:       "digitalocean_database_replica",
				Provider:   "digitalocean",
				Region:     replica.Region,
				State:      replica.Status,
				Tags:       tags,
				Properties: properties,
			})
		}
	}

	d.progress <- DODiscoveryProgress{Service: "Database", ResourceType: "Database Replicas", Count: len(resources)}
	return resources
}

// discoverDatabasePools discovers all database connection pools
func (d *ComprehensiveDigitalOceanDiscoverer) discoverDatabasePools(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// First get all databases
	databases, _, err := d.client.Databases.List(ctx, &godo.ListOptions{})
	if err != nil {
		log.Printf("Failed to list databases for pools: %v", err)
		return resources
	}

	for _, db := range databases {
		// List pools for each database
		pools, _, err := d.client.Databases.ListPools(ctx, db.ID, &godo.ListOptions{})
		if err != nil {
			continue
		}

		for _, pool := range pools {
			properties := make(map[string]interface{})
			properties["mode"] = pool.Mode
			properties["size"] = pool.Size
			properties["database"] = pool.Database
			properties["user"] = pool.User
			properties["primary_db"] = db.ID

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("%s-pool-%s", db.ID, pool.Name),
				Name:       pool.Name,
				Type:       "digitalocean_database_pool",
				Provider:   "digitalocean",
				Region:     db.RegionSlug,
				State:      "active",
				Properties: properties,
			})
		}
	}

	d.progress <- DODiscoveryProgress{Service: "Database", ResourceType: "Connection Pools", Count: len(resources)}
	return resources
}

// Helper functions
func (d *ComprehensiveDigitalOceanDiscoverer) reportProgress() {
	for progress := range d.progress {
		log.Printf("[DigitalOcean] %s: Discovered %d %s", progress.Service, progress.Count, progress.ResourceType)
	}
}

func getDigitalOceanToken() string {
	// Get token from environment
	if token := os.Getenv("DIGITALOCEAN_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("DO_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("DOCTL_TOKEN"); token != "" {
		return token
	}
	return ""
}
