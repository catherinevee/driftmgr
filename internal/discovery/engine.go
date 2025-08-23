package discovery

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Engine handles resource discovery across multiple cloud providers
type Engine struct {
	providers map[string]Provider
}

// Config holds configuration for resource discovery
type Config struct {
	Provider     string
	Regions      []string
	ResourceType string
	Tags         []string
	OutputFormat string
	OutputFile   string
}

// Provider interface for cloud provider implementations
type Provider interface {
	Discover(config Config) ([]models.Resource, error)
	Name() string
	SupportedRegions() []string
	SupportedResourceTypes() []string
}

// NewEngine creates a new discovery engine with retry and caching support
func NewEngine() (*Engine, error) {
	// Try to use enhanced engine first
	enhanced, err := NewEnhancedEngine()
	if err == nil {
		// Return the enhanced engine wrapped as regular engine
		return &Engine{
			providers: enhanced.providers,
		}, nil
	}

	// Fallback to basic initialization
	providers := make(map[string]Provider)

	// Try to initialize AWS provider
	awsProvider, err := NewAWSProvider()
	if err == nil {
		providers["aws"] = awsProvider
	}

	// Try to initialize Azure provider
	azureProvider, err := NewAzureProvider()
	if err == nil {
		providers["azure"] = azureProvider
	}

	// Try to initialize GCP provider
	gcpProvider, err := NewGCPProvider()
	if err == nil {
		providers["gcp"] = gcpProvider
	}

	// Try to initialize DigitalOcean provider
	if CheckDigitalOceanCredentials() {
		doProvider, err := NewDigitalOceanProvider()
		if err == nil {
			providers["digitalocean"] = doProvider
		}
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no cloud providers could be initialized")
	}

	return &Engine{
		providers: providers,
	}, nil
}

// Discover resources using the specified configuration
func (e *Engine) Discover(config Config) ([]models.Resource, error) {
	provider, exists := e.providers[config.Provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	fmt.Printf("🔍 Discovering resources with %s provider...\n", provider.Name())

	resources, err := provider.Discover(config)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Apply post-discovery filtering
	filtered := e.applyFilters(resources, config)

	return filtered, nil
}

// OutputResources formats and outputs the discovered resources
func (e *Engine) OutputResources(resources []models.Resource, config Config) error {
	switch config.OutputFormat {
	case "json":
		return e.outputJSON(resources, config.OutputFile)
	case "csv":
		return e.outputCSV(resources, config.OutputFile)
	case "table":
		return e.outputTable(resources)
	default:
		return fmt.Errorf("unsupported output format: %s", config.OutputFormat)
	}
}

// applyFilters applies additional filtering to discovered resources
func (e *Engine) applyFilters(resources []models.Resource, config Config) []models.Resource {
	var filtered []models.Resource

	for _, resource := range resources {
		// Filter by resource type if specified
		if config.ResourceType != "" && resource.Type != config.ResourceType {
			continue
		}

		// Filter by tags if specified
		if len(config.Tags) > 0 && !e.matchesTags(resource, config.Tags) {
			continue
		}

		filtered = append(filtered, resource)
	}

	return filtered
}

// matchesTags checks if a resource matches the specified tag filters
func (e *Engine) matchesTags(resource models.Resource, tagFilters []string) bool {
	for _, filter := range tagFilters {
		parts := strings.Split(filter, ":")
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		tags := resource.GetTagsAsMap()
		resourceValue, exists := tags[key]
		if !exists || resourceValue != value {
			return false
		}
	}
	return true
}

// outputJSON outputs resources in JSON format
func (e *Engine) outputJSON(resources []models.Resource, outputFile string) error {
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if outputFile != "" {
		return os.WriteFile(outputFile, data, 0644)
	}

	fmt.Println(string(data))
	return nil
}

// outputCSV outputs resources in CSV format
func (e *Engine) outputCSV(resources []models.Resource, outputFile string) error {
	var writer *csv.Writer

	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer file.Close()
		writer = csv.NewWriter(file)
	} else {
		writer = csv.NewWriter(os.Stdout)
	}
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Name", "Type", "Provider", "Region", "Tags", "CreatedAt"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	for _, resource := range resources {
		tags, _ := json.Marshal(resource.Tags)
		record := []string{
			resource.ID,
			resource.Name,
			resource.Type,
			resource.Provider,
			resource.Region,
			string(tags),
			resource.CreatedAt.Format(time.RFC3339),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// outputTable outputs resources in table format
func (e *Engine) outputTable(resources []models.Resource) error {
	fmt.Printf("\n%-20s %-30s %-25s %-10s %-15s %-10s\n",
		"ID", "NAME", "TYPE", "PROVIDER", "REGION", "TAGS")
	fmt.Println(strings.Repeat("-", 110))

	for _, resource := range resources {
		tagStr := ""
		resourceTags := resource.GetTagsAsMap()
		if len(resourceTags) > 0 {
			var tags []string
			for k, v := range resourceTags {
				tags = append(tags, fmt.Sprintf("%s:%s", k, v))
			}
			tagStr = strings.Join(tags, ",")
			if len(tagStr) > 10 {
				tagStr = tagStr[:7] + "..."
			}
		}

		name := resource.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		resourceType := resource.Type
		if len(resourceType) > 25 {
			resourceType = resourceType[:22] + "..."
		}

		fmt.Printf("%-20s %-30s %-25s %-10s %-15s %-10s\n",
			resource.ID, name, resourceType, resource.Provider, resource.Region, tagStr)
	}

	fmt.Printf("\nTotal: %d resources\n", len(resources))
	return nil
}
