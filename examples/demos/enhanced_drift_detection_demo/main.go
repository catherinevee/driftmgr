package main

import (
	"fmt"

	"github.com/catherinevee/driftmgr/internal/drift"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/remediation"
)

func RunEnhancedDriftDemo() {
	fmt.Println("Enhanced Drift Detection and Remediation Demo")
	fmt.Println("=============================================")

	// Step 1: Enhanced Drift Detection
	fmt.Println("\n1. Enhanced Drift Detection")
	fmt.Println("----------------------------")

	// Create enhanced drift detector
	detector := drift.NewAttributeDriftDetector()

	// Configure sensitive fields
	detector.AddSensitiveField("tags.environment")
	detector.AddSensitiveField("tags.owner")
	detector.AddSensitiveField("tags.cost-center")
	detector.AddSensitiveField("security_groups")
	detector.AddSensitiveField("iam_policies")

	// Configure ignore fields
	detector.AddIgnoreField("tags.last-updated")
	detector.AddIgnoreField("tags.auto-generated")
	detector.AddIgnoreField("metadata.timestamp")

	// Add custom comparators
	detector.AddCustomComparator("security_groups", drift.SecurityGroupComparator)
	detector.AddCustomComparator("iam_policies", drift.IAMPolicyComparator)
	detector.AddCustomComparator("tags", drift.TagComparator)

	// Add severity rules
	detector.AddSeverityRule(drift.SeverityRule{
		ResourceType:  "aws_instance",
		AttributePath: "tags.environment",
		Condition:     "production",
		Severity:      "critical",
		Description:   "Production environment tags are critical",
	})

	detector.AddSeverityRule(drift.SeverityRule{
		ResourceType:  "aws_security_group",
		AttributePath: "ingress_rules",
		Condition:     "any",
		Severity:      "high",
		Description:   "Security group rule changes are high priority",
	})

	// Simulate state resources
	stateResources := []models.Resource{
		{
			ID:       "i-1234567890abcdef0",
			Name:     "web-server-1",
			Type:     "aws_instance",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Tags: map[string]string{
				"environment":  "production",
				"owner":        "devops-team",
				"cost-center":  "web-apps",
				"last-updated": "2024-01-15",
			},
		},
		{
			ID:       "sg-1234567890abcdef0",
			Name:     "web-server-sg",
			Type:     "aws_security_group",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Tags: map[string]string{
				"environment": "production",
				"owner":       "devops-team",
			},
		},
	}

	// Simulate live resources with drift
	liveResources := []models.Resource{
		{
			ID:       "i-1234567890abcdef0",
			Name:     "web-server-1",
			Type:     "aws_instance",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Tags: map[string]string{
				"environment":  "staging", // Drift: changed from production
				"owner":        "devops-team",
				"cost-center":  "web-apps",
				"last-updated": "2024-01-16", // Ignored field
			},
		},
		{
			ID:       "sg-1234567890abcdef0",
			Name:     "web-server-sg",
			Type:     "aws_security_group",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Tags: map[string]string{
				"environment": "production",
				"owner":       "devops-team",
			},
		},
		{
			ID:       "i-0987654321fedcba0",
			Name:     "unmanaged-instance",
			Type:     "aws_instance",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Tags: map[string]string{
				"environment": "development",
				"owner":       "developer",
			},
		},
	}

	// Perform enhanced drift detection
	fmt.Println("Performing enhanced drift detection...")
	result := detector.DetectDrift(stateResources, liveResources)

	// Display results
	fmt.Printf("\nDrift Analysis Results:\n")
	fmt.Printf("  Total Drifts: %d\n", result.Summary.TotalDrifts)
	fmt.Printf("  Critical: %d\n", result.Summary.CriticalDrifts)
	fmt.Printf("  High: %d\n", result.Summary.HighDrifts)
	fmt.Printf("  Medium: %d\n", result.Summary.MediumDrifts)
	fmt.Printf("  Low: %d\n", result.Summary.LowDrifts)
	fmt.Printf("  Coverage: %.2f%%\n", result.Summary.CoveragePercentage)
	fmt.Printf("  Drift Percentage: %.2f%%\n", result.Summary.DriftPercentage)

	// Display detailed results
	if len(result.DriftResults) > 0 {
		fmt.Printf("\nDetailed Drift Results:\n")
		for i, drift := range result.DriftResults {
			fmt.Printf("\n%d. %s (%s)\n", i+1, drift.ResourceName, drift.ResourceType)
			fmt.Printf("   ID: %s\n", drift.ResourceID)
			fmt.Printf("   Type: %s\n", drift.DriftType)
			fmt.Printf("   Severity: %s\n", drift.Severity)
			fmt.Printf("   Description: %s\n", drift.Description)
			if drift.RiskReasoning != "" {
				fmt.Printf("   Risk Reasoning: %s\n", drift.RiskReasoning)
			}

			if len(drift.Changes) > 0 {
				fmt.Printf("   Changes:\n")
				for _, change := range drift.Changes {
					fmt.Printf("     - %s: %v -> %v (%s)\n",
						change.Field, change.OldValue, change.NewValue, change.ChangeType)
				}
			}
		}
	}

	// Step 2: Remediation Engine
	fmt.Println("\n\n2. Remediation Engine")
	fmt.Println("---------------------")

	// Create remediation engine
	engine := remediation.NewRemediationEngine()

	// Generate remediation commands
	fmt.Println("Generating remediation commands...")
	commands := engine.GenerateRemediationCommands(result.DriftResults)

	if len(commands) > 0 {
		fmt.Printf("\nGenerated %d remediation commands:\n", len(commands))
		for i, cmd := range commands {
			fmt.Printf("\n%d. %s\n", i+1, cmd.Description)
			fmt.Printf("   Type: %s\n", cmd.Type)
			fmt.Printf("   Resource: %s\n", cmd.ResourceID)
			fmt.Printf("   Risk Level: %s\n", cmd.RiskLevel)
			fmt.Printf("   Auto-Approve: %t\n", cmd.AutoApprove)
			fmt.Printf("   Command: %s\n", cmd.Command)
		}

		// Demonstrate dry-run execution
		fmt.Println("\nDemonstrating dry-run execution...")
		for i, cmd := range commands {
			fmt.Printf("\n%d. %s\n", i+1, cmd.Description)
			fmt.Printf("   Risk Level: %s\n", cmd.RiskLevel)
			fmt.Printf("   Command: %s\n", cmd.Command)

			// Simulate execution
			if err := engine.ExecuteRemediation(cmd, false); err != nil {
				fmt.Printf("   Error: %v\n", err)
			} else {
				fmt.Printf("   Status: Completed successfully\n")
			}
		}
	} else {
		fmt.Println("No remediation commands needed")
	}

	// Step 3: Rollback Demonstration
	fmt.Println("\n\n3. Rollback Demonstration")
	fmt.Println("--------------------------")

	// List available snapshots
	snapshots := engine.ListSnapshots()
	if len(snapshots) > 0 {
		fmt.Printf("\nAvailable snapshots:\n")
		for _, snapshot := range snapshots {
			fmt.Printf("  %s (%s) - %s\n",
				snapshot.ID,
				snapshot.Timestamp.Format("2006-01-02 15:04:05"),
				snapshot.Description)
		}

		// Demonstrate rollback
		if len(snapshots) > 0 {
			snapshotID := snapshots[0].ID
			fmt.Printf("\nDemonstrating rollback to snapshot: %s\n", snapshotID)

			if err := engine.RollbackToSnapshot(snapshotID); err != nil {
				fmt.Printf("Rollback failed: %v\n", err)
			} else {
				fmt.Printf("Rollback completed successfully\n")
			}
		}
	} else {
		fmt.Println("No snapshots available")
	}

	fmt.Println("\nDemo completed successfully!")
}

// Example usage functions
func demonstrateEnhancedAnalyze() {
	fmt.Println("\nEnhanced Analyze Command Examples:")
	fmt.Println("==================================")
	fmt.Println("driftmgr> enhanced-analyze terraform")
	fmt.Println("driftmgr> enhanced-analyze terraform --sensitive-fields \"tags.environment,tags.owner\"")
	fmt.Println("driftmgr> enhanced-analyze terraform --config driftmgr.yaml --output summary")
}

func demonstrateRemediation() {
	fmt.Println("\nRemediation Command Examples:")
	fmt.Println("=============================")
	fmt.Println("driftmgr> remediate example --generate")
	fmt.Println("driftmgr> remediate example --dry-run")
	fmt.Println("driftmgr> remediate example --auto")
	fmt.Println("driftmgr> remediate-batch terraform --severity high")
	fmt.Println("driftmgr> remediate-history")
	fmt.Println("driftmgr> remediate-rollback snapshot_1234567890")
}

func main() {
	RunEnhancedDriftDemo()
}
