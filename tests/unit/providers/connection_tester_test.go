package providers_test

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCloudProvider is a mock implementation of CloudProvider
type MockCloudProvider struct {
	mock.Mock
}

func (m *MockCloudProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCloudProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	args := m.Called(ctx, region)
	return args.Get(0).([]models.Resource), args.Error(1)
}

func (m *MockCloudProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(*models.Resource), args.Error(1)
}

func (m *MockCloudProvider) ValidateCredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCloudProvider) ListRegions(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCloudProvider) SupportedResourceTypes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestConnectionTester_TestConnection_Success(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("test-provider")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"us-east-1", "us-west-2"}, nil)
	mockProvider.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{
		{ID: "resource-1", Type: "test_resource"},
	}, nil)
	mockProvider.On("SupportedResourceTypes").Return([]string{"test_resource"})

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test connection
	result, err := tester.TestConnection(context.Background(), mockProvider, "us-east-1")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Equal(t, "us-east-1", result.Region)
	assert.Greater(t, result.Latency, time.Duration(0))
	assert.Contains(t, result.Details, "available_regions")
	assert.Contains(t, result.Details, "discovered_resources")

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestConnection_CredentialValidationFailed(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("test-provider")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(assert.AnError)

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test connection
	result, err := tester.TestConnection(context.Background(), mockProvider, "us-east-1")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Equal(t, "us-east-1", result.Region)
	assert.Contains(t, result.Error, "credential validation failed")

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestConnection_RegionNotAvailable(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("test-provider")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"us-west-2"}, nil)

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test connection
	result, err := tester.TestConnection(context.Background(), mockProvider, "us-east-1")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Equal(t, "us-east-1", result.Region)
	assert.Contains(t, result.Error, "region us-east-1 not available")

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestServiceConnection_Success(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("aws")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"us-east-1"}, nil)
	mockProvider.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{
		{ID: "bucket-1", Type: "aws_s3_bucket"},
		{ID: "instance-1", Type: "aws_ec2_instance"},
	}, nil)
	mockProvider.On("SupportedResourceTypes").Return([]string{"aws_s3_bucket", "aws_ec2_instance"})

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test service connection
	result, err := tester.TestServiceConnection(context.Background(), mockProvider, "us-east-1", "aws_s3_bucket")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "aws", result.Provider)
	assert.Equal(t, "us-east-1", result.Region)
	assert.Equal(t, "aws_s3_bucket", result.Service)
	assert.Contains(t, result.Details, "s3_buckets_found")

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestAllRegions_Success(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("test-provider")
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"us-east-1", "us-west-2"}, nil)
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("DiscoverResources", mock.Anything, mock.Anything).Return([]models.Resource{}, nil)
	mockProvider.On("SupportedResourceTypes").Return([]string{"test_resource"})

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test all regions
	results, err := tester.TestAllRegions(context.Background(), mockProvider)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	for _, result := range results {
		assert.True(t, result.Success)
		assert.Equal(t, "test-provider", result.Provider)
		assert.Contains(t, []string{"us-east-1", "us-west-2"}, result.Region)
	}

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestAllServices_Success(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("aws")
	mockProvider.On("SupportedResourceTypes").Return([]string{"aws_s3_bucket", "aws_ec2_instance"})
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"us-east-1"}, nil)
	mockProvider.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{
		{ID: "bucket-1", Type: "aws_s3_bucket"},
		{ID: "instance-1", Type: "aws_ec2_instance"},
	}, nil)

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test all services
	results, err := tester.TestAllServices(context.Background(), mockProvider, "us-east-1")

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	for _, result := range results {
		assert.True(t, result.Success)
		assert.Equal(t, "aws", result.Provider)
		assert.Equal(t, "us-east-1", result.Region)
		assert.Contains(t, []string{"aws_s3_bucket", "aws_ec2_instance"}, result.Service)
	}

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestConnection_Timeout(t *testing.T) {
	// Create mock provider that takes too long
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("test-provider")
	mockProvider.On("ValidateCredentials", mock.Anything).Run(func(args mock.Arguments) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
	}).Return(nil)

	// Create connection tester with short timeout
	tester := providers.NewConnectionTester(50 * time.Millisecond)

	// Test connection
	result, err := tester.TestConnection(context.Background(), mockProvider, "us-east-1")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "context deadline exceeded")

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestAWSService_S3(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("aws")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"us-east-1"}, nil)
	mockProvider.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{
		{ID: "bucket-1", Type: "aws_s3_bucket"},
		{ID: "bucket-2", Type: "aws_s3_bucket"},
	}, nil)
	mockProvider.On("SupportedResourceTypes").Return([]string{"aws_s3_bucket"})

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test S3 service
	result, err := tester.TestServiceConnection(context.Background(), mockProvider, "us-east-1", "aws_s3_bucket")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "aws", result.Provider)
	assert.Equal(t, "us-east-1", result.Region)
	assert.Equal(t, "aws_s3_bucket", result.Service)
	assert.Equal(t, 2, result.Details["s3_buckets_found"])

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestAzureService_StorageAccount(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("azure")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"eastus"}, nil)
	mockProvider.On("DiscoverResources", mock.Anything, "eastus").Return([]models.Resource{
		{ID: "storage-1", Type: "azurerm_storage_account"},
	}, nil)
	mockProvider.On("SupportedResourceTypes").Return([]string{"azurerm_storage_account"})

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test storage account service
	result, err := tester.TestServiceConnection(context.Background(), mockProvider, "eastus", "azurerm_storage_account")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "azure", result.Provider)
	assert.Equal(t, "eastus", result.Region)
	assert.Equal(t, "azurerm_storage_account", result.Service)
	assert.Equal(t, 1, result.Details["storage_accounts_found"])

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestGCPService_StorageBucket(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("gcp")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"us-central1"}, nil)
	mockProvider.On("DiscoverResources", mock.Anything, "us-central1").Return([]models.Resource{
		{ID: "bucket-1", Type: "google_storage_bucket"},
		{ID: "bucket-2", Type: "google_storage_bucket"},
		{ID: "bucket-3", Type: "google_storage_bucket"},
	}, nil)
	mockProvider.On("SupportedResourceTypes").Return([]string{"google_storage_bucket"})

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test storage bucket service
	result, err := tester.TestServiceConnection(context.Background(), mockProvider, "us-central1", "google_storage_bucket")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "gcp", result.Provider)
	assert.Equal(t, "us-central1", result.Region)
	assert.Equal(t, "google_storage_bucket", result.Service)
	assert.Equal(t, 3, result.Details["storage_buckets_found"])

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}

func TestConnectionTester_TestDigitalOceanService_Droplet(t *testing.T) {
	// Create mock provider
	mockProvider := new(MockCloudProvider)
	mockProvider.On("Name").Return("digitalocean")
	mockProvider.On("ValidateCredentials", mock.Anything).Return(nil)
	mockProvider.On("ListRegions", mock.Anything).Return([]string{"nyc1"}, nil)
	mockProvider.On("DiscoverResources", mock.Anything, "nyc1").Return([]models.Resource{
		{ID: "droplet-1", Type: "digitalocean_droplet"},
		{ID: "droplet-2", Type: "digitalocean_droplet"},
	}, nil)
	mockProvider.On("SupportedResourceTypes").Return([]string{"digitalocean_droplet"})

	// Create connection tester
	tester := providers.NewConnectionTester(30 * time.Second)

	// Test droplet service
	result, err := tester.TestServiceConnection(context.Background(), mockProvider, "nyc1", "digitalocean_droplet")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "digitalocean", result.Provider)
	assert.Equal(t, "nyc1", result.Region)
	assert.Equal(t, "digitalocean_droplet", result.Service)
	assert.Equal(t, 2, result.Details["droplets_found"])

	// Verify mock calls
	mockProvider.AssertExpectations(t)
}
