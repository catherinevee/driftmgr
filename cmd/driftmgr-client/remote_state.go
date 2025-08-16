package main

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/state"
)

// handleRemoteState processes remote state commands
func (shell *InteractiveShell) handleRemoteState(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remote-state <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  list <backend> [config]  - List remote state files")
		fmt.Println("  parse <backend> <config> - Parse remote state file")
		fmt.Println("  detect <path>            - Detect remote state configuration")
		fmt.Println("  optimize <statefile_id>  - Optimize state file")
		fmt.Println("  analyze <statefile_id>   - Analyze state optimization")
		return
	}

	command := args[0]

	switch command {
	case "list":
		shell.handleRemoteStateList(args[1:])
	case "parse":
		shell.handleRemoteStateParse(args[1:])
	case "detect":
		shell.handleRemoteStateDetect(args[1:])
	case "optimize":
		shell.handleRemoteStateOptimize(args[1:])
	case "analyze":
		shell.handleRemoteStateAnalyze(args[1:])
	default:
		fmt.Printf("Unknown remote-state command: %s\n", command)
	}
}

// handleRemoteStateList handles listing remote state files
func (shell *InteractiveShell) handleRemoteStateList(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remote-state list <backend> [config]")
		fmt.Println("Backends: s3, azurerm, gcs")
		return
	}

	backend := args[0]
	var config *state.RemoteStateConfig

	if len(args) > 1 {
		// Parse config from command line arguments
		config = shell.parseRemoteStateConfig(args[1:])
	} else {
		// Use default config
		config = &state.RemoteStateConfig{
			Backend: backend,
		}
	}

	// Create remote state manager
	rsm, err := state.NewRemoteStateManager()
	if err != nil {
		fmt.Printf("Error creating remote state manager: %v\n", err)
		return
	}

	// List remote state files
	stateFiles, err := rsm.ListRemoteStates(config)
	if err != nil {
		fmt.Printf("Error listing remote state files: %v\n", err)
		return
	}

	fmt.Printf("Remote state files in %s backend:\n", backend)
	for i, stateFile := range stateFiles {
		fmt.Printf("  %d. %s\n", i+1, stateFile)
	}
}

// handleRemoteStateParse handles parsing remote state files
func (shell *InteractiveShell) handleRemoteStateParse(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: remote-state parse <backend> <config>")
		return
	}

	_ = args[0] // backend variable unused for now
	config := shell.parseRemoteStateConfig(args[1:])

	// Create remote state manager
	rsm, err := state.NewRemoteStateManager()
	if err != nil {
		fmt.Printf("Error creating remote state manager: %v\n", err)
		return
	}

	// Parse remote state
	stateFile, err := rsm.ParseRemoteState(config)
	if err != nil {
		fmt.Printf("Error parsing remote state: %v\n", err)
		return
	}

	fmt.Printf("Successfully parsed remote state file: %s\n", stateFile.Path)
	fmt.Printf("Resources found: %d\n", len(stateFile.Resources))
}

// handleRemoteStateDetect handles detecting remote state configuration
func (shell *InteractiveShell) handleRemoteStateDetect(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remote-state detect <terraform_path>")
		return
	}

	terraformPath := args[0]

	// Create remote state manager
	rsm, err := state.NewRemoteStateManager()
	if err != nil {
		fmt.Printf("Error creating remote state manager: %v\n", err)
		return
	}

	// Detect remote state configuration
	configs, err := rsm.DetectRemoteStateConfig(terraformPath)
	if err != nil {
		fmt.Printf("Error detecting remote state configuration: %v\n", err)
		return
	}

	fmt.Printf("Remote state configurations found: %d\n", len(configs))
	for i, config := range configs {
		fmt.Printf("  %d. Backend: %s, Key: %s\n", i+1, config.Backend, config.Key)
	}
}

// handleRemoteStateOptimize handles state file optimization
func (shell *InteractiveShell) handleRemoteStateOptimize(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remote-state optimize <statefile_id> [options]")
		fmt.Println("Options:")
		fmt.Println("  --remove-unused     - Remove unused resources")
		fmt.Println("  --remove-empty      - Remove empty modules")
		fmt.Println("  --remove-orphaned   - Remove orphaned data sources")
		fmt.Println("  --compact           - Compact attributes")
		fmt.Println("  --dry-run           - Show what would be optimized")
		return
	}

	stateFileID := args[0]

	// Parse options
	options := &state.StateOptimizationOptions{
		RemoveUnusedResources: true,
		RemoveEmptyModules:    true,
		RemoveOrphanedData:    true,
		CompactAttributes:     true,
		DryRun:                false,
	}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--remove-unused":
			options.RemoveUnusedResources = true
		case "--remove-empty":
			options.RemoveEmptyModules = true
		case "--remove-orphaned":
			options.RemoveOrphanedData = true
		case "--compact":
			options.CompactAttributes = true
		case "--dry-run":
			options.DryRun = true
		}
	}

	// Get state file
	stateFile, err := shell.client.GetStateFile(stateFileID)
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	// Create state optimizer
	optimizer := state.NewStateOptimizer(options)

	// Optimize state
	_, result, err := optimizer.OptimizeState(stateFile)
	if err != nil {
		fmt.Printf("Error optimizing state: %v\n", err)
		return
	}

	// Display results
	fmt.Printf("State optimization completed:\n")
	fmt.Printf("  Original resources: %d\n", result.OriginalResourceCount)
	fmt.Printf("  Optimized resources: %d\n", result.OptimizedResourceCount)
	fmt.Printf("  Removed resources: %d\n", len(result.RemovedResources))
	fmt.Printf("  Removed modules: %d\n", len(result.RemovedModules))
	fmt.Printf("  Removed data sources: %d\n", len(result.RemovedDataSources))
	fmt.Printf("  Optimization time: %v\n", result.OptimizationTime)

	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings:\n")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, error := range result.Errors {
			fmt.Printf("  - %s\n", error)
		}
	}

	if !options.DryRun {
		fmt.Printf("State file optimized successfully\n")
	} else {
		fmt.Printf("Dry run completed - no changes made\n")
	}
}

// handleRemoteStateAnalyze handles state optimization analysis
func (shell *InteractiveShell) handleRemoteStateAnalyze(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: remote-state analyze <statefile_id>")
		return
	}

	stateFileID := args[0]

	// Get state file
	stateFile, err := shell.client.GetStateFile(stateFileID)
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	// Create state optimizer
	optimizer := state.NewStateOptimizer(nil)

	// Analyze optimization
	result, err := optimizer.AnalyzeStateOptimization(stateFile)
	if err != nil {
		fmt.Printf("Error analyzing state optimization: %v\n", err)
		return
	}

	// Get recommendations
	recommendations := optimizer.GetOptimizationRecommendations(stateFile)

	// Display results
	fmt.Printf("State optimization analysis:\n")
	fmt.Printf("  Original resources: %d\n", result.OriginalResourceCount)
	fmt.Printf("  Would be optimized to: %d\n", result.OptimizedResourceCount)
	fmt.Printf("  Would remove resources: %d\n", len(result.RemovedResources))
	fmt.Printf("  Would remove modules: %d\n", len(result.RemovedModules))
	fmt.Printf("  Would remove data sources: %d\n", len(result.RemovedDataSources))

	if len(recommendations) > 0 {
		fmt.Printf("Recommendations:\n")
		for _, recommendation := range recommendations {
			fmt.Printf("  - %s\n", recommendation)
		}
	}
}

// parseRemoteStateConfig parses remote state configuration from command line arguments
func (shell *InteractiveShell) parseRemoteStateConfig(args []string) *state.RemoteStateConfig {
	config := &state.RemoteStateConfig{
		Config: make(map[string]string),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if i+1 < len(args) {
				value := args[i+1]
				config.Config[key] = value
				i++ // Skip next argument
			}
		} else if config.Key == "" {
			config.Key = arg
		} else if config.Bucket == "" {
			config.Bucket = arg
		} else if config.Region == "" {
			config.Region = arg
		}
	}

	return config
}
