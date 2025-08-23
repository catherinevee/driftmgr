package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-json"
)

// EnhancedStateParser provides comprehensive Terraform state parsing
type EnhancedStateParser struct {
	workspaceManager *WorkspaceManager
	providerVersions map[string]string
	ignorePatterns   []string
	schemaCache      map[string]*tfjson.ProviderSchema
}

// WorkspaceManager handles Terraform workspaces and remote backends
type WorkspaceManager struct {
	currentWorkspace string
	workspaces       map[string]*WorkspaceConfig
	remoteBackend    RemoteBackend
}

// WorkspaceConfig represents a Terraform workspace configuration
type WorkspaceConfig struct {
	Name         string
	IsDefault    bool
	StateFile    string
	Variables    map[string]interface{}
	Tags         map[string]string
	LastModified time.Time
}

// RemoteBackend interface for different backend types
type RemoteBackend interface {
	GetState(ctx context.Context, workspace string) (*TerraformState, error)
	ListWorkspaces(ctx context.Context) ([]string, error)
	GetWorkspaceVariables(ctx context.Context, workspace string) (map[string]interface{}, error)
	LockState(ctx context.Context, workspace string) error
	UnlockState(ctx context.Context, workspace string) error
}

// TerraformCloudBackend implements RemoteBackend for Terraform Cloud/Enterprise
type TerraformCloudBackend struct {
	client       *tfe.Client
	organization string
	workspace    string
}

// TerraformState represents a complete Terraform state
type TerraformState struct {
	Version          int                      `json:"version"`
	TerraformVersion string                   `json:"terraform_version"`
	Serial           int                      `json:"serial"`
	Lineage          string                   `json:"lineage"`
	Outputs          map[string]OutputValue   `json:"outputs"`
	Resources        []StateResource          `json:"resources"`
	CheckResults     []CheckResult            `json:"check_results"`
}

// StateResource represents a resource in Terraform state
type StateResource struct {
	Module    string                    `json:"module,omitempty"`
	Mode      string                    `json:"mode"`
	Type      string                    `json:"type"`
	Name      string                    `json:"name"`
	Provider  string                    `json:"provider"`
	Instances []StateResourceInstance   `json:"instances"`
	Each      string                    `json:"each,omitempty"`
}

// StateResourceInstance represents an instance of a resource
type StateResourceInstance struct {
	SchemaVersion       int                      `json:"schema_version"`
	Attributes          map[string]interface{}   `json:"attributes"`
	SensitiveAttributes []string                 `json:"sensitive_attributes,omitempty"`
	Private             string                   `json:"private,omitempty"`
	Dependencies        []string                 `json:"dependencies,omitempty"`
	CreateBeforeDestroy bool                     `json:"create_before_destroy,omitempty"`
	IndexKey            interface{}              `json:"index_key,omitempty"`
}

// OutputValue represents a Terraform output value
type OutputValue struct {
	Value     interface{} `json:"value"`
	Type      interface{} `json:"type,omitempty"`
	Sensitive bool        `json:"sensitive,omitempty"`
}

// CheckResult represents a Terraform check result
type CheckResult struct {
	ObjectKind string   `json:"object_kind"`
	ConfigAddr string   `json:"config_addr"`
	Status     string   `json:"status"`
	Objects    []Object `json:"objects"`
}

// Object represents an object in a check result
type Object struct {
	ObjectAddr string `json:"object_addr"`
	Status     string `json:"status"`
}

// NewEnhancedStateParser creates a new enhanced state parser
func NewEnhancedStateParser() *EnhancedStateParser {
	return &EnhancedStateParser{
		workspaceManager: &WorkspaceManager{
			currentWorkspace: "default",
			workspaces:       make(map[string]*WorkspaceConfig),
		},
		providerVersions: make(map[string]string),
		ignorePatterns:   []string{},
		schemaCache:      make(map[string]*tfjson.ProviderSchema),
	}
}

// ParseStateFile parses a Terraform state file with full attribute support
func (p *EnhancedStateParser) ParseStateFile(ctx context.Context, path string) (*TerraformState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Validate state version compatibility
	if err := p.validateStateVersion(&state); err != nil {
		return nil, err
	}

	// Process and enhance resource attributes
	if err := p.enhanceResourceAttributes(&state); err != nil {
		return nil, err
	}

	return &state, nil
}

// ParseRemoteState parses state from a remote backend
func (p *EnhancedStateParser) ParseRemoteState(ctx context.Context, backend RemoteBackend, workspace string) (*TerraformState, error) {
	// Lock state for consistency
	if err := backend.LockState(ctx, workspace); err != nil {
		return nil, fmt.Errorf("failed to lock state: %w", err)
	}
	defer backend.UnlockState(ctx, workspace)

	state, err := backend.GetState(ctx, workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote state: %w", err)
	}

	// Enhance with workspace-specific variables
	vars, err := backend.GetWorkspaceVariables(ctx, workspace)
	if err == nil {
		p.applyWorkspaceVariables(state, vars)
	}

	return state, nil
}

// ConnectToTerraformCloud connects to Terraform Cloud/Enterprise
func (p *EnhancedStateParser) ConnectToTerraformCloud(token, organization, workspace string) error {
	config := &tfe.Config{
		Token: token,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create Terraform Cloud client: %w", err)
	}

	backend := &TerraformCloudBackend{
		client:       client,
		organization: organization,
		workspace:    workspace,
	}

	p.workspaceManager.remoteBackend = backend
	return nil
}

// GetState implements RemoteBackend for Terraform Cloud
func (b *TerraformCloudBackend) GetState(ctx context.Context, workspace string) (*TerraformState, error) {
	// Get workspace
	ws, err := b.client.Workspaces.Read(ctx, b.organization, workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace: %w", err)
	}

	// Get current state version
	stateVersion, err := b.client.StateVersions.ReadCurrent(ctx, ws.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to read state version: %w", err)
	}

	// Download state file
	stateData, err := b.downloadState(ctx, stateVersion.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download state: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	return &state, nil
}

// downloadState downloads state from URL
func (b *TerraformCloudBackend) downloadState(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download state: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// ListWorkspaces implements RemoteBackend for Terraform Cloud
func (b *TerraformCloudBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	options := &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	}

	var workspaces []string
	for {
		list, err := b.client.Workspaces.List(ctx, b.organization, options)
		if err != nil {
			return nil, err
		}

		for _, ws := range list.Items {
			workspaces = append(workspaces, ws.Name)
		}

		if list.NextPage == 0 {
			break
		}
		options.PageNumber = list.NextPage
	}

	return workspaces, nil
}

// GetWorkspaceVariables implements RemoteBackend for Terraform Cloud
func (b *TerraformCloudBackend) GetWorkspaceVariables(ctx context.Context, workspace string) (map[string]interface{}, error) {
	ws, err := b.client.Workspaces.Read(ctx, b.organization, workspace)
	if err != nil {
		return nil, err
	}

	options := &tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	}

	variables := make(map[string]interface{})
	list, err := b.client.Variables.List(ctx, ws.ID, options)
	if err != nil {
		return nil, err
	}

	for _, v := range list.Items {
		if !v.Sensitive {
			variables[v.Key] = v.Value
		}
	}

	return variables, nil
}

// LockState implements RemoteBackend for Terraform Cloud
func (b *TerraformCloudBackend) LockState(ctx context.Context, workspace string) error {
	ws, err := b.client.Workspaces.Read(ctx, b.organization, workspace)
	if err != nil {
		return err
	}

	_, err = b.client.Workspaces.Lock(ctx, ws.ID, tfe.WorkspaceLockOptions{
		Reason: tfe.String("Drift detection in progress"),
	})
	return err
}

// UnlockState implements RemoteBackend for Terraform Cloud
func (b *TerraformCloudBackend) UnlockState(ctx context.Context, workspace string) error {
	ws, err := b.client.Workspaces.Read(ctx, b.organization, workspace)
	if err != nil {
		return err
	}

	_, err = b.client.Workspaces.Unlock(ctx, ws.ID)
	return err
}

// SwitchWorkspace switches to a different Terraform workspace
func (p *EnhancedStateParser) SwitchWorkspace(workspace string) error {
	if _, exists := p.workspaceManager.workspaces[workspace]; !exists {
		// Try to discover the workspace
		if err := p.discoverWorkspace(workspace); err != nil {
			return fmt.Errorf("workspace %s not found: %w", workspace, err)
		}
	}

	p.workspaceManager.currentWorkspace = workspace
	return nil
}

// discoverWorkspace discovers a workspace configuration
func (p *EnhancedStateParser) discoverWorkspace(workspace string) error {
	// Check local workspace
	workspaceDir := filepath.Join(".terraform", "terraform.tfstate.d", workspace)
	stateFile := filepath.Join(workspaceDir, "terraform.tfstate")

	if _, err := os.Stat(stateFile); err == nil {
		p.workspaceManager.workspaces[workspace] = &WorkspaceConfig{
			Name:         workspace,
			StateFile:    stateFile,
			LastModified: time.Now(),
		}
		return nil
	}

	// Check remote backend
	if p.workspaceManager.remoteBackend != nil {
		ctx := context.Background()
		workspaces, err := p.workspaceManager.remoteBackend.ListWorkspaces(ctx)
		if err != nil {
			return err
		}

		for _, ws := range workspaces {
			if ws == workspace {
				p.workspaceManager.workspaces[workspace] = &WorkspaceConfig{
					Name:         workspace,
					LastModified: time.Now(),
				}
				return nil
			}
		}
	}

	return fmt.Errorf("workspace not found")
}

// validateStateVersion validates Terraform state version compatibility
func (p *EnhancedStateParser) validateStateVersion(state *TerraformState) error {
	if state.Version < 4 {
		return fmt.Errorf("unsupported state version %d (minimum version 4 required)", state.Version)
	}

	// Track provider versions for compatibility checks
	for _, resource := range state.Resources {
		providerParts := strings.Split(resource.Provider, "/")
		if len(providerParts) >= 2 {
			provider := providerParts[len(providerParts)-2] + "/" + providerParts[len(providerParts)-1]
			p.providerVersions[provider] = state.TerraformVersion
		}
	}

	return nil
}

// enhanceResourceAttributes enhances resource attributes with type information
func (p *EnhancedStateParser) enhanceResourceAttributes(state *TerraformState) error {
	for i := range state.Resources {
		resource := &state.Resources[i]

		// Get provider schema if available
		schema := p.getProviderSchema(resource.Provider)
		if schema != nil {
			p.applySchema(resource, schema)
		}

		// Process sensitive attributes
		for j := range resource.Instances {
			instance := &resource.Instances[j]
			p.processSensitiveAttributes(instance)
		}

		// Resolve dependencies
		p.resolveDependencies(resource, state)
	}

	return nil
}

// getProviderSchema gets cached provider schema
func (p *EnhancedStateParser) getProviderSchema(provider string) *tfjson.ProviderSchema {
	return p.schemaCache[provider]
}

// applySchema applies provider schema to resource
func (p *EnhancedStateParser) applySchema(resource *StateResource, schema *tfjson.ProviderSchema) {
	// This would apply schema validation and type information
	// In production, this would use the actual provider schema
}

// processSensitiveAttributes processes sensitive attributes
func (p *EnhancedStateParser) processSensitiveAttributes(instance *StateResourceInstance) {
	// Mark sensitive attributes
	for _, attr := range instance.SensitiveAttributes {
		if _, exists := instance.Attributes[attr]; exists {
			// Replace sensitive values with placeholders
			instance.Attributes[attr] = "<sensitive>"
		}
	}
}

// resolveDependencies resolves resource dependencies
func (p *EnhancedStateParser) resolveDependencies(resource *StateResource, state *TerraformState) {
	for i := range resource.Instances {
		instance := &resource.Instances[i]
		
		// Parse dependency references
		var resolvedDeps []string
		for _, dep := range instance.Dependencies {
			// Resolve module-relative dependencies
			resolvedDep := p.resolveModulePath(dep, resource.Module)
			resolvedDeps = append(resolvedDeps, resolvedDep)
		}
		instance.Dependencies = resolvedDeps
	}
}

// resolveModulePath resolves module-relative paths
func (p *EnhancedStateParser) resolveModulePath(path, currentModule string) string {
	if strings.HasPrefix(path, "module.") {
		return path
	}
	if currentModule != "" {
		return fmt.Sprintf("%s.%s", currentModule, path)
	}
	return path
}

// applyWorkspaceVariables applies workspace-specific variables to state
func (p *EnhancedStateParser) applyWorkspaceVariables(state *TerraformState, variables map[string]interface{}) {
	// Apply workspace variables to outputs
	for key, output := range state.Outputs {
		if varValue, exists := variables[key]; exists {
			output.Value = varValue
			state.Outputs[key] = output
		}
	}
}

// ConvertToResources converts Terraform state to drift detection resources
func (p *EnhancedStateParser) ConvertToResources(state *TerraformState) []models.Resource {
	var resources []models.Resource

	for _, stateResource := range state.Resources {
		for _, instance := range stateResource.Instances {
			resource := p.convertInstanceToResource(stateResource, instance)
			resources = append(resources, resource)
		}
	}

	return resources
}

// convertInstanceToResource converts a state instance to a Resource
func (p *EnhancedStateParser) convertInstanceToResource(stateResource StateResource, instance StateResourceInstance) models.Resource {
	// Extract ID from attributes
	id := ""
	if idVal, exists := instance.Attributes["id"]; exists {
		id = fmt.Sprintf("%v", idVal)
	}

	// Extract name from attributes
	name := stateResource.Name
	if nameVal, exists := instance.Attributes["name"]; exists {
		name = fmt.Sprintf("%v", nameVal)
	}

	// Extract tags
	tags := make(map[string]interface{})
	if tagsVal, exists := instance.Attributes["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			tags = tagsMap
		}
	}

	// Extract region
	region := ""
	if regionVal, exists := instance.Attributes["region"]; exists {
		region = fmt.Sprintf("%v", regionVal)
	} else if locationVal, exists := instance.Attributes["location"]; exists {
		region = fmt.Sprintf("%v", locationVal)
	}

	// Determine provider from resource type
	provider := p.extractProviderFromType(stateResource.Provider)

	// Convert attributes to metadata (string map)
	metadata := make(map[string]string)
	for k, v := range instance.Attributes {
		metadata[k] = fmt.Sprintf("%v", v)
	}

	return models.Resource{
		ID:           id,
		Name:         name,
		Type:         stateResource.Type,
		Provider:     provider,
		Region:       region,
		State:        "managed",
		Tags:         tags,
		CreatedAt:    time.Now(),
		Updated:      time.Now(),
		Attributes:   instance.Attributes,
		Metadata:     metadata,
	}
}

// extractProviderFromType extracts provider name from provider string
func (p *EnhancedStateParser) extractProviderFromType(providerStr string) string {
	// Format: registry.terraform.io/hashicorp/aws
	parts := strings.Split(providerStr, "/")
	if len(parts) >= 1 {
		return parts[len(parts)-1]
	}
	return providerStr
}

// GetProviderVersions returns tracked provider versions
func (p *EnhancedStateParser) GetProviderVersions() map[string]string {
	return p.providerVersions
}

// AddIgnorePattern adds a pattern to ignore during state parsing
func (p *EnhancedStateParser) AddIgnorePattern(pattern string) {
	p.ignorePatterns = append(p.ignorePatterns, pattern)
}

// shouldIgnoreResource checks if a resource should be ignored
func (p *EnhancedStateParser) shouldIgnoreResource(resourceType string) bool {
	for _, pattern := range p.ignorePatterns {
		if matched, _ := filepath.Match(pattern, resourceType); matched {
			return true
		}
	}
	return false
}

// ParseHCLConfig parses HCL configuration files
func (p *EnhancedStateParser) ParseHCLConfig(path string) (*hcl.File, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file: %s", diags.Error())
	}
	return file, nil
}

// ValidateStateConsistency validates state consistency with configuration
func (p *EnhancedStateParser) ValidateStateConsistency(state *TerraformState, configPath string) ([]string, error) {
	var inconsistencies []string

	// Parse configuration
	config, err := p.ParseHCLConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Compare state resources with configuration
	// This is a simplified version - production would do deeper comparison
	_ = config

	return inconsistencies, nil
}

// ExportState exports state in various formats
func (p *EnhancedStateParser) ExportState(state *TerraformState, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(state, "", "  ")
	case "tfstate":
		return json.Marshal(state)
	case "csv":
		return p.exportStateAsCSV(state)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportStateAsCSV exports state as CSV
func (p *EnhancedStateParser) exportStateAsCSV(state *TerraformState) ([]byte, error) {
	var csv strings.Builder
	csv.WriteString("Type,Name,Provider,Module,ID\n")

	for _, resource := range state.Resources {
		for _, instance := range resource.Instances {
			id := ""
			if idVal, exists := instance.Attributes["id"]; exists {
				id = fmt.Sprintf("%v", idVal)
			}
			csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
				resource.Type,
				resource.Name,
				resource.Provider,
				resource.Module,
				id,
			))
		}
	}

	return []byte(csv.String()), nil
}