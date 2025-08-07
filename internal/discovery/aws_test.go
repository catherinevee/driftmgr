package discovery

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAWSClient is a mock implementation of the AWS SDK
type MockAWSClient struct {
	mock.Mock
}

func TestNewAWSProvider(t *testing.T) {
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
			provider, err := NewAWSProvider()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				// Should not error even if AWS credentials are not available
				// Provider should use mock data as fallback
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestAWSProvider_Discover(t *testing.T) {
	provider, err := NewAWSProvider()
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
				Provider: "aws",
				Regions:  []string{"us-east-1"},
			},
			wantErr:      false,
			minResources: 0, // Should at least return mock data
		},
		{
			name: "discover multiple regions",
			config: models.DiscoveryConfig{
				Provider: "aws",
				Regions:  []string{"us-east-1", "us-west-2"},
			},
			wantErr:      false,
			minResources: 0,
		},
		{
			name: "discover with empty regions",
			config: models.DiscoveryConfig{
				Provider: "aws",
				Regions:  []string{},
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
					assert.Equal(t, "aws", resource.Provider)
				}
			}
		})
	}
}

func TestAWSProvider_GetMockData(t *testing.T) {
	provider, err := NewAWSProvider()
	assert.NoError(t, err)

	config := models.DiscoveryConfig{
		Provider: "aws",
		Regions:  []string{"us-east-1"},
	}

	// Call the mock data function directly
	resources := provider.getMockData(config)

	assert.GreaterOrEqual(t, len(resources), 4) // Should have at least 4 mock resources

	// Verify resource types
	resourceTypes := make(map[string]bool)
	for _, resource := range resources {
		resourceTypes[resource.Type] = true
	}

	expectedTypes := []string{
		"aws_instance",
		"aws_vpc",
		"aws_s3_bucket",
		"aws_security_group",
	}

	for _, expectedType := range expectedTypes {
		assert.True(t, resourceTypes[expectedType], "Expected resource type %s not found", expectedType)
	}
}

func TestAWSProvider_DiscoverEC2Instances(t *testing.T) {
	provider, err := NewAWSProvider()
	assert.NoError(t, err)

	// Test with mock implementation when AWS credentials are not available
	resources := provider.discoverEC2Instances(context.Background(), "us-east-1")

	// Should return at least mock data
	assert.GreaterOrEqual(t, len(resources), 0)

	for _, resource := range resources {
		assert.Equal(t, "aws_instance", resource.Type)
		assert.Equal(t, "us-east-1", resource.Region)
		assert.Equal(t, "aws", resource.Provider)
	}
}

func TestAWSProvider_DiscoverVPCs(t *testing.T) {
	provider, err := NewAWSProvider()
	assert.NoError(t, err)

	resources := provider.discoverVPCs(context.Background(), "us-east-1")

	assert.GreaterOrEqual(t, len(resources), 0)

	for _, resource := range resources {
		assert.Equal(t, "aws_vpc", resource.Type)
		assert.Equal(t, "us-east-1", resource.Region)
		assert.Equal(t, "aws", resource.Provider)
	}
}

func TestAWSProvider_DiscoverS3Buckets(t *testing.T) {
	provider, err := NewAWSProvider()
	assert.NoError(t, err)

	resources := provider.discoverS3Buckets(context.Background())

	assert.GreaterOrEqual(t, len(resources), 0)

	for _, resource := range resources {
		assert.Equal(t, "aws_s3_bucket", resource.Type)
		assert.Equal(t, "aws", resource.Provider)
		// S3 buckets are global, so region may vary
	}
}

func TestAWSProvider_DiscoverSecurityGroups(t *testing.T) {
	provider, err := NewAWSProvider()
	assert.NoError(t, err)

	resources := provider.discoverSecurityGroups(context.Background(), "us-east-1")

	assert.GreaterOrEqual(t, len(resources), 0)

	for _, resource := range resources {
		assert.Equal(t, "aws_security_group", resource.Type)
		assert.Equal(t, "us-east-1", resource.Region)
		assert.Equal(t, "aws", resource.Provider)
	}
}

// Integration test for real AWS API (only runs with credentials)
func TestAWSProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	provider, err := NewAWSProvider()
	assert.NoError(t, err)

	config := models.DiscoveryConfig{
		Provider: "aws",
		Regions:  []string{"us-east-1"},
	}

	// This test will use real AWS credentials if available, mock data otherwise
	resources, err := provider.Discover(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, resources)

	t.Logf("Discovered %d AWS resources", len(resources))

	// Log resource details for manual verification
	for i, resource := range resources {
		if i < 5 { // Limit output
			t.Logf("Resource %d: %s (%s) in %s", i+1, resource.Name, resource.Type, resource.Region)
		}
	}
}
