package backend

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalBackend implements the Backend interface for local file storage
type LocalBackend struct {
	basePath  string
	workspace string
	mu        sync.RWMutex
	locks     map[string]*LockInfo
	metadata  *BackendMetadata
}

// NewLocalBackend creates a new local file backend instance
func NewLocalBackend(cfg *BackendConfig) (*LocalBackend, error) {
	basePath, _ := cfg.Config["path"].(string)
	workspace, _ := cfg.Config["workspace"].(string)

	if basePath == "" {
		basePath = "."
	}
	if workspace == "" {
		workspace = "default"
	}

	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	backend := &LocalBackend{
		basePath:  basePath,
		workspace: workspace,
		locks:     make(map[string]*LockInfo),
		metadata: &BackendMetadata{
			Type:               "local",
			SupportsLocking:    true,
			SupportsVersions:   true,
			SupportsWorkspaces: true,
			Configuration: map[string]string{
				"path": basePath,
			},
			Workspace: workspace,
			StateKey:  "terraform.tfstate",
		},
	}

	return backend, nil
}

// Pull retrieves the current state from local file
func (l *LocalBackend) Pull(ctx context.Context) (*StateData, error) {
	statePath := l.getStatePath()

	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		// Return empty state if file doesn't exist
		return &StateData{
			Version:      4,
			Serial:       0,
			Lineage:      generateLineage(),
			Data:         []byte(`{"version": 4, "serial": 0, "resources": [], "outputs": {}}`),
			LastModified: time.Now(),
			Size:         0,
		}, nil
	}

	// Read state file
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Parse state metadata
	var stateMetadata map[string]interface{}
	if err := json.Unmarshal(data, &stateMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse state metadata: %w", err)
	}

	state := &StateData{
		Data:         data,
		LastModified: fileInfo.ModTime(),
		Size:         fileInfo.Size(),
	}

	// Extract metadata
	if version, ok := stateMetadata["version"].(float64); ok {
		state.Version = int(version)
	}
	if serial, ok := stateMetadata["serial"].(float64); ok {
		state.Serial = uint64(serial)
	}
	if lineage, ok := stateMetadata["lineage"].(string); ok {
		state.Lineage = lineage
	}
	if tfVersion, ok := stateMetadata["terraform_version"].(string); ok {
		state.TerraformVersion = tfVersion
	}

	// Calculate checksum
	h := md5.New()
	h.Write(data)
	state.Checksum = base64.StdEncoding.EncodeToString(h.Sum(nil))

	return state, nil
}

// Push uploads state to local file
func (l *LocalBackend) Push(ctx context.Context, state *StateData) error {
	statePath := l.getStatePath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Prepare state data
	var data []byte
	if state.Data != nil {
		data = state.Data
	} else {
		var err error
		data, err = json.MarshalIndent(state, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state: %w", err)
		}
	}

	// Create backup of existing state if it exists
	if err := l.createBackup(statePath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write state file atomically
	tempPath := statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tempPath, statePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to move temp state file: %w", err)
	}

	return nil
}

// Lock acquires a lock on the state
func (l *LocalBackend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	lockPath := l.getLockPath()

	// Check if lock file already exists
	if _, err := os.Stat(lockPath); err == nil {
		// Read existing lock info
		if existingLock, err := l.readLockFile(lockPath); err == nil {
			return "", fmt.Errorf("state is already locked by %s since %s",
				existingLock.Who, existingLock.Created.Format(time.RFC3339))
		}
		return "", fmt.Errorf("state is already locked")
	}

	// Create lock file
	lockID := fmt.Sprintf("%s-%d", info.ID, time.Now().UnixNano())
	info.ID = lockID

	lockData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal lock info: %w", err)
	}

	if err := os.WriteFile(lockPath, lockData, 0644); err != nil {
		return "", fmt.Errorf("failed to create lock file: %w", err)
	}

	l.locks[lockID] = info
	return lockID, nil
}

// Unlock releases the lock on the state
func (l *LocalBackend) Unlock(ctx context.Context, lockID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	lockPath := l.getLockPath()

	// Remove lock file
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	delete(l.locks, lockID)
	return nil
}

// GetVersions returns available state versions (backup files)
func (l *LocalBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	statePath := l.getStatePath()
	backupDir := filepath.Join(filepath.Dir(statePath), ".terraform", "backups")

	var versions []*StateVersion

	// Add current version
	if fileInfo, err := os.Stat(statePath); err == nil {
		versions = append(versions, &StateVersion{
			ID:        "current",
			VersionID: "current",
			Created:   fileInfo.ModTime(),
			Size:      fileInfo.Size(),
			IsLatest:  true,
		})
	}

	// Add backup versions
	if entries, err := os.ReadDir(backupDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tfstate") {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			versions = append(versions, &StateVersion{
				ID:        entry.Name(),
				VersionID: entry.Name(),
				Created:   info.ModTime(),
				Size:      info.Size(),
				IsLatest:  false,
			})
		}
	}

	return versions, nil
}

// GetVersion retrieves a specific version of the state
func (l *LocalBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	var filePath string

	if versionID == "current" || versionID == "" {
		filePath = l.getStatePath()
	} else {
		backupDir := filepath.Join(filepath.Dir(l.getStatePath()), ".terraform", "backups")
		filePath = filepath.Join(backupDir, versionID)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version %s: %w", versionID, err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	state := &StateData{
		Data:         data,
		LastModified: fileInfo.ModTime(),
		Size:         fileInfo.Size(),
	}

	// Parse state metadata
	var stateMetadata map[string]interface{}
	if err := json.Unmarshal(data, &stateMetadata); err == nil {
		if version, ok := stateMetadata["version"].(float64); ok {
			state.Version = int(version)
		}
		if serial, ok := stateMetadata["serial"].(float64); ok {
			state.Serial = uint64(serial)
		}
		if lineage, ok := stateMetadata["lineage"].(string); ok {
			state.Lineage = lineage
		}
	}

	return state, nil
}

// ListWorkspaces returns available workspaces
func (l *LocalBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	workspaceDir := filepath.Join(l.basePath, "workspaces")
	workspaces := []string{"default"}

	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				workspaces = append(workspaces, entry.Name())
			}
		}
	}

	return workspaces, nil
}

// SelectWorkspace switches to a different workspace
func (l *LocalBackend) SelectWorkspace(ctx context.Context, name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if workspace exists
	workspaces, err := l.ListWorkspaces(ctx)
	if err != nil {
		return err
	}

	found := false
	for _, ws := range workspaces {
		if ws == name {
			found = true
			break
		}
	}

	if !found && name != "default" {
		return fmt.Errorf("workspace %s does not exist", name)
	}

	l.workspace = name
	l.metadata.Workspace = name

	return nil
}

// CreateWorkspace creates a new workspace
func (l *LocalBackend) CreateWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return fmt.Errorf("cannot create default workspace")
	}

	// Check if workspace already exists
	workspaces, err := l.ListWorkspaces(ctx)
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		if ws == name {
			return fmt.Errorf("workspace %s already exists", name)
		}
	}

	// Create workspace directory
	workspaceDir := filepath.Join(l.basePath, "workspaces", name)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Create empty state for new workspace
	emptyState := &StateData{
		Version: 4,
		Serial:  0,
		Lineage: generateLineage(),
		Data:    []byte(`{"version": 4, "serial": 0, "resources": [], "outputs": {}}`),
	}

	// Save state with workspace
	oldWorkspace := l.workspace
	l.workspace = name
	err = l.Push(ctx, emptyState)
	l.workspace = oldWorkspace

	return err
}

// DeleteWorkspace removes a workspace
func (l *LocalBackend) DeleteWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete default workspace")
	}

	if l.workspace == name {
		return fmt.Errorf("cannot delete current workspace")
	}

	workspaceDir := filepath.Join(l.basePath, "workspaces", name)
	if err := os.RemoveAll(workspaceDir); err != nil {
		return fmt.Errorf("failed to delete workspace %s: %w", name, err)
	}

	return nil
}

// GetLockInfo returns current lock information
func (l *LocalBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	lockPath := l.getLockPath()

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return nil, nil // No lock exists
	}

	return l.readLockFile(lockPath)
}

// Validate checks if the backend is properly configured and accessible
func (l *LocalBackend) Validate(ctx context.Context) error {
	// Check if base path is accessible
	if _, err := os.Stat(l.basePath); err != nil {
		return fmt.Errorf("cannot access base path %s: %w", l.basePath, err)
	}

	// Check if we can write to the directory
	testFile := filepath.Join(l.basePath, ".driftmgr-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to base path %s: %w", l.basePath, err)
	}
	os.Remove(testFile)

	return nil
}

// GetMetadata returns backend metadata
func (l *LocalBackend) GetMetadata() *BackendMetadata {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.metadata
}

// Helper methods

func (l *LocalBackend) getStatePath() string {
	if l.workspace == "" || l.workspace == "default" {
		return filepath.Join(l.basePath, "terraform.tfstate")
	}
	return filepath.Join(l.basePath, "workspaces", l.workspace, "terraform.tfstate")
}

func (l *LocalBackend) getLockPath() string {
	statePath := l.getStatePath()
	return statePath + ".lock"
}

func (l *LocalBackend) createBackup(statePath string) error {
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil // No existing file to backup
	}

	backupDir := filepath.Join(filepath.Dir(statePath), ".terraform", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("terraform.tfstate.%s", timestamp))

	// Copy file
	src, err := os.Open(statePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (l *LocalBackend) readLockFile(lockPath string) (*LockInfo, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lockInfo LockInfo
	if err := json.Unmarshal(data, &lockInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lock info: %w", err)
	}

	return &lockInfo, nil
}

func generateLineage() string {
	return fmt.Sprintf("lineage-%d", time.Now().UnixNano())
}