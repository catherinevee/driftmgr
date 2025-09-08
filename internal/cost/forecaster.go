package cost

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// CostForecaster provides cost forecasting capabilities
type CostForecaster struct {
	analyzer       *CostAnalyzer
	optimizer      *CostOptimizer
	historicalData []CostDataPoint
	forecastModels map[string]ForecastModel
}

// CostDataPoint represents a single cost data point
type CostDataPoint struct {
	Timestamp     time.Time              `json:"timestamp"`
	TotalCost     float64                `json:"total_cost"`
	ResourceCount int                    `json:"resource_count"`
	CostByType    map[string]float64     `json:"cost_by_type"`
	CostByRegion  map[string]float64     `json:"cost_by_region"`
	Currency      string                 `json:"currency"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ForecastModel represents a forecasting model
type ForecastModel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Accuracy    float64                `json:"accuracy"`
	LastUpdated time.Time              `json:"last_updated"`
	Enabled     bool                   `json:"enabled"`
}

// CostForecast represents a cost forecast
type CostForecast struct {
	GeneratedAt     time.Time                `json:"generated_at"`
	ForecastPeriod  time.Duration            `json:"forecast_period"`
	BaseCost        float64                  `json:"base_cost"`
	ForecastedCost  float64                  `json:"forecasted_cost"`
	Confidence      float64                  `json:"confidence"`
	Currency        string                   `json:"currency"`
	DataPoints      []CostDataPoint          `json:"data_points"`
	Trends          []TrendPoint             `json:"trends"`
	Scenarios       []ForecastScenario       `json:"scenarios"`
	Recommendations []ForecastRecommendation `json:"recommendations"`
	Metadata        map[string]interface{}   `json:"metadata,omitempty"`
}

// ForecastScenario represents a forecast scenario
type ForecastScenario struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Probability float64                `json:"probability"`
	Cost        float64                `json:"cost"`
	Factors     map[string]interface{} `json:"factors"`
}

// ForecastRecommendation represents a forecast-based recommendation
type ForecastRecommendation struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Confidence  float64                `json:"confidence"`
	Timeframe   string                 `json:"timeframe"`
	Actions     []string               `json:"actions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewCostForecaster creates a new cost forecaster
func NewCostForecaster(analyzer *CostAnalyzer, optimizer *CostOptimizer) *CostForecaster {
	return &CostForecaster{
		analyzer:       analyzer,
		optimizer:      optimizer,
		historicalData: []CostDataPoint{},
		forecastModels: make(map[string]ForecastModel),
	}
}

// GenerateForecast generates a cost forecast for the specified period
func (cf *CostForecaster) GenerateForecast(ctx context.Context, period time.Duration, resources []*models.Resource) (*CostForecast, error) {
	// Collect historical data
	historicalData, err := cf.collectHistoricalData(ctx, period)
	if err != nil {
		return nil, fmt.Errorf("failed to collect historical data: %w", err)
	}

	// Calculate base cost from current resources
	baseCost, err := cf.calculateBaseCost(ctx, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate base cost: %w", err)
	}

	// Generate forecast using multiple models
	forecast := &CostForecast{
		GeneratedAt:     time.Now(),
		ForecastPeriod:  period,
		BaseCost:        baseCost,
		Currency:        "USD",
		DataPoints:      historicalData,
		Scenarios:       []ForecastScenario{},
		Recommendations: []ForecastRecommendation{},
		Metadata:        make(map[string]interface{}),
	}

	// Apply linear regression model
	linearForecast, linearConfidence := cf.applyLinearRegression(historicalData, period)
	forecast.ForecastedCost = linearForecast
	forecast.Confidence = linearConfidence

	// Generate scenarios
	cf.generateScenarios(forecast, historicalData, period)

	// Generate trends
	cf.generateTrends(forecast, historicalData)

	// Generate recommendations
	cf.generateRecommendations(forecast, resources)

	return forecast, nil
}

// collectHistoricalData collects historical cost data
func (cf *CostForecaster) collectHistoricalData(ctx context.Context, period time.Duration) ([]CostDataPoint, error) {
	// In a real implementation, this would query a time-series database
	// For now, generate mock historical data
	now := time.Now()
	dataPoints := []CostDataPoint{}

	// Generate data points for the last 30 days
	for i := 30; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * 24 * time.Hour)

		// Simulate cost growth over time
		baseCost := 3000.0
		growthFactor := 1.0 + (float64(30-i) * 0.02) // 2% growth per day
		totalCost := baseCost * growthFactor

		dataPoint := CostDataPoint{
			Timestamp:     timestamp,
			TotalCost:     totalCost,
			ResourceCount: 100 + i, // Simulate resource growth
			CostByType: map[string]float64{
				"compute":  totalCost * 0.4,
				"storage":  totalCost * 0.2,
				"database": totalCost * 0.2,
				"network":  totalCost * 0.1,
				"other":    totalCost * 0.1,
			},
			CostByRegion: map[string]float64{
				"us-east-1": totalCost * 0.5,
				"us-west-2": totalCost * 0.3,
				"eu-west-1": totalCost * 0.2,
			},
			Currency: "USD",
			Metadata: map[string]interface{}{
				"data_quality": "high",
				"source":       "billing_api",
			},
		}

		dataPoints = append(dataPoints, dataPoint)
	}

	return dataPoints, nil
}

// calculateBaseCost calculates the base cost from current resources
func (cf *CostForecaster) calculateBaseCost(ctx context.Context, resources []*models.Resource) (float64, error) {
	totalCost := 0.0

	for _, resource := range resources {
		// Estimate cost based on resource type and attributes
		cost := cf.estimateResourceCost(resource)
		totalCost += cost
	}

	return totalCost, nil
}

// estimateResourceCost estimates the cost of a single resource
func (cf *CostForecaster) estimateResourceCost(resource *models.Resource) float64 {
	// This is a simplified cost estimation
	// In reality, you'd use actual pricing APIs

	switch {
	case cf.isComputeInstance(resource):
		return cf.estimateComputeCost(resource)
	case cf.isStorageResource(resource):
		return cf.estimateStorageCost(resource)
	case cf.isDatabaseResource(resource):
		return cf.estimateDatabaseCost(resource)
	case cf.isLoadBalancer(resource):
		return cf.estimateLoadBalancerCost(resource)
	default:
		return 10.0 // Default cost for unknown resources
	}
}

// estimateComputeCost estimates the cost of compute resources
func (cf *CostForecaster) estimateComputeCost(resource *models.Resource) float64 {
	// Simplified cost estimation based on instance type
	if instanceType, ok := resource.Attributes["instance_type"].(string); ok {
		switch {
		case instanceType == "t3.micro":
			return 8.0
		case instanceType == "t3.small":
			return 16.0
		case instanceType == "t3.medium":
			return 32.0
		case instanceType == "t3.large":
			return 64.0
		case instanceType == "m5.large":
			return 80.0
		case instanceType == "m5.xlarge":
			return 160.0
		case instanceType == "m5.2xlarge":
			return 320.0
		default:
			return 50.0 // Default for unknown instance types
		}
	}
	return 50.0
}

// estimateStorageCost estimates the cost of storage resources
func (cf *CostForecaster) estimateStorageCost(resource *models.Resource) float64 {
	if size, ok := resource.Attributes["size"].(int); ok {
		// Estimate $0.10 per GB per month
		return float64(size) * 0.10
	}
	return 20.0 // Default storage cost
}

// estimateDatabaseCost estimates the cost of database resources
func (cf *CostForecaster) estimateDatabaseCost(resource *models.Resource) float64 {
	if instanceClass, ok := resource.Attributes["instance_class"].(string); ok {
		switch {
		case instanceClass == "db.t3.micro":
			return 15.0
		case instanceClass == "db.t3.small":
			return 30.0
		case instanceClass == "db.t3.medium":
			return 60.0
		case instanceClass == "db.r5.large":
			return 120.0
		case instanceClass == "db.r5.xlarge":
			return 240.0
		default:
			return 100.0 // Default database cost
		}
	}
	return 100.0
}

// estimateLoadBalancerCost estimates the cost of load balancer resources
func (cf *CostForecaster) estimateLoadBalancerCost(resource *models.Resource) float64 {
	// Load balancers typically cost around $20-50 per month
	return 35.0
}

// applyLinearRegression applies linear regression to forecast costs
func (cf *CostForecaster) applyLinearRegression(dataPoints []CostDataPoint, period time.Duration) (float64, float64) {
	if len(dataPoints) < 2 {
		return 0.0, 0.0
	}

	// Convert timestamps to numeric values (days since first data point)
	firstTimestamp := dataPoints[0].Timestamp
	xValues := make([]float64, len(dataPoints))
	yValues := make([]float64, len(dataPoints))

	for i, point := range dataPoints {
		xValues[i] = float64(point.Timestamp.Sub(firstTimestamp).Hours() / 24)
		yValues[i] = point.TotalCost
	}

	// Calculate linear regression coefficients
	n := float64(len(dataPoints))
	sumX, sumY, sumXY, sumXX := 0.0, 0.0, 0.0, 0.0

	for i := 0; i < len(dataPoints); i++ {
		sumX += xValues[i]
		sumY += yValues[i]
		sumXY += xValues[i] * yValues[i]
		sumXX += xValues[i] * xValues[i]
	}

	// Calculate slope and intercept
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Calculate R-squared for confidence
	yMean := sumY / n
	ssRes, ssTot := 0.0, 0.0

	for i := 0; i < len(dataPoints); i++ {
		predicted := slope*xValues[i] + intercept
		ssRes += math.Pow(yValues[i]-predicted, 2)
		ssTot += math.Pow(yValues[i]-yMean, 2)
	}

	rSquared := 1.0 - (ssRes / ssTot)
	confidence := math.Max(0.0, math.Min(1.0, rSquared))

	// Forecast for the specified period
	forecastDays := period.Hours() / 24
	forecastX := float64(len(dataPoints)) + forecastDays
	forecastedCost := slope*forecastX + intercept

	return forecastedCost, confidence
}

// generateScenarios generates different forecast scenarios
func (cf *CostForecaster) generateScenarios(forecast *CostForecast, dataPoints []CostDataPoint, period time.Duration) {
	baseForecast := forecast.ForecastedCost

	// Optimistic scenario (10% lower costs)
	optimistic := ForecastScenario{
		Name:        "Optimistic",
		Description: "Best-case scenario with cost optimizations applied",
		Probability: 0.2,
		Cost:        baseForecast * 0.9,
		Factors: map[string]interface{}{
			"optimization_applied": true,
			"resource_efficiency":  "high",
			"demand_growth":        "moderate",
		},
	}

	// Pessimistic scenario (20% higher costs)
	pessimistic := ForecastScenario{
		Name:        "Pessimistic",
		Description: "Worst-case scenario with high demand growth",
		Probability: 0.2,
		Cost:        baseForecast * 1.2,
		Factors: map[string]interface{}{
			"optimization_applied": false,
			"resource_efficiency":  "low",
			"demand_growth":        "high",
		},
	}

	// Realistic scenario (base forecast)
	realistic := ForecastScenario{
		Name:        "Realistic",
		Description: "Most likely scenario based on current trends",
		Probability: 0.6,
		Cost:        baseForecast,
		Factors: map[string]interface{}{
			"optimization_applied": "partial",
			"resource_efficiency":  "medium",
			"demand_growth":        "moderate",
		},
	}

	forecast.Scenarios = []ForecastScenario{optimistic, realistic, pessimistic}
}

// generateTrends generates cost trends from historical data
func (cf *CostForecaster) generateTrends(forecast *CostForecast, dataPoints []CostDataPoint) {
	if len(dataPoints) < 2 {
		return
	}

	// Calculate weekly trends
	weeklyTrends := make(map[string]float64)
	weeklyCounts := make(map[string]int)

	for _, point := range dataPoints {
		week := point.Timestamp.Format("2006-W01")
		weeklyTrends[week] += point.TotalCost
		weeklyCounts[week]++
	}

	// Convert to trend data points
	for week, totalCost := range weeklyTrends {
		avgCost := totalCost / float64(weeklyCounts[week])
		trend := TrendPoint{
			Timestamp:   time.Now(), // Simplified - would parse week string
			MonthlyCost: avgCost,
		}
		forecast.Trends = append(forecast.Trends, trend)
	}

	// Sort trends by timestamp
	sort.Slice(forecast.Trends, func(i, j int) bool {
		return forecast.Trends[i].Timestamp.Before(forecast.Trends[j].Timestamp)
	})
}

// generateRecommendations generates forecast-based recommendations
func (cf *CostForecaster) generateRecommendations(forecast *CostForecast, resources []*models.Resource) {
	// Cost growth recommendation
	if forecast.ForecastedCost > forecast.BaseCost*1.5 {
		recommendation := ForecastRecommendation{
			Type:        "cost_growth_alert",
			Description: "Costs are projected to grow significantly. Consider implementing cost controls.",
			Impact:      "high",
			Confidence:  0.8,
			Timeframe:   "1-3 months",
			Actions: []string{
				"Implement resource tagging for cost allocation",
				"Set up cost alerts and budgets",
				"Review and optimize resource usage",
				"Consider reserved instances for predictable workloads",
			},
			Metadata: map[string]interface{}{
				"projected_growth": (forecast.ForecastedCost - forecast.BaseCost) / forecast.BaseCost,
			},
		}
		forecast.Recommendations = append(forecast.Recommendations, recommendation)
	}

	// Resource optimization recommendation
	if len(resources) > 100 {
		recommendation := ForecastRecommendation{
			Type:        "resource_optimization",
			Description: "Large number of resources detected. Consider consolidation and optimization.",
			Impact:      "medium",
			Confidence:  0.7,
			Timeframe:   "2-4 weeks",
			Actions: []string{
				"Audit unused resources",
				"Consolidate similar resources",
				"Implement auto-scaling policies",
				"Review resource sizing",
			},
			Metadata: map[string]interface{}{
				"resource_count": len(resources),
			},
		}
		forecast.Recommendations = append(forecast.Recommendations, recommendation)
	}

	// Budget planning recommendation
	recommendation := ForecastRecommendation{
		Type:        "budget_planning",
		Description: "Set up budget alerts and monitoring for cost control.",
		Impact:      "medium",
		Confidence:  0.9,
		Timeframe:   "1 week",
		Actions: []string{
			"Create monthly and quarterly budgets",
			"Set up cost alerts at 80% and 100% of budget",
			"Implement cost allocation tags",
			"Regular cost review meetings",
		},
		Metadata: map[string]interface{}{
			"current_cost":    forecast.BaseCost,
			"forecasted_cost": forecast.ForecastedCost,
		},
	}
	forecast.Recommendations = append(forecast.Recommendations, recommendation)
}

// Helper methods (reused from optimizer)

// isComputeInstance checks if the resource is a compute instance
func (cf *CostForecaster) isComputeInstance(resource *models.Resource) bool {
	computeTypes := []string{
		"aws_instance", "aws_ec2_instance", "aws_spot_instance",
		"azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine",
		"google_compute_instance", "digitalocean_droplet",
	}

	for _, t := range computeTypes {
		if resource.Type == t {
			return true
		}
	}
	return false
}

// isStorageResource checks if the resource is a storage resource
func (cf *CostForecaster) isStorageResource(resource *models.Resource) bool {
	storageTypes := []string{
		"aws_s3_bucket", "aws_ebs_volume", "aws_efs_file_system",
		"azurerm_storage_account", "azurerm_managed_disk",
		"google_storage_bucket", "google_compute_disk",
		"digitalocean_volume", "digitalocean_spaces_bucket",
	}

	for _, t := range storageTypes {
		if resource.Type == t {
			return true
		}
	}
	return false
}

// isDatabaseResource checks if the resource is a database resource
func (cf *CostForecaster) isDatabaseResource(resource *models.Resource) bool {
	dbTypes := []string{
		"aws_db_instance", "aws_rds_cluster", "aws_dynamodb_table",
		"azurerm_sql_database", "azurerm_mssql_database", "azurerm_cosmosdb_account",
		"google_sql_database_instance", "google_spanner_instance",
		"digitalocean_database_cluster",
	}

	for _, t := range dbTypes {
		if resource.Type == t {
			return true
		}
	}
	return false
}

// isLoadBalancer checks if the resource is a load balancer
func (cf *CostForecaster) isLoadBalancer(resource *models.Resource) bool {
	lbTypes := []string{
		"aws_lb", "aws_elb", "aws_alb", "aws_nlb",
		"azurerm_lb", "azurerm_application_gateway",
		"google_compute_backend_service", "google_compute_url_map",
		"digitalocean_loadbalancer",
	}

	for _, t := range lbTypes {
		if resource.Type == t {
			return true
		}
	}
	return false
}

// AddHistoricalDataPoint adds a new historical data point
func (cf *CostForecaster) AddHistoricalDataPoint(dataPoint CostDataPoint) {
	cf.historicalData = append(cf.historicalData, dataPoint)

	// Keep only the last 365 days of data
	cutoff := time.Now().Add(-365 * 24 * time.Hour)
	filtered := []CostDataPoint{}
	for _, point := range cf.historicalData {
		if point.Timestamp.After(cutoff) {
			filtered = append(filtered, point)
		}
	}
	cf.historicalData = filtered
}

// GetHistoricalData returns historical cost data
func (cf *CostForecaster) GetHistoricalData() []CostDataPoint {
	return cf.historicalData
}

// RegisterForecastModel registers a new forecasting model
func (cf *CostForecaster) RegisterForecastModel(model ForecastModel) {
	cf.forecastModels[model.ID] = model
}

// GetForecastModels returns all registered forecasting models
func (cf *CostForecaster) GetForecastModels() map[string]ForecastModel {
	return cf.forecastModels
}
