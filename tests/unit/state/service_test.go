package state

import (
	"context"
	"testing"
	"time"

	statebusiness "github.com/catherinevee/driftmgr/internal/business/state"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the state repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateStateFile(ctx context.Context, stateFile *models.StateFile) error {
	args := m.Called(ctx, stateFile)
	return args.Error(0)
}

func (m *MockRepository) GetStateFileByID(ctx context.Context, id string) (*models.StateFile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateFile), args.Error(1)
}

func (m *MockRepository) GetStateFileByPath(ctx context.Context, path string) (*models.StateFile, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateFile), args.Error(1)
}

func (m *MockRepository) UpdateStateFile(ctx context.Context, stateFile *models.StateFile) error {
	args := m.Called(ctx, stateFile)
	return args.Error(0)
}

func (m *MockRepository) DeleteStateFile(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListStateFiles(ctx context.Context, req *models.StateFileListRequest) (*models.StateFileListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateFileListResponse), args.Error(1)
}

func (m *MockRepository) GetStateFilesByBackend(ctx context.Context, backendID string) ([]models.StateFile, error) {
	args := m.Called(ctx, backendID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateFile), args.Error(1)
}

func (m *MockRepository) GetStateFilesByEnvironment(ctx context.Context, environmentID string) ([]models.StateFile, error) {
	args := m.Called(ctx, environmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateFile), args.Error(1)
}

func (m *MockRepository) GetStateFilesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.StateFile, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateFile), args.Error(1)
}

func (m *MockRepository) CreateResource(ctx context.Context, resource *models.StateResource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func (m *MockRepository) GetResourceByID(ctx context.Context, id string) (*models.StateResource, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateResource), args.Error(1)
}

func (m *MockRepository) GetResourceByAddress(ctx context.Context, stateFileID, address string) (*models.StateResource, error) {
	args := m.Called(ctx, stateFileID, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateResource), args.Error(1)
}

func (m *MockRepository) UpdateResource(ctx context.Context, resource *models.StateResource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func (m *MockRepository) DeleteResource(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListResources(ctx context.Context, req *models.ResourceListRequest) (*models.ResourceListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ResourceListResponse), args.Error(1)
}

func (m *MockRepository) GetResourcesByStateFile(ctx context.Context, stateFileID string) ([]models.StateResource, error) {
	args := m.Called(ctx, stateFileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateResource), args.Error(1)
}

func (m *MockRepository) GetResourcesByType(ctx context.Context, resourceType string) ([]models.StateResource, error) {
	args := m.Called(ctx, resourceType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateResource), args.Error(1)
}

func (m *MockRepository) GetResourcesByProvider(ctx context.Context, provider string) ([]models.StateResource, error) {
	args := m.Called(ctx, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateResource), args.Error(1)
}

func (m *MockRepository) GetResourcesByModule(ctx context.Context, module string) ([]models.StateResource, error) {
	args := m.Called(ctx, module)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateResource), args.Error(1)
}

func (m *MockRepository) CreateBackend(ctx context.Context, backend *models.Backend) error {
	args := m.Called(ctx, backend)
	return args.Error(0)
}

func (m *MockRepository) GetBackendByID(ctx context.Context, id string) (*models.Backend, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backend), args.Error(1)
}

func (m *MockRepository) GetBackendByName(ctx context.Context, name string) (*models.Backend, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backend), args.Error(1)
}

func (m *MockRepository) UpdateBackend(ctx context.Context, backend *models.Backend) error {
	args := m.Called(ctx, backend)
	return args.Error(0)
}

func (m *MockRepository) DeleteBackend(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListBackends(ctx context.Context, req *models.BackendListRequest) (*models.BackendListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BackendListResponse), args.Error(1)
}

func (m *MockRepository) GetBackendsByEnvironment(ctx context.Context, environmentID string) ([]models.Backend, error) {
	args := m.Called(ctx, environmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Backend), args.Error(1)
}

func (m *MockRepository) GetBackendsByType(ctx context.Context, backendType models.BackendType) ([]models.Backend, error) {
	args := m.Called(ctx, backendType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Backend), args.Error(1)
}

func (m *MockRepository) GetDefaultBackend(ctx context.Context, environmentID string) (*models.Backend, error) {
	args := m.Called(ctx, environmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backend), args.Error(1)
}

func (m *MockRepository) CreateStateOperation(ctx context.Context, operation *models.StateOperation) error {
	args := m.Called(ctx, operation)
	return args.Error(0)
}

func (m *MockRepository) GetStateOperationByID(ctx context.Context, id string) (*models.StateOperation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateOperation), args.Error(1)
}

func (m *MockRepository) UpdateStateOperation(ctx context.Context, operation *models.StateOperation) error {
	args := m.Called(ctx, operation)
	return args.Error(0)
}

func (m *MockRepository) DeleteStateOperation(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListStateOperations(ctx context.Context, stateFileID string) ([]models.StateOperation, error) {
	args := m.Called(ctx, stateFileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateOperation), args.Error(1)
}

func (m *MockRepository) CreateStateLock(ctx context.Context, lock *models.StateLock) error {
	args := m.Called(ctx, lock)
	return args.Error(0)
}

func (m *MockRepository) GetStateLockByID(ctx context.Context, id string) (*models.StateLock, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateLock), args.Error(1)
}

func (m *MockRepository) GetStateLockByStateFile(ctx context.Context, stateFileID string) (*models.StateLock, error) {
	args := m.Called(ctx, stateFileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateLock), args.Error(1)
}

func (m *MockRepository) DeleteStateLock(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeleteStateLockByStateFile(ctx context.Context, stateFileID string) error {
	args := m.Called(ctx, stateFileID)
	return args.Error(0)
}

func (m *MockRepository) GetStateFileHistory(ctx context.Context, stateFileID string) ([]models.StateFile, error) {
	args := m.Called(ctx, stateFileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateFile), args.Error(1)
}

func (m *MockRepository) GetResourceHistory(ctx context.Context, resourceID string) ([]models.StateResource, error) {
	args := m.Called(ctx, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StateResource), args.Error(1)
}

func (m *MockRepository) GetStateFileStatistics(ctx context.Context, startDate, endDate time.Time) (*models.StateFileStatistics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StateFileStatistics), args.Error(1)
}

func (m *MockRepository) GetResourceStatistics(ctx context.Context, startDate, endDate time.Time) (*models.ResourceStatistics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ResourceStatistics), args.Error(1)
}

func (m *MockRepository) DeleteOldStateFiles(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockRepository) DeleteOldOperations(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockRepository) DeleteOldLocks(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockRepository) CleanupOrphanedResources(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) GetStateFileCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetResourceCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetBackendCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetActiveLocksCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestService_ListStateFiles(t *testing.T) {
	mockRepo := &MockRepository{}
	service := statebusiness.NewService(mockRepo, nil)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		req := &models.StateFileListRequest{
			Limit:  50,
			Offset: 0,
		}
		response := &models.StateFileListResponse{
			StateFiles: []models.StateFile{},
			Total:      0,
			Limit:      50,
			Offset:     0,
		}
		mockRepo.On("ListStateFiles", ctx, req).Return(response, nil)

		// Execute
		result, err := service.ListStateFiles(ctx, req)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.Total)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		// Setup mock expectations
		req := &models.StateFileListRequest{
			Limit:  50,
			Offset: 0,
		}
		mockRepo.On("ListStateFiles", ctx, req).Return(nil, models.ErrDatabaseConnection)

		// Execute
		result, err := service.ListStateFiles(ctx, req)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, models.ErrDatabaseConnection, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetStateFile(t *testing.T) {
	mockRepo := &MockRepository{}
	service := statebusiness.NewService(mockRepo, nil)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		stateFileID := "550e8400-e29b-41d4-a716-446655440000"
		stateFile := &models.StateFile{
			ID:        stateFileID,
			Name:      "test-state",
			Path:      "/path/to/state.tfstate",
			Version:   1,
			Serial:    1,
			Lineage:   "test-lineage",
			Size:      1024,
			Checksum:  "test-checksum",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mockRepo.On("GetStateFileByID", ctx, stateFileID).Return(stateFile, nil)

		// Execute
		result, err := service.GetStateFile(ctx, stateFileID)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, stateFileID, result.ID)
		assert.Equal(t, "test-state", result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		// Setup mock expectations
		stateFileID := "nonexistent-id"
		mockRepo.On("GetStateFileByID", ctx, stateFileID).Return(nil, models.ErrStateFileNotFound)

		// Execute
		result, err := service.GetStateFile(ctx, stateFileID)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, models.ErrStateFileNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_ImportResource(t *testing.T) {
	mockRepo := &MockRepository{}
	service := statebusiness.NewService(mockRepo, nil)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		// Setup test data
		stateFileID := "550e8400-e29b-41d4-a716-446655440000"
		importReq := &models.ImportResourceRequest{
			ResourceAddress: "aws_instance.test",
			ResourceID:      "i-1234567890abcdef0",
			Configuration:   map[string]interface{}{"instance_type": "t2.micro"},
		}

		// Setup mock expectations
		stateFile := &models.StateFile{
			ID:        stateFileID,
			Name:      "test-state",
			Path:      "/path/to/state.tfstate",
			Version:   1,
			Serial:    1,
			Lineage:   "test-lineage",
			Size:      1024,
			Checksum:  "test-checksum",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mockRepo.On("GetStateFileByID", ctx, stateFileID).Return(stateFile, nil)
		mockRepo.On("GetStateLockByStateFile", ctx, stateFileID).Return(nil, models.ErrStateLockNotFound)
		mockRepo.On("GetResourceByAddress", ctx, stateFileID, "aws_instance.test").Return(nil, models.ErrResourceNotFound)
		mockRepo.On("CreateResource", ctx, mock.AnythingOfType("*models.StateResource")).Return(nil)
		mockRepo.On("UpdateStateFile", ctx, mock.AnythingOfType("*models.StateFile")).Return(nil)
		mockRepo.On("CreateStateOperation", ctx, mock.AnythingOfType("*models.StateOperation")).Return(nil)

		// Execute
		result, err := service.ImportResource(ctx, stateFileID, importReq)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "aws_instance.test", result.ResourceAddress)
		assert.Equal(t, "imported", result.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("state file not found", func(t *testing.T) {
		// Setup test data
		stateFileID := "nonexistent-id"
		importReq := &models.ImportResourceRequest{
			ResourceAddress: "aws_instance.test",
			ResourceID:      "i-1234567890abcdef0",
			Configuration:   map[string]interface{}{"instance_type": "t2.micro"},
		}

		// Setup mock expectations
		mockRepo.On("GetStateFileByID", ctx, stateFileID).Return(nil, models.ErrStateFileNotFound)

		// Execute
		result, err := service.ImportResource(ctx, stateFileID, importReq)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, models.ErrStateFileNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("state file locked", func(t *testing.T) {
		// Setup test data
		stateFileID := "550e8400-e29b-41d4-a716-446655440000"
		importReq := &models.ImportResourceRequest{
			ResourceAddress: "aws_instance.test",
			ResourceID:      "i-1234567890abcdef0",
			Configuration:   map[string]interface{}{"instance_type": "t2.micro"},
		}

		// Setup mock expectations
		stateFile := &models.StateFile{
			ID:        stateFileID,
			Name:      "test-state",
			Path:      "/path/to/state.tfstate",
			Version:   1,
			Serial:    1,
			Lineage:   "test-lineage",
			Size:      1024,
			Checksum:  "test-checksum",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		lock := &models.StateLock{
			ID:          "550e8400-e29b-41d4-a716-446655440005",
			StateFileID: stateFileID,
			LockID:      "test-lock-id",
			Operation:   "plan",
			Who:         "test-user",
			Version:     "1.0.0",
			Created:     time.Now(),
			Path:        "/path/to/state.tfstate",
		}
		mockRepo.On("GetStateFileByID", ctx, stateFileID).Return(stateFile, nil)
		mockRepo.On("GetStateLockByStateFile", ctx, stateFileID).Return(lock, nil)

		// Execute
		result, err := service.ImportResource(ctx, stateFileID, importReq)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, models.ErrStateFileLocked, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_CreateBackend(t *testing.T) {
	mockRepo := &MockRepository{}
	service := statebusiness.NewService(mockRepo, nil)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		// Setup test data
		createReq := &models.BackendCreateRequest{
			EnvironmentID: "550e8400-e29b-41d4-a716-446655440003",
			Type:          models.BackendTypeS3,
			Name:          "test-backend",
			Description:   "Test backend",
			Configuration: map[string]interface{}{
				"bucket": "test-bucket",
				"key":    "test-key",
			},
			IsDefault: false,
		}

		// Setup mock expectations
		mockRepo.On("GetBackendByName", ctx, "test-backend").Return(nil, models.ErrBackendNotFound)
		mockRepo.On("CreateBackend", ctx, mock.AnythingOfType("*models.Backend")).Return(nil)

		// Execute
		result, err := service.CreateBackend(ctx, createReq)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-backend", result.Name)
		assert.Equal(t, models.BackendTypeS3, result.Type)
		mockRepo.AssertExpectations(t)
	})

	t.Run("backend already exists", func(t *testing.T) {
		// Setup test data
		createReq := &models.BackendCreateRequest{
			EnvironmentID: "550e8400-e29b-41d4-a716-446655440003",
			Type:          models.BackendTypeS3,
			Name:          "existing-backend",
			Description:   "Existing backend",
			Configuration: map[string]interface{}{
				"bucket": "test-bucket",
				"key":    "test-key",
			},
			IsDefault: false,
		}

		// Setup mock expectations
		existingBackend := &models.Backend{
			ID:            "550e8400-e29b-41d4-a716-446655440001",
			EnvironmentID: "550e8400-e29b-41d4-a716-446655440003",
			Type:          models.BackendTypeS3,
			Name:          "existing-backend",
			Description:   "Existing backend",
			Configuration: map[string]interface{}{
				"bucket": "test-bucket",
				"key":    "test-key",
			},
			IsActive:  true,
			IsDefault: false,
			CreatedBy: "test-user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mockRepo.On("GetBackendByName", ctx, "existing-backend").Return(existingBackend, nil)

		// Execute
		result, err := service.CreateBackend(ctx, createReq)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, models.ErrBackendExists, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_Health(t *testing.T) {
	mockRepo := &MockRepository{}
	service := statebusiness.NewService(mockRepo, nil)
	ctx := context.Background()

	t.Run("healthy", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("Health", ctx).Return(nil)
		mockRepo.On("GetStateFileCount", ctx).Return(5, nil)
		mockRepo.On("GetResourceCount", ctx).Return(10, nil)
		mockRepo.On("GetBackendCount", ctx).Return(2, nil)
		mockRepo.On("GetActiveLocksCount", ctx).Return(0, nil)

		// Execute
		result, err := service.Health(ctx)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "healthy", result.Status)
		assert.Equal(t, "state_management", result.Service)
		assert.Equal(t, 5, result.StateFileCount)
		assert.Equal(t, 10, result.ResourceCount)
		assert.Equal(t, 2, result.BackendCount)
		assert.Equal(t, 0, result.ActiveLocksCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("unhealthy", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("Health", ctx).Return(models.ErrDatabaseConnection)

		// Execute
		result, err := service.Health(ctx)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, models.ErrDatabaseConnection, err)
		mockRepo.AssertExpectations(t)
	})
}
