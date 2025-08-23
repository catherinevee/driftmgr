package visualization

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// TerraformStateParser parses Terraform state files
type TerraformStateParser struct {
	statePath string
}

// NewTerraformStateParser creates a new state parser
func NewTerraformStateParser(statePath string) *TerraformStateParser {
	return &TerraformStateParser{
		statePath: statePath,
	}
}

// ParseStateFile parses a Terraform state file and extracts diagram data
func (p *TerraformStateParser) ParseStateFile() (*models.DiagramData, error) {
	// Read state file
	data, err := os.ReadFile(p.statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse JSON
	var stateFile models.StateFile
	if err := json.Unmarshal(data, &stateFile); err != nil {
		return nil, fmt.Errorf("failed to parse state file JSON: %w", err)
	}

	// Extract diagram data
	diagramData := &models.DiagramData{
		Resources:    []models.Resource{},
		DataSources:  []models.DataSource{},
		Dependencies: []models.Dependency{},
		Modules:      []models.Module{},
		Path:         p.statePath,
		ParsedAt:     time.Now(),
	}

	// Process resources
	for _, tfResource := range stateFile.Resources {
		if len(tfResource.Instances) == 0 {
			continue
		}

		instance := tfResource.Instances[0]

		if tfResource.Mode == "data" {
			// Handle data sources
			dataSource := p.extractDataSource(tfResource, instance)
			diagramData.DataSources = append(diagramData.DataSources, dataSource)
		} else {
			// Handle managed resources
			resource := p.extractResource(tfResource, instance)
			diagramData.Resources = append(diagramData.Resources, resource)
		}
	}

	// Extract dependencies
	dependencies := p.extractDependencies(stateFile.Resources)
	diagramData.Dependencies = dependencies

	// Extract modules
	modules := p.extractModules(stateFile.Resources)
	diagramData.Modules = modules

	return diagramData, nil
}

// extractResource extracts a resource from Terraform state
func (p *TerraformStateParser) extractResource(tfResource models.TerraformResource, instance models.TerraformResourceInstance) models.Resource {
	attributes := instance.Attributes

	// Extract basic information
	id := p.getStringAttribute(attributes, "id", "")
	name := p.getStringAttribute(attributes, "name", tfResource.Name)
	resourceType := tfResource.Type

	// Extract provider information
	provider := "unknown"
	if strings.Contains(resourceType, "aws_") {
		provider = "aws"
	} else if strings.Contains(resourceType, "azure_") {
		provider = "azure"
	} else if strings.Contains(resourceType, "gcp_") {
		provider = "gcp"
	}

	// Extract region
	region := p.getStringAttribute(attributes, "region", "")
	if region == "" {
		region = p.getStringAttribute(attributes, "location", "")
	}

	// Extract tags
	tags := p.extractTags(attributes)

	// Extract state
	state := "active"
	if p.getStringAttribute(attributes, "lifecycle", "") == "destroy" {
		state = "destroying"
	}

	return models.Resource{
		ID:       id,
		Name:     name,
		Type:     resourceType,
		Provider: provider,
		Region:   region,
		Tags:     tags,
		State:    state,
	}
}

// extractDataSource extracts a data source from Terraform state
func (p *TerraformStateParser) extractDataSource(tfResource models.TerraformResource, instance models.TerraformResourceInstance) models.DataSource {
	attributes := instance.Attributes

	// Extract basic information
	id := p.getStringAttribute(attributes, "id", "")
	name := p.getStringAttribute(attributes, "name", tfResource.Name)
	dataSourceType := tfResource.Type

	// Extract provider information
	provider := "unknown"
	if strings.Contains(dataSourceType, "aws_") {
		provider = "aws"
	} else if strings.Contains(dataSourceType, "azure_") {
		provider = "azure"
	} else if strings.Contains(dataSourceType, "gcp_") {
		provider = "gcp"
	}

	// Extract region
	region := p.getStringAttribute(attributes, "region", "")
	if region == "" {
		region = p.getStringAttribute(attributes, "location", "")
	}

	// Extract configuration
	config := p.extractConfig(attributes)

	return models.DataSource{
		ID:       id,
		Name:     name,
		Type:     dataSourceType,
		Provider: provider,
		Region:   region,
		Config:   config,
	}
}

// extractDependencies extracts dependencies between resources
func (p *TerraformStateParser) extractDependencies(resources []models.TerraformResource) []models.Dependency {
	var dependencies []models.Dependency

	for _, resource := range resources {
		if len(resource.Instances) == 0 {
			continue
		}

		instance := resource.Instances[0]
		attributes := instance.Attributes

		// Extract dependencies from various attribute patterns
		deps := p.findDependenciesInAttributes(attributes, resource.Name)
		dependencies = append(dependencies, deps...)
	}

	return dependencies
}

// findDependenciesInAttributes finds dependencies in resource attributes
func (p *TerraformStateParser) findDependenciesInAttributes(attributes map[string]interface{}, resourceName string) []models.Dependency {
	var dependencies []models.Dependency

	// Common dependency patterns
	dependencyPatterns := []string{
		"subnet_id", "vpc_id", "security_group_ids", "route_table_id",
		"target_group_arns", "load_balancer_arn", "cluster_arn",
		"bucket", "table_name", "function_name", "role_arn",
		"resource_group_name", "virtual_network_name", "storage_account_name",
		"network", "subnetwork", "service_account", "project",
	}

	for _, pattern := range dependencyPatterns {
		if value, exists := attributes[pattern]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				// Extract resource ID from the reference
				resourceID := p.extractResourceIDFromReference(strValue)
				if resourceID != "" {
					dependencies = append(dependencies, models.Dependency{
						From: resourceName,
						To:   resourceID,
						Type: "depends_on",
					})
				}
			}
		}
	}

	return dependencies
}

// extractResourceIDFromReference extracts resource ID from Terraform reference
func (p *TerraformStateParser) extractResourceIDFromReference(ref string) string {
	// Handle Terraform references like:
	// - aws_subnet.main.id
	// - azurerm_resource_group.main.name
	// - google_compute_network.main.id

	parts := strings.Split(ref, ".")
	if len(parts) >= 2 {
		// Return the resource name (second part)
		return parts[1]
	}

	return ""
}

// extractModules extracts module information
func (p *TerraformStateParser) extractModules(resources []models.TerraformResource) []models.Module {
	modules := make(map[string]*models.Module)

	for _, resource := range resources {
		// Extract module information from resource path
		moduleName := p.extractModuleName(resource.Name)

		if moduleName != "" {
			if module, exists := modules[moduleName]; exists {
				module.Resources = append(module.Resources, resource.Name)
			} else {
				modules[moduleName] = &models.Module{
					Name:      moduleName,
					Source:    "./modules/" + moduleName,
					Version:   "1.0.0",
					Resources: []string{resource.Name},
				}
			}
		}
	}

	// Convert map to slice
	var result []models.Module
	for _, module := range modules {
		result = append(result, *module)
	}

	return result
}

// extractModuleName extracts module name from resource name
func (p *TerraformStateParser) extractModuleName(resourceName string) string {
	// Handle module resource names like:
	// - module.vpc.aws_vpc.main
	// - module.database.aws_rds_instance.main

	parts := strings.Split(resourceName, ".")
	if len(parts) >= 2 && parts[0] == "module" {
		return parts[1]
	}

	return ""
}

// extractTags extracts tags from resource attributes
func (p *TerraformStateParser) extractTags(attributes map[string]interface{}) map[string]string {
	tags := make(map[string]string)

	// Handle different tag formats
	if tagsMap, exists := attributes["tags"]; exists {
		if tagsInterface, ok := tagsMap.(map[string]interface{}); ok {
			for key, value := range tagsInterface {
				if strValue, ok := value.(string); ok {
					tags[key] = strValue
				}
			}
		}
	}

	// Handle individual tag attributes
	for key, value := range attributes {
		if strings.HasPrefix(key, "tag_") {
			if strValue, ok := value.(string); ok {
				tagKey := strings.TrimPrefix(key, "tag_")
				tags[tagKey] = strValue
			}
		}
	}

	return tags
}

// extractConfig extracts configuration from data source attributes
func (p *TerraformStateParser) extractConfig(attributes map[string]interface{}) map[string]string {
	config := make(map[string]string)

	for key, value := range attributes {
		if strValue, ok := value.(string); ok {
			config[key] = strValue
		}
	}

	return config
}

// getStringAttribute safely extracts a string attribute
func (p *TerraformStateParser) getStringAttribute(attributes map[string]interface{}, key, defaultValue string) string {
	if value, exists := attributes[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

// ParseTerraformFiles parses Terraform configuration files to extract additional information
func (p *TerraformStateParser) ParseTerraformFiles(terraformPath string) error {
	if terraformPath == "" {
		return nil
	}

	// Look for Terraform files in the specified path
	tfFiles := []string{"main.tf", "variables.tf", "outputs.tf", "providers.tf"}

	for _, filename := range tfFiles {
		filepath := filepath.Join(terraformPath, filename)
		if _, err := os.Stat(filepath); err == nil {
			// File exists, could be parsed for additional context
			// For now, we'll just note that it exists
		}
	}

	return nil
}
