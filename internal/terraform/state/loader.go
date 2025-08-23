package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// StateLoader loads Terraform state files
type StateLoader struct {
	path string
}

// NewStateLoader creates a new state loader
func NewStateLoader(path string) *StateLoader {
	return &StateLoader{
		path: path,
	}
}

// Load loads the state file
func (l *StateLoader) Load() (*TerraformState, error) {
	data, err := ioutil.ReadFile(l.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// LoadStateFile loads a state file with context support
func (l *StateLoader) LoadStateFile(ctx context.Context, path string, options interface{}) (*State, error) {
	if path == "" {
		path = l.path
	}
	return LoadStateFile(path)
}

// TerraformState represents a Terraform state
type TerraformState struct {
	Version          int             `json:"version"`
	TerraformVersion string          `json:"terraform_version"`
	Serial           int             `json:"serial"`
	Lineage          string          `json:"lineage"`
	Resources        []StateResource `json:"resources"`
}

// StateResource represents a resource in the state
type StateResource struct {
	Module    string             `json:"module,omitempty"`
	Mode      string             `json:"mode"`
	Type      string             `json:"type"`
	Name      string             `json:"name"`
	Provider  string             `json:"provider"`
	Instances []ResourceInstance `json:"instances"`
}

// ResourceInstance represents an instance of a resource
type ResourceInstance struct {
	SchemaVersion int                    `json:"schema_version"`
	Attributes    map[string]interface{} `json:"attributes"`
	Private       string                 `json:"private,omitempty"`
}

// GetResourceCount returns the total count of resources
func (s *TerraformState) GetResourceCount() int {
	count := 0
	for _, r := range s.Resources {
		count += len(r.Instances)
	}
	return count
}

// GetResourcesByType returns resources of a specific type
func (s *TerraformState) GetResourcesByType(resourceType string) []StateResource {
	var resources []StateResource
	for _, r := range s.Resources {
		if r.Type == resourceType {
			resources = append(resources, r)
		}
	}
	return resources
}
