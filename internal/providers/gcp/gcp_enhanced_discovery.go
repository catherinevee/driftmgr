package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// GCPEnhancedDiscoverer provides comprehensive GCP resource discovery
type GCPEnhancedDiscoverer struct {
	projectID string
	cliPath   string
}

// NewGCPEnhancedDiscoverer creates a new GCP enhanced discoverer
func NewGCPEnhancedDiscoverer(projectID string) (*GCPEnhancedDiscoverer, error) {
	cliPath, err := exec.LookPath("gcloud")
	if err != nil {
		return nil, fmt.Errorf("gcloud CLI not found: %w", err)
	}

	if projectID == "" {
		// Try to get current project
		cmd := exec.Command(cliPath, "config", "get-value", "project")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get GCP project ID: %w", err)
		}
		projectID = strings.TrimSpace(string(output))
	}

	return &GCPEnhancedDiscoverer{
		projectID: projectID,
		cliPath:   cliPath,
	}, nil
}

// DiscoverAllGCPResources discovers all GCP resources comprehensively
func (g *GCPEnhancedDiscoverer) DiscoverAllGCPResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var allResources []models.Resource

	// If no regions specified, use common GCP regions
	if len(regions) == 0 {
		regions = []string{
			"us-central1", "us-east1", "us-west1", "us-west2", "us-west3", "us-west4",
			"europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6", "europe-north1",
			"asia-east1", "asia-northeast1", "asia-southeast1", "asia-south1",
		}
	}

	// Discover different resource categories
	log.Printf("Discovering GCP Compute resources...")
	computeResources, _ := g.discoverComputeResources(ctx, regions)
	allResources = append(allResources, computeResources...)

	log.Printf("Discovering GCP Storage resources...")
	storageResources, _ := g.discoverStorageResources(ctx, regions)
	allResources = append(allResources, storageResources...)

	log.Printf("Discovering GCP Database resources...")
	dbResources, _ := g.discoverDatabaseResources(ctx, regions)
	allResources = append(allResources, dbResources...)

	log.Printf("Discovering GCP Networking resources...")
	networkResources, _ := g.discoverNetworkingResources(ctx, regions)
	allResources = append(allResources, networkResources...)

	log.Printf("Discovering GCP Container resources...")
	containerResources, _ := g.discoverContainerResources(ctx, regions)
	allResources = append(allResources, containerResources...)

	log.Printf("Discovering GCP Functions resources...")
	functionResources, _ := g.discoverFunctionResources(ctx, regions)
	allResources = append(allResources, functionResources...)

	log.Printf("Discovering GCP BigData resources...")
	bigDataResources, _ := g.discoverBigDataResources(ctx, regions)
	allResources = append(allResources, bigDataResources...)

	log.Printf("Discovering GCP Security resources...")
	securityResources, _ := g.discoverSecurityResources(ctx, regions)
	allResources = append(allResources, securityResources...)

	log.Printf("Discovering GCP AI/ML resources...")
	aiResources, _ := g.discoverAIMLResources(ctx, regions)
	allResources = append(allResources, aiResources...)

	log.Printf("Discovering GCP Management resources...")
	mgmtResources, _ := g.discoverManagementResources(ctx, regions)
	allResources = append(allResources, mgmtResources...)

	log.Printf("GCP enhanced discovery completed: %d resources found", len(allResources))
	return allResources, nil
}

// discoverComputeResources discovers Compute Engine resources
func (g *GCPEnhancedDiscoverer) discoverComputeResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Compute instances
	instances, _ := g.executeGCloudCommand(ctx, []string{"compute", "instances", "list", "--format", "json"})
	for _, instance := range instances {
		if g.matchesRegions(getStringValueGCP(instance, "zone"), regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(instance, "selfLink"),
				Name:     getStringValueGCP(instance, "name"),
				Type:     "gcp_compute_instance",
				Provider: "gcp",
				Region:   extractRegionFromZone(getStringValueGCP(instance, "zone")),
				Status:   getStringValueGCP(instance, "status"),
				Tags:     extractLabelsGCP(instance),
				Attributes: map[string]interface{}{
					"machine_type": getStringValueGCP(instance, "machineType"),
					"zone":         getStringValueGCP(instance, "zone"),
					"self_link":    getStringValueGCP(instance, "selfLink"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(instance, "creationTimestamp")),
			})
		}
	}

	// Instance templates
	templates, _ := g.executeGCloudCommand(ctx, []string{"compute", "instance-templates", "list", "--format", "json"})
	for _, template := range templates {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(template, "selfLink"),
			Name:     getStringValueGCP(template, "name"),
			Type:     "gcp_compute_instance_template",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(template),
			Attributes: map[string]interface{}{
				"self_link": getStringValueGCP(template, "selfLink"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(template, "creationTimestamp")),
		})
	}

	// Instance groups
	groups, _ := g.executeGCloudCommand(ctx, []string{"compute", "instance-groups", "list", "--format", "json"})
	for _, group := range groups {
		if g.matchesRegions(getStringValueGCP(group, "zone"), regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(group, "selfLink"),
				Name:     getStringValueGCP(group, "name"),
				Type:     "gcp_compute_instance_group",
				Provider: "gcp",
				Region:   extractRegionFromZone(getStringValueGCP(group, "zone")),
				Status:   "active",
				Tags:     extractLabelsGCP(group),
				Attributes: map[string]interface{}{
					"size":      getIntValueGCP(group, "size"),
					"self_link": getStringValueGCP(group, "selfLink"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(group, "creationTimestamp")),
			})
		}
	}

	// Managed instance groups
	migs, _ := g.executeGCloudCommand(ctx, []string{"compute", "instance-groups", "managed", "list", "--format", "json"})
	for _, mig := range migs {
		if g.matchesRegions(getStringValueGCP(mig, "zone"), regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(mig, "selfLink"),
				Name:     getStringValueGCP(mig, "name"),
				Type:     "gcp_compute_instance_group_manager",
				Provider: "gcp",
				Region:   extractRegionFromZone(getStringValueGCP(mig, "zone")),
				Status:   getStringValueGCP(mig, "status"),
				Tags:     extractLabelsGCP(mig),
				Attributes: map[string]interface{}{
					"target_size": getIntValueGCP(mig, "targetSize"),
					"self_link":   getStringValueGCP(mig, "selfLink"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(mig, "creationTimestamp")),
			})
		}
	}

	// Disks
	disks, _ := g.executeGCloudCommand(ctx, []string{"compute", "disks", "list", "--format", "json"})
	for _, disk := range disks {
		if g.matchesRegions(getStringValueGCP(disk, "zone"), regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(disk, "selfLink"),
				Name:     getStringValueGCP(disk, "name"),
				Type:     "gcp_compute_disk",
				Provider: "gcp",
				Region:   extractRegionFromZone(getStringValueGCP(disk, "zone")),
				Status:   getStringValueGCP(disk, "status"),
				Tags:     extractLabelsGCP(disk),
				Attributes: map[string]interface{}{
					"size_gb":      getStringValueGCP(disk, "sizeGb"),
					"type":         getStringValueGCP(disk, "type"),
					"source_image": getStringValueGCP(disk, "sourceImage"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(disk, "creationTimestamp")),
			})
		}
	}

	// Images
	images, _ := g.executeGCloudCommand(ctx, []string{"compute", "images", "list", "--format", "json", "--filter", fmt.Sprintf("labels.project=%s", g.projectID)})
	for _, image := range images {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(image, "selfLink"),
			Name:     getStringValueGCP(image, "name"),
			Type:     "gcp_compute_image",
			Provider: "gcp",
			Region:   "global",
			Status:   getStringValueGCP(image, "status"),
			Tags:     extractLabelsGCP(image),
			Attributes: map[string]interface{}{
				"family":       getStringValueGCP(image, "family"),
				"disk_size_gb": getStringValueGCP(image, "diskSizeGb"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(image, "creationTimestamp")),
		})
	}

	// Snapshots
	snapshots, _ := g.executeGCloudCommand(ctx, []string{"compute", "snapshots", "list", "--format", "json"})
	for _, snapshot := range snapshots {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(snapshot, "selfLink"),
			Name:     getStringValueGCP(snapshot, "name"),
			Type:     "gcp_compute_snapshot",
			Provider: "gcp",
			Region:   "global",
			Status:   getStringValueGCP(snapshot, "status"),
			Tags:     extractLabelsGCP(snapshot),
			Attributes: map[string]interface{}{
				"disk_size_gb": getStringValueGCP(snapshot, "diskSizeGb"),
				"source_disk":  getStringValueGCP(snapshot, "sourceDisk"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(snapshot, "creationTimestamp")),
		})
	}

	return resources, nil
}

// discoverStorageResources discovers Cloud Storage resources
func (g *GCPEnhancedDiscoverer) discoverStorageResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Storage buckets
	buckets, _ := g.executeGCloudCommand(ctx, []string{"storage", "buckets", "list", "--format", "json"})
	for _, bucket := range buckets {
		location := getStringValueGCP(bucket, "location")
		if g.matchesRegions(location, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(bucket, "selfLink"),
				Name:     getStringValueGCP(bucket, "name"),
				Type:     "gcp_storage_bucket",
				Provider: "gcp",
				Region:   strings.ToLower(location),
				Status:   "active",
				Tags:     extractLabelsGCP(bucket),
				Attributes: map[string]interface{}{
					"storage_class": getStringValueGCP(bucket, "storageClass"),
					"location_type": getStringValueGCP(bucket, "locationType"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(bucket, "timeCreated")),
			})
		}
	}

	return resources, nil
}

// discoverDatabaseResources discovers database resources
func (g *GCPEnhancedDiscoverer) discoverDatabaseResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// SQL instances
	instances, _ := g.executeGCloudCommand(ctx, []string{"sql", "instances", "list", "--format", "json"})
	for _, instance := range instances {
		region := getStringValueGCP(instance, "region")
		if g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(instance, "selfLink"),
				Name:     getStringValueGCP(instance, "name"),
				Type:     "gcp_sql_database_instance",
				Provider: "gcp",
				Region:   region,
				Status:   getStringValueGCP(instance, "state"),
				Tags:     extractLabelsGCP(instance),
				Attributes: map[string]interface{}{
					"database_version": getStringValueGCP(instance, "databaseVersion"),
					"instance_type":    getStringValueGCP(instance, "instanceType"),
					"tier":             getNestedStringValueGCP(instance, "settings.tier"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(instance, "createTime")),
			})
		}
	}

	// Spanner instances
	spanners, _ := g.executeGCloudCommand(ctx, []string{"spanner", "instances", "list", "--format", "json"})
	for _, spanner := range spanners {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(spanner, "name"),
			Name:     extractResourceNameFromPathGCP(getStringValueGCP(spanner, "name")),
			Type:     "gcp_spanner_instance",
			Provider: "gcp",
			Region:   getStringValueGCP(spanner, "config"),
			Status:   getStringValueGCP(spanner, "state"),
			Tags:     extractLabelsGCP(spanner),
			Attributes: map[string]interface{}{
				"node_count":   getIntValueGCP(spanner, "nodeCount"),
				"display_name": getStringValueGCP(spanner, "displayName"),
			},
			CreatedAt: time.Now(), // Spanner doesn't expose creation time in list
		})
	}

	// Firestore databases
	firestores, _ := g.executeGCloudCommand(ctx, []string{"firestore", "databases", "list", "--format", "json"})
	for _, firestore := range firestores {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(firestore, "name"),
			Name:     extractResourceNameFromPathGCP(getStringValueGCP(firestore, "name")),
			Type:     "gcp_firestore_database",
			Provider: "gcp",
			Region:   getStringValueGCP(firestore, "locationId"),
			Status:   getStringValueGCP(firestore, "type"),
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"type":             getStringValueGCP(firestore, "type"),
				"concurrency_mode": getStringValueGCP(firestore, "concurrencyMode"),
			},
			CreatedAt: time.Now(),
		})
	}

	// BigTable instances
	bigtables, _ := g.executeGCloudCommand(ctx, []string{"bigtable", "instances", "list", "--format", "json"})
	for _, bigtable := range bigtables {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(bigtable, "name"),
			Name:     extractResourceNameFromPathGCP(getStringValueGCP(bigtable, "name")),
			Type:     "gcp_bigtable_instance",
			Provider: "gcp",
			Region:   "multi-regional", // BigTable can be multi-regional
			Status:   getStringValueGCP(bigtable, "state"),
			Tags:     extractLabelsGCP(bigtable),
			Attributes: map[string]interface{}{
				"type":         getStringValueGCP(bigtable, "type"),
				"display_name": getStringValueGCP(bigtable, "displayName"),
			},
			CreatedAt: time.Now(),
		})
	}

	return resources, nil
}

// discoverNetworkingResources discovers networking resources
func (g *GCPEnhancedDiscoverer) discoverNetworkingResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// VPC networks
	networks, _ := g.executeGCloudCommand(ctx, []string{"compute", "networks", "list", "--format", "json"})
	for _, network := range networks {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(network, "selfLink"),
			Name:     getStringValueGCP(network, "name"),
			Type:     "gcp_compute_network",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(network),
			Attributes: map[string]interface{}{
				"routing_mode":            getStringValueGCP(network, "routingConfig.routingMode"),
				"auto_create_subnetworks": getBoolValueGCP(network, "autoCreateSubnetworks"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(network, "creationTimestamp")),
		})
	}

	// Subnets
	subnets, _ := g.executeGCloudCommand(ctx, []string{"compute", "networks", "subnets", "list", "--format", "json"})
	for _, subnet := range subnets {
		region := getStringValueGCP(subnet, "region")
		if g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(subnet, "selfLink"),
				Name:     getStringValueGCP(subnet, "name"),
				Type:     "gcp_compute_subnetwork",
				Provider: "gcp",
				Region:   extractRegionFromPath(region),
				Status:   "active",
				Tags:     extractLabelsGCP(subnet),
				Attributes: map[string]interface{}{
					"ip_cidr_range": getStringValueGCP(subnet, "ipCidrRange"),
					"network":       getStringValueGCP(subnet, "network"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(subnet, "creationTimestamp")),
			})
		}
	}

	// Firewalls
	firewalls, _ := g.executeGCloudCommand(ctx, []string{"compute", "firewall-rules", "list", "--format", "json"})
	for _, firewall := range firewalls {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(firewall, "selfLink"),
			Name:     getStringValueGCP(firewall, "name"),
			Type:     "gcp_compute_firewall",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(firewall),
			Attributes: map[string]interface{}{
				"direction": getStringValueGCP(firewall, "direction"),
				"priority":  getIntValueGCP(firewall, "priority"),
				"network":   getStringValueGCP(firewall, "network"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(firewall, "creationTimestamp")),
		})
	}

	// Load balancers (HTTP/HTTPS)
	lbs, _ := g.executeGCloudCommand(ctx, []string{"compute", "url-maps", "list", "--format", "json"})
	for _, lb := range lbs {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(lb, "selfLink"),
			Name:     getStringValueGCP(lb, "name"),
			Type:     "gcp_compute_url_map",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(lb),
			Attributes: map[string]interface{}{
				"default_service": getStringValueGCP(lb, "defaultService"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(lb, "creationTimestamp")),
		})
	}

	// Backend services
	backends, _ := g.executeGCloudCommand(ctx, []string{"compute", "backend-services", "list", "--format", "json"})
	for _, backend := range backends {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(backend, "selfLink"),
			Name:     getStringValueGCP(backend, "name"),
			Type:     "gcp_compute_backend_service",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(backend),
			Attributes: map[string]interface{}{
				"protocol":  getStringValueGCP(backend, "protocol"),
				"port_name": getStringValueGCP(backend, "portName"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(backend, "creationTimestamp")),
		})
	}

	// Static IPs
	addresses, _ := g.executeGCloudCommand(ctx, []string{"compute", "addresses", "list", "--format", "json"})
	for _, address := range addresses {
		region := getStringValueGCP(address, "region")
		if region == "" || g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(address, "selfLink"),
				Name:     getStringValueGCP(address, "name"),
				Type:     "gcp_compute_address",
				Provider: "gcp",
				Region:   extractRegionFromPath(region),
				Status:   getStringValueGCP(address, "status"),
				Tags:     extractLabelsGCP(address),
				Attributes: map[string]interface{}{
					"address":      getStringValueGCP(address, "address"),
					"address_type": getStringValueGCP(address, "addressType"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(address, "creationTimestamp")),
			})
		}
	}

	return resources, nil
}

// discoverContainerResources discovers GKE and container resources
func (g *GCPEnhancedDiscoverer) discoverContainerResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// GKE clusters
	clusters, _ := g.executeGCloudCommand(ctx, []string{"container", "clusters", "list", "--format", "json"})
	for _, cluster := range clusters {
		location := getStringValueGCP(cluster, "location")
		if g.matchesRegions(location, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(cluster, "selfLink"),
				Name:     getStringValueGCP(cluster, "name"),
				Type:     "gcp_container_cluster",
				Provider: "gcp",
				Region:   location,
				Status:   getStringValueGCP(cluster, "status"),
				Tags:     extractLabelsGCP(cluster),
				Attributes: map[string]interface{}{
					"initial_node_count":     getIntValueGCP(cluster, "initialNodeCount"),
					"current_master_version": getStringValueGCP(cluster, "currentMasterVersion"),
					"node_version":           getStringValueGCP(cluster, "currentNodeVersion"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(cluster, "createTime")),
			})
		}
	}

	// Cloud Run services
	services, _ := g.executeGCloudCommand(ctx, []string{"run", "services", "list", "--format", "json"})
	for _, service := range services {
		region := getNestedStringValueGCP(service, "metadata.labels.cloud.googleapis.com/location")
		if g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getNestedStringValueGCP(service, "metadata.selfLink"),
				Name:     getNestedStringValueGCP(service, "metadata.name"),
				Type:     "gcp_cloud_run_service",
				Provider: "gcp",
				Region:   region,
				Status:   getNestedStringValueGCP(service, "status.conditions.0.status"),
				Tags:     extractNestedLabelsGCP(service, "metadata.labels"),
				Attributes: map[string]interface{}{
					"traffic": getNestedValueGCP(service, "status.traffic"),
				},
				CreatedAt: parseGCPTimeEnhanced(getNestedStringValueGCP(service, "metadata.creationTimestamp")),
			})
		}
	}

	return resources, nil
}

// discoverFunctionResources discovers Cloud Functions
func (g *GCPEnhancedDiscoverer) discoverFunctionResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Cloud Functions
	functions, _ := g.executeGCloudCommand(ctx, []string{"functions", "list", "--format", "json"})
	for _, function := range functions {
		region := extractRegionFromPath(getStringValueGCP(function, "name"))
		if g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(function, "name"),
				Name:     extractResourceNameFromPathGCP(getStringValueGCP(function, "name")),
				Type:     "gcp_cloud_function",
				Provider: "gcp",
				Region:   region,
				Status:   getStringValueGCP(function, "status"),
				Tags:     extractLabelsGCP(function),
				Attributes: map[string]interface{}{
					"runtime":     getStringValueGCP(function, "runtime"),
					"entry_point": getStringValueGCP(function, "entryPoint"),
					"trigger":     getNestedValueGCP(function, "eventTrigger"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(function, "updateTime")),
			})
		}
	}

	return resources, nil
}

// discoverBigDataResources discovers BigQuery and data analytics resources
func (g *GCPEnhancedDiscoverer) discoverBigDataResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// BigQuery datasets
	datasets, _ := g.executeGCloudCommand(ctx, []string{"bq", "ls", "--format", "json"})
	for _, dataset := range datasets {
		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("projects/%s/datasets/%s", g.projectID, getStringValueGCP(dataset, "datasetReference.datasetId")),
			Name:     getStringValueGCP(dataset, "datasetReference.datasetId"),
			Type:     "gcp_bigquery_dataset",
			Provider: "gcp",
			Region:   getStringValueGCP(dataset, "location"),
			Status:   "active",
			Tags:     extractLabelsGCP(dataset),
			Attributes: map[string]interface{}{
				"creation_time": getStringValueGCP(dataset, "creationTime"),
				"last_modified": getStringValueGCP(dataset, "lastModifiedTime"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(dataset, "creationTime")),
		})
	}

	// Dataflow jobs
	jobs, _ := g.executeGCloudCommand(ctx, []string{"dataflow", "jobs", "list", "--format", "json"})
	for _, job := range jobs {
		region := getStringValueGCP(job, "location")
		if g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(job, "id"),
				Name:     getStringValueGCP(job, "name"),
				Type:     "gcp_dataflow_job",
				Provider: "gcp",
				Region:   region,
				Status:   getStringValueGCP(job, "currentState"),
				Tags:     extractLabelsGCP(job),
				Attributes: map[string]interface{}{
					"type": getStringValueGCP(job, "type"),
					"sdk":  getNestedStringValueGCP(job, "environment.sdkPipelineOptions.options.sdkHarnessType"),
				},
				CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(job, "createTime")),
			})
		}
	}

	// Pub/Sub topics
	topics, _ := g.executeGCloudCommand(ctx, []string{"pubsub", "topics", "list", "--format", "json"})
	for _, topic := range topics {
		resources = append(resources, models.Resource{
			ID:         getStringValueGCP(topic, "name"),
			Name:       extractResourceNameFromPathGCP(getStringValueGCP(topic, "name")),
			Type:       "gcp_pubsub_topic",
			Provider:   "gcp",
			Region:     "global",
			Status:     "active",
			Tags:       extractLabelsGCP(topic),
			Attributes: make(map[string]interface{}),
			CreatedAt:  time.Now(),
		})
	}

	// Pub/Sub subscriptions
	subscriptions, _ := g.executeGCloudCommand(ctx, []string{"pubsub", "subscriptions", "list", "--format", "json"})
	for _, sub := range subscriptions {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(sub, "name"),
			Name:     extractResourceNameFromPathGCP(getStringValueGCP(sub, "name")),
			Type:     "gcp_pubsub_subscription",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(sub),
			Attributes: map[string]interface{}{
				"topic": getStringValueGCP(sub, "topic"),
			},
			CreatedAt: time.Now(),
		})
	}

	return resources, nil
}

// discoverSecurityResources discovers IAM and security resources
func (g *GCPEnhancedDiscoverer) discoverSecurityResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Service accounts
	serviceAccounts, _ := g.executeGCloudCommand(ctx, []string{"iam", "service-accounts", "list", "--format", "json"})
	for _, sa := range serviceAccounts {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(sa, "name"),
			Name:     getStringValueGCP(sa, "displayName"),
			Type:     "gcp_service_account",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"email":     getStringValueGCP(sa, "email"),
				"unique_id": getStringValueGCP(sa, "uniqueId"),
			},
			CreatedAt: time.Now(),
		})
	}

	// KMS keyrings
	keyrings, _ := g.executeGCloudCommand(ctx, []string{"kms", "keyrings", "list", "--format", "json", "--location", "global"})
	for _, keyring := range keyrings {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(keyring, "name"),
			Name:     extractResourceNameFromPathGCP(getStringValueGCP(keyring, "name")),
			Type:     "gcp_kms_key_ring",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     make(map[string]string),
			Attributes: map[string]interface{}{
				"location": "global",
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(keyring, "createTime")),
		})
	}

	// Secret Manager secrets
	secrets, _ := g.executeGCloudCommand(ctx, []string{"secrets", "list", "--format", "json"})
	for _, secret := range secrets {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(secret, "name"),
			Name:     extractResourceNameFromPathGCP(getStringValueGCP(secret, "name")),
			Type:     "gcp_secret_manager_secret",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(secret),
			Attributes: map[string]interface{}{
				"replication": getNestedValueGCP(secret, "replication"),
			},
			CreatedAt: parseGCPTimeEnhanced(getStringValueGCP(secret, "createTime")),
		})
	}

	return resources, nil
}

// discoverAIMLResources discovers AI/ML resources
func (g *GCPEnhancedDiscoverer) discoverAIMLResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// AI Platform models
	models_ai, _ := g.executeGCloudCommand(ctx, []string{"ai-platform", "models", "list", "--format", "json"})
	for _, model := range models_ai {
		resources = append(resources, models.Resource{
			ID:       getStringValueGCP(model, "name"),
			Name:     extractResourceNameFromPathGCP(getStringValueGCP(model, "name")),
			Type:     "gcp_ml_engine_model",
			Provider: "gcp",
			Region:   "global",
			Status:   "active",
			Tags:     extractLabelsGCP(model),
			Attributes: map[string]interface{}{
				"description": getStringValueGCP(model, "description"),
			},
			CreatedAt: time.Now(),
		})
	}

	return resources, nil
}

// discoverManagementResources discovers monitoring and management resources
func (g *GCPEnhancedDiscoverer) discoverManagementResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Cloud Scheduler jobs
	jobs, _ := g.executeGCloudCommand(ctx, []string{"scheduler", "jobs", "list", "--format", "json"})
	for _, job := range jobs {
		region := extractRegionFromPath(getStringValueGCP(job, "name"))
		if g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(job, "name"),
				Name:     extractResourceNameFromPathGCP(getStringValueGCP(job, "name")),
				Type:     "gcp_cloud_scheduler_job",
				Provider: "gcp",
				Region:   region,
				Status:   getStringValueGCP(job, "state"),
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"schedule": getStringValueGCP(job, "schedule"),
					"timezone": getStringValueGCP(job, "timeZone"),
				},
				CreatedAt: time.Now(),
			})
		}
	}

	// Cloud Tasks queues
	queues, _ := g.executeGCloudCommand(ctx, []string{"tasks", "queues", "list", "--format", "json"})
	for _, queue := range queues {
		region := extractRegionFromPath(getStringValueGCP(queue, "name"))
		if g.matchesRegions(region, regions) {
			resources = append(resources, models.Resource{
				ID:       getStringValueGCP(queue, "name"),
				Name:     extractResourceNameFromPathGCP(getStringValueGCP(queue, "name")),
				Type:     "gcp_cloud_tasks_queue",
				Provider: "gcp",
				Region:   region,
				Status:   getStringValueGCP(queue, "state"),
				Tags:     make(map[string]string),
				Attributes: map[string]interface{}{
					"rate_limits": getNestedValueGCP(queue, "rateLimits"),
				},
				CreatedAt: time.Now(),
			})
		}
	}

	return resources, nil
}

// Helper functions

func (g *GCPEnhancedDiscoverer) executeGCloudCommand(ctx context.Context, args []string) ([]map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, g.cliPath, args...)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to execute gcloud command: %v", err)
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		log.Printf("Warning: Failed to parse gcloud output: %v", err)
		return nil, err
	}

	return result, nil
}

func (g *GCPEnhancedDiscoverer) matchesRegions(resourceRegion string, targetRegions []string) bool {
	if len(targetRegions) == 0 {
		return true
	}

	resourceRegion = strings.ToLower(resourceRegion)
	for _, region := range targetRegions {
		if strings.Contains(resourceRegion, strings.ToLower(region)) {
			return true
		}
	}
	return false
}

func extractRegionFromZone(zone string) string {
	parts := strings.Split(zone, "/")
	zoneName := parts[len(parts)-1]
	if len(zoneName) > 2 {
		return zoneName[:len(zoneName)-2] // Remove last 2 chars (zone suffix)
	}
	return zoneName
}

func extractRegionFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "regions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "global"
}

func extractResourceNameFromPathGCP(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func getStringValueGCP(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntValueGCP(data map[string]interface{}, key string) int64 {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		}
	}
	return 0
}

func getBoolValueGCP(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getNestedValueGCP(data map[string]interface{}, path string) interface{} {
	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		if val, ok := current[key]; ok {
			if next, ok := val.(map[string]interface{}); ok {
				current = next
			} else {
				return val
			}
		} else {
			return nil
		}
	}
	return current
}

func getNestedStringValueGCP(data map[string]interface{}, path string) string {
	if val := getNestedValueGCP(data, path); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func extractLabelsGCP(data map[string]interface{}) map[string]string {
	labels := make(map[string]string)
	if labelsData, ok := data["labels"].(map[string]interface{}); ok {
		for k, v := range labelsData {
			if str, ok := v.(string); ok {
				labels[k] = str
			}
		}
	}
	return labels
}

func extractNestedLabelsGCP(data map[string]interface{}, path string) map[string]string {
	labels := make(map[string]string)
	if labelsData := getNestedValueGCP(data, path); labelsData != nil {
		if labelMap, ok := labelsData.(map[string]interface{}); ok {
			for k, v := range labelMap {
				if str, ok := v.(string); ok {
					labels[k] = str
				}
			}
		}
	}
	return labels
}

func parseGCPTimeEnhanced(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// GCP uses RFC3339 format
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t
	}

	// Alternative format
	if t, err := time.Parse("2006-01-02T15:04:05.000Z", timeStr); err == nil {
		return t
	}

	return time.Now()
}
