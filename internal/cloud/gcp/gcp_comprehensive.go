package gcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"google.golang.org/api/iterator"
)

// ComprehensiveGCPDiscoverer discovers ALL GCP resources
type ComprehensiveGCPDiscoverer struct {
	projectID string
	progress  chan GCPDiscoveryProgress
}

// GCPDiscoveryProgress tracks discovery progress for GCP
type GCPDiscoveryProgress struct {
	Service      string
	ResourceType string
	Count        int
	Message      string
}

// NewComprehensiveGCPDiscoverer creates a new comprehensive GCP discoverer
func NewComprehensiveGCPDiscoverer(projectID string) (*ComprehensiveGCPDiscoverer, error) {
	if projectID == "" {
		// Try to get project ID from environment or default credentials
		projectID = getComprehensiveGCPProjectID()
		if projectID == "" {
			return nil, fmt.Errorf("GCP project ID not specified")
		}
	}

	return &ComprehensiveGCPDiscoverer{
		projectID: projectID,
		progress:  make(chan GCPDiscoveryProgress, 100),
	}, nil
}

// DiscoverAllGCPResources discovers all GCP resources
func (d *ComprehensiveGCPDiscoverer) DiscoverAllGCPResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	// Start progress reporter
	go d.reportProgress()

	var allResources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// If no regions specified, use common GCP regions
	if len(regions) == 0 {
		regions = d.getDefaultRegions()
	}

	// Discover global resources (not region-specific)
	globalResourceTypes := []func(context.Context) []models.Resource{
		d.discoverStorageBuckets,
		// These require additional dependencies - can be enabled later
		// d.discoverCloudSQLInstances,
		// d.discoverPubSubTopics,
		// d.discoverPubSubSubscriptions,
		// d.discoverFirestoreDatabases,
		// d.discoverSpannerInstances,
	}

	for _, discoveryFunc := range globalResourceTypes {
		wg.Add(1)
		go func(fn func(context.Context) []models.Resource) {
			defer wg.Done()
			resources := fn(ctx)
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(discoveryFunc)
	}

	// Discover regional resources
	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			resources := d.discoverRegionalResources(ctx, r)
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(region)
	}

	wg.Wait()
	close(d.progress)

	log.Printf("Comprehensive GCP discovery completed: %d total resources found", len(allResources))
	return allResources, nil
}

// discoverRegionalResources discovers resources in a specific region
func (d *ComprehensiveGCPDiscoverer) discoverRegionalResources(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Compute Engine instances
	resources = append(resources, d.discoverComputeInstances(ctx, region)...)

	// These require additional dependencies - can be enabled later
	// GKE clusters
	// resources = append(resources, d.discoverGKEClusters(ctx, region)...)

	// Cloud Functions
	// resources = append(resources, d.discoverCloudFunctions(ctx, region)...)

	// VPC Networks and Subnets
	resources = append(resources, d.discoverNetworks(ctx, region)...)

	// Firewall rules
	resources = append(resources, d.discoverFirewallRules(ctx, region)...)

	// Load balancers
	resources = append(resources, d.discoverLoadBalancers(ctx, region)...)

	// Disks
	resources = append(resources, d.discoverDisks(ctx, region)...)

	return resources
}

// discoverComputeInstances discovers Compute Engine instances
func (d *ComprehensiveGCPDiscoverer) discoverComputeInstances(ctx context.Context, zone string) []models.Resource {
	var resources []models.Resource

	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		log.Printf("Failed to create compute client: %v", err)
		return resources
	}
	defer client.Close()

	req := &computepb.ListInstancesRequest{
		Project: d.projectID,
		Zone:    zone,
	}

	it := client.List(ctx, req)
	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating instances: %v", err)
			break
		}

		if instance.Name != nil {
			properties := make(map[string]interface{})
			if instance.MachineType != nil {
				properties["machine_type"] = *instance.MachineType
			}
			if instance.Status != nil {
				properties["status"] = *instance.Status
			}

			tags := make(map[string]string)
			if instance.Labels != nil {
				for k, v := range instance.Labels {
					tags[k] = v
				}
			}

			state := "unknown"
			if instance.Status != nil {
				state = *instance.Status
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("projects/%s/zones/%s/instances/%s", d.projectID, zone, *instance.Name),
				Name:       *instance.Name,
				Type:       "gcp_compute_instance",
				Provider:   "gcp",
				Region:     zone,
				State:      state,
				Tags:       tags,
				Properties: properties,
			})
		}
	}

	d.progress <- GCPDiscoveryProgress{Service: "Compute", ResourceType: "Instances", Count: len(resources)}
	return resources
}

/* // Commented out - requires additional dependencies
// discoverGKEClusters discovers GKE clusters
func (d *ComprehensiveGCPDiscoverer) discoverGKEClusters(ctx context.Context, location string) []models.Resource {
	var resources []models.Resource

	client, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		log.Printf("Failed to create GKE client: %v", err)
		return resources
	}
	defer client.Close()

	parent := fmt.Sprintf("projects/%s/locations/%s", d.projectID, location)
	req := &containerpb.ListClustersRequest{
		Parent: parent,
	}

	resp, err := client.ListClusters(ctx, req)
	if err != nil {
		log.Printf("Failed to list GKE clusters: %v", err)
		return resources
	}

	for _, cluster := range resp.Clusters {
		properties := make(map[string]interface{})
		properties["version"] = cluster.CurrentMasterVersion
		properties["node_count"] = cluster.CurrentNodeCount
		properties["status"] = cluster.Status.String()

		tags := make(map[string]string)
		if cluster.ResourceLabels != nil {
			for k, v := range cluster.ResourceLabels {
				tags[k] = v
			}
		}

		resources = append(resources, models.Resource{
			ID:         fmt.Sprintf("projects/%s/locations/%s/clusters/%s", d.projectID, location, cluster.Name),
			Name:       cluster.Name,
			Type:       "gcp_gke_cluster",
			Provider:   "gcp",
			Region:     location,
			State:      cluster.Status.String(),
			Tags:       tags,
			Properties: properties,
		})
	}

	d.progress <- GCPDiscoveryProgress{Service: "GKE", ResourceType: "Clusters", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverCloudFunctions discovers Cloud Functions
func (d *ComprehensiveGCPDiscoverer) discoverCloudFunctions(ctx context.Context, location string) []models.Resource {
	var resources []models.Resource

	client, err := functions.NewCloudFunctionsClient(ctx)
	if err != nil {
		log.Printf("Failed to create Cloud Functions client: %v", err)
		return resources
	}
	defer client.Close()

	parent := fmt.Sprintf("projects/%s/locations/%s", d.projectID, location)
	req := &functionspb.ListFunctionsRequest{
		Parent: parent,
	}

	it := client.ListFunctions(ctx, req)
	for {
		function, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating functions: %v", err)
			break
		}

		properties := make(map[string]interface{})
		properties["runtime"] = function.GetSourceArchiveUrl()
		properties["entry_point"] = function.GetEntryPoint()
		properties["trigger"] = function.GetHttpsTrigger()

		tags := make(map[string]string)
		if function.Labels != nil {
			for k, v := range function.Labels {
				tags[k] = v
			}
		}

		resources = append(resources, models.Resource{
			ID:         function.Name,
			Name:       function.Name,
			Type:       "gcp_cloud_function",
			Provider:   "gcp",
			Region:     location,
			State:      "active",
			Tags:       tags,
			Properties: properties,
		})
	}

	d.progress <- GCPDiscoveryProgress{Service: "Functions", ResourceType: "Cloud Functions", Count: len(resources)}
	return resources
}
*/

// discoverStorageBuckets discovers Cloud Storage buckets
func (d *ComprehensiveGCPDiscoverer) discoverStorageBuckets(ctx context.Context) []models.Resource {
	var resources []models.Resource

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create storage client: %v", err)
		return resources
	}
	defer client.Close()

	it := client.Buckets(ctx, d.projectID)
	for {
		bucket, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating buckets: %v", err)
			break
		}

		properties := make(map[string]interface{})
		properties["storage_class"] = bucket.StorageClass
		properties["location"] = bucket.Location
		properties["created"] = bucket.Created

		tags := make(map[string]string)
		if bucket.Labels != nil {
			for k, v := range bucket.Labels {
				tags[k] = v
			}
		}

		resources = append(resources, models.Resource{
			ID:         bucket.Name,
			Name:       bucket.Name,
			Type:       "gcp_storage_bucket",
			Provider:   "gcp",
			Region:     bucket.Location,
			State:      "active",
			Tags:       tags,
			Properties: properties,
		})
	}

	d.progress <- GCPDiscoveryProgress{Service: "Storage", ResourceType: "Buckets", Count: len(resources)}
	return resources
}

/* // Commented out - requires additional dependencies
// discoverCloudSQLInstances discovers Cloud SQL instances
func (d *ComprehensiveGCPDiscoverer) discoverCloudSQLInstances(ctx context.Context) []models.Resource {
	var resources []models.Resource

	service, err := sqladmin.NewService(ctx)
	if err != nil {
		log.Printf("Failed to create Cloud SQL client: %v", err)
		return resources
	}

	call := service.Instances.List(d.projectID)
	resp, err := call.Do()
	if err != nil {
		log.Printf("Failed to list Cloud SQL instances: %v", err)
		return resources
	}

	for _, instance := range resp.Items {
		properties := make(map[string]interface{})
		properties["database_version"] = instance.DatabaseVersion
		properties["tier"] = instance.Settings.Tier
		properties["state"] = instance.State

		tags := make(map[string]string)
		if instance.Settings != nil && instance.Settings.UserLabels != nil {
			for k, v := range instance.Settings.UserLabels {
				tags[k] = v
			}
		}

		resources = append(resources, models.Resource{
			ID:         fmt.Sprintf("projects/%s/instances/%s", d.projectID, instance.Name),
			Name:       instance.Name,
			Type:       "gcp_sql_instance",
			Provider:   "gcp",
			Region:     instance.Region,
			State:      instance.State,
			Tags:       tags,
			Properties: properties,
		})
	}

	d.progress <- GCPDiscoveryProgress{Service: "SQL", ResourceType: "Cloud SQL Instances", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverPubSubTopics discovers Pub/Sub topics
func (d *ComprehensiveGCPDiscoverer) discoverPubSubTopics(ctx context.Context) []models.Resource {
	var resources []models.Resource

	client, err := pubsub.NewClient(ctx, d.projectID)
	if err != nil {
		log.Printf("Failed to create Pub/Sub client: %v", err)
		return resources
	}
	defer client.Close()

	it := client.Topics(ctx)
	for {
		topic, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating topics: %v", err)
			break
		}

		config, err := topic.Config(ctx)
		if err != nil {
			continue
		}

		tags := make(map[string]string)
		if config.Labels != nil {
			for k, v := range config.Labels {
				tags[k] = v
			}
		}

		resources = append(resources, models.Resource{
			ID:       topic.String(),
			Name:     topic.ID(),
			Type:     "gcp_pubsub_topic",
			Provider: "gcp",
			Region:   "global",
			State:    "active",
			Tags:     tags,
		})
	}

	d.progress <- GCPDiscoveryProgress{Service: "PubSub", ResourceType: "Topics", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverPubSubSubscriptions discovers Pub/Sub subscriptions
func (d *ComprehensiveGCPDiscoverer) discoverPubSubSubscriptions(ctx context.Context) []models.Resource {
	var resources []models.Resource

	client, err := pubsub.NewClient(ctx, d.projectID)
	if err != nil {
		log.Printf("Failed to create Pub/Sub client: %v", err)
		return resources
	}
	defer client.Close()

	it := client.Subscriptions(ctx)
	for {
		sub, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating subscriptions: %v", err)
			break
		}

		config, err := sub.Config(ctx)
		if err != nil {
			continue
		}

		properties := make(map[string]interface{})
		properties["topic"] = config.Topic.String()
		properties["ack_deadline"] = config.AckDeadline

		tags := make(map[string]string)
		if config.Labels != nil {
			for k, v := range config.Labels {
				tags[k] = v
			}
		}

		resources = append(resources, models.Resource{
			ID:         sub.String(),
			Name:       sub.ID(),
			Type:       "gcp_pubsub_subscription",
			Provider:   "gcp",
			Region:     "global",
			State:      "active",
			Tags:       tags,
			Properties: properties,
		})
	}

	d.progress <- GCPDiscoveryProgress{Service: "PubSub", ResourceType: "Subscriptions", Count: len(resources)}
	return resources
}
*/

// discoverNetworks discovers VPC networks and subnets
func (d *ComprehensiveGCPDiscoverer) discoverNetworks(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	client, err := compute.NewNetworksRESTClient(ctx)
	if err != nil {
		log.Printf("Failed to create networks client: %v", err)
		return resources
	}
	defer client.Close()

	req := &computepb.ListNetworksRequest{
		Project: d.projectID,
	}

	it := client.List(ctx, req)
	for {
		network, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating networks: %v", err)
			break
		}

		if network.Name != nil {
			properties := make(map[string]interface{})
			if network.AutoCreateSubnetworks != nil {
				properties["auto_create_subnets"] = *network.AutoCreateSubnetworks
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("projects/%s/global/networks/%s", d.projectID, *network.Name),
				Name:       *network.Name,
				Type:       "gcp_vpc_network",
				Provider:   "gcp",
				Region:     "global",
				State:      "active",
				Properties: properties,
			})
		}
	}

	// Discover subnets
	subnetClient, err := compute.NewSubnetworksRESTClient(ctx)
	if err == nil {
		defer subnetClient.Close()

		subnetReq := &computepb.ListSubnetworksRequest{
			Project: d.projectID,
			Region:  region,
		}

		subnetIt := subnetClient.List(ctx, subnetReq)
		for {
			subnet, err := subnetIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}

			if subnet.Name != nil {
				properties := make(map[string]interface{})
				if subnet.IpCidrRange != nil {
					properties["cidr"] = *subnet.IpCidrRange
				}
				if subnet.Network != nil {
					properties["network"] = *subnet.Network
				}

				resources = append(resources, models.Resource{
					ID:         fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", d.projectID, region, *subnet.Name),
					Name:       *subnet.Name,
					Type:       "gcp_subnet",
					Provider:   "gcp",
					Region:     region,
					State:      "active",
					Properties: properties,
				})
			}
		}
	}

	d.progress <- GCPDiscoveryProgress{Service: "Network", ResourceType: "VPC Networks", Count: len(resources)}
	return resources
}

// discoverFirewallRules discovers firewall rules
func (d *ComprehensiveGCPDiscoverer) discoverFirewallRules(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	client, err := compute.NewFirewallsRESTClient(ctx)
	if err != nil {
		log.Printf("Failed to create firewall client: %v", err)
		return resources
	}
	defer client.Close()

	req := &computepb.ListFirewallsRequest{
		Project: d.projectID,
	}

	it := client.List(ctx, req)
	for {
		firewall, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating firewall rules: %v", err)
			break
		}

		if firewall.Name != nil {
			properties := make(map[string]interface{})
			if firewall.Direction != nil {
				properties["direction"] = *firewall.Direction
			}
			if firewall.Priority != nil {
				properties["priority"] = *firewall.Priority
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("projects/%s/global/firewalls/%s", d.projectID, *firewall.Name),
				Name:       *firewall.Name,
				Type:       "gcp_firewall_rule",
				Provider:   "gcp",
				Region:     "global",
				State:      "active",
				Properties: properties,
			})
		}
	}

	d.progress <- GCPDiscoveryProgress{Service: "Network", ResourceType: "Firewall Rules", Count: len(resources)}
	return resources
}

// discoverLoadBalancers discovers load balancers
func (d *ComprehensiveGCPDiscoverer) discoverLoadBalancers(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Backend services (global load balancers)
	client, err := compute.NewBackendServicesRESTClient(ctx)
	if err != nil {
		log.Printf("Failed to create backend services client: %v", err)
		return resources
	}
	defer client.Close()

	req := &computepb.ListBackendServicesRequest{
		Project: d.projectID,
	}

	it := client.List(ctx, req)
	for {
		backend, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating backend services: %v", err)
			break
		}

		if backend.Name != nil {
			properties := make(map[string]interface{})
			if backend.LoadBalancingScheme != nil {
				properties["scheme"] = *backend.LoadBalancingScheme
			}
			if backend.Protocol != nil {
				properties["protocol"] = *backend.Protocol
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("projects/%s/global/backendServices/%s", d.projectID, *backend.Name),
				Name:       *backend.Name,
				Type:       "gcp_load_balancer",
				Provider:   "gcp",
				Region:     "global",
				State:      "active",
				Properties: properties,
			})
		}
	}

	d.progress <- GCPDiscoveryProgress{Service: "Network", ResourceType: "Load Balancers", Count: len(resources)}
	return resources
}

// discoverDisks discovers persistent disks
func (d *ComprehensiveGCPDiscoverer) discoverDisks(ctx context.Context, zone string) []models.Resource {
	var resources []models.Resource

	client, err := compute.NewDisksRESTClient(ctx)
	if err != nil {
		log.Printf("Failed to create disks client: %v", err)
		return resources
	}
	defer client.Close()

	req := &computepb.ListDisksRequest{
		Project: d.projectID,
		Zone:    zone,
	}

	it := client.List(ctx, req)
	for {
		disk, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating disks: %v", err)
			break
		}

		if disk.Name != nil {
			properties := make(map[string]interface{})
			if disk.SizeGb != nil {
				properties["size_gb"] = *disk.SizeGb
			}
			if disk.Type != nil {
				properties["type"] = *disk.Type
			}
			if disk.Status != nil {
				properties["status"] = *disk.Status
			}

			tags := make(map[string]string)
			if disk.Labels != nil {
				for k, v := range disk.Labels {
					tags[k] = v
				}
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("projects/%s/zones/%s/disks/%s", d.projectID, zone, *disk.Name),
				Name:       *disk.Name,
				Type:       "gcp_persistent_disk",
				Provider:   "gcp",
				Region:     zone,
				State:      "active",
				Tags:       tags,
				Properties: properties,
			})
		}
	}

	d.progress <- GCPDiscoveryProgress{Service: "Compute", ResourceType: "Persistent Disks", Count: len(resources)}
	return resources
}

/* // Commented out - requires additional dependencies
// discoverFirestoreDatabases discovers Firestore databases
func (d *ComprehensiveGCPDiscoverer) discoverFirestoreDatabases(ctx context.Context) []models.Resource {
	var resources []models.Resource

	client, err := firestore.NewClient(ctx, d.projectID)
	if err != nil {
		log.Printf("Failed to create Firestore client: %v", err)
		return resources
	}
	defer client.Close()

	// Note: Firestore typically has one database per project
	resources = append(resources, models.Resource{
		ID:       fmt.Sprintf("projects/%s/databases/(default)", d.projectID),
		Name:     "(default)",
		Type:     "gcp_firestore_database",
		Provider: "gcp",
		Region:   "global",
		State:    "active",
	})

	d.progress <- GCPDiscoveryProgress{Service: "Firestore", ResourceType: "Databases", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverSpannerInstances discovers Spanner instances
func (d *ComprehensiveGCPDiscoverer) discoverSpannerInstances(ctx context.Context) []models.Resource {
	var resources []models.Resource

	client, err := spanner.NewInstanceAdminClient(ctx)
	if err != nil {
		log.Printf("Failed to create Spanner client: %v", err)
		return resources
	}
	defer client.Close()

	parent := fmt.Sprintf("projects/%s", d.projectID)
	it := client.ListInstances(ctx, &spanner.ListInstancesRequest{
		Parent: parent,
	})

	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating Spanner instances: %v", err)
			break
		}

		properties := make(map[string]interface{})
		properties["node_count"] = instance.NodeCount
		properties["state"] = instance.State.String()

		tags := make(map[string]string)
		if instance.Labels != nil {
			for k, v := range instance.Labels {
				tags[k] = v
			}
		}

		resources = append(resources, models.Resource{
			ID:         instance.Name,
			Name:       instance.DisplayName,
			Type:       "gcp_spanner_instance",
			Provider:   "gcp",
			Region:     "global",
			State:      instance.State.String(),
			Tags:       tags,
			Properties: properties,
		})
	}

	d.progress <- GCPDiscoveryProgress{Service: "Spanner", ResourceType: "Instances", Count: len(resources)}
	return resources
}
*/

// Helper functions
func (d *ComprehensiveGCPDiscoverer) getDefaultRegions() []string {
	return []string{
		"us-central1",
		"us-east1",
		"us-west1",
		"us-west2",
		"europe-west1",
		"europe-west2",
		"asia-east1",
		"asia-northeast1",
		"asia-southeast1",
	}
}

func (d *ComprehensiveGCPDiscoverer) reportProgress() {
	for progress := range d.progress {
		log.Printf("[GCP] %s: Discovered %d %s", progress.Service, progress.Count, progress.ResourceType)
	}
}

func getComprehensiveGCPProjectID() string {
	// Get project ID from environment
	if projectID := os.Getenv("GCP_PROJECT_ID"); projectID != "" {
		return projectID
	}
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		return projectID
	}
	if projectID := os.Getenv("GCLOUD_PROJECT"); projectID != "" {
		return projectID
	}
	return ""
}
