package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestMockProvider_Name(t *testing.T) {
	provider := NewMockProvider("test-provider")
	assert.Equal(t, "test-provider", provider.Name())
}

func TestMockProvider_DiscoverResources(t *testing.T) {
	ctx := context.Background()

	t.Run("Discover all resources", func(t *testing.T) {
		provider := NewMockProvider("test")
		resources, err := provider.DiscoverResources(ctx, "")
		assert.NoError(t, err)
		assert.Len(t, resources, 3)
	})

	t.Run("Discover by region", func(t *testing.T) {
		provider := NewMockProvider("test")
		resources, err := provider.DiscoverResources(ctx, "us-east-1")
		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		for _, r := range resources {
			assert.Equal(t, "us-east-1", r.Region)
		}
	})

	t.Run("Discover with error", func(t *testing.T) {
		provider := NewMockProvider("test")
		expectedErr := errors.New("discovery failed")
		provider.SetDiscoverError(expectedErr)

		resources, err := provider.DiscoverResources(ctx, "us-east-1")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resources)
	})

	t.Run("Return empty resources", func(t *testing.T) {
		provider := NewMockProvider("test")
		provider.SetReturnEmpty(true)

		resources, err := provider.DiscoverResources(ctx, "us-east-1")
		assert.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("Call count tracking", func(t *testing.T) {
		provider := NewMockProvider("test")
		assert.Equal(t, 0, provider.GetDiscoverCallCount())

		provider.DiscoverResources(ctx, "us-east-1")
		assert.Equal(t, 1, provider.GetDiscoverCallCount())

		provider.DiscoverResources(ctx, "us-west-2")
		assert.Equal(t, 2, provider.GetDiscoverCallCount())
	})
}

func TestMockProvider_GetResource(t *testing.T) {
	ctx := context.Background()

	t.Run("Get existing resource", func(t *testing.T) {
		provider := NewMockProvider("test")
		resource, err := provider.GetResource(ctx, "mock-resource-1")
		assert.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "mock-resource-1", resource.ID)
		assert.Equal(t, "Mock Resource 1", resource.Name)
	})

	t.Run("Get non-existent resource", func(t *testing.T) {
		provider := NewMockProvider("test")
		resource, err := provider.GetResource(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, resource)

		var notFoundErr *providers.NotFoundError
		assert.True(t, errors.As(err, &notFoundErr))
		assert.Equal(t, "test", notFoundErr.Provider)
		assert.Equal(t, "non-existent", notFoundErr.ResourceID)
	})

	t.Run("Get resource with error", func(t *testing.T) {
		provider := NewMockProvider("test")
		expectedErr := errors.New("get resource failed")
		provider.SetGetResourceError(expectedErr)

		resource, err := provider.GetResource(ctx, "mock-resource-1")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resource)
	})

	t.Run("Get added resource", func(t *testing.T) {
		provider := NewMockProvider("test")
		newResource := models.Resource{
			ID:       "custom-resource",
			Name:     "Custom Resource",
			Type:     "mock.custom",
			Provider: "test",
			Region:   "eu-west-1",
			Status:   "active",
		}
		provider.AddResource(newResource)

		resource, err := provider.GetResource(ctx, "custom-resource")
		assert.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "custom-resource", resource.ID)
		assert.Equal(t, "Custom Resource", resource.Name)
	})
}

func TestMockProvider_ValidateCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid credentials", func(t *testing.T) {
		provider := NewMockProvider("test")
		err := provider.ValidateCredentials(ctx)
		assert.NoError(t, err)
	})

	t.Run("Invalid credentials", func(t *testing.T) {
		provider := NewMockProvider("test")
		expectedErr := errors.New("invalid credentials")
		provider.SetValidateError(expectedErr)

		err := provider.ValidateCredentials(ctx)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("Call count tracking", func(t *testing.T) {
		provider := NewMockProvider("test")
		assert.Equal(t, 0, provider.GetValidateCallCount())

		provider.ValidateCredentials(ctx)
		assert.Equal(t, 1, provider.GetValidateCallCount())

		provider.ValidateCredentials(ctx)
		assert.Equal(t, 2, provider.GetValidateCallCount())
	})
}

func TestMockProvider_ListRegions(t *testing.T) {
	ctx := context.Background()

	t.Run("List default regions", func(t *testing.T) {
		provider := NewMockProvider("test")
		regions, err := provider.ListRegions(ctx)
		assert.NoError(t, err)
		assert.Len(t, regions, 3)
		assert.Contains(t, regions, "us-east-1")
		assert.Contains(t, regions, "us-west-2")
		assert.Contains(t, regions, "eu-west-1")
	})

	t.Run("List custom regions", func(t *testing.T) {
		provider := NewMockProvider("test")
		customRegions := []string{"ap-south-1", "ap-southeast-1", "eu-central-1"}
		provider.SetRegions(customRegions)

		regions, err := provider.ListRegions(ctx)
		assert.NoError(t, err)
		assert.Equal(t, customRegions, regions)
	})

	t.Run("List regions with error", func(t *testing.T) {
		provider := NewMockProvider("test")
		expectedErr := errors.New("list regions failed")
		provider.SetListRegionsError(expectedErr)

		regions, err := provider.ListRegions(ctx)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, regions)
	})
}

func TestMockProvider_SupportedResourceTypes(t *testing.T) {
	t.Run("Default supported types", func(t *testing.T) {
		provider := NewMockProvider("test")
		types := provider.SupportedResourceTypes()
		assert.Len(t, types, 4)
		assert.Contains(t, types, "mock.instance")
		assert.Contains(t, types, "mock.database")
		assert.Contains(t, types, "mock.storage")
		assert.Contains(t, types, "mock.network")
	})

	t.Run("Custom supported types", func(t *testing.T) {
		provider := NewMockProvider("test")
		customTypes := []string{"custom.type1", "custom.type2"}
		provider.SetSupportedTypes(customTypes)

		types := provider.SupportedResourceTypes()
		assert.Equal(t, customTypes, types)
	})
}

func TestMockProvider_SetResources(t *testing.T) {
	ctx := context.Background()
	provider := NewMockProvider("test")

	customResources := []models.Resource{
		{
			ID:       "custom-1",
			Name:     "Custom 1",
			Type:     "custom.type",
			Provider: "test",
			Region:   "us-west-1",
			Status:   "active",
		},
		{
			ID:       "custom-2",
			Name:     "Custom 2",
			Type:     "custom.type",
			Provider: "test",
			Region:   "us-west-1",
			Status:   "active",
		},
	}

	provider.SetResources(customResources)

	resources, err := provider.DiscoverResources(ctx, "us-west-1")
	assert.NoError(t, err)
	assert.Len(t, resources, 2)
	assert.Equal(t, "custom-1", resources[0].ID)
	assert.Equal(t, "custom-2", resources[1].ID)
}

func TestMockProvider_ResetCallCounts(t *testing.T) {
	ctx := context.Background()
	provider := NewMockProvider("test")

	// Make some calls
	provider.DiscoverResources(ctx, "us-east-1")
	provider.ValidateCredentials(ctx)
	provider.GetResource(ctx, "mock-resource-1")
	provider.ListRegions(ctx)

	// Verify counts
	assert.Equal(t, 1, provider.GetDiscoverCallCount())
	assert.Equal(t, 1, provider.GetValidateCallCount())

	// Reset counts
	provider.ResetCallCounts()

	// Verify reset
	assert.Equal(t, 0, provider.GetDiscoverCallCount())
	assert.Equal(t, 0, provider.GetValidateCallCount())
}

func TestMockProviderWithDrift(t *testing.T) {
	ctx := context.Background()
	provider := MockProviderWithDrift("test-drift")

	resources, err := provider.DiscoverResources(ctx, "us-east-1")
	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check drifted resource
	driftResource := resources[0]
	assert.Equal(t, "drift-resource-1", driftResource.ID)
	assert.Equal(t, "Resource with Drift", driftResource.Name)
	assert.Equal(t, 4, driftResource.Attributes["cpu"])
	assert.Equal(t, 8192, driftResource.Attributes["memory"])

	// Check deleted resource
	deletedResource := resources[1]
	assert.Equal(t, "drift-resource-2", deletedResource.ID)
	assert.Equal(t, "deleted", deletedResource.Status)
	assert.Equal(t, "14.0", deletedResource.Attributes["version"])
}

func TestMockProviderFactory(t *testing.T) {
	factory := NewMockProviderFactory()

	t.Run("Create new provider", func(t *testing.T) {
		config := providers.ProviderConfig{
			Name: "test-provider",
			Credentials: map[string]string{
				"api_key": "test-key",
			},
			Region: "us-east-1",
		}

		provider, err := factory.CreateProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test-provider", provider.Name())
	})

	t.Run("Get existing provider", func(t *testing.T) {
		config := providers.ProviderConfig{
			Name: "existing-provider",
		}

		// Create first time
		provider1, err := factory.CreateProvider(config)
		assert.NoError(t, err)

		// Get second time - should return same instance
		provider2, err := factory.CreateProvider(config)
		assert.NoError(t, err)
		assert.Equal(t, provider1, provider2)
	})

	t.Run("Register and get provider", func(t *testing.T) {
		mockProvider := NewMockProvider("registered")
		factory.RegisterProvider("registered", mockProvider)

		provider, err := factory.GetProvider("registered")
		assert.NoError(t, err)
		assert.Equal(t, mockProvider, provider)
	})

	t.Run("Get non-existent provider", func(t *testing.T) {
		provider, err := factory.GetProvider("non-existent")
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "provider non-existent not found")
	})
}

func TestMockProvider_ConcurrentAccess(t *testing.T) {
	provider := NewMockProvider("test")
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 4)

	go func() {
		for i := 0; i < 10; i++ {
			provider.DiscoverResources(ctx, "us-east-1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			provider.GetResource(ctx, "mock-resource-1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			provider.ValidateCredentials(ctx)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			provider.ListRegions(ctx)
		}
		done <- true
	}()

	// Wait for all goroutines to finish
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify no race conditions occurred
	assert.True(t, provider.GetDiscoverCallCount() > 0)
	assert.True(t, provider.GetValidateCallCount() > 0)
}
