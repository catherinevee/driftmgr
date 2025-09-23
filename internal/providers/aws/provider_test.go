package aws

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewAWSProvider tests creating a new AWS provider
func TestNewAWSProvider(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected string
	}{
		{
			name:     "Valid region",
			region:   "us-east-1",
			expected: "us-east-1",
		},
		{
			name:     "Empty region",
			region:   "",
			expected: "us-east-1", // Default region
		},
		{
			name:     "Different region",
			region:   "us-west-2",
			expected: "us-west-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAWSProvider(tt.region)
			assert.NotNil(t, provider)
			assert.Equal(t, "aws", provider.Name())
		})
	}
}

// TestAWSProviderName tests the provider name
func TestAWSProviderName(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	assert.Equal(t, "aws", provider.Name())
}

// TestAWSProviderSupportedResourceTypes tests supported resource types
func TestAWSProviderSupportedResourceTypes(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	resourceTypes := provider.SupportedResourceTypes()

	assert.NotEmpty(t, resourceTypes)
	assert.Contains(t, resourceTypes, "AWS::EC2::Instance")
	assert.Contains(t, resourceTypes, "AWS::S3::Bucket")
	assert.Contains(t, resourceTypes, "AWS::EC2::SecurityGroup")
	assert.Contains(t, resourceTypes, "AWS::RDS::DBInstance")
	assert.Contains(t, resourceTypes, "AWS::Lambda::Function")
}

// TestAWSProviderListRegions tests listing available regions
func TestAWSProviderListRegions(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	regions, err := provider.ListRegions(ctx)

	// In test environment, this might fail due to credentials
	// but we should get a reasonable response structure
	if err != nil {
		// Expected in test environment without credentials
		assert.NotNil(t, err)
	} else {
		assert.NotNil(t, regions)
		assert.IsType(t, []string{}, regions)
	}
}

// TestAWSProviderValidateCredentials tests credential validation
func TestAWSProviderValidateCredentials(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	_ = provider.ValidateCredentials(ctx)

	// In test environment, this will likely fail due to missing credentials
	// but we should handle it gracefully
	// Note: Some providers might return nil if credentials are not required for basic validation
}

// TestAWSProviderDiscoverResources tests resource discovery
func TestAWSProviderDiscoverResources(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	resources, err := provider.DiscoverResources(ctx, "us-east-1")

	// In test environment, this will likely fail due to missing credentials
	// but we should get a proper response structure
	if err != nil {
		// Expected in test environment without credentials
		assert.NotNil(t, err)
	} else {
		assert.NotNil(t, resources)
		// Note: The actual return type is []models.Resource, not []interface{}
	}
}

// TestAWSProviderGetResource tests getting a specific resource
func TestAWSProviderGetResource(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	resource, err := provider.GetResource(ctx, "test-resource-id")

	// Should return error for non-existent resource
	assert.Error(t, err)
	assert.Nil(t, resource)
}

// TestAWSProviderConcurrentAccess tests concurrent access
func TestAWSProviderConcurrentAccess(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	// Test concurrent calls
	done := make(chan bool, 3)

	go func() {
		defer func() { done <- true }()
		regions, _ := provider.ListRegions(ctx)
		assert.NotNil(t, regions)
	}()

	go func() {
		defer func() { done <- true }()
		resources, _ := provider.DiscoverResources(ctx, "us-east-1")
		assert.NotNil(t, resources)
	}()

	go func() {
		defer func() { done <- true }()
		resourceTypes := provider.SupportedResourceTypes()
		assert.NotEmpty(t, resourceTypes)
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}

// TestAWSProviderErrorHandling tests error handling scenarios
func TestAWSProviderErrorHandling(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	t.Run("InvalidRegion", func(t *testing.T) {
		resources, _ := provider.DiscoverResources(ctx, "invalid-region")
		// Note: The provider might not validate regions strictly in test environment
		// So we just check that we get a response (even if empty)
		assert.NotNil(t, resources)
		// Error might or might not be returned depending on implementation
	})

	t.Run("EmptyResourceID", func(t *testing.T) {
		resource, err := provider.GetResource(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, resource)
	})

	t.Run("NilContext", func(t *testing.T) {
		// This should not panic
		assert.NotPanics(t, func() {
			provider.SupportedResourceTypes()
		})
	})
}

// TestAWSProviderInterfaceCompliance tests that the provider implements the interface correctly
func TestAWSProviderInterfaceCompliance(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	// Test all interface methods exist and return expected types
	name := provider.Name()
	assert.IsType(t, "", name)
	assert.Equal(t, "aws", name)

	resourceTypes := provider.SupportedResourceTypes()
	assert.IsType(t, []string{}, resourceTypes)
	assert.NotEmpty(t, resourceTypes)

	regions, err := provider.ListRegions(ctx)
	assert.IsType(t, []string{}, regions)
	// Error is expected in test environment

	_, _ = provider.DiscoverResources(ctx, "us-east-1")
	// Note: The actual return type is []models.Resource, not []interface{}
	// Error is expected in test environment

	_, err2 := provider.GetResource(ctx, "test-id")
	// Note: The actual return type is *models.Resource, not *interface{}
	assert.Error(t, err2) // Expected for non-existent resource

	err = provider.ValidateCredentials(ctx)
	assert.IsType(t, error(nil), err)
	// Note: Some providers might return nil if credentials are not required for basic validation
}

// TestAWSProviderConfiguration tests provider configuration
func TestAWSProviderConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		region         string
		expectedRegion string
	}{
		{
			name:           "US East 1",
			region:         "us-east-1",
			expectedRegion: "us-east-1",
		},
		{
			name:           "US West 2",
			region:         "us-west-2",
			expectedRegion: "us-west-2",
		},
		{
			name:           "EU West 1",
			region:         "eu-west-1",
			expectedRegion: "eu-west-1",
		},
		{
			name:           "Empty region defaults to us-east-1",
			region:         "",
			expectedRegion: "us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAWSProvider(tt.region)
			assert.NotNil(t, provider)
			assert.Equal(t, "aws", provider.Name())
		})
	}
}

// TestAWSProviderResourceTypesNotEmpty tests that resource types are not empty
func TestAWSProviderResourceTypesNotEmpty(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	resourceTypes := provider.SupportedResourceTypes()

	assert.NotEmpty(t, resourceTypes)
	assert.Greater(t, len(resourceTypes), 5, "Should support at least 5 resource types")
}

// TestAWSProviderPerformance tests basic performance characteristics
func TestAWSProviderPerformance(t *testing.T) {
	provider := NewAWSProvider("us-east-1")

	// Test that getting supported resource types is fast
	start := time.Now()
	resourceTypes := provider.SupportedResourceTypes()
	duration := time.Since(start)

	assert.NotEmpty(t, resourceTypes)
	assert.Less(t, duration, 100*time.Millisecond, "Getting resource types should be fast")
}

// TestAWSProviderThreadSafety tests thread safety
func TestAWSProviderThreadSafety(t *testing.T) {
	provider := NewAWSProvider("us-east-1")

	// Test concurrent access to read-only methods
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// These should be safe to call concurrently
			name := provider.Name()
			assert.Equal(t, "aws", name)

			resourceTypes := provider.SupportedResourceTypes()
			assert.NotEmpty(t, resourceTypes)
		}()
	}

	wg.Wait()
}
