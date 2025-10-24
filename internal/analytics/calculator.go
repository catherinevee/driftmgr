package analytics

import (
	"fmt"
	"math"
	"sort"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Calculator handles calculations and statistical analysis for analytics
type Calculator struct {
	// In a real implementation, this would have access to statistical libraries
}

// NewCalculator creates a new calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateSummary calculates summary statistics for analytics data
func (c *Calculator) CalculateSummary(data []map[string]interface{}, query *models.AnalyticsQuery) (models.AnalyticsSummary, error) {
	if len(data) == 0 {
		return models.AnalyticsSummary{}, nil
	}

	summary := models.AnalyticsSummary{
		TotalRecords: len(data),
	}

	// Calculate basic statistics
	values := c.extractNumericValues(data, query)
	if len(values) > 0 {
		summary.TotalValue = c.sum(values)
		summary.AverageValue = c.average(values)
		summary.MinValue = c.min(values)
		summary.MaxValue = c.max(values)
	}

	// Calculate trend
	trend, trendPercentage := c.calculateTrend(data, query)
	summary.Trend = trend
	summary.TrendPercentage = trendPercentage

	// Generate insights
	summary.Insights = c.generateInsights(data, query, summary)

	// Generate recommendations
	summary.Recommendations = c.generateRecommendations(data, query, summary)

	return summary, nil
}

// CalculateTrendSummary calculates summary with trend analysis
func (c *Calculator) CalculateTrendSummary(data []map[string]interface{}, query *models.AnalyticsQuery) (models.AnalyticsSummary, error) {
	summary, err := c.CalculateSummary(data, query)
	if err != nil {
		return models.AnalyticsSummary{}, err
	}

	// Enhanced trend analysis
	trend, trendPercentage := c.calculateAdvancedTrend(data, query)
	summary.Trend = trend
	summary.TrendPercentage = trendPercentage

	// Add trend-specific insights
	trendInsights := c.generateTrendInsights(data, query, summary)
	summary.Insights = append(summary.Insights, trendInsights...)

	return summary, nil
}

// CalculateComparisonSummary calculates summary with comparison analysis
func (c *Calculator) CalculateComparisonSummary(data []map[string]interface{}, query *models.AnalyticsQuery) (models.AnalyticsSummary, error) {
	summary, err := c.CalculateSummary(data, query)
	if err != nil {
		return models.AnalyticsSummary{}, err
	}

	// Comparison analysis
	comparisonInsights := c.generateComparisonInsights(data, query, summary)
	summary.Insights = append(summary.Insights, comparisonInsights...)

	// Comparison-specific recommendations
	comparisonRecommendations := c.generateComparisonRecommendations(data, query, summary)
	summary.Recommendations = append(summary.Recommendations, comparisonRecommendations...)

	return summary, nil
}

// Helper methods

// extractNumericValues extracts numeric values from data for calculation
func (c *Calculator) extractNumericValues(data []map[string]interface{}, query *models.AnalyticsQuery) []float64 {
	var values []float64

	// Determine which field to extract based on query type
	var field string
	switch query.QueryType {
	case models.AnalyticsQueryTypeCostAnalysis:
		field = "cost"
	case models.AnalyticsQueryTypeResourceCount:
		field = "count"
	case models.AnalyticsQueryTypeComplianceStatus:
		field = "compliance_rate"
	case models.AnalyticsQueryTypeDriftAnalysis:
		field = "drift_rate"
	case models.AnalyticsQueryTypePerformance:
		field = "average"
	case models.AnalyticsQueryTypeSecurity:
		field = "risk_score"
	default:
		field = "value"
	}

	for _, item := range data {
		if val, ok := item[field]; ok {
			if num, ok := val.(float64); ok {
				values = append(values, num)
			}
		}
	}

	return values
}

// sum calculates the sum of values
func (c *Calculator) sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

// average calculates the average of values
func (c *Calculator) average(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	return c.sum(values) / float64(len(values))
}

// min calculates the minimum value
func (c *Calculator) min(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

// max calculates the maximum value
func (c *Calculator) max(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

// median calculates the median value
func (c *Calculator) median(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// standardDeviation calculates the standard deviation
func (c *Calculator) standardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	mean := c.average(values)
	sumSquaredDiff := 0.0

	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values))
	return math.Sqrt(variance)
}

// calculateTrend calculates basic trend analysis
func (c *Calculator) calculateTrend(data []map[string]interface{}, query *models.AnalyticsQuery) (models.TrendDirection, float64) {
	if len(data) < 2 {
		return models.TrendDirectionUnknown, 0.0
	}

	// Extract values for trend calculation
	values := c.extractNumericValues(data, query)
	if len(values) < 2 {
		return models.TrendDirectionUnknown, 0.0
	}

	// Simple linear trend calculation
	first := values[0]
	last := values[len(values)-1]

	if first == 0 {
		return models.TrendDirectionUnknown, 0.0
	}

	changePercent := ((last - first) / first) * 100

	if math.Abs(changePercent) < 1.0 {
		return models.TrendDirectionStable, changePercent
	} else if changePercent > 0 {
		return models.TrendDirectionUp, changePercent
	} else {
		return models.TrendDirectionDown, changePercent
	}
}

// calculateAdvancedTrend calculates advanced trend analysis
func (c *Calculator) calculateAdvancedTrend(data []map[string]interface{}, query *models.AnalyticsQuery) (models.TrendDirection, float64) {
	if len(data) < 3 {
		return c.calculateTrend(data, query)
	}

	// Extract values for trend calculation
	values := c.extractNumericValues(data, query)
	if len(values) < 3 {
		return c.calculateTrend(data, query)
	}

	// Calculate trend using linear regression
	slope, rSquared := c.linearRegression(values)

	// Determine trend direction based on slope and R-squared
	if rSquared < 0.5 {
		return models.TrendDirectionUnknown, 0.0
	}

	if math.Abs(slope) < 0.01 {
		return models.TrendDirectionStable, 0.0
	} else if slope > 0 {
		return models.TrendDirectionUp, slope * 100
	} else {
		return models.TrendDirectionDown, slope * 100
	}
}

// linearRegression performs simple linear regression
func (c *Calculator) linearRegression(values []float64) (slope, rSquared float64) {
	n := float64(len(values))
	if n < 2 {
		return 0.0, 0.0
	}

	// Calculate means
	sumX, sumY := 0.0, 0.0
	for i, v := range values {
		sumX += float64(i)
		sumY += v
	}
	meanX := sumX / n
	meanY := sumY / n

	// Calculate slope and intercept
	var numerator, denominator float64
	for i, v := range values {
		x := float64(i)
		numerator += (x - meanX) * (v - meanY)
		denominator += (x - meanX) * (x - meanX)
	}

	if denominator == 0 {
		return 0.0, 0.0
	}

	slope = numerator / denominator

	// Calculate R-squared
	var ssRes, ssTot float64
	for i, v := range values {
		x := float64(i)
		predicted := slope*x + (meanY - slope*meanX)
		ssRes += (v - predicted) * (v - predicted)
		ssTot += (v - meanY) * (v - meanY)
	}

	if ssTot == 0 {
		rSquared = 0.0
	} else {
		rSquared = 1 - (ssRes / ssTot)
	}

	return slope, rSquared
}

// generateInsights generates insights from the data and summary
func (c *Calculator) generateInsights(data []map[string]interface{}, query *models.AnalyticsQuery, summary models.AnalyticsSummary) []string {
	var insights []string

	switch query.QueryType {
	case models.AnalyticsQueryTypeCostAnalysis:
		insights = c.generateCostInsights(data, summary)
	case models.AnalyticsQueryTypeResourceCount:
		insights = c.generateResourceInsights(data, summary)
	case models.AnalyticsQueryTypeComplianceStatus:
		insights = c.generateComplianceInsights(data, summary)
	case models.AnalyticsQueryTypeDriftAnalysis:
		insights = c.generateDriftInsights(data, summary)
	case models.AnalyticsQueryTypePerformance:
		insights = c.generatePerformanceInsights(data, summary)
	case models.AnalyticsQueryTypeSecurity:
		insights = c.generateSecurityInsights(data, summary)
	default:
		insights = c.generateGeneralInsights(data, summary)
	}

	return insights
}

// generateCostInsights generates cost-specific insights
func (c *Calculator) generateCostInsights(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.Trend == models.TrendDirectionUp && summary.TrendPercentage > 10 {
		insights = append(insights, fmt.Sprintf("Costs are increasing by %.1f%%, which may require attention", summary.TrendPercentage))
	}

	if summary.AverageValue > 1000 {
		insights = append(insights, "Average cost per resource is high, consider optimization opportunities")
	}

	// Find highest cost provider
	var maxCost float64
	var maxProvider string
	for _, item := range data {
		if cost, ok := item["cost"].(float64); ok {
			if cost > maxCost {
				maxCost = cost
				if provider, ok := item["provider"].(string); ok {
					maxProvider = provider
				}
			}
		}
	}

	if maxProvider != "" {
		insights = append(insights, fmt.Sprintf("%s has the highest costs at $%.2f", maxProvider, maxCost))
	}

	return insights
}

// generateResourceInsights generates resource-specific insights
func (c *Calculator) generateResourceInsights(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.TotalRecords > 1000 {
		insights = append(insights, "Large number of resources detected, consider resource optimization")
	}

	// Count by provider
	providerCounts := make(map[string]int)
	for _, item := range data {
		if provider, ok := item["provider"].(string); ok {
			providerCounts[provider]++
		}
	}

	if len(providerCounts) > 1 {
		insights = append(insights, fmt.Sprintf("Resources distributed across %d providers", len(providerCounts)))
	}

	return insights
}

// generateComplianceInsights generates compliance-specific insights
func (c *Calculator) generateComplianceInsights(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.AverageValue < 90 {
		insights = append(insights, "Overall compliance rate is below 90%, immediate attention required")
	} else if summary.AverageValue < 95 {
		insights = append(insights, "Compliance rate is below 95%, consider improvements")
	} else {
		insights = append(insights, "Good compliance rate maintained")
	}

	// Check for critical violations
	var totalCritical int
	for _, item := range data {
		if critical, ok := item["critical_violations"].(int); ok {
			totalCritical += critical
		}
	}

	if totalCritical > 0 {
		insights = append(insights, fmt.Sprintf("%d critical compliance violations detected", totalCritical))
	}

	return insights
}

// generateDriftInsights generates drift-specific insights
func (c *Calculator) generateDriftInsights(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.AverageValue > 10 {
		insights = append(insights, "High drift rate detected, consider implementing drift prevention measures")
	} else if summary.AverageValue > 5 {
		insights = append(insights, "Moderate drift rate, monitor closely")
	} else {
		insights = append(insights, "Low drift rate, good state management")
	}

	// Check for critical drift
	var totalCritical int
	for _, item := range data {
		if critical, ok := item["critical_drift"].(int); ok {
			totalCritical += critical
		}
	}

	if totalCritical > 0 {
		insights = append(insights, fmt.Sprintf("%d critical drift instances detected", totalCritical))
	}

	return insights
}

// generatePerformanceInsights generates performance-specific insights
func (c *Calculator) generatePerformanceInsights(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.AverageValue > 80 {
		insights = append(insights, "High resource utilization detected, consider scaling")
	} else if summary.AverageValue < 20 {
		insights = append(insights, "Low resource utilization, consider rightsizing")
	} else {
		insights = append(insights, "Resource utilization within normal range")
	}

	return insights
}

// generateSecurityInsights generates security-specific insights
func (c *Calculator) generateSecurityInsights(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.AverageValue > 8 {
		insights = append(insights, "High security risk detected, immediate attention required")
	} else if summary.AverageValue > 6 {
		insights = append(insights, "Moderate security risk, review security policies")
	} else {
		insights = append(insights, "Security risk within acceptable range")
	}

	// Count critical security issues
	var totalCritical int
	for _, item := range data {
		if severity, ok := item["severity"].(string); ok && severity == "critical" {
			totalCritical++
		}
	}

	if totalCritical > 0 {
		insights = append(insights, fmt.Sprintf("%d critical security issues detected", totalCritical))
	}

	return insights
}

// generateGeneralInsights generates general insights
func (c *Calculator) generateGeneralInsights(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.Trend == models.TrendDirectionUp {
		insights = append(insights, fmt.Sprintf("Upward trend detected with %.1f%% change", summary.TrendPercentage))
	} else if summary.Trend == models.TrendDirectionDown {
		insights = append(insights, fmt.Sprintf("Downward trend detected with %.1f%% change", summary.TrendPercentage))
	} else {
		insights = append(insights, "Stable trend observed")
	}

	return insights
}

// generateTrendInsights generates trend-specific insights
func (c *Calculator) generateTrendInsights(data []map[string]interface{}, query *models.AnalyticsQuery, summary models.AnalyticsSummary) []string {
	var insights []string

	if summary.Trend == models.TrendDirectionUp && summary.TrendPercentage > 20 {
		insights = append(insights, "Significant upward trend detected, consider proactive measures")
	} else if summary.Trend == models.TrendDirectionDown && summary.TrendPercentage < -20 {
		insights = append(insights, "Significant downward trend detected, investigate causes")
	}

	return insights
}

// generateComparisonInsights generates comparison-specific insights
func (c *Calculator) generateComparisonInsights(data []map[string]interface{}, query *models.AnalyticsQuery, summary models.AnalyticsSummary) []string {
	var insights []string

	// Compare current vs previous periods
	currentData := make([]map[string]interface{}, 0)
	previousData := make([]map[string]interface{}, 0)

	for _, item := range data {
		if period, ok := item["period"].(string); ok {
			if period == "current" {
				currentData = append(currentData, item)
			} else if period == "previous" {
				previousData = append(previousData, item)
			}
		}
	}

	if len(currentData) > 0 && len(previousData) > 0 {
		currentTotal := c.calculateTotal(currentData)
		previousTotal := c.calculateTotal(previousData)

		if previousTotal > 0 {
			changePercent := ((currentTotal - previousTotal) / previousTotal) * 100
			insights = append(insights, fmt.Sprintf("Change from previous period: %.1f%%", changePercent))
		}
	}

	return insights
}

// calculateTotal calculates total from data
func (c *Calculator) calculateTotal(data []map[string]interface{}) float64 {
	total := 0.0
	for _, item := range data {
		if cost, ok := item["cost"].(float64); ok {
			total += cost
		}
	}
	return total
}

// generateRecommendations generates recommendations based on the data and summary
func (c *Calculator) generateRecommendations(data []map[string]interface{}, query *models.AnalyticsQuery, summary models.AnalyticsSummary) []string {
	var recommendations []string

	switch query.QueryType {
	case models.AnalyticsQueryTypeCostAnalysis:
		recommendations = c.generateCostRecommendations(data, summary)
	case models.AnalyticsQueryTypeResourceCount:
		recommendations = c.generateResourceRecommendations(data, summary)
	case models.AnalyticsQueryTypeComplianceStatus:
		recommendations = c.generateComplianceRecommendations(data, summary)
	case models.AnalyticsQueryTypeDriftAnalysis:
		recommendations = c.generateDriftRecommendations(data, summary)
	case models.AnalyticsQueryTypePerformance:
		recommendations = c.generatePerformanceRecommendations(data, summary)
	case models.AnalyticsQueryTypeSecurity:
		recommendations = c.generateSecurityRecommendations(data, summary)
	default:
		recommendations = c.generateGeneralRecommendations(data, summary)
	}

	return recommendations
}

// generateCostRecommendations generates cost-specific recommendations
func (c *Calculator) generateCostRecommendations(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var recommendations []string

	if summary.Trend == models.TrendDirectionUp && summary.TrendPercentage > 10 {
		recommendations = append(recommendations, "Review resource usage and consider rightsizing")
		recommendations = append(recommendations, "Implement cost monitoring and alerting")
	}

	if summary.AverageValue > 1000 {
		recommendations = append(recommendations, "Consider reserved instances for predictable workloads")
		recommendations = append(recommendations, "Review and optimize storage costs")
	}

	return recommendations
}

// generateResourceRecommendations generates resource-specific recommendations
func (c *Calculator) generateResourceRecommendations(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var recommendations []string

	if summary.TotalRecords > 1000 {
		recommendations = append(recommendations, "Implement resource tagging strategy")
		recommendations = append(recommendations, "Consider resource consolidation")
	}

	return recommendations
}

// generateComplianceRecommendations generates compliance-specific recommendations
func (c *Calculator) generateComplianceRecommendations(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var recommendations []string

	if summary.AverageValue < 95 {
		recommendations = append(recommendations, "Implement automated compliance checking")
		recommendations = append(recommendations, "Review and update security policies")
	}

	return recommendations
}

// generateDriftRecommendations generates drift-specific recommendations
func (c *Calculator) generateDriftRecommendations(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var recommendations []string

	if summary.AverageValue > 5 {
		recommendations = append(recommendations, "Implement drift detection and prevention")
		recommendations = append(recommendations, "Review infrastructure as code practices")
	}

	return recommendations
}

// generatePerformanceRecommendations generates performance-specific recommendations
func (c *Calculator) generatePerformanceRecommendations(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var recommendations []string

	if summary.AverageValue > 80 {
		recommendations = append(recommendations, "Consider horizontal scaling")
		recommendations = append(recommendations, "Review resource allocation")
	} else if summary.AverageValue < 20 {
		recommendations = append(recommendations, "Consider rightsizing resources")
		recommendations = append(recommendations, "Review resource utilization patterns")
	}

	return recommendations
}

// generateSecurityRecommendations generates security-specific recommendations
func (c *Calculator) generateSecurityRecommendations(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var recommendations []string

	if summary.AverageValue > 6 {
		recommendations = append(recommendations, "Implement security hardening measures")
		recommendations = append(recommendations, "Review access controls and permissions")
	}

	return recommendations
}

// generateGeneralRecommendations generates general recommendations
func (c *Calculator) generateGeneralRecommendations(data []map[string]interface{}, summary models.AnalyticsSummary) []string {
	var recommendations []string

	if summary.Trend == models.TrendDirectionUp {
		recommendations = append(recommendations, "Monitor trends closely and plan for growth")
	}

	return recommendations
}

// generateComparisonRecommendations generates comparison-specific recommendations
func (c *Calculator) generateComparisonRecommendations(data []map[string]interface{}, query *models.AnalyticsQuery, summary models.AnalyticsSummary) []string {
	var recommendations []string

	// Add comparison-specific recommendations
	recommendations = append(recommendations, "Continue monitoring trends and patterns")
	recommendations = append(recommendations, "Implement regular comparison analysis")

	return recommendations
}
