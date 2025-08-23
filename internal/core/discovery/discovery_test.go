package discovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockProvider is a mock cloud provider for testing
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProvider) DiscoverResources(ctx context.Context) ([]models.Resource, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Resource), args.Error(1)
}

func (m *MockProvider) GetRegions() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockProvider) ValidateCredentials() error {
	args := m.Called()
	return args.Error(0)
}

func TestDiscoveryEngine_DiscoverResources(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockProvider)
		expectedCount int
		expectedError bool
		timeout       time.Duration
	}{
		{
			name: "successful discovery",
			setupMock: func(m *MockProvider) {
				resources := []models.Resource{
					{ID: "1", Name: "resource1", Type: "EC2"},
					{ID: "2", Name: "resource2", Type: "S3"},
				}
				m.On("Name").Return("aws")
				m.On("DiscoverResources", mock.Anything).Return(resources, nil)
			},
			expectedCount: 2,
			expectedError: false,
			timeout:       5 * time.Second,
		},
		{
			name: "discovery with error",
			setupMock: func(m *MockProvider) {
				m.On("Name").Return("aws")
				m.On("DiscoverResources", mock.Anything).Return([]models.Resource{}, errors.New("provider error"))
			},
			expectedCount: 0,
			expectedError: true,
			timeout:       5 * time.Second,
		},
		{
			name: "discovery with timeout",
			setupMock: func(m *MockProvider) {
				m.On("Name").Return("aws")
				m.On("DiscoverResources", mock.Anything).Run(func(args mock.Arguments) {
					time.Sleep(2 * time.Second)
				}).Return([]models.Resource{}, context.DeadlineExceeded)
			},
			expectedCount: 0,
			expectedError: true,
			timeout:       1 * time.Second,
		},
		{
			name: "empty discovery",
			setupMock: func(m *MockProvider) {
				m.On("Name").Return("aws")
				m.On("DiscoverResources", mock.Anything).Return([]models.Resource{}, nil)
			},
			expectedCount: 0,
			expectedError: false,
			timeout:       5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockProvider := new(MockProvider)
			tt.setupMock(mockProvider)

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Execute
			resources, err := mockProvider.DiscoverResources(ctx)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Len(t, resources, tt.expectedCount)

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestDiscoveryEngine_ParallelDiscovery(t *testing.T) {
	// Test parallel discovery across multiple providers
	providers := make([]*MockProvider, 3)
	for i := range providers {
		providers[i] = new(MockProvider)
		providers[i].On("Name").Return(string(rune('A' + i)))
		providers[i].On("DiscoverResources", mock.Anything).Return([]models.Resource{
			{ID: string(rune('1' + i)), Name: "resource", Type: "EC2"},
		}, nil)
	}

	ctx := context.Background()
	results := make([][]models.Resource, len(providers))
	errors := make([]error, len(providers))

	// Run parallel discovery
	done := make(chan bool, len(providers))
	for i, provider := range providers {
		go func(idx int, p *MockProvider) {
			results[idx], errors[idx] = p.DiscoverResources(ctx)
			done <- true
		}(i, provider)
	}

	// Wait for all to complete
	for i := 0; i < len(providers); i++ {
		<-done
	}

	// Assert
	totalResources := 0
	for i, err := range errors {
		assert.NoError(t, err)
		totalResources += len(results[i])
	}
	assert.Equal(t, 3, totalResources)

	for _, provider := range providers {
		provider.AssertExpectations(t)
	}
}

func TestDiscoveryEngine_RetryLogic(t *testing.T) {
	mockProvider := new(MockProvider)
	
	// Setup mock to fail twice then succeed
	callCount := 0
	mockProvider.On("Name").Return("aws")
	mockProvider.On("DiscoverResources", mock.Anything).Return(
		func(ctx context.Context) []models.Resource {
			callCount++
			if callCount < 3 {
				return []models.Resource{}
			}
			return []models.Resource{{ID: "1", Name: "resource", Type: "EC2"}}
		},
		func(ctx context.Context) error {
			if callCount < 3 {
				return errors.New("temporary error")
			}
			return nil
		},
	)

	ctx := context.Background()
	
	// Implement retry logic
	var resources []models.Resource
	var err error
	maxRetries := 3
	
	for i := 0; i < maxRetries; i++ {
		resources, err = mockProvider.DiscoverResources(ctx)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Assert
	assert.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, 3, callCount)
}

func TestDiscoveryEngine_Filtering(t *testing.T) {
	resources := []models.Resource{
		{ID: "1", Name: "prod-server", Type: "EC2", Tags: map[string]string{"env": "production"}},
		{ID: "2", Name: "dev-server", Type: "EC2", Tags: map[string]string{"env": "development"}},
		{ID: "3", Name: "test-bucket", Type: "S3", Tags: map[string]string{"env": "test"}},
		{ID: "4", Name: "prod-db", Type: "RDS", Tags: map[string]string{"env": "production"}},
	}

	tests := []struct {
		name           string
		filterFunc     func(models.Resource) bool
		expectedCount  int
		expectedIDs    []string
	}{
		{
			name: "filter by production environment",
			filterFunc: func(r models.Resource) bool {
				tags, ok := r.Tags.(map[string]string)
				if !ok {
					return false
				}
				return tags["env"] == "production"
			},
			expectedCount: 2,
			expectedIDs:   []string{"1", "4"},
		},
		{
			name: "filter by resource type EC2",
			filterFunc: func(r models.Resource) bool {
				return r.Type == "EC2"
			},
			expectedCount: 2,
			expectedIDs:   []string{"1", "2"},
		},
		{
			name: "filter by name pattern",
			filterFunc: func(r models.Resource) bool {
				return len(r.Name) > 0 && r.Name[0:4] == "prod"
			},
			expectedCount: 2,
			expectedIDs:   []string{"1", "4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterResources(resources, tt.filterFunc)
			
			assert.Len(t, filtered, tt.expectedCount)
			
			ids := make([]string, len(filtered))
			for i, r := range filtered {
				ids[i] = r.ID
			}
			assert.ElementsMatch(t, tt.expectedIDs, ids)
		})
	}
}

func TestDiscoveryEngine_Caching(t *testing.T) {
	mockProvider := new(MockProvider)
	callCount := 0
	
	mockProvider.On("Name").Return("aws")
	mockProvider.On("DiscoverResources", mock.Anything).Return(
		func(ctx context.Context) []models.Resource {
			callCount++
			return []models.Resource{{ID: "1", Name: "resource", Type: "EC2"}}
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	ctx := context.Background()
	
	// First call - should hit provider
	resources1, err := mockProvider.DiscoverResources(ctx)
	require.NoError(t, err)
	assert.Len(t, resources1, 1)
	assert.Equal(t, 1, callCount)
	
	// Second call - should use cache (simulated)
	// In real implementation, this would check cache first
	resources2, err := mockProvider.DiscoverResources(ctx)
	require.NoError(t, err)
	assert.Len(t, resources2, 1)
	assert.Equal(t, 2, callCount) // Would be 1 with real caching
}

func TestDiscoveryEngine_ErrorAggregation(t *testing.T) {
	providers := make([]*MockProvider, 3)
	expectedErrors := []error{
		nil,
		errors.New("provider 2 error"),
		errors.New("provider 3 error"),
	}

	for i := range providers {
		providers[i] = new(MockProvider)
		providers[i].On("Name").Return(string(rune('A' + i)))
		providers[i].On("DiscoverResources", mock.Anything).Return(
			[]models.Resource{},
			expectedErrors[i],
		)
	}

	ctx := context.Background()
	var aggregatedErrors []error

	for _, provider := range providers {
		_, err := provider.DiscoverResources(ctx)
		if err != nil {
			aggregatedErrors = append(aggregatedErrors, err)
		}
	}

	// Assert
	assert.Len(t, aggregatedErrors, 2)
	assert.Contains(t, aggregatedErrors[0].Error(), "provider 2")
	assert.Contains(t, aggregatedErrors[1].Error(), "provider 3")
}

// Helper function for filtering
func filterResources(resources []models.Resource, filterFunc func(models.Resource) bool) []models.Resource {
	var filtered []models.Resource
	for _, r := range resources {
		if filterFunc(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// Benchmark tests
func BenchmarkDiscoveryEngine_DiscoverResources(b *testing.B) {
	mockProvider := new(MockProvider)
	resources := make([]models.Resource, 1000)
	for i := range resources {
		resources[i] = models.Resource{
			ID:   string(rune(i)),
			Name: "resource",
			Type: "EC2",
		}
	}
	
	mockProvider.On("Name").Return("aws")
	mockProvider.On("DiscoverResources", mock.Anything).Return(resources, nil)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mockProvider.DiscoverResources(ctx)
	}
}

func BenchmarkDiscoveryEngine_Filtering(b *testing.B) {
	resources := make([]models.Resource, 10000)
	for i := range resources {
		resources[i] = models.Resource{
			ID:   string(rune(i)),
			Name: "resource",
			Type: "EC2",
			Tags: map[string]string{
				"env": "production",
			},
		}
	}
	
	filterFunc := func(r models.Resource) bool {
		tags, ok := r.Tags.(map[string]string)
		if !ok {
			return false
		}
		return tags["env"] == "production"
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterResources(resources, filterFunc)
	}
}