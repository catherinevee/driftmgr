package state

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// StateOptimizer optimizes Terraform state files
type StateOptimizer struct {
	options *StateOptimizationOptions
}

// StateOptimizationOptions defines optimization options
type StateOptimizationOptions struct {
	RemoveUnusedResources bool
	RemoveEmptyModules    bool
	RemoveOrphanedData    bool
	CompactAttributes     bool
	RemoveDeprecated      bool
	DryRun                bool
	BackupBeforeOptimize  bool
}

// OptimizationResult represents the result of state optimization
type OptimizationResult struct {
	OriginalResourceCount  int
	OptimizedResourceCount int
	RemovedResources       []string
	RemovedModules         []string
	RemovedDataSources     []string
	SizeReduction          int64 // bytes
	OptimizationTime       time.Duration
	Warnings               []string
	Errors                 []string
}

// NewStateOptimizer creates a new state optimizer
func NewStateOptimizer(options *StateOptimizationOptions) *StateOptimizer {
	if options == nil {
		options = &StateOptimizationOptions{
			RemoveUnusedResources: true,
			RemoveEmptyModules:    true,
			RemoveOrphanedData:    true,
			CompactAttributes:     true,
			RemoveDeprecated:      false,
			DryRun:                false,
			BackupBeforeOptimize:  true,
		}
	}

	return &StateOptimizer{
		options: options,
	}
}

// OptimizeState optimizes a Terraform state file
func (so *StateOptimizer) OptimizeState(stateFile *models.StateFile) (*models.StateFile, *OptimizationResult, error) {
	startTime := time.Now()
	result := &OptimizationResult{
		OriginalResourceCount: len(stateFile.Resources),
		RemovedResources:      []string{},
		RemovedModules:        []string{},
		RemovedDataSources:    []string{},
		Warnings:              []string{},
		Errors:                []string{},
	}

	// Create a copy of the state file for optimization
	optimizedState := so.copyStateFile(stateFile)

	// Remove unused resources
	if so.options.RemoveUnusedResources {
		so.removeUnusedResources(optimizedState, result)
	}

	// Remove empty modules
	if so.options.RemoveEmptyModules {
		so.removeEmptyModules(optimizedState, result)
	}

	// Remove orphaned data sources
	if so.options.RemoveOrphanedData {
		so.removeOrphanedDataSources(optimizedState, result)
	}

	// Compact attributes
	if so.options.CompactAttributes {
		so.compactAttributes(optimizedState, result)
	}

	// Remove deprecated resources
	if so.options.RemoveDeprecated {
		so.removeDeprecatedResources(optimizedState, result)
	}

	result.OptimizedResourceCount = len(optimizedState.Resources)
	result.OptimizationTime = time.Since(startTime)

	return optimizedState, result, nil
}

// copyStateFile creates a deep copy of a state file
func (so *StateOptimizer) copyStateFile(original *models.StateFile) *models.StateFile {
	copied := &models.StateFile{
		Path:                original.Path,
		Version:             original.Version,
		TerraformVersion:    original.TerraformVersion,
		Serial:              original.Serial,
		Lineage:             original.Lineage,
		Outputs:             original.Outputs,
		Resources:           make([]models.TerraformResource, len(original.Resources)),
		ManagedByTerragrunt: original.ManagedByTerragrunt,
		TerragruntConfig:    original.TerragruntConfig,
	}

	for i, resource := range original.Resources {
		copied.Resources[i] = so.copyTerraformResource(resource)
	}

	return copied
}

// copyTerraformResource creates a deep copy of a Terraform resource
func (so *StateOptimizer) copyTerraformResource(original models.TerraformResource) models.TerraformResource {
	copied := models.TerraformResource{
		Name:      original.Name,
		Type:      original.Type,
		Mode:      original.Mode,
		Provider:  original.Provider,
		Instances: make([]models.TerraformResourceInstance, len(original.Instances)),
	}

	for i, instance := range original.Instances {
		copied.Instances[i] = so.copyTerraformInstance(instance)
	}

	return copied
}

// copyTerraformInstance creates a deep copy of a Terraform instance
func (so *StateOptimizer) copyTerraformInstance(original models.TerraformResourceInstance) models.TerraformResourceInstance {
	copied := models.TerraformResourceInstance{
		SchemaVersion: original.SchemaVersion,
		Private:       original.Private,
		Attributes:    make(map[string]interface{}),
	}

	for key, value := range original.Attributes {
		copied.Attributes[key] = value
	}

	return copied
}

// removeUnusedResources removes resources that are not referenced by other resources
func (so *StateOptimizer) removeUnusedResources(stateFile *models.StateFile, result *OptimizationResult) {
	// Build dependency graph
	dependencies := so.buildDependencyGraph(stateFile.Resources)

	// Find unused resources (resources with no incoming dependencies)
	var unusedResources []int
	for i, resource := range stateFile.Resources {
		if len(dependencies[i]) == 0 && !so.isRootResource(resource) {
			unusedResources = append(unusedResources, i)
		}
	}

	// Remove unused resources (in reverse order to maintain indices)
	sort.Sort(sort.Reverse(sort.IntSlice(unusedResources)))
	for _, index := range unusedResources {
		if index < len(stateFile.Resources) {
			resourceName := stateFile.Resources[index].Name
			result.RemovedResources = append(result.RemovedResources, resourceName)

			if !so.options.DryRun {
				stateFile.Resources = append(stateFile.Resources[:index], stateFile.Resources[index+1:]...)
			}
		}
	}
}

// buildDependencyGraph builds a dependency graph for resources
func (so *StateOptimizer) buildDependencyGraph(resources []models.TerraformResource) map[int][]int {
	dependencies := make(map[int][]int)

	// Initialize empty dependency lists
	for i := range resources {
		dependencies[i] = []int{}
	}

	// Find dependencies between resources
	for i, resource := range resources {
		for j, otherResource := range resources {
			if i != j && so.hasDependency(resource, otherResource) {
				dependencies[j] = append(dependencies[j], i)
			}
		}
	}

	return dependencies
}

// hasDependency checks if one resource depends on another
func (so *StateOptimizer) hasDependency(resource, other models.TerraformResource) bool {
	// Check if resource references other resource in its attributes
	for _, instance := range resource.Instances {
		for _, value := range instance.Attributes {
			if strValue, ok := value.(string); ok {
				if strings.Contains(strValue, other.Name) || strings.Contains(strValue, other.Type) {
					return true
				}
			}
		}
	}
	return false
}

// isRootResource checks if a resource is a root resource (should not be removed)
func (so *StateOptimizer) isRootResource(resource models.TerraformResource) bool {
	// Root resources are typically:
	// - VPCs, networks, or other foundational infrastructure
	// - Resources with specific tags indicating they are root
	// - Resources that are explicitly marked as root

	rootTypes := []string{
		"aws_vpc", "aws_default_vpc", "aws_main_route_table_association",
		"azurerm_virtual_network", "azurerm_resource_group",
		"google_compute_network", "google_compute_subnetwork",
	}

	for _, rootType := range rootTypes {
		if resource.Type == rootType {
			return true
		}
	}

	// Check for root tags
	for _, instance := range resource.Instances {
		if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
			if rootTag, exists := tags["root"]; exists {
				if strRoot, ok := rootTag.(string); ok && strRoot == "true" {
					return true
				}
			}
		}
	}

	return false
}

// removeEmptyModules removes modules that contain no resources
func (so *StateOptimizer) removeEmptyModules(stateFile *models.StateFile, result *OptimizationResult) {
	// Group resources by module
	moduleResources := make(map[string][]int)

	for i, resource := range stateFile.Resources {
		moduleName := so.extractModuleName(resource.Name)
		if moduleName != "" {
			moduleResources[moduleName] = append(moduleResources[moduleName], i)
		}
	}

	// Find empty modules
	var emptyModuleIndices []int
	for moduleName, indices := range moduleResources {
		if len(indices) == 0 {
			result.RemovedModules = append(result.RemovedModules, moduleName)
			emptyModuleIndices = append(emptyModuleIndices, indices...)
		}
	}

	// Remove resources from empty modules
	if !so.options.DryRun && len(emptyModuleIndices) > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(emptyModuleIndices)))
		for _, index := range emptyModuleIndices {
			if index < len(stateFile.Resources) {
				stateFile.Resources = append(stateFile.Resources[:index], stateFile.Resources[index+1:]...)
			}
		}
	}
}

// extractModuleName extracts module name from resource name
func (so *StateOptimizer) extractModuleName(resourceName string) string {
	parts := strings.Split(resourceName, ".")
	if len(parts) >= 2 && parts[0] == "module" {
		return parts[1]
	}
	return ""
}

// removeOrphanedDataSources removes data sources that are not referenced
func (so *StateOptimizer) removeOrphanedDataSources(stateFile *models.StateFile, result *OptimizationResult) {
	var orphanedDataIndices []int

	for i, resource := range stateFile.Resources {
		if resource.Mode == "data" {
			// Check if this data source is referenced by any managed resource
			isReferenced := false
			for _, otherResource := range stateFile.Resources {
				if otherResource.Mode != "data" && so.referencesDataSource(otherResource, resource) {
					isReferenced = true
					break
				}
			}

			if !isReferenced {
				orphanedDataIndices = append(orphanedDataIndices, i)
				result.RemovedDataSources = append(result.RemovedDataSources, resource.Name)
			}
		}
	}

	// Remove orphaned data sources
	if !so.options.DryRun && len(orphanedDataIndices) > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(orphanedDataIndices)))
		for _, index := range orphanedDataIndices {
			if index < len(stateFile.Resources) {
				stateFile.Resources = append(stateFile.Resources[:index], stateFile.Resources[index+1:]...)
			}
		}
	}
}

// referencesDataSource checks if a resource references a data source
func (so *StateOptimizer) referencesDataSource(resource, dataSource models.TerraformResource) bool {
	dataSourceRef := fmt.Sprintf("data.%s.%s", dataSource.Type, dataSource.Name)

	for _, instance := range resource.Instances {
		for _, value := range instance.Attributes {
			if strValue, ok := value.(string); ok {
				if strings.Contains(strValue, dataSourceRef) {
					return true
				}
			}
		}
	}
	return false
}

// compactAttributes removes unnecessary attributes to reduce state file size
func (so *StateOptimizer) compactAttributes(stateFile *models.StateFile, result *OptimizationResult) {
	// Attributes that can be safely removed for optimization
	removableAttributes := []string{
		"timeouts",
		"lifecycle",
		"depends_on",
		"provider",
		"terraform_meta",
	}

	for i := range stateFile.Resources {
		for j := range stateFile.Resources[i].Instances {
			instance := &stateFile.Resources[i].Instances[j]

			for _, attr := range removableAttributes {
				if _, exists := instance.Attributes[attr]; exists {
					delete(instance.Attributes, attr)
				}
			}
		}
	}
}

// removeDeprecatedResources removes deprecated resource types
func (so *StateOptimizer) removeDeprecatedResources(stateFile *models.StateFile, result *OptimizationResult) {
	deprecatedTypes := []string{
		"aws_autoscaling_policy",                // replaced by aws_autoscaling_policy_v2
		"aws_autoscaling_schedule",              // replaced by aws_autoscaling_schedule_v2
		"aws_cloudformation_stack_set_instance", // replaced by aws_cloudformation_stack_set_instance_v2
	}

	var deprecatedIndices []int

	for i, resource := range stateFile.Resources {
		for _, deprecatedType := range deprecatedTypes {
			if resource.Type == deprecatedType {
				deprecatedIndices = append(deprecatedIndices, i)
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Deprecated resource type found: %s", resource.Type))
				break
			}
		}
	}

	// Remove deprecated resources
	if !so.options.DryRun && len(deprecatedIndices) > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(deprecatedIndices)))
		for _, index := range deprecatedIndices {
			if index < len(stateFile.Resources) {
				result.RemovedResources = append(result.RemovedResources,
					stateFile.Resources[index].Name)
				stateFile.Resources = append(stateFile.Resources[:index],
					stateFile.Resources[index+1:]...)
			}
		}
	}
}

// AnalyzeStateOptimization analyzes potential optimizations without applying them
func (so *StateOptimizer) AnalyzeStateOptimization(stateFile *models.StateFile) (*OptimizationResult, error) {
	// Create a temporary optimizer with dry run enabled
	tempOptions := *so.options
	tempOptions.DryRun = true

	tempOptimizer := NewStateOptimizer(&tempOptions)

	_, result, err := tempOptimizer.OptimizeState(stateFile)
	return result, err
}

// GetOptimizationRecommendations provides recommendations for state optimization
func (so *StateOptimizer) GetOptimizationRecommendations(stateFile *models.StateFile) []string {
	var recommendations []string

	// Count resource types
	resourceTypeCount := make(map[string]int)
	for _, resource := range stateFile.Resources {
		resourceTypeCount[resource.Type]++
	}

	// Check for potential optimizations
	if len(stateFile.Resources) > 1000 {
		recommendations = append(recommendations,
			"Large state file detected (>1000 resources). Consider splitting into multiple state files.")
	}

	// Check for unused resource types
	for resourceType, count := range resourceTypeCount {
		if count == 1 {
			recommendations = append(recommendations,
				fmt.Sprintf("Single instance of %s found. Consider if this resource is necessary.", resourceType))
		}
	}

	// Check for data sources
	dataSourceCount := 0
	for _, resource := range stateFile.Resources {
		if resource.Mode == "data" {
			dataSourceCount++
		}
	}

	if dataSourceCount > len(stateFile.Resources)/2 {
		recommendations = append(recommendations,
			"High ratio of data sources to managed resources. Consider if all data sources are necessary.")
	}

	return recommendations
}
