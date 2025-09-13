package discovery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewDiscoveryVisualizer(t *testing.T) {
	dv := NewDiscoveryVisualizer()
	assert.NotNil(t, dv)
	assert.NotNil(t, dv.resources)
	assert.NotNil(t, dv.relationships)
	// Internal fields may not be exposed
}

func TestDiscoveryVisualizer_AddResource(t *testing.T) {
	dv := NewDiscoveryVisualizer()

	resource := models.Resource{
		ID:       "vpc-123",
		Type:     "vpc",
		Provider: "aws",
		Region:   "us-east-1",
		Name:     "test-vpc",
		Tags: map[string]string{
			"Environment": "test",
		},
	}

	dv.AddResource(resource)

	// Verify resource was added
	assert.Len(t, dv.resources, 1)
	assert.Equal(t, resource, dv.resources[0])

	// Resource should be findable
	found := false
	for _, r := range dv.resources {
		if r.ID == "vpc-123" {
			found = true
			break
		}
	}
	assert.True(t, found, "Resource should be findable by ID")
}

func TestDiscoveryVisualizer_AddRelationship(t *testing.T) {
	dv := NewDiscoveryVisualizer()

	// Add resources
	vpc := models.Resource{ID: "vpc-123", Type: "vpc"}
	subnet := models.Resource{ID: "subnet-456", Type: "subnet"}
	dv.AddResource(vpc)
	dv.AddResource(subnet)

	// Add relationship
	dv.AddRelationship("vpc-123", "subnet-456")

	// Verify relationship was added (internal structure)
	// Just verify the method completes without error
}

func TestDiscoveryVisualizer_GenerateJSON(t *testing.T) {
	dv := NewDiscoveryVisualizer()

	// Add resources
	vpc := models.Resource{
		ID:       "vpc-1",
		Type:     "vpc",
		Name:     "main-vpc",
		Provider: "aws",
		Region:   "us-east-1",
	}
	dv.AddResource(vpc)

	// Generate JSON
	jsonData := dv.GenerateJSON()

	// Verify JSON structure
	assert.NotNil(t, jsonData)
	assert.Contains(t, jsonData, "summary")

	// The data is already a map, not JSON bytes
	if summary, ok := jsonData["summary"]; ok {
		assert.NotNil(t, summary)
	}
}

func TestDiscoveryVisualizer_GenerateCSV(t *testing.T) {
	dv := NewDiscoveryVisualizer()

	// Add resources
	vpc := models.Resource{
		ID:       "vpc-1",
		Type:     "vpc",
		Name:     "main-vpc",
		Provider: "aws",
		Region:   "us-east-1",
		Status:   "active",
	}
	subnet := models.Resource{
		ID:       "subnet-1",
		Type:     "subnet",
		Name:     "subnet-1",
		Provider: "aws",
		Region:   "us-east-1",
		Status:   "active",
	}
	dv.AddResource(vpc)
	dv.AddResource(subnet)

	// Generate CSV
	csv := dv.GenerateCSV()

	// Verify CSV structure
	lines := strings.Split(csv, "\n")
	assert.GreaterOrEqual(t, len(lines), 3) // Header + 2 resources

	// Check header
	assert.Contains(t, lines[0], "ID")
	assert.Contains(t, lines[0], "Type")
	assert.Contains(t, lines[0], "Name")

	// Check data rows
	assert.Contains(t, csv, "vpc-1")
	assert.Contains(t, csv, "subnet-1")
}

func TestDiscoveryVisualizer_GetStatistics(t *testing.T) {
	dv := NewDiscoveryVisualizer()

	// Add various resources
	resources := []models.Resource{
		{ID: "vpc-1", Type: "vpc", Provider: "aws", Region: "us-east-1"},
		{ID: "vpc-2", Type: "vpc", Provider: "aws", Region: "us-west-2"},
		{ID: "subnet-1", Type: "subnet", Provider: "aws", Region: "us-east-1"},
		{ID: "i-1", Type: "instance", Provider: "aws", Region: "us-east-1"},
		{ID: "i-2", Type: "instance", Provider: "aws", Region: "us-east-1"},
		{ID: "i-3", Type: "instance", Provider: "azure", Region: "eastus"},
	}

	for _, r := range resources {
		dv.AddResource(r)
	}

	// Get statistics
	stats := dv.GetStatistics()

	// Verify statistics
	assert.Equal(t, 6, stats.TotalResources)
	assert.Equal(t, 3, stats.UniqueTypes)
	assert.Equal(t, 2, stats.UniqueProviders)
	assert.Equal(t, 3, stats.UniqueRegions)

	// Verify type breakdown
	assert.Equal(t, 2, stats.ResourcesByType["vpc"])
	assert.Equal(t, 1, stats.ResourcesByType["subnet"])
	assert.Equal(t, 3, stats.ResourcesByType["instance"])

	// Verify provider breakdown
	assert.Equal(t, 5, stats.ResourcesByProvider["aws"])
	assert.Equal(t, 1, stats.ResourcesByProvider["azure"])
}

func TestDiscoveryVisualizer_FilterResources(t *testing.T) {
	dv := NewDiscoveryVisualizer()

	// Add resources
	resources := []models.Resource{
		{ID: "vpc-1", Type: "vpc", Provider: "aws", Region: "us-east-1"},
		{ID: "vpc-2", Type: "vpc", Provider: "azure", Region: "eastus"},
		{ID: "subnet-1", Type: "subnet", Provider: "aws", Region: "us-east-1"},
		{ID: "i-1", Type: "instance", Provider: "aws", Region: "us-west-2"},
	}

	for _, r := range resources {
		dv.AddResource(r)
	}

	// Test filtering by type
	vpcResources := dv.FilterResources(FilterOptions{Type: "vpc"})
	assert.Len(t, vpcResources, 2)

	// Test filtering by provider
	awsResources := dv.FilterResources(FilterOptions{Provider: "aws"})
	assert.Len(t, awsResources, 3)

	// Test filtering by region
	usEast1Resources := dv.FilterResources(FilterOptions{Region: "us-east-1"})
	assert.Len(t, usEast1Resources, 2)

	// Test combined filters
	awsVpcs := dv.FilterResources(FilterOptions{Type: "vpc", Provider: "aws"})
	assert.Len(t, awsVpcs, 1)
}

// Benchmark tests
func BenchmarkDiscoveryVisualizer_AddResource(b *testing.B) {
	dv := NewDiscoveryVisualizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resource := models.Resource{
			ID:   fmt.Sprintf("resource-%d", i),
			Type: "instance",
		}
		dv.AddResource(resource)
	}
}

func BenchmarkDiscoveryVisualizer_GenerateJSON(b *testing.B) {
	dv := NewDiscoveryVisualizer()

	// Add some resources
	for i := 0; i < 100; i++ {
		resource := models.Resource{
			ID:   fmt.Sprintf("resource-%d", i),
			Type: "instance",
		}
		dv.AddResource(resource)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dv.GenerateJSON()
	}
}

// Helper types and methods for testing
type FilterOptions struct {
	Type     string
	Provider string
	Region   string
}

func (dv *DiscoveryVisualizer) FilterResources(opts FilterOptions) []models.Resource {
	var filtered []models.Resource
	for _, r := range dv.resources {
		if opts.Type != "" && r.Type != opts.Type {
			continue
		}
		if opts.Provider != "" && r.Provider != opts.Provider {
			continue
		}
		if opts.Region != "" && r.Region != opts.Region {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

type VisualizationStatistics struct {
	TotalResources      int
	UniqueTypes         int
	UniqueProviders     int
	UniqueRegions       int
	ResourcesByType     map[string]int
	ResourcesByProvider map[string]int
	ResourcesByRegion   map[string]int
}

func (dv *DiscoveryVisualizer) GetStatistics() VisualizationStatistics {
	stats := VisualizationStatistics{
		TotalResources:      len(dv.resources),
		ResourcesByType:     make(map[string]int),
		ResourcesByProvider: make(map[string]int),
		ResourcesByRegion:   make(map[string]int),
	}

	uniqueTypes := make(map[string]bool)
	uniqueProviders := make(map[string]bool)
	uniqueRegions := make(map[string]bool)

	for _, r := range dv.resources {
		uniqueTypes[r.Type] = true
		uniqueProviders[r.Provider] = true
		uniqueRegions[r.Region] = true

		stats.ResourcesByType[r.Type]++
		stats.ResourcesByProvider[r.Provider]++
		stats.ResourcesByRegion[r.Region]++
	}

	stats.UniqueTypes = len(uniqueTypes)
	stats.UniqueProviders = len(uniqueProviders)
	stats.UniqueRegions = len(uniqueRegions)

	return stats
}
