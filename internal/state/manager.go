package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Backend interface for state storage backends
type Backend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, data []byte) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
	ListStates(ctx context.Context) ([]string, error)
	ListStateVersions(ctx context.Context, key string) ([]StateVersion, error)
	GetStateVersion(ctx context.Context, key string, version int) ([]byte, error)
}

// StateManager manages Terraform state operations
type StateManager struct {
	backend   Backend
	parser    *Parser
	validator *Validator
	cache     *StateCache
	mu        sync.RWMutex
}

// StateMetadata contains metadata about a state
type StateMetadata struct {
	Key           string    `json:"key"`
	Version       int       `json:"version"`
	Serial        int       `json:"serial"`
	Lineage       string    `json:"lineage"`
	LastModified  time.Time `json:"last_modified"`
	Size          int64     `json:"size"`
	Checksum      string    `json:"checksum"`
	ResourceCount int       `json:"resource_count"`
}

// StateVersion represents a version of a state file
type StateVersion struct {
	Version   int       `json:"version"`
	Serial    int       `json:"serial"`
	Timestamp time.Time `json:"timestamp"`
	Checksum  string    `json:"checksum"`
}

// NewStateManager creates a new state manager
func NewStateManager(backend Backend) *StateManager {
	return &StateManager{
		backend:   backend,
		parser:    NewParser(),
		validator: NewValidator(),
		cache:     NewStateCache(1 * time.Hour),
	}
}

// GetState retrieves and parses a state file
func (sm *StateManager) GetState(ctx context.Context, key string) (*TerraformState, error) {
	// Check cache first
	if cached := sm.cache.Get(key); cached != nil {
		return cached, nil
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Get from backend
	data, err := sm.backend.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get state from backend: %w", err)
	}

	// Parse state
	state, err := sm.parser.Parse(data)
	if err != nil {
		// Try legacy format
		state, err = sm.parser.ParseLegacy(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse state: %w", err)
		}
	}

	// Validate state
	if err := sm.validator.Validate(state); err != nil {
		return nil, fmt.Errorf("state validation failed: %w", err)
	}

	// Cache the state
	sm.cache.Set(key, state)

	return state, nil
}

// UpdateState updates a state file
func (sm *StateManager) UpdateState(ctx context.Context, key string, state *TerraformState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate before saving
	if err := sm.validator.Validate(state); err != nil {
		return fmt.Errorf("state validation failed: %w", err)
	}

	// Increment serial
	state.Serial++

	// Lock state
	if err := sm.backend.Lock(ctx, key); err != nil {
		return fmt.Errorf("failed to lock state: %w", err)
	}
	defer sm.backend.Unlock(ctx, key)

	// Serialize state
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Upload to backend
	if err := sm.backend.Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to upload state: %w", err)
	}

	// Invalidate cache
	sm.cache.Delete(key)

	return nil
}

// DeleteState removes a state file
func (sm *StateManager) DeleteState(ctx context.Context, key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Lock state
	if err := sm.backend.Lock(ctx, key); err != nil {
		return fmt.Errorf("failed to lock state: %w", err)
	}
	defer sm.backend.Unlock(ctx, key)

	// Delete from backend
	if err := sm.backend.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	// Remove from cache
	sm.cache.Delete(key)

	return nil
}

// ListStates lists all available states
func (sm *StateManager) ListStates(ctx context.Context) ([]string, error) {
	return sm.backend.ListStates(ctx)
}

// GetStateMetadata retrieves metadata about a state without parsing the full state
func (sm *StateManager) GetStateMetadata(ctx context.Context, key string) (*StateMetadata, error) {
	data, err := sm.backend.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Parse just the metadata fields
	var meta struct {
		Version          int               `json:"version"`
		Serial           int               `json:"serial"`
		Lineage          string            `json:"lineage"`
		TerraformVersion string            `json:"terraform_version"`
		Resources        []json.RawMessage `json:"resources"`
	}

	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse state metadata: %w", err)
	}

	// Calculate checksum
	h := sha256.Sum256(data)
	checksum := hex.EncodeToString(h[:])

	return &StateMetadata{
		Key:           key,
		Version:       meta.Version,
		Serial:        meta.Serial,
		Lineage:       meta.Lineage,
		LastModified:  time.Now(), // Would be better to get from backend
		Size:          int64(len(data)),
		Checksum:      checksum,
		ResourceCount: len(meta.Resources),
	}, nil
}

// GetStateHistory retrieves the version history of a state
func (sm *StateManager) GetStateHistory(ctx context.Context, key string) ([]StateVersion, error) {
	return sm.backend.ListStateVersions(ctx, key)
}

// RestoreStateVersion restores a specific version of a state
func (sm *StateManager) RestoreStateVersion(ctx context.Context, key string, version int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Get the specific version
	data, err := sm.backend.GetStateVersion(ctx, key, version)
	if err != nil {
		return fmt.Errorf("failed to get state version: %w", err)
	}

	// Parse to verify it's valid
	state, err := sm.parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse historical state: %w", err)
	}

	// Increment serial for the restore
	state.Serial++

	// Lock state
	if err := sm.backend.Lock(ctx, key); err != nil {
		return fmt.Errorf("failed to lock state: %w", err)
	}
	defer sm.backend.Unlock(ctx, key)

	// Serialize and upload
	newData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	if err := sm.backend.Put(ctx, key, newData); err != nil {
		return fmt.Errorf("failed to restore state: %w", err)
	}

	// Invalidate cache
	sm.cache.Delete(key)

	return nil
}

// ImportResource adds a new resource to the state
func (sm *StateManager) ImportResource(ctx context.Context, key string, resource Resource) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Get current state
	state, err := sm.GetState(ctx, key)
	if err != nil {
		// If state doesn't exist, create new one
		state = &TerraformState{
			Version:   4,
			Serial:    0,
			Lineage:   generateLineage(),
			Resources: []Resource{},
			Outputs:   make(map[string]OutputValue),
		}
	}

	// Check if resource already exists
	for _, existingResource := range state.Resources {
		if existingResource.Type == resource.Type && existingResource.Name == resource.Name {
			return fmt.Errorf("resource %s.%s already exists in state", resource.Type, resource.Name)
		}
	}

	// Add resource
	state.Resources = append(state.Resources, resource)

	// Update state
	return sm.UpdateState(ctx, key, state)
}

// RemoveResource removes a resource from the state
func (sm *StateManager) RemoveResource(ctx context.Context, key string, resourceAddress string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Get current state
	state, err := sm.GetState(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Find and remove resource
	found := false
	newResources := make([]Resource, 0, len(state.Resources)-1)

	for _, resource := range state.Resources {
		address := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		if address == resourceAddress {
			found = true
			continue
		}
		newResources = append(newResources, resource)
	}

	if !found {
		return fmt.Errorf("resource %s not found in state", resourceAddress)
	}

	state.Resources = newResources

	// Update state
	return sm.UpdateState(ctx, key, state)
}

// MoveResource moves a resource within the state
func (sm *StateManager) MoveResource(ctx context.Context, key string, fromAddress, toAddress string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Get current state
	state, err := sm.GetState(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Find source resource
	var sourceResource *Resource
	newResources := make([]Resource, 0, len(state.Resources))

	for _, resource := range state.Resources {
		address := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		if address == fromAddress {
			sourceResource = &resource
			continue
		}
		newResources = append(newResources, resource)
	}

	if sourceResource == nil {
		return fmt.Errorf("source resource %s not found", fromAddress)
	}

	// Parse new address
	parts := strings.Split(toAddress, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid destination address: %s", toAddress)
	}

	// Update resource with new type and name
	sourceResource.Type = parts[0]
	sourceResource.Name = parts[1]

	// Add back to resources
	newResources = append(newResources, *sourceResource)
	state.Resources = newResources

	// Update state
	return sm.UpdateState(ctx, key, state)
}

// RefreshState updates the state with actual cloud resource data
func (sm *StateManager) RefreshState(ctx context.Context, key string, actualResources map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Get current state
	state, err := sm.GetState(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Update resource attributes with actual data
	for i, resource := range state.Resources {
		for j := range resource.Instances {
			address := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
			if len(resource.Instances) > 1 {
				address = fmt.Sprintf("%s[%d]", address, j)
			}

			if actualData, ok := actualResources[address]; ok {
				// Update instance attributes
				if attrs, ok := actualData.(map[string]interface{}); ok {
					state.Resources[i].Instances[j].Attributes = attrs
				}
			}
		}
	}

	// Update state
	return sm.UpdateState(ctx, key, state)
}

// CompareStates compares two states and returns differences
func (sm *StateManager) CompareStates(ctx context.Context, key1, key2 string) ([]StateDifference, error) {
	state1, err := sm.GetState(ctx, key1)
	if err != nil {
		return nil, fmt.Errorf("failed to get first state: %w", err)
	}

	state2, err := sm.GetState(ctx, key2)
	if err != nil {
		return nil, fmt.Errorf("failed to get second state: %w", err)
	}

	return sm.compareStateObjects(state1, state2), nil
}

// StateDifference represents a difference between two states
type StateDifference struct {
	Type        string      `json:"type"` // added, removed, modified
	Resource    string      `json:"resource"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Description string      `json:"description"`
}

func (sm *StateManager) compareStateObjects(state1, state2 *TerraformState) []StateDifference {
	var differences []StateDifference

	// Build resource maps
	resources1 := make(map[string]Resource)
	resources2 := make(map[string]Resource)

	for _, r := range state1.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		resources1[key] = r
	}

	for _, r := range state2.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		resources2[key] = r
	}

	// Check for removed resources
	for key, r1 := range resources1 {
		if _, exists := resources2[key]; !exists {
			differences = append(differences, StateDifference{
				Type:        "removed",
				Resource:    key,
				OldValue:    r1,
				Description: fmt.Sprintf("Resource %s was removed", key),
			})
		}
	}

	// Check for added and modified resources
	for key, r2 := range resources2 {
		r1, exists := resources1[key]
		if !exists {
			differences = append(differences, StateDifference{
				Type:        "added",
				Resource:    key,
				NewValue:    r2,
				Description: fmt.Sprintf("Resource %s was added", key),
			})
		} else {
			// Compare resource details
			if len(r1.Instances) != len(r2.Instances) {
				differences = append(differences, StateDifference{
					Type:        "modified",
					Resource:    key,
					OldValue:    len(r1.Instances),
					NewValue:    len(r2.Instances),
					Description: fmt.Sprintf("Instance count changed from %d to %d", len(r1.Instances), len(r2.Instances)),
				})
			}
		}
	}

	return differences
}

// generateLineage generates a new lineage ID
func generateLineage() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
