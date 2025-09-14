package cost

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// CostOptimizer provides advanced cost optimization capabilities
type CostOptimizer struct {
	analyzer        *CostAnalyzer
	recommendations map[string][]OptimizationRecommendation
	mu              sync.RWMutex
	config          *OptimizerConfig
	eventBus        EventBus
}

// OptimizerConfig contains configuration for the cost optimizer
type OptimizerConfig struct {
	EnableAutoOptimization bool          `json:"enable_auto_optimization"`
	OptimizationInterval   time.Duration `json:"optimization_interval"`
	MinSavingsThreshold    float64       `json:"min_savings_threshold"`
	MaxRiskLevel           string        `json:"max_risk_level"`
	ExcludedResourceTypes  []string      `json:"excluded_resource_types"`
	IncludedRegions        []string      `json:"included_regions"`
	DryRunMode             bool          `json:"dry_run_mode"`
}

// EventBus interface for cost optimization events
type EventBus interface {
	PublishCostEvent(event CostEvent) error
}

// CostEvent represents a cost-related event
type CostEvent struct {
	Type       string                 `json:"type"`
	ResourceID string                 `json:"resource_id"`
	Amount     float64                `json:"amount"`
	Currency   string                 `json:"currency"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// OptimizationStrategy represents a cost optimization strategy
type OptimizationStrategy struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Type        string                  `json:"type"`
	Priority    int                     `json:"priority"`
	Conditions  []OptimizationCondition `json:"conditions"`
	Actions     []OptimizationAction    `json:"actions"`
	Enabled     bool                    `json:"enabled"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// OptimizationCondition represents a condition for applying an optimization
type OptimizationCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type"`
}

// OptimizationAction represents an action to take for optimization
type OptimizationAction struct {
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
	RiskLevel   string                 `json:"risk_level"`
}

// CostOptimizationReport represents a cost optimization analysis report
type CostOptimizationReport struct {
	GeneratedAt      time.Time                    `json:"generated_at"`
	TotalResources   int                          `json:"total_resources"`
	Recommendations  []OptimizationRecommendation `json:"recommendations"`
	PotentialSavings float64                      `json:"potential_savings"`
	Currency         string                       `json:"currency"`
	Summary          map[string]interface{}       `json:"summary"`
}

// NewCostOptimizer creates a new cost optimizer
func NewCostOptimizer(analyzer *CostAnalyzer, eventBus EventBus) *CostOptimizer {
	config := &OptimizerConfig{
		EnableAutoOptimization: false,
		OptimizationInterval:   24 * time.Hour,
		MinSavingsThreshold:    10.0, // $10 minimum savings
		MaxRiskLevel:           "medium",
		ExcludedResourceTypes:  []string{},
		IncludedRegions:        []string{},
		DryRunMode:             true,
	}

	return &CostOptimizer{
		analyzer:        analyzer,
		recommendations: make(map[string][]OptimizationRecommendation),
		config:          config,
		eventBus:        eventBus,
	}
}

// AnalyzeCostOptimization analyzes resources for cost optimization opportunities
func (co *CostOptimizer) AnalyzeCostOptimization(ctx context.Context, resources []*models.Resource) (*CostOptimizationReport, error) {
	co.mu.Lock()
	defer co.mu.Unlock()

	report := &CostOptimizationReport{
		GeneratedAt:      time.Now(),
		TotalResources:   len(resources),
		Recommendations:  []OptimizationRecommendation{},
		PotentialSavings: 0.0,
		Currency:         "USD",
		Summary:          make(map[string]interface{}),
	}

	// Analyze each resource for optimization opportunities
	for _, resource := range resources {
		recommendations := co.analyzeResource(ctx, resource)
		report.Recommendations = append(report.Recommendations, recommendations...)
	}

	// Calculate total potential savings
	for _, rec := range report.Recommendations {
		report.PotentialSavings += rec.EstimatedSavings
	}

	// Sort recommendations by savings (highest first)
	sort.Slice(report.Recommendations, func(i, j int) bool {
		return report.Recommendations[i].EstimatedSavings > report.Recommendations[j].EstimatedSavings
	})

	// Generate summary statistics
	co.generateSummary(report)

	// Store recommendations
	co.recommendations["latest"] = report.Recommendations

	// Publish event
	if co.eventBus != nil {
		event := CostEvent{
			Type:      "optimization_analysis_completed",
			Amount:    report.PotentialSavings,
			Currency:  report.Currency,
			Message:   fmt.Sprintf("Cost optimization analysis completed. Potential savings: $%.2f", report.PotentialSavings),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"total_resources":   report.TotalResources,
				"recommendations":   len(report.Recommendations),
				"potential_savings": report.PotentialSavings,
			},
		}
		_ = co.eventBus.PublishCostEvent(event)
	}

	return report, nil
}

// analyzeResource analyzes a single resource for optimization opportunities
func (co *CostOptimizer) analyzeResource(ctx context.Context, resource *models.Resource) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	// Check if resource type is excluded
	if co.isResourceTypeExcluded(resource.Type) {
		return recommendations
	}

	// Check if region is included
	if !co.isRegionIncluded(resource.Region) {
		return recommendations
	}

	// Analyze based on resource type
	switch {
	case co.isComputeInstance(resource):
		recommendations = append(recommendations, co.analyzeComputeInstance(ctx, resource)...)
	case co.isStorageResource(resource):
		recommendations = append(recommendations, co.analyzeStorageResource(ctx, resource)...)
	case co.isDatabaseResource(resource):
		recommendations = append(recommendations, co.analyzeDatabaseResource(ctx, resource)...)
	case co.isLoadBalancer(resource):
		recommendations = append(recommendations, co.analyzeLoadBalancer(ctx, resource)...)
	default:
		recommendations = append(recommendations, co.analyzeGenericResource(ctx, resource)...)
	}

	return recommendations
}

// isComputeInstance checks if the resource is a compute instance
func (co *CostOptimizer) isComputeInstance(resource *models.Resource) bool {
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
func (co *CostOptimizer) isStorageResource(resource *models.Resource) bool {
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
func (co *CostOptimizer) isDatabaseResource(resource *models.Resource) bool {
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
func (co *CostOptimizer) isLoadBalancer(resource *models.Resource) bool {
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

// analyzeComputeInstance analyzes compute instances for optimization opportunities
func (co *CostOptimizer) analyzeComputeInstance(ctx context.Context, resource *models.Resource) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	// Check for oversized instances
	if instanceType, ok := resource.Attributes["instance_type"].(string); ok {
		if co.isOversizedInstance(instanceType) {
			recommendations = append(recommendations, OptimizationRecommendation{
				ResourceAddress:    resource.ID,
				RecommendationType: "rightsize_instance",
				Description:        fmt.Sprintf("Instance %s appears to be oversized. Consider downsizing to a smaller instance type.", instanceType),
				EstimatedSavings:   50.0, // Estimated monthly savings
				Impact:             "medium",
				Confidence:         0.8,
			})
		}
	}

	// Check for unused instances
	if state, ok := resource.Attributes["state"].(string); ok {
		if state == "stopped" {
			recommendations = append(recommendations, OptimizationRecommendation{
				ResourceAddress:    resource.ID,
				RecommendationType: "terminate_unused",
				Description:        "Instance is stopped and may be unused. Consider terminating if no longer needed.",
				EstimatedSavings:   100.0, // Estimated monthly savings
				Impact:             "high",
				Confidence:         0.9,
			})
		}
	}

	// Check for missing auto-scaling
	if _, ok := resource.Attributes["auto_scaling"]; !ok {
		recommendations = append(recommendations, OptimizationRecommendation{
			ResourceAddress:    resource.ID,
			RecommendationType: "enable_auto_scaling",
			Description:        "Enable auto-scaling to automatically adjust capacity based on demand.",
			EstimatedSavings:   30.0, // Estimated monthly savings
			Impact:             "medium",
			Confidence:         0.7,
		})
	}

	return recommendations
}

// analyzeStorageResource analyzes storage resources for optimization opportunities
func (co *CostOptimizer) analyzeStorageResource(ctx context.Context, resource *models.Resource) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	// Check for unused storage
	if size, ok := resource.Attributes["size"].(int); ok {
		if size > 1000 { // Large storage
			recommendations = append(recommendations, OptimizationRecommendation{
				ResourceAddress:    resource.ID,
				RecommendationType: "optimize_storage",
				Description:        fmt.Sprintf("Large storage volume (%d GB) detected. Consider optimizing storage class or reducing size.", size),
				EstimatedSavings:   25.0, // Estimated monthly savings
				Impact:             "medium",
				Confidence:         0.6,
			})
		}
	}

	// Check for missing lifecycle policies
	if _, ok := resource.Attributes["lifecycle_rule"]; !ok {
		recommendations = append(recommendations, OptimizationRecommendation{
			ResourceAddress:    resource.ID,
			RecommendationType: "add_lifecycle_policy",
			Description:        "Add lifecycle policies to automatically transition or delete old data.",
			EstimatedSavings:   15.0, // Estimated monthly savings
			Impact:             "low",
			Confidence:         0.8,
		})
	}

	return recommendations
}

// analyzeDatabaseResource analyzes database resources for optimization opportunities
func (co *CostOptimizer) analyzeDatabaseResource(ctx context.Context, resource *models.Resource) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	// Check for oversized databases
	if instanceClass, ok := resource.Attributes["instance_class"].(string); ok {
		if co.isOversizedDatabase(instanceClass) {
			recommendations = append(recommendations, OptimizationRecommendation{
				ResourceAddress:    resource.ID,
				RecommendationType: "rightsize_database",
				Description:        fmt.Sprintf("Database instance class %s appears to be oversized.", instanceClass),
				EstimatedSavings:   75.0, // Estimated monthly savings
				Impact:             "medium",
				Confidence:         0.7,
			})
		}
	}

	// Check for missing backup optimization
	if backupRetention, ok := resource.Attributes["backup_retention_period"].(int); ok {
		if backupRetention > 30 {
			recommendations = append(recommendations, OptimizationRecommendation{
				ResourceAddress:    resource.ID,
				RecommendationType: "optimize_backup_retention",
				Description:        fmt.Sprintf("Backup retention period (%d days) is longer than necessary.", backupRetention),
				EstimatedSavings:   20.0, // Estimated monthly savings
				Impact:             "low",
				Confidence:         0.8,
			})
		}
	}

	return recommendations
}

// analyzeLoadBalancer analyzes load balancers for optimization opportunities
func (co *CostOptimizer) analyzeLoadBalancer(ctx context.Context, resource *models.Resource) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	// Check for unused load balancers
	if scheme, ok := resource.Attributes["scheme"].(string); ok {
		if scheme == "internal" {
			recommendations = append(recommendations, OptimizationRecommendation{
				ResourceAddress:    resource.ID,
				RecommendationType: "review_load_balancer",
				Description:        "Internal load balancer detected. Verify if it's still needed.",
				EstimatedSavings:   40.0, // Estimated monthly savings
				Impact:             "medium",
				Confidence:         0.6,
			})
		}
	}

	return recommendations
}

// analyzeGenericResource analyzes generic resources for optimization opportunities
func (co *CostOptimizer) analyzeGenericResource(ctx context.Context, resource *models.Resource) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	// Check for missing tags (cost allocation)
	if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
		if len(tags) == 0 {
			recommendations = append(recommendations, OptimizationRecommendation{
				ResourceAddress:    resource.ID,
				RecommendationType: "add_cost_allocation_tags",
				Description:        "Add tags for better cost allocation and tracking.",
				EstimatedSavings:   0.0, // No direct savings, but helps with cost management
				Impact:             "low",
				Confidence:         1.0,
			})
		}
	}

	return recommendations
}

// Helper methods

// isResourceTypeExcluded checks if a resource type is excluded from optimization
func (co *CostOptimizer) isResourceTypeExcluded(resourceType string) bool {
	for _, excluded := range co.config.ExcludedResourceTypes {
		if resourceType == excluded {
			return true
		}
	}
	return false
}

// isRegionIncluded checks if a region is included in optimization
func (co *CostOptimizer) isRegionIncluded(region string) bool {
	if len(co.config.IncludedRegions) == 0 {
		return true // Include all regions if none specified
	}

	for _, included := range co.config.IncludedRegions {
		if region == included {
			return true
		}
	}
	return false
}

// isOversizedInstance checks if an instance type is oversized
func (co *CostOptimizer) isOversizedInstance(instanceType string) bool {
	// This is a simplified check - in reality, you'd analyze actual usage metrics
	oversizedTypes := []string{
		"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m5.16xlarge", "m5.24xlarge",
		"c5.2xlarge", "c5.4xlarge", "c5.9xlarge", "c5.12xlarge", "c5.18xlarge", "c5.24xlarge",
		"r5.2xlarge", "r5.4xlarge", "r5.8xlarge", "r5.12xlarge", "r5.16xlarge", "r5.24xlarge",
	}

	for _, t := range oversizedTypes {
		if instanceType == t {
			return true
		}
	}
	return false
}

// isOversizedDatabase checks if a database instance class is oversized
func (co *CostOptimizer) isOversizedDatabase(instanceClass string) bool {
	// This is a simplified check - in reality, you'd analyze actual usage metrics
	oversizedClasses := []string{
		"db.r5.2xlarge", "db.r5.4xlarge", "db.r5.8xlarge", "db.r5.12xlarge", "db.r5.16xlarge", "db.r5.24xlarge",
		"db.m5.2xlarge", "db.m5.4xlarge", "db.m5.8xlarge", "db.m5.12xlarge", "db.m5.16xlarge", "db.m5.24xlarge",
	}

	for _, c := range oversizedClasses {
		if instanceClass == c {
			return true
		}
	}
	return false
}

// generateSummary generates summary statistics for the optimization report
func (co *CostOptimizer) generateSummary(report *CostOptimizationReport) {
	// Count recommendations by type
	typeCounts := make(map[string]int)
	impactCounts := make(map[string]int)
	totalConfidence := 0.0

	for _, rec := range report.Recommendations {
		typeCounts[rec.RecommendationType]++
		impactCounts[rec.Impact]++
		totalConfidence += rec.Confidence
	}

	report.Summary["recommendation_types"] = typeCounts
	report.Summary["impact_distribution"] = impactCounts
	report.Summary["average_confidence"] = totalConfidence / float64(len(report.Recommendations))
	report.Summary["high_impact_recommendations"] = impactCounts["high"]
	report.Summary["medium_impact_recommendations"] = impactCounts["medium"]
	report.Summary["low_impact_recommendations"] = impactCounts["low"]
}

// GetOptimizationTrends returns cost optimization trends over time
func (co *CostOptimizer) GetOptimizationTrends(timeRange time.Duration) ([]TrendPoint, error) {
	// This would typically query a time-series database
	// For now, return mock data
	trends := []TrendPoint{
		{
			Timestamp:   time.Now().Add(-7 * 24 * time.Hour),
			MonthlyCost: 5000.0,
		},
		{
			Timestamp:   time.Now().Add(-6 * 24 * time.Hour),
			MonthlyCost: 4800.0,
		},
		{
			Timestamp:   time.Now().Add(-5 * 24 * time.Hour),
			MonthlyCost: 4600.0,
		},
		{
			Timestamp:   time.Now().Add(-4 * 24 * time.Hour),
			MonthlyCost: 4400.0,
		},
		{
			Timestamp:   time.Now().Add(-3 * 24 * time.Hour),
			MonthlyCost: 4200.0,
		},
		{
			Timestamp:   time.Now().Add(-2 * 24 * time.Hour),
			MonthlyCost: 4000.0,
		},
		{
			Timestamp:   time.Now().Add(-1 * 24 * time.Hour),
			MonthlyCost: 3800.0,
		},
		{
			Timestamp:   time.Now(),
			MonthlyCost: 3600.0,
		},
	}

	return trends, nil
}

// SetConfig updates the optimizer configuration
func (co *CostOptimizer) SetConfig(config *OptimizerConfig) {
	co.mu.Lock()
	defer co.mu.Unlock()
	co.config = config
}

// GetConfig returns the current optimizer configuration
func (co *CostOptimizer) GetConfig() *OptimizerConfig {
	co.mu.RLock()
	defer co.mu.RUnlock()
	return co.config
}

// GetLatestRecommendations returns the latest optimization recommendations
func (co *CostOptimizer) GetLatestRecommendations() []OptimizationRecommendation {
	co.mu.RLock()
	defer co.mu.RUnlock()
	return co.recommendations["latest"]
}
