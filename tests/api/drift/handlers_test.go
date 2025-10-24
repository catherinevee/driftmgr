package drift

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	driftapi "github.com/catherinevee/driftmgr/internal/api/drift"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of the drift service
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateDriftResult(ctx context.Context, req *models.DriftResultRequest) (*models.DriftResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.DriftResult), args.Error(1)
}

func (m *MockService) GetDriftResult(ctx context.Context, id string) (*models.DriftResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DriftResult), args.Error(1)
}

func (m *MockService) ListDriftResults(ctx context.Context, query *models.DriftResultQuery) (*models.PaginatedDriftResults, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(*models.PaginatedDriftResults), args.Error(1)
}

func (m *MockService) GetDriftHistory(ctx context.Context, req *models.DriftHistoryRequest) (*models.DriftHistoryResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.DriftHistoryResponse), args.Error(1)
}

func (m *MockService) GetDriftSummary(ctx context.Context, provider string) (*models.DriftSummaryResponse, error) {
	args := m.Called(ctx, provider)
	return args.Get(0).(*models.DriftSummaryResponse), args.Error(1)
}

func (m *MockService) UpdateDriftResult(ctx context.Context, result *models.DriftResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockService) DeleteDriftResult(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) DeleteDriftResultsByProvider(ctx context.Context, provider string) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockService) DeleteDriftResultsByDateRange(ctx context.Context, startDate, endDate time.Time) error {
	args := m.Called(ctx, startDate, endDate)
	return args.Error(0)
}

func (m *MockService) GetDriftTrend(ctx context.Context, provider string, days int) ([]*models.DriftResult, error) {
	args := m.Called(ctx, provider, days)
	return args.Get(0).([]*models.DriftResult), args.Error(1)
}

func (m *MockService) GetTopDriftedResources(ctx context.Context, limit int) ([]string, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockService) GetDriftBySeverity(ctx context.Context, provider string) (map[string]int, error) {
	args := m.Called(ctx, provider)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *MockService) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Test helper functions

func createTestDriftResult() *models.DriftResult {
	return &models.DriftResult{
		ID:         "test-id-123",
		Timestamp:  time.Now(),
		Provider:   "aws",
		Status:     "completed",
		DriftCount: 2,
		Resources: []models.DriftedResource{
			{
				Address:    "aws_instance.test",
				Type:       "aws_instance",
				Provider:   "aws",
				Region:     "us-west-2",
				DriftType:  "modified",
				Severity:   "high",
				DetectedAt: time.Now(),
			},
		},
		Summary: models.DriftSummary{
			TotalResources:   1,
			DriftedResources: 1,
			HighDrift:        1,
		},
		Duration:  time.Minute * 5,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestHandler() (*driftapi.Handler, *MockService) {
	mockService := &MockService{}
	handler := driftapi.NewHandler(mockService)
	return handler, mockService
}

func createTestRouter(handler *driftapi.Handler) *mux.Router {
	router := mux.NewRouter()
	driftapi.RegisterRoutes(router, handler)
	return router
}

// Test cases

func TestGetDriftResult_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testResult := createTestDriftResult()
	mockService.On("GetDriftResult", mock.Anything, "test-id-123").Return(testResult, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/results/test-id-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, testResult.ID, response.ID)
	assert.Equal(t, testResult.Provider, response.Provider)

	mockService.AssertExpectations(t)
}

func TestGetDriftResult_NotFound(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	mockService.On("GetDriftResult", mock.Anything, "non-existent-id").Return(nil, models.ErrDriftResultNotFound)

	req := httptest.NewRequest("GET", "/api/v1/drift/results/non-existent-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestGetDriftResult_InvalidID(t *testing.T) {
	handler, _ := createTestHandler()
	router := createTestRouter(handler)

	req := httptest.NewRequest("GET", "/api/v1/drift/results/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListDriftResults_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testResults := &models.PaginatedDriftResults{
		Results: []models.DriftResult{*createTestDriftResult()},
		Total:   1,
		Page:    1,
		PerPage: 50,
		Pages:   1,
	}

	mockService.On("ListDriftResults", mock.Anything, mock.AnythingOfType("*models.DriftResultQuery")).Return(testResults, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/results", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.PaginatedDriftResults
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.Results, 1)

	mockService.AssertExpectations(t)
}

func TestListDriftResults_WithFilters(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testResults := &models.PaginatedDriftResults{
		Results: []models.DriftResult{*createTestDriftResult()},
		Total:   1,
		Page:    1,
		PerPage: 10,
		Pages:   1,
	}

	mockService.On("ListDriftResults", mock.Anything, mock.AnythingOfType("*models.DriftResultQuery")).Return(testResults, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/results?provider=aws&status=completed&limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PaginatedDriftResults
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 10, response.PerPage)

	mockService.AssertExpectations(t)
}

func TestGetDriftHistory_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testHistory := &models.DriftHistoryResponse{
		Results: []models.DriftResult{*createTestDriftResult()},
		Total:   1,
		Limit:   50,
		Offset:  0,
	}

	mockService.On("GetDriftHistory", mock.Anything, mock.AnythingOfType("*models.DriftHistoryRequest")).Return(testHistory, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/history", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftHistoryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.Results, 1)

	mockService.AssertExpectations(t)
}

func TestGetDriftSummary_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testSummary := &models.DriftSummaryResponse{
		Summary: models.DriftSummary{
			TotalResources:   10,
			DriftedResources: 3,
			CriticalDrift:    1,
			HighDrift:        2,
		},
		LastUpdated: time.Now(),
		Provider:    "aws",
	}

	mockService.On("GetDriftSummary", mock.Anything, "aws").Return(testSummary, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/summary?provider=aws", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftSummaryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "aws", response.Provider)
	assert.Equal(t, 10, response.Summary.TotalResources)

	mockService.AssertExpectations(t)
}

func TestDeleteDriftResult_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	mockService.On("DeleteDriftResult", mock.Anything, "test-id-123").Return(nil)

	req := httptest.NewRequest("DELETE", "/api/v1/drift/results/test-id-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftResultResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-id-123", response.ID)
	assert.Equal(t, "deleted", response.Status)

	mockService.AssertExpectations(t)
}

func TestDeleteDriftResult_NotFound(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	mockService.On("DeleteDriftResult", mock.Anything, "non-existent-id").Return(models.ErrDriftResultNotFound)

	req := httptest.NewRequest("DELETE", "/api/v1/drift/results/non-existent-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestGetDriftTrend_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testTrend := []*models.DriftResult{createTestDriftResult()}
	mockService.On("GetDriftTrend", mock.Anything, "aws", 30).Return(testTrend, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/trend?provider=aws&days=30", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response []models.DriftResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)

	mockService.AssertExpectations(t)
}

func TestGetTopDriftedResources_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testResources := []string{"aws_instance.test", "aws_s3_bucket.test"}
	mockService.On("GetTopDriftedResources", mock.Anything, 10).Return(testResources, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/top-resources?limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "resources")
	assert.Contains(t, response, "count")

	mockService.AssertExpectations(t)
}

func TestGetDriftBySeverity_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testSeverity := map[string]int{
		"critical": 1,
		"high":     2,
		"medium":   3,
		"low":      4,
	}
	mockService.On("GetDriftBySeverity", mock.Anything, "aws").Return(testSeverity, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/severity?provider=aws", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]int
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, response["critical"])
	assert.Equal(t, 2, response["high"])

	mockService.AssertExpectations(t)
}

func TestHealth_Success(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	mockService.On("Health", mock.Anything).Return(nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])

	mockService.AssertExpectations(t)
}

func TestHealth_Unhealthy(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	mockService.On("Health", mock.Anything).Return(assert.AnError)

	req := httptest.NewRequest("GET", "/api/v1/drift/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	mockService.AssertExpectations(t)
}

// Test error handling

func TestHandler_InternalServerError(t *testing.T) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	mockService.On("GetDriftResult", mock.Anything, "test-id-123").Return(nil, assert.AnError)

	req := httptest.NewRequest("GET", "/api/v1/drift/results/test-id-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// Test middleware

func TestCORSMiddleware(t *testing.T) {
	// Skip this test for now as it requires more complex setup
	t.Skip("CORS middleware test requires more complex setup")
}

func TestRecoveryMiddleware(t *testing.T) {
	// This test would require a handler that panics
	// For now, we'll just test that the middleware is properly registered
	handler, _ := createTestHandler()
	router := mux.NewRouter()
	driftapi.RegisterRoutes(router, handler)

	req := httptest.NewRequest("GET", "/api/v1/drift/results/test-id", nil)
	w := httptest.NewRecorder()

	// This should not panic
	router.ServeHTTP(w, req)
}

// Benchmark tests

func BenchmarkGetDriftResult(b *testing.B) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testResult := createTestDriftResult()
	mockService.On("GetDriftResult", mock.Anything, "test-id-123").Return(testResult, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/results/test-id-123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkListDriftResults(b *testing.B) {
	handler, mockService := createTestHandler()
	router := createTestRouter(handler)

	testResults := &models.PaginatedDriftResults{
		Results: make([]models.DriftResult, 100),
		Total:   100,
		Page:    1,
		PerPage: 50,
		Pages:   2,
	}

	for i := 0; i < 100; i++ {
		testResults.Results[i] = *createTestDriftResult()
	}

	mockService.On("ListDriftResults", mock.Anything, mock.AnythingOfType("*models.DriftResultQuery")).Return(testResults, nil)

	req := httptest.NewRequest("GET", "/api/v1/drift/results", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
