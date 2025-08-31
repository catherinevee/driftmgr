package aggregator

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/state/discovery"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// StateAggregator aggregates multiple Terraform state files
type StateAggregator struct {
	discoveryService    *discovery.StateDiscoveryService
	states              map[string]*state.State
	resourceIndex       map[string]*ResourceLocation
	duplicateResources  map[string][]*ResourceLocation
	orphanedResources   map[string]*OrphanedResource
	crossStateRefs      map[string][]*CrossStateReference
	stateRelationships  map[string][]string
	environmentStates   map[string][]string
	workspaceStates     map[string][]string
	moduleRegistry      map[string]*ModuleInfo
	aggregationMetrics  *AggregationMetrics
	mu                  sync.RWMutex
}

// ResourceLocation tracks where a resource is managed
type ResourceLocation struct {
	StateFile      string
	ResourceID     string
	ResourceType   string
	ResourceName   string
	Workspace      string
	Environment    string
	Module         string
	Provider       string
	LastModified   time.Time
	IsManaged      bool
	IsTainted      bool
	HasDrift       bool
}

// OrphanedResource represents a resource no longer in any state
type OrphanedResource struct {
	ResourceID      string
	ResourceType    string
	LastSeenState   string
	LastSeenTime    time.Time
	RemovalDetected time.Time
	LikelyReason    string
	RelatedResources []string
}

// CrossStateReference represents dependencies between states
type CrossStateReference struct {
	SourceState     string
	TargetState     string
	ReferenceType   ReferenceType
	ResourceID      string
	DataSource      string
	OutputVariable  string
	RemoteState     string
}

// ReferenceType defines the type of cross-state reference
type ReferenceType string

const (
	ReferenceTypeDataSource   ReferenceType = "data_source"
	ReferenceTypeRemoteState  ReferenceType = "remote_state"
	ReferenceTypeOutput       ReferenceType = "output"
	ReferenceTypeModule       ReferenceType = "module"
	ReferenceTypeDependency   ReferenceType = "dependency"
)

// ModuleInfo tracks Terraform module usage
type ModuleInfo struct {
	Name           string
	Source         string
	Version        string
	UsedInStates   []string
	ResourceCount  int
	LastUpdated    time.Time
	Dependencies   []string
}

// AggregationMetrics tracks aggregation statistics
type AggregationMetrics struct {
	TotalStates           int
	TotalResources        int
	UniqueResources       int
	DuplicateResources    int
	OrphanedResources     int
	CrossStateReferences  int
	PartialStates         int
	ConflictingResources  int
	StatesByEnvironment   map[string]int
	StatesByWorkspace     map[string]int
	ResourcesByProvider   map[string]int
	ProcessingTime        time.Duration
}

// AggregationResult contains the complete aggregation analysis
type AggregationResult struct {
	AllResources         map[string]*ResourceLocation
	DuplicateResources   map[string][]*ResourceLocation
	OrphanedResources    map[string]*OrphanedResource
	CrossStateReferences map[string][]*CrossStateReference
	StateRelationships   map[string][]string
	Metrics              *AggregationMetrics
	Issues               []AggregationIssue
	Recommendations      []string
}

// AggregationIssue represents an issue found during aggregation
type AggregationIssue struct {
	Type        IssueType
	Severity    IssueSeverity
	StateFile   string
	ResourceID  string
	Description string
	Resolution  string
}

// IssueType defines the type of aggregation issue
type IssueType string

const (
	IssueTypeDuplicate      IssueType = "duplicate_resource"
	IssueTypeOrphaned       IssueType = "orphaned_resource"
	IssueTypeConflict       IssueType = "resource_conflict"
	IssueTypePartialState   IssueType = "partial_state"
	IssueTypeMissingDepends IssueType = "missing_dependency"
	IssueTypeVersionMismatch IssueType = "version_mismatch"
)

// IssueSeverity defines the severity of an issue
type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical"
	SeverityHigh     IssueSeverity = "high"
	SeverityMedium   IssueSeverity = "medium"
	SeverityLow      IssueSeverity = "low"
	SeverityInfo     IssueSeverity = "info"
)

// NewStateAggregator creates a new state aggregator
func NewStateAggregator(discoveryService *discovery.StateDiscoveryService) *StateAggregator {
	return &StateAggregator{
		discoveryService:   discoveryService,
		states:             make(map[string]*state.State),
		resourceIndex:      make(map[string]*ResourceLocation),
		duplicateResources: make(map[string][]*ResourceLocation),
		orphanedResources:  make(map[string]*OrphanedResource),
		crossStateRefs:     make(map[string][]*CrossStateReference),
		stateRelationships: make(map[string][]string),
		environmentStates:  make(map[string][]string),
		workspaceStates:    make(map[string][]string),
		moduleRegistry:     make(map[string]*ModuleInfo),
		aggregationMetrics: &AggregationMetrics{
			StatesByEnvironment:  make(map[string]int),
			StatesByWorkspace:    make(map[string]int),
			ResourcesByProvider:  make(map[string]int),
		},
	}
}

// AggregateStates aggregates all discovered state files
func (sa *StateAggregator) AggregateStates(ctx context.Context) (*AggregationResult, error) {
	startTime := time.Now()
	
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	// Get all discovered states
	discoveredStates := sa.discoveryService.GetAllDiscoveredStates()
	
	// Load and parse all state files
	if err := sa.loadAllStates(ctx, discoveredStates); err != nil {
		return nil, fmt.Errorf("failed to load states: %w", err)
	}
	
	// Build resource index
	sa.buildResourceIndex()
	
	// Detect duplicate resources
	sa.detectDuplicateResources()
	
	// Find orphaned resources
	sa.findOrphanedResources()
	
	// Analyze cross-state references
	sa.analyzeCrossStateReferences()
	
	// Build state relationships
	sa.buildStateRelationships()
	
	// Analyze modules
	sa.analyzeModules()
	
	// Calculate metrics
	sa.calculateMetrics()
	sa.aggregationMetrics.ProcessingTime = time.Since(startTime)
	
	// Generate issues and recommendations
	issues := sa.generateIssues()
	recommendations := sa.generateRecommendations()
	
	return &AggregationResult{
		AllResources:         sa.resourceIndex,
		DuplicateResources:   sa.duplicateResources,
		OrphanedResources:    sa.orphanedResources,
		CrossStateReferences: sa.crossStateRefs,
		StateRelationships:   sa.stateRelationships,
		Metrics:              sa.aggregationMetrics,
		Issues:               issues,
		Recommendations:      recommendations,
	}, nil
}

// loadAllStates loads all state files
func (sa *StateAggregator) loadAllStates(ctx context.Context, discoveredStates map[string]*discovery.DiscoveredState) error {
	for path, discoveredState := range discoveredStates {
		// Skip backups unless they're the only version
		if discoveredState.Type == discovery.StateTypeBackup {
			mainStatePath := strings.TrimSuffix(path, ".backup")
			if _, exists := discoveredStates[mainStatePath]; exists {
				continue // Skip backup if main state exists
			}
		}
		
		// Load state file
		loader := state.NewStateLoader(path)
		stateFile, err := loader.LoadStateFile(ctx, path, nil)
		if err != nil {
			// Don't fail completely, just skip this state
			fmt.Printf("Warning: Failed to load state %s: %v\n", path, err)
			continue
		}
		
		sa.states[path] = stateFile
		
		// Track by environment and workspace
		if discoveredState.Environment != "" {
			sa.environmentStates[discoveredState.Environment] = append(
				sa.environmentStates[discoveredState.Environment], path)
		}
		if discoveredState.Workspace != "" {
			sa.workspaceStates[discoveredState.Workspace] = append(
				sa.workspaceStates[discoveredState.Workspace], path)
		}
	}
	
	sa.aggregationMetrics.TotalStates = len(sa.states)
	return nil
}

// buildResourceIndex builds an index of all resources across states
func (sa *StateAggregator) buildResourceIndex() {
	for statePath, stateFile := range sa.states {
		discoveredState := sa.discoveryService.GetAllDiscoveredStates()[statePath]
		
		for _, resource := range stateFile.Resources {
			resourceKey := sa.generateResourceKey(resource)
			
			location := &ResourceLocation{
				StateFile:    statePath,
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				ResourceName: resource.Name,
				Module:       resource.Module,
				Provider:     resource.Provider,
				LastModified: discoveredState.LastModified,
				IsManaged:    true,
				Workspace:    discoveredState.Workspace,
				Environment:  discoveredState.Environment,
			}
			
			// Check if resource already indexed (duplicate)
			if existing, exists := sa.resourceIndex[resourceKey]; exists {
				// Add to duplicates
				if sa.duplicateResources[resourceKey] == nil {
					sa.duplicateResources[resourceKey] = []*ResourceLocation{existing}
				}
				sa.duplicateResources[resourceKey] = append(
					sa.duplicateResources[resourceKey], location)
			} else {
				sa.resourceIndex[resourceKey] = location
			}
			
			// Track provider
			provider := sa.extractProvider(resource.Type)
			sa.aggregationMetrics.ResourcesByProvider[provider]++
		}
	}
	
	sa.aggregationMetrics.TotalResources = len(sa.resourceIndex) + len(sa.duplicateResources)
	sa.aggregationMetrics.UniqueResources = len(sa.resourceIndex)
}

// generateResourceKey generates a unique key for a resource
func (sa *StateAggregator) generateResourceKey(resource state.Resource) string {
	// Use type and name as primary key
	key := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
	
	// Add module if present
	if resource.Module != "" {
		key = fmt.Sprintf("%s.%s", resource.Module, key)
	}
	
	return key
}

// extractProvider extracts provider from resource type
func (sa *StateAggregator) extractProvider(resourceType string) string {
	if strings.HasPrefix(resourceType, "aws_") {
		return "aws"
	}
	if strings.HasPrefix(resourceType, "azurerm_") {
		return "azure"
	}
	if strings.HasPrefix(resourceType, "google_") {
		return "gcp"
	}
	if strings.HasPrefix(resourceType, "digitalocean_") {
		return "digitalocean"
	}
	return "unknown"
}

// detectDuplicateResources detects resources managed in multiple states
func (sa *StateAggregator) detectDuplicateResources() {
	sa.aggregationMetrics.DuplicateResources = len(sa.duplicateResources)
	
	// Analyze duplicate patterns
	for resourceKey, locations := range sa.duplicateResources {
		// Check if duplicates are in different environments
		envs := make(map[string]bool)
		for _, loc := range locations {
			if loc.Environment != "" {
				envs[loc.Environment] = true
			}
		}
		
		// If same resource in different environments, might be intentional
		if len(envs) > 1 {
			// Mark as potentially intentional
			for _, loc := range locations {
				loc.HasDrift = false
			}
		} else {
			// Same environment duplicates are problematic
			sa.aggregationMetrics.ConflictingResources++
		}
	}
}

// findOrphanedResources finds resources that were previously managed
func (sa *StateAggregator) findOrphanedResources() {
	// Look for backup states with resources not in current states
	for statePath, stateFile := range sa.states {
		if strings.Contains(statePath, ".backup") {
			mainStatePath := strings.TrimSuffix(statePath, ".backup")
			if mainState, exists := sa.states[mainStatePath]; exists {
				// Compare backup with current
				sa.compareStatesForOrphans(statePath, stateFile, mainStatePath, mainState)
			}
		}
	}
	
	sa.aggregationMetrics.OrphanedResources = len(sa.orphanedResources)
}

// compareStatesForOrphans compares states to find orphaned resources
func (sa *StateAggregator) compareStatesForOrphans(backupPath string, backupState *state.State, 
	currentPath string, currentState *state.State) {
	
	// Build current resource map
	currentResources := make(map[string]bool)
	for _, resource := range currentState.Resources {
		key := sa.generateResourceKey(resource)
		currentResources[key] = true
	}
	
	// Find resources in backup but not in current
	for _, resource := range backupState.Resources {
		key := sa.generateResourceKey(resource)
		if !currentResources[key] {
			// Resource was removed
			orphaned := &OrphanedResource{
				ResourceID:      resource.ID,
				ResourceType:    resource.Type,
				LastSeenState:   backupPath,
				LastSeenTime:    sa.getStateModTime(backupPath),
				RemovalDetected: time.Now(),
				LikelyReason:    sa.detectRemovalReason(resource),
			}
			
			sa.orphanedResources[key] = orphaned
		}
	}
}

// detectRemovalReason attempts to determine why a resource was removed
func (sa *StateAggregator) detectRemovalReason(resource state.Resource) string {
	// Check for common patterns
	if strings.Contains(resource.Name, "temp") || strings.Contains(resource.Name, "test") {
		return "Temporary resource removed"
	}
	
	if resource.Module != "" {
		return "Module refactoring"
	}
	
	// Check if tainted
	if resource.Mode == "tainted" {
		return "Tainted resource replaced"
	}
	
	return "Manual removal or terraform destroy"
}

// analyzeCrossStateReferences analyzes references between states
func (sa *StateAggregator) analyzeCrossStateReferences() {
	for statePath1, state1 := range sa.states {
		for statePath2, state2 := range sa.states {
			if statePath1 == statePath2 {
				continue
			}
			
			// Check for references
			refs := sa.findReferences(state1, state2)
			if len(refs) > 0 {
				for _, ref := range refs {
					ref.SourceState = statePath1
					ref.TargetState = statePath2
				}
				sa.crossStateRefs[statePath1] = append(sa.crossStateRefs[statePath1], refs...)
			}
		}
	}
	
	// Count total references
	for _, refs := range sa.crossStateRefs {
		sa.aggregationMetrics.CrossStateReferences += len(refs)
	}
}

// findReferences finds references between two states
func (sa *StateAggregator) findReferences(state1, state2 *state.State) []*CrossStateReference {
	refs := []*CrossStateReference{}
	
	// Check for resources that might reference each other
	for _, res1 := range state1.Resources {
		for _, res2 := range state2.Resources {
			// Check if resource types indicate potential reference
			if sa.canReference(res1.Type, res2.Type) {
				ref := &CrossStateReference{
					ReferenceType: ReferenceTypeDependency,
					ResourceID:    res2.ID,
				}
				refs = append(refs, ref)
			}
		}
	}
	
	return refs
}

// canReference checks if one resource type can reference another
func (sa *StateAggregator) canReference(type1, type2 string) bool {
	// Define reference patterns
	references := map[string][]string{
		"aws_instance": {"aws_security_group", "aws_subnet", "aws_key_pair"},
		"aws_security_group_rule": {"aws_security_group"},
		"aws_route": {"aws_route_table", "aws_nat_gateway", "aws_internet_gateway"},
		"azurerm_virtual_machine": {"azurerm_network_interface", "azurerm_subnet"},
		"google_compute_instance": {"google_compute_network", "google_compute_subnetwork"},
	}
	
	if refs, ok := references[type1]; ok {
		for _, ref := range refs {
			if ref == type2 {
				return true
			}
		}
	}
	
	return false
}

// buildStateRelationships builds relationships between state files
func (sa *StateAggregator) buildStateRelationships() {
	discoveredStates := sa.discoveryService.GetAllDiscoveredStates()
	
	for path1, state1 := range discoveredStates {
		for path2, state2 := range discoveredStates {
			if path1 == path2 {
				continue
			}
			
			// Check various relationship criteria
			related := false
			
			// Same environment
			if state1.Environment == state2.Environment && state1.Environment != "unknown" {
				related = true
			}
			
			// Same workspace
			if state1.Workspace == state2.Workspace && state1.Workspace != "default" {
				related = true
			}
			
			// Parent-child directory relationship
			dir1 := filepath.Dir(path1)
			dir2 := filepath.Dir(path2)
			if strings.HasPrefix(dir1, dir2) || strings.HasPrefix(dir2, dir1) {
				related = true
			}
			
			// Share modules
			for _, mod1 := range state1.Modules {
				for _, mod2 := range state2.Modules {
					if mod1 == mod2 {
						related = true
						break
					}
				}
			}
			
			if related {
				sa.stateRelationships[path1] = append(sa.stateRelationships[path1], path2)
			}
		}
	}
}

// analyzeModules analyzes Terraform module usage across states
func (sa *StateAggregator) analyzeModules() {
	for statePath, stateFile := range sa.states {
		modulesSeen := make(map[string]bool)
		
		for _, resource := range stateFile.Resources {
			if resource.Module != "" && !modulesSeen[resource.Module] {
				modulesSeen[resource.Module] = true
				
				if moduleInfo, exists := sa.moduleRegistry[resource.Module]; exists {
					moduleInfo.UsedInStates = append(moduleInfo.UsedInStates, statePath)
					moduleInfo.ResourceCount++
				} else {
					sa.moduleRegistry[resource.Module] = &ModuleInfo{
						Name:          resource.Module,
						UsedInStates:  []string{statePath},
						ResourceCount: 1,
						LastUpdated:   sa.getStateModTime(statePath),
					}
				}
			}
		}
	}
}

// calculateMetrics calculates aggregation metrics
func (sa *StateAggregator) calculateMetrics() {
	// Count states by environment
	for env, states := range sa.environmentStates {
		sa.aggregationMetrics.StatesByEnvironment[env] = len(states)
	}
	
	// Count states by workspace
	for workspace, states := range sa.workspaceStates {
		sa.aggregationMetrics.StatesByWorkspace[workspace] = len(states)
	}
	
	// Count partial states
	discoveredStates := sa.discoveryService.GetAllDiscoveredStates()
	for _, state := range discoveredStates {
		if state.IsPartial {
			sa.aggregationMetrics.PartialStates++
		}
	}
}

// generateIssues generates issues found during aggregation
func (sa *StateAggregator) generateIssues() []AggregationIssue {
	issues := []AggregationIssue{}
	
	// Duplicate resource issues
	for resourceKey, locations := range sa.duplicateResources {
		// Check if duplicates are in same environment
		envs := make(map[string]bool)
		for _, loc := range locations {
			if loc.Environment != "" {
				envs[loc.Environment] = true
			}
		}
		
		if len(envs) == 1 || len(envs) == 0 {
			// Same environment duplicates are critical
			issue := AggregationIssue{
				Type:        IssueTypeDuplicate,
				Severity:    SeverityCritical,
				ResourceID:  resourceKey,
				Description: fmt.Sprintf("Resource %s managed in %d state files", resourceKey, len(locations)),
				Resolution:  "Consolidate resource management to a single state file",
			}
			issues = append(issues, issue)
		}
	}
	
	// Orphaned resource issues
	for resourceKey, orphaned := range sa.orphanedResources {
		issue := AggregationIssue{
			Type:        IssueTypeOrphaned,
			Severity:    SeverityHigh,
			StateFile:   orphaned.LastSeenState,
			ResourceID:  resourceKey,
			Description: fmt.Sprintf("Resource %s no longer in state (last seen: %s)", 
				orphaned.ResourceType, orphaned.LastSeenTime.Format("2006-01-02")),
			Resolution:  "Verify if resource should be imported or if removal was intentional",
		}
		issues = append(issues, issue)
	}
	
	// Partial state issues
	discoveredStates := sa.discoveryService.GetAllDiscoveredStates()
	for path, state := range discoveredStates {
		if state.IsPartial {
			issue := AggregationIssue{
				Type:        IssueTypePartialState,
				Severity:    SeverityMedium,
				StateFile:   path,
				Description: "State file appears to be incomplete or partially imported",
				Resolution:  "Review and complete resource imports",
			}
			issues = append(issues, issue)
		}
	}
	
	return issues
}

// generateRecommendations generates recommendations based on analysis
func (sa *StateAggregator) generateRecommendations() []string {
	recommendations := []string{}
	
	// Check for too many state files
	if sa.aggregationMetrics.TotalStates > 10 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider consolidating %d state files using workspaces or modules",
				sa.aggregationMetrics.TotalStates))
	}
	
	// Check for duplicate resources
	if sa.aggregationMetrics.DuplicateResources > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Resolve %d duplicate resource definitions across state files",
				sa.aggregationMetrics.DuplicateResources))
	}
	
	// Check for orphaned resources
	if sa.aggregationMetrics.OrphanedResources > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Review %d orphaned resources for re-import or cleanup",
				sa.aggregationMetrics.OrphanedResources))
	}
	
	// Check for environment segregation
	if len(sa.aggregationMetrics.StatesByEnvironment) > 1 {
		mixed := false
		for _, states := range sa.environmentStates {
			if len(states) > 1 {
				// Multiple states for same environment
				for i, state1 := range states {
					for j, state2 := range states {
						if i != j {
							dir1 := filepath.Dir(state1)
							dir2 := filepath.Dir(state2)
							if dir1 == dir2 {
								mixed = true
								break
							}
						}
					}
				}
			}
		}
		
		if mixed {
			recommendations = append(recommendations,
				"Separate environment states into different directories")
		}
	}
	
	// Check for workspace usage
	if len(sa.aggregationMetrics.StatesByWorkspace) == 1 && 
	   sa.aggregationMetrics.StatesByWorkspace["default"] > 0 {
		recommendations = append(recommendations,
			"Consider using Terraform workspaces for environment separation")
	}
	
	// Check for module reuse
	moduleReuse := 0
	for _, moduleInfo := range sa.moduleRegistry {
		if len(moduleInfo.UsedInStates) > 1 {
			moduleReuse++
		}
	}
	
	if moduleReuse > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Good: %d modules are reused across states", moduleReuse))
	}
	
	return recommendations
}

// getStateModTime gets the modification time of a state file
func (sa *StateAggregator) getStateModTime(path string) time.Time {
	if state := sa.discoveryService.GetAllDiscoveredStates()[path]; state != nil {
		return state.LastModified
	}
	return time.Time{}
}

// GetResourceByKey returns a resource by its key
func (sa *StateAggregator) GetResourceByKey(key string) *ResourceLocation {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.resourceIndex[key]
}

// GetAllResources returns all indexed resources
func (sa *StateAggregator) GetAllResources() map[string]*ResourceLocation {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.resourceIndex
}

// GetDuplicateResources returns duplicate resources
func (sa *StateAggregator) GetDuplicateResources() map[string][]*ResourceLocation {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.duplicateResources
}

// GetOrphanedResources returns orphaned resources
func (sa *StateAggregator) GetOrphanedResources() map[string]*OrphanedResource {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.orphanedResources
}

// GetMetrics returns aggregation metrics
func (sa *StateAggregator) GetMetrics() *AggregationMetrics {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.aggregationMetrics
}

// CompareWithCloudResources compares aggregated state with actual cloud resources
func (sa *StateAggregator) CompareWithCloudResources(cloudResources []models.Resource) *ComparisonResult {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	
	result := &ComparisonResult{
		InStateOnly:     []string{},
		InCloudOnly:     []string{},
		InBoth:          []string{},
		StateResources:  len(sa.resourceIndex),
		CloudResources:  len(cloudResources),
		CoveragePercent: 0,
	}
	
	// Build cloud resource map
	cloudMap := make(map[string]bool)
	for _, resource := range cloudResources {
		key := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		cloudMap[key] = true
		
		if _, exists := sa.resourceIndex[key]; exists {
			result.InBoth = append(result.InBoth, key)
		} else {
			result.InCloudOnly = append(result.InCloudOnly, key)
		}
	}
	
	// Find resources only in state
	for key := range sa.resourceIndex {
		if !cloudMap[key] {
			result.InStateOnly = append(result.InStateOnly, key)
		}
	}
	
	// Calculate coverage
	if len(cloudResources) > 0 {
		result.CoveragePercent = float64(len(result.InBoth)) / float64(len(cloudResources)) * 100
	}
	
	// Sort for consistent output
	sort.Strings(result.InStateOnly)
	sort.Strings(result.InCloudOnly)
	sort.Strings(result.InBoth)
	
	return result
}

// ComparisonResult contains the comparison between state and cloud
type ComparisonResult struct {
	InStateOnly     []string
	InCloudOnly     []string
	InBoth          []string
	StateResources  int
	CloudResources  int
	CoveragePercent float64
}