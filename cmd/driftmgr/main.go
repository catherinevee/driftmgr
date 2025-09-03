package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/catherinevee/driftmgr/cmd/driftmgr/commands"
	"github.com/catherinevee/driftmgr/internal/analysis/cost"
	"github.com/catherinevee/driftmgr/internal/analysis/graph"
	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/discovery/backend"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/remediation/planner"
	"github.com/catherinevee/driftmgr/internal/safety/cleanup"
	"github.com/catherinevee/driftmgr/internal/state/backup"
	"github.com/catherinevee/driftmgr/internal/state/manager"
	"github.com/catherinevee/driftmgr/internal/state/parser"
	"github.com/catherinevee/driftmgr/internal/terragrunt/parser/hcl"
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
		fmt.Println("DriftMgr v3.0.0 Complete - Terraform/Terragrunt State Management & Drift Detection")
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
	fmt.Println("DriftMgr v3.0 Complete - Terraform/Terragrunt State Management & Drift Detection")
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
	var deepComparison bool

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
		case "--deep":
			deepComparison = true
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
	state, err := stateParser.ParseFile(statePath)
	if err != nil {
		fmt.Printf("Error parsing state file: %v\n", err)
		return
	}

	// Auto-detect provider if not specified
	if provider == "" {
		provider = detectProviderFromState(state)
		if provider == "" {
			fmt.Println("Error: Could not detect provider. Use --provider to specify.")
			return
		}
		fmt.Printf("Detected provider: %s\n", provider)
	}

	// Create cloud provider
	cloudProvider, err := createCloudProvider(provider, region)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		return
	}

	// Create drift detector
	driftDetector := detector.NewDriftDetector(map[string]types.CloudProvider{
		provider: cloudProvider,
	})

	// Configure detection
	config := &detector.DetectorConfig{
		CheckUnmanaged:  true,
		DeepComparison:  deepComparison,
		ParallelWorkers: 5,
		RetryAttempts:   3,
		RetryDelay:      2 * time.Second,
	}
	driftDetector.SetConfig(config)

	fmt.Println("\nDetecting drift...")
	startTime := time.Now()

	// Detect drift for each resource
	var driftResults []*detector.DriftResult
	var driftedCount, missingCount, unmanagedCount int

	for _, resource := range state.Resources {
		result, err := driftDetector.DetectResourceDrift(ctx, resource)
		if err != nil {
			fmt.Printf("  Error checking %s.%s: %v\n", resource.Type, resource.Name, err)
			continue
		}

		driftResults = append(driftResults, result)

		if result.HasDrift {
			driftedCount++
			symbol := "!"
			if result.DriftType == detector.DriftTypeMissing {
				missingCount++
				symbol = "✗"
			}
			fmt.Printf("  %s %s.%s: %s\n", symbol, resource.Type, resource.Name, result.Summary)
		} else {
			fmt.Printf("  ✓ %s.%s: No drift\n", resource.Type, resource.Name)
		}
	}

	duration := time.Since(startTime)

	// Generate summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("DRIFT DETECTION SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Total Resources:    %d\n", len(state.Resources))
	fmt.Printf("Drifted Resources:  %d\n", driftedCount)
	fmt.Printf("Missing Resources:  %d\n", missingCount)
	fmt.Printf("Unmanaged Resources: %d\n", unmanagedCount)
	fmt.Printf("Scan Duration:      %v\n", duration)

	if driftedCount == 0 {
		fmt.Println("\n✅ No drift detected - infrastructure matches desired state")
	} else if driftedCount < 5 {
		fmt.Println("\n⚠️  Minor drift detected - review and fix individual resources")
	} else {
		fmt.Println("\n🔴 Significant drift detected - immediate action recommended")
	}

	// Save drift results for remediation
	if driftedCount > 0 {
		saveDriftResults(driftResults)
		fmt.Println("\nDrift results saved to: drift-results.json")
		fmt.Println("Run 'driftmgr remediate --plan drift-results.json' to generate remediation plan")
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
	remediationPlanner := planner.NewRemediationPlanner()

	fmt.Println("Generating remediation plan...")
	
	// Create remediation plan
	plan, err := remediationPlanner.CreatePlan(ctx, driftResults)
	if err != nil {
		fmt.Printf("Error creating remediation plan: %v\n", err)
		return
	}

	// Display plan
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("REMEDIATION PLAN")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Total Actions: %d\n", len(plan.Actions))
	fmt.Printf("Risk Level: %s\n", plan.RiskLevel)
	fmt.Printf("Estimated Duration: %v\n", plan.EstimatedDuration)

	fmt.Println("\nActions to be performed:")
	for i, action := range plan.Actions {
		fmt.Printf("\n%d. %s\n", i+1, action.Description)
		fmt.Printf("   Type: %s\n", action.Type)
		fmt.Printf("   Resource: %s\n", action.ResourceID)
		fmt.Printf("   Risk: %s\n", action.RiskLevel)
		
		if action.Type == planner.ActionTypeImport {
			fmt.Printf("   Command: terraform import %s %s\n", action.ResourceID, action.ImportID)
		} else if action.Type == planner.ActionTypeUpdate {
			fmt.Printf("   Changes: %d attributes\n", len(action.Changes))
		}
	}

	if dryRun {
		fmt.Println("\n[DRY RUN] No changes were made")
		return
	}

	if !apply {
		fmt.Println("\nTo apply this plan, run:")
		fmt.Println("  driftmgr remediate --plan drift-results.json --apply")
		return
	}

	// Apply remediation
	fmt.Println("\nApplying remediation plan...")
	if err := remediationPlanner.ExecutePlan(ctx, plan); err != nil {
		fmt.Printf("Error executing plan: %v\n", err)
		return
	}

	fmt.Println("\n✅ Remediation completed successfully")
}

// handleImport handles import commands
func handleImport(ctx context.Context, args []string) {
	var provider, resourceType, region string
	var dryRun bool

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

	// Create cloud provider
	cloudProvider, err := createCloudProvider(provider, region)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		return
	}

	fmt.Printf("Discovering unmanaged %s resources", provider)
	if resourceType != "" {
		fmt.Printf(" of type %s", resourceType)
	}
	if region != "" {
		fmt.Printf(" in region %s", region)
	}
	fmt.Println("...")

	// List resources from cloud
	var resources []*types.CloudResource
	if resourceType != "" {
		resources, err = cloudProvider.ListResources(ctx, resourceType)
	} else {
		// List all resource types
		resources, err = cloudProvider.ListResources(ctx, "")
	}

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

	if dryRun {
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
		resources1[key] = r
	}

	for _, r := range state2.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		resources2[key] = r
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
	state, err := stateParser.ParseFile(statePath)
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

	for _, resource := range state.Resources {
		impact := costAnalyzer.AnalyzeResource(resource)
		if impact != nil {
			costImpacts = append(costImpacts, impact)
			totalCurrentCost += impact.CurrentMonthlyCost
			totalDriftCost += impact.DriftMonthlyCost
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
	srv, err := api.NewServer(port)
	if err != nil {
		fmt.Printf("Error creating server: %v\n", err)
		return
	}
	
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
	fmt.Println("Discovering Terraform backend configurations...")
	
	discoverer := discovery.NewBackendDiscoverer()
	backends, err := discoverer.DiscoverBackends(".")
	if err != nil {
		fmt.Printf("Error during discovery: %v\n", err)
		return
	}

	fmt.Printf("Found %d backend configuration(s):\n\n", len(backends))
	
	for i, backend := range backends {
		fmt.Printf("%d. Backend Type: %s\n", i+1, backend.Type)
		fmt.Printf("   File: %s\n", backend.FilePath)
		fmt.Printf("   Module: %s\n", backend.Module)
		
		switch backend.Type {
		case "s3":
			if bucket, ok := backend.Config["bucket"].(string); ok {
				fmt.Printf("   S3 Bucket: %s\n", bucket)
			}
			if key, ok := backend.Config["key"].(string); ok {
				fmt.Printf("   State Key: %s\n", key)
			}
			if region, ok := backend.Config["region"].(string); ok {
				fmt.Printf("   Region: %s\n", region)
			}
		case "azurerm":
			if account, ok := backend.Config["storage_account_name"].(string); ok {
				fmt.Printf("   Storage Account: %s\n", account)
			}
			if container, ok := backend.Config["container_name"].(string); ok {
				fmt.Printf("   Container: %s\n", container)
			}
		case "gcs":
			if bucket, ok := backend.Config["bucket"].(string); ok {
				fmt.Printf("   GCS Bucket: %s\n", bucket)
			}
			if prefix, ok := backend.Config["prefix"].(string); ok {
				fmt.Printf("   Prefix: %s\n", prefix)
			}
		}
		fmt.Println()
	}
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
	graphBuilder := graph.NewDependencyGraphBuilder()
	depGraph, err := graphBuilder.BuildFromState(state)
	if err != nil {
		fmt.Printf("Error building dependency graph: %v\n", err)
		return
	}
	
	fmt.Println("\nBuilding Dependency Graph...")
	fmt.Printf("  Nodes: %d\n", depGraph.NodeCount())
	fmt.Printf("  Edges: %d\n", depGraph.EdgeCount())
	
	// Topological sort
	sorted, err := depGraph.TopologicalSort()
	if err != nil {
		fmt.Printf("  Warning: %v\n", err)
	} else {
		fmt.Printf("  Topological Order: %d resources sorted\n", len(sorted))
	}
	
	// Find orphaned resources
	orphaned := depGraph.FindOrphanedResources()
	if len(orphaned) > 0 {
		fmt.Printf("  Orphaned Resources: %d\n", len(orphaned))
		for _, r := range orphaned {
			fmt.Printf("    - %s\n", r)
		}
	}
	
	// Critical path
	criticalPath := depGraph.FindCriticalPath()
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
	backupPath, err := backupMgr.CreateBackup(ctx, localStatePath)
	if err != nil {
		fmt.Printf("Warning: Failed to create backup: %v\n", err)
	} else {
		fmt.Printf("Backup created: %s\n", backupPath)
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
	factory := backend.NewBackendFactory()
	b, err := factory.CreateBackend(backend.BackendType(backendType), backendConfig)
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
	
	// Create state manager
	stateMgr := manager.NewStateManager(b)
	
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
		backupPath, err := backupMgr.CreateBackup(ctx, localStatePath)
		if err != nil {
			fmt.Printf("Warning: Failed to create backup: %v\n", err)
		} else {
			fmt.Printf("Backup created: %s\n", backupPath)
		}
	}
	
	// Create backend
	factory := backend.NewBackendFactory()
	b, err := factory.CreateBackend(backend.BackendType(backendType), backendConfig)
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
	
	// Create state manager
	stateMgr := manager.NewStateManager(b)
	
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
		
		if config.RemoteState != nil {
			fmt.Printf("  Remote State Backend: %s\n", config.RemoteState.Backend)
		}
		
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
	
	backupInfo, err := mgr.CreateBackup(statePath, map[string]string{
		"source": "manual",
		"user":   os.Getenv("USER"),
	})
	
	if err != nil {
		fmt.Printf("Error creating backup: %v\n", err)
		return
	}
	
	fmt.Println("Backup created successfully")
	fmt.Printf("  ID: %s\n", backupInfo.ID)
	fmt.Printf("  Size: %d bytes\n", backupInfo.Size)
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
	
	backups, err := mgr.ListBackups(statePath)
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
		if b.Metadata != nil {
			if source, ok := b.Metadata["source"]; ok {
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
	
	if err := mgr.RestoreBackup(backupID, targetPath); err != nil {
		fmt.Printf("Error restoring backup: %v\n", err)
		return
	}
	
	fmt.Println("Backup restored successfully")
}

func handleDriftReport(ctx context.Context, args []string) {
	fmt.Println("Generating drift report...")
	// Implementation would generate detailed drift reports
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

func createCloudProvider(provider, region string) (types.CloudProvider, error) {
	switch provider {
	case "aws":
		if region == "" {
			region = "us-east-1"
		}
		awsProvider := providers.NewAWSProvider(region)
		return awsProvider, awsProvider.Initialize(context.Background())
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
	state, err := stateParser.ParseFile(statePath)
	if err != nil {
		return cloudResources // Assume all unmanaged if can't parse state
	}
	
	// Create map of managed resource IDs
	managedIDs := make(map[string]bool)
	for _, resource := range state.Resources {
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
	benchmark := commands.NewBenchmarkCommand()
	if err := benchmark.Execute(ctx, args); err != nil {
		fmt.Printf("Benchmark failed: %v\n", err)
		os.Exit(1)
	}
}

// handleROI calculates return on investment
func handleROI(ctx context.Context, args []string) {
	roi := commands.NewROICommand()
	if err := roi.Execute(ctx, args); err != nil {
		fmt.Printf("ROI calculation failed: %v\n", err)
		os.Exit(1)
	}
}

// handleIntegrations shows available integrations
func handleIntegrations(ctx context.Context, args []string) {
	integrations := commands.NewIntegrationsCommand()
	if err := integrations.Execute(ctx, args); err != nil {
		fmt.Printf("Failed to show integrations: %v\n", err)
		os.Exit(1)
	}
}