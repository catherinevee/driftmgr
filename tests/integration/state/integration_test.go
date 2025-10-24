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
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of the state repository
type MockRepository struct {
	stateFiles map[string]*models.StateFile
	resources  map[string]*models.StateResource
	backends   map[string]*models.Backend
	operations map[string]*models.StateOperation
	locks      map[string]*models.StateLock
	nextID     int
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		stateFiles: make(map[string]*models.StateFile),
		resources:  make(map[string]*models.StateResource),
		backends:   make(map[string]*models.Backend),
		operations: make(map[string]*models.StateOperation),
		locks:      make(map[string]*models.StateLock),
		nextID:     1,
	}
}

func (m *MockRepository) CreateStateFile(ctx context.Context, stateFile *models.StateFile) error {
	if _, exists := m.stateFiles[stateFile.ID]; exists {
		return models.ErrStateFileExists
	}
	m.stateFiles[stateFile.ID] = stateFile
	return nil
}

func (m *MockRepository) GetStateFileByID(ctx context.Context, id string) (*models.StateFile, error) {
	if stateFile, exists := m.stateFiles[id]; exists {
		return stateFile, nil
	}
	return nil, models.ErrStateFileNotFound
}

func (m *MockRepository) GetStateFileByPath(ctx context.Context, path string) (*models.StateFile, error) {
	for _, stateFile := range m.stateFiles {
		if stateFile.Path == path {
			return stateFile, nil
		}
	}
	return nil, models.ErrStateFileNotFound
}

func (m *MockRepository) UpdateStateFile(ctx context.Context, stateFile *models.StateFile) error {
	if _, exists := m.stateFiles[stateFile.ID]; !exists {
		return models.ErrStateFileNotFound
	}
	m.stateFiles[stateFile.ID] = stateFile
	return nil
}

func (m *MockRepository) DeleteStateFile(ctx context.Context, id string) error {
	if _, exists := m.stateFiles[id]; !exists {
		return models.ErrStateFileNotFound
	}
	delete(m.stateFiles, id)
	return nil
}

func (m *MockRepository) ListStateFiles(ctx context.Context, req *models.StateFileListRequest) (*models.StateFileListResponse, error) {
	stateFiles := make([]models.StateFile, 0, len(m.stateFiles))
	for _, stateFile := range m.stateFiles {
		stateFiles = append(stateFiles, *stateFile)
	}

	// Simple pagination
	start := req.Offset
	end := start + req.Limit
	if end > len(stateFiles) {
		end = len(stateFiles)
	}
	if start > len(stateFiles) {
		start = len(stateFiles)
	}

	var paginatedStateFiles []models.StateFile
	if start < len(stateFiles) {
		paginatedStateFiles = stateFiles[start:end]
	}

	return &models.StateFileListResponse{
		StateFiles: paginatedStateFiles,
		Total:      len(stateFiles),
		Limit:      req.Limit,
		Offset:     req.Offset,
	}, nil
}

func (m *MockRepository) GetStateFilesByBackend(ctx context.Context, backendID string) ([]models.StateFile, error) {
	var stateFiles []models.StateFile
	for _, stateFile := range m.stateFiles {
		if stateFile.BackendID == backendID {
			stateFiles = append(stateFiles, *stateFile)
		}
	}
	return stateFiles, nil
}

func (m *MockRepository) GetStateFilesByEnvironment(ctx context.Context, environmentID string) ([]models.StateFile, error) {
	var stateFiles []models.StateFile
	for _, stateFile := range m.stateFiles {
		// This would need to be implemented based on the actual relationship
		// For now, we'll return all state files
		stateFiles = append(stateFiles, *stateFile)
	}
	return stateFiles, nil
}

func (m *MockRepository) GetStateFilesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.StateFile, error) {
	var stateFiles []models.StateFile
	for _, stateFile := range m.stateFiles {
		if stateFile.CreatedAt.After(startDate) && stateFile.CreatedAt.Before(endDate) {
			stateFiles = append(stateFiles, *stateFile)
		}
	}
	return stateFiles, nil
}

func (m *MockRepository) CreateResource(ctx context.Context, resource *models.StateResource) error {
	if _, exists := m.resources[resource.ID]; exists {
		return models.ErrResourceExists
	}
	m.resources[resource.ID] = resource
	return nil
}

func (m *MockRepository) GetResourceByID(ctx context.Context, id string) (*models.StateResource, error) {
	if resource, exists := m.resources[id]; exists {
		return resource, nil
	}
	return nil, models.ErrResourceNotFound
}

func (m *MockRepository) GetResourceByAddress(ctx context.Context, stateFileID, address string) (*models.StateResource, error) {
	for _, resource := range m.resources {
		if resource.StateFileID == stateFileID && resource.Address == address {
			return resource, nil
		}
	}
	return nil, models.ErrResourceNotFound
}

func (m *MockRepository) UpdateResource(ctx context.Context, resource *models.StateResource) error {
	if _, exists := m.resources[resource.ID]; !exists {
		return models.ErrResourceNotFound
	}
	m.resources[resource.ID] = resource
	return nil
}

func (m *MockRepository) DeleteResource(ctx context.Context, id string) error {
	if _, exists := m.resources[id]; !exists {
		return models.ErrResourceNotFound
	}
	delete(m.resources, id)
	return nil
}

func (m *MockRepository) ListResources(ctx context.Context, req *models.ResourceListRequest) (*models.ResourceListResponse, error) {
	resources := make([]models.StateResource, 0, len(m.resources))
	for _, resource := range m.resources {
		resources = append(resources, *resource)
	}

	// Simple pagination
	start := req.Offset
	end := start + req.Limit
	if end > len(resources) {
		end = len(resources)
	}
	if start > len(resources) {
		start = len(resources)
	}

	var paginatedResources []models.StateResource
	if start < len(resources) {
		paginatedResources = resources[start:end]
	}

	return &models.ResourceListResponse{
		Resources: paginatedResources,
		Total:     len(resources),
		Limit:     req.Limit,
		Offset:    req.Offset,
	}, nil
}

func (m *MockRepository) GetResourcesByStateFile(ctx context.Context, stateFileID string) ([]models.StateResource, error) {
	var resources []models.StateResource
	for _, resource := range m.resources {
		if resource.StateFileID == stateFileID {
			resources = append(resources, *resource)
		}
	}
	return resources, nil
}

func (m *MockRepository) GetResourcesByType(ctx context.Context, resourceType string) ([]models.StateResource, error) {
	var resources []models.StateResource
	for _, resource := range m.resources {
		if resource.Type == resourceType {
			resources = append(resources, *resource)
		}
	}
	return resources, nil
}

func (m *MockRepository) GetResourcesByProvider(ctx context.Context, provider string) ([]models.StateResource, error) {
	var resources []models.StateResource
	for _, resource := range m.resources {
		if resource.Provider == provider {
			resources = append(resources, *resource)
		}
	}
	return resources, nil
}

func (m *MockRepository) GetResourcesByModule(ctx context.Context, module string) ([]models.StateResource, error) {
	var resources []models.StateResource
	for _, resource := range m.resources {
		// This would need to be implemented based on the actual module structure
		// For now, we'll return all resources
		resources = append(resources, *resource)
	}
	return resources, nil
}

func (m *MockRepository) CreateBackend(ctx context.Context, backend *models.Backend) error {
	if _, exists := m.backends[backend.ID]; exists {
		return models.ErrBackendExists
	}
	m.backends[backend.ID] = backend
	return nil
}

func (m *MockRepository) GetBackendByID(ctx context.Context, id string) (*models.Backend, error) {
	if backend, exists := m.backends[id]; exists {
		return backend, nil
	}
	return nil, models.ErrBackendNotFound
}

func (m *MockRepository) GetBackendByName(ctx context.Context, name string) (*models.Backend, error) {
	for _, backend := range m.backends {
		if backend.Name == name {
			return backend, nil
		}
	}
	return nil, models.ErrBackendNotFound
}

func (m *MockRepository) UpdateBackend(ctx context.Context, backend *models.Backend) error {
	if _, exists := m.backends[backend.ID]; !exists {
		return models.ErrBackendNotFound
	}
	m.backends[backend.ID] = backend
	return nil
}

func (m *MockRepository) DeleteBackend(ctx context.Context, id string) error {
	if _, exists := m.backends[id]; !exists {
		return models.ErrBackendNotFound
	}
	delete(m.backends, id)
	return nil
}

func (m *MockRepository) ListBackends(ctx context.Context, req *models.BackendListRequest) (*models.BackendListResponse, error) {
	backends := make([]models.Backend, 0, len(m.backends))
	for _, backend := range m.backends {
		backends = append(backends, *backend)
	}

	// Simple pagination
	start := req.Offset
	end := start + req.Limit
	if end > len(backends) {
		end = len(backends)
	}
	if start > len(backends) {
		start = len(backends)
	}

	var paginatedBackends []models.Backend
	if start < len(backends) {
		paginatedBackends = backends[start:end]
	}

	return &models.BackendListResponse{
		Backends: paginatedBackends,
		Total:    len(backends),
		Limit:    req.Limit,
		Offset:   req.Offset,
	}, nil
}

func (m *MockRepository) GetBackendsByEnvironment(ctx context.Context, environmentID string) ([]models.Backend, error) {
	var backends []models.Backend
	for _, backend := range m.backends {
		if backend.EnvironmentID == environmentID {
			backends = append(backends, *backend)
		}
	}
	return backends, nil
}

func (m *MockRepository) GetBackendsByType(ctx context.Context, backendType models.BackendType) ([]models.Backend, error) {
	var backends []models.Backend
	for _, backend := range m.backends {
		if backend.Type == backendType {
			backends = append(backends, *backend)
		}
	}
	return backends, nil
}

func (m *MockRepository) GetDefaultBackend(ctx context.Context, environmentID string) (*models.Backend, error) {
	for _, backend := range m.backends {
		if backend.EnvironmentID == environmentID && backend.IsDefault {
			return backend, nil
		}
	}
	return nil, models.ErrBackendNotFound
}

func (m *MockRepository) CreateStateOperation(ctx context.Context, operation *models.StateOperation) error {
	if _, exists := m.operations[operation.ID]; exists {
		return models.ErrStateOperationExists
	}
	m.operations[operation.ID] = operation
	return nil
}

func (m *MockRepository) GetStateOperationByID(ctx context.Context, id string) (*models.StateOperation, error) {
	if operation, exists := m.operations[id]; exists {
		return operation, nil
	}
	return nil, models.ErrStateOperationNotFound
}

func (m *MockRepository) UpdateStateOperation(ctx context.Context, operation *models.StateOperation) error {
	if _, exists := m.operations[operation.ID]; !exists {
		return models.ErrStateOperationNotFound
	}
	m.operations[operation.ID] = operation
	return nil
}

func (m *MockRepository) DeleteStateOperation(ctx context.Context, id string) error {
	if _, exists := m.operations[id]; !exists {
		return models.ErrStateOperationNotFound
	}
	delete(m.operations, id)
	return nil
}

func (m *MockRepository) ListStateOperations(ctx context.Context, stateFileID string) ([]models.StateOperation, error) {
	var operations []models.StateOperation
	for _, operation := range m.operations {
		if operation.StateFileID == stateFileID {
			operations = append(operations, *operation)
		}
	}
	return operations, nil
}

func (m *MockRepository) CreateStateLock(ctx context.Context, lock *models.StateLock) error {
	if _, exists := m.locks[lock.ID]; exists {
		return models.ErrStateLockExists
	}
	m.locks[lock.ID] = lock
	return nil
}

func (m *MockRepository) GetStateLockByID(ctx context.Context, id string) (*models.StateLock, error) {
	if lock, exists := m.locks[id]; exists {
		return lock, nil
	}
	return nil, models.ErrStateLockNotFound
}

func (m *MockRepository) GetStateLockByStateFile(ctx context.Context, stateFileID string) (*models.StateLock, error) {
	for _, lock := range m.locks {
		if lock.StateFileID == stateFileID {
			return lock, nil
		}
	}
	return nil, models.ErrStateLockNotFound
}

func (m *MockRepository) DeleteStateLock(ctx context.Context, id string) error {
	if _, exists := m.locks[id]; !exists {
		return models.ErrStateLockNotFound
	}
	delete(m.locks, id)
	return nil
}

func (m *MockRepository) DeleteStateLockByStateFile(ctx context.Context, stateFileID string) error {
	for id, lock := range m.locks {
		if lock.StateFileID == stateFileID {
			delete(m.locks, id)
			return nil
		}
	}
	return models.ErrStateLockNotFound
}

func (m *MockRepository) GetStateFileHistory(ctx context.Context, stateFileID string) ([]models.StateFile, error) {
	var history []models.StateFile
	for _, stateFile := range m.stateFiles {
		if stateFile.ID == stateFileID {
			history = append(history, *stateFile)
		}
	}
	return history, nil
}

func (m *MockRepository) GetResourceHistory(ctx context.Context, resourceID string) ([]models.StateResource, error) {
	var history []models.StateResource
	for _, resource := range m.resources {
		if resource.ID == resourceID {
			history = append(history, *resource)
		}
	}
	return history, nil
}

func (m *MockRepository) GetStateFileStatistics(ctx context.Context, startDate, endDate time.Time) (*models.StateFileStatistics, error) {
	return &models.StateFileStatistics{
		TotalFiles:     len(m.stateFiles),
		TotalSize:      1024 * len(m.stateFiles),
		AverageSize:    1024,
		FilesByBackend: make(map[string]int),
		FilesByDate:    make(map[string]int),
	}, nil
}

func (m *MockRepository) GetResourceStatistics(ctx context.Context, startDate, endDate time.Time) (*models.ResourceStatistics, error) {
	return &models.ResourceStatistics{
		TotalResources:      len(m.resources),
		ResourcesByType:     make(map[string]int),
		ResourcesByProvider: make(map[string]int),
		ResourcesByModule:   make(map[string]int),
	}, nil
}

func (m *MockRepository) DeleteOldStateFiles(ctx context.Context, olderThan time.Time) error {
	for id, stateFile := range m.stateFiles {
		if stateFile.CreatedAt.Before(olderThan) {
			delete(m.stateFiles, id)
		}
	}
	return nil
}

func (m *MockRepository) DeleteOldOperations(ctx context.Context, olderThan time.Time) error {
	for id, operation := range m.operations {
		if operation.CreatedAt.Before(olderThan) {
			delete(m.operations, id)
		}
	}
	return nil
}

func (m *MockRepository) DeleteOldLocks(ctx context.Context, olderThan time.Time) error {
	for id, lock := range m.locks {
		if lock.Created.Before(olderThan) {
			delete(m.locks, id)
		}
	}
	return nil
}

func (m *MockRepository) CleanupOrphanedResources(ctx context.Context) error {
	// Simple cleanup - remove resources that don't have a corresponding state file
	for id, resource := range m.resources {
		if _, exists := m.stateFiles[resource.StateFileID]; !exists {
			delete(m.resources, id)
		}
	}
	return nil
}

func (m *MockRepository) Health(ctx context.Context) error {
	return nil
}

func (m *MockRepository) GetStateFileCount(ctx context.Context) (int, error) {
	return len(m.stateFiles), nil
}

func (m *MockRepository) GetResourceCount(ctx context.Context) (int, error) {
	return len(m.resources), nil
}

func (m *MockRepository) GetBackendCount(ctx context.Context) (int, error) {
	return len(m.backends), nil
}

func (m *MockRepository) GetActiveLocksCount(ctx context.Context) (int, error) {
	return len(m.locks), nil
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (m *MockRepository) Close() error {
	return nil
}

// Test setup
func setupIntegrationTest() (*stateapi.Handler, *MockRepository) {
	mockRepo := NewMockRepository()
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

// Integration Tests

func TestStateManagement_EndToEnd(t *testing.T) {
	handler, mockRepo := setupIntegrationTest()
	ctx := context.Background()

	// Create a test backend
	backend := createTestBackend()
	err := mockRepo.CreateBackend(ctx, backend)
	require.NoError(t, err)

	// Create a test state file
	stateFile := createTestStateFile()
	err = mockRepo.CreateStateFile(ctx, stateFile)
	require.NoError(t, err)

	// Test listing state files
	req := &models.StateFileListRequest{
		Limit:  50,
		Offset: 0,
	}
	response, err := mockRepo.ListStateFiles(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.StateFiles, 1)

	// Test getting state file by ID
	retrievedStateFile, err := mockRepo.GetStateFileByID(ctx, stateFile.ID)
	require.NoError(t, err)
	assert.Equal(t, stateFile.ID, retrievedStateFile.ID)
	assert.Equal(t, stateFile.Name, retrievedStateFile.Name)

	// Test importing a resource
	importReq := &models.ImportResourceRequest{
		ResourceAddress: "aws_instance.test",
		ResourceID:      "i-1234567890abcdef0",
		Configuration:   map[string]interface{}{"instance_type": "t2.micro"},
	}

	// Create a resource
	resource := &models.StateResource{
		ID:            "550e8400-e29b-41d4-a716-446655440002",
		StateFileID:   stateFile.ID,
		Address:       importReq.ResourceAddress,
		Type:          "aws_instance",
		Provider:      "aws",
		Instance:      importReq.ResourceID,
		Attributes:    importReq.Configuration,
		Mode:          "managed",
		SchemaVersion: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = mockRepo.CreateResource(ctx, resource)
	require.NoError(t, err)

	// Test listing resources
	resourceReq := &models.ResourceListRequest{
		Limit:  50,
		Offset: 0,
	}
	resourceResponse, err := mockRepo.ListResources(ctx, resourceReq)
	require.NoError(t, err)
	assert.Equal(t, 1, resourceResponse.Total)
	assert.Len(t, resourceResponse.Resources, 1)

	// Test getting resource by ID
	retrievedResource, err := mockRepo.GetResourceByID(ctx, resource.ID)
	require.NoError(t, err)
	assert.Equal(t, resource.ID, retrievedResource.ID)
	assert.Equal(t, resource.Address, retrievedResource.Address)

	// Test health check
	health, err := mockRepo.Health(ctx)
	require.NoError(t, health)

	stateFileCount, err := mockRepo.GetStateFileCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, stateFileCount)

	resourceCount, err := mockRepo.GetResourceCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, resourceCount)

	backendCount, err := mockRepo.GetBackendCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, backendCount)
}

func TestStateManagement_API_EndToEnd(t *testing.T) {
	handler, mockRepo := setupIntegrationTest()

	// Create a test backend
	backend := createTestBackend()
	err := mockRepo.CreateBackend(context.Background(), backend)
	require.NoError(t, err)

	// Create a test state file
	stateFile := createTestStateFile()
	err = mockRepo.CreateStateFile(context.Background(), stateFile)
	require.NoError(t, err)

	// Test API endpoints
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/files", handler.ListStateFiles).Methods("GET")
	router.HandleFunc("/api/v1/state/files/{id}", handler.GetStateFile).Methods("GET")
	router.HandleFunc("/api/v1/state/resources", handler.ListResources).Methods("GET")
	router.HandleFunc("/api/v1/state/backends", handler.ListBackends).Methods("GET")
	router.HandleFunc("/api/v1/state/health", handler.Health).Methods("GET")

	// Test listing state files
	httpReq := httptest.NewRequest("GET", "/api/v1/state/files", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse models.StateFileListResponse
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, listResponse.Total)

	// Test getting state file by ID
	httpReq = httptest.NewRequest("GET", "/api/v1/state/files/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)

	var stateFileResponse models.StateFile
	err = json.Unmarshal(w.Body.Bytes(), &stateFileResponse)
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", stateFileResponse.ID)

	// Test listing resources
	httpReq = httptest.NewRequest("GET", "/api/v1/state/resources", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)

	var resourceListResponse models.ResourceListResponse
	err = json.Unmarshal(w.Body.Bytes(), &resourceListResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, resourceListResponse.Total) // No resources created yet

	// Test listing backends
	httpReq = httptest.NewRequest("GET", "/api/v1/state/backends", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)

	var backendListResponse models.BackendListResponse
	err = json.Unmarshal(w.Body.Bytes(), &backendListResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, backendListResponse.Total)

	// Test health check
	httpReq = httptest.NewRequest("GET", "/api/v1/state/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)

	var healthResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &healthResponse)
	require.NoError(t, err)
	assert.Equal(t, "healthy", healthResponse["status"])
	assert.Equal(t, "state_management", healthResponse["service"])
}

func TestStateManagement_ResourceImport_EndToEnd(t *testing.T) {
	handler, mockRepo := setupIntegrationTest()

	// Create a test backend
	backend := createTestBackend()
	err := mockRepo.CreateBackend(context.Background(), backend)
	require.NoError(t, err)

	// Create a test state file
	stateFile := createTestStateFile()
	err = mockRepo.CreateStateFile(context.Background(), stateFile)
	require.NoError(t, err)

	// Test resource import API
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/files/{id}/import", handler.ImportResource).Methods("POST")

	// Create import request
	importReq := models.ImportResourceRequest{
		ResourceAddress: "aws_instance.test",
		ResourceID:      "i-1234567890abcdef0",
		Configuration:   map[string]interface{}{"instance_type": "t2.micro"},
	}
	reqBody, _ := json.Marshal(importReq)

	// Test import resource
	httpReq := httptest.NewRequest("POST", "/api/v1/state/files/550e8400-e29b-41d4-a716-446655440000/import", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusCreated, w.Code)

	var importResponse models.ImportResourceResponse
	err = json.Unmarshal(w.Body.Bytes(), &importResponse)
	require.NoError(t, err)
	assert.Equal(t, "aws_instance.test", importResponse.ResourceAddress)
	assert.Equal(t, "imported", importResponse.Status)

	// Verify resource was created
	resourceCount, err := mockRepo.GetResourceCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, resourceCount)
}

func TestStateManagement_BackendCreation_EndToEnd(t *testing.T) {
	handler, mockRepo := setupIntegrationTest()

	// Test backend creation API
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/state/backends", handler.CreateBackend).Methods("POST")

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

	// Test create backend
	httpReq := httptest.NewRequest("POST", "/api/v1/state/backends", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusCreated, w.Code)

	var backendResponse models.Backend
	err := json.Unmarshal(w.Body.Bytes(), &backendResponse)
	require.NoError(t, err)
	assert.Equal(t, "test-backend", backendResponse.Name)
	assert.Equal(t, models.BackendTypeS3, backendResponse.Type)

	// Verify backend was created
	backendCount, err := mockRepo.GetBackendCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, backendCount)
}
