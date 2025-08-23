package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/integration/terraform"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// TerraformStateInfo holds information about a discovered state file
type TerraformStateInfo struct {
	Path             string                 `json:"path"`
	Type             string                 `json:"type"` // local, s3, azurerm, gcs, remote
	Size             int64                  `json:"size"`
	ModifiedTime     time.Time              `json:"modified_time"`
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	ResourceCount    int                    `json:"resource_count"`
	Provider         string                 `json:"provider"`
	Backend          string                 `json:"backend,omitempty"`
	Workspace        string                 `json:"workspace,omitempty"`
	Resources        []StateResourceSummary `json:"resources,omitempty"`
	IsBackup         bool                   `json:"is_backup"`
	IsRemote         bool                   `json:"is_remote"`
	Error            string                 `json:"error,omitempty"`
}

// StateResourceSummary provides a summary of resources in a state file
type StateResourceSummary struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Mode     string `json:"mode"` // managed, data
	Count    int    `json:"count"`
}

// handleTfStateCommand handles terraform state file operations
func handleTfStateCommand(args []string) {
	if len(args) == 0 {
		args = []string{"list"} // Default to list
	}

	switch args[0] {
	case "list":
		handleTfStateList(args[1:])
	case "show":
		handleTfStateShow(args[1:])
	case "analyze":
		handleTfStateAnalyze(args[1:])
	case "find":
		handleTfStateFind(args[1:])
	case "--help", "-h":
		showTfStateHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
		showTfStateHelp()
		os.Exit(1)
	}
}

// handleTfStateList lists all discovered terraform state files
func handleTfStateList(args []string) {
	var dir, format, output string
	var showDetails, includeBackups, recursive bool

	dir = "." // Default to current directory
	format = "table"
	recursive = true

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--details":
			showDetails = true
		case "--include-backups":
			includeBackups = true
		case "--no-recursive":
			recursive = false
		case "--help", "-h":
			fmt.Println("Usage: driftmgr tfstate list [flags]")
			fmt.Println()
			fmt.Println("List all Terraform state files")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --dir string          Directory to scan (default: current)")
			fmt.Println("  --format string       Output format: table, json, summary (default: table)")
			fmt.Println("  --output string       Output file path")
			fmt.Println("  --details             Show detailed information")
			fmt.Println("  --include-backups     Include backup files (.backup)")
			fmt.Println("  --no-recursive        Don't scan subdirectories")
			return
		}
	}

	fmt.Printf("Scanning for Terraform state files in: %s\n", dir)
	if recursive {
		fmt.Println("Scanning recursively...")
	}
	fmt.Println()

	// Discover state files
	stateFiles := discoverStateFiles(dir, recursive, includeBackups)

	if len(stateFiles) == 0 {
		fmt.Println("No Terraform state files found.")
		return
	}

	// Sort by path
	sort.Slice(stateFiles, func(i, j int) bool {
		return stateFiles[i].Path < stateFiles[j].Path
	})

	// Display or save results
	switch format {
	case "json":
		displayStateFilesJSON(stateFiles, output)
	case "summary":
		displayStateFilesSummary(stateFiles)
	default: // table
		displayStateFilesTable(stateFiles, showDetails)
	}
}

// handleTfStateShow shows details of a specific state file
func handleTfStateShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: State file path required\n")
		fmt.Println("Usage: driftmgr tfstate show <path> [flags]")
		os.Exit(1)
	}

	statePath := args[0]
	var format string
	var showResources bool
	format = "detail"

	// Parse additional arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--resources":
			showResources = true
		case "--help", "-h":
			fmt.Println("Usage: driftmgr tfstate show <path> [flags]")
			fmt.Println()
			fmt.Println("Show details of a Terraform state file")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --format string    Output format: detail, json (default: detail)")
			fmt.Println("  --resources        Show all resources in the state")
			return
		}
	}

	// Check if file exists
	info, err := os.Stat(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot access state file: %v\n", err)
		os.Exit(1)
	}

	// Read and parse state file
	stateInfo := analyzeStateFile(statePath, info)

	if showResources {
		// Load full resource details
		stateInfo.Resources = getStateResources(statePath)
	}

	// Display results
	switch format {
	case "json":
		data, _ := json.MarshalIndent(stateInfo, "", "  ")
		fmt.Println(string(data))
	default: // detail
		displayStateFileDetail(stateInfo)
	}
}

// handleTfStateAnalyze analyzes state files for issues
func handleTfStateAnalyze(args []string) {
	var dir string
	var recursive, checkDrift, checkSize bool

	dir = "."
	recursive = true
	checkDrift = true

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			}
		case "--no-recursive":
			recursive = false
		case "--no-drift-check":
			checkDrift = false
		case "--check-size":
			checkSize = true
		case "--help", "-h":
			fmt.Println("Usage: driftmgr tfstate analyze [flags]")
			fmt.Println()
			fmt.Println("Analyze Terraform state files for issues")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --dir string       Directory to scan (default: current)")
			fmt.Println("  --no-recursive     Don't scan subdirectories")
			fmt.Println("  --no-drift-check   Skip drift detection")
			fmt.Println("  --check-size       Check for large state files")
			return
		}
	}

	fmt.Printf("Analyzing Terraform state files in: %s\n", dir)
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	// Discover state files
	stateFiles := discoverStateFiles(dir, recursive, false)

	if len(stateFiles) == 0 {
		fmt.Println("No Terraform state files found.")
		return
	}

	// Analyze each state file
	var issues []StateIssue
	for _, stateFile := range stateFiles {
		fileIssues := analyzeStateFileIssues(stateFile, checkSize, checkDrift)
		issues = append(issues, fileIssues...)
	}

	// Display analysis results
	displayAnalysisResults(stateFiles, issues)
}

// handleTfStateFind finds state files containing specific resources
func handleTfStateFind(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Search term required\n")
		fmt.Println("Usage: driftmgr tfstate find <resource> [flags]")
		os.Exit(1)
	}

	searchTerm := args[0]
	var dir string
	var recursive, exactMatch bool

	dir = "."
	recursive = true

	// Parse additional arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			}
		case "--exact":
			exactMatch = true
		case "--no-recursive":
			recursive = false
		case "--help", "-h":
			fmt.Println("Usage: driftmgr tfstate find <resource> [flags]")
			fmt.Println()
			fmt.Println("Find state files containing specific resources")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --dir string     Directory to scan (default: current)")
			fmt.Println("  --exact          Exact match only")
			fmt.Println("  --no-recursive   Don't scan subdirectories")
			return
		}
	}

	fmt.Printf("Searching for '%s' in Terraform state files...\n", searchTerm)
	fmt.Println()

	// Discover and search state files
	stateFiles := discoverStateFiles(dir, recursive, false)
	var matches []StateSearchResult

	for _, stateFile := range stateFiles {
		if resources := searchStateFile(stateFile.Path, searchTerm, exactMatch); len(resources) > 0 {
			matches = append(matches, StateSearchResult{
				StateFile: stateFile,
				Resources: resources,
			})
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No state files found containing '%s'\n", searchTerm)
		return
	}

	// Display search results
	displaySearchResults(matches, searchTerm)
}

// StateIssue represents an issue found in a state file
type StateIssue struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// StateSearchResult represents search results from a state file
type StateSearchResult struct {
	StateFile TerraformStateInfo
	Resources []string
}

// discoverStateFiles discovers all terraform state files in a directory
func discoverStateFiles(dir string, recursive, includeBackups bool) []TerraformStateInfo {
	var stateFiles []TerraformStateInfo

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories if not recursive
		if info.IsDir() && path != dir && !recursive {
			return filepath.SkipDir
		}

		// Check if it's a state file
		if isStateFile(path, includeBackups) {
			stateInfo := analyzeStateFile(path, info)
			stateFiles = append(stateFiles, stateInfo)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error scanning directory: %v\n", err)
	}

	// Also check for Terraform backend configurations
	scanner := terraform.NewBackendScanner(dir)
	configs, _ := scanner.ScanDirectory()

	for _, config := range configs {
		if config.IsStateFile {
			// Already included
			continue
		}

		// Add remote state references
		if config.Type != "" && config.Type != "local" {
			stateFiles = append(stateFiles, TerraformStateInfo{
				Path:      config.WorkingDir,
				Type:      string(config.Type),
				Backend:   string(config.Type),
				IsRemote:  true,
				Workspace: strings.Join(config.Workspaces, ","),
			})
		}
	}

	return stateFiles
}

// isStateFile checks if a file is a terraform state file
func isStateFile(path string, includeBackups bool) bool {
	base := filepath.Base(path)

	// Check for state files
	if base == "terraform.tfstate" || strings.HasSuffix(base, ".tfstate") {
		return true
	}

	// Check for backup files if requested
	if includeBackups {
		if base == "terraform.tfstate.backup" || strings.HasSuffix(base, ".tfstate.backup") {
			return true
		}
		if strings.Contains(base, ".tfstate.") && strings.Contains(base, ".backup") {
			return true
		}
	}

	return false
}

// analyzeStateFile analyzes a terraform state file
func analyzeStateFile(path string, info os.FileInfo) TerraformStateInfo {
	stateInfo := TerraformStateInfo{
		Path:         path,
		Type:         "local",
		Size:         info.Size(),
		ModifiedTime: info.ModTime(),
		IsBackup:     strings.Contains(path, ".backup"),
	}

	// Try to parse the state file
	// parser := state.NewParser() // Not implemented yet
	stateData, err := state.LoadStateFile(path)
	if err != nil {
		stateInfo.Error = err.Error()
		return stateInfo
	}

	// Extract state information
	if stateData != nil {
		stateInfo.Version = stateData.Version
		stateInfo.TerraformVersion = stateData.TerraformVersion

		// Count resources
		stateInfo.ResourceCount = len(stateData.Resources)

		// Determine primary provider
		providerCount := make(map[string]int)
		for _, resource := range stateData.Resources {
			// Extract provider name from full path
			parts := strings.Split(resource.Provider, "/")
			if len(parts) > 0 {
				providerName := parts[len(parts)-1]
				providerName = strings.Split(providerName, ".")[0]
				providerCount[providerName]++
			}
		}

		// Find dominant provider
		maxCount := 0
		for provider, count := range providerCount {
			if count > maxCount {
				maxCount = count
				stateInfo.Provider = provider
			}
		}
	}

	return stateInfo
}

// getStateResources extracts detailed resource information from a state file
func getStateResources(path string) []StateResourceSummary {
	var resources []StateResourceSummary

	// parser := state.NewParser() // Not implemented yet
	stateData, err := state.LoadStateFile(path)
	if err != nil {
		return resources
	}

	// Extract resources
	resourceTypes := make(map[string]*StateResourceSummary)

	for _, resource := range stateData.Resources {
		resType := resource.Type
		resName := resource.Name
		resMode := resource.Mode
		if resMode == "" {
			resMode = "managed"
		}
		resProvider := resource.Provider
		// Extract provider name from full path
		parts := strings.Split(resProvider, "/")
		if len(parts) > 0 {
			resProvider = strings.Split(parts[len(parts)-1], ".")[0]
			resProvider = strings.TrimSuffix(resProvider, "]")
			resProvider = strings.TrimSuffix(resProvider, "\"")
		}

		// Group by type
		key := fmt.Sprintf("%s:%s", resType, resMode)
		if summary, exists := resourceTypes[key]; exists {
			summary.Count++
		} else {
			resourceTypes[key] = &StateResourceSummary{
				Type:     resType,
				Name:     resName,
				Provider: resProvider,
				Mode:     resMode,
				Count:    1,
			}
		}
	}

	// Convert map to slice
	for _, summary := range resourceTypes {
		resources = append(resources, *summary)
	}

	// Sort by type
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Type < resources[j].Type
	})

	return resources
}

// analyzeStateFileIssues analyzes a state file for potential issues
func analyzeStateFileIssues(stateFile TerraformStateInfo, checkSize, checkDrift bool) []StateIssue {
	var issues []StateIssue

	// Check for large state files (>10MB)
	if checkSize && stateFile.Size > 10*1024*1024 {
		issues = append(issues, StateIssue{
			Path:        stateFile.Path,
			Type:        "size",
			Severity:    "warning",
			Description: fmt.Sprintf("Large state file: %.2f MB", float64(stateFile.Size)/(1024*1024)),
		})
	}

	// Check for old state files (>30 days)
	if time.Since(stateFile.ModifiedTime) > 30*24*time.Hour {
		issues = append(issues, StateIssue{
			Path:        stateFile.Path,
			Type:        "age",
			Severity:    "info",
			Description: fmt.Sprintf("State file not updated for %d days", int(time.Since(stateFile.ModifiedTime).Hours()/24)),
		})
	}

	// Check for backup files
	if stateFile.IsBackup {
		issues = append(issues, StateIssue{
			Path:        stateFile.Path,
			Type:        "backup",
			Severity:    "info",
			Description: "Backup state file found",
		})
	}

	// Check for errors
	if stateFile.Error != "" {
		issues = append(issues, StateIssue{
			Path:        stateFile.Path,
			Type:        "error",
			Severity:    "error",
			Description: fmt.Sprintf("Failed to parse: %s", stateFile.Error),
		})
	}

	// Check for empty state
	if stateFile.ResourceCount == 0 && stateFile.Error == "" {
		issues = append(issues, StateIssue{
			Path:        stateFile.Path,
			Type:        "empty",
			Severity:    "warning",
			Description: "State file contains no resources",
		})
	}

	// Check for very old Terraform versions
	if stateFile.TerraformVersion != "" {
		parts := strings.Split(stateFile.TerraformVersion, ".")
		if len(parts) > 0 {
			if parts[0] == "0" || (parts[0] == "1" && len(parts) > 1 && parts[1] == "0") {
				issues = append(issues, StateIssue{
					Path:        stateFile.Path,
					Type:        "version",
					Severity:    "warning",
					Description: fmt.Sprintf("Old Terraform version: %s", stateFile.TerraformVersion),
				})
			}
		}
	}

	return issues
}

// searchStateFile searches a state file for specific resources
func searchStateFile(path, searchTerm string, exactMatch bool) []string {
	var matches []string

	// parser := state.NewParser() // Not implemented yet
	stateData, err := state.LoadStateFile(path)
	if err != nil {
		return matches
	}

	// Search in resources
	for _, resource := range stateData.Resources {
		resType := resource.Type
		resName := resource.Name

		resourceID := fmt.Sprintf("%s.%s", resType, resName)

		if exactMatch {
			if resourceID == searchTerm || resType == searchTerm || resName == searchTerm {
				matches = append(matches, resourceID)
			}
		} else {
			if strings.Contains(resourceID, searchTerm) ||
				strings.Contains(resType, searchTerm) ||
				strings.Contains(resName, searchTerm) {
				matches = append(matches, resourceID)
			}
		}
	}

	return matches
}

// Display functions

func displayStateFilesTable(stateFiles []TerraformStateInfo, showDetails bool) {
	fmt.Printf("Found %d Terraform state file(s)\n", len(stateFiles))
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	for i, stateFile := range stateFiles {
		// Determine icon based on state
		icon := "✓"
		if stateFile.Error != "" {
			icon = "✗"
		} else if stateFile.IsBackup {
			icon = "↻"
		} else if stateFile.IsRemote {
			icon = "☁"
		}

		fmt.Printf("%s %s\n", icon, stateFile.Path)

		if showDetails {
			if stateFile.Type != "local" {
				fmt.Printf("  Type: %s\n", stateFile.Type)
			}
			if stateFile.Backend != "" {
				fmt.Printf("  Backend: %s\n", stateFile.Backend)
			}
			if !stateFile.IsRemote {
				fmt.Printf("  Size: %.2f KB\n", float64(stateFile.Size)/1024)
				fmt.Printf("  Modified: %s\n", stateFile.ModifiedTime.Format("2006-01-02 15:04:05"))
			}
			if stateFile.TerraformVersion != "" {
				fmt.Printf("  Terraform: %s\n", stateFile.TerraformVersion)
			}
			if stateFile.ResourceCount > 0 {
				fmt.Printf("  Resources: %d\n", stateFile.ResourceCount)
			}
			if stateFile.Provider != "" {
				fmt.Printf("  Provider: %s\n", stateFile.Provider)
			}
			if stateFile.Error != "" {
				fmt.Printf("  Error: %s\n", stateFile.Error)
			}
		}

		if i < len(stateFiles)-1 {
			fmt.Println()
		}
	}
}

func displayStateFilesSummary(stateFiles []TerraformStateInfo) {
	totalSize := int64(0)
	totalResources := 0
	providerCount := make(map[string]int)
	backupCount := 0
	remoteCount := 0
	errorCount := 0

	for _, stateFile := range stateFiles {
		totalSize += stateFile.Size
		totalResources += stateFile.ResourceCount
		if stateFile.Provider != "" {
			providerCount[stateFile.Provider]++
		}
		if stateFile.IsBackup {
			backupCount++
		}
		if stateFile.IsRemote {
			remoteCount++
		}
		if stateFile.Error != "" {
			errorCount++
		}
	}

	fmt.Println("Terraform State Files Summary")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("Total Files: %d\n", len(stateFiles))
	fmt.Printf("  Local: %d\n", len(stateFiles)-remoteCount)
	fmt.Printf("  Remote: %d\n", remoteCount)
	fmt.Printf("  Backups: %d\n", backupCount)
	if errorCount > 0 {
		fmt.Printf("  Errors: %d\n", errorCount)
	}
	fmt.Println()

	fmt.Printf("Total Size: %.2f MB\n", float64(totalSize)/(1024*1024))
	fmt.Printf("Total Resources: %d\n", totalResources)
	if len(stateFiles) > 0 {
		fmt.Printf("Average Resources per State: %d\n", totalResources/len(stateFiles))
	}
	fmt.Println()

	if len(providerCount) > 0 {
		fmt.Println("Providers:")
		for provider, count := range providerCount {
			fmt.Printf("  %s: %d state file(s)\n", provider, count)
		}
	}
}

func displayStateFilesJSON(stateFiles []TerraformStateInfo, output string) {
	data, err := json.MarshalIndent(stateFiles, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		if err := os.WriteFile(output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Results saved to %s\n", output)
	} else {
		fmt.Println(string(data))
	}
}

func displayStateFileDetail(stateInfo TerraformStateInfo) {
	fmt.Println("Terraform State File Details")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("Path: %s\n", stateInfo.Path)
	fmt.Printf("Type: %s\n", stateInfo.Type)

	if !stateInfo.IsRemote {
		fmt.Printf("Size: %.2f KB\n", float64(stateInfo.Size)/1024)
		fmt.Printf("Modified: %s\n", stateInfo.ModifiedTime.Format("2006-01-02 15:04:05"))
	}

	if stateInfo.Version > 0 {
		fmt.Printf("State Version: %d\n", stateInfo.Version)
	}
	if stateInfo.TerraformVersion != "" {
		fmt.Printf("Terraform Version: %s\n", stateInfo.TerraformVersion)
	}
	if stateInfo.Backend != "" {
		fmt.Printf("Backend: %s\n", stateInfo.Backend)
	}
	if stateInfo.Workspace != "" {
		fmt.Printf("Workspace: %s\n", stateInfo.Workspace)
	}

	fmt.Println()
	fmt.Printf("Resources: %d\n", stateInfo.ResourceCount)
	if stateInfo.Provider != "" {
		fmt.Printf("Primary Provider: %s\n", stateInfo.Provider)
	}

	if len(stateInfo.Resources) > 0 {
		fmt.Println()
		fmt.Println("Resource Types:")
		for _, resource := range stateInfo.Resources {
			fmt.Printf("  - %s (%s): %d\n", resource.Type, resource.Mode, resource.Count)
		}
	}

	if stateInfo.IsBackup {
		fmt.Println()
		fmt.Println("⚠ This is a backup file")
	}

	if stateInfo.Error != "" {
		fmt.Println()
		fmt.Printf("Error: %s\n", stateInfo.Error)
	}
}

func displayAnalysisResults(stateFiles []TerraformStateInfo, issues []StateIssue) {
	fmt.Printf("Analyzed %d state file(s)\n", len(stateFiles))
	fmt.Printf("Found %d issue(s)\n", len(issues))
	fmt.Println()

	if len(issues) == 0 {
		fmt.Println("[OK] No issues found!")
		return
	}

	// Group issues by severity
	bySeverity := make(map[string][]StateIssue)
	for _, issue := range issues {
		bySeverity[issue.Severity] = append(bySeverity[issue.Severity], issue)
	}

	// Display errors first
	if errors, ok := bySeverity["error"]; ok {
		fmt.Printf("🔴 Errors (%d)\n", len(errors))
		fmt.Println("-" + strings.Repeat("-", 70))
		for _, issue := range errors {
			fmt.Printf("  %s\n", issue.Path)
			fmt.Printf("    %s\n", issue.Description)
		}
		fmt.Println()
	}

	// Display warnings
	if warnings, ok := bySeverity["warning"]; ok {
		fmt.Printf("🟡 Warnings (%d)\n", len(warnings))
		fmt.Println("-" + strings.Repeat("-", 70))
		for _, issue := range warnings {
			fmt.Printf("  %s\n", issue.Path)
			fmt.Printf("    %s\n", issue.Description)
		}
		fmt.Println()
	}

	// Display info
	if infos, ok := bySeverity["info"]; ok {
		fmt.Printf("[INFO] Information (%d)\n", len(infos))
		fmt.Println("-" + strings.Repeat("-", 70))
		for _, issue := range infos {
			fmt.Printf("  %s\n", issue.Path)
			fmt.Printf("    %s\n", issue.Description)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("Summary")
	fmt.Println("-" + strings.Repeat("-", 70))
	if len(bySeverity["error"]) > 0 {
		fmt.Printf("[ERROR] %d state file(s) have errors\n", len(bySeverity["error"]))
	}
	if len(bySeverity["warning"]) > 0 {
		fmt.Printf("[WARNING]  %d state file(s) have warnings\n", len(bySeverity["warning"]))
	}
	fmt.Printf("[INFO]  %d informational message(s)\n", len(bySeverity["info"]))
}

func displaySearchResults(matches []StateSearchResult, searchTerm string) {
	fmt.Printf("Found '%s' in %d state file(s)\n", searchTerm, len(matches))
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	for _, match := range matches {
		fmt.Printf("📁 %s\n", match.StateFile.Path)
		fmt.Printf("   Resources: %d match(es)\n", len(match.Resources))
		for _, resource := range match.Resources {
			fmt.Printf("     - %s\n", resource)
		}
		fmt.Println()
	}

	// Summary
	totalMatches := 0
	for _, match := range matches {
		totalMatches += len(match.Resources)
	}

	fmt.Println("Summary")
	fmt.Println("-" + strings.Repeat("-", 70))
	fmt.Printf("Total Matches: %d resource(s) in %d file(s)\n", totalMatches, len(matches))
}

// showTfStateHelp displays help for tfstate command
func showTfStateHelp() {
	fmt.Println("Usage: driftmgr tfstate [subcommand] [flags]")
	fmt.Println()
	fmt.Println("Manage and analyze Terraform state files")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list       List all discovered state files")
	fmt.Println("  show       Show details of a specific state file")
	fmt.Println("  analyze    Analyze state files for issues")
	fmt.Println("  find       Find state files containing specific resources")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all state files in current directory")
	fmt.Println("  driftmgr tfstate list")
	fmt.Println()
	fmt.Println("  # Show details of a specific state file")
	fmt.Println("  driftmgr tfstate show terraform.tfstate --resources")
	fmt.Println()
	fmt.Println("  # Analyze state files for issues")
	fmt.Println("  driftmgr tfstate analyze --check-size")
	fmt.Println()
	fmt.Println("  # Find state files containing EC2 instances")
	fmt.Println("  driftmgr tfstate find aws_instance")
	fmt.Println()
	fmt.Println("Use 'driftmgr tfstate [subcommand] --help' for more information")
}
