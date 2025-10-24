package repositories

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/services"
)

// MemoryStateRepository is an in-memory implementation of StateRepository
type MemoryStateRepository struct {
	states map[string]*models.StateFile
	mu     sync.RWMutex
}

// NewMemoryStateRepository creates a new in-memory state repository
func NewMemoryStateRepository() *MemoryStateRepository {
	return &MemoryStateRepository{
		states: make(map[string]*models.StateFile),
	}
}

// Create creates a new state file
func (r *MemoryStateRepository) Create(ctx context.Context, state *models.StateFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state already exists
	if _, exists := r.states[state.ID]; exists {
		return fmt.Errorf("state file with ID %s already exists", state.ID)
	}

	// Set timestamps
	now := time.Now()
	state.CreatedAt = now
	state.UpdatedAt = now

	// Store state
	r.states[state.ID] = state

	return nil
}

// GetByID retrieves a state file by ID
func (r *MemoryStateRepository) GetByID(ctx context.Context, id string) (*models.StateFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.states[id]
	if !exists {
		return nil, fmt.Errorf("state file with ID %s not found", id)
	}

	// Return a copy to prevent external modifications
	stateCopy := *state
	return &stateCopy, nil
}

// GetAll retrieves all state files with optional filtering
func (r *MemoryStateRepository) GetAll(ctx context.Context, filters services.StateFilters) ([]*models.StateFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.StateFile

	for _, state := range r.states {
		// Apply filters
		if filters.BackendID != "" && state.BackendID != filters.BackendID {
			continue
		}

		// Return a copy to prevent external modifications
		stateCopy := *state
		results = append(results, &stateCopy)
	}

	// Apply pagination
	if filters.Offset > 0 && filters.Offset < len(results) {
		results = results[filters.Offset:]
	}
	if filters.Limit > 0 && filters.Limit < len(results) {
		results = results[:filters.Limit]
	}

	return results, nil
}

// Update updates an existing state file
func (r *MemoryStateRepository) Update(ctx context.Context, state *models.StateFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state exists
	if _, exists := r.states[state.ID]; !exists {
		return fmt.Errorf("state file with ID %s not found", state.ID)
	}

	// Update timestamp
	state.UpdatedAt = time.Now()

	// Store updated state
	r.states[state.ID] = state

	return nil
}

// Delete deletes a state file
func (r *MemoryStateRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state exists
	if _, exists := r.states[id]; !exists {
		return fmt.Errorf("state file with ID %s not found", id)
	}

	// Delete state
	delete(r.states, id)

	return nil
}

// GetStateDetails retrieves detailed information about a state file
func (r *MemoryStateRepository) GetStateDetails(ctx context.Context, id string) (*services.StateDetails, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.states[id]
	if !exists {
		return nil, fmt.Errorf("state file with ID %s not found", id)
	}

	// Convert to detailed state information
	details := &services.StateDetails{
		ID:          state.ID,
		BackendID:   state.BackendID,
		Version:     state.Version,
		Serial:      int(state.Serial),
		Lineage:     state.Lineage,
		Resources:   []services.StateResource{}, // This would be populated from actual state data
		Outputs:     make(map[string]interface{}),
		IsLocked:    false, // StateFile doesn't have IsLocked field
		LastUpdated: state.UpdatedAt,
		CreatedAt:   state.CreatedAt,
	}

	return details, nil
}

// ImportResource imports a resource into the state file
func (r *MemoryStateRepository) ImportResource(ctx context.Context, req *services.ImportRequest) (*services.ImportResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state exists
	state, exists := r.states[req.StateID]
	if !exists {
		return nil, fmt.Errorf("state file with ID %s not found", req.StateID)
	}

	// Simulate resource import
	result := &services.ImportResult{
		Success:    true,
		Message:    fmt.Sprintf("Resource %s.%s imported successfully", req.ResourceType, req.ResourceName),
		ResourceID: req.ResourceID,
		Details: map[string]interface{}{
			"resource_type": req.ResourceType,
			"resource_name": req.ResourceName,
			"provider":      req.Provider,
			"state_id":      req.StateID,
		},
		ImportedAt: time.Now(),
	}

	// Update state version
	state.Version++
	state.Serial++
	state.UpdatedAt = time.Now()

	return result, nil
}

// RemoveResource removes a resource from the state file
func (r *MemoryStateRepository) RemoveResource(ctx context.Context, req *services.RemoveResourceRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state exists
	state, exists := r.states[req.StateID]
	if !exists {
		return fmt.Errorf("state file with ID %s not found", req.StateID)
	}

	// Simulate resource removal
	// In a real implementation, this would remove the resource from the state file

	// Update state version
	state.Version++
	state.Serial++
	state.UpdatedAt = time.Now()

	return nil
}

// MoveResource moves a resource within the state file
func (r *MemoryStateRepository) MoveResource(ctx context.Context, req *services.MoveResourceRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state exists
	state, exists := r.states[req.StateID]
	if !exists {
		return fmt.Errorf("state file with ID %s not found", req.StateID)
	}

	// Simulate resource move
	// In a real implementation, this would move the resource within the state file

	// Update state version
	state.Version++
	state.Serial++
	state.UpdatedAt = time.Now()

	return nil
}

// LockState locks a state file
func (r *MemoryStateRepository) LockState(ctx context.Context, req *services.LockStateRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state exists
	state, exists := r.states[req.StateID]
	if !exists {
		return fmt.Errorf("state file with ID %s not found", req.StateID)
	}

	// For now, we'll simulate locking by updating the timestamp
	// In a real implementation, you'd have a separate lock table or field
	state.UpdatedAt = time.Now()

	return nil
}

// UnlockState unlocks a state file
func (r *MemoryStateRepository) UnlockState(ctx context.Context, req *services.UnlockStateRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if state exists
	state, exists := r.states[req.StateID]
	if !exists {
		return fmt.Errorf("state file with ID %s not found", req.StateID)
	}

	// For now, we'll simulate unlocking by updating the timestamp
	// In a real implementation, you'd have a separate lock table or field
	state.UpdatedAt = time.Now()

	return nil
}
