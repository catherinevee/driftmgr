package cqrs

import (
	"context"
	"fmt"
	"time"
)

// Query represents a query in the CQRS pattern
type Query interface {
	Name() string
	Validate() error
}

// QueryHandler handles queries
type QueryHandler interface {
	Handle(ctx context.Context, query Query) (interface{}, error)
}

// QueryBus dispatches queries to handlers
type QueryBus struct {
	handlers map[string]QueryHandler
}

// NewQueryBus creates a new query bus
func NewQueryBus() *QueryBus {
	return &QueryBus{
		handlers: make(map[string]QueryHandler),
	}
}

// Register registers a query handler
func (b *QueryBus) Register(queryName string, handler QueryHandler) {
	b.handlers[queryName] = handler
}

// Dispatch dispatches a query to its handler
func (b *QueryBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	handler, exists := b.handlers[query.Name()]
	if !exists {
		return nil, fmt.Errorf("no handler registered for query: %s", query.Name())
	}

	return handler.Handle(ctx, query)
}

// Discovery Queries

// GetDiscoveryStatusQuery gets the status of a discovery job
type GetDiscoveryStatusQuery struct {
	JobID string
}

func (q GetDiscoveryStatusQuery) Name() string { return "GetDiscoveryStatus" }

func (q GetDiscoveryStatusQuery) Validate() error {
	if q.JobID == "" {
		return fmt.Errorf("job ID is required")
	}
	return nil
}

// GetCachedResourcesQuery gets cached resources
type GetCachedResourcesQuery struct {
	Provider string
	Region   string
}

func (q GetCachedResourcesQuery) Name() string { return "GetCachedResources" }

func (q GetCachedResourcesQuery) Validate() error {
	return nil
}

// State Queries

// GetStateFileQuery gets a state file by ID
type GetStateFileQuery struct {
	FileID string
}

func (q GetStateFileQuery) Name() string { return "GetStateFile" }

func (q GetStateFileQuery) Validate() error {
	if q.FileID == "" {
		return fmt.Errorf("file ID is required")
	}
	return nil
}

// ListStateFilesQuery lists all state files
type ListStateFilesQuery struct {
	Filter string
	Limit  int
}

func (q ListStateFilesQuery) Name() string { return "ListStateFiles" }

func (q ListStateFilesQuery) Validate() error {
	return nil
}

// GetStateAnalysisQuery gets state analysis results
type GetStateAnalysisQuery struct {
	AnalysisID string
}

func (q GetStateAnalysisQuery) Name() string { return "GetStateAnalysis" }

func (q GetStateAnalysisQuery) Validate() error {
	return nil
}

// Drift Queries

// GetDriftReportQuery gets a drift report
type GetDriftReportQuery struct {
	ReportID string
}

func (q GetDriftReportQuery) Name() string { return "GetDriftReport" }

func (q GetDriftReportQuery) Validate() error {
	if q.ReportID == "" {
		return fmt.Errorf("report ID is required")
	}
	return nil
}

// ListDriftReportsQuery lists drift reports
type ListDriftReportsQuery struct {
	Provider  string
	StartDate time.Time
	EndDate   time.Time
	Limit     int
}

func (q ListDriftReportsQuery) Name() string { return "ListDriftReports" }

func (q ListDriftReportsQuery) Validate() error {
	return nil
}

// Remediation Queries

// GetRemediationPlanQuery gets a remediation plan
type GetRemediationPlanQuery struct {
	PlanID string
}

func (q GetRemediationPlanQuery) Name() string { return "GetRemediationPlan" }

func (q GetRemediationPlanQuery) Validate() error {
	if q.PlanID == "" {
		return fmt.Errorf("plan ID is required")
	}
	return nil
}

// GetRemediationResultsQuery gets remediation results
type GetRemediationResultsQuery struct {
	JobID string
}

func (q GetRemediationResultsQuery) Name() string { return "GetRemediationResults" }

func (q GetRemediationResultsQuery) Validate() error {
	if q.JobID == "" {
		return fmt.Errorf("job ID is required")
	}
	return nil
}

// Job Queries

// GetJobStatusQuery gets the status of a job
type GetJobStatusQuery struct {
	JobID string
}

func (q GetJobStatusQuery) Name() string { return "GetJobStatus" }

func (q GetJobStatusQuery) Validate() error {
	if q.JobID == "" {
		return fmt.Errorf("job ID is required")
	}
	return nil
}

// ListJobsQuery lists jobs
type ListJobsQuery struct {
	Type   string
	Status string
	Limit  int
}

func (q ListJobsQuery) Name() string { return "ListJobs" }

func (q ListJobsQuery) Validate() error {
	return nil
}

// Statistics Queries

// GetStatisticsQuery gets system statistics
type GetStatisticsQuery struct {
	Provider  string
	TimeRange string // "hour", "day", "week", "month"
}

func (q GetStatisticsQuery) Name() string { return "GetStatistics" }

func (q GetStatisticsQuery) Validate() error {
	return nil
}

// GetComplianceScoreQuery gets compliance score
type GetComplianceScoreQuery struct {
	Provider    string
	StateFileID string
}

func (q GetComplianceScoreQuery) Name() string { return "GetComplianceScore" }

func (q GetComplianceScoreQuery) Validate() error {
	return nil
}

// Audit Queries

// GetAuditLogsQuery gets audit logs
type GetAuditLogsQuery struct {
	StartDate time.Time
	EndDate   time.Time
	User      string
	Action    string
	Limit     int
}

func (q GetAuditLogsQuery) Name() string { return "GetAuditLogs" }

func (q GetAuditLogsQuery) Validate() error {
	return nil
}

// Provider Queries

// GetProviderStatusQuery gets provider status
type GetProviderStatusQuery struct {
	Provider string
}

func (q GetProviderStatusQuery) Name() string { return "GetProviderStatus" }

func (q GetProviderStatusQuery) Validate() error {
	if q.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	return nil
}

// ListProvidersQuery lists configured providers
type ListProvidersQuery struct{}

func (q ListProvidersQuery) Name() string { return "ListProviders" }

func (q ListProvidersQuery) Validate() error {
	return nil
}

// Query Results

// QueryResult represents the result of a query execution
type QueryResult struct {
	Data      interface{}
	Count     int
	Cached    bool
	Timestamp time.Time
}

// NewQueryResult creates a new query result
func NewQueryResult(data interface{}, count int, cached bool) QueryResult {
	return QueryResult{
		Data:      data,
		Count:     count,
		Cached:    cached,
		Timestamp: time.Now(),
	}
}