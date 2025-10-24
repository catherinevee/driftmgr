package drift

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/models"
)

// DriftRepository defines the interface for drift data access
type DriftRepository interface {
	GetDriftsByProviderAndRegion(ctx context.Context, provider, region string) ([]models.DriftRecord, error)
}

// DriftSummary represents a summary of drift detection results
type DriftSummary struct {
	ID                  string                   `json:"id"`
	Provider            string                   `json:"provider"`
	Region              string                   `json:"region"`
	TotalResources      int                      `json:"total_resources"`
	DriftedResources    int                      `json:"drifted_resources"`
	CompliantResources  int                      `json:"compliant_resources"`
	DriftPercentage     float64                  `json:"drift_percentage"`
	ComplianceRate      float64                  `json:"compliance_rate"`
	DriftTypes          map[string]int           `json:"drift_types"`
	ResourceTypes       map[string]int           `json:"resource_types"`
	SeverityLevels      map[string]int           `json:"severity_levels"`
	TopDriftedResources []DriftedResourceSummary `json:"top_drifted_resources"`
	DriftTrends         []DriftTrend             `json:"drift_trends"`
	GeneratedAt         time.Time                `json:"generated_at"`
	GeneratedBy         string                   `json:"generated_by"`
	Metadata            map[string]interface{}   `json:"metadata"`
}

// DriftedResourceSummary represents a summary of a drifted resource
type DriftedResourceSummary struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	DriftCount   int                    `json:"drift_count"`
	Severity     string                 `json:"severity"`
	LastDetected time.Time              `json:"last_detected"`
	DriftTypes   []string               `json:"drift_types"`
	Impact       string                 `json:"impact"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// DriftTrend represents a trend in drift detection over time
type DriftTrend struct {
	Date             time.Time `json:"date"`
	TotalResources   int       `json:"total_resources"`
	DriftedResources int       `json:"drifted_resources"`
	DriftPercentage  float64   `json:"drift_percentage"`
	NewDrifts        int       `json:"new_drifts"`
	ResolvedDrifts   int       `json:"resolved_drifts"`
}

// DriftSummaryService provides drift summary functionality
type DriftSummaryService struct {
	driftRepo DriftRepository
	eventBus  *events.EventBus
	config    DriftSummaryConfig
}

// DriftSummaryConfig represents configuration for the drift summary service
type DriftSummaryConfig struct {
	TopResourcesLimit int            `json:"top_resources_limit"`
	TrendDays         int            `json:"trend_days"`
	SeverityWeights   map[string]int `json:"severity_weights"`
	ImpactWeights     map[string]int `json:"impact_weights"`
}

// NewDriftSummaryService creates a new drift summary service
func NewDriftSummaryService(driftRepo DriftRepository, eventBus *events.EventBus, config DriftSummaryConfig) *DriftSummaryService {
	return &DriftSummaryService{
		driftRepo: driftRepo,
		eventBus:  eventBus,
		config:    config,
	}
}

// GenerateSummary generates a drift summary for a specific provider and region
func (dss *DriftSummaryService) GenerateSummary(ctx context.Context, provider, region string) (*DriftSummary, error) {
	// Get all drift records for the provider and region
	drifts, err := dss.driftRepo.GetDriftsByProviderAndRegion(ctx, provider, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get drifts: %w", err)
	}

	// Generate summary
	summary := &DriftSummary{
		ID:             generateSummaryID(provider, region),
		Provider:       provider,
		Region:         region,
		GeneratedAt:    time.Now(),
		GeneratedBy:    "system",
		DriftTypes:     make(map[string]int),
		ResourceTypes:  make(map[string]int),
		SeverityLevels: make(map[string]int),
		Metadata:       make(map[string]interface{}),
	}

	// Calculate basic metrics
	dss.calculateBasicMetrics(summary, drifts)

	// Calculate drift types and resource types
	dss.calculateDriftTypes(summary, drifts)

	// Calculate severity levels
	dss.calculateSeverityLevels(summary, drifts)

	// Get top drifted resources
	dss.calculateTopDriftedResources(summary, drifts)

	// Calculate drift trends
	trends, err := dss.calculateDriftTrends(ctx, provider, region)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate drift trends: %w", err)
	}
	summary.DriftTrends = trends

	// Publish summary generated event
	dss.publishSummaryEvent(summary)

	return summary, nil
}

// GenerateMultiProviderSummary generates a drift summary across multiple providers
func (dss *DriftSummaryService) GenerateMultiProviderSummary(ctx context.Context, providers []string, region string) (*DriftSummary, error) {
	summary := &DriftSummary{
		ID:             generateSummaryID("multi", region),
		Provider:       "multi",
		Region:         region,
		GeneratedAt:    time.Now(),
		GeneratedBy:    "system",
		DriftTypes:     make(map[string]int),
		ResourceTypes:  make(map[string]int),
		SeverityLevels: make(map[string]int),
		Metadata:       make(map[string]interface{}),
	}

	var allDrifts []models.DriftRecord

	// Collect drifts from all providers
	for _, provider := range providers {
		drifts, err := dss.driftRepo.GetDriftsByProviderAndRegion(ctx, provider, region)
		if err != nil {
			return nil, fmt.Errorf("failed to get drifts for provider %s: %w", provider, err)
		}
		allDrifts = append(allDrifts, drifts...)
	}

	// Calculate metrics
	dss.calculateBasicMetrics(summary, allDrifts)
	dss.calculateDriftTypes(summary, allDrifts)
	dss.calculateSeverityLevels(summary, allDrifts)
	dss.calculateTopDriftedResources(summary, allDrifts)

	// Calculate trends across all providers
	trends, err := dss.calculateMultiProviderTrends(ctx, providers, region)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate multi-provider trends: %w", err)
	}
	summary.DriftTrends = trends

	// Publish summary generated event
	dss.publishSummaryEvent(summary)

	return summary, nil
}

// GetDriftTrends returns drift trends for a specific time period
func (dss *DriftSummaryService) GetDriftTrends(ctx context.Context, provider, region string, days int) ([]DriftTrend, error) {
	return dss.calculateDriftTrends(ctx, provider, region)
}

// GetDriftStatistics returns detailed drift statistics
func (dss *DriftSummaryService) GetDriftStatistics(ctx context.Context, provider, region string) (map[string]interface{}, error) {
	drifts, err := dss.driftRepo.GetDriftsByProviderAndRegion(ctx, provider, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get drifts: %w", err)
	}

	stats := make(map[string]interface{})

	// Basic statistics
	stats["total_drifts"] = len(drifts)
	stats["unique_resources"] = dss.getUniqueResourceCount(drifts)
	stats["average_drifts_per_resource"] = dss.getAverageDriftsPerResource(drifts)

	// Time-based statistics
	stats["drifts_last_24h"] = dss.getDriftsInTimeRange(drifts, 24*time.Hour)
	stats["drifts_last_7d"] = dss.getDriftsInTimeRange(drifts, 7*24*time.Hour)
	stats["drifts_last_30d"] = dss.getDriftsInTimeRange(drifts, 30*24*time.Hour)

	// Severity distribution
	stats["severity_distribution"] = dss.getSeverityDistribution(drifts)

	// Resource type distribution
	stats["resource_type_distribution"] = dss.getResourceTypeDistribution(drifts)

	// Drift type distribution
	stats["drift_type_distribution"] = dss.getDriftTypeDistribution(drifts)

	return stats, nil
}

// Helper methods

func (dss *DriftSummaryService) calculateBasicMetrics(summary *DriftSummary, drifts []models.DriftRecord) {
	summary.TotalResources = dss.getUniqueResourceCount(drifts)
	summary.DriftedResources = len(drifts)

	if summary.TotalResources > 0 {
		summary.DriftPercentage = float64(summary.DriftedResources) / float64(summary.TotalResources) * 100
		summary.ComplianceRate = 100 - summary.DriftPercentage
	}

	summary.CompliantResources = summary.TotalResources - summary.DriftedResources
}

func (dss *DriftSummaryService) calculateDriftTypes(summary *DriftSummary, drifts []models.DriftRecord) {
	for _, drift := range drifts {
		summary.DriftTypes[drift.DriftType]++
		summary.ResourceTypes[drift.ResourceType]++
	}
}

func (dss *DriftSummaryService) calculateSeverityLevels(summary *DriftSummary, drifts []models.DriftRecord) {
	for _, drift := range drifts {
		summary.SeverityLevels[drift.Severity]++
	}
}

func (dss *DriftSummaryService) calculateTopDriftedResources(summary *DriftSummary, drifts []models.DriftRecord) {
	resourceDriftCount := make(map[string]int)
	resourceDriftTypes := make(map[string][]string)
	resourceLastDetected := make(map[string]time.Time)

	// Count drifts per resource
	for _, drift := range drifts {
		resourceDriftCount[drift.ResourceID]++
		resourceDriftTypes[drift.ResourceID] = append(resourceDriftTypes[drift.ResourceID], drift.DriftType)
		if drift.DetectedAt.After(resourceLastDetected[drift.ResourceID]) {
			resourceLastDetected[drift.ResourceID] = drift.DetectedAt
		}
	}

	// Create drifted resource summaries
	var driftedResources []DriftedResourceSummary
	for resourceID, count := range resourceDriftCount {
		// Get the most recent drift for this resource
		var mostRecentDrift models.DriftRecord
		for _, drift := range drifts {
			if drift.ResourceID == resourceID {
				if drift.DetectedAt.After(mostRecentDrift.DetectedAt) {
					mostRecentDrift = drift
				}
			}
		}

		driftedResource := DriftedResourceSummary{
			ResourceID:   resourceID,
			ResourceName: mostRecentDrift.ResourceName,
			ResourceType: mostRecentDrift.ResourceType,
			Provider:     mostRecentDrift.Provider,
			Region:       mostRecentDrift.Region,
			DriftCount:   count,
			Severity:     mostRecentDrift.Severity,
			LastDetected: resourceLastDetected[resourceID],
			DriftTypes:   dss.getUniqueDriftTypes(resourceDriftTypes[resourceID]),
			Impact:       dss.calculateImpact(mostRecentDrift),
			Status:       mostRecentDrift.Status,
			Metadata:     make(map[string]interface{}),
		}

		driftedResources = append(driftedResources, driftedResource)
	}

	// Sort by drift count (descending)
	sort.Slice(driftedResources, func(i, j int) bool {
		return driftedResources[i].DriftCount > driftedResources[j].DriftCount
	})

	// Limit to top resources
	limit := dss.config.TopResourcesLimit
	if limit <= 0 {
		limit = 10
	}
	if len(driftedResources) > limit {
		driftedResources = driftedResources[:limit]
	}

	summary.TopDriftedResources = driftedResources
}

func (dss *DriftSummaryService) calculateDriftTrends(ctx context.Context, provider, region string) ([]DriftTrend, error) {
	// This is a simplified implementation
	// In a real implementation, you would query historical drift data
	trends := []DriftTrend{
		{
			Date:             time.Now().AddDate(0, 0, -7),
			TotalResources:   100,
			DriftedResources: 15,
			DriftPercentage:  15.0,
			NewDrifts:        5,
			ResolvedDrifts:   2,
		},
		{
			Date:             time.Now().AddDate(0, 0, -6),
			TotalResources:   105,
			DriftedResources: 18,
			DriftPercentage:  17.1,
			NewDrifts:        8,
			ResolvedDrifts:   5,
		},
		{
			Date:             time.Now().AddDate(0, 0, -5),
			TotalResources:   110,
			DriftedResources: 20,
			DriftPercentage:  18.2,
			NewDrifts:        7,
			ResolvedDrifts:   5,
		},
		{
			Date:             time.Now().AddDate(0, 0, -4),
			TotalResources:   115,
			DriftedResources: 22,
			DriftPercentage:  19.1,
			NewDrifts:        6,
			ResolvedDrifts:   4,
		},
		{
			Date:             time.Now().AddDate(0, 0, -3),
			TotalResources:   120,
			DriftedResources: 25,
			DriftPercentage:  20.8,
			NewDrifts:        8,
			ResolvedDrifts:   5,
		},
		{
			Date:             time.Now().AddDate(0, 0, -2),
			TotalResources:   125,
			DriftedResources: 28,
			DriftPercentage:  22.4,
			NewDrifts:        9,
			ResolvedDrifts:   6,
		},
		{
			Date:             time.Now().AddDate(0, 0, -1),
			TotalResources:   130,
			DriftedResources: 30,
			DriftPercentage:  23.1,
			NewDrifts:        7,
			ResolvedDrifts:   5,
		},
		{
			Date:             time.Now(),
			TotalResources:   135,
			DriftedResources: 32,
			DriftPercentage:  23.7,
			NewDrifts:        8,
			ResolvedDrifts:   6,
		},
	}

	return trends, nil
}

func (dss *DriftSummaryService) calculateMultiProviderTrends(ctx context.Context, providers []string, region string) ([]DriftTrend, error) {
	// This is a simplified implementation
	// In a real implementation, you would aggregate trends from all providers
	return dss.calculateDriftTrends(ctx, "multi", region)
}

func (dss *DriftSummaryService) getUniqueResourceCount(drifts []models.DriftRecord) int {
	uniqueResources := make(map[string]bool)
	for _, drift := range drifts {
		uniqueResources[drift.ResourceID] = true
	}
	return len(uniqueResources)
}

func (dss *DriftSummaryService) getAverageDriftsPerResource(drifts []models.DriftRecord) float64 {
	if len(drifts) == 0 {
		return 0
	}
	uniqueCount := dss.getUniqueResourceCount(drifts)
	return float64(len(drifts)) / float64(uniqueCount)
}

func (dss *DriftSummaryService) getDriftsInTimeRange(drifts []models.DriftRecord, duration time.Duration) int {
	cutoff := time.Now().Add(-duration)
	count := 0
	for _, drift := range drifts {
		if drift.DetectedAt.After(cutoff) {
			count++
		}
	}
	return count
}

func (dss *DriftSummaryService) getSeverityDistribution(drifts []models.DriftRecord) map[string]int {
	distribution := make(map[string]int)
	for _, drift := range drifts {
		distribution[drift.Severity]++
	}
	return distribution
}

func (dss *DriftSummaryService) getResourceTypeDistribution(drifts []models.DriftRecord) map[string]int {
	distribution := make(map[string]int)
	for _, drift := range drifts {
		distribution[drift.ResourceType]++
	}
	return distribution
}

func (dss *DriftSummaryService) getDriftTypeDistribution(drifts []models.DriftRecord) map[string]int {
	distribution := make(map[string]int)
	for _, drift := range drifts {
		distribution[drift.DriftType]++
	}
	return distribution
}

func (dss *DriftSummaryService) getUniqueDriftTypes(driftTypes []string) []string {
	unique := make(map[string]bool)
	var result []string
	for _, driftType := range driftTypes {
		if !unique[driftType] {
			unique[driftType] = true
			result = append(result, driftType)
		}
	}
	return result
}

func (dss *DriftSummaryService) calculateImpact(drift models.DriftRecord) string {
	// This is a simplified implementation
	// In a real implementation, you would calculate impact based on various factors
	switch drift.Severity {
	case "critical":
		return "high"
	case "high":
		return "medium"
	case "medium":
		return "low"
	case "low":
		return "minimal"
	default:
		return "unknown"
	}
}

func (dss *DriftSummaryService) publishSummaryEvent(summary *DriftSummary) {
	if dss.eventBus == nil {
		return
	}

	event := events.Event{
		Type:      events.EventType("drift.summary.generated"),
		Timestamp: time.Now(),
		Source:    "drift_summary_service",
		Data: map[string]interface{}{
			"summary_id":        summary.ID,
			"provider":          summary.Provider,
			"region":            summary.Region,
			"total_resources":   summary.TotalResources,
			"drifted_resources": summary.DriftedResources,
			"drift_percentage":  summary.DriftPercentage,
			"compliance_rate":   summary.ComplianceRate,
		},
	}

	dss.eventBus.Publish(event)
}

func generateSummaryID(provider, region string) string {
	return fmt.Sprintf("summary-%s-%s-%d", provider, region, time.Now().Unix())
}
