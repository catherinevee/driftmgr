package cache

import (
	"fmt"
	"strings"
)

// NewRelationshipCache creates a new relationship cache
func NewRelationshipCache() *RelationshipCache {
	return &RelationshipCache{
		dependencies:    make(map[string][]string),
		parentChild:     make(map[string][]string),
		crossProvider:   make(map[string][]string),
		networkTopology: make(map[string]*NetworkNode),
		iamRelations:    make(map[string]*IAMRelation),
	}
}

// UpdateRelationships updates all relationships for a cache entry
func (rc *RelationshipCache) UpdateRelationships(key string, entry *CacheEntry) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	// Extract resource information
	if entry.Metadata == nil {
		return
	}
	
	resourceType := entry.Metadata.ResourceType
	provider := entry.Metadata.Provider
	
	// Update dependencies based on resource type
	rc.updateDependencies(key, entry, resourceType)
	
	// Update parent-child relationships
	rc.updateParentChild(key, entry, resourceType)
	
	// Update cross-provider correlations
	rc.updateCrossProvider(key, entry, provider)
	
	// Update network topology for network resources
	if isNetworkResource(resourceType) {
		rc.updateNetworkTopology(key, entry)
	}
	
	// Update IAM relations for IAM resources
	if isIAMResource(resourceType) {
		rc.updateIAMRelations(key, entry)
	}
}

// GetRelationships retrieves all relationships for a key
func (rc *RelationshipCache) GetRelationships(key string) []Relationship {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	relationships := make([]Relationship, 0)
	
	// Add dependencies
	if deps, exists := rc.dependencies[key]; exists {
		for _, dep := range deps {
			relationships = append(relationships, Relationship{
				Type:      "dependency",
				Direction: "outbound",
				TargetKey: dep,
				Strength:  1.0,
			})
		}
	}
	
	// Add reverse dependencies
	for source, deps := range rc.dependencies {
		for _, dep := range deps {
			if dep == key {
				relationships = append(relationships, Relationship{
					Type:      "dependency",
					Direction: "inbound",
					TargetKey: source,
					Strength:  1.0,
				})
			}
		}
	}
	
	// Add parent-child relationships
	if children, exists := rc.parentChild[key]; exists {
		for _, child := range children {
			relationships = append(relationships, Relationship{
				Type:      "parent_child",
				Direction: "child",
				TargetKey: child,
				Strength:  1.0,
			})
		}
	}
	
	// Add cross-provider correlations
	if correlated, exists := rc.crossProvider[key]; exists {
		for _, target := range correlated {
			relationships = append(relationships, Relationship{
				Type:      "cross_provider",
				Direction: "bidirectional",
				TargetKey: target,
				Strength:  0.8,
			})
		}
	}
	
	return relationships
}

// GetDependencyGraph returns the complete dependency graph
func (rc *RelationshipCache) GetDependencyGraph() map[string][]string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	// Create a copy to avoid external modifications
	graph := make(map[string][]string)
	for k, v := range rc.dependencies {
		deps := make([]string, len(v))
		copy(deps, v)
		graph[k] = deps
	}
	
	return graph
}

// GetNetworkTopology returns network topology information
func (rc *RelationshipCache) GetNetworkTopology() map[string]*NetworkNode {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	// Create a copy to avoid external modifications
	topology := make(map[string]*NetworkNode)
	for k, v := range rc.networkTopology {
		topology[k] = v
	}
	
	return topology
}

// GetIAMRelations returns IAM relationship information
func (rc *RelationshipCache) GetIAMRelations() map[string]*IAMRelation {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	// Create a copy to avoid external modifications
	relations := make(map[string]*IAMRelation)
	for k, v := range rc.iamRelations {
		relations[k] = v
	}
	
	return relations
}

// InvalidatePattern invalidates relationships matching pattern
func (rc *RelationshipCache) InvalidatePattern(pattern string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	// Clean up dependencies
	for key := range rc.dependencies {
		if matchesPattern(key, pattern) {
			delete(rc.dependencies, key)
		}
	}
	
	// Clean up parent-child
	for key := range rc.parentChild {
		if matchesPattern(key, pattern) {
			delete(rc.parentChild, key)
		}
	}
	
	// Clean up cross-provider
	for key := range rc.crossProvider {
		if matchesPattern(key, pattern) {
			delete(rc.crossProvider, key)
		}
	}
	
	// Clean up network topology
	for key := range rc.networkTopology {
		if matchesPattern(key, pattern) {
			delete(rc.networkTopology, key)
		}
	}
	
	// Clean up IAM relations
	for key := range rc.iamRelations {
		if matchesPattern(key, pattern) {
			delete(rc.iamRelations, key)
		}
	}
}

// FindCycles detects circular dependencies
func (rc *RelationshipCache) FindCycles() [][]string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	cycles := make([][]string, 0)
	
	for node := range rc.dependencies {
		if !visited[node] {
			if path := rc.findCyclesDFS(node, visited, recStack, []string{}); path != nil {
				cycles = append(cycles, path)
			}
		}
	}
	
	return cycles
}

// GetImpactRadius calculates resources affected by changes to a resource
func (rc *RelationshipCache) GetImpactRadius(key string, maxDepth int) []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	impacted := make(map[string]bool)
	rc.calculateImpact(key, impacted, 0, maxDepth)
	
	result := make([]string, 0, len(impacted))
	for k := range impacted {
		if k != key {
			result = append(result, k)
		}
	}
	
	return result
}

// CorrelateAcrossProviders finds resources that represent the same logical entity
func (rc *RelationshipCache) CorrelateAcrossProviders(entry *CacheEntry) []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	return rc.correlateAcrossProvidersInternal(entry)
}

// correlateAcrossProvidersInternal is the internal version that doesn't acquire locks
// Must be called with lock already held
func (rc *RelationshipCache) correlateAcrossProvidersInternal(entry *CacheEntry) []string {
	correlated := make([]string, 0)
	
	// Extract identifying information from the entry
	if entry.Metadata == nil {
		return correlated
	}
	
	// Look for resources with similar names or tags across different providers
	resourceName := extractResourceName(entry)
	resourceTags := entry.Tags
	
	for key, node := range rc.networkTopology {
		// Skip same provider
		if extractProvider(key) == entry.Metadata.Provider {
			continue
		}
		
		// Check for name similarity
		if similarNames(resourceName, extractResourceName2(node.ID)) {
			correlated = append(correlated, key)
			continue
		}
		
		// Check for matching tags
		if hasMatchingTags(resourceTags, extractTags(node)) {
			correlated = append(correlated, key)
		}
	}
	
	return correlated
}

// Private helper methods

func (rc *RelationshipCache) updateDependencies(key string, entry *CacheEntry, resourceType string) {
	// Extract dependencies from resource configuration
	if entry.Value == nil {
		return
	}
	
	// Parse resource configuration to find references to other resources
	deps := extractDependencies(entry.Value, resourceType)
	
	if len(deps) > 0 {
		rc.dependencies[key] = deps
	}
}

func (rc *RelationshipCache) updateParentChild(key string, entry *CacheEntry, resourceType string) {
	// Determine parent-child relationships based on resource type
	parent := extractParentResource(entry.Value, resourceType)
	
	if parent != "" {
		// Add this resource as child of parent
		if rc.parentChild[parent] == nil {
			rc.parentChild[parent] = make([]string, 0)
		}
		rc.parentChild[parent] = append(rc.parentChild[parent], key)
	}
}

func (rc *RelationshipCache) updateCrossProvider(key string, entry *CacheEntry, provider string) {
	// Find resources with similar purpose across providers
	// Note: We're already holding the lock, so use internal version
	correlated := rc.correlateAcrossProvidersInternal(entry)
	
	if len(correlated) > 0 {
		rc.crossProvider[key] = correlated
		
		// Add bidirectional relationships
		for _, target := range correlated {
			if rc.crossProvider[target] == nil {
				rc.crossProvider[target] = make([]string, 0)
			}
			rc.crossProvider[target] = append(rc.crossProvider[target], key)
		}
	}
}

func (rc *RelationshipCache) updateNetworkTopology(key string, entry *CacheEntry) {
	node := &NetworkNode{
		ID:          key,
		Type:        entry.Metadata.ResourceType,
		Connections: extractNetworkConnections(entry.Value),
	}
	
	// Extract security rules if applicable
	if hasSecurityRules(entry.Metadata.ResourceType) {
		node.SecurityRules = extractSecurityRules(entry.Value)
	}
	
	// Extract route tables if applicable
	if hasRouteTables(entry.Metadata.ResourceType) {
		node.RouteTables = extractRouteTables(entry.Value)
	}
	
	rc.networkTopology[key] = node
}

func (rc *RelationshipCache) updateIAMRelations(key string, entry *CacheEntry) {
	relation := &IAMRelation{
		Principal: extractPrincipal(entry.Value),
		Resources: extractIAMResources(entry.Value),
		Actions:   extractIAMActions(entry.Value),
		Effect:    extractIAMEffect(entry.Value),
	}
	
	rc.iamRelations[key] = relation
}

func (rc *RelationshipCache) findCyclesDFS(node string, visited, recStack map[string]bool, path []string) []string {
	visited[node] = true
	recStack[node] = true
	path = append(path, node)
	
	if deps, exists := rc.dependencies[node]; exists {
		for _, dep := range deps {
			if !visited[dep] {
				if cycle := rc.findCyclesDFS(dep, visited, recStack, path); cycle != nil {
					return cycle
				}
			} else if recStack[dep] {
				// Found a cycle
				cycleStart := -1
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					return path[cycleStart:]
				}
			}
		}
	}
	
	recStack[node] = false
	return nil
}

func (rc *RelationshipCache) calculateImpact(key string, impacted map[string]bool, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}
	
	impacted[key] = true
	
	// Check direct dependencies
	if deps, exists := rc.dependencies[key]; exists {
		for _, dep := range deps {
			if !impacted[dep] {
				rc.calculateImpact(dep, impacted, depth+1, maxDepth)
			}
		}
	}
	
	// Check children
	if children, exists := rc.parentChild[key]; exists {
		for _, child := range children {
			if !impacted[child] {
				rc.calculateImpact(child, impacted, depth+1, maxDepth)
			}
		}
	}
	
	// Check reverse dependencies
	for source, deps := range rc.dependencies {
		for _, dep := range deps {
			if dep == key && !impacted[source] {
				rc.calculateImpact(source, impacted, depth+1, maxDepth)
			}
		}
	}
}

// Helper functions for resource type checking

func isNetworkResource(resourceType string) bool {
	networkTypes := []string{
		"vpc", "subnet", "security_group", "network_acl", "route_table",
		"internet_gateway", "nat_gateway", "vpn_connection", "load_balancer",
		"network_interface", "elastic_ip", "vnet", "nsg", "firewall",
	}
	
	resourceTypeLower := strings.ToLower(resourceType)
	for _, nt := range networkTypes {
		if strings.Contains(resourceTypeLower, nt) {
			return true
		}
	}
	
	return false
}

func isIAMResource(resourceType string) bool {
	iamTypes := []string{
		"iam_role", "iam_policy", "iam_user", "iam_group",
		"service_account", "role_assignment", "managed_identity",
		"access_key", "permission", "grant",
	}
	
	resourceTypeLower := strings.ToLower(resourceType)
	for _, it := range iamTypes {
		if strings.Contains(resourceTypeLower, it) {
			return true
		}
	}
	
	return false
}

func hasSecurityRules(resourceType string) bool {
	return strings.Contains(strings.ToLower(resourceType), "security_group") ||
		   strings.Contains(strings.ToLower(resourceType), "firewall") ||
		   strings.Contains(strings.ToLower(resourceType), "nsg")
}

func hasRouteTables(resourceType string) bool {
	return strings.Contains(strings.ToLower(resourceType), "route") ||
		   strings.Contains(strings.ToLower(resourceType), "vpc") ||
		   strings.Contains(strings.ToLower(resourceType), "vnet")
}

// Data extraction helper functions

func extractDependencies(value interface{}, resourceType string) []string {
	deps := make([]string, 0)
	
	// Type-specific dependency extraction logic
	if data, ok := value.(map[string]interface{}); ok {
		// Look for common dependency patterns
		if refs, exists := data["references"]; exists {
			if refList, ok := refs.([]interface{}); ok {
				for _, ref := range refList {
					if refStr, ok := ref.(string); ok {
						deps = append(deps, refStr)
					}
				}
			}
		}
		
		// Check for subnet_id, vpc_id, security_group_ids, etc.
		dependencyFields := []string{
			"subnet_id", "vpc_id", "security_group_ids", "role_arn",
			"target_group_arn", "cluster_id", "db_subnet_group_name",
		}
		
		for _, field := range dependencyFields {
			if val, exists := data[field]; exists {
				switch v := val.(type) {
				case string:
					if v != "" {
						deps = append(deps, v)
					}
				case []interface{}:
					for _, item := range v {
						if str, ok := item.(string); ok && str != "" {
							deps = append(deps, str)
						}
					}
				}
			}
		}
	}
	
	return deps
}

func extractParentResource(value interface{}, resourceType string) string {
	if data, ok := value.(map[string]interface{}); ok {
		// Look for parent resource indicators
		parentFields := []string{"parent", "parent_id", "vpc_id", "cluster_id", "resource_group"}
		
		for _, field := range parentFields {
			if val, exists := data[field]; exists {
				if str, ok := val.(string); ok && str != "" {
					return str
				}
			}
		}
	}
	
	return ""
}

func extractNetworkConnections(value interface{}) []string {
	connections := make([]string, 0)
	
	if data, ok := value.(map[string]interface{}); ok {
		// Look for connection-related fields
		connectionFields := []string{
			"peer_vpc_id", "transit_gateway_id", "vpn_gateway_id",
			"target_id", "destination_id", "endpoint_id",
		}
		
		for _, field := range connectionFields {
			if val, exists := data[field]; exists {
				if str, ok := val.(string); ok && str != "" {
					connections = append(connections, str)
				}
			}
		}
	}
	
	return connections
}

func extractSecurityRules(value interface{}) []SecurityRule {
	rules := make([]SecurityRule, 0)
	
	if data, ok := value.(map[string]interface{}); ok {
		// Look for ingress/egress rules
		if ingressRules, exists := data["ingress"]; exists {
			if ruleList, ok := ingressRules.([]interface{}); ok {
				for _, rule := range ruleList {
					if ruleData, ok := rule.(map[string]interface{}); ok {
						rules = append(rules, SecurityRule{
							Direction: "ingress",
							Protocol:  extractString(ruleData, "protocol"),
							Port:      extractInt(ruleData, "from_port"),
							Source:    extractString(ruleData, "cidr_blocks"),
						})
					}
				}
			}
		}
		
		if egressRules, exists := data["egress"]; exists {
			if ruleList, ok := egressRules.([]interface{}); ok {
				for _, rule := range ruleList {
					if ruleData, ok := rule.(map[string]interface{}); ok {
						rules = append(rules, SecurityRule{
							Direction: "egress",
							Protocol:  extractString(ruleData, "protocol"),
							Port:      extractInt(ruleData, "from_port"),
							Target:    extractString(ruleData, "cidr_blocks"),
						})
					}
				}
			}
		}
	}
	
	return rules
}

func extractRouteTables(value interface{}) []RouteTable {
	routes := make([]RouteTable, 0)
	
	if data, ok := value.(map[string]interface{}); ok {
		if routeList, exists := data["routes"]; exists {
			if routes, ok := routeList.([]interface{}); ok {
				for _, route := range routes {
					if routeData, ok := route.(map[string]interface{}); ok {
						routes = append(routes, RouteTable{
							Destination: extractString(routeData, "destination_cidr_block"),
							Target:      extractString(routeData, "gateway_id"),
							Type:        extractString(routeData, "type"),
						})
					}
				}
			}
		}
	}
	
	return routes
}

func extractPrincipal(value interface{}) string {
	if data, ok := value.(map[string]interface{}); ok {
		return extractString(data, "principal")
	}
	return ""
}

func extractIAMResources(value interface{}) []string {
	resources := make([]string, 0)
	
	if data, ok := value.(map[string]interface{}); ok {
		if res, exists := data["resources"]; exists {
			if resList, ok := res.([]interface{}); ok {
				for _, r := range resList {
					if str, ok := r.(string); ok {
						resources = append(resources, str)
					}
				}
			}
		}
	}
	
	return resources
}

func extractIAMActions(value interface{}) []string {
	actions := make([]string, 0)
	
	if data, ok := value.(map[string]interface{}); ok {
		if acts, exists := data["actions"]; exists {
			if actList, ok := acts.([]interface{}); ok {
				for _, a := range actList {
					if str, ok := a.(string); ok {
						actions = append(actions, str)
					}
				}
			}
		}
	}
	
	return actions
}

func extractIAMEffect(value interface{}) string {
	if data, ok := value.(map[string]interface{}); ok {
		return extractString(data, "effect")
	}
	return "Allow"
}

func extractResourceName(entry *CacheEntry) string {
	if entry.Metadata != nil && entry.Metadata.Custom != nil {
		if name, exists := entry.Metadata.Custom["name"]; exists {
			return name
		}
	}
	
	if data, ok := entry.Value.(map[string]interface{}); ok {
		return extractString(data, "name")
	}
	
	return ""
}

func extractResourceName2(id string) string {
	// Extract resource name from ID
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return id
}

func extractProvider(key string) string {
	// Extract provider from key format
	parts := strings.Split(key, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func extractTags(node *NetworkNode) map[string]string {
	if node == nil {
		return make(map[string]string)
	}

	tags := make(map[string]string)

	// NetworkNode has limited fields - just extract what we have
	if node.ID != "" {
		tags["ID"] = node.ID
	}
	if node.Type != "" {
		tags["ResourceType"] = node.Type
	}

	// Add connection count as a tag
	if len(node.Connections) > 0 {
		tags["ConnectionCount"] = fmt.Sprintf("%d", len(node.Connections))
	}

	// Add security rules count
	if len(node.SecurityRules) > 0 {
		tags["SecurityRuleCount"] = fmt.Sprintf("%d", len(node.SecurityRules))
	}

	return tags
}

func similarNames(name1, name2 string) bool {
	// Simple similarity check - can be enhanced with fuzzy matching
	name1 = strings.ToLower(name1)
	name2 = strings.ToLower(name2)
	
	if name1 == name2 {
		return true
	}
	
	// Check if one contains the other
	if strings.Contains(name1, name2) || strings.Contains(name2, name1) {
		return true
	}
	
	// Check for common patterns (e.g., prod-web-01 and production-web-01)
	return haveSimilarPattern(name1, name2)
}

func hasMatchingTags(tags1, tags2 map[string]string) bool {
	if len(tags1) == 0 || len(tags2) == 0 {
		return false
	}
	
	matchCount := 0
	for k, v := range tags1 {
		if v2, exists := tags2[k]; exists && v == v2 {
			matchCount++
		}
	}
	
	// Consider matching if at least 2 tags match or 50% of tags match
	return matchCount >= 2 || float64(matchCount)/float64(len(tags1)) >= 0.5
}

func haveSimilarPattern(s1, s2 string) bool {
	// Remove common variations
	replacements := [][]string{
		{"prod", "production"},
		{"dev", "development"},
		{"test", "testing"},
		{"stg", "staging"},
	}
	
	for _, pair := range replacements {
		s1 = strings.ReplaceAll(s1, pair[0], pair[1])
		s2 = strings.ReplaceAll(s2, pair[0], pair[1])
	}
	
	return s1 == s2
}

func extractString(data map[string]interface{}, key string) string {
	if val, exists := data[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func extractInt(data map[string]interface{}, key string) int {
	if val, exists := data[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case float32:
			return int(v)
		}
	}
	return 0
}