package deletion

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/models"
)

// DependencyManager handles resource dependencies and deletion order
type DependencyManager struct {
	dependencies map[string][]string
	reverseDeps  map[string][]string
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager() *DependencyManager {
	return &DependencyManager{
		dependencies: make(map[string][]string),
		reverseDeps:  make(map[string][]string),
	}
}

// AddDependency adds a dependency relationship
func (dm *DependencyManager) AddDependency(resourceID, dependsOn string) {
	dm.dependencies[resourceID] = append(dm.dependencies[resourceID], dependsOn)
	dm.reverseDeps[dependsOn] = append(dm.reverseDeps[dependsOn], resourceID)
}

// GetDeletionOrder returns resources in proper deletion order
func (dm *DependencyManager) GetDeletionOrder(resources []models.Resource) ([]models.Resource, error) {
	// Build dependency graph
	dm.buildDependencyGraph(resources)

	// Check for circular dependencies
	if err := dm.checkCircularDependencies(); err != nil {
		return nil, err
	}

	// Sort resources by dependency order
	return dm.topologicalSort(resources), nil
}

// buildDependencyGraph builds the dependency graph for resources
func (dm *DependencyManager) buildDependencyGraph(resources []models.Resource) {
	// Clear existing dependencies
	dm.dependencies = make(map[string][]string)
	dm.reverseDeps = make(map[string][]string)

	// Build dependencies based on resource types and relationships
	for _, resource := range resources {
		deps := dm.getResourceDependencies(resource, resources)
		if len(deps) > 0 {
			dm.dependencies[resource.ID] = deps
			for _, dep := range deps {
				dm.reverseDeps[dep] = append(dm.reverseDeps[dep], resource.ID)
			}
		}
	}
}

// getResourceDependencies returns dependencies for a specific resource
func (dm *DependencyManager) getResourceDependencies(resource models.Resource, allResources []models.Resource) []string {
	// Use the generic dependency system for all resource types
	return dm.getGenericDependencies(resource, allResources)
}

// getEC2Dependencies returns dependencies for EC2 instances
func (dm *DependencyManager) getEC2Dependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// EC2 instances depend on:
	// - Security Groups
	// - Network Interfaces
	// - EBS Volumes
	// - IAM Roles

	for _, other := range allResources {
		if other.ID == resource.ID {
			continue
		}

		// Check for security groups
		if other.Type == "security_group" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for network interfaces
		if other.Type == "network_interface" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for EBS volumes
		if other.Type == "ebs_volume" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}
	}

	return deps
}

// getRDSDependencies returns dependencies for RDS instances
func (dm *DependencyManager) getRDSDependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// RDS instances depend on:
	// - Subnet Groups
	// - Security Groups
	// - Parameter Groups
	// - Option Groups

	for _, other := range allResources {
		if other.ID == resource.ID {
			continue
		}

		// Check for subnet groups
		if other.Type == "db_subnet_group" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for security groups
		if other.Type == "security_group" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}
	}

	return deps
}

// getEKSDependencies returns dependencies for EKS clusters
func (dm *DependencyManager) getEKSDependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// EKS clusters depend on:
	// - VPC
	// - Subnets
	// - Security Groups
	// - IAM Roles

	for _, other := range allResources {
		if other.ID == resource.ID {
			continue
		}

		// Check for VPC
		if other.Type == "vpc" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for subnets
		if other.Type == "subnet" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}
	}

	return deps
}

// getGenericDependencies returns dependencies for any resource type based on configuration
func (dm *DependencyManager) getGenericDependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// Get dependency configuration for this resource type
	dependencyConfig := dm.getResourceDependencyConfig(resource.Type)
	if dependencyConfig == nil {
		return deps
	}

	// Check for each dependency type
	for _, depType := range dependencyConfig.DependencyTypes {
		for _, other := range allResources {
			if other.ID == resource.ID {
				continue
			}

			if other.Type == depType && dm.hasResourceReference(resource, other) {
				deps = append(deps, other.ID)
			}
		}
	}

	return deps
}

// ResourceDependencyConfig defines dependency types for a resource
type ResourceDependencyConfig struct {
	DependencyTypes []string
	ReferenceMethod string // "tags", "naming", "both"
}

// getResourceDependencyConfig returns dependency configuration for a resource type
func (dm *DependencyManager) getResourceDependencyConfig(resourceType string) *ResourceDependencyConfig {
	configs := map[string]*ResourceDependencyConfig{
		"eks_cluster": {
			DependencyTypes: []string{"vpc", "subnet", "security_group", "iam_role"},
			ReferenceMethod: "both",
		},
		"ecs_cluster": {
			DependencyTypes: []string{"vpc", "subnet", "security_group"},
			ReferenceMethod: "both",
		},
		"rds_instance": {
			DependencyTypes: []string{"vpc", "subnet", "security_group", "db_subnet_group"},
			ReferenceMethod: "both",
		},
		"elasticache_cluster": {
			DependencyTypes: []string{"vpc", "subnet", "security_group", "elasticache_subnet_group"},
			ReferenceMethod: "both",
		},
		"load_balancer": {
			DependencyTypes: []string{"vpc", "subnet", "security_group", "target_group"},
			ReferenceMethod: "both",
		},
		"ec2_instance": {
			DependencyTypes: []string{"security_group", "network_interface", "ebs_volume", "iam_role"},
			ReferenceMethod: "both",
		},
		"vpc": {
			DependencyTypes: []string{"nat_gateway", "internet_gateway", "route_table"},
			ReferenceMethod: "both",
		},
	}

	return configs[resourceType]
}

// hasResourceReference checks if one resource references another using the configured method
func (dm *DependencyManager) hasResourceReference(resource, other models.Resource) bool {
	config := dm.getResourceDependencyConfig(resource.Type)
	if config == nil {
		return dm.hasTagReference(resource, other)
	}

	switch config.ReferenceMethod {
	case "tags":
		return dm.hasTagReference(resource, other)
	case "naming":
		return dm.hasNamingReference(resource, other)
	case "both":
		return dm.hasTagReference(resource, other) || dm.hasNamingReference(resource, other)
	default:
		return dm.hasTagReference(resource, other)
	}
}

// hasNamingReference checks if resources are related through naming conventions
func (dm *DependencyManager) hasNamingReference(resource, other models.Resource) bool {
	resourceName := strings.ToLower(resource.Name)
	otherName := strings.ToLower(other.Name)

	// Check for common naming patterns
	patterns := []string{
		resourceName,
		strings.ReplaceAll(resourceName, "-", ""),
		strings.ReplaceAll(resourceName, "_", ""),
	}

	for _, pattern := range patterns {
		if strings.Contains(otherName, pattern) || strings.Contains(pattern, otherName) {
			return true
		}
	}

	// Check for environment-based naming
	if strings.Contains(resourceName, "prod") && strings.Contains(otherName, "prod") {
		return true
	}
	if strings.Contains(resourceName, "dev") && strings.Contains(otherName, "dev") {
		return true
	}
	if strings.Contains(resourceName, "staging") && strings.Contains(otherName, "staging") {
		return true
	}

	return false
}

// getElastiCacheDependencies returns dependencies for ElastiCache clusters
func (dm *DependencyManager) getElastiCacheDependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// ElastiCache clusters depend on:
	// - Subnet Groups
	// - Security Groups
	// - Parameter Groups

	for _, other := range allResources {
		if other.ID == resource.ID {
			continue
		}

		// Check for subnet groups
		if other.Type == "elasticache_subnet_group" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for security groups
		if other.Type == "security_group" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}
	}

	return deps
}

// getLoadBalancerDependencies returns dependencies for load balancers
func (dm *DependencyManager) getLoadBalancerDependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// Load balancers depend on:
	// - VPC
	// - Subnets
	// - Security Groups
	// - Target Groups

	for _, other := range allResources {
		if other.ID == resource.ID {
			continue
		}

		// Check for VPC
		if other.Type == "vpc" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for target groups
		if other.Type == "target_group" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}
	}

	return deps
}

// getVirtualMachineDependencies returns dependencies for Azure VMs
func (dm *DependencyManager) getVirtualMachineDependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// Azure VMs depend on:
	// - Network Interfaces
	// - Disks
	// - Virtual Networks

	for _, other := range allResources {
		if other.ID == resource.ID {
			continue
		}

		// Check for network interfaces
		if other.Type == "network_interface" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for disks
		if other.Type == "disk" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}
	}

	return deps
}

// getKubernetesDependencies returns dependencies for Kubernetes clusters
func (dm *DependencyManager) getKubernetesDependencies(resource models.Resource, allResources []models.Resource) []string {
	var deps []string

	// Kubernetes clusters depend on:
	// - Virtual Networks
	// - Subnets
	// - Network Security Groups

	for _, other := range allResources {
		if other.ID == resource.ID {
			continue
		}

		// Check for virtual networks
		if other.Type == "virtual_network" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}

		// Check for subnets
		if other.Type == "subnet" && dm.hasTagReference(resource, other) {
			deps = append(deps, other.ID)
		}
	}

	return deps
}

// hasTagReference checks if one resource references another through tags
func (dm *DependencyManager) hasTagReference(resource, other models.Resource) bool {
	// Check if resource tags contain references to other resource
	for key, value := range resource.Tags {
		if strings.Contains(strings.ToLower(key), other.Type) ||
			strings.Contains(strings.ToLower(value), other.Name) ||
			strings.Contains(strings.ToLower(value), other.ID) {
			return true
		}
	}
	return false
}

// checkCircularDependencies checks for circular dependencies
func (dm *DependencyManager) checkCircularDependencies() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for resourceID := range dm.dependencies {
		if !visited[resourceID] {
			if dm.hasCycle(resourceID, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving resource %s", resourceID)
			}
		}
	}

	return nil
}

// hasCycle checks for cycles in the dependency graph using DFS
func (dm *DependencyManager) hasCycle(resourceID string, visited, recStack map[string]bool) bool {
	visited[resourceID] = true
	recStack[resourceID] = true

	for _, dep := range dm.dependencies[resourceID] {
		if !visited[dep] {
			if dm.hasCycle(dep, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[resourceID] = false
	return false
}

// topologicalSort performs topological sort on resources
func (dm *DependencyManager) topologicalSort(resources []models.Resource) []models.Resource {
	// Create a map for quick lookup
	resourceMap := make(map[string]models.Resource)
	for _, resource := range resources {
		resourceMap[resource.ID] = resource
	}

	// Calculate in-degrees
	inDegree := make(map[string]int)
	for _, resource := range resources {
		inDegree[resource.ID] = 0
	}

	for _, deps := range dm.dependencies {
		for _, dep := range deps {
			if _, exists := resourceMap[dep]; exists {
				inDegree[dep]++
			}
		}
	}

	// Find resources with no dependencies
	var queue []string
	for resourceID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, resourceID)
		}
	}

	// Process queue
	var result []models.Resource
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if resource, exists := resourceMap[current]; exists {
			result = append(result, resource)
		}

		// Reduce in-degree for dependent resources
		for _, dep := range dm.reverseDeps[current] {
			if _, exists := resourceMap[dep]; exists {
				inDegree[dep]--
				if inDegree[dep] == 0 {
					queue = append(queue, dep)
				}
			}
		}
	}

	// Add any remaining resources (should be none if no cycles)
	for _, resource := range resources {
		found := false
		for _, r := range result {
			if r.ID == resource.ID {
				found = true
				break
			}
		}
		if !found {
			result = append(result, resource)
		}
	}

	return result
}

// GetDependencyGraph returns a string representation of the dependency graph
func (dm *DependencyManager) GetDependencyGraph() string {
	var result strings.Builder
	result.WriteString("Dependency Graph:\n")

	for resourceID, deps := range dm.dependencies {
		result.WriteString(fmt.Sprintf("  %s depends on:\n", resourceID))
		for _, dep := range deps {
			result.WriteString(fmt.Sprintf("    - %s\n", dep))
		}
	}

	return result.String()
}
