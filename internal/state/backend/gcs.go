package backend

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// GCSBackend implements the Backend interface for Google Cloud Storage
// This is a stub implementation for testing purposes
type GCSBackend struct {
	bucket      string
	prefix      string
	credentials string
	project     string
	workspace   string

	// Mock storage for testing
	objects  map[string][]byte
	metadata map[string]map[string]string
	versions map[string][]*GCSVersion
	locks    map[string]*LockInfo

	mu          sync.RWMutex
	backendMeta *BackendMetadata
}

// GCSVersion represents a version of an object in GCS
type GCSVersion struct {
	Generation   int64
	Data         []byte
	LastModified time.Time
	Size         int64
	ETag         string
	IsLatest     bool
}

// NewGCSBackend creates a new GCS backend instance
func NewGCSBackend(cfg *BackendConfig) (*GCSBackend, error) {
	// Extract GCS-specific configuration
	bucket, _ := cfg.Config["bucket"].(string)
	prefix, _ := cfg.Config["prefix"].(string)
	credentials, _ := cfg.Config["credentials"].(string)
	project, _ := cfg.Config["project"].(string)
	workspace, _ := cfg.Config["workspace"].(string)

	if bucket == "" {
		return nil, fmt.Errorf("bucket is required for GCS backend")
	}

	if workspace == "" {
		workspace = "default"
	}

	if prefix == "" {
		prefix = "terraform/state"
	}

	backend := &GCSBackend{
		bucket:      bucket,
		prefix:      prefix,
		credentials: credentials,
		project:     project,
		workspace:   workspace,
		objects:     make(map[string][]byte),
		metadata:    make(map[string]map[string]string),
		versions:    make(map[string][]*GCSVersion),
		locks:       make(map[string]*LockInfo),
		backendMeta: &BackendMetadata{
			Type:               "gcs",
			SupportsLocking:    true,
			SupportsVersions:   true,
			SupportsWorkspaces: true,
			Configuration: map[string]string{
				"bucket":  bucket,
				"prefix":  prefix,
				"project": project,
			},
			Workspace: workspace,
			StateKey:  "default.tfstate",
		},
	}

	return backend, nil
}

// Pull retrieves the current state from GCS
func (g *GCSBackend) Pull(ctx context.Context) (*StateData, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	objectName := g.getObjectName()

	// Check if object exists
	data, exists := g.objects[objectName]
	if !exists {
		// Return empty state if not found
		return &StateData{
			Version:      4,
			Serial:       0,
			Lineage:      generateLineage(),
			Data:         []byte(`{"version": 4, "serial": 0, "resources": [], "outputs": {}}`),
			LastModified: time.Now(),
			Size:         0,
		}, nil
	}

	// Parse state metadata
	var stateMetadata map[string]interface{}
	if err := json.Unmarshal(data, &stateMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse state metadata: %w", err)
	}

	state := &StateData{
		Data:         data,
		LastModified: time.Now(),
		Size:         int64(len(data)),
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

// Push uploads state to GCS
func (g *GCSBackend) Push(ctx context.Context, state *StateData) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	objectName := g.getObjectName()

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

	// Store object
	g.objects[objectName] = data

	// Store metadata
	g.metadata[objectName] = map[string]string{
		"terraform-version": state.TerraformVersion,
		"serial":            fmt.Sprintf("%d", state.Serial),
		"lineage":           state.Lineage,
	}

	// Create version
	generation := time.Now().UnixNano()
	version := &GCSVersion{
		Generation:   generation,
		Data:         data,
		LastModified: time.Now(),
		Size:         int64(len(data)),
		ETag:         fmt.Sprintf("mock-etag-%d", generation),
		IsLatest:     true,
	}

	// Mark previous versions as not latest
	if versions, exists := g.versions[objectName]; exists {
		for _, v := range versions {
			v.IsLatest = false
		}
	}

	g.versions[objectName] = append(g.versions[objectName], version)

	return nil
}

// Lock acquires a lock on the state (stub implementation)
func (g *GCSBackend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	lockKey := g.getLockKey()

	// Check if already locked
	if _, exists := g.locks[lockKey]; exists {
		return "", fmt.Errorf("state is already locked")
	}

	// Create lock
	lockID := fmt.Sprintf("gcs-lock-%d", time.Now().UnixNano())
	info.ID = lockID
	g.locks[lockKey] = info

	return lockID, nil
}

// Unlock releases the lock on the state
func (g *GCSBackend) Unlock(ctx context.Context, lockID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	lockKey := g.getLockKey()
	delete(g.locks, lockKey)

	return nil
}

// GetVersions returns available state versions
func (g *GCSBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	objectName := g.getObjectName()
	var versions []*StateVersion

	if gcsVersions, exists := g.versions[objectName]; exists {
		for i, v := range gcsVersions {
			version := &StateVersion{
				ID:        fmt.Sprintf("gen-%d", v.Generation),
				VersionID: fmt.Sprintf("%d", v.Generation),
				Created:   v.LastModified,
				Size:      v.Size,
				IsLatest:  v.IsLatest,
				Checksum:  v.ETag,
			}

			// Extract serial from metadata
			if i < 5 { // Only process recent versions
				if metadata, exists := g.metadata[objectName]; exists {
					if serial, ok := metadata["serial"]; ok {
						var s uint64
						fmt.Sscanf(serial, "%d", &s)
						version.Serial = s
					}
				}
			}

			versions = append(versions, version)
		}
	}

	return versions, nil
}

// GetVersion retrieves a specific version of the state
func (g *GCSBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	objectName := g.getObjectName()

	if versionID == "current" || versionID == "" {
		return g.Pull(ctx)
	}

	// Find specific version by generation
	if gcsVersions, exists := g.versions[objectName]; exists {
		for _, v := range gcsVersions {
			if fmt.Sprintf("%d", v.Generation) == versionID {
				state := &StateData{
					Data:         v.Data,
					LastModified: v.LastModified,
					Size:         v.Size,
				}

				// Parse metadata from data
				var stateMetadata map[string]interface{}
				if err := json.Unmarshal(v.Data, &stateMetadata); err == nil {
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
		}
	}

	return nil, fmt.Errorf("version %s not found", versionID)
}

// ListWorkspaces returns available workspaces
func (g *GCSBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	workspaceMap := make(map[string]bool)
	workspaceMap["default"] = true

	// Look for workspace objects
	for objectName := range g.objects {
		if g.isWorkspaceObject(objectName) {
			workspace := g.extractWorkspaceFromObject(objectName)
			if workspace != "" && workspace != "default" {
				workspaceMap[workspace] = true
			}
		}
	}

	workspaces := make([]string, 0, len(workspaceMap))
	for ws := range workspaceMap {
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// SelectWorkspace switches to a different workspace
func (g *GCSBackend) SelectWorkspace(ctx context.Context, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if workspace exists
	workspaces, err := g.ListWorkspaces(ctx)
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

	g.workspace = name
	g.backendMeta.Workspace = name

	return nil
}

// CreateWorkspace creates a new workspace
func (g *GCSBackend) CreateWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return fmt.Errorf("cannot create default workspace")
	}

	// Check if workspace already exists
	workspaces, err := g.ListWorkspaces(ctx)
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		if ws == name {
			return fmt.Errorf("workspace %s already exists", name)
		}
	}

	// Create empty state for new workspace
	emptyState := &StateData{
		Version: 4,
		Serial:  0,
		Lineage: generateLineage(),
		Data:    []byte(`{"version": 4, "serial": 0, "resources": [], "outputs": {}}`),
	}

	// Save state with workspace
	oldWorkspace := g.workspace
	g.workspace = name
	err = g.Push(ctx, emptyState)
	g.workspace = oldWorkspace

	return err
}

// DeleteWorkspace removes a workspace
func (g *GCSBackend) DeleteWorkspace(ctx context.Context, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if name == "default" {
		return fmt.Errorf("cannot delete default workspace")
	}

	if g.workspace == name {
		return fmt.Errorf("cannot delete current workspace")
	}

	// Remove workspace objects
	objectsToDelete := make([]string, 0)
	for objectName := range g.objects {
		if g.isWorkspaceObject(objectName) {
			workspace := g.extractWorkspaceFromObject(objectName)
			if workspace == name {
				objectsToDelete = append(objectsToDelete, objectName)
			}
		}
	}

	for _, objectName := range objectsToDelete {
		delete(g.objects, objectName)
		delete(g.metadata, objectName)
		delete(g.versions, objectName)
	}

	return nil
}

// GetLockInfo returns current lock information
func (g *GCSBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	lockKey := g.getLockKey()
	if lockInfo, exists := g.locks[lockKey]; exists {
		return lockInfo, nil
	}

	return nil, nil
}

// Validate checks if the backend is properly configured and accessible
func (g *GCSBackend) Validate(ctx context.Context) error {
	// For stub implementation, always return success
	return nil
}

// GetMetadata returns backend metadata
func (g *GCSBackend) GetMetadata() *BackendMetadata {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.backendMeta
}

// Helper methods

func (g *GCSBackend) getObjectName() string {
	if g.workspace == "" || g.workspace == "default" {
		return fmt.Sprintf("%s/default.tfstate", g.prefix)
	}
	return fmt.Sprintf("%s/env:/%s/default.tfstate", g.prefix, g.workspace)
}

func (g *GCSBackend) getLockKey() string {
	return fmt.Sprintf("%s.lock", g.getObjectName())
}

func (g *GCSBackend) isWorkspaceObject(objectName string) bool {
	return objectName == g.getObjectName() ||
		(objectName != g.getObjectName() && objectName[len(objectName)-8:] == ".tfstate")
}

func (g *GCSBackend) extractWorkspaceFromObject(objectName string) string {
	// Extract workspace from object name like "prefix/env:/workspace/default.tfstate"
	if !strings.Contains(objectName, "/env:/") {
		return "default"
	}

	parts := strings.Split(objectName, "/env:/")
	if len(parts) < 2 {
		return "default"
	}

	workspaceParts := strings.Split(parts[1], "/")
	if len(workspaceParts) > 0 {
		return workspaceParts[0]
	}

	return "default"
}
