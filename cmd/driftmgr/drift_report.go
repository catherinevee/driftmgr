package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/drift"
)

// handleDriftReport generates a drift report
func handleDriftReport(args []string) {
	var statePath, format, outputPath, provider string
	var includeRecommendations, includeCosts bool

	// Default values
	format = "summary"
	includeRecommendations = true

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(args) {
				outputPath = args[i+1]
				i++
			}
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--include-costs":
			includeCosts = true
		case "--no-recommendations":
			includeRecommendations = false
		case "--help", "-h":
			showDriftReportHelp()
			return
		}
	}

	// Auto-detect state file if not specified
	if statePath == "" {
		statePath = autoDetectStateFile()
		if statePath == "" {
			fmt.Fprintf(os.Stderr, "Error: No state file specified and none found automatically\n")
			fmt.Println("Use --state <path> to specify a state file")
			os.Exit(1)
		}
		fmt.Printf("Using detected state file: %s\n", statePath)
	}

	// Load and analyze drift
	detector := drift.NewTerraformDriftDetector(statePath, provider)

	// Detect drift using the state file and provider
	ctx := context.Background()
	report, err := detector.DetectDrift(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting drift: %v\n", err)
		os.Exit(1)
	}

	// Generate report based on format
	switch format {
	case "json":
		generateJSONReport(report, outputPath)
	case "html":
		generateHTMLReport(report, outputPath, includeRecommendations, includeCosts)
	case "pdf":
		generatePDFReport(report, outputPath)
	case "markdown":
		generateMarkdownReport(report, outputPath, includeRecommendations, includeCosts)
	case "summary":
		generateSummaryReport(report, includeRecommendations, includeCosts)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", format)
		showDriftReportHelp()
		os.Exit(1)
	}
}

// autoDetectStateFile attempts to find a Terraform state file automatically
func autoDetectStateFile() string {
	// Check common locations in order of preference
	commonPaths := []string{
		"terraform.tfstate",
		"./terraform.tfstate",
		".terraform/terraform.tfstate",
		"infrastructure/terraform.tfstate",
		"terraform/terraform.tfstate",
		"state/terraform.tfstate",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Search for any .tfstate file in current directory
	matches, err := filepath.Glob("*.tfstate")
	if err == nil && len(matches) > 0 {
		return matches[0]
	}

	return ""
}

// generateSummaryReport generates a summary drift report to console
func generateSummaryReport(report *drift.TerraformDriftReport, includeRecommendations, includeCosts bool) {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("                    DRIFT DETECTION REPORT")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("State File: %s\n", report.StateFile)
	fmt.Printf("Provider: %s\n", report.Provider)
	fmt.Printf("Scan Duration: %s\n", report.Duration)
	fmt.Println()

	// Executive Summary
	fmt.Println("EXECUTIVE SUMMARY")
	fmt.Println("-" + strings.Repeat("-", 70))
	fmt.Printf("Total Resources: %d\n", report.TotalResources)
	fmt.Printf("Drifted Resources: %d (%.1f%%)\n",
		report.DriftedCount,
		float64(report.DriftedCount)/float64(report.TotalResources)*100)
	fmt.Printf("Missing Resources: %d\n", report.MissingCount)
	fmt.Printf("Unmanaged Resources: %d\n", report.UnmanagedCount)
	fmt.Println()

	// Severity Breakdown
	if len(report.Resources) > 0 {
		fmt.Println("SEVERITY BREAKDOWN")
		fmt.Println("-" + strings.Repeat("-", 70))

		if report.Summary != nil {
			if report.Summary.CriticalCount > 0 {
				fmt.Printf("  Critical: %d\n", report.Summary.CriticalCount)
			}
			if report.Summary.HighCount > 0 {
				fmt.Printf("  High: %d\n", report.Summary.HighCount)
			}
			if report.Summary.MediumCount > 0 {
				fmt.Printf("  Medium: %d\n", report.Summary.MediumCount)
			}
			if report.Summary.LowCount > 0 {
				fmt.Printf("  Low: %d\n", report.Summary.LowCount)
			}
		}
		fmt.Println()
	}

	// Detailed Drift Items
	if len(report.Resources) > 0 {
		fmt.Println("DRIFT DETAILS")
		fmt.Println("-" + strings.Repeat("-", 70))

		// Group by drift type
		byType := make(map[string][]drift.TerraformResource)
		for _, item := range report.Resources {
			byType[item.DriftType] = append(byType[item.DriftType], item)
		}

		for driftType, items := range byType {
			fmt.Printf("\n%s (%d resources):\n", strings.ToUpper(driftType), len(items))
			for _, item := range items {
				icon := getDriftIcon(string(item.Severity))
				fmt.Printf("  %s %s (%s)\n", icon, item.ResourceName, item.ResourceType)

				if item.DriftType == "modified" && len(item.Differences) > 0 {
					fmt.Println("    Changes:")
					for _, change := range item.Differences {
						fmt.Printf("      - %s: %v → %v\n", change.Path, change.StateValue, change.ActualValue)
					}
				}

				if item.Severity != "" {
					fmt.Printf("    Severity: %s\n", item.Severity)
				}
			}
		}
		fmt.Println()
	}

	// Cost Impact
	if includeCosts && report.Summary != nil {
		fmt.Println("COST IMPACT ANALYSIS")
		fmt.Println("-" + strings.Repeat("-", 70))
		fmt.Println("Cost impact analysis not yet available")

		fmt.Println()
	}

	// Recommendations
	if includeRecommendations {
		fmt.Println("RECOMMENDATIONS")
		fmt.Println("-" + strings.Repeat("-", 70))
		if report.DriftedCount > 0 {
			fmt.Println("1. Review the drift details above")
			fmt.Println("   Priority: High")
			fmt.Println("   Identify which changes are intentional vs unintentional")
			fmt.Println()

			fmt.Println("2. Run drift fix in dry-run mode")
			fmt.Println("   Priority: High")
			fmt.Println("   Review the proposed fixes before applying")
			fmt.Printf("   Command: driftmgr drift fix --dry-run --state %s\n", report.StateFile)
			fmt.Println()

			if report.MissingCount > 0 {
				fmt.Println("3. Investigate missing resources")
				fmt.Println("   Priority: Critical")
				fmt.Printf("   %d resources exist in state but not in cloud\n", report.MissingCount)
				fmt.Println()
			}
		}
	}

	// Action Summary
	fmt.Println("ACTION REQUIRED")
	fmt.Println("-" + strings.Repeat("-", 70))
	if report.DriftedCount == 0 {
		fmt.Println("[OK] No drift detected - infrastructure matches desired state")
	} else if report.DriftedCount <= 5 {
		fmt.Println("[WARNING]  Minor drift detected - review and fix individual resources")
		fmt.Println("   Run: driftmgr drift fix --dry-run")
	} else {
		fmt.Println("🔴 Significant drift detected - immediate action recommended")
		fmt.Println("   1. Review drift details above")
		fmt.Println("   2. Run: driftmgr drift fix --dry-run")
		fmt.Println("   3. Apply fixes after review")
	}
	fmt.Println()
}

// generateJSONReport generates a JSON format drift report
func generateJSONReport(report *drift.TerraformDriftReport, outputPath string) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON report: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		err = os.WriteFile(outputPath, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing report to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Report saved to: %s\n", outputPath)
	} else {
		fmt.Println(string(data))
	}
}

// generateHTMLReport generates an HTML format drift report
func generateHTMLReport(report *drift.TerraformDriftReport, outputPath string, includeRecommendations, includeCosts bool) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Drift Detection Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007bff; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .summary { background: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .metric { display: inline-block; margin: 10px 20px 10px 0; }
        .metric-value { font-size: 24px; font-weight: bold; color: #007bff; }
        .metric-label { color: #666; font-size: 14px; }
        .critical { color: #dc3545; }
        .high { color: #fd7e14; }
        .medium { color: #ffc107; }
        .low { color: #28a745; }
        .drift-item { background: #fff; border-left: 4px solid #007bff; padding: 15px; margin: 10px 0; }
        .drift-modified { border-left-color: #ffc107; }
        .drift-missing { border-left-color: #dc3545; }
        .drift-unmanaged { border-left-color: #6c757d; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #007bff; color: white; }
        .recommendation { background: #e7f3ff; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .footer { text-align: center; color: #666; margin-top: 40px; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔍 Drift Detection Report</h1>
        
        <div class="summary">
            <div class="metric">
                <div class="metric-value">` + fmt.Sprintf("%d", report.TotalResources) + `</div>
                <div class="metric-label">Total Resources</div>
            </div>
            <div class="metric">
                <div class="metric-value ` + getSeverityClass(report.DriftedCount) + `">` +
		fmt.Sprintf("%d", report.DriftedCount) + `</div>
                <div class="metric-label">Drifted Resources</div>
            </div>
            <div class="metric">
                <div class="metric-value">` +
		fmt.Sprintf("%.1f%%", float64(report.DriftedCount)/float64(report.TotalResources)*100) + `</div>
                <div class="metric-label">Drift Percentage</div>
            </div>
        </div>`

	// Add drift details
	if len(report.Resources) > 0 {
		html += `<h2>Drift Details</h2>`
		for _, item := range report.Resources {
			html += fmt.Sprintf(`
        <div class="drift-item drift-%s">
            <strong>%s</strong> (%s)<br>
            Type: %s | Severity: <span class="%s">%s</span><br>`,
				item.DriftType, item.ResourceName, item.ResourceType,
				item.DriftType, item.Severity, item.Severity)

			if len(item.Differences) > 0 {
				html += `<br>Changes:<ul>`
				for _, change := range item.Differences {
					html += fmt.Sprintf("<li>%s: %v → %v</li>", change.Path, change.StateValue, change.ActualValue)
				}
				html += `</ul>`
			}
			html += `</div>`
		}
	}

	html += `
        <div class="footer">
            Generated by DriftMgr on ` + time.Now().Format("2006-01-02 15:04:05") + `
        </div>
    </div>
</body>
</html>`

	if outputPath == "" {
		outputPath = "drift-report.html"
	}

	err := os.WriteFile(outputPath, []byte(html), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing HTML report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("HTML report saved to: %s\n", outputPath)
}

// generateMarkdownReport generates a Markdown format drift report
func generateMarkdownReport(report *drift.TerraformDriftReport, outputPath string, includeRecommendations, includeCosts bool) {
	var md strings.Builder

	md.WriteString("# Drift Detection Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**State File**: `%s`\n\n", report.StateFile))
	md.WriteString(fmt.Sprintf("**Provider**: %s\n\n", report.Provider))

	md.WriteString("## Summary\n\n")
	md.WriteString("| Metric | Value |\n")
	md.WriteString("|--------|-------|\n")
	md.WriteString(fmt.Sprintf("| Total Resources | %d |\n", report.TotalResources))
	md.WriteString(fmt.Sprintf("| Drifted Resources | %d |\n", report.DriftedCount))
	md.WriteString(fmt.Sprintf("| Missing Resources | %d |\n", report.MissingCount))
	md.WriteString(fmt.Sprintf("| Unmanaged Resources | %d |\n", report.UnmanagedCount))
	md.WriteString(fmt.Sprintf("| Drift Percentage | %.1f%% |\n\n",
		float64(report.DriftedCount)/float64(report.TotalResources)*100))

	if len(report.Resources) > 0 {
		md.WriteString("## Drift Details\n\n")

		// Group by type
		byType := make(map[string][]drift.TerraformResource)
		for _, item := range report.Resources {
			byType[item.DriftType] = append(byType[item.DriftType], item)
		}

		for driftType, items := range byType {
			md.WriteString(fmt.Sprintf("### %s (%d resources)\n\n", strings.Title(driftType), len(items)))

			for _, item := range items {
				icon := getDriftIcon(string(item.Severity))
				md.WriteString(fmt.Sprintf("- %s **%s** (`%s`)\n", icon, item.ResourceName, item.ResourceType))
				md.WriteString(fmt.Sprintf("  - Severity: %s\n", string(item.Severity)))

				if len(item.Differences) > 0 {
					md.WriteString("  - Changes:\n")
					for _, change := range item.Differences {
						md.WriteString(fmt.Sprintf("    - `%s`: %v → %v\n", change.Path, change.StateValue, change.ActualValue))
					}
				}
			}
			md.WriteString("\n")
		}
	}

	// TODO: Add recommendations support in the future
	// Currently recommendations are not part of the TerraformDriftReport structure

	content := md.String()

	if outputPath == "" {
		fmt.Print(content)
	} else {
		err := os.WriteFile(outputPath, []byte(content), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing Markdown report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Markdown report saved to: %s\n", outputPath)
	}
}

// generatePDFReport generates a PDF format drift report (placeholder)
func generatePDFReport(report *drift.TerraformDriftReport, outputPath string) {
	fmt.Println("PDF report generation is not yet implemented.")
	fmt.Println("Please use --format html and convert to PDF using a browser or tool.")
}

// Helper functions

func getDriftIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func getSeverityClass(count int) string {
	if count == 0 {
		return "low"
	} else if count <= 5 {
		return "medium"
	} else if count <= 10 {
		return "high"
	} else {
		return "critical"
	}
}

func showDriftReportHelp() {
	fmt.Println("Usage: driftmgr drift report [options]")
	fmt.Println()
	fmt.Println("Generate comprehensive drift detection reports")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --state <path>           Path to Terraform state file (auto-detected if not specified)")
	fmt.Println("  --format <format>        Output format: summary, json, html, markdown, pdf")
	fmt.Println("                          (default: summary)")
	fmt.Println("  --output <path>          Output file path (console output if not specified)")
	fmt.Println("  --provider <provider>    Cloud provider (aws, azure, gcp, digitalocean)")
	fmt.Println("  --include-costs          Include cost impact analysis")
	fmt.Println("  --no-recommendations     Exclude remediation recommendations")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate summary report to console")
	fmt.Println("  driftmgr drift report")
	fmt.Println()
	fmt.Println("  # Generate HTML report with costs")
	fmt.Println("  driftmgr drift report --format html --output report.html --include-costs")
	fmt.Println()
	fmt.Println("  # Generate JSON report for automation")
	fmt.Println("  driftmgr drift report --format json --output drift.json")
	fmt.Println()
	fmt.Println("  # Generate Markdown report for documentation")
	fmt.Println("  driftmgr drift report --format markdown --output DRIFT_REPORT.md")
}
