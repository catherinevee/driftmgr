package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/catherinevee/driftmgr/internal/config"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
)

func main() {
	// Example 1: Basic Enhanced Discovery Setup
	exampleBasicEnhancedDiscovery()

	// Example 2: Plugin-Based Discovery
	examplePluginBasedDiscovery()

	// Example 3: Filtered Discovery
	exampleFilteredDiscovery()

	// Example 4: Hierarchical Discovery
	exampleHierarchicalDiscovery()

	// Example 5: Cached Discovery
	exampleCachedDiscovery()
}

// Example 1: Basic Enhanced Discovery Setup
func exampleBasicEnhancedDiscovery() {
	fmt.Println("=== Example 1: Basic Enhanced Discovery ===")

	// Create configuration
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			ConcurrencyLimit: 10,
			Timeout:          5 * time.Minute,
			RetryAttempts:    3,
			RetryDelay:       1 * time.Second,
			BatchSize:        100,
			EnableCaching:    true,
			CacheTTL:         1 * time.Hour,
			CacheMaxSize:     1000,
		},
	}

	// Create enhanced discoverer
	discoverer := discovery.NewEnhancedDiscoverer(cfg)

	// Discover resources across multiple providers and regions
	ctx := context.Background()
	providers := []string{"aws", "azure", "gcp"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}

	resources, err := discoverer.DiscoverAllResourcesEnhanced(ctx, providers, regions)
	if err != nil {
		log.Printf("Discovery failed: %v", err)
		return
	}

	fmt.Printf("Discovered %d resources\n", len(resources))

	// Get discovery quality metrics
	quality := discoverer.GetDiscoveryQuality()
	fmt.Printf("Discovery Quality - Completeness: %.2f%%, Accuracy: %.2f%%, Freshness: %v\n",
		quality.Completeness*100, quality.Accuracy*100, quality.Freshness)
}

// Example 2: Plugin-Based Discovery
func examplePluginBasedDiscovery() {
	fmt.Println("\n=== Example 2: Plugin-Based Discovery ===")

	// Create configuration
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			ConcurrencyLimit: 10,
			Timeout:          5 * time.Minute,
			EnableCaching:    true,
			CacheTTL:         1 * time.Hour,
			CacheMaxSize:     1000,
		},
	}

	// Create enhanced discoverer
	discoverer := discovery.NewEnhancedDiscoverer(cfg)

	// Create plugin loader
	pluginLoader := discovery.NewPluginLoader(discoverer)

	// Load plugins from configuration file
	err := pluginLoader.LoadPluginsFromFile("config/discovery-plugins.yaml")
	if err != nil {
		log.Printf("Failed to load plugins: %v", err)
		return
	}

	// Get enabled plugins for AWS
	awsPlugins := pluginLoader.GetEnabledPlugins("aws")
	fmt.Printf("Enabled AWS plugins: %d\n", len(awsPlugins))

	for _, plugin := range awsPlugins {
		fmt.Printf("  - %s (Priority: %d, Dependencies: %v)\n",
			plugin.Name, plugin.Priority, plugin.Dependencies)
	}

	// Discover resources using plugins
	ctx := context.Background()
	resources, err := discoverer.DiscoverAllResourcesEnhanced(ctx, []string{"aws"}, []string{"us-east-1"})
	if err != nil {
		log.Printf("Plugin-based discovery failed: %v", err)
		return
	}

	fmt.Printf("Plugin-based discovery found %d resources\n", len(resources))
}

// Example 3: Filtered Discovery
func exampleFilteredDiscovery() {
	fmt.Println("\n=== Example 3: Filtered Discovery ===")

	// Create configuration
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			ConcurrencyLimit: 10,
			Timeout:          5 * time.Minute,
			EnableCaching:    true,
			CacheTTL:         1 * time.Hour,
			CacheMaxSize:     1000,
		},
	}

	// Create enhanced discoverer
	discoverer := discovery.NewEnhancedDiscoverer(cfg)

	// Set up intelligent filtering
	filter := &discovery.DiscoveryFilter{
		IncludeTags: map[string]string{
			"Environment": "production",
			"Team":        "platform",
		},
		ExcludeTags: map[string]string{
			"Temporary": "true",
			"Test":      "true",
		},
		ResourceTypes: []string{
			"aws_instance",
			"aws_rds_instance",
			"aws_lambda_function",
		},
		AgeThreshold:  24 * time.Hour,
		CostThreshold: 100.0,
		SecurityScore: 5,
		Environment:   "production",
	}

	// Apply filter to discoverer
	discoverer.SetFilter(filter)

	// Discover resources with filtering
	ctx := context.Background()
	resources, err := discoverer.DiscoverAllResourcesEnhanced(ctx, []string{"aws"}, []string{"us-east-1"})
	if err != nil {
		log.Printf("Filtered discovery failed: %v", err)
		return
	}

	fmt.Printf("Filtered discovery found %d resources\n", len(resources))

	// Analyze filtered results
	resourceTypes := make(map[string]int)
	for _, resource := range resources {
		resourceTypes[resource.Type]++
	}

	fmt.Println("Resource types found:")
	for resourceType, count := range resourceTypes {
		fmt.Printf("  - %s: %d\n", resourceType, count)
	}
}

// Example 4: Hierarchical Discovery
func exampleHierarchicalDiscovery() {
	fmt.Println("\n=== Example 4: Hierarchical Discovery ===")

	// Create configuration
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			ConcurrencyLimit: 10,
			Timeout:          5 * time.Minute,
			EnableCaching:    true,
			CacheTTL:         1 * time.Hour,
			CacheMaxSize:     1000,
		},
	}

	// Create enhanced discoverer
	discoverer := discovery.NewEnhancedDiscoverer(cfg)

	// Discover resources (hierarchy is built automatically)
	ctx := context.Background()
	resources, err := discoverer.DiscoverAllResourcesEnhanced(ctx, []string{"aws"}, []string{"us-east-1"})
	if err != nil {
		log.Printf("Hierarchical discovery failed: %v", err)
		return
	}

	fmt.Printf("Hierarchical discovery found %d resources\n", len(resources))

	// The hierarchy is automatically built during discovery
	// You can access it through the discoverer's hierarchy field
	// This would typically be exposed through a method like GetHierarchy()

	// Example of how you might traverse the hierarchy:
	fmt.Println("Resource hierarchy example:")
	fmt.Println("  - VPC (Level 1)")
	fmt.Println("    - Subnet (Level 2)")
	fmt.Println("      - EC2 Instance (Level 3)")
	fmt.Println("    - Security Group (Level 2)")
	fmt.Println("      - RDS Instance (Level 3)")
}

// Example 5: Cached Discovery
func exampleCachedDiscovery() {
	fmt.Println("\n=== Example 5: Cached Discovery ===")

	// Create configuration with caching enabled
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			ConcurrencyLimit: 10,
			Timeout:          5 * time.Minute,
			EnableCaching:    true,
			CacheTTL:         1 * time.Hour,
			CacheMaxSize:     1000,
		},
	}

	// Create enhanced discoverer
	discoverer := discovery.NewEnhancedDiscoverer(cfg)

	// First discovery (will populate cache)
	ctx := context.Background()
	start := time.Now()
	resources1, err := discoverer.DiscoverAllResourcesEnhanced(ctx, []string{"aws"}, []string{"us-east-1"})
	if err != nil {
		log.Printf("First discovery failed: %v", err)
		return
	}
	firstDuration := time.Since(start)

	fmt.Printf("First discovery: %d resources in %v\n", len(resources1), firstDuration)

	// Second discovery (should use cache)
	start = time.Now()
	resources2, err := discoverer.DiscoverAllResourcesEnhanced(ctx, []string{"aws"}, []string{"us-east-1"})
	if err != nil {
		log.Printf("Second discovery failed: %v", err)
		return
	}
	secondDuration := time.Since(start)

	fmt.Printf("Second discovery: %d resources in %v\n", len(resources2), secondDuration)
	fmt.Printf("Cache speedup: %.2fx faster\n", float64(firstDuration)/float64(secondDuration))

	// Demonstrate cache statistics
	fmt.Println("Cache benefits:")
	fmt.Println("  - Faster subsequent discoveries")
	fmt.Println("  - Reduced API calls to cloud providers")
	fmt.Println("  - Consistent results within TTL window")
	fmt.Println("  - Automatic cache invalidation")
}

// Example 6: Advanced Discovery with All Features
func exampleAdvancedDiscovery() {
	fmt.Println("\n=== Example 6: Advanced Discovery with All Features ===")

	// Create comprehensive configuration
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			ConcurrencyLimit:     15,
			Timeout:              10 * time.Minute,
			RetryAttempts:        5,
			RetryDelay:           2 * time.Second,
			BatchSize:            200,
			EnableCaching:        true,
			CacheTTL:             2 * time.Hour,
			CacheMaxSize:         2000,
			MaxConcurrentRegions: 8,
			APITimeout:           45 * time.Second,
			QualityThresholds: config.QualityThresholds{
				Completeness: 0.85,
				Accuracy:     0.90,
				Freshness:    30 * time.Minute,
			},
			DefaultFilters: config.DiscoveryFilters{
				IncludeTags: map[string]string{
					"Environment": "production",
					"ManagedBy":   "terraform",
				},
				ExcludeTags: map[string]string{
					"Temporary": "true",
				},
				AgeThreshold:  48 * time.Hour,
				CostThreshold: 500.0,
				SecurityScore: 7,
				Environment:   "production",
			},
		},
	}

	// Create enhanced discoverer
	discoverer := discovery.NewEnhancedDiscoverer(cfg)

	// Load plugins
	pluginLoader := discovery.NewPluginLoader(discoverer)
	err := pluginLoader.LoadPluginsFromFile("config/discovery-plugins.yaml")
	if err != nil {
		log.Printf("Failed to load plugins: %v", err)
		return
	}

	// Set up advanced filtering
	advancedFilter := &discovery.DiscoveryFilter{
		IncludeTags: map[string]string{
			"Environment": "production",
			"Team":        "platform",
			"Critical":    "true",
		},
		ExcludeTags: map[string]string{
			"Temporary":  "true",
			"Test":       "true",
			"Deprecated": "true",
		},
		ResourceTypes: []string{
			"aws_instance",
			"aws_rds_instance",
			"aws_lambda_function",
			"aws_ecs_cluster",
			"aws_eks_cluster",
		},
		AgeThreshold:  24 * time.Hour,
		CostThreshold: 100.0,
		SecurityScore: 8,
		Environment:   "production",
	}

	discoverer.SetFilter(advancedFilter)

	// Perform comprehensive discovery
	ctx := context.Background()
	providers := []string{"aws", "azure", "gcp"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "eu-central-1"}

	start := time.Now()
	resources, err := discoverer.DiscoverAllResourcesEnhanced(ctx, providers, regions)
	if err != nil {
		log.Printf("Advanced discovery failed: %v", err)
		return
	}
	duration := time.Since(start)

	fmt.Printf("Advanced discovery completed in %v\n", duration)
	fmt.Printf("Total resources discovered: %d\n", len(resources))

	// Analyze results by provider
	providerStats := make(map[string]int)
	resourceTypeStats := make(map[string]int)

	for _, resource := range resources {
		providerStats[resource.Provider]++
		resourceTypeStats[resource.Type]++
	}

	fmt.Println("\nResources by provider:")
	for provider, count := range providerStats {
		fmt.Printf("  - %s: %d\n", provider, count)
	}

	fmt.Println("\nTop resource types:")
	// Sort resource types by count (simplified)
	for resourceType, count := range resourceTypeStats {
		if count > 5 {
			fmt.Printf("  - %s: %d\n", resourceType, count)
		}
	}

	// Get quality metrics
	quality := discoverer.GetDiscoveryQuality()
	fmt.Printf("\nDiscovery Quality Metrics:\n")
	fmt.Printf("  - Completeness: %.2f%%\n", quality.Completeness*100)
	fmt.Printf("  - Accuracy: %.2f%%\n", quality.Accuracy*100)
	fmt.Printf("  - Freshness: %v\n", quality.Freshness)
	fmt.Printf("  - Coverage by provider:\n")
	for provider, coverage := range quality.Coverage {
		fmt.Printf("    - %s: %.2f%%\n", provider, coverage*100)
	}
}

// Helper function to create sample resources for testing
func createSampleResources() []models.Resource {
	return []models.Resource{
		{
			ID:       "vpc-12345",
			Name:     "main-vpc",
			Type:     "aws_vpc",
			Provider: "aws",
			Region:   "us-east-1",
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
			},
			State:   "active",
			Created: time.Now().Add(-24 * time.Hour),
			Updated: time.Now(),
		},
		{
			ID:       "i-67890",
			Name:     "web-server-1",
			Type:     "aws_instance",
			Provider: "aws",
			Region:   "us-east-1",
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
				"VPC":         "vpc-12345",
			},
			State:   "running",
			Created: time.Now().Add(-12 * time.Hour),
			Updated: time.Now(),
		},
		{
			ID:       "db-abc123",
			Name:     "main-database",
			Type:     "aws_rds_instance",
			Provider: "aws",
			Region:   "us-east-1",
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
				"VPC":         "vpc-12345",
			},
			State:   "available",
			Created: time.Now().Add(-48 * time.Hour),
			Updated: time.Now(),
		},
	}
}
