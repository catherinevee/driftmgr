package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Engine represents the main analytics engine
type Engine struct {
	aggregator *Aggregator
	calculator *Calculator
	predictor  *Predictor
	reporter   *Reporter
	visualizer *Visualizer
	queries    map[string]*models.AnalyticsQuery
	results    map[string]*models.AnalyticsResult
	mu         sync.RWMutex
}

// NewEngine creates a new analytics engine
func NewEngine() *Engine {
	return &Engine{
		aggregator: NewAggregator(),
		calculator: NewCalculator(),
		predictor:  NewPredictor(),
		reporter:   NewReporter(),
		visualizer: NewVisualizer(),
		queries:    make(map[string]*models.AnalyticsQuery),
		results:    make(map[string]*models.AnalyticsResult),
	}
}

// CreateQuery creates a new analytics query
func (e *Engine) CreateQuery(ctx context.Context, req *models.AnalyticsQueryCreateRequest) (*models.AnalyticsQuery, error) {
	query := &models.AnalyticsQuery{
		ID:           generateQueryID(),
		Name:         req.Name,
		Description:  req.Description,
		QueryType:    req.QueryType,
		Parameters:   req.Parameters,
		Filters:      req.Filters,
		GroupBy:      req.GroupBy,
		Aggregations: req.Aggregations,
		TimeRange:    req.TimeRange,
		IsPublic:     req.IsPublic,
		CreatedBy:    getCurrentUser(ctx),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.queries[query.ID] = query
	return query, nil
}

// GetQuery retrieves an analytics query by ID
func (e *Engine) GetQuery(ctx context.Context, queryID string) (*models.AnalyticsQuery, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	query, exists := e.queries[queryID]
	if !exists {
		return nil, fmt.Errorf("query %s not found", queryID)
	}

	return query, nil
}

// ListQueries lists analytics queries with optional filtering
func (e *Engine) ListQueries(ctx context.Context, req *models.AnalyticsQueryListRequest) (*models.AnalyticsQueryListResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var queries []models.AnalyticsQuery
	for _, query := range e.queries {
		// Apply filters
		if req.QueryType != nil && query.QueryType != *req.QueryType {
			continue
		}
		if req.IsPublic != nil && query.IsPublic != *req.IsPublic {
			continue
		}
		if req.CreatedBy != nil && query.CreatedBy != *req.CreatedBy {
			continue
		}

		queries = append(queries, *query)
	}

	// Apply pagination
	total := len(queries)
	start := req.Offset
	end := start + req.Limit

	if start >= total {
		return &models.AnalyticsQueryListResponse{
			Queries: []models.AnalyticsQuery{},
			Total:   total,
			Limit:   req.Limit,
			Offset:  req.Offset,
		}, nil
	}

	if end > total {
		end = total
	}

	return &models.AnalyticsQueryListResponse{
		Queries: queries[start:end],
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// UpdateQuery updates an analytics query
func (e *Engine) UpdateQuery(ctx context.Context, queryID string, req *models.AnalyticsQueryCreateRequest) (*models.AnalyticsQuery, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	query, exists := e.queries[queryID]
	if !exists {
		return nil, fmt.Errorf("query %s not found", queryID)
	}

	// Update query fields
	query.Name = req.Name
	query.Description = req.Description
	query.QueryType = req.QueryType
	query.Parameters = req.Parameters
	query.Filters = req.Filters
	query.GroupBy = req.GroupBy
	query.Aggregations = req.Aggregations
	query.TimeRange = req.TimeRange
	query.IsPublic = req.IsPublic
	query.UpdatedAt = time.Now()

	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	return query, nil
}

// DeleteQuery deletes an analytics query
func (e *Engine) DeleteQuery(ctx context.Context, queryID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.queries[queryID]; !exists {
		return fmt.Errorf("query %s not found", queryID)
	}

	delete(e.queries, queryID)
	return nil
}

// ExecuteQuery executes an analytics query
func (e *Engine) ExecuteQuery(ctx context.Context, queryID string, req *models.AnalyticsQueryExecuteRequest) (*models.AnalyticsResult, error) {
	// Get the query
	query, err := e.GetQuery(ctx, queryID)
	if err != nil {
		return nil, err
	}

	// Create result
	result := &models.AnalyticsResult{
		ID:          generateResultID(),
		QueryID:     queryID,
		Status:      models.AnalyticsResultStatusPending,
		GeneratedAt: time.Now(),
	}

	// Store result
	e.mu.Lock()
	e.results[result.ID] = result
	e.mu.Unlock()

	// Execute query
	go e.executeQueryAsync(ctx, query, result, req)

	return result, nil
}

// executeQueryAsync executes a query asynchronously
func (e *Engine) executeQueryAsync(ctx context.Context, query *models.AnalyticsQuery, result *models.AnalyticsResult, req *models.AnalyticsQueryExecuteRequest) {
	start := time.Now()
	result.Status = models.AnalyticsResultStatusRunning

	// Execute based on query type
	var data []map[string]interface{}
	var summary models.AnalyticsSummary
	var err error

	switch query.QueryType {
	case models.AnalyticsQueryTypeResourceCount:
		data, summary, err = e.executeResourceCountQuery(ctx, query, req)
	case models.AnalyticsQueryTypeCostAnalysis:
		data, summary, err = e.executeCostAnalysisQuery(ctx, query, req)
	case models.AnalyticsQueryTypeComplianceStatus:
		data, summary, err = e.executeComplianceStatusQuery(ctx, query, req)
	case models.AnalyticsQueryTypeDriftAnalysis:
		data, summary, err = e.executeDriftAnalysisQuery(ctx, query, req)
	case models.AnalyticsQueryTypePerformance:
		data, summary, err = e.executePerformanceQuery(ctx, query, req)
	case models.AnalyticsQueryTypeSecurity:
		data, summary, err = e.executeSecurityQuery(ctx, query, req)
	case models.AnalyticsQueryTypeTrend:
		data, summary, err = e.executeTrendQuery(ctx, query, req)
	case models.AnalyticsQueryTypeComparison:
		data, summary, err = e.executeComparisonQuery(ctx, query, req)
	default:
		err = fmt.Errorf("unsupported query type: %s", query.QueryType)
	}

	// Update result
	e.mu.Lock()
	defer e.mu.Unlock()

	result.ExecutionTime = time.Since(start)
	if err != nil {
		result.SetError(err)
	} else {
		result.Data = data
		result.Summary = summary
		result.Status = models.AnalyticsResultStatusCompleted
	}
}

// executeResourceCountQuery executes a resource count query
func (e *Engine) executeResourceCountQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate resource data
	data, err := e.aggregator.AggregateResources(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary
	summary, err := e.calculator.CalculateSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// executeCostAnalysisQuery executes a cost analysis query
func (e *Engine) executeCostAnalysisQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate cost data
	data, err := e.aggregator.AggregateCosts(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary
	summary, err := e.calculator.CalculateSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// executeComplianceStatusQuery executes a compliance status query
func (e *Engine) executeComplianceStatusQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate compliance data
	data, err := e.aggregator.AggregateCompliance(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary
	summary, err := e.calculator.CalculateSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// executeDriftAnalysisQuery executes a drift analysis query
func (e *Engine) executeDriftAnalysisQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate drift data
	data, err := e.aggregator.AggregateDrift(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary
	summary, err := e.calculator.CalculateSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// executePerformanceQuery executes a performance query
func (e *Engine) executePerformanceQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate performance data
	data, err := e.aggregator.AggregatePerformance(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary
	summary, err := e.calculator.CalculateSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// executeSecurityQuery executes a security query
func (e *Engine) executeSecurityQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate security data
	data, err := e.aggregator.AggregateSecurity(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary
	summary, err := e.calculator.CalculateSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// executeTrendQuery executes a trend query
func (e *Engine) executeTrendQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate trend data
	data, err := e.aggregator.AggregateTrends(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary with trend analysis
	summary, err := e.calculator.CalculateTrendSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// executeComparisonQuery executes a comparison query
func (e *Engine) executeComparisonQuery(ctx context.Context, query *models.AnalyticsQuery, req *models.AnalyticsQueryExecuteRequest) ([]map[string]interface{}, models.AnalyticsSummary, error) {
	// Aggregate comparison data
	data, err := e.aggregator.AggregateComparison(ctx, query, req)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	// Calculate summary with comparison analysis
	summary, err := e.calculator.CalculateComparisonSummary(data, query)
	if err != nil {
		return nil, models.AnalyticsSummary{}, err
	}

	return data, summary, nil
}

// GetResult retrieves an analytics result by ID
func (e *Engine) GetResult(ctx context.Context, resultID string) (*models.AnalyticsResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result, exists := e.results[resultID]
	if !exists {
		return nil, fmt.Errorf("result %s not found", resultID)
	}

	return result, nil
}

// ListResults lists analytics results with optional filtering
func (e *Engine) ListResults(ctx context.Context, req *models.AnalyticsResultListRequest) (*models.AnalyticsResultListResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []models.AnalyticsResult
	for _, result := range e.results {
		// Apply filters
		if req.QueryID != nil && result.QueryID != *req.QueryID {
			continue
		}
		if req.Status != nil && result.Status != *req.Status {
			continue
		}
		if req.StartTime != nil && result.GeneratedAt.Before(*req.StartTime) {
			continue
		}
		if req.EndTime != nil && result.GeneratedAt.After(*req.EndTime) {
			continue
		}

		results = append(results, *result)
	}

	// Apply pagination
	total := len(results)
	start := req.Offset
	end := start + req.Limit

	if start >= total {
		return &models.AnalyticsResultListResponse{
			Results: []models.AnalyticsResult{},
			Total:   total,
			Limit:   req.Limit,
			Offset:  req.Offset,
		}, nil
	}

	if end > total {
		end = total
	}

	return &models.AnalyticsResultListResponse{
		Results: results[start:end],
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// GenerateReport generates a report from analytics results
func (e *Engine) GenerateReport(ctx context.Context, queryIDs []string, format models.ReportFormat) ([]byte, error) {
	var results []models.AnalyticsResult

	// Get results for all queries
	for _, queryID := range queryIDs {
		result, err := e.GetResult(ctx, queryID)
		if err != nil {
			return nil, fmt.Errorf("failed to get result for query %s: %w", queryID, err)
		}
		results = append(results, *result)
	}

	// Generate report
	return e.reporter.GenerateReport(ctx, results, format)
}

// GetDashboardData gets data for dashboard widgets
func (e *Engine) GetDashboardData(ctx context.Context, widgetIDs []string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	for _, widgetID := range widgetIDs {
		widgetData, err := e.getWidgetData(ctx, widgetID)
		if err != nil {
			return nil, fmt.Errorf("failed to get data for widget %s: %w", widgetID, err)
		}
		data[widgetID] = widgetData
	}

	return data, nil
}

// getWidgetData gets data for a specific widget
func (e *Engine) getWidgetData(ctx context.Context, widgetID string) (interface{}, error) {
	// This would typically query the database for widget configuration
	// and execute the associated query
	return map[string]interface{}{
		"widget_id": widgetID,
		"data":      []interface{}{},
		"metadata":  map[string]interface{}{},
	}, nil
}

// GetStatistics returns analytics engine statistics
func (e *Engine) GetStatistics() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"total_queries":     len(e.queries),
		"total_results":     len(e.results),
		"active_queries":    e.getActiveQueryCount(),
		"completed_results": e.getCompletedResultCount(),
		"failed_results":    e.getFailedResultCount(),
	}
}

// Helper methods

func generateQueryID() string {
	return fmt.Sprintf("query-%d", time.Now().UnixNano())
}

func generateResultID() string {
	return fmt.Sprintf("result-%d", time.Now().UnixNano())
}

func getCurrentUser(ctx context.Context) string {
	// In a real implementation, this would extract the user from the context
	return "system"
}

func (e *Engine) getActiveQueryCount() int {
	count := 0
	for _, result := range e.results {
		if result.Status == models.AnalyticsResultStatusRunning {
			count++
		}
	}
	return count
}

func (e *Engine) getCompletedResultCount() int {
	count := 0
	for _, result := range e.results {
		if result.Status == models.AnalyticsResultStatusCompleted {
			count++
		}
	}
	return count
}

func (e *Engine) getFailedResultCount() int {
	count := 0
	for _, result := range e.results {
		if result.Status == models.AnalyticsResultStatusFailed {
			count++
		}
	}
	return count
}
