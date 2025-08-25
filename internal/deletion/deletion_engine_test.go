package deletion

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockProvider) DeleteResource(ctx context.Context, resourceID string) error {
	args := m.Called(ctx, resourceID)
	return args.Error(0)
}

func (m *MockProvider) GetResourceDependencies(ctx context.Context, resourceID string) ([]string, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockProvider) ValidateCredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestDeletionEngine_NewEngine(t *testing.T) {
	engine := NewDeletionEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.providers)
	assert.NotNil(t, engine.logger)
}

func TestDeletionEngine_RegisterProvider(t *testing.T) {
	engine := NewDeletionEngine()
	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider")

	engine.RegisterProvider("test", mockProvider)
	assert.Contains(t, engine.providers, "test")
}

func TestDeletionEngine_DiscoverResources(t *testing.T) {
	tests := []struct {
		name          string
		providers     []string
		mockResources []models.Resource
		expectError   bool
	}{
		{
			name:      "successful discovery",
			providers: []string{"aws"},
			mockResources: []models.Resource{
				{ID: "i-123", Type: "aws_instance", Provider: "aws"},
				{ID: "sg-456", Type: "aws_security_group", Provider: "aws"},
			},
			expectError: false,
		},
		{
			name:          "empty discovery",
			providers:     []string{"aws"},
			mockResources: []models.Resource{},
			expectError:   false,
		},
		{
			name:        "invalid provider",
			providers:   []string{"invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewDeletionEngine()

			if tt.name != "invalid provider" {
				mockProvider := new(MockProvider)
				mockProvider.On("Name").Return("aws")
				mockProvider.On("DiscoverResources", mock.Anything).Return(tt.mockResources, nil)
				engine.RegisterProvider("aws", mockProvider)
			}

			options := DeletionOptions{
				Providers: tt.providers,
				DryRun:    true,
			}

			resources, err := engine.DiscoverResources(context.Background(), options)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.mockResources), len(resources))
			}
		})
	}
}

func TestDeletionEngine_DeleteResources(t *testing.T) {
	tests := []struct {
		name        string
		resources   []models.Resource
		dryRun      bool
		force       bool
		expectError bool
	}{
		{
			name: "dry run deletion",
			resources: []models.Resource{
				{ID: "i-123", Type: "aws_instance", Provider: "aws"},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "actual deletion",
			resources: []models.Resource{
				{ID: "i-123", Type: "aws_instance", Provider: "aws"},
			},
			dryRun:      false,
			force:       true,
			expectError: false,
		},
		{
			name: "deletion with dependencies",
			resources: []models.Resource{
				{ID: "sg-456", Type: "aws_security_group", Provider: "aws"},
				{ID: "i-123", Type: "aws_instance", Provider: "aws"},
			},
			dryRun:      false,
			force:       true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewDeletionEngine()

			mockProvider := new(MockProvider)
			mockProvider.On("Name").Return("aws")
			mockProvider.On("DeleteResource", mock.Anything, mock.Anything).Return(nil)
			mockProvider.On("GetResourceDependencies", mock.Anything, "i-123").Return([]string{"sg-456"}, nil)
			mockProvider.On("GetResourceDependencies", mock.Anything, "sg-456").Return([]string{}, nil)
			engine.RegisterProvider("aws", mockProvider)

			options := DeletionOptions{
				DryRun:     tt.dryRun,
				Force:      tt.force,
				BatchSize:  10,
				MaxRetries: 3,
				RetryDelay: time.Second,
			}

			result, err := engine.DeleteResources(context.Background(), tt.resources, options)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.dryRun {
					assert.Equal(t, 0, result.DeletedResources)
				}
			}
		})
	}
}

func TestDeletionEngine_SafetyChecks(t *testing.T) {
	tests := []struct {
		name       string
		resources  []models.Resource
		options    DeletionOptions
		expectPass bool
	}{
		{
			name: "pass safety check with force",
			resources: []models.Resource{
				{ID: "i-123", Type: "aws_instance", Provider: "aws"},
			},
			options: DeletionOptions{
				Force: true,
			},
			expectPass: true,
		},
		{
			name: "fail safety check for critical resource",
			resources: []models.Resource{
				{
					ID:   "prod-db",
					Type: "aws_rds_instance",
					Tags: map[string]string{"Environment": "production"},
				},
			},
			options: DeletionOptions{
				Force: false,
			},
			expectPass: false,
		},
		{
			name: "pass with tag filter",
			resources: []models.Resource{
				{
					ID:   "test-instance",
					Type: "aws_instance",
					Tags: map[string]string{"Environment": "test"},
				},
			},
			options: DeletionOptions{
				TagFilters: map[string]string{"Environment": "test"},
			},
			expectPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewDeletionEngine()

			passed := engine.performSafetyChecks(tt.resources, tt.options)
			assert.Equal(t, tt.expectPass, passed)
		})
	}
}

func TestDeletionEngine_DependencyOrdering(t *testing.T) {
	engine := NewDeletionEngine()

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("aws")

	// Setup dependency chain: instance -> security group -> vpc
	mockProvider.On("GetResourceDependencies", mock.Anything, "i-123").Return([]string{"sg-456"}, nil)
	mockProvider.On("GetResourceDependencies", mock.Anything, "sg-456").Return([]string{"vpc-789"}, nil)
	mockProvider.On("GetResourceDependencies", mock.Anything, "vpc-789").Return([]string{}, nil)

	engine.RegisterProvider("aws", mockProvider)

	resources := []models.Resource{
		{ID: "vpc-789", Type: "aws_vpc", Provider: "aws"},
		{ID: "i-123", Type: "aws_instance", Provider: "aws"},
		{ID: "sg-456", Type: "aws_security_group", Provider: "aws"},
	}

	ordered, err := engine.orderResourcesByDependencies(context.Background(), resources)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(ordered))

	// Should delete in reverse dependency order
	assert.Equal(t, "i-123", ordered[0].ID)
	assert.Equal(t, "sg-456", ordered[1].ID)
	assert.Equal(t, "vpc-789", ordered[2].ID)
}

func TestDeletionEngine_RetryLogic(t *testing.T) {
	engine := NewDeletionEngine()

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("aws")

	// Simulate failure then success
	callCount := 0
	mockProvider.On("DeleteResource", mock.Anything, "i-123").Return(
		func(ctx context.Context, id string) error {
			callCount++
			if callCount < 3 {
				return assert.AnError
			}
			return nil
		},
	)

	engine.RegisterProvider("aws", mockProvider)

	resource := models.Resource{ID: "i-123", Type: "aws_instance", Provider: "aws"}
	options := DeletionOptions{
		MaxRetries: 3,
		RetryDelay: time.Millisecond,
	}

	err := engine.deleteResourceWithRetry(context.Background(), resource, options)
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestDeletionEngine_BatchProcessing(t *testing.T) {
	engine := NewDeletionEngine()

	// Create 25 resources
	resources := make([]models.Resource, 25)
	for i := 0; i < 25; i++ {
		resources[i] = models.Resource{
			ID:       fmt.Sprintf("resource-%d", i),
			Type:     "aws_instance",
			Provider: "aws",
		}
	}

	options := DeletionOptions{
		BatchSize: 10,
		DryRun:    true,
	}

	batches := engine.batchResources(resources, options.BatchSize)
	assert.Equal(t, 3, len(batches))
	assert.Equal(t, 10, len(batches[0]))
	assert.Equal(t, 10, len(batches[1]))
	assert.Equal(t, 5, len(batches[2]))
}

func TestDeletionEngine_ProtectedResources(t *testing.T) {
	engine := NewDeletionEngine()

	tests := []struct {
		name      string
		resource  models.Resource
		protected bool
	}{
		{
			name: "production database protected",
			resource: models.Resource{
				ID:   "prod-db",
				Type: "aws_rds_instance",
				Tags: map[string]string{
					"Environment": "production",
					"Protected":   "true",
				},
			},
			protected: true,
		},
		{
			name: "default VPC protected",
			resource: models.Resource{
				ID:   "vpc-default",
				Name: "default",
				Type: "aws_vpc",
			},
			protected: true,
		},
		{
			name: "test resource not protected",
			resource: models.Resource{
				ID:   "test-instance",
				Type: "aws_instance",
				Tags: map[string]string{"Environment": "test"},
			},
			protected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isProtected := engine.isProtectedResource(tt.resource)
			assert.Equal(t, tt.protected, isProtected)
		})
	}
}

func TestDeletionEngine_ConcurrentDeletion(t *testing.T) {
	engine := NewDeletionEngine()

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("aws")
	mockProvider.On("DeleteResource", mock.Anything, mock.Anything).Return(nil).After(10 * time.Millisecond)
	mockProvider.On("GetResourceDependencies", mock.Anything, mock.Anything).Return([]string{}, nil)

	engine.RegisterProvider("aws", mockProvider)

	// Create independent resources that can be deleted concurrently
	resources := []models.Resource{
		{ID: "i-1", Type: "aws_instance", Provider: "aws"},
		{ID: "i-2", Type: "aws_instance", Provider: "aws"},
		{ID: "i-3", Type: "aws_instance", Provider: "aws"},
		{ID: "i-4", Type: "aws_instance", Provider: "aws"},
		{ID: "i-5", Type: "aws_instance", Provider: "aws"},
	}

	options := DeletionOptions{
		Parallel:   true,
		MaxWorkers: 3,
		Force:      true,
	}

	start := time.Now()
	result, err := engine.DeleteResources(context.Background(), resources, options)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 5, result.DeletedResources)
	// With parallelism, should complete faster than serial execution
	assert.Less(t, duration, 100*time.Millisecond)
}
