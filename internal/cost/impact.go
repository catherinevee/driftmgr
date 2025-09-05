package cost

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/state"
)

// ResourceCostImpact represents cost impact of a resource with drift
type ResourceCostImpact struct {
	ResourceID          string
	ResourceType        string
	ResourceName        string
	CurrentMonthlyCost  float64
	DriftMonthlyCost    float64
	PotentialSavings    float64
	Reason              string
	Recommendations     []string
}

// Recommendation represents a cost optimization recommendation
type Recommendation struct {
	ID               string
	Priority         string
	Description      string
	EstimatedSavings float64
	Implementation   string
	Risk             string
}

// GetRecommendations generates cost optimization recommendations
func (ca *CostAnalyzer) GetRecommendations(impacts []*ResourceCostImpact) []Recommendation {
	var recommendations []Recommendation

	// Analyze overall cost patterns
	totalCurrentCost := 0.0
	totalDriftCost := 0.0
	for _, impact := range impacts {
		totalCurrentCost += impact.CurrentMonthlyCost
		totalDriftCost += impact.DriftMonthlyCost
	}

	// Generate recommendations based on analysis
	if totalDriftCost > totalCurrentCost*1.1 {
		recommendations = append(recommendations, Recommendation{
			ID:               "COST_001",
			Priority:         "HIGH",
			Description:      "Review and optimize oversized resources",
			EstimatedSavings: totalDriftCost - totalCurrentCost,
			Implementation:   "Right-size instances based on actual utilization",
			Risk:             "LOW",
		})
	}

	// Check for idle resources
	for _, impact := range impacts {
		if isIdleResource(impact) {
			recommendations = append(recommendations, Recommendation{
				ID:               fmt.Sprintf("IDLE_%s", impact.ResourceID),
				Priority:         "MEDIUM",
				Description:      fmt.Sprintf("Consider removing idle resource: %s", impact.ResourceName),
				EstimatedSavings: impact.CurrentMonthlyCost,
				Implementation:   "Remove or stop unused resources",
				Risk:             "MEDIUM",
			})
		}
	}

	// Reserved instance recommendations
	if shouldRecommendReservedInstances(impacts) {
		savingsEstimate := totalCurrentCost * 0.3 // Typical 30% savings
		recommendations = append(recommendations, Recommendation{
			ID:               "RI_001",
			Priority:         "MEDIUM",
			Description:      "Consider Reserved Instances for long-running workloads",
			EstimatedSavings: savingsEstimate,
			Implementation:   "Purchase 1-year or 3-year Reserved Instances",
			Risk:             "LOW",
		})
	}

	// Spot instance recommendations
	if shouldRecommendSpotInstances(impacts) {
		savingsEstimate := totalCurrentCost * 0.5 // Typical 50% savings
		recommendations = append(recommendations, Recommendation{
			ID:               "SPOT_001",
			Priority:         "LOW",
			Description:      "Consider Spot Instances for fault-tolerant workloads",
			EstimatedSavings: savingsEstimate,
			Implementation:   "Use Spot Instances for batch processing and dev/test",
			Risk:             "HIGH",
		})
	}

	// Storage optimization
	storageOptimization := analyzeStorageOptimization(impacts)
	if storageOptimization.EstimatedSavings > 0 {
		recommendations = append(recommendations, storageOptimization)
	}

	// Database optimization
	dbOptimization := analyzeDatabaseOptimization(impacts)
	if dbOptimization.EstimatedSavings > 0 {
		recommendations = append(recommendations, dbOptimization)
	}

	return recommendations
}

// AnalyzeResourceSimple provides a simplified cost impact analysis for a resource
func (ca *CostAnalyzer) AnalyzeResourceSimple(resource *state.Resource) *ResourceCostImpact {
	// Generate a resource ID from type and name
	resourceID := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
	if resource.Module != "" {
		resourceID = fmt.Sprintf("module.%s.%s", resource.Module, resourceID)
	}
	
	impact := &ResourceCostImpact{
		ResourceID:   resourceID,
		ResourceType: resource.Type,
		ResourceName: resource.Name,
	}

	// Calculate estimated costs based on resource type
	impact.CurrentMonthlyCost = estimateResourceCost(resource)
	impact.DriftMonthlyCost = estimateDriftCost(resource)
	impact.PotentialSavings = calculatePotentialSavings(resource)

	// Add reason for cost change
	if impact.DriftMonthlyCost > impact.CurrentMonthlyCost {
		impact.Reason = analyzeCostIncrease(resource)
	} else if impact.DriftMonthlyCost < impact.CurrentMonthlyCost {
		impact.Reason = "Resource optimization detected"
	}

	return impact
}

// Helper functions

func isIdleResource(impact *ResourceCostImpact) bool {
	// Check if resource appears to be idle based on type and cost
	if impact.CurrentMonthlyCost > 10 && strings.Contains(impact.ResourceType, "instance") {
		// In production, would check CloudWatch metrics for utilization
		// For now, use heuristic based on resource name
		lowerName := strings.ToLower(impact.ResourceName)
		if strings.Contains(lowerName, "unused") || 
		   strings.Contains(lowerName, "idle") ||
		   strings.Contains(lowerName, "temp") {
			return true
		}
	}
	return false
}

func shouldRecommendReservedInstances(impacts []*ResourceCostImpact) bool {
	// Check if there are long-running instances that would benefit from RIs
	instanceCount := 0
	totalInstanceCost := 0.0
	
	for _, impact := range impacts {
		if strings.Contains(impact.ResourceType, "instance") {
			instanceCount++
			totalInstanceCost += impact.CurrentMonthlyCost
		}
	}
	
	// Recommend RIs if significant instance spend
	return instanceCount > 5 && totalInstanceCost > 1000
}

func shouldRecommendSpotInstances(impacts []*ResourceCostImpact) bool {
	// Check for workloads suitable for spot instances
	for _, impact := range impacts {
		if strings.Contains(impact.ResourceType, "instance") {
			// Check for batch, dev, or test workloads
			lowerName := strings.ToLower(impact.ResourceName)
			if strings.Contains(lowerName, "batch") ||
				strings.Contains(lowerName, "dev") ||
				strings.Contains(lowerName, "test") ||
				strings.Contains(lowerName, "staging") {
				return true
			}
		}
	}
	return false
}

func analyzeStorageOptimization(impacts []*ResourceCostImpact) Recommendation {
	totalStorageCost := 0.0
	storageResources := 0
	
	for _, impact := range impacts {
		if strings.Contains(impact.ResourceType, "storage") || 
		   strings.Contains(impact.ResourceType, "volume") ||
		   strings.Contains(impact.ResourceType, "bucket") {
			storageResources++
			totalStorageCost += impact.CurrentMonthlyCost
		}
	}
	
	if storageResources > 0 && totalStorageCost > 100 {
		return Recommendation{
			ID:               "STORAGE_001",
			Priority:         "MEDIUM",
			Description:      "Optimize storage tiers and lifecycle policies",
			EstimatedSavings: totalStorageCost * 0.25, // Estimate 25% savings
			Implementation:   "Move infrequently accessed data to cheaper storage tiers",
			Risk:             "LOW",
		}
	}
	
	return Recommendation{}
}

func analyzeDatabaseOptimization(impacts []*ResourceCostImpact) Recommendation {
	totalDBCost := 0.0
	dbResources := 0
	
	for _, impact := range impacts {
		if strings.Contains(impact.ResourceType, "db") || 
		   strings.Contains(impact.ResourceType, "database") ||
		   strings.Contains(impact.ResourceType, "rds") {
			dbResources++
			totalDBCost += impact.CurrentMonthlyCost
		}
	}
	
	if dbResources > 0 && totalDBCost > 500 {
		return Recommendation{
			ID:               "DB_001",
			Priority:         "HIGH",
			Description:      "Optimize database instances and storage",
			EstimatedSavings: totalDBCost * 0.3, // Estimate 30% savings
			Implementation:   "Right-size DB instances, enable auto-pause for dev/test",
			Risk:             "MEDIUM",
		}
	}
	
	return Recommendation{}
}

func estimateResourceCost(resource *state.Resource) float64 {
	// Base cost estimation based on resource type
	baseCosts := map[string]float64{
		"aws_instance":            50.0,
		"aws_db_instance":         100.0,
		"aws_s3_bucket":           5.0,
		"aws_ebs_volume":          10.0,
		"azurerm_virtual_machine": 60.0,
		"azurerm_storage_account": 20.0,
		"google_compute_instance": 55.0,
		"google_storage_bucket":   8.0,
	}
	
	baseCost := 10.0 // Default
	for resourceType, cost := range baseCosts {
		if strings.HasPrefix(resource.Type, resourceType) {
			baseCost = cost
			break
		}
	}
	
	// Adjust based on instance attributes if available
	if len(resource.Instances) > 0 {
		instance := resource.Instances[0]
		
		// Check instance type for compute resources
		if instanceType, ok := instance.Attributes["instance_type"].(string); ok {
			baseCost = getInstanceTypeCost(instanceType)
		}
		
		// Check storage size
		if size, ok := instance.Attributes["size"].(float64); ok {
			if strings.Contains(resource.Type, "volume") || strings.Contains(resource.Type, "storage") {
				baseCost = size * 0.1 // $0.10 per GB
			}
		}
	}
	
	return baseCost
}

func estimateDriftCost(resource *state.Resource) float64 {
	currentCost := estimateResourceCost(resource)
	
	// Simulate drift impact
	driftMultiplier := 1.0
	
	if len(resource.Instances) > 0 {
		instance := resource.Instances[0]
		
		// Check for common drift patterns
		if _, ok := instance.Attributes["instance_type"].(string); ok {
			driftMultiplier *= 1.2 // Assume 20% increase due to upsizing
		}
		
		if _, ok := instance.Attributes["size"].(float64); ok {
			driftMultiplier *= 1.1 // Assume 10% increase in storage
		}
		
		if monitoring, ok := instance.Attributes["monitoring"].(bool); ok && monitoring {
			driftMultiplier *= 1.05 // 5% increase for enhanced monitoring
		}
		
		if encryption, ok := instance.Attributes["encryption"].(bool); ok && encryption {
			driftMultiplier *= 1.02 // 2% increase for encryption
		}
	}
	
	return currentCost * driftMultiplier
}

func calculatePotentialSavings(resource *state.Resource) float64 {
	currentCost := estimateResourceCost(resource)
	
	// Calculate potential optimized cost
	optimizedCost := currentCost
	
	// Right-sizing optimization
	if strings.Contains(resource.Type, "instance") {
		optimizedCost *= 0.8 // Assume 20% reduction through right-sizing
	}
	
	// Storage optimization
	if strings.Contains(resource.Type, "storage") || strings.Contains(resource.Type, "volume") {
		optimizedCost *= 0.85 // Assume 15% reduction through storage optimization
	}
	
	// Database optimization
	if strings.Contains(resource.Type, "db") || strings.Contains(resource.Type, "database") {
		optimizedCost *= 0.7 // Assume 30% reduction through DB optimization
	}
	
	if optimizedCost < currentCost {
		return currentCost - optimizedCost
	}
	
	return 0.0
}

func analyzeCostIncrease(resource *state.Resource) string {
	reasons := []string{}
	
	// Check for common cost increase patterns
	if strings.Contains(resource.Type, "instance") {
		reasons = append(reasons, "Instance type may have been upsized")
	}
	
	if strings.Contains(resource.Type, "storage") || strings.Contains(resource.Type, "volume") {
		reasons = append(reasons, "Storage capacity increased")
	}
	
	if len(resource.Instances) > 0 {
		instance := resource.Instances[0]
		
		if _, ok := instance.Attributes["multi_az"].(bool); ok {
			reasons = append(reasons, "Multi-AZ deployment enabled")
		}
		
		if _, ok := instance.Attributes["encryption"].(bool); ok {
			reasons = append(reasons, "Encryption enabled")
		}
		
		if _, ok := instance.Attributes["backup_retention_period"].(float64); ok {
			reasons = append(reasons, "Backup retention increased")
		}
	}
	
	if len(reasons) > 0 {
		return strings.Join(reasons, "; ")
	}
	
	return "Configuration drift detected"
}

func getInstanceTypeCost(instanceType string) float64 {
	// Monthly costs for common instance types
	costs := map[string]float64{
		"t2.micro":    8.50,
		"t2.small":    17.00,
		"t2.medium":   34.00,
		"t2.large":    68.00,
		"t3.micro":    7.50,
		"t3.small":    15.00,
		"t3.medium":   30.00,
		"t3.large":    60.00,
		"m5.large":    70.00,
		"m5.xlarge":   140.00,
		"m5.2xlarge":  280.00,
		"c5.large":    62.00,
		"c5.xlarge":   124.00,
		"r5.large":    92.00,
		"r5.xlarge":   184.00,
	}
	
	if cost, ok := costs[instanceType]; ok {
		return cost
	}
	
	// Default based on size indicators
	if strings.Contains(instanceType, "micro") {
		return 10.0
	} else if strings.Contains(instanceType, "small") {
		return 20.0
	} else if strings.Contains(instanceType, "medium") {
		return 40.0
	} else if strings.Contains(instanceType, "large") {
		return 80.0
	} else if strings.Contains(instanceType, "xlarge") {
		return 160.0
	}
	
	return 50.0 // Default monthly cost
}