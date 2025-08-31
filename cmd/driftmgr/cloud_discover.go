package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/config"
	"github.com/catherinevee/driftmgr/internal/core/color"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/core/progress"
	"github.com/catherinevee/driftmgr/internal/credentials"
	"github.com/catherinevee/driftmgr/internal/services"
	"github.com/catherinevee/driftmgr/internal/utils/graceful"
)

// handleCloudDiscover handles cloud resource discovery using local credentials
func handleCloudDiscover(args []string) {
	ctx := context.Background()

	// Load configuration
	configPath := "configs/config.yaml"
	if envPath := os.Getenv("DRIFTMGR_CONFIG"); envPath != "" {
		configPath = envPath
	}
	configManager, err := config.NewManager(configPath)
	if err != nil {
		// Continue with defaults if config fails to load
		fmt.Printf("Warning: Failed to load configuration: %v\n", err)
	}

	// Parse provider from args with config defaults
	provider := ""
	outputFile := ""
	format := "summary"
	showCredentials := false
	autoDiscover := false
	allAccounts := false
	forceRefresh := false

	// Load defaults from config if available
	if configManager != nil {
		cfg := configManager.Get()
		if cfg != nil {
			if cfg.Provider != "" {
				provider = cfg.Provider
			}
			autoDiscover = cfg.Settings.AutoDiscovery
		}
	}

	for i, arg := range args {
		switch arg {
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
			}
		case "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
			}
		case "--show-credentials", "--credentials":
			showCredentials = true
		case "--auto":
			autoDiscover = true
		case "--all-accounts":
			allAccounts = true
		case "--force-refresh":
			forceRefresh = true
		case "aws", "azure", "gcp", "digitalocean", "all":
			provider = arg
		case "--help", "-h":
			showDiscoverHelp()
			return
		}
	}

	// Show credential status if requested
	if showCredentials {
		showCredentialStatus(ctx)
		return
	}

	// Initialize services
	serviceManager, err := services.NewManager(nil)
	if err != nil {
		graceful.HandleError(err, "Failed to create service manager")
		return
	}
	discoveryService := serviceManager.Discovery
	cacheManager := cache.GetGlobalCache()
	
	// Initialize discovery engine
	engine, err := discovery.NewEnhancedEngine()
	if err != nil {
		graceful.HandleError(err, "Failed to create discovery engine")
		return
	}

	// If auto-discover, detect all configured providers
	if autoDiscover {
		fmt.Println("Auto-discovering all configured cloud providers...")
		provider = "all"
	}

	// Perform discovery
	var resources []models.Resource

	if provider == "" || provider == "all" {
		// Show loading animation
		loadingAnim := progress.NewLoadingAnimation("Discovering resources across all providers")
		loadingAnim.Start()

		if allAccounts {
			fmt.Println("Including all accessible accounts/subscriptions...")
		}
		// Discover from all configured providers
		detector := credentials.NewCredentialDetector()
		creds := detector.DetectAll()

		// Count configured providers for progress bar
		configuredCount := 0
		for _, cred := range creds {
			if cred.Status == "configured" {
				configuredCount++
			}
		}

		loadingAnim.Stop()

		if configuredCount > 0 {
			bar := progress.NewBar(configuredCount, "Discovering providers")

			for _, cred := range creds {
				if cred.Status == "configured" {
					providerName := strings.ToLower(cred.Provider)

					// Show spinner for individual provider
					spinner := progress.NewSpinner(fmt.Sprintf("Scanning %s", cred.Provider))
					spinner.Start()

					// Use service layer for discovery
					req := services.DiscoveryRequest{
						Provider: providerName,
						Options: map[string]interface{}{
							"allAccounts": allAccounts,
						},
					}
					resp, err := discoveryService.StartDiscovery(ctx, req)
					if err != nil {
						spinner.Error(fmt.Sprintf("Failed to discover %s resources", providerName))
						// Try cache fallback
						if !forceRefresh {
							fallbackKey := fmt.Sprintf("discovery:%s:%t", providerName, allAccounts)
							if fallback, found, _ := cacheManager.GetWithAge(fallbackKey); found {
								if fallbackResources, ok := fallback.([]models.Resource); ok {
									color.Printf(color.Yellow, "Using cached fallback for %s\n", providerName)
									resources = append(resources, fallbackResources...)
								}
							}
						}
						continue
					}

					// Wait for discovery to complete if async
					var providerResources []models.Resource
					if resp.JobID != "" {
						providerResources = waitForDiscoveryJob(ctx, discoveryService, resp.JobID)
					} else if resp.Resources != nil {
						// Convert cloud.Resource to models.Resource
						for _, r := range resp.Resources {
							// Parse time strings
							var createdAt, updatedAt time.Time
							if r.CreatedAt != "" {
								createdAt, _ = time.Parse(time.RFC3339, r.CreatedAt)
							}
							if r.ModifiedAt != "" {
								updatedAt, _ = time.Parse(time.RFC3339, r.ModifiedAt)
							}
							
							providerResources = append(providerResources, models.Resource{
								ID:           r.ID,
								Name:         r.Name,
								Type:         r.Type,
								Provider:     r.Provider,
								Region:       r.Region,
								AccountID:    r.AccountID,
								Status:       r.Status,
								State:        r.State,
								CreatedAt:    createdAt,
								LastModified: updatedAt,
								Tags:         r.Tags,
								Properties:   r.Properties,
							})
						}
					}
					
					spinner.Success(fmt.Sprintf("Found %d resources in %s", len(providerResources), cred.Provider))
					resources = append(resources, providerResources...)
					
					// Cache the provider results
					providerCacheKey := fmt.Sprintf("discovery:%s:%t", providerName, allAccounts)
					cacheManager.Set(providerCacheKey, providerResources, 5*time.Minute)
					
					bar.Increment()
				}
			}
			bar.Complete()
		}
		
		// Cache the combined results
		if len(resources) > 0 {
			cacheKey := fmt.Sprintf("discovery:all:%t", allAccounts)
			cacheManager.Set(cacheKey, resources, 5*time.Minute)
		}
	} else {
		// Single provider discovery with spinner
		spinner := progress.NewBarSpinner(fmt.Sprintf("Discovering %s resources", provider))
		spinner.Start()

		if allAccounts {
			spinner.UpdateMessage(fmt.Sprintf("Discovering %s resources (all accounts)", provider))
		}

		config := discovery.Config{
			Provider: provider,
		}
		var err error
		resources, err = engine.Discover(config)
		if err != nil {
			spinner.Error(fmt.Sprintf("Failed to discover %s resources", provider))
			graceful.HandleError(err, "Discovery failed")
			return
		}

		spinner.Success(fmt.Sprintf("Discovered %d %s resources", len(resources), provider))
	}

	if len(resources) == 0 {
		fmt.Println("No resources found.")
		return
	}

	// Display or save results based on format
	switch format {
	case "json":
		displayResourcesJSON(resources, outputFile)
	case "table":
		displayResourcesTable(resources)
	case "summary":
		displayResourcesSummary(resources)
	case "terraform":
		displayResourcesTerraform(resources, outputFile)
	default:
		displayResourcesSummary(resources)
	}
}

// showCredentialStatus displays the status of configured credentials
func showCredentialStatus(ctx context.Context) {
	fmt.Println("Checking credential status...")

	// Use the credential detector
	detector := credentials.NewCredentialDetector()
	creds := detector.DetectAll()

	// Create a map for easy lookup
	credMap := make(map[string]bool)
	for _, cred := range creds {
		credMap[strings.ToLower(cred.Provider)] = true
	}

	// Check each provider
	providers := []string{"AWS", "Azure", "GCP", "DigitalOcean"}
	for _, provider := range providers {
		if credMap[strings.ToLower(provider)] {
			fmt.Printf("%s: Configured\n", provider)
		} else {
			fmt.Printf("%s: Not configured\n", provider)
		}
	}
}

// showCredentialHelp displays help for configuring credentials
func showCredentialHelp() {
	fmt.Println()
	fmt.Println("AWS:")
	fmt.Println("  Option 1: Set environment variables")
	fmt.Println("    export AWS_ACCESS_KEY_ID=your_access_key")
	fmt.Println("    export AWS_SECRET_ACCESS_KEY=your_secret_key")
	fmt.Println("    export AWS_REGION=us-east-1")
	fmt.Println()
	fmt.Println("  Option 2: Configure AWS CLI")
	fmt.Println("    aws configure")
	fmt.Println()
	fmt.Println("Azure:")
	fmt.Println("  Option 1: Set environment variables")
	fmt.Println("    export AZURE_SUBSCRIPTION_ID=your_subscription_id")
	fmt.Println("    export AZURE_TENANT_ID=your_tenant_id")
	fmt.Println("    export AZURE_CLIENT_ID=your_client_id")
	fmt.Println("    export AZURE_CLIENT_SECRET=your_client_secret")
	fmt.Println()
	fmt.Println("  Option 2: Use Azure CLI")
	fmt.Println("    az login")
	fmt.Println()
	fmt.Println("GCP:")
	fmt.Println("  Option 1: Set environment variables")
	fmt.Println("    export GCP_PROJECT_ID=your_project_id")
	fmt.Println("    export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json")
	fmt.Println()
	fmt.Println("  Option 2: Use gcloud CLI")
	fmt.Println("    gcloud auth application-default login")
	fmt.Println()
	fmt.Println("DigitalOcean:")
	fmt.Println("  Set environment variable")
	fmt.Println("    export DIGITALOCEAN_TOKEN=your_token")
	fmt.Println()
	fmt.Println("Or create ~/.driftmgr/credentials.yaml")
}

// showDiscoverHelp displays help for the discover command
func showDiscoverHelp() {
	fmt.Println("Usage: driftmgr discover [flags]")
	fmt.Println()
	fmt.Println("Discover cloud resources using local credentials with intelligent caching")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --auto              Auto-discover all configured providers")
	fmt.Println("  --all-accounts      Include all accessible accounts/subscriptions")
	fmt.Println("  --provider string   Specific provider (aws, azure, gcp, digitalocean)")
	fmt.Println("  --format string     Output format (summary, json, table, terraform)")
	fmt.Println("  --output string     Output file path")
	fmt.Println("  --show-credentials  Show credential status")
	fmt.Println()
	fmt.Println("Cache Flags:")
	fmt.Println("  --force-refresh     Force live discovery, bypass cache")
	fmt.Println("  --no-cache          Alias for --force-refresh")
	fmt.Println("  --cache-status      Show cache status for discovery data")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  driftmgr discover --auto")
	fmt.Println("  driftmgr discover --provider aws --all-accounts")
	fmt.Println("  driftmgr discover --format json --output resources.json")
	fmt.Println("  driftmgr discover --force-refresh  # Force live data fetch")
	fmt.Println("  driftmgr discover --cache-status   # View cache status")
	fmt.Println()
	fmt.Println("Multi-Account Support:")
	fmt.Println("  AWS:   Discovers resources across all profiles in ~/.aws/credentials")
	fmt.Println("  Azure: Discovers resources across all accessible subscriptions")
	fmt.Println("  GCP:   Discovers resources across all accessible projects")
	fmt.Println()
	fmt.Println("Caching:")
	fmt.Println("  - Discovery results are cached for 5 minutes")
	fmt.Println("  - Cache is automatically used when available")
	fmt.Println("  - Failed API calls fall back to cached data if available")
	fmt.Println("  - Data source and timestamp are displayed with results")
}

// displayResourcesSummary displays a summary of discovered resources
func displayResourcesSummary(resources []models.Resource) {
	// Group resources by provider
	providerCount := make(map[string]int)
	typeCount := make(map[string]int)

	for _, resource := range resources {
		providerCount[resource.Provider]++
		typeCount[resource.Type]++
	}

	fmt.Printf("\n%s\n", color.Header("=== Discovery Summary ==="))
	fmt.Printf("%s %s\n\n", color.Label("Total Resources:"), color.Count(len(resources)))

	fmt.Println(color.Subheader("By Provider:"))
	for provider, count := range providerCount {
		var providerColor string
		switch strings.ToLower(provider) {
		case "aws":
			providerColor = color.AWS(provider)
		case "azure":
			providerColor = color.Azure(provider)
		case "gcp":
			providerColor = color.GCP(provider)
		case "digitalocean":
			providerColor = color.DigitalOcean(provider)
		default:
			providerColor = provider
		}
		fmt.Printf("  %s: %s\n", providerColor, color.Count(count))
	}

	fmt.Printf("\n%s\n", color.Subheader("Top Resource Types:"))
	displayed := 0
	for resType, count := range typeCount {
		if displayed >= 10 {
			break
		}
		fmt.Printf("  %s: %d\n", resType, count)
		displayed++
	}
}

// displayResourcesTable displays resources in table format
func displayResourcesTable(resources []models.Resource) {
	fmt.Println("\n=== Discovered Resources ===")
	fmt.Printf("%-15s %-30s %-40s %-15s %-10s\n", "Provider", "Type", "Name", "Region", "State")
	fmt.Println(strings.Repeat("-", 120))

	for _, resource := range resources {
		name := resource.Name
		if len(name) > 38 {
			name = name[:35] + "..."
		}
		fmt.Printf("%-15s %-30s %-40s %-15s %-10s\n",
			resource.Provider,
			resource.Type,
			name,
			resource.Region,
			resource.State,
		)
	}
}

// displayResourcesJSON displays or saves resources as JSON
func displayResourcesJSON(resources []models.Resource, outputFile string) {
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		graceful.HandleError(err, "Failed to marshal resources")
		return
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			graceful.HandleError(err, "Failed to write output file")
			return
		}
		fmt.Printf("Results saved to %s\n", outputFile)
	} else {
		fmt.Println(string(data))
	}
}

// displayResourcesTerraform displays resources in Terraform import format
func displayResourcesTerraform(resources []models.Resource, outputFile string) {
	var output strings.Builder

	output.WriteString("# Terraform import commands for discovered resources\n")
	output.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	// Group by provider
	byProvider := make(map[string][]models.Resource)
	for _, resource := range resources {
		byProvider[resource.Provider] = append(byProvider[resource.Provider], resource)
	}

	for provider, providerResources := range byProvider {
		output.WriteString(fmt.Sprintf("# %s Resources (%d)\n", strings.ToUpper(provider), len(providerResources)))

		for _, resource := range providerResources {
			// Generate terraform import command
			resourceAddr := fmt.Sprintf("%s.%s", resource.Type, strings.ReplaceAll(resource.Name, "-", "_"))
			output.WriteString(fmt.Sprintf("terraform import %s %s\n", resourceAddr, resource.ID))
		}
		output.WriteString("\n")
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
			graceful.HandleError(err, "Failed to write output file")
			return
		}
		fmt.Printf("Terraform import commands saved to %s\n", outputFile)
	} else {
		fmt.Print(output.String())
	}
}

// waitForDiscoveryJob waits for an async discovery job to complete
func waitForDiscoveryJob(ctx context.Context, service *services.DiscoveryService, jobID string) []models.Resource {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return []models.Resource{}
		case <-ticker.C:
			status, err := service.GetDiscoveryStatus(ctx, jobID)
			if err != nil {
				continue
			}
			if status.Status == "completed" {
				if status.Resources != nil {
					// Convert cloud.Resource to models.Resource
					var resources []models.Resource
					for _, r := range status.Resources {
						// Parse time strings
						var createdAt, updatedAt time.Time
						if r.CreatedAt != "" {
							createdAt, _ = time.Parse(time.RFC3339, r.CreatedAt)
						}
						if r.ModifiedAt != "" {
							updatedAt, _ = time.Parse(time.RFC3339, r.ModifiedAt)
						}
						
						resources = append(resources, models.Resource{
							ID:           r.ID,
							Name:         r.Name,
							Type:         r.Type,
							Provider:     r.Provider,
							Region:       r.Region,
							AccountID:    r.AccountID,
							Status:       r.Status,
							State:        r.State,
							CreatedAt:    createdAt,
							LastModified: updatedAt,
							Tags:         r.Tags,
							Properties:   r.Properties,
						})
					}
					return resources
				}
				return []models.Resource{}
			}
			if status.Status == "failed" {
				return []models.Resource{}
			}
		}
	}
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// showDiscoverCacheStatus displays cache status for discovery data
func showDiscoverCacheStatus() {
	cacheManager := cache.GetGlobalCache()
	
	fmt.Println("Discovery Cache Status:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("%-30s %-15s %-20s %-10s\n", "Provider", "Status", "Last Updated", "Size")
	fmt.Println("───────────────────────────────────────────────────────────────")
	
	providers := []string{"aws", "azure", "gcp", "digitalocean", "all"}
	for _, provider := range providers {
		for _, allAccounts := range []bool{false, true} {
			cacheKey := fmt.Sprintf("discovery:%s:%t", provider, allAccounts)
			if data, found, age := cacheManager.GetWithAge(cacheKey); found {
				var size int
				if resources, ok := data.([]models.Resource); ok {
					size = len(resources)
				}
				
				accountStr := ""
				if allAccounts {
					accountStr = " (all accounts)"
				}
				
				fmt.Printf("%-30s %-15s %-20s %-10d\n", 
					provider+accountStr,
					"cached",
					formatDuration(age)+" ago",
					size)
			}
		}
	}
	fmt.Println("═══════════════════════════════════════════════════════════════")
	
	// Show cache configuration
	fmt.Println("\nCache Configuration:")
	fmt.Println("  TTL: 5 minutes")
	fmt.Println("  Auto-refresh: Disabled")
	fmt.Println("  Fallback on error: Enabled")
	fmt.Println("\nUse --force-refresh to bypass cache and fetch live data.")
}
