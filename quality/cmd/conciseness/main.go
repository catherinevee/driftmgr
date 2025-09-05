package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/catherinevee/driftmgr/quality"
)

func main() {
	var (
		projectPath = flag.String("project", ".", "Path to the project to analyze")
		outputPath  = flag.String("output", "", "Path to output JSON report")
	)
	flag.Parse()

	if *outputPath == "" {
		log.Fatal("Output path is required")
	}

	analyzer := quality.NewConcisenessAnalyzer()
	allIssues := []quality.ConcisenessIssue{}

	// Walk through Go files
	err := filepath.Walk(*projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and test files
		if strings.Contains(path, "vendor") || strings.Contains(path, ".git") {
			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			issues, err := analyzer.AnalyzeFile(path)
			if err != nil {
				log.Printf("Warning: failed to analyze %s: %v", path, err)
				return nil
			}
			allIssues = append(allIssues, issues...)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to walk project: %v", err)
	}

	// Create report
	report := map[string]interface{}{
		"project":      *projectPath,
		"total_issues": len(allIssues),
		"issues":       allIssues,
		"summary": map[string]int{
			"verbose_conditionals": countByType(allIssues, "verbose_conditional"),
			"unnecessary_else":     countByType(allIssues, "unnecessary_else"),
			"redundant_variables":  countByType(allIssues, "redundant_variable"),
			"verbose_loops":        countByType(allIssues, "verbose_loop"),
			"verbose_nil_checks":   countByType(allIssues, "verbose_nil_check"),
		},
		"potential_savings": estimateSavings(allIssues),
	}

	// Write output
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal report: %v", err)
	}

	if err := os.WriteFile(*outputPath, data, 0644); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	fmt.Printf("Conciseness analysis complete. Found %d issues.\n", len(allIssues))
	fmt.Printf("Report saved to %s\n", *outputPath)

	if len(allIssues) > 0 {
		fmt.Println("\nTop issues:")
		for i, issue := range allIssues {
			if i >= 5 {
				break
			}
			fmt.Printf("  - %s:%d - %s\n", filepath.Base(issue.File), issue.Line, issue.Message)
		}
	}
}

func countByType(issues []quality.ConcisenessIssue, issueType string) int {
	count := 0
	for _, issue := range issues {
		if issue.Type == issueType {
			count++
		}
	}
	return count
}

func estimateSavings(issues []quality.ConcisenessIssue) map[string]interface{} {
	// Estimate lines that could be saved
	linesReduced := 0
	for _, issue := range issues {
		switch issue.Type {
		case "unnecessary_else":
			linesReduced += 2 // else block can be unindented
		case "redundant_variable":
			linesReduced += 1 // one line can be removed
		case "verbose_loop":
			linesReduced += 2 // loop can be simplified
		}
	}

	return map[string]interface{}{
		"lines_reducible":     linesReduced,
		"readability_impact":  "positive",
		"estimated_reduction": fmt.Sprintf("%.1f%%", float64(linesReduced)*0.1), // rough estimate
	}
}
