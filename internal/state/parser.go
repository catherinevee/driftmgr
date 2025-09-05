package state

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// StateFile represents a Terraform state file
type StateFile struct {
	*TerraformState
	Path string `json:"path,omitempty"`
}

// State is an alias for TerraformState
type State = TerraformState

// StateParser handles parsing of Terraform state files
type StateParser struct{}

// NewStateParser creates a new state parser
func NewStateParser() *StateParser {
	return &StateParser{}
}

// ParseFile parses a Terraform state file
func (p *StateParser) ParseFile(path string) (*StateFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	stateFile, err := p.Parse(data)
	if err != nil {
		return nil, err
	}
	stateFile.Path = path
	return stateFile, nil
}

// Parse parses Terraform state data
func (p *StateParser) Parse(data []byte) (*StateFile, error) {
	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	return &StateFile{
		TerraformState: &state,
	}, nil
}

// TerraformState represents a complete Terraform state file
type TerraformState struct {
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
	Outputs          map[string]OutputValue `json:"outputs,omitempty"`
	Resources        []Resource             `json:"resources"`
	CheckResults     []CheckResult          `json:"check_results,omitempty"`
	Modules          []Module               `json:"modules,omitempty"`
}

// Module represents a module in the state
type Module struct {
	Path      []string               `json:"path"`
	Outputs   map[string]OutputValue `json:"outputs,omitempty"`
	Resources map[string]Resource    `json:"resources,omitempty"`
}

// Resource represents a resource in the state
type Resource struct {
	ID        string     `json:"id,omitempty"`
	Module    string     `json:"module,omitempty"`
	Mode      string     `json:"mode"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Provider  string     `json:"provider"`
	Instances []Instance `json:"instances"`
	DependsOn []string   `json:"depends_on,omitempty"`
	EachMode  string     `json:"each,omitempty"`
}

// Instance represents an instance of a resource
type Instance struct {
	SchemaVersion       int                    `json:"schema_version"`
	Attributes          map[string]interface{} `json:"attributes,omitempty"`
	AttributesFlat      map[string]string      `json:"attributes_flat,omitempty"`
	SensitiveAttributes []string               `json:"sensitive_attributes,omitempty"`
	Private             string                 `json:"private,omitempty"`
	Dependencies        []string               `json:"dependencies,omitempty"`
	CreateBeforeDestroy bool                   `json:"create_before_destroy,omitempty"`
	IndexKey            interface{}            `json:"index_key,omitempty"`
	Status              string                 `json:"status,omitempty"`
	StatusReason        string                 `json:"status_reason,omitempty"`
}

// OutputValue represents an output value in the state
type OutputValue struct {
	Value     interface{} `json:"value"`
	Type      interface{} `json:"type,omitempty"`
	Sensitive bool        `json:"sensitive,omitempty"`
}

// CheckResult represents a check result in the state
type CheckResult struct {
	ObjectKind string        `json:"object_kind"`
	ConfigAddr string        `json:"config_addr"`
	Status     string        `json:"status"`
	Objects    []CheckObject `json:"objects,omitempty"`
}

// CheckObject represents an object in a check result
type CheckObject struct {
	ObjectAddr string `json:"object_addr"`
	Status     string `json:"status"`
}

// Parser handles parsing of Terraform state files
type Parser struct {
	supportedVersions map[int]bool
}

// NewParser creates a new state parser
func NewParser() *Parser {
	return &Parser{
		supportedVersions: map[int]bool{
			3: true, // Terraform 0.11.x
			4: true, // Terraform 0.12.x - 1.x.x
		},
	}
}

// Parse parses raw state data into a TerraformState structure
func (p *Parser) Parse(data []byte) (*TerraformState, error) {
	var state TerraformState

	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Validate version
	if !p.supportedVersions[state.Version] {
		return nil, fmt.Errorf("unsupported state version: %d", state.Version)
	}

	// Normalize the state based on version
	if err := p.normalizeState(&state); err != nil {
		return nil, fmt.Errorf("failed to normalize state: %w", err)
	}

	return &state, nil
}

// ParseLegacy parses legacy Terraform state formats (< 0.11)
func (p *Parser) ParseLegacy(data []byte) (*TerraformState, error) {
	// Try to parse as version 2 state
	var legacyState struct {
		Version int `json:"version"`
		Modules []struct {
			Path      []string               `json:"path"`
			Outputs   map[string]OutputValue `json:"outputs"`
			Resources map[string]struct {
				Type      string                 `json:"type"`
				Primary   map[string]interface{} `json:"primary"`
				Provider  string                 `json:"provider"`
				DependsOn []string               `json:"depends_on"`
			} `json:"resources"`
		} `json:"modules"`
		Serial  int    `json:"serial"`
		Lineage string `json:"lineage"`
	}

	if err := json.Unmarshal(data, &legacyState); err != nil {
		return nil, fmt.Errorf("failed to parse legacy state: %w", err)
	}

	// Convert to modern format
	state := &TerraformState{
		Version:   4,
		Serial:    legacyState.Serial,
		Lineage:   legacyState.Lineage,
		Resources: make([]Resource, 0),
		Outputs:   make(map[string]OutputValue),
	}

	// Convert modules to resources
	for _, module := range legacyState.Modules {
		modulePath := strings.Join(module.Path, ".")

		// Add outputs
		for k, v := range module.Outputs {
			outputKey := k
			if modulePath != "" && modulePath != "root" {
				outputKey = modulePath + "." + k
			}
			state.Outputs[outputKey] = v
		}

		// Add resources
		for name, res := range module.Resources {
			parts := strings.SplitN(name, ".", 2)
			if len(parts) != 2 {
				continue
			}

			resource := Resource{
				Module:    modulePath,
				Mode:      "managed",
				Type:      res.Type,
				Name:      parts[1],
				Provider:  res.Provider,
				DependsOn: res.DependsOn,
				Instances: []Instance{
					{
						SchemaVersion: 0,
						Attributes:    res.Primary,
					},
				},
			}

			state.Resources = append(state.Resources, resource)
		}
	}

	return state, nil
}

// normalizeState normalizes the state structure across different versions
func (p *Parser) normalizeState(state *TerraformState) error {
	// Normalize provider names
	for i := range state.Resources {
		state.Resources[i].Provider = p.normalizeProviderName(state.Resources[i].Provider)

		// Ensure instances have proper structure
		for j := range state.Resources[i].Instances {
			if state.Resources[i].Instances[j].Attributes == nil {
				state.Resources[i].Instances[j].Attributes = make(map[string]interface{})
			}
		}
	}

	return nil
}

// normalizeProviderName normalizes provider names to a consistent format
func (p *Parser) normalizeProviderName(provider string) string {
	// Remove registry prefix if present
	if strings.Contains(provider, "registry.terraform.io/") {
		parts := strings.Split(provider, "/")
		if len(parts) >= 3 {
			provider = parts[len(parts)-1]
		}
	}

	// Remove version suffix if present
	if idx := strings.LastIndex(provider, "."); idx > 0 {
		provider = provider[:idx]
	}

	// Handle aliased providers
	parts := strings.Split(provider, ".")
	if len(parts) > 1 {
		return parts[0]
	}

	return provider
}

// GetResourceByAddress returns a resource by its address
func (p *Parser) GetResourceByAddress(state *TerraformState, address string) (*Resource, *Instance, error) {
	// Parse address (e.g., "aws_instance.example[0]")
	parts := strings.Split(address, ".")
	if len(parts) < 2 {
		return nil, nil, fmt.Errorf("invalid resource address: %s", address)
	}

	resourceType := parts[0]
	resourceName := strings.Split(parts[1], "[")[0]

	// Extract index if present
	var index int
	if strings.Contains(parts[1], "[") {
		fmt.Sscanf(parts[1], "%*[^[][%d]", &index)
	}

	// Find resource
	for _, resource := range state.Resources {
		if resource.Type == resourceType && resource.Name == resourceName {
			if index < len(resource.Instances) {
				return &resource, &resource.Instances[index], nil
			}
			return &resource, nil, fmt.Errorf("instance index %d not found", index)
		}
	}

	return nil, nil, fmt.Errorf("resource not found: %s", address)
}

// ExtractProviders extracts unique providers from the state
func (p *Parser) ExtractProviders(state *TerraformState) []string {
	providers := make(map[string]bool)

	for _, resource := range state.Resources {
		provider := p.normalizeProviderName(resource.Provider)
		providers[provider] = true
	}

	result := make([]string, 0, len(providers))
	for provider := range providers {
		result = append(result, provider)
	}

	return result
}

// GetResourceCount returns the total number of resource instances
func (p *Parser) GetResourceCount(state *TerraformState) int {
	count := 0
	for _, resource := range state.Resources {
		count += len(resource.Instances)
	}
	return count
}

// GetResourcesByType returns all resources of a specific type
func (p *Parser) GetResourcesByType(state *TerraformState, resourceType string) []Resource {
	var resources []Resource

	for _, resource := range state.Resources {
		if resource.Type == resourceType {
			resources = append(resources, resource)
		}
	}

	return resources
}

// GetResourcesByProvider returns all resources for a specific provider
func (p *Parser) GetResourcesByProvider(state *TerraformState, provider string) []Resource {
	normalizedProvider := p.normalizeProviderName(provider)
	var resources []Resource

	for _, resource := range state.Resources {
		if p.normalizeProviderName(resource.Provider) == normalizedProvider {
			resources = append(resources, resource)
		}
	}

	return resources
}

// ValidateState performs basic validation on the state
func (p *Parser) ValidateState(state *TerraformState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	if state.Version == 0 {
		return fmt.Errorf("invalid state version: 0")
	}

	if state.Lineage == "" {
		return fmt.Errorf("state lineage is empty")
	}

	// Check for duplicate resource addresses
	addresses := make(map[string]bool)
	for _, resource := range state.Resources {
		for i := range resource.Instances {
			address := fmt.Sprintf("%s.%s[%d]", resource.Type, resource.Name, i)
			if len(resource.Instances) == 1 {
				address = fmt.Sprintf("%s.%s", resource.Type, resource.Name)
			}

			if addresses[address] {
				return fmt.Errorf("duplicate resource address: %s", address)
			}
			addresses[address] = true
		}
	}

	return nil
}

// MergeStates merges multiple state files into one
func (p *Parser) MergeStates(states ...*TerraformState) (*TerraformState, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("no states to merge")
	}

	// Use first state as base
	merged := &TerraformState{
		Version:          states[0].Version,
		TerraformVersion: states[0].TerraformVersion,
		Serial:           states[0].Serial,
		Lineage:          states[0].Lineage,
		Resources:        make([]Resource, 0),
		Outputs:          make(map[string]OutputValue),
	}

	// Track resource addresses to avoid duplicates
	resourceMap := make(map[string]bool)

	// Merge all states
	for _, state := range states {
		if state == nil {
			continue
		}

		// Merge resources
		for _, resource := range state.Resources {
			address := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
			if !resourceMap[address] {
				merged.Resources = append(merged.Resources, resource)
				resourceMap[address] = true
			}
		}

		// Merge outputs
		for k, v := range state.Outputs {
			merged.Outputs[k] = v
		}

		// Update serial to highest
		if state.Serial > merged.Serial {
			merged.Serial = state.Serial
		}
	}

	return merged, nil
}
