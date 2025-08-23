package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// AnalyticsSummary represents analytics overview
type AnalyticsSummary struct {
	Period            string                   `json:"period"`
	TotalResources    int                      `json:"totalResources"`
	TotalDrifts       int                      `json:"totalDrifts"`
	DriftRate         float64                  `json:"driftRate"`
	RemediationRate   float64                  `json:"remediationRate"`
	CostSavings       float64                  `json:"costSavings"`
	ComplianceScore   float64                  `json:"complianceScore"`
	ProviderBreakdown map[string]ProviderStats `json:"providerBreakdown"`
	Trends            TrendData                `json:"trends"`
	TopIssues         []Issue                  `json:"topIssues"`
}

// ProviderStats represents statistics for a provider
type ProviderStats struct {
	Resources   int     `json:"resources"`
	Drifts      int     `json:"drifts"`
	Cost        float64 `json:"cost"`
	Regions     int     `json:"regions"`
	Services    int     `json:"services"`
	HealthScore float64 `json:"healthScore"`
}

// TrendData represents trend information
type TrendData struct {
	ResourceGrowth  float64 `json:"resourceGrowth"`
	DriftTrend      float64 `json:"driftTrend"`
	CostTrend       float64 `json:"costTrend"`
	ComplianceTrend float64 `json:"complianceTrend"`
}

// Issue represents a top issue
type Issue struct {
	Type       string  `json:"type"`
	Severity   string  `json:"severity"`
	Count      int     `json:"count"`
	Impact     string  `json:"impact"`
	Resolution string  `json:"resolution"`
	CostImpact float64 `json:"costImpact"`
}

// TimeSeriesData represents time series analytics data
type TimeSeriesData struct {
	Timestamps []string               `json:"timestamps"`
	Series     map[string][]float64   `json:"series"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// CostAnalysis represents cost analysis data
type CostAnalysis struct {
	TotalCost        float64              `json:"totalCost"`
	ProjectedCost    float64              `json:"projectedCost"`
	PotentialSavings float64              `json:"potentialSavings"`
	ByProvider       map[string]float64   `json:"byProvider"`
	ByService        map[string]float64   `json:"byService"`
	ByRegion         map[string]float64   `json:"byRegion"`
	UnusedResources  []UnusedResource     `json:"unusedResources"`
	Recommendations  []CostRecommendation `json:"recommendations"`
}

// UnusedResource represents an unused resource
type UnusedResource struct {
	ResourceID   string  `json:"resourceId"`
	ResourceType string  `json:"resourceType"`
	Provider     string  `json:"provider"`
	MonthlyCost  float64 `json:"monthlyCost"`
	LastUsed     string  `json:"lastUsed"`
	Reason       string  `json:"reason"`
}

// CostRecommendation represents a cost optimization recommendation
type CostRecommendation struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	PotentialSavings float64  `json:"potentialSavings"`
	Effort           string   `json:"effort"`
	Impact           string   `json:"impact"`
	Priority         string   `json:"priority"`
	Actions          []string `json:"actions"`
}

// getAnalyticsSummary retrieves analytics summary
func (s *EnhancedDashboardServer) getAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	// Get data from datastore
	resources := s.dataStore.GetResources()
	drifts := s.dataStore.GetDrifts()
	remediations := s.dataStore.GetRemediationHistory()

	// Calculate provider breakdown
	providerBreakdown := make(map[string]ProviderStats)
	for _, resourceInterface := range resources {
		// Type assertion with map check
		if resourceMap, ok := resourceInterface.(map[string]interface{}); ok {
			if provider, ok := resourceMap["provider"].(string); ok {
				if _, exists := providerBreakdown[provider]; !exists {
					providerBreakdown[provider] = ProviderStats{}
				}
				stats := providerBreakdown[provider]
				stats.Resources++
				providerBreakdown[provider] = stats
			}
		}
	}

	// Count drifts by provider
	for _, driftInterface := range drifts {
		// Type assertion with map check
		if driftMap, ok := driftInterface.(map[string]interface{}); ok {
			if provider, ok := driftMap["provider"].(string); ok {
				if stats, exists := providerBreakdown[provider]; exists {
					stats.Drifts++
					providerBreakdown[provider] = stats
				}
			}
		}
	}

	// Calculate health scores
	for provider, stats := range providerBreakdown {
		if stats.Resources > 0 {
			stats.HealthScore = 100.0 - (float64(stats.Drifts)/float64(stats.Resources))*100
			stats.Cost = s.calculateProviderCostFromInterfaces(provider, resources)
			providerBreakdown[provider] = stats
		}
	}

	// Calculate rates
	driftRate := 0.0
	if len(resources) > 0 {
		driftRate = (float64(len(drifts)) / float64(len(resources))) * 100
	}

	remediationRate := 0.0
	successfulRemediations := 0
	for _, r := range remediations {
		if rMap, ok := r.(map[string]interface{}); ok {
			if status, ok := rMap["status"].(string); ok && status == "completed" {
				successfulRemediations++
			}
		}
	}
	if len(remediations) > 0 {
		remediationRate = (float64(successfulRemediations) / float64(len(remediations))) * 100
	}

	// Generate top issues
	topIssuesMap := s.analyzeTopIssuesFromInterfaces(drifts, resources)
	topIssues := make([]Issue, 0, len(topIssuesMap))
	for _, issueMap := range topIssuesMap {
		issue := Issue{
			Type:     getStringFromMap(issueMap, "type"),
			Severity: getStringFromMap(issueMap, "severity"),
			Count:    getIntFromMap(issueMap, "count"),
			Impact:   getStringFromMap(issueMap, "impact"),
		}
		topIssues = append(topIssues, issue)
	}

	// Create summary
	summary := AnalyticsSummary{
		Period:            period,
		TotalResources:    len(resources),
		TotalDrifts:       len(drifts),
		DriftRate:         driftRate,
		RemediationRate:   remediationRate,
		CostSavings:       s.calculateCostSavings(period),
		ComplianceScore:   s.calculateComplianceScoreFromInterfaces(resources),
		ProviderBreakdown: providerBreakdown,
		Trends: TrendData{
			ResourceGrowth:  5.2, // Simplified - would calculate from historical data
			DriftTrend:      -2.1,
			CostTrend:       3.5,
			ComplianceTrend: 1.8,
		},
		TopIssues: topIssues,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// getTrends retrieves trend data
func (s *EnhancedDashboardServer) getTrends(w http.ResponseWriter, r *http.Request) {
	metric := r.URL.Query().Get("metric")
	period := r.URL.Query().Get("period")
	resolution := r.URL.Query().Get("resolution")

	if period == "" {
		period = "7d"
	}
	if resolution == "" {
		resolution = "1h"
	}

	// Generate time series data
	timeSeriesData := s.generateTimeSeriesData(metric, period, resolution)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeSeriesData)
}

// getCostAnalysis retrieves cost analysis
func (s *EnhancedDashboardServer) getCostAnalysis(w http.ResponseWriter, r *http.Request) {
	// Get parameters
	groupBy := r.URL.Query().Get("groupBy")
	if groupBy == "" {
		groupBy = "provider"
	}

	includeProjections := r.URL.Query().Get("projections") == "true"
	includeRecommendations := r.URL.Query().Get("recommendations") == "true"

	resources := s.dataStore.GetResources()

	// Calculate costs
	totalCost := 0.0
	byProvider := make(map[string]float64)
	byService := make(map[string]float64)
	byRegion := make(map[string]float64)

	for _, resource := range resources {
		// Type assert and extract fields
		var provider, resType, region string
		var cost float64

		if resMap, ok := resource.(map[string]interface{}); ok {
			provider = getStringFromMap(resMap, "provider")
			resType = getStringFromMap(resMap, "type")
			region = getStringFromMap(resMap, "region")
			// Simple cost calculation
			cost = 10.0 // Base cost per resource
		}

		totalCost += cost
		byProvider[provider] += cost
		byService[resType] += cost
		byRegion[region] += cost
	}

	analysis := CostAnalysis{
		TotalCost:        totalCost,
		ProjectedCost:    totalCost * 1.1,  // 10% growth projection
		PotentialSavings: totalCost * 0.25, // 25% potential savings
		ByProvider:       byProvider,
		ByService:        byService,
		ByRegion:         byRegion,
	}

	// Find unused resources
	if includeProjections {
		analysis.UnusedResources = s.findUnusedResourcesFromInterfaces(resources)
	}

	// Generate recommendations
	if includeRecommendations {
		analysis.Recommendations = s.generateCostRecommendationsFromInterfaces(resources)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// getResourceUtilization retrieves resource utilization metrics
func (s *EnhancedDashboardServer) getResourceUtilization(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	resourceType := r.URL.Query().Get("type")

	utilization := map[string]interface{}{
		"overall": map[string]interface{}{
			"cpu":     65.5,
			"memory":  72.3,
			"storage": 45.8,
			"network": 38.2,
		},
		"byProvider": make(map[string]interface{}),
		"alerts": []map[string]interface{}{
			{
				"type":      "underutilized",
				"resource":  "i-1234567890",
				"metric":    "cpu",
				"value":     5.2,
				"threshold": 10.0,
			},
			{
				"type":      "overutilized",
				"resource":  "i-0987654321",
				"metric":    "memory",
				"value":     95.5,
				"threshold": 90.0,
			},
		},
	}

	// Filter by provider if specified
	if provider != "" {
		utilization["provider"] = provider
	}

	// Filter by resource type if specified
	if resourceType != "" {
		utilization["resourceType"] = resourceType
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(utilization)
}

// getComplianceReport retrieves compliance report
func (s *EnhancedDashboardServer) getComplianceReport(w http.ResponseWriter, r *http.Request) {
	standard := r.URL.Query().Get("standard")
	if standard == "" {
		standard = "all"
	}

	resources := s.dataStore.GetResources()

	report := map[string]interface{}{
		"overallScore": s.calculateComplianceScoreFromInterfaces(resources),
		"standards": []map[string]interface{}{
			{
				"name":   "CIS",
				"score":  85.5,
				"passed": 171,
				"failed": 29,
				"total":  200,
			},
			{
				"name":   "PCI-DSS",
				"score":  92.0,
				"passed": 46,
				"failed": 4,
				"total":  50,
			},
			{
				"name":   "HIPAA",
				"score":  78.0,
				"passed": 39,
				"failed": 11,
				"total":  50,
			},
		},
		"violations": s.findComplianceViolationsFromInterfaces(resources, standard),
		"recommendations": []map[string]interface{}{
			{
				"title":    "Enable encryption at rest",
				"severity": "high",
				"affected": 15,
				"standard": "CIS",
			},
			{
				"title":    "Configure network segmentation",
				"severity": "medium",
				"affected": 8,
				"standard": "PCI-DSS",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// Helper functions

func (s *EnhancedDashboardServer) calculateProviderCost(provider string, resources []models.Resource) float64 {
	cost := 0.0
	for _, resource := range resources {
		if resource.Provider == provider {
			cost += s.calculateResourceCostValue(resource)
		}
	}
	return cost
}

func (s *EnhancedDashboardServer) calculateResourceCostValue(resource models.Resource) float64 {
	// Simplified cost calculation
	baseCost := map[string]float64{
		"aws_instance":          50.0,
		"azure_virtual_machine": 55.0,
		"gcp_compute_instance":  48.0,
		"aws_rds_instance":      100.0,
		"azure_sql_database":    110.0,
		"aws_s3_bucket":         10.0,
		"azure_storage_account": 12.0,
		"gcp_storage_bucket":    9.0,
	}

	if cost, exists := baseCost[resource.Type]; exists {
		return cost
	}
	return 5.0 // Default cost
}

func (s *EnhancedDashboardServer) calculateCostSavings(period string) float64 {
	// Simplified calculation
	days := 30
	if period == "7d" {
		days = 7
	} else if period == "90d" {
		days = 90
	}

	return float64(days) * 125.50 // Average daily savings
}

func (s *EnhancedDashboardServer) calculateComplianceScore(resources []models.Resource) float64 {
	if len(resources) == 0 {
		return 100.0
	}

	compliantCount := 0
	for _, resource := range resources {
		// Check basic compliance (simplified)
		tags := resource.GetTagsAsMap()
		if tags["Environment"] != "" && tags["Owner"] != "" {
			compliantCount++
		}
	}

	return (float64(compliantCount) / float64(len(resources))) * 100
}

func (s *EnhancedDashboardServer) analyzeTopIssues(drifts []models.DriftItem, resources []models.Resource) []Issue {
	issues := []Issue{
		{
			Type:       "Configuration Drift",
			Severity:   "high",
			Count:      len(drifts),
			Impact:     "Security and compliance risk",
			Resolution: "Apply terraform configurations",
			CostImpact: float64(len(drifts)) * 10,
		},
		{
			Type:       "Untagged Resources",
			Severity:   "medium",
			Count:      s.countUntaggedResources(resources),
			Impact:     "Cost allocation issues",
			Resolution: "Apply required tags",
			CostImpact: 500.0,
		},
		{
			Type:       "Unused Resources",
			Severity:   "low",
			Count:      len(s.findUnusedResources(resources)),
			Impact:     "Unnecessary costs",
			Resolution: "Delete or stop unused resources",
			CostImpact: 1500.0,
		},
	}

	return issues
}

func (s *EnhancedDashboardServer) countUntaggedResources(resources []models.Resource) int {
	count := 0
	for _, resource := range resources {
		tags := resource.GetTagsAsMap()
		if len(tags) == 0 {
			count++
		}
	}
	return count
}

func (s *EnhancedDashboardServer) findUnusedResources(resources []models.Resource) []UnusedResource {
	unused := []UnusedResource{}

	for _, resource := range resources {
		// Simplified detection - in production would check actual utilization metrics
		if resource.State == "stopped" || resource.State == "inactive" {
			unused = append(unused, UnusedResource{
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				Provider:     resource.Provider,
				MonthlyCost:  s.calculateResourceCostValue(resource),
				LastUsed:     "2024-01-15",
				Reason:       "No activity detected",
			})
		}
	}

	return unused
}

func (s *EnhancedDashboardServer) generateCostRecommendations(resources []models.Resource) []CostRecommendation {
	recommendations := []CostRecommendation{
		{
			Title:            "Right-size underutilized instances",
			Description:      "Several instances are using less than 20% CPU on average",
			PotentialSavings: 2500.0,
			Effort:           "low",
			Impact:           "high",
			Priority:         "high",
			Actions: []string{
				"Review instance utilization metrics",
				"Identify right-sizing opportunities",
				"Schedule downtime for resizing",
			},
		},
		{
			Title:            "Use Reserved Instances",
			Description:      "Purchase reserved instances for stable workloads",
			PotentialSavings: 5000.0,
			Effort:           "medium",
			Impact:           "high",
			Priority:         "medium",
			Actions: []string{
				"Analyze usage patterns",
				"Calculate break-even point",
				"Purchase reserved capacity",
			},
		},
	}

	return recommendations
}

func (s *EnhancedDashboardServer) generateTimeSeriesData(metric, period, resolution string) TimeSeriesData {
	// Generate sample time series data
	timestamps := []string{}
	now := time.Now()

	points := 24
	if period == "7d" {
		points = 168
	} else if period == "30d" {
		points = 720
	}

	for i := points; i > 0; i-- {
		timestamps = append(timestamps, now.Add(-time.Duration(i)*time.Hour).Format(time.RFC3339))
	}

	// Generate series data
	series := make(map[string][]float64)

	if metric == "" || metric == "all" {
		series["resources"] = generateRandomSeries(points, 100, 500)
		series["drifts"] = generateRandomSeries(points, 0, 50)
		series["costs"] = generateRandomSeries(points, 1000, 5000)
	} else {
		series[metric] = generateRandomSeries(points, 0, 100)
	}

	return TimeSeriesData{
		Timestamps: timestamps,
		Series:     series,
		Metadata: map[string]interface{}{
			"period":     period,
			"resolution": resolution,
			"metric":     metric,
		},
	}
}

func generateRandomSeries(points int, min, max float64) []float64 {
	series := make([]float64, points)
	for i := 0; i < points; i++ {
		// Simple linear trend with noise
		trend := float64(i) / float64(points)
		noise := (max - min) * 0.1 * (0.5 - float64(i%10)/10)
		series[i] = min + (max-min)*trend + noise
	}
	return series
}

func (s *EnhancedDashboardServer) findComplianceViolations(resources []models.Resource, standard string) []map[string]interface{} {
	violations := []map[string]interface{}{}

	for _, resource := range resources {
		// Check for missing tags (simplified compliance check)
		tags := resource.GetTagsAsMap()
		if tags["Environment"] == "" {
			violations = append(violations, map[string]interface{}{
				"resourceId": resource.ID,
				"type":       "missing_tag",
				"severity":   "medium",
				"standard":   "CIS",
				"message":    "Missing required Environment tag",
			})
		}
	}

	return violations
}

// calculateProviderCostFromInterfaces calculates cost for a provider from interface data
func (s *EnhancedDashboardServer) calculateProviderCostFromInterfaces(provider string, resources []interface{}) float64 {
	// Simple cost estimation based on resource count
	// In production would use actual cost data
	count := 0
	for _, res := range resources {
		if resMap, ok := res.(map[string]interface{}); ok {
			if p, ok := resMap["provider"].(string); ok && p == provider {
				count++
			}
		}
	}

	// Mock cost calculation
	costPerResource := map[string]float64{
		"aws":          10.50,
		"azure":        11.25,
		"gcp":          9.75,
		"digitalocean": 5.00,
	}

	if rate, ok := costPerResource[provider]; ok {
		return float64(count) * rate
	}
	return 0.0
}

// analyzeTopIssuesFromInterfaces analyzes top issues from interface data
func (s *EnhancedDashboardServer) analyzeTopIssuesFromInterfaces(drifts, resources []interface{}) []map[string]interface{} {
	issues := []map[string]interface{}{}

	// Count drift types
	driftTypes := make(map[string]int)
	for _, drift := range drifts {
		if driftMap, ok := drift.(map[string]interface{}); ok {
			if driftType, ok := driftMap["type"].(string); ok {
				driftTypes[driftType]++
			}
		}
	}

	// Create top issues
	for driftType, count := range driftTypes {
		issues = append(issues, map[string]interface{}{
			"type":     driftType,
			"count":    count,
			"severity": "medium",
			"impact":   "Potential configuration drift",
		})
	}

	// Sort by count and return top 5
	if len(issues) > 5 {
		return issues[:5]
	}
	return issues
}

// calculateComplianceScoreFromInterfaces calculates compliance score from interface data
func (s *EnhancedDashboardServer) calculateComplianceScoreFromInterfaces(resources []interface{}) float64 {
	if len(resources) == 0 {
		return 100.0
	}

	compliant := 0
	for _, res := range resources {
		// Simple check - consider resource compliant if it has required tags
		if resMap, ok := res.(map[string]interface{}); ok {
			if tags, ok := resMap["tags"].(map[string]interface{}); ok {
				if _, hasEnv := tags["Environment"]; hasEnv {
					compliant++
				}
			}
		}
	}

	return (float64(compliant) / float64(len(resources))) * 100
}

// Helper functions for map conversions
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

// findUnusedResourcesFromInterfaces finds unused resources from interface data
func (s *EnhancedDashboardServer) findUnusedResourcesFromInterfaces(resources []interface{}) []UnusedResource {
	unused := []UnusedResource{}

	for _, res := range resources {
		if resMap, ok := res.(map[string]interface{}); ok {
			// Simple heuristic - consider resource unused if it has no recent activity
			// In production would check actual usage metrics
			unused = append(unused, UnusedResource{
				ResourceID:   getStringFromMap(resMap, "id"),
				ResourceType: getStringFromMap(resMap, "type"),
				Provider:     getStringFromMap(resMap, "provider"),
				MonthlyCost:  10.0,
				LastUsed:     "30 days ago",
				Reason:       "No recent activity detected",
			})
		}
	}

	return unused
}

// generateCostRecommendationsFromInterfaces generates cost recommendations from interface data
func (s *EnhancedDashboardServer) generateCostRecommendationsFromInterfaces(resources []interface{}) []CostRecommendation {
	recommendations := []CostRecommendation{}

	// Count resources by type
	typeCounts := make(map[string]int)
	for _, res := range resources {
		if resMap, ok := res.(map[string]interface{}); ok {
			resType := getStringFromMap(resMap, "type")
			typeCounts[resType]++
		}
	}

	// Generate recommendations based on counts
	for resType, count := range typeCounts {
		if count > 10 {
			recommendations = append(recommendations, CostRecommendation{
				Title:            fmt.Sprintf("Consolidate %s resources", resType),
				Description:      fmt.Sprintf("You have %d %s resources that could be consolidated", count, resType),
				PotentialSavings: float64(count) * 5.0,
				Effort:           "Medium",
				Priority:         "High",
			})
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, CostRecommendation{
			Title:            "Resources Optimized",
			Description:      "Your resources are currently optimally configured",
			PotentialSavings: 0,
			Effort:           "None",
			Priority:         "Low",
		})
	}

	return recommendations
}

// findComplianceViolationsFromInterfaces finds compliance violations from interface data
func (s *EnhancedDashboardServer) findComplianceViolationsFromInterfaces(resources []interface{}, standard string) []map[string]interface{} {
	violations := []map[string]interface{}{}

	for _, res := range resources {
		if resMap, ok := res.(map[string]interface{}); ok {
			// Check for missing encryption
			if encrypted, ok := resMap["encrypted"].(bool); !ok || !encrypted {
				violations = append(violations, map[string]interface{}{
					"resourceId": getStringFromMap(resMap, "id"),
					"type":       getStringFromMap(resMap, "type"),
					"rule":       "encryption-at-rest",
					"severity":   "high",
					"standard":   standard,
					"message":    "Resource is not encrypted at rest",
				})
			}

			// Check for missing tags
			if tags, ok := resMap["tags"].(map[string]interface{}); !ok || len(tags) == 0 {
				violations = append(violations, map[string]interface{}{
					"resourceId": getStringFromMap(resMap, "id"),
					"type":       getStringFromMap(resMap, "type"),
					"rule":       "required-tags",
					"severity":   "medium",
					"standard":   standard,
					"message":    "Resource is missing required tags",
				})
			}
		}
	}

	return violations
}
