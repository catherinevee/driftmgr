package backend

import (
	"context"
	"io"
	"time"
)

// Backend defines the interface for state backend operations
type Backend interface {
	// Pull retrieves the current state from the backend
	Pull(ctx context.Context) (*StateData, error)
	
	// Push uploads state to the backend
	Push(ctx context.Context, state *StateData) error
	
	// Lock acquires a lock on the state for safe operations
	Lock(ctx context.Context, info *LockInfo) (string, error)
	
	// Unlock releases the lock on the state
	Unlock(ctx context.Context, lockID string) error
	
	// GetVersions returns available state versions/history
	GetVersions(ctx context.Context) ([]*StateVersion, error)
	
	// GetVersion retrieves a specific version of the state
	GetVersion(ctx context.Context, versionID string) (*StateData, error)
	
	// ListWorkspaces returns available workspaces
	ListWorkspaces(ctx context.Context) ([]string, error)
	
	// SelectWorkspace switches to a different workspace
	SelectWorkspace(ctx context.Context, name string) error
	
	// CreateWorkspace creates a new workspace
	CreateWorkspace(ctx context.Context, name string) error
	
	// DeleteWorkspace removes a workspace
	DeleteWorkspace(ctx context.Context, name string) error
	
	// GetLockInfo returns current lock information
	GetLockInfo(ctx context.Context) (*LockInfo, error)
	
	// Validate checks if the backend is properly configured and accessible
	Validate(ctx context.Context) error
	
	// GetMetadata returns backend metadata
	GetMetadata() *BackendMetadata
}

// StateData represents Terraform state data
type StateData struct {
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           uint64                 `json:"serial"`
	Lineage          string                 `json:"lineage"`
	Data             []byte                 `json:"-"`
	Resources        []StateResource        `json:"resources,omitempty"`
	Outputs          map[string]interface{} `json:"outputs,omitempty"`
	Backend          *BackendState          `json:"backend,omitempty"`
	Checksum         string                 `json:"checksum,omitempty"`
	LastModified     time.Time              `json:"last_modified"`
	Size             int64                  `json:"size"`
}

// StateResource represents a resource in the state
type StateResource struct {
	Mode      string                 `json:"mode"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Provider  string                 `json:"provider"`
	Instances []StateResourceInstance `json:"instances"`
}

// StateResourceInstance represents an instance of a resource
type StateResourceInstance struct {
	SchemaVersion       int                    `json:"schema_version"`
	Attributes          map[string]interface{} `json:"attributes"`
	Dependencies        []string               `json:"dependencies,omitempty"`
	CreateBeforeDestroy bool                   `json:"create_before_destroy,omitempty"`
}

// BackendState represents the backend configuration in state
type BackendState struct {
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	Hash      string                 `json:"hash"`
	Workspace string                 `json:"workspace,omitempty"`
}

// LockInfo represents state lock information
type LockInfo struct {
	ID        string    `json:"ID"`
	Path      string    `json:"Path"`
	Operation string    `json:"Operation"`
	Who       string    `json:"Who"`
	Version   string    `json:"Version"`
	Created   time.Time `json:"Created"`
	Info      string    `json:"Info"`
}

// StateVersion represents a version of the state
type StateVersion struct {
	ID           string    `json:"id"`
	VersionID    string    `json:"version_id"`
	Serial       uint64    `json:"serial"`
	Created      time.Time `json:"created"`
	CreatedBy    string    `json:"created_by,omitempty"`
	Size         int64     `json:"size"`
	Checksum     string    `json:"checksum,omitempty"`
	IsLatest     bool      `json:"is_latest"`
	Description  string    `json:"description,omitempty"`
}

// BackendMetadata contains metadata about the backend
type BackendMetadata struct {
	Type             string            `json:"type"`
	SupportsLocking  bool              `json:"supports_locking"`
	SupportsVersions bool              `json:"supports_versions"`
	SupportsWorkspaces bool            `json:"supports_workspaces"`
	Configuration    map[string]string `json:"configuration"`
	Workspace        string            `json:"workspace"`
	StateKey         string            `json:"state_key"`
	LockTable        string            `json:"lock_table,omitempty"`
}

// ConnectionPool manages backend connections
type ConnectionPool interface {
	// Get retrieves a connection from the pool
	Get(ctx context.Context) (io.Closer, error)
	
	// Put returns a connection to the pool
	Put(conn io.Closer)
	
	// Close closes all connections in the pool
	Close() error
	
	// Stats returns pool statistics
	Stats() *PoolStats
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	Active      int       `json:"active"`
	Idle        int       `json:"idle"`
	MaxOpen     int       `json:"max_open"`
	MaxIdle     int       `json:"max_idle"`
	WaitCount   int64     `json:"wait_count"`
	WaitDuration time.Duration `json:"wait_duration"`
	IdleTimeout time.Duration `json:"idle_timeout"`
	Created     int64     `json:"created"`
	Closed      int64     `json:"closed"`
}

// BackendFactory creates backend instances based on configuration
type BackendFactory interface {
	// CreateBackend creates a backend instance from configuration
	CreateBackend(config *BackendConfig) (Backend, error)
	
	// GetSupportedTypes returns list of supported backend types
	GetSupportedTypes() []string
	
	// ValidateConfig validates backend configuration
	ValidateConfig(config *BackendConfig) error
}

// BackendConfig represents backend configuration
type BackendConfig struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
	
	// Connection pool settings
	MaxConnections     int           `json:"max_connections,omitempty"`
	MaxIdleConnections int           `json:"max_idle_connections,omitempty"`
	ConnectionTimeout  time.Duration `json:"connection_timeout,omitempty"`
	IdleTimeout        time.Duration `json:"idle_timeout,omitempty"`
	
	// Retry settings
	MaxRetries     int           `json:"max_retries,omitempty"`
	RetryDelay     time.Duration `json:"retry_delay,omitempty"`
	RetryBackoff   float64       `json:"retry_backoff,omitempty"`
	
	// Lock settings
	LockTimeout    time.Duration `json:"lock_timeout,omitempty"`
	LockRetryDelay time.Duration `json:"lock_retry_delay,omitempty"`
}