package main

import (
	"fmt"

	"github.com/catherinevee/driftmgr/internal/analysis"
	"github.com/catherinevee/driftmgr/internal/models"
)

// handleImpactAnalysis processes impact analysis commands
func (shell *InteractiveShell) handleImpactAnalysis(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  analyze <drift_id>      - Analyze impact of drift")
		fmt.Println("  business <drift_id>     - Analyze business impact")
		fmt.Println("  cost <drift_id>         - Analyze cost impact")
		fmt.Println("  security <drift_id>     - Analyze security impact")
		fmt.Println("  performance <drift_id>  - Analyze performance impact")
		fmt.Println("  compliance <drift_id>   - Analyze compliance impact")
		fmt.Println("  dependencies <drift_id> - Analyze dependency impact")
		return
	}

	command := args[0]

	switch command {
	case "analyze":
		shell.handleImpactAnalyze(args[1:])
	case "business":
		shell.handleBusinessImpact(args[1:])
	case "cost":
		shell.handleCostImpact(args[1:])
	case "security":
		shell.handleSecurityImpact(args[1:])
	case "performance":
		shell.handlePerformanceImpact(args[1:])
	case "compliance":
		shell.handleComplianceImpact(args[1:])
	case "dependencies":
		shell.handleDependencyImpact(args[1:])
	default:
		fmt.Printf("Unknown impact-analysis command: %s\n", command)
	}
}

// handleImpactAnalyze handles comprehensive impact analysis
func (shell *InteractiveShell) handleImpactAnalyze(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis analyze <drift_id> [options]")
		fmt.Println("Options:")
		fmt.Println("  --business              - Enable business impact analysis")
		fmt.Println("  --cost                  - Enable cost impact analysis")
		fmt.Println("  --security              - Enable security impact analysis")
		fmt.Println("  --performance           - Enable performance impact analysis")
		fmt.Println("  --compliance            - Enable compliance impact analysis")
		return
	}

	driftID := args[0]

	// Parse options
	config := &analysis.ImpactAnalysisConfig{
		EnableBusinessImpact:    true,
		EnableCostImpact:        true,
		EnableSecurityImpact:    true,
		EnablePerformanceImpact: true,
		EnableComplianceImpact:  true,
	}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--business":
			config.EnableBusinessImpact = true
		case "--cost":
			config.EnableCostImpact = true
		case "--security":
			config.EnableSecurityImpact = true
		case "--performance":
			config.EnablePerformanceImpact = true
		case "--compliance":
			config.EnableComplianceImpact = true
		}
	}

	// Get drift result
	driftResult := shell.getDriftResult(driftID)
	if driftResult == nil {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	// Get state file
	stateFile, err := shell.client.GetStateFile("terraform") // Assuming default state file
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	// Create impact analyzer
	analyzer := analysis.NewImpactAnalyzer(config)

	// Analyze impact
	result, err := analyzer.AnalyzeImpact(*driftResult, stateFile)
	if err != nil {
		fmt.Printf("Error analyzing impact: %v\n", err)
		return
	}

	// Display comprehensive results
	fmt.Printf("Impact Analysis Results for Drift: %s\n", driftID)
	fmt.Printf("Resource: %s (%s)\n", result.ResourceName, result.ResourceType)
	fmt.Printf("Change Type: %s\n", result.ChangeType)
	fmt.Printf("Overall Risk Score: %.2f\n", result.RiskScore)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))

	// Business Impact
	if config.EnableBusinessImpact {
		fmt.Printf("\nBusiness Impact:\n")
		fmt.Printf("  Level: %s\n", result.BusinessImpact.Level)
		fmt.Printf("  Description: %s\n", result.BusinessImpact.Description)
		fmt.Printf("  Services: %v\n", result.BusinessImpact.Services)
		fmt.Printf("  Users Affected: %d\n", result.BusinessImpact.Users)
		fmt.Printf("  Revenue Impact: $%.2f\n", result.BusinessImpact.Revenue)
		fmt.Printf("  Downtime: %s\n", result.BusinessImpact.Downtime)
	}

	// Cost Impact
	if config.EnableCostImpact {
		fmt.Printf("\nCost Impact:\n")
		fmt.Printf("  Level: %s\n", result.CostImpact.Level)
		fmt.Printf("  Monthly Cost: $%.2f\n", result.CostImpact.MonthlyCost)
		fmt.Printf("  Annual Cost: $%.2f\n", result.CostImpact.AnnualCost)
		fmt.Printf("  Cost Change: $%.2f\n", result.CostImpact.CostChange)
		fmt.Printf("  Description: %s\n", result.CostImpact.Description)
	}

	// Security Impact
	if config.EnableSecurityImpact {
		fmt.Printf("\nSecurity Impact:\n")
		fmt.Printf("  Level: %s\n", result.SecurityImpact.Level)
		fmt.Printf("  Risk Factors: %v\n", result.SecurityImpact.RiskFactors)
		fmt.Printf("  Vulnerabilities: %v\n", result.SecurityImpact.Vulnerabilities)
		fmt.Printf("  Compliance: %v\n", result.SecurityImpact.Compliance)
		fmt.Printf("  Description: %s\n", result.SecurityImpact.Description)
	}

	// Performance Impact
	if config.EnablePerformanceImpact {
		fmt.Printf("\nPerformance Impact:\n")
		fmt.Printf("  Level: %s\n", result.PerformanceImpact.Level)
		fmt.Printf("  Latency: %.2f ms\n", result.PerformanceImpact.Latency)
		fmt.Printf("  Throughput: %.2f req/sec\n", result.PerformanceImpact.Throughput)
		fmt.Printf("  Availability: %.2f%%\n", result.PerformanceImpact.Availability)
		fmt.Printf("  Description: %s\n", result.PerformanceImpact.Description)
	}

	// Compliance Impact
	if config.EnableComplianceImpact {
		fmt.Printf("\nCompliance Impact:\n")
		fmt.Printf("  Level: %s\n", result.ComplianceImpact.Level)
		fmt.Printf("  Standards: %v\n", result.ComplianceImpact.Standards)
		fmt.Printf("  Requirements: %v\n", result.ComplianceImpact.Requirements)
		fmt.Printf("  Description: %s\n", result.ComplianceImpact.Description)
	}

	// Dependencies
	if len(result.Dependencies) > 0 {
		fmt.Printf("\nDependency Impact:\n")
		for i, dep := range result.Dependencies {
			fmt.Printf("  %d. %s (%s) - %s (Risk: %.2f)\n",
				i+1, dep.ResourceName, dep.ResourceID, dep.Description, dep.RiskScore)
		}
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Printf("\nRecommendations:\n")
		for i, rec := range result.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}
}

// handleBusinessImpact handles business impact analysis
func (shell *InteractiveShell) handleBusinessImpact(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis business <drift_id>")
		return
	}

	driftID := args[0]
	driftResult := shell.getDriftResult(driftID)
	if driftResult == nil {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	stateFile, err := shell.client.GetStateFile("terraform")
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	config := &analysis.ImpactAnalysisConfig{
		EnableBusinessImpact:    true,
		EnableCostImpact:        false,
		EnableSecurityImpact:    false,
		EnablePerformanceImpact: false,
		EnableComplianceImpact:  false,
	}

	analyzer := analysis.NewImpactAnalyzer(config)
	result, err := analyzer.AnalyzeImpact(*driftResult, stateFile)
	if err != nil {
		fmt.Printf("Error analyzing business impact: %v\n", err)
		return
	}

	fmt.Printf("Business Impact Analysis for Drift: %s\n", driftID)
	fmt.Printf("Level: %s\n", result.BusinessImpact.Level)
	fmt.Printf("Description: %s\n", result.BusinessImpact.Description)
	fmt.Printf("Services Affected: %v\n", result.BusinessImpact.Services)
	fmt.Printf("Users Affected: %d\n", result.BusinessImpact.Users)
	fmt.Printf("Revenue Impact: $%.2f\n", result.BusinessImpact.Revenue)
	fmt.Printf("Downtime: %s\n", result.BusinessImpact.Downtime)
}

// handleCostImpact handles cost impact analysis
func (shell *InteractiveShell) handleCostImpact(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis cost <drift_id>")
		return
	}

	driftID := args[0]
	driftResult := shell.getDriftResult(driftID)
	if driftResult == nil {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	stateFile, err := shell.client.GetStateFile("terraform")
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	config := &analysis.ImpactAnalysisConfig{
		EnableBusinessImpact:    false,
		EnableCostImpact:        true,
		EnableSecurityImpact:    false,
		EnablePerformanceImpact: false,
		EnableComplianceImpact:  false,
	}

	analyzer := analysis.NewImpactAnalyzer(config)
	result, err := analyzer.AnalyzeImpact(*driftResult, stateFile)
	if err != nil {
		fmt.Printf("Error analyzing cost impact: %v\n", err)
		return
	}

	fmt.Printf("Cost Impact Analysis for Drift: %s\n", driftID)
	fmt.Printf("Level: %s\n", result.CostImpact.Level)
	fmt.Printf("Monthly Cost: $%.2f\n", result.CostImpact.MonthlyCost)
	fmt.Printf("Annual Cost: $%.2f\n", result.CostImpact.AnnualCost)
	fmt.Printf("Cost Change: $%.2f\n", result.CostImpact.CostChange)
	fmt.Printf("Currency: %s\n", result.CostImpact.Currency)
	fmt.Printf("Description: %s\n", result.CostImpact.Description)
}

// handleSecurityImpact handles security impact analysis
func (shell *InteractiveShell) handleSecurityImpact(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis security <drift_id>")
		return
	}

	driftID := args[0]
	driftResult := shell.getDriftResult(driftID)
	if driftResult == nil {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	stateFile, err := shell.client.GetStateFile("terraform")
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	config := &analysis.ImpactAnalysisConfig{
		EnableBusinessImpact:    false,
		EnableCostImpact:        false,
		EnableSecurityImpact:    true,
		EnablePerformanceImpact: false,
		EnableComplianceImpact:  false,
	}

	analyzer := analysis.NewImpactAnalyzer(config)
	result, err := analyzer.AnalyzeImpact(*driftResult, stateFile)
	if err != nil {
		fmt.Printf("Error analyzing security impact: %v\n", err)
		return
	}

	fmt.Printf("Security Impact Analysis for Drift: %s\n", driftID)
	fmt.Printf("Level: %s\n", result.SecurityImpact.Level)
	fmt.Printf("Risk Factors: %v\n", result.SecurityImpact.RiskFactors)
	fmt.Printf("Vulnerabilities: %v\n", result.SecurityImpact.Vulnerabilities)
	fmt.Printf("Compliance: %v\n", result.SecurityImpact.Compliance)
	fmt.Printf("Description: %s\n", result.SecurityImpact.Description)
}

// handlePerformanceImpact handles performance impact analysis
func (shell *InteractiveShell) handlePerformanceImpact(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis performance <drift_id>")
		return
	}

	driftID := args[0]
	driftResult := shell.getDriftResult(driftID)
	if driftResult == nil {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	stateFile, err := shell.client.GetStateFile("terraform")
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	config := &analysis.ImpactAnalysisConfig{
		EnableBusinessImpact:    false,
		EnableCostImpact:        false,
		EnableSecurityImpact:    false,
		EnablePerformanceImpact: true,
		EnableComplianceImpact:  false,
	}

	analyzer := analysis.NewImpactAnalyzer(config)
	result, err := analyzer.AnalyzeImpact(*driftResult, stateFile)
	if err != nil {
		fmt.Printf("Error analyzing performance impact: %v\n", err)
		return
	}

	fmt.Printf("Performance Impact Analysis for Drift: %s\n", driftID)
	fmt.Printf("Level: %s\n", result.PerformanceImpact.Level)
	fmt.Printf("Latency: %.2f ms\n", result.PerformanceImpact.Latency)
	fmt.Printf("Throughput: %.2f req/sec\n", result.PerformanceImpact.Throughput)
	fmt.Printf("Availability: %.2f%%\n", result.PerformanceImpact.Availability)
	fmt.Printf("Description: %s\n", result.PerformanceImpact.Description)
}

// handleComplianceImpact handles compliance impact analysis
func (shell *InteractiveShell) handleComplianceImpact(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis compliance <drift_id>")
		return
	}

	driftID := args[0]
	driftResult := shell.getDriftResult(driftID)
	if driftResult == nil {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	stateFile, err := shell.client.GetStateFile("terraform")
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	config := &analysis.ImpactAnalysisConfig{
		EnableBusinessImpact:    false,
		EnableCostImpact:        false,
		EnableSecurityImpact:    false,
		EnablePerformanceImpact: false,
		EnableComplianceImpact:  true,
	}

	analyzer := analysis.NewImpactAnalyzer(config)
	result, err := analyzer.AnalyzeImpact(*driftResult, stateFile)
	if err != nil {
		fmt.Printf("Error analyzing compliance impact: %v\n", err)
		return
	}

	fmt.Printf("Compliance Impact Analysis for Drift: %s\n", driftID)
	fmt.Printf("Level: %s\n", result.ComplianceImpact.Level)
	fmt.Printf("Standards: %v\n", result.ComplianceImpact.Standards)
	fmt.Printf("Requirements: %v\n", result.ComplianceImpact.Requirements)
	fmt.Printf("Description: %s\n", result.ComplianceImpact.Description)
}

// handleDependencyImpact handles dependency impact analysis
func (shell *InteractiveShell) handleDependencyImpact(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: impact-analysis dependencies <drift_id>")
		return
	}

	driftID := args[0]
	driftResult := shell.getDriftResult(driftID)
	if driftResult == nil {
		fmt.Printf("No drift found with ID: %s\n", driftID)
		return
	}

	stateFile, err := shell.client.GetStateFile("terraform")
	if err != nil {
		fmt.Printf("Error getting state file: %v\n", err)
		return
	}

	config := &analysis.ImpactAnalysisConfig{
		EnableBusinessImpact:    false,
		EnableCostImpact:        false,
		EnableSecurityImpact:    false,
		EnablePerformanceImpact: false,
		EnableComplianceImpact:  false,
	}

	analyzer := analysis.NewImpactAnalyzer(config)
	result, err := analyzer.AnalyzeImpact(*driftResult, stateFile)
	if err != nil {
		fmt.Printf("Error analyzing dependency impact: %v\n", err)
		return
	}

	fmt.Printf("Dependency Impact Analysis for Drift: %s\n", driftID)
	if len(result.Dependencies) == 0 {
		fmt.Printf("No dependencies found\n")
		return
	}

	for i, dep := range result.Dependencies {
		fmt.Printf("  %d. %s (%s)\n", i+1, dep.ResourceName, dep.ResourceID)
		fmt.Printf("     Impact Level: %s\n", dep.ImpactLevel)
		fmt.Printf("     Description: %s\n", dep.Description)
		fmt.Printf("     Risk Score: %.2f\n", dep.RiskScore)
	}
}

// getDriftResult gets a drift result by ID (placeholder implementation)
func (shell *InteractiveShell) getDriftResult(driftID string) *models.DriftResult {
	// In a real implementation, this would fetch from storage
	// For now, return nil
	return nil
}
