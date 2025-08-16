package remediation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// TerraformRemediationEngine provides Terraform-native remediation capabilities
type TerraformRemediationEngine struct {
	workingDir    string
	terraformPath string
}

// TerraformRemediationPlan represents a Terraform remediation plan
type TerraformRemediationPlan struct {
	ID          string                         `json:"id"`
	Description string                         `json:"description"`
	Resources   []TerraformRemediationResource `json:"resources"`
	PlanOutput  string                         `json:"plan_output"`
	PlanFile    string                         `json:"plan_file"`
	CreatedAt   time.Time                      `json:"created_at"`
	Status      string                         `json:"status"`
}

// TerraformRemediationResource represents a resource to be remediated
type TerraformRemediationResource struct {
	ResourceID    string                 `json:"resource_id"`
	ResourceType  string                 `json:"resource_type"`
	ResourceName  string                 `json:"resource_name"`
	Provider      string                 `json:"provider"`
	Action        string                 `json:"action"` // create, update, delete, import
	Configuration map[string]interface{} `json:"configuration"`
	ImportID      string                 `json:"import_id,omitempty"`
	Dependencies  []string               `json:"dependencies"`
}

// NewTerraformRemediationEngine creates a new Terraform remediation engine
func NewTerraformRemediationEngine(workingDir string) (*TerraformRemediationEngine, error) {
	// Find terraform executable
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("terraform not found in PATH: %w", err)
	}

	// Create working directory if it doesn't exist
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	return &TerraformRemediationEngine{
		workingDir:    workingDir,
		terraformPath: terraformPath,
	}, nil
}

// GenerateTerraformConfiguration generates Terraform configuration for remediation
func (tre *TerraformRemediationEngine) GenerateTerraformConfiguration(drifts []models.DriftResult) (*TerraformRemediationPlan, error) {
	plan := &TerraformRemediationPlan{
		ID:          fmt.Sprintf("plan_%d", time.Now().Unix()),
		Description: "Terraform remediation plan",
		Resources:   []TerraformRemediationResource{},
		CreatedAt:   time.Now(),
		Status:      "created",
	}

	// Group drifts by action type
	createResources := []models.DriftResult{}
	updateResources := []models.DriftResult{}
	deleteResources := []models.DriftResult{}
	importResources := []models.DriftResult{}

	for _, drift := range drifts {
		switch drift.DriftType {
		case "missing":
			createResources = append(createResources, drift)
		case "modified":
			updateResources = append(updateResources, drift)
		case "extra":
			deleteResources = append(deleteResources, drift)
		case "unmanaged":
			importResources = append(importResources, drift)
		}
	}

	// Generate resources in dependency order
	allResources := []TerraformRemediationResource{}

	// Handle imports first (they need to be imported before other operations)
	for _, drift := range importResources {
		resource := tre.createImportResource(drift)
		allResources = append(allResources, resource)
	}

	// Handle creates (dependencies first)
	createResources = tre.sortByDependencies(createResources)
	for _, drift := range createResources {
		resource := tre.createResource(drift)
		allResources = append(allResources, resource)
	}

	// Handle updates
	for _, drift := range updateResources {
		resource := tre.updateResource(drift)
		allResources = append(allResources, resource)
	}

	// Handle deletes (reverse dependency order)
	deleteResources = tre.sortByDependencies(deleteResources)
	for i := len(deleteResources) - 1; i >= 0; i-- {
		resource := tre.deleteResource(deleteResources[i])
		allResources = append(allResources, resource)
	}

	plan.Resources = allResources

	return plan, nil
}

// createImportResource creates a resource for importing unmanaged resources
func (tre *TerraformRemediationEngine) createImportResource(drift models.DriftResult) TerraformRemediationResource {
	return TerraformRemediationResource{
		ResourceID:   drift.ResourceID,
		ResourceType: drift.ResourceType,
		ResourceName: drift.ResourceName,
		Provider:     drift.Provider,
		Action:       "import",
		Configuration: map[string]interface{}{
			"id": drift.ResourceID,
		},
		ImportID:     drift.ResourceID,
		Dependencies: []string{},
	}
}

// createResource creates a resource for creating missing resources
func (tre *TerraformRemediationEngine) createResource(drift models.DriftResult) TerraformRemediationResource {
	return TerraformRemediationResource{
		ResourceID:    drift.ResourceID,
		ResourceType:  drift.ResourceType,
		ResourceName:  drift.ResourceName,
		Provider:      drift.Provider,
		Action:        "create",
		Configuration: map[string]interface{}{},
		Dependencies:  []string{},
	}
}

// updateResource creates a resource for updating modified resources
func (tre *TerraformRemediationEngine) updateResource(drift models.DriftResult) TerraformRemediationResource {
	return TerraformRemediationResource{
		ResourceID:    drift.ResourceID,
		ResourceType:  drift.ResourceType,
		ResourceName:  drift.ResourceName,
		Provider:      drift.Provider,
		Action:        "update",
		Configuration: map[string]interface{}{},
		Dependencies:  []string{},
	}
}

// deleteResource creates a resource for deleting extra resources
func (tre *TerraformRemediationEngine) deleteResource(drift models.DriftResult) TerraformRemediationResource {
	return TerraformRemediationResource{
		ResourceID:   drift.ResourceID,
		ResourceType: drift.ResourceType,
		ResourceName: drift.ResourceName,
		Provider:     drift.Provider,
		Action:       "delete",
		Configuration: map[string]interface{}{
			"id": drift.ResourceID,
		},
		Dependencies: []string{},
	}
}

// extractDependencies extracts dependencies from resource configuration
func (tre *TerraformRemediationEngine) extractDependencies(config map[string]interface{}) []string {
	var dependencies []string

	// Common dependency patterns
	dependencyPatterns := []string{
		"subnet_id", "vpc_id", "security_group_ids", "route_table_id",
		"target_group_arns", "load_balancer_arn", "cluster_arn",
		"bucket", "table_name", "function_name", "role_arn",
		"resource_group_name", "virtual_network_name", "storage_account_name",
		"network", "subnetwork", "service_account", "project",
	}

	for _, pattern := range dependencyPatterns {
		if value, exists := config[pattern]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				// Extract resource reference
				if strings.Contains(strValue, ".") {
					parts := strings.Split(strValue, ".")
					if len(parts) >= 2 {
						dependencies = append(dependencies, parts[1])
					}
				}
			}
		}
	}

	return dependencies
}

// sortByDependencies sorts resources by their dependencies
func (tre *TerraformRemediationEngine) sortByDependencies(drifts []models.DriftResult) []models.DriftResult {
	// Simple topological sort implementation
	// In a real implementation, you'd want a more sophisticated algorithm
	sorted := make([]models.DriftResult, len(drifts))
	copy(sorted, drifts)

	// Sort by resource type to ensure dependencies come first
	// This is a simplified approach - a proper implementation would build a dependency graph
	resourceTypeOrder := map[string]int{
		"aws_vpc":             1,
		"aws_subnet":          2,
		"aws_security_group":  3,
		"aws_instance":        4,
		"aws_lb":              5,
		"aws_lb_target_group": 6,
		"aws_lb_listener":     7,
	}

	// Sort by dependency order
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			orderI := resourceTypeOrder[sorted[i].ResourceType]
			orderJ := resourceTypeOrder[sorted[j].ResourceType]

			if orderI > orderJ {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// GenerateTerraformFiles generates Terraform configuration files
func (tre *TerraformRemediationEngine) GenerateTerraformFiles(plan *TerraformRemediationPlan) error {
	// Generate main.tf
	mainContent := tre.generateMainTf(plan)
	mainPath := filepath.Join(tre.workingDir, "main.tf")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("failed to write main.tf: %w", err)
	}

	// Generate variables.tf
	variablesContent := tre.generateVariablesTf(plan)
	variablesPath := filepath.Join(tre.workingDir, "variables.tf")
	if err := os.WriteFile(variablesPath, []byte(variablesContent), 0644); err != nil {
		return fmt.Errorf("failed to write variables.tf: %w", err)
	}

	// Generate outputs.tf
	outputsContent := tre.generateOutputsTf(plan)
	outputsPath := filepath.Join(tre.workingDir, "outputs.tf")
	if err := os.WriteFile(outputsPath, []byte(outputsContent), 0644); err != nil {
		return fmt.Errorf("failed to write outputs.tf: %w", err)
	}

	return nil
}

// generateMainTf generates the main.tf file content
func (tre *TerraformRemediationEngine) generateMainTf(plan *TerraformRemediationPlan) string {
	var content strings.Builder

	// Add provider configuration
	content.WriteString("terraform {\n")
	content.WriteString("  required_version = \">= 1.0\"\n")
	content.WriteString("}\n\n")

	// Add provider blocks
	providers := tre.getUniqueProviders(plan)
	for _, provider := range providers {
		content.WriteString(fmt.Sprintf("provider \"%s\" {\n", provider))
		content.WriteString("  # Configure your provider here\n")
		content.WriteString("}\n\n")
	}

	// Add resources
	for _, resource := range plan.Resources {
		if resource.Action == "import" {
			// Skip imports in main.tf - they'll be handled separately
			continue
		}

		content.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n",
			resource.ResourceType, resource.ResourceName))

		// Add configuration
		for key, value := range resource.Configuration {
			if key == "id" {
				continue // Skip ID for new resources
			}
			content.WriteString(fmt.Sprintf("  %s = %s\n", key, tre.formatValue(value)))
		}

		// Add dependencies
		if len(resource.Dependencies) > 0 {
			content.WriteString("  depends_on = [\n")
			for _, dep := range resource.Dependencies {
				content.WriteString(fmt.Sprintf("    %s,\n", dep))
			}
			content.WriteString("  ]\n")
		}

		content.WriteString("}\n\n")
	}

	return content.String()
}

// generateVariablesTf generates the variables.tf file content
func (tre *TerraformRemediationEngine) generateVariablesTf(plan *TerraformRemediationPlan) string {
	var content strings.Builder

	content.WriteString("# Variables for Terraform remediation\n\n")

	// Add common variables
	content.WriteString("variable \"environment\" {\n")
	content.WriteString("  description = \"Environment name\"\n")
	content.WriteString("  type        = string\n")
	content.WriteString("  default     = \"remediation\"\n")
	content.WriteString("}\n\n")

	content.WriteString("variable \"region\" {\n")
	content.WriteString("  description = \"AWS region\"\n")
	content.WriteString("  type        = string\n")
	content.WriteString("}\n\n")

	return content.String()
}

// generateOutputsTf generates the outputs.tf file content
func (tre *TerraformRemediationEngine) generateOutputsTf(plan *TerraformRemediationPlan) string {
	var content strings.Builder

	content.WriteString("# Outputs for Terraform remediation\n\n")

	// Add outputs for created/updated resources
	for _, resource := range plan.Resources {
		if resource.Action == "create" || resource.Action == "update" {
			content.WriteString(fmt.Sprintf("output \"%s_id\" {\n", resource.ResourceName))
			content.WriteString(fmt.Sprintf("  description = \"ID of %s\"\n", resource.ResourceName))
			content.WriteString(fmt.Sprintf("  value       = %s.%s.id\n",
				resource.ResourceType, resource.ResourceName))
			content.WriteString("}\n\n")
		}
	}

	return content.String()
}

// getUniqueProviders gets unique providers from the plan
func (tre *TerraformRemediationEngine) getUniqueProviders(plan *TerraformRemediationPlan) []string {
	providers := make(map[string]bool)
	for _, resource := range plan.Resources {
		providers[resource.Provider] = true
	}

	var uniqueProviders []string
	for provider := range providers {
		uniqueProviders = append(uniqueProviders, provider)
	}

	return uniqueProviders
}

// formatValue formats a value for Terraform configuration
func (tre *TerraformRemediationEngine) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case int, int32, int64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case []interface{}:
		var elements []string
		for _, element := range v {
			elements = append(elements, tre.formatValue(element))
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	case map[string]interface{}:
		var pairs []string
		for key, val := range v {
			pairs = append(pairs, fmt.Sprintf("%s = %s", key, tre.formatValue(val)))
		}
		return fmt.Sprintf("{\n    %s\n  }", strings.Join(pairs, "\n    "))
	default:
		return fmt.Sprintf("\"%v\"", v)
	}
}

// ValidatePlan validates the Terraform plan
func (tre *TerraformRemediationEngine) ValidatePlan(plan *TerraformRemediationPlan) (*TerraformValidationResult, error) {
	// Generate Terraform files
	if err := tre.GenerateTerraformFiles(plan); err != nil {
		return nil, fmt.Errorf("failed to generate Terraform files: %w", err)
	}

	// Initialize Terraform
	if err := tre.terraformInit(); err != nil {
		return nil, fmt.Errorf("failed to initialize Terraform: %w", err)
	}

	// Run terraform plan
	planOutput, planFile, err := tre.terraformPlan()
	if err != nil {
		return nil, fmt.Errorf("failed to run terraform plan: %w", err)
	}

	// Parse plan output
	validationResult := tre.parsePlanOutput(planOutput)

	// Store plan file path
	plan.PlanOutput = planOutput
	plan.PlanFile = planFile
	plan.Status = "validated"

	return validationResult, nil
}

// TerraformValidationResult represents the result of Terraform plan validation
type TerraformValidationResult struct {
	Valid         bool     `json:"valid"`
	Changes       int      `json:"changes"`
	Additions     int      `json:"additions"`
	Modifications int      `json:"modifications"`
	Destructions  int      `json:"destructions"`
	Warnings      []string `json:"warnings"`
	Errors        []string `json:"errors"`
	PlanOutput    string   `json:"plan_output"`
}

// terraformInit runs terraform init
func (tre *TerraformRemediationEngine) terraformInit() error {
	cmd := exec.Command(tre.terraformPath, "init")
	cmd.Dir = tre.workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// terraformPlan runs terraform plan
func (tre *TerraformRemediationEngine) terraformPlan() (string, string, error) {
	planFile := filepath.Join(tre.workingDir, "plan.tfplan")

	cmd := exec.Command(tre.terraformPath, "plan", "-out="+planFile)
	cmd.Dir = tre.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), "", fmt.Errorf("terraform plan failed: %w", err)
	}

	return string(output), planFile, nil
}

// parsePlanOutput parses terraform plan output
func (tre *TerraformRemediationEngine) parsePlanOutput(output string) *TerraformValidationResult {
	result := &TerraformValidationResult{
		Valid:      true,
		Warnings:   []string{},
		Errors:     []string{},
		PlanOutput: output,
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Count changes
		if strings.Contains(line, "Plan:") {
			if strings.Contains(line, "to add") {
				result.Additions++
			}
			if strings.Contains(line, "to change") {
				result.Modifications++
			}
			if strings.Contains(line, "to destroy") {
				result.Destructions++
			}
		}

		// Check for warnings
		if strings.Contains(line, "Warning:") {
			result.Warnings = append(result.Warnings, line)
		}

		// Check for errors
		if strings.Contains(line, "Error:") {
			result.Errors = append(result.Errors, line)
			result.Valid = false
		}
	}

	result.Changes = result.Additions + result.Modifications + result.Destructions

	return result
}

// ExecutePlan executes the Terraform plan
func (tre *TerraformRemediationEngine) ExecutePlan(plan *TerraformRemediationPlan) error {
	if plan.PlanFile == "" {
		return fmt.Errorf("no plan file available. Run ValidatePlan first")
	}

	// Execute terraform apply
	cmd := exec.Command(tre.terraformPath, "apply", plan.PlanFile)
	cmd.Dir = tre.workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	plan.Status = "executed"
	return nil
}

// GenerateImportCommands generates terraform import commands for unmanaged resources
func (tre *TerraformRemediationEngine) GenerateImportCommands(plan *TerraformRemediationPlan) []string {
	var commands []string

	for _, resource := range plan.Resources {
		if resource.Action == "import" {
			command := fmt.Sprintf("terraform import %s.%s %s",
				resource.ResourceType, resource.ResourceName, resource.ImportID)
			commands = append(commands, command)
		}
	}

	return commands
}

// RollbackPlan creates a rollback plan
func (tre *TerraformRemediationEngine) RollbackPlan(plan *TerraformRemediationPlan) (*TerraformRemediationPlan, error) {
	rollbackPlan := &TerraformRemediationPlan{
		ID:          fmt.Sprintf("rollback_%s", plan.ID),
		Description: "Rollback plan for " + plan.Description,
		Resources:   []TerraformRemediationResource{},
		CreatedAt:   time.Now(),
		Status:      "created",
	}

	// Reverse the actions
	for _, resource := range plan.Resources {
		rollbackResource := resource
		switch resource.Action {
		case "create":
			rollbackResource.Action = "delete"
		case "delete":
			rollbackResource.Action = "create"
		case "update":
			// For updates, we need to restore the previous state
			// This would require storing the previous state
			rollbackResource.Action = "update"
		case "import":
			rollbackResource.Action = "delete"
		}
		rollbackPlan.Resources = append(rollbackPlan.Resources, rollbackResource)
	}

	return rollbackPlan, nil
}
