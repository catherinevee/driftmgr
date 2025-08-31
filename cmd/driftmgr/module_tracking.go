package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/fatih/color"
)

type ModuleAnalysis struct {
	Modules         map[string]*ModuleInfo
	VersionMismatch []VersionMismatch
	Outdated        []OutdatedModule
	Security        []SecurityIssue
	Recommendations []string
	Timestamp       time.Time
}

type ModuleInfo struct {
	Name           string
	Source         string
	Version        string
	LatestVersion  string
	UsedIn         []string
	ResourceCount  int
	LastUpdated    time.Time
	Dependencies   []string
	SecurityIssues int
}

type VersionMismatch struct {
	Module           string
	VersionsInUse    map[string][]string // version -> locations using it
	RecommendVersion string
}

type OutdatedModule struct {
	Module        string
	CurrentVersion string
	LatestVersion  string
	VersionsBehind int
	Locations      []string
}

type SecurityIssue struct {
	Module      string
	Version     string
	Severity    string
	Description string
	FixVersion  string
}

// handleModuleTracking tracks and analyzes Terraform module versions
func handleModuleTracking(args []string) {
	ctx := context.Background()
	
	var (
		rootDir      = "."
		checkUpdates = true
		jsonOutput   = false
		verbose      = false
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 < len(args) {
				rootDir = args[i+1]
				i++
			}
		case "--no-updates":
			checkUpdates = false
		case "--json":
			jsonOutput = true
		case "--verbose", "-v":
			verbose = true
		case "--help", "-h":
			showModuleTrackingHelp()
			return
		}
	}
	
	// Header
	if !jsonOutput {
		fmt.Println(color.CyanString("📦 Module Version Tracking"))
		fmt.Println(strings.Repeat("=", 51))
	}
	
	// Find all Terraform files
	tfFiles := findTerraformFiles(rootDir)
	if len(tfFiles) == 0 {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No Terraform files found"))
		os.Exit(1)
	}
	
	if !jsonOutput {
		fmt.Printf("Found %d Terraform files\n\n", len(tfFiles))
	}
	
	// Parse modules from files
	analysis := ModuleAnalysis{
		Modules:   make(map[string]*ModuleInfo),
		Timestamp: time.Now(),
	}
	
	for _, file := range tfFiles {
		modules := parseModulesFromFile(file)
		for _, mod := range modules {
			moduleKey := fmt.Sprintf("%s@%s", mod.Source, mod.Version)
			if existing, ok := analysis.Modules[moduleKey]; ok {
				existing.UsedIn = append(existing.UsedIn, file)
			} else {
				mod.UsedIn = []string{file}
				analysis.Modules[moduleKey] = mod
			}
		}
	}
	
	// Check for version mismatches
	moduleVersions := make(map[string]map[string][]string)
	for _, mod := range analysis.Modules {
		source := mod.Source
		version := mod.Version
		if version == "" {
			version = "unspecified"
		}
		
		if moduleVersions[source] == nil {
			moduleVersions[source] = make(map[string][]string)
		}
		moduleVersions[source][version] = append(moduleVersions[source][version], mod.UsedIn...)
	}
	
	for source, versions := range moduleVersions {
		if len(versions) > 1 {
			mismatch := VersionMismatch{
				Module:        source,
				VersionsInUse: versions,
			}
			
			// Find recommended version (most recent or most used)
			var maxVersion string
			var maxUsage int
			for ver, locations := range versions {
				if len(locations) > maxUsage {
					maxUsage = len(locations)
					maxVersion = ver
				}
			}
			mismatch.RecommendVersion = maxVersion
			
			analysis.VersionMismatch = append(analysis.VersionMismatch, mismatch)
		}
	}
	
	// Check for updates if enabled
	if checkUpdates {
		if !jsonOutput {
			fmt.Println("Checking for module updates...")
		}
		
		for _, mod := range analysis.Modules {
			if strings.HasPrefix(mod.Source, "terraform-aws-modules/") ||
			   strings.HasPrefix(mod.Source, "Azure/") ||
			   strings.HasPrefix(mod.Source, "terraform-google-modules/") {
				latest := checkLatestVersion(mod.Source)
				if latest != "" && latest != mod.Version {
					mod.LatestVersion = latest
					
					outdated := OutdatedModule{
						Module:         mod.Source,
						CurrentVersion: mod.Version,
						LatestVersion:  latest,
						Locations:      mod.UsedIn,
					}
					
					if mod.Version != "" {
						outdated.VersionsBehind = calculateVersionDiff(mod.Version, latest)
					}
					
					analysis.Outdated = append(analysis.Outdated, outdated)
				}
			}
		}
	}
	
	// Load state to get resource counts
	statePath := findTerraformStateFile()
	if statePath != "" {
		loader := state.NewStateLoader(statePath)
		if stateFile, err := loader.LoadStateFile(ctx, statePath, nil); err == nil {
			// Map resources to modules
			for _, res := range stateFile.Resources {
				if res.Module != "" {
					for _, mod := range analysis.Modules {
						if strings.Contains(res.Module, mod.Name) {
							mod.ResourceCount++
						}
					}
				}
			}
		}
	}
	
	// Generate recommendations
	analysis.Recommendations = generateModuleRecommendations(analysis)
	
	// Display results
	if jsonOutput {
		displayModuleAnalysisJSON(analysis)
	} else {
		displayModuleAnalysisResults(analysis, verbose)
	}
}

func findTerraformFiles(rootDir string) []string {
	var files []string
	
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip hidden directories and terraform cache
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || info.Name() == "terraform.tfstate.d") {
			return filepath.SkipDir
		}
		
		// Find .tf files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tf") {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files
}

func parseModulesFromFile(filePath string) []*ModuleInfo {
	modules := []*ModuleInfo{}
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return modules
	}
	
	// Use simple parsing for now
	// Full HCL parsing would require proper schema definition
	return parseModulesSimple(string(content))
}

func parseModulesSimple(content string) []*ModuleInfo {
	modules := []*ModuleInfo{}
	lines := strings.Split(content, "\n")
	
	var currentModule *ModuleInfo
	inModule := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check for module block start
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := strings.Trim(parts[1], "\"")
				currentModule = &ModuleInfo{Name: name}
				inModule = true
			}
		} else if inModule {
			// Look for source
			if strings.Contains(line, "source") && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					source := strings.TrimSpace(parts[1])
					source = strings.Trim(source, "\"")
					currentModule.Source = source
				}
			}
			// Look for version
			if strings.Contains(line, "version") && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					version := strings.TrimSpace(parts[1])
					version = strings.Trim(version, "\"")
					currentModule.Version = version
				}
			}
			// Check for block end
			if line == "}" {
				if currentModule != nil && currentModule.Source != "" {
					modules = append(modules, currentModule)
				}
				currentModule = nil
				inModule = false
			}
		}
	}
	
	return modules
}

func checkLatestVersion(source string) string {
	// Check Terraform Registry API
	if strings.Contains(source, "/") {
		parts := strings.Split(source, "/")
		if len(parts) >= 2 {
			namespace := parts[0]
			name := parts[1]
			
			// Simple version check against registry
			url := fmt.Sprintf("https://registry.terraform.io/v1/modules/%s/%s", namespace, name)
			
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(url)
			if err != nil || resp.StatusCode != 200 {
				return ""
			}
			defer resp.Body.Close()
			
			var data map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return ""
			}
			
			if version, ok := data["version"].(string); ok {
				return version
			}
		}
	}
	
	return ""
}

func calculateVersionDiff(current, latest string) int {
	// Simple version difference calculation
	currentParts := strings.Split(strings.TrimPrefix(current, "v"), ".")
	latestParts := strings.Split(strings.TrimPrefix(latest, "v"), ".")
	
	diff := 0
	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		var c, l int
		fmt.Sscanf(currentParts[i], "%d", &c)
		fmt.Sscanf(latestParts[i], "%d", &l)
		if l > c {
			diff = l - c
			break
		}
	}
	
	return diff
}

func generateModuleRecommendations(analysis ModuleAnalysis) []string {
	recommendations := []string{}
	
	// Check for version mismatches
	if len(analysis.VersionMismatch) > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Standardize module versions - %d modules have version mismatches", 
				len(analysis.VersionMismatch)))
	}
	
	// Check for outdated modules
	criticalUpdates := 0
	for _, outdated := range analysis.Outdated {
		if outdated.VersionsBehind > 2 {
			criticalUpdates++
		}
	}
	
	if criticalUpdates > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Update %d critically outdated modules (>2 versions behind)", criticalUpdates))
	}
	
	// Check for modules without versions
	unversioned := 0
	for _, mod := range analysis.Modules {
		if mod.Version == "" || mod.Version == "unspecified" {
			unversioned++
		}
	}
	
	if unversioned > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Pin versions for %d modules currently without version constraints", unversioned))
	}
	
	// Check for local modules that could be published
	localModules := 0
	for _, mod := range analysis.Modules {
		if strings.HasPrefix(mod.Source, "./") || strings.HasPrefix(mod.Source, "../") {
			localModules++
		}
	}
	
	if localModules > 3 {
		recommendations = append(recommendations,
			"Consider publishing frequently-used local modules to a registry")
	}
	
	return recommendations
}

func displayModuleAnalysisResults(analysis ModuleAnalysis, verbose bool) {
	// Module Summary
	fmt.Println(color.CyanString("📊 Module Summary"))
	fmt.Println(strings.Repeat("-", 31))
	fmt.Printf("Total Modules: %d\n", len(analysis.Modules))
	
	// Group by source
	sourceCount := make(map[string]int)
	for _, mod := range analysis.Modules {
		base := strings.Split(mod.Source, "/")[0]
		sourceCount[base]++
	}
	
	// Show top sources
	sources := []string{}
	for source := range sourceCount {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool {
		return sourceCount[sources[i]] > sourceCount[sources[j]]
	})
	
	fmt.Println("\nModule Sources:")
	for i, source := range sources {
		if i >= 5 && !verbose {
			fmt.Printf("  ... and %d more\n", len(sources)-5)
			break
		}
		fmt.Printf("  • %s: %d modules\n", source, sourceCount[source])
	}
	
	// Version Mismatches
	if len(analysis.VersionMismatch) > 0 {
		fmt.Println("\n" + color.YellowString("⚠️  Version Mismatches"))
		fmt.Println(strings.Repeat("-", 31))
		
		for _, mismatch := range analysis.VersionMismatch {
			fmt.Printf("\n%s:\n", mismatch.Module)
			for version, locations := range mismatch.VersionsInUse {
				fmt.Printf("  • %s used in %d location(s)\n", version, len(locations))
				if verbose {
					for _, loc := range locations {
						fmt.Printf("    - %s\n", loc)
					}
				}
			}
			fmt.Printf("  Recommended: %s\n", color.GreenString(mismatch.RecommendVersion))
		}
	}
	
	// Outdated Modules
	if len(analysis.Outdated) > 0 {
		fmt.Println("\n" + color.YellowString("🔄 Outdated Modules"))
		fmt.Println(strings.Repeat("-", 31))
		
		// Sort by versions behind
		sort.Slice(analysis.Outdated, func(i, j int) bool {
			return analysis.Outdated[i].VersionsBehind > analysis.Outdated[j].VersionsBehind
		})
		
		for _, outdated := range analysis.Outdated {
			severity := color.YellowString
			if outdated.VersionsBehind > 2 {
				severity = color.RedString
			}
			
			fmt.Printf("\n%s:\n", outdated.Module)
			fmt.Printf("  Current: %s\n", outdated.CurrentVersion)
			fmt.Printf("  Latest:  %s\n", severity(outdated.LatestVersion))
			if outdated.VersionsBehind > 0 {
				fmt.Printf("  Behind:  %d version(s)\n", outdated.VersionsBehind)
			}
			if verbose {
				fmt.Printf("  Used in:\n")
				for _, loc := range outdated.Locations {
					fmt.Printf("    - %s\n", loc)
				}
			}
		}
	}
	
	// Module Details
	if verbose {
		fmt.Println("\n" + color.CyanString("📦 Module Details"))
		fmt.Println(strings.Repeat("-", 31))
		
		for key, mod := range analysis.Modules {
			fmt.Printf("\n%s:\n", key)
			fmt.Printf("  Source:    %s\n", mod.Source)
			fmt.Printf("  Version:   %s\n", mod.Version)
			if mod.ResourceCount > 0 {
				fmt.Printf("  Resources: %d\n", mod.ResourceCount)
			}
			fmt.Printf("  Used in:   %d file(s)\n", len(mod.UsedIn))
		}
	}
	
	// Recommendations
	if len(analysis.Recommendations) > 0 {
		fmt.Println("\n" + color.CyanString("💡 Recommendations"))
		fmt.Println(strings.Repeat("-", 31))
		
		for _, rec := range analysis.Recommendations {
			fmt.Printf("• %s\n", rec)
		}
	}
}

func displayModuleAnalysisJSON(analysis ModuleAnalysis) {
	output := map[string]interface{}{
		"timestamp":        analysis.Timestamp.Format(time.RFC3339),
		"module_count":     len(analysis.Modules),
		"version_mismatches": len(analysis.VersionMismatch),
		"outdated_count":   len(analysis.Outdated),
		"modules":          analysis.Modules,
		"mismatches":       analysis.VersionMismatch,
		"outdated":         analysis.Outdated,
		"recommendations":  analysis.Recommendations,
	}
	
	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func showModuleTrackingHelp() {
	fmt.Println("Usage: driftmgr module [flags]")
	fmt.Println()
	fmt.Println("Track and analyze Terraform module versions")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --dir string    Root directory to scan (default: current directory)")
	fmt.Println("  --no-updates    Skip checking for module updates")
	fmt.Println("  --verbose, -v   Show detailed module information")
	fmt.Println("  --json          Output in JSON format")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Analyze modules in current directory")
	fmt.Println("  driftmgr module")
	fmt.Println()
	fmt.Println("  # Check specific directory without update checks")
	fmt.Println("  driftmgr module --dir ./infrastructure --no-updates")
	fmt.Println()
	fmt.Println("  # Detailed analysis with all information")
	fmt.Println("  driftmgr module --verbose")
}