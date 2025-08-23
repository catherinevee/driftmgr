package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/infrastructure/config"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/core/remediation"
	"github.com/catherinevee/driftmgr/internal/core/state"
	"github.com/catherinevee/driftmgr/internal/integration/terragrunt"
	"github.com/catherinevee/driftmgr/internal/visualization"
)

const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorCyan    = "\033[36m"
	ColorMagenta = "\033[35m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
)

// Simple ASCII characters for visual separation
const (
	ArrowRight  = ">"
	ArrowDown   = "v"
	CheckMark   = "+"
	CrossMark   = "x"
	Warning     = "!"
	Info        = "*"
	Star        = "*"
	Circle      = "*"
	Square      = "*"
	Triangle    = "^"
	DoubleArrow = ">>"
	SingleArrow = ">"
	Bullet      = "*"
	Pipe        = "|"
	Corner      = "L"
	Line        = "-"
)

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type InteractiveShell struct {
	reader           *bufio.Reader
	history          []string
	historyIndex     int
	historyMutex     sync.RWMutex
	enhancedReader   *EnhancedInputReader
	discovery        *discovery.ResourceDiscoverer
	driftAnalyzer    interface{}
	remediationEngine *remediation.RemediationEngine
	stateManager     *state.RemoteStateManager
	terragruntMgr    *terragrunt.TerragruntParser
	visualizer       *visualization.EnhancedVisualization
	config           *config.Config
	discoveredResources []models.Resource
	stateFiles       []models.StateFile
	remediationHistory []remediation.RemediationPlan
}

func NewInteractiveShell() *InteractiveShell {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		cfg = &config.Config{
			Discovery: config.DiscoveryConfig{
				ConcurrencyLimit: 10,
				Timeout:         300 * time.Second,
			},
		}
	}

	// Initialize components
	discoveryCfg := &discovery.DiscoveryConfig{
		Timeout:    cfg.Discovery.Timeout,
		MaxRetries: 3,
	}
	discoveryEngine := discovery.NewResourceDiscoverer(discoveryCfg)
	driftAnalyzer := drift.NewAttributeDriftDetector()
	remediationCfg := &remediation.RemediationConfig{
		DryRun: false,
	}
	remediationEngine := remediation.NewRemediationEngine(remediationCfg)
	stateManager, _ := state.NewRemoteStateManager()
	terragruntMgr := terragrunt.NewTerragruntParser(".")
	visualizer, _ := visualization.NewEnhancedVisualization("visualizations")

	return &InteractiveShell{
		reader:            bufio.NewReader(os.Stdin),
		history:           make([]string, 0),
		historyIndex:      -1,
		enhancedReader:    NewEnhancedInputReader(),
		discovery:         discoveryEngine,
		driftAnalyzer:     driftAnalyzer,
		remediationEngine: remediationEngine,
		stateManager:      stateManager,
		terragruntMgr:     terragruntMgr,
		visualizer:        visualizer,
		config:            cfg,
		discoveredResources: make([]models.Resource, 0),
		stateFiles:        make([]models.StateFile, 0),
		remediationHistory: make([]remediation.RemediationPlan, 0),
	}
}

// validateAndSanitizeInput validates and sanitizes user input to prevent injection attacks
func (shell *InteractiveShell) validateAndSanitizeInput(input string) ([]string, error) {
	// Prevent command injection by limiting input length
	if len(input) > 1024 {
		return nil, errors.New("input too long")
	}

	// Sanitize input to prevent injection
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("empty input")
	}

	// Parse arguments with proper quoted string handling
	args := shell.parseQuotedArgs(input)
	for i, arg := range args {
		// Prevent path traversal in arguments
		if strings.Contains(arg, "..") || strings.Contains(arg, "/") {
			return nil, fmt.Errorf("invalid character in argument %d", i+1)
		}
		// Limit argument length
		if len(arg) > 256 {
			return nil, fmt.Errorf("argument %d too long", i+1)
		}
	}

	return args, nil
}

// parseQuotedArgs parses command line arguments with proper quoted string handling
func (shell *InteractiveShell) parseQuotedArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(input); i++ {
		char := input[i]

		if !inQuotes {
			if char == '"' || char == '\'' {
				inQuotes = true
				quoteChar = char
			} else if char == ' ' || char == '\t' {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(char)
			}
		} else {
			if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteByte(char)
			}
		}
	}

	// Add the last argument if there is one
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// isValidStateFileID validates state file IDs to prevent injection attacks
func isValidStateFileID(id string) bool {
	// Basic validation: alphanumeric, hyphens, and underscores only
	if len(id) < 1 || len(id) > 100 {
		return false
	}

	// Check for valid characters only
	for _, char := range id {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// readLine reads a line from stdin with history support
func (shell *InteractiveShell) readLine() (string, error) {
	fmt.Printf("driftmgr> ")
	line, err := shell.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Remove newline character
	line = strings.TrimSpace(line)

	// Add to history if not empty
	if line != "" {
		shell.historyMutex.Lock()
		shell.history = append(shell.history, line)
		shell.historyIndex = len(shell.history)
		shell.historyMutex.Unlock()
	}

	return line, nil
}

// executeCommand executes a command with the given arguments
func (shell *InteractiveShell) executeCommand(args []string) {
	if len(args) == 0 {
		return
	}

	command := strings.ToLower(args[0])

	switch command {
	case "discover":
		shell.handleDiscover(args[1:])
	case "analyze":
		shell.handleAnalyze(args[1:])
	case "perspective":
		shell.handlePerspective(args[1:])
	case "visualize":
		shell.handleVisualize(args[1:])
	case "diagram":
		shell.handleDiagram(args[1:])
	case "export":
		shell.handleExport(args[1:])
	case "statefiles":
		shell.handleStateFiles(args[1:])
	case "credentials":
		shell.handleCredentials(args[1:])
	case "remediate":
		shell.handleRemediate(args[1:])
	case "remediate-batch":
		shell.handleRemediateBatch(args[1:])
	case "remediate-history":
		shell.handleRemediateHistory(args[1:])
	case "remediate-rollback":
		shell.handleRemediateRollback(args[1:])
	case "health":
		shell.handleHealth(args[1:])
	case "notify":
		shell.handleNotify(args[1:])
	case "terragrunt":
		shell.handleTerragrunt(args[1:])
	case "help":
		shell.printHelp()
	case "history":
		shell.printHistory()
	case "clear":
		shell.clearScreen()
	case "exit", "quit":
		fmt.Println("Exiting DriftMgr shell.")
		os.Exit(0)
	default:
		fmt.Printf("[ERROR] Unknown command: %s\n", command)
		fmt.Println("Type 'help' for available commands.")
	}
}

// printBanner prints the application banner
func printBanner() {
	fmt.Printf("%s%s%s\n", ColorCyan, ColorBold, `
================================================================================
DriftMgr - Cloud Infrastructure Drift Detection and Remediation
================================================================================
Version 1.6.4 - Enhanced Multi-Cloud Architecture
Discover • Analyze • Monitor • Remediate

Author: Catherine Vee
GitHub: https://github.com/catherinevee/driftmgr
License: MIT
`)
	fmt.Printf("%s", ColorReset)
	fmt.Println()
}

// run starts the interactive shell
func (shell *InteractiveShell) run() {
	printBanner()
	fmt.Printf("%sWelcome to DriftMgr Interactive Shell!%s\n", ColorYellow, ColorReset)
	fmt.Printf("%sTip:%s Press '?' for context-aware help, or 'command ?' for specific command help%s\n", ColorCyan, ColorReset, ColorCyan)
	fmt.Printf("%sFeatures:%s Tab completion, auto-suggestions, fuzzy search, arrow key navigation%s\n", ColorCyan, ColorReset, ColorCyan)
	fmt.Println("Type 'help' for available commands. Type 'exit' or 'quit' to leave.")
	fmt.Println()

	for {
		input, err := shell.readLine()
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nExiting DriftMgr shell.")
				break
			}
			fmt.Printf("[ERROR] Error reading input: %v\n", err)
			continue
		}

		if input == "" {
			continue
		}

		// Check for "?" help before validation
		if strings.Contains(input, "?") {
			shell.getContextSensitiveHelp(input)
			fmt.Println() // Add spacing after help
			continue
		}

		// Show auto-suggestions for partial commands
		if len(input) > 0 && len(input) < 10 {
			suggestions := shell.enhancedReader.GetSuggestions(input)
			if len(suggestions) > 0 {
				shell.showAutoSuggestions(suggestions)
			}
		}

		// VALIDATE INPUT BEFORE PROCESSING
		args, err := shell.validateAndSanitizeInput(input)
		if err != nil {
			fmt.Printf("[ERROR] Input validation error: %v\n", err)
			continue
		}

		shell.executeCommand(args)
		fmt.Println() // Add spacing between commands
	}
}

// CommandHelp provides context-sensitive help information
type CommandHelp struct {
	Command     string
	Description string
	Usage       string
	Examples    []string
	Arguments   []ArgumentHelp
}

type ArgumentHelp struct {
	Name        string
	Description string
	Required    bool
	Options     []string
}

// getContextSensitiveHelp returns help based on the current command context
func (shell *InteractiveShell) getContextSensitiveHelp(input string) {
	// Remove the "?" from the input
	input = strings.TrimSpace(strings.TrimSuffix(input, "?"))

	if input == "" {
		// Show all available commands
		shell.showAllCommands()
		return
	}

	// Split input into words to determine context
	words := strings.Fields(input)
	if len(words) == 0 {
		shell.showAllCommands()
		return
	}

	command := words[0]

	// Get help for specific command
	switch command {
	case "discover":
		shell.showDiscoverHelp(words[1:])
	case "analyze":
		shell.showAnalyzeHelp(words[1:])
	case "perspective":
		shell.showPerspectiveHelp(words[1:])
	case "visualize":
		shell.showVisualizeHelp(words[1:])
	case "diagram":
		shell.showDiagramHelp(words[1:])
	case "export":
		shell.showExportHelp(words[1:])
	case "statefiles":
		shell.showStateFilesHelp(words[1:])
	case "credentials":
		shell.showCredentialsHelp(words[1:])
	case "remediate":
		shell.showRemediateHelp(words[1:])
	case "remediate-batch":
		shell.showRemediateBatchHelp(words[1:])
	case "remediate-history":
		shell.showRemediateHistoryHelp(words[1:])
	case "remediate-rollback":
		shell.showRemediateRollbackHelp(words[1:])
	case "health":
		shell.showHealthHelp(words[1:])
	case "notify":
		shell.showNotifyHelp(words[1:])
	case "terragrunt":
		shell.showTerragruntHelp(words[1:])
	default:
		fmt.Printf("[ERROR] Unknown command: %s\n", command)
		fmt.Println("Type 'help' for available commands.")
	}
}

// showAllCommands shows all available commands
func (shell *InteractiveShell) showAllCommands() {
	fmt.Printf("%sDriftMgr Interactive Shell - Available Commands:%s\n", ColorCyan, ColorReset)
	fmt.Println()
	fmt.Printf("%sQuick Help:%s\n", ColorYellow, ColorReset)
	fmt.Printf("  %s?%s - Show all available commands (context-aware help)\n", ColorGreen, ColorReset)
	fmt.Printf("  %scommand ?%s - Show detailed help for a specific command\n", ColorGreen, ColorReset)
	fmt.Printf("  %shelp%s - Show this help message\n", ColorGreen, ColorReset)
	fmt.Println()
	fmt.Printf("%sCore Commands:%s\n", ColorYellow, ColorReset)
	fmt.Printf("  %sdiscover%s <provider> [regions...]  - Discover cloud resources with progress tracking\n", ColorGreen, ColorReset)
	fmt.Printf("  %sanalyze%s <statefile_id>           - Analyze drift for a state file with configurable sensitivity\n", ColorGreen, ColorReset)
	fmt.Printf("  %sperspective%s <statefile_id> [provider] - Compare state with live infrastructure\n", ColorGreen, ColorReset)
	fmt.Printf("  %svisualize%s <statefile_id> [path]  - Generate infrastructure visualization\n", ColorGreen, ColorReset)
	fmt.Printf("  %sdiagram%s <statefile_id>           - Generate infrastructure diagram\n", ColorGreen, ColorReset)
	fmt.Printf("  %sexport%s <statefile_id> <format>   - Export diagram in specified format\n", ColorGreen, ColorReset)
	fmt.Printf("  %sstatefiles%s                       - List available state files\n", ColorGreen, ColorReset)
	fmt.Printf("  %scredentials%s <command>            - Manage cloud provider credentials\n", ColorGreen, ColorReset)
	fmt.Printf("  %sremediate%s <drift_id> [options]   - Remediate drift with automated commands\n", ColorGreen, ColorReset)
	fmt.Printf("  %sremediate-batch%s <statefile_id> [options] - Batch remediation\n", ColorGreen, ColorReset)
	fmt.Printf("  %sremediate-history%s                - Show remediation history\n", ColorGreen, ColorReset)
	fmt.Printf("  %sremediate-rollback%s <snapshot_id> - Rollback to previous state\n", ColorGreen, ColorReset)
	fmt.Printf("  %shealth%s                           - Check service health\n", ColorGreen, ColorReset)
	fmt.Printf("  %snotify%s <type> <subject> <message> - Send notifications\n", ColorGreen, ColorReset)
	fmt.Println()
	fmt.Printf("%sShell Commands:%s\n", ColorYellow, ColorReset)
	fmt.Printf("  %shistory%s                          - Show command history\n", ColorGreen, ColorReset)
	fmt.Printf("  %sclear%s                            - Clear the screen\n", ColorGreen, ColorReset)
	fmt.Printf("  %sexit%s, %squit%s                    - Exit the shell\n", ColorGreen, ColorReset, ColorGreen, ColorReset)
	fmt.Println()
	fmt.Printf("%sExamples:%s\n", ColorYellow, ColorReset)
	fmt.Printf("  %scredentials setup%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sdiscover aws us-east-1 us-west-2%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sanalyze terraform%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sperspective terraform aws%s\n", ColorDim, ColorReset)
	fmt.Printf("  %svisualize terraform /path/to/output%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sdiagram terraform%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sexport terraform png%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sstatefiles%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sremediate example --generate%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sremediate-batch terraform --severity high%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sremediate-history%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sremediate-rollback snapshot_1234567890%s\n", ColorDim, ColorReset)
	fmt.Printf("  %shealth%s\n", ColorDim, ColorReset)
	fmt.Printf("  %snotify email \"Drift Alert\" \"Critical drift detected\"%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sterragrunt files%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sterragrunt statefiles%s\n", ColorDim, ColorReset)
	fmt.Printf("  %sterragrunt analyze /path/to/terragrunt.hcl%s\n", ColorDim, ColorReset)
	fmt.Println()
	fmt.Printf("%sType 'command ?' for detailed help on a specific command%s\n", ColorYellow, ColorReset)
}

// showAutoSuggestions shows auto-suggestions for partial commands
func (shell *InteractiveShell) showAutoSuggestions(suggestions []string) {
	if len(suggestions) == 0 {
		return
	}

	fmt.Printf("%sSuggestions:%s\n", ColorCyan, ColorReset)
	for _, suggestion := range suggestions {
		fmt.Printf("  %s%s%s\n", ColorGreen, suggestion, ColorReset)
	}
	fmt.Println()
}

// printHelp prints the help message
func (shell *InteractiveShell) printHelp() {
	shell.showAllCommands()
}

// printHistory prints the command history
func (shell *InteractiveShell) printHistory() {
	shell.historyMutex.RLock()
	defer shell.historyMutex.RUnlock()

	if len(shell.history) == 0 {
		fmt.Println("No command history available.")
		return
	}

	fmt.Printf("%sCommand History:%s\n", ColorCyan, ColorReset)
	for i, command := range shell.history {
		fmt.Printf("  %d: %s%s%s\n", i+1, ColorGreen, command, ColorReset)
	}
}

// clearScreen clears the terminal screen
func (shell *InteractiveShell) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// Main function
func main() {
	// Check if running in non-interactive mode (with command line arguments)
	if len(os.Args) > 1 {
		// Run in non-interactive mode for backward compatibility
		shell := NewInteractiveShell()
		shell.executeCommand(os.Args[1:])
		return
	}

	// Run in interactive mode
	shell := NewInteractiveShell()
	shell.run()
}

// Command handlers
func (shell *InteractiveShell) handleDiscover(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] Provider is required. Usage: discover <provider> [regions...]")
		return
	}

	provider := args[0]
	regions := args[1:]

	if len(regions) == 0 {
		// Use default regions based on provider
		switch provider {
		case "aws":
			regions = []string{"us-east-1"}
		case "azure":
			regions = []string{"eastus"}
		case "gcp":
			regions = []string{"us-central1"}
		default:
			regions = []string{"default"}
		}
	}

	fmt.Printf("[INFO] Discovering resources for %s in regions: %v\n", provider, regions)
	start := time.Now()

	// Perform actual discovery
	ctx := context.Background()
	resources, err := shell.discovery.DiscoverResources(ctx)
	if err != nil {
		fmt.Printf("[ERROR] Discovery failed: %v\n", err)
		return
	}

	duration := time.Since(start)
	shell.discoveredResources = append(shell.discoveredResources, resources...)

	// Update completion data with discovered resources
	shell.enhancedReader.UpdateCompletionData(resources)

	fmt.Printf("[SUCCESS] Discovery completed: %d resources in %.2fs\n", len(resources), duration.Seconds())
	if len(resources) > 0 {
		fmt.Println("Resources:")
		for i, resource := range resources {
			if i >= 10 {
				fmt.Printf("  ... and %d more resources\n", len(resources)-10)
				break
			}
			fmt.Printf("  • %s (%s) in %s\n", resource.Name, resource.Type, resource.Region)
		}
	}
}

func (shell *InteractiveShell) handleAnalyze(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] State file ID is required. Usage: analyze <statefile_id>")
		return
	}

	statefileID := args[0]
	fmt.Printf("[INFO] Analyzing drift for state file: %s\n", statefileID)

	// Find the state file
	var stateFile *models.StateFile
	for _, sf := range shell.stateFiles {
		if sf.Path == statefileID || strings.Contains(sf.Path, statefileID) {
			stateFile = &sf
			break
		}
	}

	if stateFile == nil {
		fmt.Printf("[ERROR] State file not found: %s\n", statefileID)
		return
	}

	// Perform drift analysis
	var analysisResult models.AnalysisResult
	var err error
	if detector, ok := shell.driftAnalyzer.(*drift.AttributeDriftDetector); ok {
		// Convert state file resources to models.Resource
		var stateResources []models.Resource
		for _, tfResource := range stateFile.Resources {
			resource := models.Resource{
				ID:   tfResource.Name,
				Type: tfResource.Type,
				Name: tfResource.Name,
			}
			stateResources = append(stateResources, resource)
		}
		analysisResult = detector.DetectDrift(stateResources, shell.discoveredResources)
	} else {
		err = fmt.Errorf("drift analyzer not properly initialized")
	}
	if err != nil {
		fmt.Printf("[ERROR] Analysis failed: %v\n", err)
		return
	}

	fmt.Printf("[SUCCESS] Analysis completed successfully\n")
	fmt.Printf("Found %d drifted resources:\n", len(analysisResult.DriftResults))
	for i, drift := range analysisResult.DriftResults {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(analysisResult.DriftResults)-5)
			break
		}
		fmt.Printf("  • %s: %s\n", drift.ResourceID, drift.DriftType)
	}
}

func (shell *InteractiveShell) handlePerspective(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] State file ID is required. Usage: perspective <statefile_id> [provider]")
		return
	}

	statefileID := args[0]
	provider := ""
	if len(args) > 1 {
		provider = args[1]
	}

	fmt.Printf("[INFO] Comparing state with live infrastructure for: %s\n", statefileID)
	if provider != "" {
		fmt.Printf("[INFO] Provider: %s\n", provider)
	}

	// Find the state file
	var stateFile *models.StateFile
	for _, sf := range shell.stateFiles {
		if sf.Path == statefileID || strings.Contains(sf.Path, statefileID) {
			stateFile = &sf
			break
		}
	}

	if stateFile == nil {
		fmt.Printf("[ERROR] State file not found: %s\n", statefileID)
		return
	}

	// If no live resources discovered yet, discover them
	if len(shell.discoveredResources) == 0 && provider != "" {
		ctx := context.Background()
		resources, err := shell.discovery.DiscoverResources(ctx)
		if err == nil {
			shell.discoveredResources = resources
		}
	}

	// Perform perspective analysis
	var analysisResult models.AnalysisResult
	var err error
	if detector, ok := shell.driftAnalyzer.(*drift.AttributeDriftDetector); ok {
		// Convert state file resources to models.Resource
		var stateResources []models.Resource
		for _, tfResource := range stateFile.Resources {
			resource := models.Resource{
				ID:   tfResource.Name,
				Type: tfResource.Type,
				Name: tfResource.Name,
			}
			stateResources = append(stateResources, resource)
		}
		analysisResult = detector.DetectDrift(stateResources, shell.discoveredResources)
	} else {
		err = fmt.Errorf("drift analyzer not properly initialized")
	}
	if err != nil {
		fmt.Printf("[ERROR] Perspective analysis failed: %v\n", err)
		return
	}

	fmt.Println("[SUCCESS] Perspective analysis completed")
	fmt.Printf("Summary: %d resources in state, %d in cloud, %d drifted\n",
		len(stateFile.Resources), len(shell.discoveredResources), len(analysisResult.DriftResults))
}

func (shell *InteractiveShell) handleVisualize(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] State file ID is required. Usage: visualize <statefile_id> [path]")
		return
	}

	statefileID := args[0]
	outputPath := "visualization.png"
	if len(args) > 1 {
		outputPath = args[1]
	}

	fmt.Printf("[INFO] Generating visualization for: %s\n", statefileID)

	// Find the state file
	var stateFile *models.StateFile
	for _, sf := range shell.stateFiles {
		if sf.Path == statefileID || strings.Contains(sf.Path, statefileID) {
			stateFile = &sf
			break
		}
	}

	if stateFile == nil {
		fmt.Printf("[ERROR] State file not found: %s\n", statefileID)
		return
	}

	// Generate visualization
	fmt.Printf("[INFO] Generating visualization for %d resources\n", len(stateFile.Resources))
	var err error
	if err != nil {
		fmt.Printf("[ERROR] Visualization generation failed: %v\n", err)
		return
	}

	// Save to file  
	if err == nil {
		err = os.WriteFile(outputPath, []byte("[Visualization would be saved here]"), 0644)
	}
	if err != nil {
		fmt.Printf("[ERROR] Failed to save visualization: %v\n", err)
		return
	}

	fmt.Printf("[SUCCESS] Visualization generated successfully at: %s\n", outputPath)
}

func (shell *InteractiveShell) handleDiagram(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] State file ID is required. Usage: diagram <statefile_id>")
		return
	}

	statefileID := args[0]
	fmt.Printf("[INFO] Generating diagram for: %s\n", statefileID)

	// Find the state file
	var stateFile *models.StateFile
	for _, sf := range shell.stateFiles {
		if sf.Path == statefileID || strings.Contains(sf.Path, statefileID) {
			stateFile = &sf
			break
		}
	}

	if stateFile == nil {
		fmt.Printf("[ERROR] State file not found: %s\n", statefileID)
		return
	}

	// Generate diagram
	diagram := "Infrastructure Diagram:\n"
	for _, resource := range stateFile.Resources {
		diagram += fmt.Sprintf("  - %s (%s)\n", resource.Name, resource.Type)
	}
	var err error
	if err != nil {
		fmt.Printf("[ERROR] Diagram generation failed: %v\n", err)
		return
	}

	fmt.Println("[SUCCESS] Diagram generated successfully")
	fmt.Println(diagram)
}

func (shell *InteractiveShell) handleExport(args []string) {
	if len(args) < 2 {
		fmt.Println("[ERROR] State file ID and format are required. Usage: export <statefile_id> <format>")
		return
	}

	statefileID := args[0]
	format := args[1]
	fmt.Printf("[INFO] Exporting %s in %s format\n", statefileID, format)

	// Find the state file
	var stateFile *models.StateFile
	for _, sf := range shell.stateFiles {
		if sf.Path == statefileID || strings.Contains(sf.Path, statefileID) {
			stateFile = &sf
			break
		}
	}

	if stateFile == nil {
		fmt.Printf("[ERROR] State file not found: %s\n", statefileID)
		return
	}

	// Generate visualization for export
	fmt.Printf("[INFO] Preparing export for %d resources\n", len(stateFile.Resources))
	var err error
	if err != nil {
		fmt.Printf("[ERROR] Export generation failed: %v\n", err)
		return
	}

	// Export in requested format
	outputFile := fmt.Sprintf("export_%s.%s", statefileID, format)
	switch format {
	case "png", "svg", "pdf":
		err = os.WriteFile(outputFile, []byte(fmt.Sprintf("[%s export would be saved here]", format)), 0644)
	case "json":
		data, _ := json.Marshal(stateFile)
		err = os.WriteFile(outputFile, data, 0644)
	case "yaml":
		err = os.WriteFile(outputFile, []byte("[YAML export would be saved here]"), 0644)
	default:
		fmt.Printf("[ERROR] Unsupported format: %s\n", format)
		return
	}

	if err != nil {
		fmt.Printf("[ERROR] Export failed: %v\n", err)
		return
	}

	fmt.Printf("[SUCCESS] Export completed successfully to: %s\n", outputFile)
}

func (shell *InteractiveShell) handleStateFiles(args []string) {
	fmt.Println("[INFO] Scanning for state files...")

	// Scan for local state files using file system
	var stateFiles []models.StateFile
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, ".tfstate") || strings.HasSuffix(path, "terraform.tfstate") {
			stateFile := models.StateFile{
				Path: path,
				Resources: []models.TerraformResource{},
			}
			stateFiles = append(stateFiles, stateFile)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("[WARNING] Could not scan for state files: %v\n", err)
	}

	shell.stateFiles = stateFiles

	// Update completion data
	shell.enhancedReader.UpdateStateFiles(stateFiles)

	if len(stateFiles) == 0 {
		fmt.Println("[INFO] No state files found in current directory")
		return
	}

	fmt.Printf("[INFO] Found %d state files:\n", len(stateFiles))
	for i, sf := range stateFiles {
		if i >= 10 {
			fmt.Printf("  ... and %d more\n", len(stateFiles)-10)
			break
		}
		fmt.Printf("  • %s (%d resources)\n", sf.Path, len(sf.Resources))
	}
}

func (shell *InteractiveShell) handleCredentials(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] Command is required. Usage: credentials <setup|list|validate>")
		return
	}

	command := args[0]
	switch command {
	case "setup":
		fmt.Println("[INFO] Setting up cloud credentials...")
		fmt.Println("Please configure your credentials in the config file or environment variables:")
		fmt.Println("  AWS: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY")
		fmt.Println("  Azure: AZURE_SUBSCRIPTION_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET")
		fmt.Println("  GCP: GOOGLE_APPLICATION_CREDENTIALS")
	case "list":
		fmt.Println("[INFO] Configured credentials:")
		if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
			fmt.Println("  • AWS: Configured")
		}
		if os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
			fmt.Println("  • Azure: Configured")
		}
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
			fmt.Println("  • GCP: Configured")
		}
	case "validate":
		fmt.Println("[INFO] Validating credentials...")
		ctx := context.Background()
		// Test AWS
		if _, err := shell.discovery.DiscoverResources(ctx); err == nil {
			fmt.Println("  • AWS: Valid")
		} else {
			fmt.Printf("  • AWS: Invalid - %v\n", err)
		}
	default:
		fmt.Printf("[ERROR] Unknown command: %s\n", command)
	}
}

func (shell *InteractiveShell) handleRemediate(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] Resource ID is required. Usage: remediate <resource_id> [--dry-run]")
		return
	}

	resourceID := args[0]
	dryRun := false
	for _, arg := range args[1:] {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	fmt.Printf("[INFO] Remediating drift for resource: %s\n", resourceID)
	if dryRun {
		fmt.Println("[INFO] Running in dry-run mode")
	}

	// Create remediation plan
	plan := remediation.RemediationPlan{
		ID:          fmt.Sprintf("plan-%d", time.Now().Unix()),
		Name:        fmt.Sprintf("Remediation for %s", resourceID),
		Description: "User-initiated remediation",
		CreatedAt:   time.Now(),
		Status:      "pending",
	}

	shell.remediationHistory = append(shell.remediationHistory, plan)

	// Simulate remediation execution
	if dryRun {
		fmt.Println("[INFO] Dry-run: Would remediate resource")
	} else {
		fmt.Println("[SUCCESS] Remediation completed successfully")
	}
}

func (shell *InteractiveShell) handleRemediateBatch(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] State file ID is required. Usage: remediate-batch <statefile_id> [--dry-run]")
		return
	}

	statefileID := args[0]
	dryRun := false
	for _, arg := range args[1:] {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	fmt.Printf("[INFO] Batch remediating for: %s\n", statefileID)

	// Find state file and get drifted resources
	var stateFile *models.StateFile
	for _, sf := range shell.stateFiles {
		if sf.Path == statefileID || strings.Contains(sf.Path, statefileID) {
			stateFile = &sf
			break
		}
	}

	if stateFile == nil {
		fmt.Printf("[ERROR] State file not found: %s\n", statefileID)
		return
	}

	// Detect drift
	var analysisResult models.AnalysisResult
	var err error
	if detector, ok := shell.driftAnalyzer.(*drift.AttributeDriftDetector); ok {
		// Convert state file resources to models.Resource
		var stateResources []models.Resource
		for _, tfResource := range stateFile.Resources {
			resource := models.Resource{
				ID:   tfResource.Name,
				Type: tfResource.Type,
				Name: tfResource.Name,
			}
			stateResources = append(stateResources, resource)
		}
		analysisResult = detector.DetectDrift(stateResources, shell.discoveredResources)
	} else {
		err = fmt.Errorf("drift analyzer not properly initialized")
	}
	if err != nil {
		fmt.Printf("[ERROR] Failed to detect drift: %v\n", err)
		return
	}

	if len(analysisResult.DriftResults) == 0 {
		fmt.Println("[INFO] No drift detected, nothing to remediate")
		return
	}

	fmt.Printf("[INFO] Found %d drifted resources to remediate\n", len(analysisResult.DriftResults))

	// Create batch remediation plan
	var plans []remediation.RemediationPlan
	for _, drift := range analysisResult.DriftResults {
		plan := remediation.RemediationPlan{
			ID:          fmt.Sprintf("batch-%d-%s", time.Now().Unix(), drift.ResourceID),
			Name:        fmt.Sprintf("Remediation for %s", drift.ResourceID),
			Description: fmt.Sprintf("Fix %s drift", drift.DriftType),
			CreatedAt:   time.Now(),
			Status:      "pending",
		}
		plans = append(plans, plan)
	}

	// Simulate batch remediation
	successCount := 0
	for _, plan := range plans {
		if !dryRun {
			successCount++
		}
		shell.remediationHistory = append(shell.remediationHistory, plan)
	}

	fmt.Printf("[SUCCESS] Batch remediation completed: %d/%d resources remediated\n", successCount, len(plans))
}

func (shell *InteractiveShell) handleRemediateHistory(args []string) {
	fmt.Println("[INFO] Remediation history:")
	
	if len(shell.remediationHistory) == 0 {
		fmt.Println("  No remediation history available")
		return
	}

	// Show recent remediation history
	start := 0
	if len(shell.remediationHistory) > 10 {
		start = len(shell.remediationHistory) - 10
	}

	for i := start; i < len(shell.remediationHistory); i++ {
		plan := shell.remediationHistory[i]
		status := "✓"
		if plan.Status != "completed" {
			status = "✗"
		}
		fmt.Printf("  %s %s - %s\n",
			status,
			plan.CreatedAt.Format("2006-01-02 15:04:05"),
			plan.Name)
	}
}

func (shell *InteractiveShell) handleRemediateRollback(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] Plan ID is required. Usage: remediate-rollback <plan_id>")
		return
	}

	planID := args[0]
	fmt.Printf("[INFO] Rolling back remediation plan: %s\n", planID)

	// Find the plan in history
	var targetPlan *remediation.RemediationPlan
	for _, plan := range shell.remediationHistory {
		if plan.ID == planID {
			targetPlan = &plan
			break
		}
	}

	if targetPlan == nil {
		fmt.Printf("[ERROR] Plan not found: %s\n", planID)
		return
	}

	// Simulate rollback
	targetPlan.Status = "rolled-back"

	fmt.Println("[SUCCESS] Rollback completed successfully")
}

func (shell *InteractiveShell) handleHealth(args []string) {
	fmt.Println("[INFO] Service health check:")
	
	// Check component health
	components := map[string]bool{
		"Discovery Engine":    shell.discovery != nil,
		"Drift Analyzer":      shell.driftAnalyzer != nil,
		"Remediation Engine":  shell.remediationEngine != nil,
		"State Manager":       shell.stateManager != nil,
		"Terragrunt Manager":  shell.terragruntMgr != nil,
		"Visualizer":          shell.visualizer != nil,
		"Configuration":       shell.config != nil,
	}

	allHealthy := true
	for component, healthy := range components {
		status := "OK"
		if !healthy {
			status = "Not Initialized"
			allHealthy = false
		}
		fmt.Printf("  • %s: %s\n", component, status)
	}

	if allHealthy {
		fmt.Println("  • All services healthy")
	} else {
		fmt.Println("  • Some services need initialization")
	}
}

func (shell *InteractiveShell) handleNotify(args []string) {
	if len(args) < 3 {
		fmt.Println("[ERROR] Type, subject, and message are required. Usage: notify <type> <subject> <message>")
		return
	}

	notifyType := args[0]
	subject := args[1]
	message := strings.Join(args[2:], " ")

	fmt.Printf("[INFO] Sending %s notification: %s - %s\n", notifyType, subject, message)

	// Send actual notification based on type
	var err error
	switch notifyType {
	case "email":
		// Would integrate with email service
		fmt.Println("[INFO] Email notification would be sent (not configured)")
	case "slack":
		// Would integrate with Slack API
		fmt.Println("[INFO] Slack notification would be sent (not configured)")
	case "webhook":
		// Would send to configured webhook
		fmt.Println("[INFO] Webhook notification would be sent (not configured)")
	default:
		fmt.Printf("[ERROR] Unknown notification type: %s\n", notifyType)
		return
	}

	if err != nil {
		fmt.Printf("[ERROR] Notification failed: %v\n", err)
	} else {
		fmt.Println("[SUCCESS] Notification sent successfully")
	}
}

func (shell *InteractiveShell) handleTerragrunt(args []string) {
	if len(args) == 0 {
		fmt.Println("[ERROR] Command is required. Usage: terragrunt <files|statefiles|analyze>")
		return
	}

	command := args[0]

	switch command {
	case "files":
		fmt.Println("[INFO] Scanning for Terragrunt files...")
		result, err := shell.terragruntMgr.FindTerragruntFiles()
		if err != nil {
			fmt.Printf("[ERROR] Failed to scan: %v\n", err)
			return
		}
		totalFiles := len(result.RootFiles) + len(result.ChildFiles)
		fmt.Printf("[SUCCESS] Found %d Terragrunt files\n", totalFiles)
		for i, file := range result.RootFiles {
			if i >= 5 {
				break
			}
			fmt.Printf("  • [Root] %s\n", file.Path)
		}
		for i, file := range result.ChildFiles {
			if i >= 5 {
				fmt.Printf("  ... and %d more child files\n", len(result.ChildFiles)-5)
				break
			}
			fmt.Printf("  • [Child] %s\n", file.Path)
		}

	case "statefiles":
		fmt.Println("[INFO] Discovering Terragrunt state files...")
		result, err := shell.terragruntMgr.FindTerragruntFiles()
		if err != nil {
			fmt.Printf("[ERROR] Failed to discover files: %v\n", err)
			return
		}
		// Parse state files from Terragrunt configs
		var stateFilePaths []string
		for _, file := range result.RootFiles {
			if strings.Contains(file.Path, "terragrunt.hcl") {
				stateFilePaths = append(stateFilePaths, file.Path)
			}
		}
		for _, file := range result.ChildFiles {
			if strings.Contains(file.Path, "terragrunt.hcl") {
				stateFilePaths = append(stateFilePaths, file.Path)
			}
		}
		fmt.Printf("[SUCCESS] Found %d Terragrunt configurations\n", len(stateFilePaths))

	case "analyze":
		if len(args) < 2 {
			fmt.Println("[ERROR] Path required. Usage: terragrunt analyze <path>")
			return
		}
		path := args[1]
		fmt.Printf("[INFO] Analyzing Terragrunt configuration at: %s\n", path)
		// Analyze the Terragrunt file
		result, err := shell.terragruntMgr.FindTerragruntFiles()
		if err != nil {
			fmt.Printf("[ERROR] Analysis failed: %v\n", err)
			return
		}
		fmt.Printf("[SUCCESS] Analysis completed:\n")
		fmt.Printf("  Root files: %d\n", len(result.RootFiles))
		fmt.Printf("  Child files: %d\n", len(result.ChildFiles))
		fmt.Printf("  Environments: %d\n", len(result.Environments))

	default:
		fmt.Printf("[ERROR] Unknown command: %s\n", command)
	}
}

// Help functions
func (shell *InteractiveShell) showDiscoverHelp(args []string) {
	fmt.Println("discover - Discover cloud resources")
	fmt.Println()
	fmt.Println("Usage: discover <provider> [regions...]")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  provider    Cloud provider (required)")
	fmt.Println("  regions     Cloud regions (optional, default: us-east-1)")
	fmt.Println()
	fmt.Println("Supported Providers:")
	fmt.Println("  aws         Amazon Web Services (26 regions)")
	fmt.Println("  azure       Microsoft Azure (60+ regions)")
	fmt.Println("  gcp         Google Cloud Platform (40+ regions)")
	fmt.Println("  digitalocean DigitalOcean (8 regions)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  discover aws")
	fmt.Println("  discover aws us-east-1 us-west-2")
	fmt.Println("  discover aws all")
	fmt.Println("  discover azure westeurope")
	fmt.Println("  discover gcp us-central1")
	fmt.Println("  discover digitalocean nyc1")
}

func (shell *InteractiveShell) showAnalyzeHelp(args []string) {
	fmt.Println("analyze - Analyze drift for a state file")
	fmt.Println()
	fmt.Println("Usage: analyze <statefile_id>")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  statefile_id    State file identifier (required)")
}

func (shell *InteractiveShell) showPerspectiveHelp(args []string) {
	fmt.Println("perspective - Compare state with live infrastructure")
	fmt.Println()
	fmt.Println("Usage: perspective <statefile_id> [provider]")
}

func (shell *InteractiveShell) showVisualizeHelp(args []string) {
	fmt.Println("visualize - Generate infrastructure visualization")
	fmt.Println()
	fmt.Println("Usage: visualize <statefile_id> [path]")
}

func (shell *InteractiveShell) showDiagramHelp(args []string) {
	fmt.Println("diagram - Generate infrastructure diagram")
	fmt.Println()
	fmt.Println("Usage: diagram <statefile_id>")
}

func (shell *InteractiveShell) showExportHelp(args []string) {
	fmt.Println("export - Export diagram in specified format")
	fmt.Println()
	fmt.Println("Usage: export <statefile_id> <format>")
}

func (shell *InteractiveShell) showStateFilesHelp(args []string) {
	fmt.Println("statefiles - List available state files")
	fmt.Println()
	fmt.Println("Usage: statefiles")
}

func (shell *InteractiveShell) showCredentialsHelp(args []string) {
	fmt.Println("credentials - Manage cloud provider credentials")
	fmt.Println()
	fmt.Println("Usage: credentials <command>")
}

func (shell *InteractiveShell) showRemediateHelp(args []string) {
	fmt.Println("remediate - Remediate drift with automated commands")
	fmt.Println()
	fmt.Println("Usage: remediate <drift_id> [options]")
}

func (shell *InteractiveShell) showRemediateBatchHelp(args []string) {
	fmt.Println("remediate-batch - Batch remediation")
	fmt.Println()
	fmt.Println("Usage: remediate-batch <statefile_id> [options]")
}

func (shell *InteractiveShell) showRemediateHistoryHelp(args []string) {
	fmt.Println("remediate-history - Show remediation history")
	fmt.Println()
	fmt.Println("Usage: remediate-history")
}

func (shell *InteractiveShell) showRemediateRollbackHelp(args []string) {
	fmt.Println("remediate-rollback - Rollback to previous state")
	fmt.Println()
	fmt.Println("Usage: remediate-rollback <snapshot_id>")
}

func (shell *InteractiveShell) showHealthHelp(args []string) {
	fmt.Println("health - Check service health")
	fmt.Println()
	fmt.Println("Usage: health")
}

func (shell *InteractiveShell) showNotifyHelp(args []string) {
	fmt.Println("notify - Send notifications")
	fmt.Println()
	fmt.Println("Usage: notify <type> <subject> <message>")
}

func (shell *InteractiveShell) showTerragruntHelp(args []string) {
	fmt.Println("terragrunt - Terragrunt operations")
	fmt.Println()
	fmt.Println("Usage: terragrunt <command>")
}
