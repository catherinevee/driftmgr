package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/catherinevee/driftmgr/quality"
)

func main() {
	var (
		inputPath  = flag.String("input", "", "Path to quality report JSON")
		format     = flag.String("format", "markdown", "Output format (markdown, json, text)")
		outputPath = flag.String("output", "", "Path to output file")
	)
	flag.Parse()

	if *inputPath == "" {
		log.Fatal("Input path is required")
	}
	if *outputPath == "" {
		log.Fatal("Output path is required")
	}

	// Load report
	reportData, err := quality.LoadReport(*inputPath)
	if err != nil {
		// Try loading as generic JSON
		data, err := os.ReadFile(*inputPath)
		if err != nil {
			log.Fatalf("Failed to read report: %v", err)
		}
		
		var genericReport map[string]interface{}
		if err := json.Unmarshal(data, &genericReport); err != nil {
			log.Fatalf("Failed to parse report: %v", err)
		}

		// Convert to markdown
		output := generateMarkdownFromGeneric(genericReport)
		if err := os.WriteFile(*outputPath, []byte(output), 0644); err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}
		fmt.Printf("Report generated: %s\n", *outputPath)
		return
	}

	// Generate output based on format
	var output string
	switch *format {
	case "markdown":
		output = generateMarkdown(reportData)
	case "json":
		data, _ := json.MarshalIndent(reportData, "", "  ")
		output = string(data)
	case "text":
		output = generateText(reportData)
	default:
		log.Fatalf("Unknown format: %s", *format)
	}

	// Write output
	if err := os.WriteFile(*outputPath, []byte(output), 0644); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	fmt.Printf("Report generated: %s\n", *outputPath)
}

func generateMarkdown(report *quality.Report) string {
	var sb strings.Builder

	sb.WriteString("## Code Quality Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))
	
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Quality Score:** %.1f/100\n", report.Summary.QualityScore))
	sb.WriteString(fmt.Sprintf("- **UAT Pass Rate:** %.1f%%\n", report.Summary.UATPassRate))
	sb.WriteString(fmt.Sprintf("- **Code Coverage:** %.1f%%\n", report.Summary.CodeCoverage))
	sb.WriteString(fmt.Sprintf("- **Technical Debt:** %d hours\n", report.Summary.TechnicalDebt))
	sb.WriteString(fmt.Sprintf("- **Risk Level:** %s\n", report.Summary.RiskLevel))
	sb.WriteString(fmt.Sprintf("- **Total Violations:** %d\n\n", report.Summary.TotalViolations))

	sb.WriteString("### Quality Metrics\n\n")
	sb.WriteString(fmt.Sprintf("- **Average Complexity:** %.1f\n", report.QualityMetrics.AvgComplexity))
	sb.WriteString(fmt.Sprintf("- **Max Complexity:** %d\n", report.QualityMetrics.MaxComplexity))
	sb.WriteString(fmt.Sprintf("- **Complex Files:** %d\n", report.QualityMetrics.ComplexFiles))
	sb.WriteString(fmt.Sprintf("- **Maintainability Index:** %.1f\n", report.QualityMetrics.MaintainabilityIndex))
	sb.WriteString(fmt.Sprintf("- **Documentation Coverage:** %.1f%%\n", report.QualityMetrics.DocCoverage))
	sb.WriteString(fmt.Sprintf("- **Code Duplication:** %.1f%%\n\n", report.QualityMetrics.DuplicationPercent))

	if len(report.Recommendations) > 0 {
		sb.WriteString("### Recommendations\n\n")
		for _, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("#### %s (%s Priority)\n", rec.Title, rec.Priority))
			sb.WriteString(fmt.Sprintf("- **Effort:** %s\n", rec.Effort))
			sb.WriteString(fmt.Sprintf("- **Impact:** %s\n", rec.Impact))
			sb.WriteString(fmt.Sprintf("- %s\n\n", rec.Description))
		}
	}

	if len(report.ActionItems) > 0 {
		sb.WriteString("### Action Items\n\n")
		for _, item := range report.ActionItems {
			status := "⬜"
			if item.Completed {
				status = "✅"
			}
			sb.WriteString(fmt.Sprintf("%s **%s** - %s\n", status, item.Priority, item.Description))
		}
	}

	return sb.String()
}

func generateMarkdownFromGeneric(data map[string]interface{}) string {
	var sb strings.Builder
	
	sb.WriteString("## Code Quality Report\n\n")
	
	if summary, ok := data["summary"].(map[string]interface{}); ok {
		sb.WriteString("### Summary\n\n")
		if score, ok := summary["quality_score"].(float64); ok {
			sb.WriteString(fmt.Sprintf("- **Quality Score:** %.1f/100\n", score))
		}
		if coverage, ok := summary["test_coverage"].(float64); ok {
			sb.WriteString(fmt.Sprintf("- **Test Coverage:** %.1f%%\n", coverage))
		}
		if files, ok := summary["total_files"].(float64); ok {
			sb.WriteString(fmt.Sprintf("- **Total Files:** %d\n", int(files)))
		}
		if lines, ok := summary["total_lines"].(float64); ok {
			sb.WriteString(fmt.Sprintf("- **Total Lines:** %d\n", int(lines)))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

func generateText(report *quality.Report) string {
	var sb strings.Builder

	sb.WriteString("CODE QUALITY REPORT\n")
	sb.WriteString("==================\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))
	
	sb.WriteString("SUMMARY\n")
	sb.WriteString("-------\n")
	sb.WriteString(fmt.Sprintf("Quality Score: %.1f/100\n", report.Summary.QualityScore))
	sb.WriteString(fmt.Sprintf("UAT Pass Rate: %.1f%%\n", report.Summary.UATPassRate))
	sb.WriteString(fmt.Sprintf("Code Coverage: %.1f%%\n", report.Summary.CodeCoverage))
	sb.WriteString(fmt.Sprintf("Risk Level: %s\n", report.Summary.RiskLevel))

	return sb.String()
}