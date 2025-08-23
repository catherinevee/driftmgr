package discovery

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCloudProvider mocks a cloud provider
type MockCloudProvider struct {
	mock.Mock
}

func (m *MockCloudProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCloudProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	args := m.Called(ctx, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Resource), args.Error(1)
}

func (m *MockCloudProvider) GetRegions() []string {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *MockCloudProvider) ValidateCredentials() error {
	args := m.Called()
	return args.Error(0)
}

func TestEngine_Discovery(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockCloudProvider)
		regions       []string
		expectedCount int
		expectedError bool
	}{
		{
			name: "successful single region discovery",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("GetRegions").Return([]string{"us-east-1"})
				m.On("ValidateCredentials").Return(nil)
				m.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{
					{ID: "i-123", Name: "instance1", Type: "EC2"},
					{ID: "i-456", Name: "instance2", Type: "EC2"},
				}, nil)
			},
			regions:       []string{"us-east-1"},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "successful multi-region discovery",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("GetRegions").Return([]string{"us-east-1", "us-west-2"})
				m.On("ValidateCredentials").Return(nil)
				m.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{
					{ID: "i-123", Name: "instance1", Type: "EC2"},
				}, nil)
				m.On("DiscoverResources", mock.Anything, "us-west-2").Return([]models.Resource{
					{ID: "i-789", Name: "instance3", Type: "EC2"},
				}, nil)
			},
			regions:       []string{"us-east-1", "us-west-2"},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "partial failure in multi-region",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("GetRegions").Return([]string{"us-east-1", "us-west-2"})
				m.On("ValidateCredentials").Return(nil)
				m.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{
					{ID: "i-123", Name: "instance1", Type: "EC2"},
				}, nil)
				m.On("DiscoverResources", mock.Anything, "us-west-2").Return(nil, errors.New("region unavailable"))
			},
			regions:       []string{"us-east-1", "us-west-2"},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "credential validation failure",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("ValidateCredentials").Return(errors.New("invalid credentials"))
			},
			regions:       []string{"us-east-1"},
			expectedCount: 0,
			expectedError: true,
		},
		{
			name: "empty resource discovery",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("GetRegions").Return([]string{"us-east-1"})
				m.On("ValidateCredentials").Return(nil)
				m.On("DiscoverResources", mock.Anything, "us-east-1").Return([]models.Resource{}, nil)
			},
			regions:       []string{"us-east-1"},
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(MockCloudProvider)
			tt.setupMock(mockProvider)

			// Create a simple engine wrapper for testing
			engine := &testEngine{
				provider: mockProvider,
			}

			ctx := context.Background()
			resources, err := engine.DiscoverAll(ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(resources))
			}

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestEngine_ParallelDiscovery(t *testing.T) {
	mockProvider := new(MockCloudProvider)
	
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	
	mockProvider.On("Name").Return("aws")
	mockProvider.On("GetRegions").Return(regions)
	mockProvider.On("ValidateCredentials").Return(nil)
	
	// Setup mock for each region
	for i, region := range regions {
		resources := []models.Resource{
			{ID: fmt.Sprintf("i-%d-1", i), Name: fmt.Sprintf("instance-%s-1", region), Type: "EC2"},
			{ID: fmt.Sprintf("i-%d-2", i), Name: fmt.Sprintf("instance-%s-2", region), Type: "EC2"},
		}
		mockProvider.On("DiscoverResources", mock.Anything, region).Return(resources, nil)
	}

	engine := &testEngine{
		provider: mockProvider,
	}

	ctx := context.Background()
	start := time.Now()
	resources, err := engine.DiscoverAllParallel(ctx)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 8, len(resources)) // 2 resources per region * 4 regions
	
	// Verify parallel execution (should be faster than sequential)
	assert.Less(t, duration, 2*time.Second)
	
	mockProvider.AssertExpectations(t)
}

func TestEngine_Timeout(t *testing.T) {
	mockProvider := new(MockCloudProvider)
	
	mockProvider.On("Name").Return("aws")
	mockProvider.On("GetRegions").Return([]string{"us-east-1"})
	mockProvider.On("ValidateCredentials").Return(nil)
	mockProvider.On("DiscoverResources", mock.Anything, "us-east-1").Run(func(args mock.Arguments) {
		// Simulate long-running discovery
		time.Sleep(2 * time.Second)
	}).Return([]models.Resource{}, nil)

	engine := &testEngine{
		provider: mockProvider,
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resources, err := engine.DiscoverAll(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
	assert.Empty(t, resources)
}

func TestEngine_Filtering(t *testing.T) {
	resources := []models.Resource{
		{ID: "i-1", Name: "prod-server", Type: "EC2", Tags: map[string]string{"env": "production"}},
		{ID: "i-2", Name: "dev-server", Type: "EC2", Tags: map[string]string{"env": "development"}},
		{ID: "s3-1", Name: "data-bucket", Type: "S3", Tags: map[string]string{"env": "production"}},
		{ID: "rds-1", Name: "prod-db", Type: "RDS", Tags: map[string]string{"env": "production"}},
	}

	tests := []struct {
		name          string
		filter        func(models.Resource) bool
		expectedCount int
		expectedIDs   []string
	}{
		{
			name: "filter by production environment",
			filter: func(r models.Resource) bool {
				tags, ok := r.Tags.(map[string]string)
				if !ok {
					return false
				}
				return tags["env"] == "production"
			},
			expectedCount: 3,
			expectedIDs:   []string{"i-1", "s3-1", "rds-1"},
		},
		{
			name: "filter by EC2 type",
			filter: func(r models.Resource) bool {
				return r.Type == "EC2"
			},
			expectedCount: 2,
			expectedIDs:   []string{"i-1", "i-2"},
		},
		{
			name: "filter by name prefix",
			filter: func(r models.Resource) bool {
				return len(r.Name) > 4 && r.Name[:4] == "prod"
			},
			expectedCount: 2,
			expectedIDs:   []string{"i-1", "rds-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterResourcesEngine(resources, tt.filter)
			
			assert.Equal(t, tt.expectedCount, len(filtered))
			
			ids := make([]string, len(filtered))
			for i, r := range filtered {
				ids[i] = r.ID
			}
			assert.ElementsMatch(t, tt.expectedIDs, ids)
		})
	}
}

func TestEngine_Deduplication(t *testing.T) {
	resources := []models.Resource{
		{ID: "i-1", Name: "server1", Type: "EC2"},
		{ID: "i-1", Name: "server1", Type: "EC2"}, // Duplicate
		{ID: "i-2", Name: "server2", Type: "EC2"},
		{ID: "i-1", Name: "server1-updated", Type: "EC2"}, // Same ID, different name
		{ID: "i-3", Name: "server3", Type: "EC2"},
	}

	deduplicated := deduplicateResources(resources)
	
	assert.Equal(t, 3, len(deduplicated))
	
	// Verify unique IDs
	ids := make(map[string]bool)
	for _, r := range deduplicated {
		assert.False(t, ids[r.ID], "Duplicate ID found: %s", r.ID)
		ids[r.ID] = true
	}
}

func TestEngine_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockCloudProvider)
		expectedError string
	}{
		{
			name: "network error",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("ValidateCredentials").Return(nil)
				m.On("GetRegions").Return([]string{"us-east-1"})
				m.On("DiscoverResources", mock.Anything, "us-east-1").Return(nil, errors.New("network timeout"))
			},
			expectedError: "network timeout",
		},
		{
			name: "permission error",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("ValidateCredentials").Return(errors.New("access denied"))
			},
			expectedError: "access denied",
		},
		{
			name: "rate limit error",
			setupMock: func(m *MockCloudProvider) {
				m.On("Name").Return("aws")
				m.On("ValidateCredentials").Return(nil)
				m.On("GetRegions").Return([]string{"us-east-1"})
				m.On("DiscoverResources", mock.Anything, "us-east-1").Return(nil, errors.New("rate limit exceeded"))
			},
			expectedError: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(MockCloudProvider)
			tt.setupMock(mockProvider)

			engine := &testEngine{
				provider: mockProvider,
			}

			ctx := context.Background()
			_, err := engine.DiscoverAll(ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			
			mockProvider.AssertExpectations(t)
		})
	}
}

// Test helper structures

type testEngine struct {
	provider *MockCloudProvider
}

func (e *testEngine) DiscoverAll(ctx context.Context) ([]models.Resource, error) {
	if err := e.provider.ValidateCredentials(); err != nil {
		return nil, err
	}

	regions := e.provider.GetRegions()
	var allResources []models.Resource

	for _, region := range regions {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			resources, err := e.provider.DiscoverResources(ctx, region)
			if err != nil {
				// Log error but continue with other regions
				continue
			}
			allResources = append(allResources, resources...)
		}
	}

	return allResources, nil
}

func (e *testEngine) DiscoverAllParallel(ctx context.Context) ([]models.Resource, error) {
	if err := e.provider.ValidateCredentials(); err != nil {
		return nil, err
	}

	regions := e.provider.GetRegions()
	resultCh := make(chan []models.Resource, len(regions))
	errCh := make(chan error, len(regions))

	for _, region := range regions {
		go func(r string) {
			resources, err := e.provider.DiscoverResources(ctx, r)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- resources
		}(region)
	}

	var allResources []models.Resource
	for i := 0; i < len(regions); i++ {
		select {
		case resources := <-resultCh:
			allResources = append(allResources, resources...)
		case <-errCh:
			// Continue with other regions
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return allResources, nil
}

// Helper functions

func filterResourcesEngine(resources []models.Resource, filter func(models.Resource) bool) []models.Resource {
	var filtered []models.Resource
	for _, r := range resources {
		if filter(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func deduplicateResources(resources []models.Resource) []models.Resource {
	seen := make(map[string]bool)
	var deduplicated []models.Resource
	
	for _, r := range resources {
		if !seen[r.ID] {
			seen[r.ID] = true
			deduplicated = append(deduplicated, r)
		}
	}
	
	return deduplicated
}