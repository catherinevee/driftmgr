package discovery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/infrastructure/config"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDiscoveryPlugin for testing
type MockDiscoveryPlugin struct {
	mock.Mock
}

func (m *MockDiscoveryPlugin) Discover(ctx context.Context, provider, region string) ([]models.Resource, error) {
	args := m.Called(ctx, provider, region)
	return args.Get(0).([]models.Resource), args.Error(1)
}

func TestNewEnhancedDiscoverer(t *testing.T) {
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Timeout:          30 * time.Second,
			ConcurrencyLimit: 10,
			BatchSize:        100,
			RetryAttempts:    3,
			RetryDelay:       time.Second,
			EnableCaching:    true,
			CacheTTL:         5 * time.Minute,
			Regions:          []string{"us-east-1", "us-west-2"},
		},
	}

	discoverer := NewEnhancedDiscoverer(cfg)
	
	assert.NotNil(t, discoverer)
	assert.NotNil(t, discoverer.config)
	assert.NotNil(t, discoverer.cache)
	assert.NotNil(t, discoverer.plugins)
	assert.Equal(t, cfg, discoverer.config)
}

func TestDiscoverResources(t *testing.T) {
	testCases := []struct {
		name           string
		regions        []string
		mockResources  []models.Resource
		expectedCount  int
		expectError    bool
		errorMessage   string
	}{
		{
			name:    "successful discovery",
			regions: []string{"us-east-1"},
			mockResources: []models.Resource{
				{
					ID:       "i-12345",
					Name:     "test-instance",
					Type:     "ec2_instance",
					Provider: "aws",
					Region:   "us-east-1",
					State:    "running",
				},
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "empty discovery",
			regions:       []string{"us-east-1"},
			mockResources: []models.Resource{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:         "discovery with timeout",
			regions:      []string{"us-east-1"},
			expectError:  true,
			errorMessage: "context deadline exceeded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Discovery: config.DiscoveryConfig{
					Timeout:       30 * time.Second,
					EnableCaching: false,
					Regions:       tc.regions,
				},
			}

			discoverer := NewEnhancedDiscoverer(cfg)

			// Create test context
			ctx := context.Background()
			if tc.name == "discovery with timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond) // Ensure timeout
			}

			// Mock the provider discovery
			if tc.mockResources != nil {
				plugin := &DiscoveryPlugin{
					Name:    "test",
					Enabled: true,
					DiscoveryFn: func(ctx context.Context, provider, region string) ([]models.Resource, error) {
						if tc.expectError {
							return nil, context.DeadlineExceeded
						}
						return tc.mockResources, nil
					},
				}
				discoverer.RegisterPlugin(plugin)
			}

			resources, err := discoverer.DiscoverResources(ctx)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, resources, tc.expectedCount)
			}
		})
	}
}

func TestDiscoverResourcesWithCache(t *testing.T) {
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Timeout:       30 * time.Second,
			EnableCaching: true,
			CacheTTL:      5 * time.Minute,
			Regions:       []string{"us-east-1"},
		},
	}

	discoverer := NewEnhancedDiscoverer(cfg)

	// Mock resources
	mockResources := []models.Resource{
		{
			ID:       "i-12345",
			Name:     "cached-instance",
			Type:     "ec2_instance",
			Provider: "aws",
			Region:   "us-east-1",
		},
	}

	// First call - should hit the actual discovery
	callCount := 0
	plugin := &DiscoveryPlugin{
		Name:    "aws",
		Enabled: true,
		DiscoveryFn: func(ctx context.Context, provider, region string) ([]models.Resource, error) {
			callCount++
			return mockResources, nil
		},
	}
	discoverer.RegisterPlugin(plugin)

	ctx := context.Background()

	// First call
	resources1, err := discoverer.DiscoverResources(ctx)
	require.NoError(t, err)
	assert.Len(t, resources1, 1)
	assert.Equal(t, 1, callCount)

	// Second call - should hit cache
	resources2, err := discoverer.DiscoverResources(ctx)
	require.NoError(t, err)
	assert.Len(t, resources2, 1)
	assert.Equal(t, 1, callCount) // Should still be 1, cache was used
}

func TestRegisterPlugin(t *testing.T) {
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Timeout: 30 * time.Second,
		},
	}

	discoverer := NewEnhancedDiscoverer(cfg)

	plugin := &DiscoveryPlugin{
		Name:    "custom-plugin",
		Enabled: true,
		Priority: 10,
		DiscoveryFn: func(ctx context.Context, provider, region string) ([]models.Resource, error) {
			return []models.Resource{}, nil
		},
	}

	discoverer.RegisterPlugin(plugin)

	// Verify plugin was registered
	registeredPlugin, exists := discoverer.plugins["custom-plugin"]
	assert.True(t, exists)
	assert.Equal(t, plugin, registeredPlugin)
	assert.Equal(t, "custom-plugin", registeredPlugin.Name)
	assert.True(t, registeredPlugin.Enabled)
}

func TestDiscoverProviderResources(t *testing.T) {
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Timeout: 30 * time.Second,
		},
	}

	discoverer := NewEnhancedDiscoverer(cfg)
	ctx := context.Background()

	// Test with plugin
	plugin := &DiscoveryPlugin{
		Name:    "aws",
		Enabled: true,
		DiscoveryFn: func(ctx context.Context, provider, region string) ([]models.Resource, error) {
			return []models.Resource{
				{ID: "plugin-resource", Provider: provider, Region: region},
			}, nil
		},
	}
	discoverer.RegisterPlugin(plugin)

	resources, err := discoverer.discoverProviderResources(ctx, "aws", "us-east-1")
	require.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, "plugin-resource", resources[0].ID)

	// Test with unsupported provider
	_, err = discoverer.discoverProviderResources(ctx, "unsupported", "region")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

func TestConcurrentDiscovery(t *testing.T) {
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Timeout:          30 * time.Second,
			ConcurrencyLimit: 5,
			EnableCaching:    false,
			Regions:          []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
	}

	discoverer := NewEnhancedDiscoverer(cfg)
	
	// Track concurrent executions
	concurrentCalls := 0
	maxConcurrent := 0
	var mu sync.Mutex

	plugin := &DiscoveryPlugin{
		Name:    "aws",
		Enabled: true,
		DiscoveryFn: func(ctx context.Context, provider, region string) ([]models.Resource, error) {
			mu.Lock()
			concurrentCalls++
			if concurrentCalls > maxConcurrent {
				maxConcurrent = concurrentCalls
			}
			mu.Unlock()

			// Simulate work
			time.Sleep(100 * time.Millisecond)

			mu.Lock()
			concurrentCalls--
			mu.Unlock()

			return []models.Resource{
				{ID: region + "-resource", Region: region},
			}, nil
		},
	}
	discoverer.RegisterPlugin(plugin)

	ctx := context.Background()
	resources, err := discoverer.DiscoverResources(ctx)

	require.NoError(t, err)
	assert.Len(t, resources, 9) // 3 providers * 3 regions
	assert.LessOrEqual(t, maxConcurrent, cfg.Discovery.ConcurrencyLimit)
}

func BenchmarkDiscoverResources(b *testing.B) {
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Timeout:       30 * time.Second,
			EnableCaching: false,
			Regions:       []string{"us-east-1", "us-west-2"},
		},
	}

	discoverer := NewEnhancedDiscoverer(cfg)

	plugin := &DiscoveryPlugin{
		Name:    "aws",
		Enabled: true,
		DiscoveryFn: func(ctx context.Context, provider, region string) ([]models.Resource, error) {
			return []models.Resource{
				{ID: "resource-1"},
				{ID: "resource-2"},
			}, nil
		},
	}
	discoverer.RegisterPlugin(plugin)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = discoverer.DiscoverResources(ctx)
	}
}