package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
)

// StateFileInfo represents information about a discovered state file
type StateFileInfo struct {
	Path             string                 `json:"path"`
	FullPath         string                 `json:"full_path"`
	Backend          string                 `json:"backend"`
	BackendConfig    map[string]interface{} `json:"backend_config,omitempty"`
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
	Resources        []StateResourceInfo    `json:"resources"`
	ResourceCount    int                    `json:"resource_count"`
	Providers        map[string]string      `json:"providers"`
	LastModified     time.Time              `json:"last_modified"`
	Size             int64                  `json:"size"`
	IsRemote         bool                   `json:"is_remote"`
	WorkspaceName    string                 `json:"workspace_name,omitempty"`
	Environment      string                 `json:"environment,omitempty"`
	Metadata         map[string]string      `json:"metadata"`
}

// StateResourceInfo represents a resource in a state file
type StateResourceInfo struct {
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Provider  string            `json:"provider"`
	Module    string            `json:"module,omitempty"`
	Mode      string            `json:"mode"` // managed, data
	Instances int               `json:"instances"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// StateDiscovery handles automatic discovery of Terraform state files
type StateDiscovery struct {
	rootPaths       []string
	excludePaths    []string
	maxDepth        int
	followSymlinks  bool
	discoveredFiles map[string]*StateFileInfo
	backends        map[string]BackendConfig
}

// BackendConfig represents a Terraform backend configuration
type BackendConfig struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// NewStateDiscovery creates a new state discovery instance
func NewStateDiscovery(rootPaths []string) *StateDiscovery {
	if len(rootPaths) == 0 {
		rootPaths = []string{"."}
	}

	return &StateDiscovery{
		rootPaths:       rootPaths,
		excludePaths:    defaultExcludePaths(),
		maxDepth:        10,
		followSymlinks:  false,
		discoveredFiles: make(map[string]*StateFileInfo),
		backends:        make(map[string]BackendConfig),
	}
}

// defaultExcludePaths returns default paths to exclude from scanning
func defaultExcludePaths() []string {
	return []string{
		".git",
		".terraform",
		"node_modules",
		"vendor",
		".venv",
		"venv",
		"__pycache__",
		".cache",
		".tmp",
		"temp",
		"tmp",
	}
}

// DiscoverStateFiles discovers all Terraform state files
func (sd *StateDiscovery) DiscoverStateFiles(ctx context.Context) ([]*StateFileInfo, error) {
	fmt.Println("🔍 Auto-discovering Terraform state files...")
	
	var allFiles []*StateFileInfo
	
	for _, rootPath := range sd.rootPaths {
		fmt.Printf("  Scanning: %s\n", rootPath)
		
		// Discover local state files
		localFiles, err := sd.discoverLocalStateFiles(ctx, rootPath)
		if err != nil {
			fmt.Printf("  ⚠️  Warning: Error scanning %s: %v\n", rootPath, err)
			continue
		}
		
		// Discover backend configurations
		backends, err := sd.discoverBackendConfigs(ctx, rootPath)
		if err != nil {
			fmt.Printf("  ⚠️  Warning: Error discovering backends in %s: %v\n", rootPath, err)
		} else {
			for path, backend := range backends {
				sd.backends[path] = backend
			}
		}
		
		allFiles = append(allFiles, localFiles...)
	}
	
	// Discover remote state files from backends
	remoteFiles, err := sd.discoverRemoteStateFiles(ctx)
	if err != nil {
		fmt.Printf("  ⚠️  Warning: Error discovering remote state files: %v\n", err)
	} else {
		allFiles = append(allFiles, remoteFiles...)
	}
	
	// Store discovered files
	for _, file := range allFiles {
		sd.discoveredFiles[file.Path] = file
	}
	
	fmt.Printf("✅ Discovered %d state files\n", len(allFiles))
	return allFiles, nil
}

// discoverLocalStateFiles discovers local .tfstate files
func (sd *StateDiscovery) discoverLocalStateFiles(ctx context.Context, rootPath string) ([]*StateFileInfo, error) {
	var stateFiles []*StateFileInfo
	
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Skip excluded directories
		if d.IsDir() {
			for _, exclude := range sd.excludePaths {
				if strings.Contains(path, exclude) {
					return filepath.SkipDir
				}
			}
			
			// Check max depth
			depth := strings.Count(filepath.ToSlash(path), "/") - strings.Count(filepath.ToSlash(rootPath), "/")
			if depth > sd.maxDepth {
				return filepath.SkipDir
			}
			
			return nil
		}
		
		// Check if it's a state file
		if isStateFile(path) {
			info, err := sd.analyzeStateFile(path)
			if err != nil {
				fmt.Printf("    ⚠️  Warning: Failed to analyze %s: %v\n", path, err)
				return nil
			}
			
			stateFiles = append(stateFiles, info)
			fmt.Printf("    ✓ Found: %s (%d resources)\n", info.Path, info.ResourceCount)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	
	return stateFiles, nil
}

// isStateFile checks if a file is a Terraform state file
func isStateFile(path string) bool {
	name := filepath.Base(path)
	
	// Check for standard state file patterns
	patterns := []string{
		"terraform.tfstate",
		"terraform.tfstate.backup",
		".terraform.tfstate",
	}
	
	for _, pattern := range patterns {
		if name == pattern {
			return true
		}
	}
	
	// Check for custom state files with .tfstate extension
	if strings.HasSuffix(name, ".tfstate") || strings.HasSuffix(name, ".tfstate.backup") {
		return true
	}
	
	return false
}

// analyzeStateFile analyzes a Terraform state file
func (sd *StateDiscovery) analyzeStateFile(path string) (*StateFileInfo, error) {
	// Get file info
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	
	// Read state file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	// Parse state file
	var state tfjson.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	
	// Get relative path
	relPath, err := filepath.Rel(".", path)
	if err != nil {
		relPath = path
	}
	
	// Determine environment from path
	environment := sd.detectEnvironment(path)
	
	// Determine workspace name
	workspaceName := sd.detectWorkspace(path)
	
	// Build state file info
	info := &StateFileInfo{
		Path:             relPath,
		FullPath:         absPath,
		Backend:          "local",
		Version:          4, // Default to version 4
		TerraformVersion: state.TerraformVersion,
		Serial:           0, // Serial is not directly available in tfjson.State
		Lineage:          "", // Lineage is not directly available in tfjson.State
		Resources:        []StateResourceInfo{},
		Providers:        make(map[string]string),
		LastModified:     fileInfo.ModTime(),
		Size:             fileInfo.Size(),
		IsRemote:         false,
		WorkspaceName:    workspaceName,
		Environment:      environment,
		Metadata:         make(map[string]string),
	}
	
	// Extract resource information
	if state.Values != nil && state.Values.RootModule != nil {
		info.Resources = sd.extractResources(state.Values.RootModule)
		info.ResourceCount = len(info.Resources)
		
		// Extract provider information
		for _, res := range state.Values.RootModule.Resources {
			if res.ProviderName != "" {
				parts := strings.Split(res.ProviderName, "/")
				provider := parts[len(parts)-1]
				info.Providers[provider] = res.ProviderName
			}
		}
	}
	
	return info, nil
}

// extractResources extracts resource information from a module
func (sd *StateDiscovery) extractResources(module *tfjson.StateModule) []StateResourceInfo {
	var resources []StateResourceInfo
	
	// Process resources in current module
	for _, res := range module.Resources {
		resInfo := StateResourceInfo{
			Type:      res.Type,
			Name:      res.Name,
			Provider:  res.ProviderName,
			Mode:      string(res.Mode),
			Instances: 1, // Each resource counts as one instance
			Metadata:  make(map[string]string),
		}
		
		// Add module path if not root
		if module.Address != "" && module.Address != "root" {
			resInfo.Module = module.Address
		}
		
		resources = append(resources, resInfo)
	}
	
	// Process child modules recursively
	for _, childModule := range module.ChildModules {
		childResources := sd.extractResources(childModule)
		resources = append(resources, childResources...)
	}
	
	return resources
}

// discoverBackendConfigs discovers Terraform backend configurations
func (sd *StateDiscovery) discoverBackendConfigs(ctx context.Context, rootPath string) (map[string]BackendConfig, error) {
	backends := make(map[string]BackendConfig)
	
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Skip excluded directories
		if d.IsDir() {
			for _, exclude := range sd.excludePaths {
				if strings.Contains(path, exclude) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		
		// Check for Terraform configuration files
		if strings.HasSuffix(path, ".tf") || strings.HasSuffix(path, ".tf.json") {
			backend, err := sd.extractBackendConfig(path)
			if err == nil && backend.Type != "" {
				dir := filepath.Dir(path)
				backends[dir] = backend
				fmt.Printf("    ✓ Found backend config in: %s (type: %s)\n", dir, backend.Type)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to discover backends: %w", err)
	}
	
	return backends, nil
}

// extractBackendConfig extracts backend configuration from a Terraform file
func (sd *StateDiscovery) extractBackendConfig(path string) (BackendConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BackendConfig{}, err
	}
	
	content := string(data)
	
	// Simple pattern matching for backend configuration
	// In production, use HCL parser for accurate extraction
	if strings.Contains(content, "backend ") {
		// Extract backend type
		backendType := ""
		if idx := strings.Index(content, `backend "`); idx != -1 {
			start := idx + 9
			end := strings.Index(content[start:], `"`)
			if end != -1 {
				backendType = content[start : start+end]
			}
		}
		
		if backendType != "" {
			return BackendConfig{
				Type:   backendType,
				Config: make(map[string]interface{}),
			}, nil
		}
	}
	
	return BackendConfig{}, fmt.Errorf("no backend configuration found")
}

// discoverRemoteStateFiles discovers remote state files from configured backends
func (sd *StateDiscovery) discoverRemoteStateFiles(ctx context.Context) ([]*StateFileInfo, error) {
	var remoteFiles []*StateFileInfo
	
	for path, backend := range sd.backends {
		fmt.Printf("  Checking remote backend in %s (type: %s)\n", path, backend.Type)
		
		switch backend.Type {
		case "s3":
			files, err := sd.discoverS3StateFiles(ctx, path, backend)
			if err != nil {
				fmt.Printf("    ⚠️  Warning: Failed to discover S3 state: %v\n", err)
			} else {
				remoteFiles = append(remoteFiles, files...)
			}
			
		case "azurerm":
			files, err := sd.discoverAzureStateFiles(ctx, path, backend)
			if err != nil {
				fmt.Printf("    ⚠️  Warning: Failed to discover Azure state: %v\n", err)
			} else {
				remoteFiles = append(remoteFiles, files...)
			}
			
		case "gcs":
			files, err := sd.discoverGCSStateFiles(ctx, path, backend)
			if err != nil {
				fmt.Printf("    ⚠️  Warning: Failed to discover GCS state: %v\n", err)
			} else {
				remoteFiles = append(remoteFiles, files...)
			}
			
		case "remote":
			files, err := sd.discoverTerraformCloudStateFiles(ctx, path, backend)
			if err != nil {
				fmt.Printf("    ⚠️  Warning: Failed to discover Terraform Cloud state: %v\n", err)
			} else {
				remoteFiles = append(remoteFiles, files...)
			}
			
		default:
			fmt.Printf("    ℹ️  Backend type '%s' discovery not implemented\n", backend.Type)
		}
	}
	
	return remoteFiles, nil
}

// discoverS3StateFiles discovers state files in S3 backend
func (sd *StateDiscovery) discoverS3StateFiles(ctx context.Context, path string, backend BackendConfig) ([]*StateFileInfo, error) {
	// Check for .terraform directory with state information
	terraformDir := filepath.Join(path, ".terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		return nil, nil
	}
	
	// Read terraform.tfstate in .terraform directory
	tfstatePath := filepath.Join(terraformDir, "terraform.tfstate")
	if _, err := os.Stat(tfstatePath); err == nil {
		data, err := os.ReadFile(tfstatePath)
		if err == nil {
			var state map[string]interface{}
			if err := json.Unmarshal(data, &state); err == nil {
				if backend, ok := state["backend"].(map[string]interface{}); ok {
					if config, ok := backend["config"].(map[string]interface{}); ok {
						bucket := config["bucket"]
						key := config["key"]
						region := config["region"]
						
						info := &StateFileInfo{
							Path:          fmt.Sprintf("s3://%v/%v", bucket, key),
							FullPath:      fmt.Sprintf("s3://%v/%v", bucket, key),
							Backend:       "s3",
							BackendConfig: config,
							IsRemote:      true,
							Environment:   sd.detectEnvironment(path),
							WorkspaceName: sd.detectWorkspace(path),
							Metadata: map[string]string{
								"bucket": fmt.Sprintf("%v", bucket),
								"key":    fmt.Sprintf("%v", key),
								"region": fmt.Sprintf("%v", region),
							},
						}
						
						fmt.Printf("    ✓ Found S3 state: %s\n", info.Path)
						return []*StateFileInfo{info}, nil
					}
				}
			}
		}
	}
	
	return nil, nil
}

// discoverAzureStateFiles discovers state files in Azure backend
func (sd *StateDiscovery) discoverAzureStateFiles(ctx context.Context, path string, backend BackendConfig) ([]*StateFileInfo, error) {
	// Similar implementation for Azure
	terraformDir := filepath.Join(path, ".terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		return nil, nil
	}
	
	tfstatePath := filepath.Join(terraformDir, "terraform.tfstate")
	if _, err := os.Stat(tfstatePath); err == nil {
		data, err := os.ReadFile(tfstatePath)
		if err == nil {
			var state map[string]interface{}
			if err := json.Unmarshal(data, &state); err == nil {
				if backend, ok := state["backend"].(map[string]interface{}); ok {
					if config, ok := backend["config"].(map[string]interface{}); ok {
						storageAccount := config["storage_account_name"]
						container := config["container_name"]
						key := config["key"]
						
						info := &StateFileInfo{
							Path:          fmt.Sprintf("azurerm://%v/%v/%v", storageAccount, container, key),
							FullPath:      fmt.Sprintf("azurerm://%v/%v/%v", storageAccount, container, key),
							Backend:       "azurerm",
							BackendConfig: config,
							IsRemote:      true,
							Environment:   sd.detectEnvironment(path),
							WorkspaceName: sd.detectWorkspace(path),
							Metadata: map[string]string{
								"storage_account": fmt.Sprintf("%v", storageAccount),
								"container":       fmt.Sprintf("%v", container),
								"key":             fmt.Sprintf("%v", key),
							},
						}
						
						fmt.Printf("    ✓ Found Azure state: %s\n", info.Path)
						return []*StateFileInfo{info}, nil
					}
				}
			}
		}
	}
	
	return nil, nil
}

// discoverGCSStateFiles discovers state files in Google Cloud Storage backend
func (sd *StateDiscovery) discoverGCSStateFiles(ctx context.Context, path string, backend BackendConfig) ([]*StateFileInfo, error) {
	terraformDir := filepath.Join(path, ".terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		return nil, nil
	}
	
	tfstatePath := filepath.Join(terraformDir, "terraform.tfstate")
	if _, err := os.Stat(tfstatePath); err == nil {
		data, err := os.ReadFile(tfstatePath)
		if err == nil {
			var state map[string]interface{}
			if err := json.Unmarshal(data, &state); err == nil {
				if backend, ok := state["backend"].(map[string]interface{}); ok {
					if config, ok := backend["config"].(map[string]interface{}); ok {
						bucket := config["bucket"]
						prefix := config["prefix"]
						
						info := &StateFileInfo{
							Path:          fmt.Sprintf("gs://%v/%v", bucket, prefix),
							FullPath:      fmt.Sprintf("gs://%v/%v", bucket, prefix),
							Backend:       "gcs",
							BackendConfig: config,
							IsRemote:      true,
							Environment:   sd.detectEnvironment(path),
							WorkspaceName: sd.detectWorkspace(path),
							Metadata: map[string]string{
								"bucket": fmt.Sprintf("%v", bucket),
								"prefix": fmt.Sprintf("%v", prefix),
							},
						}
						
						fmt.Printf("    ✓ Found GCS state: %s\n", info.Path)
						return []*StateFileInfo{info}, nil
					}
				}
			}
		}
	}
	
	return nil, nil
}

// discoverTerraformCloudStateFiles discovers state files in Terraform Cloud
func (sd *StateDiscovery) discoverTerraformCloudStateFiles(ctx context.Context, path string, backend BackendConfig) ([]*StateFileInfo, error) {
	terraformDir := filepath.Join(path, ".terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		return nil, nil
	}
	
	tfstatePath := filepath.Join(terraformDir, "terraform.tfstate")
	if _, err := os.Stat(tfstatePath); err == nil {
		data, err := os.ReadFile(tfstatePath)
		if err == nil {
			var state map[string]interface{}
			if err := json.Unmarshal(data, &state); err == nil {
				if backend, ok := state["backend"].(map[string]interface{}); ok {
					if config, ok := backend["config"].(map[string]interface{}); ok {
						organization := config["organization"]
						workspace := config["workspaces"]
						
						info := &StateFileInfo{
							Path:          fmt.Sprintf("remote://%v/%v", organization, workspace),
							FullPath:      fmt.Sprintf("remote://%v/%v", organization, workspace),
							Backend:       "remote",
							BackendConfig: config,
							IsRemote:      true,
							Environment:   sd.detectEnvironment(path),
							WorkspaceName: fmt.Sprintf("%v", workspace),
							Metadata: map[string]string{
								"organization": fmt.Sprintf("%v", organization),
								"workspace":    fmt.Sprintf("%v", workspace),
							},
						}
						
						fmt.Printf("    ✓ Found Terraform Cloud state: %s\n", info.Path)
						return []*StateFileInfo{info}, nil
					}
				}
			}
		}
	}
	
	return nil, nil
}

// detectEnvironment attempts to detect the environment from the path
func (sd *StateDiscovery) detectEnvironment(path string) string {
	pathLower := strings.ToLower(path)
	
	environments := []string{
		"production", "prod",
		"staging", "stage",
		"development", "dev",
		"testing", "test",
		"qa",
		"uat",
		"sandbox",
	}
	
	for _, env := range environments {
		if strings.Contains(pathLower, env) {
			// Normalize environment names
			switch env {
			case "prod":
				return "production"
			case "stage":
				return "staging"
			case "dev":
				return "development"
			case "test":
				return "testing"
			default:
				return env
			}
		}
	}
	
	return "unknown"
}

// detectWorkspace attempts to detect the workspace name from the path
func (sd *StateDiscovery) detectWorkspace(path string) string {
	// Check if path contains terraform.tfstate.d (workspace directory)
	if strings.Contains(path, "terraform.tfstate.d") {
		parts := strings.Split(filepath.ToSlash(path), "/")
		for i, part := range parts {
			if part == "terraform.tfstate.d" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	
	// Check parent directory name
	dir := filepath.Dir(path)
	base := filepath.Base(dir)
	if base != "." && base != "/" && !strings.HasPrefix(base, ".") {
		return base
	}
	
	return "default"
}

// GetStateFileInfo returns information about a specific state file
func (sd *StateDiscovery) GetStateFileInfo(path string) (*StateFileInfo, error) {
	if info, ok := sd.discoveredFiles[path]; ok {
		return info, nil
	}
	
	// Try to analyze the file directly
	return sd.analyzeStateFile(path)
}

// RefreshStateFiles refreshes the discovery of state files
func (sd *StateDiscovery) RefreshStateFiles(ctx context.Context) error {
	// Clear existing discoveries
	sd.discoveredFiles = make(map[string]*StateFileInfo)
	sd.backends = make(map[string]BackendConfig)
	
	// Re-discover
	_, err := sd.DiscoverStateFiles(ctx)
	return err
}

// SetExcludePaths sets paths to exclude from discovery
func (sd *StateDiscovery) SetExcludePaths(paths []string) {
	sd.excludePaths = paths
}

// SetMaxDepth sets the maximum directory depth for discovery
func (sd *StateDiscovery) SetMaxDepth(depth int) {
	sd.maxDepth = depth
}

// SetFollowSymlinks sets whether to follow symbolic links
func (sd *StateDiscovery) SetFollowSymlinks(follow bool) {
	sd.followSymlinks = follow
}