package detector

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCloudProvider implements providers.CloudProvider for testing
type MockCloudProvider struct {
	name          string
	resources     []models.Resource
	discoverError error
}

func (m *MockCloudProvider) Name() string {
	return m.name
}

func (m *MockCloudProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	if m.discoverError != nil {
		return nil, m.discoverError
	}
	return m.resources, nil
}

func (m *MockCloudProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	for _, resource := range m.resources {
		if resource.ID == resourceID {
			return &resource, nil
		}
	}
	return nil, providers.NewNotFoundError(m.name, resourceID, "test-region")
}

func (m *MockCloudProvider) ValidateCredentials(ctx context.Context) error {
	return nil
}

func (m *MockCloudProvider) ListRegions(ctx context.Context) ([]string, error) {
	return []string{"us-east-1", "us-west-2"}, nil
}

func (m *MockCloudProvider) SupportedResourceTypes() []string {
	return []string{"aws_instance", "aws_s3_bucket"}
}

// TestDetectDrift tests the core drift detection functionality
func TestDetectDrift(t *testing.T) {
	// Create mock provider with test resources
	mockProvider := &MockCloudProvider{
		name: "aws",
		resources: []models.Resource{
			{
				ID:        "i-1234567890abcdef0",
				Name:      "test-instance",
				Type:      "aws_instance",
				Provider:  "aws",
				Region:    "us-east-1",
				AccountID: "123456789012",
				Tags:      map[string]string{"Environment": "test"},
				Properties: map[string]interface{}{
					"instance_type": "t3.micro",
					"ami":           "ami-0c02fb55956c7d316",
				},
				CreatedAt: time.Now().Add(-24 * time.Hour),
				Updated:   time.Now().Add(-1 * time.Hour),
			},
		},
	}

	providers := map[string]providers.CloudProvider{
		"aws": mockProvider,
	}

	detector := NewDriftDetector(providers)

	// Create test state
	testState := &state.TerraformState{
		Version: 4,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "test_instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id":            "i-1234567890abcdef0",
							"instance_type": "t3.small", // Different from discovered
							"ami":           "ami-0c02fb55956c7d316",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	report, err := detector.DetectDrift(ctx, testState)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 1, report.TotalResources)
	assert.Greater(t, len(report.DriftResults), 0)
	assert.NotNil(t, report.Summary)
	assert.Contains(t, report.Summary.ByProvider, "aws")
}

// TestDetectDriftWithTimeout tests drift detection with timeout
func TestDetectDriftWithTimeout(t *testing.T) {
	// Create provider that takes too long
	mockProvider := &MockCloudProvider{
		name:          "aws",
		discoverError: context.DeadlineExceeded,
	}

	providers := map[string]providers.CloudProvider{
		"aws": mockProvider,
	}

	detector := NewDriftDetector(providers)
	detector.config.Timeout = 1 * time.Millisecond // Very short timeout

	testState := &state.TerraformState{
		Version: 4,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "test_instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	report, err := detector.DetectDrift(ctx, testState)

	// Should handle timeout gracefully
	assert.NoError(t, err)
	assert.NotNil(t, report)
}

// TestDetectDriftEmptyState tests drift detection with empty state
func TestDetectDriftEmptyState(t *testing.T) {
	mockProvider := &MockCloudProvider{
		name:      "aws",
		resources: []models.Resource{},
	}

	providers := map[string]providers.CloudProvider{
		"aws": mockProvider,
	}

	detector := NewDriftDetector(providers)

	// Empty state
	testState := &state.TerraformState{
		Version:   4,
		Resources: []state.Resource{},
	}

	ctx := context.Background()
	report, err := detector.DetectDrift(ctx, testState)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 0, report.TotalResources)
	assert.Equal(t, 0, len(report.DriftResults))
}

// TestDetectDriftProviderError tests drift detection with provider errors
func TestDetectDriftProviderError(t *testing.T) {
	mockProvider := &MockCloudProvider{
		name:          "aws",
		discoverError: assert.AnError,
	}

	providers := map[string]providers.CloudProvider{
		"aws": mockProvider,
	}

	detector := NewDriftDetector(providers)

	testState := &state.TerraformState{
		Version: 4,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "test_instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	report, err := detector.DetectDrift(ctx, testState)

	// Should handle provider errors gracefully
	assert.NoError(t, err)
	assert.NotNil(t, report)
}

// TestDetectDriftMultipleProviders tests drift detection with multiple providers
func TestDetectDriftMultipleProviders(t *testing.T) {
	awsProvider := &MockCloudProvider{
		name: "aws",
		resources: []models.Resource{
			{
				ID:       "i-1234567890abcdef0",
				Name:     "aws-instance",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
			},
		},
	}

	azureProvider := &MockCloudProvider{
		name: "azure",
		resources: []models.Resource{
			{
				ID:       "/subscriptions/123/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm",
				Name:     "azure-vm",
				Type:     "azurerm_virtual_machine",
				Provider: "azure",
				Region:   "eastus",
			},
		},
	}

	providers := map[string]providers.CloudProvider{
		"aws":   awsProvider,
		"azure": azureProvider,
	}

	detector := NewDriftDetector(providers)

	testState := &state.TerraformState{
		Version: 4,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "aws_instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
			{
				Type: "azurerm_virtual_machine",
				Name: "azure_vm",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "/subscriptions/123/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	report, err := detector.DetectDrift(ctx, testState)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 2, report.TotalResources)
	assert.NotNil(t, report.Summary)
	assert.Contains(t, report.Summary.ByProvider, "aws")
	assert.Contains(t, report.Summary.ByProvider, "azure")
}

// TestDriftReportGeneration tests drift report generation
func TestDriftReportGeneration(t *testing.T) {
	mockProvider := &MockCloudProvider{
		name: "aws",
		resources: []models.Resource{
			{
				ID:       "i-1234567890abcdef0",
				Name:     "test-instance",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
			},
		},
	}

	providers := map[string]providers.CloudProvider{
		"aws": mockProvider,
	}

	detector := NewDriftDetector(providers)

	testState := &state.TerraformState{
		Version: 4,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "test_instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890abcdef0",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	report, err := detector.DetectDrift(ctx, testState)

	require.NoError(t, err)
	assert.NotNil(t, report)

	// Test report structure
	assert.NotZero(t, report.Timestamp)
	assert.GreaterOrEqual(t, report.TotalResources, 0)
	assert.NotNil(t, report.DriftResults)
	assert.NotNil(t, report.Summary)
	assert.NotNil(t, report.Summary.ByProvider)
	assert.NotNil(t, report.Summary.ByType)
	assert.NotNil(t, report.Summary.BySeverity)
}

// TestDetectorConfig tests detector configuration
func TestDetectorConfig(t *testing.T) {
	config := &DetectorConfig{
		MaxWorkers:        5,
		Timeout:           2 * time.Minute,
		CheckUnmanaged:    false,
		DeepComparison:    false,
		ParallelDiscovery: false,
		RetryAttempts:     1,
		RetryDelay:        1 * time.Second,
	}

	detector := &DriftDetector{
		config: config,
	}

	assert.Equal(t, 5, detector.config.MaxWorkers)
	assert.Equal(t, 2*time.Minute, detector.config.Timeout)
	assert.False(t, detector.config.CheckUnmanaged)
	assert.False(t, detector.config.DeepComparison)
	assert.False(t, detector.config.ParallelDiscovery)
	assert.Equal(t, 1, detector.config.RetryAttempts)
	assert.Equal(t, 1*time.Second, detector.config.RetryDelay)
}
