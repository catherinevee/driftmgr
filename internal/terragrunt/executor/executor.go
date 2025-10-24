package executor

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/terragrunt/parser"
)

type TerragruntExecutor struct {
	parser          *parser.TerragruntParser
	terragruntPath  string
	workDir         string
	parallelism     int
	autoApprove     bool
	nonInteractive  bool
	includeExternal bool
	logLevel        string
	envVars         map[string]string
	mu              sync.RWMutex
	execHistory     []ExecutionResult
}

type ExecutionResult struct {
	ConfigPath   string    `json:"config_path"`
	Command      string    `json:"command"`
	Args         []string  `json:"args"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	ExitCode     int       `json:"exit_code"`
	Output       string    `json:"output"`
	Error        string    `json:"error,omitempty"`
	Dependencies []string  `json:"dependencies"`
	RetryCount   int       `json:"retry_count"`
}

type ExecutionOptions struct {
	Command                string        `json:"command"`
	Args                   []string      `json:"args"`
	TargetPaths            []string      `json:"target_paths,omitempty"`
	ExcludePaths           []string      `json:"exclude_paths,omitempty"`
	IncludeDepends         bool          `json:"include_depends"`
	IgnoreDependencyErrors bool          `json:"ignore_dependency_errors"`
	Parallelism            int           `json:"parallelism"`
	RetryMaxAttempts       int           `json:"retry_max_attempts"`
	RetryInterval          time.Duration `json:"retry_interval"`
}

func NewTerragruntExecutor(parser *parser.TerragruntParser, workDir string) *TerragruntExecutor {
	return &TerragruntExecutor{
		parser:         parser,
		terragruntPath: "terragrunt",
		workDir:        workDir,
		parallelism:    1,
		autoApprove:    false,
		nonInteractive: true,
		logLevel:       "info",
		envVars:        make(map[string]string),
		execHistory:    make([]ExecutionResult, 0),
	}
}

func (te *TerragruntExecutor) RunAll(ctx context.Context, opts ExecutionOptions) ([]ExecutionResult, error) {
	// Get execution order
	executionOrder, err := te.parser.GetDependencyOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency order: %w", err)
	}

	// Filter paths based on options
	targetPaths := te.filterPaths(executionOrder, opts.TargetPaths, opts.ExcludePaths)

	if len(targetPaths) == 0 {
		return nil, errors.New("no matching configurations found")
	}

	// Execute based on parallelism
	if opts.Parallelism > 1 {
		return te.executeParallel(ctx, targetPaths, opts)
	}

	return te.executeSequential(ctx, targetPaths, opts)
}

func (te *TerragruntExecutor) Run(ctx context.Context, configPath string, opts ExecutionOptions) (*ExecutionResult, error) {
	config, err := te.parser.GetConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Build command
	args := te.buildCommandArgs(opts.Command, opts.Args, config)

	// Execute with retry logic
	var result *ExecutionResult
	maxAttempts := opts.RetryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result = te.executeCommand(ctx, filepath.Dir(configPath), args)
		result.ConfigPath = configPath
		result.Command = opts.Command
		result.Args = opts.Args
		result.Dependencies = te.parser.GetDependencies(configPath)
		result.RetryCount = attempt - 1

		if result.ExitCode == 0 {
			break
		}

		// Check if error is retryable
		// RetryableErrors can be added to TerragruntConfig if needed for advanced retry logic
		// if !te.isRetryableError(result.Error, config.RetryableErrors) {
		//	break
		// }

		if attempt < maxAttempts {
			time.Sleep(opts.RetryInterval)
		}
	}

	// Store result
	te.mu.Lock()
	te.execHistory = append(te.execHistory, *result)
	te.mu.Unlock()

	if result.ExitCode != 0 {
		return result, fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Error)
	}

	return result, nil
}

func (te *TerragruntExecutor) executeCommand(ctx context.Context, workDir string, args []string) *ExecutionResult {
	result := &ExecutionResult{
		StartTime: time.Now(),
	}

	// Create command
	cmd := exec.CommandContext(ctx, te.terragruntPath, args...)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range te.envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set Terragrunt-specific environment variables
	if te.nonInteractive {
		cmd.Env = append(cmd.Env, "TF_INPUT=false")
		cmd.Env = append(cmd.Env, "TERRAGRUNT_NON_INTERACTIVE=true")
	}

	if te.autoApprove {
		cmd.Env = append(cmd.Env, "TERRAGRUNT_AUTO_APPROVE=true")
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	result.EndTime = time.Now()
	result.Output = stdout.String()

	if err != nil {
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = err.Error()
		}

		// Extract exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

func (te *TerragruntExecutor) executeSequential(ctx context.Context, paths []string, opts ExecutionOptions) ([]ExecutionResult, error) {
	results := make([]ExecutionResult, 0, len(paths))

	for _, path := range paths {
		result, err := te.Run(ctx, path, opts)
		if err != nil && !opts.IgnoreDependencyErrors {
			return results, err
		}

		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

func (te *TerragruntExecutor) executeParallel(ctx context.Context, paths []string, opts ExecutionOptions) ([]ExecutionResult, error) {
	// Group paths by dependency level
	levels := te.groupByDependencyLevel(paths)
	results := make([]ExecutionResult, 0)

	for _, level := range levels {
		// Execute each level in parallel
		levelResults := make([]ExecutionResult, len(level))
		var wg sync.WaitGroup
		errCh := make(chan error, len(level))

		for i, path := range level {
			wg.Add(1)
			go func(idx int, p string) {
				defer wg.Done()

				result, err := te.Run(ctx, p, opts)
				if err != nil && !opts.IgnoreDependencyErrors {
					errCh <- err
					return
				}

				if result != nil {
					levelResults[idx] = *result
				}
			}(i, path)
		}

		wg.Wait()
		close(errCh)

		// Check for errors
		for err := range errCh {
			if err != nil {
				return results, err
			}
		}

		// Append level results
		for _, result := range levelResults {
			if result.ConfigPath != "" {
				results = append(results, result)
			}
		}
	}

	return results, nil
}

func (te *TerragruntExecutor) groupByDependencyLevel(paths []string) [][]string {
	// Calculate dependency depth for each path
	depths := make(map[string]int)

	var calculateDepth func(string) int
	calculateDepth = func(path string) int {
		if depth, exists := depths[path]; exists {
			return depth
		}

		deps := te.parser.GetDependencies(path)
		maxDepth := 0

		for _, dep := range deps {
			depDepth := calculateDepth(dep)
			if depDepth > maxDepth {
				maxDepth = depDepth
			}
		}

		depths[path] = maxDepth + 1
		return maxDepth + 1
	}

	// Calculate depths for all paths
	for _, path := range paths {
		calculateDepth(path)
	}

	// Group by depth level
	levels := make(map[int][]string)
	maxLevel := 0

	for path, depth := range depths {
		levels[depth] = append(levels[depth], path)
		if depth > maxLevel {
			maxLevel = depth
		}
	}

	// Convert to ordered slice
	result := make([][]string, 0, maxLevel)
	for i := 1; i <= maxLevel; i++ {
		if paths, exists := levels[i]; exists {
			result = append(result, paths)
		}
	}

	return result
}

func (te *TerragruntExecutor) buildCommandArgs(command string, args []string, config *parser.TerragruntConfig) []string {
	cmdArgs := make([]string, 0)

	// Add command
	cmdArgs = append(cmdArgs, command)

	// Add standard flags based on command
	switch command {
	case "apply", "destroy":
		if te.autoApprove {
			cmdArgs = append(cmdArgs, "-auto-approve")
		}
		if te.nonInteractive {
			cmdArgs = append(cmdArgs, "-input=false")
		}
	case "plan":
		if te.nonInteractive {
			cmdArgs = append(cmdArgs, "-input=false")
		}
		cmdArgs = append(cmdArgs, "-out=tfplan")
	}

	// Add log level
	cmdArgs = append(cmdArgs, fmt.Sprintf("--terragrunt-log-level=%s", te.logLevel))

	// Add IAM role if specified
	if config.IamRole != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--terragrunt-iam-role=%s", config.IamRole))
	}

	// Add parallelism
	if te.parallelism > 1 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--terragrunt-parallelism=%d", te.parallelism))
	}

	// Add custom args
	cmdArgs = append(cmdArgs, args...)

	return cmdArgs
}

func (te *TerragruntExecutor) filterPaths(allPaths []string, targetPaths, excludePaths []string) []string {
	filtered := make([]string, 0)

	// Create exclude map for faster lookup
	excludeMap := make(map[string]bool)
	for _, path := range excludePaths {
		excludeMap[path] = true
	}

	// If target paths specified, use only those
	if len(targetPaths) > 0 {
		for _, path := range targetPaths {
			if !excludeMap[path] {
				filtered = append(filtered, path)
			}
		}
		return filtered
	}

	// Otherwise, use all paths except excluded
	for _, path := range allPaths {
		if !excludeMap[path] {
			filtered = append(filtered, path)
		}
	}

	return filtered
}

func (te *TerragruntExecutor) isRetryableError(errorMsg string, retryableErrors []string) bool {
	if errorMsg == "" {
		return false
	}

	// Default retryable errors
	defaultRetryable := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"EOF",
		"rate limit",
		"throttled",
		"429",
		"503",
		"504",
	}

	// Check custom retryable errors
	for _, pattern := range retryableErrors {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(pattern)) {
			return true
		}
	}

	// Check default retryable errors
	for _, pattern := range defaultRetryable {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

func (te *TerragruntExecutor) ValidateAll(ctx context.Context) ([]ValidationResult, error) {
	configs := te.parser.GetAllConfigs()
	results := make([]ValidationResult, 0, len(configs))

	for path, config := range configs {
		result := te.validateConfig(ctx, path, config)
		results = append(results, result)
	}

	return results, nil
}

type ValidationResult struct {
	ConfigPath string   `json:"config_path"`
	Valid      bool     `json:"valid"`
	Errors     []string `json:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

func (te *TerragruntExecutor) validateConfig(ctx context.Context, path string, config *parser.TerragruntConfig) ValidationResult {
	result := ValidationResult{
		ConfigPath: path,
		Valid:      true,
		Errors:     make([]string, 0),
		Warnings:   make([]string, 0),
	}

	// Check terraform source
	if config.TerraformSource == "" {
		result.Errors = append(result.Errors, "terraform source not specified")
		result.Valid = false
	}

	// Check remote state configuration
	if config.RemoteState == nil {
		result.Warnings = append(result.Warnings, "no remote state configuration")
	} else {
		if config.RemoteState.Backend == "" {
			result.Errors = append(result.Errors, "remote state backend not specified")
			result.Valid = false
		}
	}

	// Check dependencies
	for _, dep := range config.Dependencies {
		if dep.ConfigPath == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("dependency %s has no config_path", dep.Name))
			result.Valid = false
		} else {
			// Check if dependency path exists
			depPath := filepath.Join(filepath.Dir(path), dep.ConfigPath, "terragrunt.hcl")
			if _, err := os.Stat(depPath); os.IsNotExist(err) {
				result.Errors = append(result.Errors, fmt.Sprintf("dependency %s path does not exist: %s", dep.Name, depPath))
				result.Valid = false
			}
		}
	}

	// Check for circular dependencies
	if te.hasCircularDependency(path, make(map[string]bool)) {
		result.Errors = append(result.Errors, "circular dependency detected")
		result.Valid = false
	}

	// Validate with terragrunt validate
	validateResult := te.executeCommand(ctx, filepath.Dir(path), []string{"validate"})
	if validateResult.ExitCode != 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("terragrunt validate failed: %s", validateResult.Error))
		result.Valid = false
	}

	return result
}

func (te *TerragruntExecutor) hasCircularDependency(path string, visiting map[string]bool) bool {
	if visiting[path] {
		return true
	}

	visiting[path] = true
	defer delete(visiting, path)

	deps := te.parser.GetDependencies(path)
	for _, dep := range deps {
		if te.hasCircularDependency(dep, visiting) {
			return true
		}
	}

	return false
}

func (te *TerragruntExecutor) GetExecutionHistory() []ExecutionResult {
	te.mu.RLock()
	defer te.mu.RUnlock()

	history := make([]ExecutionResult, len(te.execHistory))
	copy(history, te.execHistory)
	return history
}

func (te *TerragruntExecutor) SetEnvVar(key, value string) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.envVars[key] = value
}

func (te *TerragruntExecutor) SetAutoApprove(enabled bool) {
	te.autoApprove = enabled
}

func (te *TerragruntExecutor) SetNonInteractive(enabled bool) {
	te.nonInteractive = enabled
}

func (te *TerragruntExecutor) SetLogLevel(level string) {
	te.logLevel = level
}

func (te *TerragruntExecutor) SetParallelism(parallelism int) {
	te.parallelism = parallelism
}

func (te *TerragruntExecutor) StreamOutput(ctx context.Context, configPath string, opts ExecutionOptions, outputChan chan<- string) error {
	config, err := te.parser.GetConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	args := te.buildCommandArgs(opts.Command, opts.Args, config)

	cmd := exec.CommandContext(ctx, te.terragruntPath, args...)
	cmd.Dir = filepath.Dir(configPath)

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range te.envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return err
	}

	// Stream output
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case outputChan <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case outputChan <- fmt.Sprintf("[ERROR] %s", scanner.Text()):
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()
	wg.Wait()
	close(outputChan)

	return err
}
