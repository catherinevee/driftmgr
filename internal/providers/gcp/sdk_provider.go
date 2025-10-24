package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
	"github.com/catherinevee/driftmgr/pkg/models"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/option"
)

// GCPSDKProvider implements CloudProvider for GCP using GCP SDK
type GCPSDKProvider struct {
	projectID       string
	region          string
	computeClient   *compute.InstancesClient
	storageClient   *storage.Client
	resourceManager *cloudresourcemanager.Service
}

// NewGCPSDKProvider creates a new GCP provider using GCP SDK
func NewGCPSDKProvider(projectID, region string) (*GCPSDKProvider, error) {
	ctx := context.Background()

	// Create compute client
	computeClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP compute client: %w", err)
	}

	// Create storage client
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP storage client: %w", err)
	}

	// Create resource manager service
	resourceManager, err := cloudresourcemanager.NewService(ctx, option.WithScopes(
		cloudresourcemanager.CloudPlatformScope,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP resource manager service: %w", err)
	}

	if region == "" {
		region = "us-central1"
	}

	return &GCPSDKProvider{
		projectID:       projectID,
		region:          region,
		computeClient:   computeClient,
		storageClient:   storageClient,
		resourceManager: resourceManager,
	}, nil
}

// Name returns the provider name
func (p *GCPSDKProvider) Name() string {
	return "gcp"
}

// DiscoverResources discovers resources in the specified region
func (p *GCPSDKProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	var resources []models.Resource

	// If region is specified, use it
	if region != "" {
		p.region = region
	}

	// Discover compute instances
	instances, err := p.discoverComputeInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover compute instances: %w", err)
	}
	resources = append(resources, instances...)

	// Discover storage buckets
	buckets, err := p.discoverStorageBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover storage buckets: %w", err)
	}
	resources = append(resources, buckets...)

	return resources, nil
}

// discoverComputeInstances discovers GCP compute instances
func (p *GCPSDKProvider) discoverComputeInstances(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// List instances in the project
	req := &computepb.AggregatedListInstancesRequest{
		Project: p.projectID,
	}

	it := p.computeClient.AggregatedList(ctx, req)
	for {
		pair, err := it.Next()
		if err != nil {
			break
		}

		if pair.Value == nil {
			continue
		}

		for _, instance := range pair.Value.Instances {
			// Extract zone from the instance URL
			zone := ""
			if instance.Zone != nil {
				zone = p.extractZoneFromURL(*instance.Zone)
			}

			// Convert labels
			labels := make(map[string]string)
			if instance.Labels != nil {
				for k, v := range instance.Labels {
					labels[k] = v
				}
			}

			// Create attributes
			attributes := make(map[string]interface{})
			attributes["name"] = instance.Name
			if instance.MachineType != nil {
				attributes["machine_type"] = p.extractMachineTypeFromURL(*instance.MachineType)
			}
			attributes["zone"] = zone
			attributes["status"] = instance.Status
			attributes["labels"] = labels
			attributes["creation_timestamp"] = instance.CreationTimestamp

			resource := models.Resource{
				ID:           *instance.Name,
				Type:         "google_compute_instance",
				Provider:     "gcp",
				Region:       zone,
				Attributes:   attributes,
				Tags:         labels,
				CreatedAt:    time.Now(), // Use current time as fallback
				LastModified: time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverStorageBuckets discovers GCP storage buckets
func (p *GCPSDKProvider) discoverStorageBuckets(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// List buckets in the project
	it := p.storageClient.Buckets(ctx, p.projectID)
	for {
		bucket, err := it.Next()
		if err != nil {
			break
		}

		// Convert labels
		labels := make(map[string]string)
		if bucket.Labels != nil {
			for k, v := range bucket.Labels {
				labels[k] = v
			}
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = bucket.Name
		attributes["location"] = bucket.Location
		attributes["storage_class"] = bucket.StorageClass
		attributes["labels"] = labels
		attributes["time_created"] = bucket.Created

		resource := models.Resource{
			ID:           bucket.Name,
			Type:         "google_storage_bucket",
			Provider:     "gcp",
			Region:       bucket.Location,
			Attributes:   attributes,
			Tags:         labels,
			CreatedAt:    bucket.Created,
			LastModified: bucket.Created,
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// extractZoneFromURL extracts zone from GCP resource URL
func (p *GCPSDKProvider) extractZoneFromURL(url string) string {
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "zones" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractMachineTypeFromURL extracts machine type from GCP resource URL
func (p *GCPSDKProvider) extractMachineTypeFromURL(url string) string {
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "machineTypes" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// GetResource retrieves a specific resource by ID
func (p *GCPSDKProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	// Try to determine resource type from ID and get the resource
	// For now, we'll try compute instances first
	req := &computepb.GetInstanceRequest{
		Project:  p.projectID,
		Zone:     p.region + "-a", // Default zone
		Instance: resourceID,
	}

	instance, err := p.computeClient.Get(ctx, req)
	if err != nil {
		// Try storage bucket
		bucket := p.storageClient.Bucket(resourceID)
		attrs, err := bucket.Attrs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource %s: %w", resourceID, err)
		}

		// Convert labels
		labels := make(map[string]string)
		if attrs.Labels != nil {
			for k, v := range attrs.Labels {
				labels[k] = v
			}
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = attrs.Name
		attributes["location"] = attrs.Location
		attributes["storage_class"] = attrs.StorageClass
		attributes["labels"] = labels
		attributes["time_created"] = attrs.Created

		return &models.Resource{
			ID:           attrs.Name,
			Type:         "google_storage_bucket",
			Provider:     "gcp",
			Region:       attrs.Location,
			Attributes:   attributes,
			Tags:         labels,
			CreatedAt:    attrs.Created,
			LastModified: attrs.Created,
		}, nil
	}

	// Convert labels
	labels := make(map[string]string)
	if instance.Labels != nil {
		for k, v := range instance.Labels {
			labels[k] = v
		}
	}

	// Create attributes
	attributes := make(map[string]interface{})
	attributes["name"] = instance.Name
	attributes["machine_type"] = p.extractMachineTypeFromURL(*instance.MachineType)
	attributes["zone"] = p.extractZoneFromURL(*instance.Zone)
	attributes["status"] = instance.Status
	attributes["labels"] = labels
	attributes["creation_timestamp"] = instance.CreationTimestamp

	return &models.Resource{
		ID:           *instance.Name,
		Type:         "google_compute_instance",
		Provider:     "gcp",
		Region:       p.extractZoneFromURL(*instance.Zone),
		Attributes:   attributes,
		Tags:         labels,
		CreatedAt:    time.Now(),
		LastModified: time.Now(),
	}, nil
}

// ValidateCredentials checks if the provider credentials are valid
func (p *GCPSDKProvider) ValidateCredentials(ctx context.Context) error {
	// Test connection by getting project info
	project, err := p.resourceManager.Projects.Get(p.projectID).Do()
	if err != nil {
		return fmt.Errorf("failed to validate GCP credentials: %w", err)
	}

	if project == nil {
		return fmt.Errorf("project %s not found", p.projectID)
	}

	return nil
}

// ListRegions returns available regions for the provider
func (p *GCPSDKProvider) ListRegions(ctx context.Context) ([]string, error) {
	// Return common GCP regions
	return []string{
		"us-central1", "us-east1", "us-east4", "us-west1", "us-west2", "us-west3", "us-west4",
		"europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6",
		"asia-east1", "asia-east2", "asia-northeast1", "asia-northeast2", "asia-northeast3",
		"asia-south1", "asia-southeast1", "asia-southeast2",
		"australia-southeast1", "southamerica-east1", "northamerica-northeast1",
	}, nil
}

// SupportedResourceTypes returns the list of supported resource types
func (p *GCPSDKProvider) SupportedResourceTypes() []string {
	return []string{
		"google_compute_instance",
		"google_compute_network",
		"google_compute_subnetwork",
		"google_compute_firewall",
		"google_compute_disk",
		"google_storage_bucket",
		"google_sql_database_instance",
		"google_sql_database",
		"google_container_cluster",
		"google_pubsub_topic",
		"google_pubsub_subscription",
		"google_cloud_function",
		"google_cloud_run_service",
		"google_redis_instance",
		"google_kms_key_ring",
		"google_kms_crypto_key",
	}
}

// TestConnection tests the connection to GCP
func (p *GCPSDKProvider) TestConnection(ctx context.Context) error {
	return p.ValidateCredentials(ctx)
}

// GetProjectID returns the project ID
func (p *GCPSDKProvider) GetProjectID() string {
	return p.projectID
}

// GetRegion returns the region
func (p *GCPSDKProvider) GetRegion() string {
	return p.region
}

// ListProjects lists all projects accessible to the current credentials
func (p *GCPSDKProvider) ListProjects(ctx context.Context) ([]string, error) {
	var projectIDs []string

	req := p.resourceManager.Projects.List()
	resp, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	for _, project := range resp.Projects {
		projectIDs = append(projectIDs, project.ProjectId)
	}

	return projectIDs, nil
}

// GetResourceByType retrieves a specific resource by type and name
func (p *GCPSDKProvider) GetResourceByType(ctx context.Context, resourceType, resourceName string) (*models.Resource, error) {
	switch resourceType {
	case "google_compute_instance":
		req := &computepb.GetInstanceRequest{
			Project:  p.projectID,
			Zone:     p.region + "-a", // Default zone
			Instance: resourceName,
		}

		instance, err := p.computeClient.Get(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to get compute instance: %w", err)
		}

		// Convert labels
		labels := make(map[string]string)
		if instance.Labels != nil {
			for k, v := range instance.Labels {
				labels[k] = v
			}
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = instance.Name
		attributes["machine_type"] = p.extractMachineTypeFromURL(*instance.MachineType)
		attributes["zone"] = p.extractZoneFromURL(*instance.Zone)
		attributes["status"] = instance.Status
		attributes["labels"] = labels
		attributes["creation_timestamp"] = instance.CreationTimestamp

		return &models.Resource{
			ID:           *instance.Name,
			Type:         "google_compute_instance",
			Provider:     "gcp",
			Region:       p.extractZoneFromURL(*instance.Zone),
			Attributes:   attributes,
			Tags:         labels,
			CreatedAt:    time.Now(),
			LastModified: time.Now(),
		}, nil

	case "google_storage_bucket":
		bucket := p.storageClient.Bucket(resourceName)
		attrs, err := bucket.Attrs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get storage bucket: %w", err)
		}

		// Convert labels
		labels := make(map[string]string)
		if attrs.Labels != nil {
			for k, v := range attrs.Labels {
				labels[k] = v
			}
		}

		// Create attributes
		attributes := make(map[string]interface{})
		attributes["name"] = attrs.Name
		attributes["location"] = attrs.Location
		attributes["storage_class"] = attrs.StorageClass
		attributes["labels"] = labels
		attributes["time_created"] = attrs.Created

		return &models.Resource{
			ID:           attrs.Name,
			Type:         "google_storage_bucket",
			Provider:     "gcp",
			Region:       attrs.Location,
			Attributes:   attributes,
			Tags:         labels,
			CreatedAt:    attrs.Created,
			LastModified: attrs.Created,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// ListResourcesByType lists all resources of a specific type
func (p *GCPSDKProvider) ListResourcesByType(ctx context.Context, resourceType string) ([]models.Resource, error) {
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
