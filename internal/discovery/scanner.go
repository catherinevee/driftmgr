package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// BackendConfig represents a discovered Terraform backend configuration
type BackendConfig struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Attributes   map[string]interface{} `json:"attributes"`
	FilePath     string                 `json:"file_path"`
	Module       string                 `json:"module,omitempty"`
	Workspace    string                 `json:"workspace,omitempty"`
	ConfigPath   string                 `json:"config_path"`
	Config       map[string]interface{} `json:"config"`
	WorkspaceDir string                 `json:"workspace_dir,omitempty"`
}

// Scanner discovers Terraform backend configurations
type Scanner struct {
	rootDir     string
	workers     int
	ignoreRules []string
	mu          sync.RWMutex
	backends    []BackendConfig
}

// NewScanner creates a new backend scanner
func NewScanner(rootDir string, workers int) *Scanner {
	if workers <= 0 {
		workers = 4
	}
	return &Scanner{
		rootDir:  rootDir,
		workers:  workers,
		backends: make([]BackendConfig, 0),
		ignoreRules: []string{
			".terraform",
			".git",
			"node_modules",
			"vendor",
		},
	}
}

// Scan discovers all backend configurations in the directory tree
func (s *Scanner) Scan(ctx context.Context) ([]BackendConfig, error) {
	fileChan := make(chan string, 100)
	resultChan := make(chan BackendConfig, 100)
	errChan := make(chan error, 1)

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go s.worker(ctx, fileChan, resultChan, errChan, &wg)
	}

	// Walk filesystem in separate goroutine
	go func() {
		defer close(fileChan)
		err := filepath.WalkDir(s.rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip ignored directories
			if d.IsDir() && s.shouldIgnoreDir(d.Name()) {
				return filepath.SkipDir
			}

			// Process Terraform files
			if !d.IsDir() && s.isTerraformFile(path) {
				select {
				case fileChan <- path:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		})
		if err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
	}()

	// Collect results in separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect all results
	for backend := range resultChan {
		s.mu.Lock()
		s.backends = append(s.backends, backend)
		s.mu.Unlock()
	}

	// Check for errors
	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	return s.backends, nil
}

// worker processes files and extracts backend configurations
func (s *Scanner) worker(ctx context.Context, files <-chan string, results chan<- BackendConfig, errors chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	parser := hclparse.NewParser()

	for file := range files {
		select {
		case <-ctx.Done():
			return
		default:
			backends, err := s.parseBackendsFromFile(file, parser)
			if err != nil {
				// Log error but continue processing
				continue
			}
			for _, backend := range backends {
				results <- backend
			}
		}
	}
}

// parseBackendsFromFile extracts backend configuration from a Terraform file
func (s *Scanner) parseBackendsFromFile(filePath string, parser *hclparse.Parser) ([]BackendConfig, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse HCL file
	file, diags := parser.ParseHCL(src, filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	// Extract terraform blocks
	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "terraform"},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to extract terraform blocks: %s", diags.Error())
	}

	var backends []BackendConfig

	// Process each terraform block
	for _, block := range content.Blocks {
		// Look for backend configuration
		backendContent, _, _ := block.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "backend", LabelNames: []string{"type"}},
			},
		})

		for _, backendBlock := range backendContent.Blocks {
			if len(backendBlock.Labels) > 0 {
				config := BackendConfig{
					Type:       backendBlock.Labels[0],
					FilePath:   filePath,
					Module:     s.extractModuleName(filePath),
					Attributes: make(map[string]interface{}),
				}

				// Parse backend attributes
				attrs, diags := backendBlock.Body.JustAttributes()
				if !diags.HasErrors() {
					for name, attr := range attrs {
						val, err := s.extractAttributeValue(attr)
						if err == nil {
							config.Attributes[name] = val
						}
					}
				}

				// Extract workspace if defined
				if ws, ok := config.Attributes["workspace"].(string); ok {
					config.Workspace = ws
				}

				backends = append(backends, config)
			}
		}
	}

	return backends, nil
}

// extractAttributeValue extracts the value from an HCL attribute
func (s *Scanner) extractAttributeValue(attr *hcl.Attribute) (interface{}, error) {
	// Try to get literal value
	val, diags := attr.Expr.Value(nil)
	if !diags.HasErrors() {
		// Check the type using cty
		ty := val.Type()
		if ty == cty.String {
			return val.AsString(), nil
		} else if ty == cty.Bool {
			return val.True(), nil
		} else if ty == cty.Number {
			f, _ := val.AsBigFloat().Float64()
			return f, nil
		}
	}

	// For complex expressions, return as string representation
	// This handles variables, references, and complex expressions
	return fmt.Sprintf("%v", attr.Expr), nil
}

// extractModuleName extracts module name from file path
func (s *Scanner) extractModuleName(filePath string) string {
	rel, err := filepath.Rel(s.rootDir, filePath)
	if err != nil {
		return ""
	}

	dir := filepath.Dir(rel)
	if dir == "." {
		return "root"
	}

	// Clean and return module path
	return strings.ReplaceAll(filepath.ToSlash(dir), "/", ".")
}

// shouldIgnoreDir checks if a directory should be ignored
func (s *Scanner) shouldIgnoreDir(name string) bool {
	for _, rule := range s.ignoreRules {
		if name == rule || strings.HasPrefix(name, rule) {
			return true
		}
	}
	return false
}

// isTerraformFile checks if a file is a Terraform configuration file
func (s *Scanner) isTerraformFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".tf" || ext == ".tf.json"
}

// GetBackendsByType returns all backends of a specific type
func (s *Scanner) GetBackendsByType(backendType string) []BackendConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []BackendConfig
	for _, backend := range s.backends {
		if backend.Type == backendType {
			filtered = append(filtered, backend)
		}
	}
	return filtered
}

// GetUniqueBackends returns unique backend configurations
func (s *Scanner) GetUniqueBackends() []BackendConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var unique []BackendConfig

	for _, backend := range s.backends {
		key := s.backendKey(backend)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, backend)
		}
	}

	return unique
}

// backendKey generates a unique key for a backend configuration
func (s *Scanner) backendKey(config BackendConfig) string {
	key := config.Type

	// Add key attributes based on backend type
	switch config.Type {
	case "s3":
		if bucket, ok := config.Attributes["bucket"].(string); ok {
			key += ":" + bucket
		}
		if k, ok := config.Attributes["key"].(string); ok {
			key += ":" + k
		}
	case "azurerm":
		if account, ok := config.Attributes["storage_account_name"].(string); ok {
			key += ":" + account
		}
		if container, ok := config.Attributes["container_name"].(string); ok {
			key += ":" + container
		}
	case "gcs":
		if bucket, ok := config.Attributes["bucket"].(string); ok {
			key += ":" + bucket
		}
		if prefix, ok := config.Attributes["prefix"].(string); ok {
			key += ":" + prefix
		}
	}

	if config.Workspace != "" {
		key += ":" + config.Workspace
	}

	return key
}
