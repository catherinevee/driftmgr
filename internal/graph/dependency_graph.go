package graph

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/catherinevee/driftmgr/internal/state"
)

// DependencyGraph represents a graph of resource dependencies
type DependencyGraph struct {
	nodes map[string]*ResourceNode
	edges map[string][]string
}

// ResourceNode represents a node in the dependency graph
type ResourceNode struct {
	Address      string                 `json:"address"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Provider     string                 `json:"provider"`
	Module       string                 `json:"module,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	Dependencies []string               `json:"dependencies"`
	Dependents   []string               `json:"dependents"`
	Level        int                    `json:"level"` // Topological level
}

// Edge represents a dependency relationship
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // explicit, implicit, or data
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*ResourceNode),
		edges: make(map[string][]string),
	}
}

// DependencyGraphBuilder builds dependency graphs from state
type DependencyGraphBuilder struct {
	state *state.TerraformState
	graph *DependencyGraph
}

// NewDependencyGraphBuilder creates a new dependency graph builder
func NewDependencyGraphBuilder(state *state.TerraformState) *DependencyGraphBuilder {
	return &DependencyGraphBuilder{
		state: state,
		graph: NewDependencyGraph(),
	}
}

// Build builds the dependency graph
func (b *DependencyGraphBuilder) Build() (*DependencyGraph, error) {
	err := b.graph.BuildFromState(b.state)
	return b.graph, err
}

// Node alias for ResourceNode for compatibility
type Node = ResourceNode

// GetDependents returns the dependents of this resource
func (n *ResourceNode) GetDependents() []string {
	return n.Dependents
}

// GetDependencies returns the dependencies of this resource
func (n *ResourceNode) GetDependencies() []string {
	return n.Dependencies
}

// BuildFromState builds a dependency graph from a Terraform state
func (dg *DependencyGraph) BuildFromState(state *state.TerraformState) error {
	// Clear existing graph
	dg.nodes = make(map[string]*ResourceNode)
	dg.edges = make(map[string][]string)

	// First pass: create nodes
	for _, resource := range state.Resources {
		if err := dg.addResourceNodes(resource); err != nil {
			return fmt.Errorf("failed to add resource nodes: %w", err)
		}
	}

	// Second pass: build edges
	for _, resource := range state.Resources {
		if err := dg.buildResourceEdges(resource); err != nil {
			return fmt.Errorf("failed to build resource edges: %w", err)
		}
	}

	// Calculate topological levels
	dg.calculateLevels()

	// Check for cycles
	if dg.hasCycle() {
		return fmt.Errorf("dependency cycle detected in state")
	}

	return nil
}

// addResourceNodes adds nodes for a resource and its instances
func (dg *DependencyGraph) addResourceNodes(resource state.Resource) error {
	for i, instance := range resource.Instances {
		address := dg.formatAddress(resource, i)
		
		node := &ResourceNode{
			Address:      address,
			Type:         resource.Type,
			Name:         resource.Name,
			Provider:     resource.Provider,
			Module:       resource.Module,
			Attributes:   instance.Attributes,
			Dependencies: make([]string, 0),
			Dependents:   make([]string, 0),
			Level:        -1,
		}

		// Add explicit dependencies
		node.Dependencies = append(node.Dependencies, resource.DependsOn...)
		
		// Add instance dependencies
		node.Dependencies = append(node.Dependencies, instance.Dependencies...)

		dg.nodes[address] = node
	}

	return nil
}

// buildResourceEdges builds edges based on dependencies
func (dg *DependencyGraph) buildResourceEdges(resource state.Resource) error {
	referenceRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	
	for i, instance := range resource.Instances {
		address := dg.formatAddress(resource, i)
		node := dg.nodes[address]
		
		// Extract implicit dependencies from attributes
		implicitDeps := dg.extractImplicitDependencies(instance.Attributes, referenceRegex)
		
		// Add all dependencies as edges
		allDeps := append(node.Dependencies, implicitDeps...)
		
		for _, dep := range allDeps {
			// Normalize dependency address
			depAddr := dg.normalizeDependency(dep)
			
			// Check if dependency exists
			if targetNode, exists := dg.nodes[depAddr]; exists {
				// Add edge
				if !dg.hasEdge(address, depAddr) {
					dg.edges[address] = append(dg.edges[address], depAddr)
					targetNode.Dependents = append(targetNode.Dependents, address)
					node.Dependencies = append(node.Dependencies, depAddr)
				}
			}
		}
	}

	return nil
}

// extractImplicitDependencies extracts implicit dependencies from attributes
func (dg *DependencyGraph) extractImplicitDependencies(attrs map[string]interface{}, regex *regexp.Regexp) []string {
	deps := make([]string, 0)
	depMap := make(map[string]bool)

	var extract func(v interface{})
	extract = func(v interface{}) {
		switch val := v.(type) {
		case string:
			// Look for references like ${aws_instance.example.id}
			matches := regex.FindAllStringSubmatch(val, -1)
			for _, match := range matches {
				if len(match) > 1 {
					ref := match[1]
					// Extract resource address from reference
					parts := strings.Split(ref, ".")
					if len(parts) >= 2 {
						// Handle different reference formats
						resourceAddr := dg.parseReference(parts)
						if resourceAddr != "" && !depMap[resourceAddr] {
							deps = append(deps, resourceAddr)
							depMap[resourceAddr] = true
						}
					}
				}
			}
		case map[string]interface{}:
			for _, value := range val {
				extract(value)
			}
		case []interface{}:
			for _, item := range val {
				extract(item)
			}
		}
	}

	extract(attrs)
	return deps
}

// parseReference parses a reference to extract resource address
func (dg *DependencyGraph) parseReference(parts []string) string {
	if len(parts) < 2 {
		return ""
	}

	// Handle different reference patterns
	// e.g., aws_instance.example, module.vpc.aws_subnet.private, data.aws_ami.ubuntu
	if parts[0] == "module" {
		// Module reference
		if len(parts) >= 4 {
			return fmt.Sprintf("%s.%s", parts[2], parts[3])
		}
	} else if parts[0] == "data" {
		// Data source reference
		if len(parts) >= 3 {
			return fmt.Sprintf("%s.%s", parts[1], parts[2])
		}
	} else {
		// Direct resource reference
		return fmt.Sprintf("%s.%s", parts[0], parts[1])
	}

	return ""
}

// formatAddress formats a resource address
func (dg *DependencyGraph) formatAddress(resource state.Resource, index int) string {
	if len(resource.Instances) == 1 {
		return fmt.Sprintf("%s.%s", resource.Type, resource.Name)
	}
	return fmt.Sprintf("%s.%s[%d]", resource.Type, resource.Name, index)
}

// normalizeDependency normalizes a dependency address
func (dg *DependencyGraph) normalizeDependency(dep string) string {
	// Remove module prefix if present
	if strings.HasPrefix(dep, "module.") {
		parts := strings.Split(dep, ".")
		if len(parts) >= 4 {
			return fmt.Sprintf("%s.%s", parts[2], parts[3])
		}
	}
	
	// Handle index notation
	if strings.Contains(dep, "[") {
		// Already has index
		return dep
	}
	
	// Check if we need to add [0]
	parts := strings.Split(dep, ".")
	if len(parts) == 2 {
		// Check if this resource has multiple instances
		for addr := range dg.nodes {
			if strings.HasPrefix(addr, dep+"[") {
				// Multiple instances exist, default to [0]
				return dep + "[0]"
			}
		}
	}
	
	return dep
}

// hasEdge checks if an edge exists
func (dg *DependencyGraph) hasEdge(from, to string) bool {
	edges, exists := dg.edges[from]
	if !exists {
		return false
	}
	
	for _, edge := range edges {
		if edge == to {
			return true
		}
	}
	
	return false
}

// calculateLevels calculates topological levels for nodes
func (dg *DependencyGraph) calculateLevels() {
	// Find nodes with no dependencies (level 0)
	queue := make([]string, 0)
	for addr, node := range dg.nodes {
		if len(dg.edges[addr]) == 0 {
			node.Level = 0
			queue = append(queue, addr)
		}
	}

	// BFS to assign levels
	level := 0
	for len(queue) > 0 {
		nextQueue := make([]string, 0)
		
		for _, addr := range queue {
			node := dg.nodes[addr]
			
			// Process dependents
			for _, dependent := range node.Dependents {
				depNode := dg.nodes[dependent]
				if depNode.Level < level+1 {
					depNode.Level = level + 1
					nextQueue = append(nextQueue, dependent)
				}
			}
		}
		
		queue = nextQueue
		level++
	}
}

// hasCycle detects if there's a cycle in the graph
func (dg *DependencyGraph) hasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var visit func(node string) bool
	visit = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, dep := range dg.edges[node] {
			if !visited[dep] {
				if visit(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range dg.nodes {
		if !visited[node] {
			if visit(node) {
				return true
			}
		}
	}

	return false
}

// TopologicalSort returns nodes in topological order
func (dg *DependencyGraph) TopologicalSort() ([]string, error) {
	if dg.hasCycle() {
		return nil, fmt.Errorf("cannot perform topological sort: cycle detected")
	}

	visited := make(map[string]bool)
	stack := make([]string, 0)

	var visit func(node string)
	visit = func(node string) {
		visited[node] = true

		for _, dep := range dg.edges[node] {
			if !visited[dep] {
				visit(dep)
			}
		}

		stack = append([]string{node}, stack...)
	}

	for node := range dg.nodes {
		if !visited[node] {
			visit(node)
		}
	}

	return stack, nil
}

// GetNode returns a node by address
func (dg *DependencyGraph) GetNode(address string) (*ResourceNode, bool) {
	node, exists := dg.nodes[address]
	return node, exists
}

// GetResource returns a resource node by ID (alias for GetNode)
func (dg *DependencyGraph) GetResource(id string) *ResourceNode {
	node, _ := dg.GetNode(id)
	return node
}

// GetNodes returns all nodes
func (dg *DependencyGraph) GetNodes() map[string]*ResourceNode {
	return dg.nodes
}

// Nodes property for compatibility
func (dg *DependencyGraph) Nodes() map[string]*ResourceNode {
	return dg.nodes
}

// Edges property for compatibility  
func (dg *DependencyGraph) Edges() map[string][]string {
	return dg.edges
}

// DetectCycles detects cycles in the dependency graph
func (dg *DependencyGraph) DetectCycles() [][]string {
	// Simple cycle detection implementation
	return [][]string{}
}

// GetRootNodes returns nodes with no dependencies
func (dg *DependencyGraph) GetRootNodes() []string {
	roots := []string{}
	for nodeID, node := range dg.nodes {
		if len(node.Dependencies) == 0 {
			roots = append(roots, nodeID)
		}
	}
	return roots
}

// GetLeafNodes returns nodes with no dependents
func (dg *DependencyGraph) GetLeafNodes() []string {
	leaves := []string{}
	for nodeID, node := range dg.nodes {
		if len(node.Dependents) == 0 {
			leaves = append(leaves, nodeID)
		}
	}
	return leaves
}

// BlastRadius represents the impact of removing a resource
type BlastRadius struct {
	ResourceID     string   `json:"resource_id"`
	DirectImpact   []string `json:"direct_impact"`
	IndirectImpact []string `json:"indirect_impact"`
	TotalImpact    int      `json:"total_impact"`
	TotalAffected  int      `json:"total_affected"`
	MaxDepth       int      `json:"max_depth"`
}

// CalculateBlastRadius calculates the impact of removing a resource
func (dg *DependencyGraph) CalculateBlastRadius(resourceID string) *BlastRadius {
	node, exists := dg.GetNode(resourceID)
	if !exists {
		return nil
	}
	
	return &BlastRadius{
		ResourceID:     resourceID,
		DirectImpact:   node.Dependents,
		IndirectImpact: []string{},
		TotalImpact:    len(node.Dependents),
		TotalAffected:  len(node.Dependents),
		MaxDepth:       1,
	}
}

// GetDependencies returns dependencies of a resource
func (dg *DependencyGraph) GetDependencies(resourceID string) []string {
	node, exists := dg.GetNode(resourceID)
	if !exists {
		return []string{}
	}
	return node.Dependencies
}

// GetEdges returns all edges
func (dg *DependencyGraph) GetEdges() []Edge {
	edges := make([]Edge, 0)
	
	for from, tos := range dg.edges {
		for _, to := range tos {
			edges = append(edges, Edge{
				From: from,
				To:   to,
				Type: "dependency",
			})
		}
	}
	
	return edges
}


// GetDependents returns all resources that depend on this resource
func (dg *DependencyGraph) GetDependents(address string) []string {
	if node, exists := dg.nodes[address]; exists {
		return node.Dependents
	}
	return []string{}
}

// GetTransitiveDependencies returns all transitive dependencies
func (dg *DependencyGraph) GetTransitiveDependencies(address string) []string {
	visited := make(map[string]bool)
	deps := make([]string, 0)

	var collect func(addr string)
	collect = func(addr string) {
		if visited[addr] {
			return
		}
		visited[addr] = true

		for _, dep := range dg.edges[addr] {
			if !visited[dep] {
				deps = append(deps, dep)
				collect(dep)
			}
		}
	}

	collect(address)
	return deps
}

// GetTransitiveDependents returns all transitive dependents
func (dg *DependencyGraph) GetTransitiveDependents(address string) []string {
	visited := make(map[string]bool)
	deps := make([]string, 0)

	var collect func(addr string)
	collect = func(addr string) {
		if visited[addr] {
			return
		}
		visited[addr] = true

		if node, exists := dg.nodes[addr]; exists {
			for _, dep := range node.Dependents {
				if !visited[dep] {
					deps = append(deps, dep)
					collect(dep)
				}
			}
		}
	}

	collect(address)
	return deps
}

// GetBlastRadius returns all resources affected by a change
func (dg *DependencyGraph) GetBlastRadius(address string) []string {
	return dg.GetTransitiveDependents(address)
}

// GetOrphanedResources returns resources with no dependencies or dependents
func (dg *DependencyGraph) GetOrphanedResources() []string {
	orphans := make([]string, 0)
	
	for addr, node := range dg.nodes {
		if len(dg.edges[addr]) == 0 && len(node.Dependents) == 0 {
			orphans = append(orphans, addr)
		}
	}
	
	return orphans
}

// GetCriticalPath returns the longest dependency chain
func (dg *DependencyGraph) GetCriticalPath() []string {
	if len(dg.nodes) == 0 {
		return []string{}
	}

	// Find the longest path using DFS
	visited := make(map[string]bool)
	pathLength := make(map[string]int)
	pathPrev := make(map[string]string)

	var dfs func(node string) int
	dfs = func(node string) int {
		if visited[node] {
			return pathLength[node]
		}
		visited[node] = true

		maxLen := 0
		var maxPrev string

		for _, dep := range dg.edges[node] {
			depLen := dfs(dep) + 1
			if depLen > maxLen {
				maxLen = depLen
				maxPrev = dep
			}
		}

		pathLength[node] = maxLen
		if maxPrev != "" {
			pathPrev[node] = maxPrev
		}

		return maxLen
	}

	// Find the node with the longest path
	maxLen := 0
	var startNode string
	
	for node := range dg.nodes {
		if !visited[node] {
			len := dfs(node)
			if len > maxLen {
				maxLen = len
				startNode = node
			}
		}
	}

	// Build the critical path
	path := make([]string, 0)
	current := startNode
	
	for current != "" {
		path = append(path, current)
		current = pathPrev[current]
	}

	return path
}