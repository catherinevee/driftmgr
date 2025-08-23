package cost

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// CostAnalyzer provides cost analysis for cloud resources
type CostAnalyzer struct {
	client *costexplorer.Client
}

// CostAnalysis represents cost data for a resource
type CostAnalysis struct {
	ResourceID       string
	MonthlyCost      float64
	DailyCost        float64
	CostTrend        CostTrend
	CostBreakdown    []CostBreakdown
	OptimizationTips []OptimizationTip
	LastUpdated      time.Time
}

// CostTrend shows cost movement over time
type CostTrend struct {
	Direction  string // "increasing", "decreasing", "stable"
	Percentage float64
	Period     string
}

// CostBreakdown shows cost by service/dimension
type CostBreakdown struct {
	Service    string
	Cost       float64
	Percentage float64
}

// OptimizationTip provides cost optimization recommendations
type OptimizationTip struct {
	Category         string
	Description      string
	PotentialSavings float64
	Difficulty       string // "easy", "medium", "hard"
}

// NewCostAnalyzer creates a new cost analyzer
func NewCostAnalyzer(client *costexplorer.Client) *CostAnalyzer {
	return &CostAnalyzer{
		client: client,
	}
}

// AnalyzeResourceCost analyzes cost for a specific resource
func (ca *CostAnalyzer) AnalyzeResourceCost(ctx context.Context, resource models.Resource) (*CostAnalysis, error) {
	// Get cost data for the last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	// Build cost explorer request
	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: awsString(startDate.Format("2006-01-02")),
			End:   awsString(endDate.Format("2006-01-02")),
		},
		Granularity: types.GranularityDaily,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  awsString("SERVICE"),
			},
		},
		Filter: &types.Expression{
			And: []types.Expression{
				{
					Dimensions: &types.DimensionValues{
						Key:    types.DimensionResourceId,
						Values: []string{resource.ID},
					},
				},
			},
		},
	}

	// Execute cost explorer query
	result, err := ca.client.GetCostAndUsage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost data: %w", err)
	}

	// Process results
	costAnalysis := &CostAnalysis{
		ResourceID:  resource.ID,
		LastUpdated: time.Now(),
	}

	// Calculate total costs
	var totalCost float64
	var costBreakdowns []CostBreakdown

	for _, result := range result.ResultsByTime {
		for _, group := range result.Groups {
			cost := parseCost(*group.Metrics["UnblendedCost"].Amount)
			totalCost += cost

			// Add to breakdown
			costBreakdowns = append(costBreakdowns, CostBreakdown{
				Service: group.Keys[0],
				Cost:    cost,
			})
		}
	}

	costAnalysis.MonthlyCost = totalCost
	costAnalysis.DailyCost = totalCost / 30
	costAnalysis.CostBreakdown = costBreakdowns

	// Calculate cost trend
	costAnalysis.CostTrend = ca.calculateCostTrend(result.ResultsByTime)

	// Generate optimization tips
	costAnalysis.OptimizationTips = ca.generateOptimizationTips(resource, costAnalysis)

	return costAnalysis, nil
}

// AnalyzeResourcesCost analyzes cost for multiple resources
func (ca *CostAnalyzer) AnalyzeResourcesCost(ctx context.Context, resources []models.Resource) (map[string]*CostAnalysis, error) {
	results := make(map[string]*CostAnalysis)

	// Process resources in batches to avoid API limits
	batchSize := 10
	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		for _, resource := range batch {
			costAnalysis, err := ca.AnalyzeResourceCost(ctx, resource)
			if err != nil {
				// Log error but continue with other resources
				fmt.Printf("Warning: Failed to analyze cost for resource %s: %v\n", resource.ID, err)
				continue
			}
			results[resource.ID] = costAnalysis
		}
	}

	return results, nil
}

// GetCostOptimizationRecommendations provides cost optimization recommendations
func (ca *CostAnalyzer) GetCostOptimizationRecommendations(ctx context.Context, resource models.Resource) ([]OptimizationTip, error) {
	var recommendations []OptimizationTip

	// Check for common optimization opportunities based on resource type
	switch resource.Type {
	case "ec2_instance":
		recommendations = append(recommendations, ca.getEC2OptimizationTips(resource)...)
	case "rds_instance":
		recommendations = append(recommendations, ca.getRDSOptimizationTips(resource)...)
	case "s3_bucket":
		recommendations = append(recommendations, ca.getS3OptimizationTips(resource)...)
	case "eks_cluster":
		recommendations = append(recommendations, ca.getEKSOptimizationTips(resource)...)
	}

	return recommendations, nil
}

// calculateCostTrend calculates the cost trend over time
func (ca *CostAnalyzer) calculateCostTrend(resultsByTime []types.ResultByTime) CostTrend {
	if len(resultsByTime) < 2 {
		return CostTrend{
			Direction: "stable",
			Period:    "30 days",
		}
	}

	// Calculate average cost for first and second half
	midPoint := len(resultsByTime) / 2
	var firstHalfCost, secondHalfCost float64

	for i := 0; i < midPoint; i++ {
		for _, group := range resultsByTime[i].Groups {
			firstHalfCost += parseCost(*group.Metrics["UnblendedCost"].Amount)
		}
	}

	for i := midPoint; i < len(resultsByTime); i++ {
		for _, group := range resultsByTime[i].Groups {
			secondHalfCost += parseCost(*group.Metrics["UnblendedCost"].Amount)
		}
	}

	// Calculate percentage change
	var percentage float64
	var direction string

	if firstHalfCost > 0 {
		percentage = ((secondHalfCost - firstHalfCost) / firstHalfCost) * 100
	}

	if percentage > 5 {
		direction = "increasing"
	} else if percentage < -5 {
		direction = "decreasing"
	} else {
		direction = "stable"
	}

	return CostTrend{
		Direction:  direction,
		Percentage: percentage,
		Period:     "30 days",
	}
}

// generateOptimizationTips generates optimization tips based on resource type and cost data
func (ca *CostAnalyzer) generateOptimizationTips(resource models.Resource, costAnalysis *CostAnalysis) []OptimizationTip {
	var tips []OptimizationTip

	// Add general tips based on cost
	if costAnalysis.MonthlyCost > 100 {
		tips = append(tips, OptimizationTip{
			Category:         "High Cost",
			Description:      "This resource has high monthly costs. Consider reviewing usage patterns.",
			PotentialSavings: costAnalysis.MonthlyCost * 0.2, // Assume 20% potential savings
			Difficulty:       "medium",
		})
	}

	// Add resource-specific tips
	switch resource.Type {
	case "ec2_instance":
		tips = append(tips, ca.getEC2OptimizationTips(resource)...)
	case "rds_instance":
		tips = append(tips, ca.getRDSOptimizationTips(resource)...)
	case "s3_bucket":
		tips = append(tips, ca.getS3OptimizationTips(resource)...)
	case "eks_cluster":
		tips = append(tips, ca.getEKSOptimizationTips(resource)...)
	}

	return tips
}

// getEC2OptimizationTips provides EC2-specific optimization tips
func (ca *CostAnalyzer) getEC2OptimizationTips(resource models.Resource) []OptimizationTip {
	return []OptimizationTip{
		{
			Category:         "EC2 Optimization",
			Description:      "Consider using Spot Instances for non-critical workloads",
			PotentialSavings: 50.0,
			Difficulty:       "easy",
		},
		{
			Category:         "EC2 Optimization",
			Description:      "Review instance size - consider downsizing if underutilized",
			PotentialSavings: 30.0,
			Difficulty:       "medium",
		},
		{
			Category:         "EC2 Optimization",
			Description:      "Enable auto-scaling to optimize instance count",
			PotentialSavings: 25.0,
			Difficulty:       "hard",
		},
	}
}

// getRDSOptimizationTips provides RDS-specific optimization tips
func (ca *CostAnalyzer) getRDSOptimizationTips(resource models.Resource) []OptimizationTip {
	return []OptimizationTip{
		{
			Category:         "RDS Optimization",
			Description:      "Consider using Aurora Serverless for variable workloads",
			PotentialSavings: 40.0,
			Difficulty:       "medium",
		},
		{
			Category:         "RDS Optimization",
			Description:      "Review storage allocation and consider reducing if over-provisioned",
			PotentialSavings: 20.0,
			Difficulty:       "easy",
		},
	}
}

// getS3OptimizationTips provides S3-specific optimization tips
func (ca *CostAnalyzer) getS3OptimizationTips(resource models.Resource) []OptimizationTip {
	return []OptimizationTip{
		{
			Category:         "S3 Optimization",
			Description:      "Implement lifecycle policies to move data to cheaper storage tiers",
			PotentialSavings: 60.0,
			Difficulty:       "medium",
		},
		{
			Category:         "S3 Optimization",
			Description:      "Enable intelligent tiering for automatic cost optimization",
			PotentialSavings: 30.0,
			Difficulty:       "easy",
		},
	}
}

// getEKSOptimizationTips provides EKS-specific optimization tips
func (ca *CostAnalyzer) getEKSOptimizationTips(resource models.Resource) []OptimizationTip {
	return []OptimizationTip{
		{
			Category:         "EKS Optimization",
			Description:      "Use Spot Instances for worker nodes in non-critical workloads",
			PotentialSavings: 70.0,
			Difficulty:       "medium",
		},
		{
			Category:         "EKS Optimization",
			Description:      "Implement cluster autoscaler to optimize node count",
			PotentialSavings: 40.0,
			Difficulty:       "hard",
		},
		{
			Category:         "EKS Optimization",
			Description:      "Review and optimize resource requests/limits",
			PotentialSavings: 25.0,
			Difficulty:       "medium",
		},
	}
}

// parseCost parses cost string to float64
func parseCost(costStr string) float64 {
	var cost float64
	fmt.Sscanf(costStr, "%f", &cost)
	return cost
}

// Helper function to create AWS string pointer
func awsString(s string) *string {
	return &s
}
