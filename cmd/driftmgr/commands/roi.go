package commands

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/cost"
	"github.com/catherinevee/driftmgr/internal/providers"
)

// ROICalculator calculates return on investment for DriftMgr
type ROICalculator struct {
	costAnalyzer *cost.CostAnalyzer
	providers    map[string]providers.CloudProvider
}

// ROIMetrics represents ROI calculation metrics
type ROIMetrics struct {
	// Infrastructure metrics
	TotalResources   int     `json:"total_resources"`
	MonthlyInfraCost float64 `json:"monthly_infrastructure_cost"`
	AnnualInfraCost  float64 `json:"annual_infrastructure_cost"`

	// Drift metrics
	DriftRate        float64 `json:"drift_rate"`
	MonthlyDrifts    int     `json:"monthly_drifts"`
	IncidentRate     float64 `json:"incident_rate"`
	MonthlyIncidents int     `json:"monthly_incidents"`

	// Cost metrics
	AvgIncidentCost    float64 `json:"avg_incident_cost"`
	AvgRemediationTime float64 `json:"avg_remediation_time_hours"`
	EngineerHourlyRate float64 `json:"engineer_hourly_rate"`

	// Savings calculations
	IncidentSavings   float64 `json:"incident_savings_annual"`
	TimeSavings       float64 `json:"time_savings_annual"`
	ComplianceSavings float64 `json:"compliance_savings_annual"`
	TotalSavings      float64 `json:"total_savings_annual"`

	// ROI calculations
	DriftMgrCost  float64 `json:"driftmgr_annual_cost"`
	NetROI        float64 `json:"net_roi"`
	ROIPercentage float64 `json:"roi_percentage"`
	PaybackPeriod float64 `json:"payback_period_months"`
}

// HandleROI calculates return on investment
func HandleROI(args []string) {
	fmt.Println("\n💰 DriftMgr ROI Calculator")
	fmt.Println("=" + strings.Repeat("=", 40))

	calculator := &ROICalculator{
		costAnalyzer: cost.NewCostAnalyzer(),
		providers:    make(map[string]providers.CloudProvider),
	}

	// Initialize providers for cost analysis
	calculator.initializeProviders()

	// Calculate ROI metrics
	metrics := calculator.calculateROI()

	// Display results
	displayROIResults(metrics)
}

// initializeProviders initializes cloud providers for cost analysis
func (r *ROICalculator) initializeProviders() {
	providerNames := []string{"aws", "azure", "gcp", "digitalocean"}
	factory := providers.NewProviderFactory(nil)

	for _, providerName := range providerNames {
		provider, err := factory.CreateProvider(providerName)
		if err != nil {
			// Provider not available, skip
			continue
		}
		r.providers[providerName] = provider
	}
}

// calculateROI calculates comprehensive ROI metrics
func (r *ROICalculator) calculateROI() *ROIMetrics {
	metrics := &ROIMetrics{
		// Default values based on industry averages
		DriftRate:          0.15,  // 15% drift rate
		IncidentRate:       0.10,  // 10% of drifts cause incidents
		AvgIncidentCost:    5000,  // $5,000 per incident
		AvgRemediationTime: 4,     // 4 hours manual remediation
		EngineerHourlyRate: 150,   // $150/hour engineer rate
		ComplianceSavings:  50000, // $50,000 compliance violation prevention
		DriftMgrCost:       12000, // $12,000 annual DriftMgr cost
	}

	// Calculate infrastructure metrics
	r.calculateInfrastructureMetrics(metrics)

	// Calculate drift and incident metrics
	r.calculateDriftMetrics(metrics)

	// Calculate savings
	r.calculateSavings(metrics)

	// Calculate ROI
	r.calculateROIValues(metrics)

	return metrics
}

// calculateInfrastructureMetrics calculates infrastructure cost metrics
func (r *ROICalculator) calculateInfrastructureMetrics(metrics *ROIMetrics) {
	totalResources := 0
	monthlyCost := 0.0

	// Sample infrastructure costs based on provider
	providerCosts := map[string]float64{
		"aws":          2500, // $2,500/month for typical AWS setup
		"azure":        2200, // $2,200/month for typical Azure setup
		"gcp":          2000, // $2,000/month for typical GCP setup
		"digitalocean": 800,  // $800/month for typical DO setup
	}

	// Estimate resources and costs based on available providers
	for providerName := range r.providers {
		if cost, exists := providerCosts[providerName]; exists {
			monthlyCost += cost
			// Estimate resources based on cost (rough approximation)
			totalResources += int(cost / 10) // ~$10 per resource
		}
	}

	// If no providers available, use default estimates
	if totalResources == 0 {
		totalResources = 500 // Default 500 resources
		monthlyCost = 2000   // Default $2,000/month
	}

	metrics.TotalResources = totalResources
	metrics.MonthlyInfraCost = monthlyCost
	metrics.AnnualInfraCost = monthlyCost * 12
}

// calculateDriftMetrics calculates drift and incident metrics
func (r *ROICalculator) calculateDriftMetrics(metrics *ROIMetrics) {
	// Calculate monthly drifts based on drift rate
	metrics.MonthlyDrifts = int(float64(metrics.TotalResources) * metrics.DriftRate)

	// Calculate monthly incidents based on incident rate
	metrics.MonthlyIncidents = int(float64(metrics.MonthlyDrifts) * metrics.IncidentRate)
}

// calculateSavings calculates various types of savings
func (r *ROICalculator) calculateSavings(metrics *ROIMetrics) {
	// Incident cost savings (prevented incidents)
	metrics.IncidentSavings = float64(metrics.MonthlyIncidents) * metrics.AvgIncidentCost * 12

	// Time savings (automated vs manual remediation)
	timeSavingsPerDrift := metrics.AvgRemediationTime * metrics.EngineerHourlyRate * 0.8 // 80% time reduction
	metrics.TimeSavings = float64(metrics.MonthlyDrifts) * timeSavingsPerDrift * 12

	// Total savings
	metrics.TotalSavings = metrics.IncidentSavings + metrics.TimeSavings + metrics.ComplianceSavings
}

// calculateROIValues calculates ROI percentages and payback period
func (r *ROICalculator) calculateROIValues(metrics *ROIMetrics) {
	// Net ROI (savings - cost)
	metrics.NetROI = metrics.TotalSavings - metrics.DriftMgrCost

	// ROI percentage
	if metrics.DriftMgrCost > 0 {
		metrics.ROIPercentage = (metrics.NetROI / metrics.DriftMgrCost) * 100
	}

	// Payback period in months
	if metrics.TotalSavings > 0 {
		metrics.PaybackPeriod = (metrics.DriftMgrCost / metrics.TotalSavings) * 12
	}
}

// displayROIResults displays the ROI calculation results
func displayROIResults(metrics *ROIMetrics) {
	fmt.Println("\n📊 Infrastructure Analysis:")
	fmt.Printf("  • Total Resources: %d\n", metrics.TotalResources)
	fmt.Printf("  • Monthly Infrastructure Cost: $%.2f\n", metrics.MonthlyInfraCost)
	fmt.Printf("  • Annual Infrastructure Cost: $%.2f\n", metrics.AnnualInfraCost)

	fmt.Println("\n🔍 Drift & Incident Analysis:")
	fmt.Printf("  • Drift Rate: %.1f%%\n", metrics.DriftRate*100)
	fmt.Printf("  • Monthly Drifts: %d\n", metrics.MonthlyDrifts)
	fmt.Printf("  • Incident Rate: %.1f%%\n", metrics.IncidentRate*100)
	fmt.Printf("  • Monthly Incidents: %d\n", metrics.MonthlyIncidents)

	fmt.Println("\n💰 Cost Analysis:")
	fmt.Printf("  • Average Incident Cost: $%.2f\n", metrics.AvgIncidentCost)
	fmt.Printf("  • Average Remediation Time: %.1f hours\n", metrics.AvgRemediationTime)
	fmt.Printf("  • Engineer Hourly Rate: $%.2f\n", metrics.EngineerHourlyRate)

	fmt.Println("\n💵 Annual Savings Breakdown:")
	fmt.Printf("  • Incident Prevention: $%.2f\n", metrics.IncidentSavings)
	fmt.Printf("  • Time Savings: $%.2f\n", metrics.TimeSavings)
	fmt.Printf("  • Compliance Savings: $%.2f\n", metrics.ComplianceSavings)
	fmt.Printf("  • Total Annual Savings: $%.2f\n", metrics.TotalSavings)

	fmt.Println("\n📈 ROI Analysis:")
	fmt.Printf("  • DriftMgr Annual Cost: $%.2f\n", metrics.DriftMgrCost)
	fmt.Printf("  • Net Annual ROI: $%.2f\n", metrics.NetROI)
	fmt.Printf("  • ROI Percentage: %.1f%%\n", metrics.ROIPercentage)
	fmt.Printf("  • Payback Period: %.1f months\n", metrics.PaybackPeriod)

	// ROI assessment
	fmt.Println("\n🎯 ROI Assessment:")
	if metrics.ROIPercentage > 300 {
		fmt.Println("  • EXCELLENT ROI - Highly recommended investment")
	} else if metrics.ROIPercentage > 200 {
		fmt.Println("  • STRONG ROI - Strongly recommended investment")
	} else if metrics.ROIPercentage > 100 {
		fmt.Println("  • GOOD ROI - Recommended investment")
	} else if metrics.ROIPercentage > 50 {
		fmt.Println("  • MODERATE ROI - Consider investment")
	} else {
		fmt.Println("  • LOW ROI - Evaluate carefully")
	}

	// Additional insights
	fmt.Println("\n💡 Key Insights:")
	if metrics.PaybackPeriod < 6 {
		fmt.Println("  • Quick payback period - low risk investment")
	}
	if metrics.MonthlyIncidents > 5 {
		fmt.Println("  • High incident frequency - significant savings potential")
	}
	if metrics.TotalResources > 1000 {
		fmt.Println("  • Large infrastructure - economies of scale benefits")
	}

	// Recommendations
	fmt.Println("\n🚀 Recommendations:")
	if metrics.ROIPercentage > 100 {
		fmt.Println("  • Implement DriftMgr immediately")
		fmt.Println("  • Consider enterprise features for additional savings")
	} else {
		fmt.Println("  • Start with pilot implementation")
		fmt.Println("  • Focus on high-risk resources first")
	}
}
