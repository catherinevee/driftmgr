package bi

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BIService provides a unified interface for business intelligence operations
type BIService struct {
	engine       *BIEngine
	reportEngine *ReportEngine
	exportEngine *ExportEngine
	mu           sync.RWMutex
	config       *BIServiceConfig
}

// BIServiceConfig represents configuration for the BI service
type BIServiceConfig struct {
	DashboardEnabled    bool          `json:"dashboard_enabled"`
	ReportEnabled       bool          `json:"report_enabled"`
	ExportEnabled       bool          `json:"export_enabled"`
	AutoRefresh         bool          `json:"auto_refresh"`
	RefreshInterval     time.Duration `json:"refresh_interval"`
	NotificationEnabled bool          `json:"notification_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
}

// NewBIService creates a new BI service
func NewBIService() *BIService {
	config := &BIServiceConfig{
		DashboardEnabled:    true,
		ReportEnabled:       true,
		ExportEnabled:       true,
		AutoRefresh:         true,
		RefreshInterval:     5 * time.Minute,
		NotificationEnabled: true,
		AuditLogging:        true,
	}

	// Create components
	engine := NewBIEngine()
	reportEngine := NewReportEngine()
	exportEngine := NewExportEngine()

	return &BIService{
		engine:       engine,
		reportEngine: reportEngine,
		exportEngine: exportEngine,
		config:       config,
	}
}

// GetBIEngine returns the BI engine
func (s *BIService) GetBIEngine() *BIEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.engine
}

// GetReportEngine returns the report engine
func (s *BIService) GetReportEngine() *ReportEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reportEngine
}

// GetExportEngine returns the export engine
func (s *BIService) GetExportEngine() *ExportEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exportEngine
}

// Start starts the BI service
func (bis *BIService) Start(ctx context.Context) error {
	// Create default dashboards, reports, and datasets
	if err := bis.createDefaultBIComponents(ctx); err != nil {
		return fmt.Errorf("failed to create default BI components: %w", err)
	}

	// Start background refresh if enabled
	if bis.config.AutoRefresh {
		go bis.backgroundRefresh(ctx)
	}

	return nil
}

// Stop stops the BI service
func (bis *BIService) Stop(ctx context.Context) error {
	// Stop background processes
	return nil
}

// CreateDashboard creates a new dashboard
func (bis *BIService) CreateDashboard(ctx context.Context, dashboard *Dashboard) error {
	return bis.engine.CreateDashboard(ctx, dashboard)
}

// CreateReport creates a new report
func (bis *BIService) CreateReport(ctx context.Context, report *Report) error {
	return bis.engine.CreateReport(ctx, report)
}

// CreateDataset creates a new dataset
func (bis *BIService) CreateDataset(ctx context.Context, dataset *Dataset) error {
	return bis.engine.CreateDataset(ctx, dataset)
}

// CreateQuery creates a new query
func (bis *BIService) CreateQuery(ctx context.Context, query *Query) error {
	return bis.engine.CreateQuery(ctx, query)
}

// ExecuteQuery executes a query
func (bis *BIService) ExecuteQuery(ctx context.Context, queryID string, parameters map[string]interface{}) (*QueryResult, error) {
	return bis.engine.ExecuteQuery(ctx, queryID, parameters)
}

// GenerateReport generates a report
func (bis *BIService) GenerateReport(ctx context.Context, reportID string, parameters map[string]interface{}) (*ReportResult, error) {
	return bis.reportEngine.GenerateReport(ctx, reportID, parameters)
}

// ExportData exports data in various formats
func (bis *BIService) ExportData(ctx context.Context, queryID string, format string, parameters map[string]interface{}) (*ExportResult, error) {
	return bis.exportEngine.ExportData(ctx, queryID, format, parameters)
}

// GetBIStatus returns the overall BI status
func (bis *BIService) GetBIStatus(ctx context.Context) (*BIStatus, error) {
	status := &BIStatus{
		OverallStatus: "Unknown",
		Dashboards:    make(map[string]int),
		Reports:       make(map[string]int),
		Datasets:      make(map[string]int),
		Queries:       make(map[string]int),
		LastRefresh:   time.Time{},
		Metadata:      make(map[string]interface{}),
	}

	// Get dashboard counts
	dashboards, err := bis.engine.ListDashboards(ctx)
	if err == nil {
		for _, dashboard := range dashboards {
			status.Dashboards[dashboard.Category]++
		}
	}

	// Get report counts
	reports, err := bis.engine.ListReports(ctx)
	if err == nil {
		for _, report := range reports {
			status.Reports[report.Category]++
		}
	}

	// Get dataset counts
	datasets, err := bis.engine.ListDatasets(ctx)
	if err == nil {
		for _, dataset := range datasets {
			status.Datasets[dataset.Type]++
		}
	}

	// Get query counts
	queries, err := bis.engine.ListQueries(ctx)
	if err == nil {
		for range queries {
			status.Queries["total"]++
		}
	}

	// Determine overall status
	totalComponents := len(dashboards) + len(reports) + len(datasets) + len(queries)
	if totalComponents > 0 {
		status.OverallStatus = "Active"
	} else {
		status.OverallStatus = "Inactive"
	}

	return status, nil
}

// BIStatus represents the overall BI status
type BIStatus struct {
	OverallStatus string                 `json:"overall_status"`
	Dashboards    map[string]int         `json:"dashboards"`
	Reports       map[string]int         `json:"reports"`
	Datasets      map[string]int         `json:"datasets"`
	Queries       map[string]int         `json:"queries"`
	LastRefresh   time.Time              `json:"last_refresh"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// createDefaultBIComponents creates default BI components
func (bis *BIService) createDefaultBIComponents(ctx context.Context) error {
	// Create default dashboards
	dashboards := []*Dashboard{
		{
			Name:        "Infrastructure Overview",
			Description: "Overview of infrastructure health and performance",
			Category:    "infrastructure",
			Widgets: []Widget{
				{
					ID:    "widget_1",
					Type:  "metric",
					Title: "Total Resources",
					Query: "SELECT COUNT(*) FROM resources",
					Config: map[string]interface{}{
						"format": "number",
						"color":  "blue",
					},
					Position: WidgetPosition{X: 0, Y: 0},
					Size:     WidgetSize{Width: 2, Height: 1},
				},
				{
					ID:    "widget_2",
					Type:  "chart",
					Title: "Resource Distribution",
					Query: "SELECT provider, COUNT(*) FROM resources GROUP BY provider",
					Config: map[string]interface{}{
						"chart_type": "pie",
						"colors":     []string{"#FF6384", "#36A2EB", "#FFCE56"},
					},
					Position: WidgetPosition{X: 2, Y: 0},
					Size:     WidgetSize{Width: 2, Height: 2},
				},
			},
			Layout: DashboardLayout{
				Columns: 4,
				Rows:    3,
				Theme:   "light",
			},
			Filters: []Filter{
				{
					ID:    "filter_1",
					Name:  "Date Range",
					Type:  "date",
					Field: "created_at",
				},
				{
					ID:    "filter_2",
					Name:  "Provider",
					Type:  "select",
					Field: "provider",
					Options: []FilterOption{
						{Label: "AWS", Value: "aws"},
						{Label: "Azure", Value: "azure"},
						{Label: "GCP", Value: "gcp"},
					},
				},
			},
			RefreshRate: 5 * time.Minute,
			Public:      true,
		},
		{
			Name:        "Cost Analysis",
			Description: "Cost analysis and optimization insights",
			Category:    "cost",
			Widgets: []Widget{
				{
					ID:    "widget_3",
					Type:  "metric",
					Title: "Total Cost",
					Query: "SELECT SUM(cost) FROM cost_data",
					Config: map[string]interface{}{
						"format": "currency",
						"color":  "green",
					},
					Position: WidgetPosition{X: 0, Y: 0},
					Size:     WidgetSize{Width: 2, Height: 1},
				},
				{
					ID:    "widget_4",
					Type:  "chart",
					Title: "Cost Trend",
					Query: "SELECT date, cost FROM cost_data ORDER BY date",
					Config: map[string]interface{}{
						"chart_type": "line",
						"color":      "#36A2EB",
					},
					Position: WidgetPosition{X: 0, Y: 1},
					Size:     WidgetSize{Width: 4, Height: 2},
				},
			},
			Layout: DashboardLayout{
				Columns: 4,
				Rows:    3,
				Theme:   "light",
			},
			RefreshRate: 1 * time.Hour,
			Public:      false,
		},
		{
			Name:        "Security Dashboard",
			Description: "Security monitoring and compliance status",
			Category:    "security",
			Widgets: []Widget{
				{
					ID:    "widget_5",
					Type:  "metric",
					Title: "Security Score",
					Query: "SELECT AVG(security_score) FROM security_metrics",
					Config: map[string]interface{}{
						"format": "percentage",
						"color":  "orange",
					},
					Position: WidgetPosition{X: 0, Y: 0},
					Size:     WidgetSize{Width: 2, Height: 1},
				},
				{
					ID:    "widget_6",
					Type:  "table",
					Title: "Security Violations",
					Query: "SELECT resource_id, violation_type, severity FROM security_violations ORDER BY severity DESC",
					Config: map[string]interface{}{
						"page_size": 10,
						"sortable":  true,
					},
					Position: WidgetPosition{X: 0, Y: 1},
					Size:     WidgetSize{Width: 4, Height: 2},
				},
			},
			Layout: DashboardLayout{
				Columns: 4,
				Rows:    3,
				Theme:   "dark",
			},
			RefreshRate: 1 * time.Minute,
			Public:      false,
		},
	}

	// Create dashboards
	for _, dashboard := range dashboards {
		if err := bis.engine.CreateDashboard(ctx, dashboard); err != nil {
			return fmt.Errorf("failed to create dashboard %s: %w", dashboard.Name, err)
		}
	}

	// Create default reports
	reports := []*Report{
		{
			Name:        "Monthly Infrastructure Report",
			Description: "Monthly infrastructure health and performance report",
			Category:    "infrastructure",
			Type:        "scheduled",
			Query:       "SELECT * FROM infrastructure_metrics WHERE date >= ? AND date <= ?",
			Format:      "pdf",
			Schedule: &ReportSchedule{
				Frequency: "monthly",
				Time:      time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC), // 9 AM
				Timezone:  "UTC",
			},
			Recipients: []string{"admin@company.com", "ops@company.com"},
			Parameters: map[string]interface{}{
				"start_date": "{{start_of_month}}",
				"end_date":   "{{end_of_month}}",
			},
		},
		{
			Name:        "Cost Optimization Report",
			Description: "Weekly cost optimization recommendations",
			Category:    "cost",
			Type:        "scheduled",
			Query:       "SELECT * FROM cost_optimization WHERE date >= ? AND date <= ?",
			Format:      "excel",
			Schedule: &ReportSchedule{
				Frequency: "weekly",
				Time:      time.Date(0, 0, 0, 8, 0, 0, 0, time.UTC), // 8 AM
				Timezone:  "UTC",
			},
			Recipients: []string{"finance@company.com"},
			Parameters: map[string]interface{}{
				"start_date": "{{start_of_week}}",
				"end_date":   "{{end_of_week}}",
			},
		},
		{
			Name:        "Security Compliance Report",
			Description: "Security compliance status report",
			Category:    "security",
			Type:        "on-demand",
			Query:       "SELECT * FROM security_compliance WHERE date = ?",
			Format:      "pdf",
			Recipients:  []string{"security@company.com"},
			Parameters: map[string]interface{}{
				"date": "{{current_date}}",
			},
		},
	}

	// Create reports
	for _, report := range reports {
		if err := bis.engine.CreateReport(ctx, report); err != nil {
			return fmt.Errorf("failed to create report %s: %w", report.Name, err)
		}
	}

	// Create default datasets
	datasets := []*Dataset{
		{
			Name:        "Infrastructure Metrics",
			Description: "Infrastructure performance and health metrics",
			Source:      "infrastructure_metrics",
			Type:        "table",
			Schema: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "id", "type": "string"},
					{"name": "resource_id", "type": "string"},
					{"name": "metric_name", "type": "string"},
					{"name": "value", "type": "float"},
					{"name": "timestamp", "type": "datetime"},
				},
			},
			RefreshRate: 1 * time.Minute,
		},
		{
			Name:        "Cost Data",
			Description: "Cost and billing data",
			Source:      "cost_data",
			Type:        "table",
			Schema: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "id", "type": "string"},
					{"name": "resource_id", "type": "string"},
					{"name": "cost", "type": "float"},
					{"name": "currency", "type": "string"},
					{"name": "date", "type": "date"},
				},
			},
			RefreshRate: 1 * time.Hour,
		},
		{
			Name:        "Security Metrics",
			Description: "Security and compliance metrics",
			Source:      "security_metrics",
			Type:        "table",
			Schema: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "id", "type": "string"},
					{"name": "resource_id", "type": "string"},
					{"name": "security_score", "type": "float"},
					{"name": "compliance_status", "type": "string"},
					{"name": "timestamp", "type": "datetime"},
				},
			},
			RefreshRate: 5 * time.Minute,
		},
	}

	// Create datasets
	for _, dataset := range datasets {
		if err := bis.engine.CreateDataset(ctx, dataset); err != nil {
			return fmt.Errorf("failed to create dataset %s: %w", dataset.Name, err)
		}
	}

	// Create default queries
	queries := []*Query{
		{
			Name:        "Resource Count by Provider",
			Description: "Count of resources grouped by cloud provider",
			SQL:         "SELECT provider, COUNT(*) as count FROM resources GROUP BY provider",
			Parameters:  []QueryParameter{},
			Cache:       true,
			CacheTTL:    1 * time.Hour,
		},
		{
			Name:        "Top Cost Resources",
			Description: "Top 10 most expensive resources",
			SQL:         "SELECT resource_id, cost FROM cost_data ORDER BY cost DESC LIMIT 10",
			Parameters:  []QueryParameter{},
			Cache:       true,
			CacheTTL:    30 * time.Minute,
		},
		{
			Name:        "Security Violations by Severity",
			Description: "Count of security violations by severity level",
			SQL:         "SELECT severity, COUNT(*) as count FROM security_violations GROUP BY severity",
			Parameters:  []QueryParameter{},
			Cache:       true,
			CacheTTL:    15 * time.Minute,
		},
		{
			Name:        "Resource Metrics by Date Range",
			Description: "Resource metrics filtered by date range",
			SQL:         "SELECT * FROM infrastructure_metrics WHERE timestamp BETWEEN ? AND ?",
			Parameters: []QueryParameter{
				{Name: "start_date", Type: "datetime", Required: true, Description: "Start date for the query"},
				{Name: "end_date", Type: "datetime", Required: true, Description: "End date for the query"},
			},
			Cache:    true,
			CacheTTL: 1 * time.Hour,
		},
	}

	// Create queries
	for _, query := range queries {
		if err := bis.engine.CreateQuery(ctx, query); err != nil {
			return fmt.Errorf("failed to create query %s: %w", query.Name, err)
		}
	}

	return nil
}

// backgroundRefresh runs background refresh
func (bis *BIService) backgroundRefresh(ctx context.Context) {
	ticker := time.NewTicker(bis.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bis.performBackgroundRefresh(ctx)
		}
	}
}

// performBackgroundRefresh performs background refresh
func (bis *BIService) performBackgroundRefresh(ctx context.Context) {
	// This would refresh datasets and update dashboards
	// For now, it's a placeholder
	fmt.Println("Performing background BI refresh...")
}

// SetConfig updates the BI service configuration
func (bis *BIService) SetConfig(config *BIServiceConfig) {
	bis.mu.Lock()
	defer bis.mu.Unlock()
	bis.config = config
}

// GetConfig returns the current BI service configuration
func (bis *BIService) GetConfig() *BIServiceConfig {
	bis.mu.RLock()
	defer bis.mu.RUnlock()
	return bis.config
}
