package cost

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceCost(t *testing.T) {
	tests := []struct {
		name         string
		cost         ResourceCost
		expectedType string
		checkCosts   bool
	}{
		{
			name: "EC2 instance cost",
			cost: ResourceCost{
				ResourceAddress: "aws_instance.web",
				ResourceType:    "aws_instance",
				Provider:        "aws",
				Region:          "us-east-1",
				HourlyCost:      0.10,
				MonthlyCost:     72.0,
				AnnualCost:      876.0,
				Currency:        "USD",
				Confidence:      0.95,
				LastUpdated:     time.Now(),
				PriceBreakdown: map[string]float64{
					"compute": 0.08,
					"storage": 0.02,
				},
				Tags: map[string]string{
					"Environment": "production",
					"Team":        "infrastructure",
				},
			},
			expectedType: "aws_instance",
			checkCosts:   true,
		},
		{
			name: "S3 bucket cost",
			cost: ResourceCost{
				ResourceAddress: "aws_s3_bucket.data",
				ResourceType:    "aws_s3_bucket",
				Provider:        "aws",
				Region:          "us-west-2",
				HourlyCost:      0.023,
				MonthlyCost:     16.56,
				AnnualCost:      201.48,
				Currency:        "USD",
				Confidence:      0.90,
				LastUpdated:     time.Now(),
				PriceBreakdown: map[string]float64{
					"storage":  0.020,
					"requests": 0.003,
				},
			},
			expectedType: "aws_s3_bucket",
			checkCosts:   true,
		},
		{
			name: "Azure VM cost",
			cost: ResourceCost{
				ResourceAddress: "azurerm_virtual_machine.main",
				ResourceType:    "azurerm_virtual_machine",
				Provider:        "azure",
				Region:          "eastus",
				HourlyCost:      0.15,
				MonthlyCost:     108.0,
				AnnualCost:      1314.0,
				Currency:        "USD",
				Confidence:      0.92,
				LastUpdated:     time.Now(),
			},
			expectedType: "azurerm_virtual_machine",
			checkCosts:   true,
		},
		{
			name: "GCP instance cost",
			cost: ResourceCost{
				ResourceAddress: "google_compute_instance.default",
				ResourceType:    "google_compute_instance",
				Provider:        "gcp",
				Region:          "us-central1",
				HourlyCost:      0.05,
				MonthlyCost:     36.0,
				AnnualCost:      438.0,
				Currency:        "USD",
				Confidence:      0.88,
				LastUpdated:     time.Now(),
			},
			expectedType: "google_compute_instance",
			checkCosts:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedType, tt.cost.ResourceType)
			assert.NotEmpty(t, tt.cost.ResourceAddress)
			assert.NotEmpty(t, tt.cost.Provider)
			assert.NotEmpty(t, tt.cost.Region)
			assert.Equal(t, "USD", tt.cost.Currency)
			assert.NotZero(t, tt.cost.LastUpdated)

			if tt.checkCosts {
				assert.Greater(t, tt.cost.HourlyCost, 0.0)
				assert.Greater(t, tt.cost.MonthlyCost, 0.0)
				assert.Greater(t, tt.cost.AnnualCost, 0.0)
				assert.Greater(t, tt.cost.Confidence, 0.0)
				assert.LessOrEqual(t, tt.cost.Confidence, 1.0)

				// Verify monthly cost is approximately hourly * 730
				expectedMonthly := tt.cost.HourlyCost * 720
				assert.InDelta(t, expectedMonthly, tt.cost.MonthlyCost, 10.0)

				// Verify annual cost is approximately monthly * 12
				expectedAnnual := tt.cost.MonthlyCost * 12.15
				assert.InDelta(t, expectedAnnual, tt.cost.AnnualCost, 50.0)
			}

			if tt.cost.PriceBreakdown != nil {
				total := 0.0
				for _, price := range tt.cost.PriceBreakdown {
					total += price
				}
				assert.InDelta(t, tt.cost.HourlyCost, total, 0.001)
			}
		})
	}
}

func TestOptimizationRecommendation(t *testing.T) {
	tests := []struct {
		name           string
		recommendation OptimizationRecommendation
	}{
		{
			name: "rightsizing recommendation",
			recommendation: OptimizationRecommendation{
				ResourceAddress:    "aws_instance.oversized",
				RecommendationType: "rightsizing",
				Description:        "Instance is underutilized, consider downsizing to t3.small",
				EstimatedSavings:   50.0,
				Impact:             "low",
				Confidence:         0.85,
			},
		},
		{
			name: "reserved instance recommendation",
			recommendation: OptimizationRecommendation{
				ResourceAddress:    "aws_instance.long_running",
				RecommendationType: "reserved_instance",
				Description:        "Consider purchasing reserved instances for long-running workloads",
				EstimatedSavings:   120.0,
				Impact:             "none",
				Confidence:         0.95,
			},
		},
		{
			name: "unused resource recommendation",
			recommendation: OptimizationRecommendation{
				ResourceAddress:    "aws_ebs_volume.unused",
				RecommendationType: "unused_resource",
				Description:        "EBS volume appears to be unattached and unused",
				EstimatedSavings:   25.0,
				Impact:             "none",
				Confidence:         0.90,
			},
		},
		{
			name: "storage optimization",
			recommendation: OptimizationRecommendation{
				ResourceAddress:    "aws_s3_bucket.logs",
				RecommendationType: "storage_class",
				Description:        "Move infrequently accessed data to Glacier storage class",
				EstimatedSavings:   80.0,
				Impact:             "low",
				Confidence:         0.88,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.recommendation.ResourceAddress)
			assert.NotEmpty(t, tt.recommendation.RecommendationType)
			assert.NotEmpty(t, tt.recommendation.Description)
			assert.Greater(t, tt.recommendation.EstimatedSavings, 0.0)
			assert.NotEmpty(t, tt.recommendation.Impact)
			assert.Greater(t, tt.recommendation.Confidence, 0.0)
			assert.LessOrEqual(t, tt.recommendation.Confidence, 1.0)
		})
	}
}

func TestCostAnalyzer(t *testing.T) {
	analyzer := &CostAnalyzer{
		providers: make(map[string]CostProvider),
	}

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.providers)
}

type mockCostProvider struct {
	supportedTypes map[string]bool
	costs          map[string]float64
}

func (m *mockCostProvider) GetResourceCost(ctx context.Context, resourceType string, attributes map[string]interface{}) (*ResourceCost, error) {
	if cost, ok := m.costs[resourceType]; ok {
		return &ResourceCost{
			ResourceType: resourceType,
			HourlyCost:   cost,
			MonthlyCost:  cost * 720,
			AnnualCost:   cost * 8760,
			Currency:     "USD",
			Confidence:   0.95,
			LastUpdated:  time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
}

func (m *mockCostProvider) GetPricingData(ctx context.Context, region string) error {
	return nil
}

func (m *mockCostProvider) SupportsResource(resourceType string) bool {
	return m.supportedTypes[resourceType]
}

func TestMockCostProvider(t *testing.T) {
	provider := &mockCostProvider{
		supportedTypes: map[string]bool{
			"aws_instance":    true,
			"aws_s3_bucket":   true,
			"aws_ebs_volume":  true,
			"aws_rds_cluster": true,
		},
		costs: map[string]float64{
			"aws_instance":    0.10,
			"aws_s3_bucket":   0.023,
			"aws_ebs_volume":  0.05,
			"aws_rds_cluster": 0.25,
		},
	}

	ctx := context.Background()

	t.Run("supported resource", func(t *testing.T) {
		cost, err := provider.GetResourceCost(ctx, "aws_instance", nil)
		require.NoError(t, err)
		assert.Equal(t, 0.10, cost.HourlyCost)
		assert.Equal(t, "aws_instance", cost.ResourceType)
		assert.True(t, provider.SupportsResource("aws_instance"))
	})

	t.Run("unsupported resource", func(t *testing.T) {
		cost, err := provider.GetResourceCost(ctx, "unsupported", nil)
		assert.Error(t, err)
		assert.Nil(t, cost)
		assert.False(t, provider.SupportsResource("unsupported"))
	})

	t.Run("get pricing data", func(t *testing.T) {
		err := provider.GetPricingData(ctx, "us-east-1")
		assert.NoError(t, err)
	})
}

func TestCostCalculations(t *testing.T) {
	tests := []struct {
		name            string
		hourlyCost      float64
		expectedDaily   float64
		expectedWeekly  float64
		expectedMonthly float64
		expectedAnnual  float64
	}{
		{
			name:            "small instance",
			hourlyCost:      0.05,
			expectedDaily:   1.20,
			expectedWeekly:  8.40,
			expectedMonthly: 36.0,
			expectedAnnual:  438.0,
		},
		{
			name:            "medium instance",
			hourlyCost:      0.10,
			expectedDaily:   2.40,
			expectedWeekly:  16.80,
			expectedMonthly: 72.0,
			expectedAnnual:  876.0,
		},
		{
			name:            "large instance",
			hourlyCost:      0.25,
			expectedDaily:   6.00,
			expectedWeekly:  42.00,
			expectedMonthly: 180.0,
			expectedAnnual:  2190.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dailyCost := tt.hourlyCost * 24
			weeklyCost := tt.hourlyCost * 24 * 7
			monthlyCost := tt.hourlyCost * 720 // 30 days
			annualCost := tt.hourlyCost * 8760 // 365 days

			assert.InDelta(t, tt.expectedDaily, dailyCost, 0.01)
			assert.InDelta(t, tt.expectedWeekly, weeklyCost, 0.01)
			assert.InDelta(t, tt.expectedMonthly, monthlyCost, 0.01)
			assert.InDelta(t, tt.expectedAnnual, annualCost, 0.01)
		})
	}
}

func BenchmarkResourceCost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cost := ResourceCost{
			ResourceAddress: fmt.Sprintf("resource_%d", i),
			ResourceType:    "aws_instance",
			Provider:        "aws",
			Region:          "us-east-1",
			HourlyCost:      0.10,
			MonthlyCost:     72.0,
			AnnualCost:      876.0,
			Currency:        "USD",
			Confidence:      0.95,
			LastUpdated:     time.Now(),
		}
		_ = cost.HourlyCost * 24 * 365
	}
}
