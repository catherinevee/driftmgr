package main

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/drift"
	"github.com/catherinevee/driftmgr/internal/models"
)

// handleEnhancedAnalyze processes enhanced drift analysis with configurable sensitivity
func (shell *InteractiveShell) handleEnhancedAnalyze(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: enhanced-analyze <statefile_id> [options]")
		fmt.Println("Options:")
		fmt.Println("  --sensitive-fields <fields>  Comma-separated list of sensitive fields")
		fmt.Println("  --ignore-fields <fields>     Comma-separated list of fields to ignore")
		fmt.Println("  --config <file>              Load configuration from file")
		fmt.Println("  --output <format>            Output format (json, table, summary)")
		return
	}

	stateFileID := args[0]

	// Parse options
	var sensitiveFields, ignoreFields []string
	var configFile, outputFormat string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--sensitive-fields":
			if i+1 < len(args) {
				sensitiveFields = strings.Split(args[i+1], ",")
				i++
			}
		case "--ignore-fields":
			if i+1 < len(args) {
				ignoreFields = strings.Split(args[i+1], ",")
				i++
			}
		case "--config":
			if i+1 < len(args) {
				configFile = args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(args) {
				outputFormat = args[i+1]
				i++
			}
		}
	}

	// Create enhanced drift detector
	detector := drift.NewAttributeDriftDetector()

	// Load configuration if provided
	if configFile != "" {
		if err := shell.loadDriftConfiguration(detector, configFile); err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			return
		}
	}

	// Add command-line sensitive fields
	for _, field := range sensitiveFields {
		detector.AddSensitiveField(strings.TrimSpace(field))
	}

	// Add command-line ignore fields
	for _, field := range ignoreFields {
		detector.AddIgnoreField(strings.TrimSpace(field))
	}

	// Add default sensitive fields if none specified
	if len(sensitiveFields) == 0 && configFile == "" {
		detector.AddSensitiveField("tags.environment")
		detector.AddSensitiveField("tags.owner")
		detector.AddSensitiveField("tags.cost-center")
		detector.AddSensitiveField("security_groups")
		detector.AddSensitiveField("iam_policies")
	}

	// Add default ignore fields if none specified
	if len(ignoreFields) == 0 && configFile == "" {
		detector.AddIgnoreField("tags.last-updated")
		detector.AddIgnoreField("tags.auto-generated")
		detector.AddIgnoreField("metadata.timestamp")
	}

	// Add custom comparators for complex attributes
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

	fmt.Printf("Performing enhanced drift analysis for state file: %s\n", stateFileID)

	// Get state file resources
	stateFile, err := shell.client.GetStateFile(stateFileID)
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	// Convert Terraform resources to our Resource format
	stateResources := shell.convertTerraformResources(stateFile)

	// Discover live resources
	fmt.Println("Discovering live resources...")
	discoveryResponse, err := shell.client.DiscoverResources("aws", []string{"all"})
	if err != nil {
		fmt.Printf("Error discovering resources: %v\n", err)
		return
	}

	// Perform enhanced drift detection
	fmt.Println("Analyzing drift with enhanced detection...")
	result := detector.DetectDrift(stateResources, discoveryResponse.Resources)

	// Display results based on output format
	shell.displayEnhancedResults(result, outputFormat)
}

// loadDriftConfiguration loads configuration from YAML file
func (shell *InteractiveShell) loadDriftConfiguration(detector *drift.AttributeDriftDetector, configFile string) error {
	// This would parse the driftmgr.yaml configuration file
	// For now, we'll use a simplified approach

	// Example configuration loading
	sensitiveFields := []string{
		"tags.environment",
		"tags.owner",
		"tags.cost-center",
		"tags.security-level",
		"security_groups",
		"iam_policies",
		"encryption_settings",
	}

	for _, field := range sensitiveFields {
		detector.AddSensitiveField(field)
	}

	// Add severity rules from configuration
	severityRules := []drift.SeverityRule{
		{
			ResourceType:  "aws_instance",
			AttributePath: "tags.environment",
			Condition:     "production",
			Severity:      "critical",
			Description:   "Production environment changes are critical",
		},
		{
			ResourceType:  "aws_security_group",
			AttributePath: "ingress_rules",
			Condition:     "any",
			Severity:      "high",
			Description:   "Security group changes are high priority",
		},
		{
			ResourceType:  "aws_s3_bucket",
			AttributePath: "encryption",
			Condition:     "disabled",
			Severity:      "critical",
			Description:   "S3 bucket encryption is critical for compliance",
		},
	}

	for _, rule := range severityRules {
		detector.AddSeverityRule(rule)
	}

	return nil
}

// convertTerraformResources converts Terraform state resources to our Resource format
func (shell *InteractiveShell) convertTerraformResources(stateFile *models.StateFile) []models.Resource {
	var resources []models.Resource

	for _, tfResource := range stateFile.Resources {
		for _, instance := range tfResource.Instances {
			if id, ok := instance.Attributes["id"].(string); ok {
				// Extract name from tags or use the resource name
				name := tfResource.Name
				if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
					if nameTag, ok := tags["Name"].(string); ok {
						name = nameTag
					}
				}

				// Extract region from attributes or use default
				region := "us-east-1" // default
				if regionAttr, ok := instance.Attributes["availability_zone"].(string); ok {
					if strings.Contains(regionAttr, "us-east-1") {
						region = "us-east-1"
					} else if strings.Contains(regionAttr, "us-west-2") {
						region = "us-west-2"
					}
				}

				// Extract tags
				tags := make(map[string]string)
				if tagsAttr, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
					for key, value := range tagsAttr {
						if strValue, ok := value.(string); ok {
							tags[key] = strValue
						}
					}
				}

				resource := models.Resource{
					ID:       id,
					Name:     name,
					Type:     tfResource.Type,
					Provider: "aws",
					Region:   region,
					State:    "active",
					Tags:     tags,
				}

				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// displayEnhancedResults displays the enhanced drift analysis results
func (shell *InteractiveShell) displayEnhancedResults(result models.AnalysisResult, format string) {
	switch format {
	case "json":
		shell.displayJSONResults(result)
	case "table":
		shell.displayTableResults(result)
	case "summary":
		shell.displaySummaryResults(result)
	default:
		shell.displayDetailedResults(result)
	}
}

// displayDetailedResults shows comprehensive drift analysis results
func (shell *InteractiveShell) displayDetailedResults(result models.AnalysisResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("ENHANCED DRIFT ANALYSIS RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	// Summary section
	fmt.Printf("\nSUMMARY:\n")
	fmt.Printf("   Total Drifts: %d\n", result.Summary.TotalDrifts)
	fmt.Printf("   Critical: %d\n", result.Summary.CriticalDrifts)
	fmt.Printf("   High: %d\n", result.Summary.HighDrifts)
	fmt.Printf("   Medium: %d\n", result.Summary.MediumDrifts)
	fmt.Printf("   Low: %d\n", result.Summary.LowDrifts)
	fmt.Printf("   Coverage: %.2f%%\n", result.Summary.CoveragePercentage)
	fmt.Printf("   Drift Percentage: %.2f%%\n", result.Summary.DriftPercentage)
	fmt.Printf("   Perspective: %.2f%%\n", result.Summary.PerspectivePercentage)

	// Detailed results
	if len(result.DriftResults) > 0 {
		fmt.Printf("\nDETAILED DRIFT RESULTS:\n")
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
	} else {
		fmt.Printf("\nNo drift detected!\n")
	}

	// Recommendations
	if len(result.DriftResults) > 0 {
		fmt.Printf("\nRECOMMENDATIONS:\n")
		for _, drift := range result.DriftResults {
			switch drift.DriftType {
			case "missing":
				fmt.Printf("   • Import missing resource: %s\n", drift.ResourceName)
			case "extra":
				fmt.Printf("   • Review unmanaged resource: %s\n", drift.ResourceName)
			case "modified":
				fmt.Printf("   • Fix drift in resource: %s\n", drift.ResourceName)
			}
		}
	}
}

// displaySummaryResults shows only summary information
func (shell *InteractiveShell) displaySummaryResults(result models.AnalysisResult) {
	fmt.Printf("Drift Summary: %d total drifts (Critical: %d, High: %d, Medium: %d, Low: %d)\n",
		result.Summary.TotalDrifts,
		result.Summary.CriticalDrifts,
		result.Summary.HighDrifts,
		result.Summary.MediumDrifts,
		result.Summary.LowDrifts)
}

// displayJSONResults shows results in JSON format
func (shell *InteractiveShell) displayJSONResults(result models.AnalysisResult) {
	// Implementation for JSON output
	fmt.Printf("JSON output would be displayed here\n")
}

// displayTableResults shows results in table format
func (shell *InteractiveShell) displayTableResults(result models.AnalysisResult) {
	// Implementation for table output
	fmt.Printf("Table output would be displayed here\n")
}
