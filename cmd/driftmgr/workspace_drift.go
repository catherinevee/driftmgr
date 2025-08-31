package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/fatih/color"
)

type WorkspaceDriftAnalysis struct {
	Workspaces      map[string]*WorkspaceInfo
	DriftMatrix     map[string]map[string]DriftComparison
	CommonDrifts    []CommonDrift
	WorkspaceHealth map[string]float64
	Timestamp       time.Time
}

type WorkspaceInfo struct {
	Name          string
	StatePath     string
	ResourceCount int
	LastModified  time.Time
	Environment   string
}

type DriftComparison struct {
	Source       string
	Target       string
	AddedCount   int
	RemovedCount int
	ModifiedCount int
	DriftPercent float64
	Details      []DriftDetail
}

type DriftDetail struct {
	ResourceType string
	ResourceName string
	DriftType    string // added, removed, modified
	Attributes   []string
}

type CommonDrift struct {
	ResourceType string
	ResourceName string
	Workspaces   []string
	DriftType    string
}

// handleWorkspaceDrift compares drift across Terraform workspaces
func handleWorkspaceDrift(args []string) {
	ctx := context.Background()
	
	var (
		workspaceDir = ".terraform/terraform.tfstate.d"
		baseline     = ""
		verbose      = false
		jsonOutput   = false
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 < len(args) {
				workspaceDir = args[i+1]
				i++
			}
		case "--baseline":
			if i+1 < len(args) {
				baseline = args[i+1]
				i++
			}
		case "--verbose", "-v":
			verbose = true
		case "--json":
			jsonOutput = true
		case "--help", "-h":
			showWorkspaceDriftHelp()
			return
		}
	}
	
	// Header
	if !jsonOutput {
		fmt.Println(color.CyanString("🔄 Workspace Drift Analysis"))
		fmt.Println(strings.Repeat("=", 51))
	}
	
	// Discover workspaces
	workspaces := discoverWorkspaces(workspaceDir)
	
	// Add default workspace if it exists
	if _, err := os.Stat("terraform.tfstate"); err == nil {
		workspaces["default"] = "terraform.tfstate"
	}
	
	if len(workspaces) == 0 {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No Terraform workspaces found"))
		os.Exit(1)
	}
	
	if !jsonOutput {
		fmt.Printf("Found %d workspaces\n\n", len(workspaces))
	}
	
	// Load workspace states
	analysis := WorkspaceDriftAnalysis{
		Workspaces:      make(map[string]*WorkspaceInfo),
		DriftMatrix:     make(map[string]map[string]DriftComparison),
		WorkspaceHealth: make(map[string]float64),
		Timestamp:       time.Now(),
	}
	
	states := make(map[string]*state.State)
	loader := state.NewStateLoader("")
	
	for name, path := range workspaces {
		if !jsonOutput {
			fmt.Printf("Loading workspace: %s\n", name)
		}
		
		stateFile, err := loader.LoadStateFile(ctx, path, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, color.YellowString("Warning: Failed to load %s: %v\n"), name, err)
			continue
		}
		
		info, err := os.Stat(path)
		var lastMod time.Time
		if err == nil {
			lastMod = info.ModTime()
		}
		
		states[name] = stateFile
		analysis.Workspaces[name] = &WorkspaceInfo{
			Name:          name,
			StatePath:     path,
			ResourceCount: len(stateFile.Resources),
			LastModified:  lastMod,
			Environment:   detectEnvironment(name),
		}
	}
	
	if len(states) < 2 {
		fmt.Fprintln(os.Stderr, color.YellowString("Warning: Need at least 2 workspaces for comparison"))
		os.Exit(1)
	}
	
	// Compare workspaces
	if !jsonOutput {
		fmt.Println("\nComparing workspaces...")
	}
	
	// If baseline specified, compare all against baseline
	if baseline != "" {
		if _, ok := states[baseline]; !ok {
			fmt.Fprintf(os.Stderr, color.RedString("Error: Baseline workspace '%s' not found\n"), baseline)
			os.Exit(1)
		}
		
		for name, stateFile := range states {
			if name == baseline {
				continue
			}
			comparison := compareWorkspaceStates(states[baseline], stateFile)
			if analysis.DriftMatrix[baseline] == nil {
				analysis.DriftMatrix[baseline] = make(map[string]DriftComparison)
			}
			analysis.DriftMatrix[baseline][name] = comparison
		}
	} else {
		// Compare all pairs
		names := []string{}
		for name := range states {
			names = append(names, name)
		}
		sort.Strings(names)
		
		for i, name1 := range names {
			for j := i + 1; j < len(names); j++ {
				name2 := names[j]
				comparison := compareWorkspaceStates(states[name1], states[name2])
				if analysis.DriftMatrix[name1] == nil {
					analysis.DriftMatrix[name1] = make(map[string]DriftComparison)
				}
				analysis.DriftMatrix[name1][name2] = comparison
			}
		}
	}
	
	// Find common drifts
	analysis.CommonDrifts = findCommonDrifts(states)
	
	// Calculate workspace health
	for name, stateFile := range states {
		analysis.WorkspaceHealth[name] = calculateWorkspaceHealth(stateFile)
	}
	
	// Display results
	if jsonOutput {
		displayWorkspaceDriftJSON(analysis)
	} else {
		displayWorkspaceDriftResults(analysis, verbose)
	}
}

func discoverWorkspaces(dir string) map[string]string {
	workspaces := make(map[string]string)
	
	// Check if workspace directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return workspaces
	}
	
	// Find all workspace state files
	pattern := filepath.Join(dir, "*", "terraform.tfstate")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return workspaces
	}
	
	for _, match := range matches {
		// Extract workspace name from path
		parts := strings.Split(filepath.ToSlash(match), "/")
		if len(parts) >= 2 {
			workspaceName := parts[len(parts)-2]
			workspaces[workspaceName] = match
		}
	}
	
	return workspaces
}

func compareWorkspaceStates(state1, state2 *state.State) DriftComparison {
	comparison := DriftComparison{
		Details: []DriftDetail{},
	}
	
	// Build resource maps
	resources1 := make(map[string]*state.Resource)
	resources2 := make(map[string]*state.Resource)
	
	for i := range state1.Resources {
		res := &state1.Resources[i]
		key := fmt.Sprintf("%s.%s", res.Type, res.Name)
		resources1[key] = res
	}
	
	for i := range state2.Resources {
		res := &state2.Resources[i]
		key := fmt.Sprintf("%s.%s", res.Type, res.Name)
		resources2[key] = res
	}
	
	// Find added resources (in state2 but not in state1)
	for key, res := range resources2 {
		if _, exists := resources1[key]; !exists {
			comparison.AddedCount++
			comparison.Details = append(comparison.Details, DriftDetail{
				ResourceType: res.Type,
				ResourceName: res.Name,
				DriftType:    "added",
			})
		}
	}
	
	// Find removed and modified resources
	for key, res1 := range resources1 {
		if res2, exists := resources2[key]; exists {
			// Compare resource attributes
			if !resourcesEqual(res1, res2) {
				comparison.ModifiedCount++
				comparison.Details = append(comparison.Details, DriftDetail{
					ResourceType: res1.Type,
					ResourceName: res1.Name,
					DriftType:    "modified",
					Attributes:   findDifferentAttributes(res1, res2),
				})
			}
		} else {
			// Resource removed
			comparison.RemovedCount++
			comparison.Details = append(comparison.Details, DriftDetail{
				ResourceType: res1.Type,
				ResourceName: res1.Name,
				DriftType:    "removed",
			})
		}
	}
	
	// Calculate drift percentage
	totalResources := len(resources1) + len(resources2)
	if totalResources > 0 {
		driftedResources := comparison.AddedCount + comparison.RemovedCount + comparison.ModifiedCount
		comparison.DriftPercent = float64(driftedResources) / float64(totalResources/2) * 100
	}
	
	return comparison
}

func resourcesEqual(res1, res2 *state.Resource) bool {
	// Simple comparison - in real implementation would deep compare attributes
	return res1.ID == res2.ID && res1.Type == res2.Type && res1.Name == res2.Name
}

func findDifferentAttributes(res1, res2 *state.Resource) []string {
	// Simplified - would normally deep compare attributes
	attrs := []string{}
	
	if res1.ID != res2.ID {
		attrs = append(attrs, "id")
	}
	
	// Would compare other attributes here
	if len(attrs) == 0 {
		attrs = append(attrs, "configuration")
	}
	
	return attrs
}

func findCommonDrifts(states map[string]*state.State) []CommonDrift {
	// Track resource presence across workspaces
	resourcePresence := make(map[string][]string)
	
	for workspace, stateFile := range states {
		for _, res := range stateFile.Resources {
			key := fmt.Sprintf("%s.%s", res.Type, res.Name)
			resourcePresence[key] = append(resourcePresence[key], workspace)
		}
	}
	
	// Find resources that aren't in all workspaces
	commonDrifts := []CommonDrift{}
	totalWorkspaces := len(states)
	
	for resource, workspaces := range resourcePresence {
		if len(workspaces) != totalWorkspaces {
			parts := strings.Split(resource, ".")
			driftType := "partial"
			if len(workspaces) == 1 {
				driftType = "unique"
			}
			
			commonDrifts = append(commonDrifts, CommonDrift{
				ResourceType: parts[0],
				ResourceName: parts[1],
				Workspaces:   workspaces,
				DriftType:    driftType,
			})
		}
	}
	
	return commonDrifts
}

func calculateWorkspaceHealth(stateFile *state.State) float64 {
	if len(stateFile.Resources) == 0 {
		return 0.0
	}
	
	// Simple health calculation
	health := 100.0
	
	// Penalize for missing tags
	untagged := 0
	for _, res := range stateFile.Resources {
		if res.Tags == nil {
			untagged++
		}
	}
	
	if untagged > 0 {
		health -= float64(untagged) / float64(len(stateFile.Resources)) * 20
	}
	
	// Penalize for old state
	if stateFile.Version < 4 {
		health -= 10
	}
	
	if health < 0 {
		health = 0
	}
	
	return health
}

func detectEnvironment(workspaceName string) string {
	lower := strings.ToLower(workspaceName)
	
	if strings.Contains(lower, "prod") {
		return "production"
	} else if strings.Contains(lower, "staging") || strings.Contains(lower, "stage") {
		return "staging"
	} else if strings.Contains(lower, "dev") {
		return "development"
	} else if strings.Contains(lower, "test") || strings.Contains(lower, "qa") {
		return "testing"
	}
	
	return workspaceName
}

func displayWorkspaceDriftResults(analysis WorkspaceDriftAnalysis, verbose bool) {
	// Workspace Summary
	fmt.Println(color.CyanString("📊 Workspace Summary"))
	fmt.Println(strings.Repeat("-", 51))
	
	// Sort workspaces by name
	names := []string{}
	for name := range analysis.Workspaces {
		names = append(names, name)
	}
	sort.Strings(names)
	
	for _, name := range names {
		ws := analysis.Workspaces[name]
		healthColor := color.GreenString
		if analysis.WorkspaceHealth[name] < 70 {
			healthColor = color.YellowString
		}
		if analysis.WorkspaceHealth[name] < 50 {
			healthColor = color.RedString
		}
		
		fmt.Printf("  %s:\n", color.CyanString(name))
		fmt.Printf("    Environment: %s\n", ws.Environment)
		fmt.Printf("    Resources:   %d\n", ws.ResourceCount)
		fmt.Printf("    Health:      %s\n", healthColor(fmt.Sprintf("%.0f%%", analysis.WorkspaceHealth[name])))
		fmt.Printf("    Modified:    %s\n", ws.LastModified.Format("2006-01-02 15:04"))
	}
	
	// Drift Matrix
	fmt.Println("\n" + color.CyanString("🔀 Drift Comparison"))
	fmt.Println(strings.Repeat("-", 51))
	
	for source, targets := range analysis.DriftMatrix {
		for target, comparison := range targets {
			driftColor := color.GreenString
			if comparison.DriftPercent > 10 {
				driftColor = color.YellowString
			}
			if comparison.DriftPercent > 30 {
				driftColor = color.RedString
			}
			
			fmt.Printf("\n%s → %s\n", source, target)
			fmt.Printf("  Drift:    %s\n", driftColor(fmt.Sprintf("%.1f%%", comparison.DriftPercent)))
			fmt.Printf("  Added:    %d resources\n", comparison.AddedCount)
			fmt.Printf("  Removed:  %d resources\n", comparison.RemovedCount)
			fmt.Printf("  Modified: %d resources\n", comparison.ModifiedCount)
			
			if verbose && len(comparison.Details) > 0 {
				fmt.Println("  Details:")
				for i, detail := range comparison.Details {
					if i >= 5 && !verbose {
						fmt.Printf("    ... and %d more\n", len(comparison.Details)-5)
						break
					}
					
					symbol := "?"
					var symbolColor func(string) string
					switch detail.DriftType {
					case "added":
						symbol = "+"
						symbolColor = func(s string) string { return color.GreenString(s) }
					case "removed":
						symbol = "-"
						symbolColor = func(s string) string { return color.RedString(s) }
					case "modified":
						symbol = "~"
						symbolColor = func(s string) string { return color.YellowString(s) }
					default:
						symbolColor = func(s string) string { return s }
					}
					
					fmt.Printf("    %s %s.%s", symbolColor(symbol), detail.ResourceType, detail.ResourceName)
					if len(detail.Attributes) > 0 {
						fmt.Printf(" (%s)", strings.Join(detail.Attributes, ", "))
					}
					fmt.Println()
				}
			}
		}
	}
	
	// Common Drifts
	if len(analysis.CommonDrifts) > 0 {
		fmt.Println("\n" + color.CyanString("⚠️  Notable Differences"))
		fmt.Println(strings.Repeat("-", 51))
		
		unique := []CommonDrift{}
		partial := []CommonDrift{}
		
		for _, drift := range analysis.CommonDrifts {
			if drift.DriftType == "unique" {
				unique = append(unique, drift)
			} else {
				partial = append(partial, drift)
			}
		}
		
		if len(unique) > 0 {
			fmt.Println("\nUnique to specific workspaces:")
			for _, drift := range unique {
				fmt.Printf("  • %s.%s only in: %s\n", 
					drift.ResourceType, 
					drift.ResourceName,
					strings.Join(drift.Workspaces, ", "))
			}
		}
		
		if len(partial) > 0 && verbose {
			fmt.Println("\nMissing from some workspaces:")
			for _, drift := range partial {
				fmt.Printf("  • %s.%s in: %s\n", 
					drift.ResourceType, 
					drift.ResourceName,
					strings.Join(drift.Workspaces, ", "))
			}
		}
	}
	
	// Recommendations
	fmt.Println("\n" + color.CyanString("💡 Recommendations"))
	fmt.Println(strings.Repeat("-", 51))
	
	// Find highest drift
	maxDrift := 0.0
	var maxSource, maxTarget string
	for source, targets := range analysis.DriftMatrix {
		for target, comparison := range targets {
			if comparison.DriftPercent > maxDrift {
				maxDrift = comparison.DriftPercent
				maxSource = source
				maxTarget = target
			}
		}
	}
	
	if maxDrift > 30 {
		fmt.Printf("• %s High drift between %s and %s (%.1f%%)\n", 
			color.RedString("⚠"), maxSource, maxTarget, maxDrift)
		fmt.Println("  Consider syncing these workspaces or reviewing differences")
	}
	
	// Check workspace health
	unhealthyCount := 0
	for _, health := range analysis.WorkspaceHealth {
		if health < 70 {
			unhealthyCount++
		}
	}
	
	if unhealthyCount > 0 {
		fmt.Printf("• %d workspace(s) have health below 70%%\n", unhealthyCount)
		fmt.Println("  Run 'driftmgr state health' for detailed analysis")
	}
	
	if len(analysis.CommonDrifts) > 5 {
		fmt.Printf("• Found %d resources with workspace differences\n", len(analysis.CommonDrifts))
		fmt.Println("  Review if these differences are intentional")
	}
}

func displayWorkspaceDriftJSON(analysis WorkspaceDriftAnalysis) {
	// Simplified JSON output
	fmt.Println("{")
	fmt.Printf("  \"timestamp\": \"%s\",\n", analysis.Timestamp.Format(time.RFC3339))
	fmt.Printf("  \"workspace_count\": %d,\n", len(analysis.Workspaces))
	
	// Workspaces
	fmt.Println("  \"workspaces\": {")
	first := true
	for name, ws := range analysis.Workspaces {
		if !first {
			fmt.Println(",")
		}
		fmt.Printf("    \"%s\": {\n", name)
		fmt.Printf("      \"resources\": %d,\n", ws.ResourceCount)
		fmt.Printf("      \"health\": %.2f,\n", analysis.WorkspaceHealth[name])
		fmt.Printf("      \"environment\": \"%s\"\n", ws.Environment)
		fmt.Print("    }")
		first = false
	}
	fmt.Println("\n  },")
	
	// Drift summary
	fmt.Println("  \"drift_summary\": {")
	first = true
	for source, targets := range analysis.DriftMatrix {
		for target, comparison := range targets {
			if !first {
				fmt.Println(",")
			}
			fmt.Printf("    \"%s_to_%s\": %.2f", source, target, comparison.DriftPercent)
			first = false
		}
	}
	fmt.Println("\n  },")
	
	fmt.Printf("  \"common_drifts\": %d\n", len(analysis.CommonDrifts))
	fmt.Println("}")
}

func showWorkspaceDriftHelp() {
	fmt.Println("Usage: driftmgr workspace [flags]")
	fmt.Println()
	fmt.Println("Compare drift across Terraform workspaces")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --dir string       Directory containing workspace states (default: .terraform/terraform.tfstate.d)")
	fmt.Println("  --baseline string  Compare all workspaces against this baseline")
	fmt.Println("  --verbose, -v      Show detailed drift information")
	fmt.Println("  --json             Output in JSON format")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Compare all workspaces")
	fmt.Println("  driftmgr workspace")
	fmt.Println()
	fmt.Println("  # Compare against production baseline")
	fmt.Println("  driftmgr workspace --baseline production")
	fmt.Println()
	fmt.Println("  # Detailed comparison with custom directory")
	fmt.Println("  driftmgr workspace --dir ./states --verbose")
}