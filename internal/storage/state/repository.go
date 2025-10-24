package state

import (
	"context"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Repository defines the interface for state management data operations
type Repository interface {
	// State file management
	CreateStateFile(ctx context.Context, stateFile *models.StateFile) error
	GetStateFileByID(ctx context.Context, id string) (*models.StateFile, error)
	GetStateFileByPath(ctx context.Context, path string) (*models.StateFile, error)
	UpdateStateFile(ctx context.Context, stateFile *models.StateFile) error
	DeleteStateFile(ctx context.Context, id string) error
	ListStateFiles(ctx context.Context, req *models.StateFileListRequest) (*models.StateFileListResponse, error)

	// State file filtering and search
	GetStateFilesByBackend(ctx context.Context, backendID string) ([]models.StateFile, error)
	GetStateFilesByEnvironment(ctx context.Context, environmentID string) ([]models.StateFile, error)
	GetStateFilesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.StateFile, error)

	// Resource management
	CreateResource(ctx context.Context, resource *models.StateResource) error
	GetResourceByID(ctx context.Context, id string) (*models.StateResource, error)
	GetResourceByAddress(ctx context.Context, stateFileID, address string) (*models.StateResource, error)
	UpdateResource(ctx context.Context, resource *models.StateResource) error
	DeleteResource(ctx context.Context, id string) error
	ListResources(ctx context.Context, req *models.ResourceListRequest) (*models.ResourceListResponse, error)

	// Resource filtering and search
	GetResourcesByStateFile(ctx context.Context, stateFileID string) ([]models.StateResource, error)
	GetResourcesByType(ctx context.Context, resourceType string) ([]models.StateResource, error)
	GetResourcesByProvider(ctx context.Context, provider string) ([]models.StateResource, error)
	GetResourcesByModule(ctx context.Context, module string) ([]models.StateResource, error)

	// Backend management
	CreateBackend(ctx context.Context, backend *models.Backend) error
	GetBackendByID(ctx context.Context, id string) (*models.Backend, error)
	GetBackendByName(ctx context.Context, name string) (*models.Backend, error)
	UpdateBackend(ctx context.Context, backend *models.Backend) error
	DeleteBackend(ctx context.Context, id string) error
	ListBackends(ctx context.Context, req *models.BackendListRequest) (*models.BackendListResponse, error)

	// Backend filtering and search
	GetBackendsByEnvironment(ctx context.Context, environmentID string) ([]models.Backend, error)
	GetBackendsByType(ctx context.Context, backendType models.BackendType) ([]models.Backend, error)
	GetDefaultBackend(ctx context.Context, environmentID string) (*models.Backend, error)

	// State operations
	CreateStateOperation(ctx context.Context, operation *models.StateOperation) error
	GetStateOperationByID(ctx context.Context, id string) (*models.StateOperation, error)
	UpdateStateOperation(ctx context.Context, operation *models.StateOperation) error
	DeleteStateOperation(ctx context.Context, id string) error
	ListStateOperations(ctx context.Context, stateFileID string) ([]models.StateOperation, error)

	// State locking
	CreateStateLock(ctx context.Context, lock *models.StateLock) error
	GetStateLockByID(ctx context.Context, id string) (*models.StateLock, error)
	GetStateLockByStateFile(ctx context.Context, stateFileID string) (*models.StateLock, error)
	DeleteStateLock(ctx context.Context, id string) error
	DeleteStateLockByStateFile(ctx context.Context, stateFileID string) error

	// History and analytics
	GetStateFileHistory(ctx context.Context, stateFileID string) ([]models.StateFile, error)
	GetResourceHistory(ctx context.Context, resourceID string) ([]models.StateResource, error)
	GetStateFileStatistics(ctx context.Context, startDate, endDate time.Time) (*models.StateFileStatistics, error)
	GetResourceStatistics(ctx context.Context, startDate, endDate time.Time) (*models.ResourceStatistics, error)

	// Cleanup and maintenance
	DeleteOldStateFiles(ctx context.Context, olderThan time.Time) error
	DeleteOldOperations(ctx context.Context, olderThan time.Time) error
	DeleteOldLocks(ctx context.Context, olderThan time.Time) error
	CleanupOrphanedResources(ctx context.Context) error

	// Health and monitoring
	Health(ctx context.Context) error
	GetStateFileCount(ctx context.Context) (int, error)
	GetResourceCount(ctx context.Context) (int, error)
	GetBackendCount(ctx context.Context) (int, error)
	GetActiveLocksCount(ctx context.Context) (int, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(context.Context) error) error

	// Close connection
	Close() error
}

// RepositoryConfig holds configuration for the repository
type RepositoryConfig struct {
	DatabaseURL    string        `json:"database_url"`
	MaxConns       int           `json:"max_conns"`
	MinConns       int           `json:"min_conns"`
	MaxLifetime    time.Duration `json:"max_lifetime"`
	MaxIdleTime    time.Duration `json:"max_idle_time"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
	QueryTimeout   time.Duration `json:"query_timeout"`
}

// StateFileStatistics represents statistics about state files
type StateFileStatistics struct {
	TotalStateFiles      int                   `json:"total_state_files"`
	TotalResources       int                   `json:"total_resources"`
	TotalSize            int64                 `json:"total_size"`
	AverageSize          int64                 `json:"average_size"`
	StateFilesByBackend  map[string]int        `json:"state_files_by_backend"`
	ResourcesByType      map[string]int        `json:"resources_by_type"`
	ResourcesByProvider  map[string]int        `json:"resources_by_provider"`
	TopStateFiles        []StateFileSize       `json:"top_state_files"`
	TopResources         []ResourceCount       `json:"top_resources"`
	DailyStateFileCounts []DailyStateFileCount `json:"daily_state_file_counts"`
}

// ResourceStatistics represents statistics about resources
type ResourceStatistics struct {
	TotalResources      int                  `json:"total_resources"`
	ResourcesByType     map[string]int       `json:"resources_by_type"`
	ResourcesByProvider map[string]int       `json:"resources_by_provider"`
	ResourcesByModule   map[string]int       `json:"resources_by_module"`
	TopResourceTypes    []ResourceTypeCount  `json:"top_resource_types"`
	TopProviders        []ProviderCount      `json:"top_providers"`
	DailyResourceCounts []DailyResourceCount `json:"daily_resource_counts"`
}

// StateFileSize represents state file size information
type StateFileSize struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	ResourceCount int    `json:"resource_count"`
}

// ResourceCount represents resource count information
type ResourceCount struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Count   int    `json:"count"`
}

// ResourceTypeCount represents resource type count
type ResourceTypeCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// ProviderCount represents provider count
type ProviderCount struct {
	Provider string `json:"provider"`
	Count    int    `json:"count"`
}

// DailyStateFileCount represents daily state file count
type DailyStateFileCount struct {
	Date      time.Time `json:"date"`
	Count     int       `json:"count"`
	TotalSize int64     `json:"total_size"`
}

// DailyResourceCount represents daily resource count
type DailyResourceCount struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

// RepositoryFactory creates repository instances
type RepositoryFactory interface {
	CreateRepository(config *RepositoryConfig) (Repository, error)
}

// DefaultRepositoryFactory is the default implementation of RepositoryFactory
type DefaultRepositoryFactory struct{}

// CreateRepository creates a new repository instance
func (f *DefaultRepositoryFactory) CreateRepository(config *RepositoryConfig) (Repository, error) {
	// PostgreSQL repository implementation available in Phase 4 documentation
	return nil, models.ErrNotImplemented
}
