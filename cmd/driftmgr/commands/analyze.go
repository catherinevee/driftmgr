package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/catherinevee/driftmgr/internal/cost"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/graph"
	"github.com/catherinevee/driftmgr/internal/health"
	"github.com/catherinevee/driftmgr/internal/state"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze Terraform state files",
	Long:  `Perform various analyses on Terraform state files including dependencies, health, and cost.`,
}

var analyzeDependenciesCmd = &cobra.Command{
	Use:   "dependencies [state-file]",
	Short: "Analyze resource dependencies in state",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAnalyzeDependencies,
}

var analyzeHealthCmd = &cobra.Command{
	Use:   "health [state-file]",
	Short: "Analyze resource health and compliance",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAnalyzeHealth,
}

var analyzeCostCmd = &cobra.Command{
	Use:   "cost [state-file]",
	Short: "Analyze resource costs and optimization opportunities",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAnalyzeCost,
}

var analyzeBlastRadiusCmd = &cobra.Command{
	Use:   "blast-radius [resource-id]",
	Short: "Calculate blast radius for a resource",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyzeBlastRadius,
}

var (
	analyzeBackend    string
	analyzeWorkspace  string
	analyzeOutput     string
	analyzeFormat     string
	analyzeProvider   string
	analyzeSeverity   string
	analyzeMaxDepth   int
	analyzeShowCycles bool
)

func init() {
	analyzeCmd.AddCommand(analyzeDependenciesCmd)
	analyzeCmd.AddCommand(analyzeHealthCmd)
	analyzeCmd.AddCommand(analyzeCostCmd)
	analyzeCmd.AddCommand(analyzeBlastRadiusCmd)

	// Common flags
	analyzeCmd.PersistentFlags().StringVar(&analyzeBackend, "backend", "", "Backend ID to use")
	analyzeCmd.PersistentFlags().StringVar(&analyzeWorkspace, "workspace", "default", "Terraform workspace")
	analyzeCmd.PersistentFlags().StringVarP(&analyzeOutput, "output", "o", "", "Output file path")
	analyzeCmd.PersistentFlags().StringVar(&analyzeFormat, "format", "text", "Output format (text, json, yaml)")
	
	// Specific flags
	analyzeDependenciesCmd.Flags().IntVar(&analyzeMaxDepth, "max-depth", 10, "Maximum dependency depth")
	analyzeDependenciesCmd.Flags().BoolVar(&analyzeShowCycles, "show-cycles", false, "Show circular dependencies")
	
	analyzeHealthCmd.Flags().StringVar(&analyzeProvider, "provider", "", "Filter by provider (aws, azure, gcp)")
	analyzeHealthCmd.Flags().StringVar(&analyzeSeverity, "min-severity", "low", "Minimum severity to report (low, medium, high, critical)")
	
	analyzeCostCmd.Flags().StringVar(&analyzeProvider, "provider", "", "Filter by provider (aws, azure, gcp)")
}

func runAnalyzeDependencies(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Get state
	state, err := getStateForAnalysis(ctx, args)
	if err != nil {
		return err
	}
	
	// Build dependency graph
	builder := graph.NewDependencyGraphBuilder(state)
	depGraph, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}
	
	fmt.Printf("Analyzing dependencies for %d resources...\n\n", len(depGraph.Nodes()))
	
	// Detect cycles
	cycles := depGraph.DetectCycles()
	if len(cycles) > 0 {
		fmt.Printf("⚠️  Found %d circular dependencies:\n", len(cycles))
		if analyzeShowCycles {
			for i, cycle := range cycles {
				fmt.Printf("\nCycle %d:\n", i+1)
				for j, nodeID := range cycle {
					node, _ := depGraph.GetNode(nodeID)
					if node != nil {
						fmt.Printf("  %d. %s (%s)\n", j+1, node.Address, node.Type)
					}
				}
			}
		}
		fmt.Println()
	} else {
		fmt.Println("✓ No circular dependencies detected")
	}
	
	// Get topological order
	order, err := depGraph.TopologicalSort()
	if err != nil && len(cycles) == 0 {
		return fmt.Errorf("failed to sort dependencies: %w", err)
	}
	
	if len(order) > 0 {
		fmt.Printf("\nDependency Order (%d resources):\n", len(order))
		for i, nodeID := range order {
			if i >= 20 && !cmd.Flag("verbose").Changed {
				fmt.Printf("  ... and %d more\n", len(order)-20)
				break
			}
			
			node, _ := depGraph.GetNode(nodeID)
			if node != nil {
				deps := depGraph.GetDependencies(nodeID)
				if len(deps) > 0 {
					fmt.Printf("  %d. %s → depends on %d resource(s)\n", i+1, nodeID, len(deps))
				} else {
					fmt.Printf("  %d. %s (no dependencies)\n", i+1, nodeID)
				}
			}
		}
	}
	
	// Find root and leaf nodes
	roots := depGraph.GetRootNodes()
	leaves := depGraph.GetLeafNodes()
	
	fmt.Printf("\nGraph Statistics:\n")
	fmt.Printf("  Total Resources: %d\n", len(depGraph.Nodes()))
	fmt.Printf("  Total Dependencies: %d\n", len(depGraph.Edges()))
	fmt.Printf("  Root Resources: %d\n", len(roots))
	fmt.Printf("  Leaf Resources: %d\n", len(leaves))
	fmt.Printf("  Circular Dependencies: %d\n", len(cycles))
	
	// Export if requested
	if analyzeOutput != "" {
		if err := exportAnalysisResult(depGraph, analyzeOutput, analyzeFormat); err != nil {
			return fmt.Errorf("failed to export results: %w", err)
		}
		fmt.Printf("\nResults exported to %s\n", analyzeOutput)
	}
	
	return nil
}

func runAnalyzeHealth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Get state
	state, err := getStateForAnalysis(ctx, args)
	if err != nil {
		return err
	}
	
	// Build dependency graph for health analysis
	builder := graph.NewDependencyGraphBuilder(state)
	depGraph, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}
	
	// Create health analyzer
	config := health.AnalyzerConfig{
		CheckSecurity:      true,
		CheckCompliance:    true,
		CheckBestPractices: true,
		CheckDeprecation:   true,
	}
	analyzer := health.NewResourceHealthAnalyzer(depGraph, config)
	
	fmt.Printf("Analyzing health for %d resources...\n\n", len(state.Resources))
	
	// Analyze each resource
	var issues []health.HealthIssue
	resourceCount := 0
	
	// Analyze resources directly from state.Resources
	for _, resource := range state.Resources {
		if analyzeProvider != "" && !strings.HasPrefix(resource.Type, analyzeProvider) {
			continue
		}
		
		resourceCount++
		// Analyze each instance of the resource
		for _, instance := range resource.Instances {
			result := analyzer.AnalyzeResource(&resource, &instance)
			
			// Filter by severity
			for _, issue := range result.Issues {
				if shouldIncludeIssue(issue, analyzeSeverity) {
					issues = append(issues, issue)
				}
			}
		}
	}
	
	// Sort issues by severity
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Severity > issues[j].Severity
	})
	
	// Display results
	if len(issues) == 0 {
		fmt.Printf("✓ No health issues found in %d resources\n", resourceCount)
	} else {
		fmt.Printf("Found %d health issues in %d resources:\n\n", len(issues), resourceCount)
		
		// Group by severity
		severityGroups := make(map[health.Severity][]health.HealthIssue)
		for _, issue := range issues {
			severityGroups[issue.Severity] = append(severityGroups[issue.Severity], issue)
		}
		
		// Display by severity
		severities := []health.Severity{
			health.SeverityCritical,
			health.SeverityHigh,
			health.SeverityMedium,
			health.SeverityLow,
		}
		
		for _, severity := range severities {
			if issues, ok := severityGroups[severity]; ok && len(issues) > 0 {
				fmt.Printf("%s (%d issues):\n", getSeverityLabel(severity), len(issues))
				
				for i, issue := range issues {
					if i >= 5 && !cmd.Flag("verbose").Changed {
						fmt.Printf("  ... and %d more\n", len(issues)-5)
						break
					}
					
					fmt.Printf("  • [%s] %s: %s\n", issue.Category, issue.ResourceID, issue.Message)
					if issue.Recommendation != "" {
						fmt.Printf("    → %s\n", issue.Recommendation)
					}
				}
				fmt.Println()
			}
		}
		
		// Summary
		fmt.Println("Summary:")
		fmt.Printf("  Critical: %d\n", len(severityGroups[health.SeverityCritical]))
		fmt.Printf("  High: %d\n", len(severityGroups[health.SeverityHigh]))
		fmt.Printf("  Medium: %d\n", len(severityGroups[health.SeverityMedium]))
		fmt.Printf("  Low: %d\n", len(severityGroups[health.SeverityLow]))
	}
	
	// Export if requested
	if analyzeOutput != "" {
		if err := exportAnalysisResult(issues, analyzeOutput, analyzeFormat); err != nil {
			return fmt.Errorf("failed to export results: %w", err)
		}
		fmt.Printf("\nResults exported to %s\n", analyzeOutput)
	}
	
	return nil
}

func runAnalyzeCost(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Get state
	state, err := getStateForAnalysis(ctx, args)
	if err != nil {
		return err
	}
	
	// Create cost analyzer
	analyzer := cost.NewCostAnalyzer()
	
	fmt.Printf("Analyzing costs for resources...\n\n")
	
	// Analyze costs
	totalMonthlyCost := 0.0
	totalAnnualCost := 0.0
	resourceCosts := make([]cost.ResourceCost, 0)
	optimizations := make([]cost.OptimizationRecommendation, 0)
	
	for _, module := range state.Modules {
		for _, resource := range module.Resources {
			if analyzeProvider != "" && !strings.HasPrefix(resource.Type, analyzeProvider) {
				continue
			}
			
			// Calculate cost
			resourceCost, err := analyzer.CalculateResourceCost(ctx, &resource)
			if err == nil && resourceCost != nil && resourceCost.MonthlyCost > 0 {
				resourceCosts = append(resourceCosts, *resourceCost)
				totalMonthlyCost += resourceCost.MonthlyCost
				totalAnnualCost += resourceCost.AnnualCost
			}
			
			// Optimization recommendations would go here
		}
	}
	
	// Sort by cost
	sort.Slice(resourceCosts, func(i, j int) bool {
		return resourceCosts[i].MonthlyCost > resourceCosts[j].MonthlyCost
	})
	
	// Display top cost resources
	fmt.Println("Top Cost Resources:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-40s %-15s %-12s %-12s\n", "Resource", "Type", "Monthly", "Annual")
	fmt.Println(strings.Repeat("-", 80))
	
	displayCount := 10
	if cmd.Flag("verbose").Changed {
		displayCount = len(resourceCosts)
	}
	
	for i, rc := range resourceCosts {
		if i >= displayCount {
			fmt.Printf("... and %d more resources\n", len(resourceCosts)-displayCount)
			break
		}
		
		fmt.Printf("%-40s %-15s $%-11.2f $%-11.2f\n",
			truncateString(rc.ResourceAddress, 40),
			truncateString(rc.ResourceType, 15),
			rc.MonthlyCost,
			rc.AnnualCost,
		)
		
		// Usage metrics details would go here if verbose
	}
	
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-40s %-15s $%-11.2f $%-11.2f\n", "TOTAL", "", totalMonthlyCost, totalAnnualCost)
	
	// Display optimizations
	if len(optimizations) > 0 {
		fmt.Printf("\n\nCost Optimization Recommendations:\n")
		fmt.Println(strings.Repeat("-", 80))
		
		// Sort by potential savings
		sort.Slice(optimizations, func(i, j int) bool {
			return optimizations[i].EstimatedSavings > optimizations[j].EstimatedSavings
		})
		
		totalSavings := 0.0
		for i, opt := range optimizations {
			if i >= 10 && !cmd.Flag("verbose").Changed {
				fmt.Printf("\n... and %d more recommendations\n", len(optimizations)-10)
				break
			}
			
			fmt.Printf("\n%d. %s\n", i+1, opt.RecommendationType)
			fmt.Printf("   Resource: %s\n", opt.ResourceAddress)
			fmt.Printf("   Impact: %s\n", opt.Impact)
			fmt.Printf("   Potential Savings: $%.2f/month ($%.2f/year)\n",
				opt.EstimatedSavings,
				opt.EstimatedSavings*12)
			fmt.Printf("   Recommendation: %s\n", opt.Description)
			
			totalSavings += opt.EstimatedSavings
		}
		
		fmt.Printf("\nTotal Potential Savings: $%.2f/month ($%.2f/year)\n", totalSavings, totalSavings*12)
	}
	
	// Export if requested
	if analyzeOutput != "" {
		result := map[string]interface{}{
			"total_monthly_cost": totalMonthlyCost,
			"total_annual_cost":  totalAnnualCost,
			"resources":          resourceCosts,
			"optimizations":      optimizations,
		}
		
		if err := exportAnalysisResult(result, analyzeOutput, analyzeFormat); err != nil {
			return fmt.Errorf("failed to export results: %w", err)
		}
		fmt.Printf("\nResults exported to %s\n", analyzeOutput)
	}
	
	return nil
}

func runAnalyzeBlastRadius(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	resourceID := args[0]
	
	// Get state
	state, err := getStateForAnalysis(ctx, []string{})
	if err != nil {
		return err
	}
	
	// Build dependency graph
	builder := graph.NewDependencyGraphBuilder(state)
	depGraph, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}
	
	// Calculate blast radius
	radius := depGraph.CalculateBlastRadius(resourceID)
	
	if radius == nil {
		return fmt.Errorf("resource %s not found in state", resourceID)
	}
	
	fmt.Printf("Blast Radius Analysis for %s\n", resourceID)
	fmt.Println(strings.Repeat("=", 60))
	
	// Direct impact
	fmt.Printf("\nDirect Impact (%d resources):\n", len(radius.DirectImpact))
	for _, id := range radius.DirectImpact {
		node, _ := depGraph.GetNode(id)
		if node != nil {
			fmt.Printf("  • %s (%s)\n", id, node.Type)
		}
	}
	
	// Indirect impact
	if len(radius.IndirectImpact) > 0 {
		fmt.Printf("\nIndirect Impact (%d resources):\n", len(radius.IndirectImpact))
		displayCount := 10
		if cmd.Flag("verbose").Changed {
			displayCount = len(radius.IndirectImpact)
		}
		
		for i, id := range radius.IndirectImpact {
			if i >= displayCount {
				fmt.Printf("  ... and %d more\n", len(radius.IndirectImpact)-displayCount)
				break
			}
			
			node, _ := depGraph.GetNode(id)
			if node != nil {
				fmt.Printf("  • %s (%s)\n", id, node.Type)
			}
		}
	}
	
	// Summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total Affected Resources: %d\n", radius.TotalAffected)
	fmt.Printf("  Direct Dependencies: %d\n", len(radius.DirectImpact))
	fmt.Printf("  Indirect Dependencies: %d\n", len(radius.IndirectImpact))
	fmt.Printf("  Impact Radius: %d levels\n", radius.MaxDepth)
	
	// Critical resources
	criticalTypes := []string{"database", "storage", "network", "security"}
	criticalCount := 0
	
	for _, id := range append(radius.DirectImpact, radius.IndirectImpact...) {
		node, _ := depGraph.GetNode(id)
		if node != nil {
			for _, critical := range criticalTypes {
				if strings.Contains(strings.ToLower(node.Type), critical) {
					criticalCount++
					break
				}
			}
		}
	}
	
	if criticalCount > 0 {
		fmt.Printf("\n⚠️  Warning: %d critical resources in blast radius\n", criticalCount)
	}
	
	return nil
}

func getStateForAnalysis(ctx context.Context, args []string) (*state.TerraformState, error) {
	var stateData []byte
	var err error
	
	if len(args) > 0 && args[0] != "" {
		// Read from file
		stateData, err = os.ReadFile(args[0])
		if err != nil {
			return nil, fmt.Errorf("failed to read state file: %w", err)
		}
	} else if analyzeBackend != "" {
		// Backend functionality not yet implemented
		return nil, fmt.Errorf("backend functionality not yet available")
	} else {
		// Try default terraform.tfstate
		stateData, err = os.ReadFile("terraform.tfstate")
		if err != nil {
			return nil, fmt.Errorf("no state file specified and terraform.tfstate not found")
		}
	}
	
	// Parse state
	stateParser := state.NewParser()
	state, err := stateParser.Parse(stateData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}
	
	return state, nil
}

func shouldIncludeIssue(issue health.HealthIssue, minSeverity string) bool {
	severityMap := map[string]health.Severity{
		"low":      health.SeverityLow,
		"medium":   health.SeverityMedium,
		"high":     health.SeverityHigh,
		"critical": health.SeverityCritical,
	}
	
	minSev, ok := severityMap[strings.ToLower(minSeverity)]
	if !ok {
		minSev = health.SeverityLow
	}
	
	return issue.Severity >= minSev
}

func getSeverityLabel(severity health.Severity) string {
	labels := map[health.Severity]string{
		health.SeverityCritical: "🔴 CRITICAL",
		health.SeverityHigh:     "🟠 HIGH",
		health.SeverityMedium:   "🟡 MEDIUM",
		health.SeverityLow:      "🟢 LOW",
	}
	
	if label, ok := labels[severity]; ok {
		return label
	}
	return "UNKNOWN"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func exportAnalysisResult(data interface{}, outputPath string, format string) error {
	var output []byte
	var err error
	
	switch strings.ToLower(format) {
	case "json":
		output, err = json.MarshalIndent(data, "", "  ")
	case "yaml":
		// Would need to import a YAML library
		return fmt.Errorf("YAML format not yet implemented")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return err
	}
	
	return os.WriteFile(outputPath, output, 0644)
}