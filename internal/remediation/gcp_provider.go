package remediation

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/dataproc/v2/apiv1"
	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/catherinevee/driftmgr/pkg/models"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/sqladmin/v1"
)

// GCPProvider implements CloudProvider for Google Cloud Platform
type GCPProvider struct {
	projectID        string
	storageClient    *storage.Client
	resourceManager  *cloudresourcemanager.Service
	containerService *container.Service
	computeService   *compute.Service
	sqlService       *sqladmin.Service
	pubsubClient     *pubsub.Client
	functionsService *cloudfunctions.Service
	dataprocClient   *dataproc.ClusterControllerClient
	bigqueryClient   *bigquery.Client
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

	// Initialize compute service
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	// Initialize SQL Admin service
	sqlService, err := sqladmin.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create sql admin service: %w", err)
	}

	// Initialize Pub/Sub client
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	// Initialize Cloud Functions service
	functionsService, err := cloudfunctions.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create functions service: %w", err)
	}

	// Initialize Dataproc client
	dataprocClient, err := dataproc.NewClusterControllerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataproc client: %w", err)
	}

	// Initialize BigQuery client
	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bigquery client: %w", err)
	}

	return &GCPProvider{
		projectID:        projectID,
		storageClient:    storageClient,
		resourceManager:  resourceManager,
		containerService: containerService,
		computeService:   computeService,
		sqlService:       sqlService,
		pubsubClient:     pubsubClient,
		functionsService: functionsService,
		dataprocClient:   dataprocClient,
		bigqueryClient:   bigqueryClient,
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

	// List all instances across all zones
	aggregatedList, err := gp.computeService.Instances.AggregatedList(gp.projectID).Context(ctx).Do()
	if err != nil {
		return resources, fmt.Errorf("failed to list compute instances: %w", err)
	}

	for zone, instancesScopedList := range aggregatedList.Items {
		// Extract zone name from the key
		zoneParts := strings.Split(zone, "/")
		zoneName := zoneParts[len(zoneParts)-1]

		for _, instance := range instancesScopedList.Instances {
			// Parse creation timestamp
			createdTime := time.Now()
			if instance.CreationTimestamp != "" {
				if parsed, err := time.Parse(time.RFC3339, instance.CreationTimestamp); err == nil {
					createdTime = parsed
				}
			}

			// Extract network interfaces
			var networkInterfaces []map[string]interface{}
			for _, nic := range instance.NetworkInterfaces {
				nicData := map[string]interface{}{
					"network":     nic.Network,
					"subnetwork":  nic.Subnetwork,
					"internal_ip": nic.NetworkIP,
				}
				if len(nic.AccessConfigs) > 0 && nic.AccessConfigs[0].NatIP != "" {
					nicData["external_ip"] = nic.AccessConfigs[0].NatIP
				}
				networkInterfaces = append(networkInterfaces, nicData)
			}

			// Extract disks
			var disks []string
			for _, disk := range instance.Disks {
				if disk.Source != "" {
					disks = append(disks, disk.Source)
				}
			}

			resources = append(resources, models.Resource{
				ID:       fmt.Sprintf("%d", instance.Id),
				Name:     instance.Name,
				Type:     "compute.googleapis.com/instances",
				Provider: "gcp",
				Region:   zoneName,
				State:    instance.Status,
				Created:  createdTime,
				Metadata: map[string]string{
					"zone":                zoneName,
					"machine_type":        instance.MachineType,
					"status":              instance.Status,
					"network_interfaces":  fmt.Sprintf("%v", networkInterfaces),
					"disks":               fmt.Sprintf("%v", disks),
					"tags":                fmt.Sprintf("%v", instance.Tags),
					"labels":              fmt.Sprintf("%v", instance.Labels),
					"can_ip_forward":      fmt.Sprintf("%v", instance.CanIpForward),
					"deletion_protection": fmt.Sprintf("%v", instance.DeletionProtection),
				},
			})
		}
	}

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

	// List all VPC networks
	networkList, err := gp.computeService.Networks.List(gp.projectID).Context(ctx).Do()
	if err != nil {
		return resources, fmt.Errorf("failed to list VPC networks: %w", err)
	}

	for _, network := range networkList.Items {
		// Parse creation timestamp
		createdTime := time.Now()
		if network.CreationTimestamp != "" {
			if parsed, err := time.Parse(time.RFC3339, network.CreationTimestamp); err == nil {
				createdTime = parsed
			}
		}

		// List subnetworks for this network
		var subnetworks []string
		for _, subnet := range network.Subnetworks {
			subnetworks = append(subnetworks, subnet)
		}

		// Get peerings
		var peerings []map[string]interface{}
		for _, peering := range network.Peerings {
			peeringData := map[string]interface{}{
				"name":               peering.Name,
				"network":            peering.Network,
				"state":              peering.State,
				"auto_create_routes": peering.AutoCreateRoutes,
			}
			peerings = append(peerings, peeringData)
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%d", network.Id),
			Name:     network.Name,
			Type:     "compute.googleapis.com/networks",
			Provider: "gcp",
			Region:   "global",
			State:    "ACTIVE",
			Created:  createdTime,
			Metadata: map[string]string{
				"auto_create_subnetworks": fmt.Sprintf("%v", network.AutoCreateSubnetworks),
				"mtu":                     fmt.Sprintf("%d", network.Mtu),
				"routing_mode":            fmt.Sprintf("%v", network.RoutingConfig),
				"subnetworks":             fmt.Sprintf("%v", subnetworks),
				"peerings":                fmt.Sprintf("%v", peerings),
				"description":             network.Description,
			},
		})
	}

	// Also discover subnetworks
	subnetList, err := gp.computeService.Subnetworks.AggregatedList(gp.projectID).Context(ctx).Do()
	if err == nil {
		for region, subnetsScopedList := range subnetList.Items {
			// Extract region name
			regionParts := strings.Split(region, "/")
			regionName := regionParts[len(regionParts)-1]

			for _, subnet := range subnetsScopedList.Subnetworks {
				// Parse creation timestamp
				createdTime := time.Now()
				if subnet.CreationTimestamp != "" {
					if parsed, err := time.Parse(time.RFC3339, subnet.CreationTimestamp); err == nil {
						createdTime = parsed
					}
				}

				resources = append(resources, models.Resource{
					ID:       fmt.Sprintf("%d", subnet.Id),
					Name:     subnet.Name,
					Type:     "compute.googleapis.com/subnetworks",
					Provider: "gcp",
					Region:   regionName,
					State:    "ACTIVE",
					Created:  createdTime,
					Metadata: map[string]string{
						"network":                  subnet.Network,
						"ip_cidr_range":            subnet.IpCidrRange,
						"gateway_address":          subnet.GatewayAddress,
						"private_ip_google_access": fmt.Sprintf("%v", subnet.PrivateIpGoogleAccess),
						"enable_flow_logs":         fmt.Sprintf("%v", subnet.EnableFlowLogs),
						"purpose":                  subnet.Purpose,
						"role":                     subnet.Role,
					},
				})
			}
		}
	}

	return resources, nil
}

func (gp *GCPProvider) discoverLoadBalancers(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover Global Forwarding Rules (Global Load Balancers)
	globalForwardingRules, err := gp.computeService.GlobalForwardingRules.List(gp.projectID).Context(ctx).Do()
	if err == nil {
		for _, rule := range globalForwardingRules.Items {
			// Parse creation timestamp
			createdTime := time.Now()
			if rule.CreationTimestamp != "" {
				if parsed, err := time.Parse(time.RFC3339, rule.CreationTimestamp); err == nil {
					createdTime = parsed
				}
			}

			resources = append(resources, models.Resource{
				ID:       fmt.Sprintf("%d", rule.Id),
				Name:     rule.Name,
				Type:     "compute.googleapis.com/globalForwardingRules",
				Provider: "gcp",
				Region:   "global",
				State:    "ACTIVE",
				Created:  createdTime,
				Metadata: map[string]string{
					"ip_address":            rule.IPAddress,
					"ip_protocol":           rule.IPProtocol,
					"port_range":            rule.PortRange,
					"target":                rule.Target,
					"load_balancing_scheme": rule.LoadBalancingScheme,
					"network_tier":          rule.NetworkTier,
				},
			})
		}
	}

	// Discover Regional Forwarding Rules
	regionalForwardingRules, err := gp.computeService.ForwardingRules.AggregatedList(gp.projectID).Context(ctx).Do()
	if err == nil {
		for region, forwardingRulesScopedList := range regionalForwardingRules.Items {
			// Extract region name
			regionParts := strings.Split(region, "/")
			regionName := regionParts[len(regionParts)-1]

			for _, rule := range forwardingRulesScopedList.ForwardingRules {
				// Parse creation timestamp
				createdTime := time.Now()
				if rule.CreationTimestamp != "" {
					if parsed, err := time.Parse(time.RFC3339, rule.CreationTimestamp); err == nil {
						createdTime = parsed
					}
				}

				resources = append(resources, models.Resource{
					ID:       fmt.Sprintf("%d", rule.Id),
					Name:     rule.Name,
					Type:     "compute.googleapis.com/forwardingRules",
					Provider: "gcp",
					Region:   regionName,
					State:    "ACTIVE",
					Created:  createdTime,
					Metadata: map[string]string{
						"ip_address":            rule.IPAddress,
						"ip_protocol":           rule.IPProtocol,
						"port_range":            rule.PortRange,
						"target":                rule.Target,
						"backend_service":       rule.BackendService,
						"load_balancing_scheme": rule.LoadBalancingScheme,
						"network":               rule.Network,
						"subnetwork":            rule.Subnetwork,
					},
				})
			}
		}
	}

	// Discover Backend Services
	backendServices, err := gp.computeService.BackendServices.List(gp.projectID).Context(ctx).Do()
	if err == nil {
		for _, service := range backendServices.Items {
			// Parse creation timestamp
			createdTime := time.Now()
			if service.CreationTimestamp != "" {
				if parsed, err := time.Parse(time.RFC3339, service.CreationTimestamp); err == nil {
					createdTime = parsed
				}
			}

			var backends []map[string]interface{}
			for _, backend := range service.Backends {
				backendData := map[string]interface{}{
					"group":           backend.Group,
					"balancing_mode":  backend.BalancingMode,
					"capacity_scaler": backend.CapacityScaler,
					"max_utilization": backend.MaxUtilization,
				}
				backends = append(backends, backendData)
			}

			resources = append(resources, models.Resource{
				ID:       fmt.Sprintf("%d", service.Id),
				Name:     service.Name,
				Type:     "compute.googleapis.com/backendServices",
				Provider: "gcp",
				Region:   "global",
				State:    "ACTIVE",
				Created:  createdTime,
				Metadata: map[string]string{
					"protocol":              service.Protocol,
					"port":                  fmt.Sprintf("%d", service.Port),
					"port_name":             service.PortName,
					"timeout_sec":           fmt.Sprintf("%d", service.TimeoutSec),
					"enable_cdn":            fmt.Sprintf("%v", service.EnableCDN),
					"session_affinity":      service.SessionAffinity,
					"affinity_cookie_ttl":   fmt.Sprintf("%d", service.AffinityCookieTtlSec),
					"load_balancing_scheme": service.LoadBalancingScheme,
					"backends":              fmt.Sprintf("%v", backends),
					"health_checks":         fmt.Sprintf("%v", service.HealthChecks),
				},
			})
		}
	}

	// Discover Target Pools
	targetPools, err := gp.computeService.TargetPools.AggregatedList(gp.projectID).Context(ctx).Do()
	if err == nil {
		for region, targetPoolsScopedList := range targetPools.Items {
			// Extract region name
			regionParts := strings.Split(region, "/")
			regionName := regionParts[len(regionParts)-1]

			for _, pool := range targetPoolsScopedList.TargetPools {
				// Parse creation timestamp
				createdTime := time.Now()
				if pool.CreationTimestamp != "" {
					if parsed, err := time.Parse(time.RFC3339, pool.CreationTimestamp); err == nil {
						createdTime = parsed
					}
				}

				resources = append(resources, models.Resource{
					ID:       fmt.Sprintf("%d", pool.Id),
					Name:     pool.Name,
					Type:     "compute.googleapis.com/targetPools",
					Provider: "gcp",
					Region:   regionName,
					State:    "ACTIVE",
					Created:  createdTime,
					Metadata: map[string]string{
						"instances":        fmt.Sprintf("%v", pool.Instances),
						"health_checks":    fmt.Sprintf("%v", pool.HealthChecks),
						"session_affinity": pool.SessionAffinity,
						"failover_ratio":   fmt.Sprintf("%f", pool.FailoverRatio),
						"backup_pool":      pool.BackupPool,
					},
				})
			}
		}
	}

	return resources, nil
}

func (gp *GCPProvider) discoverSQLInstances(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// List all SQL instances
	instancesList, err := gp.sqlService.Instances.List(gp.projectID).Context(ctx).Do()
	if err != nil {
		return resources, fmt.Errorf("failed to list SQL instances: %w", err)
	}

	for _, instance := range instancesList.Items {
		// Parse creation time if available
		createdTime := time.Now()
		if instance.ServerCaCert != nil && instance.ServerCaCert.CreateTime != "" {
			if parsed, err := time.Parse(time.RFC3339, instance.ServerCaCert.CreateTime); err == nil {
				createdTime = parsed
			}
		}

		// Extract IP addresses
		var ipAddresses []map[string]interface{}
		for _, ip := range instance.IpAddresses {
			ipData := map[string]interface{}{
				"type":       ip.Type,
				"ip_address": ip.IpAddress,
			}
			ipAddresses = append(ipAddresses, ipData)
		}

		// Extract replica configuration if exists
		var replicaConfig map[string]interface{}
		if instance.ReplicaConfiguration != nil {
			replicaConfig = map[string]interface{}{
				"kind":                        instance.ReplicaConfiguration.Kind,
				"mysql_replica_configuration": instance.ReplicaConfiguration.MysqlReplicaConfiguration,
				"failover_target":             instance.ReplicaConfiguration.FailoverTarget,
			}
		}

		// Extract backup configuration
		var backupConfig map[string]interface{}
		if instance.Settings != nil && instance.Settings.BackupConfiguration != nil {
			backupConfig = map[string]interface{}{
				"enabled":                        instance.Settings.BackupConfiguration.Enabled,
				"start_time":                     instance.Settings.BackupConfiguration.StartTime,
				"point_in_time_recovery_enabled": instance.Settings.BackupConfiguration.PointInTimeRecoveryEnabled,
				"transaction_log_retention_days": instance.Settings.BackupConfiguration.TransactionLogRetentionDays,
				"location":                       instance.Settings.BackupConfiguration.Location,
			}
		}

		resources = append(resources, models.Resource{
			ID:       instance.Name,
			Name:     instance.Name,
			Type:     "sqladmin.googleapis.com/instances",
			Provider: "gcp",
			Region:   instance.Region,
			State:    instance.State,
			Created:  createdTime,
			Metadata: map[string]string{
				"database_version":       instance.DatabaseVersion,
				"tier":                   instance.Settings.Tier,
				"backend_type":           instance.BackendType,
				"connection_name":        instance.ConnectionName,
				"instance_type":          instance.InstanceType,
				"master_instance_name":   instance.MasterInstanceName,
				"max_disk_size":          fmt.Sprintf("%d", instance.MaxDiskSize),
				"current_disk_size":      fmt.Sprintf("%d", instance.CurrentDiskSize),
				"ip_addresses":           fmt.Sprintf("%v", ipAddresses),
				"replica_configuration":  fmt.Sprintf("%v", replicaConfig),
				"backup_configuration":   fmt.Sprintf("%v", backupConfig),
				"maintenance_version":    instance.MaintenanceVersion,
				"disk_encryption_status": fmt.Sprintf("%v", instance.DiskEncryptionStatus),
			},
		})
	}

	return resources, nil
}

func (gp *GCPProvider) discoverPubSubTopics(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// Get all topics in the project
	it := gp.pubsubClient.Topics(ctx)
	for {
		topic, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list Pub/Sub topics: %w", err)
		}

		// Get topic configuration
		config, err := topic.Config(ctx)
		if err != nil {
			log.Printf("Failed to get config for topic %s: %v", topic.ID(), err)
			continue
		}

		// Get subscriptions for this topic
		var subscriptions []string
		subIt := topic.Subscriptions(ctx)
		for {
			sub, err := subIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to list subscriptions for topic %s: %v", topic.ID(), err)
				break
			}
			subscriptions = append(subscriptions, sub.ID())
		}

		resources = append(resources, models.Resource{
			ID:       topic.ID(),
			Name:     topic.ID(),
			Type:     "pubsub.googleapis.com/topics",
			Provider: "gcp",
			Region:   "global",
			State:    "ACTIVE",
			Created:  time.Now(), // PubSub API doesn't provide creation time
			Metadata: map[string]string{
				"message_storage_policy": fmt.Sprintf("%v", config.MessageStoragePolicy),
				"kms_key_name":           config.KMSKeyName,
				"schema_settings":        fmt.Sprintf("%v", config.SchemaSettings),
				"retention_duration":     fmt.Sprintf("%v", config.RetentionDuration),
				"subscriptions":          fmt.Sprintf("%v", subscriptions),
				"labels":                 fmt.Sprintf("%v", config.Labels),
			},
		})
	}

	// Also discover subscriptions
	subIt := gp.pubsubClient.Subscriptions(ctx)
	for {
		sub, err := subIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list Pub/Sub subscriptions: %w", err)
		}

		// Get subscription configuration
		config, err := sub.Config(ctx)
		if err != nil {
			log.Printf("Failed to get config for subscription %s: %v", sub.ID(), err)
			continue
		}

		resources = append(resources, models.Resource{
			ID:       sub.ID(),
			Name:     sub.ID(),
			Type:     "pubsub.googleapis.com/subscriptions",
			Provider: "gcp",
			Region:   "global",
			State:    "ACTIVE",
			Created:  time.Now(), // PubSub API doesn't provide creation time
			Metadata: map[string]string{
				"topic":                      config.Topic.ID(),
				"ack_deadline":               fmt.Sprintf("%v", config.AckDeadline),
				"retain_acked_messages":      fmt.Sprintf("%v", config.RetainAckedMessages),
				"message_retention_duration": fmt.Sprintf("%v", config.RetentionDuration),
				"enable_message_ordering":    fmt.Sprintf("%v", config.EnableMessageOrdering),
				"dead_letter_policy":         fmt.Sprintf("%v", config.DeadLetterPolicy),
				"retry_policy":               fmt.Sprintf("%v", config.RetryPolicy),
				"push_config":                fmt.Sprintf("%v", config.PushConfig),
				"labels":                     fmt.Sprintf("%v", config.Labels),
			},
		})
	}

	return resources, nil
}

func (gp *GCPProvider) discoverCloudFunctions(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// List all locations first
	locationsResp, err := gp.functionsService.Projects.Locations.List(fmt.Sprintf("projects/%s", gp.projectID)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list Cloud Functions locations: %w", err)
	}

	// Iterate through each location to find functions
	for _, location := range locationsResp.Locations {
		// List functions in this location
		functionsResp, err := gp.functionsService.Projects.Locations.Functions.List(location.Name).Context(ctx).Do()
		if err != nil {
			log.Printf("Failed to list functions in location %s: %v", location.Name, err)
			continue
		}

		for _, function := range functionsResp.Functions {
			// Parse update time
			updatedTime := time.Now()
			if function.UpdateTime != "" {
				if parsed, err := time.Parse(time.RFC3339, function.UpdateTime); err == nil {
					updatedTime = parsed
				}
			}

			// Determine state from status
			state := "UNKNOWN"
			switch function.Status {
			case "ACTIVE":
				state = "ACTIVE"
			case "OFFLINE":
				state = "OFFLINE"
			case "DEPLOY_IN_PROGRESS":
				state = "DEPLOYING"
			case "DELETE_IN_PROGRESS":
				state = "DELETING"
			}

			// Extract region from location name
			locationParts := strings.Split(location.Name, "/")
			region := "global"
			if len(locationParts) >= 4 {
				region = locationParts[3]
			}

			resources = append(resources, models.Resource{
				ID:       function.Name,
				Name:     function.Name,
				Type:     "cloudfunctions.googleapis.com/functions",
				Provider: "gcp",
				Region:   region,
				State:    state,
				Created:  updatedTime, // Using update time as creation time is not available
				Metadata: map[string]string{
					"entry_point":           function.EntryPoint,
					"timeout":               function.Timeout,
					"available_memory_mb":   fmt.Sprintf("%d", function.AvailableMemoryMb),
					"runtime":               function.Runtime,
					"source_archive_url":    function.SourceArchiveUrl,
					"source_repository":     fmt.Sprintf("%v", function.SourceRepository),
					"event_trigger":         fmt.Sprintf("%v", function.EventTrigger),
					"https_trigger":         fmt.Sprintf("%v", function.HttpsTrigger),
					"service_account":       function.ServiceAccountEmail,
					"vpc_connector":         function.VpcConnector,
					"ingress_settings":      function.IngressSettings,
					"max_instances":         fmt.Sprintf("%d", function.MaxInstances),
					"min_instances":         fmt.Sprintf("%d", function.MinInstances),
					"environment_variables": fmt.Sprintf("%v", function.EnvironmentVariables),
					"labels":                fmt.Sprintf("%v", function.Labels),
					"version_id":            fmt.Sprintf("%d", function.VersionId),
					"build_id":              function.BuildId,
				},
			})
		}
	}

	return resources, nil
}

func (gp *GCPProvider) discoverDataprocClusters(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// List all regions (common Dataproc regions)
	regions := []string{
		"us-central1", "us-east1", "us-west1", "us-west2",
		"europe-west1", "europe-west2", "europe-west3", "europe-west4",
		"asia-east1", "asia-northeast1", "asia-southeast1",
	}

	for _, region := range regions {
		// Create list request
		req := &dataprocpb.ListClustersRequest{
			ProjectId: gp.projectID,
			Region:    region,
		}

		// List clusters in this region
		it := gp.dataprocClient.ListClusters(ctx, req)
		for {
			cluster, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to list Dataproc clusters in region %s: %v", region, err)
				break
			}

			// Determine state from cluster status
			state := "UNKNOWN"
			if cluster.Status != nil {
				switch cluster.Status.State {
				case dataprocpb.ClusterStatus_RUNNING:
					state = "RUNNING"
				case dataprocpb.ClusterStatus_CREATING:
					state = "CREATING"
				case dataprocpb.ClusterStatus_ERROR:
					state = "ERROR"
				case dataprocpb.ClusterStatus_DELETING:
					state = "DELETING"
				case dataprocpb.ClusterStatus_UPDATING:
					state = "UPDATING"
				case dataprocpb.ClusterStatus_STOPPING:
					state = "STOPPING"
				case dataprocpb.ClusterStatus_STOPPED:
					state = "STOPPED"
				case dataprocpb.ClusterStatus_STARTING:
					state = "STARTING"
				}
			}

			// Parse creation time from status
			createdTime := time.Now()
			if cluster.Status != nil && cluster.Status.StateStartTime != nil {
				createdTime = cluster.Status.StateStartTime.AsTime()
			}

			// Extract master and worker configuration
			masterConfig := make(map[string]interface{})
			workerConfig := make(map[string]interface{})
			secondaryWorkerConfig := make(map[string]interface{})

			if cluster.Config != nil {
				if cluster.Config.MasterConfig != nil {
					masterConfig = map[string]interface{}{
						"num_instances":  cluster.Config.MasterConfig.NumInstances,
						"machine_type":   cluster.Config.MasterConfig.MachineTypeUri,
						"disk_size_gb":   cluster.Config.MasterConfig.DiskConfig.GetBootDiskSizeGb(),
						"disk_type":      cluster.Config.MasterConfig.DiskConfig.GetBootDiskType(),
						"preemptibility": cluster.Config.MasterConfig.Preemptibility,
					}
				}
				if cluster.Config.WorkerConfig != nil {
					workerConfig = map[string]interface{}{
						"num_instances":  cluster.Config.WorkerConfig.NumInstances,
						"machine_type":   cluster.Config.WorkerConfig.MachineTypeUri,
						"disk_size_gb":   cluster.Config.WorkerConfig.DiskConfig.GetBootDiskSizeGb(),
						"disk_type":      cluster.Config.WorkerConfig.DiskConfig.GetBootDiskType(),
						"preemptibility": cluster.Config.WorkerConfig.Preemptibility,
					}
				}
				if cluster.Config.SecondaryWorkerConfig != nil {
					secondaryWorkerConfig = map[string]interface{}{
						"num_instances":  cluster.Config.SecondaryWorkerConfig.NumInstances,
						"machine_type":   cluster.Config.SecondaryWorkerConfig.MachineTypeUri,
						"disk_size_gb":   cluster.Config.SecondaryWorkerConfig.DiskConfig.GetBootDiskSizeGb(),
						"disk_type":      cluster.Config.SecondaryWorkerConfig.DiskConfig.GetBootDiskType(),
						"preemptibility": cluster.Config.SecondaryWorkerConfig.Preemptibility,
					}
				}
			}

			resources = append(resources, models.Resource{
				ID:       cluster.ClusterName,
				Name:     cluster.ClusterName,
				Type:     "dataproc.googleapis.com/clusters",
				Provider: "gcp",
				Region:   region,
				State:    state,
				Created:  createdTime,
				Metadata: map[string]string{
					"cluster_uuid":            cluster.ClusterUuid,
					"project_id":              cluster.ProjectId,
					"master_config":           fmt.Sprintf("%v", masterConfig),
					"worker_config":           fmt.Sprintf("%v", workerConfig),
					"secondary_worker_config": fmt.Sprintf("%v", secondaryWorkerConfig),
					"software_config":         fmt.Sprintf("%v", cluster.Config.GetSoftwareConfig()),
					"lifecycle_config":        fmt.Sprintf("%v", cluster.Config.GetLifecycleConfig()),
					"autoscaling_config":      fmt.Sprintf("%v", cluster.Config.GetAutoscalingConfig()),
					"security_config":         fmt.Sprintf("%v", cluster.Config.GetSecurityConfig()),
					"labels":                  fmt.Sprintf("%v", cluster.Labels),
					"metrics":                 fmt.Sprintf("%v", cluster.Metrics),
				},
			})
		}
	}

	return resources, nil
}

func (gp *GCPProvider) discoverBigQueryDatasets(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// List all datasets in the project
	it := gp.bigqueryClient.Datasets(ctx)
	for {
		dataset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list BigQuery datasets: %w", err)
		}

		// Get dataset metadata
		metadata, err := dataset.Metadata(ctx)
		if err != nil {
			log.Printf("Failed to get metadata for dataset %s: %v", dataset.DatasetID, err)
			continue
		}

		// Parse creation time
		createdTime := time.Now()
		if metadata.CreationTime.Unix() > 0 {
			createdTime = metadata.CreationTime
		}

		// Get table count in dataset
		tableCount := 0
		tableIt := dataset.Tables(ctx)
		for {
			_, err := tableIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to count tables in dataset %s: %v", dataset.DatasetID, err)
				break
			}
			tableCount++
		}

		// Convert access entries to simpler format
		var accessList []map[string]interface{}
		for _, access := range metadata.Access {
			accessEntry := make(map[string]interface{})
			if access.Entity != "" {
				accessEntry["entity"] = access.Entity
			}
			if access.EntityType != 0 {
				accessEntry["entity_type"] = access.EntityType
			}
			if access.Role != "" {
				accessEntry["role"] = string(access.Role)
			}
			if access.View != nil {
				// View FullID may not be available
				accessEntry["view"] = fmt.Sprintf("%v", access.View)
			}
			accessList = append(accessList, accessEntry)
		}

		resources = append(resources, models.Resource{
			ID:       dataset.DatasetID,
			Name:     dataset.DatasetID,
			Type:     "bigquery.googleapis.com/datasets",
			Provider: "gcp",
			Region:   metadata.Location,
			State:    "ACTIVE",
			Created:  createdTime,
			Metadata: map[string]string{
				"description":                  metadata.Description,
				"friendly_name":                metadata.Name,
				"location":                     metadata.Location,
				"default_table_expiration":     fmt.Sprintf("%v", metadata.DefaultTableExpiration),
				"default_partition_expiration": fmt.Sprintf("%v", metadata.DefaultPartitionExpiration),
				"labels":                       fmt.Sprintf("%v", metadata.Labels),
				"access":                       fmt.Sprintf("%v", accessList),
				"table_count":                  fmt.Sprintf("%d", tableCount),
				"last_modified":                fmt.Sprintf("%v", metadata.LastModifiedTime),
				"etag":                         metadata.ETag,
			},
		})

		// Also discover tables in this dataset
		tableIt = dataset.Tables(ctx)
		for {
			table, err := tableIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to list tables in dataset %s: %v", dataset.DatasetID, err)
				break
			}

			// Get table metadata
			tableMetadata, err := table.Metadata(ctx)
			if err != nil {
				log.Printf("Failed to get metadata for table %s.%s: %v", dataset.DatasetID, table.TableID, err)
				continue
			}

			// Parse table creation time
			tableCreatedTime := time.Now()
			if tableMetadata.CreationTime.Unix() > 0 {
				tableCreatedTime = tableMetadata.CreationTime
			}

			resources = append(resources, models.Resource{
				ID:       fmt.Sprintf("%s.%s", dataset.DatasetID, table.TableID),
				Name:     table.TableID,
				Type:     "bigquery.googleapis.com/tables",
				Provider: "gcp",
				Region:   metadata.Location,
				State:    "ACTIVE",
				Created:  tableCreatedTime,
				Metadata: map[string]string{
					"dataset_id":               dataset.DatasetID,
					"description":              tableMetadata.Description,
					"type":                     string(tableMetadata.Type),
					"num_bytes":                fmt.Sprintf("%d", tableMetadata.NumBytes),
					"num_rows":                 fmt.Sprintf("%d", tableMetadata.NumRows),
					"last_modified":            fmt.Sprintf("%v", tableMetadata.LastModifiedTime),
					"expiration_time":          fmt.Sprintf("%v", tableMetadata.ExpirationTime),
					"labels":                   fmt.Sprintf("%v", tableMetadata.Labels),
					"clustering_fields":        fmt.Sprintf("%v", tableMetadata.Clustering),
					"time_partitioning":        fmt.Sprintf("%v", tableMetadata.TimePartitioning),
					"range_partitioning":       fmt.Sprintf("%v", tableMetadata.RangePartitioning),
					"require_partition_filter": fmt.Sprintf("%v", tableMetadata.RequirePartitionFilter),
					"schema":                   fmt.Sprintf("%v", tableMetadata.Schema),
					"etag":                     tableMetadata.ETag,
				},
			})
		}
	}

	return resources, nil
}

// Helper methods for resource deletion
func (gp *GCPProvider) deleteComputeInstance(ctx context.Context, resource models.Resource) error {
	// Extract zone from resource metadata or region
	zone := resource.Region
	if z, ok := resource.Metadata["zone"]; ok {
		zone = z
	}

	// Delete the compute instance
	_, err := gp.computeService.Instances.Delete(gp.projectID, zone, resource.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete compute instance %s: %w", resource.Name, err)
	}

	// Wait for operation to complete (best effort)
	time.Sleep(2 * time.Second)

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
	// Delete the SQL instance
	_, err := gp.sqlService.Instances.Delete(gp.projectID, resource.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete SQL instance %s: %w", resource.Name, err)
	}

	return nil
}

func (gp *GCPProvider) deletePubSubTopic(ctx context.Context, resource models.Resource) error {
	// Get the topic
	topic := gp.pubsubClient.Topic(resource.Name)

	// Delete the topic
	err := topic.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete Pub/Sub topic %s: %w", resource.Name, err)
	}

	return nil
}

func (gp *GCPProvider) deleteCloudFunction(ctx context.Context, resource models.Resource) error {
	// Construct function name
	location := "us-central1" // Default location
	if loc, ok := resource.Metadata["location"]; ok {
		location = loc
	}
	functionName := fmt.Sprintf("projects/%s/locations/%s/functions/%s", gp.projectID, location, resource.Name)

	// Delete the cloud function
	_, err := gp.functionsService.Projects.Locations.Functions.Delete(functionName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete Cloud Function %s: %w", resource.Name, err)
	}

	return nil
}

func (gp *GCPProvider) deleteDataprocCluster(ctx context.Context, resource models.Resource) error {
	// Extract region from resource metadata
	region := "us-central1" // Default region
	if r, ok := resource.Metadata["region"]; ok {
		region = r
	}

	// Create delete request
	req := &dataprocpb.DeleteClusterRequest{
		ProjectId:   gp.projectID,
		Region:      region,
		ClusterName: resource.Name,
	}

	// Delete the Dataproc cluster
	op, err := gp.dataprocClient.DeleteCluster(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete Dataproc cluster %s: %w", resource.Name, err)
	}

	// Wait for operation to complete (best effort)
	_ = op.Wait(ctx)

	return nil
}

func (gp *GCPProvider) deleteBigQueryDataset(ctx context.Context, resource models.Resource) error {
	// Get the dataset
	dataset := gp.bigqueryClient.Dataset(resource.Name)

	// Delete the dataset and all its contents
	err := dataset.DeleteWithContents(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete BigQuery dataset %s: %w", resource.Name, err)
	}

	return nil
}

func (gp *GCPProvider) deleteVPCNetwork(ctx context.Context, resource models.Resource) error {
	// First, delete all subnets in the network
	subnetsList, err := gp.computeService.Subnetworks.AggregatedList(gp.projectID).Context(ctx).Do()
	if err == nil {
		for _, subnetsScopedList := range subnetsList.Items {
			for _, subnet := range subnetsScopedList.Subnetworks {
				// Check if subnet belongs to this network
				if subnet.Network == resource.ID || strings.HasSuffix(subnet.Network, "/"+resource.Name) {
					// Extract region from subnet self link
					parts := strings.Split(subnet.SelfLink, "/")
					for i, part := range parts {
						if part == "regions" && i+1 < len(parts) {
							region := parts[i+1]
							// Delete the subnet
							gp.computeService.Subnetworks.Delete(gp.projectID, region, subnet.Name).Context(ctx).Do()
							break
						}
					}
				}
			}
		}
	}

	// Delete firewall rules associated with the network
	firewallList, err := gp.computeService.Firewalls.List(gp.projectID).Context(ctx).Do()
	if err == nil {
		for _, firewall := range firewallList.Items {
			if firewall.Network == resource.ID || strings.HasSuffix(firewall.Network, "/"+resource.Name) {
				gp.computeService.Firewalls.Delete(gp.projectID, firewall.Name).Context(ctx).Do()
			}
		}
	}

	// Wait a bit for subnets and firewall rules to be deleted
	time.Sleep(5 * time.Second)

	// Delete the VPC network
	_, err = gp.computeService.Networks.Delete(gp.projectID, resource.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete VPC network %s: %w", resource.Name, err)
	}

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
