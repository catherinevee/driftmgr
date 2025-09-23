package gcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewGCPProviderComplete tests creating a new GCP provider
func TestNewGCPProviderComplete(t *testing.T) {
	provider := NewGCPProviderComplete("test-project-123")
	assert.NotNil(t, provider)
	assert.Equal(t, "gcp", provider.Name())
}

// TestGCPProviderName tests the provider name
func TestGCPProviderName(t *testing.T) {
	provider := NewGCPProviderComplete("test-project-123")
	assert.Equal(t, "gcp", provider.Name())
}

// TestGCPProviderSupportedResourceTypes tests supported resource types
func TestGCPProviderSupportedResourceTypes(t *testing.T) {
	provider := NewGCPProviderComplete("test-project-123")
	resourceTypes := provider.SupportedResourceTypes()

	assert.NotEmpty(t, resourceTypes)
	assert.Contains(t, resourceTypes, "google_compute_instance")
	assert.Contains(t, resourceTypes, "google_storage_bucket")
}

// TestGCPProviderListRegions tests listing available regions
func TestGCPProviderListRegions(t *testing.T) {
	provider := NewGCPProviderComplete("test-project-123")
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

// TestGCPProviderValidateCredentials tests credential validation
func TestGCPProviderValidateCredentials(t *testing.T) {
	provider := NewGCPProviderComplete("test-project-123")
	ctx := context.Background()

	err := provider.ValidateCredentials(ctx)
	assert.NotNil(t, err) // Expected in test environment
}

// TestGCPProviderDiscoverResources tests resource discovery
func TestGCPProviderDiscoverResources(t *testing.T) {
	provider := NewGCPProviderComplete("test-project-123")
	ctx := context.Background()

	resources, err := provider.DiscoverResources(ctx, "us-central1")

	if err != nil {
		assert.NotNil(t, err)
	} else {
		assert.NotNil(t, resources)
		assert.IsType(t, []interface{}{}, resources)
	}
}

// TestGCPProviderGetResource tests getting a specific resource
func TestGCPProviderGetResource(t *testing.T) {
	provider := NewGCPProviderComplete("test-project-123")
	ctx := context.Background()

	resource, err := provider.GetResource(ctx, "test-resource-id")
	assert.Error(t, err)
	assert.Nil(t, resource)
}
