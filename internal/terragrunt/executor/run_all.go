package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/terragrunt/parser"
	"github.com/catherinevee/driftmgr/internal/terragrunt/resolver"
)

// RunAllOptions configures the run-all execution
type RunAllOptions struct {
	Command            string            `json:"command"`             // Terraform command to run (plan, apply, destroy, etc.)
	Args               []string          `json:"args"`                // Additional arguments
	Parallelism        int               `json:"parallelism"`         // Max parallel executions
	IgnoreErrors       bool              `json:"ignore_errors"`       // Continue on error
	IgnoreDependencies bool              `json:"ignore_dependencies"` // Ignore dependency order
	IncludeSkipped     bool              `json:"include_skipped"`     // Include skipped modules
	TargetModules      []string          `json:"target_modules"`      // Specific modules to target
	ExcludeModules     []string          `json:"exclude_modules"`     // Modules to exclude
	Environment        map[string]string `json:"environment"`         // Environment variables
	Timeout            time.Duration     `json:"timeout"`             // Timeout per module
	DryRun             bool              `json:"dry_run"`             // Don't actually execute
	AutoApprove        bool              `json:"auto_approve"`        // Auto-approve for apply/destroy
}

// RunAllResult contains the results of a run-all execution
type RunAllResult struct {
	StartTime      time.Time                    `json:"start_time"`
	EndTime        time.Time                    `json:"end_time"`
	Duration       time.Duration                `json:"duration"`
	ModuleResults  map[string]*ModuleExecResult `json:"module_results"`
	SuccessCount   int                          `json:"success_count"`
	FailureCount   int                          `json:"failure_count"`
	SkippedCount   int                          `json:"skipped_count"`
	ExecutionOrder *resolver.ExecutionOrder     `json:"execution_order"`
	Errors         []error                      `json:"errors,omitempty"`
}

// ModuleExecResult contains the result of executing a single module
type ModuleExecResult struct {
	Module    string                `json:"module"`
	Status    resolver.ModuleStatus `json:"status"`
	StartTime time.Time             `json:"start_time"`
	EndTime   time.Time             `json:"end_time"`
	Duration  time.Duration         `json:"duration"`
	Command   string                `json:"command"`
	Output    string                `json:"output"`
	Error     error                 `json:"error,omitempty"`
	ExitCode  int                   `json:"exit_code"`
	Changes   *TerraformChanges     `json:"changes,omitempty"`
}

// TerraformChanges represents changes detected by terraform
type TerraformChanges struct {
	Add     int `json:"add"`
	Change  int `json:"change"`
	Destroy int `json:"destroy"`
}

// TerragruntExecutor executes terragrunt commands across modules
type TerragruntExecutor struct {
	resolver  *resolver.DependencyResolver
	options   *RunAllOptions
	results   *RunAllResult
	mu        sync.Mutex
	wg        sync.WaitGroup
	semaphore chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewTerragruntExecutor creates a new terragrunt executor
func NewTerragruntExecutor(options *RunAllOptions) *TerragruntExecutor {
	if options.Parallelism <= 0 {
		options.Parallelism = 10 // Default parallelism
	}

	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Minute // Default timeout per module
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TerragruntExecutor{
		resolver:  resolver.NewDependencyResolver(),
		options:   options,
		semaphore: make(chan struct{}, options.Parallelism),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// RunAll executes the command across all modules
func (e *TerragruntExecutor) RunAll(rootDir string) (*RunAllResult, error) {
	// Initialize results
	e.results = &RunAllResult{
		StartTime:     time.Now(),
		ModuleResults: make(map[string]*ModuleExecResult),
	}

	// Resolve dependencies
	graph, err := e.resolver.ResolveDirectory(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Filter modules based on options
	modules := e.filterModules(graph)
	if len(modules) == 0 {
		return e.results, fmt.Errorf("no modules to execute")
	}

	// Get execution order
	if !e.options.IgnoreDependencies {
		execOrder, err := e.resolver.GetExecutionOrder(e.options.IncludeSkipped)
		if err != nil {
			return nil, fmt.Errorf("failed to determine execution order: %w", err)
		}
		e.results.ExecutionOrder = execOrder

		// Execute in dependency order
		for _, group := range execOrder.Groups {
			if err := e.executeGroup(group, graph); err != nil {
				if !e.options.IgnoreErrors {
					e.cancel()
					break
				}
			}
		}
	} else {
		// Execute all modules in parallel (ignoring dependencies)
		if err := e.executeGroup(modules, graph); err != nil && !e.options.IgnoreErrors {
			e.cancel()
		}
	}

	// Wait for all executions to complete
	e.wg.Wait()

	// Finalize results
	e.results.EndTime = time.Now()
	e.results.Duration = e.results.EndTime.Sub(e.results.StartTime)

	return e.results, nil
}

// filterModules filters modules based on options
func (e *TerragruntExecutor) filterModules(graph *resolver.DependencyGraph) []string {
	var modules []string

	for path, module := range graph.Modules {
		// Skip if module should be skipped
		if module.Config.Skip && !e.options.IncludeSkipped {
			continue
		}

		// Check if module is in target list
		if len(e.options.TargetModules) > 0 {
			found := false
			for _, target := range e.options.TargetModules {
				if path == target || filepath.Base(path) == target {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check if module is in exclude list
		excluded := false
		for _, exclude := range e.options.ExcludeModules {
			if path == exclude || filepath.Base(path) == exclude {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		modules = append(modules, path)
	}

	return modules
}

// executeGroup executes a group of modules in parallel
func (e *TerragruntExecutor) executeGroup(group []string, graph *resolver.DependencyGraph) error {
	var groupErrors []error

	for _, modulePath := range group {
		module := graph.Modules[modulePath]
		if module == nil {
			continue
		}

		// Acquire semaphore
		e.semaphore <- struct{}{}
		e.wg.Add(1)

		go func(m *resolver.Module) {
			defer func() {
				<-e.semaphore
				e.wg.Done()
			}()

			// Check if context is cancelled
			select {
			case <-e.ctx.Done():
				e.recordModuleResult(m.Path, &ModuleExecResult{
					Module:    m.Path,
					Status:    resolver.ModuleStatusSkipped,
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Error:     e.ctx.Err(),
				})
				return
			default:
			}

			// Execute module
			result := e.executeModule(m)
			e.recordModuleResult(m.Path, result)

			if result.Error != nil && !e.options.IgnoreErrors {
				e.mu.Lock()
				groupErrors = append(groupErrors, result.Error)
				e.mu.Unlock()
				e.cancel()
			}
		}(module)
	}

	// Wait for this group to complete before returning
	e.wg.Wait()

	if len(groupErrors) > 0 {
		return groupErrors[0]
	}

	return nil
}

// executeModule executes terraform/terragrunt on a single module
func (e *TerragruntExecutor) executeModule(module *resolver.Module) *ModuleExecResult {
	result := &ModuleExecResult{
		Module:    module.Path,
		Status:    resolver.ModuleStatusRunning,
		StartTime: time.Now(),
	}

	if e.options.DryRun {
		result.Status = resolver.ModuleStatusCompleted
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Output = fmt.Sprintf("[DRY RUN] Would execute: terragrunt %s %v", e.options.Command, e.options.Args)
		return result
	}

	// Build command
	args := []string{e.options.Command}
	args = append(args, e.options.Args...)

	// Add auto-approve if needed
	if e.options.AutoApprove && (e.options.Command == "apply" || e.options.Command == "destroy") {
		args = append(args, "-auto-approve")
	}

	cmd := exec.CommandContext(e.ctx, "terragrunt", args...)
	cmd.Dir = module.Path

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range e.options.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(e.ctx, e.options.Timeout)
	defer cancel()

	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmd.Dir = module.Path
	cmd.Env = cmd.Env

	// Capture output
	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Status = resolver.ModuleStatusFailed
		result.Error = fmt.Errorf("failed to execute terragrunt in %s: %w", module.Path, err)
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
	} else {
		result.Status = resolver.ModuleStatusCompleted
		// Parse output for changes if this was a plan/apply
		if e.options.Command == "plan" || e.options.Command == "apply" {
			result.Changes = parseChanges(result.Output)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// recordModuleResult records the result of a module execution
func (e *TerragruntExecutor) recordModuleResult(path string, result *ModuleExecResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.results.ModuleResults[path] = result

	switch result.Status {
	case resolver.ModuleStatusCompleted:
		e.results.SuccessCount++
	case resolver.ModuleStatusFailed:
		e.results.FailureCount++
		if result.Error != nil {
			e.results.Errors = append(e.results.Errors, result.Error)
		}
	case resolver.ModuleStatusSkipped:
		e.results.SkippedCount++
	}
}

// parseChanges parses terraform output for resource changes
func parseChanges(output string) *TerraformChanges {
	// This is a simplified parser - in production you'd want more robust parsing
	changes := &TerraformChanges{}

	// Look for plan summary lines like:
	// Plan: 1 to add, 2 to change, 3 to destroy.
	// or
	// Apply complete! Resources: 1 added, 2 changed, 3 destroyed.

	// This would need proper implementation based on terraform output format
	// For now, return empty changes

	return changes
}

// GetProgress returns current execution progress
func (e *TerragruntExecutor) GetProgress() map[string]interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()

	total := len(e.results.ModuleResults)
	completed := e.results.SuccessCount + e.results.FailureCount + e.results.SkippedCount

	return map[string]interface{}{
		"total":     total,
		"completed": completed,
		"success":   e.results.SuccessCount,
		"failed":    e.results.FailureCount,
		"skipped":   e.results.SkippedCount,
		"running":   total - completed,
		"duration":  time.Since(e.results.StartTime).String(),
	}
}

// Cancel cancels the execution
func (e *TerragruntExecutor) Cancel() {
	e.cancel()
}

// GeneratePlanSummary generates a summary of the execution plan
func (e *TerragruntExecutor) GeneratePlanSummary(rootDir string) (*PlanSummary, error) {
	graph, err := e.resolver.ResolveDirectory(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	modules := e.filterModules(graph)
	execOrder, err := e.resolver.GetExecutionOrder(e.options.IncludeSkipped)
	if err != nil {
		return nil, err
	}

	summary := &PlanSummary{
		TotalModules:   len(modules),
		ExecutionOrder: execOrder,
		Parallelism:    e.options.Parallelism,
		Command:        e.options.Command,
		Modules:        make([]ModulePlan, 0, len(modules)),
	}

	for _, module := range modules {
		deps, _ := e.resolver.GetModuleDependencies(module, false)
		dependents, _ := e.resolver.GetModuleDependents(module, false)

		summary.Modules = append(summary.Modules, ModulePlan{
			Path:         module,
			Dependencies: deps,
			Dependents:   dependents,
			Skipped:      graph.Modules[module].Config.Skip,
		})
	}

	return summary, nil
}

// PlanSummary contains a summary of the execution plan
type PlanSummary struct {
	TotalModules   int                      `json:"total_modules"`
	ExecutionOrder *resolver.ExecutionOrder `json:"execution_order"`
	Parallelism    int                      `json:"parallelism"`
	Command        string                   `json:"command"`
	Modules        []ModulePlan             `json:"modules"`
}

// ModulePlan contains plan information for a single module
type ModulePlan struct {
	Path         string   `json:"path"`
	Dependencies []string `json:"dependencies"`
	Dependents   []string `json:"dependents"`
	Skipped      bool     `json:"skipped"`
}
