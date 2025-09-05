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

type ConcisenessReport struct {
	Issues []quality.ConcisenessIssue `json:"issues"`
}

func main() {
	var (
		inputPath    = flag.String("input", "", "Path to conciseness report JSON")
		dryRun       = flag.Bool("dry-run", false, "Show changes without applying")
		safeOnly     = flag.Bool("safe-only", false, "Apply only safe refactorings")
		autoApprove  = flag.Bool("auto-approve", false, "Automatically approve all changes")
		outputPath   = flag.String("output", "", "Path to output diff file (for dry-run)")
	)
	flag.Parse()

	// For automated refactoring, load from current directory analysis
	if *autoApprove && *safeOnly {
		// Run safe automated improvements
		projectPath := "."
		applySafeRefactorings(projectPath, *dryRun, *outputPath)
		return
	}

	if *inputPath == "" {
		// If no input, analyze current directory
		analyzer := quality.NewConcisenessAnalyzer()
		projectPath := "."
		
		var allIssues []quality.ConcisenessIssue
		err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
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
			log.Fatalf("Failed to analyze project: %v", err)
		}
		
		processIssues(allIssues, *dryRun, *safeOnly, *outputPath)
		return
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

	processIssues(report.Issues, *dryRun, *safeOnly, *outputPath)
}

func processIssues(issues []quality.ConcisenessIssue, dryRun, safeOnly bool, outputPath string) {
	if len(issues) == 0 {
		fmt.Println("No issues to refactor")
		return
	}

	var diffs []string
	refactored := 0

	// Group by file
	fileIssues := make(map[string][]quality.ConcisenessIssue)
	for _, issue := range issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	for file, fileIssueList := range fileIssues {
		if safeOnly && !isSafeRefactoring(fileIssueList) {
			continue
		}

		if dryRun {
			// Generate diff
			for _, issue := range fileIssueList {
				diff := fmt.Sprintf("--- %s\n+++ %s\n@@ -%d,1 +%d,1 @@\n-%s\n+%s\n",
					file, file, issue.Line, issue.Line,
					issue.Original, issue.Improved)
				diffs = append(diffs, diff)
			}
			refactored += len(fileIssueList)
		} else {
			// Actually refactor
			err := quality.RefactorFile(file)
			if err != nil {
				log.Printf("Failed to refactor %s: %v", file, err)
			} else {
				refactored += len(fileIssueList)
				fmt.Printf("Refactored %s (%d issues)\n", filepath.Base(file), len(fileIssueList))
			}
		}
	}

	if dryRun {
		output := strings.Join(diffs, "\n")
		if outputPath != "" {
			if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
				log.Fatalf("Failed to write diff: %v", err)
			}
			fmt.Printf("Diff saved to %s\n", outputPath)
		} else {
			fmt.Println(output)
		}
	}

	fmt.Printf("\nRefactoring complete: %d issues %s\n", refactored,
		map[bool]string{true: "would be fixed", false: "fixed"}[dryRun])
}

func applySafeRefactorings(projectPath string, dryRun bool, outputPath string) {
	analyzer := quality.NewConcisenessAnalyzer()
	refactored := 0
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip vendor, .git, and test files
		if strings.Contains(path, "vendor") || 
		   strings.Contains(path, ".git") || 
		   strings.HasSuffix(path, "_test.go") {
			return nil
		}
		
		if strings.HasSuffix(path, ".go") {
			issues, err := analyzer.AnalyzeFile(path)
			if err != nil {
				return nil // Skip files with errors
			}
			
			// Only apply safe refactorings
			safeIssues := filterSafeIssues(issues)
			if len(safeIssues) > 0 && !dryRun {
				if err := quality.RefactorFile(path); err == nil {
					refactored += len(safeIssues)
					fmt.Printf("Refactored %s (%d safe issues)\n", filepath.Base(path), len(safeIssues))
				}
			} else if len(safeIssues) > 0 && dryRun {
				fmt.Printf("Would refactor %s (%d safe issues)\n", filepath.Base(path), len(safeIssues))
				refactored += len(safeIssues)
			}
		}
		return nil
	})
	
	if err != nil {
		log.Fatalf("Failed to walk project: %v", err)
	}
	
	fmt.Printf("\nAutomatic refactoring complete: %d safe issues %s\n", 
		refactored, map[bool]string{true: "would be fixed", false: "fixed"}[dryRun])
}

func isSafeRefactoring(issues []quality.ConcisenessIssue) bool {
	for _, issue := range issues {
		if !isSafeIssueType(issue.Type) {
			return false
		}
	}
	return true
}

func filterSafeIssues(issues []quality.ConcisenessIssue) []quality.ConcisenessIssue {
	var safe []quality.ConcisenessIssue
	for _, issue := range issues {
		if isSafeIssueType(issue.Type) {
			safe = append(safe, issue)
		}
	}
	return safe
}

func isSafeIssueType(issueType string) bool {
	// These are considered safe to auto-refactor
	safeTypes := map[string]bool{
		"verbose_conditional": true,  // if x == true -> if x
		"redundant_variable":  true,  // x := val; return x -> return val
	}
	return safeTypes[issueType]
}