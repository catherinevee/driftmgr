package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TerraformCloudConfig represents Terraform Cloud backend configuration
type TerraformCloudConfig struct {
	Hostname       string `json:"hostname"`
	Organization   string `json:"organization"`
	Workspace      string `json:"workspace"`
	Token          string `json:"token"`
	SkipValidation bool   `json:"skip_validation"`
}

// TerraformCloudBackend implements the Backend interface for Terraform Cloud
type TerraformCloudBackend struct {
	config    *TerraformCloudConfig
	client    *http.Client
	baseURL   string
	workspace string
}

// TerraformCloudAPI represents the Terraform Cloud API structure
type TerraformCloudAPI struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			State string `json:"state"`
		} `json:"attributes"`
	} `json:"data"`
}

// TerraformCloudWorkspace represents a Terraform Cloud workspace
type TerraformCloudWorkspace struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

// TerraformCloudWorkspaces represents a list of workspaces
type TerraformCloudWorkspaces struct {
	Data []TerraformCloudWorkspace `json:"data"`
}

// NewTerraformCloudBackend creates a new Terraform Cloud backend
func NewTerraformCloudBackend(config *BackendConfig) (Backend, error) {
	if config == nil {
		return nil, &ValidationError{Field: "config", Message: "config cannot be nil"}
	}

	// Extract Terraform Cloud-specific configuration
	tcConfig := &TerraformCloudConfig{
		Hostname:       getStringFromConfig(config.Config, "hostname"),
		Organization:   getStringFromConfig(config.Config, "organization"),
		Workspace:      getStringFromConfig(config.Config, "workspace"),
		Token:          getStringFromConfig(config.Config, "token"),
		SkipValidation: getBoolFromConfig(config.Config, "skip_validation"),
	}

	// Set default hostname if not provided
	if tcConfig.Hostname == "" {
		tcConfig.Hostname = "app.terraform.io"
	}

	// Validate required fields
	if tcConfig.Organization == "" {
		return nil, &ValidationError{Field: "organization", Message: "organization is required"}
	}
	if tcConfig.Workspace == "" {
		return nil, &ValidationError{Field: "workspace", Message: "workspace is required"}
	}
	if tcConfig.Token == "" {
		return nil, &ValidationError{Field: "token", Message: "token is required"}
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Build base URL
	baseURL := fmt.Sprintf("https://%s/api/v2", tcConfig.Hostname)

	backend := &TerraformCloudBackend{
		config:    tcConfig,
		client:    client,
		baseURL:   baseURL,
		workspace: tcConfig.Workspace,
	}

	// Validate connection if not skipped
	if !tcConfig.SkipValidation {
		if err := backend.validateConnection(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to validate Terraform Cloud connection: %w", err)
		}
	}

	return backend, nil
}

// Pull retrieves the current state from Terraform Cloud
func (tc *TerraformCloudBackend) Pull(ctx context.Context) (*StateData, error) {
	// Get workspace ID
	workspaceID, err := tc.getWorkspaceID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace ID: %w", err)
	}

	// Get current state
	url := fmt.Sprintf("%s/workspaces/%s/current-state-version", tc.baseURL, workspaceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Return empty state if no state exists
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get state: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var apiResp TerraformCloudAPI
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Download state data
	stateData, err := tc.downloadStateData(ctx, apiResp.Data.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to download state data: %w", err)
	}

	return stateData, nil
}

// Push stores the state to Terraform Cloud
func (tc *TerraformCloudBackend) Push(ctx context.Context, stateData *StateData) error {
	if stateData == nil {
		return &ValidationError{Field: "stateData", Message: "state data cannot be nil"}
	}

	// Get workspace ID
	workspaceID, err := tc.getWorkspaceID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get workspace ID: %w", err)
	}

	// Create state version
	stateVersionID, err := tc.createStateVersion(ctx, workspaceID, stateData)
	if err != nil {
		return fmt.Errorf("failed to create state version: %w", err)
	}

	// Upload state data
	if err := tc.uploadStateData(ctx, stateVersionID, stateData.Data); err != nil {
		return fmt.Errorf("failed to upload state data: %w", err)
	}

	return nil
}

// Lock acquires a lock on the state
func (tc *TerraformCloudBackend) Lock(ctx context.Context, lockInfo *LockInfo) (string, error) {
	if lockInfo == nil {
		return "", &ValidationError{Field: "lockInfo", Message: "lock info cannot be nil"}
	}

	// Terraform Cloud handles locking automatically through its API
	// We'll return a mock lock ID for compatibility
	lockID := fmt.Sprintf("tc-lock-%d", time.Now().UnixNano())
	return lockID, nil
}

// Unlock releases a lock on the state
func (tc *TerraformCloudBackend) Unlock(ctx context.Context, lockID string) error {
	// Terraform Cloud handles unlocking automatically
	// No action needed for compatibility
	return nil
}

// ListWorkspaces returns all available workspaces
func (tc *TerraformCloudBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	// List workspaces in the organization
	url := fmt.Sprintf("%s/organizations/%s/workspaces", tc.baseURL, tc.config.Organization)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list workspaces: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var workspacesResp TerraformCloudWorkspaces
	if err := json.NewDecoder(resp.Body).Decode(&workspacesResp); err != nil {
		return nil, fmt.Errorf("failed to parse workspaces response: %w", err)
	}

	// Extract workspace names
	var workspaces []string
	for _, ws := range workspacesResp.Data {
		workspaces = append(workspaces, ws.Data.Attributes.Name)
	}

	return workspaces, nil
}

// SelectWorkspace selects a workspace
func (tc *TerraformCloudBackend) SelectWorkspace(ctx context.Context, workspace string) error {
	if workspace == "" {
		workspace = tc.config.Workspace
	}

	// Validate workspace exists
	workspaces, err := tc.ListWorkspaces(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate workspace: %w", err)
	}

	found := false
	for _, ws := range workspaces {
		if ws == workspace {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("workspace %s not found", workspace)
	}

	tc.workspace = workspace
	return nil
}

// DeleteWorkspace deletes a workspace
func (tc *TerraformCloudBackend) DeleteWorkspace(ctx context.Context, workspace string) error {
	if workspace == "" || workspace == tc.config.Workspace {
		return fmt.Errorf("cannot delete current workspace")
	}

	// Get workspace ID
	workspaceID, err := tc.getWorkspaceIDByName(ctx, workspace)
	if err != nil {
		return fmt.Errorf("failed to get workspace ID: %w", err)
	}

	// Delete workspace
	url := fmt.Sprintf("%s/workspaces/%s", tc.baseURL, workspaceID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete workspace: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// GetVersions returns all versions of the state
func (tc *TerraformCloudBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	// Get workspace ID
	workspaceID, err := tc.getWorkspaceID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace ID: %w", err)
	}

	// List state versions
	url := fmt.Sprintf("%s/workspaces/%s/state-versions", tc.baseURL, workspaceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list state versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list state versions: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response and convert to StateVersion format
	// This is a simplified implementation - you'd want to parse the full response
	var versions []*StateVersion
	versions = append(versions, &StateVersion{
		VersionID: "current",
		Serial:    0,
		Created:   time.Now(),
		Checksum:  "current",
	})

	return versions, nil
}

// GetVersion retrieves a specific version of the state
func (tc *TerraformCloudBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	// Get specific state version
	url := fmt.Sprintf("%s/state-versions/%s", tc.baseURL, versionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get state version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get state version: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var apiResp TerraformCloudAPI
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Download state data
	return tc.downloadStateData(ctx, apiResp.Data.ID)
}

// CreateWorkspace creates a new workspace
func (tc *TerraformCloudBackend) CreateWorkspace(ctx context.Context, name string) error {
	if name == "" {
		return &ValidationError{Field: "name", Message: "workspace name cannot be empty"}
	}

	// Create workspace payload
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workspaces",
			"attributes": map[string]interface{}{
				"name": name,
			},
			"relationships": map[string]interface{}{
				"organization": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "organizations",
						"id":   tc.config.Organization,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create workspace
	url := fmt.Sprintf("%s/organizations/%s/workspaces", tc.baseURL, tc.config.Organization)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create workspace: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// GetLockInfo returns current lock information
func (tc *TerraformCloudBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	// Terraform Cloud handles locking internally
	// Return nil to indicate no external lock management
	return nil, nil
}

// Validate checks if the backend is properly configured and accessible
func (tc *TerraformCloudBackend) Validate(ctx context.Context) error {
	// Test API access by getting organization info
	url := fmt.Sprintf("%s/organizations/%s", tc.baseURL, tc.config.Organization)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to validate connection: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Validate workspace exists
	workspaceID, err := tc.getWorkspaceID(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate workspace: %w", err)
	}

	fmt.Printf("Terraform Cloud backend validation successful for workspace: %s (ID: %s)\n", tc.workspace, workspaceID)
	return nil
}

// GetMetadata returns backend metadata
func (tc *TerraformCloudBackend) GetMetadata() *BackendMetadata {
	return &BackendMetadata{
		Type:               "remote",
		SupportsLocking:    true,
		SupportsVersions:   true,
		SupportsWorkspaces: true,
		Configuration: map[string]string{
			"hostname":     tc.config.Hostname,
			"organization": tc.config.Organization,
			"workspace":    tc.workspace,
		},
		Workspace: tc.workspace,
		StateKey:  tc.workspace,
	}
}

// Close closes the Terraform Cloud backend
func (tc *TerraformCloudBackend) Close() error {
	// No resources to close for HTTP client
	return nil
}

// Helper methods

// validateConnection validates the connection to Terraform Cloud
func (tc *TerraformCloudBackend) validateConnection(ctx context.Context) error {
	// Test API access by getting organization info
	url := fmt.Sprintf("%s/organizations/%s", tc.baseURL, tc.config.Organization)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to validate connection: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// getWorkspaceID gets the workspace ID for the current workspace
func (tc *TerraformCloudBackend) getWorkspaceID(ctx context.Context) (string, error) {
	return tc.getWorkspaceIDByName(ctx, tc.workspace)
}

// getWorkspaceIDByName gets the workspace ID by workspace name
func (tc *TerraformCloudBackend) getWorkspaceIDByName(ctx context.Context, workspaceName string) (string, error) {
	// Get workspace by name
	url := fmt.Sprintf("%s/organizations/%s/workspaces/%s", tc.baseURL, tc.config.Organization, workspaceName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get workspace: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var workspaceResp TerraformCloudWorkspace
	if err := json.NewDecoder(resp.Body).Decode(&workspaceResp); err != nil {
		return "", fmt.Errorf("failed to parse workspace response: %w", err)
	}

	return workspaceResp.Data.ID, nil
}

// createStateVersion creates a new state version
func (tc *TerraformCloudBackend) createStateVersion(ctx context.Context, workspaceID string, stateData *StateData) (string, error) {
	// Create state version payload
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "state-versions",
			"attributes": map[string]interface{}{
				"serial": stateData.Serial,
			},
			"relationships": map[string]interface{}{
				"workspace": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "workspaces",
						"id":   workspaceID,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create state version
	url := fmt.Sprintf("%s/state-versions", tc.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create state version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create state version: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response to get state version ID
	var apiResp TerraformCloudAPI
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return apiResp.Data.ID, nil
}

// uploadStateData uploads state data to a state version
func (tc *TerraformCloudBackend) uploadStateData(ctx context.Context, stateVersionID string, data []byte) error {
	// Upload state data
	url := fmt.Sprintf("%s/state-versions/%s/upload", tc.baseURL, stateVersionID)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload state data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload state data: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// downloadStateData downloads state data from a state version
func (tc *TerraformCloudBackend) downloadStateData(ctx context.Context, stateVersionID string) (*StateData, error) {
	// Download state data
	url := fmt.Sprintf("%s/state-versions/%s/download", tc.baseURL, stateVersionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tc.config.Token)

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download state data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to download state data: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Read data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}

	// Parse state data
	stateData := &StateData{
		Data:         data,
		Resources:    []StateResource{},
		Outputs:      make(map[string]interface{}),
		LastModified: time.Now(),
		Size:         int64(len(data)),
	}

	// Try to extract basic metadata from JSON
	stateData.Version = 4 // Default Terraform state version
	stateData.Serial = 0  // Would be extracted from JSON

	return stateData, nil
}
