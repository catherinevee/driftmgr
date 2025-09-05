package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/catherinevee/driftmgr/internal/simulation"
)

// SimulateDriftCommand handles the simulate-drift command
func SimulateDriftCommand(args []string) error {
	fs := flag.NewFlagSet("simulate-drift", flag.ExitOnError)
	
	// Command flags
	stateFile := fs.String("state", "", "Path to Terraform state file (required)")
	provider := fs.String("provider", "", "Cloud provider (aws, azure, gcp)")
	driftType := fs.String("type", "random", "Type of drift to simulate (tag-change, rule-addition, resource-creation, attribute-change, random)")
	targetResource := fs.String("target", "", "Specific resource to target (optional)")
	autoRollback := fs.Bool("auto-rollback", true, "Automatically rollback drift after detection")
	dryRun := fs.Bool("dry-run", false, "Preview drift without making changes")
	rollback := fs.Bool("rollback", false, "Rollback previous drift simulation")
	detect := fs.Bool("detect", true, "Run drift detection after simulation")
	verbose := fs.Bool("verbose", false, "Verbose output")
	
	// Help text
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: driftmgr simulate-drift --state <file> [options]

Simulate controlled drift in cloud resources for testing drift detection.

This command creates low-cost or free modifications to cloud resources to test
DriftMgr's drift detection capabilities. All changes can be automatically rolled back.

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Drift Types:
  tag-change        Add or modify tags/labels on resources
  rule-addition     Add security group or firewall rules
  resource-creation Create new resources not in state
  attribute-change  Modify resource attributes
  random           Randomly select a drift type

Examples:
  # Simulate tag drift on AWS resources
  driftmgr simulate-drift --state terraform.tfstate --provider aws --type tag-change

  # Create unmanaged resources in Azure
  driftmgr simulate-drift --state terraform.tfstate --provider azure --type resource-creation

  # Simulate random drift and auto-detect
  driftmgr simulate-drift --state terraform.tfstate --provider aws

  # Rollback previous drift simulation
  driftmgr simulate-drift --rollback

  # Dry run to see what would be changed
  driftmgr simulate-drift --state terraform.tfstate --provider aws --dry-run

Safety Features:
  - All modifications use free tier resources or cost $0
  - Automatic rollback after detection
  - Dry-run mode for preview
  - Resources auto-expire after 24 hours
  - Test IP ranges (192.0.2.0/32) for network rules
`)
	}
	
	if err := fs.Parse(args); err != nil {
		return err
	}
	
	// Handle rollback mode
	if *rollback {
		return handleRollback()
	}
	
	// Validate required flags
	if *stateFile == "" {
		return fmt.Errorf("--state flag is required")
	}
	
	// Auto-detect provider from state if not specified
	if *provider == "" {
		detectedProvider := detectProviderFromState(*stateFile)
		if detectedProvider == "" {
			return fmt.Errorf("--provider flag is required or could not be auto-detected from state")
		}
		*provider = detectedProvider
		fmt.Printf("Auto-detected provider: %s\n", *provider)
	}
	
	// Convert drift type string to enum
	var driftTypeEnum simulation.DriftType
	switch strings.ToLower(*driftType) {
	case "tag-change", "tag":
		driftTypeEnum = simulation.DriftTypeTagChange
	case "rule-addition", "rule":
		driftTypeEnum = simulation.DriftTypeRuleAddition
	case "resource-creation", "resource":
		driftTypeEnum = simulation.DriftTypeResourceCreation
	case "attribute-change", "attribute":
		driftTypeEnum = simulation.DriftTypeAttributeChange
	case "random", "":
		driftTypeEnum = simulation.DriftTypeRandom
	default:
		return fmt.Errorf("invalid drift type: %s", *driftType)
	}
	
	// Print simulation plan
	fmt.Println("\n=== Drift Simulation Plan ===")
	fmt.Printf("State File: %s\n", *stateFile)
	fmt.Printf("Provider: %s\n", *provider)
	fmt.Printf("Drift Type: %s\n", *driftType)
	if *targetResource != "" {
		fmt.Printf("Target Resource: %s\n", *targetResource)
	}
	fmt.Printf("Auto Rollback: %v\n", *autoRollback)
	fmt.Printf("Detect After: %v\n", *detect)
	
	if *dryRun {
		fmt.Println("\n[DRY RUN MODE - No actual changes will be made]")
	}
	
	// Confirm with user
	if !*dryRun {
		fmt.Print("\nProceed with drift simulation? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Simulation cancelled")
			return nil
		}
	}
	
	// Create simulator
	config := simulation.SimulatorConfig{
		StateFile:      *stateFile,
		Provider:       *provider,
		DriftType:      driftTypeEnum,
		TargetResource: *targetResource,
		AutoRollback:   *autoRollback,
		DryRun:         *dryRun,
	}
	
	simulator, err := simulation.NewDriftSimulator(config)
	if err != nil {
		return fmt.Errorf("failed to create drift simulator: %w", err)
	}
	
	ctx := context.Background()
	
	// Simulate drift
	fmt.Println("\n🔄 Simulating drift...")
	result, err := simulator.SimulateDrift(ctx)
	if err != nil {
		return fmt.Errorf("drift simulation failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("drift simulation failed: %s", result.ErrorMessage)
	}
	
	// Display results
	fmt.Println("\n✅ Drift Simulation Successful!")
	fmt.Printf("Provider: %s\n", result.Provider)
	fmt.Printf("Resource Type: %s\n", result.ResourceType)
	fmt.Printf("Resource ID: %s\n", result.ResourceID)
	fmt.Printf("Drift Type: %s\n", result.DriftType)
	fmt.Printf("Cost Estimate: %s\n", result.CostEstimate)
	
	fmt.Println("\nChanges Applied:")
	for key, value := range result.Changes {
		fmt.Printf("  • %s: %v\n", key, value)
	}
	
	// Save rollback data
	if result.RollbackData != nil {
		saveRollbackData(result.RollbackData)
		fmt.Println("\n💾 Rollback data saved (use --rollback to undo)")
	}
	
	// Run drift detection
	if *detect && !*dryRun {
		fmt.Println("\n🔍 Running drift detection...")
		drifts, err := simulator.DetectDrift(ctx)
		if err != nil {
			fmt.Printf("Warning: Drift detection failed: %v\n", err)
		} else if len(drifts) > 0 {
			fmt.Printf("\n⚠️  Drift Detected! Found %d drift(s):\n\n", len(drifts))
			for i, drift := range drifts {
				fmt.Printf("%d. %s (%s)\n", i+1, drift.ResourceID, drift.ResourceType)
				fmt.Printf("   Type: %s\n", drift.DriftType)
				fmt.Printf("   Impact: %s\n", drift.Impact)
				
				if *verbose {
					if len(drift.Before) > 0 {
						fmt.Println("   Before:")
						for k, v := range drift.Before {
							fmt.Printf("     %s: %v\n", k, v)
						}
					}
					if len(drift.After) > 0 {
						fmt.Println("   After:")
						for k, v := range drift.After {
							fmt.Printf("     %s: %v\n", k, v)
						}
					}
				}
			}
			
			// Store detected drifts in result
			result.DetectedDrift = drifts
		} else {
			fmt.Println("\n✅ No drift detected (this might be an error)")
		}
	}
	
	// Auto-rollback if enabled
	if *autoRollback && !*dryRun && result.RollbackData != nil {
		fmt.Println("\n🔄 Auto-rollback in 5 seconds (press Ctrl+C to keep drift)...")
		time.Sleep(5 * time.Second)
		
		fmt.Println("Rolling back drift...")
		if err := simulator.Rollback(ctx); err != nil {
			fmt.Printf("⚠️  Rollback failed: %v\n", err)
			fmt.Println("You can manually rollback later with: driftmgr simulate-drift --rollback")
		} else {
			fmt.Println("✅ Drift rolled back successfully!")
			clearRollbackData()
		}
	}
	
	// Generate report
	if *verbose {
		report := simulator.GenerateReport(result, result.DetectedDrift)
		fmt.Println("\n" + report)
	}
	
	return nil
}

// detectProviderFromState attempts to detect the cloud provider from the state file
func detectProviderFromState(stateFile string) string {
	// Read state file
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return ""
	}
	
	content := string(data)
	
	// Check for provider-specific resource types
	if strings.Contains(content, "aws_") {
		return "aws"
	}
	if strings.Contains(content, "azurerm_") {
		return "azure"
	}
	if strings.Contains(content, "google_") {
		return "gcp"
	}
	
	return ""
}

// handleRollback handles rolling back a previous drift simulation
func handleRollback() error {
	// Load rollback data
	data := loadRollbackData()
	if data == nil {
		return fmt.Errorf("no rollback data found. Did you run a simulation first?")
	}
	
	fmt.Println("\n=== Rollback Information ===")
	fmt.Printf("Provider: %s\n", data.Provider)
	fmt.Printf("Resource Type: %s\n", data.ResourceType)
	fmt.Printf("Resource ID: %s\n", data.ResourceID)
	fmt.Printf("Action: %s\n", data.Action)
	fmt.Printf("Timestamp: %s\n", data.Timestamp.Format("2006-01-02 15:04:05"))
	
	fmt.Print("\nProceed with rollback? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Rollback cancelled")
		return nil
	}
	
	// Create appropriate simulator based on provider
	var simulator *simulation.DriftSimulator
	config := simulation.SimulatorConfig{
		Provider: data.Provider,
	}
	
	simulator, err := simulation.NewDriftSimulator(config)
	if err != nil {
		return fmt.Errorf("failed to create simulator for rollback: %w", err)
	}
	
	ctx := context.Background()
	
	fmt.Println("\n🔄 Executing rollback...")
	if err := simulator.Rollback(ctx); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	fmt.Println("✅ Rollback completed successfully!")
	clearRollbackData()
	
	return nil
}

// saveRollbackData saves rollback data to a file
func saveRollbackData(data *simulation.RollbackData) {
	// Save to .driftmgr/rollback.json
	dir := ".driftmgr"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	
	file := fmt.Sprintf("%s/rollback.json", dir)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	
	os.WriteFile(file, jsonData, 0644)
}

// loadRollbackData loads rollback data from file
func loadRollbackData() *simulation.RollbackData {
	file := ".driftmgr/rollback.json"
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	
	var rollback simulation.RollbackData
	if err := json.Unmarshal(data, &rollback); err != nil {
		return nil
	}
	
	return &rollback
}

// clearRollbackData removes rollback data file
func clearRollbackData() {
	os.Remove(".driftmgr/rollback.json")
}