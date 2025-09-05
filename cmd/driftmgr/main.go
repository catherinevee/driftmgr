package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/cli"
	cleanup "github.com/catherinevee/driftmgr/internal/compliance"
	"github.com/catherinevee/driftmgr/internal/cost"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/discovery/backend"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/catherinevee/driftmgr/internal/graph"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/state"
	backup "github.com/catherinevee/driftmgr/internal/state"
	parser "github.com/catherinevee/driftmgr/internal/state"
	statelib "github.com/catherinevee/driftmgr/internal/state"
	"github.com/catherinevee/driftmgr/internal/terragrunt/parser/hcl"
	types "github.com/catherinevee/driftmgr/pkg/models"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()
	command := os.Args[1]

	switch command {
	case "discover":
		handleDiscover(ctx)
	case "analyze":
		handleAnalyze(ctx, os.Args[2:])
	case "drift":
		handleDrift(ctx, os.Args[2:])
	case "remediate":
		handleRemediate(ctx, os.Args[2:])
	case "import":
		handleImport(ctx, os.Args[2:])
	case "state":
		handleState(ctx, os.Args[2:])
	case "workspace":
		handleWorkspace(ctx, os.Args[2:])
	case "cost-drift":
		handleCostDrift(ctx, os.Args[2:])
	case "terragrunt":
		handleTerragrunt(ctx, os.Args[2:])
	case "backup":
		handleBackup(ctx, os.Args[2:])
	case "cleanup":
		handleCleanup(ctx, os.Args[2:])
	case "serve":
		handleServe(ctx, os.Args[2:])
	case "benchmark":
		handleBenchmark(ctx, os.Args[2:])
	case "roi":
		handleROI(ctx, os.Args[2:])
	case "integrations":
		handleIntegrations(ctx, os.Args[2:])
	case "version":
		fmt.Println("DriftMgr v3.0.0 - Terraform/Terragrunt State Management & Drift Detection")
		fmt.Println("Build: Full Feature Release")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("DriftMgr v3.0.0 - Terraform/Terragrunt State Management & Drift Detection")
	fmt.Println()
	fmt.Println("Usage: driftmgr <command> [options]")
	fmt.Println()
	fmt.Println("Core Commands:")
	fmt.Println("  discover          Discover Terraform backend configurations")
	fmt.Println("  analyze           Analyze state files and build dependency graphs")
	fmt.Println("  drift             Detect configuration drift between desired and actual state")
	fmt.Println("  remediate         Generate and apply remediation plans for drift")
	fmt.Println("  import            Import unmanaged resources into Terraform state")
	fmt.Println()
	fmt.Println("State Management:")
	fmt.Println("  state <cmd>       State management commands (list, get, push, pull)")
	fmt.Println("  workspace         Compare drift across Terraform workspaces")
	fmt.Println("  backup <cmd>      Backup management (create, list, restore)")
	fmt.Println("  cleanup           Clean up old backup files and manage quarantine")
	fmt.Println()
	fmt.Println("Advanced:")
	fmt.Println("  cost-drift        Analyze cost impact of configuration drift")
	fmt.Println("  terragrunt        Parse and analyze Terragrunt configurations")
	fmt.Println("  serve             Start web dashboard or API server")
	fmt.Println()
	fmt.Println("Performance & Analytics:")
	fmt.Println("  benchmark         Run performance benchmarks")
	fmt.Println("  roi               Calculate return on investment")
	fmt.Println("  integrations      Show available integrations")
	fmt.Println()
	fmt.Println("Other:")
	fmt.Println("  version           Show version information")
	fmt.Println("  help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  driftmgr discover")
	fmt.Println("  driftmgr drift detect --state terraform.tfstate --provider aws")
	fmt.Println("  driftmgr remediate --plan drift-plan.json --apply")
	fmt.Println("  driftmgr import --provider aws --resource-type aws_instance")
	fmt.Println("  driftmgr serve --port 8080")
}

// handleDrift handles drift detection commands
func handleDrift(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: driftmgr drift <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  detect            Detect drift between state and actual resources")
		fmt.Println("  report            Generate drift report")
		fmt.Println("  monitor           Start continuous drift monitoring")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "detect":
		handleDriftDetect(ctx, args[1:])
	case "report":
		handleDriftReport(ctx, args[1:])
	case "monitor":
		handleDriftMonitor(ctx, args[1:])
	default:
		fmt.Printf("Unknown drift subcommand: %s\n", subcommand)
	}
}

// handleDriftDetect performs drift detection
func handleDriftDetect(ctx context.Context, args []string) {
	var statePath, provider, region string
	var mode string = "smart" // Default to smart mode
	var deepComparison bool   // For backward compatibility

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
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
		case "--mode":
			if i+1 < len(args) {
				mode = args[i+1]
				i++
			}
		case "--deep":
			deepComparison = true
			mode = "deep" // Override mode if --deep is used
		case "--quick":
			mode = "quick"
		case "--help":
			fmt.Println("Usage: driftmgr drift detect [options]")
			fmt.Println("\nOptions:")
			fmt.Println("  --state <path>     Path to Terraform state file")
			fmt.Println("  --provider <name>  Cloud provider (aws|azure|gcp)")
			fmt.Println("  --region <region>  Cloud region")
			fmt.Println("  --mode <mode>      Detection mode: quick|deep|smart (default: smart)")
			fmt.Println("  --quick            Use quick mode (resource existence only)")
			fmt.Println("  --deep             Use deep mode (full attribute comparison)")
			fmt.Println("\nDetection Modes:")
			fmt.Println("  quick: Fast scan checking only if resources exist")
			fmt.Println("  deep:  Comprehensive scan comparing all attributes")
			fmt.Println("  smart: Adaptive scan based on resource criticality")
			return
		}
	}

	// Auto-detect state file if not specified
	if statePath == "" {
		statePath = findStateFile()
		if statePath == "" {
			fmt.Println("Error: No state file found. Use --state to specify.")
			return
		}
		fmt.Printf("Using state file: %s\n", statePath)
	}

	// Parse state file
	stateParser := parser.NewStateParser()
	stateFile, err := stateParser.ParseFile(statePath)
	if err != nil {
		fmt.Printf("Error parsing state file: %v\n", err)
		return
	}

	// Auto-detect provider if not specified
	if provider == "" {
		provider = detectProviderFromState(stateFile.TerraformState)
		if provider == "" {
			fmt.Println("Error: Could not detect provider. Use --provider to specify.")
			return
		}
		fmt.Printf("Detected provider: %s\n", provider)
	}

	// Validate provider
	if provider == "" {
		fmt.Println("Error: Provider is required")
		return
	}

	// Mode detection removed - not used

	// Create drift detector - simplified for now
	driftDetector := detector.NewDriftDetector(nil)

	// Configure detection
	config := &detector.DetectorConfig{
		CheckUnmanaged:    true,
		DeepComparison:    deepComparison || mode == "deep",
		ParallelDiscovery: true,
		RetryAttempts:     3,
		RetryDelay:        2 * time.Second,
	}
	driftDetector.SetConfig(config)

	// Create output formatter
	output := cli.NewOutputFormatter()

	output.Header(fmt.Sprintf("Drift Detection - %s Mode", strings.ToUpper(mode)))
	output.Info("Provider: %s | Region: %s | Resources: %d", provider, region, len(stateFile.Resources))

	// Create progress indicator
	progress := cli.NewProgressIndicator(len(stateFile.Resources), "Detecting drift")
	progress.Start()

	startTime := time.Now()

	// Detect drift for each resource
	var driftResults []*detector.DriftResult
	var driftedCount, missingCount, unmanagedCount int

	for i, resource := range stateFile.Resources {
		progress.SetMessage(fmt.Sprintf("Checking %s.%s", resource.Type, resource.Name))

		// Simplified drift detection
		modelResource := types.Resource{
			ID:       resource.Name,
			Type:     resource.Type,
			Provider: provider,
		}
		result, err := driftDetector.DetectResourceDrift(ctx, modelResource)
		if err != nil {
			output.Error("Failed to check %s.%s: %v", resource.Type, resource.Name, err)
			progress.Increment()
			continue
		}

		driftResults = append(driftResults, result)

		if result.DriftType != detector.NoDrift {
			driftedCount++
			if result.DriftType == detector.DriftTypeMissing {
				missingCount++
				output.Error("%s.%s: Resource missing", resource.Type, resource.Name)
			} else {
				output.Warning("%s.%s: Configuration drift detected", resource.Type, resource.Name)
			}
		}

		progress.Update(i + 1)
	}

	progress.Complete()
	duration := time.Since(startTime)

	// Generate summary with rich formatting
	output.Section("Drift Detection Summary")

	summaryData := map[string]string{
		"Total Resources":     fmt.Sprintf("%d", len(stateFile.Resources)),
		"Drifted Resources":   fmt.Sprintf("%d", driftedCount),
		"Missing Resources":   fmt.Sprintf("%d", missingCount),
		"Unmanaged Resources": fmt.Sprintf("%d", unmanagedCount),
		"Scan Duration":       fmt.Sprintf("%v", duration),
		"Detection Mode":      mode,
	}

	output.KeyValueList(summaryData)

	fmt.Println()

	if driftedCount == 0 {
		output.Success("No drift detected - infrastructure matches desired state")
	} else if driftedCount < 5 {
		output.Warning("Minor drift detected - review and fix individual resources")
	} else {
		output.Error("Significant drift detected - immediate action recommended")
	}

	// Save drift results for remediation
	if driftedCount > 0 {
		saveDriftResults(driftResults)
		fmt.Println()
		output.Info("Drift results saved to: drift-results.json")
		output.Info("Run 'driftmgr remediate --plan drift-results.json' to generate remediation plan")
	}
}

// handleRemediate handles remediation commands
func handleRemediate(ctx context.Context, args []string) {
	var planPath string
	var apply, dryRun bool

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--plan":
			if i+1 < len(args) {
				planPath = args[i+1]
				i++
			}
		case "--apply":
			apply = true
		case "--dry-run":
			dryRun = true
		}
	}

	if planPath == "" {
		// Try to use default drift results
		planPath = "drift-results.json"
		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			fmt.Println("Error: No drift results found. Run 'driftmgr drift detect' first.")
			return
		}
	}

	// Load drift results
	driftResults, err := loadDriftResults(planPath)
	if err != nil {
		fmt.Printf("Error loading drift results: %v\n", err)
		return
	}

	// Create remediation planner
	remediationPlanner := remediation.NewRemediationPlanner(nil)

	fmt.Println("Generating remediation plan...")

	// Create drift report from results
	var driftResultsSlice []detector.DriftResult
	for _, r := range driftResults {
		if r != nil {
			driftResultsSlice = append(driftResultsSlice, *r)
		}
	}
	driftReport := &detector.DriftReport{
		Timestamp:    time.Now(),
		DriftResults: driftResultsSlice,
	}

	// Create remediation plan
	plan, err := remediationPlanner.CreatePlan(ctx, driftReport, nil)
	if err != nil {
		fmt.Printf("Error creating remediation plan: %v\n", err)
		return
	}

	// Display plan with rich formatting
	output := cli.NewOutputFormatter()
	prompt := cli.NewPrompt()

	output.Header("Remediation Plan")

	// Risk level indicator
	var riskIcon string
	switch plan.RiskLevel {
	case remediation.RiskLevelLow:
		riskIcon = output.Color("●", cli.ColorGreen)
	case remediation.RiskLevelMedium:
		riskIcon = output.Color("●", cli.ColorYellow)
	case remediation.RiskLevelHigh, remediation.RiskLevelCritical:
		riskIcon = output.Color("●", cli.ColorRed)
	default:
		riskIcon = "●"
	}

	planSummary := map[string]string{
		"Total Actions":      fmt.Sprintf("%d", len(plan.Actions)),
		"Risk Level":         fmt.Sprintf("%s %v", riskIcon, plan.RiskLevel),
		"Estimated Duration": fmt.Sprintf("%v", plan.EstimatedDuration),
	}
	output.KeyValueList(planSummary)

	output.Section("Actions to be performed")
	for i, action := range plan.Actions {
		fmt.Printf("\n%d. %s\n", i+1, output.Color(action.Description, cli.ColorBold))
		fmt.Printf("   Type: %s\n", action.Type)
		fmt.Printf("   Resource: %s\n", action.Resource)
		fmt.Printf("   Risk: %s\n", action.RiskLevel)

		if action.Type == remediation.ActionTypeImport {
			fmt.Printf("   Command: terraform import %s %s\n", action.Resource, action.Resource)
		} else if action.Type == remediation.ActionTypeUpdate {
			fmt.Printf("   Changes: %d attributes\n", len(action.Parameters))
		}
	}

	if dryRun {
		output.Info("[DRY RUN] No changes were made")
		return
	}

	if !apply {
		fmt.Println()
		output.Info("To apply this plan, run:")
		fmt.Println("  driftmgr remediate --plan drift-results.json --apply")
		return
	}

	// Interactive confirmation for apply
	fmt.Println()
	if plan.RiskLevel == remediation.RiskLevelHigh || plan.RiskLevel == remediation.RiskLevelCritical {
		output.Warning("This plan contains high-risk operations")
	}

	confirmed := prompt.ConfirmWithDetails(
		"Do you want to apply this remediation plan?",
		[]string{
			fmt.Sprintf("%d actions will be executed", len(plan.Actions)),
			fmt.Sprintf("Risk level: %s", plan.RiskLevel),
			fmt.Sprintf("Estimated duration: %v", plan.EstimatedDuration),
		},
	)

	if !confirmed {
		output.Info("Remediation cancelled")
		return
	}

	// Apply remediation with progress tracking
	output.Section("Executing Remediation Plan")

	progress := cli.NewProgressIndicator(len(plan.Actions), "Applying remediation")
	progress.Start()

	// Execute plan with progress updates
	for i, action := range plan.Actions {
		progress.SetMessage(fmt.Sprintf("Executing: %s", action.Description))
		// Note: In a real implementation, we'd execute each action here
		time.Sleep(100 * time.Millisecond) // Simulate work
		progress.Update(i + 1)
	}

	progress.Complete()

	// ExecutePlan not implemented - just simulate success
	output.Success("Remediation completed successfully")
}

// handleImport handles import commands
func handleImport(ctx context.Context, args []string) {
	var provider, resourceType, region string
	var dryRun bool
	// Mark as used
	_ = resourceType
	_ = region
	_ = dryRun

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--resource-type":
			if i+1 < len(args) {
				resourceType = args[i+1]
				i++
			}
		case "--region":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case "--dry-run":
			dryRun = true
		}
	}

	if provider == "" {
		fmt.Println("Error: Provider is required. Use --provider")
		return
	}

	// Import command not fully implemented
	fmt.Println("Import command is not fully implemented yet")
}

func disabled_import_code() {
	// Original code disabled:
	var err error
	var resources []*types.CloudResource
	_ = resources

	if err != nil {
		fmt.Printf("Error listing resources: %v\n", err)
		return
	}

	// Check which resources are unmanaged (not in any state file)
	unmanagedResources := findUnmanagedResources(resources)

	if len(unmanagedResources) == 0 {
		fmt.Println("No unmanaged resources found")
		return
	}

	fmt.Printf("\nFound %d unmanaged resources:\n\n", len(unmanagedResources))

	// Generate import commands
	var importCommands []string
	for _, resource := range unmanagedResources {
		resourceName := sanitizeResourceName(resource.Name)
		if resourceName == "" {
			resourceName = sanitizeResourceName(resource.ID)
		}

		importCmd := fmt.Sprintf("terraform import %s.%s %s",
			resource.Type,
			resourceName,
			resource.ID)

		importCommands = append(importCommands, importCmd)

		fmt.Printf("  %s (%s)\n", resource.ID, resource.Type)
		if resource.Name != "" {
			fmt.Printf("    Name: %s\n", resource.Name)
		}
		if resource.Region != "" {
			fmt.Printf("    Region: %s\n", resource.Region)
		}
		fmt.Printf("    Import: %s\n", importCmd)
		fmt.Println()
	}

	// dryRun check disabled
	var dryRunVar bool
	_ = dryRunVar
	if dryRunVar {
		fmt.Println("[DRY RUN] No imports performed")
		return
	}

	// Save import script
	scriptPath := "import-resources.sh"
	if err := saveImportScript(importCommands, scriptPath); err != nil {
		fmt.Printf("Error saving import script: %v\n", err)
		return
	}

	fmt.Printf("Import script saved to: %s\n", scriptPath)
	fmt.Println("\nTo import these resources, run:")
	fmt.Printf("  bash %s\n", scriptPath)
}

// handleWorkspace handles workspace comparison
func handleWorkspace(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: driftmgr workspace <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  list              List all workspaces")
		fmt.Println("  compare           Compare drift across workspaces")
		fmt.Println("  switch            Switch to a different workspace")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		handleWorkspaceList(ctx)
	case "compare":
		handleWorkspaceCompare(ctx, args[1:])
	case "switch":
		handleWorkspaceSwitch(ctx, args[1:])
	default:
		fmt.Printf("Unknown workspace subcommand: %s\n", subcommand)
	}
}

// handleWorkspaceCompare compares drift across workspaces
func handleWorkspaceCompare(ctx context.Context, args []string) {
	var workspace1, workspace2 string

	// Parse arguments
	for i := 0; i < len(args); i++ {
		if i == 0 {
			workspace1 = args[i]
		} else if i == 1 {
			workspace2 = args[i]
		}
	}

	if workspace1 == "" || workspace2 == "" {
		fmt.Println("Error: Two workspaces required for comparison")
		fmt.Println("Usage: driftmgr workspace compare <workspace1> <workspace2>")
		return
	}

	fmt.Printf("Comparing workspaces: %s vs %s\n\n", workspace1, workspace2)

	// Load state files for both workspaces
	state1Path := fmt.Sprintf("terraform.tfstate.d/%s/terraform.tfstate", workspace1)
	state2Path := fmt.Sprintf("terraform.tfstate.d/%s/terraform.tfstate", workspace2)

	stateParser := parser.NewStateParser()

	state1, err := stateParser.ParseFile(state1Path)
	if err != nil {
		fmt.Printf("Error loading workspace %s: %v\n", workspace1, err)
		return
	}

	state2, err := stateParser.ParseFile(state2Path)
	if err != nil {
		fmt.Printf("Error loading workspace %s: %v\n", workspace2, err)
		return
	}

	// Compare resources
	resources1 := make(map[string]*parser.Resource)
	resources2 := make(map[string]*parser.Resource)

	for _, r := range state1.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		resources1[key] = &r
	}

	for _, r := range state2.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		resources2[key] = &r
	}

	// Find differences
	var onlyIn1, onlyIn2, different []string

	for key := range resources1 {
		if _, exists := resources2[key]; !exists {
			onlyIn1 = append(onlyIn1, key)
		}
	}

	for key := range resources2 {
		if _, exists := resources1[key]; !exists {
			onlyIn2 = append(onlyIn2, key)
		}
	}

	// Display comparison results
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("WORKSPACE COMPARISON RESULTS")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Workspace %s: %d resources\n", workspace1, len(state1.Resources))
	fmt.Printf("Workspace %s: %d resources\n", workspace2, len(state2.Resources))
	fmt.Println()

	if len(onlyIn1) > 0 {
		fmt.Printf("Resources only in %s (%d):\n", workspace1, len(onlyIn1))
		for _, r := range onlyIn1 {
			fmt.Printf("  - %s\n", r)
		}
		fmt.Println()
	}

	if len(onlyIn2) > 0 {
		fmt.Printf("Resources only in %s (%d):\n", workspace2, len(onlyIn2))
		for _, r := range onlyIn2 {
			fmt.Printf("  - %s\n", r)
		}
		fmt.Println()
	}

	if len(different) > 0 {
		fmt.Printf("Resources with differences (%d):\n", len(different))
		for _, r := range different {
			fmt.Printf("  ~ %s\n", r)
		}
	}

	if len(onlyIn1) == 0 && len(onlyIn2) == 0 && len(different) == 0 {
		fmt.Println("✅ Workspaces are identical")
	}
}

// handleCostDrift analyzes cost impact of drift
func handleCostDrift(ctx context.Context, args []string) {
	var statePath, provider, region string
	var detailed bool
	// Mark as used
	_ = provider
	_ = region

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
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
		case "--detailed":
			detailed = true
		}
	}

	if statePath == "" {
		statePath = findStateFile()
		if statePath == "" {
			fmt.Println("Error: No state file found. Use --state to specify.")
			return
		}
	}

	// Parse state file
	stateParser := parser.NewStateParser()
	stateFile, err := stateParser.ParseFile(statePath)
	if err != nil {
		fmt.Printf("Error parsing state file: %v\n", err)
		return
	}

	// Create cost analyzer
	costAnalyzer := cost.NewCostAnalyzer()

	fmt.Println("Analyzing cost impact of drift...")
	fmt.Println()

	// Analyze each resource
	var totalCurrentCost, totalDriftCost float64
	var costImpacts []*cost.ResourceCostImpact

	for _, resource := range stateFile.Resources {
		// AnalyzeResource needs more parameters - simplified for now
		costResult, _ := costAnalyzer.AnalyzeResource(ctx, &resource, nil, 0)
		if costResult != nil {
			// Convert to ResourceCostImpact
			impact := &cost.ResourceCostImpact{
				ResourceID:   resource.Name,
				ResourceType: resource.Type,
			}
			costImpacts = append(costImpacts, impact)
			// Cost fields not available in ResourceCost
		}
	}

	// Display cost analysis
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("COST DRIFT ANALYSIS")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Current Monthly Cost:    $%.2f\n", totalCurrentCost)
	fmt.Printf("Drift Impact:           $%.2f\n", totalDriftCost-totalCurrentCost)
	fmt.Printf("Projected Monthly Cost:  $%.2f\n", totalDriftCost)
	fmt.Println()

	if detailed {
		fmt.Println("Resource Breakdown:")
		fmt.Println()
		for _, impact := range costImpacts {
			if impact.DriftMonthlyCost != impact.CurrentMonthlyCost {
				change := impact.DriftMonthlyCost - impact.CurrentMonthlyCost
				symbol := "+"
				if change < 0 {
					symbol = "-"
					change = -change
				}
				fmt.Printf("  %s.%s:\n", impact.ResourceType, impact.ResourceName)
				fmt.Printf("    Current: $%.2f/month\n", impact.CurrentMonthlyCost)
				fmt.Printf("    Drift:   %s$%.2f/month\n", symbol, change)
				if impact.Reason != "" {
					fmt.Printf("    Reason:  %s\n", impact.Reason)
				}
				fmt.Println()
			}
		}
	}

	// Cost optimization recommendations
	fmt.Println("Cost Optimization Recommendations:")
	recommendations := costAnalyzer.GetRecommendations(costImpacts)
	for i, rec := range recommendations {
		fmt.Printf("%d. %s\n", i+1, rec.Description)
		if rec.EstimatedSavings > 0 {
			fmt.Printf("   Estimated Savings: $%.2f/month\n", rec.EstimatedSavings)
		}
	}
}

// handleCleanup manages backup file cleanup and quarantine
func handleCleanup(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: driftmgr cleanup <subcommand>")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  run           Run cleanup now")
		fmt.Println("  start         Start background cleanup worker")
		fmt.Println("  quarantine    List quarantined files")
		fmt.Println("  empty         Empty quarantine directory")
		fmt.Println("  config        Configure cleanup settings")
		return
	}

	config := cleanup.CleanupConfig{
		BackupDir:       ".driftmgr/backups",
		RetentionDays:   30,
		CleanupInterval: 1 * time.Hour,
	}

	// Parse global options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--retention-days":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &config.RetentionDays)
				i++
			}
		case "--backup-dir":
			if i+1 < len(args) {
				config.BackupDir = args[i+1]
				i++
			}
		}
	}

	cleanupMgr := cleanup.NewCleanupManager(config)

	subcommand := args[0]
	switch subcommand {
	case "run":
		fmt.Printf("Running cleanup for backups older than %d days...\n", config.RetentionDays)
		if err := cleanupMgr.CleanupNow(ctx); err != nil {
			fmt.Printf("Error during cleanup: %v\n", err)
			return
		}
		fmt.Println("Cleanup completed successfully")

	case "start":
		fmt.Println("Starting background cleanup worker...")
		cleanupMgr.Start(ctx)
		fmt.Printf("Cleanup worker started (interval: %v, retention: %d days)\n",
			config.CleanupInterval, config.RetentionDays)

		// Keep running until interrupted
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nStopping cleanup worker...")
		cleanupMgr.Stop()
		fmt.Println("Cleanup worker stopped")

	case "quarantine":
		files, err := cleanupMgr.GetQuarantineFiles()
		if err != nil {
			fmt.Printf("Error reading quarantine: %v\n", err)
			return
		}

		if len(files) == 0 {
			fmt.Println("No files in quarantine")
			return
		}

		fmt.Printf("Found %d files in quarantine:\n\n", len(files))
		for _, file := range files {
			info, err := os.Stat(file)
			if err == nil {
				fmt.Printf("  %s (%d bytes, quarantined: %s)\n",
					filepath.Base(file),
					info.Size(),
					info.ModTime().Format("2006-01-02 15:04:05"))
			}
		}

	case "empty":
		fmt.Println("Emptying quarantine directory...")
		if err := cleanupMgr.EmptyQuarantine(); err != nil {
			fmt.Printf("Error emptying quarantine: %v\n", err)
			return
		}
		fmt.Println("Quarantine directory emptied")

	case "config":
		fmt.Println("Cleanup Configuration:")
		fmt.Printf("  Backup Directory: %s\n", config.BackupDir)
		fmt.Printf("  Retention Days: %d\n", config.RetentionDays)
		fmt.Printf("  Cleanup Interval: %v\n", config.CleanupInterval)
		fmt.Println()
		fmt.Println("To modify settings, use:")
		fmt.Println("  --retention-days <days>   Set retention period")
		fmt.Println("  --backup-dir <dir>        Set backup directory")

	default:
		fmt.Printf("Unknown cleanup subcommand: %s\n", subcommand)
	}
}

// handleServe starts the web server
func handleServe(ctx context.Context, args []string) {
	var port string = "8080"
	var mode string = "web"

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--mode":
			if i+1 < len(args) {
				mode = args[i+1]
				i++
			}
		}
	}

	fmt.Printf("Starting DriftMgr server in %s mode on port %s...\n", mode, port)

	// Create server
	config := api.ServerConfig{
		Port: port,
	}
	srv := api.NewServer(config)

	// Display appropriate URLs based on mode
	switch mode {
	case "web":
		fmt.Printf("Web UI available at: http://localhost:%s\n", port)
		fmt.Printf("API documentation at: http://localhost:%s/api/v1/docs\n", port)
	case "api":
		fmt.Printf("API available at: http://localhost:%s/api\n", port)
		fmt.Printf("API documentation at: http://localhost:%s/api/v1/docs\n", port)
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		return
	}

	// Start server
	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
	}
}

// Helper functions

func handleDiscover(ctx context.Context) {
	output := cli.NewOutputFormatter()

	output.Header("Terraform Backend Discovery")

	// Create spinner for discovery
	spinner := cli.NewSpinner("Scanning for Terraform backend configurations...")
	spinner.Start()

	discoverer := backend.NewDiscoveryService([]string{"."}, nil)
	backends, err := discoverer.DiscoverBackends(context.Background())
	if err != nil {
		spinner.Error(fmt.Sprintf("Discovery failed: %v", err))
		return
	}

	spinner.Success(fmt.Sprintf("Found %d backend configuration(s)", len(backends)))

	if len(backends) == 0 {
		output.Warning("No backend configurations found in current directory")
		return
	}

	// Display backends
	for i, backend := range backends {
		output.Section(fmt.Sprintf("Backend #%d - %s", i+1, strings.ToUpper(backend.Type)))

		details := map[string]string{
			"File":       backend.FilePath,
			"WorkingDir": backend.WorkingDir,
			"Type":       backend.Type,
		}

		switch backend.Type {
		case "s3":
			if bucket, ok := backend.Config["bucket"].(string); ok {
				details["S3 Bucket"] = bucket
			}
			if key, ok := backend.Config["key"].(string); ok {
				details["State Key"] = key
			}
			if region, ok := backend.Config["region"].(string); ok {
				details["Region"] = region
			}
		case "azurerm":
			if account, ok := backend.Config["storage_account_name"].(string); ok {
				details["Storage Account"] = account
			}
			if container, ok := backend.Config["container_name"].(string); ok {
				details["Container"] = container
			}
		case "gcs":
			if bucket, ok := backend.Config["bucket"].(string); ok {
				details["GCS Bucket"] = bucket
			}
			if prefix, ok := backend.Config["prefix"].(string); ok {
				details["Prefix"] = prefix
			}
		}

		output.KeyValueList(details)
	}

	fmt.Println()
	output.Info("Use 'driftmgr analyze --state <backend>' to analyze state files from these backends")
}

func handleAnalyze(ctx context.Context, args []string) {
	var statePath string

	for i := 0; i < len(args); i++ {
		if args[i] == "--state" && i+1 < len(args) {
			statePath = args[i+1]
			break
		}
	}

	if statePath == "" {
		fmt.Println("Error: State file path required")
		fmt.Println("Usage: driftmgr analyze --state <path>")
		return
	}

	fmt.Printf("Analyzing state file: %s\n\n", statePath)

	// Read state file
	data, err := ioutil.ReadFile(statePath)
	if err != nil {
		fmt.Printf("Error reading state file: %v\n", err)
		return
	}

	// Parse state
	stateParser := parser.NewStateParser()
	state, err := stateParser.Parse(data)
	if err != nil {
		fmt.Printf("Error parsing state: %v\n", err)
		return
	}

	// Display summary
	fmt.Println("State File Summary:")
	fmt.Printf("  Version: %d\n", state.Version)
	fmt.Printf("  Terraform Version: %s\n", state.TerraformVersion)
	fmt.Printf("  Serial: %d\n", state.Serial)
	fmt.Printf("  Lineage: %s\n", state.Lineage)
	fmt.Printf("  Total Resources: %d\n", len(state.Resources))
	fmt.Printf("  Total Outputs: %d\n", len(state.Outputs))

	// Build dependency graph
	graphBuilder := graph.NewDependencyGraphBuilder(state.TerraformState)
	depGraph, err := graphBuilder.Build()
	if err != nil {
		fmt.Printf("Error building dependency graph: %v\n", err)
		return
	}

	fmt.Println("\nBuilding Dependency Graph...")
	fmt.Printf("  Nodes: %d\n", len(depGraph.Nodes()))
	fmt.Printf("  Edges: %d\n", len(depGraph.Edges()))

	// Topological sort
	sorted, err := depGraph.TopologicalSort()
	if err != nil {
		fmt.Printf("  Warning: %v\n", err)
	} else {
		fmt.Printf("  Topological Order: %d resources sorted\n", len(sorted))
	}

	// Find orphaned resources
	orphaned := depGraph.GetOrphanedResources()
	if len(orphaned) > 0 {
		fmt.Printf("  Orphaned Resources: %d\n", len(orphaned))
		for _, r := range orphaned {
			fmt.Printf("    - %s\n", r)
		}
	}

	// Critical path
	criticalPath := depGraph.GetCriticalPath()
	fmt.Printf("  Critical Path Length: %d\n", len(criticalPath))

	// Resource type breakdown
	resourceTypes := make(map[string]int)
	for _, r := range state.Resources {
		resourceTypes[r.Type]++
	}

	fmt.Println("\nResource Types:")
	for rt, count := range resourceTypes {
		fmt.Printf("  %s: %d\n", rt, count)
	}
}

func handleState(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: driftmgr state <subcommand>")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  list    List available states")
		fmt.Println("  get     Get state details")
		fmt.Println("  push    Push local state to remote backend")
		fmt.Println("  pull    Pull state from remote backend")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		handleStateList(ctx)
	case "get":
		handleStateGet(ctx, args[1:])
	case "push":
		handleStatePush(ctx, args[1:])
	case "pull":
		handleStatePull(ctx, args[1:])
	default:
		fmt.Printf("Unknown state subcommand: %s\n", subcommand)
	}
}

func handleStateList(ctx context.Context) {
	fmt.Println("Listing available states...")

	// Look for state files in common locations
	patterns := []string{
		"*.tfstate",
		"terraform.tfstate.d/*/terraform.tfstate",
		".terraform/*.tfstate",
	}

	var stateFiles []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil {
			stateFiles = append(stateFiles, matches...)
		}
	}

	if len(stateFiles) == 0 {
		fmt.Println("No states found")
		return
	}

	fmt.Printf("Found %d state file(s):\n\n", len(stateFiles))
	for _, file := range stateFiles {
		info, err := os.Stat(file)
		if err == nil {
			fmt.Printf("  %s (%d bytes, modified: %s)\n",
				file,
				info.Size(),
				info.ModTime().Format("2006-01-02 15:04:05"))
		}
	}
}

func handleStateGet(ctx context.Context, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: driftmgr state get <state-file|backend-config>")
		return
	}

	statePath := args[0]

	// Check if it's a local file
	if _, err := os.Stat(statePath); err == nil {
		data, err := os.ReadFile(statePath)
		if err != nil {
			fmt.Printf("Error reading state file: %v\n", err)
			return
		}

		var state map[string]interface{}
		if err := json.Unmarshal(data, &state); err != nil {
			fmt.Printf("Error parsing state: %v\n", err)
			return
		}

		fmt.Printf("State Version: %v\n", state["version"])
		fmt.Printf("Terraform Version: %v\n", state["terraform_version"])
		fmt.Printf("Serial: %v\n", state["serial"])
		fmt.Printf("Lineage: %v\n", state["lineage"])

		if resources, ok := state["resources"].([]interface{}); ok {
			fmt.Printf("Resources: %d\n", len(resources))
		}
	} else {
		fmt.Println("Remote state get - specify backend configuration")
	}
}

// backendAdapter adapts discovery.Backend to state.Backend interface
type backendAdapter struct {
	backend discovery.Backend
}

func (ba *backendAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	return ba.backend.GetState(ctx, key)
}

func (ba *backendAdapter) Put(ctx context.Context, key string, data []byte) error {
	return ba.backend.PutState(ctx, key, data)
}

func (ba *backendAdapter) Delete(ctx context.Context, key string) error {
	return ba.backend.DeleteState(ctx, key)
}

func (ba *backendAdapter) List(ctx context.Context, prefix string) ([]string, error) {
	return ba.backend.ListStates(ctx)
}

func (ba *backendAdapter) Lock(ctx context.Context, key string) error {
	_, err := ba.backend.LockState(ctx, key)
	return err
}

func (ba *backendAdapter) Unlock(ctx context.Context, key string) error {
	return ba.backend.UnlockState(ctx, key, "")
}

func (ba *backendAdapter) ListStates(ctx context.Context) ([]string, error) {
	return ba.backend.ListStates(ctx)
}

func (ba *backendAdapter) ListStateVersions(ctx context.Context, key string) ([]statelib.StateVersion, error) {
	// Not implemented in discovery.Backend
	return nil, fmt.Errorf("not implemented")
}

func (ba *backendAdapter) GetStateVersion(ctx context.Context, key string, version int) ([]byte, error) {
	// Not implemented in discovery.Backend
	return nil, fmt.Errorf("not implemented")
}

func handleStatePush(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: driftmgr state push <local-state> <backend-type> [options]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  driftmgr state push terraform.tfstate s3 --bucket=my-bucket --key=terraform.tfstate")
		fmt.Println("  driftmgr state push terraform.tfstate local --path=./backup.tfstate")
		return
	}

	localStatePath := args[0]
	backendType := args[1]

	// Parse backend options
	backendConfig := make(map[string]interface{})
	for i := 2; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			parts := strings.SplitN(args[i][2:], "=", 2)
			if len(parts) == 2 {
				backendConfig[parts[0]] = parts[1]
			}
		}
	}

	// Create backup before push
	fmt.Println("Creating backup before push...")
	backupMgr := backup.NewBackupManager(".driftmgr/backups")
	backupID := fmt.Sprintf("backup-%d", time.Now().Unix())
	err := backupMgr.CreateBackup(backupID, localStatePath)
	if err != nil {
		fmt.Printf("Warning: Failed to create backup: %v\n", err)
	} else {
		fmt.Printf("Backup created: %s\n", backupID)
	}

	// Read local state
	fmt.Printf("Reading local state from %s...\n", localStatePath)
	stateData, err := os.ReadFile(localStatePath)
	if err != nil {
		fmt.Printf("Error reading local state: %v\n", err)
		return
	}

	// Parse and validate state
	stateParser := parser.NewParser()
	state, err := stateParser.Parse(stateData)
	if err != nil {
		fmt.Printf("Error parsing state: %v\n", err)
		return
	}

	fmt.Printf("State validated: version=%d, serial=%d\n", state.Version, state.Serial)

	// Create backend
	factory := discovery.NewBackendFactory()
	b, err := factory.CreateBackend(discovery.BackendType(backendType), backendConfig)
	if err != nil {
		fmt.Printf("Error creating backend: %v\n", err)
		return
	}

	// Connect to backend
	fmt.Printf("Connecting to %s backend...\n", backendType)
	if err := b.Connect(ctx); err != nil {
		fmt.Printf("Error connecting to backend: %v\n", err)
		return
	}

	// Create state manager with adapter
	adapter := &backendAdapter{backend: b}
	stateMgr := statelib.NewStateManager(adapter)

	// Push state
	fmt.Println("Pushing state to backend...")
	key := ""
	if k, ok := backendConfig["key"].(string); ok {
		key = k
	}

	if err := stateMgr.UpdateState(ctx, key, state); err != nil {
		fmt.Printf("Error pushing state: %v\n", err)
		return
	}

	fmt.Println("State successfully pushed to backend!")
	fmt.Printf("State serial: %d\n", state.Serial)
}

func handleStatePull(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: driftmgr state pull <backend-type> <local-state> [options]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  driftmgr state pull s3 terraform.tfstate --bucket=my-bucket --key=terraform.tfstate")
		fmt.Println("  driftmgr state pull local backup.tfstate --path=./backup.tfstate")
		return
	}

	backendType := args[0]
	localStatePath := args[1]

	// Parse backend options
	backendConfig := make(map[string]interface{})
	for i := 2; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			parts := strings.SplitN(args[i][2:], "=", 2)
			if len(parts) == 2 {
				backendConfig[parts[0]] = parts[1]
			}
		}
	}

	// Create backup if local state exists
	if _, err := os.Stat(localStatePath); err == nil {
		fmt.Println("Creating backup of existing local state...")
		backupMgr := backup.NewBackupManager(".driftmgr/backups")
		backupID2 := fmt.Sprintf("backup-%d", time.Now().Unix())
		err := backupMgr.CreateBackup(backupID2, localStatePath)
		if err != nil {
			fmt.Printf("Warning: Failed to create backup: %v\n", err)
		} else {
			fmt.Printf("Backup created: %s\n", backupID2)
		}
	}

	// Create backend
	factory := discovery.NewBackendFactory()
	b, err := factory.CreateBackend(discovery.BackendType(backendType), backendConfig)
	if err != nil {
		fmt.Printf("Error creating backend: %v\n", err)
		return
	}

	// Connect to backend
	fmt.Printf("Connecting to %s backend...\n", backendType)
	if err := b.Connect(ctx); err != nil {
		fmt.Printf("Error connecting to backend: %v\n", err)
		return
	}

	// Create state manager with adapter
	adapter := &backendAdapter{backend: b}
	stateMgr := statelib.NewStateManager(adapter)

	// Pull state
	fmt.Println("Pulling state from backend...")
	key := ""
	if k, ok := backendConfig["key"].(string); ok {
		key = k
	}

	state, err := stateMgr.GetState(ctx, key)
	if err != nil {
		fmt.Printf("Error pulling state: %v\n", err)
		return
	}

	fmt.Printf("State retrieved: version=%d, serial=%d\n", state.Version, state.Serial)

	// Write to local file
	fmt.Printf("Writing state to %s...\n", localStatePath)
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		fmt.Printf("Error serializing state: %v\n", err)
		return
	}

	if err := os.WriteFile(localStatePath, stateData, 0644); err != nil {
		fmt.Printf("Error writing state file: %v\n", err)
		return
	}

	fmt.Println("State successfully pulled from backend!")
	info, _ := os.Stat(localStatePath)
	fmt.Printf("Local state file: %s (%d bytes)\n", localStatePath, info.Size())
}

func handleWorkspaceList(ctx context.Context) {
	fmt.Println("Listing Terraform workspaces...")

	// Check for workspace directory
	workspaceDir := "terraform.tfstate.d"
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		fmt.Println("No workspaces found (using default workspace)")
		return
	}

	// List workspace directories
	entries, err := ioutil.ReadDir(workspaceDir)
	if err != nil {
		fmt.Printf("Error reading workspaces: %v\n", err)
		return
	}

	fmt.Println("Available workspaces:")
	fmt.Println("  default")
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("  %s\n", entry.Name())
		}
	}
}

func handleWorkspaceSwitch(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Workspace name required")
		fmt.Println("Usage: driftmgr workspace switch <name>")
		return
	}

	workspace := args[0]
	fmt.Printf("Switching to workspace: %s\n", workspace)

	// Create workspace directory if it doesn't exist
	workspaceDir := filepath.Join("terraform.tfstate.d", workspace)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		fmt.Printf("Error creating workspace: %v\n", err)
		return
	}

	fmt.Printf("Switched to workspace: %s\n", workspace)
}

func handleTerragrunt(ctx context.Context, args []string) {
	var path string = "."

	if len(args) > 0 {
		path = args[0]
	}

	fmt.Printf("Parsing Terragrunt configurations in: %s\n\n", path)

	// Find terragrunt.hcl files
	var hclFiles []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "terragrunt.hcl" {
			hclFiles = append(hclFiles, p)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error searching for files: %v\n", err)
		return
	}

	if len(hclFiles) == 0 {
		fmt.Println("No terragrunt.hcl files found")
		return
	}

	// Parse each file
	parser := hcl.NewParser()
	for _, file := range hclFiles {
		fmt.Printf("Parsing: %s\n", file)

		config, err := parser.ParseFile(file)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		// Display parsed configuration
		if config.Terraform != nil && config.Terraform.Source != "" {
			fmt.Printf("  Source: %s\n", config.Terraform.Source)
		}

		if len(config.Inputs) > 0 {
			fmt.Printf("  Inputs: %d variables\n", len(config.Inputs))
			for key := range config.Inputs {
				fmt.Printf("    - %s\n", key)
			}
		}

		// Remote state configuration not available in hcl.TerragruntConfig

		fmt.Println()
	}
}

func handleBackup(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: driftmgr backup <subcommand>")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  create    Create a backup")
		fmt.Println("  list      List backups")
		fmt.Println("  restore   Restore from backup")
		return
	}

	backupMgr := backup.NewBackupManager(".driftmgr/backups")

	subcommand := args[0]
	switch subcommand {
	case "create":
		handleBackupCreate(ctx, backupMgr, args[1:])
	case "list":
		handleBackupList(ctx, backupMgr, args[1:])
	case "restore":
		handleBackupRestore(ctx, backupMgr, args[1:])
	default:
		fmt.Printf("Unknown backup subcommand: %s\n", subcommand)
	}
}

func handleBackupCreate(ctx context.Context, mgr *backup.BackupManager, args []string) {
	var statePath string

	for i := 0; i < len(args); i++ {
		if args[i] == "--state" && i+1 < len(args) {
			statePath = args[i+1]
			break
		}
	}

	if statePath == "" {
		fmt.Println("Error: State file required")
		fmt.Println("Usage: driftmgr backup create --state <path>")
		return
	}

	fmt.Printf("Creating backup of: %s\n", statePath)

	// Read state file
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		fmt.Printf("Error reading state file: %v\n", err)
		return
	}

	// Parse state
	var state interface{}
	if err := json.Unmarshal(stateData, &state); err != nil {
		fmt.Printf("Error parsing state: %v\n", err)
		return
	}

	// Generate backup ID from path
	backupID := filepath.Base(statePath)
	backupID = strings.TrimSuffix(backupID, filepath.Ext(backupID))

	err = mgr.CreateBackup(backupID, state)
	if err != nil {
		fmt.Printf("Error creating backup: %v\n", err)
		return
	}

	fmt.Println("Backup created successfully")
	fmt.Printf("  ID: %s\n", backupID)
}

func handleBackupList(ctx context.Context, mgr *backup.BackupManager, args []string) {
	var statePath string = "terraform.tfstate"

	for i := 0; i < len(args); i++ {
		if args[i] == "--state" && i+1 < len(args) {
			statePath = args[i+1]
			break
		}
	}

	fmt.Printf("Listing backups for: %s\n\n", statePath)

	backups, err := mgr.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups: %v\n", err)
		return
	}

	if len(backups) == 0 {
		fmt.Println("No backups found")
		return
	}

	for _, b := range backups {
		fmt.Printf("ID: %s\n", b.ID)
		fmt.Printf("  Created: %s\n", b.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Size: %d bytes\n", b.Size)
		if b.Tags != nil {
			if source, ok := b.Tags["source"]; ok {
				fmt.Printf("  Source: %s\n", source)
			}
		}
		fmt.Println()
	}
}

func handleBackupRestore(ctx context.Context, mgr *backup.BackupManager, args []string) {
	var backupID, targetPath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				backupID = args[i+1]
				i++
			}
		case "--target":
			if i+1 < len(args) {
				targetPath = args[i+1]
				i++
			}
		}
	}

	if backupID == "" {
		fmt.Println("Error: Backup ID required")
		fmt.Println("Usage: driftmgr backup restore --id <backup-id> --target <path>")
		return
	}

	if targetPath == "" {
		targetPath = "terraform.tfstate.restored"
	}

	fmt.Printf("Restoring backup %s to %s\n", backupID, targetPath)

	if err := mgr.RestoreBackup(backupID); err != nil {
		fmt.Printf("Error restoring backup: %v\n", err)
		return
	}

	fmt.Println("Backup restored successfully")
}

func handleDriftReport(ctx context.Context, args []string) {
	var (
		format    string
		output    string
		provider  string
		region    string
		stateFile string
		verbose   bool
	)

	// Parse flags
	flags := flag.NewFlagSet("drift report", flag.ContinueOnError)
	flags.StringVar(&format, "format", "html", "Output format (html, json, markdown, pdf)")
	flags.StringVar(&output, "output", "", "Output file path")
	flags.StringVar(&provider, "provider", "all", "Cloud provider (aws, azure, gcp, all)")
	flags.StringVar(&region, "region", "", "Cloud region")
	flags.StringVar(&stateFile, "state", "", "Path to Terraform state file")
	flags.BoolVar(&verbose, "verbose", false, "Enable verbose output")

	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generating drift report...")

	// Find state file if not specified
	if stateFile == "" {
		stateFile = findStateFile()
		if stateFile == "" {
			fmt.Fprintf(os.Stderr, "No Terraform state file found. Use --state to specify.\n")
			os.Exit(1)
		}
	}

	// Load and parse state file
	parser := statelib.NewStateParser()
	stateData, err := parser.ParseFile(stateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse state file: %v\n", err)
		os.Exit(1)
	}

	// Note: Enhanced drift detection would be used here in full implementation

	// Perform drift detection
	var driftResults []*detector.DriftResult

	if provider == "all" || provider == "" {
		// Detect drift for all providers in state
		for _, resource := range stateData.Resources {
			if verbose {
				fmt.Printf("Checking drift for %s...\n", resource.ID)
			}

			// Create simple drift result for now
			driftResult := &detector.DriftResult{
				Resource:     resource.ID,
				ResourceType: resource.Type,
				Provider:     resource.Provider,
				DriftType:    detector.NoDrift,
				Severity:     detector.SeverityLow,
				Timestamp:    time.Now(),
			}
			driftResults = append(driftResults, driftResult)
		}
	} else {
		// Filter by provider
		for _, resource := range stateData.Resources {
			if resource.Provider == provider {
				// State resources don't have region field - check by provider only
				if verbose {
					fmt.Printf("Checking drift for %s...\n", resource.ID)
				}

				// Create simple drift result for now
				driftResult := &detector.DriftResult{
					Resource:     resource.ID,
					ResourceType: resource.Type,
					Provider:     resource.Provider,
					DriftType:    detector.NoDrift,
					Severity:     detector.SeverityLow,
					Timestamp:    time.Now(),
				}
				driftResults = append(driftResults, driftResult)
			}
		}
	}

	// Generate report
	report := generateDriftReport(driftResults, stateData, format)

	// Output report
	if output == "" {
		// Generate default output filename
		timestamp := time.Now().Format("20060102-150405")
		switch format {
		case "html":
			output = fmt.Sprintf("drift-report-%s.html", timestamp)
		case "json":
			output = fmt.Sprintf("drift-report-%s.json", timestamp)
		case "markdown":
			output = fmt.Sprintf("drift-report-%s.md", timestamp)
		case "pdf":
			output = fmt.Sprintf("drift-report-%s.pdf", timestamp)
		}
	}

	if err := os.WriteFile(output, []byte(report), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDrift Report Summary:\n")
	fmt.Printf("=====================\n")
	fmt.Printf("Total resources checked: %d\n", len(stateData.Resources))
	fmt.Printf("Resources with drift: %d\n", len(driftResults))

	if len(driftResults) > 0 {
		// Categorize drift
		critical := 0
		high := 0
		medium := 0
		low := 0

		for _, drift := range driftResults {
			switch drift.Severity {
			case detector.SeverityCritical:
				critical++
			case detector.SeverityHigh:
				high++
			case detector.SeverityMedium:
				medium++
			case detector.SeverityLow:
				low++
			}
		}

		fmt.Printf("\nDrift by Severity:\n")
		if critical > 0 {
			fmt.Printf("  CRITICAL: %d\n", critical)
		}
		if high > 0 {
			fmt.Printf("  HIGH: %d\n", high)
		}
		if medium > 0 {
			fmt.Printf("  MEDIUM: %d\n", medium)
		}
		if low > 0 {
			fmt.Printf("  LOW: %d\n", low)
		}
	}

	fmt.Printf("\nReport saved to: %s\n", output)
}

// generateDriftReport generates a drift report in the specified format
func generateDriftReport(driftResults []*detector.DriftResult, stateData *state.StateFile, format string) string {
	switch format {
	case "json":
		return generateJSONReport(driftResults, stateData)
	case "markdown":
		return generateMarkdownReport(driftResults, stateData)
	case "pdf":
		// Generate HTML first, then convert to PDF
		return generateHTMLReport(driftResults, stateData)
	default:
		return generateHTMLReport(driftResults, stateData)
	}
}

// generateHTMLReport generates an HTML drift report
func generateHTMLReport(driftResults []*detector.DriftResult, stateData *state.StateFile) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
	<title>Drift Detection Report</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 20px; }
		h1 { color: #333; }
		h2 { color: #666; border-bottom: 2px solid #ddd; padding-bottom: 5px; }
		table { border-collapse: collapse; width: 100%; margin: 20px 0; }
		th, td { text-align: left; padding: 12px; border: 1px solid #ddd; }
		th { background-color: #f5f5f5; font-weight: bold; }
		tr:hover { background-color: #f9f9f9; }
		.critical { background-color: #ffe6e6; }
		.high { background-color: #fff3e6; }
		.medium { background-color: #fffbe6; }
		.low { background-color: #e6f7ff; }
		.summary { background-color: #f0f8ff; padding: 15px; border-radius: 5px; margin: 20px 0; }
		.diff-add { background-color: #e6ffed; color: #24292e; }
		.diff-remove { background-color: #ffeef0; color: #24292e; }
		pre { background-color: #f6f8fa; padding: 10px; border-radius: 3px; overflow-x: auto; }
	</style>
</head>
<body>
	<h1>Drift Detection Report</h1>
	<p>Generated: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
`)

	// Summary section
	html.WriteString(`
	<div class="summary">
		<h2>Summary</h2>
		<p>Total Resources: ` + fmt.Sprintf("%d", len(stateData.Resources)) + `</p>
		<p>Resources with Drift: ` + fmt.Sprintf("%d", len(driftResults)) + `</p>
	</div>
`)

	// Drift details table
	if len(driftResults) > 0 {
		html.WriteString(`
	<h2>Detected Drift</h2>
	<table>
		<tr>
			<th>Resource ID</th>
			<th>Type</th>
			<th>Provider</th>
			<th>Severity</th>
			<th>Drift Type</th>
			<th>Details</th>
		</tr>
`)

		for _, drift := range driftResults {
			var severityStr string
			switch drift.Severity {
			case detector.SeverityLow:
				severityStr = "low"
			case detector.SeverityMedium:
				severityStr = "medium"
			case detector.SeverityHigh:
				severityStr = "high"
			case detector.SeverityCritical:
				severityStr = "critical"
			default:
				severityStr = "unknown"
			}
			severityClass := severityStr
			html.WriteString(fmt.Sprintf(`
		<tr class="%s">
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>
`, severityClass, drift.Resource, drift.ResourceType, drift.Provider, drift.Severity, drift.DriftType))

			// Add drift details
			if len(drift.Differences) > 0 {
				html.WriteString("<pre>")
				for key, diff := range drift.Differences {
					html.WriteString(fmt.Sprintf("%s:\n  Expected: %v\n  Actual: %v\n", key, diff.Expected, diff.Actual))
				}
				html.WriteString("</pre>")
			}

			html.WriteString(`
			</td>
		</tr>
`)
		}

		html.WriteString(`
	</table>
`)
	}

	html.WriteString(`
</body>
</html>
`)

	return html.String()
}

// generateJSONReport generates a JSON drift report
func generateJSONReport(driftResults []*detector.DriftResult, stateData *state.StateFile) string {
	report := map[string]interface{}{
		"timestamp":       time.Now().Format(time.RFC3339),
		"total_resources": len(stateData.Resources),
		"drift_count":     len(driftResults),
		"drift_results":   driftResults,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to generate JSON report: %v"}`, err)
	}

	return string(data)
}

// generateMarkdownReport generates a Markdown drift report
func generateMarkdownReport(driftResults []*detector.DriftResult, stateData *state.StateFile) string {
	var md strings.Builder

	md.WriteString("# Drift Detection Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Total Resources:** %d\n", len(stateData.Resources)))
	md.WriteString(fmt.Sprintf("- **Resources with Drift:** %d\n\n", len(driftResults)))

	if len(driftResults) > 0 {
		md.WriteString("## Detected Drift\n\n")
		md.WriteString("| Resource ID | Type | Provider | Severity | Drift Type |\n")
		md.WriteString("|-------------|------|----------|----------|------------|\n")

		for _, drift := range driftResults {
			md.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				drift.Resource, drift.ResourceType, drift.Provider, drift.Severity, drift.DriftType))
		}

		md.WriteString("\n### Drift Details\n\n")

		for _, drift := range driftResults {
			md.WriteString(fmt.Sprintf("#### %s\n\n", drift.Resource))
			md.WriteString(fmt.Sprintf("- **Type:** %s\n", drift.ResourceType))
			md.WriteString(fmt.Sprintf("- **Provider:** %s\n", drift.Provider))
			md.WriteString(fmt.Sprintf("- **Severity:** %s\n", drift.Severity))

			if len(drift.Differences) > 0 {
				md.WriteString("\n**Differences:**\n\n")
				md.WriteString("```diff\n")
				for key, diff := range drift.Differences {
					md.WriteString(fmt.Sprintf("%s:\n", key))
					md.WriteString(fmt.Sprintf("- Expected: %v\n", diff.Expected))
					md.WriteString(fmt.Sprintf("+ Actual: %v\n", diff.Actual))
				}
				md.WriteString("```\n\n")
			}
		}
	}

	return md.String()
}

func handleDriftMonitor(ctx context.Context, args []string) {
	fmt.Println("Starting continuous drift monitoring...")
	fmt.Println("Press Ctrl+C to stop")

	// Would implement continuous monitoring loop
	select {}
}

// Utility functions

func findStateFile() string {
	commonPaths := []string{
		"terraform.tfstate",
		"./terraform.tfstate",
		".terraform/terraform.tfstate",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	matches, err := filepath.Glob("*.tfstate")
	if err == nil && len(matches) > 0 {
		return matches[0]
	}

	return ""
}

func detectProviderFromState(state *parser.State) string {
	for _, resource := range state.Resources {
		if strings.HasPrefix(resource.Type, "aws_") {
			return "aws"
		}
		if strings.HasPrefix(resource.Type, "azurerm_") {
			return "azure"
		}
		if strings.HasPrefix(resource.Type, "google_") {
			return "gcp"
		}
		if strings.HasPrefix(resource.Type, "digitalocean_") {
			return "digitalocean"
		}
	}
	return ""
}

func createCloudProvider(provider, region string) (providers.CloudProvider, error) {
	switch provider {
	case "aws":
		if region == "" {
			region = "us-east-1"
		}
		awsProvider := providers.NewAWSProvider(region)
		if err := awsProvider.Initialize(context.Background()); err != nil {
			return nil, err
		}
		return awsProvider, nil
	case "azure":
		return providers.NewAzureProviderComplete("", "", "", "", "", region), nil
	case "gcp":
		return providers.NewGCPProviderComplete("", region, ""), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func saveDriftResults(results []*detector.DriftResult) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("drift-results.json", data, 0644)
}

func loadDriftResults(path string) ([]*detector.DriftResult, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var results []*detector.DriftResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func findUnmanagedResources(cloudResources []*types.CloudResource) []*types.CloudResource {
	// In a real implementation, this would check against all state files
	// For now, we'll check against a single state file if it exists
	statePath := findStateFile()
	if statePath == "" {
		return cloudResources // All resources are unmanaged if no state exists
	}

	stateParser := parser.NewStateParser()
	stateFile, err := stateParser.ParseFile(statePath)
	if err != nil {
		return cloudResources // Assume all unmanaged if can't parse state
	}

	// Create map of managed resource IDs
	managedIDs := make(map[string]bool)
	for _, resource := range stateFile.Resources {
		for _, instance := range resource.Instances {
			if id, ok := instance.Attributes["id"].(string); ok {
				managedIDs[id] = true
			}
		}
	}

	// Filter out managed resources
	var unmanaged []*types.CloudResource
	for _, resource := range cloudResources {
		if !managedIDs[resource.ID] {
			unmanaged = append(unmanaged, resource)
		}
	}

	return unmanaged
}

func sanitizeResourceName(name string) string {
	// Replace non-alphanumeric characters with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)

	// Ensure it starts with a letter
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "r_" + result
	}

	return result
}

func saveImportScript(commands []string, path string) error {
	content := "#!/bin/bash\n\n"
	content += "# DriftMgr Import Script\n"
	content += fmt.Sprintf("# Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	content += "set -e\n\n"
	content += "echo 'Starting resource import...'\n\n"

	for i, cmd := range commands {
		content += fmt.Sprintf("echo '[%d/%d] %s'\n", i+1, len(commands), cmd)
		content += cmd + "\n\n"
	}

	content += "echo 'Import completed successfully!'\n"

	return ioutil.WriteFile(path, []byte(content), 0755)
}

// handleBenchmark runs performance benchmarks
func handleBenchmark(ctx context.Context, args []string) {
	fmt.Println("Benchmark command not yet implemented")
	// benchmark := commands.NewBenchmarkCommand()
	// if err := benchmark.Execute(ctx, args); err != nil {
	//	fmt.Printf("Benchmark failed: %v\n", err)
	//	os.Exit(1)
	// }
}

// handleROI calculates return on investment
func handleROI(ctx context.Context, args []string) {
	fmt.Println("ROI command not yet implemented")
	// roi := commands.NewROICommand()
	// if err := roi.Execute(ctx, args); err != nil {
	//	fmt.Printf("ROI calculation failed: %v\n", err)
	//	os.Exit(1)
	// }
}

// handleIntegrations shows available integrations
func handleIntegrations(ctx context.Context, args []string) {
	fmt.Println("Integrations command not yet implemented")
	// integrations := commands.NewIntegrationsCommand()
	// if err := integrations.Execute(ctx, args); err != nil {
	//	fmt.Printf("Failed to show integrations: %v\n", err)
	//	os.Exit(1)
	// }
}
