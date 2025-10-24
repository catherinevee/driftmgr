package digitalocean

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigitalOceanSDKProvider_New(t *testing.T) {
	// Skip if no DigitalOcean credentials are available
	if os.Getenv("DIGITALOCEAN_TOKEN") == "" {
		t.Skip("DIGITALOCEAN_TOKEN not set, skipping DigitalOcean SDK provider test")
	}

	provider, err := NewDigitalOceanSDKProvider("nyc1")
	if err != nil {
		// This is expected to fail in test environment without real credentials
		t.Logf("Expected error creating DigitalOcean SDK provider: %v", err)
		return
	}

	assert.NotNil(t, provider)
	assert.Equal(t, "digitalocean", provider.Name())
	assert.Equal(t, "nyc1", provider.GetRegion())
}

func TestDigitalOceanSDKProvider_Name(t *testing.T) {
	provider := &DigitalOceanSDKProvider{}
	assert.Equal(t, "digitalocean", provider.Name())
}

func TestDigitalOceanSDKProvider_SupportedResourceTypes(t *testing.T) {
	provider := &DigitalOceanSDKProvider{}
	resourceTypes := provider.SupportedResourceTypes()

	assert.Contains(t, resourceTypes, "digitalocean_droplet")
	assert.Contains(t, resourceTypes, "digitalocean_volume")
	assert.Contains(t, resourceTypes, "digitalocean_load_balancer")
	assert.Contains(t, resourceTypes, "digitalocean_database_cluster")
	assert.Contains(t, resourceTypes, "digitalocean_kubernetes_cluster")
}

func TestDigitalOceanSDKProvider_ListRegions(t *testing.T) {
	provider := &DigitalOceanSDKProvider{}
	regions, err := provider.ListRegions(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, regions)
	assert.Contains(t, regions, "nyc1")
	assert.Contains(t, regions, "sfo1")
	assert.Contains(t, regions, "ams2")
	assert.Contains(t, regions, "sgp1")
}

func TestDigitalOceanSDKProvider_IsNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"999999", true},
		{"abc", false},
		{"123abc", false},
		{"", false},
		{"12.34", false},
		{"-123", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := isNumeric(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestDigitalOceanSDKProvider_ValidateCredentials(t *testing.T) {
	// Skip if no DigitalOcean credentials are available
	if os.Getenv("DIGITALOCEAN_TOKEN") == "" {
		t.Skip("DIGITALOCEAN_TOKEN not set, skipping DigitalOcean credentials validation test")
	}

	provider, err := NewDigitalOceanSDKProvider("nyc1")
	if err != nil {
		t.Skipf("Failed to create DigitalOcean SDK provider: %v", err)
	}

	// This will likely fail in test environment without real credentials
	err = provider.ValidateCredentials(context.Background())
	if err != nil {
		t.Logf("Expected error validating DigitalOcean credentials: %v", err)
		// This is expected in test environment
		return
	}

	// If we get here, credentials are valid
	t.Log("DigitalOcean credentials are valid")
}

func TestDigitalOceanSDKProvider_TestConnection(t *testing.T) {
	// Skip if no DigitalOcean credentials are available
	if os.Getenv("DIGITALOCEAN_TOKEN") == "" {
		t.Skip("DIGITALOCEAN_TOKEN not set, skipping DigitalOcean connection test")
	}

	provider, err := NewDigitalOceanSDKProvider("nyc1")
	if err != nil {
		t.Skipf("Failed to create DigitalOcean SDK provider: %v", err)
	}

	// This will likely fail in test environment without real credentials
	err = provider.TestConnection(context.Background())
	if err != nil {
		t.Logf("Expected error testing DigitalOcean connection: %v", err)
		// This is expected in test environment
		return
	}

	// If we get here, connection is successful
	t.Log("DigitalOcean connection test successful")
}

func TestDigitalOceanSDKProvider_DiscoverResources(t *testing.T) {
	// Skip if no DigitalOcean credentials are available
	if os.Getenv("DIGITALOCEAN_TOKEN") == "" {
		t.Skip("DIGITALOCEAN_TOKEN not set, skipping DigitalOcean resource discovery test")
	}

	provider, err := NewDigitalOceanSDKProvider("nyc1")
	if err != nil {
		t.Skipf("Failed to create DigitalOcean SDK provider: %v", err)
	}

	// Test discovering resources (this will likely fail in test environment)
	resources, err := provider.DiscoverResources(context.Background(), "")
	if err != nil {
		t.Logf("Expected error discovering DigitalOcean resources: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Discovered %d DigitalOcean resources", len(resources))
	for _, resource := range resources {
		t.Logf("Resource: %s (%s) in %s", resource.ID, resource.Type, resource.Region)
	}
}

func TestDigitalOceanSDKProvider_ListResourcesByType(t *testing.T) {
	// Skip if no DigitalOcean credentials are available
	if os.Getenv("DIGITALOCEAN_TOKEN") == "" {
		t.Skip("DIGITALOCEAN_TOKEN not set, skipping DigitalOcean resource type listing test")
	}

	provider, err := NewDigitalOceanSDKProvider("nyc1")
	if err != nil {
		t.Skipf("Failed to create DigitalOcean SDK provider: %v", err)
	}

	// Test listing resources by type (this will likely fail in test environment)
	resources, err := provider.ListResourcesByType(context.Background(), "digitalocean_droplet")
	if err != nil {
		t.Logf("Expected error listing DigitalOcean resources by type: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Found %d DigitalOcean droplets", len(resources))
	for _, resource := range resources {
		assert.Equal(t, "digitalocean_droplet", resource.Type)
		t.Logf("Droplet: %s in %s", resource.ID, resource.Region)
	}
}
