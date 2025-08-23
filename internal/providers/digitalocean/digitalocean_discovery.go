package discovery

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// DigitalOceanDiscoverer provides DigitalOcean resource discovery capabilities
type DigitalOceanDiscoverer struct {
	apiToken string
	region   string
}

// NewDigitalOceanDiscoverer creates a new DigitalOcean discoverer
func NewDigitalOceanDiscoverer(apiToken, region string) *DigitalOceanDiscoverer {
	return &DigitalOceanDiscoverer{
		apiToken: apiToken,
		region:   region,
	}
}

// DiscoverDigitalOceanResources discovers all DigitalOcean resources
func DiscoverDigitalOceanResources(regions []string, provider string) []models.Resource {
	var resources []models.Resource

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
		log.Printf("DigitalOcean API token not found. Set DIGITALOCEAN_TOKEN environment variable or configure credentials.")
		return resources
	}

	// If no regions specified, use default regions
	if len(regions) == 0 {
		regions = []string{"nyc1", "sfo2", "lon1", "fra1", "sgp1", "tor1", "ams3", "blr1"}
	}

	for _, region := range regions {
		log.Printf("Scanning DigitalOcean region: %s", region)

		discoverer := NewDigitalOceanDiscoverer(apiToken, region)

		// Discover different resource types
		resources = append(resources, discoverer.discoverDroplets(provider)...)
		resources = append(resources, discoverer.discoverLoadBalancers(provider)...)
		resources = append(resources, discoverer.discoverDatabases(provider)...)
		resources = append(resources, discoverer.discoverKubernetesClusters(provider)...)
		resources = append(resources, discoverer.discoverSpaces(provider)...)
		resources = append(resources, discoverer.discoverVolumes(provider)...)
		resources = append(resources, discoverer.discoverSnapshots(provider)...)
		resources = append(resources, discoverer.discoverNetworks(provider)...)
		resources = append(resources, discoverer.discoverFirewalls(provider)...)
		resources = append(resources, discoverer.discoverDomains(provider)...)
		resources = append(resources, discoverer.discoverCertificates(provider)...)
		resources = append(resources, discoverer.discoverProjects(provider)...)
	}

	return resources
}

// discoverDroplets discovers DigitalOcean droplets (VMs)
func (d *DigitalOceanDiscoverer) discoverDroplets(provider string) []models.Resource {
	var resources []models.Resource

	// Use doctl CLI to list droplets
	cmd := exec.Command("doctl", "compute", "droplet", "list", "--format", "ID,Name,Region,Size,Status,Image,Memory,VCPUs,Disk,Tags,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover droplets: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		// Parse droplet information
		id := fields[0]
		name := fields[1]
		region := fields[2]
		size := fields[3]
		status := fields[4]
		image := fields[5]
		memory := fields[6]
		vcpus := fields[7]
		disk := fields[8]
		tags := fields[9]
		created := fields[10]

		// Only include droplets from the specified region
		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_droplet",
			Provider:     provider,
			Region:       region,
			Status:       status,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"size":   size,
				"image":  image,
				"memory": memory,
				"vcpus":  vcpus,
				"disk":   disk,
				"tags":   tags,
			},
			Attributes: map[string]interface{}{
				"size":   size,
				"image":  image,
				"memory": memory,
				"vcpus":  vcpus,
				"disk":   disk,
				"tags":   tags,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverLoadBalancers discovers DigitalOcean load balancers
func (d *DigitalOceanDiscoverer) discoverLoadBalancers(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "compute", "load-balancer", "list", "--format", "ID,Name,Region,Status,Algorithm,RedirectHttpToHttps,StickySessions,HealthCheck,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover load balancers: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		id := fields[0]
		name := fields[1]
		region := fields[2]
		status := fields[3]
		algorithm := fields[4]
		redirectHttps := fields[5]
		stickySessions := fields[6]
		healthCheck := fields[7]
		created := fields[8]

		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_loadbalancer",
			Provider:     provider,
			Region:       region,
			Status:       status,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"algorithm":       algorithm,
				"redirect_https":  redirectHttps,
				"sticky_sessions": stickySessions,
				"health_check":    healthCheck,
			},
			Attributes: map[string]interface{}{
				"algorithm":       algorithm,
				"redirect_https":  redirectHttps,
				"sticky_sessions": stickySessions,
				"health_check":    healthCheck,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverDatabases discovers DigitalOcean managed databases
func (d *DigitalOceanDiscoverer) discoverDatabases(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "databases", "list", "--format", "ID,Name,Engine,Version,Region,Status,Size,NodeCount,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover databases: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		id := fields[0]
		name := fields[1]
		engine := fields[2]
		version := fields[3]
		region := fields[4]
		status := fields[5]
		size := fields[6]
		nodeCount := fields[7]
		created := fields[8]

		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_database_cluster",
			Provider:     provider,
			Region:       region,
			Status:       status,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"engine":     engine,
				"version":    version,
				"size":       size,
				"node_count": nodeCount,
			},
			Attributes: map[string]interface{}{
				"engine":     engine,
				"version":    version,
				"size":       size,
				"node_count": nodeCount,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverKubernetesClusters discovers DigitalOcean Kubernetes clusters
func (d *DigitalOceanDiscoverer) discoverKubernetesClusters(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "kubernetes", "cluster", "list", "--format", "ID,Name,Region,Version,Status,NodePools,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover Kubernetes clusters: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		id := fields[0]
		name := fields[1]
		region := fields[2]
		version := fields[3]
		status := fields[4]
		nodePools := fields[5]
		created := fields[6]

		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_kubernetes_cluster",
			Provider:     provider,
			Region:       region,
			Status:       status,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"version":    version,
				"node_pools": nodePools,
			},
			Attributes: map[string]interface{}{
				"version":    version,
				"node_pools": nodePools,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverSpaces discovers DigitalOcean Spaces (object storage)
func (d *DigitalOceanDiscoverer) discoverSpaces(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "spaces", "list", "--format", "Name,Region,Size,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover Spaces: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		region := fields[1]
		size := fields[2]
		created := fields[3]

		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           name, // Spaces use name as ID
			Name:         name,
			Type:         "digitalocean_spaces_bucket",
			Provider:     provider,
			Region:       region,
			Status:       "active",
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"size": size,
			},
			Attributes: map[string]interface{}{
				"size": size,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverVolumes discovers DigitalOcean block storage volumes
func (d *DigitalOceanDiscoverer) discoverVolumes(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "compute", "volume", "list", "--format", "ID,Name,Region,Size,Status,DropletIDs,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover volumes: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		id := fields[0]
		name := fields[1]
		region := fields[2]
		size := fields[3]
		status := fields[4]
		dropletIDs := fields[5]
		created := fields[6]

		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_volume",
			Provider:     provider,
			Region:       region,
			Status:       status,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"size":        size,
				"droplet_ids": dropletIDs,
			},
			Attributes: map[string]interface{}{
				"size":        size,
				"droplet_ids": dropletIDs,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverSnapshots discovers DigitalOcean snapshots
func (d *DigitalOceanDiscoverer) discoverSnapshots(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "compute", "snapshot", "list", "--format", "ID,Name,Region,Size,Status,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover snapshots: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		id := fields[0]
		name := fields[1]
		region := fields[2]
		size := fields[3]
		status := fields[4]
		created := fields[5]

		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_snapshot",
			Provider:     provider,
			Region:       region,
			Status:       status,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"size": size,
			},
			Attributes: map[string]interface{}{
				"size": size,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverNetworks discovers DigitalOcean VPC networks
func (d *DigitalOceanDiscoverer) discoverNetworks(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "compute", "vpc", "list", "--format", "ID,Name,Region,IPRange,Default,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover VPCs: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		id := fields[0]
		name := fields[1]
		region := fields[2]
		ipRange := fields[3]
		isDefault := fields[4]
		created := fields[5]

		if region != d.region {
			continue
		}

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_vpc",
			Provider:     provider,
			Region:       region,
			Status:       "active",
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"ip_range":   ipRange,
				"is_default": isDefault,
			},
			Attributes: map[string]interface{}{
				"ip_range":   ipRange,
				"is_default": isDefault,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverFirewalls discovers DigitalOcean firewalls
func (d *DigitalOceanDiscoverer) discoverFirewalls(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "compute", "firewall", "list", "--format", "ID,Name,Status,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover firewalls: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		id := fields[0]
		name := fields[1]
		status := fields[2]
		created := fields[3]

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_firewall",
			Provider:     provider,
			Region:       d.region,
			Status:       status,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags:         map[string]string{},
			Attributes:   map[string]interface{}{},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverDomains discovers DigitalOcean domains
func (d *DigitalOceanDiscoverer) discoverDomains(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "domains", "list", "--format", "Name,TTL,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover domains: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		ttl := fields[1]
		created := fields[2]

		resource := models.Resource{
			ID:           name, // Domains use name as ID
			Name:         name,
			Type:         "digitalocean_domain",
			Provider:     provider,
			Region:       "global", // Domains are global
			Status:       "active",
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"ttl": ttl,
			},
			Attributes: map[string]interface{}{
				"ttl": ttl,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverCertificates discovers DigitalOcean SSL certificates
func (d *DigitalOceanDiscoverer) discoverCertificates(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "certificates", "list", "--format", "ID,Name,Type,State,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover certificates: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		id := fields[0]
		name := fields[1]
		certType := fields[2]
		state := fields[3]
		created := fields[4]

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_certificate",
			Provider:     provider,
			Region:       "global", // Certificates are global
			Status:       state,
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"type": certType,
			},
			Attributes: map[string]interface{}{
				"type": certType,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// discoverProjects discovers DigitalOcean projects
func (d *DigitalOceanDiscoverer) discoverProjects(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("doctl", "projects", "list", "--format", "ID,Name,Description,Environment,Created", "--no-header")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIGITALOCEAN_TOKEN=%s", d.apiToken))

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to discover projects: %v", err)
		return resources
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		id := fields[0]
		name := fields[1]
		description := fields[2]
		environment := fields[3]
		created := fields[4]

		resource := models.Resource{
			ID:           id,
			Name:         name,
			Type:         "digitalocean_project",
			Provider:     provider,
			Region:       "global", // Projects are global
			Status:       "active",
			CreatedAt:    parseDigitalOceanTime(created),
			LastModified: time.Now(),
			Tags: map[string]string{
				"description": description,
				"environment": environment,
			},
			Attributes: map[string]interface{}{
				"description": description,
				"environment": environment,
			},
		}

		resources = append(resources, resource)
	}

	return resources
}

// parseDigitalOceanTime parses DigitalOcean time format
func parseDigitalOceanTime(timeStr string) time.Time {
	// DigitalOcean uses format like "2023-01-01 12:00:00 +0000 UTC"
	// Try to parse it
	if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", timeStr); err == nil {
		return t
	}

	// Fallback to current time if parsing fails
	return time.Now()
}
