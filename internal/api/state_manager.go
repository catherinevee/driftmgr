package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// StateFileInfo represents information about a state file
type StateFileInfo struct {
	Path         string                 `json:"path"`
	FullPath     string                 `json:"full_path"`
	Provider     string                 `json:"provider"`
	Backend      string                 `json:"backend"`
	Size         int64                  `json:"size"`
	LastModified time.Time              `json:"last_modified"`
	Version      int                    `json:"version"`
	Serial       int                    `json:"serial"`
	Resources    []StateResourceSummary `json:"resources,omitempty"`
	ResourceCount int                   `json:"resource_count"`
	Outputs      map[string]interface{} `json:"outputs,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// StateResourceSummary provides a summary of a state resource
type StateResourceSummary struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Count    int    `json:"count"`
}

// StateFileManager manages terraform state files
type StateFileManager struct {
	mu          sync.RWMutex
	basePaths   []string // Base paths to search for state files
	stateFiles  map[string]*StateFileInfo
	stateLoader *state.StateLoader
}

// NewStateFileManager creates a new state file manager
func NewStateFileManager() *StateFileManager {
	return &StateFileManager{
		basePaths: []string{
			".",
			"terraform-states",
			"terraform",
			"infrastructure",
			os.Getenv("HOME") + "/.terraform",
			os.Getenv("HOME") + "/terraform",
		},
		stateFiles:  make(map[string]*StateFileInfo),
		stateLoader: state.NewStateLoader(""),
	}
}

// DiscoverStateFiles discovers terraform state files in the configured paths
func (sfm *StateFileManager) DiscoverStateFiles(ctx context.Context) ([]*StateFileInfo, error) {
	sfm.mu.Lock()
	defer sfm.mu.Unlock()

	var discovered []*StateFileInfo

	for _, basePath := range sfm.basePaths {
		if basePath == "" {
			continue
		}

		// Check if path exists
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}

		// Walk the directory tree looking for state files
		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip directories we can't read
			}

			// Check for terraform state files
			if !info.IsDir() && (strings.HasSuffix(path, ".tfstate") || strings.HasSuffix(path, ".tfstate.backup")) {
				stateInfo := sfm.loadStateFileInfo(path, info)
				if stateInfo != nil {
					discovered = append(discovered, stateInfo)
					sfm.stateFiles[path] = stateInfo
				}
			}

			return nil
		})

		if err != nil {
			// Log error but continue with other paths
			continue
		}
	}

	// Also check for remote state configurations
	remoteStates := sfm.discoverRemoteStates(ctx)
	discovered = append(discovered, remoteStates...)

	return discovered, nil
}

// loadStateFileInfo loads information about a state file
func (sfm *StateFileManager) loadStateFileInfo(path string, info os.FileInfo) *StateFileInfo {
	stateInfo := &StateFileInfo{
		Path:         path,
		FullPath:     path,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		Backend:      "local",
	}

	// Try to read and parse the state file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		stateInfo.Error = fmt.Sprintf("failed to read file: %v", err)
		return stateInfo
	}

	// Parse the state file
	var tfState map[string]interface{}
	if err := json.Unmarshal(data, &tfState); err != nil {
		stateInfo.Error = fmt.Sprintf("invalid JSON: %v", err)
		return stateInfo
	}

	// Extract metadata
	if version, ok := tfState["version"].(float64); ok {
		stateInfo.Version = int(version)
	}
	if serial, ok := tfState["serial"].(float64); ok {
		stateInfo.Serial = int(serial)
	}
	if outputs, ok := tfState["outputs"].(map[string]interface{}); ok {
		stateInfo.Outputs = outputs
	}

	// Process resources
	if resources, ok := tfState["resources"].([]interface{}); ok {
		resourceMap := make(map[string]*StateResourceSummary)
		
		for _, r := range resources {
			if resource, ok := r.(map[string]interface{}); ok {
				resType, _ := resource["type"].(string)
				resName, _ := resource["name"].(string)
				provider, _ := resource["provider"].(string)
				
				// Detect provider from resource type or provider field
				if provider == "" && resType != "" {
					provider = sfm.detectProviderFromResourceType(resType)
				}
				
				if stateInfo.Provider == "" && provider != "" {
					stateInfo.Provider = sfm.normalizeProvider(provider)
				}
				
				// Group resources by type
				key := resType
				if summary, exists := resourceMap[key]; exists {
					summary.Count++
				} else {
					resourceMap[key] = &StateResourceSummary{
						Type:     resType,
						Name:     resName,
						Provider: provider,
						Count:    1,
					}
				}
				
				// Count instances
				if instances, ok := resource["instances"].([]interface{}); ok {
					stateInfo.ResourceCount += len(instances)
				} else {
					stateInfo.ResourceCount++
				}
			}
		}
		
		// Convert map to slice
		for _, summary := range resourceMap {
			stateInfo.Resources = append(stateInfo.Resources, *summary)
		}
	}

	return stateInfo
}

// detectProviderFromResourceType detects provider from resource type
func (sfm *StateFileManager) detectProviderFromResourceType(resourceType string) string {
	if strings.HasPrefix(resourceType, "aws_") {
		return "aws"
	} else if strings.HasPrefix(resourceType, "azurerm_") || strings.HasPrefix(resourceType, "azuread_") {
		return "azure"
	} else if strings.HasPrefix(resourceType, "google_") || strings.HasPrefix(resourceType, "gcp_") {
		return "gcp"
	} else if strings.HasPrefix(resourceType, "digitalocean_") {
		return "digitalocean"
	}
	return ""
}

// normalizeProvider normalizes provider name from terraform format
func (sfm *StateFileManager) normalizeProvider(provider string) string {
	// Remove registry prefix and version
	parts := strings.Split(provider, "/")
	if len(parts) > 0 {
		provider = parts[len(parts)-1]
	}
	
	// Remove brackets and quotes
	provider = strings.Trim(provider, `[]"`)
	
	// Extract just the provider name
	if strings.Contains(provider, "aws") {
		return "aws"
	} else if strings.Contains(provider, "azure") {
		return "azure"
	} else if strings.Contains(provider, "google") || strings.Contains(provider, "gcp") {
		return "gcp"
	} else if strings.Contains(provider, "digitalocean") {
		return "digitalocean"
	}
	
	return provider
}

// discoverRemoteStates discovers remote state configurations
func (sfm *StateFileManager) discoverRemoteStates(ctx context.Context) []*StateFileInfo {
	var remoteStates []*StateFileInfo

	// Check for backend configurations in terraform files
	backendConfigs := sfm.findBackendConfigurations()
	
	for _, config := range backendConfigs {
		stateInfo := &StateFileInfo{
			Path:     config.Path,
			FullPath: config.FullPath,
			Backend:  config.Type,
			Provider: config.Provider,
		}
		
		// Try to get metadata from remote backend
		if config.Type == "s3" {
			// Would connect to S3 and get state metadata
			stateInfo.Backend = "s3"
		} else if config.Type == "azurerm" {
			// Would connect to Azure Storage and get state metadata
			stateInfo.Backend = "azurerm"
		} else if config.Type == "gcs" {
			// Would connect to GCS and get state metadata
			stateInfo.Backend = "gcs"
		}
		
		remoteStates = append(remoteStates, stateInfo)
	}

	return remoteStates
}

// BackendConfig represents a terraform backend configuration
type BackendConfig struct {
	Type     string
	Path     string
	FullPath string
	Provider string
	Config   map[string]interface{}
}

// findBackendConfigurations finds terraform backend configurations
func (sfm *StateFileManager) findBackendConfigurations() []BackendConfig {
	var configs []BackendConfig

	// Look for *.tf files with backend configurations
	for _, basePath := range sfm.basePaths {
		if basePath == "" {
			continue
		}

		filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			if strings.HasSuffix(path, ".tf") {
				// Read and parse terraform file for backend config
				// This is simplified - real implementation would use HCL parser
				data, err := ioutil.ReadFile(path)
				if err == nil && strings.Contains(string(data), "backend") {
					// Extract backend type from the file
					// This is a simplified check
					config := BackendConfig{
						Path:     path,
						FullPath: path,
					}
					
					if strings.Contains(string(data), `backend "s3"`) {
						config.Type = "s3"
						config.Provider = "aws"
					} else if strings.Contains(string(data), `backend "azurerm"`) {
						config.Type = "azurerm"
						config.Provider = "azure"
					} else if strings.Contains(string(data), `backend "gcs"`) {
						config.Type = "gcs"
						config.Provider = "gcp"
					}
					
					if config.Type != "" {
						configs = append(configs, config)
					}
				}
			}
			return nil
		})
	}

	return configs
}

// AddStateFile adds a state file to the manager
func (sfm *StateFileManager) AddStateFile(path string, modTime time.Time) {
	sfm.mu.Lock()
	defer sfm.mu.Unlock()
	
	info := &StateFileInfo{
		Path:         path,
		FullPath:     path,
		LastModified: modTime,
		Backend:      "local",
	}
	
	// Try to load more info
	if fileInfo, err := os.Stat(path); err == nil {
		if fullInfo := sfm.loadStateFileInfo(path, fileInfo); fullInfo != nil {
			info = fullInfo
		}
	}
	
	sfm.stateFiles[path] = info
}

// GetStateFiles returns all discovered state files
func (sfm *StateFileManager) GetStateFiles() map[string]*StateFileInfo {
	sfm.mu.RLock()
	defer sfm.mu.RUnlock()
	
	// Return a copy to avoid concurrent modification issues
	result := make(map[string]*StateFileInfo)
	for k, v := range sfm.stateFiles {
		result[k] = v
	}
	return result
}

// GetStateFile retrieves a specific state file
func (sfm *StateFileManager) GetStateFile(path string) (*StateFileInfo, error) {
	sfm.mu.RLock()
	defer sfm.mu.RUnlock()

	// Try from cache first
	if stateInfo, exists := sfm.stateFiles[path]; exists {
		return stateInfo, nil
	}

	// Try to load it directly
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("state file not found: %s", path)
	}

	stateInfo := sfm.loadStateFileInfo(path, info)
	if stateInfo == nil {
		return nil, fmt.Errorf("failed to load state file: %s", path)
	}

	return stateInfo, nil
}

// LoadStateContent loads the full content of a state file
func (sfm *StateFileManager) LoadStateContent(path string) (map[string]interface{}, error) {
	// Decode base64 if needed
	decodedPath, err := base64.StdEncoding.DecodeString(path)
	if err == nil {
		path = string(decodedPath)
	}

	// Read the state file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		// Try terraform-states directory
		altPath := filepath.Join("terraform-states", path)
		data, err = ioutil.ReadFile(altPath)
		if err != nil {
			return nil, fmt.Errorf("state file not found: %s", path)
		}
	}

	// Parse the state file
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("invalid state file format: %v", err)
	}

	return state, nil
}

// ImportStateResources imports resources from a state file into the discovery cache
func (sfm *StateFileManager) ImportStateResources(ctx context.Context, statePath string) ([]apimodels.Resource, error) {
	stateContent, err := sfm.LoadStateContent(statePath)
	if err != nil {
		return nil, err
	}

	var importedResources []apimodels.Resource

	// Extract resources from state
	if resources, ok := stateContent["resources"].([]interface{}); ok {
		for _, r := range resources {
			if resource, ok := r.(map[string]interface{}); ok {
				resType, _ := resource["type"].(string)
				resName, _ := resource["name"].(string)
				provider, _ := resource["provider"].(string)
				
				// Process instances
				if instances, ok := resource["instances"].([]interface{}); ok {
					for i, inst := range instances {
						if instance, ok := inst.(map[string]interface{}); ok {
							attrs, _ := instance["attributes"].(map[string]interface{})
							
							// Create resource from state
							imported := apimodels.Resource{
								ID:       fmt.Sprintf("%s.%s[%d]", resType, resName, i),
								Name:     resName,
								Type:     resType,
								Provider: sfm.normalizeProvider(provider),
								Status:   "managed",
								Tags:     make(map[string]string),
							}
							
							// Extract common attributes
							if id, ok := attrs["id"].(string); ok {
								imported.ID = id
							}
							if region, ok := attrs["region"].(string); ok {
								imported.Region = region
							} else if location, ok := attrs["location"].(string); ok {
								imported.Region = location
							}
							
							// Extract tags
							if tags, ok := attrs["tags"].(map[string]interface{}); ok {
								for k, v := range tags {
									if str, ok := v.(string); ok {
										imported.Tags[k] = str
									}
								}
							}
							
							// Add source metadata
							imported.Tags["source"] = "terraform_state"
							imported.Tags["state_path"] = statePath
							
							importedResources = append(importedResources, imported)
						}
					}
				}
			}
		}
	}

	return importedResources, nil
}

// RefreshStateFile refreshes the information for a state file
func (sfm *StateFileManager) RefreshStateFile(path string) (*StateFileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	stateInfo := sfm.loadStateFileInfo(path, info)
	if stateInfo == nil {
		return nil, fmt.Errorf("failed to refresh state file: %s", path)
	}

	sfm.mu.Lock()
	sfm.stateFiles[path] = stateInfo
	sfm.mu.Unlock()

	return stateInfo, nil
}