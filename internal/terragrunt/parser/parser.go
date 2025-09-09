package parser

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type TerragruntParser struct {
	rootDir      string
	hclParser    *hclparse.Parser
	configs      map[string]*TerragruntConfig
	dependencies map[string][]string
	mu           sync.RWMutex
}

// TerragruntConfig is defined in hcl_parser.go

// RemoteStateConfig, GenerateConfig, and DependencyConfig are defined in hcl_parser.go

func NewTerragruntParser(rootDir string) *TerragruntParser {
	return &TerragruntParser{
		rootDir:      rootDir,
		hclParser:    hclparse.NewParser(),
		configs:      make(map[string]*TerragruntConfig),
		dependencies: make(map[string][]string),
	}
}

func (tp *TerragruntParser) ParseAll() error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Find all terragrunt.hcl files
	terragruntFiles := make([]string, 0)
	err := filepath.WalkDir(tp.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip .terragrunt-cache directories
			if strings.Contains(path, ".terragrunt-cache") {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() == "terragrunt.hcl" {
			terragruntFiles = append(terragruntFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Parse each file
	var wg sync.WaitGroup
	errCh := make(chan error, len(terragruntFiles))
	configCh := make(chan struct {
		path   string
		config *TerragruntConfig
	}, len(terragruntFiles))

	for _, file := range terragruntFiles {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()

			config, err := tp.parseFile(f)
			if err != nil {
				errCh <- fmt.Errorf("failed to parse %s: %w", f, err)
				return
			}

			configCh <- struct {
				path   string
				config *TerragruntConfig
			}{path: f, config: config}
		}(file)
	}

	wg.Wait()
	close(errCh)
	close(configCh)

	// Check for errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	// Store configs
	for cfg := range configCh {
		tp.configs[cfg.path] = cfg.config
	}

	// Build dependency graph
	tp.buildDependencyGraph()

	return nil
}

func (tp *TerragruntParser) parseFile(filePath string) (*TerragruntConfig, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	config := &TerragruntConfig{
		FilePath:     filePath,
		Inputs:       make(map[string]interface{}),
		Locals:       make(map[string]interface{}),
		Dependencies: make([]Dependency, 0),
		Include:      make([]IncludeConfig, 0),
	}

	// Parse HCL
	file, diags := tp.hclParser.ParseHCL(content, filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}

	// Extract configuration blocks
	body := file.Body
	if body == nil {
		// Simplified parsing for basic HCL structure
		if err := tp.parseSimplifiedHCL(string(content), config); err != nil {
			return nil, err
		}
	} else {
		if err := tp.parseHCLBody(body, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (tp *TerragruntParser) parseSimplifiedHCL(content string, config *TerragruntConfig) error {
	// Extract terraform block
	if match := regexp.MustCompile(`terraform\s*\{[^}]*source\s*=\s*"([^"]+)"`).FindStringSubmatch(content); len(match) > 1 {
		config.TerraformSource = match[1]
	}

	// Extract remote_state block
	if remoteStateMatch := regexp.MustCompile(`remote_state\s*\{([^}]+)\}`).FindStringSubmatch(content); len(remoteStateMatch) > 1 {
		config.RemoteState = tp.parseRemoteState(remoteStateMatch[1])
	}

	// Extract dependencies
	depMatches := regexp.MustCompile(`dependency\s+"([^"]+)"\s*\{([^}]+)\}`).FindAllStringSubmatch(content, -1)
	for _, match := range depMatches {
		if len(match) > 2 {
			dep := Dependency{
				Name:    match[1],
				Enabled: true,
			}

			// Extract config_path
			if pathMatch := regexp.MustCompile(`config_path\s*=\s*"([^"]+)"`).FindStringSubmatch(match[2]); len(pathMatch) > 1 {
				dep.ConfigPath = pathMatch[1]
			}

			config.Dependencies = append(config.Dependencies, dep)
		}
	}

	// Extract include blocks
	includeMatches := regexp.MustCompile(`include\s*(?:"[^"]+"\s*)?\{[^}]*path\s*=\s*"([^"]+)"`).FindAllStringSubmatch(content, -1)
	for _, match := range includeMatches {
		if len(match) > 1 {
			// TODO: Convert to IncludeConfig struct
			// config.Include = append(config.Include, IncludeConfig{Path: match[1]})
		}
	}

	// Extract inputs
	if inputsMatch := regexp.MustCompile(`inputs\s*=\s*\{([^}]+)\}`).FindStringSubmatch(content); len(inputsMatch) > 1 {
		config.Inputs = tp.parseInputs(inputsMatch[1])
	}

	// Extract IAM role
	if iamMatch := regexp.MustCompile(`iam_role\s*=\s*"([^"]+)"`).FindStringSubmatch(content); len(iamMatch) > 1 {
		config.IamRole = iamMatch[1]
	}

	// Extract prevent_destroy
	// TODO: Add PreventDestroy field to TerragruntConfig or handle differently
	// if regexp.MustCompile(`prevent_destroy\s*=\s*true`).MatchString(content) {
	//	config.PreventDestroy = true
	// }

	return nil
}

func (tp *TerragruntParser) parseRemoteState(content string) *RemoteStateConfig {
	rs := &RemoteStateConfig{
		Config: make(map[string]interface{}),
	}

	// Extract backend
	if match := regexp.MustCompile(`backend\s*=\s*"([^"]+)"`).FindStringSubmatch(content); len(match) > 1 {
		rs.Backend = match[1]
	}

	// Extract config block
	if configMatch := regexp.MustCompile(`config\s*=\s*\{([^}]+)\}`).FindStringSubmatch(content); len(configMatch) > 1 {
		rs.Config = tp.parseConfigBlock(configMatch[1])
	}

	// Extract generate block
	if genMatch := regexp.MustCompile(`generate\s*=\s*\{([^}]+)\}`).FindStringSubmatch(content); len(genMatch) > 1 {
		rs.Generate = tp.parseGenerateBlock(genMatch[1])
	}

	return rs
}

func (tp *TerragruntParser) parseConfigBlock(content string) map[string]interface{} {
	config := make(map[string]interface{})

	// Parse key-value pairs
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Match key = "value" pattern
		if match := regexp.MustCompile(`(\w+)\s*=\s*"([^"]+)"`).FindStringSubmatch(line); len(match) > 2 {
			config[match[1]] = match[2]
		} else if match := regexp.MustCompile(`(\w+)\s*=\s*(\w+)`).FindStringSubmatch(line); len(match) > 2 {
			// Handle boolean or numeric values
			if match[2] == "true" {
				config[match[1]] = true
			} else if match[2] == "false" {
				config[match[1]] = false
			} else {
				config[match[1]] = match[2]
			}
		}
	}

	return config
}

func (tp *TerragruntParser) parseGenerateBlock(content string) *GenerateConfig {
	gen := &GenerateConfig{}

	if match := regexp.MustCompile(`path\s*=\s*"([^"]+)"`).FindStringSubmatch(content); len(match) > 1 {
		gen.Path = match[1]
	}

	if match := regexp.MustCompile(`if_exists\s*=\s*"([^"]+)"`).FindStringSubmatch(content); len(match) > 1 {
		gen.IfExists = match[1]
	}

	return gen
}

func (tp *TerragruntParser) parseInputs(content string) map[string]interface{} {
	inputs := make(map[string]interface{})

	// Simple key-value parsing
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Match key = value patterns
		if match := regexp.MustCompile(`(\w+)\s*=\s*(.+)`).FindStringSubmatch(line); len(match) > 2 {
			key := match[1]
			value := strings.TrimSpace(match[2])

			// Remove trailing comma if present
			value = strings.TrimSuffix(value, ",")

			// Parse value type
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				inputs[key] = strings.Trim(value, "\"")
			} else if value == "true" || value == "false" {
				inputs[key] = value == "true"
			} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				// Array value
				inputs[key] = tp.parseArrayValue(value)
			} else {
				inputs[key] = value
			}
		}
	}

	return inputs
}

func (tp *TerragruntParser) parseArrayValue(value string) []string {
	// Remove brackets
	value = strings.Trim(value, "[]")

	// Split by comma and clean up
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		item = strings.Trim(item, "\"")
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}

func (tp *TerragruntParser) parseHCLBody(body hcl.Body, config *TerragruntConfig) error {
	// This would be a more complete HCL parsing implementation
	// For now, fall back to simplified parsing
	return errors.New("full HCL parsing not implemented")
}

func (tp *TerragruntParser) buildDependencyGraph() {
	tp.dependencies = make(map[string][]string)

	for path, config := range tp.configs {
		deps := make([]string, 0)

		// Add explicit dependencies
		for _, dep := range config.Dependencies {
			if dep.ConfigPath != "" {
				// Resolve relative path
				depPath := filepath.Join(filepath.Dir(path), dep.ConfigPath, "terragrunt.hcl")
				depPath = filepath.Clean(depPath)
				deps = append(deps, depPath)
			}
		}

		// Add include dependencies
		// TODO: Fix include path processing
		// for _, includeConfig := range config.Include {
		//	// Resolve include path
		//	incPath := filepath.Join(filepath.Dir(path), includeConfig.Path)
		//	incPath = filepath.Clean(incPath)
		//	deps = append(deps, incPath)
		// }

		tp.dependencies[path] = deps
	}
}

func (tp *TerragruntParser) GetConfig(path string) (*TerragruntConfig, error) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	config, exists := tp.configs[path]
	if !exists {
		return nil, fmt.Errorf("config not found for path: %s", path)
	}

	return config, nil
}

func (tp *TerragruntParser) GetAllConfigs() map[string]*TerragruntConfig {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	configs := make(map[string]*TerragruntConfig)
	for k, v := range tp.configs {
		configs[k] = v
	}

	return configs
}

func (tp *TerragruntParser) GetDependencies(path string) []string {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	deps, exists := tp.dependencies[path]
	if !exists {
		return []string{}
	}

	return deps
}

func (tp *TerragruntParser) GetDependencyOrder() ([]string, error) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	// Topological sort for execution order
	visited := make(map[string]bool)
	stack := make([]string, 0)

	var visit func(string) error
	visit = func(path string) error {
		if visited[path] {
			return nil
		}

		visited[path] = true

		// Visit dependencies first
		for _, dep := range tp.dependencies[path] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		stack = append(stack, path)
		return nil
	}

	// Visit all configs
	for path := range tp.configs {
		if err := visit(path); err != nil {
			return nil, err
		}
	}

	return stack, nil
}

func (tp *TerragruntParser) ValidateDependencies() []error {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	errors := make([]error, 0)

	// Check for circular dependencies
	for path := range tp.configs {
		if tp.hasCircularDependency(path, make(map[string]bool)) {
			errors = append(errors, fmt.Errorf("circular dependency detected for %s", path))
		}
	}

	// Check for missing dependencies
	for path, deps := range tp.dependencies {
		for _, dep := range deps {
			if _, exists := tp.configs[dep]; !exists && !strings.HasSuffix(dep, ".hcl") {
				errors = append(errors, fmt.Errorf("missing dependency %s for %s", dep, path))
			}
		}
	}

	return errors
}

func (tp *TerragruntParser) hasCircularDependency(path string, visiting map[string]bool) bool {
	if visiting[path] {
		return true
	}

	visiting[path] = true
	defer delete(visiting, path)

	for _, dep := range tp.dependencies[path] {
		if tp.hasCircularDependency(dep, visiting) {
			return true
		}
	}

	return false
}
