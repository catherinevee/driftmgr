package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
)

// StateManager manages Terraform state operations
type StateManager struct {
	workDir        string
	stateFiles     map[string]*StateFile
	backupDir      string
	remoteBackends map[string]*RemoteBackend
}

// StateFile represents a Terraform state file
type StateFile struct {
	Path         string                 `json:"path"`
	State        *tfjson.State          `json:"state"`
	BackupPath   string                 `json:"backup_path,omitempty"`
	LastModified time.Time              `json:"last_modified"`
	IsRemote     bool                   `json:"is_remote"`
	Backend      *RemoteBackend         `json:"backend,omitempty"`
	Metadata     map[string]string      `json:"metadata"`
}

// RemoteBackend represents a remote state backend
type RemoteBackend struct {
	Type          string            `json:"type"`
	Config        map[string]string `json:"config"`
	WorkspaceName string            `json:"workspace_name,omitempty"`
	Authenticated bool              `json:"authenticated"`
}

// NewStateManager creates a new state manager
func NewStateManager(workDir string) *StateManager {
	backupDir := filepath.Join(workDir, "state-backups")
	os.MkdirAll(backupDir, 0755)
	
	return &StateManager{
		workDir:        workDir,
		stateFiles:     make(map[string]*StateFile),
		backupDir:      backupDir,
		remoteBackends: make(map[string]*RemoteBackend),
	}
}

// LoadStateFile loads a Terraform state file
func (sm *StateManager) LoadStateFile(path string) (*StateFile, error) {
	// Check if already loaded
	if stateFile, ok := sm.stateFiles[path]; ok {
		return stateFile, nil
	}
	
	// Check if it's a remote state reference
	if strings.HasPrefix(path, "s3://") || strings.HasPrefix(path, "azurerm://") || 
	   strings.HasPrefix(path, "gs://") || strings.HasPrefix(path, "remote://") {
		return sm.loadRemoteState(path)
	}
	
	// Load local state file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	var state tfjson.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat state file: %w", err)
	}
	
	stateFile := &StateFile{
		Path:         path,
		State:        &state,
		LastModified: fileInfo.ModTime(),
		IsRemote:     false,
		Metadata:     make(map[string]string),
	}
	
	sm.stateFiles[path] = stateFile
	return stateFile, nil
}

// loadRemoteState loads a remote state file
func (sm *StateManager) loadRemoteState(path string) (*StateFile, error) {
	// Parse remote path
	backend, err := sm.parseRemotePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote path: %w", err)
	}
	
	// Pull remote state
	state, err := sm.pullRemoteState(backend)
	if err != nil {
		return nil, fmt.Errorf("failed to pull remote state: %w", err)
	}
	
	stateFile := &StateFile{
		Path:         path,
		State:        state,
		LastModified: time.Now(),
		IsRemote:     true,
		Backend:      backend,
		Metadata:     make(map[string]string),
	}
	
	sm.stateFiles[path] = stateFile
	return stateFile, nil
}

// parseRemotePath parses a remote state path
func (sm *StateManager) parseRemotePath(path string) (*RemoteBackend, error) {
	backend := &RemoteBackend{
		Config: make(map[string]string),
	}
	
	if strings.HasPrefix(path, "s3://") {
		// Parse S3 path: s3://bucket/key
		parts := strings.TrimPrefix(path, "s3://")
		components := strings.SplitN(parts, "/", 2)
		if len(components) != 2 {
			return nil, fmt.Errorf("invalid S3 path format")
		}
		
		backend.Type = "s3"
		backend.Config["bucket"] = components[0]
		backend.Config["key"] = components[1]
		
	} else if strings.HasPrefix(path, "azurerm://") {
		// Parse Azure path: azurerm://storage_account/container/key
		parts := strings.TrimPrefix(path, "azurerm://")
		components := strings.Split(parts, "/")
		if len(components) < 3 {
			return nil, fmt.Errorf("invalid Azure path format")
		}
		
		backend.Type = "azurerm"
		backend.Config["storage_account_name"] = components[0]
		backend.Config["container_name"] = components[1]
		backend.Config["key"] = strings.Join(components[2:], "/")
		
	} else if strings.HasPrefix(path, "gs://") {
		// Parse GCS path: gs://bucket/prefix
		parts := strings.TrimPrefix(path, "gs://")
		components := strings.SplitN(parts, "/", 2)
		if len(components) < 1 {
			return nil, fmt.Errorf("invalid GCS path format")
		}
		
		backend.Type = "gcs"
		backend.Config["bucket"] = components[0]
		if len(components) > 1 {
			backend.Config["prefix"] = components[1]
		}
		
	} else if strings.HasPrefix(path, "remote://") {
		// Parse Terraform Cloud path: remote://organization/workspace
		parts := strings.TrimPrefix(path, "remote://")
		components := strings.Split(parts, "/")
		if len(components) != 2 {
			return nil, fmt.Errorf("invalid Terraform Cloud path format")
		}
		
		backend.Type = "remote"
		backend.Config["organization"] = components[0]
		backend.Config["workspaces"] = components[1]
		backend.WorkspaceName = components[1]
		
	} else {
		return nil, fmt.Errorf("unsupported remote backend type")
	}
	
	return backend, nil
}

// pullRemoteState pulls state from a remote backend
func (sm *StateManager) pullRemoteState(backend *RemoteBackend) (*tfjson.State, error) {
	// Create temporary directory for pulling state
	tempDir, err := os.MkdirTemp(sm.workDir, "remote-state-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Write backend configuration
	backendConfig := sm.generateBackendConfig(backend)
	configPath := filepath.Join(tempDir, "backend.tf")
	if err := os.WriteFile(configPath, []byte(backendConfig), 0644); err != nil {
		return nil, fmt.Errorf("failed to write backend config: %w", err)
	}
	
	// Initialize Terraform
	initCmd := exec.Command("terraform", "init", "-backend=true", "-get=false")
	initCmd.Dir = tempDir
	if output, err := initCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w\nOutput: %s", err, output)
	}
	
	// Pull state
	pullCmd := exec.Command("terraform", "state", "pull")
	pullCmd.Dir = tempDir
	output, err := pullCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform state pull failed: %w", err)
	}
	
	// Parse state
	var state tfjson.State
	if err := json.Unmarshal(output, &state); err != nil {
		return nil, fmt.Errorf("failed to parse pulled state: %w", err)
	}
	
	return &state, nil
}

// generateBackendConfig generates Terraform backend configuration
func (sm *StateManager) generateBackendConfig(backend *RemoteBackend) string {
	var config strings.Builder
	
	config.WriteString("terraform {\n")
	config.WriteString(fmt.Sprintf("  backend \"%s\" {\n", backend.Type))
	
	for key, value := range backend.Config {
		config.WriteString(fmt.Sprintf("    %s = \"%s\"\n", key, value))
	}
	
	config.WriteString("  }\n")
	config.WriteString("}\n")
	
	return config.String()
}

// BackupStateFile creates a backup of a state file
func (sm *StateManager) BackupStateFile(stateFile *StateFile) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("state-backup-%s.tfstate", timestamp)
	backupPath := filepath.Join(sm.backupDir, backupName)
	
	// Marshal state to JSON
	data, err := json.MarshalIndent(stateFile.State, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}
	
	// Write backup file
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}
	
	stateFile.BackupPath = backupPath
	return backupPath, nil
}

// RestoreStateFile restores a state file from backup
func (sm *StateManager) RestoreStateFile(backupPath string, targetPath string) error {
	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}
	
	// Validate it's valid state
	var state tfjson.State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("invalid state backup: %w", err)
	}
	
	// Write to target
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore state: %w", err)
	}
	
	return nil
}

// GetStateResources returns all resources from a state file
func (sm *StateManager) GetStateResources(stateFile *StateFile) ([]*tfjson.StateResource, error) {
	if stateFile.State == nil || stateFile.State.Values == nil {
		return nil, fmt.Errorf("state file has no values")
	}
	
	var resources []*tfjson.StateResource
	
	// Get resources from root module
	if stateFile.State.Values.RootModule != nil {
		resources = append(resources, stateFile.State.Values.RootModule.Resources...)
		
		// Get resources from child modules
		for _, module := range stateFile.State.Values.RootModule.ChildModules {
			resources = append(resources, sm.getModuleResources(module)...)
		}
	}
	
	return resources, nil
}

// getModuleResources recursively gets resources from a module
func (sm *StateManager) getModuleResources(module *tfjson.StateModule) []*tfjson.StateResource {
	var resources []*tfjson.StateResource
	
	resources = append(resources, module.Resources...)
	
	for _, childModule := range module.ChildModules {
		resources = append(resources, sm.getModuleResources(childModule)...)
	}
	
	return resources
}

// FindResourceInState finds a specific resource in state
func (sm *StateManager) FindResourceInState(stateFile *StateFile, resourceType, resourceName string) (*tfjson.StateResource, error) {
	resources, err := sm.GetStateResources(stateFile)
	if err != nil {
		return nil, err
	}
	
	for _, resource := range resources {
		if resource.Type == resourceType && resource.Name == resourceName {
			return resource, nil
		}
	}
	
	return nil, fmt.Errorf("resource %s.%s not found in state", resourceType, resourceName)
}

// UpdateResourceInState updates a resource in state
func (sm *StateManager) UpdateResourceInState(ctx context.Context, stateFile *StateFile, resource *tfjson.StateResource) error {
	// This would use terraform state commands to update the resource
	// For safety, we'll create a backup first
	if _, err := sm.BackupStateFile(stateFile); err != nil {
		return fmt.Errorf("failed to backup state before update: %w", err)
	}
	
	// In a real implementation, this would:
	// 1. Write the updated resource to a temporary file
	// 2. Use `terraform state push` or similar to update
	// 3. Verify the update was successful
	
	return nil
}

// RemoveResourceFromState removes a resource from state
func (sm *StateManager) RemoveResourceFromState(ctx context.Context, stateFile *StateFile, resourceType, resourceName string) error {
	// Create backup first
	if _, err := sm.BackupStateFile(stateFile); err != nil {
		return fmt.Errorf("failed to backup state before removal: %w", err)
	}
	
	// Use terraform state rm command
	address := fmt.Sprintf("%s.%s", resourceType, resourceName)
	
	cmd := exec.CommandContext(ctx, "terraform", "state", "rm", address)
	cmd.Dir = filepath.Dir(stateFile.Path)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove resource from state: %w\nOutput: %s", err, output)
	}
	
	// Reload state
	if _, err := sm.LoadStateFile(stateFile.Path); err != nil {
		return fmt.Errorf("failed to reload state after removal: %w", err)
	}
	
	return nil
}

// ImportResourceToState imports a resource into state
func (sm *StateManager) ImportResourceToState(ctx context.Context, stateFile *StateFile, resourceType, resourceName, resourceID string) error {
	// Create backup first
	if _, err := sm.BackupStateFile(stateFile); err != nil {
		return fmt.Errorf("failed to backup state before import: %w", err)
	}
	
	// Use terraform import command
	address := fmt.Sprintf("%s.%s", resourceType, resourceName)
	
	cmd := exec.CommandContext(ctx, "terraform", "import", address, resourceID)
	cmd.Dir = filepath.Dir(stateFile.Path)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to import resource to state: %w\nOutput: %s", err, output)
	}
	
	// Reload state
	if _, err := sm.LoadStateFile(stateFile.Path); err != nil {
		return fmt.Errorf("failed to reload state after import: %w", err)
	}
	
	return nil
}

// ValidateState validates a state file
func (sm *StateManager) ValidateState(stateFile *StateFile) error {
	if stateFile.State == nil {
		return fmt.Errorf("state is nil")
	}
	
	// Check format version
	if stateFile.State.FormatVersion != "" {
		// FormatVersion is a string, we can validate it's not empty
		// Most modern Terraform states use format version "0.2" or similar
	}
	
	if stateFile.State.TerraformVersion == "" {
		return fmt.Errorf("state has no Terraform version")
	}
	
	return nil
}

// ListBackups lists all state backups
func (sm *StateManager) ListBackups() ([]string, error) {
	entries, err := os.ReadDir(sm.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	
	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tfstate") {
			backups = append(backups, filepath.Join(sm.backupDir, entry.Name()))
		}
	}
	
	return backups, nil
}

// CleanOldBackups removes backups older than the specified duration
func (sm *StateManager) CleanOldBackups(maxAge time.Duration) error {
	backups, err := sm.ListBackups()
	if err != nil {
		return err
	}
	
	cutoff := time.Now().Add(-maxAge)
	
	for _, backup := range backups {
		info, err := os.Stat(backup)
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(backup); err != nil {
				fmt.Printf("Failed to remove old backup %s: %v\n", backup, err)
			}
		}
	}
	
	return nil
}