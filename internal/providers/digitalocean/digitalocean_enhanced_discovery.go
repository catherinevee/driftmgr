package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// DigitalOceanEnhancedDiscoverer provides comprehensive DigitalOcean resource discovery
type DigitalOceanEnhancedDiscoverer struct {
	apiToken string
	cliPath  string
}

// NewDigitalOceanEnhancedDiscoverer creates a new DigitalOcean enhanced discoverer
func NewDigitalOceanEnhancedDiscoverer(apiToken string) (*DigitalOceanEnhancedDiscoverer, error) {
	cliPath, err := exec.LookPath("doctl")
	if err != nil {
		return nil, fmt.Errorf("doctl CLI not found: %w", err)
	}

	if apiToken == "" {
		// Try environment variable
		apiToken = os.Getenv("DIGITALOCEAN_TOKEN")
		if apiToken == "" {
			// Try credentials file
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
	}

	if apiToken == "" {
		return nil, fmt.Errorf("DigitalOcean API token not found")
	}

	return &DigitalOceanEnhancedDiscoverer{
		apiToken: apiToken,
		cliPath:  cliPath,
	}, nil
}

// DiscoverAllDigitalOceanResources discovers all DigitalOcean resources comprehensively
func (d *DigitalOceanEnhancedDiscoverer) DiscoverAllDigitalOceanResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var allResources []models.Resource

	// If no regions specified, use all DigitalOcean regions
	if len(regions) == 0 {
		regions = []string{
			"nyc1", "nyc2", "nyc3", "ams2", "ams3", "sfo1", "sfo2", "sfo3",
			"sgp1", "lon1", "fra1", "tor1", "blr1", "syd1",
		}
	}

	// Discover different resource categories
	log.Printf("Discovering DigitalOcean Compute resources...")
	computeResources, _ := d.discoverComputeResources(ctx, regions)
	allResources = append(allResources, computeResources...)

	log.Printf("Discovering DigitalOcean Storage resources...")
	storageResources, _ := d.discoverStorageResources(ctx, regions)
	allResources = append(allResources, storageResources...)

	log.Printf("Discovering DigitalOcean Database resources...")
	dbResources, _ := d.discoverDatabaseResources(ctx, regions)
	allResources = append(allResources, dbResources...)

	log.Printf("Discovering DigitalOcean Networking resources...")
	networkResources, _ := d.discoverNetworkingResources(ctx, regions)
	allResources = append(allResources, networkResources...)

	log.Printf("Discovering DigitalOcean Container resources...")
	containerResources, _ := d.discoverContainerResources(ctx, regions)
	allResources = append(allResources, containerResources...)

	log.Printf("Discovering DigitalOcean DNS resources...")
	dnsResources, _ := d.discoverDNSResources(ctx, regions)
	allResources = append(allResources, dnsResources...)

	log.Printf("Discovering DigitalOcean Security resources...")
	securityResources, _ := d.discoverSecurityResources(ctx, regions)
	allResources = append(allResources, securityResources...)

	log.Printf("Discovering DigitalOcean App Platform resources...")
	appResources, _ := d.discoverAppPlatformResources(ctx, regions)
	allResources = append(allResources, appResources...)

	log.Printf("Discovering DigitalOcean Monitoring resources...")
	monitoringResources, _ := d.discoverMonitoringResources(ctx, regions)
	allResources = append(allResources, monitoringResources...)

	log.Printf("DigitalOcean enhanced discovery completed: %d resources found", len(allResources))
	return allResources, nil
}

// discoverComputeResources discovers compute resources
func (d *DigitalOceanEnhancedDiscoverer) discoverComputeResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Droplets
	droplets, _ := d.executeCommand(ctx, []string{"compute", "droplet", "list", "--format", "json"})
	for _, droplet := range droplets {
		region := getStringField(droplet, "region.slug")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       fmt.Sprintf("%v", droplet["id"]),
				Name:     getStringField(droplet, "name"),
				Type:     "digitalocean_droplet",
				Provider: "digitalocean",
				Region:   region,
				Status:   getStringField(droplet, "status"),
				Tags:     convertTagsToMap(droplet["tags"]),
				Attributes: map[string]interface{}{
					"memory":    droplet["memory"],
					"vcpus":     droplet["vcpus"],
					"disk":      droplet["disk"],
					"image":     getNestedField(droplet, "image.slug"),
					"size_slug": getNestedField(droplet, "size_slug"),
					"vpc_uuid":  droplet["vpc_uuid"],
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(droplet, "created_at")),
			})
		}
	}

	// Droplet sizes
	sizes, _ := d.executeCommand(ctx, []string{"compute", "size", "list", "--format", "json"})
	for _, size := range sizes {
		if len(size["regions"].([]interface{})) > 0 {
			for _, regionInterface := range size["regions"].([]interface{}) {
				region := regionInterface.(string)
				if d.matchesRegions(region, regions) {
					resources = append(resources, models.Resource{
						ID:       getStringField(size, "slug"),
						Name:     getStringField(size, "slug"),
						Type:     "digitalocean_size",
						Provider: "digitalocean",
						Region:   region,
						Status:   "available",
						Tags:     make(map[string]string),
						Attributes: map[string]interface{}{
							"memory":        size["memory"],
							"vcpus":         size["vcpus"],
							"disk":          size["disk"],
							"price_monthly": size["price_monthly"],
							"price_hourly":  size["price_hourly"],
							"transfer":      size["transfer"],
							"available":     size["available"],
						},
						CreatedAt: time.Now(),
					})
					break // Only add once per size
				}
			}
		}
	}

	// Images
	images, _ := d.executeCommand(ctx, []string{"compute", "image", "list", "--format", "json", "--public"})
	for _, image := range images {
		if len(image["regions"].([]interface{})) > 0 {
			for _, regionInterface := range image["regions"].([]interface{}) {
				region := regionInterface.(string)
				if d.matchesRegions(region, regions) {
					resources = append(resources, models.Resource{
						ID:       fmt.Sprintf("%v", image["id"]),
						Name:     getStringField(image, "name"),
						Type:     "digitalocean_image",
						Provider: "digitalocean",
						Region:   region,
						Status:   getStringField(image, "status"),
						Tags:     make(map[string]string),
						Attributes: map[string]interface{}{
							"distribution":   getStringField(image, "distribution"),
							"slug":           getStringField(image, "slug"),
							"public":         image["public"],
							"min_disk_size":  image["min_disk_size"],
							"size_gigabytes": image["size_gigabytes"],
						},
						CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(image, "created_at")),
					})
					break
				}
			}
		}
	}

	// SSH Keys
	keys, _ := d.executeCommand(ctx, []string{"compute", "ssh-key", "list", "--format", "json"})
	for _, key := range keys {
		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%v", key["id"]),
			Name:     getStringField(key, "name"),
			Type:     "digitalocean_ssh_key",
			Provider: "digitalocean",
			Region:   "global",
			Status:   "active",
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"fingerprint": getStringField(key, "fingerprint"),
				"public_key":  getStringField(key, "public_key"),
			},
			CreatedAt: time.Now(),
		})
	}

	return resources, nil
}

// discoverStorageResources discovers storage resources
func (d *DigitalOceanEnhancedDiscoverer) discoverStorageResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Volumes
	volumes, _ := d.executeCommand(ctx, []string{"compute", "volume", "list", "--format", "json"})
	for _, volume := range volumes {
		region := getNestedField(volume, "region.slug")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(volume, "id"),
				Name:     getStringField(volume, "name"),
				Type:     "digitalocean_volume",
				Provider: "digitalocean",
				Region:   region,
				Status:   "available",
				Tags:     convertTagsToMap(volume["tags"]),
				Attributes: map[string]interface{}{
					"size_gigabytes":   volume["size_gigabytes"],
					"filesystem_type":  getStringField(volume, "filesystem_type"),
					"filesystem_label": getStringField(volume, "filesystem_label"),
					"droplet_ids":      volume["droplet_ids"],
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(volume, "created_at")),
			})
		}
	}

	// Snapshots
	snapshots, _ := d.executeCommand(ctx, []string{"compute", "snapshot", "list", "--format", "json"})
	for _, snapshot := range snapshots {
		regions_list := snapshot["regions"].([]interface{})
		for _, regionInterface := range regions_list {
			region := regionInterface.(string)
			if d.matchesRegions(region, regions) {
				resources = append(resources, models.Resource{
					ID:       getStringField(snapshot, "id"),
					Name:     getStringField(snapshot, "name"),
					Type:     "digitalocean_snapshot",
					Provider: "digitalocean",
					Region:   region,
					Status:   "completed",
					Tags:     convertTagsToMap(snapshot["tags"]),
					Attributes: map[string]interface{}{
						"resource_id":    getStringField(snapshot, "resource_id"),
						"resource_type":  getStringField(snapshot, "resource_type"),
						"min_disk_size":  snapshot["min_disk_size"],
						"size_gigabytes": snapshot["size_gigabytes"],
					},
					CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(snapshot, "created_at")),
				})
				break
			}
		}
	}

	// Spaces (Object Storage)
	spaces, _ := d.executeCommand(ctx, []string{"spaces", "list", "--format", "json"})
	for _, space := range spaces {
		region := getStringField(space, "region")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(space, "name"),
				Name:     getStringField(space, "name"),
				Type:     "digitalocean_spaces_bucket",
				Provider: "digitalocean",
				Region:   region,
				Status:   "active",
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"endpoint": fmt.Sprintf("%s.%s.digitaloceanspaces.com", getStringField(space, "name"), region),
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(space, "created_at")),
			})
		}
	}

	return resources, nil
}

// discoverDatabaseResources discovers database resources
func (d *DigitalOceanEnhancedDiscoverer) discoverDatabaseResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Database clusters
	databases, _ := d.executeCommand(ctx, []string{"databases", "list", "--format", "json"})
	for _, db := range databases {
		region := getStringField(db, "region")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(db, "id"),
				Name:     getStringField(db, "name"),
				Type:     "digitalocean_database_cluster",
				Provider: "digitalocean",
				Region:   region,
				Status:   getStringField(db, "status"),
				Tags:     convertTagsToMap(db["tags"]),
				Attributes: map[string]interface{}{
					"engine":               getStringField(db, "engine"),
					"version":              getStringField(db, "version"),
					"num_nodes":            db["num_nodes"],
					"size":                 getStringField(db, "size"),
					"private_network_uuid": getStringField(db, "private_network_uuid"),
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(db, "created_at")),
			})
		}
	}

	// Database connection pools
	for _, db := range databases {
		dbID := getStringField(db, "id")
		pools, _ := d.executeCommand(ctx, []string{"databases", "pool", "list", dbID, "--format", "json"})
		for _, pool := range pools {
			resources = append(resources, models.Resource{
				ID:       getStringField(pool, "name"),
				Name:     getStringField(pool, "name"),
				Type:     "digitalocean_database_connection_pool",
				Provider: "digitalocean",
				Region:   getStringField(db, "region"),
				Status:   "active",
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"database_id": dbID,
					"mode":        getStringField(pool, "mode"),
					"size":        pool["size"],
					"db":          getStringField(pool, "db"),
					"user":        getStringField(pool, "user"),
				},
				CreatedAt: time.Now(),
			})
		}
	}

	// Database users
	for _, db := range databases {
		dbID := getStringField(db, "id")
		users, _ := d.executeCommand(ctx, []string{"databases", "user", "list", dbID, "--format", "json"})
		for _, user := range users {
			resources = append(resources, models.Resource{
				ID:       fmt.Sprintf("%s-%s", dbID, getStringField(user, "name")),
				Name:     getStringField(user, "name"),
				Type:     "digitalocean_database_user",
				Provider: "digitalocean",
				Region:   getStringField(db, "region"),
				Status:   "active",
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"database_id": dbID,
					"role":        getStringField(user, "role"),
				},
				CreatedAt: time.Now(),
			})
		}
	}

	return resources, nil
}

// discoverNetworkingResources discovers networking resources
func (d *DigitalOceanEnhancedDiscoverer) discoverNetworkingResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// VPCs
	vpcs, _ := d.executeCommand(ctx, []string{"compute", "vpc", "list", "--format", "json"})
	for _, vpc := range vpcs {
		region := getStringField(vpc, "region")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(vpc, "id"),
				Name:     getStringField(vpc, "name"),
				Type:     "digitalocean_vpc",
				Provider: "digitalocean",
				Region:   region,
				Status:   "available",
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"ip_range":    getStringField(vpc, "ip_range"),
					"default":     vpc["default"],
					"description": getStringField(vpc, "description"),
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(vpc, "created_at")),
			})
		}
	}

	// Load Balancers
	lbs, _ := d.executeCommand(ctx, []string{"compute", "load-balancer", "list", "--format", "json"})
	for _, lb := range lbs {
		region := getNestedField(lb, "region.slug")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(lb, "id"),
				Name:     getStringField(lb, "name"),
				Type:     "digitalocean_loadbalancer",
				Provider: "digitalocean",
				Region:   region,
				Status:   getStringField(lb, "status"),
				Tags:     convertTagsToMap(lb["tags"]),
				Attributes: map[string]interface{}{
					"ip":                     getStringField(lb, "ip"),
					"algorithm":              getStringField(lb, "algorithm"),
					"vpc_uuid":               getStringField(lb, "vpc_uuid"),
					"redirect_http_to_https": lb["redirect_http_to_https"],
					"enable_proxy_protocol":  lb["enable_proxy_protocol"],
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(lb, "created_at")),
			})
		}
	}

	// Firewalls
	firewalls, _ := d.executeCommand(ctx, []string{"compute", "firewall", "list", "--format", "json"})
	for _, firewall := range firewalls {
		resources = append(resources, models.Resource{
			ID:       getStringField(firewall, "id"),
			Name:     getStringField(firewall, "name"),
			Type:     "digitalocean_firewall",
			Provider: "digitalocean",
			Region:   "global",
			Status:   getStringField(firewall, "status"),
			Tags:     convertTagsToMap(firewall["tags"]),
			Attributes: map[string]interface{}{
				"inbound_rules":  firewall["inbound_rules"],
				"outbound_rules": firewall["outbound_rules"],
				"droplet_ids":    firewall["droplet_ids"],
			},
			CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(firewall, "created_at")),
		})
	}

	// Reserved IPs
	ips, _ := d.executeCommand(ctx, []string{"compute", "reserved-ip", "list", "--format", "json"})
	for _, ip := range ips {
		region := getNestedField(ip, "region.slug")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(ip, "ip"),
				Name:     getStringField(ip, "ip"),
				Type:     "digitalocean_reserved_ip",
				Provider: "digitalocean",
				Region:   region,
				Status:   "assigned",
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"type":    getStringField(ip, "type"),
					"droplet": ip["droplet"],
				},
				CreatedAt: time.Now(),
			})
		}
	}

	return resources, nil
}

// discoverContainerResources discovers Kubernetes and container resources
func (d *DigitalOceanEnhancedDiscoverer) discoverContainerResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Kubernetes clusters
	clusters, _ := d.executeCommand(ctx, []string{"kubernetes", "cluster", "list", "--format", "json"})
	for _, cluster := range clusters {
		region := getNestedField(cluster, "region.slug")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(cluster, "id"),
				Name:     getStringField(cluster, "name"),
				Type:     "digitalocean_kubernetes_cluster",
				Provider: "digitalocean",
				Region:   region,
				Status:   getNestedField(cluster, "status.state"),
				Tags:     convertTagsToMap(cluster["tags"]),
				Attributes: map[string]interface{}{
					"version":        getStringField(cluster, "version"),
					"cluster_subnet": getStringField(cluster, "cluster_subnet"),
					"service_subnet": getStringField(cluster, "service_subnet"),
					"vpc_uuid":       getStringField(cluster, "vpc_uuid"),
					"ipv4":           getStringField(cluster, "ipv4"),
					"endpoint":       getStringField(cluster, "endpoint"),
					"auto_upgrade":   cluster["auto_upgrade"],
					"surge_upgrade":  cluster["surge_upgrade"],
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(cluster, "created_at")),
			})

			// Node pools for this cluster
			clusterID := getStringField(cluster, "id")
			pools, _ := d.executeCommand(ctx, []string{"kubernetes", "cluster", "node-pool", "list", clusterID, "--format", "json"})
			for _, pool := range pools {
				resources = append(resources, models.Resource{
					ID:       getStringField(pool, "id"),
					Name:     getStringField(pool, "name"),
					Type:     "digitalocean_kubernetes_node_pool",
					Provider: "digitalocean",
					Region:   region,
					Status:   "active",
					Tags:     convertTagsToMap(pool["tags"]),
					Attributes: map[string]interface{}{
						"cluster_id": clusterID,
						"size":       getStringField(pool, "size"),
						"count":      pool["count"],
						"auto_scale": pool["auto_scale"],
						"min_nodes":  pool["min_nodes"],
						"max_nodes":  pool["max_nodes"],
					},
					CreatedAt: time.Now(),
				})
			}
		}
	}

	// Container Registry
	registries, _ := d.executeCommand(ctx, []string{"registry", "list", "--format", "json"})
	for _, registry := range registries {
		resources = append(resources, models.Resource{
			ID:       getStringField(registry, "name"),
			Name:     getStringField(registry, "name"),
			Type:     "digitalocean_container_registry",
			Provider: "digitalocean",
			Region:   "global",
			Status:   "active",
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"created_at": getStringField(registry, "created_at"),
			},
			CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(registry, "created_at")),
		})
	}

	return resources, nil
}

// discoverDNSResources discovers DNS resources
func (d *DigitalOceanEnhancedDiscoverer) discoverDNSResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Domains
	domains, _ := d.executeCommand(ctx, []string{"domains", "list", "--format", "json"})
	for _, domain := range domains {
		resources = append(resources, models.Resource{
			ID:       getStringField(domain, "name"),
			Name:     getStringField(domain, "name"),
			Type:     "digitalocean_domain",
			Provider: "digitalocean",
			Region:   "global",
			Status:   "active",
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"ttl":       domain["ttl"],
				"zone_file": getStringField(domain, "zone_file"),
			},
			CreatedAt: time.Now(),
		})

		// DNS records for this domain
		domainName := getStringField(domain, "name")
		records, _ := d.executeCommand(ctx, []string{"domains", "records", "list", domainName, "--format", "json"})
		for _, record := range records {
			resources = append(resources, models.Resource{
				ID:       fmt.Sprintf("%v", record["id"]),
				Name:     fmt.Sprintf("%s.%s", getStringField(record, "name"), domainName),
				Type:     "digitalocean_record",
				Provider: "digitalocean",
				Region:   "global",
				Status:   "active",
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"domain":   domainName,
					"type":     getStringField(record, "type"),
					"data":     getStringField(record, "data"),
					"priority": record["priority"],
					"port":     record["port"],
					"ttl":      record["ttl"],
					"weight":   record["weight"],
					"flags":    record["flags"],
					"tag":      getStringField(record, "tag"),
				},
				CreatedAt: time.Now(),
			})
		}
	}

	return resources, nil
}

// discoverSecurityResources discovers security resources
func (d *DigitalOceanEnhancedDiscoverer) discoverSecurityResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Certificates
	certs, _ := d.executeCommand(ctx, []string{"certificates", "list", "--format", "json"})
	for _, cert := range certs {
		resources = append(resources, models.Resource{
			ID:       getStringField(cert, "id"),
			Name:     getStringField(cert, "name"),
			Type:     "digitalocean_certificate",
			Provider: "digitalocean",
			Region:   "global",
			Status:   getStringField(cert, "state"),
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"type":             getStringField(cert, "type"),
				"not_after":        getStringField(cert, "not_after"),
				"sha1_fingerprint": getStringField(cert, "sha1_fingerprint"),
				"dns_names":        cert["dns_names"],
			},
			CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(cert, "created_at")),
		})
	}

	return resources, nil
}

// discoverAppPlatformResources discovers App Platform resources
func (d *DigitalOceanEnhancedDiscoverer) discoverAppPlatformResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Apps
	apps, _ := d.executeCommand(ctx, []string{"apps", "list", "--format", "json"})
	for _, app := range apps {
		region := getNestedField(app, "spec.region")
		if d.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringField(app, "id"),
				Name:     getNestedField(app, "spec.name"),
				Type:     "digitalocean_app",
				Provider: "digitalocean",
				Region:   region,
				Status:   getNestedField(app, "last_deployment_created_at"),
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"live_url":               getStringField(app, "live_url"),
					"in_progress_deployment": app["in_progress_deployment"],
					"created_at":             getStringField(app, "created_at"),
					"updated_at":             getStringField(app, "updated_at"),
				},
				CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(app, "created_at")),
			})
		}
	}

	return resources, nil
}

// discoverMonitoringResources discovers monitoring resources
func (d *DigitalOceanEnhancedDiscoverer) discoverMonitoringResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Projects
	projects, _ := d.executeCommand(ctx, []string{"projects", "list", "--format", "json"})
	for _, project := range projects {
		resources = append(resources, models.Resource{
			ID:       getStringField(project, "id"),
			Name:     getStringField(project, "name"),
			Type:     "digitalocean_project",
			Provider: "digitalocean",
			Region:   "global",
			Status:   "active",
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"description": getStringField(project, "description"),
				"purpose":     getStringField(project, "purpose"),
				"environment": getStringField(project, "environment"),
				"is_default":  project["is_default"],
			},
			CreatedAt: parseDigitalOceanTimeEnhanced(getStringField(project, "created_at")),
		})
	}

	// Monitoring alerts
	alerts, _ := d.executeCommand(ctx, []string{"monitoring", "alert", "list", "--format", "json"})
	for _, alert := range alerts {
		resources = append(resources, models.Resource{
			ID:       getStringField(alert, "uuid"),
			Name:     getStringField(alert, "description"),
			Type:     "digitalocean_monitor_alert",
			Provider: "digitalocean",
			Region:   "global",
			Status:   "active",
			Tags:     convertTagsToMap(alert["tags"]),
			Attributes: map[string]interface{}{
				"type":     getStringField(alert, "type"),
				"compare":  getStringField(alert, "compare"),
				"value":    alert["value"],
				"window":   getStringField(alert, "window"),
				"entities": alert["entities"],
			},
			CreatedAt: time.Now(),
		})
	}

	return resources, nil
}

// Helper functions

func (d *DigitalOceanEnhancedDiscoverer) executeCommand(ctx context.Context, args []string) ([]map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, d.cliPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to execute doctl command: %v", err)
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		log.Printf("Warning: Failed to parse doctl output: %v", err)
		return nil, err
	}

	return result, nil
}

func (d *DigitalOceanEnhancedDiscoverer) matchesRegions(resourceRegion string, targetRegions []string) bool {
	if len(targetRegions) == 0 {
		return true
	}

	for _, region := range targetRegions {
		if strings.EqualFold(resourceRegion, region) {
			return true
		}
	}
	return false
}

func getStringField(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getNestedField(data map[string]interface{}, path string) string {
	keys := strings.Split(path, ".")
	current := data

	for i, key := range keys {
		if val, ok := current[key]; ok {
			if i == len(keys)-1 {
				if str, ok := val.(string); ok {
					return str
				}
			} else if next, ok := val.(map[string]interface{}); ok {
				current = next
			} else {
				return ""
			}
		} else {
			return ""
		}
	}
	return ""
}

func convertTagsToMap(tags interface{}) map[string]string {
	result := make(map[string]string)
	if tagsList, ok := tags.([]interface{}); ok {
		for i, tag := range tagsList {
			if tagStr, ok := tag.(string); ok {
				result[fmt.Sprintf("tag_%d", i)] = tagStr
			}
		}
	}
	return result
}

func parseDigitalOceanTimeEnhanced(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// DigitalOcean uses RFC3339 format
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t
	}

	// Alternative format
	if t, err := time.Parse("2006-01-02T15:04:05Z", timeStr); err == nil {
		return t
	}

	return time.Now()
}
