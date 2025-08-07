package importer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Engine handles resource import operations
type Engine struct {
	// No longer using semaphore - will use channel-based approach
}

// Config holds configuration for import operations
type Config struct {
	InputFile      string
	Parallelism    int
	DryRun         bool
	GenerateConfig bool
	ValidateAfter  bool
}

// NewEngine creates a new import engine
func NewEngine() *Engine {
	return &Engine{}
}

// Import imports resources based on the provided configuration
func (e *Engine) Import(config Config) (*models.ImportResult, error) {
	resources, err := e.loadResources(config.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load resources: %w", err)
	}

	fmt.Printf("📦 Loaded %d resources for import\n", len(resources))

	result := &models.ImportResult{
		Commands: make([]models.ImportCommand, 0, len(resources)),
		Errors:   make([]models.ImportError, 0),
	}

	startTime := time.Now()

	// Generate import commands
	commands := e.generateImportCommands(resources)
	result.Commands = commands

	if config.DryRun {
		fmt.Println("🔍 Dry run mode - showing commands that would be executed:")
		for i, cmd := range commands {
			fmt.Printf("  %d. %s\n", i+1, cmd.Command)
		}
		result.Successful = len(commands)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Execute imports
	successful, failed, errors := e.executeImports(commands, config)

	result.Successful = successful
	result.Failed = failed
	result.Errors = errors
	result.Duration = time.Since(startTime)

	// Generate Terraform configuration if requested
	if config.GenerateConfig && result.Successful > 0 {
		err := e.generateTerraformConfig(commands)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to generate Terraform configuration: %v\n", err)
		} else {
			fmt.Println("✅ Generated Terraform configuration files")
		}
	}

	// Validate state after import if requested
	if config.ValidateAfter && result.Successful > 0 {
		err := e.validateState()
		if err != nil {
			fmt.Printf("⚠️  Warning: State validation failed: %v\n", err)
		} else {
			fmt.Println("✅ State validation passed")
		}
	}

	return result, nil
}

// loadResources loads resources from the input file
func (e *Engine) loadResources(inputFile string) ([]models.Resource, error) {
	ext := filepath.Ext(strings.ToLower(inputFile))

	switch ext {
	case ".csv":
		return e.loadResourcesFromCSV(inputFile)
	case ".json":
		return e.loadResourcesFromJSON(inputFile)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

// loadResourcesFromCSV loads resources from a CSV file
func (e *Engine) loadResourcesFromCSV(filename string) ([]models.Resource, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must contain at least a header and one data row")
	}

	// Parse header
	header := records[0]
	var resources []models.Resource

	for _, record := range records[1:] {
		if len(record) != len(header) {
			continue // Skip malformed rows
		}

		resource := models.Resource{
			Tags:     make(map[string]string),
			Metadata: make(map[string]interface{}),
		}

		// Map CSV columns to resource fields
		for i, value := range record {
			switch strings.ToLower(header[i]) {
			case "id":
				resource.ID = value
				resource.ImportID = value // Default import ID to resource ID
			case "name":
				resource.Name = value
			case "type":
				resource.Type = value
				resource.TerraformType = value
			case "provider":
				resource.Provider = value
			case "region":
				resource.Region = value
			case "import_id":
				resource.ImportID = value
			case "terraform_type":
				resource.TerraformType = value
			}
		}

		// Validate required fields
		if resource.ID == "" || resource.Type == "" || resource.ImportID == "" {
			continue // Skip incomplete resources
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// loadResourcesFromJSON loads resources from a JSON file
func (e *Engine) loadResourcesFromJSON(filename string) ([]models.Resource, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var resources []models.Resource
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return resources, nil
}

// generateImportCommands generates terraform import commands for resources
func (e *Engine) generateImportCommands(resources []models.Resource) []models.ImportCommand {
	var commands []models.ImportCommand

	for _, resource := range resources {
		resourceName := e.generateResourceName(resource)
		terraformType := resource.TerraformType
		if terraformType == "" {
			terraformType = resource.Type
		}

		command := models.ImportCommand{
			ResourceType: terraformType,
			ResourceName: resourceName,
			ResourceID:   resource.ImportID,
			Command:      fmt.Sprintf("terraform import %s.%s %s", terraformType, resourceName, resource.ImportID),
		}

		// Generate basic Terraform configuration block
		command.Configuration = e.generateResourceConfig(resource, resourceName)

		commands = append(commands, command)
	}

	return commands
}

// generateResourceName generates a valid Terraform resource name
func (e *Engine) generateResourceName(resource models.Resource) string {
	name := resource.Name
	if name == "" {
		name = resource.ID
	}

	// Clean the name to make it valid for Terraform
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, " ", "_")

	// Remove any non-alphanumeric characters except underscores
	var result strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '_' {
			result.WriteRune(char)
		}
	}

	name = result.String()

	// Ensure it starts with a letter
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "resource_" + name
	}

	// Ensure it's not empty
	if name == "" {
		name = "imported_resource"
	}

	return name
}

// generateResourceConfig generates a basic Terraform configuration block
func (e *Engine) generateResourceConfig(resource models.Resource, resourceName string) string {
	config := fmt.Sprintf("resource \"%s\" \"%s\" {\n", resource.TerraformType, resourceName)
	config += "  # Configuration will be imported automatically\n"
	config += "  # Please review and update as needed\n"

	// Add basic lifecycle block to prevent accidental deletion
	config += "\n  lifecycle {\n"
	config += "    prevent_destroy = true\n"
	config += "  }\n"

	if len(resource.Tags) > 0 {
		config += "\n  tags = {\n"
		for key, value := range resource.Tags {
			config += fmt.Sprintf("    \"%s\" = \"%s\"\n", key, value)
		}
		config += "  }\n"
	}

	config += "}\n"
	return config
}

// executeImports executes the terraform import commands
func (e *Engine) executeImports(commands []models.ImportCommand, config Config) (successful, failed int, errors []models.ImportError) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var importErrors []models.ImportError

	// Channel to limit parallelism
	sem := make(chan struct{}, config.Parallelism)

	fmt.Printf("🚀 Executing %d import operations with parallelism of %d...\n", len(commands), config.Parallelism)

	for i, cmd := range commands {
		wg.Add(1)
		go func(index int, command models.ImportCommand) {
			defer wg.Done()

			// Acquire semaphore to limit parallelism
			sem <- struct{}{}
			defer func() { <-sem }()

			fmt.Printf("  [%d/%d] Importing %s.%s...\n", index+1, len(commands), command.ResourceType, command.ResourceName)

			err := e.executeImportCommand(command)

			mu.Lock()
			if err != nil {
				failed++
				importErrors = append(importErrors, models.ImportError{
					Resource: command.ResourceName,
					Error:    err.Error(),
					Code:     "IMPORT_FAILED",
				})
				fmt.Printf("    ❌ Failed: %v\n", err)
			} else {
				successful++
				fmt.Printf("    ✅ Success\n")
			}
			mu.Unlock()
		}(i, cmd)
	}

	wg.Wait()

	return successful, failed, importErrors
}

// executeImportCommand executes a single terraform import command
func (e *Engine) executeImportCommand(command models.ImportCommand) error {
	// Split the command to get arguments
	parts := strings.Fields(command.Command)
	if len(parts) < 4 || parts[0] != "terraform" || parts[1] != "import" {
		return fmt.Errorf("invalid terraform import command: %s", command.Command)
	}

	// Execute the terraform import command
	cmd := exec.Command("terraform", parts[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("terraform import failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// generateTerraformConfig generates Terraform configuration files
func (e *Engine) generateTerraformConfig(commands []models.ImportCommand) error {
	// Group commands by resource type for organization
	configsByType := make(map[string][]models.ImportCommand)

	for _, cmd := range commands {
		configsByType[cmd.ResourceType] = append(configsByType[cmd.ResourceType], cmd)
	}

	// Generate separate files for each resource type
	for resourceType, cmds := range configsByType {
		filename := fmt.Sprintf("imported_%s.tf", strings.ReplaceAll(resourceType, "_", "-"))

		var content strings.Builder
		content.WriteString(fmt.Sprintf("# Imported %s resources\n\n", resourceType))

		for _, cmd := range cmds {
			content.WriteString(cmd.Configuration)
			content.WriteString("\n")
		}

		if err := os.WriteFile(filename, []byte(content.String()), 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", filename, err)
		}

		fmt.Printf("  Generated %s (%d resources)\n", filename, len(cmds))
	}

	return nil
}

// validateState validates the Terraform state after import
func (e *Engine) validateState() error {
	// Run terraform plan to check for any drift
	cmd := exec.Command("terraform", "plan", "-detailed-exitcode")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Exit code 2 means there are changes, which might be expected
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 2 {
			fmt.Println("  State validation shows configuration drift (expected after import)")
			return nil
		}
		return fmt.Errorf("terraform plan failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
