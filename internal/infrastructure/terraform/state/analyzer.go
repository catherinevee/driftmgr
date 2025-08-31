package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// StateAnalyzer provides deep analysis capabilities for Terraform state files
type StateAnalyzer struct {
	perspectiveCache map[string]*StatePerspective
}

// NewStateAnalyzer creates a new state analyzer
func NewStateAnalyzer() *StateAnalyzer {
	return &StateAnalyzer{
		perspectiveCache: make(map[string]*StatePerspective),
	}
}

// StatePerspective represents the view of infrastructure from a state file's perspective
type StatePerspective struct {
	StateFileID      string                   `json:"state_file_id"`
	StateFilePath    string                   `json:"state_file_path"`
	Timestamp        time.Time                `json:"timestamp"`
	ManagedResources []ManagedResource        `json:"managed_resources"`
	ResourceGraph    *ResourceGraph           `json:"resource_graph"`
	Dependencies     map[string][]string      `json:"dependencies"`
	Outputs          map[string]interface{}   `json:"outputs"`
	DataSources      []DataSource             `json:"data_sources"`
	ProviderVersions map[string]string        `json:"provider_versions"`
	Statistics       PerspectiveStatistics    `json:"statistics"`
	OutOfBand        []OutOfBandResource      `json:"out_of_band_resources"`
	Conflicts        []ResourceConflict       `json:"conflicts"`
}

// ManagedResource represents a resource managed by the state file
type ManagedResource struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Provider     string                 `json:"provider"`
	Module       string                 `json:"module,omitempty"`
	Mode         string                 `json:"mode"` // managed, data
	Attributes   map[string]interface{} `json:"attributes"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Lifecycle    LifecycleConfig        `json:"lifecycle,omitempty"`
	Count        *int                   `json:"count,omitempty"`
	ForEach      interface{}            `json:"for_each,omitempty"`
	CloudID      string                 `json:"cloud_id,omitempty"` // Actual cloud resource ID
	Status       string                 `json:"status"`              // exists, missing, drifted
}

// OutOfBandResource represents a resource not managed by the state file
type OutOfBandResource struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	Name             string                 `json:"name"`
	Provider         string                 `json:"provider"`
	Region           string                 `json:"region,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	DiscoverySource  string                 `json:"discovery_source"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	SuggestedImport  string                 `json:"suggested_import,omitempty"`
	ConflictsWith    []string               `json:"conflicts_with,omitempty"`
	AdoptionPriority string                 `json:"adoption_priority"` // high, medium, low
	Reason           string                 `json:"reason"`            // Why it's out of band
}

// ResourceGraph represents the dependency graph of resources
type ResourceGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode represents a node in the resource graph
type GraphNode struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Provider string                 `json:"provider"`
	Module   string                 `json:"module,omitempty"`
	Status   string                 `json:"status"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GraphEdge represents an edge in the resource graph
type GraphEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Type     string `json:"type"` // depends_on, references, creates
	Label    string `json:"label,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DataSource represents a data source in the state
type DataSource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"`
	Attributes map[string]interface{} `json:"attributes"`
	DependsOn  []string               `json:"depends_on,omitempty"`
}

// LifecycleConfig represents resource lifecycle configuration
type LifecycleConfig struct {
	CreateBeforeDestroy bool     `json:"create_before_destroy,omitempty"`
	PreventDestroy      bool     `json:"prevent_destroy,omitempty"`
	IgnoreChanges       []string `json:"ignore_changes,omitempty"`
}

// PerspectiveStatistics contains statistics about the perspective
type PerspectiveStatistics struct {
	TotalManaged          int            `json:"total_managed"`
	TotalOutOfBand        int            `json:"total_out_of_band"`
	TotalDataSources      int            `json:"total_data_sources"`
	ResourcesByProvider   map[string]int `json:"resources_by_provider"`
	ResourcesByType       map[string]int `json:"resources_by_type"`
	ResourcesByModule     map[string]int `json:"resources_by_module"`
	CoveragePercentage    float64        `json:"coverage_percentage"`
	DriftPercentage       float64        `json:"drift_percentage"`
	AdoptionOpportunities int            `json:"adoption_opportunities"`
}

// ResourceConflict represents a conflict between state and reality
type ResourceConflict struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	ConflictType string                 `json:"conflict_type"` // duplicate, ownership, configuration
	StateValue   interface{}            `json:"state_value"`
	ActualValue  interface{}            `json:"actual_value"`
	Severity     string                 `json:"severity"` // critical, high, medium, low
	Resolution   string                 `json:"resolution,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// AnalyzePerspective generates a perspective view from a state file
func (sa *StateAnalyzer) AnalyzePerspective(ctx context.Context, stateFile *StateFile, cloudResources []interface{}) (*StatePerspective, error) {
	perspective := &StatePerspective{
		StateFileID:      stateFile.ID,
		StateFilePath:    stateFile.Path,
		Timestamp:        time.Now(),
		ManagedResources: []ManagedResource{},
		Dependencies:     make(map[string][]string),
		Outputs:          make(map[string]interface{}),
		DataSources:      []DataSource{},
		ProviderVersions: make(map[string]string),
		OutOfBand:        []OutOfBandResource{},
		Conflicts:        []ResourceConflict{},
	}
	
	// Parse state file resources
	managedMap := make(map[string]*ManagedResource)
	for _, resource := range stateFile.Resources {
		// Convert Resource to StateResource for compatibility
		stateResource := StateResource{
			Module:    resource.Module,
			Mode:      resource.Mode,
			Type:      resource.Type,
			Name:      resource.Name,
			Provider:  resource.Provider,
		}
		// Convert Instances
		for _, inst := range resource.Instances {
			stateResource.Instances = append(stateResource.Instances, ResourceInstance{
				SchemaVersion: inst.SchemaVersion,
				Attributes:    inst.Attributes,
				Private:       inst.Private,
				Dependencies:  inst.DependsOn,
			})
		}
		managed := sa.parseStateResource(stateResource)
		perspective.ManagedResources = append(perspective.ManagedResources, managed)
		managedMap[managed.ID] = &managed
	}
	
	// Build resource graph
	perspective.ResourceGraph = sa.buildResourceGraph(perspective.ManagedResources)
	
	// Extract dependencies
	perspective.Dependencies = sa.extractDependencies(perspective.ManagedResources)
	
	// Identify out-of-band resources
	perspective.OutOfBand = sa.identifyOutOfBandResources(managedMap, cloudResources)
	
	// Detect conflicts
	perspective.Conflicts = sa.detectConflicts(managedMap, cloudResources)
	
	// Calculate statistics
	perspective.Statistics = sa.calculateStatistics(perspective)
	
	// Cache the perspective
	sa.perspectiveCache[stateFile.ID] = perspective
	
	return perspective, nil
}

// parseStateResource converts a state resource to a managed resource
func (sa *StateAnalyzer) parseStateResource(resource StateResource) ManagedResource {
	managed := ManagedResource{
		ID:           fmt.Sprintf("%s.%s", resource.Type, resource.Name),
		Type:         resource.Type,
		Name:         resource.Name,
		Provider:     resource.Provider,
		Mode:         resource.Mode,
		Attributes:   make(map[string]interface{}),
		Dependencies: []string{},
		Status:       "exists",
	}
	
	// Extract module path if present
	if resource.Module != "" {
		managed.Module = resource.Module
	}
	
	// Parse instances
	if len(resource.Instances) > 0 {
		// For simplicity, use the first instance
		instance := resource.Instances[0]
		managed.Attributes = instance.Attributes
		
		// Extract dependencies
		if instance.Dependencies != nil {
			managed.Dependencies = instance.Dependencies
		}
		
		// Extract cloud resource ID
		if id, ok := instance.Attributes["id"].(string); ok {
			managed.CloudID = id
		}
	}
	
	return managed
}

// buildResourceGraph builds a dependency graph from managed resources
func (sa *StateAnalyzer) buildResourceGraph(resources []ManagedResource) *ResourceGraph {
	graph := &ResourceGraph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}
	
	// Create nodes
	for _, resource := range resources {
		node := GraphNode{
			ID:       resource.ID,
			Label:    resource.Name,
			Type:     resource.Type,
			Provider: resource.Provider,
			Module:   resource.Module,
			Status:   resource.Status,
			Metadata: map[string]interface{}{
				"cloud_id": resource.CloudID,
			},
		}
		graph.Nodes = append(graph.Nodes, node)
	}
	
	// Create edges based on dependencies
	for _, resource := range resources {
		for _, dep := range resource.Dependencies {
			edge := GraphEdge{
				From:  resource.ID,
				To:    dep,
				Type:  "depends_on",
				Label: "depends on",
			}
			graph.Edges = append(graph.Edges, edge)
		}
		
		// Analyze attributes for implicit dependencies
		implicitDeps := sa.findImplicitDependencies(resource)
		for _, dep := range implicitDeps {
			edge := GraphEdge{
				From:  resource.ID,
				To:    dep,
				Type:  "references",
				Label: "references",
			}
			graph.Edges = append(graph.Edges, edge)
		}
	}
	
	return graph
}

// findImplicitDependencies finds implicit dependencies from resource attributes
func (sa *StateAnalyzer) findImplicitDependencies(resource ManagedResource) []string {
	deps := []string{}
	
	// Look for references in attributes
	for key, value := range resource.Attributes {
		if valueStr, ok := value.(string); ok {
			// Check for resource references (e.g., "${aws_vpc.main.id}")
			if strings.Contains(valueStr, "${") && strings.Contains(valueStr, ".") {
				// Extract resource reference
				parts := strings.Split(valueStr, ".")
				if len(parts) >= 2 {
					resourceRef := fmt.Sprintf("%s.%s", parts[0], parts[1])
					resourceRef = strings.TrimPrefix(resourceRef, "${")
					if resourceRef != resource.ID {
						deps = append(deps, resourceRef)
					}
				}
			}
			
			// Check for direct resource ID references
			if strings.Contains(key, "_id") || strings.Contains(key, "_arn") {
				// This might reference another resource
				// Would need more sophisticated parsing in production
			}
		}
	}
	
	return deps
}

// extractDependencies extracts all dependencies from resources
func (sa *StateAnalyzer) extractDependencies(resources []ManagedResource) map[string][]string {
	deps := make(map[string][]string)
	
	for _, resource := range resources {
		if len(resource.Dependencies) > 0 {
			deps[resource.ID] = resource.Dependencies
		}
	}
	
	return deps
}

// identifyOutOfBandResources identifies resources not managed by the state file
func (sa *StateAnalyzer) identifyOutOfBandResources(managedMap map[string]*ManagedResource, cloudResources []interface{}) []OutOfBandResource {
	outOfBand := []OutOfBandResource{}
	
	// Convert cloud resources to a map for easier lookup
	cloudMap := make(map[string]interface{})
	for _, resource := range cloudResources {
		if resMap, ok := resource.(map[string]interface{}); ok {
			if id, ok := resMap["id"].(string); ok {
				cloudMap[id] = resMap
			}
		}
	}
	
	// Check each cloud resource against managed resources
	for cloudID, cloudResource := range cloudMap {
		isManaged := false
		
		// Check if this cloud resource is managed
		for _, managed := range managedMap {
			if managed.CloudID == cloudID {
				isManaged = true
				break
			}
		}
		
		if !isManaged {
			// This is an out-of-band resource
			if resMap, ok := cloudResource.(map[string]interface{}); ok {
				oob := OutOfBandResource{
					ID:              cloudID,
					Type:            getStringValue(resMap, "type"),
					Name:            getStringValue(resMap, "name"),
					Provider:        getStringValue(resMap, "provider"),
					Region:          getStringValue(resMap, "region"),
					DiscoverySource: "cloud_api",
					Attributes:      resMap,
					Reason:          "Not found in state file",
				}
				
				// Determine adoption priority
				oob.AdoptionPriority = sa.determineAdoptionPriority(oob)
				
				// Generate import suggestion
				oob.SuggestedImport = sa.generateImportCommand(oob)
				
				outOfBand = append(outOfBand, oob)
			}
		}
	}
	
	return outOfBand
}

// detectConflicts detects conflicts between state and cloud resources
func (sa *StateAnalyzer) detectConflicts(managedMap map[string]*ManagedResource, cloudResources []interface{}) []ResourceConflict {
	conflicts := []ResourceConflict{}
	
	// Check for various types of conflicts
	for _, managed := range managedMap {
		// Find corresponding cloud resource
		var cloudResource map[string]interface{}
		for _, res := range cloudResources {
			if resMap, ok := res.(map[string]interface{}); ok {
				if cloudID, ok := resMap["id"].(string); ok && cloudID == managed.CloudID {
					cloudResource = resMap
					break
				}
			}
		}
		
		if cloudResource == nil {
			// Resource exists in state but not in cloud
			conflict := ResourceConflict{
				ResourceID:   managed.ID,
				ResourceType: managed.Type,
				ConflictType: "missing",
				StateValue:   "exists",
				ActualValue:  "missing",
				Severity:     "high",
				Resolution:   fmt.Sprintf("Resource %s exists in state but not in cloud. Consider removing from state or recreating.", managed.ID),
			}
			conflicts = append(conflicts, conflict)
		} else {
			// Check for configuration drift
			drifts := sa.compareConfigurations(managed.Attributes, cloudResource)
			for _, drift := range drifts {
				conflicts = append(conflicts, drift)
			}
		}
	}
	
	return conflicts
}

// compareConfigurations compares state and cloud configurations
func (sa *StateAnalyzer) compareConfigurations(stateAttrs, cloudAttrs map[string]interface{}) []ResourceConflict {
	conflicts := []ResourceConflict{}
	
	// Compare key attributes
	for key, stateValue := range stateAttrs {
		if cloudValue, exists := cloudAttrs[key]; exists {
			if !sa.valuesEqual(stateValue, cloudValue) {
				conflict := ResourceConflict{
					ConflictType: "configuration",
					StateValue:   stateValue,
					ActualValue:  cloudValue,
					Severity:     sa.determineSeverity(key),
					Details: map[string]interface{}{
						"attribute": key,
					},
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}
	
	return conflicts
}

// valuesEqual compares two values for equality
func (sa *StateAnalyzer) valuesEqual(v1, v2 interface{}) bool {
	// Simple comparison - would need more sophisticated logic in production
	return fmt.Sprintf("%v", v1) == fmt.Sprintf("%v", v2)
}

// determineSeverity determines the severity of a configuration drift
func (sa *StateAnalyzer) determineSeverity(attribute string) string {
	// Security-related attributes are critical
	securityAttrs := []string{"security_group", "encryption", "public", "password", "key", "secret", "token"}
	for _, secAttr := range securityAttrs {
		if strings.Contains(strings.ToLower(attribute), secAttr) {
			return "critical"
		}
	}
	
	// Network-related attributes are high
	networkAttrs := []string{"subnet", "vpc", "cidr", "ip", "port", "protocol"}
	for _, netAttr := range networkAttrs {
		if strings.Contains(strings.ToLower(attribute), netAttr) {
			return "high"
		}
	}
	
	// Size/capacity attributes are medium
	sizeAttrs := []string{"size", "count", "capacity", "instance_type", "memory", "cpu"}
	for _, sizeAttr := range sizeAttrs {
		if strings.Contains(strings.ToLower(attribute), sizeAttr) {
			return "medium"
		}
	}
	
	// Everything else is low
	return "low"
}

// determineAdoptionPriority determines the priority for adopting an out-of-band resource
func (sa *StateAnalyzer) determineAdoptionPriority(resource OutOfBandResource) string {
	// Critical infrastructure should be high priority
	criticalTypes := []string{"vpc", "subnet", "security_group", "iam", "database", "load_balancer"}
	for _, critical := range criticalTypes {
		if strings.Contains(strings.ToLower(resource.Type), critical) {
			return "high"
		}
	}
	
	// Storage and compute are medium priority
	mediumTypes := []string{"instance", "volume", "bucket", "container", "function"}
	for _, medium := range mediumTypes {
		if strings.Contains(strings.ToLower(resource.Type), medium) {
			return "medium"
		}
	}
	
	// Everything else is low priority
	return "low"
}

// generateImportCommand generates a Terraform import command for a resource
func (sa *StateAnalyzer) generateImportCommand(resource OutOfBandResource) string {
	// Generate the Terraform resource address
	resourceAddress := fmt.Sprintf("%s.%s", resource.Type, sanitizeResourceName(resource.Name))
	
	// Generate the import command
	return fmt.Sprintf("terraform import %s %s", resourceAddress, resource.ID)
}

// calculateStatistics calculates statistics for the perspective
func (sa *StateAnalyzer) calculateStatistics(perspective *StatePerspective) PerspectiveStatistics {
	stats := PerspectiveStatistics{
		TotalManaged:         len(perspective.ManagedResources),
		TotalOutOfBand:       len(perspective.OutOfBand),
		TotalDataSources:     len(perspective.DataSources),
		ResourcesByProvider:  make(map[string]int),
		ResourcesByType:      make(map[string]int),
		ResourcesByModule:    make(map[string]int),
	}
	
	// Count resources by provider and type
	for _, resource := range perspective.ManagedResources {
		stats.ResourcesByProvider[resource.Provider]++
		stats.ResourcesByType[resource.Type]++
		if resource.Module != "" {
			stats.ResourcesByModule[resource.Module]++
		}
	}
	
	// Calculate coverage percentage
	totalResources := stats.TotalManaged + stats.TotalOutOfBand
	if totalResources > 0 {
		stats.CoveragePercentage = float64(stats.TotalManaged) / float64(totalResources) * 100
	}
	
	// Calculate drift percentage
	driftedCount := 0
	for _, resource := range perspective.ManagedResources {
		if resource.Status == "drifted" {
			driftedCount++
		}
	}
	if stats.TotalManaged > 0 {
		stats.DriftPercentage = float64(driftedCount) / float64(stats.TotalManaged) * 100
	}
	
	// Count adoption opportunities (high priority out-of-band resources)
	for _, oob := range perspective.OutOfBand {
		if oob.AdoptionPriority == "high" {
			stats.AdoptionOpportunities++
		}
	}
	
	return stats
}

// GetPerspective returns a cached perspective if available
func (sa *StateAnalyzer) GetPerspective(stateFileID string) (*StatePerspective, bool) {
	perspective, exists := sa.perspectiveCache[stateFileID]
	return perspective, exists
}

// ComparePerspectives compares two perspectives to find differences
func (sa *StateAnalyzer) ComparePerspectives(p1, p2 *StatePerspective) *PerspectiveComparison {
	comparison := &PerspectiveComparison{
		Perspective1: p1.StateFileID,
		Perspective2: p2.StateFileID,
		Timestamp:    time.Now(),
	}
	
	// Compare managed resources
	p1Resources := make(map[string]*ManagedResource)
	for i := range p1.ManagedResources {
		r := &p1.ManagedResources[i]
		p1Resources[r.ID] = r
	}
	
	p2Resources := make(map[string]*ManagedResource)
	for i := range p2.ManagedResources {
		r := &p2.ManagedResources[i]
		p2Resources[r.ID] = r
	}
	
	// Find resources only in p1
	for id := range p1Resources {
		if _, exists := p2Resources[id]; !exists {
			comparison.OnlyInFirst = append(comparison.OnlyInFirst, id)
		}
	}
	
	// Find resources only in p2
	for id := range p2Resources {
		if _, exists := p1Resources[id]; !exists {
			comparison.OnlyInSecond = append(comparison.OnlyInSecond, id)
		}
	}
	
	// Find shared resources
	for id := range p1Resources {
		if _, exists := p2Resources[id]; exists {
			comparison.Shared = append(comparison.Shared, id)
		}
	}
	
	// Find conflicts
	comparison.Conflicts = sa.findPerspectiveConflicts(p1Resources, p2Resources)
	
	return comparison
}

// PerspectiveComparison represents the comparison between two perspectives
type PerspectiveComparison struct {
	Perspective1  string              `json:"perspective_1"`
	Perspective2  string              `json:"perspective_2"`
	Timestamp     time.Time           `json:"timestamp"`
	OnlyInFirst   []string            `json:"only_in_first"`
	OnlyInSecond  []string            `json:"only_in_second"`
	Shared        []string            `json:"shared"`
	Conflicts     []PerspectiveConflict `json:"conflicts"`
}

// PerspectiveConflict represents a conflict between two perspectives
type PerspectiveConflict struct {
	ResourceID    string      `json:"resource_id"`
	ConflictType  string      `json:"conflict_type"`
	Perspective1Value interface{} `json:"perspective_1_value"`
	Perspective2Value interface{} `json:"perspective_2_value"`
}

// findPerspectiveConflicts finds conflicts between two sets of resources
func (sa *StateAnalyzer) findPerspectiveConflicts(p1Resources, p2Resources map[string]*ManagedResource) []PerspectiveConflict {
	conflicts := []PerspectiveConflict{}
	
	for id, r1 := range p1Resources {
		if r2, exists := p2Resources[id]; exists {
			// Compare cloud IDs
			if r1.CloudID != r2.CloudID {
				conflict := PerspectiveConflict{
					ResourceID:        id,
					ConflictType:      "cloud_id_mismatch",
					Perspective1Value: r1.CloudID,
					Perspective2Value: r2.CloudID,
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}
	
	return conflicts
}

// Helper functions

func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func sanitizeResourceName(name string) string {
	// Replace invalid characters with underscores
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

// Analyzer provides state analysis capabilities (alias for StateAnalyzer)
type Analyzer struct {
	*StateAnalyzer
}

// NewAnalyzer creates a new state analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		StateAnalyzer: NewStateAnalyzer(),
	}
}

// AnalysisOptions contains options for state analysis
type AnalysisOptions struct {
	CheckDrift    bool
	CheckCoverage bool
	CheckOrphans  bool
	Provider      string
	Region        string
}

// AnalysisResult contains the results of state analysis
type AnalysisResult struct {
	DriftSummary      *DriftSummary      `json:"drift_summary,omitempty"`
	CoverageAnalysis  *CoverageAnalysis  `json:"coverage_analysis,omitempty"`
	OrphanedResources []*models.Resource `json:"orphaned_resources,omitempty"`
	Issues            []Issue            `json:"issues,omitempty"`
}

// DriftSummary contains drift analysis results
type DriftSummary struct {
	TotalResources   int     `json:"total_resources"`
	DriftedResources int     `json:"drifted_resources"`
	MissingResources int     `json:"missing_resources"`
	ExtraResources   int     `json:"extra_resources"`
	DriftPercentage  float64 `json:"drift_percentage"`
}

// CoverageAnalysis contains coverage analysis results
type CoverageAnalysis struct {
	ManagedResources   int      `json:"managed_resources"`
	UnmanagedResources int      `json:"unmanaged_resources"`
	CoveragePercentage float64  `json:"coverage_percentage"`
	UncoveredTypes     []string `json:"uncovered_types,omitempty"`
}

// Issue represents an issue found during analysis
type Issue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Resource string `json:"resource,omitempty"`
}

// AnalyzeState analyzes a Terraform state with given options
func (a *Analyzer) AnalyzeState(ctx context.Context, stateData *State, options AnalysisOptions) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Issues: []Issue{},
	}

	// Convert State to StateFile for compatibility
	stateFile := &StateFile{
		Version:   stateData.Version,
		Resources: stateData.Resources,
	}

	// Perform drift analysis
	if options.CheckDrift {
		result.DriftSummary = a.analyzeDrift(ctx, stateFile, options)
	}

	// Perform coverage analysis
	if options.CheckCoverage {
		result.CoverageAnalysis = a.analyzeCoverage(ctx, stateFile, options)
	}

	// Find orphaned resources
	if options.CheckOrphans {
		result.OrphanedResources = a.findOrphanedResources(ctx, stateFile, options)
	}

	// Check for common issues
	a.checkForIssues(stateFile, result)

	return result, nil
}

// analyzeDrift analyzes drift in the state
func (a *Analyzer) analyzeDrift(ctx context.Context, stateFile *StateFile, options AnalysisOptions) *DriftSummary {
	summary := &DriftSummary{
		TotalResources: len(stateFile.Resources),
	}

	// Analyze each resource for drift
	for _, resource := range stateFile.Resources {
		// Apply filters
		if options.Provider != "" && !strings.Contains(resource.Provider, options.Provider) {
			continue
		}

		// Check resource status
		if len(resource.Instances) == 0 {
			summary.MissingResources++
		} else {
			// Check for drift in attributes
			for _, instance := range resource.Instances {
				// Convert Instance to ResourceInstance for compatibility
				resInstance := ResourceInstance{
					SchemaVersion: instance.SchemaVersion,
					Attributes:    instance.Attributes,
					Private:       instance.Private,
					Dependencies:  instance.Dependencies,
				}
				if a.hasAttributeDrift(resInstance) {
					summary.DriftedResources++
					break
				}
			}
		}
	}

	// Calculate drift percentage
	if summary.TotalResources > 0 {
		summary.DriftPercentage = float64(summary.DriftedResources+summary.MissingResources) / float64(summary.TotalResources) * 100
	}

	return summary
}

// analyzeCoverage analyzes resource coverage
func (a *Analyzer) analyzeCoverage(ctx context.Context, stateFile *StateFile, options AnalysisOptions) *CoverageAnalysis {
	coverage := &CoverageAnalysis{
		ManagedResources: len(stateFile.Resources),
		UncoveredTypes:   []string{},
	}

	// Track resource types
	coveredTypes := make(map[string]bool)
	for _, resource := range stateFile.Resources {
		coveredTypes[resource.Type] = true
	}

	// Common resource types that should be managed
	commonTypes := []string{
		"aws_instance", "aws_security_group", "aws_vpc", "aws_subnet",
		"azurerm_virtual_machine", "azurerm_network_security_group",
		"google_compute_instance", "google_compute_network",
	}

	// Find uncovered types
	for _, resType := range commonTypes {
		if !coveredTypes[resType] {
			// Check if this provider is relevant
			if options.Provider == "" || strings.Contains(resType, options.Provider) {
				coverage.UncoveredTypes = append(coverage.UncoveredTypes, resType)
			}
		}
	}

	// Estimate unmanaged resources (simplified)
	coverage.UnmanagedResources = len(coverage.UncoveredTypes) * 2 // Rough estimate

	// Calculate coverage percentage
	totalEstimated := coverage.ManagedResources + coverage.UnmanagedResources
	if totalEstimated > 0 {
		coverage.CoveragePercentage = float64(coverage.ManagedResources) / float64(totalEstimated) * 100
	}

	return coverage
}

// findOrphanedResources finds orphaned resources in the state
func (a *Analyzer) findOrphanedResources(ctx context.Context, stateFile *StateFile, options AnalysisOptions) []*models.Resource {
	orphaned := []*models.Resource{}

	for _, resource := range stateFile.Resources {
		// Apply filters
		if options.Provider != "" && !strings.Contains(resource.Provider, options.Provider) {
			continue
		}

		// Check if resource has no instances or is marked for deletion
		if len(resource.Instances) == 0 {
			orphanedResource := &models.Resource{
				ID:       fmt.Sprintf("%s.%s", resource.Type, resource.Name),
				Name:     resource.Name,
				Type:     resource.Type,
				Provider: resource.Provider,
				Status:   "orphaned",
			}
			orphaned = append(orphaned, orphanedResource)
		}
	}

	return orphaned
}

// hasAttributeDrift checks if an instance has attribute drift
func (a *Analyzer) hasAttributeDrift(instance ResourceInstance) bool {
	// Simplified drift detection - check for common drift indicators
	if instance.Attributes == nil {
		return true
	}

	// Check for lifecycle changes
	if lifecycle, ok := instance.Attributes["lifecycle"].(map[string]interface{}); ok {
		if _, hasChanges := lifecycle["ignore_changes"]; hasChanges {
			return true
		}
	}

	return false
}

// checkForIssues checks for common issues in the state
func (a *Analyzer) checkForIssues(stateFile *StateFile, result *AnalysisResult) {
	// Check for old state version
	if stateFile.Version < 4 {
		result.Issues = append(result.Issues, Issue{
			Severity: "warning",
			Message:  fmt.Sprintf("State file uses old version %d. Consider upgrading to version 4.", stateFile.Version),
		})
	}

	// Check for resources without providers
	for _, resource := range stateFile.Resources {
		if resource.Provider == "" {
			result.Issues = append(result.Issues, Issue{
				Severity: "warning",
				Message:  fmt.Sprintf("Resource %s.%s has no provider specified", resource.Type, resource.Name),
				Resource: fmt.Sprintf("%s.%s", resource.Type, resource.Name),
			})
		}
	}

	// Check for duplicate resources
	seen := make(map[string]bool)
	for _, resource := range stateFile.Resources {
		key := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		if seen[key] {
			result.Issues = append(result.Issues, Issue{
				Severity: "error",
				Message:  fmt.Sprintf("Duplicate resource found: %s", key),
				Resource: key,
			})
		}
		seen[key] = true
	}
}

