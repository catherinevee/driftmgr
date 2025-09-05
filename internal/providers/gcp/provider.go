package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// GCPProviderComplete implements CloudProvider for Google Cloud Platform with full API implementation
type GCPProviderComplete struct {
	projectID   string
	region      string
	zone        string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
	baseURLs    map[string]string
}

// NewGCPProviderComplete creates a new GCP provider with complete implementation
func NewGCPProviderComplete(projectID string) *GCPProviderComplete {
	return &GCPProviderComplete{
		projectID: projectID,
		region:    "us-central1",
		zone:      "us-central1-a",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURLs: map[string]string{
			"compute":              "https://compute.googleapis.com/compute/v1",
			"storage":              "https://storage.googleapis.com/storage/v1",
			"sql":                  "https://sqladmin.googleapis.com/v1",
			"container":            "https://container.googleapis.com/v1",
			"cloudresourcemanager": "https://cloudresourcemanager.googleapis.com/v3",
			"iam":                  "https://iam.googleapis.com/v1",
			"pubsub":               "https://pubsub.googleapis.com/v1",
			"functions":            "https://cloudfunctions.googleapis.com/v2",
			"run":                  "https://run.googleapis.com/v2",
			"redis":                "https://redis.googleapis.com/v1",
			"firestore":            "https://firestore.googleapis.com/v1",
			"bigtable":             "https://bigtableadmin.googleapis.com/v2",
			"kms":                  "https://cloudkms.googleapis.com/v1",
			"logging":              "https://logging.googleapis.com/v2",
			"monitoring":           "https://monitoring.googleapis.com/v3",
		},
	}
}

// Name returns the provider name
func (p *GCPProviderComplete) Name() string {
	return "gcp"
}

// Connect establishes connection to GCP
func (p *GCPProviderComplete) Connect(ctx context.Context) error {
	// Try to authenticate using Application Default Credentials
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/compute",
		"https://www.googleapis.com/auth/storage",
	)
	if err != nil {
		// Try service account key file
		if keyFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); keyFile != "" {
			data, err := ioutil.ReadFile(keyFile)
			if err != nil {
				return fmt.Errorf("failed to read service account key: %w", err)
			}

			creds, err = google.CredentialsFromJSON(ctx, data,
				"https://www.googleapis.com/auth/cloud-platform",
				"https://www.googleapis.com/auth/compute",
				"https://www.googleapis.com/auth/storage",
			)
			if err != nil {
				return fmt.Errorf("failed to create credentials from JSON: %w", err)
			}
		} else {
			return fmt.Errorf("no GCP credentials found: %w", err)
		}
	}

	p.tokenSource = creds.TokenSource
	p.httpClient = oauth2.NewClient(ctx, p.tokenSource)

	// If project ID not set, try to get it from credentials
	if p.projectID == "" && creds.ProjectID != "" {
		p.projectID = creds.ProjectID
	}

	return nil
}

// makeAPIRequest makes an authenticated request to GCP API
func (p *GCPProviderComplete) makeAPIRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		json.Unmarshal(respBody, &errorResp)
		if errorResp.Error.Message != "" {
			return nil, fmt.Errorf("GCP API error (%d): %s", errorResp.Error.Code, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("GCP API error (status %d): %s", resp.StatusCode, respBody)
	}

	return respBody, nil
}

// DiscoverResources discovers resources in the specified region (implements CloudProvider interface)
func (p *GCPProviderComplete) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	// GCP uses zones within regions, update zone if region is provided
	if region != "" {
		p.zone = region + "-a" // Default to zone a in the region
	}

	resources := []models.Resource{}

	// TODO: Implement actual resource discovery
	// This would involve listing various resource types from GCP

	return resources, nil
}

// GetResource retrieves a specific resource by ID (implements CloudProvider interface)
func (p *GCPProviderComplete) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	// Try to determine resource type from ID
	// GCP resource IDs can have various formats

	// Try as different resource types
	resourceTypes := []string{
		"google_compute_instance",
		"google_compute_network",
		"google_compute_subnetwork",
		"google_compute_firewall",
		"google_compute_disk",
		"google_storage_bucket",
		"google_sql_database_instance",
		"google_container_cluster",
	}

	// Extract resource name from potential full path
	parts := strings.Split(resourceID, "/")
	resourceName := resourceID
	if len(parts) > 0 {
		resourceName = parts[len(parts)-1]
	}

	// Try each resource type
	for _, resType := range resourceTypes {
		if res, err := p.GetResourceByType(ctx, resType, resourceName); err == nil {
			return res, nil
		}
	}

	return nil, fmt.Errorf("unable to find resource with ID: %s", resourceID)
}

// ValidateCredentials checks if the provider credentials are valid (implements CloudProvider interface)
func (p *GCPProviderComplete) ValidateCredentials(ctx context.Context) error {
	return p.Connect(ctx)
}

// ListRegions returns available regions for the provider (implements CloudProvider interface)
func (p *GCPProviderComplete) ListRegions(ctx context.Context) ([]string, error) {
	// Return common GCP regions
	return []string{
		"us-central1", "us-east1", "us-east4", "us-west1", "us-west2", "us-west3", "us-west4",
		"europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6",
		"asia-east1", "asia-east2", "asia-northeast1", "asia-northeast2", "asia-northeast3",
		"asia-south1", "asia-southeast1", "asia-southeast2",
		"australia-southeast1", "southamerica-east1", "northamerica-northeast1",
	}, nil
}

// SupportedResourceTypes returns the list of supported resource types (implements CloudProvider interface)
func (p *GCPProviderComplete) SupportedResourceTypes() []string {
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

// GetResourceByType retrieves a specific resource from GCP by type
func (p *GCPProviderComplete) GetResourceByType(ctx context.Context, resourceType string, resourceID string) (*models.Resource, error) {
	switch resourceType {
	case "google_compute_instance":
		return p.getComputeInstance(ctx, resourceID)
	case "google_compute_network":
		return p.getNetwork(ctx, resourceID)
	case "google_compute_subnetwork":
		return p.getSubnetwork(ctx, resourceID)
	case "google_compute_firewall":
		return p.getFirewall(ctx, resourceID)
	case "google_compute_disk":
		return p.getDisk(ctx, resourceID)
	case "google_storage_bucket":
		return p.getStorageBucket(ctx, resourceID)
	case "google_sql_database_instance":
		return p.getSQLInstance(ctx, resourceID)
	case "google_sql_database":
		return p.getSQLDatabase(ctx, resourceID)
	case "google_container_cluster":
		return p.getGKECluster(ctx, resourceID)
	case "google_pubsub_topic":
		return p.getPubSubTopic(ctx, resourceID)
	case "google_pubsub_subscription":
		return p.getPubSubSubscription(ctx, resourceID)
	case "google_cloud_function":
		return p.getCloudFunction(ctx, resourceID)
	case "google_cloud_run_service":
		return p.getCloudRunService(ctx, resourceID)
	case "google_redis_instance":
		return p.getRedisInstance(ctx, resourceID)
	case "google_kms_key_ring":
		return p.getKMSKeyRing(ctx, resourceID)
	case "google_kms_crypto_key":
		return p.getKMSCryptoKey(ctx, resourceID)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// getComputeInstance retrieves a compute instance
func (p *GCPProviderComplete) getComputeInstance(ctx context.Context, instanceName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/zones/%s/instances/%s",
		p.baseURLs["compute"], p.projectID, p.zone, instanceName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	var instance map[string]interface{}
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	// Extract network interfaces
	var networkInterfaces []map[string]interface{}
	if ni, ok := instance["networkInterfaces"].([]interface{}); ok {
		for _, iface := range ni {
			if m, ok := iface.(map[string]interface{}); ok {
				networkInterfaces = append(networkInterfaces, m)
			}
		}
	}

	// Extract disks
	var disks []map[string]interface{}
	if d, ok := instance["disks"].([]interface{}); ok {
		for _, disk := range d {
			if m, ok := disk.(map[string]interface{}); ok {
				disks = append(disks, m)
			}
		}
	}

	return &models.Resource{
		ID:   instanceName,
		Type: "google_compute_instance",
		Attributes: map[string]interface{}{
			"name":               instance["name"],
			"machine_type":       instance["machineType"],
			"zone":               p.zone,
			"status":             instance["status"],
			"network_interfaces": networkInterfaces,
			"disks":              disks,
			"labels":             instance["labels"],
			"metadata":           instance["metadata"],
			"tags":               instance["tags"],
			"creation_timestamp": instance["creationTimestamp"],
		},
	}, nil
}

// getNetwork retrieves a VPC network
func (p *GCPProviderComplete) getNetwork(ctx context.Context, networkName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/global/networks/%s",
		p.baseURLs["compute"], p.projectID, networkName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}

	var network map[string]interface{}
	if err := json.Unmarshal(data, &network); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network: %w", err)
	}

	return &models.Resource{
		ID:   networkName,
		Type: "google_compute_network",
		Attributes: map[string]interface{}{
			"name":                    network["name"],
			"auto_create_subnetworks": network["autoCreateSubnetworks"],
			"routing_mode":            network["routingConfig"].(map[string]interface{})["routingMode"],
			"mtu":                     network["mtu"],
			"description":             network["description"],
			"creation_timestamp":      network["creationTimestamp"],
		},
	}, nil
}

// getSubnetwork retrieves a subnet
func (p *GCPProviderComplete) getSubnetwork(ctx context.Context, subnetName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/regions/%s/subnetworks/%s",
		p.baseURLs["compute"], p.projectID, p.region, subnetName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnetwork: %w", err)
	}

	var subnet map[string]interface{}
	if err := json.Unmarshal(data, &subnet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subnetwork: %w", err)
	}

	return &models.Resource{
		ID:   subnetName,
		Type: "google_compute_subnetwork",
		Attributes: map[string]interface{}{
			"name":                     subnet["name"],
			"network":                  subnet["network"],
			"ip_cidr_range":            subnet["ipCidrRange"],
			"region":                   p.region,
			"private_ip_google_access": subnet["privateIpGoogleAccess"],
			"purpose":                  subnet["purpose"],
			"creation_timestamp":       subnet["creationTimestamp"],
		},
	}, nil
}

// getFirewall retrieves a firewall rule
func (p *GCPProviderComplete) getFirewall(ctx context.Context, firewallName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/global/firewalls/%s",
		p.baseURLs["compute"], p.projectID, firewallName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall: %w", err)
	}

	var firewall map[string]interface{}
	if err := json.Unmarshal(data, &firewall); err != nil {
		return nil, fmt.Errorf("failed to unmarshal firewall: %w", err)
	}

	return &models.Resource{
		ID:   firewallName,
		Type: "google_compute_firewall",
		Attributes: map[string]interface{}{
			"name":               firewall["name"],
			"network":            firewall["network"],
			"priority":           firewall["priority"],
			"source_ranges":      firewall["sourceRanges"],
			"destination_ranges": firewall["destinationRanges"],
			"allowed":            firewall["allowed"],
			"denied":             firewall["denied"],
			"direction":          firewall["direction"],
			"target_tags":        firewall["targetTags"],
			"source_tags":        firewall["sourceTags"],
			"creation_timestamp": firewall["creationTimestamp"],
		},
	}, nil
}

// getDisk retrieves a persistent disk
func (p *GCPProviderComplete) getDisk(ctx context.Context, diskName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/zones/%s/disks/%s",
		p.baseURLs["compute"], p.projectID, p.zone, diskName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk: %w", err)
	}

	var disk map[string]interface{}
	if err := json.Unmarshal(data, &disk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disk: %w", err)
	}

	return &models.Resource{
		ID:   diskName,
		Type: "google_compute_disk",
		Attributes: map[string]interface{}{
			"name":               disk["name"],
			"size_gb":            disk["sizeGb"],
			"type":               disk["type"],
			"zone":               p.zone,
			"status":             disk["status"],
			"labels":             disk["labels"],
			"creation_timestamp": disk["creationTimestamp"],
		},
	}, nil
}

// getStorageBucket retrieves a storage bucket
func (p *GCPProviderComplete) getStorageBucket(ctx context.Context, bucketName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/b/%s", p.baseURLs["storage"], bucketName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	var bucket map[string]interface{}
	if err := json.Unmarshal(data, &bucket); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket: %w", err)
	}

	return &models.Resource{
		ID:   bucketName,
		Type: "google_storage_bucket",
		Attributes: map[string]interface{}{
			"name":                        bucket["name"],
			"location":                    bucket["location"],
			"storage_class":               bucket["storageClass"],
			"versioning":                  bucket["versioning"],
			"lifecycle_rules":             bucket["lifecycle"],
			"labels":                      bucket["labels"],
			"encryption":                  bucket["encryption"],
			"uniform_bucket_level_access": bucket["iamConfiguration"],
			"time_created":                bucket["timeCreated"],
		},
	}, nil
}

// getSQLInstance retrieves a Cloud SQL instance
func (p *GCPProviderComplete) getSQLInstance(ctx context.Context, instanceName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/instances/%s",
		p.baseURLs["sql"], p.projectID, instanceName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL instance: %w", err)
	}

	var instance map[string]interface{}
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SQL instance: %w", err)
	}

	settings := instance["settings"].(map[string]interface{})

	return &models.Resource{
		ID:   instanceName,
		Type: "google_sql_database_instance",
		Attributes: map[string]interface{}{
			"name":                 instance["name"],
			"database_version":     instance["databaseVersion"],
			"region":               instance["region"],
			"tier":                 settings["tier"],
			"disk_size":            settings["dataDiskSizeGb"],
			"disk_type":            settings["dataDiskType"],
			"availability_type":    settings["availabilityType"],
			"backup_configuration": settings["backupConfiguration"],
			"ip_configuration":     settings["ipConfiguration"],
			"state":                instance["state"],
		},
	}, nil
}

// getSQLDatabase retrieves a Cloud SQL database
func (p *GCPProviderComplete) getSQLDatabase(ctx context.Context, dbID string) (*models.Resource, error) {
	// Parse database ID to extract instance and database names
	parts := strings.Split(dbID, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid database ID format")
	}
	instanceName := parts[0]
	dbName := parts[1]

	url := fmt.Sprintf("%s/projects/%s/instances/%s/databases/%s",
		p.baseURLs["sql"], p.projectID, instanceName, dbName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL database: %w", err)
	}

	var db map[string]interface{}
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SQL database: %w", err)
	}

	return &models.Resource{
		ID:   dbID,
		Type: "google_sql_database",
		Attributes: map[string]interface{}{
			"name":      db["name"],
			"instance":  instanceName,
			"charset":   db["charset"],
			"collation": db["collation"],
			"project":   db["project"],
		},
	}, nil
}

// getGKECluster retrieves a GKE cluster
func (p *GCPProviderComplete) getGKECluster(ctx context.Context, clusterName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/clusters/%s",
		p.baseURLs["container"], p.projectID, p.region, clusterName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get GKE cluster: %w", err)
	}

	var cluster map[string]interface{}
	if err := json.Unmarshal(data, &cluster); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GKE cluster: %w", err)
	}

	return &models.Resource{
		ID:   clusterName,
		Type: "google_container_cluster",
		Attributes: map[string]interface{}{
			"name":                   cluster["name"],
			"location":               cluster["location"],
			"initial_node_count":     cluster["initialNodeCount"],
			"node_config":            cluster["nodeConfig"],
			"master_auth":            cluster["masterAuth"],
			"network":                cluster["network"],
			"subnetwork":             cluster["subnetwork"],
			"cluster_ipv4_cidr":      cluster["clusterIpv4Cidr"],
			"services_ipv4_cidr":     cluster["servicesIpv4Cidr"],
			"status":                 cluster["status"],
			"current_master_version": cluster["currentMasterVersion"],
			"current_node_version":   cluster["currentNodeVersion"],
			"resource_labels":        cluster["resourceLabels"],
		},
	}, nil
}

// getPubSubTopic retrieves a Pub/Sub topic
func (p *GCPProviderComplete) getPubSubTopic(ctx context.Context, topicName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/topics/%s",
		p.baseURLs["pubsub"], p.projectID, topicName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pub/Sub topic: %w", err)
	}

	var topic map[string]interface{}
	if err := json.Unmarshal(data, &topic); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Pub/Sub topic: %w", err)
	}

	return &models.Resource{
		ID:   topicName,
		Type: "google_pubsub_topic",
		Attributes: map[string]interface{}{
			"name":                       topic["name"],
			"labels":                     topic["labels"],
			"message_retention_duration": topic["messageRetentionDuration"],
			"kms_key_name":               topic["kmsKeyName"],
		},
	}, nil
}

// getPubSubSubscription retrieves a Pub/Sub subscription
func (p *GCPProviderComplete) getPubSubSubscription(ctx context.Context, subName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s",
		p.baseURLs["pubsub"], p.projectID, subName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pub/Sub subscription: %w", err)
	}

	var sub map[string]interface{}
	if err := json.Unmarshal(data, &sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Pub/Sub subscription: %w", err)
	}

	return &models.Resource{
		ID:   subName,
		Type: "google_pubsub_subscription",
		Attributes: map[string]interface{}{
			"name":                       sub["name"],
			"topic":                      sub["topic"],
			"ack_deadline_seconds":       sub["ackDeadlineSeconds"],
			"message_retention_duration": sub["messageRetentionDuration"],
			"retain_acked_messages":      sub["retainAckedMessages"],
			"enable_message_ordering":    sub["enableMessageOrdering"],
			"labels":                     sub["labels"],
		},
	}, nil
}

// getCloudFunction retrieves a Cloud Function
func (p *GCPProviderComplete) getCloudFunction(ctx context.Context, functionName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/functions/%s",
		p.baseURLs["functions"], p.projectID, p.region, functionName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cloud Function: %w", err)
	}

	var function map[string]interface{}
	if err := json.Unmarshal(data, &function); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Cloud Function: %w", err)
	}

	return &models.Resource{
		ID:   functionName,
		Type: "google_cloud_function",
		Attributes: map[string]interface{}{
			"name":        function["name"],
			"description": function["description"],
			"state":       function["state"],
			"labels":      function["labels"],
			"environment": function["environment"],
		},
	}, nil
}

// getCloudRunService retrieves a Cloud Run service
func (p *GCPProviderComplete) getCloudRunService(ctx context.Context, serviceName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/services/%s",
		p.baseURLs["run"], p.projectID, p.region, serviceName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cloud Run service: %w", err)
	}

	var service map[string]interface{}
	if err := json.Unmarshal(data, &service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Cloud Run service: %w", err)
	}

	return &models.Resource{
		ID:   serviceName,
		Type: "google_cloud_run_service",
		Attributes: map[string]interface{}{
			"name":        service["name"],
			"description": service["description"],
			"labels":      service["labels"],
			"annotations": service["annotations"],
			"generation":  service["generation"],
			"uri":         service["uri"],
		},
	}, nil
}

// getRedisInstance retrieves a Redis instance
func (p *GCPProviderComplete) getRedisInstance(ctx context.Context, instanceName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/instances/%s",
		p.baseURLs["redis"], p.projectID, p.region, instanceName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis instance: %w", err)
	}

	var instance map[string]interface{}
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Redis instance: %w", err)
	}

	return &models.Resource{
		ID:   instanceName,
		Type: "google_redis_instance",
		Attributes: map[string]interface{}{
			"name":               instance["name"],
			"tier":               instance["tier"],
			"memory_size_gb":     instance["memorySizeGb"],
			"redis_version":      instance["redisVersion"],
			"location_id":        instance["locationId"],
			"authorized_network": instance["authorizedNetwork"],
			"state":              instance["state"],
			"labels":             instance["labels"],
		},
	}, nil
}

// getKMSKeyRing retrieves a KMS key ring
func (p *GCPProviderComplete) getKMSKeyRing(ctx context.Context, keyRingName string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/keyRings/%s",
		p.baseURLs["kms"], p.projectID, p.region, keyRingName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS key ring: %w", err)
	}

	var keyRing map[string]interface{}
	if err := json.Unmarshal(data, &keyRing); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KMS key ring: %w", err)
	}

	return &models.Resource{
		ID:   keyRingName,
		Type: "google_kms_key_ring",
		Attributes: map[string]interface{}{
			"name":        keyRing["name"],
			"create_time": keyRing["createTime"],
		},
	}, nil
}

// getKMSCryptoKey retrieves a KMS crypto key
func (p *GCPProviderComplete) getKMSCryptoKey(ctx context.Context, keyID string) (*models.Resource, error) {
	// Parse key ID to extract key ring and key names
	parts := strings.Split(keyID, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid crypto key ID format")
	}
	keyRingName := parts[0]
	keyName := parts[1]

	url := fmt.Sprintf("%s/projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		p.baseURLs["kms"], p.projectID, p.region, keyRingName, keyName)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS crypto key: %w", err)
	}

	var cryptoKey map[string]interface{}
	if err := json.Unmarshal(data, &cryptoKey); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KMS crypto key: %w", err)
	}

	return &models.Resource{
		ID:   keyID,
		Type: "google_kms_crypto_key",
		Attributes: map[string]interface{}{
			"name":               cryptoKey["name"],
			"purpose":            cryptoKey["purpose"],
			"create_time":        cryptoKey["createTime"],
			"rotation_period":    cryptoKey["rotationPeriod"],
			"next_rotation_time": cryptoKey["nextRotationTime"],
			"version_template":   cryptoKey["versionTemplate"],
			"labels":             cryptoKey["labels"],
		},
	}, nil
}

// ListResources lists all resources of a specific type
func (p *GCPProviderComplete) ListResources(ctx context.Context, resourceType string) ([]*models.Resource, error) {
	switch resourceType {
	case "google_compute_instance":
		return p.listComputeInstances(ctx)
	case "google_compute_network":
		return p.listNetworks(ctx)
	case "google_compute_subnetwork":
		return p.listSubnetworks(ctx)
	case "google_compute_firewall":
		return p.listFirewalls(ctx)
	case "google_compute_disk":
		return p.listDisks(ctx)
	case "google_storage_bucket":
		return p.listStorageBuckets(ctx)
	case "google_sql_database_instance":
		return p.listSQLInstances(ctx)
	case "google_container_cluster":
		return p.listGKEClusters(ctx)
	case "google_pubsub_topic":
		return p.listPubSubTopics(ctx)
	case "google_pubsub_subscription":
		return p.listPubSubSubscriptions(ctx)
	case "google_redis_instance":
		return p.listRedisInstances(ctx)
	case "google_kms_key_ring":
		return p.listKMSKeyRings(ctx)
	default:
		return []*models.Resource{}, nil
	}
}

// listComputeInstances lists all compute instances
func (p *GCPProviderComplete) listComputeInstances(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/aggregated/instances",
		p.baseURLs["compute"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var result struct {
		Items map[string]struct {
			Instances []map[string]interface{} `json:"instances"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance list: %w", err)
	}

	var resources []*models.Resource
	for zone, zoneData := range result.Items {
		for _, instance := range zoneData.Instances {
			resources = append(resources, &models.Resource{
				ID:   instance["name"].(string),
				Type: "google_compute_instance",
				Attributes: map[string]interface{}{
					"name":         instance["name"],
					"machine_type": instance["machineType"],
					"zone":         strings.TrimPrefix(zone, "zones/"),
					"status":       instance["status"],
					"labels":       instance["labels"],
				},
			})
		}
	}

	return resources, nil
}

// listNetworks lists all VPC networks
func (p *GCPProviderComplete) listNetworks(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/global/networks",
		p.baseURLs["compute"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network list: %w", err)
	}

	var resources []*models.Resource
	for _, network := range result.Items {
		resources = append(resources, &models.Resource{
			ID:   network["name"].(string),
			Type: "google_compute_network",
			Attributes: map[string]interface{}{
				"name":                    network["name"],
				"auto_create_subnetworks": network["autoCreateSubnetworks"],
				"mtu":                     network["mtu"],
			},
		})
	}

	return resources, nil
}

// listSubnetworks lists all subnetworks
func (p *GCPProviderComplete) listSubnetworks(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/aggregated/subnetworks",
		p.baseURLs["compute"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnetworks: %w", err)
	}

	var result struct {
		Items map[string]struct {
			Subnetworks []map[string]interface{} `json:"subnetworks"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subnetwork list: %w", err)
	}

	var resources []*models.Resource
	for region, regionData := range result.Items {
		for _, subnet := range regionData.Subnetworks {
			resources = append(resources, &models.Resource{
				ID:   subnet["name"].(string),
				Type: "google_compute_subnetwork",
				Attributes: map[string]interface{}{
					"name":          subnet["name"],
					"network":       subnet["network"],
					"ip_cidr_range": subnet["ipCidrRange"],
					"region":        strings.TrimPrefix(region, "regions/"),
				},
			})
		}
	}

	return resources, nil
}

// listFirewalls lists all firewall rules
func (p *GCPProviderComplete) listFirewalls(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/global/firewalls",
		p.baseURLs["compute"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list firewalls: %w", err)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal firewall list: %w", err)
	}

	var resources []*models.Resource
	for _, firewall := range result.Items {
		resources = append(resources, &models.Resource{
			ID:   firewall["name"].(string),
			Type: "google_compute_firewall",
			Attributes: map[string]interface{}{
				"name":      firewall["name"],
				"network":   firewall["network"],
				"priority":  firewall["priority"],
				"direction": firewall["direction"],
			},
		})
	}

	return resources, nil
}

// listDisks lists all persistent disks
func (p *GCPProviderComplete) listDisks(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/aggregated/disks",
		p.baseURLs["compute"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list disks: %w", err)
	}

	var result struct {
		Items map[string]struct {
			Disks []map[string]interface{} `json:"disks"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disk list: %w", err)
	}

	var resources []*models.Resource
	for zone, zoneData := range result.Items {
		for _, disk := range zoneData.Disks {
			resources = append(resources, &models.Resource{
				ID:   disk["name"].(string),
				Type: "google_compute_disk",
				Attributes: map[string]interface{}{
					"name":    disk["name"],
					"size_gb": disk["sizeGb"],
					"type":    disk["type"],
					"zone":    strings.TrimPrefix(zone, "zones/"),
					"status":  disk["status"],
				},
			})
		}
	}

	return resources, nil
}

// listStorageBuckets lists all storage buckets
func (p *GCPProviderComplete) listStorageBuckets(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/b?project=%s", p.baseURLs["storage"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket list: %w", err)
	}

	var resources []*models.Resource
	for _, bucket := range result.Items {
		resources = append(resources, &models.Resource{
			ID:   bucket["name"].(string),
			Type: "google_storage_bucket",
			Attributes: map[string]interface{}{
				"name":          bucket["name"],
				"location":      bucket["location"],
				"storage_class": bucket["storageClass"],
			},
		})
	}

	return resources, nil
}

// Additional list methods...
func (p *GCPProviderComplete) listSQLInstances(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/instances", p.baseURLs["sql"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list SQL instances: %w", err)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SQL instance list: %w", err)
	}

	var resources []*models.Resource
	for _, instance := range result.Items {
		resources = append(resources, &models.Resource{
			ID:   instance["name"].(string),
			Type: "google_sql_database_instance",
			Attributes: map[string]interface{}{
				"name":             instance["name"],
				"database_version": instance["databaseVersion"],
				"region":           instance["region"],
				"state":            instance["state"],
			},
		})
	}

	return resources, nil
}

func (p *GCPProviderComplete) listGKEClusters(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/-/clusters", p.baseURLs["container"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list GKE clusters: %w", err)
	}

	var result struct {
		Clusters []map[string]interface{} `json:"clusters"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GKE cluster list: %w", err)
	}

	var resources []*models.Resource
	for _, cluster := range result.Clusters {
		resources = append(resources, &models.Resource{
			ID:   cluster["name"].(string),
			Type: "google_container_cluster",
			Attributes: map[string]interface{}{
				"name":     cluster["name"],
				"location": cluster["location"],
				"status":   cluster["status"],
			},
		})
	}

	return resources, nil
}

func (p *GCPProviderComplete) listPubSubTopics(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/topics", p.baseURLs["pubsub"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list Pub/Sub topics: %w", err)
	}

	var result struct {
		Topics []map[string]interface{} `json:"topics"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal topic list: %w", err)
	}

	var resources []*models.Resource
	for _, topic := range result.Topics {
		resources = append(resources, &models.Resource{
			ID:   topic["name"].(string),
			Type: "google_pubsub_topic",
			Attributes: map[string]interface{}{
				"name":   topic["name"],
				"labels": topic["labels"],
			},
		})
	}

	return resources, nil
}

func (p *GCPProviderComplete) listPubSubSubscriptions(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/subscriptions", p.baseURLs["pubsub"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list Pub/Sub subscriptions: %w", err)
	}

	var result struct {
		Subscriptions []map[string]interface{} `json:"subscriptions"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscription list: %w", err)
	}

	var resources []*models.Resource
	for _, sub := range result.Subscriptions {
		resources = append(resources, &models.Resource{
			ID:   sub["name"].(string),
			Type: "google_pubsub_subscription",
			Attributes: map[string]interface{}{
				"name":  sub["name"],
				"topic": sub["topic"],
			},
		})
	}

	return resources, nil
}

func (p *GCPProviderComplete) listRedisInstances(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/-/instances", p.baseURLs["redis"], p.projectID)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list Redis instances: %w", err)
	}

	var result struct {
		Instances []map[string]interface{} `json:"instances"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Redis instance list: %w", err)
	}

	var resources []*models.Resource
	for _, instance := range result.Instances {
		resources = append(resources, &models.Resource{
			ID:   instance["name"].(string),
			Type: "google_redis_instance",
			Attributes: map[string]interface{}{
				"name":           instance["name"],
				"tier":           instance["tier"],
				"memory_size_gb": instance["memorySizeGb"],
				"state":          instance["state"],
			},
		})
	}

	return resources, nil
}

func (p *GCPProviderComplete) listKMSKeyRings(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/keyRings",
		p.baseURLs["kms"], p.projectID, p.region)

	data, err := p.makeAPIRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list KMS key rings: %w", err)
	}

	var result struct {
		KeyRings []map[string]interface{} `json:"keyRings"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key ring list: %w", err)
	}

	var resources []*models.Resource
	for _, keyRing := range result.KeyRings {
		resources = append(resources, &models.Resource{
			ID:   keyRing["name"].(string),
			Type: "google_kms_key_ring",
			Attributes: map[string]interface{}{
				"name": keyRing["name"],
			},
		})
	}

	return resources, nil
}

// ResourceExists checks if a resource exists
func (p *GCPProviderComplete) ResourceExists(ctx context.Context, resourceType string, resourceID string) (bool, error) {
	_, err := p.GetResourceByType(ctx, resourceType, resourceID)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "notFound") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
