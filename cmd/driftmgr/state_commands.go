package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/color"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// handleStateDiscover discovers resources from Terraform state files
func handleStateDiscover(args []string) {
	var (
		statePath  = ""
		provider   = ""
		region     = ""
		outputJSON = false
		verbose    = false
	)

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Println("Usage: driftmgr state discover [flags]")
			fmt.Println()
			fmt.Println("Discover resources from Terraform state files")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --state-path PATH    Path to Terraform state file or directory")
			fmt.Println("  --provider PROVIDER  Filter by provider (aws, azure, gcp, digitalocean)")
			fmt.Println("  --region REGION      Filter by region")
			fmt.Println("  --json              Output results in JSON format")
			fmt.Println("  --verbose           Show detailed information")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr state discover --state-path ./terraform.tfstate")
			fmt.Println("  driftmgr state discover --state-path ./ --provider aws")
			fmt.Println("  driftmgr state discover --json")
			return
		case "--state-path":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--region":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case "--json":
			outputJSON = true
		case "--verbose":
			verbose = true
		}
	}

	// Default to current directory if no path specified
	if statePath == "" {
		statePath = "."
	}

	// Initialize discovery service
	discoveryService := state.NewDiscoveryService()
	ctx := context.Background()

	// Find state files
	var stateFiles []string
	fileInfo, err := os.Stat(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing path: %v\n", err)
		os.Exit(1)
	}

	if fileInfo.IsDir() {
		// Scan directory for state files
		err = filepath.Walk(statePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (strings.HasSuffix(path, ".tfstate") || strings.HasSuffix(path, ".tfstate.backup")) {
				stateFiles = append(stateFiles, path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		stateFiles = append(stateFiles, statePath)
	}

	if len(stateFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No Terraform state files found in %s\n", statePath)
		os.Exit(1)
	}

	// Process each state file
	allResources := make(map[string]*models.Resource)
	for _, stateFile := range stateFiles {
		if verbose {
			fmt.Printf("Processing state file: %s\n", stateFile)
		}

		resources, err := discoveryService.DiscoverFromStateFile(ctx, stateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", stateFile, err)
			continue
		}

		// Apply filters
		for id, resource := range resources {
			// Filter by provider
			if provider != "" && !strings.Contains(strings.ToLower(resource.Provider), strings.ToLower(provider)) {
				continue
			}

			// Filter by region
			if region != "" && resource.Region != region {
				continue
			}

			allResources[id] = resource
		}
	}

	// Output results
	if outputJSON {
		outputStateDiscoveryJSON(allResources)
	} else {
		outputStateDiscoveryTable(allResources, verbose)
	}
}

// handleStateAnalyze analyzes Terraform state for drift and issues
func handleStateAnalyze(args []string) {
	var (
		statePath     = ""
		provider      = ""
		region        = ""
		checkDrift    = false
		checkCoverage = false
		checkOrphans  = false
		outputJSON    = false
		verbose       = false
	)

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Println("Usage: driftmgr state analyze [flags]")
			fmt.Println()
			fmt.Println("Analyze Terraform state for drift, coverage, and issues")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --state-path PATH    Path to Terraform state file or directory")
			fmt.Println("  --provider PROVIDER  Filter by provider")
			fmt.Println("  --region REGION      Filter by region")
			fmt.Println("  --check-drift       Check for configuration drift")
			fmt.Println("  --check-coverage    Check state coverage")
			fmt.Println("  --check-orphans     Check for orphaned resources")
			fmt.Println("  --json              Output results in JSON format")
			fmt.Println("  --verbose           Show detailed analysis")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr state analyze --state-path ./terraform.tfstate --check-drift")
			fmt.Println("  driftmgr state analyze --check-coverage --check-orphans")
			return
		case "--state-path":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--region":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case "--check-drift":
			checkDrift = true
		case "--check-coverage":
			checkCoverage = true
		case "--check-orphans":
			checkOrphans = true
		case "--json":
			outputJSON = true
		case "--verbose":
			verbose = true
		}
	}

	// Default to all checks if none specified
	if !checkDrift && !checkCoverage && !checkOrphans {
		checkDrift = true
		checkCoverage = true
		checkOrphans = true
	}

	// Default to current directory if no path specified
	if statePath == "" {
		statePath = "."
	}

	// Initialize services
	ctx := context.Background()
	analyzer := state.NewAnalyzer()
	_ = state.NewDiscoveryService() // Will be used for enhanced discovery

	// Find and load state files
	var stateFiles []string
	fileInfo, err := os.Stat(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing path: %v\n", err)
		os.Exit(1)
	}

	if fileInfo.IsDir() {
		err = filepath.Walk(statePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".tfstate") {
				stateFiles = append(stateFiles, path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		stateFiles = append(stateFiles, statePath)
	}

	// Perform analysis
	analysisResults := make(map[string]interface{})

	for _, stateFile := range stateFiles {
		if verbose {
			fmt.Printf("Analyzing state file: %s\n", stateFile)
		}

		// Load state
		stateData, err := state.LoadStateFile(stateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading state file %s: %v\n", stateFile, err)
			continue
		}

		// Analyze the state
		result, err := analyzer.AnalyzeState(ctx, stateData, state.AnalysisOptions{
			CheckDrift:    checkDrift,
			CheckCoverage: checkCoverage,
			CheckOrphans:  checkOrphans,
			Provider:      provider,
			Region:        region,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing state: %v\n", err)
			continue
		}

		analysisResults[stateFile] = result
	}

	// Output results
	if outputJSON {
		outputStateAnalysisJSON(analysisResults)
	} else {
		outputStateAnalysisTable(analysisResults, verbose)
	}
}

// handleStateCompare compares multiple Terraform state files
func handleStateCompare(args []string) {
	var (
		sourceState = ""
		targetState = ""
		showAdded   = true
		showRemoved = true
		showChanged = true
		outputJSON  = false
		verbose     = false
	)

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Println("Usage: driftmgr state compare [flags]")
			fmt.Println()
			fmt.Println("Compare multiple Terraform state files")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --source PATH       Source state file")
			fmt.Println("  --target PATH       Target state file to compare against")
			fmt.Println("  --no-added         Don't show added resources")
			fmt.Println("  --no-removed       Don't show removed resources")
			fmt.Println("  --no-changed       Don't show changed resources")
			fmt.Println("  --json             Output results in JSON format")
			fmt.Println("  --verbose          Show detailed comparison")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr state compare --source prod.tfstate --target staging.tfstate")
			fmt.Println("  driftmgr state compare --source old.tfstate --target new.tfstate --json")
			return
		case "--source":
			if i+1 < len(args) {
				sourceState = args[i+1]
				i++
			}
		case "--target":
			if i+1 < len(args) {
				targetState = args[i+1]
				i++
			}
		case "--no-added":
			showAdded = false
		case "--no-removed":
			showRemoved = false
		case "--no-changed":
			showChanged = false
		case "--json":
			outputJSON = true
		case "--verbose":
			verbose = true
		}
	}

	// Validate required arguments
	if sourceState == "" || targetState == "" {
		fmt.Fprintf(os.Stderr, "Error: Both --source and --target state files are required\n")
		fmt.Println("Run 'driftmgr state compare --help' for usage")
		os.Exit(1)
	}

	// Load state files
	ctx := context.Background()
	discoveryService := state.NewDiscoveryService()

	// Load source state
	sourceResources, err := discoveryService.DiscoverFromStateFile(ctx, sourceState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading source state: %v\n", err)
		os.Exit(1)
	}

	// Load target state
	targetResources, err := discoveryService.DiscoverFromStateFile(ctx, targetState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading target state: %v\n", err)
		os.Exit(1)
	}

	// Compare states
	comparison := compareStates(sourceResources, targetResources)

	// Filter results based on flags
	if !showAdded {
		comparison.Added = nil
	}
	if !showRemoved {
		comparison.Removed = nil
	}
	if !showChanged {
		comparison.Changed = nil
	}

	// Output results
	if outputJSON {
		outputStateComparisonJSON(comparison)
	} else {
		outputStateComparisonTable(comparison, sourceState, targetState, verbose)
	}
}

// StateComparison represents the comparison between two states
type StateComparison struct {
	Added   map[string]*models.Resource `json:"added,omitempty"`
	Removed map[string]*models.Resource `json:"removed,omitempty"`
	Changed map[string]ChangedResource  `json:"changed,omitempty"`
}

// ChangedResource represents a resource that changed between states
type ChangedResource struct {
	Old *models.Resource `json:"old"`
	New *models.Resource `json:"new"`
}

// compareStates compares two sets of resources
func compareStates(source, target map[string]*models.Resource) StateComparison {
	comparison := StateComparison{
		Added:   make(map[string]*models.Resource),
		Removed: make(map[string]*models.Resource),
		Changed: make(map[string]ChangedResource),
	}

	// Find removed and changed resources
	for id, sourceResource := range source {
		if targetResource, exists := target[id]; exists {
			// Check if resource changed
			if hasResourceChanged(sourceResource, targetResource) {
				comparison.Changed[id] = ChangedResource{
					Old: sourceResource,
					New: targetResource,
				}
			}
		} else {
			// Resource was removed
			comparison.Removed[id] = sourceResource
		}
	}

	// Find added resources
	for id, targetResource := range target {
		if _, exists := source[id]; !exists {
			comparison.Added[id] = targetResource
		}
	}

	return comparison
}

// hasResourceChanged checks if a resource has changed
func hasResourceChanged(old, new *models.Resource) bool {
	// Compare basic properties
	if old.Type != new.Type || old.Provider != new.Provider || old.Region != new.Region {
		return true
	}

	// Compare tags if they are maps
	if oldTags, ok := old.Tags.(map[string]string); ok {
		if newTags, ok := new.Tags.(map[string]string); ok {
			if len(oldTags) != len(newTags) {
				return true
			}
			for k, v := range oldTags {
				if newV, exists := newTags[k]; !exists || v != newV {
					return true
				}
			}
		} else {
			return true // Different tag types
		}
	} else if new.Tags != nil {
		return true // Old has no tags but new does
	}

	// Compare properties (simplified comparison)
	oldProps, _ := json.Marshal(old.Properties)
	newProps, _ := json.Marshal(new.Properties)
	return string(oldProps) != string(newProps)
}

// Output functions
func outputStateDiscoveryJSON(resources map[string]*models.Resource) {
	output := map[string]interface{}{
		"total_resources": len(resources),
		"resources":       resources,
		"timestamp":       time.Now().UTC(),
	}
	
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}

func outputStateDiscoveryTable(resources map[string]*models.Resource, verbose bool) {
	if len(resources) == 0 {
		fmt.Println("No resources found")
		return
	}

	// Group resources by provider and type
	byProvider := make(map[string]map[string][]*models.Resource)
	for _, resource := range resources {
		if _, exists := byProvider[resource.Provider]; !exists {
			byProvider[resource.Provider] = make(map[string][]*models.Resource)
		}
		byProvider[resource.Provider][resource.Type] = append(byProvider[resource.Provider][resource.Type], resource)
	}

	// Display summary
	fmt.Println("=== Terraform State Discovery ===")
	fmt.Printf("Total Resources: %d\n\n", len(resources))

	// Display by provider
	for provider, types := range byProvider {
		color.Printf(color.Cyan, "[%s]\n", strings.ToUpper(provider))
		
		// Sort types for consistent output
		var typeNames []string
		for typeName := range types {
			typeNames = append(typeNames, typeName)
		}
		sort.Strings(typeNames)

		for _, typeName := range typeNames {
			resources := types[typeName]
			fmt.Printf("  %s (%d):\n", typeName, len(resources))
			
			if verbose {
				for _, resource := range resources {
					fmt.Printf("    - %s", resource.Name)
					if resource.Region != "" {
						fmt.Printf(" [%s]", resource.Region)
					}
					if tags, ok := resource.Tags.(map[string]string); ok && len(tags) > 0 {
						fmt.Printf(" {%d tags}", len(tags))
					}
					fmt.Println()
				}
			}
		}
		fmt.Println()
	}
}

func outputStateAnalysisJSON(results map[string]interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(results)
}

func outputStateAnalysisTable(results map[string]interface{}, verbose bool) {
	fmt.Println("=== Terraform State Analysis ===")
	fmt.Println()

	for stateFile, result := range results {
		fmt.Printf("State File: %s\n", stateFile)
		fmt.Println(strings.Repeat("-", 50))

		if analysisResult, ok := result.(*state.AnalysisResult); ok {
			// Display drift summary
			if analysisResult.DriftSummary != nil {
				color.Printf(color.Yellow, "\n📊 Drift Summary:\n")
				fmt.Printf("  Total Resources: %d\n", analysisResult.DriftSummary.TotalResources)
				fmt.Printf("  Drifted: %d\n", analysisResult.DriftSummary.DriftedResources)
				fmt.Printf("  Missing: %d\n", analysisResult.DriftSummary.MissingResources)
				fmt.Printf("  Extra: %d\n", analysisResult.DriftSummary.ExtraResources)
				
				if analysisResult.DriftSummary.DriftPercentage > 0 {
					color.Printf(color.Red, "  Drift Percentage: %.2f%%\n", analysisResult.DriftSummary.DriftPercentage)
				} else {
					color.Printf(color.Green, "  Drift Percentage: 0.00%%\n")
				}
			}

			// Display coverage analysis
			if analysisResult.CoverageAnalysis != nil {
				color.Printf(color.Cyan, "\n📈 Coverage Analysis:\n")
				fmt.Printf("  Managed Resources: %d\n", analysisResult.CoverageAnalysis.ManagedResources)
				fmt.Printf("  Unmanaged Resources: %d\n", analysisResult.CoverageAnalysis.UnmanagedResources)
				fmt.Printf("  Coverage: %.2f%%\n", analysisResult.CoverageAnalysis.CoveragePercentage)
				
				if verbose && len(analysisResult.CoverageAnalysis.UncoveredTypes) > 0 {
					fmt.Println("  Uncovered Resource Types:")
					for _, resType := range analysisResult.CoverageAnalysis.UncoveredTypes {
						fmt.Printf("    - %s\n", resType)
					}
				}
			}

			// Display orphaned resources
			if analysisResult.OrphanedResources != nil && len(analysisResult.OrphanedResources) > 0 {
				color.Printf(color.Magenta, "\n🔍 Orphaned Resources: %d\n", len(analysisResult.OrphanedResources))
				if verbose {
					for _, resource := range analysisResult.OrphanedResources {
						fmt.Printf("    - %s (%s)\n", resource.Name, resource.Type)
					}
				}
			}

			// Display issues
			if len(analysisResult.Issues) > 0 {
				color.Printf(color.Red, "\n⚠️  Issues Found: %d\n", len(analysisResult.Issues))
				for _, issue := range analysisResult.Issues {
					fmt.Printf("  - %s: %s\n", issue.Severity, issue.Message)
				}
			}
		}
		fmt.Println()
	}
}

func outputStateComparisonJSON(comparison StateComparison) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(comparison)
}

func outputStateComparisonTable(comparison StateComparison, source, target string, verbose bool) {
	fmt.Println("=== State Comparison ===")
	fmt.Printf("Source: %s\n", source)
	fmt.Printf("Target: %s\n", target)
	fmt.Println()

	// Summary
	totalChanges := len(comparison.Added) + len(comparison.Removed) + len(comparison.Changed)
	if totalChanges == 0 {
		color.Printf(color.Green, "✓ No differences found between states\n")
		return
	}

	fmt.Printf("Total Changes: %d\n", totalChanges)
	fmt.Println(strings.Repeat("-", 50))

	// Added resources
	if len(comparison.Added) > 0 {
		color.Printf(color.Green, "\n➕ Added Resources (%d):\n", len(comparison.Added))
		for id, resource := range comparison.Added {
			fmt.Printf("  + %s (%s)", resource.Name, resource.Type)
			if verbose {
				fmt.Printf(" [%s]", id)
			}
			fmt.Println()
		}
	}

	// Removed resources
	if len(comparison.Removed) > 0 {
		color.Printf(color.Red, "\n➖ Removed Resources (%d):\n", len(comparison.Removed))
		for id, resource := range comparison.Removed {
			fmt.Printf("  - %s (%s)", resource.Name, resource.Type)
			if verbose {
				fmt.Printf(" [%s]", id)
			}
			fmt.Println()
		}
	}

	// Changed resources
	if len(comparison.Changed) > 0 {
		color.Printf(color.Yellow, "\n🔄 Changed Resources (%d):\n", len(comparison.Changed))
		for id, change := range comparison.Changed {
			fmt.Printf("  ~ %s (%s)", change.New.Name, change.New.Type)
			if verbose {
				fmt.Printf(" [%s]", id)
				// Show what changed
				if change.Old.Region != change.New.Region {
					fmt.Printf("\n    Region: %s → %s", change.Old.Region, change.New.Region)
				}
				// Compare tags if both are maps
				oldTags, oldOk := change.Old.Tags.(map[string]string)
				newTags, newOk := change.New.Tags.(map[string]string)
				if oldOk && newOk && len(oldTags) != len(newTags) {
					fmt.Printf("\n    Tags: %d → %d", len(oldTags), len(newTags))
				}
			}
			fmt.Println()
		}
	}
}