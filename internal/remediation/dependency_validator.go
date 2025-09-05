package remediation

import (
	"fmt"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// DependencyValidator validates and manages resource dependencies
type DependencyValidator struct {
	resources map[string]models.Resource
	graph     map[string][]string // resource ID -> dependent resource IDs
}

// NewDependencyValidator creates a new dependency validator
func NewDependencyValidator() *DependencyValidator {
	return &DependencyValidator{
		resources: make(map[string]models.Resource),
		graph:     make(map[string][]string),
	}
}

// ValidateDeletionOrder checks if resources can be deleted in the given order
func (dv *DependencyValidator) ValidateDeletionOrder(resources []models.Resource) error {
	// Build dependency graph
	for _, resource := range resources {
		dv.resources[resource.ID] = resource
		dependencies := dv.findDependencies(resource)
		for _, dep := range dependencies {
			dv.graph[dep] = append(dv.graph[dep], resource.ID)
		}
	}

	// Check for circular dependencies
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	for _, resource := range resources {
		if !visited[resource.ID] {
			if dv.hasCycle(resource.ID, visited, recStack) {
				return fmt.Errorf("circular dependency detected")
			}
		}
	}

	return nil
}

// OrderResourcesForDeletion returns resources in safe deletion order
func (dv *DependencyValidator) OrderResourcesForDeletion(resources []models.Resource) []models.Resource {
	// Build dependency graph
	for _, resource := range resources {
		dv.resources[resource.ID] = resource
		dependencies := dv.findDependencies(resource)
		for _, dep := range dependencies {
			dv.graph[dep] = append(dv.graph[dep], resource.ID)
		}
	}

	// Topological sort
	ordered := make([]models.Resource, 0, len(resources))
	visited := make(map[string]bool)

	var visit func(string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		// Visit dependents first
		for _, dependent := range dv.graph[id] {
			visit(dependent)
		}

		// Add to ordered list
		if resource, ok := dv.resources[id]; ok {
			ordered = append(ordered, resource)
		}
	}

	for _, resource := range resources {
		visit(resource.ID)
	}

	return ordered
}

// findDependencies finds resources that this resource depends on
func (dv *DependencyValidator) findDependencies(resource models.Resource) []string {
	switch resource.Provider {
	case "aws":
		return dv.findAWSDependencies(resource)
	case "azure":
		return dv.findAzureDependencies(resource)
	case "gcp":
		return dv.findGCPDependencies(resource)
	default:
		return []string{}
	}
}

// findAWSDependencies finds AWS resource dependencies
func (dv *DependencyValidator) findAWSDependencies(resource models.Resource) []string {
	var deps []string

	switch resource.Type {
	case "ec2_instance":
		// EC2 instances depend on security groups, subnets, ENIs
		// Metadata is now map[string]string, so we parse from strings
		if sgStr, ok := resource.Metadata["security_groups"]; ok {
			// Parse security groups from string representation
			deps = append(deps, sgStr)
		}
		if subnetID, ok := resource.Metadata["subnet_id"]; ok {
			deps = append(deps, subnetID)
		}
		if eniStr, ok := resource.Metadata["network_interfaces"]; ok {
			// Parse ENIs from string representation
			deps = append(deps, eniStr)
		}

	case "security_group":
		// Security groups may reference other security groups
		if rulesStr, ok := resource.Metadata["ingress_rules"]; ok {
			// Rules are stored as string representation
			deps = append(deps, rulesStr)
		}

	case "subnet":
		// Subnets depend on VPCs
		if vpcID, ok := resource.Metadata["vpc_id"]; ok {
			deps = append(deps, vpcID)
		}

	case "route_table":
		// Route tables depend on VPCs and may reference NAT gateways, Internet gateways
		if vpcID, ok := resource.Metadata["vpc_id"]; ok {
			deps = append(deps, vpcID)
		}
		if routesStr, ok := resource.Metadata["routes"]; ok {
			// Routes stored as string representation
			deps = append(deps, routesStr)
		}

	case "nat_gateway":
		// NAT gateways depend on subnets and Elastic IPs
		if subnetID, ok := resource.Metadata["subnet_id"]; ok {
			deps = append(deps, subnetID)
		}
		if eipID, ok := resource.Metadata["allocation_id"]; ok {
			deps = append(deps, eipID)
		}

	case "internet_gateway":
		// Internet gateways depend on VPCs
		if vpcID, ok := resource.Metadata["vpc_id"]; ok {
			deps = append(deps, vpcID)
		}

	case "rds_instance":
		// RDS instances depend on DB subnet groups and security groups
		if subnetGroup, ok := resource.Metadata["db_subnet_group"]; ok {
			deps = append(deps, subnetGroup)
		}
		if sgStr, ok := resource.Metadata["vpc_security_groups"]; ok {
			// Security groups stored as string representation
			deps = append(deps, sgStr)
		}

	case "elb", "alb", "nlb":
		// Load balancers depend on subnets and security groups
		if subnetStr, ok := resource.Metadata["subnets"]; ok {
			// Subnets stored as string representation
			deps = append(deps, subnetStr)
		}
		if sgStr, ok := resource.Metadata["security_groups"]; ok {
			// Security groups stored as string representation
			deps = append(deps, sgStr)
		}

	case "lambda_function":
		// Lambda functions may depend on VPCs, security groups, and IAM roles
		if vpcConfig, ok := resource.Metadata["vpc_config"]; ok {
			// VPC config stored as string representation
			deps = append(deps, vpcConfig)
		}
		if roleARN, ok := resource.Metadata["role"]; ok {
			deps = append(deps, roleARN)
		}

	case "ecs_service":
		// ECS services depend on clusters, task definitions, and target groups
		if clusterARN, ok := resource.Metadata["cluster"]; ok {
			deps = append(deps, clusterARN)
		}
		if taskDef, ok := resource.Metadata["task_definition"]; ok {
			deps = append(deps, taskDef)
		}
		if targetGroups, ok := resource.Metadata["target_groups"]; ok {
			// Target groups stored as string representation
			deps = append(deps, targetGroups)
		}

	case "eks_nodegroup":
		// EKS node groups depend on EKS clusters
		if clusterName, ok := resource.Metadata["cluster_name"]; ok {
			deps = append(deps, clusterName)
		}
	}

	return deps
}

// findAzureDependencies finds Azure resource dependencies
func (dv *DependencyValidator) findAzureDependencies(resource models.Resource) []string {
	var deps []string

	switch resource.Type {
	case "Microsoft.Compute/virtualMachines":
		// VMs depend on network interfaces, availability sets, etc.
		if nicStr, ok := resource.Metadata["network_interfaces"]; ok {
			deps = append(deps, nicStr)
		}
		if availSet, ok := resource.Metadata["availability_set"]; ok {
			deps = append(deps, availSet)
		}

	case "Microsoft.Network/networkInterfaces":
		// NICs depend on subnets and security groups
		if subnetID, ok := resource.Metadata["subnet_id"]; ok {
			deps = append(deps, subnetID)
		}
		if nsgID, ok := resource.Metadata["network_security_group_id"]; ok {
			deps = append(deps, nsgID)
		}

	case "Microsoft.Network/virtualNetworks/subnets":
		// Subnets depend on virtual networks
		if vnetID, ok := resource.Metadata["virtual_network_id"]; ok {
			deps = append(deps, vnetID)
		}

	case "Microsoft.Storage/storageAccounts":
		// Storage accounts may depend on virtual networks for service endpoints
		if vnetStr, ok := resource.Metadata["network_acls"]; ok {
			deps = append(deps, vnetStr)
		}
	}

	return deps
}

// findGCPDependencies finds GCP resource dependencies
func (dv *DependencyValidator) findGCPDependencies(resource models.Resource) []string {
	var deps []string

	switch resource.Type {
	case "compute.v1.instance":
		// Compute instances depend on networks, subnetworks, disks
		if networkStr, ok := resource.Metadata["network_interfaces"]; ok {
			deps = append(deps, networkStr)
		}
		if diskStr, ok := resource.Metadata["disks"]; ok {
			deps = append(deps, diskStr)
		}

	case "compute.v1.subnetwork":
		// Subnetworks depend on networks
		if networkID, ok := resource.Metadata["network"]; ok {
			deps = append(deps, networkID)
		}

	case "compute.v1.firewall":
		// Firewall rules depend on networks
		if networkID, ok := resource.Metadata["network"]; ok {
			deps = append(deps, networkID)
		}

	case "storage.v1.bucket":
		// Buckets may have lifecycle dependencies
		if lifecycleStr, ok := resource.Metadata["lifecycle_rules"]; ok {
			deps = append(deps, lifecycleStr)
		}
	}

	return deps
}

// hasCycle checks for circular dependencies using DFS
func (dv *DependencyValidator) hasCycle(node string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range dv.graph[node] {
		if !visited[neighbor] {
			if dv.hasCycle(neighbor, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// GetDependents returns resources that depend on the given resource
func (dv *DependencyValidator) GetDependents(resourceID string) []string {
	return dv.graph[resourceID]
}

// GetDependencies returns resources that the given resource depends on
func (dv *DependencyValidator) GetDependencies(resource models.Resource) []string {
	return dv.findDependencies(resource)
}