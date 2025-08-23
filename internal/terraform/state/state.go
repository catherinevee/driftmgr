package state

import (
	"encoding/json"
	"fmt"
	"os"
)

// State represents a Terraform state file
type State struct {
	Version          int        `json:"version"`
	TerraformVersion string     `json:"terraform_version"`
	Serial           int        `json:"serial"`
	Lineage          string     `json:"lineage"`
	Resources        []Resource `json:"resources"`
}

// Resource represents a resource in the state file
type Resource struct {
	Module    string     `json:"module,omitempty"`
	Mode      string     `json:"mode"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Provider  string     `json:"provider"`
	Instances []Instance `json:"instances"`
}

// Instance represents an instance of a resource
type Instance struct {
	SchemaVersion int                    `json:"schema_version"`
	Attributes    map[string]interface{} `json:"attributes"`
	Private       string                 `json:"private,omitempty"`
	Dependencies  []string               `json:"dependencies,omitempty"`
}

// LoadStateFile loads a Terraform state file
func LoadStateFile(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// GetResourceCount returns the total number of resources in the state
func (s *State) GetResourceCount() int {
	count := 0
	for _, resource := range s.Resources {
		count += len(resource.Instances)
	}
	return count
}

// GetProviders returns a list of unique providers used in the state
func (s *State) GetProviders() []string {
	providerMap := make(map[string]bool)
	for _, resource := range s.Resources {
		providerMap[resource.Provider] = true
	}

	providers := make([]string, 0, len(providerMap))
	for provider := range providerMap {
		providers = append(providers, provider)
	}
	return providers
}

// GetResourcesByType returns resources filtered by type
func (s *State) GetResourcesByType(resourceType string) []Resource {
	var filtered []Resource
	for _, resource := range s.Resources {
		if resource.Type == resourceType {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// GetResourcesByProvider returns resources filtered by provider
func (s *State) GetResourcesByProvider(provider string) []Resource {
	var filtered []Resource
	for _, resource := range s.Resources {
		if resource.Provider == provider {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// StateFile is an alias for State to maintain compatibility
type StateFile = State
