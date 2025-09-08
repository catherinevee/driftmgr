package digitalocean

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// DigitalOceanProvider implements CloudProvider for DigitalOcean
type DigitalOceanProvider struct {
	apiToken   string
	httpClient *http.Client
	baseURL    string
	region     string
}

// DigitalOceanResource represents a DigitalOcean resource
type DigitalOceanResource struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Region   string                 `json:"region"`
	Status   string                 `json:"status"`
	Created  time.Time              `json:"created_at"`
	Updated  time.Time              `json:"updated_at"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DigitalOceanAPIResponse represents a generic API response
type DigitalOceanAPIResponse struct {
	Links   map[string]interface{} `json:"links"`
	Meta    map[string]interface{} `json:"meta"`
	Message string                 `json:"message,omitempty"`
}

// DropletResponse represents the response for droplets
type DropletResponse struct {
	DigitalOceanAPIResponse
	Droplets []Droplet `json:"droplets"`
}

// Droplet represents a DigitalOcean droplet
type Droplet struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Memory      int                    `json:"memory"`
	VCPUs       int                    `json:"vcpus"`
	Disk        int                    `json:"disk"`
	Locked      bool                   `json:"locked"`
	Status      string                 `json:"status"`
	Kernel      map[string]interface{} `json:"kernel"`
	CreatedAt   time.Time              `json:"created_at"`
	Features    []string               `json:"features"`
	BackupIDs   []int                  `json:"backup_ids"`
	SnapshotIDs []int                  `json:"snapshot_ids"`
	Image       map[string]interface{} `json:"image"`
	Size        map[string]interface{} `json:"size"`
	SizeSlug    string                 `json:"size_slug"`
	Networks    map[string]interface{} `json:"networks"`
	Region      map[string]interface{} `json:"region"`
	Tags        []string               `json:"tags"`
	VPCUUID     string                 `json:"vpc_uuid"`
}

// VolumeResponse represents the response for volumes
type VolumeResponse struct {
	DigitalOceanAPIResponse
	Volumes []Volume `json:"volumes"`
}

// Volume represents a DigitalOcean volume
type Volume struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	SizeGigabytes   int                    `json:"size_gigabytes"`
	Description     string                 `json:"description"`
	DropletIDs      []int                  `json:"droplet_ids"`
	Region          map[string]interface{} `json:"region"`
	CreatedAt       time.Time              `json:"created_at"`
	Tags            []string               `json:"tags"`
	FilesystemType  string                 `json:"filesystem_type"`
	FilesystemLabel string                 `json:"filesystem_label"`
}

// LoadBalancerResponse represents the response for load balancers
type LoadBalancerResponse struct {
	DigitalOceanAPIResponse
	LoadBalancers []LoadBalancer `json:"load_balancers"`
}

// LoadBalancer represents a DigitalOcean load balancer
type LoadBalancer struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	IP                  string                 `json:"ip"`
	Algorithm           string                 `json:"algorithm"`
	Status              string                 `json:"status"`
	CreatedAt           time.Time              `json:"created_at"`
	ForwardingRules     []ForwardingRule       `json:"forwarding_rules"`
	HealthCheck         map[string]interface{} `json:"health_check"`
	StickySessions      map[string]interface{} `json:"sticky_sessions"`
	Region              map[string]interface{} `json:"region"`
	Tag                 string                 `json:"tag"`
	DropletIDs          []int                  `json:"droplet_ids"`
	RedirectHTTPToHTTPS bool                   `json:"redirect_http_to_https"`
	EnableProxyProtocol bool                   `json:"enable_proxy_protocol"`
	VPCUUID             string                 `json:"vpc_uuid"`
}

// ForwardingRule represents a load balancer forwarding rule
type ForwardingRule struct {
	EntryProtocol  string `json:"entry_protocol"`
	EntryPort      int    `json:"entry_port"`
	TargetProtocol string `json:"target_protocol"`
	TargetPort     int    `json:"target_port"`
	CertificateID  string `json:"certificate_id,omitempty"`
	TlsPassthrough bool   `json:"tls_passthrough"`
}

// DatabaseResponse represents the response for databases
type DatabaseResponse struct {
	DigitalOceanAPIResponse
	Databases []Database `json:"databases"`
}

// Database represents a DigitalOcean database
type Database struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Engine             string                 `json:"engine"`
	Version            string                 `json:"version"`
	NumNodes           int                    `json:"num_nodes"`
	Size               string                 `json:"size"`
	DBNames            []string               `json:"db_names"`
	Users              []DatabaseUser         `json:"users"`
	Region             string                 `json:"region"`
	Status             string                 `json:"status"`
	CreatedAt          time.Time              `json:"created_at"`
	MaintenanceWindow  map[string]interface{} `json:"maintenance_window"`
	Tags               []string               `json:"tags"`
	PrivateNetworkUUID string                 `json:"private_network_uuid"`
}

// DatabaseUser represents a database user
type DatabaseUser struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password,omitempty"`
}

// NewDigitalOceanProvider creates a new DigitalOcean provider
func NewDigitalOceanProvider(region string) *DigitalOceanProvider {
	if region == "" {
		region = "nyc1"
	}

	return &DigitalOceanProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.digitalocean.com/v2",
		region:  region,
	}
}

// Name returns the provider name
func (p *DigitalOceanProvider) Name() string {
	return "digitalocean"
}

// Initialize initializes the DigitalOcean provider
func (p *DigitalOceanProvider) Initialize(ctx context.Context) error {
	// Get API token from environment
	p.apiToken = os.Getenv("DIGITALOCEAN_TOKEN")
	if p.apiToken == "" {
		return fmt.Errorf("DIGITALOCEAN_TOKEN environment variable is required")
	}

	// Test API connection
	return p.ValidateCredentials(ctx)
}

// Connect establishes connection to DigitalOcean
func (p *DigitalOceanProvider) Connect(ctx context.Context) error {
	return p.Initialize(ctx)
}

// ValidateCredentials validates DigitalOcean credentials
func (p *DigitalOceanProvider) ValidateCredentials(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/account", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid credentials: status %d", resp.StatusCode)
	}

	return nil
}

// GetResource retrieves a specific resource by ID
func (p *DigitalOceanProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	// Try to determine resource type from ID format
	var resourceType string

	switch {
	case strings.HasPrefix(resourceID, "droplet-") || isNumeric(resourceID):
		resourceType = "digitalocean_droplet"
	case strings.HasPrefix(resourceID, "vol-"):
		resourceType = "digitalocean_volume"
	case strings.HasPrefix(resourceID, "lb-"):
		resourceType = "digitalocean_loadbalancer"
	case strings.HasPrefix(resourceID, "db-"):
		resourceType = "digitalocean_database_cluster"
	default:
		// Try different resource types
		for _, rt := range []string{"digitalocean_droplet", "digitalocean_volume", "digitalocean_loadbalancer", "digitalocean_database_cluster"} {
			if resource, err := p.GetResourceByType(ctx, rt, resourceID); err == nil {
				return resource, nil
			}
		}
		return nil, fmt.Errorf("unable to determine resource type for ID: %s", resourceID)
	}

	return p.GetResourceByType(ctx, resourceType, resourceID)
}

// GetResourceByType retrieves a resource by type and ID
func (p *DigitalOceanProvider) GetResourceByType(ctx context.Context, resourceType, resourceID string) (*models.Resource, error) {
	switch resourceType {
	case "digitalocean_droplet":
		return p.getDroplet(ctx, resourceID)
	case "digitalocean_volume":
		return p.getVolume(ctx, resourceID)
	case "digitalocean_loadbalancer":
		return p.getLoadBalancer(ctx, resourceID)
	case "digitalocean_database_cluster":
		return p.getDatabase(ctx, resourceID)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// ListResources lists all resources of a specific type
func (p *DigitalOceanProvider) ListResources(ctx context.Context, resourceType string) ([]*models.Resource, error) {
	switch resourceType {
	case "digitalocean_droplet":
		return p.listDroplets(ctx)
	case "digitalocean_volume":
		return p.listVolumes(ctx)
	case "digitalocean_loadbalancer":
		return p.listLoadBalancers(ctx)
	case "digitalocean_database_cluster":
		return p.listDatabases(ctx)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// getDroplet retrieves a specific droplet
func (p *DigitalOceanProvider) getDroplet(ctx context.Context, dropletID string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/droplets/%s", p.baseURL, dropletID)

	var response struct {
		Droplet Droplet `json:"droplet"`
	}

	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	return p.convertDropletToResource(response.Droplet), nil
}

// listDroplets lists all droplets
func (p *DigitalOceanProvider) listDroplets(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/droplets", p.baseURL)

	var response DropletResponse
	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	resources := make([]*models.Resource, len(response.Droplets))
	for i, droplet := range response.Droplets {
		resources[i] = p.convertDropletToResource(droplet)
	}

	return resources, nil
}

// getVolume retrieves a specific volume
func (p *DigitalOceanProvider) getVolume(ctx context.Context, volumeID string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/volumes/%s", p.baseURL, volumeID)

	var response struct {
		Volume Volume `json:"volume"`
	}

	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	return p.convertVolumeToResource(response.Volume), nil
}

// listVolumes lists all volumes
func (p *DigitalOceanProvider) listVolumes(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/volumes", p.baseURL)

	var response VolumeResponse
	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	resources := make([]*models.Resource, len(response.Volumes))
	for i, volume := range response.Volumes {
		resources[i] = p.convertVolumeToResource(volume)
	}

	return resources, nil
}

// getLoadBalancer retrieves a specific load balancer
func (p *DigitalOceanProvider) getLoadBalancer(ctx context.Context, lbID string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/load_balancers/%s", p.baseURL, lbID)

	var response struct {
		LoadBalancer LoadBalancer `json:"load_balancer"`
	}

	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	return p.convertLoadBalancerToResource(response.LoadBalancer), nil
}

// listLoadBalancers lists all load balancers
func (p *DigitalOceanProvider) listLoadBalancers(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/load_balancers", p.baseURL)

	var response LoadBalancerResponse
	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	resources := make([]*models.Resource, len(response.LoadBalancers))
	for i, lb := range response.LoadBalancers {
		resources[i] = p.convertLoadBalancerToResource(lb)
	}

	return resources, nil
}

// getDatabase retrieves a specific database
func (p *DigitalOceanProvider) getDatabase(ctx context.Context, dbID string) (*models.Resource, error) {
	url := fmt.Sprintf("%s/databases/%s", p.baseURL, dbID)

	var response struct {
		Database Database `json:"database"`
	}

	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	return p.convertDatabaseToResource(response.Database), nil
}

// listDatabases lists all databases
func (p *DigitalOceanProvider) listDatabases(ctx context.Context) ([]*models.Resource, error) {
	url := fmt.Sprintf("%s/databases", p.baseURL)

	var response DatabaseResponse
	if err := p.makeRequest(ctx, url, &response); err != nil {
		return nil, err
	}

	resources := make([]*models.Resource, len(response.Databases))
	for i, db := range response.Databases {
		resources[i] = p.convertDatabaseToResource(db)
	}

	return resources, nil
}

// makeRequest makes an HTTP request to the DigitalOcean API
func (p *DigitalOceanProvider) makeRequest(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// convertDropletToResource converts a DigitalOcean droplet to a models.Resource
func (p *DigitalOceanProvider) convertDropletToResource(droplet Droplet) *models.Resource {
	attributes := map[string]interface{}{
		"id":           droplet.ID,
		"name":         droplet.Name,
		"memory":       droplet.Memory,
		"vcpus":        droplet.VCPUs,
		"disk":         droplet.Disk,
		"locked":       droplet.Locked,
		"status":       droplet.Status,
		"size_slug":    droplet.SizeSlug,
		"created_at":   droplet.CreatedAt,
		"features":     droplet.Features,
		"backup_ids":   droplet.BackupIDs,
		"snapshot_ids": droplet.SnapshotIDs,
		"image":        droplet.Image,
		"size":         droplet.Size,
		"networks":     droplet.Networks,
		"region":       droplet.Region,
		"tags":         droplet.Tags,
		"vpc_uuid":     droplet.VPCUUID,
	}

	return &models.Resource{
		ID:         fmt.Sprintf("%d", droplet.ID),
		Type:       "digitalocean_droplet",
		Provider:   "digitalocean",
		Region:     p.extractRegion(droplet.Region),
		Attributes: attributes,
	}
}

// convertVolumeToResource converts a DigitalOcean volume to a models.Resource
func (p *DigitalOceanProvider) convertVolumeToResource(volume Volume) *models.Resource {
	attributes := map[string]interface{}{
		"id":               volume.ID,
		"name":             volume.Name,
		"size_gigabytes":   volume.SizeGigabytes,
		"description":      volume.Description,
		"droplet_ids":      volume.DropletIDs,
		"created_at":       volume.CreatedAt,
		"tags":             volume.Tags,
		"filesystem_type":  volume.FilesystemType,
		"filesystem_label": volume.FilesystemLabel,
		"region":           volume.Region,
	}

	return &models.Resource{
		ID:         volume.ID,
		Type:       "digitalocean_volume",
		Provider:   "digitalocean",
		Region:     p.extractRegion(volume.Region),
		Attributes: attributes,
	}
}

// convertLoadBalancerToResource converts a DigitalOcean load balancer to a models.Resource
func (p *DigitalOceanProvider) convertLoadBalancerToResource(lb LoadBalancer) *models.Resource {
	attributes := map[string]interface{}{
		"id":                     lb.ID,
		"name":                   lb.Name,
		"ip":                     lb.IP,
		"algorithm":              lb.Algorithm,
		"status":                 lb.Status,
		"created_at":             lb.CreatedAt,
		"forwarding_rules":       lb.ForwardingRules,
		"health_check":           lb.HealthCheck,
		"sticky_sessions":        lb.StickySessions,
		"tag":                    lb.Tag,
		"droplet_ids":            lb.DropletIDs,
		"redirect_http_to_https": lb.RedirectHTTPToHTTPS,
		"enable_proxy_protocol":  lb.EnableProxyProtocol,
		"vpc_uuid":               lb.VPCUUID,
		"region":                 lb.Region,
	}

	return &models.Resource{
		ID:         lb.ID,
		Type:       "digitalocean_loadbalancer",
		Provider:   "digitalocean",
		Region:     p.extractRegion(lb.Region),
		Attributes: attributes,
	}
}

// convertDatabaseToResource converts a DigitalOcean database to a models.Resource
func (p *DigitalOceanProvider) convertDatabaseToResource(db Database) *models.Resource {
	attributes := map[string]interface{}{
		"id":                   db.ID,
		"name":                 db.Name,
		"engine":               db.Engine,
		"version":              db.Version,
		"num_nodes":            db.NumNodes,
		"size":                 db.Size,
		"db_names":             db.DBNames,
		"users":                db.Users,
		"status":               db.Status,
		"created_at":           db.CreatedAt,
		"maintenance_window":   db.MaintenanceWindow,
		"tags":                 db.Tags,
		"private_network_uuid": db.PrivateNetworkUUID,
	}

	return &models.Resource{
		ID:         db.ID,
		Type:       "digitalocean_database_cluster",
		Provider:   "digitalocean",
		Region:     db.Region,
		Attributes: attributes,
	}
}

// extractRegion extracts region from DigitalOcean region object
func (p *DigitalOceanProvider) extractRegion(region interface{}) string {
	if regionMap, ok := region.(map[string]interface{}); ok {
		if slug, ok := regionMap["slug"].(string); ok {
			return slug
		}
	}
	return p.region
}

// DiscoverResources discovers resources in the specified region
func (p *DigitalOceanProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	var allResources []models.Resource

	// Discover all supported resource types
	resourceTypes := p.SupportedResourceTypes()

	for _, resourceType := range resourceTypes {
		resources, err := p.ListResources(ctx, resourceType)
		if err != nil {
			// Log error but continue with other resource types
			fmt.Printf("Warning: Failed to discover %s resources: %v\n", resourceType, err)
			continue
		}

		// Filter by region if specified
		if region != "" {
			for _, resource := range resources {
				if resource.Region == region {
					allResources = append(allResources, *resource)
				}
			}
		} else {
			for _, resource := range resources {
				allResources = append(allResources, *resource)
			}
		}
	}

	return allResources, nil
}

// ListRegions returns available regions for DigitalOcean
func (p *DigitalOceanProvider) ListRegions(ctx context.Context) ([]string, error) {
	// DigitalOcean regions
	regions := []string{
		"nyc1", "nyc2", "nyc3", "sfo1", "sfo2", "sfo3",
		"ams2", "ams3", "sgp1", "lon1", "fra1", "tor1",
		"blr1", "syd1", "sfo3", "nyc3", "ams3", "sgp1",
	}
	return regions, nil
}

// SupportedResourceTypes returns the list of supported resource types
func (p *DigitalOceanProvider) SupportedResourceTypes() []string {
	return []string{
		"digitalocean_droplet",
		"digitalocean_volume",
		"digitalocean_loadbalancer",
		"digitalocean_database_cluster",
		"digitalocean_kubernetes_cluster",
		"digitalocean_vpc",
		"digitalocean_firewall",
		"digitalocean_domain",
		"digitalocean_record",
		"digitalocean_spaces_bucket",
		"digitalocean_certificate",
		"digitalocean_snapshot",
		"digitalocean_image",
		"digitalocean_ssh_key",
		"digitalocean_tag",
		"digitalocean_project",
		"digitalocean_team",
		"digitalocean_alert_policy",
		"digitalocean_monitoring_alert",
		"digitalocean_cdn",
	}
}

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
