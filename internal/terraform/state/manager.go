package state

import (
	"context"
	"fmt"
	"sync"

	"github.com/catherinevee/driftmgr/internal/providers/cloud"
)

// Manager manages Terraform state operations
type Manager struct {
	loader    *Loader
	analyzer  *Analyzer
	discovery *DiscoveryService
	mu        sync.RWMutex
	states    map[string]*State
}

// NewManager creates a new state manager
func NewManager() *Manager {
	return &Manager{
		loader:    NewLoader(),
		analyzer:  NewAnalyzer(),
		discovery: NewDiscoveryService(),
		states:    make(map[string]*State),
	}
}

// LoadState loads a Terraform state file
func (m *Manager) LoadState(ctx context.Context, path string) (*State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	state, err := m.loader.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	
	m.states[path] = state
	return state, nil
}

// GetState returns a loaded state
func (m *Manager) GetState(path string) (*State, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	state, exists := m.states[path]
	return state, exists
}

// AnalyzeState analyzes a Terraform state
func (m *Manager) AnalyzeState(ctx context.Context, state *State) (*AnalysisResult, error) {
	options := AnalysisOptions{
		CheckDrift:    true,
		CheckCoverage: true,
		CheckOrphans:  true,
	}
	return m.analyzer.AnalyzeState(ctx, state, options)
}

// DiscoverResources discovers resources from a state
func (m *Manager) DiscoverResources(ctx context.Context, state *State) ([]cloud.Resource, error) {
	if state == nil || state.Resources == nil {
		return []cloud.Resource{}, nil
	}
	
	var resources []cloud.Resource
	for _, r := range state.Resources {
		if r.Type != "" && r.Name != "" {
			resources = append(resources, cloud.Resource{
				ID:       r.Name,
				Type:     r.Type,
				Provider: r.Provider,
				Region:   "", // Would need to extract from attributes
				Tags:     make(map[string]string),
			})
		}
	}
	
	return resources, nil
}

// CompareStates compares two states
func (m *Manager) CompareStates(state1, state2 *State) (*StateComparison, error) {
	// Simple comparison implementation
	resources1 := make(map[string]cloud.Resource)
	resources2 := make(map[string]cloud.Resource)
	
	if state1 != nil && state1.Resources != nil {
		for _, r := range state1.Resources {
			if r.Type != "" && r.Name != "" {
				key := fmt.Sprintf("%s.%s", r.Type, r.Name)
				resources1[key] = cloud.Resource{
					ID:   r.Name,
					Type: r.Type,
				}
			}
		}
	}
	
	if state2 != nil && state2.Resources != nil {
		for _, r := range state2.Resources {
			if r.Type != "" && r.Name != "" {
				key := fmt.Sprintf("%s.%s", r.Type, r.Name)
				resources2[key] = cloud.Resource{
					ID:   r.Name,
					Type: r.Type,
				}
			}
		}
	}
	
	comparison := &StateComparison{
		Added:    []cloud.Resource{},
		Removed:  []cloud.Resource{},
		Modified: []cloud.Resource{},
		Same:     []cloud.Resource{},
	}
	
	// Find added and same resources
	for key, r2 := range resources2 {
		if _, exists := resources1[key]; !exists {
			comparison.Added = append(comparison.Added, r2)
		} else {
			comparison.Same = append(comparison.Same, r2)
		}
	}
	
	// Find removed resources
	for key, r1 := range resources1 {
		if _, exists := resources2[key]; !exists {
			comparison.Removed = append(comparison.Removed, r1)
		}
	}
	
	return comparison, nil
}

// ListStates returns all loaded states
func (m *Manager) ListStates() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var paths []string
	for path := range m.states {
		paths = append(paths, path)
	}
	return paths
}

// ClearStates removes all loaded states
func (m *Manager) ClearStates() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.states = make(map[string]*State)
}

// StateComparison represents a comparison between two states
type StateComparison struct {
	Added    []cloud.Resource `json:"added"`
	Removed  []cloud.Resource `json:"removed"`
	Modified []cloud.Resource `json:"modified"`
	Same     []cloud.Resource `json:"same"`
}