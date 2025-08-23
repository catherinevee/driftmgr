package deletion

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/iterator"
)

// GCPProvider implements CloudProvider for Google Cloud Platform
type GCPProvider struct {
	projectID        string
	storageClient    *storage.Client
	resourceManager  *cloudresourcemanager.Service
	containerService *container.Service
}

// NewGCPProvider creates a new GCP provider
func NewGCPProvider() (*GCPProvider, error) {
	ctx := context.Background()

	// Get project ID from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}
	if projectID == "" {
		return nil, fmt.Errorf("GCP project ID not configured. Please set GOOGLE_CLOUD_PROJECT or GCP_PROJECT environment variable")
	}

	// Initialize storage client
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	// Initialize resource manager service
	resourceManager, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager service: %w", err)
	}

	// Initialize container service
	containerService, err := container.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create container service: %w", err)
	}

	return &GCPProvider{
		projectID:        projectID,
		storageClient:    storageClient,
		resourceManager:  resourceManager,
		containerService: containerService,
	}, nil
}

// ValidateCredentials validates GCP credentials
func (gp *GCPProvider) ValidateCredentials(ctx context.Context, accountID string) error {
	// Test the credentials by listing projects
	_, err := gp.resourceManager.Projects.List().Do()
	if err != nil {
		return fmt.Errorf("failed to validate GCP credentials: %w", err)
	}

	return nil
}

// ListResources lists all GCP resources
func (gp *GCPProvider) ListResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Define resource discovery functions
	discoveryFuncs := []struct {
		name string
		fn   func(context.Context, string) ([]models.Resource, error)
	}{
		{"ComputeInstances", gp.discoverComputeInstances},
		{"StorageBuckets", gp.discoverStorageBuckets},
		{"KubernetesClusters", gp.discoverKubernetesClusters},
		{"VPCNetworks", gp.discoverVPCNetworks},
		{"LoadBalancers", gp.discoverLoadBalancers},
		{"SQLInstances", gp.discoverSQLInstances},
		{"PubSubTopics", gp.discoverPubSubTopics},
		{"CloudFunctions", gp.discoverCloudFunctions},
		{"DataprocClusters", gp.discoverDataprocClusters},
		{"BigQueryDatasets", gp.discoverBigQueryDatasets},
	}

	for _, discovery := range discoveryFuncs {
		wg.Add(1)
		go func(d struct {
			name string
			fn   func(context.Context, string) ([]models.Resource, error)
		}) {
			defer wg.Done()

			res, err := d.fn(ctx, accountID)
			if err != nil {
				log.Printf("Error discovering %s resources: %v", d.name, err)
				return
			}

			mu.Lock()
			resources = append(resources, res...)
			mu.Unlock()
		}(discovery)
	}

	wg.Wait()
	return resources, nil
}

// DeleteResources deletes GCP resources in the correct order
// DeleteResource implements the CloudProvider interface for single resource deletion
func (gp *GCPProvider) DeleteResource(ctx context.Context, resource models.Resource) error {
	return gp.deleteResource(ctx, resource, DeletionOptions{})
}

func (gp *GCPProvider) DeleteResources(ctx context.Context, accountID string, options DeletionOptions) (*DeletionResult, error) {
	startTime := time.Now()
	result := &DeletionResult{
		AccountID: accountID,
		Provider:  "gcp",
		StartTime: startTime,
		Errors:    []DeletionError{},
		Warnings:  []string{},
		Details:   make(map[string]interface{}),
	}

	// List all resources first
	resources, err := gp.ListResources(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	result.TotalResources = len(resources)

	// Filter resources based on options
	filteredResources := gp.filterResources(resources, options)

	if options.DryRun {
		result.DeletedResources = len(filteredResources)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		return result, nil
	}

	// Delete resources in dependency order
	deletionOrder := []string{
		"compute.googleapis.com/instances",
		"container.googleapis.com/clusters",
		"storage.googleapis.com/buckets",
		"sqladmin.googleapis.com/instances",
		"pubsub.googleapis.com/topics",
		"cloudfunctions.googleapis.com/functions",
		"dataproc.googleapis.com/clusters",
		"bigquery.googleapis.com/datasets",
		"compute.googleapis.com/forwardingRules",
		"compute.googleapis.com/targetPools",
		"compute.googleapis.com/networks",
	}

	// Group resources by type
	resourceGroups := make(map[string][]models.Resource)
	for _, resource := range filteredResources {
		resourceGroups[resource.Type] = append(resourceGroups[resource.Type], resource)
	}

	// Delete resources in order
	for _, resourceType := range deletionOrder {
		if resources, exists := resourceGroups[resourceType]; exists {
			for _, resource := range resources {
				if err := gp.deleteResource(ctx, resource, options); err != nil {
					result.Errors = append(result.Errors, DeletionError{
						ResourceID:   resource.ID,
						ResourceType: resource.Type,
						Error:        err.Error(),
						Timestamp:    time.Now(),
					})
					result.FailedResources++
				} else {
					result.DeletedResources++
				}

				// Send progress update
				if options.ProgressCallback != nil {
					options.ProgressCallback(ProgressUpdate{
						Type:      "deletion_progress",
						Message:   fmt.Sprintf("Deleted %s: %s", resource.Type, resource.Name),
						Progress:  result.DeletedResources + result.FailedResources,
						Total:     result.TotalResources,
						Current:   resource.Name,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	return result, nil
}

// deleteResource deletes a single GCP resource
func (gp *GCPProvider) deleteResource(ctx context.Context, resource models.Resource, options DeletionOptions) error {
	switch resource.Type {
	case "compute.googleapis.com/instances":
		return gp.deleteComputeInstance(ctx, resource)
	case "storage.googleapis.com/buckets":
		return gp.deleteStorageBucket(ctx, resource)
	case "container.googleapis.com/clusters":
		return gp.deleteKubernetesCluster(ctx, resource)
	case "sqladmin.googleapis.com/instances":
		return gp.deleteSQLInstance(ctx, resource)
	case "pubsub.googleapis.com/topics":
		return gp.deletePubSubTopic(ctx, resource)
	case "cloudfunctions.googleapis.com/functions":
		return gp.deleteCloudFunction(ctx, resource)
	case "dataproc.googleapis.com/clusters":
		return gp.deleteDataprocCluster(ctx, resource)
	case "bigquery.googleapis.com/datasets":
		return gp.deleteBigQueryDataset(ctx, resource)
	case "compute.googleapis.com/networks":
		return gp.deleteVPCNetwork(ctx, resource)
	default:
		return fmt.Errorf("unsupported resource type: %s", resource.Type)
	}
}

// filterResources filters resources based on deletion options
func (gp *GCPProvider) filterResources(resources []models.Resource, options DeletionOptions) []models.Resource {
	var filtered []models.Resource

	for _, resource := range resources {
		// Check if resource should be excluded
		if gp.shouldExcludeResource(resource, options) {
			continue
		}

		// Check if resource should be included
		if len(options.IncludeResources) > 0 && !gp.shouldIncludeResource(resource, options) {
			continue
		}

		// Check resource type filter
		if len(options.ResourceTypes) > 0 && !gp.containsString(options.ResourceTypes, resource.Type) {
			continue
		}

		// Check region filter
		if len(options.Regions) > 0 && !gp.containsString(options.Regions, resource.Region) {
			continue
		}

		filtered = append(filtered, resource)
	}

	return filtered
}

// Helper methods for resource discovery
func (gp *GCPProvider) discoverComputeInstances(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for Compute Instances discovery
	// This would use the compute API to list instances
	// For now, return empty list as compute client is not available

	return resources, nil
}

func (gp *GCPProvider) discoverStorageBuckets(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	it := gp.storageClient.Buckets(ctx, gp.projectID)
	for {
		bucket, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			continue
		}

		resources = append(resources, models.Resource{
			ID:       bucket.Name,
			Name:     bucket.Name,
			Type:     "storage.googleapis.com/buckets",
			Provider: "gcp",
			Region:   bucket.Location,
			State:    "ACTIVE",
			Created:  bucket.Created,
		})
	}

	return resources, nil
}

func (gp *GCPProvider) discoverKubernetesClusters(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// List clusters across all zones
	zones := []string{"us-central1-a", "us-central1-b", "us-west1-a", "us-west1-b", "europe-west1-a", "europe-west1-b"}

	for _, zone := range zones {
		clusters, err := gp.containerService.Projects.Zones.Clusters.List(gp.projectID, zone).Do()
		if err != nil {
			continue
		}

		for _, cluster := range clusters.Clusters {
			resources = append(resources, models.Resource{
				ID:       cluster.Id,
				Name:     cluster.Name,
				Type:     "container.googleapis.com/clusters",
				Provider: "gcp",
				Region:   zone,
				State:    cluster.Status,
				Created:  time.Now(),
			})
		}
	}

	return resources, nil
}

func (gp *GCPProvider) discoverVPCNetworks(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for VPC Networks discovery
	// This would use the compute API to list networks

	return resources, nil
}

func (gp *GCPProvider) discoverLoadBalancers(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for Load Balancers discovery
	// This would use the compute API to list forwarding rules and target pools

	return resources, nil
}

func (gp *GCPProvider) discoverSQLInstances(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for SQL Instances discovery
	// This would use the SQL Admin API

	return resources, nil
}

func (gp *GCPProvider) discoverPubSubTopics(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for Pub/Sub Topics discovery
	// This would use the Pub/Sub API

	return resources, nil
}

func (gp *GCPProvider) discoverCloudFunctions(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for Cloud Functions discovery
	// This would use the Cloud Functions API

	return resources, nil
}

func (gp *GCPProvider) discoverDataprocClusters(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for Dataproc Clusters discovery
	// This would use the Dataproc API

	return resources, nil
}

func (gp *GCPProvider) discoverBigQueryDatasets(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Implementation for BigQuery Datasets discovery
	// This would use the BigQuery API

	return resources, nil
}

// Helper methods for resource deletion
func (gp *GCPProvider) deleteComputeInstance(ctx context.Context, resource models.Resource) error {
	// Implementation for Compute Instance deletion
	// This would use the compute API to delete instances
	// For now, return nil as compute client is not available
	return nil
}

func (gp *GCPProvider) deleteStorageBucket(ctx context.Context, resource models.Resource) error {
	bucket := gp.storageClient.Bucket(resource.Name)

	// Delete all objects first
	it := bucket.Objects(ctx, nil)
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			continue
		}

		if err := bucket.Object(obj.Name).Delete(ctx); err != nil {
			return err
		}
	}

	// Delete the bucket
	return bucket.Delete(ctx)
}

func (gp *GCPProvider) deleteKubernetesCluster(ctx context.Context, resource models.Resource) error {
	// Extract zone from resource region
	zone := resource.Region

	_, err := gp.containerService.Projects.Zones.Clusters.Delete(gp.projectID, zone, resource.Name).Do()
	return err
}

func (gp *GCPProvider) deleteSQLInstance(ctx context.Context, resource models.Resource) error {
	// Implementation for SQL Instance deletion
	// This would use the SQL Admin API
	return nil
}

func (gp *GCPProvider) deletePubSubTopic(ctx context.Context, resource models.Resource) error {
	// Implementation for Pub/Sub Topic deletion
	// This would use the Pub/Sub API
	return nil
}

func (gp *GCPProvider) deleteCloudFunction(ctx context.Context, resource models.Resource) error {
	// Implementation for Cloud Function deletion
	// This would use the Cloud Functions API
	return nil
}

func (gp *GCPProvider) deleteDataprocCluster(ctx context.Context, resource models.Resource) error {
	// Implementation for Dataproc Cluster deletion
	// This would use the Dataproc API
	return nil
}

func (gp *GCPProvider) deleteBigQueryDataset(ctx context.Context, resource models.Resource) error {
	// Implementation for BigQuery Dataset deletion
	// This would use the BigQuery API
	return nil
}

func (gp *GCPProvider) deleteVPCNetwork(ctx context.Context, resource models.Resource) error {
	// Implementation for VPC Network deletion
	// This would use the compute API
	return nil
}

// Helper utility methods
func (gp *GCPProvider) shouldExcludeResource(resource models.Resource, options DeletionOptions) bool {
	for _, excludeID := range options.ExcludeResources {
		if resource.ID == excludeID || resource.Name == excludeID {
			return true
		}
	}
	return false
}

func (gp *GCPProvider) shouldIncludeResource(resource models.Resource, options DeletionOptions) bool {
	for _, includeID := range options.IncludeResources {
		if resource.ID == includeID || resource.Name == includeID {
			return true
		}
	}
	return false
}

func (gp *GCPProvider) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
