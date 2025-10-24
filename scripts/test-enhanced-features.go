package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/filtering"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/providers"
)

func main() {
	fmt.Println("Testing Enhanced Features...")

	// Create event bus
	eventBus := events.NewEventBus()

	// Create provider factory
	factory := providers.NewProviderFactory(map[string]interface{}{})

	// Create resource filter service
	resourceFilterService := filtering.NewResourceFilterService()

	// Create drift summary service (with mock repository)
	driftRepo := &MockDriftRepository{}
	driftSummaryConfig := drift.DriftSummaryConfig{
		TopResourcesLimit: 10,
		TrendDays:         7,
		SeverityWeights: map[string]int{
			"critical": 4,
			"high":     3,
			"medium":   2,
			"low":      1,
		},
		ImpactWeights: map[string]int{
			"high":    3,
			"medium":  2,
			"low":     1,
			"minimal": 0,
		},
	}
	driftSummaryService := drift.NewDriftSummaryService(driftRepo, eventBus, driftSummaryConfig)

	// Create importance filter service
	importanceConfig := filtering.ImportanceConfig{
		DefaultWeights: map[string]float64{
			"resource_type":   0.3,
			"drift_severity":  0.4,
			"drift_frequency": 0.2,
			"resource_age":    0.1,
		},
		ResourceTypeWeights: map[string]map[string]float64{
			"aws_s3_bucket": {"critical": 0.9, "high": 0.8, "medium": 0.6, "low": 0.4},
		},
		SeverityWeights: map[string]float64{
			"critical": 1.0,
			"high":     0.8,
			"medium":   0.6,
			"low":      0.4,
			"minimal":  0.2,
		},
		ImpactWeights: map[string]float64{
			"high":    1.0,
			"medium":  0.7,
			"low":     0.4,
			"minimal": 0.1,
		},
		TimeDecayFactor: 0.1,
		MaxScore:        1.0,
		MinScore:        0.0,
	}
	importanceFilterService := filtering.NewImportanceFilterService(importanceConfig, eventBus)

	// Test resource filtering
	fmt.Println("\n=== Testing Resource Filtering ===")
	testResourceFiltering(resourceFilterService, factory)

	// Test drift summary
	fmt.Println("\n=== Testing Drift Summary ===")
	testDriftSummary(driftSummaryService)

	// Test importance filtering
	fmt.Println("\n=== Testing Importance Filtering ===")
	testImportanceFiltering(importanceFilterService, factory)

	fmt.Println("\nEnhanced features testing completed!")
}

func testResourceFiltering(resourceFilterService *filtering.ResourceFilterService, factory *providers.ProviderFactory) {
	// Create AWS provider
	provider, err := factory.CreateProvider("aws")
	if err != nil {
		log.Printf("Failed to create AWS provider: %v", err)
		return
	}

	// Get resources
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resources, err := provider.DiscoverResources(ctx, "us-east-1")
	if err != nil {
		log.Printf("Failed to discover resources: %v", err)
		return
	}

	fmt.Printf("Discovered %d resources\n", len(resources))

	// Test basic filtering
	filter := filtering.ResourceFilter{
		Conditions: []filtering.FilterCondition{
			{
				Field:    "type",
				Operator: filtering.FilterOperatorContains,
				Value:    "s3",
			},
		},
		Logic: filtering.FilterLogicAnd,
	}

	filteredResources, err := resourceFilterService.FilterResources(resources, filter)
	if err != nil {
		log.Printf("Failed to filter resources: %v", err)
		return
	}

	fmt.Printf("Filtered to %d S3 resources\n", len(filteredResources))

	// Test complex filtering
	complexFilter := filtering.ResourceFilter{
		Conditions: []filtering.FilterCondition{
			{
				Field:    "provider",
				Operator: filtering.FilterOperatorEquals,
				Value:    "aws",
			},
			{
				Field:    "region",
				Operator: filtering.FilterOperatorEquals,
				Value:    "us-east-1",
			},
		},
		Logic: filtering.FilterLogicAnd,
		Limit: 5,
	}

	complexFiltered, err := resourceFilterService.FilterResources(resources, complexFilter)
	if err != nil {
		log.Printf("Failed to apply complex filter: %v", err)
		return
	}

	fmt.Printf("Complex filter returned %d resources\n", len(complexFiltered))
}

func testDriftSummary(driftSummaryService *drift.DriftSummaryService) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Generate drift summary
	summary, err := driftSummaryService.GenerateSummary(ctx, "aws", "us-east-1")
	if err != nil {
		log.Printf("Failed to generate drift summary: %v", err)
		return
	}

	fmt.Printf("Drift Summary for %s in %s:\n", summary.Provider, summary.Region)
	fmt.Printf("  Total Resources: %d\n", summary.TotalResources)
	fmt.Printf("  Drifted Resources: %d\n", summary.DriftedResources)
	fmt.Printf("  Compliance Rate: %.2f%%\n", summary.ComplianceRate)
	fmt.Printf("  Drift Percentage: %.2f%%\n", summary.DriftPercentage)

	// Get drift statistics
	stats, err := driftSummaryService.GetDriftStatistics(ctx, "aws", "us-east-1")
	if err != nil {
		log.Printf("Failed to get drift statistics: %v", err)
		return
	}

	fmt.Printf("Drift Statistics:\n")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}
}

func testImportanceFiltering(importanceFilterService *filtering.ImportanceFilterService, factory *providers.ProviderFactory) {
	// Create AWS provider
	provider, err := factory.CreateProvider("aws")
	if err != nil {
		log.Printf("Failed to create AWS provider: %v", err)
		return
	}

	// Get resources
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resources, err := provider.DiscoverResources(ctx, "us-east-1")
	if err != nil {
		log.Printf("Failed to discover resources: %v", err)
		return
	}

	// Calculate importance scores
	var drifts []models.DriftRecord
	scores, err := importanceFilterService.CalculateImportanceScores(ctx, resources, drifts)
	if err != nil {
		log.Printf("Failed to calculate importance scores: %v", err)
		return
	}

	fmt.Printf("Calculated importance scores for %d resources\n", len(scores))

	// Show top 5 most important resources
	topResources := importanceFilterService.GetTopImportantResources(scores, 5)
	fmt.Printf("Top 5 Most Important Resources:\n")
	for i, score := range topResources {
		fmt.Printf("  %d. %s (%s) - Score: %.3f, Level: %s\n",
			i+1, score.ResourceName, score.ResourceType, score.Score, score.ImportanceLevel)
	}

	// Get importance distribution
	distribution := importanceFilterService.GetImportanceDistribution(scores)
	fmt.Printf("Importance Distribution:\n")
	for level, count := range distribution {
		fmt.Printf("  %s: %d resources\n", level, count)
	}

	// Get importance statistics
	stats := importanceFilterService.GetImportanceStatistics(scores)
	fmt.Printf("Importance Statistics:\n")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}
}

// MockDriftRepository is a mock implementation for testing
type MockDriftRepository struct{}

func (m *MockDriftRepository) GetDriftsByProviderAndRegion(ctx context.Context, provider, region string) ([]models.DriftRecord, error) {
	// Return mock drift records
	return []models.DriftRecord{
		{
			ID:           "drift-1",
			ResourceID:   "resource-1",
			ResourceName: "test-bucket",
			ResourceType: "aws_s3_bucket",
			Provider:     provider,
			Region:       region,
			DriftType:    "configuration",
			Severity:     "high",
			Status:       "active",
			DetectedAt:   time.Now().Add(-24 * time.Hour),
			Description:  "S3 bucket has incorrect encryption settings",
		},
		{
			ID:           "drift-2",
			ResourceID:   "resource-2",
			ResourceName: "test-instance",
			ResourceType: "aws_ec2_instance",
			Provider:     provider,
			Region:       region,
			DriftType:    "state",
			Severity:     "medium",
			Status:       "active",
			DetectedAt:   time.Now().Add(-12 * time.Hour),
			Description:  "EC2 instance has incorrect security group",
		},
	}, nil
}
