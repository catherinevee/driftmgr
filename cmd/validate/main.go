package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/state/parser"
)

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Stats   Stats    `json:"stats"`
}

// Stats contains validation statistics
type Stats struct {
	TotalResources   int `json:"total_resources"`
	ValidResources   int `json:"valid_resources"`
	InvalidResources int `json:"invalid_resources"`
	TotalProviders   int `json:"total_providers"`
	ValidProviders   int `json:"valid_providers"`
	InvalidProviders int `json:"invalid_providers"`
}

func main() {
	var (
		statePath  = flag.String("state", "", "Path to Terraform state file to validate")
		configPath = flag.String("config", "", "Path to configuration file to validate")
		provider   = flag.String("provider", "", "Provider to validate (aws, azure, gcp, digitalocean)")
		outputJSON = flag.Bool("json", false, "Output results in JSON format")
		strict     = flag.Bool("strict", false, "Strict validation mode")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *statePath == "" && *configPath == "" && *provider == "" {
		fmt.Println("Usage: validate-discovery [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate state file if provided
	if *statePath != "" {
		validateStateFile(*statePath, &result, *verbose)
	}

	// Validate configuration if provided
	if *configPath != "" {
		validateConfigFile(*configPath, &result, *verbose)
	}

	// Validate provider if specified
	if *provider != "" {
		validateProvider(*provider, &result, *verbose)
	}

	// Apply strict mode
	if *strict && len(result.Warnings) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Strict mode: %d warnings treated as errors", len(result.Warnings)))
	}

	// Output results
	if *outputJSON {
		outputJSONResult(result)
	} else {
		outputTextResult(result, *verbose)
	}

	// Exit with appropriate code
	if !result.Valid {
		os.Exit(1)
	}
}

func validateStateFile(path string, result *ValidationResult, verbose bool) {
	if verbose {
		fmt.Printf("Validating state file: %s\n", path)
	}

	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("State file not found: %s", path))
		return
	}

	// Parse state file
	stateParser := parser.NewParser()
	state, err := stateParser.ParseFile(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse state file: %v", err))
		return
	}

	// Validate state structure
	if state.Version < 3 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Old state format version %d", state.Version))
	}

	// Count resources
	result.Stats.TotalResources = len(state.Resources)
	
	// Validate each resource
	for _, resource := range state.Resources {
		if err := validateResource(resource); err != nil {
			result.Stats.InvalidResources++
			result.Warnings = append(result.Warnings, fmt.Sprintf("Resource %s: %v", resource.Name, err))
		} else {
			result.Stats.ValidResources++
		}
	}

	if verbose {
		fmt.Printf("State validation complete: %d resources validated\n", result.Stats.TotalResources)
	}
}

func validateConfigFile(path string, result *ValidationResult, verbose bool) {
	if verbose {
		fmt.Printf("Validating configuration file: %s\n", path)
	}

	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Config file not found: %s", path))
		return
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read config file: %v", err))
		return
	}

	// Validate based on extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		validateYAMLConfig(data, result)
	case ".json":
		validateJSONConfig(data, result)
	case ".toml":
		validateTOMLConfig(data, result)
	default:
		result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown config format: %s", ext))
	}

	if verbose {
		fmt.Println("Configuration validation complete")
	}
}

func validateProvider(providerName string, result *ValidationResult, verbose bool) {
	if verbose {
		fmt.Printf("Validating provider: %s\n", providerName)
	}

	result.Stats.TotalProviders = 1

	// Create provider instance
	factory := providers.NewFactory()
	provider, err := factory.Create(providerName)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid provider %s: %v", providerName, err))
		result.Stats.InvalidProviders = 1
		return
	}

	// Validate provider credentials
	if err := provider.Validate(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Provider %s validation failed: %v", providerName, err))
		result.Stats.InvalidProviders = 1
		return
	}

	// Test provider connectivity
	if err := provider.TestConnection(); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Provider %s connection test failed: %v", providerName, err))
	}

	result.Stats.ValidProviders = 1

	if verbose {
		fmt.Printf("Provider %s validation complete\n", providerName)
	}
}

func validateResource(resource interface{}) error {
	// Basic resource validation
	resourceMap, ok := resource.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid resource type")
	}

	// Check required fields
	if _, ok := resourceMap["type"]; !ok {
		return fmt.Errorf("missing resource type")
	}
	if _, ok := resourceMap["name"]; !ok {
		return fmt.Errorf("missing resource name")
	}

	return nil
}

func validateYAMLConfig(data []byte, result *ValidationResult) {
	// Basic YAML validation
	// In production, would use yaml.Unmarshal
	if !strings.Contains(string(data), ":") {
		result.Errors = append(result.Errors, "Invalid YAML format")
		result.Valid = false
	}
}

func validateJSONConfig(data []byte, result *ValidationResult) {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON: %v", err))
		result.Valid = false
	}
}

func validateTOMLConfig(data []byte, result *ValidationResult) {
	// Basic TOML validation
	if !strings.Contains(string(data), "=") {
		result.Errors = append(result.Errors, "Invalid TOML format")
		result.Valid = false
	}
}

func outputJSONResult(result ValidationResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal result: %v", err)
	}
	fmt.Println(string(data))
}

func outputTextResult(result ValidationResult, verbose bool) {
	if result.Valid {
		fmt.Println("✅ Validation PASSED")
	} else {
		fmt.Println("❌ Validation FAILED")
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if verbose || !result.Valid {
		fmt.Println("\nStatistics:")
		fmt.Printf("  Total Resources: %d\n", result.Stats.TotalResources)
		fmt.Printf("  Valid Resources: %d\n", result.Stats.ValidResources)
		fmt.Printf("  Invalid Resources: %d\n", result.Stats.InvalidResources)
		fmt.Printf("  Total Providers: %d\n", result.Stats.TotalProviders)
		fmt.Printf("  Valid Providers: %d\n", result.Stats.ValidProviders)
		fmt.Printf("  Invalid Providers: %d\n", result.Stats.InvalidProviders)
	}
}