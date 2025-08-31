package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// handleTerraformImportGeneration generates Terraform import commands for unmanaged resources
func handleTerraformImportGeneration(args []string) {
	ctx := context.Background()
	
	// Parse arguments
	var (
		provider     = ""
		statePath    = ""
		outputFile   = ""
		resourceType = ""
		generateTF   = false
		dryRun       = false
	)
	
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--state":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		case "--type":
			if i+1 < len(args) {
				resourceType = args[i+1]
				i++
			}
		case "--generate-tf", "--gen-tf":
			generateTF = true
		case "--dry-run":
			dryRun = true
		case "--help", "-h":
			showTerraformImportHelp()
			return
		}
	}
	
	// Auto-detect state file if not specified
	if statePath == "" {
		statePath = findTerraformState()
	}
	
	if statePath == "" {
		fmt.Fprintf(os.Stderr, "Error: No Terraform state file found. Specify with --state\n")
		os.Exit(1)
	}
	
	fmt.Printf("Analyzing state file: %s\n", statePath)
	
	// Load state file
	loader := state.NewStateLoader(statePath)
	stateFile, err := loader.LoadStateFile(ctx, statePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load state file: %v\n", err)
		os.Exit(1)
	}
	
	// Get managed resources from state
	managedResources := make(map[string]bool)
	for _, resource := range stateFile.Resources {
		key := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		managedResources[key] = true
	}
	
	// Discover actual resources
	if provider == "" {
		provider = detectProviderFromStateFile(stateFile)
	}
	
	fmt.Printf("Discovering %s resources...\n", provider)
	discoveryService, err := discovery.InitializeServiceSilent(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize discovery: %v\n", err)
		os.Exit(1)
	}
	
	discoveryResult, err := discoveryService.DiscoverProvider(ctx, provider, discovery.DiscoveryOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to discover resources: %v\n", err)
		os.Exit(1)
	}
	
	// Find unmanaged resources
	var unmanagedResources []models.Resource
	for _, resource := range discoveryResult.Resources {
		terraformType := mapToTerraformType(provider, resource.Type)
		terraformName := sanitizeResourceName(resource.Name)
		key := fmt.Sprintf("%s.%s", terraformType, terraformName)
		
		if !managedResources[key] {
			if resourceType == "" || terraformType == resourceType {
				unmanagedResources = append(unmanagedResources, resource)
			}
		}
	}
	
	if len(unmanagedResources) == 0 {
		fmt.Println("✓ No unmanaged resources found!")
		return
	}
	
	fmt.Printf("\nFound %d unmanaged resources\n", len(unmanagedResources))
	
	// Generate import commands and optionally Terraform configurations
	var output strings.Builder
	
	// Header
	output.WriteString("#!/bin/bash\n")
	output.WriteString(fmt.Sprintf("# Terraform Import Commands\n"))
	output.WriteString(fmt.Sprintf("# Generated: %s\n", time.Now().Format(time.RFC3339)))
	output.WriteString(fmt.Sprintf("# State File: %s\n", statePath))
	output.WriteString(fmt.Sprintf("# Provider: %s\n\n", provider))
	
	if generateTF {
		// Also create .tf file content
		var tfContent strings.Builder
		tfContent.WriteString(fmt.Sprintf("# Terraform Configuration for Imported Resources\n"))
		tfContent.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))
		
		for _, resource := range unmanagedResources {
			terraformType := mapToTerraformType(provider, resource.Type)
			terraformName := sanitizeResourceName(resource.Name)
			
			// Generate import command
			importID := getImportID(provider, resource)
			output.WriteString(fmt.Sprintf("echo \"Importing %s.%s...\"\n", terraformType, terraformName))
			output.WriteString(fmt.Sprintf("terraform import %s.%s %s\n\n", terraformType, terraformName, importID))
			
			// Generate Terraform configuration
			tfContent.WriteString(generateTerraformResource(provider, terraformType, terraformName, resource))
		}
		
		// Write .tf file
		tfFile := strings.Replace(outputFile, ".sh", ".tf", 1)
		if tfFile == outputFile {
			tfFile = "imported_resources.tf"
		}
		
		if !dryRun {
			if err := os.WriteFile(tfFile, []byte(tfContent.String()), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to write .tf file: %v\n", err)
			} else {
				fmt.Printf("✓ Terraform configuration written to: %s\n", tfFile)
			}
		} else {
			fmt.Println("\n--- Terraform Configuration (DRY RUN) ---")
			fmt.Print(tfContent.String())
		}
	} else {
		// Just generate import commands
		for _, resource := range unmanagedResources {
			terraformType := mapToTerraformType(provider, resource.Type)
			terraformName := sanitizeResourceName(resource.Name)
			importID := getImportID(provider, resource)
			
			output.WriteString(fmt.Sprintf("# Resource: %s (%s)\n", resource.Name, resource.Type))
			output.WriteString(fmt.Sprintf("terraform import %s.%s %s\n\n", terraformType, terraformName, importID))
		}
	}
	
	// Add summary
	output.WriteString(fmt.Sprintf("\n# Total resources to import: %d\n", len(unmanagedResources)))
	output.WriteString("echo \"Import complete!\"\n")
	
	// Write or display output
	if !dryRun {
		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(output.String()), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to write file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ Import script written to: %s\n", outputFile)
			fmt.Println("\nRun the script with:")
			fmt.Printf("  bash %s\n", outputFile)
		} else {
			fmt.Print(output.String())
		}
	} else {
		fmt.Println("\n--- DRY RUN - No files written ---")
		fmt.Print(output.String())
	}
}

// mapToTerraformType maps cloud resource types to Terraform resource types
func mapToTerraformType(provider, resourceType string) string {
	switch provider {
	case "aws":
		switch resourceType {
		case "EC2::Instance":
			return "aws_instance"
		case "EC2::SecurityGroup":
			return "aws_security_group"
		case "EC2::VPC":
			return "aws_vpc"
		case "EC2::Subnet":
			return "aws_subnet"
		case "S3::Bucket":
			return "aws_s3_bucket"
		case "RDS::DBInstance":
			return "aws_db_instance"
		case "Lambda::Function":
			return "aws_lambda_function"
		case "IAM::Role":
			return "aws_iam_role"
		case "IAM::Policy":
			return "aws_iam_policy"
		default:
			// Convert AWS resource type to Terraform format
			parts := strings.Split(resourceType, "::")
			if len(parts) == 2 {
				return fmt.Sprintf("aws_%s", strings.ToLower(parts[1]))
			}
			return fmt.Sprintf("aws_%s", strings.ToLower(strings.ReplaceAll(resourceType, "::", "_")))
		}
	case "azure":
		switch resourceType {
		case "Microsoft.Compute/virtualMachines":
			return "azurerm_virtual_machine"
		case "Microsoft.Network/virtualNetworks":
			return "azurerm_virtual_network"
		case "Microsoft.Network/networkSecurityGroups":
			return "azurerm_network_security_group"
		case "Microsoft.Storage/storageAccounts":
			return "azurerm_storage_account"
		default:
			// Convert Azure resource type to Terraform format
			parts := strings.Split(resourceType, "/")
			if len(parts) == 2 {
				return fmt.Sprintf("azurerm_%s", strings.ToLower(parts[1]))
			}
			return fmt.Sprintf("azurerm_%s", strings.ToLower(strings.ReplaceAll(resourceType, "/", "_")))
		}
	case "gcp":
		switch resourceType {
		case "compute.v1.instance":
			return "google_compute_instance"
		case "compute.v1.network":
			return "google_compute_network"
		case "storage.v1.bucket":
			return "google_storage_bucket"
		default:
			// Convert GCP resource type to Terraform format
			parts := strings.Split(resourceType, ".")
			if len(parts) >= 2 {
				return fmt.Sprintf("google_%s_%s", parts[0], parts[len(parts)-1])
			}
			return fmt.Sprintf("google_%s", strings.ReplaceAll(resourceType, ".", "_"))
		}
	default:
		return strings.ToLower(resourceType)
	}
}

// sanitizeResourceName creates a valid Terraform resource name
func sanitizeResourceName(name string) string {
	// Replace invalid characters with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	
	// Ensure it starts with a letter
	if len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
		name = "resource_" + name
	}
	
	return strings.ToLower(name)
}

// getImportID returns the import ID for a resource
func getImportID(provider string, resource models.Resource) string {
	switch provider {
	case "aws":
		// AWS resources typically use their ID directly
		return resource.ID
	case "azure":
		// Azure resources use full resource ID
		if resource.Properties != nil {
			if resourceID, ok := resource.Properties["id"].(string); ok {
				return resourceID
			}
		}
		return resource.ID
	case "gcp":
		// GCP resources often need project/zone/name format
		if resource.Properties != nil {
			if selfLink, ok := resource.Properties["selfLink"].(string); ok {
				return selfLink
			}
		}
		return resource.ID
	default:
		return resource.ID
	}
}

// generateTerraformResource generates a basic Terraform resource configuration
func generateTerraformResource(provider, tfType, tfName string, resource models.Resource) string {
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n", tfType, tfName))
	
	// Add basic required arguments based on resource type
	switch tfType {
	case "aws_instance":
		output.WriteString("  # Required arguments - update with actual values\n")
		output.WriteString("  ami           = \"PLACEHOLDER_AMI\"\n")
		output.WriteString("  instance_type = \"t2.micro\"\n")
	case "aws_security_group":
		output.WriteString("  # Required arguments - update with actual values\n")
		output.WriteString(fmt.Sprintf("  name = \"%s\"\n", resource.Name))
	case "aws_s3_bucket":
		output.WriteString(fmt.Sprintf("  bucket = \"%s\"\n", resource.Name))
	case "azurerm_virtual_machine":
		output.WriteString("  # Required arguments - update with actual values\n")
		output.WriteString(fmt.Sprintf("  name                = \"%s\"\n", resource.Name))
		output.WriteString("  location            = \"PLACEHOLDER_LOCATION\"\n")
		output.WriteString("  resource_group_name = \"PLACEHOLDER_RG\"\n")
	case "google_compute_instance":
		output.WriteString("  # Required arguments - update with actual values\n")
		output.WriteString(fmt.Sprintf("  name         = \"%s\"\n", resource.Name))
		output.WriteString("  machine_type = \"f1-micro\"\n")
		output.WriteString("  zone         = \"PLACEHOLDER_ZONE\"\n")
	default:
		output.WriteString("  # Add required arguments for this resource type\n")
		if resource.Name != "" {
			output.WriteString(fmt.Sprintf("  name = \"%s\"\n", resource.Name))
		}
	}
	
	// Add tags if present
	if resource.Tags != nil {
		if tags, ok := resource.Tags.(map[string]string); ok && len(tags) > 0 {
			output.WriteString("\n  tags = {\n")
			for k, v := range tags {
				output.WriteString(fmt.Sprintf("    %s = \"%s\"\n", k, v))
			}
			output.WriteString("  }\n")
		}
	}
	
	// Add lifecycle block to prevent destruction during import
	output.WriteString("\n  lifecycle {\n")
	output.WriteString("    prevent_destroy = true\n")
	output.WriteString("  }\n")
	
	output.WriteString("}\n\n")
	
	return output.String()
}

// findTerraformState searches for Terraform state files in common locations
func findTerraformState() string {
	// Check common locations
	locations := []string{
		"terraform.tfstate",
		"terraform.tfstate.backup",
		".terraform/terraform.tfstate",
		"terraform.tfstate.d/*/terraform.tfstate",
	}
	
	for _, loc := range locations {
		if matches, err := filepath.Glob(loc); err == nil && len(matches) > 0 {
			return matches[0]
		}
	}
	
	return ""
}

// detectProviderFromStateFile detects the cloud provider from state file
func detectProviderFromStateFile(stateFile *state.State) string {
	for _, resource := range stateFile.Resources {
		if strings.HasPrefix(resource.Type, "aws_") {
			return "aws"
		}
		if strings.HasPrefix(resource.Type, "azurerm_") {
			return "azure"
		}
		if strings.HasPrefix(resource.Type, "google_") {
			return "gcp"
		}
		if strings.HasPrefix(resource.Type, "digitalocean_") {
			return "digitalocean"
		}
	}
	return ""
}

// showTerraformImportHelp displays help for the import generation command
func showTerraformImportHelp() {
	fmt.Println("Usage: driftmgr generate-import [flags]")
	fmt.Println()
	fmt.Println("Generate Terraform import commands for unmanaged resources")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --state string       Path to Terraform state file (auto-detected if not specified)")
	fmt.Println("  --provider string    Cloud provider (aws, azure, gcp)")
	fmt.Println("  --type string        Filter by resource type (e.g., aws_instance)")
	fmt.Println("  --output string      Output file for import script")
	fmt.Println("  --generate-tf        Also generate .tf configuration files")
	fmt.Println("  --dry-run           Show what would be generated without writing files")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate import commands for all unmanaged resources")
	fmt.Println("  driftmgr generate-import")
	fmt.Println()
	fmt.Println("  # Generate import script with Terraform configurations")
	fmt.Println("  driftmgr generate-import --generate-tf --output import.sh")
	fmt.Println()
	fmt.Println("  # Generate imports for specific resource type")
	fmt.Println("  driftmgr generate-import --type aws_instance --dry-run")
}