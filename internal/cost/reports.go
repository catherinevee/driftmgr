package cost

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// CostReporter generates comprehensive cost reports
type CostReporter struct {
	analyzer     *CostAnalyzer
	optimizer    *CostOptimizer
	forecaster   *CostForecaster
	alertManager *CostAlertManager
}

// CostReport represents a comprehensive cost report
type CostReport struct {
	GeneratedAt      time.Time                    `json:"generated_at"`
	ReportPeriod     ReportPeriod                 `json:"report_period"`
	ExecutiveSummary ExecutiveSummary             `json:"executive_summary"`
	CostBreakdown    CostBreakdown                `json:"cost_breakdown"`
	Trends           []TrendPoint                 `json:"trends"`
	Optimizations    []OptimizationRecommendation `json:"optimizations"`
	Forecasts        *CostForecast                `json:"forecasts,omitempty"`
	Alerts           []*CostAlert                 `json:"alerts"`
	Recommendations  []ReportRecommendation       `json:"recommendations"`
	Metadata         map[string]interface{}       `json:"metadata,omitempty"`
}

// ReportPeriod represents the time period for the report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Duration  string    `json:"duration"`
}

// ExecutiveSummary provides a high-level summary of costs
type ExecutiveSummary struct {
	TotalCost         float64      `json:"total_cost"`
	PreviousPeriod    float64      `json:"previous_period_cost"`
	CostChange        float64      `json:"cost_change"`
	CostChangePercent float64      `json:"cost_change_percent"`
	ResourceCount     int          `json:"resource_count"`
	TopCostDrivers    []CostDriver `json:"top_cost_drivers"`
	KeyInsights       []string     `json:"key_insights"`
	Currency          string       `json:"currency"`
}

// CostDriver represents a cost driver
type CostDriver struct {
	Type          string  `json:"type"`
	Cost          float64 `json:"cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"`
	Trend         string  `json:"trend"`
}

// CostBreakdown provides detailed cost breakdown
type CostBreakdown struct {
	ByType        map[string]TypeBreakdown        `json:"by_type"`
	ByRegion      map[string]RegionBreakdown      `json:"by_region"`
	ByAccount     map[string]AccountBreakdown     `json:"by_account"`
	ByProject     map[string]ProjectBreakdown     `json:"by_project"`
	ByEnvironment map[string]EnvironmentBreakdown `json:"by_environment"`
}

// TypeBreakdown represents cost breakdown by resource type
type TypeBreakdown struct {
	Cost          float64 `json:"cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"`
	AverageCost   float64 `json:"average_cost"`
	Trend         string  `json:"trend"`
}

// RegionBreakdown represents cost breakdown by region
type RegionBreakdown struct {
	Cost          float64 `json:"cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"`
	AverageCost   float64 `json:"average_cost"`
	Trend         string  `json:"trend"`
}

// AccountBreakdown represents cost breakdown by account
type AccountBreakdown struct {
	Cost          float64 `json:"cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"`
	AverageCost   float64 `json:"average_cost"`
	Trend         string  `json:"trend"`
}

// ProjectBreakdown represents cost breakdown by project
type ProjectBreakdown struct {
	Cost          float64 `json:"cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"`
	AverageCost   float64 `json:"average_cost"`
	Trend         string  `json:"trend"`
}

// EnvironmentBreakdown represents cost breakdown by environment
type EnvironmentBreakdown struct {
	Cost          float64 `json:"cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"`
	AverageCost   float64 `json:"average_cost"`
	Trend         string  `json:"trend"`
}

// ReportRecommendation represents a recommendation in the report
type ReportRecommendation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    string                 `json:"priority"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Effort      string                 `json:"effort"`
	Savings     float64                `json:"savings"`
	Timeframe   string                 `json:"timeframe"`
	Actions     []string               `json:"actions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewCostReporter creates a new cost reporter
func NewCostReporter(analyzer *CostAnalyzer, optimizer *CostOptimizer, forecaster *CostForecaster, alertManager *CostAlertManager) *CostReporter {
	return &CostReporter{
		analyzer:     analyzer,
		optimizer:    optimizer,
		forecaster:   forecaster,
		alertManager: alertManager,
	}
}

// GenerateReport generates a comprehensive cost report
func (cr *CostReporter) GenerateReport(ctx context.Context, period ReportPeriod, resources []*models.Resource) (*CostReport, error) {
	report := &CostReport{
		GeneratedAt:  time.Now(),
		ReportPeriod: period,
		Metadata:     make(map[string]interface{}),
	}

	// Generate cost data for the period (simplified for now)
	costData := &CostAnalysis{
		TotalCost:     5000.0,
		ResourceCount: len(resources),
		Currency:      "USD",
		CostByType:    make(map[string]float64),
		CostByRegion:  make(map[string]float64),
	}

	// Generate executive summary
	executiveSummary, err := cr.generateExecutiveSummary(ctx, costData, period)
	if err != nil {
		return nil, fmt.Errorf("failed to generate executive summary: %w", err)
	}
	report.ExecutiveSummary = *executiveSummary

	// Generate cost breakdown
	costBreakdown, err := cr.generateCostBreakdown(ctx, costData, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cost breakdown: %w", err)
	}
	report.CostBreakdown = *costBreakdown

	// Generate trends
	trends, err := cr.generateTrends(ctx, period)
	if err != nil {
		return nil, fmt.Errorf("failed to generate trends: %w", err)
	}
	report.Trends = trends

	// Generate optimization recommendations
	optimizations, err := cr.generateOptimizations(ctx, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to generate optimizations: %w", err)
	}
	report.Optimizations = optimizations

	// Generate forecasts
	forecast, err := cr.generateForecast(ctx, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to generate forecast: %w", err)
	}
	report.Forecasts = forecast

	// Get active alerts
	alerts := cr.alertManager.GetActiveAlerts()
	report.Alerts = alerts

	// Generate recommendations
	recommendations, err := cr.generateRecommendations(ctx, costData, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}
	report.Recommendations = recommendations

	return report, nil
}

// generateExecutiveSummary generates the executive summary
func (cr *CostReporter) generateExecutiveSummary(ctx context.Context, costData *CostAnalysis, period ReportPeriod) (*ExecutiveSummary, error) {
	summary := &ExecutiveSummary{
		TotalCost:      costData.TotalCost,
		ResourceCount:  costData.ResourceCount,
		Currency:       costData.Currency,
		TopCostDrivers: []CostDriver{},
		KeyInsights:    []string{},
	}

	// Calculate previous period cost (simplified)
	summary.PreviousPeriod = costData.TotalCost * 0.95 // Assume 5% growth
	summary.CostChange = summary.TotalCost - summary.PreviousPeriod
	summary.CostChangePercent = (summary.CostChange / summary.PreviousPeriod) * 100

	// Identify top cost drivers
	for resourceType, cost := range costData.CostByType {
		percentage := (cost / costData.TotalCost) * 100
		driver := CostDriver{
			Type:          resourceType,
			Cost:          cost,
			Percentage:    percentage,
			ResourceCount: cr.getResourceCountByType(resourceType),
			Trend:         "stable", // Would calculate actual trend
		}
		summary.TopCostDrivers = append(summary.TopCostDrivers, driver)
	}

	// Sort by cost (highest first)
	sort.Slice(summary.TopCostDrivers, func(i, j int) bool {
		return summary.TopCostDrivers[i].Cost > summary.TopCostDrivers[j].Cost
	})

	// Keep only top 5
	if len(summary.TopCostDrivers) > 5 {
		summary.TopCostDrivers = summary.TopCostDrivers[:5]
	}

	// Generate key insights
	summary.KeyInsights = cr.generateKeyInsights(summary, costData)

	return summary, nil
}

// generateCostBreakdown generates detailed cost breakdown
func (cr *CostReporter) generateCostBreakdown(ctx context.Context, costData *CostAnalysis, resources []*models.Resource) (*CostBreakdown, error) {
	breakdown := &CostBreakdown{
		ByType:        make(map[string]TypeBreakdown),
		ByRegion:      make(map[string]RegionBreakdown),
		ByAccount:     make(map[string]AccountBreakdown),
		ByProject:     make(map[string]ProjectBreakdown),
		ByEnvironment: make(map[string]EnvironmentBreakdown),
	}

	// Breakdown by type
	for resourceType, cost := range costData.CostByType {
		resourceCount := cr.getResourceCountByType(resourceType)
		breakdown.ByType[resourceType] = TypeBreakdown{
			Cost:          cost,
			Percentage:    (cost / costData.TotalCost) * 100,
			ResourceCount: resourceCount,
			AverageCost:   cost / float64(resourceCount),
			Trend:         "stable", // Would calculate actual trend
		}
	}

	// Breakdown by region
	for region, cost := range costData.CostByRegion {
		resourceCount := cr.getResourceCountByRegion(region, resources)
		breakdown.ByRegion[region] = RegionBreakdown{
			Cost:          cost,
			Percentage:    (cost / costData.TotalCost) * 100,
			ResourceCount: resourceCount,
			AverageCost:   cost / float64(resourceCount),
			Trend:         "stable", // Would calculate actual trend
		}
	}

	// Breakdown by account (simplified)
	accountCosts := make(map[string]float64)
	accountCounts := make(map[string]int)
	for _, resource := range resources {
		account := cr.getResourceAccount(resource)
		accountCosts[account] += cr.estimateResourceCost(resource)
		accountCounts[account]++
	}

	for account, cost := range accountCosts {
		breakdown.ByAccount[account] = AccountBreakdown{
			Cost:          cost,
			Percentage:    (cost / costData.TotalCost) * 100,
			ResourceCount: accountCounts[account],
			AverageCost:   cost / float64(accountCounts[account]),
			Trend:         "stable", // Would calculate actual trend
		}
	}

	// Breakdown by project (simplified)
	projectCosts := make(map[string]float64)
	projectCounts := make(map[string]int)
	for _, resource := range resources {
		project := cr.getResourceProject(resource)
		projectCosts[project] += cr.estimateResourceCost(resource)
		projectCounts[project]++
	}

	for project, cost := range projectCosts {
		breakdown.ByProject[project] = ProjectBreakdown{
			Cost:          cost,
			Percentage:    (cost / costData.TotalCost) * 100,
			ResourceCount: projectCounts[project],
			AverageCost:   cost / float64(projectCounts[project]),
			Trend:         "stable", // Would calculate actual trend
		}
	}

	// Breakdown by environment (simplified)
	envCosts := make(map[string]float64)
	envCounts := make(map[string]int)
	for _, resource := range resources {
		env := cr.getResourceEnvironment(resource)
		envCosts[env] += cr.estimateResourceCost(resource)
		envCounts[env]++
	}

	for env, cost := range envCosts {
		breakdown.ByEnvironment[env] = EnvironmentBreakdown{
			Cost:          cost,
			Percentage:    (cost / costData.TotalCost) * 100,
			ResourceCount: envCounts[env],
			AverageCost:   cost / float64(envCounts[env]),
			Trend:         "stable", // Would calculate actual trend
		}
	}

	return breakdown, nil
}

// generateTrends generates cost trends
func (cr *CostReporter) generateTrends(ctx context.Context, period ReportPeriod) ([]TrendPoint, error) {
	// Get trends from optimizer
	trends, err := cr.optimizer.GetOptimizationTrends(period.EndDate.Sub(period.StartDate))
	if err != nil {
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}

	return trends, nil
}

// generateOptimizations generates optimization recommendations
func (cr *CostReporter) generateOptimizations(ctx context.Context, resources []*models.Resource) ([]OptimizationRecommendation, error) {
	// Get optimization recommendations from optimizer
	report, err := cr.optimizer.AnalyzeCostOptimization(ctx, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze cost optimization: %w", err)
	}

	return report.Recommendations, nil
}

// generateForecast generates cost forecast
func (cr *CostReporter) generateForecast(ctx context.Context, resources []*models.Resource) (*CostForecast, error) {
	// Generate 30-day forecast
	forecast, err := cr.forecaster.GenerateForecast(ctx, 30*24*time.Hour, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to generate forecast: %w", err)
	}

	return forecast, nil
}

// generateRecommendations generates report recommendations
func (cr *CostReporter) generateRecommendations(ctx context.Context, costData *CostAnalysis, resources []*models.Resource) ([]ReportRecommendation, error) {
	var recommendations []ReportRecommendation

	// High cost recommendation
	if costData.TotalCost > 10000 {
		recommendation := ReportRecommendation{
			ID:          "high_cost_alert",
			Type:        "cost_control",
			Priority:    "high",
			Title:       "Implement Cost Controls",
			Description: "Total costs exceed $10,000. Implement immediate cost controls.",
			Impact:      "high",
			Effort:      "medium",
			Savings:     1000.0,
			Timeframe:   "1-2 weeks",
			Actions: []string{
				"Set up cost alerts and budgets",
				"Review and terminate unused resources",
				"Implement resource tagging",
				"Schedule regular cost reviews",
			},
		}
		recommendations = append(recommendations, recommendation)
	}

	// Resource optimization recommendation
	if len(resources) > 200 {
		recommendation := ReportRecommendation{
			ID:          "resource_optimization",
			Type:        "optimization",
			Priority:    "medium",
			Title:       "Optimize Resource Usage",
			Description: "Large number of resources detected. Consider optimization opportunities.",
			Impact:      "medium",
			Effort:      "high",
			Savings:     500.0,
			Timeframe:   "2-4 weeks",
			Actions: []string{
				"Audit resource utilization",
				"Implement auto-scaling",
				"Consolidate similar resources",
				"Review resource sizing",
			},
		}
		recommendations = append(recommendations, recommendation)
	}

	// Tagging recommendation
	untaggedCount := cr.countUntaggedResources(resources)
	if untaggedCount > 50 {
		recommendation := ReportRecommendation{
			ID:          "implement_tagging",
			Type:        "governance",
			Priority:    "medium",
			Title:       "Implement Resource Tagging",
			Description: "Many resources lack proper tagging for cost allocation.",
			Impact:      "medium",
			Effort:      "medium",
			Savings:     200.0,
			Timeframe:   "1-2 weeks",
			Actions: []string{
				"Define tagging strategy",
				"Implement automated tagging",
				"Tag existing resources",
				"Set up tag compliance monitoring",
			},
		}
		recommendations = append(recommendations, recommendation)
	}

	return recommendations, nil
}

// Helper methods

// getResourceCountByType returns the count of resources by type
func (cr *CostReporter) getResourceCountByType(resourceType string) int {
	// This would typically query the resource database
	// For now, return a mock count
	return 10
}

// getResourceCountByRegion returns the count of resources by region
func (cr *CostReporter) getResourceCountByRegion(region string, resources []*models.Resource) int {
	count := 0
	for _, resource := range resources {
		if resource.Region == region {
			count++
		}
	}
	return count
}

// getResourceAccount returns the account for a resource
func (cr *CostReporter) getResourceAccount(resource *models.Resource) string {
	if account, ok := resource.Attributes["account"].(string); ok {
		return account
	}
	return "default"
}

// getResourceProject returns the project for a resource
func (cr *CostReporter) getResourceProject(resource *models.Resource) string {
	if project, ok := resource.Attributes["project"].(string); ok {
		return project
	}
	return "default"
}

// getResourceEnvironment returns the environment for a resource
func (cr *CostReporter) getResourceEnvironment(resource *models.Resource) string {
	if env, ok := resource.Attributes["environment"].(string); ok {
		return env
	}
	return "production"
}

// estimateResourceCost estimates the cost of a resource
func (cr *CostReporter) estimateResourceCost(resource *models.Resource) float64 {
	// Simplified cost estimation
	return 50.0
}

// countUntaggedResources counts resources without proper tags
func (cr *CostReporter) countUntaggedResources(resources []*models.Resource) int {
	count := 0
	for _, resource := range resources {
		if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
			if len(tags) == 0 {
				count++
			}
		} else {
			count++
		}
	}
	return count
}

// generateKeyInsights generates key insights for the executive summary
func (cr *CostReporter) generateKeyInsights(summary *ExecutiveSummary, costData *CostAnalysis) []string {
	var insights []string

	// Cost change insight
	if summary.CostChangePercent > 10 {
		insights = append(insights, fmt.Sprintf("Costs increased by %.1f%% compared to previous period", summary.CostChangePercent))
	} else if summary.CostChangePercent < -10 {
		insights = append(insights, fmt.Sprintf("Costs decreased by %.1f%% compared to previous period", -summary.CostChangePercent))
	} else {
		insights = append(insights, "Costs remained relatively stable compared to previous period")
	}

	// Top cost driver insight
	if len(summary.TopCostDrivers) > 0 {
		topDriver := summary.TopCostDrivers[0]
		insights = append(insights, fmt.Sprintf("%s accounts for %.1f%% of total costs", topDriver.Type, topDriver.Percentage))
	}

	// Resource count insight
	insights = append(insights, fmt.Sprintf("Managing %d resources across multiple cloud providers", summary.ResourceCount))

	// Cost per resource insight
	if summary.ResourceCount > 0 {
		avgCostPerResource := summary.TotalCost / float64(summary.ResourceCount)
		insights = append(insights, fmt.Sprintf("Average cost per resource: $%.2f", avgCostPerResource))
	}

	return insights
}
