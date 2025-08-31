package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/fatih/color"
)

type QuickCheckResult struct {
	SafeToApply bool
	Confidence  float64
	Issues      []QuickCheckIssue
	Summary     QuickCheckSummary
	Timestamp   time.Time
}

type QuickCheckIssue struct {
	Severity    string
	Resource    string
	Type        string
	Description string
	Impact      string
}

type QuickCheckSummary struct {
	TotalResources      int
	DriftedResources    int
	CriticalDrifts      int
	DestructiveChanges  int
	SecurityImpact      bool
	EstimatedRisk       string
}

// handleQuickCheck performs a quick drift check for safe-to-apply assessment
func handleQuickCheck(args []string) {
	ctx := context.Background()
	
	var (
		statePath      = ""
		verbose        = false
		jsonOutput     = false
		threshold      = 80.0 // confidence threshold for safe-to-apply
		smart          = true  // Smart filtering enabled by default
		securityOnly   = false // Security-focused detection
		noFilter       = false // Disable all filtering
		ignoreCosmetic = true  // Ignore cosmetic changes by default
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--verbose", "-v":
			verbose = true
		case "--json":
			jsonOutput = true
		case "--threshold":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%f", &threshold)
				i++
			}
		case "--smart":
			smart = true
			noFilter = false
		case "--security-only":
			securityOnly = true
			smart = false
		case "--no-filter":
			noFilter = true
			smart = false
			ignoreCosmetic = false
		case "--include-cosmetic":
			ignoreCosmetic = false
		case "--help", "-h":
			showQuickCheckHelp()
			return
		}
	}
	
	// Find state file
	if statePath == "" {
		statePath = findTerraformStateFile()
	}
	
	if statePath == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No Terraform state file found"))
		os.Exit(1)
	}
	
	// Header
	if !jsonOutput {
		fmt.Println(color.CyanString("🔍 Quick Drift Check"))
		fmt.Println(strings.Repeat("=", 51))
		fmt.Printf("State File: %s\n", statePath)
		fmt.Printf("Threshold: %.0f%%\n\n", threshold)
	}
	
	// Load state
	loader := state.NewStateLoader(statePath)
	stateFile, err := loader.LoadStateFile(ctx, statePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error loading state: %v\n"), err)
		os.Exit(1)
	}
	
	// Initialize discovery
	discoveryService, err := discovery.InitializeServiceSilent(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error initializing discovery: %v\n"), err)
		os.Exit(1)
	}
	
	// Detect provider from state
	provider := detectProvider(stateFile)
	if provider == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: Could not detect cloud provider from state"))
		os.Exit(1)
	}
	
	// Discover actual resources
	if !jsonOutput {
		fmt.Printf("Checking %s resources...\n\n", provider)
	}
	
	discoveryResult, err := discoveryService.DiscoverProvider(ctx, provider, discovery.DiscoveryOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error discovering resources: %v\n"), err)
		os.Exit(1)
	}
	
	// Perform quick check analysis with filtering options
	result := performQuickCheck(stateFile, discoveryResult, smart, securityOnly, noFilter, ignoreCosmetic)
	
	// Display results
	if jsonOutput {
		displayQuickCheckJSON(result)
	} else {
		displayQuickCheckResults(result, verbose, threshold)
	}
	
	// Exit code based on safety
	if !result.SafeToApply {
		os.Exit(1)
	}
}

func performQuickCheck(stateFile *state.State, discovery *discovery.Result, smart, securityOnly, noFilter, ignoreCosmetic bool) QuickCheckResult {
	result := QuickCheckResult{
		SafeToApply: true,
		Confidence:  100.0,
		Issues:      []QuickCheckIssue{},
		Summary: QuickCheckSummary{
			TotalResources: len(stateFile.Resources),
		},
		Timestamp: time.Now(),
	}
	
	// Map discovered resources
	discoveredMap := make(map[string]bool)
	for _, res := range discovery.Resources {
		discoveredMap[res.ID] = true
	}
	
	// Check each state resource
	for _, resource := range stateFile.Resources {
		// Check if resource exists
		if !discoveredMap[resource.ID] {
			// Apply filtering based on flags
			if shouldFilterDrift(resource, smart, securityOnly, noFilter, ignoreCosmetic) {
				continue
			}
			
			result.Summary.DriftedResources++
			
			// Check if it's a destructive change
			if isDestructiveResource(resource.Type) {
				result.Summary.DestructiveChanges++
				result.Issues = append(result.Issues, QuickCheckIssue{
					Severity:    "CRITICAL",
					Resource:    fmt.Sprintf("%s.%s", resource.Type, resource.Name),
					Type:        "MISSING_RESOURCE",
					Description: "Resource exists in state but not in cloud",
					Impact:      "Terraform will attempt to recreate this resource",
				})
				result.Confidence -= 20.0
			} else {
				result.Issues = append(result.Issues, QuickCheckIssue{
					Severity:    "WARNING",
					Resource:    fmt.Sprintf("%s.%s", resource.Type, resource.Name),
					Type:        "DRIFT",
					Description: "Resource configuration has drifted",
					Impact:      "Terraform will update this resource",
				})
				result.Confidence -= 5.0
			}
		}
		
		// Check for security-sensitive resources
		if isSecurityResource(resource.Type) {
			result.Summary.SecurityImpact = true
			if result.Summary.DriftedResources > 0 {
				result.Issues = append(result.Issues, QuickCheckIssue{
					Severity:    "HIGH",
					Resource:    fmt.Sprintf("%s.%s", resource.Type, resource.Name),
					Type:        "SECURITY",
					Description: "Security-sensitive resource has potential drift",
					Impact:      "Review security implications before applying",
				})
				result.Confidence -= 10.0
			}
		}
	}
	
	// Calculate risk level
	if result.Confidence >= 90 {
		result.Summary.EstimatedRisk = "LOW"
	} else if result.Confidence >= 70 {
		result.Summary.EstimatedRisk = "MEDIUM"
	} else if result.Confidence >= 50 {
		result.Summary.EstimatedRisk = "HIGH"
	} else {
		result.Summary.EstimatedRisk = "CRITICAL"
		result.SafeToApply = false
	}
	
	// Check for critical issues
	for _, issue := range result.Issues {
		if issue.Severity == "CRITICAL" {
			result.Summary.CriticalDrifts++
		}
	}
	
	if result.Summary.CriticalDrifts > 0 {
		result.SafeToApply = false
	}
	
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	
	return result
}

func displayQuickCheckResults(result QuickCheckResult, verbose bool, threshold float64) {
	// Summary Box
	if result.SafeToApply {
		fmt.Println(color.GreenString("✅ SAFE TO APPLY"))
	} else {
		fmt.Println(color.RedString("❌ NOT SAFE TO APPLY"))
	}
	
	fmt.Printf("\nConfidence: %s\n", formatConfidence(result.Confidence))
	fmt.Printf("Risk Level: %s\n", formatRisk(result.Summary.EstimatedRisk))
	
	// Statistics
	fmt.Println("\n" + color.CyanString("📊 Statistics"))
	fmt.Println(strings.Repeat("-", 31))
	fmt.Printf("Total Resources:     %d\n", result.Summary.TotalResources)
	fmt.Printf("Drifted Resources:   %d\n", result.Summary.DriftedResources)
	fmt.Printf("Critical Drifts:     %d\n", result.Summary.CriticalDrifts)
	fmt.Printf("Destructive Changes: %d\n", result.Summary.DestructiveChanges)
	
	if result.Summary.SecurityImpact {
		fmt.Printf("Security Impact:     %s\n", color.YellowString("YES"))
	}
	
	// Issues
	if len(result.Issues) > 0 {
		fmt.Println("\n" + color.CyanString("⚠️  Issues Found"))
		fmt.Println(strings.Repeat("-", 31))
		
		// Group by severity
		critical := []QuickCheckIssue{}
		high := []QuickCheckIssue{}
		warnings := []QuickCheckIssue{}
		
		for _, issue := range result.Issues {
			switch issue.Severity {
			case "CRITICAL":
				critical = append(critical, issue)
			case "HIGH":
				high = append(high, issue)
			default:
				warnings = append(warnings, issue)
			}
		}
		
		// Show critical first
		if len(critical) > 0 {
			fmt.Println(color.RedString("\nCRITICAL:"))
			for _, issue := range critical {
				if verbose {
					fmt.Printf("  • %s\n", issue.Resource)
					fmt.Printf("    %s\n", issue.Description)
					fmt.Printf("    Impact: %s\n", issue.Impact)
				} else {
					fmt.Printf("  • %s: %s\n", issue.Resource, issue.Description)
				}
			}
		}
		
		// Show high severity
		if len(high) > 0 {
			fmt.Println(color.YellowString("\nHIGH:"))
			for _, issue := range high {
				if verbose {
					fmt.Printf("  • %s\n", issue.Resource)
					fmt.Printf("    %s\n", issue.Description)
					fmt.Printf("    Impact: %s\n", issue.Impact)
				} else {
					fmt.Printf("  • %s: %s\n", issue.Resource, issue.Description)
				}
			}
		}
		
		// Show warnings if verbose
		if verbose && len(warnings) > 0 {
			fmt.Println("\nWARNING:")
			for _, issue := range warnings {
				fmt.Printf("  • %s: %s\n", issue.Resource, issue.Description)
			}
		}
	}
	
	// Recommendations
	fmt.Println("\n" + color.CyanString("💡 Recommendations"))
	fmt.Println(strings.Repeat("-", 31))
	
	if result.SafeToApply {
		fmt.Println(color.GreenString("• Safe to run terraform apply"))
		if result.Summary.DriftedResources > 0 {
			fmt.Println("• Review the planned changes with terraform plan")
		}
	} else {
		fmt.Println(color.RedString("• DO NOT run terraform apply without review"))
		if result.Summary.CriticalDrifts > 0 {
			fmt.Println(color.YellowString("• Investigate critical drift issues first"))
		}
		if result.Summary.DestructiveChanges > 0 {
			fmt.Println(color.YellowString("• Review destructive changes carefully"))
		}
		if result.Summary.SecurityImpact {
			fmt.Println(color.YellowString("• Security review required"))
		}
		fmt.Println("• Run 'terraform plan' to see detailed changes")
		fmt.Println("• Consider using 'driftmgr state fix' to generate remediation")
	}
}

func formatConfidence(confidence float64) string {
	bar := ""
	filled := int(confidence / 10)
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	
	colorFunc := color.GreenString
	if confidence < 50 {
		colorFunc = color.RedString
	} else if confidence < 80 {
		colorFunc = color.YellowString
	}
	
	return fmt.Sprintf("%s %.1f%%", colorFunc(bar), confidence)
}

func formatRisk(risk string) string {
	switch risk {
	case "LOW":
		return color.GreenString(risk)
	case "MEDIUM":
		return color.YellowString(risk)
	case "HIGH":
		return color.RedString(risk)
	case "CRITICAL":
		return color.RedString("⚠ " + risk)
	default:
		return risk
	}
}

func isDestructiveResource(resourceType string) bool {
	destructive := []string{
		"aws_instance",
		"aws_db_instance",
		"aws_s3_bucket",
		"azurerm_virtual_machine",
		"azurerm_storage_account",
		"google_compute_instance",
		"google_storage_bucket",
	}
	
	for _, d := range destructive {
		if resourceType == d {
			return true
		}
	}
	return false
}

func isSecurityResource(resourceType string) bool {
	security := []string{
		"aws_security_group",
		"aws_iam_role",
		"aws_iam_policy",
		"aws_kms_key",
		"azurerm_network_security_group",
		"azurerm_role_assignment",
		"google_compute_firewall",
		"google_project_iam_member",
	}
	
	for _, s := range security {
		if resourceType == s {
			return true
		}
	}
	return false
}

func detectProvider(stateFile *state.State) string {
	for _, resource := range stateFile.Resources {
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

func findTerraformStateFile() string {
	// Check for Terragrunt first
	if _, err := os.Stat("terragrunt.hcl"); err == nil {
		// Look for Terragrunt state
		patterns := []string{
			".terragrunt-cache/*/terraform.tfstate",
			"terraform.tfstate",
		}
		for _, pattern := range patterns {
			if matches, err := filepath.Glob(pattern); err == nil && len(matches) > 0 {
				return matches[0]
			}
		}
	}
	
	// Standard Terraform locations
	locations := []string{
		"terraform.tfstate",
		".terraform/terraform.tfstate",
		"terraform.tfstate.d/*/terraform.tfstate",
	}
	
	for _, loc := range locations {
		if matches, err := filepath.Glob(loc); err == nil && len(matches) > 0 {
			return matches[0]
		}
	}
	
	return ""
}

func displayQuickCheckJSON(result QuickCheckResult) {
	// Simple JSON output for scripting
	fmt.Printf(`{
  "safe_to_apply": %t,
  "confidence": %.2f,
  "risk": "%s",
  "summary": {
    "total_resources": %d,
    "drifted_resources": %d,
    "critical_drifts": %d,
    "destructive_changes": %d,
    "security_impact": %t
  },
  "issues": %d,
  "timestamp": "%s"
}
`, 
		result.SafeToApply,
		result.Confidence,
		result.Summary.EstimatedRisk,
		result.Summary.TotalResources,
		result.Summary.DriftedResources,
		result.Summary.CriticalDrifts,
		result.Summary.DestructiveChanges,
		result.Summary.SecurityImpact,
		len(result.Issues),
		result.Timestamp.Format(time.RFC3339))
}

func showQuickCheckHelp() {
	fmt.Println("Usage: driftmgr check [flags]")
	fmt.Println()
	fmt.Println("Quick drift check - is it safe to apply?")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --state string      Path to Terraform state file (auto-detected if not specified)")
	fmt.Println("  --threshold float   Confidence threshold for safe-to-apply (default: 80.0)")
	fmt.Println("  --verbose, -v       Show detailed information")
	fmt.Println("  --json              Output in JSON format")
	fmt.Println()
	fmt.Println("Filtering Flags:")
	fmt.Println("  --smart             Use intelligent filtering to reduce noise (default)")
	fmt.Println("  --security-only     Focus only on security-critical drift")
	fmt.Println("  --no-filter         Show all drift without filtering")
	fmt.Println("  --include-cosmetic  Include cosmetic changes (tags, descriptions)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Quick check with auto-detection and smart filtering")
	fmt.Println("  driftmgr check")
	fmt.Println()
	fmt.Println("  # Security-focused check")
	fmt.Println("  driftmgr check --security-only")
	fmt.Println()
	fmt.Println("  # Check specific state file without filtering")
	fmt.Println("  driftmgr check --state terraform.tfstate --no-filter")
	fmt.Println()
	fmt.Println("  # Verbose output with custom threshold")
	fmt.Println("  driftmgr check --verbose --threshold 90")
}

// Filtering helper functions

func shouldFilterDrift(resource interface{}, smart, securityOnly, noFilter, ignoreCosmetic bool) bool {
	// If no filtering, show everything
	if noFilter {
		return false
	}
	
	// Security-only mode
	if securityOnly {
		return !isSecurityRelated(resource)
	}
	
	// Smart filtering mode
	if smart {
		// Filter out known cosmetic changes
		if ignoreCosmetic && isCosmeticChange(resource) {
			return true
		}
		
		// Filter out expected drift
		if isExpectedDrift(resource) {
			return true
		}
	}
	
	return false
}

func isSecurityRelated(resource interface{}) bool {
	// Check if resource type is security-related
	securityTypes := []string{
		"security_group", "iam_role", "iam_policy", "kms_key",
		"network_acl", "firewall", "ssl_certificate", "secret",
		"vault", "key", "encryption", "auth", "access",
	}
	
	resourceStr := fmt.Sprintf("%v", resource)
	for _, secType := range securityTypes {
		if strings.Contains(strings.ToLower(resourceStr), secType) {
			return true
		}
	}
	
	return false
}

func isCosmeticChange(resource interface{}) bool {
	// Identify cosmetic changes that don't affect functionality
	cosmeticAttrs := []string{
		"tags", "description", "comment", "created_at", 
		"updated_at", "last_modified", "etag", "version_id",
	}
	
	resourceStr := fmt.Sprintf("%v", resource)
	for _, attr := range cosmeticAttrs {
		if strings.Contains(strings.ToLower(resourceStr), attr) {
			return true
		}
	}
	
	return false
}

func isExpectedDrift(resource interface{}) bool {
	// Identify drift that is expected and safe
	expectedPatterns := []string{
		"autoscaling", "timestamp", "metric", "log",
		"monitoring", "backup", "snapshot",
	}
	
	resourceStr := fmt.Sprintf("%v", resource)
	for _, pattern := range expectedPatterns {
		if strings.Contains(strings.ToLower(resourceStr), pattern) {
			return true
		}
	}
	
	return false
}