package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/fingerprint"
	"github.com/catherinevee/driftmgr/internal/state/aggregator"
)

type NodeType string

const (
	NodeTypeResource      NodeType = "resource"
	NodeTypeDataSource    NodeType = "data_source"
	NodeTypeModule        NodeType = "module"
	NodeTypeProvider      NodeType = "provider"
	NodeTypeVirtualGroup  NodeType = "virtual_group"
)

type EdgeType string

const (
	EdgeTypeDependsOn     EdgeType = "depends_on"
	EdgeTypeReferences    EdgeType = "references"
	EdgeTypeImplicit      EdgeType = "implicit"
	EdgeTypeNetwork       EdgeType = "network"
	EdgeTypeIAM           EdgeType = "iam"
	EdgeTypeDataFlow      EdgeType = "data_flow"
	EdgeTypeParentChild   EdgeType = "parent_child"
	EdgeTypeSecurityGroup EdgeType = "security_group"
)

type Node struct {
	ID         string                 `json:"id"`
	Type       NodeType               `json:"type"`
	Provider   string                 `json:"provider"`
	Region     string                 `json:"region"`
	ResourceID string                 `json:"resource_id"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Metadata   NodeMetadata           `json:"metadata"`
	Score      float64                `json:"score"`
}

type NodeMetadata struct {
	CreatedAt        time.Time              `json:"created_at"`
	ModifiedAt       time.Time              `json:"modified_at"`
	StateFile        string                 `json:"state_file,omitempty"`
	Module           string                 `json:"module,omitempty"`
	ImportCandidate  bool                   `json:"import_candidate"`
	DriftStatus      string                 `json:"drift_status,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
	Cost             float64                `json:"cost,omitempty"`
	Compliance       map[string]bool        `json:"compliance,omitempty"`
	Fingerprint      *fingerprint.Fingerprint `json:"fingerprint,omitempty"`
}

type Edge struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       EdgeType               `json:"type"`
	Weight     float64                `json:"weight"`
	Properties map[string]interface{} `json:"properties"`
}

type ResourceGraph struct {
	mu              sync.RWMutex
	nodes           map[string]*Node
	edges           map[string]*Edge
	adjacencyList   map[string][]string
	reverseAdjList  map[string][]string
	nodesByType     map[NodeType][]*Node
	nodesByProvider map[string][]*Node
	nodesByRegion   map[string][]*Node
	fingerprinter   *fingerprint.ResourceFingerprinter
	aggregator      *aggregator.StateAggregator
}

func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		nodes:           make(map[string]*Node),
		edges:           make(map[string]*Edge),
		adjacencyList:   make(map[string][]string),
		reverseAdjList:  make(map[string][]string),
		nodesByType:     make(map[NodeType][]*Node),
		nodesByProvider: make(map[string][]*Node),
		nodesByRegion:   make(map[string][]*Node),
		fingerprinter:   fingerprint.NewResourceFingerprinter(),
		aggregator:      aggregator.NewStateAggregator(),
	}
}

func (g *ResourceGraph) AddNode(node *Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}

	g.nodes[node.ID] = node
	g.nodesByType[node.Type] = append(g.nodesByType[node.Type], node)
	g.nodesByProvider[node.Provider] = append(g.nodesByProvider[node.Provider], node)
	g.nodesByRegion[node.Region] = append(g.nodesByRegion[node.Region], node)

	if _, ok := g.adjacencyList[node.ID]; !ok {
		g.adjacencyList[node.ID] = []string{}
	}
	if _, ok := g.reverseAdjList[node.ID]; !ok {
		g.reverseAdjList[node.ID] = []string{}
	}

	return nil
}

func (g *ResourceGraph) AddEdge(edge *Edge) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.nodes[edge.Source]; !ok {
		return fmt.Errorf("source node %s not found", edge.Source)
	}
	if _, ok := g.nodes[edge.Target]; !ok {
		return fmt.Errorf("target node %s not found", edge.Target)
	}

	g.edges[edge.ID] = edge
	g.adjacencyList[edge.Source] = append(g.adjacencyList[edge.Source], edge.Target)
	g.reverseAdjList[edge.Target] = append(g.reverseAdjList[edge.Target], edge.Source)

	return nil
}

func (g *ResourceGraph) BuildFromResources(ctx context.Context, resources []models.Resource) error {
	for _, resource := range resources {
		node := g.resourceToNode(resource)
		if err := g.AddNode(node); err != nil {
			return fmt.Errorf("failed to add node: %w", err)
		}
	}

	if err := g.detectRelationships(ctx); err != nil {
		return fmt.Errorf("failed to detect relationships: %w", err)
	}

	return nil
}

func (g *ResourceGraph) resourceToNode(resource models.Resource) *Node {
	fingerprintResult := g.fingerprinter.FingerprintResource(resource)
	
	node := &Node{
		ID:         fmt.Sprintf("%s/%s/%s", resource.Provider, resource.Type, resource.ID),
		Type:       NodeTypeResource,
		Provider:   resource.Provider,
		Region:     resource.Region,
		ResourceID: resource.ID,
		Name:       resource.Name,
		Properties: resource.Properties,
		Metadata: NodeMetadata{
			CreatedAt:       resource.CreatedAt,
			ModifiedAt:      resource.LastModified,
			StateFile:       resource.StateFile,
			Module:          resource.Module,
			ImportCandidate: resource.ImportScore > 70,
			DriftStatus:     resource.DriftStatus,
			Tags:            resource.Tags,
			Cost:            resource.EstimatedCost,
			Compliance:      resource.ComplianceStatus,
			Fingerprint:     fingerprintResult.Fingerprint,
		},
		Score: resource.ImportScore,
	}

	return node
}

func (g *ResourceGraph) detectRelationships(ctx context.Context) error {
	g.mu.RLock()
	nodes := make([]*Node, 0, len(g.nodes))
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	g.mu.RUnlock()

	for i, source := range nodes {
		for j, target := range nodes {
			if i == j {
				continue
			}

			relationships := g.analyzeRelationship(source, target)
			for _, rel := range relationships {
				edge := &Edge{
					ID:         fmt.Sprintf("%s-%s-%s", source.ID, rel.Type, target.ID),
					Source:     source.ID,
					Target:     target.ID,
					Type:       rel.Type,
					Weight:     rel.Weight,
					Properties: rel.Properties,
				}
				if err := g.AddEdge(edge); err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
			}
		}
	}

	return nil
}

type RelationshipInfo struct {
	Type       EdgeType
	Weight     float64
	Properties map[string]interface{}
}

func (g *ResourceGraph) analyzeRelationship(source, target *Node) []RelationshipInfo {
	var relationships []RelationshipInfo

	// Check for VPC/Network relationships
	if g.isNetworkRelationship(source, target) {
		relationships = append(relationships, RelationshipInfo{
			Type:   EdgeTypeNetwork,
			Weight: 0.9,
			Properties: map[string]interface{}{
				"type": "network_association",
			},
		})
	}

	// Check for IAM relationships
	if g.isIAMRelationship(source, target) {
		relationships = append(relationships, RelationshipInfo{
			Type:   EdgeTypeIAM,
			Weight: 0.8,
			Properties: map[string]interface{}{
				"type": "iam_binding",
			},
		})
	}

	// Check for parent-child relationships
	if g.isParentChildRelationship(source, target) {
		relationships = append(relationships, RelationshipInfo{
			Type:   EdgeTypeParentChild,
			Weight: 1.0,
			Properties: map[string]interface{}{
				"type": "hierarchy",
			},
		})
	}

	// Check for security group associations
	if g.isSecurityGroupRelationship(source, target) {
		relationships = append(relationships, RelationshipInfo{
			Type:   EdgeTypeSecurityGroup,
			Weight: 0.7,
			Properties: map[string]interface{}{
				"type": "security_association",
			},
		})
	}

	// Check for data flow relationships
	if g.isDataFlowRelationship(source, target) {
		relationships = append(relationships, RelationshipInfo{
			Type:   EdgeTypeDataFlow,
			Weight: 0.6,
			Properties: map[string]interface{}{
				"type": "data_dependency",
			},
		})
	}

	// Check for implicit references
	if g.hasImplicitReference(source, target) {
		relationships = append(relationships, RelationshipInfo{
			Type:   EdgeTypeImplicit,
			Weight: 0.5,
			Properties: map[string]interface{}{
				"type": "implicit_dependency",
			},
		})
	}

	return relationships
}

func (g *ResourceGraph) isNetworkRelationship(source, target *Node) bool {
	sourceType := strings.ToLower(source.Properties["type"].(string))
	targetType := strings.ToLower(target.Properties["type"].(string))

	networkTypes := []string{"vpc", "subnet", "network", "vnet", "route", "gateway", "loadbalancer", "lb"}
	
	for _, netType := range networkTypes {
		if strings.Contains(sourceType, netType) || strings.Contains(targetType, netType) {
			if source.Region == target.Region {
				return true
			}
		}
	}

	// Check for VPC ID references
	if vpcID, ok := source.Properties["vpc_id"]; ok {
		if targetVPC, ok := target.Properties["id"]; ok && vpcID == targetVPC {
			return true
		}
	}

	return false
}

func (g *ResourceGraph) isIAMRelationship(source, target *Node) bool {
	sourceType := strings.ToLower(source.Properties["type"].(string))
	targetType := strings.ToLower(target.Properties["type"].(string))

	iamTypes := []string{"role", "policy", "user", "group", "permission", "identity", "service_account"}
	
	for _, iamType := range iamTypes {
		if strings.Contains(sourceType, iamType) || strings.Contains(targetType, iamType) {
			return true
		}
	}

	// Check for role ARN references
	if roleARN, ok := source.Properties["role_arn"]; ok {
		if targetARN, ok := target.Properties["arn"]; ok && roleARN == targetARN {
			return true
		}
	}

	return false
}

func (g *ResourceGraph) isParentChildRelationship(source, target *Node) bool {
	// Check for explicit parent references
	if parentID, ok := target.Properties["parent_id"]; ok {
		if sourceID, ok := source.Properties["id"]; ok && parentID == sourceID {
			return true
		}
	}

	// Check for resource group memberships
	if resourceGroup, ok := target.Properties["resource_group"]; ok {
		if sourceGroup, ok := source.Properties["name"]; ok && resourceGroup == sourceGroup {
			return true
		}
	}

	return false
}

func (g *ResourceGraph) isSecurityGroupRelationship(source, target *Node) bool {
	sourceType := strings.ToLower(source.Properties["type"].(string))
	targetType := strings.ToLower(target.Properties["type"].(string))

	if strings.Contains(sourceType, "security") || strings.Contains(targetType, "security") {
		// Check for security group IDs
		if sgIDs, ok := target.Properties["security_group_ids"].([]interface{}); ok {
			if sourceID, ok := source.Properties["id"]; ok {
				for _, sgID := range sgIDs {
					if sgID == sourceID {
						return true
					}
				}
			}
		}
		return true
	}

	return false
}

func (g *ResourceGraph) isDataFlowRelationship(source, target *Node) bool {
	sourceType := strings.ToLower(source.Properties["type"].(string))
	targetType := strings.ToLower(target.Properties["type"].(string))

	dataTypes := []string{"database", "storage", "queue", "stream", "topic", "bucket", "table"}
	
	for _, dataType := range dataTypes {
		if strings.Contains(sourceType, dataType) || strings.Contains(targetType, dataType) {
			return true
		}
	}

	// Check for endpoint connections
	if endpoint, ok := target.Properties["endpoint"]; ok {
		if sourceEndpoint, ok := source.Properties["endpoint"]; ok && endpoint == sourceEndpoint {
			return true
		}
	}

	return false
}

func (g *ResourceGraph) hasImplicitReference(source, target *Node) bool {
	// Check for tag similarities
	if sourceTags, ok := source.Metadata.Tags; ok {
		if targetTags, ok := target.Metadata.Tags; ok {
			matchCount := 0
			for key, value := range sourceTags {
				if targetValue, exists := targetTags[key]; exists && targetValue == value {
					matchCount++
				}
			}
			if matchCount >= 2 {
				return true
			}
		}
	}

	// Check for naming conventions
	if strings.HasPrefix(target.Name, source.Name) || strings.HasPrefix(source.Name, target.Name) {
		return true
	}

	// Check for same module
	if source.Metadata.Module != "" && source.Metadata.Module == target.Metadata.Module {
		return true
	}

	return false
}

func (g *ResourceGraph) FindConnectedComponents() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	var components [][]string

	for nodeID := range g.nodes {
		if !visited[nodeID] {
			component := g.dfs(nodeID, visited)
			components = append(components, component)
		}
	}

	// Sort components by size (largest first)
	sort.Slice(components, func(i, j int) bool {
		return len(components[i]) > len(components[j])
	})

	return components
}

func (g *ResourceGraph) dfs(nodeID string, visited map[string]bool) []string {
	var component []string
	stack := []string{nodeID}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current] {
			continue
		}

		visited[current] = true
		component = append(component, current)

		// Add neighbors (both directions for undirected traversal)
		for _, neighbor := range g.adjacencyList[current] {
			if !visited[neighbor] {
				stack = append(stack, neighbor)
			}
		}
		for _, neighbor := range g.reverseAdjList[current] {
			if !visited[neighbor] {
				stack = append(stack, neighbor)
			}
		}
	}

	return component
}

func (g *ResourceGraph) FindCriticalNodes() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	criticalNodes := []*Node{}
	
	for nodeID, node := range g.nodes {
		// Calculate centrality metrics
		inDegree := len(g.reverseAdjList[nodeID])
		outDegree := len(g.adjacencyList[nodeID])
		totalDegree := inDegree + outDegree

		// Node is critical if it has high connectivity
		if totalDegree >= 5 {
			criticalNodes = append(criticalNodes, node)
		}

		// Node is critical if it's a gateway (high betweenness)
		if inDegree > 0 && outDegree > 0 && totalDegree >= 3 {
			criticalNodes = append(criticalNodes, node)
		}
	}

	// Sort by total degree (most connected first)
	sort.Slice(criticalNodes, func(i, j int) bool {
		iDegree := len(g.adjacencyList[criticalNodes[i].ID]) + len(g.reverseAdjList[criticalNodes[i].ID])
		jDegree := len(g.adjacencyList[criticalNodes[j].ID]) + len(g.reverseAdjList[criticalNodes[j].ID])
		return iDegree > jDegree
	})

	return criticalNodes
}

func (g *ResourceGraph) FindOrphanedResources() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	orphaned := []*Node{}
	
	for nodeID, node := range g.nodes {
		inDegree := len(g.reverseAdjList[nodeID])
		outDegree := len(g.adjacencyList[nodeID])
		
		// Resource is orphaned if it has no connections
		if inDegree == 0 && outDegree == 0 {
			orphaned = append(orphaned, node)
		}
	}

	return orphaned
}

func (g *ResourceGraph) FindImportCandidates(minScore float64) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	candidates := []*Node{}
	
	for _, node := range g.nodes {
		if node.Metadata.ImportCandidate && node.Score >= minScore {
			candidates = append(candidates, node)
		}
	}

	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}

func (g *ResourceGraph) GetNodesByType(nodeType NodeType) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	return g.nodesByType[nodeType]
}

func (g *ResourceGraph) GetNodesByProvider(provider string) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	return g.nodesByProvider[provider]
}

func (g *ResourceGraph) GetNodesByRegion(region string) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	return g.nodesByRegion[region]
}

func (g *ResourceGraph) GetNode(nodeID string) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	node, exists := g.nodes[nodeID]
	return node, exists
}

func (g *ResourceGraph) GetNeighbors(nodeID string) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	neighbors := []*Node{}
	
	for _, neighborID := range g.adjacencyList[nodeID] {
		if node, exists := g.nodes[neighborID]; exists {
			neighbors = append(neighbors, node)
		}
	}
	
	return neighbors
}

func (g *ResourceGraph) GetDependents(nodeID string) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	dependents := []*Node{}
	
	for _, dependentID := range g.reverseAdjList[nodeID] {
		if node, exists := g.nodes[dependentID]; exists {
			dependents = append(dependents, node)
		}
	}
	
	return dependents
}

func (g *ResourceGraph) ExportToJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	graphData := map[string]interface{}{
		"nodes": g.nodes,
		"edges": g.edges,
		"metadata": map[string]interface{}{
			"total_nodes":      len(g.nodes),
			"total_edges":      len(g.edges),
			"providers":        len(g.nodesByProvider),
			"regions":          len(g.nodesByRegion),
			"orphaned_count":   len(g.FindOrphanedResources()),
			"import_candidates": len(g.FindImportCandidates(70)),
		},
	}

	return json.MarshalIndent(graphData, "", "  ")
}

func (g *ResourceGraph) AnalyzeBlastRadius(nodeID string) map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	affected := make(map[string]bool)
	visited := make(map[string]bool)
	
	// BFS to find all potentially affected resources
	queue := []string{nodeID}
	levels := map[string]int{nodeID: 0}
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		if visited[current] {
			continue
		}
		visited[current] = true
		affected[current] = true
		
		currentLevel := levels[current]
		if currentLevel >= 3 { // Limit depth to 3 levels
			continue
		}
		
		// Add all dependent resources
		for _, dependent := range g.adjacencyList[current] {
			if !visited[dependent] {
				queue = append(queue, dependent)
				levels[dependent] = currentLevel + 1
			}
		}
	}
	
	// Categorize affected resources
	affectedByType := make(map[NodeType]int)
	affectedByProvider := make(map[string]int)
	criticalResources := []*Node{}
	
	for affectedID := range affected {
		if node, exists := g.nodes[affectedID]; exists {
			affectedByType[node.Type]++
			affectedByProvider[node.Provider]++
			
			// Check if it's a critical resource
			degree := len(g.adjacencyList[affectedID]) + len(g.reverseAdjList[affectedID])
			if degree >= 5 {
				criticalResources = append(criticalResources, node)
			}
		}
	}
	
	return map[string]interface{}{
		"total_affected":     len(affected),
		"affected_by_type":   affectedByType,
		"affected_by_provider": affectedByProvider,
		"critical_resources": criticalResources,
		"max_depth":         3,
	}
}