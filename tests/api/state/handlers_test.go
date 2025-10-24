package state

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	stateapi "github.com/catherinevee/driftmgr/internal/api/state"
	statebusiness "github.com/catherinevee/driftmgr/internal/business/state"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

// Test setup
func setupTestHandler() (*stateapi.Handler, *MockRepository) {
	mockRepo := &MockRepository{}
	service := statebusiness.NewService(mockRepo, nil)
	handler := stateapi.NewHandler(service)
	return handler, mockRepo
}

func createTestStateFile() *models.StateFile {
	return &models.StateFile{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		BackendID:    "550e8400-e29b-41d4-a716-446655440001",
		Name:         "test-state",
		Path:         "/path/to/state.tfstate",
		Version:      1,
		Serial:       1,
		Lineage:      "test-lineage",
		Resources:    []models.StateResource{},
		Size:         1024,
		Checksum:     "test-checksum",
		LastModified: time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestResource() *models.StateResource {
	return &models.StateResource{
		ID:            "550e8400-e29b-41d4-a716-446655440002",
		StateFileID:   "550e8400-e29b-41d4-a716-446655440000",
		Address:       "aws_instance.test",
		Type:          "aws_instance",
		Provider:      "aws",
		Instance:      "i-1234567890abcdef0",
		Attributes:    map[string]interface{}{"instance_type": "t2.micro"},
		Mode:          "managed",
		SchemaVersion: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func createTestBackend() *models.Backend {
	return &models.Backend{
		ID:            "550e8400-e29b-41d4-a716-446655440001",
		EnvironmentID: "550e8400-e29b-41d4-a716-446655440003",
		Type:          models.BackendTypeS3,
		Name:          "test-backend",
		Description:   "Test backend",
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
}

// Tests

func TestListStateFiles_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testStateFiles := []models.StateFile{*createTestStateFile()}
	response := &models.StateFileListResponse{
		StateFiles: testStateFiles,
		Total:      1,
		Limit:      50,
		Offset:     0,
	}
	mockRepo.On("ListStateFiles", mock.Anything, mock.AnythingOfType("*models.StateFileListRequest")).Return(response, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/files", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/files", handler.ListStateFiles).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var listResponse models.StateFileListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, listResponse.Total)
	assert.Len(t, listResponse.StateFiles, 1)

	mockRepo.AssertExpectations(t)
}

func TestGetStateFile_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testStateFile := createTestStateFile()
	mockRepo.On("GetStateFileByID", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return(testStateFile, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/files/550e8400-e29b-41d4-a716-446655440000", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/files/{id}", handler.GetStateFile).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.StateFile
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", response.ID)
	assert.Equal(t, "test-state", response.Name)

	mockRepo.AssertExpectations(t)
}

func TestGetStateFile_NotFound(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("GetStateFileByID", mock.Anything, "nonexistent-id").Return(nil, models.ErrStateFileNotFound)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/files/nonexistent-id", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/files/{id}", handler.GetStateFile).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	mockRepo.AssertExpectations(t)
}

func TestImportResource_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testStateFile := createTestStateFile()
	mockRepo.On("GetStateFileByID", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return(testStateFile, nil)
	mockRepo.On("GetStateLockByStateFile", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return(nil, models.ErrStateLockNotFound)
	mockRepo.On("GetResourceByAddress", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", "aws_instance.test").Return(nil, models.ErrResourceNotFound)
	mockRepo.On("CreateResource", mock.Anything, mock.AnythingOfType("*models.StateResource")).Return(nil)
	mockRepo.On("UpdateStateFile", mock.Anything, mock.AnythingOfType("*models.StateFile")).Return(nil)
	mockRepo.On("CreateStateOperation", mock.Anything, mock.AnythingOfType("*models.StateOperation")).Return(nil)

	// Create import request
	importReq := models.ImportResourceRequest{
		ResourceAddress: "aws_instance.test",
		ResourceID:      "i-1234567890abcdef0",
		Configuration:   map[string]interface{}{"instance_type": "t2.micro"},
	}
	reqBody, _ := json.Marshal(importReq)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/state/files/550e8400-e29b-41d4-a716-446655440000/import", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/files/{id}/import", handler.ImportResource).Methods("POST")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.ImportResourceResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "aws_instance.test", response.ResourceAddress)
	assert.Equal(t, "imported", response.Status)

	mockRepo.AssertExpectations(t)
}

func TestListResources_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testResources := []models.StateResource{*createTestResource()}
	response := &models.ResourceListResponse{
		Resources: testResources,
		Total:     1,
		Limit:     50,
		Offset:    0,
	}
	mockRepo.On("ListResources", mock.Anything, mock.AnythingOfType("*models.ResourceListRequest")).Return(response, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/resources", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/resources", handler.ListResources).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var listResponse models.ResourceListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, listResponse.Total)
	assert.Len(t, listResponse.Resources, 1)

	mockRepo.AssertExpectations(t)
}

func TestGetResource_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testResource := createTestResource()
	mockRepo.On("GetResourceByID", mock.Anything, "550e8400-e29b-41d4-a716-446655440002").Return(testResource, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/resources/550e8400-e29b-41d4-a716-446655440002", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/resources/{id}", handler.GetResource).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.StateResource
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440002", response.ID)
	assert.Equal(t, "aws_instance.test", response.Address)

	mockRepo.AssertExpectations(t)
}

func TestListBackends_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testBackends := []models.Backend{*createTestBackend()}
	response := &models.BackendListResponse{
		Backends: testBackends,
		Total:    1,
		Limit:    50,
		Offset:   0,
	}
	mockRepo.On("ListBackends", mock.Anything, mock.AnythingOfType("*models.BackendListRequest")).Return(response, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/backends", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/backends", handler.ListBackends).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var listResponse models.BackendListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, listResponse.Total)
	assert.Len(t, listResponse.Backends, 1)

	mockRepo.AssertExpectations(t)
}

func TestCreateBackend_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("GetBackendByName", mock.Anything, "test-backend").Return(nil, models.ErrBackendNotFound)
	mockRepo.On("CreateBackend", mock.Anything, mock.AnythingOfType("*models.Backend")).Return(nil)

	// Create backend request
	backendReq := models.BackendCreateRequest{
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
	reqBody, _ := json.Marshal(backendReq)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/state/backends", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/backends", handler.CreateBackend).Methods("POST")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.Backend
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-backend", response.Name)
	assert.Equal(t, models.BackendTypeS3, response.Type)

	mockRepo.AssertExpectations(t)
}

func TestHealth_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("Health", mock.Anything).Return(nil)
	mockRepo.On("GetStateFileCount", mock.Anything).Return(5, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/health", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/health", handler.Health).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "state_management", response["service"])

	mockRepo.AssertExpectations(t)
}

func TestHealth_Unhealthy(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("Health", mock.Anything).Return(models.ErrDatabaseConnection)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/state/health", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/health", handler.Health).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	mockRepo.AssertExpectations(t)
}
