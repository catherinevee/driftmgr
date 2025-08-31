package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TFStateFile represents a discovered Terraform state file
type TFStateFile struct {
	Path             string    `json:"path"`
	Name             string    `json:"name"`
	Size             int64     `json:"size"`
	Modified         time.Time `json:"modified"`
	Provider         string    `json:"provider"`
	ResourceCount    int       `json:"resource_count"`
	TerraformVersion string    `json:"terraform_version"`
	Backend          string    `json:"backend"`
	IsRemote         bool      `json:"is_remote"`
	Error            string    `json:"error,omitempty"`
}

// TFStateDiscovery handles discovery of Terraform state files
type TFStateDiscovery struct {
	searchPaths []string
	maxDepth    int
	cloudProviders map[string]bool
}

// NewTFStateDiscovery creates a new tfstate discovery service
func NewTFStateDiscovery() *TFStateDiscovery {
	homeDir, _ := os.UserHomeDir()
	
	// Check which cloud providers are configured
	cloudProviders := make(map[string]bool)
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
		cloudProviders["aws"] = true
	}
	if os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
		cloudProviders["azure"] = true
	}
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		cloudProviders["gcp"] = true
	}
	
	return &TFStateDiscovery{
		searchPaths: []string{
			".",
			filepath.Join(homeDir, ".terraform"),
			filepath.Join(homeDir, "terraform"),
			filepath.Join(homeDir, "OneDrive", "Desktop", "github", "driftmgr"),
		},
		maxDepth: 2, // Reduced depth for faster scanning
		cloudProviders: cloudProviders,
	}
}

// DiscoverAll finds all terraform state files in common locations
func (d *TFStateDiscovery) DiscoverAll(ctx context.Context) ([]*TFStateFile, error) {
	var stateFiles []*TFStateFile
	discovered := make(map[string]bool)

	for _, searchPath := range d.searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		files, err := d.discoverInPath(ctx, searchPath, 0)
		if err != nil {
			continue // Skip paths with errors
		}

		for _, file := range files {
			if !discovered[file.Path] {
				stateFiles = append(stateFiles, file)
				discovered[file.Path] = true
			}
		}
	}

	// Also check for remote state configurations
	remoteStates := d.discoverRemoteStates(ctx)
	stateFiles = append(stateFiles, remoteStates...)

	// Discover state files from cloud storage if providers are configured
	if len(d.cloudProviders) > 0 {
		cloudStates := d.discoverCloudStates(ctx)
		stateFiles = append(stateFiles, cloudStates...)
	}

	return stateFiles, nil
}

// discoverInPath recursively searches for tfstate files
func (d *TFStateDiscovery) discoverInPath(ctx context.Context, path string, depth int) ([]*TFStateFile, error) {
	if depth > d.maxDepth {
		return nil, nil
	}

	var stateFiles []*TFStateFile

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		if err != nil {
			return nil // Skip files with errors
		}

		// Skip hidden directories and common non-terraform directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") && name != ".terraform" {
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == ".git" || name == "web" {
				return filepath.SkipDir
			}
		}

		// Check if it's a tfstate file
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".tfstate") || 
			strings.HasSuffix(info.Name(), ".tfstate.backup")) {
			
			stateFile := d.analyzeStateFile(filePath, info)
			if stateFile != nil {
				stateFiles = append(stateFiles, stateFile)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stateFiles, nil
}

// analyzeStateFile analyzes a terraform state file
func (d *TFStateDiscovery) analyzeStateFile(path string, info os.FileInfo) *TFStateFile {
	stateFile := &TFStateFile{
		Path:     path,
		Name:     info.Name(),
		Size:     info.Size(),
		Modified: info.ModTime(),
		Backend:  "local",
		IsRemote: false,
	}

	// Try to parse the state file to get more details
	data, err := ioutil.ReadFile(path)
	if err != nil {
		stateFile.Error = fmt.Sprintf("Failed to read: %v", err)
		return stateFile
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		stateFile.Error = fmt.Sprintf("Invalid JSON: %v", err)
		return stateFile
	}

	stateFile.TerraformVersion = state.TerraformVersion
	stateFile.ResourceCount = state.GetResourceCount()
	
	// Detect primary provider
	providers := make(map[string]int)
	for _, resource := range state.Resources {
		provider := extractProviderName(resource.Provider)
		providers[provider]++
	}

	maxCount := 0
	for provider, count := range providers {
		if count > maxCount {
			stateFile.Provider = provider
			maxCount = count
		}
	}

	return stateFile
}

// discoverRemoteStates discovers remote state configurations
func (d *TFStateDiscovery) discoverRemoteStates(ctx context.Context) []*TFStateFile {
	var remoteStates []*TFStateFile

	// Check for terraform backend configurations
	backendConfigs := d.findBackendConfigs()
	
	for _, config := range backendConfigs {
		remoteState := &TFStateFile{
			Path:     config.Path,
			Name:     config.Name,
			Backend:  config.Type,
			IsRemote: true,
			Modified: time.Now(),
		}

		// Try to get details from remote backend
		switch config.Type {
		case "s3":
			remoteState.Provider = "AWS"
		case "azurerm":
			remoteState.Provider = "Azure"
		case "gcs":
			remoteState.Provider = "GCP"
		case "remote":
			remoteState.Provider = "Terraform Cloud"
		}

		remoteStates = append(remoteStates, remoteState)
	}

	return remoteStates
}

// BackendConfig represents a terraform backend configuration
type BackendConfig struct {
	Path string
	Name string
	Type string
	Config map[string]interface{}
}

// findBackendConfigs searches for terraform backend configurations
func (d *TFStateDiscovery) findBackendConfigs() []BackendConfig {
	var configs []BackendConfig

	for _, searchPath := range d.searchPaths {
		filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Look for terraform configuration files
			if strings.HasSuffix(info.Name(), ".tf") || strings.HasSuffix(info.Name(), ".tf.json") {
				// Parse the file to look for backend configurations
				if backend := d.parseBackendConfig(path); backend != nil {
					configs = append(configs, *backend)
				}
			}

			return nil
		})
	}

	return configs
}

// parseBackendConfig parses a terraform file for backend configuration
func (d *TFStateDiscovery) parseBackendConfig(path string) *BackendConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}

	content := string(data)
	
	// Simple pattern matching for backend blocks
	if strings.Contains(content, "backend ") {
		config := &BackendConfig{
			Path: path,
			Name: filepath.Base(path),
		}

		// Extract backend type
		if strings.Contains(content, `backend "s3"`) {
			config.Type = "s3"
		} else if strings.Contains(content, `backend "azurerm"`) {
			config.Type = "azurerm"
		} else if strings.Contains(content, `backend "gcs"`) {
			config.Type = "gcs"
		} else if strings.Contains(content, `backend "remote"`) {
			config.Type = "remote"
		}

		if config.Type != "" {
			return config
		}
	}

	return nil
}

// extractProviderName extracts the provider name from the provider string
func extractProviderName(provider string) string {
	parts := strings.Split(provider, "/")
	if len(parts) > 0 {
		providerPart := parts[len(parts)-1]
		// Remove version info if present
		if idx := strings.Index(providerPart, "."); idx > 0 {
			providerPart = providerPart[:idx]
		}
		return providerPart
	}
	return provider
}

// GetStateDetails loads and returns detailed information about a state file
func (d *TFStateDiscovery) GetStateDetails(path string) (*TerraformState, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// discoverCloudStates discovers terraform state files stored in cloud storage
func (d *TFStateDiscovery) discoverCloudStates(ctx context.Context) []*TFStateFile {
	var cloudStates []*TFStateFile
	
	// AWS S3 buckets commonly used for terraform state
	if d.cloudProviders["aws"] {
		s3States := d.discoverS3States(ctx)
		cloudStates = append(cloudStates, s3States...)
	}
	
	// Azure Storage accounts
	if d.cloudProviders["azure"] {
		azureStates := d.discoverAzureStates(ctx)
		cloudStates = append(cloudStates, azureStates...)
	}
	
	// Google Cloud Storage
	if d.cloudProviders["gcp"] {
		gcsStates := d.discoverGCSStates(ctx)
		cloudStates = append(cloudStates, gcsStates...)
	}
	
	return cloudStates
}

// discoverS3States discovers terraform state files in S3 buckets
func (d *TFStateDiscovery) discoverS3States(ctx context.Context) []*TFStateFile {
	// TODO: Implement actual S3 bucket scanning using AWS SDK
	// This would require:
	// 1. List S3 buckets
	// 2. For each bucket, list objects matching *.tfstate pattern
	// 3. Download and parse state file metadata
	return []*TFStateFile{}
}

// discoverAzureStates discovers terraform state files in Azure Storage
func (d *TFStateDiscovery) discoverAzureStates(ctx context.Context) []*TFStateFile {
	// TODO: Implement actual Azure Storage scanning using Azure SDK
	// This would require:
	// 1. List storage accounts
	// 2. For each account, list containers
	// 3. For each container, list blobs matching *.tfstate pattern
	// 4. Download and parse state file metadata
	return []*TFStateFile{}
}

// discoverGCSStates discovers terraform state files in Google Cloud Storage
func (d *TFStateDiscovery) discoverGCSStates(ctx context.Context) []*TFStateFile {
	// TODO: Implement actual GCS bucket scanning using Google Cloud SDK
	// This would require:
	// 1. List GCS buckets
	// 2. For each bucket, list objects matching *.tfstate pattern
	// 3. Download and parse state file metadata
	return []*TFStateFile{}
}