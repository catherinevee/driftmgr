package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/olekukonko/tablewriter"
)

// DiscoveryVisualizer provides visualization capabilities for discovery results
type DiscoveryVisualizer struct {
	resources []models.Resource
	errors    map[string]error
	warnings  map[string]string
	stats     *DiscoveryStats
}

// DiscoveryStats contains statistics about the discovery results
type DiscoveryStats struct {
	TotalResources      int                `json:"total_resources"`
	ResourcesByType     map[string]int     `json:"resources_by_type"`
	ResourcesByRegion   map[string]int     `json:"resources_by_region"`
	ResourcesByProvider map[string]int     `json:"resources_by_provider"`
	Errors              int                `json:"errors"`
	Warnings            int                `json:"warnings"`
	DiscoveryTime       time.Duration      `json:"discovery_time"`
	CostEstimate        map[string]float64 `json:"cost_estimate"`
	SecurityScore       map[string]int     `json:"security_score"`
	ComplianceStatus    map[string]string  `json:"compliance_status"`
}

// NewDiscoveryVisualizer creates a new discovery visualizer
func NewDiscoveryVisualizer(resources []models.Resource, errors map[string]error, warnings map[string]string, discoveryTime time.Duration) *DiscoveryVisualizer {
	visualizer := &DiscoveryVisualizer{
		resources: resources,
		errors:    errors,
		warnings:  warnings,
		stats:     &DiscoveryStats{},
	}

	visualizer.calculateStats(discoveryTime)
	return visualizer
}

// calculateStats calculates comprehensive statistics from the discovery results
func (dv *DiscoveryVisualizer) calculateStats(discoveryTime time.Duration) {
	dv.stats.TotalResources = len(dv.resources)
	dv.stats.ResourcesByType = make(map[string]int)
	dv.stats.ResourcesByRegion = make(map[string]int)
	dv.stats.ResourcesByProvider = make(map[string]int)
	dv.stats.Errors = len(dv.errors)
	dv.stats.Warnings = len(dv.warnings)
	dv.stats.DiscoveryTime = discoveryTime
	dv.stats.CostEstimate = make(map[string]float64)
	dv.stats.SecurityScore = make(map[string]int)
	dv.stats.ComplianceStatus = make(map[string]string)

	for _, resource := range dv.resources {
		// Count by type
		dv.stats.ResourcesByType[resource.Type]++

		// Count by region
		dv.stats.ResourcesByRegion[resource.Region]++

		// Count by provider
		dv.stats.ResourcesByProvider[resource.Provider]++

		// Calculate cost estimate (basic) with safe type assertion
		if cost, ok := resource.Properties["estimated_cost"]; ok {
			switch costFloat := cost.(type) {
			case float64:
				dv.stats.CostEstimate[resource.Provider] += costFloat
			case int:
				dv.stats.CostEstimate[resource.Provider] += float64(costFloat)
			case int64:
				dv.stats.CostEstimate[resource.Provider] += float64(costFloat)
			default:
				// Log warning for unexpected type
				fmt.Printf("Warning: unexpected cost type for resource %s: %T\n", resource.ID, cost)
			}
		}

		// Calculate security score with safe type assertion
		if securityScore, ok := resource.Properties["security_score"]; ok {
			switch score := securityScore.(type) {
			case int:
				dv.stats.SecurityScore[resource.Provider] += score
			case float64:
				dv.stats.SecurityScore[resource.Provider] += int(score)
			case int64:
				dv.stats.SecurityScore[resource.Provider] += int(score)
			default:
				// Log warning for unexpected type
				fmt.Printf("Warning: unexpected security score type for resource %s: %T\n", resource.ID, securityScore)
			}
		}
	}
}

// PrintSummary prints a comprehensive summary of discovery results
func (dv *DiscoveryVisualizer) PrintSummary() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                    DISCOVERY RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	// Basic stats
	fmt.Printf("Total Resources Discovered: %d\n", dv.stats.TotalResources)
	fmt.Printf("Discovery Time: %v\n", dv.stats.DiscoveryTime)
	fmt.Printf("Errors: %d | Warnings: %d\n", dv.stats.Errors, dv.stats.Warnings)

	// Resources by provider
	fmt.Println("\nResources by Provider:")
	for provider, count := range dv.stats.ResourcesByProvider {
		fmt.Printf("  %s: %d resources\n", provider, count)
	}

	// Resources by region
	fmt.Println("\nResources by Region:")
	for region, count := range dv.stats.ResourcesByRegion {
		fmt.Printf("  %s: %d resources\n", region, count)
	}

	// Top resource types
	fmt.Println("\nTop Resource Types:")
	typeCounts := make([]struct {
		Type  string
		Count int
	}, 0, len(dv.stats.ResourcesByType))

	for resourceType, count := range dv.stats.ResourcesByType {
		typeCounts = append(typeCounts, struct {
			Type  string
			Count int
		}{resourceType, count})
	}

	sort.Slice(typeCounts, func(i, j int) bool {
		return typeCounts[i].Count > typeCounts[j].Count
	})

	for i, tc := range typeCounts {
		if i >= 10 { // Show top 10
			break
		}
		fmt.Printf("  %s: %d resources\n", tc.Type, tc.Count)
	}

	// Cost estimates
	if len(dv.stats.CostEstimate) > 0 {
		fmt.Println("\nEstimated Monthly Costs:")
		for provider, cost := range dv.stats.CostEstimate {
			fmt.Printf("  %s: $%.2f\n", provider, cost)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}

// PrintDetailedTable prints a detailed table of all resources
func (dv *DiscoveryVisualizer) PrintDetailedTable() {
	if len(dv.resources) == 0 {
		fmt.Println("No resources discovered.")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Provider", "Type", "Name", "Region", "Created", "Tags"})
	table.SetBorder(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	for _, resource := range dv.resources {
		// Format tags
		resourceTags := resource.GetTagsAsMap()
		tags := make([]string, 0, len(resourceTags))
		for k, v := range resourceTags {
			tags = append(tags, fmt.Sprintf("%s=%s", k, v))
		}
		tagsStr := strings.Join(tags, ", ")
		if len(tagsStr) > 50 {
			tagsStr = tagsStr[:47] + "..."
		}

		// Format creation time
		createdStr := "Unknown"
		if !resource.CreatedAt.IsZero() {
			createdStr = resource.CreatedAt.Format("2006-01-02 15:04")
		}

		table.Append([]string{
			resource.Provider,
			resource.Type,
			resource.Name,
			resource.Region,
			createdStr,
			tagsStr,
		})
	}

	fmt.Println("\nDetailed Resource List:")
	table.Render()
}

// PrintErrors prints all discovery errors
func (dv *DiscoveryVisualizer) PrintErrors() {
	if len(dv.errors) == 0 {
		return
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("DISCOVERY ERRORS:")
	fmt.Println(strings.Repeat("-", 80))

	for key, err := range dv.errors {
		fmt.Printf("[ERROR] %s: %v\n", key, err)
	}
}

// PrintWarnings prints all discovery warnings
func (dv *DiscoveryVisualizer) PrintWarnings() {
	if len(dv.warnings) == 0 {
		return
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("DISCOVERY WARNINGS:")
	fmt.Println(strings.Repeat("-", 80))

	for key, warning := range dv.warnings {
		fmt.Printf("[WARNING] %s: %s\n", key, warning)
	}
}

// PrintJSON prints the discovery results in JSON format
func (dv *DiscoveryVisualizer) PrintJSON() {
	result := map[string]interface{}{
		"stats":     dv.stats,
		"resources": dv.resources,
		"errors":    dv.errors,
		"warnings":  dv.warnings,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

// PrintCSV prints the discovery results in CSV format
func (dv *DiscoveryVisualizer) PrintCSV() {
	if len(dv.resources) == 0 {
		return
	}

	// Print header
	fmt.Println("Provider,Type,Name,Region,Created,Tags")

	// Print data
	for _, resource := range dv.resources {
		// Format tags
		resourceTags := resource.GetTagsAsMap()
		tags := make([]string, 0, len(resourceTags))
		for k, v := range resourceTags {
			tags = append(tags, fmt.Sprintf("%s=%s", k, v))
		}
		tagsStr := strings.Join(tags, ";")

		// Format creation time
		createdStr := "Unknown"
		if !resource.CreatedAt.IsZero() {
			createdStr = resource.CreatedAt.Format("2006-01-02 15:04")
		}

		fmt.Printf("%s,%s,%s,%s,%s,%s\n",
			resource.Provider,
			resource.Type,
			resource.Name,
			resource.Region,
			createdStr,
			tagsStr,
		)
	}
}

// PrintResourceTree prints resources in a hierarchical tree format
func (dv *DiscoveryVisualizer) PrintResourceTree() {
	if len(dv.resources) == 0 {
		fmt.Println("No resources discovered.")
		return
	}

	// Group resources by provider and region
	grouped := make(map[string]map[string][]models.Resource)

	for _, resource := range dv.resources {
		if grouped[resource.Provider] == nil {
			grouped[resource.Provider] = make(map[string][]models.Resource)
		}
		grouped[resource.Provider][resource.Region] = append(grouped[resource.Provider][resource.Region], resource)
	}

	fmt.Println("\nResource Hierarchy:")
	fmt.Println(strings.Repeat("=", 80))

	for provider, regions := range grouped {
		fmt.Printf("🌐 %s\n", strings.ToUpper(provider))

		for region, resources := range regions {
			fmt.Printf("  📍 %s (%d resources)\n", region, len(resources))

			// Group by type
			byType := make(map[string][]models.Resource)
			for _, resource := range resources {
				byType[resource.Type] = append(byType[resource.Type], resource)
			}

			for resourceType, typeResources := range byType {
				fmt.Printf("    🔧 %s (%d)\n", resourceType, len(typeResources))

				for _, resource := range typeResources {
					fmt.Printf("      • %s\n", resource.Name)
				}
			}
		}
		fmt.Println()
	}
}

// PrintSecurityReport prints a security-focused report
func (dv *DiscoveryVisualizer) PrintSecurityReport() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                    SECURITY REPORT")
	fmt.Println(strings.Repeat("=", 80))

	// Count security-related resources
	securityResources := make(map[string]int)
	complianceResources := make(map[string]int)

	for _, resource := range dv.resources {
		if strings.Contains(strings.ToLower(resource.Type), "security") ||
			strings.Contains(strings.ToLower(resource.Type), "iam") ||
			strings.Contains(strings.ToLower(resource.Type), "kms") ||
			strings.Contains(strings.ToLower(resource.Type), "waf") {
			securityResources[resource.Provider]++
		}

		if strings.Contains(strings.ToLower(resource.Type), "compliance") ||
			strings.Contains(strings.ToLower(resource.Type), "audit") ||
			strings.Contains(strings.ToLower(resource.Type), "policy") {
			complianceResources[resource.Provider]++
		}
	}

	fmt.Println("Security Resources by Provider:")
	for provider, count := range securityResources {
		fmt.Printf("  %s: %d security resources\n", provider, count)
	}

	fmt.Println("\nCompliance Resources by Provider:")
	for provider, count := range complianceResources {
		fmt.Printf("  %s: %d compliance resources\n", provider, count)
	}

	// Security score summary
	if len(dv.stats.SecurityScore) > 0 {
		fmt.Println("\nSecurity Scores:")
		for provider, score := range dv.stats.SecurityScore {
			fmt.Printf("  %s: %d points\n", provider, score)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}

// PrintCostReport prints a cost-focused report
func (dv *DiscoveryVisualizer) PrintCostReport() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                    COST ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	if len(dv.stats.CostEstimate) == 0 {
		fmt.Println("No cost data available.")
		return
	}

	totalCost := 0.0
	for _, cost := range dv.stats.CostEstimate {
		totalCost += cost
	}

	fmt.Printf("Total Estimated Monthly Cost: $%.2f\n", totalCost)

	fmt.Println("\nCost by Provider:")
	for provider, cost := range dv.stats.CostEstimate {
		percentage := (cost / totalCost) * 100
		fmt.Printf("  %s: $%.2f (%.1f%%)\n", provider, cost, percentage)
	}

	// Cost by resource type
	costByType := make(map[string]float64)
	for _, resource := range dv.resources {
		if cost, ok := resource.Properties["estimated_cost"]; ok {
			if costFloat, ok := cost.(float64); ok {
				costByType[resource.Type] += costFloat
			}
		}
	}

	if len(costByType) > 0 {
		fmt.Println("\nTop Cost by Resource Type:")
		typeCosts := make([]struct {
			Type string
			Cost float64
		}, 0, len(costByType))

		for resourceType, cost := range costByType {
			typeCosts = append(typeCosts, struct {
				Type string
				Cost float64
			}{resourceType, cost})
		}

		sort.Slice(typeCosts, func(i, j int) bool {
			return typeCosts[i].Cost > typeCosts[j].Cost
		})

		for i, tc := range typeCosts {
			if i >= 10 { // Show top 10
				break
			}
			fmt.Printf("  %s: $%.2f\n", tc.Type, tc.Cost)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}

// GetStats returns the discovery statistics
func (dv *DiscoveryVisualizer) GetStats() *DiscoveryStats {
	return dv.stats
}

// GetResources returns the discovered resources
func (dv *DiscoveryVisualizer) GetResources() []models.Resource {
	return dv.resources
}
