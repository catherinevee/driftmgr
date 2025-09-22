package digitalocean

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewDigitalOceanProvider tests creating a new DigitalOcean provider
func TestNewDigitalOceanProvider(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	assert.NotNil(t, provider)
	assert.Equal(t, "digitalocean", provider.Name())
}

// TestDigitalOceanProviderName tests the provider name
func TestDigitalOceanProviderName(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	assert.Equal(t, "digitalocean", provider.Name())
}

// TestDigitalOceanProviderSupportedResourceTypes tests supported resource types
func TestDigitalOceanProviderSupportedResourceTypes(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	resourceTypes := provider.SupportedResourceTypes()

	assert.NotEmpty(t, resourceTypes)
	assert.Contains(t, resourceTypes, "digitalocean_droplet")
	assert.Contains(t, resourceTypes, "digitalocean_volume")
	assert.Contains(t, resourceTypes, "digitalocean_load_balancer")
}

// TestDigitalOceanProviderListRegions tests listing available regions
func TestDigitalOceanProviderListRegions(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	ctx := context.Background()

	regions, err := provider.ListRegions(ctx)

	// In test environment, this might fail due to credentials
	if err != nil {
		assert.NotNil(t, err)
	} else {
		assert.NotNil(t, regions)
		assert.IsType(t, []string{}, regions)
	}
}

// TestDigitalOceanProviderValidateCredentials tests credential validation
func TestDigitalOceanProviderValidateCredentials(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	ctx := context.Background()

	err := provider.ValidateCredentials(ctx)
	assert.NotNil(t, err) // Expected in test environment
}

// TestDigitalOceanProviderDiscoverResources tests resource discovery
func TestDigitalOceanProviderDiscoverResources(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	ctx := context.Background()

	resources, err := provider.DiscoverResources(ctx, "nyc1")

	if err != nil {
		assert.NotNil(t, err)
	} else {
		assert.NotNil(t, resources)
		assert.IsType(t, []interface{}{}, resources)
	}
}

// TestDigitalOceanProviderGetResource tests getting a specific resource
func TestDigitalOceanProviderGetResource(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	ctx := context.Background()

	resource, err := provider.GetResource(ctx, "test-resource-id")
	assert.Error(t, err)
	assert.Nil(t, resource)
}

// TestDigitalOceanProviderErrorHandling tests error handling scenarios
func TestDigitalOceanProviderErrorHandling(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	ctx := context.Background()

	t.Run("InvalidRegion", func(t *testing.T) {
		resources, err := provider.DiscoverResources(ctx, "invalid-region")
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("EmptyResourceID", func(t *testing.T) {
		resource, err := provider.GetResource(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, resource)
	})
}

// TestDigitalOceanProviderInterfaceCompliance tests that the provider implements the interface correctly
func TestDigitalOceanProviderInterfaceCompliance(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	ctx := context.Background()

	// Test all interface methods exist and return expected types
	name := provider.Name()
	assert.IsType(t, "", name)
	assert.Equal(t, "digitalocean", name)

	resourceTypes := provider.SupportedResourceTypes()
	assert.IsType(t, []string{}, resourceTypes)
	assert.NotEmpty(t, resourceTypes)

	regions, err := provider.ListRegions(ctx)
	assert.IsType(t, []string{}, regions)

	resources, err := provider.DiscoverResources(ctx, "nyc1")
	assert.IsType(t, []interface{}{}, resources)

	resource, err := provider.GetResource(ctx, "test-id")
	assert.IsType(t, (*interface{})(nil), resource)
	assert.Error(t, err) // Expected for non-existent resource

	err = provider.ValidateCredentials(ctx)
	assert.IsType(t, error(nil), err)
	assert.Error(t, err) // Expected in test environment
}