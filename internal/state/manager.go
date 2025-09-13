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
		return fmt.Errorf("failed to acquire lock: %w", err)
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
	states, err := sm.backend.ListStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list states: %w", err)
	}
	return states, nil
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

// PutState saves a state file (alias for UpdateState with proper locking)
func (sm *StateManager) PutState(ctx context.Context, key string, state *TerraformState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate before saving
	if err := sm.validator.Validate(state); err != nil {
		return fmt.Errorf("state validation failed: %w", err)
	}

	// Lock state
	if err := sm.backend.Lock(ctx, key); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer sm.backend.Unlock(ctx, key)

	// Serialize state
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Upload to backend
	if err := sm.backend.Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to put state: %w", err)
	}

	// Update cache
	sm.cache.Set(key, state)

	return nil
}

// ListStateVersions lists all versions of a state
func (sm *StateManager) ListStateVersions(ctx context.Context, key string) ([]StateVersion, error) {
	versions, err := sm.backend.ListStateVersions(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to list state versions: %w", err)
	}
	return versions, nil
}

// RestoreStateVersion restores a specific version and returns the restored state
func (sm *StateManager) RestoreStateVersion(ctx context.Context, key string, version int) (*TerraformState, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Get the specific version
	data, err := sm.backend.GetStateVersion(ctx, key, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get state version: %w", err)
	}

	// Parse to verify it's valid
	state, err := sm.parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse historical state: %w", err)
	}

	// Increment serial for the restore
	state.Serial++

	// Lock state
	if err := sm.backend.Lock(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to lock state: %w", err)
	}
	defer sm.backend.Unlock(ctx, key)

	// Serialize and upload
	newData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize state: %w", err)
	}

	if err := sm.backend.Put(ctx, key, newData); err != nil {
		return nil, fmt.Errorf("failed to restore state: %w", err)
	}

	// Update cache
	sm.cache.Set(key, state)

	return state, nil
}

// StateComparison represents the comparison between two states
type StateComparison struct {
	AreEqual          bool       `json:"are_equal"`
	SerialDiff        int        `json:"serial_diff"`
	AddedResources    []Resource `json:"added_resources"`
	RemovedResources  []Resource `json:"removed_resources"`
	ModifiedResources []Resource `json:"modified_resources"`
}

// CompareStates compares two state objects and returns their differences
func (sm *StateManager) CompareStates(state1, state2 *TerraformState) *StateComparison {
	comparison := &StateComparison{
		AreEqual:          true,
		SerialDiff:        state2.Serial - state1.Serial,
		AddedResources:    []Resource{},
		RemovedResources:  []Resource{},
		ModifiedResources: []Resource{},
	}

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
			comparison.RemovedResources = append(comparison.RemovedResources, r1)
			comparison.AreEqual = false
		}
	}

	// Check for added resources
	for key, r2 := range resources2 {
		if _, exists := resources1[key]; !exists {
			comparison.AddedResources = append(comparison.AddedResources, r2)
			comparison.AreEqual = false
		}
	}

	// Check serial difference
	if state1.Serial != state2.Serial {
		comparison.AreEqual = false
	}

	return comparison
}

// MoveResource moves a resource within a single state
func (sm *StateManager) MoveResource(state *TerraformState, fromAddress, toAddress string) error {
	// Parse addresses
	fromParts := strings.Split(fromAddress, ".")
	toParts := strings.Split(toAddress, ".")

	if len(fromParts) != 2 || len(toParts) != 2 {
		return fmt.Errorf("invalid resource address format")
	}

	// Find source resource
	var sourceResource *Resource
	sourceIndex := -1

	for i, resource := range state.Resources {
		if resource.Type == fromParts[0] && resource.Name == fromParts[1] {
			sourceResource = &state.Resources[i]
			sourceIndex = i
			break
		}
	}

	if sourceResource == nil {
		return fmt.Errorf("resource not found: %s", fromAddress)
	}

	// Check if target already exists
	for _, resource := range state.Resources {
		if resource.Type == toParts[0] && resource.Name == toParts[1] {
			return fmt.Errorf("target resource already exists: %s", toAddress)
		}
	}

	// Update resource
	state.Resources[sourceIndex].Type = toParts[0]
	state.Resources[sourceIndex].Name = toParts[1]

	return nil
}

// RemoveResource removes a resource from a state
func (sm *StateManager) RemoveResource(state *TerraformState, address string) error {
	// Check if address contains index
	var resourceType, resourceName string
	var instanceIndex int = -1

	if strings.Contains(address, "[") {
		// Parse indexed address like aws_instance.cluster[1]
		parts := strings.Split(address, ".")
		if len(parts) != 2 {
			return fmt.Errorf("invalid resource address: %s", address)
		}
		resourceType = parts[0]

		// Parse name and index
		nameParts := strings.Split(parts[1], "[")
		resourceName = nameParts[0]
		if len(nameParts) > 1 {
			indexStr := strings.TrimSuffix(nameParts[1], "]")
			fmt.Sscanf(indexStr, "%d", &instanceIndex)
		}
	} else {
		// Parse simple address like aws_instance.test
		parts := strings.Split(address, ".")
		if len(parts) != 2 {
			return fmt.Errorf("invalid resource address: %s", address)
		}
		resourceType = parts[0]
		resourceName = parts[1]
	}

	// Find and remove resource or instance
	for i, resource := range state.Resources {
		if resource.Type == resourceType && resource.Name == resourceName {
			if instanceIndex >= 0 {
				// Remove specific instance
				if instanceIndex >= len(resource.Instances) {
					return fmt.Errorf("instance index out of range: %d", instanceIndex)
				}

				// Remove the instance
				state.Resources[i].Instances = append(
					resource.Instances[:instanceIndex],
					resource.Instances[instanceIndex+1:]...,
				)

				// If no instances left, remove the entire resource
				if len(state.Resources[i].Instances) == 0 {
					state.Resources = append(state.Resources[:i], state.Resources[i+1:]...)
				}
			} else {
				// Remove entire resource
				state.Resources = append(state.Resources[:i], state.Resources[i+1:]...)
			}
			return nil
		}
	}

	return fmt.Errorf("resource not found: %s", address)
}
