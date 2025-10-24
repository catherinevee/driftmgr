package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// AnalyticsQueryType represents the type of analytics query
type AnalyticsQueryType string

const (
	AnalyticsQueryTypeResourceCount    AnalyticsQueryType = "resource_count"
	AnalyticsQueryTypeCostAnalysis     AnalyticsQueryType = "cost_analysis"
	AnalyticsQueryTypeComplianceStatus AnalyticsQueryType = "compliance_status"
	AnalyticsQueryTypeDriftAnalysis    AnalyticsQueryType = "drift_analysis"
	AnalyticsQueryTypePerformance      AnalyticsQueryType = "performance"
	AnalyticsQueryTypeSecurity         AnalyticsQueryType = "security"
	AnalyticsQueryTypeTrend            AnalyticsQueryType = "trend"
	AnalyticsQueryTypeComparison       AnalyticsQueryType = "comparison"
	AnalyticsQueryTypeCustom           AnalyticsQueryType = "custom"
)

// String returns the string representation of AnalyticsQueryType
func (aqt AnalyticsQueryType) String() string {
	return string(aqt)
}

// AnalyticsQuery represents an analytics query
type AnalyticsQuery struct {
	ID           string                 `json:"id" db:"id" validate:"required,uuid"`
	Name         string                 `json:"name" db:"name" validate:"required"`
	Description  string                 `json:"description" db:"description"`
	QueryType    AnalyticsQueryType     `json:"query_type" db:"query_type" validate:"required"`
	Parameters   map[string]interface{} `json:"parameters" db:"parameters"`
	Filters      []AnalyticsFilter      `json:"filters" db:"filters"`
	GroupBy      []string               `json:"group_by" db:"group_by"`
	Aggregations []AnalyticsAggregation `json:"aggregations" db:"aggregations"`
	TimeRange    TimeRange              `json:"time_range" db:"time_range"`
	IsPublic     bool                   `json:"is_public" db:"is_public"`
	CreatedBy    string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// AnalyticsFilter represents a filter for analytics queries
type AnalyticsFilter struct {
	Field     string         `json:"field" db:"field" validate:"required"`
	Operator  FilterOperator `json:"operator" db:"operator" validate:"required"`
	Value     interface{}    `json:"value" db:"value" validate:"required"`
	ValueType string         `json:"value_type" db:"value_type" validate:"required"`
}

// FilterOperator represents the operator for filters
type FilterOperator string

const (
	FilterOperatorEquals       FilterOperator = "equals"
	FilterOperatorNotEquals    FilterOperator = "not_equals"
	FilterOperatorGreaterThan  FilterOperator = "greater_than"
	FilterOperatorLessThan     FilterOperator = "less_than"
	FilterOperatorGreaterEqual FilterOperator = "greater_equal"
	FilterOperatorLessEqual    FilterOperator = "less_equal"
	FilterOperatorContains     FilterOperator = "contains"
	FilterOperatorNotContains  FilterOperator = "not_contains"
	FilterOperatorStartsWith   FilterOperator = "starts_with"
	FilterOperatorEndsWith     FilterOperator = "ends_with"
	FilterOperatorIn           FilterOperator = "in"
	FilterOperatorNotIn        FilterOperator = "not_in"
	FilterOperatorIsNull       FilterOperator = "is_null"
	FilterOperatorIsNotNull    FilterOperator = "is_not_null"
)

// String returns the string representation of FilterOperator
func (fo FilterOperator) String() string {
	return string(fo)
}

// AnalyticsAggregation represents an aggregation for analytics queries
type AnalyticsAggregation struct {
	Field    string              `json:"field" db:"field" validate:"required"`
	Function AggregationFunction `json:"function" db:"function" validate:"required"`
	Alias    string              `json:"alias" db:"alias"`
}

// AggregationFunction represents the aggregation function
type AggregationFunction string

const (
	AggregationFunctionCount      AggregationFunction = "count"
	AggregationFunctionSum        AggregationFunction = "sum"
	AggregationFunctionAverage    AggregationFunction = "average"
	AggregationFunctionMin        AggregationFunction = "min"
	AggregationFunctionMax        AggregationFunction = "max"
	AggregationFunctionMedian     AggregationFunction = "median"
	AggregationFunctionStdDev     AggregationFunction = "std_dev"
	AggregationFunctionVariance   AggregationFunction = "variance"
	AggregationFunctionPercentile AggregationFunction = "percentile"
)

// String returns the string representation of AggregationFunction
func (af AggregationFunction) String() string {
	return string(af)
}

// TimeRange represents a time range for analytics queries
type TimeRange struct {
	StartTime *time.Time `json:"start_time" db:"start_time"`
	EndTime   *time.Time `json:"end_time" db:"end_time"`
	Duration  string     `json:"duration" db:"duration"` // e.g., "7d", "30d", "1y"
}

// AnalyticsResult represents the result of an analytics query
type AnalyticsResult struct {
	ID            string                   `json:"id" db:"id" validate:"required,uuid"`
	QueryID       string                   `json:"query_id" db:"query_id" validate:"required,uuid"`
	Data          []map[string]interface{} `json:"data" db:"data"`
	Metadata      map[string]interface{}   `json:"metadata" db:"metadata"`
	Summary       AnalyticsSummary         `json:"summary" db:"summary"`
	GeneratedAt   time.Time                `json:"generated_at" db:"generated_at"`
	ExecutionTime time.Duration            `json:"execution_time" db:"execution_time"`
	Status        AnalyticsResultStatus    `json:"status" db:"status"`
	Error         *string                  `json:"error,omitempty" db:"error"`
}

// AnalyticsSummary represents a summary of analytics results
type AnalyticsSummary struct {
	TotalRecords    int            `json:"total_records" db:"total_records"`
	TotalValue      float64        `json:"total_value" db:"total_value"`
	AverageValue    float64        `json:"average_value" db:"average_value"`
	MinValue        float64        `json:"min_value" db:"min_value"`
	MaxValue        float64        `json:"max_value" db:"max_value"`
	Trend           TrendDirection `json:"trend" db:"trend"`
	TrendPercentage float64        `json:"trend_percentage" db:"trend_percentage"`
	Insights        []string       `json:"insights" db:"insights"`
	Recommendations []string       `json:"recommendations" db:"recommendations"`
}

// AnalyticsResultStatus represents the status of an analytics result
type AnalyticsResultStatus string

const (
	AnalyticsResultStatusPending   AnalyticsResultStatus = "pending"
	AnalyticsResultStatusRunning   AnalyticsResultStatus = "running"
	AnalyticsResultStatusCompleted AnalyticsResultStatus = "completed"
	AnalyticsResultStatusFailed    AnalyticsResultStatus = "failed"
	AnalyticsResultStatusCancelled AnalyticsResultStatus = "cancelled"
)

// String returns the string representation of AnalyticsResultStatus
func (ars AnalyticsResultStatus) String() string {
	return string(ars)
}

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendDirectionUp      TrendDirection = "up"
	TrendDirectionDown    TrendDirection = "down"
	TrendDirectionStable  TrendDirection = "stable"
	TrendDirectionUnknown TrendDirection = "unknown"
)

// String returns the string representation of TrendDirection
func (td TrendDirection) String() string {
	return string(td)
}

// AnalyticsDashboard represents an analytics dashboard
type AnalyticsDashboard struct {
	ID              string            `json:"id" db:"id" validate:"required,uuid"`
	Name            string            `json:"name" db:"name" validate:"required"`
	Description     string            `json:"description" db:"description"`
	Layout          DashboardLayout   `json:"layout" db:"layout"`
	Widgets         []DashboardWidget `json:"widgets" db:"widgets"`
	Filters         []AnalyticsFilter `json:"filters" db:"filters"`
	TimeRange       TimeRange         `json:"time_range" db:"time_range"`
	IsPublic        bool              `json:"is_public" db:"is_public"`
	RefreshInterval int               `json:"refresh_interval" db:"refresh_interval"` // in seconds
	CreatedBy       string            `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

// DashboardLayout represents the layout of a dashboard
type DashboardLayout struct {
	Columns  int                    `json:"columns" db:"columns"`
	Rows     int                    `json:"rows" db:"rows"`
	GridSize int                    `json:"grid_size" db:"grid_size"`
	Widgets  []WidgetPosition       `json:"widgets" db:"widgets"`
	Theme    string                 `json:"theme" db:"theme"`
	Settings map[string]interface{} `json:"settings" db:"settings"`
}

// WidgetPosition represents the position of a widget in a dashboard
type WidgetPosition struct {
	WidgetID string `json:"widget_id" db:"widget_id"`
	X        int    `json:"x" db:"x"`
	Y        int    `json:"y" db:"y"`
	Width    int    `json:"width" db:"width"`
	Height   int    `json:"height" db:"height"`
}

// DashboardWidget represents a widget in a dashboard
type DashboardWidget struct {
	ID              string                 `json:"id" db:"id" validate:"required,uuid"`
	Type            WidgetType             `json:"type" db:"type" validate:"required"`
	Title           string                 `json:"title" db:"title" validate:"required"`
	Description     string                 `json:"description" db:"description"`
	QueryID         string                 `json:"query_id" db:"query_id"`
	Configuration   map[string]interface{} `json:"configuration" db:"configuration"`
	Position        WidgetPosition         `json:"position" db:"position"`
	RefreshInterval int                    `json:"refresh_interval" db:"refresh_interval"` // in seconds
	IsVisible       bool                   `json:"is_visible" db:"is_visible"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// WidgetType represents the type of a dashboard widget
type WidgetType string

const (
	WidgetTypeChart    WidgetType = "chart"
	WidgetTypeTable    WidgetType = "table"
	WidgetTypeMetric   WidgetType = "metric"
	WidgetTypeGauge    WidgetType = "gauge"
	WidgetTypeProgress WidgetType = "progress"
	WidgetTypeText     WidgetType = "text"
	WidgetTypeMap      WidgetType = "map"
	WidgetTypeHeatmap  WidgetType = "heatmap"
	WidgetTypeTimeline WidgetType = "timeline"
	WidgetTypeAlert    WidgetType = "alert"
	WidgetTypeCustom   WidgetType = "custom"
)

// String returns the string representation of WidgetType
func (wt WidgetType) String() string {
	return string(wt)
}

// AnalyticsReport represents an analytics report
type AnalyticsReport struct {
	ID             string                 `json:"id" db:"id" validate:"required,uuid"`
	Name           string                 `json:"name" db:"name" validate:"required"`
	Description    string                 `json:"description" db:"description"`
	Type           ReportType             `json:"type" db:"type" validate:"required"`
	Format         ReportFormat           `json:"format" db:"format" validate:"required"`
	QueryIDs       []string               `json:"query_ids" db:"query_ids"`
	DashboardID    *string                `json:"dashboard_id" db:"dashboard_id"`
	Template       string                 `json:"template" db:"template"`
	Parameters     map[string]interface{} `json:"parameters" db:"parameters"`
	Schedule       *ReportSchedule        `json:"schedule" db:"schedule"`
	Recipients     []string               `json:"recipients" db:"recipients"`
	IsActive       bool                   `json:"is_active" db:"is_active"`
	LastGenerated  *time.Time             `json:"last_generated" db:"last_generated"`
	NextGeneration *time.Time             `json:"next_generation" db:"next_generation"`
	CreatedBy      string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// ReportType represents the type of a report
type ReportType string

const (
	ReportTypeSummary     ReportType = "summary"
	ReportTypeDetailed    ReportType = "detailed"
	ReportTypeCompliance  ReportType = "compliance"
	ReportTypeCost        ReportType = "cost"
	ReportTypeSecurity    ReportType = "security"
	ReportTypePerformance ReportType = "performance"
	ReportTypeCustom      ReportType = "custom"
)

// String returns the string representation of ReportType
func (rt ReportType) String() string {
	return string(rt)
}

// ReportFormat represents the format of a report
type ReportFormat string

const (
	ReportFormatPDF   ReportFormat = "pdf"
	ReportFormatExcel ReportFormat = "excel"
	ReportFormatCSV   ReportFormat = "csv"
	ReportFormatJSON  ReportFormat = "json"
	ReportFormatHTML  ReportFormat = "html"
)

// String returns the string representation of ReportFormat
func (rf ReportFormat) String() string {
	return string(rf)
}

// ReportSchedule represents the schedule for a report
type ReportSchedule struct {
	Frequency  ScheduleFrequency `json:"frequency" db:"frequency" validate:"required"`
	DayOfWeek  *int              `json:"day_of_week" db:"day_of_week"`   // 0-6 (Sunday-Saturday)
	DayOfMonth *int              `json:"day_of_month" db:"day_of_month"` // 1-31
	Hour       int               `json:"hour" db:"hour"`                 // 0-23
	Minute     int               `json:"minute" db:"minute"`             // 0-59
	Timezone   string            `json:"timezone" db:"timezone"`
}

// ScheduleFrequency represents the frequency of a schedule
type ScheduleFrequency string

const (
	ScheduleFrequencyDaily   ScheduleFrequency = "daily"
	ScheduleFrequencyWeekly  ScheduleFrequency = "weekly"
	ScheduleFrequencyMonthly ScheduleFrequency = "monthly"
	ScheduleFrequencyYearly  ScheduleFrequency = "yearly"
)

// String returns the string representation of ScheduleFrequency
func (sf ScheduleFrequency) String() string {
	return string(sf)
}

// Request/Response Models

// AnalyticsQueryCreateRequest represents a request to create an analytics query
type AnalyticsQueryCreateRequest struct {
	Name         string                 `json:"name" validate:"required"`
	Description  string                 `json:"description"`
	QueryType    AnalyticsQueryType     `json:"query_type" validate:"required"`
	Parameters   map[string]interface{} `json:"parameters"`
	Filters      []AnalyticsFilter      `json:"filters"`
	GroupBy      []string               `json:"group_by"`
	Aggregations []AnalyticsAggregation `json:"aggregations"`
	TimeRange    TimeRange              `json:"time_range"`
	IsPublic     bool                   `json:"is_public"`
}

// AnalyticsQueryListRequest represents a request to list analytics queries
type AnalyticsQueryListRequest struct {
	QueryType *AnalyticsQueryType `json:"query_type,omitempty"`
	IsPublic  *bool               `json:"is_public,omitempty"`
	CreatedBy *string             `json:"created_by,omitempty"`
	Limit     int                 `json:"limit" validate:"min=1,max=1000"`
	Offset    int                 `json:"offset" validate:"min=0"`
	SortBy    string              `json:"sort_by" validate:"omitempty,oneof=name created_at updated_at"`
	SortOrder string              `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// AnalyticsQueryListResponse represents the response for listing analytics queries
type AnalyticsQueryListResponse struct {
	Queries []AnalyticsQuery `json:"queries"`
	Total   int              `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

// AnalyticsQueryExecuteRequest represents a request to execute an analytics query
type AnalyticsQueryExecuteRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
	Filters    []AnalyticsFilter      `json:"filters"`
	TimeRange  *TimeRange             `json:"time_range"`
}

// AnalyticsResultListRequest represents a request to list analytics results
type AnalyticsResultListRequest struct {
	QueryID   *string                `json:"query_id,omitempty"`
	Status    *AnalyticsResultStatus `json:"status,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Limit     int                    `json:"limit" validate:"min=1,max=1000"`
	Offset    int                    `json:"offset" validate:"min=0"`
	SortBy    string                 `json:"sort_by" validate:"omitempty,oneof=generated_at execution_time"`
	SortOrder string                 `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// AnalyticsResultListResponse represents the response for listing analytics results
type AnalyticsResultListResponse struct {
	Results []AnalyticsResult `json:"results"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}

// Validation methods

// Validate validates the AnalyticsQuery struct
func (aq *AnalyticsQuery) Validate() error {
	validate := validator.New()
	return validate.Struct(aq)
}

// Validate validates the AnalyticsFilter struct
func (af *AnalyticsFilter) Validate() error {
	validate := validator.New()
	return validate.Struct(af)
}

// Validate validates the AnalyticsAggregation struct
func (aa *AnalyticsAggregation) Validate() error {
	validate := validator.New()
	return validate.Struct(aa)
}

// Validate validates the AnalyticsResult struct
func (ar *AnalyticsResult) Validate() error {
	validate := validator.New()
	return validate.Struct(ar)
}

// Validate validates the AnalyticsDashboard struct
func (ad *AnalyticsDashboard) Validate() error {
	validate := validator.New()
	return validate.Struct(ad)
}

// Validate validates the DashboardWidget struct
func (dw *DashboardWidget) Validate() error {
	validate := validator.New()
	return validate.Struct(dw)
}

// Validate validates the AnalyticsReport struct
func (ar *AnalyticsReport) Validate() error {
	validate := validator.New()
	return validate.Struct(ar)
}

// Validate validates the AnalyticsQueryCreateRequest struct
func (aqcr *AnalyticsQueryCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(aqcr)
}

// Validate validates the AnalyticsQueryListRequest struct
func (aqlr *AnalyticsQueryListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(aqlr)
}

// Validate validates the AnalyticsQueryExecuteRequest struct
func (aqer *AnalyticsQueryExecuteRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(aqer)
}

// Validate validates the AnalyticsResultListRequest struct
func (arlr *AnalyticsResultListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(arlr)
}

// Helper methods

// IsScheduled returns true if the report is scheduled
func (ar *AnalyticsReport) IsScheduled() bool {
	return ar.Schedule != nil
}

// GetNextGenerationTime returns the next generation time for a scheduled report
func (ar *AnalyticsReport) GetNextGenerationTime() *time.Time {
	if ar.Schedule == nil {
		return nil
	}
	return ar.NextGeneration
}

// UpdateLastGenerated updates the last generated time
func (ar *AnalyticsReport) UpdateLastGenerated() {
	now := time.Now()
	ar.LastGenerated = &now
	ar.UpdatedAt = now
}

// IsQueryPublic returns true if the query is public
func (aq *AnalyticsQuery) IsQueryPublic() bool {
	return aq.IsPublic
}

// GetExecutionTime returns the execution time of the result
func (ar *AnalyticsResult) GetExecutionTime() time.Duration {
	return ar.ExecutionTime
}

// IsCompleted returns true if the result is completed
func (ar *AnalyticsResult) IsCompleted() bool {
	return ar.Status == AnalyticsResultStatusCompleted
}

// IsFailed returns true if the result failed
func (ar *AnalyticsResult) IsFailed() bool {
	return ar.Status == AnalyticsResultStatusFailed
}

// HasError returns true if the result has an error
func (ar *AnalyticsResult) HasError() bool {
	return ar.Error != nil
}

// GetError returns the error message
func (ar *AnalyticsResult) GetError() string {
	if ar.Error == nil {
		return ""
	}
	return *ar.Error
}

// SetError sets the error message
func (ar *AnalyticsResult) SetError(err error) {
	if err != nil {
		errStr := err.Error()
		ar.Error = &errStr
		ar.Status = AnalyticsResultStatusFailed
	}
}
