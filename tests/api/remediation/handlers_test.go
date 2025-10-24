package remediation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	remediationapi "github.com/catherinevee/driftmgr/internal/api/remediation"
	remediationbusiness "github.com/catherinevee/driftmgr/internal/business/remediation"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of the remediation repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateJob(ctx context.Context, job *models.RemediationJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRepository) GetJobByID(ctx context.Context, id string) (*models.RemediationJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RemediationJob), args.Error(1)
}

func (m *MockRepository) UpdateJob(ctx context.Context, job *models.RemediationJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRepository) DeleteJob(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListJobs(ctx context.Context, req *models.RemediationJobListRequest) (*models.RemediationJobListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RemediationJobListResponse), args.Error(1)
}

func (m *MockRepository) UpdateJobStatus(ctx context.Context, id string, status models.JobStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateJobProgress(ctx context.Context, id string, progress models.JobProgress) error {
	args := m.Called(ctx, id, progress)
	return args.Error(0)
}

func (m *MockRepository) AddJobLog(ctx context.Context, log *models.JobLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockRepository) GetJobLogs(ctx context.Context, jobID string) ([]models.JobLog, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.JobLog), args.Error(1)
}

func (m *MockRepository) GetJobsByStatus(ctx context.Context, status models.JobStatus) ([]models.RemediationJob, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RemediationJob), args.Error(1)
}

func (m *MockRepository) GetJobsByPriority(ctx context.Context, priority models.JobPriority) ([]models.RemediationJob, error) {
	args := m.Called(ctx, priority)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RemediationJob), args.Error(1)
}

func (m *MockRepository) GetJobsByUser(ctx context.Context, userID string) ([]models.RemediationJob, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RemediationJob), args.Error(1)
}

func (m *MockRepository) GetJobsByStrategy(ctx context.Context, strategyType models.StrategyType) ([]models.RemediationJob, error) {
	args := m.Called(ctx, strategyType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RemediationJob), args.Error(1)
}

func (m *MockRepository) GetJobsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.RemediationJob, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RemediationJob), args.Error(1)
}

func (m *MockRepository) CreateStrategy(ctx context.Context, strategy *models.RemediationStrategy) error {
	args := m.Called(ctx, strategy)
	return args.Error(0)
}

func (m *MockRepository) GetStrategyByID(ctx context.Context, id string) (*models.RemediationStrategy, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RemediationStrategy), args.Error(1)
}

func (m *MockRepository) GetStrategyByName(ctx context.Context, name string) (*models.RemediationStrategy, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RemediationStrategy), args.Error(1)
}

func (m *MockRepository) UpdateStrategy(ctx context.Context, strategy *models.RemediationStrategy) error {
	args := m.Called(ctx, strategy)
	return args.Error(0)
}

func (m *MockRepository) DeleteStrategy(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListStrategies(ctx context.Context) ([]models.RemediationStrategy, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RemediationStrategy), args.Error(1)
}

func (m *MockRepository) GetStrategiesByType(ctx context.Context, strategyType models.StrategyType) ([]models.RemediationStrategy, error) {
	args := m.Called(ctx, strategyType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RemediationStrategy), args.Error(1)
}

func (m *MockRepository) GetRemediationHistory(ctx context.Context, req *models.RemediationHistoryRequest) (*models.RemediationHistoryResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RemediationHistoryResponse), args.Error(1)
}

func (m *MockRepository) GetJobStatistics(ctx context.Context, startDate, endDate time.Time) (*models.JobStatistics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobStatistics), args.Error(1)
}

func (m *MockRepository) GetSuccessRate(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) GetAverageJobDuration(ctx context.Context, startDate, endDate time.Time) (time.Duration, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockRepository) DeleteOldJobs(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockRepository) CleanupCompletedJobs(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockRepository) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) GetQueueDepth(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetActiveJobsCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetFailedJobsCount(ctx context.Context) (int, error) {
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
func setupTestHandler() (*remediationapi.Handler, *MockRepository) {
	mockRepo := &MockRepository{}
	service := remediationbusiness.NewService(mockRepo, nil)
	handler := remediationapi.NewHandler(service)
	return handler, mockRepo
}

func createTestJob() *models.RemediationJob {
	return &models.RemediationJob{
		ID:            "test-job-123",
		DriftResultID: "drift-123",
		Strategy: models.RemediationStrategy{
			Type:        models.StrategyTypeTerraformApply,
			Name:        "Test Strategy",
			Description: "Test remediation strategy",
		},
		Status:    models.JobStatusPending,
		Priority:  models.JobPriorityMedium,
		CreatedBy: "test-user",
		Progress: models.JobProgress{
			TotalResources:     10,
			ProcessedResources: 0,
			Percentage:         0,
			CurrentStep:        "Initializing",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestJobRequest() *models.RemediationJobRequest {
	return &models.RemediationJobRequest{
		DriftResultID: "drift-123",
		Strategy: models.RemediationStrategy{
			Type:        models.StrategyTypeTerraformApply,
			Name:        "Test Strategy",
			Description: "Test remediation strategy",
		},
		Priority:         models.JobPriorityMedium,
		DryRun:           false,
		RequiresApproval: false,
	}
}

// Tests

func TestCreateJob_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("CreateJob", mock.Anything, mock.AnythingOfType("*models.RemediationJob")).Return(nil)
	mockRepo.On("AddJobLog", mock.Anything, mock.AnythingOfType("*models.JobLog")).Return(nil)

	// Create request
	req := createTestJobRequest()
	reqBody, _ := json.Marshal(req)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/remediation/jobs", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/jobs", handler.CreateJob).Methods("POST")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.RemediationJobResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, models.JobStatusPending, response.Status)

	mockRepo.AssertExpectations(t)
}

func TestCreateJob_InvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler()

	// Create invalid JSON request
	reqBody := []byte(`{"invalid": json}`)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/remediation/jobs", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/jobs", handler.CreateJob).Methods("POST")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestGetJob_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testJob := createTestJob()
	mockRepo.On("GetJobByID", mock.Anything, "test-job-123").Return(testJob, nil)
	mockRepo.On("GetJobLogs", mock.Anything, "test-job-123").Return([]models.JobLog{}, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/jobs/test-job-123", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/jobs/{id}", handler.GetJob).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.RemediationJob
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-job-123", response.ID)
	assert.Equal(t, models.JobStatusPending, response.Status)

	mockRepo.AssertExpectations(t)
}

func TestGetJob_NotFound(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("GetJobByID", mock.Anything, "nonexistent-job").Return(nil, models.ErrRemediationJobNotFound)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/jobs/nonexistent-job", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/jobs/{id}", handler.GetJob).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	mockRepo.AssertExpectations(t)
}

func TestListJobs_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testJobs := []models.RemediationJob{*createTestJob()}
	response := &models.RemediationJobListResponse{
		Jobs:   testJobs,
		Total:  1,
		Limit:  50,
		Offset: 0,
	}
	mockRepo.On("ListJobs", mock.Anything, mock.AnythingOfType("*models.RemediationJobListRequest")).Return(response, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/jobs", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/jobs", handler.ListJobs).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var listResponse models.RemediationJobListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, listResponse.Total)
	assert.Len(t, listResponse.Jobs, 1)

	mockRepo.AssertExpectations(t)
}

func TestCancelJob_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testJob := createTestJob()
	testJob.Status = models.JobStatusRunning // Job must be running to be cancelled

	mockRepo.On("GetJobByID", mock.Anything, "test-job-123").Return(testJob, nil)
	mockRepo.On("GetJobLogs", mock.Anything, "test-job-123").Return([]models.JobLog{}, nil)
	mockRepo.On("UpdateJobStatus", mock.Anything, "test-job-123", models.JobStatusCancelled).Return(nil)
	mockRepo.On("UpdateJob", mock.Anything, mock.AnythingOfType("*models.RemediationJob")).Return(nil)
	mockRepo.On("AddJobLog", mock.Anything, mock.AnythingOfType("*models.JobLog")).Return(nil)

	// Create cancel request
	cancelReq := models.JobCancelRequest{
		Reason: "User requested cancellation",
	}
	reqBody, _ := json.Marshal(cancelReq)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/remediation/jobs/test-job-123/cancel", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/jobs/{id}/cancel", handler.CancelJob).Methods("POST")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.JobCancelResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-job-123", response.JobID)
	assert.Equal(t, "User requested cancellation", response.Reason)

	mockRepo.AssertExpectations(t)
}

func TestGetJobProgress_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testJob := createTestJob()
	testJob.Progress = models.JobProgress{
		TotalResources:     10,
		ProcessedResources: 5,
		Percentage:         50,
		CurrentStep:        "Processing resources",
	}

	mockRepo.On("GetJobByID", mock.Anything, "test-job-123").Return(testJob, nil)
	mockRepo.On("GetJobLogs", mock.Anything, "test-job-123").Return([]models.JobLog{}, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/progress/test-job-123", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/progress/{id}", handler.GetJobProgress).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.JobProgress
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 10, response.TotalResources)
	assert.Equal(t, 5, response.ProcessedResources)
	assert.Equal(t, 50.0, response.Percentage)

	mockRepo.AssertExpectations(t)
}

func TestListStrategies_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testStrategies := []models.RemediationStrategy{
		{
			ID:          "strategy-1",
			Type:        models.StrategyTypeTerraformApply,
			Name:        "Terraform Apply",
			Description: "Apply Terraform configuration",
			IsCustom:    false,
		},
	}
	mockRepo.On("ListStrategies", mock.Anything).Return(testStrategies, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/strategies", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/strategies", handler.ListStrategies).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.RemediationStrategyListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.Strategies, 1)
	assert.Equal(t, "Terraform Apply", response.Strategies[0].Name)

	mockRepo.AssertExpectations(t)
}

func TestCreateStrategy_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("GetStrategyByName", mock.Anything, "Custom Strategy").Return(nil, models.ErrRemediationStrategyNotFound)
	mockRepo.On("CreateStrategy", mock.Anything, mock.AnythingOfType("*models.RemediationStrategy")).Return(nil)

	// Create strategy request
	strategyReq := models.RemediationStrategyRequest{
		Type:        models.StrategyTypeTerraformApply,
		Name:        "Custom Strategy",
		Description: "A custom remediation strategy",
		RetryCount:  3,
	}
	reqBody, _ := json.Marshal(strategyReq)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/remediation/strategies", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/strategies", handler.CreateStrategy).Methods("POST")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.RemediationStrategyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, "Custom Strategy", response.Name)
	assert.Equal(t, models.StrategyTypeTerraformApply, response.Type)

	mockRepo.AssertExpectations(t)
}

func TestGetRemediationHistory_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	testJobs := []models.RemediationJob{*createTestJob()}
	response := &models.RemediationHistoryResponse{
		Jobs:   testJobs,
		Total:  1,
		Limit:  50,
		Offset: 0,
	}
	mockRepo.On("GetRemediationHistory", mock.Anything, mock.AnythingOfType("*models.RemediationHistoryRequest")).Return(response, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/history", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/history", handler.GetRemediationHistory).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var historyResponse models.RemediationHistoryResponse
	err := json.Unmarshal(w.Body.Bytes(), &historyResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, historyResponse.Total)
	assert.Len(t, historyResponse.Jobs, 1)

	mockRepo.AssertExpectations(t)
}

func TestHealth_Success(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("Health", mock.Anything).Return(nil)
	mockRepo.On("GetQueueDepth", mock.Anything).Return(5, nil)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/health", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/health", handler.Health).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "remediation", response["service"])

	mockRepo.AssertExpectations(t)
}

func TestHealth_Unhealthy(t *testing.T) {
	handler, mockRepo := setupTestHandler()

	// Setup mock expectations
	mockRepo.On("Health", mock.Anything).Return(models.ErrDatabaseConnection)

	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/api/v1/remediation/health", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Create router and register route
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/remediation/health", handler.Health).Methods("GET")

	// Execute request
	router.ServeHTTP(w, httpReq)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	mockRepo.AssertExpectations(t)
}
