package deletion

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// DependencyValidator validates resource dependencies before deletion
type DependencyValidator struct {
	dependencies map[string][]string // resource ID -> dependent resource IDs
	resourceMap  map[string]*models.Resource
	provider     string
}

// NewDependencyValidator creates a new dependency validator
func NewDependencyValidator(provider string) *DependencyValidator {
	return &DependencyValidator{
		dependencies: make(map[string][]string),
		resourceMap:  make(map[string]*models.Resource),
		provider:     provider,
	}
}

// ValidateAndOrderDeletion validates dependencies and returns resources in safe deletion order
func (dv *DependencyValidator) ValidateAndOrderDeletion(ctx context.Context, resources []models.Resource) ([]models.Resource, error) {
	// Build resource map
	for i := range resources {
		dv.resourceMap[resources[i].ID] = &resources[i]
	}

	// Build dependency graph
	if err := dv.buildDependencyGraph(ctx, resources); err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Check for circular dependencies
	if cycles := dv.detectCycles(); len(cycles) > 0 {
		return nil, fmt.Errorf("circular dependencies detected: %v", cycles)
	}

	// Perform topological sort for deletion order
	orderedResources, err := dv.topologicalSort(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to order resources for deletion: %w", err)
	}

	// Validate that all dependencies can be safely deleted
	if err := dv.validateDeletionSafety(orderedResources); err != nil {
		return nil, err
	}

	return orderedResources, nil
}

// buildDependencyGraph builds a dependency graph for the resources
func (dv *DependencyValidator) buildDependencyGraph(ctx context.Context, resources []models.Resource) error {
	for _, resource := range resources {
		deps := dv.findDependencies(resource)
		if len(deps) > 0 {
			dv.dependencies[resource.ID] = deps
		}
	}
	return nil
}

// findDependencies finds dependencies for a specific resource
func (dv *DependencyValidator) findDependencies(resource models.Resource) []string {
	var deps []string

	switch dv.provider {
	case "aws":
		deps = dv.findAWSDependencies(resource)
	case "azure":
		deps = dv.findAzureDependencies(resource)
	case "gcp":
		deps = dv.findGCPDependencies(resource)
	}

	return deps
}

// findAWSDependencies finds AWS resource dependencies
func (dv *DependencyValidator) findAWSDependencies(resource models.Resource) []string {
	var deps []string

	switch resource.Type {
	case "ec2_instance":
		// EC2 instances depend on security groups, subnets, ENIs
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if sgIDs, ok := metadata["security_groups"].([]string); ok {
				deps = append(deps, sgIDs...)
			}
			if subnetID, ok := metadata["subnet_id"].(string); ok {
				deps = append(deps, subnetID)
			}
			if eniIDs, ok := metadata["network_interfaces"].([]string); ok {
				deps = append(deps, eniIDs...)
			}
		}

	case "security_group":
		// Security groups may reference other security groups
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if rules, ok := metadata["ingress_rules"].([]interface{}); ok {
				for _, rule := range rules {
					if ruleMap, ok := rule.(map[string]interface{}); ok {
						if refSG, ok := ruleMap["source_security_group_id"].(string); ok {
							deps = append(deps, refSG)
						}
					}
				}
			}
		}

	case "subnet":
		// Subnets depend on VPCs
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if vpcID, ok := metadata["vpc_id"].(string); ok {
				deps = append(deps, vpcID)
			}
		}

	case "route_table":
		// Route tables depend on VPCs and may reference NAT gateways, Internet gateways
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if vpcID, ok := metadata["vpc_id"].(string); ok {
				deps = append(deps, vpcID)
			}
			if routes, ok := metadata["routes"].([]interface{}); ok {
				for _, route := range routes {
					if routeMap, ok := route.(map[string]interface{}); ok {
						if gwID, ok := routeMap["gateway_id"].(string); ok {
							deps = append(deps, gwID)
						}
						if natID, ok := routeMap["nat_gateway_id"].(string); ok {
							deps = append(deps, natID)
						}
					}
				}
			}
		}

	case "nat_gateway":
		// NAT gateways depend on subnets and Elastic IPs
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if subnetID, ok := metadata["subnet_id"].(string); ok {
				deps = append(deps, subnetID)
			}
			if eipID, ok := metadata["allocation_id"].(string); ok {
				deps = append(deps, eipID)
			}
		}

	case "internet_gateway":
		// Internet gateways depend on VPCs
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if vpcID, ok := metadata["vpc_id"].(string); ok {
				deps = append(deps, vpcID)
			}
		}

	case "rds_instance":
		// RDS instances depend on DB subnet groups and security groups
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if subnetGroup, ok := metadata["db_subnet_group"].(string); ok {
				deps = append(deps, subnetGroup)
			}
			if sgIDs, ok := metadata["vpc_security_groups"].([]string); ok {
				deps = append(deps, sgIDs...)
			}
		}

	case "elb", "alb", "nlb":
		// Load balancers depend on subnets and security groups
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if subnetIDs, ok := metadata["subnets"].([]string); ok {
				deps = append(deps, subnetIDs...)
			}
			if sgIDs, ok := metadata["security_groups"].([]string); ok {
				deps = append(deps, sgIDs...)
			}
		}

	case "lambda_function":
		// Lambda functions may depend on VPCs, security groups, and IAM roles
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if vpcConfig, ok := metadata["vpc_config"].(map[string]interface{}); ok {
				if subnetIDs, ok := vpcConfig["subnet_ids"].([]string); ok {
					deps = append(deps, subnetIDs...)
				}
				if sgIDs, ok := vpcConfig["security_group_ids"].([]string); ok {
					deps = append(deps, sgIDs...)
				}
			}
			if roleARN, ok := metadata["role"].(string); ok {
				deps = append(deps, roleARN)
			}
		}

	case "ecs_service":
		// ECS services depend on clusters, task definitions, and target groups
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if clusterARN, ok := metadata["cluster"].(string); ok {
				deps = append(deps, clusterARN)
			}
			if taskDef, ok := metadata["task_definition"].(string); ok {
				deps = append(deps, taskDef)
			}
			if targetGroups, ok := metadata["target_groups"].([]string); ok {
				deps = append(deps, targetGroups...)
			}
		}

	case "eks_nodegroup":
		// EKS node groups depend on EKS clusters
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if clusterName, ok := metadata["cluster_name"].(string); ok {
				deps = append(deps, clusterName)
			}
		}
	}

	// Filter out dependencies that aren't in our resource list
	var validDeps []string
	for _, dep := range deps {
		if _, exists := dv.resourceMap[dep]; exists {
			validDeps = append(validDeps, dep)
		}
	}

	return validDeps
}

// findAzureDependencies finds Azure resource dependencies
func (dv *DependencyValidator) findAzureDependencies(resource models.Resource) []string {
	var deps []string

	switch resource.Type {
	case "virtual_machine":
		// VMs depend on NICs, disks, and availability sets
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if nicIDs, ok := metadata["network_interface_ids"].([]string); ok {
				deps = append(deps, nicIDs...)
			}
			if diskIDs, ok := metadata["data_disk_ids"].([]string); ok {
				deps = append(deps, diskIDs...)
			}
			if availSet, ok := metadata["availability_set_id"].(string); ok {
				deps = append(deps, availSet)
			}
		}

	case "network_interface":
		// NICs depend on subnets and security groups
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if subnetID, ok := metadata["subnet_id"].(string); ok {
				deps = append(deps, subnetID)
			}
			if nsgID, ok := metadata["network_security_group_id"].(string); ok {
				deps = append(deps, nsgID)
			}
		}

	case "subnet":
		// Subnets depend on virtual networks
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if vnetID, ok := metadata["virtual_network_id"].(string); ok {
				deps = append(deps, vnetID)
			}
		}

	case "public_ip":
		// Public IPs may be associated with NICs or load balancers
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if nicID, ok := metadata["network_interface_id"].(string); ok {
				deps = append(deps, nicID)
			}
			if lbID, ok := metadata["load_balancer_id"].(string); ok {
				deps = append(deps, lbID)
			}
		}
	}

	// Filter out dependencies that aren't in our resource list
	var validDeps []string
	for _, dep := range deps {
		if _, exists := dv.resourceMap[dep]; exists {
			validDeps = append(validDeps, dep)
		}
	}

	return validDeps
}

// findGCPDependencies finds GCP resource dependencies
func (dv *DependencyValidator) findGCPDependencies(resource models.Resource) []string {
	var deps []string

	switch resource.Type {
	case "compute.googleapis.com/instances":
		// Compute instances depend on networks and disks
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if networkID, ok := metadata["network"].(string); ok {
				deps = append(deps, networkID)
			}
			if diskIDs, ok := metadata["disks"].([]string); ok {
				deps = append(deps, diskIDs...)
			}
		}

	case "compute.googleapis.com/disks":
		// Disks may be attached to instances
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if instanceID, ok := metadata["attached_to"].(string); ok {
				deps = append(deps, instanceID)
			}
		}

	case "compute.googleapis.com/subnetworks":
		// Subnetworks depend on VPC networks
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if networkID, ok := metadata["network"].(string); ok {
				deps = append(deps, networkID)
			}
		}

	case "container.googleapis.com/clusters":
		// GKE clusters depend on networks and subnets
		if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
			if networkID, ok := metadata["network"].(string); ok {
				deps = append(deps, networkID)
			}
			if subnetID, ok := metadata["subnetwork"].(string); ok {
				deps = append(deps, subnetID)
			}
		}
	}

	// Filter out dependencies that aren't in our resource list
	var validDeps []string
	for _, dep := range deps {
		if _, exists := dv.resourceMap[dep]; exists {
			validDeps = append(validDeps, dep)
		}
	}

	return validDeps
}

// detectCycles detects circular dependencies in the dependency graph
func (dv *DependencyValidator) detectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	for resourceID := range dv.resourceMap {
		if !visited[resourceID] {
			if dv.detectCyclesUtil(resourceID, visited, recStack, &path, &cycles) {
				log.Printf("Cycle detected involving resource: %s", resourceID)
			}
		}
	}

	return cycles
}

// detectCyclesUtil is a utility function for cycle detection using DFS
func (dv *DependencyValidator) detectCyclesUtil(resourceID string, visited, recStack map[string]bool, path *[]string, cycles *[][]string) bool {
	visited[resourceID] = true
	recStack[resourceID] = true
	*path = append(*path, resourceID)

	// Check all dependencies
	if deps, exists := dv.dependencies[resourceID]; exists {
		for _, dep := range deps {
			if !visited[dep] {
				if dv.detectCyclesUtil(dep, visited, recStack, path, cycles) {
					return true
				}
			} else if recStack[dep] {
				// Found a cycle
				cycleStart := -1
				for i, id := range *path {
					if id == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(*path)-cycleStart)
					copy(cycle, (*path)[cycleStart:])
					*cycles = append(*cycles, cycle)
				}
				return true
			}
		}
	}

	recStack[resourceID] = false
	*path = (*path)[:len(*path)-1]
	return false
}

// topologicalSort performs topological sorting for deletion order
func (dv *DependencyValidator) topologicalSort(resources []models.Resource) ([]models.Resource, error) {
	// Calculate in-degree for each resource
	inDegree := make(map[string]int)
	for _, resource := range resources {
		if _, exists := inDegree[resource.ID]; !exists {
			inDegree[resource.ID] = 0
		}
		if deps, exists := dv.dependencies[resource.ID]; exists {
			for _, dep := range deps {
				inDegree[dep]++
			}
		}
	}

	// Find all resources with no dependencies (in-degree = 0)
	var queue []string
	for resourceID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, resourceID)
		}
	}

	var orderedResources []models.Resource
	processedCount := 0

	// Process resources in topological order
	for len(queue) > 0 {
		// Pop from queue
		resourceID := queue[0]
		queue = queue[1:]

		// Add to ordered list
		if resource, exists := dv.resourceMap[resourceID]; exists {
			orderedResources = append(orderedResources, *resource)
			processedCount++
		}

		// Reduce in-degree for dependent resources
		if deps, exists := dv.dependencies[resourceID]; exists {
			for _, dep := range deps {
				inDegree[dep]--
				if inDegree[dep] == 0 {
					queue = append(queue, dep)
				}
			}
		}
	}

	// Check if all resources were processed
	if processedCount != len(resources) {
		return nil, fmt.Errorf("failed to process all resources: possible circular dependency")
	}

	return orderedResources, nil
}

// validateDeletionSafety validates that resources can be safely deleted
func (dv *DependencyValidator) validateDeletionSafety(resources []models.Resource) error {
	// Check for external dependencies
	for _, resource := range resources {
		if err := dv.checkExternalDependencies(resource); err != nil {
			return fmt.Errorf("resource %s (%s) has external dependencies: %w", 
				resource.Name, resource.ID, err)
		}
	}

	// Check for protected resources
	for _, resource := range resources {
		if dv.isProtectedResource(resource) {
			return fmt.Errorf("resource %s (%s) is protected and cannot be deleted", 
				resource.Name, resource.ID)
		}
	}

	return nil
}

// checkExternalDependencies checks for dependencies outside the deletion scope
func (dv *DependencyValidator) checkExternalDependencies(resource models.Resource) error {
	// Check if resource has dependencies not in the deletion list
	if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
		// Check for specific external dependencies based on resource type
		switch resource.Type {
		case "vpc", "virtual_network", "compute.googleapis.com/networks":
			// Check if VPC/VNet has resources not in deletion list
			if hasActiveResources, ok := metadata["has_active_resources"].(bool); ok && hasActiveResources {
				return fmt.Errorf("network has active resources that would be orphaned")
			}

		case "security_group", "network_security_group":
			// Check if security group is attached to resources not being deleted
			if attachedCount, ok := metadata["attached_resource_count"].(int); ok && attachedCount > 0 {
				return fmt.Errorf("security group is attached to %d resources", attachedCount)
			}

		case "iam_role", "service_account":
			// Check if IAM role is in use
			if inUse, ok := metadata["in_use"].(bool); ok && inUse {
				return fmt.Errorf("IAM role/service account is currently in use")
			}
		}
	}

	return nil
}

// isProtectedResource checks if a resource is protected from deletion
func (dv *DependencyValidator) isProtectedResource(resource models.Resource) bool {
	// Check for deletion protection flags
	if metadata, ok := resource.Metadata.(map[string]interface{}); ok {
		if protected, ok := metadata["deletion_protection"].(bool); ok && protected {
			return true
		}
		if protected, ok := metadata["termination_protection"].(bool); ok && protected {
			return true
		}
		if locked, ok := metadata["resource_lock"].(bool); ok && locked {
			return true
		}
	}

	// Check for system/default resources that shouldn't be deleted
	if strings.Contains(strings.ToLower(resource.Name), "default") {
		switch resource.Type {
		case "vpc", "virtual_network", "security_group", "network_security_group":
			return true // Don't delete default VPCs/security groups
		}
	}

	// Check for critical tags
	if tags, ok := resource.Tags.(map[string]string); ok {
		if env, exists := tags["Environment"]; exists && env == "production" {
			if protect, exists := tags["DeletionProtection"]; exists && protect == "true" {
				return true
			}
		}
		if critical, exists := tags["Critical"]; exists && critical == "true" {
			return true
		}
	}

	return false
}

// GetDeletionOrder returns a human-readable deletion order
func (dv *DependencyValidator) GetDeletionOrder(resources []models.Resource) []string {
	var order []string
	for i, resource := range resources {
		deps := ""
		if depList, exists := dv.dependencies[resource.ID]; exists && len(depList) > 0 {
			deps = fmt.Sprintf(" (depends on: %s)", strings.Join(depList, ", "))
		}
		order = append(order, fmt.Sprintf("%d. %s (%s)%s", i+1, resource.Name, resource.Type, deps))
	}
	return order
}

// ValidateSingleResourceDeletion validates deletion of a single resource
func (dv *DependencyValidator) ValidateSingleResourceDeletion(ctx context.Context, resource models.Resource, allResources []models.Resource) error {
	// Build resource map from all resources
	for i := range allResources {
		dv.resourceMap[allResources[i].ID] = &allResources[i]
	}

	// Check if resource is protected
	if dv.isProtectedResource(resource) {
		return fmt.Errorf("resource %s is protected and cannot be deleted", resource.Name)
	}

	// Find what depends on this resource
	var dependents []string
	for _, r := range allResources {
		if r.ID == resource.ID {
			continue
		}
		deps := dv.findDependencies(r)
		for _, dep := range deps {
			if dep == resource.ID {
				dependents = append(dependents, r.Name)
			}
		}
	}

	if len(dependents) > 0 {
		return fmt.Errorf("resource %s cannot be deleted because the following resources depend on it: %s",
			resource.Name, strings.Join(dependents, ", "))
	}

	// Check for external dependencies
	if err := dv.checkExternalDependencies(resource); err != nil {
		return fmt.Errorf("resource %s has external dependencies: %w", resource.Name, err)
	}

	return nil
}

// GetResourceDependencies returns the dependencies of a specific resource
func (dv *DependencyValidator) GetResourceDependencies(resource models.Resource) []string {
	return dv.findDependencies(resource)
}

// SortByDependencyGroups groups resources by their dependency level
func (dv *DependencyValidator) SortByDependencyGroups(resources []models.Resource) [][]models.Resource {
	// Build dependency graph
	for i := range resources {
		dv.resourceMap[resources[i].ID] = &resources[i]
	}
	dv.buildDependencyGraph(context.Background(), resources)

	// Calculate dependency levels
	levels := make(map[string]int)
	var calculateLevel func(string) int
	calculateLevel = func(resourceID string) int {
		if level, exists := levels[resourceID]; exists {
			return level
		}

		maxDepLevel := -1
		if deps, exists := dv.dependencies[resourceID]; exists {
			for _, dep := range deps {
				depLevel := calculateLevel(dep)
				if depLevel > maxDepLevel {
					maxDepLevel = depLevel
				}
			}
		}

		levels[resourceID] = maxDepLevel + 1
		return maxDepLevel + 1
	}

	for _, resource := range resources {
		calculateLevel(resource.ID)
	}

	// Group resources by level
	maxLevel := 0
	for _, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	groups := make([][]models.Resource, maxLevel+1)
	for _, resource := range resources {
		level := levels[resource.ID]
		groups[level] = append(groups[level], resource)
	}

	// Sort each group for consistent ordering
	for i := range groups {
		sort.Slice(groups[i], func(a, b int) bool {
			return groups[i][a].Name < groups[i][b].Name
		})
	}

	return groups
}