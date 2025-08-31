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

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
	disc "github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/fatih/color"
)

type EnhancedDiscoveryResult struct {
	ManagedResources      []models.Resource
	UnmanagedResources    []models.Resource
	ImportCandidates      []*disc.ImportCandidate
	Categories            map[disc.ResourceCategory][]models.Resource
	ShadowITResources     []models.Resource
	OrphanedResources     []models.Resource
	ComplianceIssues      []ComplianceIssue
	CostAnalysis          CostAnalysis
	Statistics            DiscoveryStatistics
	Recommendations       []string
	Timestamp             time.Time
}

type ComplianceIssue struct {
	Resource    models.Resource
	Issues      []string
	Severity    string
	Remediation string
}

type CostAnalysis struct {
	TotalManagedCost     float64
	TotalUnmanagedCost   float64
	PotentialSavings     float64
	HighCostUnmanaged    []CostItem
	CostByCategory       map[disc.ResourceCategory]float64
}

type CostItem struct {
	Resource models.Resource
	Cost     float64
	Category disc.ResourceCategory
}

type DiscoveryStatistics struct {
	TotalResources        int
	ManagedCount          int
	UnmanagedCount        int
	ImportCandidateCount  int
	ShadowITCount         int
	OrphanedCount         int
	ComplianceIssueCount  int
	CategoriesBreakdown   map[disc.ResourceCategory]int
}

// handleEnhancedDiscovery performs intelligent resource discovery with categorization
func handleEnhancedDiscovery(args []string) {
	ctx := context.Background()
	
	var (
		provider          = ""
		region            = ""
		unmanagedOnly     = false
		categorize        = true
		importCandidates  = false
		shadowIT          = false
		createdAfter      = ""
		createdBefore     = ""
		costAnalysis      = true
		complianceCheck   = true
		output            = ""
		jsonOutput        = false
		verbose           = false
		exportImports     = false
		scoreThreshold    = 50.0
		continuous        = false
		interval          = 5 * time.Minute
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
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
		case "--unmanaged-only":
			unmanagedOnly = true
		case "--no-categorize":
			categorize = false
		case "--import-candidates":
			importCandidates = true
		case "--shadow-it":
			shadowIT = true
		case "--created-after":
			if i+1 < len(args) {
				createdAfter = args[i+1]
				i++
			}
		case "--created-before":
			if i+1 < len(args) {
				createdBefore = args[i+1]
				i++
			}
		case "--no-cost":
			costAnalysis = false
		case "--no-compliance":
			complianceCheck = false
		case "--output", "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--json":
			jsonOutput = true
		case "--verbose", "-v":
			verbose = true
		case "--export-imports":
			exportImports = true
		case "--score-threshold":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%f", &scoreThreshold)
				i++
			}
		case "--continuous":
			continuous = true
		case "--interval":
			if i+1 < len(args) {
				if duration, err := time.ParseDuration(args[i+1]); err == nil {
					interval = duration
				}
				i++
			}
		case "--help", "-h":
			showEnhancedDiscoveryHelp()
			return
		}
	}
	
	// Validate provider
	if provider == "" {
		provider = detectDefaultProvider()
		if provider == "" {
			fmt.Fprintln(os.Stderr, color.RedString("Error: No cloud provider specified or detected"))
			os.Exit(1)
		}
	}
	
	// Run discovery
	if continuous {
		runContinuousDiscovery(ctx, provider, region, interval, unmanagedOnly, categorize, importCandidates)
	} else {
		result := runEnhancedDiscovery(ctx, provider, region, unmanagedOnly, categorize, 
			importCandidates, shadowIT, createdAfter, createdBefore, 
			costAnalysis, complianceCheck, scoreThreshold)
		
		// Display or export results
		if jsonOutput {
			displayEnhancedDiscoveryJSON(result)
		} else {
			displayEnhancedDiscoveryResults(result, verbose)
		}
		
		// Export import commands if requested
		if exportImports && len(result.ImportCandidates) > 0 {
			exportImportCommands(result.ImportCandidates, output)
		}
	}
}

func runEnhancedDiscovery(ctx context.Context, provider, region string,
	unmanagedOnly, categorize, importCandidates, shadowIT bool,
	createdAfter, createdBefore string, costAnalysis, complianceCheck bool,
	scoreThreshold float64) *EnhancedDiscoveryResult {
	
	result := &EnhancedDiscoveryResult{
		Categories:  make(map[disc.ResourceCategory][]models.Resource),
		Timestamp:   time.Now(),
		Statistics: DiscoveryStatistics{
			CategoriesBreakdown: make(map[disc.ResourceCategory]int),
		},
	}
	
	// Initialize discovery service
	discoveryService, err := discovery.InitializeServiceSilent(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error initializing discovery: %v\n"), err)
		os.Exit(1)
	}
	
	// Discover cloud resources
	fmt.Printf(color.CyanString("🔍 Discovering %s resources..."), provider)
	fmt.Println()
	
	opts := discovery.DiscoveryOptions{}
	if region != "" {
		opts.Regions = []string{region}
	}
	discoveryResult, err := discoveryService.DiscoverProvider(ctx, provider, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error discovering resources: %v\n"), err)
		os.Exit(1)
	}
	
	// Load Terraform state files
	stateResources := loadAllStateResources(ctx)
	
	// Initialize categorizer
	categorizer := disc.NewResourceCategorizer()
	
	// Process each discovered resource
	for _, resource := range discoveryResult.Resources {
		// Check if resource is in state
		inState := isResourceInState(resource, stateResources)
		
		if unmanagedOnly && inState {
			continue
		}
		
		// Apply date filters
		if !passesDateFilter(resource, createdAfter, createdBefore) {
			continue
		}
		
		// Categorize resource
		category := disc.CategoryUnknown
		if categorize {
			category = categorizer.CategorizeResource(resource, inState)
		}
		
		// Add to appropriate lists
		if inState {
			result.ManagedResources = append(result.ManagedResources, resource)
		} else {
			result.UnmanagedResources = append(result.UnmanagedResources, resource)
			
			// Check for shadow IT
			if shadowIT && category == disc.CategoryShadowIT {
				result.ShadowITResources = append(result.ShadowITResources, resource)
			}
			
			// Check for orphaned resources
			if category == disc.CategoryOrphaned {
				result.OrphanedResources = append(result.OrphanedResources, resource)
			}
			
			// Score as import candidate
			if importCandidates && (category == disc.CategoryManageable || category == disc.CategoryOrphaned) {
				candidate := categorizer.ScoreImportCandidate(resource)
				if candidate.Score >= scoreThreshold {
					result.ImportCandidates = append(result.ImportCandidates, candidate)
				}
			}
		}
		
		// Add to category map
		result.Categories[category] = append(result.Categories[category], resource)
		result.Statistics.CategoriesBreakdown[category]++
	}
	
	// Sort import candidates by score
	if len(result.ImportCandidates) > 0 {
		sort.Slice(result.ImportCandidates, func(i, j int) bool {
			return result.ImportCandidates[i].Score > result.ImportCandidates[j].Score
		})
	}
	
	// Perform compliance checks
	if complianceCheck {
		result.ComplianceIssues = performComplianceChecks(result.UnmanagedResources, categorizer)
	}
	
	// Perform cost analysis
	if costAnalysis {
		result.CostAnalysis = performCostAnalysis(result, categorizer)
	}
	
	// Calculate statistics
	result.Statistics.TotalResources = len(discoveryResult.Resources)
	result.Statistics.ManagedCount = len(result.ManagedResources)
	result.Statistics.UnmanagedCount = len(result.UnmanagedResources)
	result.Statistics.ImportCandidateCount = len(result.ImportCandidates)
	result.Statistics.ShadowITCount = len(result.ShadowITResources)
	result.Statistics.OrphanedCount = len(result.OrphanedResources)
	result.Statistics.ComplianceIssueCount = len(result.ComplianceIssues)
	
	// Generate recommendations
	result.Recommendations = generateDiscoveryRecommendations(result)
	
	return result
}

func loadAllStateResources(ctx context.Context) map[string]bool {
	stateResources := make(map[string]bool)
	
	// Find all state files
	stateFiles := findAllStateFiles()
	
	for _, statePath := range stateFiles {
		loader := state.NewStateLoader(statePath)
		if stateFile, err := loader.LoadStateFile(ctx, statePath, nil); err == nil {
			for _, resource := range stateFile.Resources {
				// Create unique identifier
				key := fmt.Sprintf("%s:%s", resource.Type, resource.ID)
				stateResources[key] = true
			}
		}
	}
	
	return stateResources
}

func findAllStateFiles() []string {
	var stateFiles []string
	
	// Check standard locations
	locations := []string{
		"terraform.tfstate",
		"terraform.tfstate.backup",
		".terraform/terraform.tfstate",
	}
	
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			stateFiles = append(stateFiles, loc)
		}
	}
	
	// Check for workspace states
	workspacePattern := "terraform.tfstate.d/*/terraform.tfstate"
	if matches, err := filepath.Glob(workspacePattern); err == nil {
		stateFiles = append(stateFiles, matches...)
	}
	
	// Check for Terragrunt states
	terragruntPattern := ".terragrunt-cache/**/terraform.tfstate"
	if matches, err := filepath.Glob(terragruntPattern); err == nil {
		stateFiles = append(stateFiles, matches...)
	}
	
	return stateFiles
}

func isResourceInState(resource models.Resource, stateResources map[string]bool) bool {
	key := fmt.Sprintf("%s:%s", resource.Type, resource.ID)
	return stateResources[key]
}

func passesDateFilter(resource models.Resource, createdAfter, createdBefore string) bool {
	if createdAfter == "" && createdBefore == "" {
		return true
	}
	
	// Get resource creation time
	var createdTime time.Time
	if !resource.CreatedAt.IsZero() {
		createdTime = resource.CreatedAt
	} else if !resource.Created.IsZero() {
		createdTime = resource.Created
	} else {
		return true // Can't filter without creation time
	}
	
	// Check after filter
	if createdAfter != "" {
		if after, err := time.Parse("2006-01-02", createdAfter); err == nil {
			if createdTime.Before(after) {
				return false
			}
		}
	}
	
	// Check before filter
	if createdBefore != "" {
		if before, err := time.Parse("2006-01-02", createdBefore); err == nil {
			if createdTime.After(before) {
				return false
			}
		}
	}
	
	return true
}

func performComplianceChecks(resources []models.Resource, categorizer *disc.ResourceCategorizer) []ComplianceIssue {
	var issues []ComplianceIssue
	
	requiredTags := []string{"environment", "owner", "project", "cost-center"}
	
	for _, resource := range resources {
		var resourceIssues []string
		
		// Check required tags
		tags, ok := resource.Tags.(map[string]string)
		if !ok || tags == nil {
			resourceIssues = append(resourceIssues, "No tags present")
		} else {
			for _, required := range requiredTags {
				if _, exists := tags[required]; !exists {
					resourceIssues = append(resourceIssues, fmt.Sprintf("Missing required tag: %s", required))
				}
			}
		}
		
		// Check naming convention
		if !strings.Contains(resource.Name, "-") {
			resourceIssues = append(resourceIssues, "Does not follow naming convention")
		}
		
		// Check for security issues
		if strings.Contains(resource.Type, "SecurityGroup") || strings.Contains(resource.Type, "NetworkACL") {
			if tags == nil || tags["reviewed"] != "true" {
				resourceIssues = append(resourceIssues, "Security resource not reviewed")
			}
		}
		
		if len(resourceIssues) > 0 {
			severity := "LOW"
			if len(resourceIssues) > 3 {
				severity = "HIGH"
			} else if len(resourceIssues) > 1 {
				severity = "MEDIUM"
			}
			
			issues = append(issues, ComplianceIssue{
				Resource:    resource,
				Issues:      resourceIssues,
				Severity:    severity,
				Remediation: "Add to Terraform configuration with required tags",
			})
		}
	}
	
	return issues
}

func performCostAnalysis(result *EnhancedDiscoveryResult, categorizer *disc.ResourceCategorizer) CostAnalysis {
	analysis := CostAnalysis{
		CostByCategory: make(map[disc.ResourceCategory]float64),
	}
	
	// Simple cost estimation (would integrate with cloud pricing APIs)
	costEstimates := map[string]float64{
		"Instance":     50.0,
		"Database":     100.0,
		"LoadBalancer": 25.0,
		"Storage":      10.0,
		"VPC":          5.0,
		"NAT":          45.0,
	}
	
	// Calculate managed costs
	for _, resource := range result.ManagedResources {
		for key, cost := range costEstimates {
			if strings.Contains(resource.Type, key) {
				analysis.TotalManagedCost += cost
				break
			}
		}
	}
	
	// Calculate unmanaged costs and find high-cost items
	for _, resource := range result.UnmanagedResources {
		for key, cost := range costEstimates {
			if strings.Contains(resource.Type, key) {
				analysis.TotalUnmanagedCost += cost
				
				category := categorizer.CategorizeResource(resource, false)
				analysis.CostByCategory[category] += cost
				
				if cost > 30 {
					analysis.HighCostUnmanaged = append(analysis.HighCostUnmanaged, CostItem{
						Resource: resource,
						Cost:     cost,
						Category: category,
					})
				}
				break
			}
		}
	}
	
	// Sort high-cost items
	sort.Slice(analysis.HighCostUnmanaged, func(i, j int) bool {
		return analysis.HighCostUnmanaged[i].Cost > analysis.HighCostUnmanaged[j].Cost
	})
	
	// Calculate potential savings (assuming 20% optimization through management)
	analysis.PotentialSavings = analysis.TotalUnmanagedCost * 0.2
	
	return analysis
}

func generateDiscoveryRecommendations(result *EnhancedDiscoveryResult) []string {
	var recommendations []string
	
	// High priority imports
	highPriorityCount := 0
	for _, candidate := range result.ImportCandidates {
		if candidate.Score > 80 {
			highPriorityCount++
		}
	}
	if highPriorityCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Import %d high-priority resources (score > 80) immediately", highPriorityCount))
	}
	
	// Shadow IT concerns
	if len(result.ShadowITResources) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Review %d shadow IT resources created outside standard process", len(result.ShadowITResources)))
	}
	
	// Orphaned resources
	if len(result.OrphanedResources) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Re-import or clean up %d orphaned resources", len(result.OrphanedResources)))
	}
	
	// Compliance issues
	highComplianceCount := 0
	for _, issue := range result.ComplianceIssues {
		if issue.Severity == "HIGH" {
			highComplianceCount++
		}
	}
	if highComplianceCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Address %d high-severity compliance issues", highComplianceCount))
	}
	
	// Cost optimization
	if result.CostAnalysis.PotentialSavings > 100 {
		recommendations = append(recommendations,
			fmt.Sprintf("Potential monthly savings of $%.2f by managing unmanaged resources",
				result.CostAnalysis.PotentialSavings))
	}
	
	// Unmanageable resources
	if count := len(result.Categories[disc.CategoryUnmanageable]); count > 10 {
		recommendations = append(recommendations,
			fmt.Sprintf("Review %d system resources marked as unmanageable", count))
	}
	
	return recommendations
}

func displayEnhancedDiscoveryResults(result *EnhancedDiscoveryResult, verbose bool) {
	// Header
	fmt.Println(color.CyanString("=" + strings.Repeat("=", 60)))
	fmt.Println(color.CyanString("Enhanced Resource Discovery Report"))
	fmt.Println(color.CyanString("=" + strings.Repeat("=", 60)))
	fmt.Printf("Timestamp: %s\n\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	
	// Statistics Summary
	fmt.Println(color.CyanString("📊 Discovery Statistics"))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Total Resources:      %d\n", result.Statistics.TotalResources)
	fmt.Printf("Managed:              %s\n", color.GreenString("%d", result.Statistics.ManagedCount))
	fmt.Printf("Unmanaged:            %s\n", color.YellowString("%d", result.Statistics.UnmanagedCount))
	fmt.Printf("Import Candidates:    %s\n", color.CyanString("%d", result.Statistics.ImportCandidateCount))
	fmt.Printf("Shadow IT:            %s\n", color.RedString("%d", result.Statistics.ShadowITCount))
	fmt.Printf("Orphaned:             %s\n", color.YellowString("%d", result.Statistics.OrphanedCount))
	fmt.Printf("Compliance Issues:    %s\n", color.RedString("%d", result.Statistics.ComplianceIssueCount))
	
	// Category Breakdown
	fmt.Println("\n" + color.CyanString("📂 Resource Categories"))
	fmt.Println(strings.Repeat("-", 40))
	categories := []disc.ResourceCategory{
		disc.CategoryManaged,
		disc.CategoryManageable,
		disc.CategoryUnmanageable,
		disc.CategoryShadowIT,
		disc.CategoryOrphaned,
		disc.CategoryTemporary,
		disc.CategoryUnknown,
	}
	
	for _, cat := range categories {
		if count := result.Statistics.CategoriesBreakdown[cat]; count > 0 {
			colorFunc := getColorForCategory(cat)
			fmt.Printf("%-15s: %s\n", cat, colorFunc("%d", count))
		}
	}
	
	// Top Import Candidates
	if len(result.ImportCandidates) > 0 {
		fmt.Println("\n" + color.CyanString("🎯 Top Import Candidates"))
		fmt.Println(strings.Repeat("-", 40))
		
		displayCount := 10
		if !verbose {
			displayCount = 5
		}
		
		for i, candidate := range result.ImportCandidates {
			if i >= displayCount {
				fmt.Printf("\n... and %d more candidates\n", len(result.ImportCandidates)-displayCount)
				break
			}
			
			scoreColor := color.GreenString
			if candidate.Score < 60 {
				scoreColor = color.YellowString
			} else if candidate.Score < 40 {
				scoreColor = color.RedString
			}
			
			fmt.Printf("\n%d. %s (Score: %s)\n", i+1, 
				color.CyanString(candidate.Resource.Name),
				scoreColor("%.1f", candidate.Score))
			
			if verbose {
				fmt.Printf("   Type: %s\n", candidate.Resource.Type)
				fmt.Printf("   Import: %s\n", candidate.ImportCommand)
				fmt.Printf("   Reasons:\n")
				for _, reason := range candidate.Reasons {
					fmt.Printf("     • %s\n", reason)
				}
				if len(candidate.ComplianceIssues) > 0 {
					fmt.Printf("   ⚠ Compliance Issues:\n")
					for _, issue := range candidate.ComplianceIssues {
						fmt.Printf("     • %s\n", color.YellowString(issue))
					}
				}
			}
		}
	}
	
	// Shadow IT Resources
	if len(result.ShadowITResources) > 0 && verbose {
		fmt.Println("\n" + color.RedString("🚨 Shadow IT Resources"))
		fmt.Println(strings.Repeat("-", 40))
		
		for i, resource := range result.ShadowITResources {
			if i >= 5 {
				fmt.Printf("\n... and %d more shadow IT resources\n", len(result.ShadowITResources)-5)
				break
			}
			fmt.Printf("• %s (%s)\n", resource.Name, resource.Type)
		}
	}
	
	// Cost Analysis
	if result.CostAnalysis.TotalUnmanagedCost > 0 {
		fmt.Println("\n" + color.CyanString("💰 Cost Analysis"))
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Managed Resources:    $%.2f/month\n", result.CostAnalysis.TotalManagedCost)
		fmt.Printf("Unmanaged Resources:  %s\n", color.YellowString("$%.2f/month", result.CostAnalysis.TotalUnmanagedCost))
		fmt.Printf("Potential Savings:    %s\n", color.GreenString("$%.2f/month", result.CostAnalysis.PotentialSavings))
		
		if len(result.CostAnalysis.HighCostUnmanaged) > 0 {
			fmt.Println("\nHigh-Cost Unmanaged Resources:")
			for i, item := range result.CostAnalysis.HighCostUnmanaged {
				if i >= 3 {
					break
				}
				fmt.Printf("  • %s: $%.2f/month (%s)\n", 
					item.Resource.Name, item.Cost, item.Category)
			}
		}
	}
	
	// Compliance Issues
	if len(result.ComplianceIssues) > 0 {
		fmt.Println("\n" + color.YellowString("⚠ Compliance Issues"))
		fmt.Println(strings.Repeat("-", 40))
		
		// Group by severity
		highCount, mediumCount, lowCount := 0, 0, 0
		for _, issue := range result.ComplianceIssues {
			switch issue.Severity {
			case "HIGH":
				highCount++
			case "MEDIUM":
				mediumCount++
			case "LOW":
				lowCount++
			}
		}
		
		if highCount > 0 {
			fmt.Printf("High Severity:   %s\n", color.RedString("%d", highCount))
		}
		if mediumCount > 0 {
			fmt.Printf("Medium Severity: %s\n", color.YellowString("%d", mediumCount))
		}
		if lowCount > 0 {
			fmt.Printf("Low Severity:    %d\n", lowCount)
		}
		
		if verbose {
			fmt.Println("\nTop Issues:")
			for i, issue := range result.ComplianceIssues {
				if i >= 5 {
					break
				}
				severityColor := color.YellowString
				if issue.Severity == "HIGH" {
					severityColor = color.RedString
				}
				
				fmt.Printf("\n• %s [%s]\n", issue.Resource.Name, severityColor(issue.Severity))
				for _, iss := range issue.Issues {
					fmt.Printf("  - %s\n", iss)
				}
			}
		}
	}
	
	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Println("\n" + color.CyanString("💡 Recommendations"))
		fmt.Println(strings.Repeat("-", 40))
		for i, rec := range result.Recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
	}
	
	// Summary Actions
	fmt.Println("\n" + color.CyanString("🎬 Next Steps"))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("1. Run 'driftmgr discover --export-imports' to generate import script")
	fmt.Println("2. Review shadow IT resources for compliance")
	fmt.Println("3. Import high-priority candidates to Terraform")
	fmt.Println("4. Address compliance issues in unmanaged resources")
	if result.CostAnalysis.PotentialSavings > 50 {
		fmt.Printf("5. Realize potential savings of $%.2f/month\n", result.CostAnalysis.PotentialSavings)
	}
}

func getColorForCategory(category disc.ResourceCategory) func(string, ...interface{}) string {
	switch category {
	case disc.CategoryManaged:
		return color.GreenString
	case disc.CategoryManageable:
		return color.CyanString
	case disc.CategoryUnmanageable:
		return color.New(color.FgHiBlack).SprintfFunc()
	case disc.CategoryShadowIT:
		return color.RedString
	case disc.CategoryOrphaned:
		return color.YellowString
	case disc.CategoryTemporary:
		return color.New(color.FgHiBlack).SprintfFunc()
	default:
		return color.New(color.FgWhite).SprintfFunc()
	}
}

func displayEnhancedDiscoveryJSON(result *EnhancedDiscoveryResult) {
	output := map[string]interface{}{
		"timestamp":  result.Timestamp.Format(time.RFC3339),
		"statistics": result.Statistics,
		"categories": map[string]int{},
		"cost_analysis": map[string]float64{
			"managed_cost":      result.CostAnalysis.TotalManagedCost,
			"unmanaged_cost":    result.CostAnalysis.TotalUnmanagedCost,
			"potential_savings": result.CostAnalysis.PotentialSavings,
		},
		"import_candidates":  len(result.ImportCandidates),
		"shadow_it_count":    len(result.ShadowITResources),
		"orphaned_count":     len(result.OrphanedResources),
		"compliance_issues":  len(result.ComplianceIssues),
		"recommendations":    result.Recommendations,
	}
	
	// Add category counts
	for cat, count := range result.Statistics.CategoriesBreakdown {
		output["categories"].(map[string]int)[string(cat)] = count
	}
	
	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func exportImportCommands(candidates []*disc.ImportCandidate, outputFile string) {
	var output strings.Builder
	
	// Header
	output.WriteString("#!/bin/bash\n")
	output.WriteString("# Terraform Import Commands - Generated by DriftMgr\n")
	output.WriteString(fmt.Sprintf("# Generated: %s\n", time.Now().Format(time.RFC3339)))
	output.WriteString(fmt.Sprintf("# Total Candidates: %d\n\n", len(candidates)))
	
	// Group by score ranges
	output.WriteString("# High Priority (Score > 80)\n")
	for _, candidate := range candidates {
		if candidate.Score > 80 {
			output.WriteString(fmt.Sprintf("# Score: %.1f - %s\n", candidate.Score, strings.Join(candidate.Reasons, ", ")))
			output.WriteString(fmt.Sprintf("%s\n\n", candidate.ImportCommand))
		}
	}
	
	output.WriteString("# Medium Priority (Score 60-80)\n")
	for _, candidate := range candidates {
		if candidate.Score > 60 && candidate.Score <= 80 {
			output.WriteString(fmt.Sprintf("# Score: %.1f - %s\n", candidate.Score, candidate.Resource.Name))
			output.WriteString(fmt.Sprintf("%s\n\n", candidate.ImportCommand))
		}
	}
	
	output.WriteString("# Low Priority (Score < 60)\n")
	for _, candidate := range candidates {
		if candidate.Score <= 60 {
			output.WriteString(fmt.Sprintf("# Score: %.1f - %s\n", candidate.Score, candidate.Resource.Name))
			output.WriteString(fmt.Sprintf("%s\n\n", candidate.ImportCommand))
		}
	}
	
	// Write to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output.String()), 0755); err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error writing import script: %v\n"), err)
		} else {
			fmt.Printf(color.GreenString("✓ Import script written to: %s\n"), outputFile)
		}
	} else {
		fmt.Print(output.String())
	}
}

func runContinuousDiscovery(ctx context.Context, provider, region string, interval time.Duration,
	unmanagedOnly, categorize, importCandidates bool) {
	
	fmt.Printf(color.CyanString("Starting continuous discovery (interval: %s)\n"), interval)
	fmt.Println("Press Ctrl+C to stop")
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Run initial discovery
	runDiscoveryIteration(ctx, provider, region, unmanagedOnly, categorize, importCandidates)
	
	// Run periodic discoveries
	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n%s Running discovery...\n", time.Now().Format("15:04:05"))
			runDiscoveryIteration(ctx, provider, region, unmanagedOnly, categorize, importCandidates)
		case <-ctx.Done():
			fmt.Println("\nStopping continuous discovery")
			return
		}
	}
}

func runDiscoveryIteration(ctx context.Context, provider, region string,
	unmanagedOnly, categorize, importCandidates bool) {
	
	result := runEnhancedDiscovery(ctx, provider, region, unmanagedOnly, categorize,
		importCandidates, false, "", "", true, true, 50.0)
	
	// Display summary
	fmt.Printf("Found: %d total, %d unmanaged, %d import candidates, %d shadow IT\n",
		result.Statistics.TotalResources,
		result.Statistics.UnmanagedCount,
		result.Statistics.ImportCandidateCount,
		result.Statistics.ShadowITCount)
	
	// Alert on new shadow IT
	if result.Statistics.ShadowITCount > 0 {
		fmt.Printf(color.RedString("⚠ Alert: %d shadow IT resources detected\n"), 
			result.Statistics.ShadowITCount)
	}
	
	// Alert on high-priority imports
	highPriority := 0
	for _, candidate := range result.ImportCandidates {
		if candidate.Score > 80 {
			highPriority++
		}
	}
	if highPriority > 0 {
		fmt.Printf(color.YellowString("📌 %d high-priority resources should be imported\n"), highPriority)
	}
}

func detectDefaultProvider() string {
	// Check environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
		return "aws"
	}
	if os.Getenv("AZURE_CLIENT_ID") != "" || os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
		return "azure"
	}
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" || os.Getenv("GOOGLE_CLOUD_PROJECT") != "" {
		return "gcp"
	}
	if os.Getenv("DIGITALOCEAN_TOKEN") != "" {
		return "digitalocean"
	}
	
	return ""
}

func showEnhancedDiscoveryHelp() {
	fmt.Println("Usage: driftmgr discover [flags]")
	fmt.Println()
	fmt.Println("Enhanced resource discovery with intelligent categorization")
	fmt.Println()
	fmt.Println("Discovery Modes:")
	fmt.Println("  --unmanaged-only        Show only resources not in Terraform state")
	fmt.Println("  --import-candidates     Identify and score import candidates")
	fmt.Println("  --shadow-it             Detect shadow IT resources")
	fmt.Println("  --continuous            Run continuous discovery with alerts")
	fmt.Println()
	fmt.Println("Filtering:")
	fmt.Println("  --provider string       Cloud provider (aws, azure, gcp)")
	fmt.Println("  --region string         Cloud region")
	fmt.Println("  --created-after date    Resources created after date (YYYY-MM-DD)")
	fmt.Println("  --created-before date   Resources created before date")
	fmt.Println("  --score-threshold float Minimum score for import candidates (default: 50)")
	fmt.Println()
	fmt.Println("Analysis:")
	fmt.Println("  --no-categorize         Skip resource categorization")
	fmt.Println("  --no-cost               Skip cost analysis")
	fmt.Println("  --no-compliance         Skip compliance checks")
	fmt.Println()
	fmt.Println("Output:")
	fmt.Println("  --export-imports        Export import commands to file")
	fmt.Println("  --output, -o file       Output file for exports")
	fmt.Println("  --json                  Output in JSON format")
	fmt.Println("  --verbose, -v           Show detailed information")
	fmt.Println("  --interval duration     Discovery interval for continuous mode (default: 5m)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Find all unmanaged resources")
	fmt.Println("  driftmgr discover --unmanaged-only")
	fmt.Println()
	fmt.Println("  # Get import candidates with high scores")
	fmt.Println("  driftmgr discover --import-candidates --score-threshold 70")
	fmt.Println()
	fmt.Println("  # Detect shadow IT resources")
	fmt.Println("  driftmgr discover --shadow-it --created-after 2024-01-01")
	fmt.Println()
	fmt.Println("  # Export import commands")
	fmt.Println("  driftmgr discover --import-candidates --export-imports -o imports.sh")
	fmt.Println()
	fmt.Println("  # Continuous monitoring")
	fmt.Println("  driftmgr discover --continuous --interval 10m --shadow-it")
}