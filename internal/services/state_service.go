package services

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// StateService handles Terraform state management business logic
type StateService struct {
	repository     StateRepository
	backendService *BackendService
}

// StateRepository defines the interface for state data persistence
type StateRepository interface {
	Create(ctx context.Context, state *models.StateFile) error
	GetByID(ctx context.Context, id string) (*models.StateFile, error)
	GetAll(ctx context.Context, filters StateFilters) ([]*models.StateFile, error)
	Update(ctx context.Context, state *models.StateFile) error
	Delete(ctx context.Context, id string) error
	GetStateDetails(ctx context.Context, id string) (*models.StateDetails, error)
	ImportResource(ctx context.Context, req *models.ImportRequest) (*models.ImportResult, error)
	RemoveResource(ctx context.Context, req *models.RemoveResourceRequest) error
	MoveResource(ctx context.Context, req *models.MoveResourceRequest) error
	LockState(ctx context.Context, req *models.LockStateRequest) error
	UnlockState(ctx context.Context, req *models.UnlockStateRequest) error
}

// StateFilters represents filters for state queries
type StateFilters struct {
	BackendID   string `json:"backend_id,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
	Environment string `json:"environment,omitempty"`
	IsLocked    *bool  `json:"is_locked,omitempty"`
	CreatedBy   string `json:"created_by,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// StateDetails represents detailed state information
type StateDetails struct {
	ID          string                 `json:"id"`
	BackendID   string                 `json:"backend_id"`
	Workspace   string                 `json:"workspace"`
	Environment string                 `json:"environment"`
	Version     int                    `json:"version"`
	Serial      int                    `json:"serial"`
	Lineage     string                 `json:"lineage"`
	Resources   []StateResource        `json:"resources"`
	Outputs     map[string]interface{} `json:"outputs"`
	IsLocked    bool                   `json:"is_locked"`
	LockInfo    *StateLockInfo         `json:"lock_info,omitempty"`
	LastUpdated time.Time              `json:"last_updated"`
	CreatedAt   time.Time              `json:"created_at"`
}

// StateResource represents a resource in the state
type StateResource struct {
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Provider  string                 `json:"provider"`
	Instances []StateInstance        `json:"instances"`
	Config    map[string]interface{} `json:"config"`
}

// StateInstance represents an instance of a resource
type StateInstance struct {
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
	Meta       map[string]interface{} `json:"meta"`
}

// StateLockInfo represents state lock information
type StateLockInfo struct {
	ID        string    `json:"id"`
	Operation string    `json:"operation"`
	Info      string    `json:"info"`
	Who       string    `json:"who"`
	Version   string    `json:"version"`
	Created   time.Time `json:"created"`
	Path      string    `json:"path"`
}

// ImportRequest represents a request to import a resource into state
type ImportRequest struct {
	StateID      string `json:"state_id" validate:"required"`
	ResourceType string `json:"resource_type" validate:"required"`
	ResourceName string `json:"resource_name" validate:"required"`
	ResourceID   string `json:"resource_id" validate:"required"`
	Provider     string `json:"provider" validate:"required"`
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
	ResourceID string                 `json:"resource_id"`
	Details    map[string]interface{} `json:"details,omitempty"`
	ImportedAt time.Time              `json:"imported_at"`
}

// RemoveResourceRequest represents a request to remove a resource from state
type RemoveResourceRequest struct {
	StateID      string `json:"state_id" validate:"required"`
	ResourceType string `json:"resource_type" validate:"required"`
	ResourceName string `json:"resource_name" validate:"required"`
}

// MoveResourceRequest represents a request to move a resource in state
type MoveResourceRequest struct {
	StateID  string `json:"state_id" validate:"required"`
	FromType string `json:"from_type" validate:"required"`
	FromName string `json:"from_name" validate:"required"`
	ToType   string `json:"to_type" validate:"required"`
	ToName   string `json:"to_name" validate:"required"`
}

// LockStateRequest represents a request to lock a state file
type LockStateRequest struct {
	StateID   string `json:"state_id" validate:"required"`
	Operation string `json:"operation" validate:"required"`
	Info      string `json:"info"`
	Who       string `json:"who" validate:"required"`
	Version   string `json:"version"`
}

// UnlockStateRequest represents a request to unlock a state file
type UnlockStateRequest struct {
	StateID string `json:"state_id" validate:"required"`
	Force   bool   `json:"force"`
}

// NewStateService creates a new state service
func NewStateService(repository StateRepository, backendService *BackendService) *StateService {
	return &StateService{
		repository:     repository,
		backendService: backendService,
	}
}

// ListStateFiles retrieves all state files with optional filtering
func (s *StateService) ListStateFiles(ctx context.Context, filters StateFilters) ([]*models.StateFile, error) {
	if filters.Limit <= 0 {
		filters.Limit = 100 // Default limit
	}

	states, err := s.repository.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list state files: %w", err)
	}

	return states, nil
}

// GetStateFile retrieves a specific state file by ID
func (s *StateService) GetStateFile(ctx context.Context, id string) (*models.StateFile, error) {
	state, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get state file %s: %w", id, err)
	}

	return state, nil
}

// GetStateDetails retrieves detailed information about a state file
func (s *StateService) GetStateDetails(ctx context.Context, id string) (*StateDetails, error) {
	details, err := s.repository.GetStateDetails(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get state details for %s: %w", id, err)
	}

	// Convert models.StateDetails to service StateDetails
	serviceDetails := &StateDetails{
		ID:          details.ID,
		BackendID:   details.BackendID,
		Workspace:   details.Workspace,
		Environment: details.Environment,
		Version:     details.Version,
		Serial:      details.Serial,
		Lineage:     details.Lineage,
		Resources:   convertStateResources(details.Resources),
		Outputs:     details.Outputs,
		IsLocked:    details.IsLocked,
		LastUpdated: details.LastUpdated,
		CreatedAt:   details.CreatedAt,
	}
	return serviceDetails, nil
}

// CreateStateFile creates a new state file
func (s *StateService) CreateStateFile(ctx context.Context, req *models.StateFileRequest) (*models.StateFile, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid state file request: %w", err)
	}

	// Verify backend exists
	_, err := s.backendService.GetBackend(ctx, req.BackendID)
	if err != nil {
		return nil, fmt.Errorf("backend %s not found: %w", req.BackendID, err)
	}

	// Create the state file
	state := &models.StateFile{
		ID:           generateStateID(),
		BackendID:    req.BackendID,
		Name:         fmt.Sprintf("%s-%s", req.Workspace, req.Environment),
		Path:         fmt.Sprintf("state/%s/%s/terraform.tfstate", req.Workspace, req.Environment),
		Version:      1,
		Serial:       0,
		Lineage:      generateLineage(),
		Resources:    []models.StateResource{},
		Outputs:      make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		Size:         0,
		Checksum:     "",
		LastModified: time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save to repository
	if err := s.repository.Create(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to create state file: %w", err)
	}

	return state, nil
}

// UpdateStateFile updates an existing state file
func (s *StateService) UpdateStateFile(ctx context.Context, id string, req *models.StateFileUpdateRequest) (*models.StateFile, error) {
	// Get existing state file
	state, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get state file %s: %w", id, err)
	}

	// Update fields if provided
	if req.Workspace != nil {
		state.Name = fmt.Sprintf("%s-%s", *req.Workspace, state.Name)
		state.Path = fmt.Sprintf("state/%s/%s/terraform.tfstate", *req.Workspace, state.Name)
	}
	if req.Environment != nil {
		state.Name = fmt.Sprintf("%s-%s", state.Name, *req.Environment)
		state.Path = fmt.Sprintf("state/%s/%s/terraform.tfstate", state.Name, *req.Environment)
	}

	state.UpdatedAt = time.Now()

	// Save to repository
	if err := s.repository.Update(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to update state file: %w", err)
	}

	return state, nil
}

// DeleteStateFile deletes a state file
func (s *StateService) DeleteStateFile(ctx context.Context, id string) error {
	// Check if state file exists
	_, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get state file %s: %w", id, err)
	}

	// Check if state is locked
	details, err := s.repository.GetStateDetails(ctx, id)
	if err == nil && details.IsLocked {
		return fmt.Errorf("cannot delete locked state file")
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	return nil
}

// ImportResourceToState imports a resource into the state file
func (s *StateService) ImportResourceToState(ctx context.Context, req *ImportRequest) (*ImportResult, error) {
	// Validate the request
	if req.StateID == "" || req.ResourceType == "" || req.ResourceName == "" || req.ResourceID == "" {
		return nil, fmt.Errorf("invalid import request: missing required fields")
	}

	// Check if state file exists
	_, err := s.repository.GetByID(ctx, req.StateID)
	if err != nil {
		return nil, fmt.Errorf("state file %s not found: %w", req.StateID, err)
	}

	// Convert service request to model request
	modelReq := &models.ImportRequest{
		StateID:      req.StateID,
		ResourceType: req.ResourceType,
		ResourceName: req.ResourceName,
		ResourceID:   req.ResourceID,
		Provider:     req.Provider,
	}

	// Perform import
	modelResult, err := s.repository.ImportResource(ctx, modelReq)
	if err != nil {
		return nil, fmt.Errorf("failed to import resource: %w", err)
	}

	// Convert model result to service result
	result := &ImportResult{
		Success:    modelResult.Success,
		Message:    modelResult.Message,
		ResourceID: modelResult.ResourceID,
		Details:    modelResult.Details,
		ImportedAt: modelResult.ImportedAt,
	}

	return result, nil
}

// RemoveResourceFromState removes a resource from the state file
func (s *StateService) RemoveResourceFromState(ctx context.Context, req *RemoveResourceRequest) error {
	// Validate the request
	if req.StateID == "" || req.ResourceType == "" || req.ResourceName == "" {
		return fmt.Errorf("invalid remove request: missing required fields")
	}

	// Check if state file exists
	_, err := s.repository.GetByID(ctx, req.StateID)
	if err != nil {
		return fmt.Errorf("state file %s not found: %w", req.StateID, err)
	}

	// Convert service request to model request
	modelReq := &models.RemoveResourceRequest{
		ResourceAddress: fmt.Sprintf("%s.%s", req.ResourceType, req.ResourceName),
		Force:           false,
	}

	// Perform removal
	if err := s.repository.RemoveResource(ctx, modelReq); err != nil {
		return fmt.Errorf("failed to remove resource: %w", err)
	}

	return nil
}

// MoveResourceInState moves a resource within the state file
func (s *StateService) MoveResourceInState(ctx context.Context, req *MoveResourceRequest) error {
	// Validate the request
	if req.StateID == "" || req.FromType == "" || req.FromName == "" || req.ToType == "" || req.ToName == "" {
		return fmt.Errorf("invalid move request: missing required fields")
	}

	// Check if state file exists
	_, err := s.repository.GetByID(ctx, req.StateID)
	if err != nil {
		return fmt.Errorf("state file %s not found: %w", req.StateID, err)
	}

	// Convert service request to model request
	modelReq := &models.MoveResourceRequest{
		FromAddress: fmt.Sprintf("%s.%s", req.FromType, req.FromName),
		ToAddress:   fmt.Sprintf("%s.%s", req.ToType, req.ToName),
	}

	// Perform move
	if err := s.repository.MoveResource(ctx, modelReq); err != nil {
		return fmt.Errorf("failed to move resource: %w", err)
	}

	return nil
}

// LockStateFile locks a state file
func (s *StateService) LockStateFile(ctx context.Context, req *LockStateRequest) error {
	// Validate the request
	if req.StateID == "" || req.Operation == "" || req.Who == "" {
		return fmt.Errorf("invalid lock request: missing required fields")
	}

	// Check if state file exists
	_, err := s.repository.GetByID(ctx, req.StateID)
	if err != nil {
		return fmt.Errorf("state file %s not found: %w", req.StateID, err)
	}

	// Convert service request to model request
	modelReq := &models.LockStateRequest{
		StateID:   req.StateID,
		Operation: req.Operation,
		Info:      req.Info,
		Who:       req.Who,
		Version:   req.Version,
	}

	// Perform lock
	if err := s.repository.LockState(ctx, modelReq); err != nil {
		return fmt.Errorf("failed to lock state file: %w", err)
	}

	return nil
}

// UnlockStateFile unlocks a state file
func (s *StateService) UnlockStateFile(ctx context.Context, req *UnlockStateRequest) error {
	// Validate the request
	if req.StateID == "" {
		return fmt.Errorf("invalid unlock request: missing state ID")
	}

	// Check if state file exists
	_, err := s.repository.GetByID(ctx, req.StateID)
	if err != nil {
		return fmt.Errorf("state file %s not found: %w", req.StateID, err)
	}

	// Convert service request to model request
	modelReq := &models.UnlockStateRequest{
		StateID: req.StateID,
		Force:   req.Force,
	}

	// Perform unlock
	if err := s.repository.UnlockState(ctx, modelReq); err != nil {
		return fmt.Errorf("failed to unlock state file: %w", err)
	}

	return nil
}

// Helper functions

func generateStateID() string {
	// In a real implementation, this would generate a proper UUID
	return fmt.Sprintf("state_%d", time.Now().UnixNano())
}

func generateLineage() string {
	// In a real implementation, this would generate a proper lineage ID
	return fmt.Sprintf("lineage_%d", time.Now().UnixNano())
}

// convertStateResources converts models.StateResource to service StateResource
func convertStateResources(resources []models.StateResource) []StateResource {
	result := make([]StateResource, len(resources))
	for i, r := range resources {
		result[i] = StateResource{
			Type:     r.Type,
			Name:     r.Address, // Use address as name
			Provider: r.Provider,
			Instances: []StateInstance{
				{
					ID:         r.ID,
					Attributes: r.Attributes,
					Meta:       make(map[string]interface{}),
				},
			},
			Config: r.Attributes,
		}
	}
	return result
}
