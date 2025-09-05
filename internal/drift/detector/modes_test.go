package detector

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/catherinevee/driftmgr/internal/drift/comparator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCloudProvider for testing
type MockCloudProvider struct {
	mock.Mock
}

func (m *MockCloudProvider) GetResourceByID(ctx context.Context, resourceType, resourceID string) (*models.Resource, error) {
	args := m.Called(ctx, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Resource), args.Error(1)
}

func (m *MockCloudProvider) ListResources(ctx context.Context, resourceType string, filters map[string]string) ([]*models.Resource, error) {
	args := m.Called(ctx, resourceType, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Resource), args.Error(1)
}

func (m *MockCloudProvider) GetProviderType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCloudProvider) ValidateCredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCloudProvider) GetRegion() string {
	args := m.Called()
	return args.String(0)
}

func TestQuickModeDetection(t *testing.T) {
	ctx := context.Background()
	detector := NewModeDetector(QuickMode, nil)

	stateResource := &models.Resource{
		ID:   "test-resource-1",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami": "ami-12345",
		},
	}

	cloudResource := &models.Resource{
		ID:   "test-resource-1",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.small", // Different but should be ignored in quick mode
			"ami": "ami-12345",
		},
	}

	// Test quick mode - should only check existence
	hasDrift := detector.DetectDrift(ctx, stateResource, cloudResource)
	assert.False(t, hasDrift, "Quick mode should not detect attribute drift")

	// Test with missing resource
	hasDrift = detector.DetectDrift(ctx, stateResource, nil)
	assert.True(t, hasDrift, "Quick mode should detect missing resources")
}

func TestDeepModeDetection(t *testing.T) {
	ctx := context.Background()
	detector := NewModeDetector(DeepMode, nil)

	stateResource := &models.Resource{
		ID:   "test-resource-1",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami": "ami-12345",
			"tags": map[string]interface{}{
				"Name": "test",
				"Environment": "dev",
			},
		},
	}

	cloudResource := &models.Resource{
		ID:   "test-resource-1",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.small", // Different
			"ami": "ami-12345",
			"tags": map[string]interface{}{
				"Name": "test",
				"Environment": "dev",
			},
		},
	}

	// Test deep mode - should detect attribute differences
	hasDrift := detector.DetectDrift(ctx, stateResource, cloudResource)
	assert.True(t, hasDrift, "Deep mode should detect attribute drift")
}

func TestSmartModeDetection(t *testing.T) {
	ctx := context.Background()
	
	// Create criticality config
	criticalityConfig := &ResourceCriticalityConfig{
		ResourceTypes: map[string]ResourceCriticality{
			"aws_instance": {
				Level: CriticalityHigh,
				CriticalAttributes: []string{"instance_type", "ami"},
			},
			"aws_s3_bucket": {
				Level: CriticalityCritical,
				CriticalAttributes: []string{"bucket", "acl", "versioning"},
			},
		},
	}

	detector := NewModeDetector(SmartMode, criticalityConfig)

	// Test high criticality resource
	criticalResource := &models.Resource{
		ID:   "critical-resource",
		Type: "aws_s3_bucket",
		Name: "critical-bucket",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"bucket": "my-bucket",
			"acl": "private",
		},
	}

	cloudResource := &models.Resource{
		ID:   "critical-resource",
		Type: "aws_s3_bucket",
		Name: "critical-bucket",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"bucket": "my-bucket",
			"acl": "public-read", // Critical attribute changed
		},
	}

	// Smart mode should do deep check for critical resources
	hasDrift := detector.DetectDrift(ctx, criticalResource, cloudResource)
	assert.True(t, hasDrift, "Smart mode should detect drift in critical resources")

	// Test low criticality resource
	lowCriticalityResource := &models.Resource{
		ID:   "low-resource",
		Type: "aws_route53_record",
		Name: "dns-record",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"name": "example.com",
			"ttl": 300,
		},
	}

	cloudResourceLow := &models.Resource{
		ID:   "low-resource",
		Type: "aws_route53_record",
		Name: "dns-record",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"name": "example.com",
			"ttl": 600, // Different but low criticality
		},
	}

	// Smart mode should do quick check for low criticality resources
	hasDrift = detector.DetectDrift(ctx, lowCriticalityResource, cloudResourceLow)
	assert.False(t, hasDrift, "Smart mode should not detect minor drift in low criticality resources")
}

func TestGetRecommendedMode(t *testing.T) {
	tests := []struct {
		name             string
		resourceCount    int
		criticalResources int
		expectedMode     DetectionMode
	}{
		{
			name:             "Small infrastructure",
			resourceCount:    50,
			criticalResources: 5,
			expectedMode:     DeepMode,
		},
		{
			name:             "Large infrastructure with few critical",
			resourceCount:    1000,
			criticalResources: 10,
			expectedMode:     SmartMode,
		},
		{
			name:             "Large infrastructure",
			resourceCount:    5000,
			criticalResources: 100,
			expectedMode:     QuickMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := GetRecommendedMode(tt.resourceCount, tt.criticalResources)
			assert.Equal(t, tt.expectedMode, mode, "Recommended mode should match expected")
		})
	}
}

func TestDriftDetectionWithComparator(t *testing.T) {
	ctx := context.Background()
	detector := NewModeDetector(DeepMode, nil)

	stateResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami": "ami-12345",
			"security_groups": []string{"sg-1", "sg-2"},
			"tags": map[string]interface{}{
				"Name": "test",
				"Environment": "production",
			},
		},
	}

	cloudResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami": "ami-12345",
			"security_groups": []string{"sg-2", "sg-1"}, // Order different
			"tags": map[string]interface{}{
				"Name": "test",
				"Environment": "production",
				"Owner": "team", // Additional tag
			},
		},
	}

	// Deep mode should handle complex comparisons
	hasDrift := detector.DetectDrift(ctx, stateResource, cloudResource)
	
	// The detector should recognize that security groups order doesn't matter
	// but the additional tag is drift
	assert.True(t, hasDrift, "Should detect additional attributes as drift")
}

func BenchmarkQuickModeDetection(b *testing.B) {
	ctx := context.Background()
	detector := NewModeDetector(QuickMode, nil)
	
	stateResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
	}
	
	cloudResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectDrift(ctx, stateResource, cloudResource)
	}
}

func BenchmarkDeepModeDetection(b *testing.B) {
	ctx := context.Background()
	detector := NewModeDetector(DeepMode, nil)
	
	stateResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami": "ami-12345",
			"tags": map[string]interface{}{
				"Name": "test",
				"Environment": "dev",
			},
		},
	}
	
	cloudResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test-instance",
		Provider: "aws",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami": "ami-12345",
			"tags": map[string]interface{}{
				"Name": "test",
				"Environment": "dev",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectDrift(ctx, stateResource, cloudResource)
	}
}