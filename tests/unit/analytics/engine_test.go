package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/analytics"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the analytics repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateQuery(ctx context.Context, query *models.AnalyticsQuery) error {
	args := m.Called(ctx, query)
	return args.Error(0)
}

func (m *MockRepository) GetQuery(ctx context.Context, id uuid.UUID) (*models.AnalyticsQuery, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AnalyticsQuery), args.Error(1)
}

func (m *MockRepository) UpdateQuery(ctx context.Context, query *models.AnalyticsQuery) error {
	args := m.Called(ctx, query)
	return args.Error(0)
}

func (m *MockRepository) DeleteQuery(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListQueries(ctx context.Context, filter analytics.QueryFilter) ([]*models.AnalyticsQuery, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*models.AnalyticsQuery), args.Error(1)
}

func (m *MockRepository) GetQueryStats(ctx context.Context, id uuid.UUID) (*analytics.QueryStats, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*analytics.QueryStats), args.Error(1)
}

func (m *MockRepository) CreateResult(ctx context.Context, result *models.AnalyticsResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockRepository) GetResult(ctx context.Context, id uuid.UUID) (*models.AnalyticsResult, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AnalyticsResult), args.Error(1)
}

func (m *MockRepository) ListResults(ctx context.Context, filter analytics.ResultFilter) ([]*models.AnalyticsResult, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*models.AnalyticsResult), args.Error(1)
}

func (m *MockRepository) GetResultHistory(ctx context.Context, queryID uuid.UUID, limit int) ([]*models.AnalyticsResult, error) {
	args := m.Called(ctx, queryID, limit)
	return args.Get(0).([]*models.AnalyticsResult), args.Error(1)
}

func (m *MockRepository) GetResultStats(ctx context.Context, queryID uuid.UUID) (*analytics.ResultStats, error) {
	args := m.Called(ctx, queryID)
	return args.Get(0).(*analytics.ResultStats), args.Error(1)
}

func (m *MockRepository) CreateReport(ctx context.Context, report *models.AnalyticsReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockRepository) GetReport(ctx context.Context, id uuid.UUID) (*models.AnalyticsReport, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AnalyticsReport), args.Error(1)
}

func (m *MockRepository) UpdateReport(ctx context.Context, report *models.AnalyticsReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockRepository) DeleteReport(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListReports(ctx context.Context, filter analytics.ReportFilter) ([]*models.AnalyticsReport, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*models.AnalyticsReport), args.Error(1)
}

func (m *MockRepository) GetReportStats(ctx context.Context, id uuid.UUID) (*analytics.ReportStats, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*analytics.ReportStats), args.Error(1)
}

func (m *MockRepository) CreateDashboard(ctx context.Context, dashboard *models.AnalyticsDashboard) error {
	args := m.Called(ctx, dashboard)
	return args.Error(0)
}

func (m *MockRepository) GetDashboard(ctx context.Context, id uuid.UUID) (*models.AnalyticsDashboard, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AnalyticsDashboard), args.Error(1)
}

func (m *MockRepository) UpdateDashboard(ctx context.Context, dashboard *models.AnalyticsDashboard) error {
	args := m.Called(ctx, dashboard)
	return args.Error(0)
}

func (m *MockRepository) DeleteDashboard(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListDashboards(ctx context.Context, filter analytics.DashboardFilter) ([]*models.AnalyticsDashboard, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*models.AnalyticsDashboard), args.Error(1)
}

func (m *MockRepository) GetDashboardStats(ctx context.Context, id uuid.UUID) (*analytics.DashboardStats, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*analytics.DashboardStats), args.Error(1)
}

func (m *MockRepository) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) GetQueryCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetResultCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetReportCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetDashboardCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetActiveQueriesCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// MockAggregator is a mock implementation of the data aggregator
type MockAggregator struct {
	mock.Mock
}

func (m *MockAggregator) AggregateData(ctx context.Context, query *models.AnalyticsQuery) (*models.AnalyticsResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(*models.AnalyticsResult), args.Error(1)
}

func (m *MockAggregator) GetDataSources(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAggregator) ValidateDataSource(ctx context.Context, source string) error {
	args := m.Called(ctx, source)
	return args.Error(0)
}

// MockCalculator is a mock implementation of the statistical calculator
type MockCalculator struct {
	mock.Mock
}

func (m *MockCalculator) CalculateStatistics(ctx context.Context, data []interface{}) (*models.StatisticalSummary, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(*models.StatisticalSummary), args.Error(1)
}

func (m *MockCalculator) CalculateTrend(ctx context.Context, data []interface{}) (*models.Trend, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(*models.Trend), args.Error(1)
}

func (m *MockCalculator) CalculateCorrelation(ctx context.Context, data1, data2 []interface{}) (float64, error) {
	args := m.Called(ctx, data1, data2)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockCalculator) CalculateRegression(ctx context.Context, data []interface{}) (*models.Regression, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(*models.Regression), args.Error(1)
}

// MockPredictor is a mock implementation of the predictive analytics
type MockPredictor struct {
	mock.Mock
}

func (m *MockPredictor) PredictCost(ctx context.Context, input *models.PredictionInput) (*models.PredictionOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*models.PredictionOutput), args.Error(1)
}

func (m *MockPredictor) PredictUsage(ctx context.Context, input *models.PredictionInput) (*models.PredictionOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*models.PredictionOutput), args.Error(1)
}

func (m *MockPredictor) PredictDrift(ctx context.Context, input *models.PredictionInput) (*models.PredictionOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*models.PredictionOutput), args.Error(1)
}

func (m *MockPredictor) PredictPerformance(ctx context.Context, input *models.PredictionInput) (*models.PredictionOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*models.PredictionOutput), args.Error(1)
}

// MockReporter is a mock implementation of the report generator
type MockReporter struct {
	mock.Mock
}

func (m *MockReporter) GenerateReport(ctx context.Context, query *models.AnalyticsQuery, result *models.AnalyticsResult) (*models.AnalyticsReport, error) {
	args := m.Called(ctx, query, result)
	return args.Get(0).(*models.AnalyticsReport), args.Error(1)
}

func (m *MockReporter) ExportReport(ctx context.Context, report *models.AnalyticsReport, format string) ([]byte, error) {
	args := m.Called(ctx, report, format)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReporter) ScheduleReport(ctx context.Context, report *models.AnalyticsReport, schedule string) error {
	args := m.Called(ctx, report, schedule)
	return args.Error(0)
}

func (m *MockReporter) GetSupportedFormats() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// MockVisualizer is a mock implementation of the data visualizer
type MockVisualizer struct {
	mock.Mock
}

func (m *MockVisualizer) GenerateChart(ctx context.Context, data []interface{}, chartType string) (*models.Chart, error) {
	args := m.Called(ctx, data, chartType)
	return args.Get(0).(*models.Chart), args.Error(1)
}

func (m *MockVisualizer) GenerateDashboard(ctx context.Context, queries []*models.AnalyticsQuery) (*models.AnalyticsDashboard, error) {
	args := m.Called(ctx, queries)
	return args.Get(0).(*models.AnalyticsDashboard), args.Error(1)
}

func (m *MockVisualizer) ExportChart(ctx context.Context, chart *models.Chart, format string) ([]byte, error) {
	args := m.Called(ctx, chart, format)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockVisualizer) GetSupportedChartTypes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestAnalyticsEngine_CreateQuery(t *testing.T) {
	// Setup
	mockRepo := new(MockRepository)
	mockAggregator := new(MockAggregator)
	mockCalculator := new(MockCalculator)
	mockPredictor := new(MockPredictor)
	mockReporter := new(MockReporter)
	mockVisualizer := new(MockVisualizer)

	config := analytics.EngineConfig{
		MaxQueriesPerUser:  100,
		MaxResultsPerHour:  1000,
		QueryTimeout:       30 * time.Second,
		EnableEventLogging: true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
	}

	engine := analytics.NewEngine(
		mockRepo,
		mockAggregator,
		mockCalculator,
		mockPredictor,
		mockReporter,
		mockVisualizer,
		config,
	)

	userID := uuid.New()
	req := &models.AnalyticsQueryRequest{
		Name:         "Test Query",
		Description:  "Test query description",
		Type:         models.QueryTypeResourceCount,
		DataSource:   "test_source",
		Filters:      []models.QueryFilter{},
		Aggregations: []models.Aggregation{},
		GroupBy:      []string{},
		TimeRange:    models.TimeRange{},
		Tags:         []string{"test"},
	}

	// Mock expectations
	mockRepo.On("CreateQuery", mock.Anything, mock.AnythingOfType("*models.AnalyticsQuery")).Return(nil)

	// Execute
	query, err := engine.CreateQuery(context.Background(), userID, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, query)
	assert.Equal(t, req.Name, query.Name)
	assert.Equal(t, req.Description, query.Description)
	assert.Equal(t, req.Type, query.Type)
	assert.Equal(t, userID, query.UserID)

	mockRepo.AssertExpectations(t)
}

func TestAnalyticsEngine_ExecuteQuery(t *testing.T) {
	// Setup
	mockRepo := new(MockRepository)
	mockAggregator := new(MockAggregator)
	mockCalculator := new(MockCalculator)
	mockPredictor := new(MockPredictor)
	mockReporter := new(MockReporter)
	mockVisualizer := new(MockVisualizer)

	config := analytics.EngineConfig{
		MaxQueriesPerUser:  100,
		MaxResultsPerHour:  1000,
		QueryTimeout:       30 * time.Second,
		EnableEventLogging: true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
	}

	engine := analytics.NewEngine(
		mockRepo,
		mockRepo,
		mockAggregator,
		mockCalculator,
		mockPredictor,
		mockReporter,
		mockVisualizer,
		config,
	)

	userID := uuid.New()
	queryID := uuid.New()
	query := &models.AnalyticsQuery{
		ID:         queryID,
		UserID:     userID,
		Name:       "Test Query",
		Type:       models.QueryTypeResourceCount,
		DataSource: "test_source",
		Status:     models.QueryStatusDraft,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Mock expectations
	mockRepo.On("GetQuery", mock.Anything, queryID).Return(query, nil)
	mockAggregator.On("AggregateData", mock.Anything, query).Return(&models.AnalyticsResult{
		ID:        uuid.New(),
		QueryID:   queryID,
		Status:    models.ResultStatusCompleted,
		Data:      models.JSONB([]interface{}{1, 2, 3}),
		CreatedAt: time.Now(),
	}, nil)
	mockRepo.On("CreateResult", mock.Anything, mock.AnythingOfType("*models.AnalyticsResult")).Return(nil)

	// Execute
	result, err := engine.ExecuteQuery(context.Background(), userID, queryID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queryID, result.QueryID)
	assert.Equal(t, models.ResultStatusCompleted, result.Status)

	mockRepo.AssertExpectations(t)
	mockAggregator.AssertExpectations(t)
}

func TestAnalyticsEngine_GenerateReport(t *testing.T) {
	// Setup
	mockRepo := new(MockRepository)
	mockAggregator := new(MockAggregator)
	mockCalculator := new(MockCalculator)
	mockPredictor := new(MockPredictor)
	mockReporter := new(MockReporter)
	mockVisualizer := new(MockVisualizer)

	config := analytics.EngineConfig{
		MaxQueriesPerUser:  100,
		MaxResultsPerHour:  1000,
		QueryTimeout:       30 * time.Second,
		EnableEventLogging: true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
	}

	engine := analytics.NewEngine(
		mockRepo,
		mockRepo,
		mockAggregator,
		mockCalculator,
		mockPredictor,
		mockReporter,
		mockVisualizer,
		config,
	)

	userID := uuid.New()
	queryID := uuid.New()
	query := &models.AnalyticsQuery{
		ID:         queryID,
		UserID:     userID,
		Name:       "Test Query",
		Type:       models.QueryTypeResourceCount,
		DataSource: "test_source",
		Status:     models.QueryStatusDraft,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	result := &models.AnalyticsResult{
		ID:        uuid.New(),
		QueryID:   queryID,
		Status:    models.ResultStatusCompleted,
		Data:      models.JSONB([]interface{}{1, 2, 3}),
		CreatedAt: time.Now(),
	}

	// Mock expectations
	mockRepo.On("GetQuery", mock.Anything, queryID).Return(query, nil)
	mockRepo.On("GetResult", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(result, nil)
	mockReporter.On("GenerateReport", mock.Anything, query, result).Return(&models.AnalyticsReport{
		ID:        uuid.New(),
		QueryID:   queryID,
		UserID:    userID,
		Name:      "Test Report",
		Format:    "pdf",
		Status:    models.ReportStatusCompleted,
		CreatedAt: time.Now(),
	}, nil)
	mockRepo.On("CreateReport", mock.Anything, mock.AnythingOfType("*models.AnalyticsReport")).Return(nil)

	// Execute
	report, err := engine.GenerateReport(context.Background(), userID, queryID, "pdf")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, queryID, report.QueryID)
	assert.Equal(t, userID, report.UserID)
	assert.Equal(t, "pdf", report.Format)

	mockRepo.AssertExpectations(t)
	mockReporter.AssertExpectations(t)
}

func TestAnalyticsEngine_GenerateChart(t *testing.T) {
	// Setup
	mockRepo := new(MockRepository)
	mockAggregator := new(MockAggregator)
	mockCalculator := new(MockCalculator)
	mockPredictor := new(MockPredictor)
	mockReporter := new(MockReporter)
	mockVisualizer := new(MockVisualizer)

	config := analytics.EngineConfig{
		MaxQueriesPerUser:  100,
		MaxResultsPerHour:  1000,
		QueryTimeout:       30 * time.Second,
		EnableEventLogging: true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
	}

	engine := analytics.NewEngine(
		mockRepo,
		mockRepo,
		mockAggregator,
		mockCalculator,
		mockPredictor,
		mockReporter,
		mockVisualizer,
		config,
	)

	userID := uuid.New()
	queryID := uuid.New()
	query := &models.AnalyticsQuery{
		ID:         queryID,
		UserID:     userID,
		Name:       "Test Query",
		Type:       models.QueryTypeResourceCount,
		DataSource: "test_source",
		Status:     models.QueryStatusDraft,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	result := &models.AnalyticsResult{
		ID:        uuid.New(),
		QueryID:   queryID,
		Status:    models.ResultStatusCompleted,
		Data:      models.JSONB([]interface{}{1, 2, 3}),
		CreatedAt: time.Now(),
	}

	// Mock expectations
	mockRepo.On("GetQuery", mock.Anything, queryID).Return(query, nil)
	mockRepo.On("GetResult", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(result, nil)
	mockVisualizer.On("GenerateChart", mock.Anything, mock.AnythingOfType("[]interface{}"), "line").Return(&models.Chart{
		ID:        uuid.New(),
		QueryID:   queryID,
		Type:      "line",
		Data:      models.JSONB([]interface{}{1, 2, 3}),
		CreatedAt: time.Now(),
	}, nil)

	// Execute
	chart, err := engine.GenerateChart(context.Background(), userID, queryID, "line")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, chart)
	assert.Equal(t, queryID, chart.QueryID)
	assert.Equal(t, "line", chart.Type)

	mockRepo.AssertExpectations(t)
	mockVisualizer.AssertExpectations(t)
}

func TestAnalyticsEngine_Health(t *testing.T) {
	// Setup
	mockRepo := new(MockRepository)
	mockAggregator := new(MockAggregator)
	mockCalculator := new(MockCalculator)
	mockPredictor := new(MockPredictor)
	mockReporter := new(MockReporter)
	mockVisualizer := new(MockVisualizer)

	config := analytics.EngineConfig{
		MaxQueriesPerUser:  100,
		MaxResultsPerHour:  1000,
		QueryTimeout:       30 * time.Second,
		EnableEventLogging: true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
	}

	engine := analytics.NewEngine(
		mockRepo,
		mockRepo,
		mockAggregator,
		mockCalculator,
		mockPredictor,
		mockReporter,
		mockVisualizer,
		config,
	)

	// Mock expectations
	mockRepo.On("Health", mock.Anything).Return(nil)
	mockRepo.On("GetQueryCount", mock.Anything).Return(10, nil)
	mockRepo.On("GetResultCount", mock.Anything).Return(100, nil)
	mockRepo.On("GetReportCount", mock.Anything).Return(5, nil)
	mockRepo.On("GetDashboardCount", mock.Anything).Return(2, nil)
	mockRepo.On("GetActiveQueriesCount", mock.Anything).Return(3, nil)

	// Execute
	health, err := engine.Health(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "analytics", health.Service)
	assert.Equal(t, 10, health.QueryCount)
	assert.Equal(t, 100, health.ResultCount)
	assert.Equal(t, 5, health.ReportCount)
	assert.Equal(t, 2, health.DashboardCount)
	assert.Equal(t, 3, health.ActiveQueriesCount)

	mockRepo.AssertExpectations(t)
}
