package widgets

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
)

// Manager manages dashboard widgets
type Manager struct {
	widgetRepo    WidgetRepository
	dashboardRepo DashboardRepository
	config        ManagerConfig
}

// WidgetRepository defines the interface for widget persistence
type WidgetRepository interface {
	CreateWidget(ctx context.Context, widget *models.DashboardWidget) error
	GetWidget(ctx context.Context, id uuid.UUID) (*models.DashboardWidget, error)
	UpdateWidget(ctx context.Context, widget *models.DashboardWidget) error
	DeleteWidget(ctx context.Context, id uuid.UUID) error
	ListWidgets(ctx context.Context, filter WidgetFilter) ([]*models.DashboardWidget, error)
	GetWidgetStats(ctx context.Context, id uuid.UUID) (*WidgetStats, error)
}

// DashboardRepository defines the interface for dashboard persistence
type DashboardRepository interface {
	CreateDashboard(ctx context.Context, dashboard *models.Dashboard) error
	GetDashboard(ctx context.Context, id uuid.UUID) (*models.Dashboard, error)
	UpdateDashboard(ctx context.Context, dashboard *models.Dashboard) error
	DeleteDashboard(ctx context.Context, id uuid.UUID) error
	ListDashboards(ctx context.Context, filter DashboardFilter) ([]*models.Dashboard, error)
	GetDashboardStats(ctx context.Context, id uuid.UUID) (*DashboardStats, error)
}

// ManagerConfig holds configuration for the widget manager
type ManagerConfig struct {
	MaxWidgetsPerDashboard int           `json:"max_widgets_per_dashboard"`
	MaxDashboardsPerUser   int           `json:"max_dashboards_per_user"`
	WidgetTimeout          time.Duration `json:"widget_timeout"`
	EnableCaching          bool          `json:"enable_caching"`
	CacheTTL               time.Duration `json:"cache_ttl"`
	EnableEventLogging     bool          `json:"enable_event_logging"`
	EnableMetrics          bool          `json:"enable_metrics"`
	EnableAuditLogging     bool          `json:"enable_audit_logging"`
}

// WidgetFilter defines filters for widget queries
type WidgetFilter struct {
	DashboardID *uuid.UUID           `json:"dashboard_id,omitempty"`
	UserID      *uuid.UUID           `json:"user_id,omitempty"`
	Type        *models.WidgetType   `json:"type,omitempty"`
	Status      *models.WidgetStatus `json:"status,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Search      string               `json:"search,omitempty"`
	Limit       int                  `json:"limit,omitempty"`
	Offset      int                  `json:"offset,omitempty"`
}

// DashboardFilter defines filters for dashboard queries
type DashboardFilter struct {
	UserID *uuid.UUID              `json:"user_id,omitempty"`
	Status *models.DashboardStatus `json:"status,omitempty"`
	Tags   []string                `json:"tags,omitempty"`
	Search string                  `json:"search,omitempty"`
	Limit  int                     `json:"limit,omitempty"`
	Offset int                     `json:"offset,omitempty"`
}

// WidgetStats represents statistics for a widget
type WidgetStats struct {
	WidgetID        uuid.UUID     `json:"widget_id"`
	TotalViews      int           `json:"total_views"`
	LastView        *time.Time    `json:"last_view"`
	AverageLoadTime time.Duration `json:"average_load_time"`
	ErrorCount      int           `json:"error_count"`
	SuccessRate     float64       `json:"success_rate"`
}

// DashboardStats represents statistics for a dashboard
type DashboardStats struct {
	DashboardID     uuid.UUID     `json:"dashboard_id"`
	TotalViews      int           `json:"total_views"`
	LastView        *time.Time    `json:"last_view"`
	WidgetCount     int           `json:"widget_count"`
	AverageLoadTime time.Duration `json:"average_load_time"`
	ErrorCount      int           `json:"error_count"`
	SuccessRate     float64       `json:"success_rate"`
}

// NewManager creates a new widget manager
func NewManager(
	widgetRepo WidgetRepository,
	dashboardRepo DashboardRepository,
	config ManagerConfig,
) *Manager {
	return &Manager{
		widgetRepo:    widgetRepo,
		dashboardRepo: dashboardRepo,
		config:        config,
	}
}

// CreateWidget creates a new dashboard widget
func (m *Manager) CreateWidget(ctx context.Context, userID uuid.UUID, req *models.DashboardWidgetRequest) (*models.DashboardWidget, error) {
	// Check widget limit
	if err := m.checkWidgetLimit(ctx, req.DashboardID); err != nil {
		return nil, fmt.Errorf("widget limit exceeded: %w", err)
	}

	// Validate the widget
	if err := m.validateWidget(req); err != nil {
		return nil, fmt.Errorf("widget validation failed: %w", err)
	}

	// Create the widget
	widget := &models.DashboardWidget{
		ID:          uuid.New(),
		DashboardID: req.DashboardID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		Position:    req.Position,
		Size:        req.Size,
		Status:      models.WidgetStatusActive,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the widget
	if err := m.widgetRepo.CreateWidget(ctx, widget); err != nil {
		return nil, fmt.Errorf("failed to create widget: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "widget_created", userID, widget.ID, map[string]interface{}{
			"widget_name":  widget.Name,
			"widget_type":  widget.Type,
			"dashboard_id": widget.DashboardID,
		})
	}

	return widget, nil
}

// GetWidget retrieves a widget by ID
func (m *Manager) GetWidget(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.DashboardWidget, error) {
	widget, err := m.widgetRepo.GetWidget(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get widget: %w", err)
	}

	// Check ownership
	if widget.UserID != userID {
		return nil, fmt.Errorf("widget not found or access denied")
	}

	return widget, nil
}

// UpdateWidget updates an existing widget
func (m *Manager) UpdateWidget(ctx context.Context, userID uuid.UUID, id uuid.UUID, req *models.DashboardWidgetRequest) (*models.DashboardWidget, error) {
	// Get existing widget
	widget, err := m.widgetRepo.GetWidget(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get widget: %w", err)
	}

	// Check ownership
	if widget.UserID != userID {
		return nil, fmt.Errorf("widget not found or access denied")
	}

	// Validate the widget
	if err := m.validateWidget(req); err != nil {
		return nil, fmt.Errorf("widget validation failed: %w", err)
	}

	// Update widget fields
	widget.Name = req.Name
	widget.Description = req.Description
	widget.Type = req.Type
	widget.Config = req.Config
	widget.Position = req.Position
	widget.Size = req.Size
	widget.Tags = req.Tags
	widget.UpdatedAt = time.Now()

	// Save the updated widget
	if err := m.widgetRepo.UpdateWidget(ctx, widget); err != nil {
		return nil, fmt.Errorf("failed to update widget: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "widget_updated", userID, widget.ID, map[string]interface{}{
			"widget_name": widget.Name,
			"widget_type": widget.Type,
		})
	}

	return widget, nil
}

// DeleteWidget deletes a widget
func (m *Manager) DeleteWidget(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	// Get widget to check ownership
	widget, err := m.widgetRepo.GetWidget(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get widget: %w", err)
	}

	// Check ownership
	if widget.UserID != userID {
		return fmt.Errorf("widget not found or access denied")
	}

	// Delete the widget
	if err := m.widgetRepo.DeleteWidget(ctx, id); err != nil {
		return fmt.Errorf("failed to delete widget: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "widget_deleted", userID, widget.ID, map[string]interface{}{
			"widget_name": widget.Name,
		})
	}

	return nil
}

// ListWidgets lists widgets with optional filtering
func (m *Manager) ListWidgets(ctx context.Context, userID uuid.UUID, filter WidgetFilter) ([]*models.DashboardWidget, error) {
	// Set user ID filter
	filter.UserID = &userID

	widgets, err := m.widgetRepo.ListWidgets(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list widgets: %w", err)
	}
	return widgets, nil
}

// GetWidgetData retrieves data for a widget
func (m *Manager) GetWidgetData(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.WidgetData, error) {
	// Get widget
	widget, err := m.widgetRepo.GetWidget(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get widget: %w", err)
	}

	// Check ownership
	if widget.UserID != userID {
		return nil, fmt.Errorf("widget not found or access denied")
	}

	// Generate widget data based on type
	data, err := m.generateWidgetData(ctx, widget)
	if err != nil {
		return nil, fmt.Errorf("failed to generate widget data: %w", err)
	}

	return data, nil
}

// CreateDashboard creates a new dashboard
func (m *Manager) CreateDashboard(ctx context.Context, userID uuid.UUID, req *models.DashboardRequest) (*models.Dashboard, error) {
	// Check dashboard limit
	if err := m.checkDashboardLimit(ctx, userID); err != nil {
		return nil, fmt.Errorf("dashboard limit exceeded: %w", err)
	}

	// Validate the dashboard
	if err := m.validateDashboard(req); err != nil {
		return nil, fmt.Errorf("dashboard validation failed: %w", err)
	}

	// Create the dashboard
	dashboard := &models.Dashboard{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Layout:      req.Layout,
		Settings:    req.Settings,
		Status:      models.DashboardStatusActive,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the dashboard
	if err := m.dashboardRepo.CreateDashboard(ctx, dashboard); err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "dashboard_created", userID, dashboard.ID, map[string]interface{}{
			"dashboard_name": dashboard.Name,
		})
	}

	return dashboard, nil
}

// GetDashboard retrieves a dashboard by ID
func (m *Manager) GetDashboard(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.Dashboard, error) {
	dashboard, err := m.dashboardRepo.GetDashboard(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Check ownership
	if dashboard.UserID != userID {
		return nil, fmt.Errorf("dashboard not found or access denied")
	}

	return dashboard, nil
}

// UpdateDashboard updates an existing dashboard
func (m *Manager) UpdateDashboard(ctx context.Context, userID uuid.UUID, id uuid.UUID, req *models.DashboardRequest) (*models.Dashboard, error) {
	// Get existing dashboard
	dashboard, err := m.dashboardRepo.GetDashboard(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Check ownership
	if dashboard.UserID != userID {
		return nil, fmt.Errorf("dashboard not found or access denied")
	}

	// Validate the dashboard
	if err := m.validateDashboard(req); err != nil {
		return nil, fmt.Errorf("dashboard validation failed: %w", err)
	}

	// Update dashboard fields
	dashboard.Name = req.Name
	dashboard.Description = req.Description
	dashboard.Layout = req.Layout
	dashboard.Settings = req.Settings
	dashboard.Tags = req.Tags
	dashboard.UpdatedAt = time.Now()

	// Save the updated dashboard
	if err := m.dashboardRepo.UpdateDashboard(ctx, dashboard); err != nil {
		return nil, fmt.Errorf("failed to update dashboard: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "dashboard_updated", userID, dashboard.ID, map[string]interface{}{
			"dashboard_name": dashboard.Name,
		})
	}

	return dashboard, nil
}

// DeleteDashboard deletes a dashboard
func (m *Manager) DeleteDashboard(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	// Get dashboard to check ownership
	dashboard, err := m.dashboardRepo.GetDashboard(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Check ownership
	if dashboard.UserID != userID {
		return fmt.Errorf("dashboard not found or access denied")
	}

	// Delete the dashboard
	if err := m.dashboardRepo.DeleteDashboard(ctx, id); err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "dashboard_deleted", userID, dashboard.ID, map[string]interface{}{
			"dashboard_name": dashboard.Name,
		})
	}

	return nil
}

// ListDashboards lists dashboards with optional filtering
func (m *Manager) ListDashboards(ctx context.Context, userID uuid.UUID, filter DashboardFilter) ([]*models.Dashboard, error) {
	// Set user ID filter
	filter.UserID = &userID

	dashboards, err := m.dashboardRepo.ListDashboards(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list dashboards: %w", err)
	}
	return dashboards, nil
}

// GetDashboardData retrieves data for a dashboard
func (m *Manager) GetDashboardData(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.DashboardData, error) {
	// Get dashboard
	dashboard, err := m.dashboardRepo.GetDashboard(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Check ownership
	if dashboard.UserID != userID {
		return nil, fmt.Errorf("dashboard not found or access denied")
	}

	// Get dashboard widgets
	widgetFilter := WidgetFilter{
		DashboardID: &id,
		UserID:      &userID,
	}
	widgets, err := m.widgetRepo.ListWidgets(ctx, widgetFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard widgets: %w", err)
	}

	// Generate dashboard data
	data := &models.DashboardData{
		Dashboard:   dashboard,
		Widgets:     widgets,
		GeneratedAt: time.Now(),
	}

	return data, nil
}

// checkWidgetLimit checks if the dashboard has reached the widget limit
func (m *Manager) checkWidgetLimit(ctx context.Context, dashboardID uuid.UUID) error {
	if m.config.MaxWidgetsPerDashboard <= 0 {
		return nil // No limit
	}

	filter := WidgetFilter{
		DashboardID: &dashboardID,
		Limit:       1,
	}

	widgets, err := m.widgetRepo.ListWidgets(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check widget limit: %w", err)
	}

	if len(widgets) >= m.config.MaxWidgetsPerDashboard {
		return fmt.Errorf("widget limit exceeded: %d widgets", m.config.MaxWidgetsPerDashboard)
	}

	return nil
}

// checkDashboardLimit checks if the user has reached the dashboard limit
func (m *Manager) checkDashboardLimit(ctx context.Context, userID uuid.UUID) error {
	if m.config.MaxDashboardsPerUser <= 0 {
		return nil // No limit
	}

	filter := DashboardFilter{
		UserID: &userID,
		Limit:  1,
	}

	dashboards, err := m.dashboardRepo.ListDashboards(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check dashboard limit: %w", err)
	}

	if len(dashboards) >= m.config.MaxDashboardsPerUser {
		return fmt.Errorf("dashboard limit exceeded: %d dashboards", m.config.MaxDashboardsPerUser)
	}

	return nil
}

// validateWidget validates a widget request
func (m *Manager) validateWidget(req *models.DashboardWidgetRequest) error {
	if req.Name == "" {
		return fmt.Errorf("widget name is required")
	}

	if req.Type == "" {
		return fmt.Errorf("widget type is required")
	}

	if req.DashboardID == uuid.Nil {
		return fmt.Errorf("dashboard ID is required")
	}

	return nil
}

// validateDashboard validates a dashboard request
func (m *Manager) validateDashboard(req *models.DashboardRequest) error {
	if req.Name == "" {
		return fmt.Errorf("dashboard name is required")
	}

	return nil
}

// generateWidgetData generates data for a widget
func (m *Manager) generateWidgetData(ctx context.Context, widget *models.DashboardWidget) (*models.WidgetData, error) {
	// Create widget data based on type
	data := &models.WidgetData{
		WidgetID:    widget.ID,
		Type:        widget.Type,
		GeneratedAt: time.Now(),
		Data:        make(map[string]interface{}),
	}

	// Generate data based on widget type
	switch widget.Type {
	case models.WidgetTypeChart:
		data.Data = m.generateChartData(widget)
	case models.WidgetTypeMetric:
		data.Data = m.generateMetricData(widget)
	case models.WidgetTypeTable:
		data.Data = m.generateTableData(widget)
	case models.WidgetTypeGauge:
		data.Data = m.generateGaugeData(widget)
	case models.WidgetTypeMap:
		data.Data = m.generateMapData(widget)
	case models.WidgetTypeText:
		data.Data = m.generateTextData(widget)
	case models.WidgetTypeImage:
		data.Data = m.generateImageData(widget)
	case models.WidgetTypeVideo:
		data.Data = m.generateVideoData(widget)
	default:
		return nil, fmt.Errorf("unsupported widget type: %s", widget.Type)
	}

	return data, nil
}

// generateChartData generates data for a chart widget
func (m *Manager) generateChartData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseChartConfig(widget.Config)

	// Generate sample chart data
	return map[string]interface{}{
		"type": "line",
		"data": map[string]interface{}{
			"labels": []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
			"datasets": []map[string]interface{}{
				{
					"label":       "Sample Data",
					"data":        []float64{12, 19, 3, 5, 2, 3},
					"borderColor": "rgb(75, 192, 192)",
					"tension":     0.1,
				},
			},
		},
		"options": map[string]interface{}{
			"responsive": true,
			"plugins": map[string]interface{}{
				"title": map[string]interface{}{
					"display": true,
					"text":    config.Title,
				},
			},
		},
	}
}

// generateMetricData generates data for a metric widget
func (m *Manager) generateMetricData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseMetricConfig(widget.Config)

	// Generate sample metric data
	return map[string]interface{}{
		"value": 42.5,
		"unit":  config.Unit,
		"trend": map[string]interface{}{
			"direction":  "up",
			"percentage": 12.5,
		},
		"status": "good",
	}
}

// generateTableData generates data for a table widget
func (m *Manager) generateWidgetData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseTableConfig(widget.Config)

	// Generate sample table data
	return map[string]interface{}{
		"columns": []string{"Name", "Value", "Status"},
		"rows": [][]interface{}{
			{"Item 1", 100, "Active"},
			{"Item 2", 200, "Inactive"},
			{"Item 3", 150, "Active"},
		},
		"pagination": map[string]interface{}{
			"page":  1,
			"size":  10,
			"total": 3,
		},
	}
}

// generateGaugeData generates data for a gauge widget
func (m *Manager) generateGaugeData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseGaugeConfig(widget.Config)

	// Generate sample gauge data
	return map[string]interface{}{
		"value":  75.0,
		"min":    config.Min,
		"max":    config.Max,
		"unit":   config.Unit,
		"status": "warning",
	}
}

// generateMapData generates data for a map widget
func (m *Manager) generateMapData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseMapConfig(widget.Config)

	// Generate sample map data
	return map[string]interface{}{
		"center": map[string]interface{}{
			"lat": config.CenterLat,
			"lng": config.CenterLng,
		},
		"zoom": config.Zoom,
		"markers": []map[string]interface{}{
			{
				"lat":   40.7128,
				"lng":   -74.0060,
				"title": "New York",
				"value": 100,
			},
			{
				"lat":   34.0522,
				"lng":   -118.2437,
				"title": "Los Angeles",
				"value": 200,
			},
		},
	}
}

// generateTextData generates data for a text widget
func (m *Manager) generateTextData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseTextConfig(widget.Config)

	// Generate sample text data
	return map[string]interface{}{
		"content": config.Content,
		"format":  config.Format,
		"style":   config.Style,
	}
}

// generateImageData generates data for an image widget
func (m *Manager) generateImageData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseImageConfig(widget.Config)

	// Generate sample image data
	return map[string]interface{}{
		"url":    config.URL,
		"alt":    config.Alt,
		"width":  config.Width,
		"height": config.Height,
	}
}

// generateVideoData generates data for a video widget
func (m *Manager) generateVideoData(widget *models.DashboardWidget) map[string]interface{} {
	// Parse widget configuration
	config, _ := m.parseVideoConfig(widget.Config)

	// Generate sample video data
	return map[string]interface{}{
		"url":      config.URL,
		"poster":   config.Poster,
		"autoplay": config.Autoplay,
		"controls": config.Controls,
	}
}

// Configuration parsing structures
type ChartConfig struct {
	Title string `json:"title"`
	Type  string `json:"type"`
}

type MetricConfig struct {
	Unit string `json:"unit"`
}

type TableConfig struct {
	Columns []string `json:"columns"`
}

type GaugeConfig struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Unit string  `json:"unit"`
}

type MapConfig struct {
	CenterLat float64 `json:"center_lat"`
	CenterLng float64 `json:"center_lng"`
	Zoom      int     `json:"zoom"`
}

type TextConfig struct {
	Content string `json:"content"`
	Format  string `json:"format"`
	Style   string `json:"style"`
}

type ImageConfig struct {
	URL    string `json:"url"`
	Alt    string `json:"alt"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type VideoConfig struct {
	URL      string `json:"url"`
	Poster   string `json:"poster"`
	Autoplay bool   `json:"autoplay"`
	Controls bool   `json:"controls"`
}

// parseChartConfig parses chart configuration from JSONB
func (m *Manager) parseChartConfig(config models.JSONB) (*ChartConfig, error) {
	var chartConfig ChartConfig
	if err := config.Unmarshal(&chartConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chart config: %w", err)
	}
	return &chartConfig, nil
}

// parseMetricConfig parses metric configuration from JSONB
func (m *Manager) parseMetricConfig(config models.JSONB) (*MetricConfig, error) {
	var metricConfig MetricConfig
	if err := config.Unmarshal(&metricConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metric config: %w", err)
	}
	return &metricConfig, nil
}

// parseTableConfig parses table configuration from JSONB
func (m *Manager) parseTableConfig(config models.JSONB) (*TableConfig, error) {
	var tableConfig TableConfig
	if err := config.Unmarshal(&tableConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal table config: %w", err)
	}
	return &tableConfig, nil
}

// parseGaugeConfig parses gauge configuration from JSONB
func (m *Manager) parseGaugeConfig(config models.JSONB) (*GaugeConfig, error) {
	var gaugeConfig GaugeConfig
	if err := config.Unmarshal(&gaugeConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gauge config: %w", err)
	}
	return &gaugeConfig, nil
}

// parseMapConfig parses map configuration from JSONB
func (m *Manager) parseMapConfig(config models.JSONB) (*MapConfig, error) {
	var mapConfig MapConfig
	if err := config.Unmarshal(&mapConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal map config: %w", err)
	}
	return &mapConfig, nil
}

// parseTextConfig parses text configuration from JSONB
func (m *Manager) parseTextConfig(config models.JSONB) (*TextConfig, error) {
	var textConfig TextConfig
	if err := config.Unmarshal(&textConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal text config: %w", err)
	}
	return &textConfig, nil
}

// parseImageConfig parses image configuration from JSONB
func (m *Manager) parseImageConfig(config models.JSONB) (*ImageConfig, error) {
	var imageConfig ImageConfig
	if err := config.Unmarshal(&imageConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image config: %w", err)
	}
	return &imageConfig, nil
}

// parseVideoConfig parses video configuration from JSONB
func (m *Manager) parseVideoConfig(config models.JSONB) (*VideoConfig, error) {
	var videoConfig VideoConfig
	if err := config.Unmarshal(&videoConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal video config: %w", err)
	}
	return &videoConfig, nil
}

// logAuditEvent logs an audit event
func (m *Manager) logAuditEvent(ctx context.Context, action string, userID uuid.UUID, resourceID uuid.UUID, data map[string]interface{}) {
	// In a real implementation, this would log to an audit system
	log.Printf("AUDIT: %s by user %s for resource %s: %+v", action, userID, resourceID, data)
}
