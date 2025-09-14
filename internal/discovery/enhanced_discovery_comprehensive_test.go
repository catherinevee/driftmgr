package discovery

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/shared/config"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestEnhancedDiscoverer_NewEnhancedDiscoverer(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected bool
	}{
		{
			name:     "With basic config",
			config:   &config.Config{},
			expected: true,
		},
		{
			name:     "With discovery config",
			config:   &config.Config{Discovery: config.DiscoveryConfig{}},
			expected: true,
		},
		{
			name:     "With nil config",
			config:   nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoverer := NewEnhancedDiscoverer(tt.config)
			assert.NotNil(t, discoverer)
			assert.Equal(t, tt.config, discoverer.config)
			assert.NotNil(t, discoverer.cache)
			assert.NotNil(t, discoverer.plugins)
			assert.NotNil(t, discoverer.hierarchy)
			assert.NotNil(t, discoverer.filters)
			assert.NotNil(t, discoverer.progressTracker)
			assert.NotNil(t, discoverer.sdkIntegration)
			assert.NotNil(t, discoverer.discoveredResources)
			assert.NotNil(t, discoverer.metrics)
		})
	}
}

func TestEnhancedDiscoverer_Plugins(t *testing.T) {
	discoverer := NewEnhancedDiscoverer(&config.Config{})

	// Test plugin registration
	plugin := &DiscoveryPlugin{
		Name: "test-plugin",
		DiscoveryFn: func(ctx context.Context, provider, region string) ([]models.Resource, error) {
			return []models.Resource{}, nil
		},
	}

	discoverer.RegisterPlugin(plugin)
	assert.Contains(t, discoverer.plugins, "test-plugin")
	assert.Equal(t, plugin, discoverer.plugins["test-plugin"])
}

func TestEnhancedDiscoverer_DiscoverResources(t *testing.T) {
	// Skip this test as it makes actual cloud provider calls
	t.Skip("Skipping test that makes actual cloud provider calls")
}

func TestResourceHierarchy(t *testing.T) {
	parentResource := &models.Resource{
		ID:   "root",
		Type: "root",
		Name: "Root Resource",
	}

	hierarchy := &ResourceHierarchy{
		Parent:       parentResource,
		Children:     []*ResourceHierarchy{},
		Dependencies: []string{},
		Level:        0,
	}

	// Test adding a child resource
	childResource := &models.Resource{
		ID:   "child",
		Type: "child",
		Name: "Child Resource",
	}

	child := &ResourceHierarchy{
		Parent:       childResource,
		Children:     []*ResourceHierarchy{},
		Dependencies: []string{},
		Level:        1,
	}

	hierarchy.Children = append(hierarchy.Children, child)

	assert.Len(t, hierarchy.Children, 1)
	assert.Equal(t, "child", hierarchy.Children[0].Parent.ID)
}

func TestDiscoveryFilter_Structure(t *testing.T) {
	filter := &DiscoveryFilter{
		ResourceTypes: []string{"aws_instance", "aws_s3_bucket"},
		IncludeTags: map[string]string{
			"Environment": "production",
		},
		ExcludeTags: map[string]string{
			"Owner": "test",
		},
		AgeThreshold:  24 * time.Hour,
		UsagePatterns: []string{"production", "critical"},
		CostThreshold: 100.0,
		SecurityScore: 80,
		Environment:   "production",
	}

	// Test filter structure
	assert.Len(t, filter.ResourceTypes, 2)
	assert.Contains(t, filter.ResourceTypes, "aws_instance")
	assert.Contains(t, filter.ResourceTypes, "aws_s3_bucket")
	assert.Equal(t, "production", filter.IncludeTags["Environment"])
	assert.Equal(t, "test", filter.ExcludeTags["Owner"])
	assert.Equal(t, 24*time.Hour, filter.AgeThreshold)
	assert.Equal(t, 100.0, filter.CostThreshold)
	assert.Equal(t, 80, filter.SecurityScore)
	assert.Equal(t, "production", filter.Environment)
}

func TestProgressTracker(t *testing.T) {
	providers := []string{"aws", "azure", "gcp"}
	regions := []string{"us-east-1", "us-west-2"}
	services := []string{"EC2", "S3"}

	tracker := NewProgressTracker(providers, regions, services)

	// Test initial state
	progress := tracker.GetProgress()
	assert.NotNil(t, progress)
	assert.False(t, tracker.IsCompleted())

	// Test updating progress
	tracker.UpdateProviderProgress("aws", 1, 10)
	tracker.UpdateRegionProgress("aws", "us-east-1", 1, 5)
	tracker.UpdateServiceProgress("aws", "us-east-1", "EC2", 3)

	// Test getting progress
	providerProgress := tracker.GetProviderProgress("aws")
	assert.NotNil(t, providerProgress)
	assert.Equal(t, "aws", providerProgress.Name)

	regionProgress := tracker.GetRegionProgress("aws", "us-east-1")
	assert.NotNil(t, regionProgress)
	assert.Equal(t, "us-east-1", regionProgress.Region)

	serviceProgress := tracker.GetServiceProgress("aws", "us-east-1", "EC2")
	assert.NotNil(t, serviceProgress)
	assert.Equal(t, "EC2", serviceProgress.Service)

	// Test completion
	tracker.MarkCompleted()
	assert.True(t, tracker.IsCompleted())
}

func TestDiscoveryVisualizer(t *testing.T) {
	visualizer := NewDiscoveryVisualizer()

	// Test adding resources
	resource := models.Resource{
		ID:     "test-resource",
		Type:   "aws_instance",
		Name:   "Test Instance",
		Region: "us-east-1",
	}

	visualizer.AddResource(resource)
	assert.Len(t, visualizer.resources, 1)
	assert.Equal(t, resource, visualizer.resources[0])

	// Test adding relationships
	visualizer.AddRelationship("test-resource", "test-vpc")
	assert.Contains(t, visualizer.relationships, "test-resource")
	assert.Contains(t, visualizer.relationships["test-resource"], "test-vpc")

	// Test generating JSON
	jsonData := visualizer.GenerateJSON()
	assert.NotEmpty(t, jsonData)

	// Test generating CSV
	csvData := visualizer.GenerateCSV()
	assert.NotEmpty(t, csvData)

	// Test getting statistics
	stats := visualizer.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats.TotalResources)
}

func TestAdvancedQuery(t *testing.T) {
	query := NewAdvancedQuery()

	// Test adding resources
	resource := models.Resource{
		ID:     "test-resource",
		Type:   "aws_instance",
		Name:   "Test Instance",
		Region: "us-east-1",
		Tags: map[string]string{
			"Environment": "production",
		},
	}

	query.AddResource(resource)

	// Test querying by tags
	resources := query.FindByTags(map[string]string{"Environment": "production"})
	assert.Len(t, resources, 1)
	assert.Equal(t, "test-resource", resources[0].ID)

	// Test querying by regex
	resources = query.FindByRegex("Type", "aws_.*")
	assert.Len(t, resources, 1)
	assert.Equal(t, "aws_instance", resources[0].Type)

	// Test grouping
	grouped := query.GroupBy("Type")
	assert.Contains(t, grouped, "aws_instance")
	assert.Len(t, grouped["aws_instance"], 1)

	// Test getting statistics
	stats := query.GetStatistics()
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats["total_resources"])
}

func TestRealTimeMonitor(t *testing.T) {
	monitor := NewRealTimeMonitor()

	// Test starting monitoring
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	monitor.Start(ctx)

	// Test processing resource
	resource := models.Resource{
		ID:     "test-resource",
		Type:   "aws_instance",
		Name:   "Test Instance",
		Region: "us-east-1",
	}

	monitor.ProcessResource(resource)

	// Test getting metrics
	metrics := monitor.GetMetrics()
	assert.NotNil(t, metrics)
}

func TestSDKIntegration(t *testing.T) {
	integration := NewSDKIntegration()

	// Test initialization
	assert.NotNil(t, integration)
	assert.NotNil(t, integration.providers)
	assert.NotNil(t, integration.rateLimiters)
	assert.NotNil(t, integration.retryPolicies)
	assert.NotNil(t, integration.credentials)
	assert.NotNil(t, integration.clientCache)
	assert.NotNil(t, integration.metrics)

	// Test setting credentials (will fail because provider not registered)
	creds := Credentials{
		Provider:  "aws",
		AccessKey: "access-key",
		SecretKey: "secret-key",
		Region:    "us-east-1",
	}
	err := integration.SetCredentials("aws", creds)
	// This will fail because no provider is registered, which is expected
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider aws not registered")

	// Test getting metrics
	metrics := integration.GetMetrics()
	assert.NotNil(t, metrics)
}

func TestEnhancedDiscoverer_GetDiscoveryQuality(t *testing.T) {
	discoverer := NewEnhancedDiscoverer(&config.Config{})

	// Test initial quality
	quality := discoverer.GetDiscoveryQuality()
	assert.NotNil(t, quality)
	assert.Equal(t, 0.0, quality.Completeness)
	assert.Equal(t, 0.0, quality.Accuracy)
	assert.NotNil(t, quality.Coverage)

	// Test with some discovered resources
	discoverer.discoveredResources = []models.Resource{
		{ID: "resource-1", Type: "aws_instance"},
		{ID: "resource-2", Type: "aws_s3_bucket"},
	}

	quality = discoverer.GetDiscoveryQuality()
	assert.Greater(t, quality.Completeness, 0.0)
	assert.NotNil(t, quality.Coverage)
}

func TestResource_GetTagsAsMap(t *testing.T) {
	resource := models.Resource{
		ID:     "test-resource",
		Type:   "aws_instance",
		Name:   "Test Instance",
		Region: "us-east-1",
		Tags: map[string]string{
			"Environment": "production",
			"Owner":       "team-a",
		},
		Attributes: map[string]interface{}{
			"instance_type": "t3.micro",
			"ami":           "ami-12345678",
		},
	}

	// Test getting tags as map
	tags := resource.GetTagsAsMap()
	assert.Equal(t, "production", tags["Environment"])
	assert.Equal(t, "team-a", tags["Owner"])
}

func TestResourceCache_Operations(t *testing.T) {
	cache := &ResourceCache{data: make(map[string]interface{})}

	// Test setting and getting
	cache.Set("key1", "value1")
	val, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)

	// Test getting non-existent key
	val, found = cache.Get("nonexistent")
	assert.False(t, found)
	assert.Nil(t, val)

	// Test getting size
	size := cache.GetSize()
	assert.Equal(t, 1, size)
}

func TestResourceCache_Concurrent(t *testing.T) {
	cache := &ResourceCache{data: make(map[string]interface{})}

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := fmt.Sprintf("key%d", n)
			value := fmt.Sprintf("value%d", n)
			cache.Set(key, value)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all values were set
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		expectedValue := fmt.Sprintf("value%d", i)
		val, found := cache.Get(key)
		assert.True(t, found)
		assert.Equal(t, expectedValue, val)
	}
}
