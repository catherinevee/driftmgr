package state

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/storage/state"
)

// ServiceConfig holds configuration for the state management service
type ServiceConfig struct {
	MaxConcurrentOperations int           `json:"max_concurrent_operations"`
	DefaultTimeout          time.Duration `json:"default_timeout"`
	MaxRetryCount           int           `json:"max_retry_count"`
	RetentionDays           int           `json:"retention_days"`
	EnableAutoCleanup       bool          `json:"enable_auto_cleanup"`
	CleanupInterval         time.Duration `json:"cleanup_interval"`
	EnableStateLocking      bool          `json:"enable_state_locking"`
	LockTimeout             time.Duration `json:"lock_timeout"`
	MaxStateFileSize        int64         `json:"max_state_file_size"`
	EnableResourceCaching   bool          `json:"enable_resource_caching"`
	CacheExpiration         time.Duration `json:"cache_expiration"`
}

// Service implements the business logic for state management
type Service struct {
	repo   state.Repository
	config *ServiceConfig
}

// NewService creates a new state management service
func NewService(repo state.Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = &ServiceConfig{
			MaxConcurrentOperations: 10,
			DefaultTimeout:          time.Hour,
			MaxRetryCount:           3,
			RetentionDays:           30,
			EnableAutoCleanup:       true,
			CleanupInterval:         time.Hour * 24,
			EnableStateLocking:      true,
			LockTimeout:             time.Minute * 30,
			MaxStateFileSize:        100 * 1024 * 1024, // 100MB
			EnableResourceCaching:   true,
			CacheExpiration:         time.Hour,
		}
	}

	return &Service{
		repo:   repo,
		config: config,
	}
}

// ListStateFiles lists state files with filtering
func (s *Service) ListStateFiles(ctx context.Context, req *models.StateFileListRequest) (*models.StateFileListResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}
	if req.SortBy == "" {
		req.SortBy = "last_modified"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Get state files from repository
	response, err := s.repo.ListStateFiles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list state files: %w", err)
	}

	return response, nil
}

// GetStateFile retrieves a state file by ID
func (s *Service) GetStateFile(ctx context.Context, id string) (*models.StateFile, error) {
	if id == "" {
		return nil, models.ErrBadRequest
	}

	stateFile, err := s.repo.GetStateFileByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return stateFile, nil
}

// CreateStateFile creates a new state file
func (s *Service) CreateStateFile(ctx context.Context, stateFile *models.StateFile) error {
	// Validate state file
	if err := stateFile.Validate(); err != nil {
		return fmt.Errorf("invalid state file: %w", err)
	}

	// Check if state file already exists
	existing, err := s.repo.GetStateFileByPath(ctx, stateFile.Path)
	if err == nil && existing != nil {
		return models.ErrStateFileExists
	}

	// Set defaults
	stateFile.ID = generateStateFileID()
	stateFile.CreatedAt = time.Now()
	stateFile.UpdatedAt = time.Now()

	// Save to repository
	if err := s.repo.CreateStateFile(ctx, stateFile); err != nil {
		return fmt.Errorf("failed to create state file: %w", err)
	}

	return nil
}

// UpdateStateFile updates an existing state file
func (s *Service) UpdateStateFile(ctx context.Context, stateFile *models.StateFile) error {
	// Validate state file
	if err := stateFile.Validate(); err != nil {
		return fmt.Errorf("invalid state file: %w", err)
	}

	// Check if state file exists
	_, err := s.repo.GetStateFileByID(ctx, stateFile.ID)
	if err != nil {
		return err
	}

	// Update timestamp
	stateFile.UpdatedAt = time.Now()

	// Save to repository
	if err := s.repo.UpdateStateFile(ctx, stateFile); err != nil {
		return fmt.Errorf("failed to update state file: %w", err)
	}

	return nil
}

// DeleteStateFile deletes a state file
func (s *Service) DeleteStateFile(ctx context.Context, id string) error {
	if id == "" {
		return models.ErrBadRequest
	}

	// Check if state file exists
	_, err := s.repo.GetStateFileByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if state file is locked
	lock, err := s.repo.GetStateLockByStateFile(ctx, id)
	if err == nil && lock != nil {
		return models.ErrStateFileLocked
	}

	// Delete state file
	if err := s.repo.DeleteStateFile(ctx, id); err != nil {
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	return nil
}

// ListResources lists resources with filtering
func (s *Service) ListResources(ctx context.Context, req *models.ResourceListRequest) (*models.ResourceListResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}
	if req.SortBy == "" {
		req.SortBy = "address"
	}
	if req.SortOrder == "" {
		req.SortOrder = "asc"
	}

	// Get resources from repository
	response, err := s.repo.ListResources(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return response, nil
}

// GetResource retrieves a resource by ID
func (s *Service) GetResource(ctx context.Context, id string) (*models.StateResource, error) {
	if id == "" {
		return nil, models.ErrBadRequest
	}

	resource, err := s.repo.GetResourceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

// ImportResource imports a resource into a state file
func (s *Service) ImportResource(ctx context.Context, stateFileID string, req *models.ImportResourceRequest) (*models.ImportResourceResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get state file
	stateFile, err := s.repo.GetStateFileByID(ctx, stateFileID)
	if err != nil {
		return nil, err
	}

	// Check if state file is locked
	if s.config.EnableStateLocking {
		lock, err := s.repo.GetStateLockByStateFile(ctx, stateFileID)
		if err == nil && lock != nil {
			return nil, models.ErrStateFileLocked
		}
	}

	// Check if resource already exists
	existing, err := s.repo.GetResourceByAddress(ctx, stateFileID, req.ResourceAddress)
	if err == nil && existing != nil {
		return nil, models.ErrResourceExists
	}

	// Create resource
	resource := &models.StateResource{
		ID:            generateResourceID(),
		StateFileID:   stateFileID,
		Address:       req.ResourceAddress,
		Type:          extractResourceType(req.ResourceAddress),
		Provider:      extractProvider(req.ResourceAddress),
		Instance:      req.ResourceID,
		Attributes:    req.Configuration,
		Mode:          "managed",
		SchemaVersion: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Validate resource
	if err := resource.Validate(); err != nil {
		return nil, fmt.Errorf("invalid resource: %w", err)
	}

	// Save resource
	if err := s.repo.CreateResource(ctx, resource); err != nil {
		return nil, fmt.Errorf("failed to import resource: %w", err)
	}

	// Update state file
	stateFile.AddResource(*resource)
	if err := s.repo.UpdateStateFile(ctx, stateFile); err != nil {
		return nil, fmt.Errorf("failed to update state file: %w", err)
	}

	// Create operation record
	operation := &models.StateOperation{
		ID:            generateOperationID(),
		StateFileID:   stateFileID,
		OperationType: models.StateOperationImport,
		Status:        models.OperationStatusCompleted,
		Parameters: map[string]interface{}{
			"resource_address": req.ResourceAddress,
			"resource_id":      req.ResourceID,
		},
		Result: map[string]interface{}{
			"resource_id": resource.ID,
		},
		CreatedBy: getCurrentUser(ctx),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	operation.SetStatus(models.OperationStatusCompleted)

	if err := s.repo.CreateStateOperation(ctx, operation); err != nil {
		// Log error but don't fail the import
		fmt.Printf("Failed to create operation record: %v\n", err)
	}

	return &models.ImportResourceResponse{
		ResourceID:      resource.ID,
		ResourceAddress: resource.Address,
		Status:          "imported",
		Message:         "Resource imported successfully",
		ImportedAt:      time.Now(),
	}, nil
}

// RemoveResource removes a resource from a state file
func (s *Service) RemoveResource(ctx context.Context, stateFileID string, req *models.RemoveResourceRequest) (*models.RemoveResourceResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get state file
	stateFile, err := s.repo.GetStateFileByID(ctx, stateFileID)
	if err != nil {
		return nil, err
	}

	// Check if state file is locked
	if s.config.EnableStateLocking {
		lock, err := s.repo.GetStateLockByStateFile(ctx, stateFileID)
		if err == nil && lock != nil {
			return nil, models.ErrStateFileLocked
		}
	}

	// Get resource
	resource, err := s.repo.GetResourceByAddress(ctx, stateFileID, req.ResourceAddress)
	if err != nil {
		return nil, err
	}

	// Remove resource
	if err := s.repo.DeleteResource(ctx, resource.ID); err != nil {
		return nil, fmt.Errorf("failed to remove resource: %w", err)
	}

	// Update state file
	stateFile.RemoveResource(req.ResourceAddress)
	if err := s.repo.UpdateStateFile(ctx, stateFile); err != nil {
		return nil, fmt.Errorf("failed to update state file: %w", err)
	}

	// Create operation record
	operation := &models.StateOperation{
		ID:            generateOperationID(),
		StateFileID:   stateFileID,
		OperationType: models.StateOperationRemove,
		Status:        models.OperationStatusCompleted,
		Parameters: map[string]interface{}{
			"resource_address": req.ResourceAddress,
			"force":            req.Force,
		},
		Result: map[string]interface{}{
			"removed": true,
		},
		CreatedBy: getCurrentUser(ctx),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	operation.SetStatus(models.OperationStatusCompleted)

	if err := s.repo.CreateStateOperation(ctx, operation); err != nil {
		// Log error but don't fail the removal
		fmt.Printf("Failed to create operation record: %v\n", err)
	}

	return &models.RemoveResourceResponse{
		ResourceAddress: req.ResourceAddress,
		Status:          "removed",
		Message:         "Resource removed successfully",
		RemovedAt:       time.Now(),
	}, nil
}

// MoveResource moves a resource within a state file
func (s *Service) MoveResource(ctx context.Context, stateFileID string, req *models.MoveResourceRequest) (*models.MoveResourceResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get state file
	stateFile, err := s.repo.GetStateFileByID(ctx, stateFileID)
	if err != nil {
		return nil, err
	}

	// Check if state file is locked
	if s.config.EnableStateLocking {
		lock, err := s.repo.GetStateLockByStateFile(ctx, stateFileID)
		if err == nil && lock != nil {
			return nil, models.ErrStateFileLocked
		}
	}

	// Get source resource
	sourceResource, err := s.repo.GetResourceByAddress(ctx, stateFileID, req.FromAddress)
	if err != nil {
		return nil, err
	}

	// Check if target address already exists
	_, err = s.repo.GetResourceByAddress(ctx, stateFileID, req.ToAddress)
	if err == nil {
		return nil, models.ErrResourceExists
	}

	// Update resource address
	sourceResource.Address = req.ToAddress
	sourceResource.UpdatedAt = time.Now()

	// Save updated resource
	if err := s.repo.UpdateResource(ctx, sourceResource); err != nil {
		return nil, fmt.Errorf("failed to move resource: %w", err)
	}

	// Update state file
	stateFile.UpdateResource(req.FromAddress, *sourceResource)
	if err := s.repo.UpdateStateFile(ctx, stateFile); err != nil {
		return nil, fmt.Errorf("failed to update state file: %w", err)
	}

	// Create operation record
	operation := &models.StateOperation{
		ID:            generateOperationID(),
		StateFileID:   stateFileID,
		OperationType: models.StateOperationMove,
		Status:        models.OperationStatusCompleted,
		Parameters: map[string]interface{}{
			"from_address": req.FromAddress,
			"to_address":   req.ToAddress,
		},
		Result: map[string]interface{}{
			"moved": true,
		},
		CreatedBy: getCurrentUser(ctx),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	operation.SetStatus(models.OperationStatusCompleted)

	if err := s.repo.CreateStateOperation(ctx, operation); err != nil {
		// Log error but don't fail the move
		fmt.Printf("Failed to create operation record: %v\n", err)
	}

	return &models.MoveResourceResponse{
		FromAddress: req.FromAddress,
		ToAddress:   req.ToAddress,
		Status:      "moved",
		Message:     "Resource moved successfully",
		MovedAt:     time.Now(),
	}, nil
}

// ExportResource exports a resource configuration
func (s *Service) ExportResource(ctx context.Context, resourceID string, req *models.ExportResourceRequest) (*models.ExportResourceResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get resource
	resource, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	// Export resource configuration
	configuration := map[string]interface{}{
		"address":      resource.Address,
		"type":         resource.Type,
		"provider":     resource.Provider,
		"attributes":   resource.Attributes,
		"dependencies": resource.Dependencies,
		"depends_on":   resource.DependsOn,
		"module":       resource.Module,
		"mode":         resource.Mode,
	}

	return &models.ExportResourceResponse{
		ResourceID:      resource.ID,
		ResourceAddress: resource.Address,
		Format:          req.Format,
		Configuration:   configuration,
		ExportedAt:      time.Now(),
	}, nil
}

// ListBackends lists backends with filtering
func (s *Service) ListBackends(ctx context.Context, req *models.BackendListRequest) (*models.BackendListResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}
	if req.SortBy == "" {
		req.SortBy = "name"
	}
	if req.SortOrder == "" {
		req.SortOrder = "asc"
	}

	// Get backends from repository
	response, err := s.repo.ListBackends(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list backends: %w", err)
	}

	return response, nil
}

// GetBackend retrieves a backend by ID
func (s *Service) GetBackend(ctx context.Context, id string) (*models.Backend, error) {
	if id == "" {
		return nil, models.ErrBadRequest
	}

	backend, err := s.repo.GetBackendByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return backend, nil
}

// CreateBackend creates a new backend
func (s *Service) CreateBackend(ctx context.Context, req *models.BackendCreateRequest) (*models.Backend, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if backend with same name exists
	existing, err := s.repo.GetBackendByName(ctx, req.Name)
	if err == nil && existing != nil {
		return nil, models.ErrBackendExists
	}

	// Create backend
	backend := &models.Backend{
		ID:            generateBackendID(),
		EnvironmentID: req.EnvironmentID,
		Type:          req.Type,
		Name:          req.Name,
		Description:   req.Description,
		Configuration: req.Configuration,
		IsActive:      true,
		IsDefault:     req.IsDefault,
		CreatedBy:     getCurrentUser(ctx),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Validate backend
	if err := backend.Validate(); err != nil {
		return nil, fmt.Errorf("invalid backend: %w", err)
	}

	// Save to repository
	if err := s.repo.CreateBackend(ctx, backend); err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	return backend, nil
}

// LockStateFile locks a state file
func (s *Service) LockStateFile(ctx context.Context, stateFileID string, req *models.StateLockRequest) (*models.StateLockResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if state file exists
	_, err := s.repo.GetStateFileByID(ctx, stateFileID)
	if err != nil {
		return nil, err
	}

	// Check if already locked
	existingLock, err := s.repo.GetStateLockByStateFile(ctx, stateFileID)
	if err == nil && existingLock != nil {
		return nil, models.ErrStateFileLocked
	}

	// Create lock
	lock := &models.StateLock{
		ID:          generateLockID(),
		StateFileID: stateFileID,
		LockID:      generateLockID(),
		Operation:   req.Operation,
		Who:         getCurrentUser(ctx),
		Version:     "1.0",
		Created:     time.Now(),
		Path:        stateFileID,
		Info:        req.Info,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Validate lock
	if err := lock.Validate(); err != nil {
		return nil, fmt.Errorf("invalid lock: %w", err)
	}

	// Save lock
	if err := s.repo.CreateStateLock(ctx, lock); err != nil {
		return nil, fmt.Errorf("failed to lock state file: %w", err)
	}

	return &models.StateLockResponse{
		LockID:   lock.LockID,
		Status:   "locked",
		Message:  "State file locked successfully",
		LockedAt: time.Now(),
	}, nil
}

// UnlockStateFile unlocks a state file
func (s *Service) UnlockStateFile(ctx context.Context, stateFileID string, req *models.StateUnlockRequest) (*models.StateUnlockResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get lock
	lock, err := s.repo.GetStateLockByStateFile(ctx, stateFileID)
	if err != nil {
		return nil, err
	}

	// Check lock ID
	if lock.LockID != req.LockID {
		return nil, models.ErrStateLockNotFound
	}

	// Delete lock
	if err := s.repo.DeleteStateLock(ctx, lock.ID); err != nil {
		return nil, fmt.Errorf("failed to unlock state file: %w", err)
	}

	return &models.StateUnlockResponse{
		Status:     "unlocked",
		Message:    "State file unlocked successfully",
		UnlockedAt: time.Now(),
	}, nil
}

// Health checks the health of the state management service
func (s *Service) Health(ctx context.Context) (*models.HealthResponse, error) {
	// Check repository health
	if err := s.repo.Health(ctx); err != nil {
		return nil, fmt.Errorf("repository health check failed: %w", err)
	}

	// Get counts
	stateFileCount, err := s.repo.GetStateFileCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get state file count: %w", err)
	}

	resourceCount, err := s.repo.GetResourceCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource count: %w", err)
	}

	backendCount, err := s.repo.GetBackendCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend count: %w", err)
	}

	activeLocksCount, err := s.repo.GetActiveLocksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active locks count: %w", err)
	}

	// Check if we have too many state files
	if stateFileCount > 10000 {
		return nil, fmt.Errorf("too many state files: %d", stateFileCount)
	}

	return &models.HealthResponse{
		Status:           "healthy",
		Service:          "state_management",
		StateFileCount:   stateFileCount,
		ResourceCount:    resourceCount,
		BackendCount:     backendCount,
		ActiveLocksCount: activeLocksCount,
		CheckedAt:        time.Now(),
	}, nil
}

// Helper functions

// generateStateFileID generates a unique state file ID
func generateStateFileID() string {
	return fmt.Sprintf("state-%d-%s", time.Now().UnixNano(), randomString(8))
}

// generateResourceID generates a unique resource ID
func generateResourceID() string {
	return fmt.Sprintf("resource-%d-%s", time.Now().UnixNano(), randomString(8))
}

// generateBackendID generates a unique backend ID
func generateBackendID() string {
	return fmt.Sprintf("backend-%d-%s", time.Now().UnixNano(), randomString(8))
}

// generateOperationID generates a unique operation ID
func generateOperationID() string {
	return fmt.Sprintf("operation-%d-%s", time.Now().UnixNano(), randomString(8))
}

// generateLockID generates a unique lock ID
func generateLockID() string {
	return fmt.Sprintf("lock-%d-%s", time.Now().UnixNano(), randomString(8))
}

// getCurrentUser gets the current user from context
func getCurrentUser(ctx context.Context) string {
	// This would typically extract user from JWT token or session
	// For now, return a default user
	if user := ctx.Value("user_id"); user != nil {
		return user.(string)
	}
	return "system"
}

// extractResourceType extracts the resource type from an address
func extractResourceType(address string) string {
	// Simple extraction - in real implementation, this would be more sophisticated
	// Example: "aws_instance.web" -> "aws_instance"
	parts := splitAddress(address)
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// extractProvider extracts the provider from an address
func extractProvider(address string) string {
	// Simple extraction - in real implementation, this would be more sophisticated
	// Example: "aws_instance.web" -> "aws"
	resourceType := extractResourceType(address)
	parts := splitAddress(resourceType)
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// splitAddress splits an address by dots
func splitAddress(address string) []string {
	// Simple implementation - in real implementation, this would handle more complex cases
	// For now, just return the address as a single part
	return []string{address}
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
