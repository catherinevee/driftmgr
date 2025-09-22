package backend

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSConfig represents Google Cloud Storage backend configuration
type GCSConfig struct {
	Bucket         string `json:"bucket"`
	Prefix         string `json:"prefix"`
	ProjectID      string `json:"project_id"`
	Credentials    string `json:"credentials"`
	EncryptionKey  string `json:"encryption_key"`
	SkipValidation bool   `json:"skip_validation"`
}

// GCSBackend implements the Backend interface for Google Cloud Storage
type GCSBackend struct {
	client    *storage.Client
	config    *GCSConfig
	bucket    *storage.BucketHandle
	workspace string
}

// NewGCSBackend creates a new GCS backend
func NewGCSBackend(config *BackendConfig) (Backend, error) {
	if config == nil {
		return nil, &ValidationError{Field: "config", Message: "config cannot be nil"}
	}

	// Extract GCS-specific configuration
	gcsConfig := &GCSConfig{
		Bucket:         getStringFromConfig(config.Config, "bucket"),
		Prefix:         getStringFromConfig(config.Config, "prefix"),
		ProjectID:      getStringFromConfig(config.Config, "project_id"),
		Credentials:    getStringFromConfig(config.Config, "credentials"),
		EncryptionKey:  getStringFromConfig(config.Config, "encryption_key"),
		SkipValidation: getBoolFromConfig(config.Config, "skip_validation"),
	}

	// Validate required fields
	if gcsConfig.Bucket == "" {
		return nil, &ValidationError{Field: "bucket", Message: "bucket name is required"}
	}
	if gcsConfig.ProjectID == "" {
		return nil, &ValidationError{Field: "project_id", Message: "project ID is required"}
	}

	// Create storage client
	ctx := context.Background()
	var client *storage.Client
	var err error

	if gcsConfig.Credentials != "" {
		// Use service account credentials
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(gcsConfig.Credentials))
	} else {
		// Use default credentials (ADC, environment variables, etc.)
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Get bucket handle
	bucket := client.Bucket(gcsConfig.Bucket)

	// Validate bucket exists and is accessible
	if !gcsConfig.SkipValidation {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		attrs, err := bucket.Attrs(ctx)
		if err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to access bucket %s: %w", gcsConfig.Bucket, err)
		}

		// Log bucket information for debugging
		fmt.Printf("Connected to GCS bucket: %s (location: %s)\n", attrs.Name, attrs.Location)
	}

	return &GCSBackend{
		client:    client,
		config:    gcsConfig,
		bucket:    bucket,
		workspace: "default",
	}, nil
}

// Pull retrieves the current state from GCS
func (g *GCSBackend) Pull(ctx context.Context) (*StateData, error) {
	objectName := g.getStateObjectName()

	// Get object
	obj := g.bucket.Object(objectName)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			// Return empty state if object doesn't exist
			return &StateData{
				Version:      4,
				Serial:       0,
				Lineage:      "",
				Data:         []byte(`{"version": 4, "serial": 0, "resources": [], "outputs": {}}`),
				Resources:    []StateResource{},
				Outputs:      make(map[string]interface{}),
				LastModified: time.Now(),
				Size:         0,
			}, nil
		}
		return nil, fmt.Errorf("failed to read state object: %w", err)
	}
	defer reader.Close()

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}

	// Get object attributes for metadata
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	// Parse state data to extract metadata
	stateData, err := g.parseStateData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse state data: %w", err)
	}

	// Set metadata from object attributes
	stateData.LastModified = attrs.Updated
	stateData.Size = attrs.Size

	return stateData, nil
}

// Push stores the state to GCS
func (g *GCSBackend) Push(ctx context.Context, stateData *StateData) error {
	if stateData == nil {
		return &ValidationError{Field: "stateData", Message: "state data cannot be nil"}
	}

	objectName := g.getStateObjectName()

	// Create object writer
	obj := g.bucket.Object(objectName)
	writer := obj.NewWriter(ctx)

	// Set content type
	writer.ContentType = "application/json"

	// Set metadata
	writer.Metadata = map[string]string{
		"terraform-version": stateData.TerraformVersion,
		"serial":            fmt.Sprintf("%d", stateData.Serial),
		"lineage":           stateData.Lineage,
		"workspace":         g.workspace,
	}

	// Write data
	if _, err := writer.Write(stateData.Data); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write state data: %w", err)
	}

	// Close writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

// Lock acquires a lock on the state
func (g *GCSBackend) Lock(ctx context.Context, lockInfo *LockInfo) (string, error) {
	if lockInfo == nil {
		return "", &ValidationError{Field: "lockInfo", Message: "lock info cannot be nil"}
	}

	lockObjectName := g.getLockObjectName()

	// Check if lock already exists
	obj := g.bucket.Object(lockObjectName)
	_, err := obj.Attrs(ctx)
	if err == nil {
		return "", fmt.Errorf("state is already locked")
	}

	// Create lock object
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	// Set lock metadata
	writer.Metadata = map[string]string{
		"operation": lockInfo.Operation,
		"created":   lockInfo.Created.Format(time.RFC3339),
		"workspace": g.workspace,
	}

	// Write lock data
	lockData := fmt.Sprintf(`{"operation": "%s", "created": "%s", "workspace": "%s"}`,
		lockInfo.Operation, lockInfo.Created.Format(time.RFC3339), g.workspace)

	if _, err := writer.Write([]byte(lockData)); err != nil {
		writer.Close()
		return "", fmt.Errorf("failed to write lock data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close lock writer: %w", err)
	}

	// Return lock ID (using object name as lock ID)
	return lockObjectName, nil
}

// Unlock releases a lock on the state
func (g *GCSBackend) Unlock(ctx context.Context, lockID string) error {
	if lockID == "" {
		return &ValidationError{Field: "lockID", Message: "lock ID cannot be empty"}
	}

	// Delete lock object
	obj := g.bucket.Object(lockID)
	if err := obj.Delete(ctx); err != nil {
		if err == storage.ErrObjectNotExist {
			return nil // Lock already released
		}
		return fmt.Errorf("failed to delete lock: %w", err)
	}

	return nil
}

// ListWorkspaces returns all available workspaces
func (g *GCSBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	// List objects with workspace prefix
	query := &storage.Query{
		Prefix: g.config.Prefix + "/",
	}

	it := g.bucket.Objects(ctx, query)
	workspaces := make(map[string]bool)

	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Extract workspace from object name
		workspace := g.extractWorkspaceFromObjectName(obj.Name)
		if workspace != "" {
			workspaces[workspace] = true
		}
	}

	// Convert map to slice
	var result []string
	for workspace := range workspaces {
		result = append(result, workspace)
	}

	// Always include default workspace
	if !workspaces["default"] {
		result = append(result, "default")
	}

	return result, nil
}

// SelectWorkspace selects a workspace
func (g *GCSBackend) SelectWorkspace(ctx context.Context, workspace string) error {
	if workspace == "" {
		workspace = "default"
	}

	g.workspace = workspace
	return nil
}

// DeleteWorkspace deletes a workspace
func (g *GCSBackend) DeleteWorkspace(ctx context.Context, workspace string) error {
	if workspace == "" || workspace == "default" {
		return fmt.Errorf("cannot delete default workspace")
	}

	// List and delete all objects for this workspace
	prefix := g.getWorkspacePrefix(workspace)
	query := &storage.Query{
		Prefix: prefix,
	}

	it := g.bucket.Objects(ctx, query)
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list workspace objects: %w", err)
		}

		// Delete object
		if err := g.bucket.Object(obj.Name).Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete object %s: %w", obj.Name, err)
		}
	}

	return nil
}

// GetVersions returns all versions of the state
func (g *GCSBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	objectName := g.getStateObjectName()

	// List object generations (versions) using bucket query
	query := &storage.Query{
		Prefix:   objectName,
		Versions: true,
	}

	it := g.bucket.Objects(ctx, query)
	var versions []*StateVersion

	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list object generations: %w", err)
		}

		// Only include objects that match our exact state object name
		if obj.Name == objectName {
			versions = append(versions, &StateVersion{
				VersionID: fmt.Sprintf("%d", obj.Generation),
				Serial:    g.extractSerialFromMetadata(obj.Metadata),
				Created:   obj.Created,
				Checksum:  fmt.Sprintf("%x", md5.Sum([]byte(obj.Name))),
			})
		}
	}

	return versions, nil
}

// GetVersion retrieves a specific version of the state
func (g *GCSBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	objectName := g.getStateObjectName()
	obj := g.bucket.Object(objectName)

	// Get specific generation
	generation, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid version ID: %w", err)
	}

	reader, err := obj.Generation(generation).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read version %s: %w", versionID, err)
	}
	defer reader.Close()

	// Read data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read version data: %w", err)
	}

	// Parse state data
	return g.parseStateData(data)
}

// CreateWorkspace creates a new workspace
func (g *GCSBackend) CreateWorkspace(ctx context.Context, name string) error {
	if name == "" {
		return &ValidationError{Field: "name", Message: "workspace name cannot be empty"}
	}
	if name == "default" {
		return nil // Default workspace always exists
	}

	// Create workspace directory by creating a placeholder object
	workspacePrefix := g.getWorkspacePrefix(name)
	placeholderName := workspacePrefix + ".workspace"

	obj := g.bucket.Object(placeholderName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"
	writer.Metadata = map[string]string{
		"workspace": name,
		"created":   time.Now().Format(time.RFC3339),
	}

	placeholderData := fmt.Sprintf(`{"workspace": "%s", "created": "%s"}`, name, time.Now().Format(time.RFC3339))
	if _, err := writer.Write([]byte(placeholderData)); err != nil {
		writer.Close()
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	return writer.Close()
}

// GetLockInfo returns current lock information
func (g *GCSBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	lockObjectName := g.getLockObjectName()
	obj := g.bucket.Object(lockObjectName)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, nil // No lock
		}
		return nil, fmt.Errorf("failed to get lock info: %w", err)
	}

	// Parse lock data from metadata
	lockInfo := &LockInfo{
		ID:        lockObjectName,
		Path:      g.getStateObjectName(),
		Operation: attrs.Metadata["operation"],
		Who:       attrs.Metadata["who"],
		Version:   attrs.Metadata["version"],
		Created:   attrs.Created,
		Info:      attrs.Metadata["info"],
	}

	return lockInfo, nil
}

// Validate checks if the backend is properly configured and accessible
func (g *GCSBackend) Validate(ctx context.Context) error {
	// Test bucket access
	attrs, err := g.bucket.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed to access bucket %s: %w", g.config.Bucket, err)
	}

	// Test write access by creating a test object
	testObjectName := g.config.Prefix + "/.test"
	obj := g.bucket.Object(testObjectName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "text/plain"

	if _, err := writer.Write([]byte("test")); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write test object: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close test writer: %w", err)
	}

	// Clean up test object
	if err := obj.Delete(ctx); err != nil {
		// Log warning but don't fail validation
		fmt.Printf("Warning: failed to delete test object: %v\n", err)
	}

	fmt.Printf("GCS backend validation successful for bucket: %s (location: %s)\n", attrs.Name, attrs.Location)
	return nil
}

// GetMetadata returns backend metadata
func (g *GCSBackend) GetMetadata() *BackendMetadata {
	return &BackendMetadata{
		Type:               "gcs",
		SupportsLocking:    true,
		SupportsVersions:   true,
		SupportsWorkspaces: true,
		Configuration: map[string]string{
			"bucket":  g.config.Bucket,
			"prefix":  g.config.Prefix,
			"project": g.config.ProjectID,
		},
		Workspace: g.workspace,
		StateKey:  g.getStateObjectName(),
	}
}

// Close closes the GCS client
func (g *GCSBackend) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// Helper methods

// getStateObjectName returns the object name for the current workspace state
func (g *GCSBackend) getStateObjectName() string {
	if g.workspace == "default" {
		return g.config.Prefix + "/terraform.tfstate"
	}
	return g.config.Prefix + "/" + g.workspace + "/terraform.tfstate"
}

// getLockObjectName returns the object name for the lock
func (g *GCSBackend) getLockObjectName() string {
	if g.workspace == "default" {
		return g.config.Prefix + "/terraform.tfstate.lock"
	}
	return g.config.Prefix + "/" + g.workspace + "/terraform.tfstate.lock"
}

// getWorkspacePrefix returns the prefix for a workspace
func (g *GCSBackend) getWorkspacePrefix(workspace string) string {
	if workspace == "default" {
		return g.config.Prefix + "/"
	}
	return g.config.Prefix + "/" + workspace + "/"
}

// extractWorkspaceFromObjectName extracts workspace name from object name
func (g *GCSBackend) extractWorkspaceFromObjectName(objectName string) string {
	// Remove prefix
	if g.config.Prefix != "" {
		objectName = objectName[len(g.config.Prefix)+1:]
	}

	// Split by "/"
	parts := splitString(objectName, "/")
	if len(parts) >= 2 {
		return parts[0]
	}

	return "default"
}

// extractSerialFromMetadata extracts serial number from object metadata
func (g *GCSBackend) extractSerialFromMetadata(metadata map[string]string) uint64 {
	if serial, ok := metadata["serial"]; ok {
		if parsed, err := strconv.ParseUint(serial, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// parseStateData parses raw state data into StateData struct
func (g *GCSBackend) parseStateData(data []byte) (*StateData, error) {
	// This is a simplified parser - in production, you'd want more robust parsing
	stateData := &StateData{
		Data:         data,
		Resources:    []StateResource{},
		Outputs:      make(map[string]interface{}),
		LastModified: time.Now(),
		Size:         int64(len(data)),
	}

	// Try to extract basic metadata from JSON
	// In a real implementation, you'd parse the full JSON structure
	stateData.Version = 4 // Default Terraform state version
	stateData.Serial = 0  // Would be extracted from JSON

	return stateData, nil
}

// splitString splits a string by delimiter
func splitString(s, delimiter string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(delimiter) <= len(s) && s[i:i+len(delimiter)] == delimiter {
			result = append(result, s[start:i])
			start = i + len(delimiter)
			i += len(delimiter) - 1
		}
	}
	result = append(result, s[start:])
	return result
}
