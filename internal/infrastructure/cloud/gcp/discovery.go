package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"github.com/catherinevee/driftmgr/internal/core/models"
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
		return nil, fmt.Errorf("no GCP project ID found. Please set GOOGLE_CLOUD_PROJECT or run 'gcloud config set project PROJECT_ID'")
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

// Regions returns the list of available GCP regions
func (p *GCPProvider) Regions() []string {
	return p.SupportedRegions()
}

// Services returns the list of available GCP services
func (p *GCPProvider) Services() []string {
	return []string{
		"Compute Engine", "Cloud Storage", "Cloud SQL", "BigQuery",
		"Cloud Functions", "GKE", "App Engine", "Cloud Run",
		"Pub/Sub", "Dataflow", "Cloud Spanner", "Firestore",
	}
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
func (p *GCPProvider) Discover(ctx context.Context, options DiscoveryOptions) (*Result, error) {
	fmt.Println("  [GCP] Discovering resources using Google Cloud SDK...")

	// Convert options to config for backward compatibility
	config := Config{
		Regions: options.Regions,
	}
	if len(options.ResourceTypes) > 0 {
		config.ResourceType = options.ResourceTypes[0]
	}

	if p.projectID == "" {
		// Try to get project ID from environment or default credentials
		p.projectID = getGCPProjectID()
		if p.projectID == "" {
			return nil, fmt.Errorf("no GCP project ID configured")
		}
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
	return &Result{
		Resources: allResources,
		Metadata: map[string]interface{}{
			"provider":       "gcp",
			"resource_count": len(allResources),
			"regions":        regions,
			"project_id":     p.projectID,
		},
	}, nil
}

// GetAccountInfo returns GCP account information
func (p *GCPProvider) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	projectID := p.projectID
	if projectID == "" {
		projectID = getGCPProjectID()
	}
	if projectID == "" {
		return nil, fmt.Errorf("no GCP project found")
	}

	return &AccountInfo{
		ID:       projectID,
		Name:     fmt.Sprintf("GCP Project %s", projectID),
		Type:     "gcp",
		Provider: "gcp",
		Regions:  p.SupportedRegions(),
	}, nil
}

// discoverComputeInstances discovers compute instances in a specific region
func (p *GCPProvider) discoverComputeInstances(region string) ([]models.Resource, error) {
	// Create client with proper authentication
	opts := []option.ClientOption{}
	
	// Check for explicit credentials file
	if credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credPath != "" {
		if _, err := os.Stat(credPath); err == nil {
			opts = append(opts, option.WithCredentialsFile(credPath))
		}
	}
	
	client, err := compute.NewInstancesRESTClient(p.ctx, opts...)
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
	// Create client with proper authentication
	opts := []option.ClientOption{}
	
	// Check for explicit credentials file
	if credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credPath != "" {
		if _, err := os.Stat(credPath); err == nil {
			opts = append(opts, option.WithCredentialsFile(credPath))
		}
	}
	
	client, err := compute.NewNetworksRESTClient(p.ctx, opts...)
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
		ID:   id,
		Name: name,
		Type: "google_compute_instance",

		Provider: "gcp",
		Region:   region,
		Tags:     tags,

		CreatedAt: time.Now(), // GCP doesn't provide creation time in list API
		Metadata: map[string]string{
			"terraform_type": "google_compute_instance",
			"import_id":      fmt.Sprintf("projects/%s/zones/%s/instances/%s", p.projectID, zone, name),
			"machine_type":   machineType,
			"zone":           zone,
			"status":         status,
		},
		Attributes: map[string]interface{}{},
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
		ID:   id,
		Name: name,
		Type: "google_compute_network",

		Provider: "gcp",
		Region:   "global",
		Tags:     make(map[string]string),

		CreatedAt: time.Now(),
		Metadata: map[string]string{
			"terraform_type":          "google_compute_network",
			"import_id":               fmt.Sprintf("projects/%s/global/networks/%s", p.projectID, name),
			"auto_create_subnetworks": fmt.Sprintf("%v", autoCreate),
			"routing_mode":            routingMode,
			"mtu":                     fmt.Sprintf("%d", mtu),
		},
		Attributes: map[string]interface{}{},
	}
}

// getGCPProjectID gets the project ID from environment or config
func getGCPProjectID() string {
	// Check environment variables
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		return projectID
	}
	if projectID := os.Getenv("GCP_PROJECT"); projectID != "" {
		return projectID
	}
	if projectID := os.Getenv("GCLOUD_PROJECT"); projectID != "" {
		return projectID
	}

	// Check application default credentials
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		adcPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
		if data, err := os.ReadFile(adcPath); err == nil {
			var creds map[string]interface{}
			if err := json.Unmarshal(data, &creds); err == nil {
				// ADC file might have quota_project_id
				if projectID, ok := creds["quota_project_id"].(string); ok && projectID != "" {
					return projectID
				}
			}
		}

		// Check gcloud configurations directory
		configPath := filepath.Join(homeDir, ".config", "gcloud", "configurations", "config_default")
		if data, err := os.ReadFile(configPath); err == nil {
			// Parse the INI-style config file
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "project = ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "project = "))
				}
			}
		}
	}

	// For demo, return empty to trigger mock data
	return ""
}


// ValidateCredentials validates GCP credentials
func (p *GCPProvider) ValidateCredentials(ctx context.Context) error {
	// Check if we have a project ID
	if p.projectID == "" {
		p.projectID = getGCPProjectID()
		if p.projectID == "" {
			return fmt.Errorf("no GCP project configured")
		}
	}
	
	// Try to create a compute client to validate credentials
	opts := []option.ClientOption{}
	if credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credPath != "" {
		if _, err := os.Stat(credPath); err == nil {
			opts = append(opts, option.WithCredentialsFile(credPath))
		}
	}
	
	client, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to validate GCP credentials: %w", err)
	}
	defer client.Close()
	
	// Try to list instances in any zone to validate credentials
	zones := p.getZonesInRegion("us-central1")
	if len(zones) > 0 {
		req := &computepb.ListInstancesRequest{
			Project:    p.projectID,
			Zone:       zones[0],
			MaxResults: aws.Uint32(1), // Just check if we can make a request
		}
		
		it := client.List(ctx, req)
		_, err := it.Next()
		if err != nil && err != iterator.Done {
			// Check if it's a permission error vs other errors
			if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "401") {
				return fmt.Errorf("GCP authentication failed: %w", err)
			}
			// Other errors might be OK (e.g., no instances)
		}
	}
	
	return nil
}

// DiscoverRegion discovers resources in a specific region
func (p *GCPProvider) DiscoverRegion(ctx context.Context, region string) ([]models.Resource, error) {
	options := DiscoveryOptions{
		Regions: []string{region},
	}
	result, err := p.Discover(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Resources, nil
}
