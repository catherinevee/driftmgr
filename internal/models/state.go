package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

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

// StateFile represents a Terraform state file
type StateFile struct {
	ID           string                 `json:"id" db:"id" validate:"required,uuid"`
	BackendID    string                 `json:"backend_id" db:"backend_id" validate:"required,uuid"`
	Name         string                 `json:"name" db:"name" validate:"required,min=1,max=255"`
	Path         string                 `json:"path" db:"path" validate:"required"`
	Version      int                    `json:"version" db:"version" validate:"min=0"`
	Serial       int64                  `json:"serial" db:"serial" validate:"min=0"`
	Lineage      string                 `json:"lineage" db:"lineage" validate:"required"`
	Resources    []StateResource        `json:"resources,omitempty" db:"resources"`
	Outputs      map[string]interface{} `json:"outputs,omitempty" db:"outputs"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	Size         int64                  `json:"size" db:"size" validate:"min=0"`
	Checksum     string                 `json:"checksum" db:"checksum"`
	LastModified time.Time              `json:"last_modified" db:"last_modified"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// StateResource represents a resource in a Terraform state file
type StateResource struct {
	ID            string                 `json:"id" db:"id" validate:"required,uuid"`
	StateFileID   string                 `json:"state_file_id" db:"state_file_id" validate:"required,uuid"`
	Address       string                 `json:"address" db:"address" validate:"required"`
	Type          string                 `json:"type" db:"type" validate:"required"`
	Provider      string                 `json:"provider" db:"provider" validate:"required"`
	Instance      string                 `json:"instance" db:"instance"`
	Attributes    map[string]interface{} `json:"attributes" db:"attributes"`
	Dependencies  []string               `json:"dependencies,omitempty" db:"dependencies"`
	DependsOn     []string               `json:"depends_on,omitempty" db:"depends_on"`
	Module        string                 `json:"module,omitempty" db:"module"`
	Mode          string                 `json:"mode" db:"mode" validate:"required,oneof=managed data"`
	SchemaVersion int                    `json:"schema_version" db:"schema_version" validate:"min=0"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// Backend represents a Terraform backend configuration
type Backend struct {
	ID            string                 `json:"id" db:"id" validate:"required,uuid"`
	EnvironmentID string                 `json:"environment_id" db:"environment_id" validate:"required,uuid"`
	Type          BackendType            `json:"type" db:"type" validate:"required,oneof=s3 azurerm gcs local"`
	Name          string                 `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description   string                 `json:"description" db:"description" validate:"max=1000"`
	Configuration map[string]interface{} `json:"configuration" db:"configuration" validate:"required"`
	IsActive      bool                   `json:"is_active" db:"is_active"`
	IsDefault     bool                   `json:"is_default" db:"is_default"`
	CreatedBy     string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// BackendType represents the type of Terraform backend
type BackendType string

const (
	BackendTypeS3    BackendType = "s3"
	BackendTypeAzure BackendType = "azurerm"
	BackendTypeGCS   BackendType = "gcs"
	BackendTypeLocal BackendType = "local"
)

// String returns the string representation of BackendType
func (bt BackendType) String() string {
	return string(bt)
}

// StateOperation represents a state file operation
type StateOperation struct {
	ID            string                 `json:"id" db:"id" validate:"required,uuid"`
	StateFileID   string                 `json:"state_file_id" db:"state_file_id" validate:"required,uuid"`
	OperationType StateOperationType     `json:"operation_type" db:"operation_type" validate:"required,oneof=import remove move list get lock unlock"`
	Status        OperationStatus        `json:"status" db:"status" validate:"required,oneof=pending running completed failed"`
	Parameters    map[string]interface{} `json:"parameters" db:"parameters"`
	Result        map[string]interface{} `json:"result,omitempty" db:"result"`
	Error         *string                `json:"error,omitempty" db:"error"`
	StartedAt     *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CreatedBy     string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// StateOperationType represents the type of state operation
type StateOperationType string

const (
	StateOperationImport StateOperationType = "import"
	StateOperationRemove StateOperationType = "remove"
	StateOperationMove   StateOperationType = "move"
	StateOperationList   StateOperationType = "list"
	StateOperationGet    StateOperationType = "get"
	StateOperationLock   StateOperationType = "lock"
	StateOperationUnlock StateOperationType = "unlock"
)

// String returns the string representation of StateOperationType
func (sot StateOperationType) String() string {
	return string(sot)
}

// OperationStatus represents the status of an operation
type OperationStatus string

const (
	OperationStatusPending   OperationStatus = "pending"
	OperationStatusRunning   OperationStatus = "running"
	OperationStatusCompleted OperationStatus = "completed"
	OperationStatusFailed    OperationStatus = "failed"
)

// String returns the string representation of OperationStatus
func (os OperationStatus) String() string {
	return string(os)
}

// StateFileListRequest represents a request to list state files
type StateFileListRequest struct {
	BackendID     *string    `json:"backend_id,omitempty" validate:"omitempty,uuid"`
	EnvironmentID *string    `json:"environment_id,omitempty" validate:"omitempty,uuid"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	Limit         int        `json:"limit" validate:"min=1,max=1000"`
	Offset        int        `json:"offset" validate:"min=0"`
	SortBy        string     `json:"sort_by" validate:"omitempty,oneof=name last_modified created_at size"`
	SortOrder     string     `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// StateFileListResponse represents the response for listing state files
type StateFileListResponse struct {
	StateFiles []StateFile `json:"state_files"`
	Total      int         `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
}

// StateFileResponse represents the response for a state file
type StateFileResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Version       int       `json:"version"`
	Serial        int64     `json:"serial"`
	Lineage       string    `json:"lineage"`
	ResourceCount int       `json:"resource_count"`
	Size          int64     `json:"size"`
	LastModified  time.Time `json:"last_modified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// StateResourceListRequest represents a request to list state resources
type StateResourceListRequest struct {
	StateFileID  *string    `json:"state_file_id,omitempty" validate:"omitempty,uuid"`
	ResourceType *string    `json:"resource_type,omitempty"`
	Provider     *string    `json:"provider,omitempty"`
	Module       *string    `json:"module,omitempty"`
	Address      *string    `json:"address,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Limit        int        `json:"limit" validate:"min=1,max=1000"`
	Offset       int        `json:"offset" validate:"min=0"`
	SortBy       string     `json:"sort_by" validate:"omitempty,oneof=address type provider created_at"`
	SortOrder    string     `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// StateResourceListResponse represents the response for listing state resources
type StateResourceListResponse struct {
	Resources []StateResource `json:"resources"`
	Total     int             `json:"total"`
	Limit     int             `json:"limit"`
	Offset    int             `json:"offset"`
}

// ResourceResponse represents the response for a resource
type ResourceResponse struct {
	ID            string                 `json:"id"`
	StateFileID   string                 `json:"state_file_id"`
	Address       string                 `json:"address"`
	Type          string                 `json:"type"`
	Provider      string                 `json:"provider"`
	Instance      string                 `json:"instance"`
	Attributes    map[string]interface{} `json:"attributes"`
	Dependencies  []string               `json:"dependencies,omitempty"`
	DependsOn     []string               `json:"depends_on,omitempty"`
	Module        string                 `json:"module,omitempty"`
	Mode          string                 `json:"mode"`
	SchemaVersion int                    `json:"schema_version"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// BackendListRequest represents a request to list backends
type BackendListRequest struct {
	EnvironmentID *string      `json:"environment_id,omitempty" validate:"omitempty,uuid"`
	Type          *BackendType `json:"type,omitempty" validate:"omitempty,oneof=s3 azurerm gcs local"`
	IsActive      *bool        `json:"is_active,omitempty"`
	Limit         int          `json:"limit" validate:"min=1,max=1000"`
	Offset        int          `json:"offset" validate:"min=0"`
	SortBy        string       `json:"sort_by" validate:"omitempty,oneof=name type created_at"`
	SortOrder     string       `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// BackendListResponse represents the response for listing backends
type BackendListResponse struct {
	Backends []Backend `json:"backends"`
	Total    int       `json:"total"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}

// BackendResponse represents the response for a backend
type BackendResponse struct {
	ID            string                 `json:"id"`
	EnvironmentID string                 `json:"environment_id"`
	Type          BackendType            `json:"type"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Configuration map[string]interface{} `json:"configuration"`
	IsActive      bool                   `json:"is_active"`
	IsDefault     bool                   `json:"is_default"`
	CreatedBy     string                 `json:"created_by"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// BackendCreateRequest represents a request to create a backend
type BackendCreateRequest struct {
	EnvironmentID string                 `json:"environment_id" validate:"required,uuid"`
	Type          BackendType            `json:"type" validate:"required,oneof=s3 azurerm gcs local"`
	Name          string                 `json:"name" validate:"required,min=1,max=255"`
	Description   string                 `json:"description" validate:"max=1000"`
	Configuration map[string]interface{} `json:"configuration" validate:"required"`
	IsDefault     bool                   `json:"is_default"`
}

// ImportResourceRequest represents a request to import a resource
type ImportResourceRequest struct {
	ResourceAddress string                 `json:"resource_address" validate:"required"`
	ResourceID      string                 `json:"resource_id" validate:"required"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`
}

// ImportResourceResponse represents the response for importing a resource
type ImportResourceResponse struct {
	ResourceID      string    `json:"resource_id"`
	ResourceAddress string    `json:"resource_address"`
	Status          string    `json:"status"`
	Message         string    `json:"message"`
	ImportedAt      time.Time `json:"imported_at"`
}

// RemoveResourceRequest represents a request to remove a resource
type RemoveResourceRequest struct {
	ResourceAddress string `json:"resource_address" validate:"required"`
	Force           bool   `json:"force"`
}

// RemoveResourceResponse represents the response for removing a resource
type RemoveResourceResponse struct {
	ResourceAddress string    `json:"resource_address"`
	Status          string    `json:"status"`
	Message         string    `json:"message"`
	RemovedAt       time.Time `json:"removed_at"`
}

// MoveResourceRequest represents a request to move a resource
type MoveResourceRequest struct {
	FromAddress string `json:"from_address" validate:"required"`
	ToAddress   string `json:"to_address" validate:"required"`
}

// MoveResourceResponse represents the response for moving a resource
type MoveResourceResponse struct {
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	MovedAt     time.Time `json:"moved_at"`
}

// ExportResourceRequest represents a request to export a resource
type ExportResourceRequest struct {
	Format string `json:"format" validate:"required,oneof=json hcl yaml"`
}

// ExportResourceResponse represents the response for exporting a resource
type ExportResourceResponse struct {
	ResourceID      string                 `json:"resource_id"`
	ResourceAddress string                 `json:"resource_address"`
	Format          string                 `json:"format"`
	Configuration   map[string]interface{} `json:"configuration"`
	ExportedAt      time.Time              `json:"exported_at"`
}

// StateLock represents a state file lock
type StateLock struct {
	ID          string    `json:"id" db:"id" validate:"required,uuid"`
	StateFileID string    `json:"state_file_id" db:"state_file_id" validate:"required,uuid"`
	LockID      string    `json:"lock_id" db:"lock_id" validate:"required"`
	Operation   string    `json:"operation" db:"operation" validate:"required"`
	Who         string    `json:"who" db:"who" validate:"required"`
	Version     string    `json:"version" db:"version"`
	Created     time.Time `json:"created" db:"created"`
	Path        string    `json:"path" db:"path"`
	Info        string    `json:"info,omitempty" db:"info"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// StateLockRequest represents a request to lock a state file
type StateLockRequest struct {
	Operation string `json:"operation" validate:"required"`
	Info      string `json:"info,omitempty"`
}

// StateLockResponse represents the response for state locking
type StateLockResponse struct {
	LockID   string    `json:"lock_id"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	LockedAt time.Time `json:"locked_at"`
}

// StateUnlockRequest represents a request to unlock a state file
type StateUnlockRequest struct {
	LockID string `json:"lock_id" validate:"required"`
	Force  bool   `json:"force"`
}

// StateUnlockResponse represents the response for state unlocking
type StateUnlockResponse struct {
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	UnlockedAt time.Time `json:"unlocked_at"`
}

// Validation methods

// Validate validates the StateFile struct
func (sf *StateFile) Validate() error {
	validate := validator.New()
	return validate.Struct(sf)
}

// Validate validates the StateResource struct
func (sr *StateResource) Validate() error {
	validate := validator.New()
	return validate.Struct(sr)
}

// Validate validates the Backend struct
func (b *Backend) Validate() error {
	validate := validator.New()
	return validate.Struct(b)
}

// Validate validates the StateOperation struct
func (so *StateOperation) Validate() error {
	validate := validator.New()
	return validate.Struct(so)
}

// Validate validates the StateFileListRequest struct
func (sflr *StateFileListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(sflr)
}

// Validate validates the StateResourceListRequest struct
func (srlr *StateResourceListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(srlr)
}

// Validate validates the BackendListRequest struct
func (blr *BackendListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(blr)
}

// Validate validates the BackendCreateRequest struct
func (bcr *BackendCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(bcr)
}

// Validate validates the ImportResourceRequest struct
func (irr *ImportResourceRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(irr)
}

// Validate validates the RemoveResourceRequest struct
func (rrr *RemoveResourceRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rrr)
}

// Validate validates the MoveResourceRequest struct
func (mrr *MoveResourceRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(mrr)
}

// Validate validates the ExportResourceRequest struct
func (err *ExportResourceRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(err)
}

// Validate validates the StateLock struct
func (sl *StateLock) Validate() error {
	validate := validator.New()
	return validate.Struct(sl)
}

// Validate validates the StateLockRequest struct
func (slr *StateLockRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(slr)
}

// Validate validates the StateUnlockRequest struct
func (sur *StateUnlockRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(sur)
}

// Helper methods

// IsLocked returns true if the state file is locked
func (sf *StateFile) IsLocked() bool {
	// This would check if there's an active lock for this state file
	// Implementation would depend on the locking mechanism
	return false
}

// GetResourceCount returns the number of resources in the state file
func (sf *StateFile) GetResourceCount() int {
	return len(sf.Resources)
}

// GetResourceByAddress returns a resource by its address
func (sf *StateFile) GetResourceByAddress(address string) *StateResource {
	for i := range sf.Resources {
		if sf.Resources[i].Address == address {
			return &sf.Resources[i]
		}
	}
	return nil
}

// AddResource adds a resource to the state file
func (sf *StateFile) AddResource(resource StateResource) {
	sf.Resources = append(sf.Resources, resource)
	sf.UpdatedAt = time.Now()
}

// RemoveResource removes a resource from the state file
func (sf *StateFile) RemoveResource(address string) bool {
	for i, resource := range sf.Resources {
		if resource.Address == address {
			sf.Resources = append(sf.Resources[:i], sf.Resources[i+1:]...)
			sf.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// UpdateResource updates a resource in the state file
func (sf *StateFile) UpdateResource(address string, updatedResource StateResource) bool {
	for i, resource := range sf.Resources {
		if resource.Address == address {
			sf.Resources[i] = updatedResource
			sf.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// IsActiveBackend returns true if the backend is active
func (b *Backend) IsActiveBackend() bool {
	return b.IsActive
}

// IsDefaultBackend returns true if the backend is the default backend
func (b *Backend) IsDefaultBackend() bool {
	return b.IsDefault
}

// GetConfigurationValue returns a configuration value by key
func (b *Backend) GetConfigurationValue(key string) interface{} {
	if b.Configuration == nil {
		return nil
	}
	return b.Configuration[key]
}

// SetConfigurationValue sets a configuration value
func (b *Backend) SetConfigurationValue(key string, value interface{}) {
	if b.Configuration == nil {
		b.Configuration = make(map[string]interface{})
	}
	b.Configuration[key] = value
	b.UpdatedAt = time.Now()
}

// IsCompleted returns true if the operation is completed
func (so *StateOperation) IsCompleted() bool {
	return so.Status == OperationStatusCompleted || so.Status == OperationStatusFailed
}

// IsRunning returns true if the operation is running
func (so *StateOperation) IsRunning() bool {
	return so.Status == OperationStatusRunning
}

// SetStatus updates the operation status
func (so *StateOperation) SetStatus(status OperationStatus) {
	so.Status = status
	so.UpdatedAt = time.Now()

	now := time.Now()
	switch status {
	case OperationStatusRunning:
		so.StartedAt = &now
	case OperationStatusCompleted, OperationStatusFailed:
		so.CompletedAt = &now
	}
}

// SetError sets the operation error
func (so *StateOperation) SetError(err error) {
	if err != nil {
		errStr := err.Error()
		so.Error = &errStr
		so.SetStatus(OperationStatusFailed)
	}
}

// SetResult sets the operation result
func (so *StateOperation) SetResult(result map[string]interface{}) {
	so.Result = result
	so.SetStatus(OperationStatusCompleted)
}

// StateFileStatistics represents statistics for state files
type StateFileStatistics struct {
	TotalFiles     int            `json:"total_files"`
	TotalSize      int64          `json:"total_size"`
	AverageSize    int64          `json:"average_size"`
	FilesByBackend map[string]int `json:"files_by_backend"`
	FilesByDate    map[string]int `json:"files_by_date"`
}

// ResourceStatistics represents statistics for resources
type ResourceStatistics struct {
	TotalResources      int            `json:"total_resources"`
	ResourcesByType     map[string]int `json:"resources_by_type"`
	ResourcesByProvider map[string]int `json:"resources_by_provider"`
	ResourcesByModule   map[string]int `json:"resources_by_module"`
}

// HealthResponse represents the health status of the state management service
type HealthResponse struct {
	Status           string    `json:"status"`
	Service          string    `json:"service"`
	StateFileCount   int       `json:"state_file_count"`
	ResourceCount    int       `json:"resource_count"`
	BackendCount     int       `json:"backend_count"`
	ActiveLocksCount int       `json:"active_locks_count"`
	CheckedAt        time.Time `json:"checked_at"`
}
