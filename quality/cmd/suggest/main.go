package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type ConcisenessReport struct {
	Issues  []Issue        `json:"issues"`
	Summary map[string]int `json:"summary"`
}

type Issue struct {
	Type       string `json:"Type"`
	File       string `json:"File"`
	Line       int    `json:"Line"`
	Message    string `json:"Message"`
	Original   string `json:"Original"`
	Improved   string `json:"Improved"`
	Suggestion string `json:"Suggestion"`
}

func main() {
	var (
		inputPath  = flag.String("input", "", "Path to conciseness report JSON")
		outputPath = flag.String("output", "", "Path to output suggestions markdown")
	)
	flag.Parse()

	if *inputPath == "" {
		log.Fatal("Input path is required")
	}
	if *outputPath == "" {
		log.Fatal("Output path is required")
	}

	// Load report
	data, err := os.ReadFile(*inputPath)
	if err != nil {
		log.Fatalf("Failed to read report: %v", err)
	}

	var report ConcisenessReport
	if err := json.Unmarshal(data, &report); err != nil {
		log.Fatalf("Failed to parse report: %v", err)
	}

	// Generate suggestions
	suggestions := generateSuggestions(report)

	// Write output
	if err := os.WriteFile(*outputPath, []byte(suggestions), 0644); err != nil {
		log.Fatalf("Failed to write suggestions: %v", err)
	}

	fmt.Printf("Suggestions generated: %s\n", *outputPath)
	fmt.Printf("Total issues: %d\n", len(report.Issues))
}

func generateSuggestions(report ConcisenessReport) string {
	var sb strings.Builder

	sb.WriteString("# Code Conciseness Suggestions\n\n")

	if len(report.Issues) == 0 {
		sb.WriteString("✅ No conciseness issues found! Your code is already quite concise.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Found **%d** opportunities to improve code conciseness.\n\n", len(report.Issues)))

	// Group by file
	fileIssues := make(map[string][]Issue)
	for _, issue := range report.Issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	// Priority issues
	sb.WriteString("## Priority Improvements\n\n")
	priorityCount := 0
	for file, issues := range fileIssues {
		if len(issues) >= 3 { // Files with 3+ issues are priority
			sb.WriteString(fmt.Sprintf("### %s (%d issues)\n\n", filepath.Base(file), len(issues)))
			for _, issue := range issues {
				sb.WriteString(fmt.Sprintf("- **Line %d**: %s\n", issue.Line, issue.Message))
				if issue.Original != "" && issue.Improved != "" {
					sb.WriteString("  ```go\n")
					sb.WriteString(fmt.Sprintf("  // Before:\n  %s\n", issue.Original))
					sb.WriteString(fmt.Sprintf("  // After:\n  %s\n", issue.Improved))
					sb.WriteString("  ```\n")
				}
				priorityCount++
				if priorityCount >= 10 {
					break
				}
			}
			sb.WriteString("\n")
		}
		if priorityCount >= 10 {
			break
		}
	}

	// Summary by type
	sb.WriteString("## Summary by Issue Type\n\n")
	if summary := report.Summary; summary != nil {
		for issueType, count := range summary {
			if count > 0 {
				sb.WriteString(fmt.Sprintf("- **%s**: %d occurrences\n", formatIssueType(issueType), count))
			}
		}
	}
	sb.WriteString("\n")

	// Actionable steps
	sb.WriteString("## Recommended Actions\n\n")
	sb.WriteString("1. **Quick Wins**: Focus on removing unnecessary else blocks and redundant variables\n")
	sb.WriteString("2. **Readability**: Simplify verbose conditionals (e.g., `x == true` → `x`)\n")
	sb.WriteString("3. **Loops**: Use variadic append for simple slice copies\n")
	sb.WriteString("4. **Automation**: Run `go run quality/cmd/refactor/main.go --safe-only` to auto-fix safe issues\n\n")

	// Instructions
	sb.WriteString("## How to Apply\n\n")
	sb.WriteString("To automatically apply safe refactorings:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go run quality/cmd/refactor/main.go \\\n")
	sb.WriteString("  --input conciseness-report.json \\\n")
	sb.WriteString("  --safe-only\n")
	sb.WriteString("```\n\n")
	sb.WriteString("For manual review and application:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go run quality/cmd/refactor/main.go \\\n")
	sb.WriteString("  --input conciseness-report.json \\\n")
	sb.WriteString("  --dry-run\n")
	sb.WriteString("```\n")

	return sb.String()
}

func formatIssueType(issueType string) string {
	switch issueType {
	case "verbose_conditionals":
		return "Verbose Conditionals"
	case "unnecessary_else":
		return "Unnecessary Else Blocks"
	case "redundant_variables":
		return "Redundant Variables"
	case "verbose_loops":
		return "Verbose Loops"
	case "verbose_nil_checks":
		return "Verbose Nil Checks"
	default:
		return strings.Title(strings.ReplaceAll(issueType, "_", " "))
	}
}
