package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiCloudDiscovery tests real cloud discovery across multiple providers
func TestMultiCloudDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create cloud discoverer
	discoverer := discovery.NewCloudDiscoverer()

	// Add real cloud providers
	t.Run("AWS Discovery", func(t *testing.T) {
		awsProvider, err := discovery.NewAWSProvider()
		if err != nil {
			t.Skipf("AWS provider not available: %v", err)
			return
		}

		discoverer.AddProvider("aws", awsProvider)

		config := discovery.Config{
			Regions: []string{"us-east-1"},
		}

		resources, err := discoverer.DiscoverProvider(ctx, "aws", config)
		if err != nil {
			// Skip if credentials are not available
			t.Skipf("AWS discovery failed (likely missing credentials): %v", err)
			return
		}

		assert.NotNil(t, resources)
		t.Logf("Discovered %d AWS resources", len(resources))

		for _, resource := range resources {
			assert.NotEmpty(t, resource.ID)
			assert.NotEmpty(t, resource.Type)
			assert.Equal(t, "aws", resource.Provider)
		}
	})

	t.Run("Azure Discovery", func(t *testing.T) {
		azureProvider, err := discovery.NewAzureProvider()
		if err != nil {
			t.Skipf("Azure provider not available: %v", err)
			return
		}

		discoverer.AddProvider("azure", azureProvider)

		config := discovery.Config{
			Regions: []string{"eastus"},
		}

		resources, err := discoverer.DiscoverProvider(ctx, "azure", config)
		if err != nil {
			// Skip if credentials are not available
			t.Skipf("Azure discovery failed (likely missing credentials): %v", err)
			return
		}

		assert.NotNil(t, resources)
		t.Logf("Discovered %d Azure resources", len(resources))

		for _, resource := range resources {
			assert.NotEmpty(t, resource.ID)
			assert.NotEmpty(t, resource.Type)
			assert.Equal(t, "azure", resource.Provider)
		}
	})

	t.Run("GCP Discovery", func(t *testing.T) {
		gcpProvider, err := discovery.NewGCPProvider()
		if err != nil {
			t.Skipf("GCP provider not available: %v", err)
			return
		}

		discoverer.AddProvider("gcp", gcpProvider)

		config := discovery.Config{
			Regions: []string{"us-central1"},
		}

		resources, err := discoverer.DiscoverProvider(ctx, "gcp", config)
		if err != nil {
			// Skip if credentials are not available
			t.Skipf("GCP discovery failed (likely missing credentials): %v", err)
			return
		}

		assert.NotNil(t, resources)
		t.Logf("Discovered %d GCP resources", len(resources))

		for _, resource := range resources {
			assert.NotEmpty(t, resource.ID)
			assert.NotEmpty(t, resource.Type)
			assert.Equal(t, "gcp", resource.Provider)
		}
	})

	t.Run("DigitalOcean Discovery", func(t *testing.T) {
		doProvider, err := discovery.NewDigitalOceanProvider()
		if err != nil {
			t.Skipf("DigitalOcean provider not available: %v", err)
			return
		}

		discoverer.AddProvider("digitalocean", doProvider)

		config := discovery.Config{
			Regions: []string{"nyc1"},
		}

		resources, err := discoverer.DiscoverProvider(ctx, "digitalocean", config)
		if err != nil {
			// Skip if credentials are not available
			t.Skipf("DigitalOcean discovery failed (likely missing credentials): %v", err)
			return
		}

		assert.NotNil(t, resources)
		t.Logf("Discovered %d DigitalOcean resources", len(resources))

		for _, resource := range resources {
			assert.NotEmpty(t, resource.ID)
			assert.NotEmpty(t, resource.Type)
			assert.Equal(t, "digitalocean", resource.Provider)
		}
	})

	// Test DiscoverAll
	t.Run("Discover All Providers", func(t *testing.T) {
		allResources, err := discoverer.DiscoverAll(ctx)
		require.NoError(t, err)
		assert.NotNil(t, allResources)

		totalResources := 0
		for provider, resources := range allResources {
			t.Logf("Provider %s: %d resources", provider, len(resources))
			totalResources += len(resources)
		}

		t.Logf("Total resources discovered across all providers: %d", totalResources)
	})
}
