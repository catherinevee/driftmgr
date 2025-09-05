package detector

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockProvider is a mock cloud provider for testing
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProvider) GetProviderName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProvider) DiscoverResources(ctx context.Context, config map[string]interface{}) ([]providers.CloudResource, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]providers.CloudResource), args.Error(1)
}

func (m *MockProvider) GetResource(ctx context.Context, resourceType, resourceID string) (*providers.CloudResource, error) {
	args := m.Called(ctx, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*providers.CloudResource), args.Error(1)
}

func (m *MockProvider) ValidateCredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProvider) EstimateCost(ctx context.Context, resourceType string, config map[string]interface{}) (float64, error) {
	args := m.Called(ctx, resourceType, config)
	return args.Get(0).(float64), args.Error(1)
}

func TestNewDriftDetector(t *testing.T) {
	mockProvider := new(MockProvider)
	mockProvider.On("GetProviderName").Return("mock")

	detector := NewDriftDetector(mockProvider)
	assert.NotNil(t, detector)
	assert.Equal(t, "mock", detector.provider.GetProviderName())
}

func TestDriftDetector_DetectDrift(t *testing.T) {
	tests := []struct {
		name            string
		actualResources []providers.CloudResource
		desiredState    map[string]interface{}
		expectedDrifts  int
		expectError     bool
	}{
		{
			name: "no drift",
			actualResources: []providers.CloudResource{
				{
					ID:   "i-12345",
					Type: "aws_instance",
					Name: "test-instance",
					Properties: map[string]interface{}{
						"instance_type": "t2.micro",
						"ami":           "ami-12345",
					},
				},
			},
			desiredState: map[string]interface{}{
				"resources": []map[string]interface{}{
					{
						"id":   "i-12345",
						"type": "aws_instance",
						"name": "test-instance",
						"properties": map[string]interface{}{
							"instance_type": "t2.micro",
							"ami":           "ami-12345",
						},
					},
				},
			},
			expectedDrifts: 0,
			expectError:    false,
		},
		{
			name: "configuration drift",
			actualResources: []providers.CloudResource{
				{
					ID:   "i-12345",
					Type: "aws_instance",
					Name: "test-instance",
					Properties: map[string]interface{}{
						"instance_type": "t2.medium", // Changed from t2.micro
						"ami":           "ami-12345",
					},
				},
			},
			desiredState: map[string]interface{}{
				"resources": []map[string]interface{}{
					{
						"id":   "i-12345",
						"type": "aws_instance",
						"name": "test-instance",
						"properties": map[string]interface{}{
							"instance_type": "t2.micro",
							"ami":           "ami-12345",
						},
					},
				},
			},
			expectedDrifts: 1,
			expectError:    false,
		},
		{
			name:            "missing resource",
			actualResources: []providers.CloudResource{},
			desiredState: map[string]interface{}{
				"resources": []map[string]interface{}{
					{
						"id":   "i-12345",
						"type": "aws_instance",
						"name": "test-instance",
						"properties": map[string]interface{}{
							"instance_type": "t2.micro",
							"ami":           "ami-12345",
						},
					},
				},
			},
			expectedDrifts: 1,
			expectError:    false,
		},
		{
			name: "unmanaged resource",
			actualResources: []providers.CloudResource{
				{
					ID:   "i-12345",
					Type: "aws_instance",
					Name: "test-instance",
					Properties: map[string]interface{}{
						"instance_type": "t2.micro",
						"ami":           "ami-12345",
					},
				},
				{
					ID:   "i-67890",
					Type: "aws_instance",
					Name: "unmanaged-instance",
					Properties: map[string]interface{}{
						"instance_type": "t2.small",
						"ami":           "ami-67890",
					},
				},
			},
			desiredState: map[string]interface{}{
				"resources": []map[string]interface{}{
					{
						"id":   "i-12345",
						"type": "aws_instance",
						"name": "test-instance",
						"properties": map[string]interface{}{
							"instance_type": "t2.micro",
							"ami":           "ami-12345",
						},
					},
				},
			},
			expectedDrifts: 1,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(MockProvider)
			mockProvider.On("GetProviderName").Return("mock")
			mockProvider.On("DiscoverResources", mock.Anything, mock.Anything).Return(tt.actualResources, nil)

			detector := NewDriftDetector(mockProvider)
			ctx := context.Background()

			drifts, err := detector.DetectDrift(ctx, tt.desiredState)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, drifts, tt.expectedDrifts)
			}

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestDriftDetector_AnalyzeDrift(t *testing.T) {
	detector := &DriftDetector{}

	tests := []struct {
		name     string
		drift    *DriftResult
		expected DriftAnalysis
	}{
		{
			name: "critical drift - production resource",
			drift: &DriftResult{
				ResourceID:   "i-12345",
				ResourceType: "aws_instance",
				DriftType:    ConfigurationDrift,
				Details: map[string]interface{}{
					"tags": map[string]string{
						"Environment": "production",
					},
				},
			},
			expected: DriftAnalysis{
				Severity: "critical",
				Impact:   "high",
			},
		},
		{
			name: "low severity - development resource",
			drift: &DriftResult{
				ResourceID:   "i-67890",
				ResourceType: "aws_instance",
				DriftType:    ConfigurationDrift,
				Details: map[string]interface{}{
					"tags": map[string]string{
						"Environment": "development",
					},
				},
			},
			expected: DriftAnalysis{
				Severity: "low",
				Impact:   "minimal",
			},
		},
		{
			name: "unmanaged resource",
			drift: &DriftResult{
				ResourceID:   "i-unmanaged",
				ResourceType: "aws_instance",
				DriftType:    UnmanagedResource,
				Details:      map[string]interface{}{},
			},
			expected: DriftAnalysis{
				Severity: "medium",
				Impact:   "unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := detector.AnalyzeDrift(tt.drift)
			assert.Equal(t, tt.expected.Severity, analysis.Severity)
			assert.Equal(t, tt.expected.Impact, analysis.Impact)
		})
	}
}

func TestDriftDetector_GenerateReport(t *testing.T) {
	detector := &DriftDetector{}

	drifts := []*DriftResult{
		{
			ResourceID:   "i-12345",
			ResourceType: "aws_instance",
			DriftType:    ConfigurationDrift,
			Details: map[string]interface{}{
				"changed_properties": map[string]interface{}{
					"instance_type": map[string]string{
						"actual":  "t2.medium",
						"desired": "t2.micro",
					},
				},
			},
		},
		{
			ResourceID:   "sg-67890",
			ResourceType: "aws_security_group",
			DriftType:    MissingResource,
			Details:      map[string]interface{}{},
		},
		{
			ResourceID:   "vpc-unmanaged",
			ResourceType: "aws_vpc",
			DriftType:    UnmanagedResource,
			Details:      map[string]interface{}{},
		},
	}

	report := detector.GenerateReport(drifts)

	assert.NotNil(t, report)
	assert.Equal(t, 3, report.TotalDrifts)
	assert.Equal(t, 1, report.ConfigurationDrifts)
	assert.Equal(t, 1, report.MissingResources)
	assert.Equal(t, 1, report.UnmanagedResources)
	assert.NotEmpty(t, report.Summary)
	assert.Len(t, report.Details, 3)
}

func TestDriftDetector_CompareProperties(t *testing.T) {
	detector := &DriftDetector{}

	tests := []struct {
		name     string
		actual   map[string]interface{}
		desired  map[string]interface{}
		hasDrift bool
		changes  map[string]interface{}
	}{
		{
			name: "no changes",
			actual: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":           "ami-12345",
			},
			desired: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":           "ami-12345",
			},
			hasDrift: false,
			changes:  map[string]interface{}{},
		},
		{
			name: "property changed",
			actual: map[string]interface{}{
				"instance_type": "t2.medium",
				"ami":           "ami-12345",
			},
			desired: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":           "ami-12345",
			},
			hasDrift: true,
			changes: map[string]interface{}{
				"instance_type": map[string]string{
					"actual":  "t2.medium",
					"desired": "t2.micro",
				},
			},
		},
		{
			name: "property added",
			actual: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":           "ami-12345",
				"monitoring":    true,
			},
			desired: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":           "ami-12345",
			},
			hasDrift: true,
			changes: map[string]interface{}{
				"monitoring": map[string]interface{}{
					"actual":  true,
					"desired": nil,
				},
			},
		},
		{
			name: "property removed",
			actual: map[string]interface{}{
				"instance_type": "t2.micro",
			},
			desired: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":           "ami-12345",
			},
			hasDrift: true,
			changes: map[string]interface{}{
				"ami": map[string]interface{}{
					"actual":  nil,
					"desired": "ami-12345",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasDrift, changes := detector.CompareProperties(tt.actual, tt.desired)
			assert.Equal(t, tt.hasDrift, hasDrift)
			assert.Equal(t, tt.changes, changes)
		})
	}
}

// Benchmark tests
func BenchmarkDriftDetector_DetectDrift(b *testing.B) {
	mockProvider := new(MockProvider)
	mockProvider.On("GetProviderName").Return("mock")

	resources := make([]providers.CloudResource, 100)
	for i := 0; i < 100; i++ {
		resources[i] = providers.CloudResource{
			ID:   fmt.Sprintf("resource-%d", i),
			Type: "aws_instance",
			Name: fmt.Sprintf("instance-%d", i),
			Properties: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":           "ami-12345",
			},
		}
	}

	mockProvider.On("DiscoverResources", mock.Anything, mock.Anything).Return(resources, nil)

	detector := NewDriftDetector(mockProvider)
	ctx := context.Background()

	desiredState := map[string]interface{}{
		"resources": make([]map[string]interface{}, 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectDrift(ctx, desiredState)
	}
}

func BenchmarkDriftDetector_CompareProperties(b *testing.B) {
	detector := &DriftDetector{}

	actual := map[string]interface{}{
		"instance_type": "t2.micro",
		"ami":           "ami-12345",
		"tags": map[string]string{
			"Name":        "test",
			"Environment": "prod",
			"Team":        "platform",
		},
		"security_groups": []string{"sg-1", "sg-2", "sg-3"},
	}

	desired := map[string]interface{}{
		"instance_type": "t2.medium",
		"ami":           "ami-67890",
		"tags": map[string]string{
			"Name":        "test-updated",
			"Environment": "staging",
			"Team":        "devops",
		},
		"security_groups": []string{"sg-1", "sg-4"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.CompareProperties(actual, desired)
	}
}
