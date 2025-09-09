package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	statePkg "github.com/catherinevee/driftmgr/internal/state"
)

// S3Config represents S3 backend configuration
type S3Config struct {
	Bucket         string `json:"bucket"`
	Key            string `json:"key"`
	Region         string `json:"region"`
	DynamoDBTable  string `json:"dynamodb_table"`
	Encrypt        bool   `json:"encrypt"`
	Profile        string `json:"profile"`
	RoleARN        string `json:"role_arn"`
	ExternalID     string `json:"external_id"`
	SessionName    string `json:"session_name"`
	Endpoint       string `json:"endpoint"`
	SkipValidation bool   `json:"skip_validation"`
}

// AzureConfig represents Azure backend configuration
type AzureConfig struct {
	StorageAccountName string `json:"storage_account_name"`
	ContainerName      string `json:"container_name"`
	Key                string `json:"key"`
	ResourceGroupName  string `json:"resource_group_name"`
	SubscriptionID     string `json:"subscription_id"`
	TenantID           string `json:"tenant_id"`
	ClientID           string `json:"client_id"`
	ClientSecret       string `json:"client_secret"`
	UseMSI             bool   `json:"use_msi"`
	Environment        string `json:"environment"`
	Endpoint           string `json:"endpoint"`
	Encrypt            bool   `json:"encrypt"`
}

// Adapter bridges the new Backend interface with the legacy state.Backend interface
type Adapter struct {
	backend Backend
	config  *BackendConfig
}

// NewAdapter creates a new backend adapter
func NewAdapter(backend Backend, config *BackendConfig) *Adapter {
	return &Adapter{
		backend: backend,
		config:  config,
	}
}

// Get retrieves data for a given key (legacy interface)
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	// Map key to workspace if needed
	if err := a.selectWorkspaceFromKey(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to select workspace: %w", err)
	}

	// Pull the state from the backend
	stateData, err := a.backend.Pull(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to pull state: %w", err)
	}

	// Return the raw data
	return stateData.Data, nil
}

// Put stores data for a given key (legacy interface)
func (a *Adapter) Put(ctx context.Context, key string, data []byte) error {
	// Map key to workspace if needed
	if err := a.selectWorkspaceFromKey(ctx, key); err != nil {
		return fmt.Errorf("failed to select workspace: %w", err)
	}

	// Parse the data to get state metadata
	var stateMeta struct {
		Version          int                    `json:"version"`
		TerraformVersion string                 `json:"terraform_version"`
		Serial           uint64                 `json:"serial"`
		Lineage          string                 `json:"lineage"`
		Resources        []json.RawMessage      `json:"resources"`
		Outputs          map[string]interface{} `json:"outputs"`
	}

	if err := json.Unmarshal(data, &stateMeta); err != nil {
		return fmt.Errorf("failed to parse state: %w", err)
	}

	// Create StateData from the raw data
	stateData := &StateData{
		Version:          stateMeta.Version,
		TerraformVersion: stateMeta.TerraformVersion,
		Serial:           stateMeta.Serial,
		Lineage:          stateMeta.Lineage,
		Data:             data,
		Outputs:          stateMeta.Outputs,
		LastModified:     time.Now(),
		Size:             int64(len(data)),
	}

	// Parse resources if available
	for _, rawResource := range stateMeta.Resources {
		var resource StateResource
		if err := json.Unmarshal(rawResource, &resource); err == nil {
			stateData.Resources = append(stateData.Resources, resource)
		}
	}

	// Push to backend
	return a.backend.Push(ctx, stateData)
}

// Delete removes data for a given key (legacy interface)
func (a *Adapter) Delete(ctx context.Context, key string) error {
	// Map key to workspace
	workspace := a.extractWorkspaceFromKey(key)
	if workspace != "" && workspace != "default" {
		return a.backend.DeleteWorkspace(ctx, workspace)
	}

	// For default workspace, we can't delete it, just clear it
	emptyState := &StateData{
		Version:      4,
		Serial:       0,
		Lineage:      "",
		Data:         []byte(`{"version": 4, "serial": 0, "resources": [], "outputs": {}}`),
		Resources:    []StateResource{},
		Outputs:      make(map[string]interface{}),
		LastModified: time.Now(),
	}

	return a.backend.Push(ctx, emptyState)
}

// List returns all keys with the given prefix (legacy interface)
func (a *Adapter) List(ctx context.Context, prefix string) ([]string, error) {
	// List workspaces and return them as keys
	workspaces, err := a.backend.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, ws := range workspaces {
		key := fmt.Sprintf("%s/%s", prefix, ws)
		keys = append(keys, key)
	}

	return keys, nil
}

// Lock acquires a lock on the state (legacy interface)
func (a *Adapter) Lock(ctx context.Context, key string) error {
	// Create lock info
	lockInfo := &LockInfo{
		Path:      key,
		Operation: "lock",
		Created:   time.Now(),
	}

	_, err := a.backend.Lock(ctx, lockInfo)
	return err
}

// Unlock releases a lock on the state (legacy interface)
func (a *Adapter) Unlock(ctx context.Context, key string) error {
	// We need to track lock IDs - for now, use empty string
	// In production, this would need proper lock ID management
	return a.backend.Unlock(ctx, "")
}

// ListStates returns all available states (legacy interface)
func (a *Adapter) ListStates(ctx context.Context) ([]string, error) {
	return a.backend.ListWorkspaces(ctx)
}

// ListStateVersions returns versions for a state (legacy interface)
func (a *Adapter) ListStateVersions(ctx context.Context, key string) ([]statePkg.StateVersion, error) {
	// Map key to workspace if needed
	if err := a.selectWorkspaceFromKey(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to select workspace: %w", err)
	}

	// Get versions from backend
	backendVersions, err := a.backend.GetVersions(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to legacy StateVersion format
	var versions []statePkg.StateVersion
	for _, bv := range backendVersions {
		versions = append(versions, statePkg.StateVersion{
			Version:   1, // Default version number
			Serial:    int(bv.Serial),
			Timestamp: bv.Created,
			Checksum:  bv.Checksum,
		})
	}

	return versions, nil
}

// GetStateVersion retrieves a specific version (legacy interface)
func (a *Adapter) GetStateVersion(ctx context.Context, key string, version int) ([]byte, error) {
	// Map key to workspace if needed
	if err := a.selectWorkspaceFromKey(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to select workspace: %w", err)
	}

	// Get all versions to find the right one
	versions, err := a.backend.GetVersions(ctx)
	if err != nil {
		return nil, err
	}

	// Find the version by index (simplified approach)
	if version < 0 || version >= len(versions) {
		return nil, fmt.Errorf("version %d not found", version)
	}

	// Get the specific version
	stateData, err := a.backend.GetVersion(ctx, versions[version].VersionID)
	if err != nil {
		return nil, err
	}

	return stateData.Data, nil
}

// Helper methods

// selectWorkspaceFromKey extracts workspace from key and selects it
func (a *Adapter) selectWorkspaceFromKey(ctx context.Context, key string) error {
	workspace := a.extractWorkspaceFromKey(key)
	if workspace == "" {
		workspace = "default"
	}

	return a.backend.SelectWorkspace(ctx, workspace)
}

// extractWorkspaceFromKey extracts workspace name from a key
func (a *Adapter) extractWorkspaceFromKey(key string) string {
	// Keys might be in format: "env/production" or "workspaces/production"
	parts := strings.Split(key, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return key
}

// CreateBackendAdapter creates an adapter for the appropriate backend type
func CreateBackendAdapter(config *BackendConfig) (statePkg.Backend, error) {
	var backend Backend
	var err error

	switch strings.ToLower(config.Type) {
	case "s3":
		_ = &S3Config{
			Bucket:         getStringFromConfig(config.Config, "bucket"),
			Key:            getStringFromConfig(config.Config, "key"),
			Region:         getStringFromConfig(config.Config, "region"),
			DynamoDBTable:  getStringFromConfig(config.Config, "dynamodb_table"),
			Encrypt:        getBoolFromConfig(config.Config, "encrypt"),
			Profile:        getStringFromConfig(config.Config, "profile"),
			RoleARN:        getStringFromConfig(config.Config, "role_arn"),
			ExternalID:     getStringFromConfig(config.Config, "external_id"),
			SessionName:    getStringFromConfig(config.Config, "session_name"),
			Endpoint:       getStringFromConfig(config.Config, "endpoint"),
			SkipValidation: getBoolFromConfig(config.Config, "skip_validation"),
		}
		backend, err = NewS3Backend(config)

	case "azurerm":
		// TODO: Fix Azure backend implementation
		return nil, fmt.Errorf("Azure backend temporarily disabled due to SDK compatibility issues")

	case "gcs":
		// TODO: Implement GCS backend
		return nil, fmt.Errorf("GCS backend not yet implemented")

	case "remote":
		// TODO: Implement Terraform Cloud backend
		return nil, fmt.Errorf("Terraform Cloud backend not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported backend type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	return NewAdapter(backend, config), nil
}

// Helper functions to extract values from config map

func getStringFromConfig(config map[string]interface{}, key string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolFromConfig(config map[string]interface{}, key string) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getIntFromConfig(config map[string]interface{}, key string) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}
