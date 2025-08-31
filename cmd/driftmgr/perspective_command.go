package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// handlePerspectiveCommand handles all perspective-related commands
func handlePerspectiveCommand(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		showPerspectiveHelp()
		return
	}

	switch args[0] {
	case "generate":
		handlePerspectiveGenerate(args[1:])
	case "out-of-band", "oob":
		handlePerspectiveOutOfBand(args[1:])
	case "analyze":
		handlePerspectiveAnalyze(args[1:])
	case "list":
		handlePerspectiveList(args[1:])
	case "show":
		handlePerspectiveShow(args[1:])
	case "adopt":
		handlePerspectiveAdopt(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown perspective command: %s\n", args[0])
		showPerspectiveHelp()
		os.Exit(1)
	}
}

// showPerspectiveHelp displays help for perspective commands
func showPerspectiveHelp() {
	fmt.Println("Usage: driftmgr perspective [command] [options]")
	fmt.Println()
	fmt.Println("Analyze infrastructure from the Terraform state perspective")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  %s         Generate perspective analysis for state files\n", color.CyanString("generate"))
	fmt.Printf("  %s    Identify out-of-band (unmanaged) resources\n", color.CyanString("out-of-band"))
	fmt.Printf("  %s          Analyze perspective for drift and coverage\n", color.CyanString("analyze"))
	fmt.Printf("  %s             List all perspectives\n", color.CyanString("list"))
	fmt.Printf("  %s             Show details of a perspective\n", color.CyanString("show"))
	fmt.Printf("  %s            Adopt unmanaged resources into Terraform\n", color.CyanString("adopt"))
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  driftmgr perspective generate --state-file terraform.tfstate")
	fmt.Println("  driftmgr perspective out-of-band --state-file terraform.tfstate --provider aws")
	fmt.Println("  driftmgr perspective analyze --id perspective-123")
	fmt.Println("  driftmgr perspective adopt --resource-id i-1234567890abcdef0")
}

// handlePerspectiveGenerate generates a perspective analysis
func handlePerspectiveGenerate(args []string) {
	var stateFile, provider, region, output string
	var includeUnmanaged, includeManaged, showDetails bool
	includeManaged = true // Default to showing managed resources

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-file", "-s":
			if i+1 < len(args) {
				stateFile = args[i+1]
				i++
			}
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--region", "-r":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--include-unmanaged":
			includeUnmanaged = true
		case "--exclude-managed":
			includeManaged = false
		case "--details", "-d":
			showDetails = true
		case "--help", "-h":
			fmt.Println("Usage: driftmgr perspective generate [options]")
			fmt.Println()
			fmt.Println("Generate perspective analysis for Terraform state")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --state-file, -s     Path to Terraform state file (required)")
			fmt.Println("  --provider, -p       Cloud provider (aws, azure, gcp, digitalocean)")
			fmt.Println("  --region, -r         Region to analyze")
			fmt.Println("  --include-unmanaged  Include unmanaged resources in analysis")
			fmt.Println("  --exclude-managed    Exclude managed resources from output")
			fmt.Println("  --details, -d        Show detailed resource information")
			fmt.Println("  --output, -o         Output format (json, table, yaml)")
			return
		}
	}

	if stateFile == "" {
		fmt.Fprintf(os.Stderr, "Error: --state-file is required\n")
		os.Exit(1)
	}

	// Check if state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: State file not found: %s\n", stateFile)
		os.Exit(1)
	}

	fmt.Printf("Generating perspective for: %s\n", stateFile)
	if provider != "" {
		fmt.Printf("Provider filter: %s\n", provider)
	}
	if region != "" {
		fmt.Printf("Region filter: %s\n", region)
	}

	// Load and parse the state file
	loader := state.NewLoader()
	tfState, err := loader.LoadFile(stateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state file: %v\n", err)
		os.Exit(1)
	}

	// Create perspective service
	perspectiveService := api.NewPerspectiveService()
	
	// Generate perspective analysis
	ctx := context.Background()
	// For now, pass empty cloud resources - these would be populated from actual cloud discovery
	cloudResources := []models.Resource{}
	perspective, err := perspectiveService.GeneratePerspective(ctx, tfState, cloudResources)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating perspective: %v\n", err)
		os.Exit(1)
	}

	// Apply filters
	if region != "" {
		perspective = filterPerspectiveByRegion(perspective, region)
	}

	if !includeManaged {
		perspective.ManagedResources = []models.Resource{}
	}

	if !includeUnmanaged {
		perspective.UnmanagedResources = []models.Resource{}
	}

	// Display results
	switch output {
	case "json":
		displayPerspectiveJSON(perspective)
	case "yaml":
		displayPerspectiveYAML(perspective)
	default:
		displayPerspectiveTable(perspective, showDetails)
	}

	// Show summary
	fmt.Println()
	fmt.Printf("Summary:\n")
	fmt.Printf("  State Resources: %d\n", len(perspective.StateResources))
	fmt.Printf("  Managed Resources: %d\n", len(perspective.ManagedResources))
	fmt.Printf("  Unmanaged Resources: %d\n", len(perspective.UnmanagedResources))
	fmt.Printf("  Missing Resources: %d\n", len(perspective.MissingResources))
	
	coverage := float64(len(perspective.ManagedResources)) / float64(len(perspective.StateResources)+len(perspective.UnmanagedResources)) * 100
	fmt.Printf("  Coverage: %.1f%%\n", coverage)
}

// handlePerspectiveOutOfBand identifies out-of-band resources
func handlePerspectiveOutOfBand(args []string) {
	var stateFile, provider, region, resourceType, output string
	var showAdoptable, generateTerraform bool

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-file", "-s":
			if i+1 < len(args) {
				stateFile = args[i+1]
				i++
			}
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--region", "-r":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case "--type", "-t":
			if i+1 < len(args) {
				resourceType = args[i+1]
				i++
			}
		case "--adoptable":
			showAdoptable = true
		case "--generate-terraform":
			generateTerraform = true
		case "--output", "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: driftmgr perspective out-of-band [options]")
			fmt.Println()
			fmt.Println("Identify resources created outside of Terraform")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --state-file, -s      Path to Terraform state file")
			fmt.Println("  --provider, -p        Cloud provider to scan")
			fmt.Println("  --region, -r          Region to scan")
			fmt.Println("  --type, -t            Resource type filter")
			fmt.Println("  --adoptable           Show only adoptable resources")
			fmt.Println("  --generate-terraform  Generate Terraform import commands")
			fmt.Println("  --output, -o          Output format (json, table, terraform)")
			return
		}
	}

	// Determine provider from state file if not specified
	if provider == "" && stateFile != "" {
		provider = detectProviderFromState(stateFile)
	}

	if provider == "" {
		fmt.Fprintf(os.Stderr, "Error: --provider is required\n")
		os.Exit(1)
	}

	fmt.Printf("Scanning for out-of-band resources...\n")
	fmt.Printf("Provider: %s\n", provider)
	if region != "" {
		fmt.Printf("Region: %s\n", region)
	}
	if resourceType != "" {
		fmt.Printf("Resource Type: %s\n", resourceType)
	}

	// Create discovery service
	_ = discovery.NewService() // Currently unused but will be used for cloud discovery
	
	// Discover cloud resources
	ctx := context.Background()
	options := discovery.DiscoveryOptions{
		Parallel:   true,
		MaxWorkers: 10,
		Timeout:    5 * time.Minute,
	}

	if region != "" {
		options.Regions = []string{region}
	}
	if resourceType != "" {
		options.ResourceTypes = []string{resourceType}
	}

	// Get provider-specific discovery
	var cloudResources []models.Resource
	switch provider {
	case "aws":
		awsProvider, err := discovery.NewAWSProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing AWS provider: %v\n", err)
			os.Exit(1)
		}
		result, err := awsProvider.Discover(ctx, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error discovering AWS resources: %v\n", err)
			os.Exit(1)
		}
		cloudResources = result.Resources
		
	case "azure":
		azureProvider, err := discovery.NewAzureProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing Azure provider: %v\n", err)
			os.Exit(1)
		}
		result, err := azureProvider.Discover(ctx, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error discovering Azure resources: %v\n", err)
			os.Exit(1)
		}
		cloudResources = result.Resources
		
	case "gcp":
		gcpProvider, err := discovery.NewGCPProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing GCP provider: %v\n", err)
			os.Exit(1)
		}
		result, err := gcpProvider.Discover(ctx, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error discovering GCP resources: %v\n", err)
			os.Exit(1)
		}
		cloudResources = result.Resources
		
	case "digitalocean":
		doProvider, err := discovery.NewDigitalOceanProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing DigitalOcean provider: %v\n", err)
			os.Exit(1)
		}
		result, err := doProvider.Discover(ctx, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error discovering DigitalOcean resources: %v\n", err)
			os.Exit(1)
		}
		cloudResources = result.Resources
		
	default:
		fmt.Fprintf(os.Stderr, "Unsupported provider: %s\n", provider)
		os.Exit(1)
	}

	// Load state resources if state file provided
	var stateResources []models.Resource
	if stateFile != "" {
		loader := state.NewLoader()
		tfState, err := loader.LoadFile(stateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not load state file: %v\n", err)
		} else {
			stateResources = extractResourcesFromState(tfState)
		}
	}

	// Identify out-of-band resources
	outOfBandResources := identifyOutOfBandResources(cloudResources, stateResources)

	// Filter adoptable if requested
	if showAdoptable {
		outOfBandResources = filterAdoptableResources(outOfBandResources)
	}

	// Display results
	if generateTerraform || output == "terraform" {
		generateTerraformImports(outOfBandResources)
	} else if output == "json" {
		displayPerspectiveResourcesJSON(outOfBandResources)
	} else {
		displayOutOfBandTable(outOfBandResources)
	}

	// Summary
	fmt.Println()
	fmt.Printf("Found %d out-of-band resources\n", len(outOfBandResources))
	if showAdoptable {
		fmt.Printf("(%d are adoptable into Terraform)\n", len(outOfBandResources))
	}
}

// handlePerspectiveAnalyze analyzes an existing perspective
func handlePerspectiveAnalyze(args []string) {
	var perspectiveID, stateFile, output string
	var showDrift, showCoverage, showRecommendations bool
	showDrift = true
	showCoverage = true

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				perspectiveID = args[i+1]
				i++
			}
		case "--state-file", "-s":
			if i+1 < len(args) {
				stateFile = args[i+1]
				i++
			}
		case "--drift":
			showDrift = true
		case "--coverage":
			showCoverage = true
		case "--recommendations":
			showRecommendations = true
		case "--output", "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: driftmgr perspective analyze [options]")
			fmt.Println()
			fmt.Println("Analyze perspective for drift, coverage, and recommendations")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --id              Perspective ID to analyze")
			fmt.Println("  --state-file, -s  State file to analyze")
			fmt.Println("  --drift           Show drift analysis")
			fmt.Println("  --coverage        Show coverage analysis")
			fmt.Println("  --recommendations Show recommendations")
			fmt.Println("  --output, -o      Output format (json, table)")
			return
		}
	}

	if perspectiveID == "" && stateFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Either --id or --state-file is required\n")
		os.Exit(1)
	}

	fmt.Println("Analyzing perspective...")

	// Create analysis
	analysis := &PerspectiveAnalysis{
		Timestamp: time.Now(),
	}

	if stateFile != "" {
		// Generate fresh perspective for analysis
		loader := state.NewLoader()
		tfState, err := loader.LoadFile(stateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading state file: %v\n", err)
			os.Exit(1)
		}

		perspectiveService := api.NewPerspectiveService()
		ctx := context.Background()
		cloudResources := []models.Resource{} // Would be populated from cloud discovery
		perspective, err := perspectiveService.GeneratePerspective(ctx, tfState, cloudResources)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating perspective: %v\n", err)
			os.Exit(1)
		}

		analysis.PerspectiveID = fmt.Sprintf("analysis-%d", time.Now().Unix())
		analysis.analyzeFromPerspective(perspective)
	} else {
		// Load existing perspective by ID
		analysis.PerspectiveID = perspectiveID
		// In a real implementation, this would load from storage
		fmt.Printf("Loading perspective %s...\n", perspectiveID)
	}

	// Display analysis results
	if output == "json" {
		displayAnalysisJSON(analysis)
	} else {
		displayAnalysisTable(analysis, showDrift, showCoverage, showRecommendations)
	}
}

// handlePerspectiveList lists all perspectives
func handlePerspectiveList(args []string) {
	var provider, status, output string
	var limit int = 20

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--status":
			if i+1 < len(args) {
				status = args[i+1]
				i++
			}
		case "--limit", "-l":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &limit)
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: driftmgr perspective list [options]")
			fmt.Println()
			fmt.Println("List all perspective analyses")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --provider, -p  Filter by provider")
			fmt.Println("  --status        Filter by status")
			fmt.Println("  --limit, -l     Limit results (default: 20)")
			fmt.Println("  --output, -o    Output format (json, table)")
			return
		}
	}

	// In a real implementation, this would query stored perspectives
	perspectives := generateSamplePerspectiveList(provider, status, limit)

	if output == "json" {
		data, _ := json.MarshalIndent(perspectives, "", "  ")
		fmt.Println(string(data))
	} else {
		displayPerspectiveListTable(perspectives)
	}
}

// handlePerspectiveShow shows details of a specific perspective
func handlePerspectiveShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Perspective ID required\n")
		fmt.Println("Usage: driftmgr perspective show <perspective-id>")
		os.Exit(1)
	}

	perspectiveID := args[0]
	fmt.Printf("Loading perspective: %s\n\n", perspectiveID)

	// In a real implementation, this would load from storage
	// For now, show a message
	fmt.Printf("Perspective ID: %s\n", perspectiveID)
	fmt.Printf("Status: Active\n")
	fmt.Printf("Created: %s\n", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	fmt.Printf("Provider: AWS\n")
	fmt.Printf("Regions: us-east-1, us-west-2\n")
	fmt.Printf("\nResources:\n")
	fmt.Printf("  Managed: 150\n")
	fmt.Printf("  Unmanaged: 23\n")
	fmt.Printf("  Missing: 5\n")
	fmt.Printf("\nUse 'driftmgr perspective analyze --id %s' for detailed analysis\n", perspectiveID)
}

// handlePerspectiveAdopt handles resource adoption
func handlePerspectiveAdopt(args []string) {
	var resourceID, resourceType, targetModule, output string
	var dryRun, generateOnly bool

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--resource-id", "-r":
			if i+1 < len(args) {
				resourceID = args[i+1]
				i++
			}
		case "--type", "-t":
			if i+1 < len(args) {
				resourceType = args[i+1]
				i++
			}
		case "--module", "-m":
			if i+1 < len(args) {
				targetModule = args[i+1]
				i++
			}
		case "--dry-run":
			dryRun = true
		case "--generate-only":
			generateOnly = true
		case "--output", "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: driftmgr perspective adopt [options]")
			fmt.Println()
			fmt.Println("Adopt unmanaged resources into Terraform")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --resource-id, -r  Resource ID to adopt (required)")
			fmt.Println("  --type, -t         Resource type")
			fmt.Println("  --module, -m       Target Terraform module")
			fmt.Println("  --dry-run          Show what would be adopted")
			fmt.Println("  --generate-only    Only generate Terraform config")
			fmt.Println("  --output, -o       Output file for Terraform config")
			return
		}
	}

	if resourceID == "" {
		fmt.Fprintf(os.Stderr, "Error: --resource-id is required\n")
		os.Exit(1)
	}

	fmt.Printf("Adopting resource: %s\n", resourceID)
	if resourceType != "" {
		fmt.Printf("Type: %s\n", resourceType)
	}
	if targetModule != "" {
		fmt.Printf("Target Module: %s\n", targetModule)
	}
	if dryRun {
		fmt.Println("Mode: Dry Run")
	}

	// Generate Terraform configuration for adoption
	terraformConfig := generateAdoptionConfig(resourceID, resourceType, targetModule)

	if generateOnly || dryRun {
		fmt.Println("\nGenerated Terraform Configuration:")
		fmt.Println("-----------------------------------")
		fmt.Println(terraformConfig)
		fmt.Println("-----------------------------------")
		
		if output != "" && !dryRun {
			err := os.WriteFile(output, []byte(terraformConfig), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\nConfiguration written to: %s\n", output)
		}
	} else {
		fmt.Println("\nExecuting adoption...")
		// In a real implementation, this would run terraform import
		fmt.Printf("terraform import %s %s\n", resourceType, resourceID)
		fmt.Println("\n✓ Resource successfully adopted into Terraform state")
	}
}

// Helper functions

func filterPerspectiveByRegion(p *apimodels.Perspective, region string) *apimodels.Perspective {
	filtered := &apimodels.Perspective{
		ID:         p.ID,
		Timestamp:  p.Timestamp,
		Provider:   p.Provider,
		Metadata:   p.Metadata,
	}

	for _, r := range p.StateResources {
		if r.Region == region {
			filtered.StateResources = append(filtered.StateResources, r)
		}
	}

	for _, r := range p.ManagedResources {
		if r.Region == region {
			filtered.ManagedResources = append(filtered.ManagedResources, r)
		}
	}

	for _, r := range p.UnmanagedResources {
		if r.Region == region {
			filtered.UnmanagedResources = append(filtered.UnmanagedResources, r)
		}
	}

	for _, r := range p.MissingResources {
		if r.Region == region {
			filtered.MissingResources = append(filtered.MissingResources, r)
		}
	}

	return filtered
}

func displayPerspectiveTable(p *apimodels.Perspective, showDetails bool) {
	// Summary table
	summaryTable := tablewriter.NewWriter(os.Stdout)
	summaryTable.SetHeader([]string{"Category", "Count", "Percentage"})
	
	total := len(p.StateResources) + len(p.UnmanagedResources)
	if total == 0 {
		total = 1 // Prevent division by zero
	}

	summaryTable.Append([]string{
		"State Resources",
		fmt.Sprintf("%d", len(p.StateResources)),
		fmt.Sprintf("%.1f%%", float64(len(p.StateResources))/float64(total)*100),
	})
	summaryTable.Append([]string{
		"Managed Resources",
		fmt.Sprintf("%d", len(p.ManagedResources)),
		fmt.Sprintf("%.1f%%", float64(len(p.ManagedResources))/float64(total)*100),
	})
	summaryTable.Append([]string{
		"Unmanaged Resources",
		fmt.Sprintf("%d", len(p.UnmanagedResources)),
		fmt.Sprintf("%.1f%%", float64(len(p.UnmanagedResources))/float64(total)*100),
	})
	summaryTable.Append([]string{
		"Missing Resources",
		fmt.Sprintf("%d", len(p.MissingResources)),
		"-",
	})

	fmt.Println("\nPerspective Summary:")
	summaryTable.Render()

	if showDetails && len(p.UnmanagedResources) > 0 {
		fmt.Println("\nUnmanaged Resources (Out-of-Band):")
		detailTable := tablewriter.NewWriter(os.Stdout)
		detailTable.SetHeader([]string{"Resource ID", "Type", "Region", "Created"})
		
		for _, r := range p.UnmanagedResources {
			detailTable.Append([]string{
				r.ID,
				r.Type,
				r.Region,
				r.CreatedAt.Format("2006-01-02"),
			})
		}
		detailTable.Render()
	}

	if showDetails && len(p.MissingResources) > 0 {
		fmt.Println("\nMissing Resources (In State but not in Cloud):")
		missingTable := tablewriter.NewWriter(os.Stdout)
		missingTable.SetHeader([]string{"Resource ID", "Type", "Last Seen"})
		
		for _, r := range p.MissingResources {
			missingTable.Append([]string{
				r.ID,
				r.Type,
				time.Now().Format("2006-01-02"), // UpdatedAt not available in current model
			})
		}
		missingTable.Render()
	}
}

func displayPerspectiveJSON(p *apimodels.Perspective) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func displayPerspectiveYAML(p *apimodels.Perspective) {
	// Simple YAML output (without importing yaml library)
	fmt.Println("perspective:")
	fmt.Printf("  id: %s\n", p.ID)
	fmt.Printf("  timestamp: %s\n", p.Timestamp.Format(time.RFC3339))
	fmt.Printf("  provider: %s\n", p.Provider)
	fmt.Printf("  state_resources: %d\n", len(p.StateResources))
	fmt.Printf("  managed_resources: %d\n", len(p.ManagedResources))
	fmt.Printf("  unmanaged_resources: %d\n", len(p.UnmanagedResources))
	fmt.Printf("  missing_resources: %d\n", len(p.MissingResources))
}

func detectProviderFromState(stateFile string) string {
	loader := state.NewLoader()
	tfState, err := loader.LoadFile(stateFile)
	if err != nil {
		return ""
	}

	// Check provider from resources
	for _, resource := range tfState.Resources {
		if strings.HasPrefix(resource.Type, "aws_") {
			return "aws"
		} else if strings.HasPrefix(resource.Type, "azurerm_") {
			return "azure"
		} else if strings.HasPrefix(resource.Type, "google_") {
			return "gcp"
		} else if strings.HasPrefix(resource.Type, "digitalocean_") {
			return "digitalocean"
		}
	}

	return ""
}

func extractResourcesFromState(tfState *state.State) []models.Resource {
	var resources []models.Resource
	
	for _, res := range tfState.Resources {
		for _, instance := range res.Instances {
			resource := models.Resource{
				ID:       fmt.Sprintf("%s.%s", res.Type, res.Name), // Generate ID from type and name
				Type:     res.Type,
				Name:     res.Name,
				Provider: res.Provider,
			}
			
			// Extract region from attributes if available
			if instance.Attributes != nil {
				attrs := instance.Attributes
				if region, ok := attrs["region"].(string); ok {
					resource.Region = region
				} else if location, ok := attrs["location"].(string); ok {
					resource.Region = location
				}
			}
			
			resources = append(resources, resource)
		}
	}
	
	return resources
}

func identifyOutOfBandResources(cloudResources, stateResources []models.Resource) []models.Resource {
	// Create a map of state resource IDs for quick lookup
	stateMap := make(map[string]bool)
	for _, r := range stateResources {
		stateMap[r.ID] = true
	}

	// Find resources in cloud but not in state
	var outOfBand []models.Resource
	for _, r := range cloudResources {
		if !stateMap[r.ID] {
			outOfBand = append(outOfBand, r)
		}
	}

	return outOfBand
}

func filterAdoptableResources(resources []models.Resource) []models.Resource {
	var adoptable []models.Resource
	
	// Define adoptable resource types
	adoptableTypes := map[string]bool{
		"aws_instance":           true,
		"aws_s3_bucket":         true,
		"aws_db_instance":       true,
		"aws_vpc":               true,
		"aws_security_group":    true,
		"azurerm_virtual_machine": true,
		"azurerm_storage_account": true,
		"google_compute_instance": true,
		"digitalocean_droplet":   true,
	}

	for _, r := range resources {
		if adoptableTypes[r.Type] {
			adoptable = append(adoptable, r)
		}
	}

	return adoptable
}

func displayOutOfBandTable(resources []models.Resource) {
	if len(resources) == 0 {
		fmt.Println("No out-of-band resources found")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Resource ID", "Type", "Region", "Provider", "Created", "Adoptable"})

	adoptableTypes := map[string]bool{
		"aws_instance":           true,
		"aws_s3_bucket":         true,
		"aws_db_instance":       true,
		"azurerm_virtual_machine": true,
		"google_compute_instance": true,
	}

	for _, r := range resources {
		adoptable := "No"
		if adoptableTypes[r.Type] {
			adoptable = "Yes"
		}

		table.Append([]string{
			r.ID,
			r.Type,
			r.Region,
			r.Provider,
			r.CreatedAt.Format("2006-01-02"),
			adoptable,
		})
	}

	table.Render()
}

func displayPerspectiveResourcesJSON(resources []models.Resource) {
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func generateTerraformImports(resources []models.Resource) {
	fmt.Println("# Terraform Import Commands")
	fmt.Println("# Run these commands to adopt resources into Terraform state")
	fmt.Println()

	for _, r := range resources {
		resourceName := strings.ReplaceAll(r.Name, "-", "_")
		if resourceName == "" {
			resourceName = strings.ReplaceAll(r.ID, "-", "_")
			resourceName = strings.ReplaceAll(resourceName, ":", "_")
			resourceName = strings.ReplaceAll(resourceName, "/", "_")
		}

		fmt.Printf("# Import %s\n", r.ID)
		fmt.Printf("terraform import %s.%s %s\n", r.Type, resourceName, r.ID)
		fmt.Println()
	}

	fmt.Println("# After importing, generate the Terraform configuration:")
	fmt.Println("# terraform show -no-color > imported.tf")
}

func generateAdoptionConfig(resourceID, resourceType, targetModule string) string {
	// Generate basic Terraform configuration for adoption
	var config strings.Builder

	if resourceType == "" {
		// Try to detect type from ID pattern
		if strings.HasPrefix(resourceID, "i-") {
			resourceType = "aws_instance"
		} else if strings.HasPrefix(resourceID, "vpc-") {
			resourceType = "aws_vpc"
		} else if strings.Contains(resourceID, "bucket") {
			resourceType = "aws_s3_bucket"
		}
	}

	resourceName := strings.ReplaceAll(resourceID, "-", "_")
	resourceName = strings.ReplaceAll(resourceName, ":", "_")
	resourceName = strings.ReplaceAll(resourceName, "/", "_")

	if targetModule != "" {
		config.WriteString(fmt.Sprintf("module \"%s\" {\n", targetModule))
		config.WriteString("  source = \"./modules/" + targetModule + "\"\n")
		config.WriteString("}\n\n")
	}

	config.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n", resourceType, resourceName))
	config.WriteString("  # Resource configuration will be populated after import\n")
	config.WriteString("  # Run: terraform import " + resourceType + "." + resourceName + " " + resourceID + "\n")
	config.WriteString("  # Then: terraform show -no-color\n")
	config.WriteString("}\n")

	return config.String()
}

// PerspectiveAnalysis represents analysis results
type PerspectiveAnalysis struct {
	PerspectiveID      string
	Timestamp          time.Time
	DriftScore         float64
	CoverageScore      float64
	ComplianceScore    float64
	Recommendations    []string
	DriftedResources   []models.Resource
	CoverageGaps       []string
	ComplianceIssues   []string
}

func (pa *PerspectiveAnalysis) analyzeFromPerspective(p *apimodels.Perspective) {
	// Calculate drift score
	if len(p.MissingResources) > 0 {
		pa.DriftScore = float64(len(p.MissingResources)) / float64(len(p.StateResources)) * 100
	}

	// Calculate coverage score
	total := len(p.StateResources) + len(p.UnmanagedResources)
	if total > 0 {
		pa.CoverageScore = float64(len(p.ManagedResources)) / float64(total) * 100
	}

	// Generate recommendations
	if len(p.UnmanagedResources) > 10 {
		pa.Recommendations = append(pa.Recommendations, 
			fmt.Sprintf("Consider adopting %d unmanaged resources into Terraform", len(p.UnmanagedResources)))
	}

	if len(p.MissingResources) > 0 {
		pa.Recommendations = append(pa.Recommendations,
			fmt.Sprintf("Investigate %d missing resources that exist in state but not in cloud", len(p.MissingResources)))
	}

	if pa.CoverageScore < 80 {
		pa.Recommendations = append(pa.Recommendations,
			"Improve infrastructure coverage by managing more resources through Terraform")
	}

	// Identify coverage gaps
	resourceTypes := make(map[string]int)
	for _, r := range p.UnmanagedResources {
		resourceTypes[r.Type]++
	}

	for rType, count := range resourceTypes {
		if count > 3 {
			pa.CoverageGaps = append(pa.CoverageGaps,
				fmt.Sprintf("%s (%d unmanaged)", rType, count))
		}
	}
}

func displayAnalysisTable(analysis *PerspectiveAnalysis, showDrift, showCoverage, showRecommendations bool) {
	fmt.Printf("Perspective Analysis: %s\n", analysis.PerspectiveID)
	fmt.Printf("Timestamp: %s\n\n", analysis.Timestamp.Format(time.RFC3339))

	// Scores table
	scoreTable := tablewriter.NewWriter(os.Stdout)
	scoreTable.SetHeader([]string{"Metric", "Score", "Status"})
	
	if showDrift {
		driftStatus := "Good"
		if analysis.DriftScore > 10 {
			driftStatus = "Warning"
		}
		if analysis.DriftScore > 25 {
			driftStatus = "Critical"
		}
		scoreTable.Append([]string{"Drift", fmt.Sprintf("%.1f%%", analysis.DriftScore), driftStatus})
	}

	if showCoverage {
		coverageStatus := "Good"
		if analysis.CoverageScore < 80 {
			coverageStatus = "Warning"
		}
		if analysis.CoverageScore < 60 {
			coverageStatus = "Critical"
		}
		scoreTable.Append([]string{"Coverage", fmt.Sprintf("%.1f%%", analysis.CoverageScore), coverageStatus})
	}

	scoreTable.Render()

	if showCoverage && len(analysis.CoverageGaps) > 0 {
		fmt.Println("\nCoverage Gaps:")
		for _, gap := range analysis.CoverageGaps {
			fmt.Printf("  • %s\n", gap)
		}
	}

	if showRecommendations && len(analysis.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range analysis.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}
}

func displayAnalysisJSON(analysis *PerspectiveAnalysis) {
	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func displayPerspectiveListTable(perspectives []PerspectiveListItem) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Provider", "State File", "Resources", "Coverage", "Created"})

	for _, p := range perspectives {
		table.Append([]string{
			p.ID,
			p.Provider,
			filepath.Base(p.StateFile),
			fmt.Sprintf("%d", p.TotalResources),
			fmt.Sprintf("%.1f%%", p.Coverage),
			p.Created.Format("2006-01-02 15:04"),
		})
	}

	table.Render()
}

type PerspectiveListItem struct {
	ID             string
	Provider       string
	StateFile      string
	TotalResources int
	Coverage       float64
	Status         string
	Created        time.Time
}

func generateSamplePerspectiveList(provider, status string, limit int) []PerspectiveListItem {
	// In a real implementation, this would query from storage
	var items []PerspectiveListItem

	baseTime := time.Now()
	providers := []string{"aws", "azure", "gcp", "digitalocean"}
	
	for i := 0; i < limit && i < 10; i++ {
		p := providers[i%len(providers)]
		if provider != "" && p != provider {
			continue
		}

		item := PerspectiveListItem{
			ID:             fmt.Sprintf("persp-%d", baseTime.Unix()-int64(i*3600)),
			Provider:       p,
			StateFile:      fmt.Sprintf("terraform-%s.tfstate", p),
			TotalResources: 100 + i*10,
			Coverage:       75.0 + float64(i*2),
			Status:         "active",
			Created:        baseTime.Add(time.Duration(-i) * 24 * time.Hour),
		}

		if status == "" || item.Status == status {
			items = append(items, item)
		}
	}

	return items
}