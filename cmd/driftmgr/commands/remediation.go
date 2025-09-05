package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// HandleRemediationCommand handles the remediation command group
func HandleRemediationCommand(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		showRemediationHelp()
		return
	}

	switch args[0] {
	case "generate":
		handleRemediationGenerate(args[1:])
	case "validate":
		handleRemediationValidate(args[1:])
	case "apply":
		handleRemediationApply(args[1:])
	case "rollback":
		handleRemediationRollback(args[1:])
	case "list":
		handleRemediationList(args[1:])
	case "show":
		handleRemediationShow(args[1:])
	case "discover-state":
		handleDiscoverState(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown remediation command: %s\n", args[0])
		showRemediationHelp()
		os.Exit(1)
	}
}

// showRemediationHelp shows help for remediation commands
func showRemediationHelp() {
	fmt.Println("Usage: driftmgr remediation [command] [options]")
	fmt.Println()
	fmt.Println("Terraform remediation automatically generates and applies fixes for drift")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  %s              Generate remediation plan for drift\n", color.CyanString("generate"))
	fmt.Printf("  %s              Validate a remediation plan\n", color.CyanString("validate"))
	fmt.Printf("  %s                 Apply a remediation plan\n", color.CyanString("apply"))
	fmt.Printf("  %s              Rollback applied changes\n", color.CyanString("rollback"))
	fmt.Printf("  %s                  List all remediation plans\n", color.CyanString("list"))
	fmt.Printf("  %s                  Show details of a plan\n", color.CyanString("show"))
	fmt.Printf("  %s        Auto-discover Terraform state files\n", color.CyanString("discover-state"))
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --dry-run            Simulate changes without applying")
	fmt.Println("  --auto-approve       Skip approval prompts")
	fmt.Println("  --output <format>    Output format (hcl, json)")
	fmt.Println("  --work-dir <path>    Working directory for remediation")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  driftmgr remediation generate --drift-id drift-123")
	fmt.Println("  driftmgr remediation apply plan-456 --dry-run")
	fmt.Println("  driftmgr remediation discover-state --all")
	fmt.Println("  driftmgr remediation rollback plan-789")
}

// handleRemediationGenerate handles generating a remediation plan
func handleRemediationGenerate(args []string) {
	var driftID string
	var workDir string = ".driftmgr/remediation"
	var outputFormat string = "hcl"
	var autoApprove bool

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--drift-id":
			if i+1 < len(args) {
				driftID = args[i+1]
				i++
			}
		case "--work-dir":
			if i+1 < len(args) {
				workDir = args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(args) {
				outputFormat = args[i+1]
				i++
			}
		case "--auto-approve":
			autoApprove = true
		}
	}

	if driftID == "" {
		fmt.Println("Error: --drift-id is required")
		fmt.Println("Run 'driftmgr drift detect' first to identify drift")
		os.Exit(1)
	}

	fmt.Printf("🔧 Generating remediation plan for drift: %s\n", driftID)

	// Create remediation config
	config := remediation.DefaultRemediationConfig()
	config.OutputFormat = outputFormat
	config.AutoApprove = autoApprove

	// Create remediation engine
	engine, err := remediation.NewRemediationEngine(workDir, config)
	if err != nil {
		fmt.Printf("❌ Failed to create remediation engine: %v\n", err)
		os.Exit(1)
	}

	// Get drift result from store
	driftStore := api.GetGlobalDriftStore()
	driftRecord, exists := driftStore.GetDriftByID(driftID)
	if !exists {
		fmt.Printf("❌ Drift with ID '%s' not found\n", driftID)
		fmt.Println("Run 'driftmgr drift detect' to identify drifts")
		os.Exit(1)
	}
	
	// Convert DriftRecord to DriftResult
	drift := convertDriftRecordToResult(driftRecord)

	// Generate remediation plan
	ctx := context.Background()
	plan, err := engine.GeneratePlan(ctx, drift)
	if err != nil {
		fmt.Printf("❌ Failed to generate remediation plan: %v\n", err)
		os.Exit(1)
	}

	// Save the plan
	if err := engine.SavePlan(plan); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save plan: %v\n", err)
	}

	// Display plan summary
	displayPlanSummary(plan)

	fmt.Printf("\n✅ Remediation plan generated: %s\n", color.GreenString(plan.ID))
	fmt.Printf("   Type: %s\n", plan.Type)
	fmt.Printf("   Changes: %d\n", len(plan.Changes))
	
	if plan.EstimatedImpact != nil {
		fmt.Printf("   Risk Score: %.1f/10\n", plan.EstimatedImpact.RiskScore)
		fmt.Printf("   Estimated Downtime: %s\n", plan.EstimatedImpact.EstimatedDowntime)
		
		if plan.EstimatedImpact.RequiresApproval {
			fmt.Printf("   ⚠️  %s\n", color.YellowString("Requires Approval: "+plan.EstimatedImpact.ApprovalReason))
		}
	}

	fmt.Println()
	fmt.Printf("To validate: %s\n", color.CyanString("driftmgr remediation validate "+plan.ID))
	fmt.Printf("To apply:    %s\n", color.CyanString("driftmgr remediation apply "+plan.ID))
}

// handleRemediationValidate handles validating a remediation plan
func handleRemediationValidate(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Plan ID required")
		fmt.Println("Usage: driftmgr remediation validate <plan-id>")
		os.Exit(1)
	}

	planID := args[0]
	workDir := ".driftmgr/remediation"

	fmt.Printf("🔍 Validating remediation plan: %s\n", planID)

	// Create remediation engine
	config := remediation.DefaultRemediationConfig()
	engine, err := remediation.NewRemediationEngine(workDir, config)
	if err != nil {
		fmt.Printf("❌ Failed to create remediation engine: %v\n", err)
		os.Exit(1)
	}

	// Load the plan
	plan, err := engine.GetPlan(planID)
	if err != nil {
		fmt.Printf("❌ Failed to load plan: %v\n", err)
		os.Exit(1)
	}

	// Validate the plan
	ctx := context.Background()
	result, err := engine.ValidatePlan(ctx, plan)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	// Display validation results
	if result.Valid {
		fmt.Println("✅ Plan validation passed")
	} else {
		fmt.Println("❌ Plan validation failed")
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("  • %s\n", color.RedString(err))
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  • %s\n", color.YellowString(warn))
		}
	}

	if len(result.SecurityIssues) > 0 {
		fmt.Println("\n🔒 Security Issues:")
		for _, issue := range result.SecurityIssues {
			fmt.Printf("  • %s\n", color.RedString(issue))
		}
	}

	if len(result.BestPractices) > 0 {
		fmt.Println("\n💡 Best Practice Suggestions:")
		for _, bp := range result.BestPractices {
			fmt.Printf("  • %s\n", color.CyanString(bp))
		}
	}

	if !result.Valid {
		os.Exit(1)
	}
}

// handleRemediationApply handles applying a remediation plan
func handleRemediationApply(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Plan ID required")
		fmt.Println("Usage: driftmgr remediation apply <plan-id> [options]")
		os.Exit(1)
	}

	planID := args[0]
	workDir := ".driftmgr/remediation"
	var dryRun bool
	var autoApprove bool

	// Parse additional arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--auto-approve":
			autoApprove = true
		case "--work-dir":
			if i+1 < len(args) {
				workDir = args[i+1]
				i++
			}
		}
	}

	if dryRun {
		fmt.Println("🔄 Running in DRY RUN mode - no actual changes will be made")
	}

	fmt.Printf("📦 Applying remediation plan: %s\n", planID)

	// Create remediation config
	config := remediation.DefaultRemediationConfig()
	config.DryRun = dryRun
	config.AutoApprove = autoApprove

	// Create remediation engine
	engine, err := remediation.NewRemediationEngine(workDir, config)
	if err != nil {
		fmt.Printf("❌ Failed to create remediation engine: %v\n", err)
		os.Exit(1)
	}

	// Load the plan
	plan, err := engine.GetPlan(planID)
	if err != nil {
		fmt.Printf("❌ Failed to load plan: %v\n", err)
		os.Exit(1)
	}

	// Display plan details before applying
	displayPlanSummary(plan)

	// Confirm if not auto-approve and not dry-run
	if !autoApprove && !dryRun {
		fmt.Print("\nDo you want to apply this plan? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
			fmt.Println("Application cancelled")
			return
		}
	}

	// Apply the plan
	ctx := context.Background()
	result, err := engine.ApplyPlan(ctx, plan)
	if err != nil {
		fmt.Printf("❌ Failed to apply plan: %v\n", err)
		if result != nil && result.RollbackExecuted {
			fmt.Println("ℹ️  Rollback was executed successfully")
		}
		os.Exit(1)
	}

	// Display results
	if result.Success {
		fmt.Println("✅ Remediation applied successfully")
	} else {
		fmt.Println("⚠️  Remediation completed with issues")
	}

	fmt.Printf("   Applied changes: %d\n", len(result.AppliedChanges))
	fmt.Printf("   Failed changes: %d\n", len(result.FailedChanges))
	fmt.Printf("   Execution time: %s\n", result.ExecutionTime)

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("  • %s\n", color.RedString(err))
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  • %s\n", color.YellowString(warn))
		}
	}

	if !result.Success {
		os.Exit(1)
	}
}

// handleRemediationRollback handles rolling back a remediation
func handleRemediationRollback(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Plan ID required")
		fmt.Println("Usage: driftmgr remediation rollback <plan-id>")
		os.Exit(1)
	}

	planID := args[0]
	workDir := ".driftmgr/remediation"

	fmt.Printf("↩️  Rolling back remediation plan: %s\n", planID)

	// Create remediation engine
	config := remediation.DefaultRemediationConfig()
	engine, err := remediation.NewRemediationEngine(workDir, config)
	if err != nil {
		fmt.Printf("❌ Failed to create remediation engine: %v\n", err)
		os.Exit(1)
	}

	// Load the plan
	plan, err := engine.GetPlan(planID)
	if err != nil {
		fmt.Printf("❌ Failed to load plan: %v\n", err)
		os.Exit(1)
	}

	if plan.RollbackPlan == nil {
		fmt.Println("❌ No rollback plan available for this remediation")
		os.Exit(1)
	}

	// Confirm rollback
	fmt.Print("Are you sure you want to rollback? (yes/no): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
		fmt.Println("Rollback cancelled")
		return
	}

	// Execute rollback
	rollbackManager := remediation.NewRollbackManager(workDir)
	ctx := context.Background()
	result, err := rollbackManager.ExecuteRollback(ctx, plan.RollbackPlan)
	if err != nil {
		fmt.Printf("❌ Rollback failed: %v\n", err)
		os.Exit(1)
	}

	if result.Success {
		fmt.Println("✅ Rollback completed successfully")
	} else {
		fmt.Println("⚠️  Rollback completed with issues")
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("  • %s\n", color.RedString(err))
		}
	}
}

// handleRemediationList handles listing remediation plans
func handleRemediationList(args []string) {
	workDir := ".driftmgr/remediation"
	
	fmt.Println("📋 Remediation Plans")

	// Create remediation engine
	config := remediation.DefaultRemediationConfig()
	engine, err := remediation.NewRemediationEngine(workDir, config)
	if err != nil {
		fmt.Printf("❌ Failed to create remediation engine: %v\n", err)
		os.Exit(1)
	}

	// List plans
	plans, err := engine.ListPlans()
	if err != nil {
		fmt.Printf("❌ Failed to list plans: %v\n", err)
		os.Exit(1)
	}

	if len(plans) == 0 {
		fmt.Println("No remediation plans found")
		return
	}

	// Display as table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Plan ID", "Resource", "Type", "Changes", "Risk", "Created", "Applied"})
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator(" ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, plan := range plans {
		riskScore := "N/A"
		if plan.EstimatedImpact != nil {
			riskScore = fmt.Sprintf("%.1f", plan.EstimatedImpact.RiskScore)
		}

		applied := "-"
		if plan.AppliedAt != nil {
			applied = plan.AppliedAt.Format("2006-01-02 15:04")
		}

		table.Append([]string{
			plan.ID,
			fmt.Sprintf("%s.%s", plan.ResourceType, plan.ResourceName),
			string(plan.Type),
			fmt.Sprintf("%d", len(plan.Changes)),
			riskScore,
			plan.CreatedAt.Format("2006-01-02 15:04"),
			applied,
		})
	}

	table.Render()
}

// handleRemediationShow handles showing details of a remediation plan
func handleRemediationShow(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Plan ID required")
		fmt.Println("Usage: driftmgr remediation show <plan-id>")
		os.Exit(1)
	}

	planID := args[0]
	workDir := ".driftmgr/remediation"

	// Create remediation engine
	config := remediation.DefaultRemediationConfig()
	engine, err := remediation.NewRemediationEngine(workDir, config)
	if err != nil {
		fmt.Printf("❌ Failed to create remediation engine: %v\n", err)
		os.Exit(1)
	}

	// Load the plan
	plan, err := engine.GetPlan(planID)
	if err != nil {
		fmt.Printf("❌ Failed to load plan: %v\n", err)
		os.Exit(1)
	}

	// Display detailed plan information
	fmt.Printf("📄 Remediation Plan: %s\n", color.CyanString(plan.ID))
	fmt.Println(strings.Repeat("─", 60))
	
	fmt.Printf("Resource:     %s.%s\n", plan.ResourceType, plan.ResourceName)
	fmt.Printf("Provider:     %s\n", plan.Provider)
	fmt.Printf("Type:         %s\n", plan.Type)
	fmt.Printf("Created:      %s\n", plan.CreatedAt.Format("2006-01-02 15:04:05"))
	
	if plan.AppliedAt != nil {
		fmt.Printf("Applied:      %s\n", plan.AppliedAt.Format("2006-01-02 15:04:05"))
	}

	if plan.EstimatedImpact != nil {
		fmt.Println("\n📊 Impact Assessment:")
		fmt.Printf("  Severity:    %s\n", plan.EstimatedImpact.Severity)
		fmt.Printf("  Risk Score:  %.1f/10\n", plan.EstimatedImpact.RiskScore)
		fmt.Printf("  Downtime:    %s\n", plan.EstimatedImpact.EstimatedDowntime)
		
		if plan.EstimatedImpact.RequiresApproval {
			fmt.Printf("  Approval:    %s\n", color.YellowString("Required - "+plan.EstimatedImpact.ApprovalReason))
		}
	}

	fmt.Printf("\n📝 Changes (%d):\n", len(plan.Changes))
	for i, change := range plan.Changes {
		fmt.Printf("  %d. %s %s\n", i+1, change.Action, change.Path)
		if change.Action == "update" {
			fmt.Printf("     Old: %v\n", change.OldValue)
			fmt.Printf("     New: %v\n", change.NewValue)
		}
		fmt.Printf("     Sensitivity: %s\n", change.Sensitivity)
	}

	if len(plan.ImportCommands) > 0 {
		fmt.Println("\n🔧 Import Commands:")
		for _, cmd := range plan.ImportCommands {
			fmt.Printf("  %s\n", cmd)
		}
	}

	if len(plan.ValidationErrors) > 0 {
		fmt.Println("\n⚠️  Validation Errors:")
		for _, err := range plan.ValidationErrors {
			fmt.Printf("  • %s\n", color.RedString(err))
		}
	}

	fmt.Println("\n📄 Terraform Code:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(plan.TerraformCode)
	fmt.Println(strings.Repeat("─", 60))
}

// handleDiscoverState handles auto-discovery of Terraform state files
func handleDiscoverState(args []string) {
	var all bool
	var paths []string

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			all = true
		case "--path":
			if i+1 < len(args) {
				paths = append(paths, args[i+1])
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "--") {
				paths = append(paths, args[i])
			}
		}
	}

	// Default to current directory
	if len(paths) == 0 && !all {
		paths = []string{"."}
	}

	// If --all, scan common locations
	if all {
		homeDir, _ := os.UserHomeDir()
		paths = []string{
			".",
			filepath.Join(homeDir, "terraform"),
			filepath.Join(homeDir, "infrastructure"),
			filepath.Join(homeDir, "projects"),
			"/terraform",
			"/infrastructure",
		}
	}

	fmt.Println("🔍 Auto-discovering Terraform state files...")
	
	// Create state discovery
	discovery := remediation.NewStateDiscovery(paths)
	
	// Discover state files
	ctx := context.Background()
	stateFiles, err := discovery.DiscoverStateFiles(ctx)
	if err != nil {
		fmt.Printf("⚠️  Warning: Some errors occurred during discovery: %v\n", err)
	}

	if len(stateFiles) == 0 {
		fmt.Println("No Terraform state files found")
		return
	}

	fmt.Printf("\n📁 Discovered %d state files:\n\n", len(stateFiles))

	// Group by backend type
	byBackend := make(map[string][]*remediation.StateFileInfo)
	for _, sf := range stateFiles {
		byBackend[sf.Backend] = append(byBackend[sf.Backend], sf)
	}

	// Display grouped results
	for backend, files := range byBackend {
		fmt.Printf("🔹 %s Backend (%d files):\n", strings.Title(backend), len(files))
		
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Path", "Resources", "Version", "Environment", "Workspace", "Last Modified"})
		table.SetBorder(false)
		table.SetHeaderLine(false)
		table.SetColumnSeparator(" ")
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		for _, sf := range files {
			lastMod := sf.LastModified.Format("2006-01-02 15:04")
			if sf.IsRemote {
				lastMod = "Remote"
			}

			table.Append([]string{
				truncatePath(sf.Path, 40),
				fmt.Sprintf("%d", sf.ResourceCount),
				fmt.Sprintf("v%d", sf.Version),
				sf.Environment,
				sf.WorkspaceName,
				lastMod,
			})
		}
		
		table.Render()
		fmt.Println()
	}

	// Display summary
	fmt.Println("📊 Summary:")
	totalResources := 0
	for _, sf := range stateFiles {
		totalResources += sf.ResourceCount
	}
	fmt.Printf("  • Total state files: %d\n", len(stateFiles))
	fmt.Printf("  • Total resources: %d\n", totalResources)
	
	// Count by backend
	fmt.Println("  • By backend:")
	for backend, files := range byBackend {
		fmt.Printf("    - %s: %d\n", backend, len(files))
	}

	// Show providers used
	providers := make(map[string]bool)
	for _, sf := range stateFiles {
		for provider := range sf.Providers {
			providers[provider] = true
		}
	}
	
	if len(providers) > 0 {
		fmt.Println("  • Providers used:")
		for provider := range providers {
			fmt.Printf("    - %s\n", provider)
		}
	}
}

// Helper functions

// displayPlanSummary displays a summary of a remediation plan
func displayPlanSummary(plan *remediation.RemediationPlan) {
	fmt.Println("\n📋 Plan Summary:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Resource:     %s.%s\n", plan.ResourceType, plan.ResourceName)
	fmt.Printf("Provider:     %s\n", plan.Provider)
	fmt.Printf("Action:       %s\n", plan.Type)
	fmt.Printf("Changes:      %d\n", len(plan.Changes))

	if plan.EstimatedImpact != nil {
		fmt.Printf("Risk Score:   %.1f/10\n", plan.EstimatedImpact.RiskScore)
		fmt.Printf("Severity:     %s\n", plan.EstimatedImpact.Severity)
		
		if plan.EstimatedImpact.EstimatedDowntime > 0 {
			fmt.Printf("Est. Downtime: %s\n", plan.EstimatedImpact.EstimatedDowntime)
		}
		
		if plan.EstimatedImpact.CostImpact != nil {
			cost := plan.EstimatedImpact.CostImpact
			if cost.MonthlySavings > 0 {
				fmt.Printf("Monthly Savings: $%.2f\n", cost.MonthlySavings)
			} else if cost.MonthlySavings < 0 {
				fmt.Printf("Monthly Cost Increase: $%.2f\n", -cost.MonthlySavings)
			}
		}
	}

	fmt.Println(strings.Repeat("─", 60))
}

// convertDriftRecordToResult converts a DriftRecord to DriftResult
func convertDriftRecordToResult(record *api.DriftRecord) *models.DriftResult {
	result := &models.DriftResult{
		ResourceID:   record.ResourceID,
		ResourceName: record.ResourceID, // Use ResourceID as name if not available
		ResourceType: record.ResourceType,
		Provider:     record.Provider,
		Region:       record.Region,
		DriftType:    record.DriftType,
		Severity:     record.Severity,
		Description:  fmt.Sprintf("Drift detected in %s resource", record.ResourceType),
		DetectedAt:   record.DetectedAt,
		Changes:      []models.DriftChange{},
	}
	
	// Convert changes map to DriftChange slice
	for field, value := range record.Changes {
		change := models.DriftChange{
			Field:      field,
			ChangeType: "update",
		}
		
		// Try to extract old and new values from the change
		if changeMap, ok := value.(map[string]interface{}); ok {
			if oldVal, exists := changeMap["old"]; exists {
				change.OldValue = oldVal
			}
			if newVal, exists := changeMap["new"]; exists {
				change.NewValue = newVal
			}
			if changeType, exists := changeMap["type"]; exists {
				change.ChangeType = fmt.Sprintf("%v", changeType)
			}
		} else {
			// If it's not a map, treat the whole value as the new value
			change.NewValue = value
		}
		
		result.Changes = append(result.Changes, change)
	}
	
	return result
}

// truncatePath truncates a path to fit within a max length
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	
	// Try to keep the filename visible
	parts := strings.Split(path, "/")
	if len(parts) > 2 {
		return "..." + path[len(path)-maxLen+3:]
	}
	
	return path[:maxLen-3] + "..."
}