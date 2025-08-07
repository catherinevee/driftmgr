package discovery

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewGCPProvider(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful provider creation",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGCPProvider()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				// Should not error even if GCP credentials are not available
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestGCPProvider_Discover(t *testing.T) {
	provider, err := NewGCPProvider()
	assert.NoError(t, err)
	assert.NotNil(t, provider)

	tests := []struct {
		name         string
		config       models.DiscoveryConfig
		wantErr      bool
		minResources int
	}{
		{
			name: "discover with default config",
			config: models.DiscoveryConfig{
				Provider: "gcp",
				Regions:  []string{"us-central1"},
			},
			wantErr:      false,
			minResources: 0,
		},
		{
			name: "discover multiple regions",
			config: models.DiscoveryConfig{
				Provider: "gcp",
				Regions:  []string{"us-central1", "us-east1"},
			},
			wantErr:      false,
			minResources: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := provider.Discover(context.Background(), tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(resources), tt.minResources)

				// Verify resource structure
				for _, resource := range resources {
					assert.NotEmpty(t, resource.ID)
					assert.NotEmpty(t, resource.Name)
					assert.NotEmpty(t, resource.Type)
					assert.NotEmpty(t, resource.Region)
					assert.Equal(t, "gcp", resource.Provider)
				}
			}
		})
	}
}

func TestGCPProvider_GetMockData(t *testing.T) {
	provider, err := NewGCPProvider()
	assert.NoError(t, err)

	config := models.DiscoveryConfig{
		Provider: "gcp",
		Regions:  []string{"us-central1"},
	}

	resources := provider.getMockData(config)

	assert.GreaterOrEqual(t, len(resources), 3) // Should have at least 3 mock resources

	// Verify resource types
	resourceTypes := make(map[string]bool)
	for _, resource := range resources {
		resourceTypes[resource.Type] = true
	}

	expectedTypes := []string{
		"google_compute_instance",
		"google_storage_bucket",
		"google_compute_network",
	}

	for _, expectedType := range expectedTypes {
		assert.True(t, resourceTypes[expectedType], "Expected resource type %s not found", expectedType)
	}
}

func TestGCPProvider_GCPTypeToTerraformType(t *testing.T) {
	tests := []struct {
		gcpType      string
		expectedType string
	}{
		{"compute#instance", "google_compute_instance"},
		{"storage#bucket", "google_storage_bucket"},
		{"compute#network", "google_compute_network"},
		{"compute#firewall", "google_compute_firewall"},
		{"sql#instance", "google_sql_database_instance"},
		{"unknown#type", "google_unknown_type"},
	}

	provider, err := NewGCPProvider()
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.gcpType, func(t *testing.T) {
			result := provider.gcpTypeToTerraformType(tt.gcpType)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestGCPProvider_ParseInstanceName(t *testing.T) {
	tests := []struct {
		selfLink     string
		expectedName string
	}{
		{
			"https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instances/my-instance",
			"my-instance",
		},
		{
			"projects/my-project/zones/us-central1-a/instances/my-instance",
			"my-instance",
		},
		{
			"my-instance",
			"my-instance",
		},
		{
			"",
			"",
		},
	}

	provider, err := NewGCPProvider()
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.selfLink, func(t *testing.T) {
			result := provider.parseInstanceName(tt.selfLink)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestGCPProvider_ParseZoneFromSelfLink(t *testing.T) {
	tests := []struct {
		selfLink     string
		expectedZone string
	}{
		{
			"https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instances/my-instance",
			"us-central1-a",
		},
		{
			"projects/my-project/zones/us-central1-a/instances/my-instance",
			"us-central1-a",
		},
		{
			"invalid-format",
			"",
		},
		{
			"",
			"",
		},
	}

	provider, err := NewGCPProvider()
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.selfLink, func(t *testing.T) {
			result := provider.parseZoneFromSelfLink(tt.selfLink)
			assert.Equal(t, tt.expectedZone, result)
		})
	}
}

// Integration test for real GCP API (only runs with credentials)
func TestGCPProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	provider, err := NewGCPProvider()
	assert.NoError(t, err)

	config := models.DiscoveryConfig{
		Provider: "gcp",
		Regions:  []string{"us-central1"},
	}

	// This test will use real GCP credentials if available, mock data otherwise
	resources, err := provider.Discover(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, resources)

	t.Logf("Discovered %d GCP resources", len(resources))

	// Log resource details for manual verification
	for i, resource := range resources {
		if i < 5 { // Limit output
			t.Logf("Resource %d: %s (%s) in %s", i+1, resource.Name, resource.Type, resource.Region)
		}
	}
}
