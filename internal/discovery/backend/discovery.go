package backend

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// BackendConfig represents a discovered Terraform backend configuration
type BackendConfig struct {
	Type         string                 `json:"type"`
	Config       map[string]interface{} `json:"config"`
	FilePath     string                 `json:"file_path"`
	WorkingDir   string                 `json:"working_dir"`
	IsRemote     bool                   `json:"is_remote"`
	StateFile    string                 `json:"state_file,omitempty"`
	LockTable    string                 `json:"lock_table,omitempty"`
	Workspace    string                 `json:"workspace,omitempty"`
	Discovered   string                 `json:"discovered"`
}

// S3BackendConfig represents S3 backend specific configuration
type S3BackendConfig struct {
	Bucket         string `json:"bucket"`
	Key            string `json:"key"`
	Region         string `json:"region"`
	DynamoDBTable  string `json:"dynamodb_table,omitempty"`
	Encrypt        bool   `json:"encrypt"`
	Profile        string `json:"profile,omitempty"`
	RoleARN        string `json:"role_arn,omitempty"`
	AccessKey      string `json:"access_key,omitempty"`
	SecretKey      string `json:"secret_key,omitempty"`
	SessionToken   string `json:"session_token,omitempty"`
}

// AzureBackendConfig represents Azure Storage backend configuration
type AzureBackendConfig struct {
	StorageAccountName string `json:"storage_account_name"`
	ContainerName      string `json:"container_name"`
	Key                string `json:"key"`
	AccessKey          string `json:"access_key,omitempty"`
	SASToken           string `json:"sas_token,omitempty"`
	ClientID           string `json:"client_id,omitempty"`
	ClientSecret       string `json:"client_secret,omitempty"`
	TenantID           string `json:"tenant_id,omitempty"`
	SubscriptionID     string `json:"subscription_id,omitempty"`
	UseMSI             bool   `json:"use_msi"`
}

// GCSBackendConfig represents Google Cloud Storage backend configuration
type GCSBackendConfig struct {
	Bucket      string `json:"bucket"`
	Prefix      string `json:"prefix"`
	Credentials string `json:"credentials,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	Project     string `json:"project,omitempty"`
}

// TerraformCloudConfig represents Terraform Cloud/Enterprise backend
type TerraformCloudConfig struct {
	Organization string `json:"organization"`
	Workspaces   struct {
		Name   string `json:"name,omitempty"`
		Prefix string `json:"prefix,omitempty"`
	} `json:"workspaces"`
	Hostname string `json:"hostname,omitempty"`
	Token    string `json:"token,omitempty"`
}

// DiscoveryService handles backend discovery operations
type DiscoveryService struct {
	mu              sync.RWMutex
	discoveredCache map[string]*BackendConfig
	parser          *hclparse.Parser
	rootPaths       []string
	excludePaths    []string
	maxDepth        int
}

// NewDiscoveryService creates a new backend discovery service
func NewDiscoveryService(rootPaths []string, excludePaths []string) *DiscoveryService {
	return &DiscoveryService{
		discoveredCache: make(map[string]*BackendConfig),
		parser:          hclparse.NewParser(),
		rootPaths:       rootPaths,
		excludePaths:    excludePaths,
		maxDepth:        10, // Maximum directory depth to search
	}
}

// DiscoverBackends scans the filesystem for Terraform backend configurations
func (d *DiscoveryService) DiscoverBackends(ctx context.Context) ([]*BackendConfig, error) {
	var configs []*BackendConfig
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, rootPath := range d.rootPaths {
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			
			err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return nil // Skip inaccessible paths
				}

				// Check context cancellation
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// Skip excluded paths
				if d.shouldExclude(path) {
					if entry.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Check for Terraform files
				if !entry.IsDir() && d.isTerraformFile(path) {
					config, err := d.parseBackendFromFile(path)
					if err == nil && config != nil {
						mu.Lock()
						configs = append(configs, config)
						d.discoveredCache[path] = config
						mu.Unlock()
					}
				}

				// Also check for .terraform directories with cached state
				if entry.IsDir() && entry.Name() == ".terraform" {
					stateConfigs := d.discoverCachedStates(path)
					if len(stateConfigs) > 0 {
						mu.Lock()
						configs = append(configs, stateConfigs...)
						mu.Unlock()
					}
				}

				return nil
			})
			
			if err != nil && err != ctx.Err() {
				fmt.Printf("Warning: Error walking directory %s: %v\n", root, err)
			}
		}(rootPath)
	}

	wg.Wait()
	
	// Also discover from environment variables
	envConfigs := d.discoverFromEnvironment()
	configs = append(configs, envConfigs...)

	return configs, nil
}

// parseBackendFromFile extracts backend configuration from a Terraform file
func (d *DiscoveryService) parseBackendFromFile(filePath string) (*BackendConfig, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse HCL file
	file, diags := d.parser.ParseHCL(content, filePath)
	if diags.HasErrors() {
		// Try to extract backend config using regex as fallback
		return d.parseBackendWithRegex(string(content), filePath)
	}

	// Look for backend blocks in the parsed HCL
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unable to parse HCL body")
	}

	for _, block := range body.Blocks {
		if block.Type == "terraform" {
			for _, innerBlock := range block.Body.Blocks {
				if innerBlock.Type == "backend" && len(innerBlock.Labels) > 0 {
					backendType := innerBlock.Labels[0]
					config := d.extractBackendConfig(innerBlock, backendType)
					config.FilePath = filePath
					config.WorkingDir = filepath.Dir(filePath)
					return config, nil
				}
			}
		}
	}

	return nil, nil
}

// parseBackendWithRegex uses regex as a fallback for parsing backend configs
func (d *DiscoveryService) parseBackendWithRegex(content, filePath string) (*BackendConfig, error) {
	// Regex patterns for different backend types
	patterns := map[string]*regexp.Regexp{
		"s3": regexp.MustCompile(`backend\s+"s3"\s*{([^}]+)}`),
		"azurerm": regexp.MustCompile(`backend\s+"azurerm"\s*{([^}]+)}`),
		"gcs": regexp.MustCompile(`backend\s+"gcs"\s*{([^}]+)}`),
		"remote": regexp.MustCompile(`backend\s+"remote"\s*{([^}]+)}`),
	}

	for backendType, pattern := range patterns {
		matches := pattern.FindStringSubmatch(content)
		if len(matches) > 1 {
			config := &BackendConfig{
				Type:       backendType,
				Config:     d.parseBackendProperties(matches[1]),
				FilePath:   filePath,
				WorkingDir: filepath.Dir(filePath),
				IsRemote:   true,
			}

			// Extract specific properties based on backend type
			switch backendType {
			case "s3":
				config.StateFile = d.extractProperty(matches[1], "key")
				config.LockTable = d.extractProperty(matches[1], "dynamodb_table")
			case "azurerm":
				config.StateFile = d.extractProperty(matches[1], "key")
			case "gcs":
				config.StateFile = d.extractProperty(matches[1], "prefix")
			}

			return config, nil
		}
	}

	return nil, nil
}

// extractBackendConfig extracts configuration from an HCL block
func (d *DiscoveryService) extractBackendConfig(block *hclsyntax.Block, backendType string) *BackendConfig {
	config := &BackendConfig{
		Type:     backendType,
		Config:   make(map[string]interface{}),
		IsRemote: true,
	}

	// Extract attributes from the block
	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(&hcl.EvalContext{})
		if !diags.HasErrors() {
			config.Config[name] = val.AsString()
		}
	}

	// Set specific fields based on backend type
	switch backendType {
	case "s3":
		if key, ok := config.Config["key"].(string); ok {
			config.StateFile = key
		}
		if table, ok := config.Config["dynamodb_table"].(string); ok {
			config.LockTable = table
		}
	case "azurerm":
		if key, ok := config.Config["key"].(string); ok {
			config.StateFile = key
		}
	case "gcs":
		if prefix, ok := config.Config["prefix"].(string); ok {
			config.StateFile = prefix
		}
	}

	return config
}

// discoverCachedStates looks for cached state files in .terraform directories
func (d *DiscoveryService) discoverCachedStates(terraformDir string) []*BackendConfig {
	var configs []*BackendConfig

	// Check for terraform.tfstate in the .terraform directory
	statePath := filepath.Join(terraformDir, "terraform.tfstate")
	if _, err := os.Stat(statePath); err == nil {
		// Read the state file to find backend configuration
		content, err := os.ReadFile(statePath)
		if err == nil {
			config := d.extractBackendFromState(string(content))
			if config != nil {
				config.FilePath = statePath
				config.WorkingDir = filepath.Dir(terraformDir)
				configs = append(configs, config)
			}
		}
	}

	// Check for backend configuration file
	backendConfigPath := filepath.Join(terraformDir, "terraform.tfstate.d")
	if info, err := os.Stat(backendConfigPath); err == nil && info.IsDir() {
		// Look for workspace state files
		entries, _ := os.ReadDir(backendConfigPath)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			workspaceStatePath := filepath.Join(backendConfigPath, entry.Name(), "terraform.tfstate")
			if _, err := os.Stat(workspaceStatePath); err == nil {
				content, err := os.ReadFile(workspaceStatePath)
				if err == nil {
					config := d.extractBackendFromState(string(content))
					if config != nil {
						config.FilePath = workspaceStatePath
						config.WorkingDir = filepath.Dir(terraformDir)
						config.Workspace = entry.Name()
						configs = append(configs, config)
					}
				}
			}
		}
	}

	return configs
}

// extractBackendFromState extracts backend config from a state file
func (d *DiscoveryService) extractBackendFromState(stateContent string) *BackendConfig {
	// Use regex to find backend configuration in state file
	backendPattern := regexp.MustCompile(`"backend":\s*{([^}]+)}`)
	matches := backendPattern.FindStringSubmatch(stateContent)
	
	if len(matches) > 1 {
		// Parse the backend configuration
		typePattern := regexp.MustCompile(`"type":\s*"([^"]+)"`)
		typeMatches := typePattern.FindStringSubmatch(matches[1])
		
		if len(typeMatches) > 1 {
			config := &BackendConfig{
				Type:     typeMatches[1],
				Config:   make(map[string]interface{}),
				IsRemote: true,
			}
			
			// Extract config values
			configPattern := regexp.MustCompile(`"config":\s*{([^}]+)}`)
			configMatches := configPattern.FindStringSubmatch(matches[1])
			if len(configMatches) > 1 {
				config.Config = d.parseBackendProperties(configMatches[1])
			}
			
			return config
		}
	}
	
	return nil
}

// discoverFromEnvironment checks environment variables for backend configuration
func (d *DiscoveryService) discoverFromEnvironment() []*BackendConfig {
	var configs []*BackendConfig

	// Check for S3 backend environment variables
	if bucket := os.Getenv("TF_BACKEND_S3_BUCKET"); bucket != "" {
		config := &BackendConfig{
			Type: "s3",
			Config: map[string]interface{}{
				"bucket": bucket,
				"key":    os.Getenv("TF_BACKEND_S3_KEY"),
				"region": os.Getenv("TF_BACKEND_S3_REGION"),
			},
			IsRemote:   true,
			Discovered: "environment",
		}
		
		if table := os.Getenv("TF_BACKEND_S3_DYNAMODB_TABLE"); table != "" {
			config.Config["dynamodb_table"] = table
			config.LockTable = table
		}
		
		configs = append(configs, config)
	}

	// Check for Azure backend environment variables
	if account := os.Getenv("TF_BACKEND_AZURERM_STORAGE_ACCOUNT"); account != "" {
		config := &BackendConfig{
			Type: "azurerm",
			Config: map[string]interface{}{
				"storage_account_name": account,
				"container_name":       os.Getenv("TF_BACKEND_AZURERM_CONTAINER"),
				"key":                  os.Getenv("TF_BACKEND_AZURERM_KEY"),
			},
			IsRemote:   true,
			Discovered: "environment",
		}
		configs = append(configs, config)
	}

	// Check for GCS backend environment variables
	if bucket := os.Getenv("TF_BACKEND_GCS_BUCKET"); bucket != "" {
		config := &BackendConfig{
			Type: "gcs",
			Config: map[string]interface{}{
				"bucket": bucket,
				"prefix": os.Getenv("TF_BACKEND_GCS_PREFIX"),
			},
			IsRemote:   true,
			Discovered: "environment",
		}
		configs = append(configs, config)
	}

	return configs
}

// Helper methods

func (d *DiscoveryService) isTerraformFile(path string) bool {
	ext := filepath.Ext(path)
	name := filepath.Base(path)
	return ext == ".tf" || ext == ".hcl" || 
		name == "terraform.tfvars" || 
		strings.HasSuffix(name, ".tfvars") ||
		name == "terragrunt.hcl"
}

func (d *DiscoveryService) shouldExclude(path string) bool {
	for _, exclude := range d.excludePaths {
		if strings.Contains(path, exclude) {
			return true
		}
	}
	// Common excludes
	excludePatterns := []string{
		".git", "node_modules", ".terraform/providers", 
		".terraform/modules", "vendor", ".venv",
	}
	for _, pattern := range excludePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func (d *DiscoveryService) parseBackendProperties(content string) map[string]interface{} {
	props := make(map[string]interface{})
	
	// Simple regex to extract key-value pairs
	pattern := regexp.MustCompile(`"?(\w+)"?\s*[=:]\s*"([^"]+)"`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			props[match[1]] = match[2]
		}
	}
	
	return props
}

func (d *DiscoveryService) extractProperty(content, property string) string {
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*"([^"]+)"`, property))
	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// GetDiscoveredBackends returns all discovered backend configurations
func (d *DiscoveryService) GetDiscoveredBackends() []*BackendConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	configs := make([]*BackendConfig, 0, len(d.discoveredCache))
	for _, config := range d.discoveredCache {
		configs = append(configs, config)
	}
	
	return configs
}

// ClearCache clears the discovered backends cache
func (d *DiscoveryService) ClearCache() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.discoveredCache = make(map[string]*BackendConfig)
}