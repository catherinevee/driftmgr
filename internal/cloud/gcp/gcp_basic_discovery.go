package gcp

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// GCPBasicDiscoverer discovers basic GCP resources available in all projects
type GCPBasicDiscoverer struct {
	projectID string
}

// NewGCPBasicDiscoverer creates a new basic GCP discoverer
func NewGCPBasicDiscoverer(projectID string) *GCPBasicDiscoverer {
	if projectID == "" {
		// Try to get current project
		cmd := exec.Command("gcloud", "config", "get-value", "project")
		if output, err := cmd.Output(); err == nil {
			projectID = strings.TrimSpace(string(output))
		}
	}

	return &GCPBasicDiscoverer{
		projectID: projectID,
	}
}

// DiscoverBasicGCPResources discovers basic GCP resources that don't require API enablement
func DiscoverBasicGCPResources(regions []string, provider string) []models.Resource {
	discoverer := NewGCPBasicDiscoverer("")
	return discoverer.DiscoverAll(provider)
}

// DiscoverAll discovers all basic GCP resources
func (g *GCPBasicDiscoverer) DiscoverAll(provider string) []models.Resource {
	var resources []models.Resource

	log.Printf("Starting basic GCP discovery for project: %s", g.projectID)

	// Discover the project itself
	resources = append(resources, g.discoverProject(provider)...)

	// Discover logging resources (always available)
	resources = append(resources, g.discoverLoggingSinks(provider)...)
	resources = append(resources, g.discoverLoggingBuckets(provider)...)

	// Discover IAM resources (always available)
	resources = append(resources, g.discoverServiceAccounts(provider)...)
	resources = append(resources, g.discoverIAMRoles(provider)...)

	// Discover enabled services
	resources = append(resources, g.discoverEnabledServices(provider)...)

	// Try to discover BigQuery datasets (if BigQuery API is enabled, which it often is by default)
	resources = append(resources, g.discoverBigQueryDatasets(provider)...)

	// Try to discover storage buckets (basic API usually enabled)
	resources = append(resources, g.discoverStorageBuckets(provider)...)

	// Try to discover billing info
	resources = append(resources, g.discoverBillingInfo(provider)...)

	log.Printf("Basic GCP discovery completed: %d resources found", len(resources))
	return resources
}

// discoverProject discovers the GCP project itself
func (g *GCPBasicDiscoverer) discoverProject(provider string) []models.Resource {
	var resources []models.Resource

	if g.projectID == "" {
		return resources
	}

	cmd := exec.Command("gcloud", "projects", "describe", g.projectID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to describe project %s: %v", g.projectID, err)
		return resources
	}

	var project map[string]interface{}
	if err := json.Unmarshal(output, &project); err != nil {
		log.Printf("Warning: Failed to parse project description: %v", err)
		return resources
	}

	resource := models.Resource{
		ID:        g.projectID,
		Name:      getStringValueBasic(project, "name"),
		Type:      "gcp_project",
		Provider:  provider,
		Region:    "global",
		Status:    getStringValueBasic(project, "lifecycleState"),
		CreatedAt: parseGCPTime(getStringValueBasic(project, "createTime")),
		Tags:      extractLabelsGCPBasic(project),
		Attributes: map[string]interface{}{
			"project_id":      g.projectID,
			"project_number":  getStringValueBasic(project, "projectNumber"),
			"lifecycle_state": getStringValueBasic(project, "lifecycleState"),
		},
	}

	resources = append(resources, resource)
	return resources
}

// discoverLoggingSinks discovers Cloud Logging sinks
func (g *GCPBasicDiscoverer) discoverLoggingSinks(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("gcloud", "logging", "sinks", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to list logging sinks: %v", err)
		return resources
	}

	var sinks []map[string]interface{}
	if err := json.Unmarshal(output, &sinks); err != nil {
		log.Printf("Warning: Failed to parse logging sinks: %v", err)
		return resources
	}

	for _, sink := range sinks {
		resource := models.Resource{
			ID:        getStringValueBasic(sink, "name"),
			Name:      getStringValueBasic(sink, "name"),
			Type:      "gcp_logging_sink",
			Provider:  provider,
			Region:    "global",
			Status:    "active",
			CreatedAt: time.Now(),
			Attributes: map[string]interface{}{
				"destination": getStringValueBasic(sink, "destination"),
				"filter":      getStringValueBasic(sink, "filter"),
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// discoverLoggingBuckets discovers Cloud Logging buckets
func (g *GCPBasicDiscoverer) discoverLoggingBuckets(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("gcloud", "logging", "buckets", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to list logging buckets: %v", err)
		return resources
	}

	var buckets []map[string]interface{}
	if err := json.Unmarshal(output, &buckets); err != nil {
		log.Printf("Warning: Failed to parse logging buckets: %v", err)
		return resources
	}

	for _, bucket := range buckets {
		resource := models.Resource{
			ID:        getStringValueBasic(bucket, "name"),
			Name:      getStringValueBasic(bucket, "name"),
			Type:      "gcp_logging_bucket",
			Provider:  provider,
			Region:    getStringValueBasic(bucket, "location"),
			Status:    getStringValueBasic(bucket, "lifecycleState"),
			CreatedAt: parseGCPTime(getStringValueBasic(bucket, "createTime")),
			Attributes: map[string]interface{}{
				"retention_days": getIntValue(bucket, "retentionDays"),
				"locked":         getBoolValue(bucket, "locked"),
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// discoverServiceAccounts discovers IAM service accounts
func (g *GCPBasicDiscoverer) discoverServiceAccounts(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("gcloud", "iam", "service-accounts", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to list service accounts: %v", err)
		return resources
	}

	var accounts []map[string]interface{}
	if err := json.Unmarshal(output, &accounts); err != nil {
		log.Printf("Warning: Failed to parse service accounts: %v", err)
		return resources
	}

	for _, account := range accounts {
		resource := models.Resource{
			ID:        getStringValueBasic(account, "uniqueId"),
			Name:      getStringValueBasic(account, "displayName"),
			Type:      "gcp_service_account",
			Provider:  provider,
			Region:    "global",
			Status:    "active",
			CreatedAt: time.Now(),
			Attributes: map[string]interface{}{
				"email":        getStringValueBasic(account, "email"),
				"unique_id":    getStringValueBasic(account, "uniqueId"),
				"display_name": getStringValueBasic(account, "displayName"),
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// discoverIAMRoles discovers custom IAM roles
func (g *GCPBasicDiscoverer) discoverIAMRoles(provider string) []models.Resource {
	var resources []models.Resource

	// Only list custom roles to avoid overwhelming output with predefined roles
	cmd := exec.Command("gcloud", "iam", "roles", "list", "--project", g.projectID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to list custom IAM roles: %v", err)
		return resources
	}

	var roles []map[string]interface{}
	if err := json.Unmarshal(output, &roles); err != nil {
		log.Printf("Warning: Failed to parse IAM roles: %v", err)
		return resources
	}

	for _, role := range roles {
		resource := models.Resource{
			ID:        getStringValueBasic(role, "name"),
			Name:      getStringValueBasic(role, "title"),
			Type:      "gcp_iam_role",
			Provider:  provider,
			Region:    "global",
			Status:    getStringValueBasic(role, "stage"),
			CreatedAt: time.Now(),
			Attributes: map[string]interface{}{
				"description": getStringValueBasic(role, "description"),
				"stage":       getStringValueBasic(role, "stage"),
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// discoverEnabledServices discovers enabled Google Cloud services
func (g *GCPBasicDiscoverer) discoverEnabledServices(provider string) []models.Resource {
	var resources []models.Resource

	cmd := exec.Command("gcloud", "services", "list", "--enabled", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to list enabled services: %v", err)
		return resources
	}

	var services []map[string]interface{}
	if err := json.Unmarshal(output, &services); err != nil {
		log.Printf("Warning: Failed to parse enabled services: %v", err)
		return resources
	}

	// Limit to first 15 services to avoid overwhelming output
	for i, service := range services {
		if i >= 15 {
			break
		}

		serviceName := getStringValueBasic(service, "name")
		resource := models.Resource{
			ID:        fmt.Sprintf("service-%s", strings.ReplaceAll(serviceName, ".", "-")),
			Name:      getStringValueBasic(service, "title"),
			Type:      "gcp_service",
			Provider:  provider,
			Region:    "global",
			Status:    "enabled",
			CreatedAt: time.Now(),
			Attributes: map[string]interface{}{
				"service_name": serviceName,
				"title":        getStringValueBasic(service, "title"),
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// discoverBigQueryDatasets discovers BigQuery datasets (if API is enabled)
func (g *GCPBasicDiscoverer) discoverBigQueryDatasets(provider string) []models.Resource {
	var resources []models.Resource

	// Try BigQuery datasets - many projects have this enabled by default
	cmd := exec.Command("bq", "ls", "--format", "json", "--max_results", "10")
	output, err := cmd.Output()
	if err != nil {
		// BigQuery API not enabled or bq tool not available
		return resources
	}

	var datasets []map[string]interface{}
	if err := json.Unmarshal(output, &datasets); err != nil {
		return resources
	}

	for _, dataset := range datasets {
		datasetRef := getNestedValue(dataset, "datasetReference")
		if datasetRef != nil {
			if datasetMap, ok := datasetRef.(map[string]interface{}); ok {
				datasetId := getStringValueBasic(datasetMap, "datasetId")
				resource := models.Resource{
					ID:        fmt.Sprintf("projects/%s/datasets/%s", g.projectID, datasetId),
					Name:      datasetId,
					Type:      "gcp_bigquery_dataset",
					Provider:  provider,
					Region:    getStringValueBasic(dataset, "location"),
					Status:    "active",
					CreatedAt: parseGCPTimeMillis(getStringValueBasic(dataset, "creationTime")),
					Attributes: map[string]interface{}{
						"dataset_id":    datasetId,
						"location":      getStringValueBasic(dataset, "location"),
						"creation_time": getStringValueBasic(dataset, "creationTime"),
						"last_modified": getStringValueBasic(dataset, "lastModifiedTime"),
					},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// discoverStorageBuckets discovers Cloud Storage buckets (if API is accessible)
func (g *GCPBasicDiscoverer) discoverStorageBuckets(provider string) []models.Resource {
	var resources []models.Resource

	// Try to list storage buckets
	cmd := exec.Command("gcloud", "storage", "buckets", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// Storage API not accessible or gcloud storage not available
		return resources
	}

	var buckets []map[string]interface{}
	if err := json.Unmarshal(output, &buckets); err != nil {
		return resources
	}

	for _, bucket := range buckets {
		resource := models.Resource{
			ID:        getStringValueBasic(bucket, "name"),
			Name:      getStringValueBasic(bucket, "name"),
			Type:      "gcp_storage_bucket",
			Provider:  provider,
			Region:    getStringValueBasic(bucket, "location"),
			Status:    "active",
			CreatedAt: parseGCPTime(getStringValueBasic(bucket, "timeCreated")),
			Attributes: map[string]interface{}{
				"storage_class": getStringValueBasic(bucket, "storageClass"),
				"location_type": getStringValueBasic(bucket, "locationType"),
				"time_created":  getStringValueBasic(bucket, "timeCreated"),
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// discoverBillingInfo discovers billing information
func (g *GCPBasicDiscoverer) discoverBillingInfo(provider string) []models.Resource {
	var resources []models.Resource

	// Try to get billing info for the project
	cmd := exec.Command("gcloud", "billing", "projects", "describe", g.projectID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// Billing API not accessible
		return resources
	}

	var billing map[string]interface{}
	if err := json.Unmarshal(output, &billing); err != nil {
		return resources
	}

	if billingEnabled := getBoolValue(billing, "billingEnabled"); billingEnabled {
		resource := models.Resource{
			ID:        fmt.Sprintf("billing-%s", g.projectID),
			Name:      fmt.Sprintf("Billing for %s", g.projectID),
			Type:      "gcp_billing_info",
			Provider:  provider,
			Region:    "global",
			Status:    "enabled",
			CreatedAt: time.Now(),
			Attributes: map[string]interface{}{
				"billing_account_name": getStringValueBasic(billing, "billingAccountName"),
				"billing_enabled":      billingEnabled,
				"project_id":           g.projectID,
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// Helper functions

func getStringValueBasic(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntValue(data map[string]interface{}, key string) int64 {
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

func getBoolValue(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getNestedValue(data map[string]interface{}, path string) interface{} {
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

func extractLabelsGCPBasic(data map[string]interface{}) map[string]string {
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

func parseGCPTime(timeStr string) time.Time {
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

func parseGCPTimeMillis(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// BigQuery time is in milliseconds since epoch
	if millis, err := time.Parse("1234567890123", timeStr); err == nil {
		return millis
	}

	return parseGCPTime(timeStr)
}
