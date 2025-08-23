package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/credentials"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/utils/graceful"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// handleCloudDiscover handles cloud resource discovery using local credentials
func handleCloudDiscover(args []string) {
	ctx := context.Background()

	// Parse provider from args
	provider := ""
	outputFile := ""
	format := "summary"
	showCredentials := false
	autoDiscover := false
	allAccounts := false

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

	// Initialize discovery engine
	engine, err := discovery.NewEngine()
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
		fmt.Println("Discovering resources across all configured providers...")
		if allAccounts {
			fmt.Println("Including all accessible accounts/subscriptions...")
		}
		// Discover from all configured providers
		detector := credentials.NewCredentialDetector()
		creds := detector.DetectAll()

		for _, cred := range creds {
			if cred.Status == "configured" {
				providerName := strings.ToLower(cred.Provider)
				config := discovery.Config{
					Provider:     providerName,
					OutputFormat: format,
				}
				providerResources, err := engine.Discover(config)
				if err != nil {
					fmt.Printf("Warning: Failed to discover %s resources: %v\n", providerName, err)
					continue
				}
				resources = append(resources, providerResources...)
			}
		}
	} else {
		fmt.Printf("Discovering %s resources...\n", provider)
		if allAccounts {
			fmt.Println("Including all accessible accounts/subscriptions...")
		}
		config := discovery.Config{
			Provider:     provider,
			OutputFormat: format,
		}
		var err error
		resources, err = engine.Discover(config)
		if err != nil {
			graceful.HandleError(err, "Discovery failed")
			return
		}
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
	fmt.Println("Discover cloud resources using local credentials")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --auto              Auto-discover all configured providers")
	fmt.Println("  --all-accounts      Include all accessible accounts/subscriptions")
	fmt.Println("  --provider string   Specific provider (aws, azure, gcp, digitalocean)")
	fmt.Println("  --format string     Output format (summary, json, table, terraform)")
	fmt.Println("  --output string     Output file path")
	fmt.Println("  --show-credentials  Show credential status")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  driftmgr discover --auto")
	fmt.Println("  driftmgr discover --provider aws --all-accounts")
	fmt.Println("  driftmgr discover --format json --output resources.json")
	fmt.Println()
	fmt.Println("Multi-Account Support:")
	fmt.Println("  AWS:   Discovers resources across all profiles in ~/.aws/credentials")
	fmt.Println("  Azure: Discovers resources across all accessible subscriptions")
	fmt.Println("  GCP:   Discovers resources across all accessible projects")
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

	fmt.Println("\n=== Discovery Summary ===")
	fmt.Printf("Total Resources: %d\n\n", len(resources))

	fmt.Println("By Provider:")
	for provider, count := range providerCount {
		fmt.Printf("  %s: %d\n", provider, count)
	}

	fmt.Println("\nTop Resource Types:")
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
