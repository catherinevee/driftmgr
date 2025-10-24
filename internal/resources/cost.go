package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// CostManager manages resource cost information
type CostManager struct {
	costs map[string]*models.CostInformation
	mu    sync.RWMutex
}

// NewCostManager creates a new cost manager
func NewCostManager() *CostManager {
	return &CostManager{
		costs: make(map[string]*models.CostInformation),
	}
}

// UpdateCost updates cost information for a resource
func (m *CostManager) UpdateCost(ctx context.Context, resource *models.CloudResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create cost information
	cost := m.costs[resource.ID]
	if cost == nil {
		cost = &models.CostInformation{
			Currency:      "USD",
			LastUpdated:   time.Now(),
			BillingPeriod: "monthly",
		}
	}

	// Calculate cost based on resource type
	m.calculateCost(cost, resource)
	cost.LastUpdated = time.Now()

	m.costs[resource.ID] = cost
	return nil
}

// Enrich enriches a resource with cost data
func (m *CostManager) Enrich(ctx context.Context, resource *models.CloudResource) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cost := m.costs[resource.ID]
	if cost == nil {
		return nil // No cost data available
	}

	// Create a copy of the cost information
	resource.Cost = *cost
	return nil
}

// GetCost retrieves cost information for a resource
func (m *CostManager) GetCost(ctx context.Context, resourceID string) (*models.CostInformation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cost := m.costs[resourceID]
	if cost == nil {
		return nil, fmt.Errorf("cost information for resource %s not found", resourceID)
	}

	return cost, nil
}

// DeleteCost deletes cost data for a resource
func (m *CostManager) DeleteCost(ctx context.Context, resourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.costs, resourceID)
	return nil
}

// GetTotalCost returns the total cost across all resources
func (m *CostManager) GetTotalCost(ctx context.Context) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total float64
	for _, cost := range m.costs {
		total += cost.MonthlyCost
	}

	return total, nil
}

// GetCostByProvider returns cost breakdown by provider
func (m *CostManager) GetCostByProvider(ctx context.Context, resources map[models.CloudProvider][]models.CloudResource) (map[models.CloudProvider]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	costByProvider := make(map[models.CloudProvider]float64)

	for provider, providerResources := range resources {
		var providerCost float64
		for _, resource := range providerResources {
			if cost := m.costs[resource.ID]; cost != nil {
				providerCost += cost.MonthlyCost
			}
		}
		costByProvider[provider] = providerCost
	}

	return costByProvider, nil
}

// GetCostByType returns cost breakdown by resource type
func (m *CostManager) GetCostByType(ctx context.Context, resources map[string][]models.CloudResource) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	costByType := make(map[string]float64)

	for resourceType, typeResources := range resources {
		var typeCost float64
		for _, resource := range typeResources {
			if cost := m.costs[resource.ID]; cost != nil {
				typeCost += cost.MonthlyCost
			}
		}
		costByType[resourceType] = typeCost
	}

	return costByType, nil
}

// GetStatistics returns cost statistics
func (m *CostManager) GetStatistics(ctx context.Context) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"total_resources":      len(m.costs),
		"total_monthly_cost":   0.0,
		"total_daily_cost":     0.0,
		"total_hourly_cost":    0.0,
		"average_monthly_cost": 0.0,
		"highest_cost":         0.0,
		"lowest_cost":          0.0,
		"cost_by_currency":     make(map[string]float64),
	}

	var totalMonthly, totalDaily, totalHourly float64
	var highest, lowest float64
	currencyCounts := make(map[string]float64)

	first := true
	for _, cost := range m.costs {
		totalMonthly += cost.MonthlyCost
		totalDaily += cost.DailyCost
		totalHourly += cost.HourlyCost
		currencyCounts[cost.Currency] += cost.MonthlyCost

		if first {
			highest = cost.MonthlyCost
			lowest = cost.MonthlyCost
			first = false
		} else {
			if cost.MonthlyCost > highest {
				highest = cost.MonthlyCost
			}
			if cost.MonthlyCost < lowest {
				lowest = cost.MonthlyCost
			}
		}
	}

	stats["total_monthly_cost"] = totalMonthly
	stats["total_daily_cost"] = totalDaily
	stats["total_hourly_cost"] = totalHourly
	stats["cost_by_currency"] = currencyCounts

	if len(m.costs) > 0 {
		stats["average_monthly_cost"] = totalMonthly / float64(len(m.costs))
		stats["highest_cost"] = highest
		stats["lowest_cost"] = lowest
	}

	return stats
}

// Health checks the health of the cost manager
func (m *CostManager) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Basic health check
	if len(m.costs) < 0 {
		return fmt.Errorf("cost manager has negative count")
	}

	return nil
}

// calculateCost calculates cost for a resource based on its type and configuration
func (m *CostManager) calculateCost(cost *models.CostInformation, resource *models.CloudResource) {
	// This is a simplified cost calculation
	// In production, you would integrate with actual cost APIs

	switch resource.Type {
	case "aws_instance":
		m.calculateEC2Cost(cost, resource)
	case "aws_s3_bucket":
		m.calculateS3Cost(cost, resource)
	case "aws_db_instance":
		m.calculateRDSCost(cost, resource)
	case "aws_lambda_function":
		m.calculateLambdaCost(cost, resource)
	default:
		// Default cost calculation
		cost.MonthlyCost = 10.0
		cost.DailyCost = cost.MonthlyCost / 30
		cost.HourlyCost = cost.MonthlyCost / (30 * 24)
	}

	// Set cost breakdown
	cost.CostBreakdown = map[string]float64{
		"compute": cost.MonthlyCost * 0.6,
		"storage": cost.MonthlyCost * 0.2,
		"network": cost.MonthlyCost * 0.1,
		"other":   cost.MonthlyCost * 0.1,
	}
}

// calculateEC2Cost calculates cost for EC2 instances
func (m *CostManager) calculateEC2Cost(cost *models.CostInformation, resource *models.CloudResource) {
	// Base cost for EC2 instance
	baseCost := 50.0

	// Adjust based on instance type
	if config := resource.Configuration; config != nil {
		if instanceType, ok := config["instance_type"].(string); ok {
			switch instanceType {
			case "t2.micro":
				baseCost = 8.0
			case "t2.small":
				baseCost = 16.0
			case "t2.medium":
				baseCost = 32.0
			case "t2.large":
				baseCost = 64.0
			case "m5.large":
				baseCost = 80.0
			case "m5.xlarge":
				baseCost = 160.0
			case "c5.large":
				baseCost = 70.0
			case "c5.xlarge":
				baseCost = 140.0
			}
		}
	}

	cost.MonthlyCost = baseCost
	cost.DailyCost = baseCost / 30
	cost.HourlyCost = baseCost / (30 * 24)
}

// calculateS3Cost calculates cost for S3 buckets
func (m *CostManager) calculateS3Cost(cost *models.CostInformation, resource *models.CloudResource) {
	// Base cost for S3 storage (per GB per month)
	baseCost := 0.023      // $0.023 per GB per month
	estimatedSize := 100.0 // Assume 100GB average

	cost.MonthlyCost = baseCost * estimatedSize
	cost.DailyCost = cost.MonthlyCost / 30
	cost.HourlyCost = cost.MonthlyCost / (30 * 24)
}

// calculateRDSCost calculates cost for RDS instances
func (m *CostManager) calculateRDSCost(cost *models.CostInformation, resource *models.CloudResource) {
	// Base cost for RDS instance
	baseCost := 100.0

	// Adjust based on instance class
	if config := resource.Configuration; config != nil {
		if instanceClass, ok := config["db_instance_class"].(string); ok {
			switch instanceClass {
			case "db.t2.micro":
				baseCost = 15.0
			case "db.t2.small":
				baseCost = 30.0
			case "db.t2.medium":
				baseCost = 60.0
			case "db.t2.large":
				baseCost = 120.0
			case "db.m5.large":
				baseCost = 150.0
			case "db.m5.xlarge":
				baseCost = 300.0
			}
		}
	}

	cost.MonthlyCost = baseCost
	cost.DailyCost = baseCost / 30
	cost.HourlyCost = baseCost / (30 * 24)
}

// calculateLambdaCost calculates cost for Lambda functions
func (m *CostManager) calculateLambdaCost(cost *models.CostInformation, resource *models.CloudResource) {
	// Base cost for Lambda (per million requests)
	baseCost := 0.20               // $0.20 per million requests
	estimatedRequests := 1000000.0 // Assume 1 million requests per month

	cost.MonthlyCost = baseCost
	cost.DailyCost = cost.MonthlyCost / 30
	cost.HourlyCost = cost.MonthlyCost / (30 * 24)
}
