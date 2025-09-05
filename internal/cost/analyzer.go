package cost

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/state"
)

// CostAnalyzer analyzes infrastructure costs
type CostAnalyzer struct {
	providers map[string]CostProvider
	cache     *CostCache
	config    *CostConfig
}

// CostProvider interface for cloud-specific cost calculations
type CostProvider interface {
	GetResourceCost(ctx context.Context, resourceType string, attributes map[string]interface{}) (*ResourceCost, error)
	GetPricingData(ctx context.Context, region string) error
	SupportsResource(resourceType string) bool
}

// ResourceCost represents the cost of a single resource
type ResourceCost struct {
	ResourceAddress string             `json:"resource_address"`
	ResourceType    string             `json:"resource_type"`
	Provider        string             `json:"provider"`
	Region          string             `json:"region"`
	HourlyCost      float64            `json:"hourly_cost"`
	MonthlyCost     float64            `json:"monthly_cost"`
	AnnualCost      float64            `json:"annual_cost"`
	Currency        string             `json:"currency"`
	PriceBreakdown  map[string]float64 `json:"price_breakdown"`
	Confidence      float64            `json:"confidence"` // 0-1 confidence in estimate
	LastUpdated     time.Time          `json:"last_updated"`
	Tags            map[string]string  `json:"tags,omitempty"`
}

// OptimizationRecommendation represents a cost optimization recommendation
type OptimizationRecommendation struct {
	ResourceAddress    string  `json:"resource_address"`
	RecommendationType string  `json:"recommendation_type"`
	Description        string  `json:"description"`
	EstimatedSavings   float64 `json:"estimated_savings"`
	Impact             string  `json:"impact"`
	Confidence         float64 `json:"confidence"`
}

// StateCostReport represents the cost analysis for an entire state
type StateCostReport struct {
	Timestamp        time.Time                       `json:"timestamp"`
	TotalHourlyCost  float64                         `json:"total_hourly_cost"`
	TotalMonthlyCost float64                         `json:"total_monthly_cost"`
	TotalAnnualCost  float64                         `json:"total_annual_cost"`
	Currency         string                          `json:"currency"`
	ResourceCosts    []ResourceCost                  `json:"resource_costs"`
	ProviderSummary  map[string]*ProviderCostSummary `json:"provider_summary"`
	TypeSummary      map[string]*TypeCostSummary     `json:"type_summary"`
	TopExpensive     []ResourceCost                  `json:"top_expensive"`
	Recommendations  []CostRecommendation            `json:"recommendations"`
}

// ProviderCostSummary summarizes costs by provider
type ProviderCostSummary struct {
	Provider      string  `json:"provider"`
	ResourceCount int     `json:"resource_count"`
	HourlyCost    float64 `json:"hourly_cost"`
	MonthlyCost   float64 `json:"monthly_cost"`
	AnnualCost    float64 `json:"annual_cost"`
	Percentage    float64 `json:"percentage"`
}

// TypeCostSummary summarizes costs by resource type
type TypeCostSummary struct {
	ResourceType  string  `json:"resource_type"`
	ResourceCount int     `json:"resource_count"`
	HourlyCost    float64 `json:"hourly_cost"`
	MonthlyCost   float64 `json:"monthly_cost"`
	AnnualCost    float64 `json:"annual_cost"`
	Percentage    float64 `json:"percentage"`
}

// CostRecommendation represents a cost optimization recommendation
type CostRecommendation struct {
	Resource         string  `json:"resource"`
	Type             string  `json:"type"`
	Description      string  `json:"description"`
	PotentialSavings float64 `json:"potential_savings_monthly"`
	Effort           string  `json:"effort"` // low, medium, high
	Risk             string  `json:"risk"`   // low, medium, high
}

// CostConfig contains configuration for cost analysis
type CostConfig struct {
	Currency         string            `json:"currency"`
	HoursPerMonth    float64           `json:"hours_per_month"`
	DefaultRegion    map[string]string `json:"default_region"`
	IncludeFreeTier  bool              `json:"include_free_tier"`
	MarkupPercentage float64           `json:"markup_percentage"`
}

// CostCache caches pricing data
type CostCache struct {
	prices    map[string]*PriceData
	ttl       time.Duration
	lastFetch map[string]time.Time
}

// PriceData represents pricing information
type PriceData struct {
	SKU         string    `json:"sku"`
	Price       float64   `json:"price"`
	Unit        string    `json:"unit"`
	Description string    `json:"description"`
	ValidUntil  time.Time `json:"valid_until"`
}

// NewCostAnalyzer creates a new cost analyzer
func NewCostAnalyzer() *CostAnalyzer {
	analyzer := &CostAnalyzer{
		providers: make(map[string]CostProvider),
		cache:     NewCostCache(24 * time.Hour),
		config: &CostConfig{
			Currency:      "USD",
			HoursPerMonth: 730, // Average hours in a month
			DefaultRegion: map[string]string{
				"aws":    "us-east-1",
				"azure":  "eastus",
				"google": "us-central1",
			},
			IncludeFreeTier:  false,
			MarkupPercentage: 0,
		},
	}

	// Register providers
	analyzer.registerProviders()

	return analyzer
}

// AnalyzeState performs cost analysis on an entire state
func (ca *CostAnalyzer) AnalyzeState(ctx context.Context, state *state.TerraformState) (*StateCostReport, error) {
	report := &StateCostReport{
		Timestamp:       time.Now(),
		Currency:        ca.config.Currency,
		ResourceCosts:   make([]ResourceCost, 0),
		ProviderSummary: make(map[string]*ProviderCostSummary),
		TypeSummary:     make(map[string]*TypeCostSummary),
		TopExpensive:    make([]ResourceCost, 0),
		Recommendations: make([]CostRecommendation, 0),
	}

	// Analyze each resource
	for _, resource := range state.Resources {
		for i, instance := range resource.Instances {
			cost, err := ca.AnalyzeResource(ctx, &resource, &instance, i)
			if err != nil {
				// Log error but continue
				continue
			}

			if cost != nil && cost.MonthlyCost > 0 {
				report.ResourceCosts = append(report.ResourceCosts, *cost)
				report.TotalHourlyCost += cost.HourlyCost
				report.TotalMonthlyCost += cost.MonthlyCost
				report.TotalAnnualCost += cost.AnnualCost

				// Update provider summary
				ca.updateProviderSummary(report, cost)

				// Update type summary
				ca.updateTypeSummary(report, cost)
			}
		}
	}

	// Calculate percentages
	ca.calculatePercentages(report)

	// Find top expensive resources
	report.TopExpensive = ca.findTopExpensive(report.ResourceCosts, 10)

	// Generate recommendations
	report.Recommendations = ca.generateRecommendations(state, report)

	return report, nil
}

// AnalyzeResource analyzes the cost of a single resource
func (ca *CostAnalyzer) AnalyzeResource(ctx context.Context, resource *state.Resource,
	instance *state.Instance, index int) (*ResourceCost, error) {

	// Extract provider name
	providerName := ca.extractProviderName(resource.Provider)

	// Get appropriate cost provider
	provider, exists := ca.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("no cost provider for %s", providerName)
	}

	// Check if provider supports this resource type
	if !provider.SupportsResource(resource.Type) {
		return nil, fmt.Errorf("resource type %s not supported by cost provider", resource.Type)
	}

	// Get resource cost
	cost, err := provider.GetResourceCost(ctx, resource.Type, instance.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource cost: %w", err)
	}

	// Set resource address
	if len(resource.Instances) == 1 {
		cost.ResourceAddress = fmt.Sprintf("%s.%s", resource.Type, resource.Name)
	} else {
		cost.ResourceAddress = fmt.Sprintf("%s.%s[%d]", resource.Type, resource.Name, index)
	}

	// Apply markup if configured
	if ca.config.MarkupPercentage > 0 {
		markup := 1 + (ca.config.MarkupPercentage / 100)
		cost.HourlyCost *= markup
		cost.MonthlyCost *= markup
		cost.AnnualCost *= markup
	}

	// Extract tags if available
	if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
		cost.Tags = make(map[string]string)
		for k, v := range tags {
			if str, ok := v.(string); ok {
				cost.Tags[k] = str
			}
		}
	}

	cost.LastUpdated = time.Now()

	return cost, nil
}

// updateProviderSummary updates the provider cost summary
func (ca *CostAnalyzer) updateProviderSummary(report *StateCostReport, cost *ResourceCost) {
	if _, exists := report.ProviderSummary[cost.Provider]; !exists {
		report.ProviderSummary[cost.Provider] = &ProviderCostSummary{
			Provider: cost.Provider,
		}
	}

	summary := report.ProviderSummary[cost.Provider]
	summary.ResourceCount++
	summary.HourlyCost += cost.HourlyCost
	summary.MonthlyCost += cost.MonthlyCost
	summary.AnnualCost += cost.AnnualCost
}

// updateTypeSummary updates the resource type cost summary
func (ca *CostAnalyzer) updateTypeSummary(report *StateCostReport, cost *ResourceCost) {
	if _, exists := report.TypeSummary[cost.ResourceType]; !exists {
		report.TypeSummary[cost.ResourceType] = &TypeCostSummary{
			ResourceType: cost.ResourceType,
		}
	}

	summary := report.TypeSummary[cost.ResourceType]
	summary.ResourceCount++
	summary.HourlyCost += cost.HourlyCost
	summary.MonthlyCost += cost.MonthlyCost
	summary.AnnualCost += cost.AnnualCost
}

// calculatePercentages calculates percentage of total cost
func (ca *CostAnalyzer) calculatePercentages(report *StateCostReport) {
	if report.TotalMonthlyCost == 0 {
		return
	}

	for _, summary := range report.ProviderSummary {
		summary.Percentage = (summary.MonthlyCost / report.TotalMonthlyCost) * 100
	}

	for _, summary := range report.TypeSummary {
		summary.Percentage = (summary.MonthlyCost / report.TotalMonthlyCost) * 100
	}
}

// findTopExpensive finds the most expensive resources
func (ca *CostAnalyzer) findTopExpensive(costs []ResourceCost, limit int) []ResourceCost {
	// Simple bubble sort for top N (could use heap for better performance)
	sorted := make([]ResourceCost, len(costs))
	copy(sorted, costs)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].MonthlyCost < sorted[j+1].MonthlyCost {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if len(sorted) > limit {
		return sorted[:limit]
	}
	return sorted
}

// generateRecommendations generates cost optimization recommendations
func (ca *CostAnalyzer) generateRecommendations(state *state.TerraformState,
	report *StateCostReport) []CostRecommendation {

	recommendations := make([]CostRecommendation, 0)

	for _, cost := range report.ResourceCosts {
		// Check for oversized instances
		if strings.Contains(cost.ResourceType, "instance") {
			if cost.Confidence < 0.8 {
				recommendations = append(recommendations, CostRecommendation{
					Resource:         cost.ResourceAddress,
					Type:             "rightsizing",
					Description:      "Consider reviewing instance size based on actual utilization",
					PotentialSavings: cost.MonthlyCost * 0.3, // Estimate 30% savings
					Effort:           "medium",
					Risk:             "low",
				})
			}
		}

		// Check for unattached volumes
		if strings.Contains(cost.ResourceType, "volume") || strings.Contains(cost.ResourceType, "disk") {
			// This would need actual attachment checking
			recommendations = append(recommendations, CostRecommendation{
				Resource:         cost.ResourceAddress,
				Type:             "unused_resource",
				Description:      "Verify if this volume is attached and in use",
				PotentialSavings: cost.MonthlyCost,
				Effort:           "low",
				Risk:             "low",
			})
		}

		// Check for old snapshots
		if strings.Contains(cost.ResourceType, "snapshot") {
			recommendations = append(recommendations, CostRecommendation{
				Resource:         cost.ResourceAddress,
				Type:             "old_snapshot",
				Description:      "Review and delete old snapshots if no longer needed",
				PotentialSavings: cost.MonthlyCost,
				Effort:           "low",
				Risk:             "medium",
			})
		}
	}

	// Check for savings plans opportunities
	if report.TotalMonthlyCost > 1000 {
		recommendations = append(recommendations, CostRecommendation{
			Resource:         "overall",
			Type:             "savings_plan",
			Description:      "Consider committed use discounts or savings plans for predictable workloads",
			PotentialSavings: report.TotalMonthlyCost * 0.2, // Estimate 20% savings
			Effort:           "low",
			Risk:             "low",
		})
	}

	return recommendations
}

// registerProviders registers cost providers
func (ca *CostAnalyzer) registerProviders() {
	ca.providers["aws"] = NewAWSCostProvider()
	ca.providers["azurerm"] = NewAzureCostProvider()
	ca.providers["google"] = NewGCPCostProvider()
}

// extractProviderName extracts the provider name
func (ca *CostAnalyzer) extractProviderName(provider string) string {
	parts := strings.Split(provider, "/")
	if len(parts) > 0 {
		// Handle provider aliases
		providerParts := strings.Split(parts[len(parts)-1], ".")
		if len(providerParts) > 0 {
			return providerParts[0]
		}
		return parts[len(parts)-1]
	}
	return provider
}

// GetCostTrend analyzes cost trends over time
func (ca *CostAnalyzer) GetCostTrend(ctx context.Context, historicalReports []*StateCostReport) *CostTrend {
	if len(historicalReports) < 2 {
		return nil
	}

	trend := &CostTrend{
		Period:      fmt.Sprintf("%d reports", len(historicalReports)),
		StartCost:   historicalReports[0].TotalMonthlyCost,
		EndCost:     historicalReports[len(historicalReports)-1].TotalMonthlyCost,
		TrendPoints: make([]TrendPoint, len(historicalReports)),
	}

	for i, report := range historicalReports {
		trend.TrendPoints[i] = TrendPoint{
			Timestamp:   report.Timestamp,
			MonthlyCost: report.TotalMonthlyCost,
		}
	}

	// Calculate trend direction
	trend.Change = trend.EndCost - trend.StartCost
	trend.ChangePercent = (trend.Change / trend.StartCost) * 100

	if trend.ChangePercent > 5 {
		trend.Direction = "increasing"
	} else if trend.ChangePercent < -5 {
		trend.Direction = "decreasing"
	} else {
		trend.Direction = "stable"
	}

	return trend
}

// CalculateResourceCost calculates the cost for a single resource
func (ca *CostAnalyzer) CalculateResourceCost(ctx context.Context, resource *state.Resource) (*ResourceCost, error) {
	// Use the AnalyzeResource method for single resource cost calculation
	if len(resource.Instances) == 0 {
		return nil, fmt.Errorf("no instances found for resource %s", resource.Name)
	}

	cost, err := ca.AnalyzeResource(ctx, resource, &resource.Instances[0], 0)
	if err != nil {
		return nil, err
	}

	return cost, nil
}

// CostTrend represents cost trend analysis
type CostTrend struct {
	Period        string       `json:"period"`
	StartCost     float64      `json:"start_cost"`
	EndCost       float64      `json:"end_cost"`
	Change        float64      `json:"change"`
	ChangePercent float64      `json:"change_percent"`
	Direction     string       `json:"direction"`
	TrendPoints   []TrendPoint `json:"trend_points"`
}

// TrendPoint represents a point in the cost trend
type TrendPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	MonthlyCost float64   `json:"monthly_cost"`
}

// NewCostCache creates a new cost cache
func NewCostCache(ttl time.Duration) *CostCache {
	return &CostCache{
		prices:    make(map[string]*PriceData),
		ttl:       ttl,
		lastFetch: make(map[string]time.Time),
	}
}

// Get retrieves price data from cache
func (cc *CostCache) Get(key string) (*PriceData, bool) {
	if price, exists := cc.prices[key]; exists {
		if time.Now().Before(price.ValidUntil) {
			return price, true
		}
	}
	return nil, false
}

// Set stores price data in cache
func (cc *CostCache) Set(key string, price *PriceData) {
	price.ValidUntil = time.Now().Add(cc.ttl)
	cc.prices[key] = price
	cc.lastFetch[key] = time.Now()
}

// Clear clears the cache
func (cc *CostCache) Clear() {
	cc.prices = make(map[string]*PriceData)
	cc.lastFetch = make(map[string]time.Time)
}
