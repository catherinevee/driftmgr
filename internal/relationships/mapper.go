package relationships

import (
	"context"
	"fmt"
	"strings"
	"sync"

	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
)

// RelationshipType defines the type of relationship between resources
type RelationshipType string

const (
	// Dependency relationships
	DependsOn     RelationshipType = "depends_on"
	RequiredBy    RelationshipType = "required_by"
	References    RelationshipType = "references"
	ReferencedBy  RelationshipType = "referenced_by"
	
	// Containment relationships
	Contains      RelationshipType = "contains"
	ContainedBy   RelationshipType = "contained_by"
	MemberOf      RelationshipType = "member_of"
	HasMember     RelationshipType = "has_member"
	
	// Network relationships
	ConnectsTo    RelationshipType = "connects_to"
	ConnectedFrom RelationshipType = "connected_from"
	Routes        RelationshipType = "routes"
	RoutedBy      RelationshipType = "routed_by"
	
	// Security relationships
	Secures       RelationshipType = "secures"
	SecuredBy     RelationshipType = "secured_by"
	Allows        RelationshipType = "allows"
	AllowedBy     RelationshipType = "allowed_by"
	
	// Management relationships
	Manages       RelationshipType = "manages"
	ManagedBy     RelationshipType = "managed_by"
	Monitors      RelationshipType = "monitors"
	MonitoredBy   RelationshipType = "monitored_by"
)

// Relationship represents a relationship between two resources
type Relationship struct {
	SourceID         string                 `json:"source_id"`
	SourceType       string                 `json:"source_type"`
	TargetID         string                 `json:"target_id"`
	TargetType       string                 `json:"target_type"`
	Type             RelationshipType       `json:"type"`
	Bidirectional    bool                   `json:"bidirectional"`
	Strength         float64                `json:"strength"` // 0.0 to 1.0
	Metadata         map[string]interface{} `json:"metadata"`
}

// ResourceNode represents a resource in the dependency graph
type ResourceNode struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	Status       string                 `json:"status"`
	Tags         map[string]string      `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	Dependencies []string               `json:"dependencies"`
	Dependents   []string               `json:"dependents"`
}

// DependencyGraph represents the complete resource dependency graph
type DependencyGraph struct {
	Nodes         map[string]*ResourceNode `json:"nodes"`
	Relationships []Relationship           `json:"relationships"`
	Cycles        [][]string               `json:"cycles"`
	Layers        [][]string               `json:"layers"` // Topologically sorted layers
}

// Mapper handles resource relationship discovery and mapping
type Mapper struct {
	resources     []apimodels.Resource
	relationships []Relationship
	graph         *DependencyGraph
	mu            sync.RWMutex
}

// NewMapper creates a new relationship mapper
func NewMapper() *Mapper {
	return &Mapper{
		resources:     []apimodels.Resource{},
		relationships: []Relationship{},
		graph: &DependencyGraph{
			Nodes:         make(map[string]*ResourceNode),
			Relationships: []Relationship{},
			Cycles:        [][]string{},
			Layers:        [][]string{},
		},
	}
}

// DiscoverRelationships discovers relationships between resources
func (m *Mapper) DiscoverRelationships(ctx context.Context, resources []apimodels.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.resources = resources
	m.relationships = []Relationship{}
	
	// Build resource nodes
	for _, res := range resources {
		node := &ResourceNode{
			ID:           res.ID,
			Name:         res.Name,
			Type:         res.Type,
			Provider:     res.Provider,
			Region:       res.Region,
			Status:       res.Status,
			Tags:         res.Tags,
			Dependencies: []string{},
			Dependents:   []string{},
		}
		m.graph.Nodes[res.ID] = node
	}
	
	// Discover relationships based on resource type and provider
	for _, res := range resources {
		switch res.Provider {
		case "aws":
			m.discoverAWSRelationships(res)
		case "azure":
			m.discoverAzureRelationships(res)
		case "gcp":
			m.discoverGCPRelationships(res)
		}
	}
	
	// Detect cycles
	m.detectCycles()
	
	// Build topological layers
	m.buildLayers()
	
	// Relationships are stored in m.relationships
	// Caller can access them via GetRelationships() method
	
	return nil
}

// discoverAWSRelationships discovers AWS-specific resource relationships
func (m *Mapper) discoverAWSRelationships(resource apimodels.Resource) {
	switch resource.Type {
	case "AWS::EC2::Instance":
		// EC2 instances depend on VPC, subnet, security groups
		m.findAndAddRelationship(resource.ID, "vpc-", DependsOn)
		m.findAndAddRelationship(resource.ID, "subnet-", DependsOn)
		m.findAndAddRelationship(resource.ID, "sg-", References)
		m.findAndAddRelationship(resource.ID, "ami-", DependsOn)
		
	case "AWS::EC2::SecurityGroup":
		// Security groups reference VPC
		m.findAndAddRelationship(resource.ID, "vpc-", ContainedBy)
		
	case "AWS::EC2::Subnet":
		// Subnets are contained in VPC
		m.findAndAddRelationship(resource.ID, "vpc-", ContainedBy)
		
	case "AWS::RDS::DBInstance":
		// RDS instances depend on VPC, subnet groups, security groups
		m.findAndAddRelationship(resource.ID, "vpc-", DependsOn)
		m.findAndAddRelationship(resource.ID, "sg-", References)
		
	case "AWS::ELB::LoadBalancer":
		// Load balancers connect to EC2 instances
		m.findAndAddRelationship(resource.ID, "i-", ConnectsTo)
		m.findAndAddRelationship(resource.ID, "vpc-", ContainedBy)
		
	case "AWS::Lambda::Function":
		// Lambda functions may reference VPC, security groups, IAM roles
		m.findAndAddRelationship(resource.ID, "vpc-", References)
		m.findAndAddRelationship(resource.ID, "sg-", References)
		m.findAndAddRelationship(resource.ID, "arn:aws:iam", References)
		
	case "AWS::S3::Bucket":
		// S3 buckets may have bucket policies referencing IAM
		m.findAndAddRelationship(resource.ID, "arn:aws:iam", AllowedBy)
	}
}

// discoverAzureRelationships discovers Azure-specific resource relationships
func (m *Mapper) discoverAzureRelationships(resource apimodels.Resource) {
	switch resource.Type {
	case "Microsoft.Compute/virtualMachines":
		// VMs depend on VNet, subnet, NSG
		m.findAndAddRelationship(resource.ID, "/virtualNetworks/", DependsOn)
		m.findAndAddRelationship(resource.ID, "/subnets/", DependsOn)
		m.findAndAddRelationship(resource.ID, "/networkSecurityGroups/", References)
		
	case "Microsoft.Network/networkSecurityGroups":
		// NSGs reference subnets
		m.findAndAddRelationship(resource.ID, "/subnets/", Secures)
		
	case "Microsoft.Network/virtualNetworks":
		// VNets contain subnets
		m.findAndAddRelationship(resource.ID, "/subnets/", Contains)
		
	case "Microsoft.Sql/servers/databases":
		// SQL databases depend on SQL servers
		m.findAndAddRelationship(resource.ID, "/servers/", ContainedBy)
	}
}

// discoverGCPRelationships discovers GCP-specific resource relationships
func (m *Mapper) discoverGCPRelationships(resource apimodels.Resource) {
	switch resource.Type {
	case "compute.v1.instance":
		// Compute instances depend on VPC, subnet, firewall rules
		m.findAndAddRelationship(resource.ID, "networks/", DependsOn)
		m.findAndAddRelationship(resource.ID, "subnetworks/", DependsOn)
		m.findAndAddRelationship(resource.ID, "firewalls/", References)
		
	case "compute.v1.firewall":
		// Firewall rules reference VPC
		m.findAndAddRelationship(resource.ID, "networks/", Secures)
		
	case "compute.v1.subnetwork":
		// Subnetworks are contained in VPC
		m.findAndAddRelationship(resource.ID, "networks/", ContainedBy)
	}
}

// findAndAddRelationship finds resources matching a pattern and adds relationships
func (m *Mapper) findAndAddRelationship(sourceID, targetPattern string, relType RelationshipType) {
	for _, res := range m.resources {
		if strings.Contains(res.ID, targetPattern) || strings.Contains(res.Name, targetPattern) {
			rel := Relationship{
				SourceID:      sourceID,
				TargetID:      res.ID,
				Type:          relType,
				Bidirectional: false,
				Strength:      0.8,
				Metadata:      map[string]interface{}{},
			}
			
			m.relationships = append(m.relationships, rel)
			m.graph.Relationships = append(m.graph.Relationships, rel)
			
			// Update node dependencies
			if sourceNode, ok := m.graph.Nodes[sourceID]; ok {
				sourceNode.Dependencies = append(sourceNode.Dependencies, res.ID)
			}
			if targetNode, ok := m.graph.Nodes[res.ID]; ok {
				targetNode.Dependents = append(targetNode.Dependents, sourceID)
			}
		}
	}
}

// detectCycles detects cycles in the dependency graph
func (m *Mapper) detectCycles() {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}
	
	for id := range m.graph.Nodes {
		if !visited[id] {
			m.detectCyclesDFS(id, visited, recStack, &path)
		}
	}
}

func (m *Mapper) detectCyclesDFS(nodeID string, visited, recStack map[string]bool, path *[]string) bool {
	visited[nodeID] = true
	recStack[nodeID] = true
	*path = append(*path, nodeID)
	
	node := m.graph.Nodes[nodeID]
	for _, dep := range node.Dependencies {
		if !visited[dep] {
			if m.detectCyclesDFS(dep, visited, recStack, path) {
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
				cycle := (*path)[cycleStart:]
				m.graph.Cycles = append(m.graph.Cycles, cycle)
			}
			return true
		}
	}
	
	recStack[nodeID] = false
	*path = (*path)[:len(*path)-1]
	return false
}

// buildLayers builds topologically sorted layers of resources
func (m *Mapper) buildLayers() {
	inDegree := make(map[string]int)
	
	// Calculate in-degrees
	for id, node := range m.graph.Nodes {
		inDegree[id] = len(node.Dependencies)
	}
	
	// Build layers
	processed := make(map[string]bool)
	for len(processed) < len(m.graph.Nodes) {
		layer := []string{}
		
		for id := range m.graph.Nodes {
			if !processed[id] && inDegree[id] == 0 {
				layer = append(layer, id)
			}
		}
		
		if len(layer) == 0 {
			// No more nodes with in-degree 0, might have cycles
			break
		}
		
		m.graph.Layers = append(m.graph.Layers, layer)
		
		// Mark as processed and reduce in-degrees
		for _, id := range layer {
			processed[id] = true
			node := m.graph.Nodes[id]
			for _, dep := range node.Dependents {
				inDegree[dep]--
			}
		}
	}
}

// GetGraph returns the current dependency graph
func (m *Mapper) GetGraph() *DependencyGraph {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.graph
}

// GetRelationships returns all discovered relationships
func (m *Mapper) GetRelationships() []Relationship {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.relationships
}

// GetDependencies returns all dependencies for a resource
func (m *Mapper) GetDependencies(resourceID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if node, ok := m.graph.Nodes[resourceID]; ok {
		return node.Dependencies
	}
	return []string{}
}

// GetDependents returns all resources that depend on a given resource
func (m *Mapper) GetDependents(resourceID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if node, ok := m.graph.Nodes[resourceID]; ok {
		return node.Dependents
	}
	return []string{}
}

// GetDeletionOrder returns the order in which resources should be deleted
func (m *Mapper) GetDeletionOrder(resourceIDs []string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Reverse topological order for deletion
	order := []string{}
	
	// Start from the deepest layer
	for i := len(m.graph.Layers) - 1; i >= 0; i-- {
		for _, id := range m.graph.Layers[i] {
			for _, targetID := range resourceIDs {
				if id == targetID {
					order = append(order, id)
					break
				}
			}
		}
	}
	
	return order
}

// GetCreationOrder returns the order in which resources should be created
func (m *Mapper) GetCreationOrder(resourceIDs []string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Topological order for creation
	order := []string{}
	
	for _, layer := range m.graph.Layers {
		for _, id := range layer {
			for _, targetID := range resourceIDs {
				if id == targetID {
					order = append(order, id)
					break
				}
			}
		}
	}
	
	return order
}

// ValidateDeletion checks if resources can be safely deleted
func (m *Mapper) ValidateDeletion(resourceIDs []string) (bool, []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	blockers := []string{}
	resourceSet := make(map[string]bool)
	
	for _, id := range resourceIDs {
		resourceSet[id] = true
	}
	
	// Check if any resource has dependents outside the deletion set
	for _, id := range resourceIDs {
		if node, ok := m.graph.Nodes[id]; ok {
			for _, dep := range node.Dependents {
				if !resourceSet[dep] {
					blockers = append(blockers, fmt.Sprintf("%s depends on %s", dep, id))
				}
			}
		}
	}
	
	return len(blockers) == 0, blockers
}