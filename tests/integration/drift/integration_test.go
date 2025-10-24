package drift

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	driftapi "github.com/catherinevee/driftmgr/internal/api/drift"
	driftbusiness "github.com/catherinevee/driftmgr/internal/business/drift"
	"github.com/catherinevee/driftmgr/internal/models"
	driftstorage "github.com/catherinevee/driftmgr/internal/storage/drift"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration test setup
var (
	testDB      *driftstorage.PostgresRepository
	testService *driftbusiness.DriftService
	testHandler *driftapi.Handler
	testRouter  *mux.Router
)

func TestMain(m *testing.M) {
	// Setup test database
	setupTestDatabase()

	// Setup test services
	setupTestServices()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupTestDatabase()

	os.Exit(code)
}

func setupTestDatabase() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/driftmgr_test?sslmode=disable"
	}

	config := &driftstorage.RepositoryConfig{
		DatabaseURL: dbURL,
		MaxConns:    10,
		MinConns:    2,
		MaxLifetime: time.Hour,
		MaxIdleTime: time.Minute * 30,
	}

	var err error
	testDB, err = driftstorage.NewPostgresRepository(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to setup test database: %v", err))
	}

	// Test database connection
	ctx := context.Background()
	if err := testDB.Health(ctx); err != nil {
		panic(fmt.Sprintf("Database health check failed: %v", err))
	}
}

func setupTestServices() {
	// Create service config
	serviceConfig := &driftbusiness.ServiceConfig{
		MaxDriftResults:   1000,
		MaxHistoryDays:    30,
		DefaultLimit:      50,
		MaxLimit:          1000,
		RetentionDays:     7,
		EnableAutoCleanup: false, // Disable for tests
		CleanupInterval:   time.Hour,
	}

	// Create service
	testService = driftbusiness.NewDriftService(testDB, serviceConfig)

	// Create handler
	testHandler = driftapi.NewHandler(testService)

	// Create router
	testRouter = mux.NewRouter()
	driftapi.RegisterRoutes(testRouter, testHandler)
}

func cleanupTestDatabase() {
	if testDB != nil {
		// Clean up test data
		ctx := context.Background()
		testDB.DeleteByDateRange(ctx, time.Time{}, time.Now().Add(time.Hour))
		testDB.Close()
	}
}

// Helper functions

func createTestDriftResult() *models.DriftResult {
	return &models.DriftResult{
		ID:         fmt.Sprintf("test-%d", time.Now().UnixNano()),
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
				Changes: []models.ResourceChange{
					{
						Field:      "instance_type",
						OldValue:   "t2.micro",
						NewValue:   "t2.small",
						ChangeType: "modified",
					},
				},
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

func makeRequest(method, url string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, url, bytes.NewReader(reqBody))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// Integration tests

func TestDriftResult_CRUD(t *testing.T) {
	ctx := context.Background()

	// Create a test drift result
	testResult := createTestDriftResult()

	// Test Create
	err := testDB.Create(ctx, testResult)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := testDB.GetByID(ctx, testResult.ID)
	require.NoError(t, err)
	assert.Equal(t, testResult.ID, retrieved.ID)
	assert.Equal(t, testResult.Provider, retrieved.Provider)
	assert.Equal(t, testResult.Status, retrieved.Status)

	// Test Update
	testResult.Status = "failed"
	testResult.Error = stringPtr("Test error")
	testResult.UpdatedAt = time.Now()

	err = testDB.Update(ctx, testResult)
	require.NoError(t, err)

	// Verify update
	updated, err := testDB.GetByID(ctx, testResult.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", updated.Status)
	assert.Equal(t, "Test error", *updated.Error)

	// Test Delete
	err = testDB.Delete(ctx, testResult.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = testDB.GetByID(ctx, testResult.ID)
	assert.Equal(t, models.ErrDriftResultNotFound, err)
}

func TestDriftResult_ListWithFilters(t *testing.T) {
	ctx := context.Background()

	// Create test data
	testResults := []*models.DriftResult{
		createTestDriftResult(),
		createTestDriftResult(),
		createTestDriftResult(),
	}

	// Set different providers
	testResults[0].Provider = "aws"
	testResults[1].Provider = "azure"
	testResults[2].Provider = "aws"

	// Set different statuses
	testResults[0].Status = "completed"
	testResults[1].Status = "failed"
	testResults[2].Status = "completed"

	// Create all test results
	for _, result := range testResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}

	// Test filter by provider
	query := &models.DriftResultQuery{
		Filter: models.DriftResultFilter{
			Provider: "aws",
		},
		Sort: models.DriftResultSort{
			Field: "timestamp",
			Order: "desc",
		},
		Limit:  10,
		Offset: 0,
	}

	results, err := testDB.List(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, 2, results.Total) // Should find 2 AWS results

	// Test filter by status
	query.Filter.Provider = ""
	query.Filter.Status = "completed"

	results, err = testDB.List(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, 2, results.Total) // Should find 2 completed results

	// Test filter by date range
	query.Filter.Status = ""
	query.Filter.StartDate = time.Now().Add(-time.Hour)
	query.Filter.EndDate = time.Now().Add(time.Hour)

	results, err = testDB.List(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, 3, results.Total) // Should find all 3 results

	// Cleanup
	for _, result := range testResults {
		testDB.Delete(ctx, result.ID)
	}
}

func TestDriftResult_History(t *testing.T) {
	ctx := context.Background()

	// Create test data
	testResults := []*models.DriftResult{
		createTestDriftResult(),
		createTestDriftResult(),
		createTestDriftResult(),
	}

	// Set different timestamps
	testResults[0].Timestamp = time.Now().Add(-time.Hour * 2)
	testResults[1].Timestamp = time.Now().Add(-time.Hour)
	testResults[2].Timestamp = time.Now()

	// Create all test results
	for _, result := range testResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}

	// Test history request
	req := &models.DriftHistoryRequest{
		Limit:  10,
		Offset: 0,
	}

	history, err := testDB.GetHistory(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 3, history.Total)
	assert.Len(t, history.Results, 3)

	// Test history with date range
	req.StartDate = time.Now().Add(-time.Hour * 3)
	req.EndDate = time.Now().Add(-time.Minute * 30)

	history, err = testDB.GetHistory(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 2, history.Total) // Should find 2 results in the range

	// Cleanup
	for _, result := range testResults {
		testDB.Delete(ctx, result.ID)
	}
}

func TestDriftResult_Summary(t *testing.T) {
	ctx := context.Background()

	// Create test data with different providers
	awsResults := []*models.DriftResult{
		createTestDriftResult(),
		createTestDriftResult(),
	}
	awsResults[0].Provider = "aws"
	awsResults[1].Provider = "aws"
	awsResults[0].Summary = models.DriftSummary{TotalResources: 5, DriftedResources: 2, CriticalDrift: 1}
	awsResults[1].Summary = models.DriftSummary{TotalResources: 3, DriftedResources: 1, HighDrift: 1}

	azureResult := createTestDriftResult()
	azureResult.Provider = "azure"
	azureResult.Summary = models.DriftSummary{TotalResources: 2, DriftedResources: 1, MediumDrift: 1}

	// Create all test results
	for _, result := range awsResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}
	err := testDB.Create(ctx, azureResult)
	require.NoError(t, err)

	// Test summary for AWS
	summary, err := testDB.GetSummary(ctx, "aws")
	require.NoError(t, err)
	assert.Equal(t, "aws", summary.Provider)
	assert.Equal(t, 8, summary.Summary.TotalResources)   // 5 + 3
	assert.Equal(t, 3, summary.Summary.DriftedResources) // 2 + 1
	assert.Equal(t, 1, summary.Summary.CriticalDrift)
	assert.Equal(t, 1, summary.Summary.HighDrift)

	// Test summary for all providers
	summary, err = testDB.GetSummary(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, 10, summary.Summary.TotalResources)  // 8 + 2
	assert.Equal(t, 4, summary.Summary.DriftedResources) // 3 + 1

	// Cleanup
	for _, result := range awsResults {
		testDB.Delete(ctx, result.ID)
	}
	testDB.Delete(ctx, azureResult.ID)
}

func TestDriftResult_Trend(t *testing.T) {
	ctx := context.Background()

	// Create test data with different timestamps
	testResults := []*models.DriftResult{
		createTestDriftResult(),
		createTestDriftResult(),
		createTestDriftResult(),
	}

	// Set timestamps for trend analysis
	testResults[0].Timestamp = time.Now().Add(-time.Hour * 24 * 2) // 2 days ago
	testResults[1].Timestamp = time.Now().Add(-time.Hour * 24)     // 1 day ago
	testResults[2].Timestamp = time.Now()                          // now

	// Create all test results
	for _, result := range testResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}

	// Test trend for last 3 days
	trend, err := testDB.GetDriftTrend(ctx, "aws", 3)
	require.NoError(t, err)
	assert.Len(t, trend, 3)

	// Test trend for last 1 day
	trend, err = testDB.GetDriftTrend(ctx, "aws", 1)
	require.NoError(t, err)
	assert.Len(t, trend, 1)

	// Cleanup
	for _, result := range testResults {
		testDB.Delete(ctx, result.ID)
	}
}

func TestDriftResult_BySeverity(t *testing.T) {
	ctx := context.Background()

	// Create test data with different severities
	testResults := []*models.DriftResult{
		createTestDriftResult(),
		createTestDriftResult(),
		createTestDriftResult(),
	}

	// Set different severities
	testResults[0].Summary = models.DriftSummary{CriticalDrift: 1, HighDrift: 2}
	testResults[1].Summary = models.DriftSummary{MediumDrift: 3, LowDrift: 1}
	testResults[2].Summary = models.DriftSummary{HighDrift: 1, LowDrift: 2}

	// Create all test results
	for _, result := range testResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}

	// Test drift by severity
	severity, err := testDB.GetDriftBySeverity(ctx, "aws")
	require.NoError(t, err)
	assert.Equal(t, 1, severity["critical"])
	assert.Equal(t, 3, severity["high"]) // 2 + 1
	assert.Equal(t, 3, severity["medium"])
	assert.Equal(t, 3, severity["low"]) // 1 + 2

	// Cleanup
	for _, result := range testResults {
		testDB.Delete(ctx, result.ID)
	}
}

func TestDriftResult_TopResources(t *testing.T) {
	ctx := context.Background()

	// Create test data with repeated resources
	testResults := []*models.DriftResult{
		createTestDriftResult(),
		createTestDriftResult(),
		createTestDriftResult(),
	}

	// Set same resource address for first two results
	testResults[0].Resources[0].Address = "aws_instance.frequent"
	testResults[1].Resources[0].Address = "aws_instance.frequent"
	testResults[2].Resources[0].Address = "aws_instance.rare"

	// Create all test results
	for _, result := range testResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}

	// Test top drifted resources
	resources, err := testDB.GetTopDriftedResources(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, resources, 2)
	assert.Equal(t, "aws_instance.frequent", resources[0]) // Should be first (most frequent)

	// Cleanup
	for _, result := range testResults {
		testDB.Delete(ctx, result.ID)
	}
}

// API Integration tests

func TestAPI_GetDriftResult(t *testing.T) {
	ctx := context.Background()

	// Create a test drift result
	testResult := createTestDriftResult()
	err := testDB.Create(ctx, testResult)
	require.NoError(t, err)

	// Test API endpoint
	w := makeRequest("GET", fmt.Sprintf("/api/v1/drift/results/%s", testResult.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftResult
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, testResult.ID, response.ID)
	assert.Equal(t, testResult.Provider, response.Provider)

	// Cleanup
	testDB.Delete(ctx, testResult.ID)
}

func TestAPI_ListDriftResults(t *testing.T) {
	ctx := context.Background()

	// Create test data
	testResults := []*models.DriftResult{
		createTestDriftResult(),
		createTestDriftResult(),
	}

	// Create all test results
	for _, result := range testResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}

	// Test API endpoint
	w := makeRequest("GET", "/api/v1/drift/results", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.PaginatedDriftResults
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Total >= 2)
	assert.True(t, len(response.Results) >= 2)

	// Cleanup
	for _, result := range testResults {
		testDB.Delete(ctx, result.ID)
	}
}

func TestAPI_GetDriftHistory(t *testing.T) {
	ctx := context.Background()

	// Create test data
	testResult := createTestDriftResult()
	err := testDB.Create(ctx, testResult)
	require.NoError(t, err)

	// Test API endpoint
	w := makeRequest("GET", "/api/v1/drift/history", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftHistoryResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Total >= 1)
	assert.True(t, len(response.Results) >= 1)

	// Cleanup
	testDB.Delete(ctx, testResult.ID)
}

func TestAPI_GetDriftSummary(t *testing.T) {
	ctx := context.Background()

	// Create test data
	testResult := createTestDriftResult()
	err := testDB.Create(ctx, testResult)
	require.NoError(t, err)

	// Test API endpoint
	w := makeRequest("GET", "/api/v1/drift/summary?provider=aws", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftSummaryResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "aws", response.Provider)
	assert.True(t, response.Summary.TotalResources >= 1)

	// Cleanup
	testDB.Delete(ctx, testResult.ID)
}

func TestAPI_DeleteDriftResult(t *testing.T) {
	ctx := context.Background()

	// Create a test drift result
	testResult := createTestDriftResult()
	err := testDB.Create(ctx, testResult)
	require.NoError(t, err)

	// Test API endpoint
	w := makeRequest("DELETE", fmt.Sprintf("/api/v1/drift/results/%s", testResult.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response models.DriftResultResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, testResult.ID, response.ID)
	assert.Equal(t, "deleted", response.Status)

	// Verify deletion
	_, err = testDB.GetByID(ctx, testResult.ID)
	assert.Equal(t, models.ErrDriftResultNotFound, err)
}

func TestAPI_Health(t *testing.T) {
	// Test health endpoint
	w := makeRequest("GET", "/api/v1/drift/health", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// Performance tests

func TestDriftResult_Performance(t *testing.T) {
	ctx := context.Background()

	// Create many test results
	numResults := 100
	testResults := make([]*models.DriftResult, numResults)

	for i := 0; i < numResults; i++ {
		testResults[i] = createTestDriftResult()
		testResults[i].ID = fmt.Sprintf("perf-test-%d", i)
	}

	// Measure creation time
	start := time.Now()
	for _, result := range testResults {
		err := testDB.Create(ctx, result)
		require.NoError(t, err)
	}
	creationTime := time.Since(start)

	// Measure query time
	start = time.Now()
	query := &models.DriftResultQuery{
		Filter: models.DriftResultFilter{},
		Sort:   models.DriftResultSort{Field: "timestamp", Order: "desc"},
		Limit:  50,
		Offset: 0,
	}
	results, err := testDB.List(ctx, query)
	require.NoError(t, err)
	queryTime := time.Since(start)

	// Verify results
	assert.Equal(t, numResults, results.Total)
	assert.Len(t, results.Results, 50) // Should return limit

	// Performance assertions
	assert.True(t, creationTime < time.Second*5, "Creation should be fast")
	assert.True(t, queryTime < time.Millisecond*100, "Query should be very fast")

	// Cleanup
	for _, result := range testResults {
		testDB.Delete(ctx, result.ID)
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
