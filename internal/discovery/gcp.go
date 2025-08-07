package discovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"github.com/catherinevee/driftmgr/internal/models"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCPProvider implements the Provider interface for Google Cloud Platform
type GCPProvider struct {
	ctx       context.Context
	projectID string
}

// NewGCPProvider creates a new GCP provider
func NewGCPProvider() (*GCPProvider, error) {
	ctx := context.Background()

	// In a real implementation, project ID would come from:
	// 1. Configuration file
	// 2. Environment variable GOOGLE_CLOUD_PROJECT
	// 3. Application default credentials
	// 4. gcloud config

	projectID := getGCPProjectID()
	if projectID == "" {
		// Return provider that will use mock data
		return &GCPProvider{
			ctx:       ctx,
			projectID: "",
		}, nil
	}

	return &GCPProvider{
		ctx:       ctx,
		projectID: projectID,
	}, nil
}

// Name returns the provider name
func (p *GCPProvider) Name() string {
	return "Google Cloud Platform"
}

// SupportedRegions returns the list of supported GCP regions
func (p *GCPProvider) SupportedRegions() []string {
	return []string{
		"us-central1", "us-east1", "us-east4", "us-west1", "us-west2", "us-west3", "us-west4",
		"europe-north1", "europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6",
		"asia-east1", "asia-east2", "asia-northeast1", "asia-northeast2", "asia-northeast3",
		"asia-south1", "asia-southeast1", "asia-southeast2",
		"australia-southeast1", "northamerica-northeast1", "southamerica-east1",
	}
}

// SupportedResourceTypes returns the list of supported GCP resource types
func (p *GCPProvider) SupportedResourceTypes() []string {
	return []string{
		"google_compute_instance",
		"google_compute_disk",
		"google_compute_network",
		"google_compute_subnetwork",
		"google_compute_firewall",
		"google_storage_bucket",
		"google_sql_database_instance",
		"google_sql_database",
		"google_container_cluster",
		"google_compute_address",
		"google_compute_global_address",
		"google_project_service",
		"google_service_account",
		"google_project_iam_member",
	}
}

// Discover discovers GCP resources
func (p *GCPProvider) Discover(config Config) ([]models.Resource, error) {
	fmt.Println("  [GCP] Discovering resources using Google Cloud SDK...")

	if p.projectID == "" {
		fmt.Println("  [GCP] Warning: No project ID configured, using mock data")
		return p.getMockResources(config), nil
	}

	var allResources []models.Resource

	// If specific regions are requested, use them
	regions := config.Regions
	if len(regions) == 0 {
		regions = []string{"us-central1"} // Default region
	}

	for _, region := range regions {
		fmt.Printf("  [GCP] Scanning region: %s\n", region)

		// Discover compute instances
		if config.ResourceType == "" || config.ResourceType == "google_compute_instance" {
			instances, err := p.discoverComputeInstances(region)
			if err != nil {
				fmt.Printf("  [GCP] Warning: Failed to discover compute instances in %s: %v\n", region, err)
			} else {
				allResources = append(allResources, instances...)
			}
		}

		// Discover networks (global resources)
		if config.ResourceType == "" || config.ResourceType == "google_compute_network" {
			networks, err := p.discoverNetworks()
			if err != nil {
				fmt.Printf("  [GCP] Warning: Failed to discover networks: %v\n", err)
			} else {
				allResources = append(allResources, networks...)
			}
		}
	}

	fmt.Printf("  [GCP] Found %d resources\n", len(allResources))
	return allResources, nil
}

// discoverComputeInstances discovers compute instances in a specific region
func (p *GCPProvider) discoverComputeInstances(region string) ([]models.Resource, error) {
	client, err := compute.NewInstancesRESTClient(p.ctx, option.WithEndpoint("https://compute.googleapis.com"))
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}
	defer client.Close()

	var resources []models.Resource

	// List zones in the region
	zones := p.getZonesInRegion(region)

	for _, zone := range zones {
		req := &computepb.ListInstancesRequest{
			Project: p.projectID,
			Zone:    zone,
		}

		it := client.List(p.ctx, req)
		for {
			instance, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to list instances: %w", err)
			}

			resource := p.convertComputeInstance(instance, zone, region)
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverNetworks discovers VPC networks (global resources)
func (p *GCPProvider) discoverNetworks() ([]models.Resource, error) {
	client, err := compute.NewNetworksRESTClient(p.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create networks client: %w", err)
	}
	defer client.Close()

	var resources []models.Resource

	req := &computepb.ListNetworksRequest{
		Project: p.projectID,
	}

	it := client.List(p.ctx, req)
	for {
		network, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list networks: %w", err)
		}

		resource := p.convertNetwork(network)
		resources = append(resources, resource)
	}

	return resources, nil
}

// Helper functions
func (p *GCPProvider) getZonesInRegion(region string) []string {
	// This is a simplified mapping. In a real implementation,
	// you would query the zones API
	zoneMap := map[string][]string{
		"us-central1": {"us-central1-a", "us-central1-b", "us-central1-c", "us-central1-f"},
		"us-east1":    {"us-east1-b", "us-east1-c", "us-east1-d"},
		"us-west1":    {"us-west1-a", "us-west1-b", "us-west1-c"},
		"us-west2":    {"us-west2-a", "us-west2-b", "us-west2-c"},
	}

	if zones, exists := zoneMap[region]; exists {
		return zones
	}

	// Default to first zone in region
	return []string{region + "-a"}
}

func (p *GCPProvider) convertComputeInstance(instance *computepb.Instance, zone, region string) models.Resource {
	name := ""
	if instance.Name != nil {
		name = *instance.Name
	}

	id := ""
	if instance.SelfLink != nil {
		id = *instance.SelfLink
	}

	tags := make(map[string]string)
	if instance.Labels != nil {
		for k, v := range instance.Labels {
			tags[k] = v
		}
	}

	// Add network tags as well
	if instance.Tags != nil && instance.Tags.Items != nil {
		for _, tag := range instance.Tags.Items {
			tags[tag] = ""
		}
	}

	machineType := ""
	if instance.MachineType != nil {
		// Extract machine type from URL
		parts := strings.Split(*instance.MachineType, "/")
		if len(parts) > 0 {
			machineType = parts[len(parts)-1]
		}
	}

	status := ""
	if instance.Status != nil {
		status = *instance.Status
	}

	return models.Resource{
		ID:            id,
		Name:          name,
		Type:          "google_compute_instance",
		TerraformType: "google_compute_instance",
		Provider:      "gcp",
		Region:        region,
		Tags:          tags,
		ImportID:      fmt.Sprintf("projects/%s/zones/%s/instances/%s", p.projectID, zone, name),
		CreatedAt:     time.Now(), // GCP doesn't provide creation time in list API
		Metadata: map[string]interface{}{
			"machine_type": machineType,
			"zone":         zone,
			"status":       status,
		},
	}
}

func (p *GCPProvider) convertNetwork(network *computepb.Network) models.Resource {
	name := ""
	if network.Name != nil {
		name = *network.Name
	}

	id := ""
	if network.SelfLink != nil {
		id = *network.SelfLink
	}

	autoCreate := false
	if network.AutoCreateSubnetworks != nil {
		autoCreate = *network.AutoCreateSubnetworks
	}

	routingMode := ""
	if network.RoutingConfig != nil && network.RoutingConfig.RoutingMode != nil {
		routingMode = *network.RoutingConfig.RoutingMode
	}

	mtu := int32(0)
	if network.Mtu != nil {
		mtu = *network.Mtu
	}

	return models.Resource{
		ID:            id,
		Name:          name,
		Type:          "google_compute_network",
		TerraformType: "google_compute_network",
		Provider:      "gcp",
		Region:        "global",
		Tags:          make(map[string]string),
		ImportID:      fmt.Sprintf("projects/%s/global/networks/%s", p.projectID, name),
		CreatedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"auto_create_subnetworks": autoCreate,
			"routing_mode":            routingMode,
			"mtu":                     mtu,
		},
	}
}

// getGCPProjectID gets the project ID from environment or config
func getGCPProjectID() string {
	// In a real implementation, this would check:
	// 1. Environment variable GOOGLE_CLOUD_PROJECT
	// 2. Application default credentials
	// 3. gcloud config
	// 4. Configuration file

	// For demo, return empty to trigger mock data
	return ""
}

// getMockResources returns mock data when real GCP discovery fails
func (p *GCPProvider) getMockResources(config Config) []models.Resource {
	fmt.Println("  [GCP] Using mock resources for demonstration")

	resources := []models.Resource{
		{
			ID:            "projects/my-project/zones/us-central1-a/instances/web-server-instance",
			Name:          "web-server-instance",
			Type:          "google_compute_instance",
			TerraformType: "google_compute_instance",
			Provider:      "gcp",
			Region:        "us-central1",
			Tags: map[string]string{
				"environment":  "production",
				"team":         "platform",
				"http-server":  "",
				"https-server": "",
			},
			ImportID:  "projects/my-project/zones/us-central1-a/instances/web-server-instance",
			CreatedAt: time.Now().Add(-36 * time.Hour),
			Metadata: map[string]interface{}{
				"machine_type": "e2-medium",
				"zone":         "us-central1-a",
				"status":       "RUNNING",
				"disk_size":    "20",
				"disk_type":    "pd-standard",
			},
		},
		{
			ID:            "projects/my-project/global/networks/default",
			Name:          "default",
			Type:          "google_compute_network",
			TerraformType: "google_compute_network",
			Provider:      "gcp",
			Region:        "global",
			Tags: map[string]string{
				"environment": "production",
			},
			ImportID:  "projects/my-project/global/networks/default",
			CreatedAt: time.Now().Add(-168 * time.Hour),
			Metadata: map[string]interface{}{
				"auto_create_subnetworks": true,
				"routing_mode":            "REGIONAL",
				"mtu":                     1460,
			},
		},
		{
			ID:            "projects/my-project/global/buckets/my-project-storage",
			Name:          "my-project-storage",
			Type:          "google_storage_bucket",
			TerraformType: "google_storage_bucket",
			Provider:      "gcp",
			Region:        "us-central1",
			Tags: map[string]string{
				"environment": "production",
				"purpose":     "data-storage",
			},
			ImportID:  "my-project-storage",
			CreatedAt: time.Now().Add(-240 * time.Hour),
			Metadata: map[string]interface{}{
				"location":      "US-CENTRAL1",
				"storage_class": "STANDARD",
				"versioning":    true,
			},
		},
	}

	// Apply basic filtering
	var filtered []models.Resource
	for _, resource := range resources {
		if config.ResourceType != "" && resource.Type != config.ResourceType {
			continue
		}
		if len(config.Regions) > 0 && !contains(config.Regions, resource.Region) && resource.Region != "global" {
			continue
		}
		filtered = append(filtered, resource)
	}

	return filtered
}
