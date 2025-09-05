package resolver

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/catherinevee/driftmgr/internal/terragrunt/parser"
)

// DependencyGraph represents the dependency relationships between Terragrunt modules
type DependencyGraph struct {
	Modules      map[string]*Module
	Dependencies map[string][]string // module path -> list of dependencies
	Dependents   map[string][]string // module path -> list of dependents
}

// Module represents a Terragrunt module with its configuration and state
type Module struct {
	Path         string                   `json:"path"`
	Config       *parser.TerragruntConfig `json:"config"`
	Dependencies []string                 `json:"dependencies"`
	Dependents   []string                 `json:"dependents"`
	Outputs      map[string]interface{}   `json:"outputs,omitempty"`
	Status       ModuleStatus             `json:"status"`
	Error        error                    `json:"error,omitempty"`
}

// ModuleStatus represents the execution status of a module
type ModuleStatus string

const (
	ModuleStatusPending   ModuleStatus = "pending"
	ModuleStatusRunning   ModuleStatus = "running"
	ModuleStatusCompleted ModuleStatus = "completed"
	ModuleStatusFailed    ModuleStatus = "failed"
	ModuleStatusSkipped   ModuleStatus = "skipped"
)

// ExecutionOrder represents the order in which modules should be executed
type ExecutionOrder struct {
	Groups [][]string `json:"groups"` // Groups of modules that can be executed in parallel
	Total  int        `json:"total"`
}

// DependencyResolver resolves dependencies between Terragrunt modules
type DependencyResolver struct {
	parser *parser.Parser
	graph  *DependencyGraph
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver() *DependencyResolver {
	return &DependencyResolver{
		parser: parser.NewParser(),
		graph: &DependencyGraph{
			Modules:      make(map[string]*Module),
			Dependencies: make(map[string][]string),
			Dependents:   make(map[string][]string),
		},
	}
}

// ResolveDirectory resolves all dependencies in a directory tree
func (r *DependencyResolver) ResolveDirectory(rootDir string) (*DependencyGraph, error) {
	// Parse all Terragrunt configurations
	configs, err := r.parser.ParseDirectory(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %w", err)
	}

	// Build the dependency graph
	for _, config := range configs {
		if err := r.addModule(config); err != nil {
			return nil, fmt.Errorf("failed to add module %s: %w", config.FilePath, err)
		}
	}

	// Validate the graph (check for cycles)
	if err := r.validateGraph(); err != nil {
		return nil, err
	}

	return r.graph, nil
}

// addModule adds a module to the dependency graph
func (r *DependencyResolver) addModule(config *parser.TerragruntConfig) error {
	modulePath := config.WorkingDir

	// Skip if already processed
	if _, exists := r.graph.Modules[modulePath]; exists {
		return nil
	}

	module := &Module{
		Path:         modulePath,
		Config:       config,
		Dependencies: []string{},
		Dependents:   []string{},
		Outputs:      make(map[string]interface{}),
		Status:       ModuleStatusPending,
	}

	// Process dependency blocks
	for _, dep := range config.DependencyBlocks {
		if dep.Skip {
			continue
		}

		depPath := r.resolveDependencyPath(modulePath, dep.ConfigPath)
		module.Dependencies = append(module.Dependencies, depPath)

		// Add to graph dependencies
		if !contains(r.graph.Dependencies[modulePath], depPath) {
			r.graph.Dependencies[modulePath] = append(r.graph.Dependencies[modulePath], depPath)
		}

		// Add to graph dependents (reverse mapping)
		if !contains(r.graph.Dependents[depPath], modulePath) {
			r.graph.Dependents[depPath] = append(r.graph.Dependents[depPath], modulePath)
		}
	}

	// Process dependencies list
	for _, dep := range config.Dependencies {
		if !dep.Enabled {
			continue
		}

		depPath := r.resolveDependencyPath(modulePath, dep.ConfigPath)
		module.Dependencies = append(module.Dependencies, depPath)

		// Add to graph dependencies
		if !contains(r.graph.Dependencies[modulePath], depPath) {
			r.graph.Dependencies[modulePath] = append(r.graph.Dependencies[modulePath], depPath)
		}

		// Add to graph dependents
		if !contains(r.graph.Dependents[depPath], modulePath) {
			r.graph.Dependents[depPath] = append(r.graph.Dependents[depPath], modulePath)
		}
	}

	r.graph.Modules[modulePath] = module
	return nil
}

// resolveDependencyPath resolves a relative dependency path to an absolute path
func (r *DependencyResolver) resolveDependencyPath(modulePath, depPath string) string {
	if filepath.IsAbs(depPath) {
		return filepath.Clean(depPath)
	}

	// Handle relative paths
	absPath := filepath.Join(modulePath, depPath)
	return filepath.Clean(absPath)
}

// validateGraph checks for cycles in the dependency graph
func (r *DependencyResolver) validateGraph() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for module := range r.graph.Modules {
		if !visited[module] {
			if r.hasCycle(module, visited, recStack) {
				return fmt.Errorf("circular dependency detected in module %s", module)
			}
		}
	}

	return nil
}

// hasCycle uses DFS to detect cycles in the graph
func (r *DependencyResolver) hasCycle(module string, visited, recStack map[string]bool) bool {
	visited[module] = true
	recStack[module] = true

	// Check all dependencies of the current module
	for _, dep := range r.graph.Dependencies[module] {
		if !visited[dep] {
			if r.hasCycle(dep, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			// Found a back edge (cycle)
			return true
		}
	}

	recStack[module] = false
	return false
}

// GetExecutionOrder determines the order in which modules should be executed
func (r *DependencyResolver) GetExecutionOrder(includeSkipped bool) (*ExecutionOrder, error) {
	// Perform topological sort
	indegree := make(map[string]int)

	// Calculate indegree for each module
	for module := range r.graph.Modules {
		if !includeSkipped && r.graph.Modules[module].Config.Skip {
			continue
		}
		indegree[module] = len(r.graph.Dependencies[module])
	}

	var groups [][]string
	processed := make(map[string]bool)

	for len(processed) < len(indegree) {
		var currentGroup []string

		// Find all modules with indegree 0 (no unprocessed dependencies)
		for module, degree := range indegree {
			if degree == 0 && !processed[module] {
				currentGroup = append(currentGroup, module)
			}
		}

		if len(currentGroup) == 0 {
			// No modules can be processed - might be a cycle we didn't detect
			return nil, fmt.Errorf("unable to determine execution order - possible circular dependency")
		}

		// Sort modules within group for consistent ordering
		sort.Strings(currentGroup)
		groups = append(groups, currentGroup)

		// Mark as processed and update indegrees
		for _, module := range currentGroup {
			processed[module] = true

			// Reduce indegree for all dependents
			for _, dependent := range r.graph.Dependents[module] {
				if _, exists := indegree[dependent]; exists {
					indegree[dependent]--
				}
			}
		}
	}

	return &ExecutionOrder{
		Groups: groups,
		Total:  len(processed),
	}, nil
}

// GetModuleDependencies returns all dependencies of a module (recursive)
func (r *DependencyResolver) GetModuleDependencies(modulePath string, recursive bool) ([]string, error) {
	module, exists := r.graph.Modules[modulePath]
	if !exists {
		return nil, fmt.Errorf("module %s not found", modulePath)
	}

	if !recursive {
		return module.Dependencies, nil
	}

	// Get all dependencies recursively
	deps := make(map[string]bool)
	r.collectDependencies(modulePath, deps)

	// Convert map to slice
	var result []string
	for dep := range deps {
		if dep != modulePath { // Don't include self
			result = append(result, dep)
		}
	}

	sort.Strings(result)
	return result, nil
}

// collectDependencies recursively collects all dependencies
func (r *DependencyResolver) collectDependencies(module string, deps map[string]bool) {
	if deps[module] {
		return // Already processed
	}

	deps[module] = true

	for _, dep := range r.graph.Dependencies[module] {
		r.collectDependencies(dep, deps)
	}
}

// GetModuleDependents returns all modules that depend on the given module
func (r *DependencyResolver) GetModuleDependents(modulePath string, recursive bool) ([]string, error) {
	_, exists := r.graph.Modules[modulePath]
	if !exists {
		return nil, fmt.Errorf("module %s not found", modulePath)
	}

	if !recursive {
		return r.graph.Dependents[modulePath], nil
	}

	// Get all dependents recursively
	deps := make(map[string]bool)
	r.collectDependents(modulePath, deps)

	// Convert map to slice
	var result []string
	for dep := range deps {
		if dep != modulePath { // Don't include self
			result = append(result, dep)
		}
	}

	sort.Strings(result)
	return result, nil
}

// collectDependents recursively collects all dependents
func (r *DependencyResolver) collectDependents(module string, deps map[string]bool) {
	if deps[module] {
		return // Already processed
	}

	deps[module] = true

	for _, dep := range r.graph.Dependents[module] {
		r.collectDependents(dep, deps)
	}
}

// GetImpactedModules returns all modules that would be impacted by changes to the given module
func (r *DependencyResolver) GetImpactedModules(modulePath string) ([]string, error) {
	return r.GetModuleDependents(modulePath, true)
}

// PrintDependencyTree prints the dependency tree for visualization
func (r *DependencyResolver) PrintDependencyTree() string {
	var sb strings.Builder

	// Find root modules (no dependencies)
	var roots []string
	for module := range r.graph.Modules {
		if len(r.graph.Dependencies[module]) == 0 {
			roots = append(roots, module)
		}
	}

	sort.Strings(roots)

	// Print tree for each root
	visited := make(map[string]bool)
	for _, root := range roots {
		r.printModuleTree(&sb, root, "", visited)
	}

	// Print any unvisited modules (in case of disconnected components)
	for module := range r.graph.Modules {
		if !visited[module] {
			r.printModuleTree(&sb, module, "", visited)
		}
	}

	return sb.String()
}

// printModuleTree recursively prints a module and its dependents
func (r *DependencyResolver) printModuleTree(sb *strings.Builder, module, prefix string, visited map[string]bool) {
	if visited[module] {
		sb.WriteString(fmt.Sprintf("%s%s (already shown)\n", prefix, filepath.Base(module)))
		return
	}

	visited[module] = true

	// Print current module
	moduleInfo := r.graph.Modules[module]
	status := ""
	if moduleInfo.Config.Skip {
		status = " [SKIP]"
	}
	sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, filepath.Base(module), status))

	// Print dependents
	dependents := r.graph.Dependents[module]
	sort.Strings(dependents)

	for i, dep := range dependents {
		if i == len(dependents)-1 {
			r.printModuleTree(sb, dep, prefix+"└── ", visited)
		} else {
			r.printModuleTree(sb, dep, prefix+"├── ", visited)
		}
	}
}

// GetStats returns statistics about the dependency graph
func (r *DependencyResolver) GetStats() map[string]interface{} {
	totalDeps := 0
	maxDeps := 0
	maxDependents := 0
	skippedCount := 0

	for module := range r.graph.Modules {
		deps := len(r.graph.Dependencies[module])
		dependents := len(r.graph.Dependents[module])

		totalDeps += deps
		if deps > maxDeps {
			maxDeps = deps
		}
		if dependents > maxDependents {
			maxDependents = dependents
		}

		if r.graph.Modules[module].Config.Skip {
			skippedCount++
		}
	}

	avgDeps := float64(0)
	if len(r.graph.Modules) > 0 {
		avgDeps = float64(totalDeps) / float64(len(r.graph.Modules))
	}

	return map[string]interface{}{
		"total_modules":      len(r.graph.Modules),
		"total_dependencies": totalDeps,
		"avg_dependencies":   avgDeps,
		"max_dependencies":   maxDeps,
		"max_dependents":     maxDependents,
		"skipped_modules":    skippedCount,
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
