package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/fatih/color"
)

// handleStateSurgery performs state manipulation operations
func handleStateSurgery(args []string) {
	if len(args) == 0 {
		showStateSurgeryHelp()
		return
	}
	
	operation := args[0]
	operationArgs := args[1:]
	
	switch operation {
	case "mv", "move":
		handleStateMove(operationArgs)
	case "rm", "remove":
		handleStateRemove(operationArgs)
	case "replace-provider":
		handleReplaceProvider(operationArgs)
	case "import":
		handleStateImport(operationArgs)
	case "pull":
		handleStatePull(operationArgs)
	case "push":
		handleStatePush(operationArgs)
	case "list", "ls":
		handleStateList(operationArgs)
	case "--help", "-h":
		showStateSurgeryHelp()
	default:
		fmt.Fprintf(os.Stderr, color.RedString("Unknown surgery operation: %s\n"), operation)
		showStateSurgeryHelp()
		os.Exit(1)
	}
}

// handleStateMove moves resources within state
func handleStateMove(args []string) {
	var (
		statePath = ""
		source    = ""
		target    = ""
		backup    = true
		dryRun    = false
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--no-backup":
			backup = false
		case "--dry-run":
			dryRun = true
		default:
			if source == "" {
				source = args[i]
			} else if target == "" {
				target = args[i]
			}
		}
	}
	
	if source == "" || target == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: Both source and target resources must be specified"))
		fmt.Println("\nUsage: driftmgr surgery mv <source> <target> [--state <file>]")
		os.Exit(1)
	}
	
	// Find state file
	if statePath == "" {
		statePath = findTerraformStateFile()
	}
	
	if statePath == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No Terraform state file found"))
		os.Exit(1)
	}
	
	fmt.Printf("Moving resource in state: %s\n", statePath)
	fmt.Printf("  From: %s\n", color.YellowString(source))
	fmt.Printf("  To:   %s\n", color.GreenString(target))
	
	if dryRun {
		fmt.Println("\n[DRY RUN - No changes will be made]")
	}
	
	// Create backup
	if backup && !dryRun {
		backupPath := fmt.Sprintf("%s.%d.backup", statePath, time.Now().Unix())
		if err := copyFile(statePath, backupPath); err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error creating backup: %v\n"), err)
			os.Exit(1)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}
	
	// Load state
	ctx := context.Background()
	loader := state.NewStateLoader(statePath)
	stateFile, err := loader.LoadStateFile(ctx, statePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error loading state: %v\n"), err)
		os.Exit(1)
	}
	
	// Find source resource
	var sourceResource *state.Resource
	for i, res := range stateFile.Resources {
		fullName := fmt.Sprintf("%s.%s", res.Type, res.Name)
		if fullName == source {
			sourceResource = &stateFile.Resources[i]
			break
		}
	}
	
	if sourceResource == nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error: Resource not found: %s\n"), source)
		os.Exit(1)
	}
	
	// Parse target
	targetParts := strings.Split(target, ".")
	if len(targetParts) != 2 {
		fmt.Fprintf(os.Stderr, color.RedString("Error: Invalid target format. Use: resource_type.resource_name\n"))
		os.Exit(1)
	}
	
	// Update resource
	sourceResource.Type = targetParts[0]
	sourceResource.Name = targetParts[1]
	
	if !dryRun {
		// Save state
		if err := saveStateFile(statePath, stateFile); err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error saving state: %v\n"), err)
			os.Exit(1)
		}
		fmt.Println(color.GreenString("✓ Resource moved successfully"))
	} else {
		fmt.Println("\nDry run complete - no changes made")
	}
}

// handleStateRemove removes resources from state
func handleStateRemove(args []string) {
	var (
		statePath = ""
		resources = []string{}
		backup    = true
		dryRun    = false
		force     = false
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--no-backup":
			backup = false
		case "--dry-run":
			dryRun = true
		case "--force", "-f":
			force = true
		default:
			resources = append(resources, args[i])
		}
	}
	
	if len(resources) == 0 {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No resources specified for removal"))
		fmt.Println("\nUsage: driftmgr surgery rm <resource> [<resource>...] [--state <file>]")
		os.Exit(1)
	}
	
	// Find state file
	if statePath == "" {
		statePath = findTerraformStateFile()
	}
	
	if statePath == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No Terraform state file found"))
		os.Exit(1)
	}
	
	fmt.Printf("Removing resources from state: %s\n", statePath)
	for _, res := range resources {
		fmt.Printf("  • %s\n", color.RedString(res))
	}
	
	if dryRun {
		fmt.Println("\n[DRY RUN - No changes will be made]")
	}
	
	// Confirm if not forced
	if !force && !dryRun {
		fmt.Print("\nAre you sure? This cannot be undone. Type 'yes' to continue: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Aborted")
			return
		}
	}
	
	// Create backup
	if backup && !dryRun {
		backupPath := fmt.Sprintf("%s.%d.backup", statePath, time.Now().Unix())
		if err := copyFile(statePath, backupPath); err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error creating backup: %v\n"), err)
			os.Exit(1)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}
	
	// Load state
	ctx := context.Background()
	loader := state.NewStateLoader(statePath)
	stateFile, err := loader.LoadStateFile(ctx, statePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error loading state: %v\n"), err)
		os.Exit(1)
	}
	
	// Remove resources
	removed := 0
	newResources := []state.Resource{}
	for _, res := range stateFile.Resources {
		fullName := fmt.Sprintf("%s.%s", res.Type, res.Name)
		shouldRemove := false
		for _, target := range resources {
			if fullName == target || matchesPattern(fullName, target) {
				shouldRemove = true
				removed++
				break
			}
		}
		if !shouldRemove {
			newResources = append(newResources, res)
		}
	}
	
	stateFile.Resources = newResources
	
	if !dryRun {
		// Save state
		if err := saveStateFile(statePath, stateFile); err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error saving state: %v\n"), err)
			os.Exit(1)
		}
		fmt.Printf(color.GreenString("✓ Removed %d resources from state\n"), removed)
	} else {
		fmt.Printf("\nDry run: would remove %d resources\n", removed)
	}
}

// handleReplaceProvider replaces provider in state
func handleReplaceProvider(args []string) {
	var (
		statePath    = ""
		oldProvider  = ""
		newProvider  = ""
		backup       = true
		dryRun       = false
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--no-backup":
			backup = false
		case "--dry-run":
			dryRun = true
		default:
			if oldProvider == "" {
				oldProvider = args[i]
			} else if newProvider == "" {
				newProvider = args[i]
			}
		}
	}
	
	if oldProvider == "" || newProvider == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: Both old and new provider must be specified"))
		fmt.Println("\nUsage: driftmgr surgery replace-provider <old> <new> [--state <file>]")
		fmt.Println("\nExample: driftmgr surgery replace-provider hashicorp/aws registry.terraform.io/hashicorp/aws")
		os.Exit(1)
	}
	
	// Find state file
	if statePath == "" {
		statePath = findTerraformStateFile()
	}
	
	if statePath == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No Terraform state file found"))
		os.Exit(1)
	}
	
	fmt.Printf("Replacing provider in state: %s\n", statePath)
	fmt.Printf("  From: %s\n", color.YellowString(oldProvider))
	fmt.Printf("  To:   %s\n", color.GreenString(newProvider))
	
	if dryRun {
		fmt.Println("\n[DRY RUN - No changes will be made]")
	}
	
	// Create backup
	if backup && !dryRun {
		backupPath := fmt.Sprintf("%s.%d.backup", statePath, time.Now().Unix())
		if err := copyFile(statePath, backupPath); err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error creating backup: %v\n"), err)
			os.Exit(1)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}
	
	// Load raw state for provider replacement
	data, err := os.ReadFile(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error reading state file: %v\n"), err)
		os.Exit(1)
	}
	
	// Parse as JSON
	var stateData map[string]interface{}
	if err := json.Unmarshal(data, &stateData); err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error parsing state file: %v\n"), err)
		os.Exit(1)
	}
	
	// Replace provider references
	replaced := replaceProviderInState(stateData, oldProvider, newProvider)
	
	if !dryRun {
		// Save state
		newData, err := json.MarshalIndent(stateData, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error encoding state: %v\n"), err)
			os.Exit(1)
		}
		
		if err := os.WriteFile(statePath, newData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, color.RedString("Error saving state: %v\n"), err)
			os.Exit(1)
		}
		
		fmt.Printf(color.GreenString("✓ Replaced %d provider references\n"), replaced)
	} else {
		fmt.Printf("\nDry run: would replace %d provider references\n", replaced)
	}
}

// handleStateImport imports resources into state
func handleStateImport(args []string) {
	var (
		statePath    = ""
		resourceType = ""
		resourceName = ""
		resourceID   = ""
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--provider":
			if i+1 < len(args) {
				// provider = args[i+1] // Not used in this function
				i++
			}
		default:
			if resourceType == "" {
				// Parse resource address (type.name)
				parts := strings.Split(args[i], ".")
				if len(parts) == 2 {
					resourceType = parts[0]
					resourceName = parts[1]
				}
			} else if resourceID == "" {
				resourceID = args[i]
			}
		}
	}
	
	if resourceType == "" || resourceName == "" || resourceID == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: Invalid import syntax"))
		fmt.Println("\nUsage: driftmgr surgery import <resource_type>.<name> <id> [--state <file>]")
		fmt.Println("\nExample: driftmgr surgery import aws_instance.web i-1234567890abcdef0")
		os.Exit(1)
	}
	
	// Find state file
	if statePath == "" {
		statePath = findTerraformStateFile()
	}
	
	if statePath == "" {
		// Create new state file
		statePath = "terraform.tfstate"
		fmt.Printf("Creating new state file: %s\n", statePath)
	}
	
	fmt.Printf("Importing resource into state:\n")
	fmt.Printf("  Resource: %s.%s\n", resourceType, resourceName)
	fmt.Printf("  ID: %s\n", resourceID)
	fmt.Printf("  State: %s\n", statePath)
	
	// This would normally call terraform import or use provider APIs
	// For now, we'll create a placeholder
	fmt.Println(color.YellowString("\n⚠ Direct import requires Terraform binary"))
	fmt.Printf("\nRun: terraform import %s.%s %s\n", resourceType, resourceName, resourceID)
}

// handleStatePull downloads state from backend
func handleStatePull(args []string) {
	fmt.Println(color.CyanString("Pulling state from backend..."))
	
	// This would integrate with various backends (S3, Azure, GCS, etc.)
	fmt.Println(color.YellowString("⚠ Backend operations require Terraform configuration"))
	fmt.Println("\nRun: terraform state pull > terraform.tfstate")
}

// handleStatePush uploads state to backend
func handleStatePush(args []string) {
	var statePath = "terraform.tfstate"
	
	for i := 0; i < len(args); i++ {
		if args[i] == "--state" && i+1 < len(args) {
			statePath = args[i+1]
		}
	}
	
	fmt.Printf("Pushing state to backend: %s\n", statePath)
	
	// This would integrate with various backends
	fmt.Println(color.YellowString("⚠ Backend operations require Terraform configuration"))
	fmt.Println("\nRun: terraform state push terraform.tfstate")
}

// handleStateList lists resources in state
func handleStateList(args []string) {
	var (
		statePath = ""
		pattern   = ""
		json      = false
	)
	
	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--json":
			json = true
		default:
			pattern = args[i]
		}
	}
	
	// Find state file
	if statePath == "" {
		statePath = findTerraformStateFile()
	}
	
	if statePath == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: No Terraform state file found"))
		os.Exit(1)
	}
	
	// Load state
	ctx := context.Background()
	loader := state.NewStateLoader(statePath)
	stateFile, err := loader.LoadStateFile(ctx, statePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Error loading state: %v\n"), err)
		os.Exit(1)
	}
	
	if json {
		// JSON output
		resources := []map[string]string{}
		for _, res := range stateFile.Resources {
			fullName := fmt.Sprintf("%s.%s", res.Type, res.Name)
			if pattern == "" || matchesPattern(fullName, pattern) {
				resources = append(resources, map[string]string{
					"address": fullName,
					"type":    res.Type,
					"name":    res.Name,
					"id":      res.ID,
				})
			}
		}
		output, _ := jsonMarshal(resources, "", "  ")
		fmt.Println(string(output))
	} else {
		// Text output
		fmt.Printf("Resources in state: %s\n\n", statePath)
		count := 0
		for _, res := range stateFile.Resources {
			fullName := fmt.Sprintf("%s.%s", res.Type, res.Name)
			if pattern == "" || matchesPattern(fullName, pattern) {
				fmt.Printf("  • %s\n", fullName)
				count++
			}
		}
		fmt.Printf("\nTotal: %d resources\n", count)
	}
}

// Helper functions
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func matchesPattern(name, pattern string) bool {
	// Simple wildcard matching
	if strings.Contains(pattern, "*") {
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		return strings.Contains(name, strings.ReplaceAll(pattern, ".*", ""))
	}
	return name == pattern
}

func saveStateFile(path string, state *state.State) error {
	// Convert state struct to raw JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func replaceProviderInState(data map[string]interface{}, oldProvider, newProvider string) int {
	replaced := 0
	
	// Walk through the state structure
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, v := range val {
				if k == "provider" {
					if str, ok := v.(string); ok && strings.Contains(str, oldProvider) {
						val[k] = strings.ReplaceAll(str, oldProvider, newProvider)
						replaced++
					}
				}
				walk(v)
			}
		case []interface{}:
			for _, item := range val {
				walk(item)
			}
		}
	}
	
	walk(data)
	return replaced
}

func jsonMarshal(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func showStateSurgeryHelp() {
	fmt.Println("Usage: driftmgr surgery <operation> [arguments] [flags]")
	fmt.Println()
	fmt.Println("Terraform state manipulation operations")
	fmt.Println()
	fmt.Println("Operations:")
	fmt.Println("  mv, move            Move resources within state")
	fmt.Println("  rm, remove          Remove resources from state")
	fmt.Println("  replace-provider    Replace provider references")
	fmt.Println("  import              Import existing resources into state")
	fmt.Println("  pull                Download state from backend")
	fmt.Println("  push                Upload state to backend")
	fmt.Println("  list, ls            List resources in state")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Move a resource")
	fmt.Println("  driftmgr surgery mv aws_instance.old aws_instance.new")
	fmt.Println()
	fmt.Println("  # Remove resources")
	fmt.Println("  driftmgr surgery rm aws_instance.temp aws_s3_bucket.old")
	fmt.Println()
	fmt.Println("  # Replace provider")
	fmt.Println("  driftmgr surgery replace-provider hashicorp/aws registry.terraform.io/hashicorp/aws")
	fmt.Println()
	fmt.Println("  # List resources")
	fmt.Println("  driftmgr surgery list aws_instance.*")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --state string    Path to state file (auto-detected if not specified)")
	fmt.Println("  --no-backup       Skip creating backup before modifications")
	fmt.Println("  --dry-run         Show what would be changed without modifying")
	fmt.Println("  --force, -f       Skip confirmation prompts")
}