package main

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/remediation"
)

// handleRemediate processes remediation commands
func (shell *InteractiveShell) handleRemediate(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remediate <drift_id> [options]")
		fmt.Println("Options:")
		fmt.Println("  --auto                    Auto-approve all actions")
		fmt.Println("  --approve                 Approve specific action")
		fmt.Println("  --rollback <snapshot_id>  Rollback to specific snapshot")
		fmt.Println("  --dry-run                 Show commands without executing")
		fmt.Println("  --generate                Generate remediation commands only")
		return
	}

	driftID := args[0]

	// Parse options
	var autoApprove, dryRun, generateOnly bool
	var rollbackSnapshot string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--auto":
			autoApprove = true
		case "--dry-run":
			dryRun = true
		case "--generate":
			generateOnly = true
		case "--rollback":
			if i+1 < len(args) {
				rollbackSnapshot = args[i+1]
				i++
			}
		}
	}

	// Create remediation engine
	engine := remediation.NewRemediationEngine()

	// Handle rollback if requested
	if rollbackSnapshot != "" {
		if err := engine.RollbackToSnapshot(rollbackSnapshot); err != nil {
			fmt.Printf("Rollback failed: %v\n", err)
			return
		}
		fmt.Printf("Successfully rolled back to snapshot: %s\n", rollbackSnapshot)
		return
	}

	// Get drift results (in a real implementation, this would fetch from storage)
	driftResults := shell.getDriftResults(driftID)
	if len(driftResults) == 0 {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	// Generate remediation commands
	fmt.Println("Generating remediation commands...")
	commands := engine.GenerateRemediationCommands(driftResults)

	if len(commands) == 0 {
		fmt.Println("No remediation commands needed")
		return
	}

	// Display commands
	shell.displayRemediationCommands(commands)

	if generateOnly {
		return
	}

	// Execute commands
	if dryRun {
		fmt.Println("\nDRY RUN - Commands would be executed:")
		for _, cmd := range commands {
			fmt.Printf("  %s\n", cmd.Command)
		}
		return
	}

	// Execute remediation
	fmt.Println("\nExecuting remediation...")
	for i, cmd := range commands {
		fmt.Printf("\n%d. %s\n", i+1, cmd.Description)
		fmt.Printf("   Risk Level: %s\n", cmd.RiskLevel)
		fmt.Printf("   Command: %s\n", cmd.Command)

		if err := engine.ExecuteRemediation(cmd, autoApprove); err != nil {
			fmt.Printf("Remediation failed: %v\n", err)
			return
		}

		fmt.Printf("Remediation completed successfully\n")
	}

	fmt.Println("\nAll remediation actions completed!")
}

// getDriftResults retrieves drift results for a specific ID
func (shell *InteractiveShell) getDriftResults(driftID string) []models.DriftResult {
	// In a real implementation, this would fetch from a database or storage
	// For demo purposes, we'll create sample drift results

	if driftID == "example" {
		return []models.DriftResult{
			{
				ResourceID:   "i-1234567890abcdef0",
				ResourceName: "web-server-1",
				ResourceType: "aws_instance",
				Provider:     "aws",
				Region:       "us-east-1",
				DriftType:    "modified",
				Severity:     "critical",
				Description:  "Production environment tag changed from 'production' to 'staging'",
				Changes: []models.DriftChange{
					{
						Field:      "tags.environment",
						OldValue:   "production",
						NewValue:   "staging",
						ChangeType: "modified",
					},
				},
			},
			{
				ResourceID:   "sg-1234567890abcdef0",
				ResourceName: "web-server-sg",
				ResourceType: "aws_security_group",
				Provider:     "aws",
				Region:       "us-east-1",
				DriftType:    "modified",
				Severity:     "high",
				Description:  "Security group ingress rule added",
				Changes: []models.DriftChange{
					{
						Field:      "ingress_rules",
						OldValue:   "[]",
						NewValue:   "[0.0.0.0/0:22]",
						ChangeType: "modified",
					},
				},
			},
		}
	}

	return []models.DriftResult{}
}

// displayRemediationCommands displays the generated remediation commands
func (shell *InteractiveShell) displayRemediationCommands(commands []remediation.RemediationCommand) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("REMEDIATION COMMANDS")
	fmt.Println(strings.Repeat("=", 80))

	for i, cmd := range commands {
		fmt.Printf("\n%d. %s\n", i+1, cmd.Description)
		fmt.Printf("   Action: %s\n", cmd.Action)
		fmt.Printf("   Resource: %s\n", cmd.ResourceID)
		fmt.Printf("   Risk Level: %s\n", cmd.RiskLevel)
		fmt.Printf("   Auto-Approve: %t\n", cmd.AutoApprove)
		fmt.Printf("   Command: %s\n", cmd.Command)
	}
}

// handleRemediateBatch processes batch remediation for multiple drifts
func (shell *InteractiveShell) handleRemediateBatch(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remediate-batch <statefile_id> [options]")
		fmt.Println("Options:")
		fmt.Println("  --auto                    Auto-approve all actions")
		fmt.Println("  --severity <level>        Only remediate drifts of specified severity or higher")
		fmt.Println("  --dry-run                 Show commands without executing")
		return
	}

	stateFileID := args[0]

	// Parse options
	var autoApprove, dryRun bool
	var severityFilter string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--auto":
			autoApprove = true
		case "--dry-run":
			dryRun = true
		case "--severity":
			if i+1 < len(args) {
				severityFilter = args[i+1]
				i++
			}
		}
	}

	// Use the parsed options to avoid linter warnings
	_ = autoApprove
	_ = dryRun
	_ = severityFilter

	fmt.Printf("Starting batch remediation for state file: %s\n", stateFileID)

	// First, perform enhanced drift analysis
	shell.handleEnhancedAnalyze([]string{stateFileID, "--output", "json"})

	// In a real implementation, this would:
	// 1. Get the drift results from the analysis
	// 2. Filter by severity if specified
	// 3. Generate and execute remediation commands
	// 4. Provide a summary of all actions taken

	fmt.Println("Batch remediation would be implemented here")
}

// handleRemediateHistory shows remediation history
func (shell *InteractiveShell) handleRemediateHistory(args []string) {
	engine := remediation.NewRemediationEngine()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("REMEDIATION HISTORY")
	fmt.Println(strings.Repeat("=", 80))

	// List available snapshots
	snapshots := engine.ListSnapshots()

	if len(snapshots) == 0 {
		fmt.Println("No remediation history found")
		return
	}

	for _, snapshot := range snapshots {
		fmt.Printf("\nSnapshot: %s\n", snapshot.ID)
		fmt.Printf("  Timestamp: %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Description: %s\n", snapshot.Description)
	}
}

// handleRemediateRollback handles rollback operations
func (shell *InteractiveShell) handleRemediateRollback(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remediate-rollback <snapshot_id>")
		return
	}

	snapshotID := args[0]
	engine := remediation.NewRemediationEngine()

	fmt.Printf("Rolling back to snapshot: %s\n", snapshotID)

	if err := engine.RollbackToSnapshot(snapshotID); err != nil {
		fmt.Printf("Rollback failed: %v\n", err)
		return
	}

	fmt.Printf("Successfully rolled back to snapshot: %s\n", snapshotID)
}
