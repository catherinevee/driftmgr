package providers

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProvider tests creating new cloud providers
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		config       map[string]interface{}
		expectError  bool
	}{
		{
			name:         "AWS Provider",
			providerType: "aws",
			config: map[string]interface{}{
				"region": "us-east-1",
			},
			expectError: false,
		},
		{
			name:         "Azure Provider",
			providerType: "azure",
			config: map[string]interface{}{
				"region": "eastus",
			},
			expectError: false,
		},
		{
			name:         "GCP Provider",
			providerType: "gcp",
			config: map[string]interface{}{
				"region": "us-central1",
			},
			expectError: false,
		},
		{
			name:         "DigitalOcean Provider",
			providerType: "digitalocean",
			config: map[string]interface{}{
				"region": "nyc1",
			},
			expectError: false,
		},
		{
			name:         "Unknown Provider",
			providerType: "unknown",
			config: map[string]interface{}{
				"region": "us-east-1",
			},
			expectError: true,
		},
		{
			name:         "Empty Provider Type",
			providerType: "",
			config: map[string]interface{}{
				"region": "us-east-1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.providerType, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, tt.providerType, provider.Name())
			}
		})
	}
}

// TestProviderValidation tests provider validation
func TestProviderValidation(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		config       map[string]interface{}
		expectError  bool
	}{
		{
			name:         "Valid AWS Config",
			providerType: "aws",
			config: map[string]interface{}{
				"region": "us-east-1",
			},
			expectError: false,
		},
		{
			name:         "Valid Azure Config",
			providerType: "azure",
			config: map[string]interface{}{
				"region": "eastus",
			},
			expectError: false,
		},
		{
			name:         "Valid GCP Config",
			providerType: "gcp",
			config: map[string]interface{}{
				"region": "us-central1",
			},
			expectError: false,
		},
		{
			name:         "Valid DigitalOcean Config",
			providerType: "digitalocean",
			config: map[string]interface{}{
				"region": "nyc1",
			},
			expectError: false,
		},
		{
			name:         "Missing Region",
			providerType: "aws",
			config: map[string]interface{}{
				"other": "value",
			},
			expectError: true,
		},
		{
			name:         "Empty Config",
			providerType: "aws",
			config:       map[string]interface{}{},
			expectError:  true,
		},
		{
			name:         "Nil Config",
			providerType: "aws",
			config:       nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.providerType, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)

				// Test validation
				err = provider.ValidateCredentials(context.Background())
				assert.NoError(t, err)
			}
		})
	}
}

// TestProviderCapabilities tests provider capabilities
func TestProviderCapabilities(t *testing.T) {
	provider, err := NewProvider("aws", map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	capabilities := provider.SupportedResourceTypes()
	assert.NotEmpty(t, capabilities)
}

// TestProviderSupportedResourceTypes tests provider supported resource types
func TestProviderSupportedResourceTypes(t *testing.T) {
	provider, err := NewProvider("aws", map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	resourceTypes := provider.SupportedResourceTypes()
	assert.NotEmpty(t, resourceTypes)
}

// TestProviderDiscovery tests resource discovery
func TestProviderDiscovery(t *testing.T) {
	provider, err := NewProvider("aws", map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	resources, err := provider.DiscoverResources(ctx, "us-east-1")

	// Discovery might fail due to missing credentials, but should not panic
	assert.NotNil(t, resources)
	// Error is expected in test environment without real credentials
}

// TestProviderGetResource tests getting a specific resource
func TestProviderGetResource(t *testing.T) {
	provider, err := NewProvider("aws", map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	resource, err := provider.GetResource(ctx, "test-resource-id")

	// Should return error for non-existent resource
	assert.Error(t, err)
	assert.Nil(t, resource)
	assert.True(t, IsNotFoundError(err))
}

// TestProviderConcurrentAccess tests concurrent access to providers
func TestProviderConcurrentAccess(t *testing.T) {
	provider, err := NewProvider("aws", map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Test concurrent discovery calls
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			resources, _ := provider.DiscoverResources(ctx, "us-east-1")
			assert.NotNil(t, resources)
			// Error is expected in test environment
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestProviderErrorHandling tests error handling
func TestProviderErrorHandling(t *testing.T) {
	t.Run("InvalidProviderType", func(t *testing.T) {
		provider, err := NewProvider("invalid", map[string]interface{}{
			"region": "us-east-1",
		})
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		provider, err := NewProvider("aws", map[string]interface{}{
			"invalid": "config",
		})
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("NilConfig", func(t *testing.T) {
		provider, err := NewProvider("aws", nil)
		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}

// TestProviderInterfaceCompliance tests that providers implement the interface correctly
func TestProviderInterfaceCompliance(t *testing.T) {
	provider, err := NewProvider("aws", map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	// Test that provider implements CloudProvider interface
	var _ CloudProvider = provider

	// Test all interface methods exist and return expected types
	name := provider.Name()
	assert.IsType(t, "", name)

	resourceTypes := provider.SupportedResourceTypes()
	assert.IsType(t, []string{}, resourceTypes)

	ctx := context.Background()
	resources, err := provider.DiscoverResources(ctx, "us-east-1")
	assert.IsType(t, []models.Resource{}, resources)

	resource, err := provider.GetResource(ctx, "test-id")
	assert.IsType(t, (*models.Resource)(nil), resource)

	err = provider.ValidateCredentials(ctx)
	assert.IsType(t, error(nil), err)

	regions, err := provider.ListRegions(ctx)
	assert.IsType(t, []string{}, regions)
}

// TestProviderConfigurationPersistence tests that configuration is properly stored
func TestProviderConfigurationPersistence(t *testing.T) {
	config := map[string]interface{}{
		"region":  "us-west-2",
		"profile": "test-profile",
		"timeout": 30,
	}

	provider, err := NewProvider("aws", config)
	require.NoError(t, err)

	// Provider should maintain its configuration
	assert.Equal(t, "aws", provider.Name())

	// Test that provider can be used with the configuration
	ctx := context.Background()
	resources, err := provider.DiscoverResources(ctx, "us-west-2")
	assert.NotNil(t, resources)
}

// TestProviderResourceTypes tests that providers return correct resource types
func TestProviderResourceTypes(t *testing.T) {
	provider, err := NewProvider("aws", map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	resources, err := provider.DiscoverResources(ctx, "us-east-1")

	// Even if discovery fails, we should get an empty slice, not nil
	assert.NotNil(t, resources)
	assert.IsType(t, []models.Resource{}, resources)

	// If resources are returned, they should have the correct provider
	for _, resource := range resources {
		assert.Equal(t, "aws", resource.Provider)
	}
}
