package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/fatih/color"
)


// BulkDeleteOptions represents options for bulk deletion
type BulkDeleteOptions struct {
	Provider       string
	Region         string
	ResourceType   string
	Tags           map[string]string
	ResourceIDs    []string
	Force          bool
	DryRun         bool
	Parallel       bool
	MaxConcurrent  int
	IncludeDeps    bool
	Wait           bool
	Confirm        bool
}

// HandleBulkDelete handles bulk resource deletion
func HandleBulkDelete(args []string) {
	opts := &BulkDeleteOptions{
		Provider:      "aws",
		Region:        "us-east-1",
		MaxConcurrent: 5,
		Tags:          make(map[string]string),
		ResourceIDs:   []string{},
	}

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				opts.Provider = args[i+1]
				i++
			}
		case "--region", "-r":
			if i+1 < len(args) {
				opts.Region = args[i+1]
				i++
			}
		case "--type", "-t":
			if i+1 < len(args) {
				opts.ResourceType = args[i+1]
				i++
			}
		case "--tag":
			if i+1 < len(args) {
				// Parse tag in format key=value
				parts := strings.Split(args[i+1], "=")
				if len(parts) == 2 {
					opts.Tags[parts[0]] = parts[1]
				}
				i++
			}
		case "--id":
			if i+1 < len(args) {
				opts.ResourceIDs = append(opts.ResourceIDs, args[i+1])
				i++
			}
		case "--ids":
			if i+1 < len(args) {
				// Parse comma-separated IDs
				ids := strings.Split(args[i+1], ",")
				opts.ResourceIDs = append(opts.ResourceIDs, ids...)
				i++
			}
		case "--force":
			opts.Force = true
		case "--dry-run":
			opts.DryRun = true
		case "--parallel":
			opts.Parallel = true
		case "--max-concurrent":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.MaxConcurrent)
				i++
			}
		case "--include-deps":
			opts.IncludeDeps = true
		case "--wait":
			opts.Wait = true
		case "--yes", "-y":
			opts.Confirm = true
		case "--help", "-h":
			showBulkDeleteHelp()
			return
		}
	}

	// Execute bulk deletion
	if err := executeBulkDelete(opts); err != nil {
		color.Red("Error: %v\n", err)
		os.Exit(1)
	}
}

func showBulkDeleteHelp() {
	fmt.Println("Usage: driftmgr delete --bulk [options]")
	fmt.Println()
	fmt.Println("Bulk delete cloud resources with filtering and safety features")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --provider, -p       Cloud provider (aws, azure, gcp)")
	fmt.Println("  --region, -r         Region to delete from")
	fmt.Println("  --type, -t           Resource type filter")
	fmt.Println("  --tag                Tag filter (key=value, can be repeated)")
	fmt.Println("  --id                 Specific resource ID (can be repeated)")
	fmt.Println("  --ids                Comma-separated resource IDs")
	fmt.Println("  --force              Skip validation checks")
	fmt.Println("  --dry-run            Show what would be deleted")
	fmt.Println("  --parallel           Delete resources in parallel")
	fmt.Println("  --max-concurrent     Max concurrent deletions (default: 5)")
	fmt.Println("  --include-deps       Include dependent resources")
	fmt.Println("  --wait               Wait for deletion completion")
	fmt.Println("  --yes, -y            Skip confirmation prompt")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Delete all untagged EC2 instances in us-west-2")
	fmt.Println("  driftmgr delete --bulk --type ec2_instance --region us-west-2")
	fmt.Println()
	fmt.Println("  # Delete resources with specific tag")
	fmt.Println("  driftmgr delete --bulk --tag Environment=test --dry-run")
	fmt.Println()
	fmt.Println("  # Delete specific resources by ID")
	fmt.Println("  driftmgr delete --bulk --ids i-1234,i-5678,i-9012 --parallel")
	fmt.Println()
	fmt.Println("  # Delete all test resources with dependencies")
	fmt.Println("  driftmgr delete --bulk --tag Environment=test --include-deps --force")
}

func executeBulkDelete(opts *BulkDeleteOptions) error {
	ctx := context.Background()

	// Step 1: Discover resources to delete
	fmt.Println("Discovering resources...")
	resources, err := discoverResourcesToDelete(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to discover resources: %w", err)
	}

	if len(resources) == 0 {
		fmt.Println("No resources found matching criteria")
		return nil
	}

	// Step 2: Display resources to be deleted
	fmt.Printf("\nFound %d resources to delete:\n", len(resources))
	displayResources(resources)

	// Step 3: Check for dependencies if requested
	if opts.IncludeDeps {
		deps, err := findDependencies(ctx, resources, opts)
		if err != nil {
			return fmt.Errorf("failed to find dependencies: %w", err)
		}
		if len(deps) > 0 {
			fmt.Printf("\nFound %d dependent resources:\n", len(deps))
			displayResources(deps)
			resources = append(resources, deps...)
		}
	}

	// Step 4: Validate deletion safety
	if !opts.Force {
		if err := validateBulkDeletion(resources); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Step 5: Confirm deletion
	if !opts.Confirm && !opts.DryRun {
		fmt.Printf("\n%s Delete %d resources? (yes/no): ", 
			color.YellowString("WARNING:"), len(resources))
		
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))
		
		if response != "yes" && response != "y" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Step 6: Execute deletion
	if opts.DryRun {
		fmt.Println("\n[DRY RUN] Would delete the following resources:")
		for _, res := range resources {
			fmt.Printf("  - %s (%s) in %s\n", res.Name, res.Type, res.Region)
		}
		return nil
	}

	fmt.Println("\nDeleting resources...")
	results := performBulkDeletion(ctx, resources, opts)

	// Step 7: Display results
	displayDeletionResults(results)

	return nil
}

func discoverResourcesToDelete(ctx context.Context, opts *BulkDeleteOptions) ([]models.Resource, error) {
	// If specific IDs provided, fetch those resources
	if len(opts.ResourceIDs) > 0 {
		return fetchResourcesByIDs(ctx, opts)
	}

	// Otherwise, discover with filters
	// Create a parallel discoverer for resource discovery
	discoveryService := discovery.NewParallelDiscoverer(discovery.ParallelDiscoveryConfig{
		MaxWorkers:     10,
		MaxConcurrency: 5,
		Timeout:        5 * time.Minute,
	})

	// Discover resources using parallel discovery
	resources, err := discoveryService.DiscoverAllResources(ctx, []string{opts.Provider}, []string{opts.Region})
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := []models.Resource{}
	for _, res := range resources {
		// Filter by type
		if opts.ResourceType != "" && !matchesResourceType(res.Type, opts.ResourceType) {
			continue
		}

		// Filter by tags
		if len(opts.Tags) > 0 {
			// Type assert Tags to map[string]string
			if resTags, ok := res.Tags.(map[string]string); ok {
				if !matchesTags(resTags, opts.Tags) {
					continue
				}
			} else {
				// If Tags is not a map, skip this resource
				continue
			}
		}

		// Filter by region
		if opts.Region != "" && res.Region != opts.Region {
			continue
		}

		filtered = append(filtered, res)
	}

	return filtered, nil
}

func fetchResourcesByIDs(ctx context.Context, opts *BulkDeleteOptions) ([]models.Resource, error) {
	// This would fetch specific resources by ID
	// For now, create resources from IDs
	resources := []models.Resource{}
	for _, id := range opts.ResourceIDs {
		resources = append(resources, models.Resource{
			ID:       id,
			Name:     id,
			Type:     opts.ResourceType,
			Provider: opts.Provider,
			Region:   opts.Region,
		})
	}
	return resources, nil
}

func matchesResourceType(actualType, filterType string) bool {
	// Handle various type formats
	actualLower := strings.ToLower(actualType)
	filterLower := strings.ToLower(filterType)
	
	// Direct match
	if actualLower == filterLower {
		return true
	}
	
	// AWS resource type matching
	if strings.Contains(actualLower, filterLower) {
		return true
	}
	
	// Handle AWS::Service::Type format
	awsParts := strings.Split(actualType, "::")
	if len(awsParts) == 3 {
		resourceType := strings.ToLower(awsParts[2])
		if resourceType == filterLower {
			return true
		}
	}
	
	return false
}

func matchesTags(resourceTags, filterTags map[string]string) bool {
	// Check if all filter tags match
	for key, value := range filterTags {
		resValue, exists := resourceTags[key]
		if !exists || resValue != value {
			return false
		}
	}
	return true
}

func findDependencies(ctx context.Context, resources []models.Resource, opts *BulkDeleteOptions) ([]models.Resource, error) {
	// This would use the relationship mapper to find dependencies
	// For now, return empty list
	return []models.Resource{}, nil
}

func validateBulkDeletion(resources []models.Resource) error {
	// Check for production resources
	for _, res := range resources {
		// Check for production indicators
		if isProductionResource(res) {
			return fmt.Errorf("resource %s appears to be a production resource", res.Name)
		}
		
		// Check for critical resources
		if isCriticalResource(res) {
			return fmt.Errorf("resource %s is marked as critical", res.Name)
		}
	}
	
	return nil
}

func isProductionResource(res models.Resource) bool {
	// Check tags for production indicators
	prodIndicators := []string{"production", "prod", "live"}
	
	// Type assert Tags to map[string]string
	tags, ok := res.Tags.(map[string]string)
	if !ok {
		return false
	}
	
	for key, value := range tags {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)
		
		if keyLower == "environment" || keyLower == "env" {
			for _, indicator := range prodIndicators {
				if valueLower == indicator {
					return true
				}
			}
		}
	}
	
	// Check name for production indicators
	nameLower := strings.ToLower(res.Name)
	for _, indicator := range prodIndicators {
		if strings.Contains(nameLower, indicator) {
			return true
		}
	}
	
	return false
}

func isCriticalResource(res models.Resource) bool {
	// Check for critical resource types
	criticalTypes := []string{
		"rds_cluster",
		"documentdb_cluster",
		"elasticache_replication_group",
		"elasticsearch_domain",
		"kms_key",
		"route53_zone",
	}
	
	typeLower := strings.ToLower(res.Type)
	for _, critical := range criticalTypes {
		if strings.Contains(typeLower, critical) {
			return true
		}
	}
	
	// Check tags for critical indicators
	if res.Tags != nil {
		// Type assert Tags to map[string]string
		if tags, ok := res.Tags.(map[string]string); ok {
			if critical, exists := tags["Critical"]; exists && strings.ToLower(critical) == "true" {
				return true
			}
			if importance, exists := tags["Importance"]; exists && strings.ToLower(importance) == "critical" {
				return true
			}
		}
	}
	
	return false
}

type DeletionResult struct {
	Resource models.Resource
	Success  bool
	Error    error
	Duration time.Duration
}

func performBulkDeletion(ctx context.Context, resources []models.Resource, opts *BulkDeleteOptions) []DeletionResult {
	results := make([]DeletionResult, len(resources))
	
	if opts.Parallel {
		// Parallel deletion with concurrency limit
		semaphore := make(chan struct{}, opts.MaxConcurrent)
		var wg sync.WaitGroup
		
		for i, res := range resources {
			wg.Add(1)
			go func(idx int, resource models.Resource) {
				defer wg.Done()
				
				semaphore <- struct{}{} // Acquire
				defer func() { <-semaphore }() // Release
				
				start := time.Now()
				err := deleteResource(ctx, resource, opts)
				results[idx] = DeletionResult{
					Resource: resource,
					Success:  err == nil,
					Error:    err,
					Duration: time.Since(start),
				}
			}(i, res)
		}
		
		wg.Wait()
	} else {
		// Sequential deletion
		for i, res := range resources {
			start := time.Now()
			err := deleteResource(ctx, res, opts)
			results[i] = DeletionResult{
				Resource: res,
				Success:  err == nil,
				Error:    err,
				Duration: time.Since(start),
			}
			
			// Show progress
			fmt.Printf("  [%d/%d] %s %s... ", 
				i+1, len(resources), 
				res.Name,
				map[bool]string{true: "✓", false: "✗"}[err == nil])
			
			if err != nil {
				color.Red("Failed: %v\n", err)
			} else {
				color.Green("Success")
			}
		}
	}
	
	return results
}

func deleteResource(ctx context.Context, resource models.Resource, opts *BulkDeleteOptions) error {
	// Get the appropriate deletion provider
	switch resource.Provider {
	case "aws":
		provider, err := remediation.NewAWSProvider()
		if err != nil {
			return err
		}
		return provider.DeleteResource(ctx, resource)
		
	case "azure":
		provider, err := remediation.NewAzureProvider()
		if err != nil {
			return err
		}
		return provider.DeleteResource(ctx, resource)
		
	case "gcp":
		// Create GCP deletion provider
		provider, err := remediation.NewGCPProvider()
		if err != nil {
			return fmt.Errorf("failed to create GCP provider: %w", err)
		}
		return provider.DeleteResource(ctx, resource)
		
	default:
		return fmt.Errorf("unsupported provider: %s", resource.Provider)
	}
}

func displayResources(resources []models.Resource) {
	// Group by type
	byType := make(map[string][]models.Resource)
	for _, res := range resources {
		byType[res.Type] = append(byType[res.Type], res)
	}
	
	// Display grouped
	for resType, resList := range byType {
		fmt.Printf("\n  %s (%d):\n", resType, len(resList))
		for i, res := range resList {
			if i >= 5 && len(resList) > 10 {
				fmt.Printf("    ... and %d more\n", len(resList)-5)
				break
			}
			fmt.Printf("    - %s", res.Name)
			if res.Region != "" {
				fmt.Printf(" (%s)", res.Region)
			}
			fmt.Println()
		}
	}
}

func displayDeletionResults(results []DeletionResult) {
	successful := 0
	failed := 0
	totalDuration := time.Duration(0)
	
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}
	
	fmt.Println("\n=== Deletion Summary ===")
	color.Green("Successful: %d\n", successful)
	if failed > 0 {
		color.Red("Failed: %d\n", failed)
	}
	fmt.Printf("Total time: %s\n", totalDuration)
	
	if failed > 0 {
		fmt.Println("\nFailed deletions:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  - %s: %v\n", result.Resource.Name, result.Error)
			}
		}
	}
}

// ExportDeletionList exports the list of resources to be deleted to a file
func ExportDeletionList(resources []models.Resource, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(resources)
}

// ImportDeletionList imports a list of resources to delete from a file
func ImportDeletionList(filename string) ([]models.Resource, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var resources []models.Resource
	decoder := json.NewDecoder(file)
	
	if err := decoder.Decode(&resources); err != nil {
		return nil, err
	}
	
	return resources, nil
}