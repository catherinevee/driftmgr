package detector

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
)

// TestNewEnhancedDetector tests the creation of a new EnhancedDetector
func TestNewEnhancedDetector(t *testing.T) {
	detector := NewEnhancedDetector()
	
	assert.NotNil(t, detector, "Expected a new EnhancedDetector instance")
	assert.NotNil(t, detector.errorHandler, "Expected error handler to be initialized")
	assert.NotNil(t, detector.recovery, "Expected recovery executor to be initialized")
	assert.NotEmpty(t, detector.correlationID, "Expected a non-empty correlation ID")
}

// TestEnhancedDetector_DetectDriftWithContext_EmptyState tests detection with an empty state
func TestEnhancedDetector_DetectDriftWithContext_EmptyState(t *testing.T) {
	detector := NewEnhancedDetector()
	state := &state.TerraformState{}

	report, err := detector.DetectDriftWithContext(context.Background(), state)

	assert.NoError(t, err, "Expected no error with empty state")
	assert.NotNil(t, report, "Expected a non-nil report")
	assert.Empty(t, report.DriftResults, "Expected no drift results for empty state")
}

// TestEnhancedDetector_GenerateTraceID tests trace ID generation
func TestEnhancedDetector_GenerateTraceID(t *testing.T) {
	detector := NewEnhancedDetector()
	traceID1 := detector.generateTraceID()
	traceID2 := detector.generateTraceID()

	assert.NotEmpty(t, traceID1, "Expected a non-empty trace ID")
	assert.NotEqual(t, traceID1, traceID2, "Expected each trace ID to be unique")
}

// TestEnhancedDetector_GetProvider tests the getProvider method
func TestEnhancedDetector_GetProvider(t *testing.T) {
	testProvider := &mockCloudProvider{}
	detector := &EnhancedDetector{
		providers: map[string]providers.CloudProvider{
			"test": testProvider,
		},
	}

	// Test with existing provider
	provider, err := detector.getProvider("test")
	assert.NoError(t, err, "Expected no error for existing provider")
	assert.Equal(t, testProvider, provider, "Expected the test provider to be returned")

	// Test with non-existing provider
	_, err = detector.getProvider("nonexistent")
	assert.Error(t, err, "Expected an error for non-existent provider")
}

// TestEnhancedDetector_IsNotFoundError tests the isNotFoundError helper function
func TestEnhancedDetector_IsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Not found error",
			err:      &providers.NotFoundError{ResourceID: "test"},
			expected: true,
		},
		{
			name:     "Other error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result, "Unexpected result for isNotFoundError")
		})
	}
}

// mockCloudProvider is a mock implementation of the CloudProvider interface
type mockCloudProvider struct{}

// Name returns the provider name
func (m *mockCloudProvider) Name() string {
	return "mock"
}

// DiscoverResources discovers resources in the specified region
func (m *mockCloudProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	return []models.Resource{
		{
			ID:       "mock-resource-1",
			Type:     "mock_resource",
			Provider: "mock",
			Region:   region,
			Properties: map[string]interface{}{
				"id":   "mock-resource-1",
				"name": "test-resource",
			},
		},
	}, nil
}

// GetResource retrieves a specific resource by ID
func (m *mockCloudProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	if resourceID == "not-found" {
		return nil, &providers.NotFoundError{
			Provider:   "mock",
			ResourceID: resourceID,
			Region:     "us-east-1",
		}
	}

	return &models.Resource{
		ID:       resourceID,
		Type:     "mock_resource",
		Provider: "mock",
		Region:   "us-east-1",
		Properties: map[string]interface{}{
			"id":   resourceID,
			"name": "test-resource",
		},
	}, nil
}

// ValidateCredentials checks if the provider credentials are valid
func (m *mockCloudProvider) ValidateCredentials(ctx context.Context) error {
	return nil
}

// ListRegions returns available regions for the provider
func (m *mockCloudProvider) ListRegions(ctx context.Context) ([]string, error) {
	return []string{"us-east-1", "us-west-2"}, nil
}

// SupportedResourceTypes returns the list of supported resource types
func (m *mockCloudProvider) SupportedResourceTypes() []string {
	return []string{"mock_resource"}
}
