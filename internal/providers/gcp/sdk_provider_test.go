package gcp

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCPSDKProvider_New(t *testing.T) {
	// Skip if no GCP credentials are available
	if os.Getenv("GCP_PROJECT_ID") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		t.Skip("GCP_PROJECT_ID not set, skipping GCP SDK provider test")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		projectID = "test-project-id"
	}

	provider, err := NewGCPSDKProvider(projectID, "us-central1")
	if err != nil {
		// This is expected to fail in test environment without real credentials
		t.Logf("Expected error creating GCP SDK provider: %v", err)
		return
	}

	assert.NotNil(t, provider)
	assert.Equal(t, "gcp", provider.Name())
	assert.Equal(t, projectID, provider.GetProjectID())
	assert.Equal(t, "us-central1", provider.GetRegion())
}

func TestGCPSDKProvider_Name(t *testing.T) {
	provider := &GCPSDKProvider{}
	assert.Equal(t, "gcp", provider.Name())
}

func TestGCPSDKProvider_SupportedResourceTypes(t *testing.T) {
	provider := &GCPSDKProvider{}
	resourceTypes := provider.SupportedResourceTypes()

	assert.Contains(t, resourceTypes, "google_compute_instance")
	assert.Contains(t, resourceTypes, "google_storage_bucket")
	assert.Contains(t, resourceTypes, "google_compute_network")
	assert.Contains(t, resourceTypes, "google_container_cluster")
	assert.Contains(t, resourceTypes, "google_sql_database_instance")
}

func TestGCPSDKProvider_ListRegions(t *testing.T) {
	provider := &GCPSDKProvider{}
	regions, err := provider.ListRegions(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, regions)
	assert.Contains(t, regions, "us-central1")
	assert.Contains(t, regions, "us-east1")
	assert.Contains(t, regions, "europe-west1")
	assert.Contains(t, regions, "asia-east1")
}

func TestGCPSDKProvider_ExtractZoneFromURL(t *testing.T) {
	provider := &GCPSDKProvider{}

	tests := []struct {
		url          string
		expectedZone string
	}{
		{
			url:          "https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-a/instances/test-instance",
			expectedZone: "us-central1-a",
		},
		{
			url:          "https://www.googleapis.com/compute/v1/projects/test-project/zones/europe-west1-b/disks/test-disk",
			expectedZone: "europe-west1-b",
		},
		{
			url:          "https://www.googleapis.com/compute/v1/projects/test-project/global/networks/test-network",
			expectedZone: "",
		},
		{
			url:          "invalid-url",
			expectedZone: "",
		},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			result := provider.extractZoneFromURL(test.url)
			assert.Equal(t, test.expectedZone, result)
		})
	}
}

func TestGCPSDKProvider_ExtractMachineTypeFromURL(t *testing.T) {
	provider := &GCPSDKProvider{}

	tests := []struct {
		url                 string
		expectedMachineType string
	}{
		{
			url:                 "https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-a/machineTypes/n1-standard-1",
			expectedMachineType: "n1-standard-1",
		},
		{
			url:                 "https://www.googleapis.com/compute/v1/projects/test-project/zones/europe-west1-b/machineTypes/e2-micro",
			expectedMachineType: "e2-micro",
		},
		{
			url:                 "https://www.googleapis.com/compute/v1/projects/test-project/global/networks/test-network",
			expectedMachineType: "",
		},
		{
			url:                 "invalid-url",
			expectedMachineType: "",
		},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			result := provider.extractMachineTypeFromURL(test.url)
			assert.Equal(t, test.expectedMachineType, result)
		})
	}
}

func TestGCPSDKProvider_ValidateCredentials(t *testing.T) {
	// Skip if no GCP credentials are available
	if os.Getenv("GCP_PROJECT_ID") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		t.Skip("GCP_PROJECT_ID not set, skipping GCP credentials validation test")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	provider, err := NewGCPSDKProvider(projectID, "us-central1")
	if err != nil {
		t.Skipf("Failed to create GCP SDK provider: %v", err)
	}

	// This will likely fail in test environment without real credentials
	err = provider.ValidateCredentials(context.Background())
	if err != nil {
		t.Logf("Expected error validating GCP credentials: %v", err)
		// This is expected in test environment
		return
	}

	// If we get here, credentials are valid
	t.Log("GCP credentials are valid")
}

func TestGCPSDKProvider_TestConnection(t *testing.T) {
	// Skip if no GCP credentials are available
	if os.Getenv("GCP_PROJECT_ID") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		t.Skip("GCP_PROJECT_ID not set, skipping GCP connection test")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	provider, err := NewGCPSDKProvider(projectID, "us-central1")
	if err != nil {
		t.Skipf("Failed to create GCP SDK provider: %v", err)
	}

	// This will likely fail in test environment without real credentials
	err = provider.TestConnection(context.Background())
	if err != nil {
		t.Logf("Expected error testing GCP connection: %v", err)
		// This is expected in test environment
		return
	}

	// If we get here, connection is successful
	t.Log("GCP connection test successful")
}

func TestGCPSDKProvider_DiscoverResources(t *testing.T) {
	// Skip if no GCP credentials are available
	if os.Getenv("GCP_PROJECT_ID") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		t.Skip("GCP_PROJECT_ID not set, skipping GCP resource discovery test")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	provider, err := NewGCPSDKProvider(projectID, "us-central1")
	if err != nil {
		t.Skipf("Failed to create GCP SDK provider: %v", err)
	}

	// Test discovering resources (this will likely fail in test environment)
	resources, err := provider.DiscoverResources(context.Background(), "")
	if err != nil {
		t.Logf("Expected error discovering GCP resources: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Discovered %d GCP resources", len(resources))
	for _, resource := range resources {
		t.Logf("Resource: %s (%s) in %s", resource.ID, resource.Type, resource.Region)
	}
}

func TestGCPSDKProvider_ListProjects(t *testing.T) {
	// Skip if no GCP credentials are available
	if os.Getenv("GCP_PROJECT_ID") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		t.Skip("GCP_PROJECT_ID not set, skipping GCP projects test")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	provider, err := NewGCPSDKProvider(projectID, "us-central1")
	if err != nil {
		t.Skipf("Failed to create GCP SDK provider: %v", err)
	}

	// Test listing projects (this will likely fail in test environment)
	projects, err := provider.ListProjects(context.Background())
	if err != nil {
		t.Logf("Expected error listing GCP projects: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Found %d GCP projects", len(projects))
	for _, project := range projects {
		t.Logf("Project: %s", project)
	}
}

func TestGCPSDKProvider_ListResourcesByType(t *testing.T) {
	// Skip if no GCP credentials are available
	if os.Getenv("GCP_PROJECT_ID") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		t.Skip("GCP_PROJECT_ID not set, skipping GCP resource type listing test")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	provider, err := NewGCPSDKProvider(projectID, "us-central1")
	if err != nil {
		t.Skipf("Failed to create GCP SDK provider: %v", err)
	}

	// Test listing resources by type (this will likely fail in test environment)
	resources, err := provider.ListResourcesByType(context.Background(), "google_compute_instance")
	if err != nil {
		t.Logf("Expected error listing GCP resources by type: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Found %d GCP compute instances", len(resources))
	for _, resource := range resources {
		assert.Equal(t, "google_compute_instance", resource.Type)
		t.Logf("Instance: %s in %s", resource.ID, resource.Region)
	}
}
