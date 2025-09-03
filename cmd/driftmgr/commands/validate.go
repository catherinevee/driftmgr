package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	// "github.com/catherinevee/driftmgr/internal/core/discovery" // Not yet implemented
)

// HandleValidate handles the validate command
func HandleValidate(args []string) {
	var provider, region, output string
	var timeout time.Duration = 5 * time.Minute

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--region", "-r":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--timeout", "-t":
			if i+1 < len(args) {
				if dur, err := time.ParseDuration(args[i+1]); err == nil {
					timeout = dur
				}
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: driftmgr validate [flags]")
			fmt.Println()
			fmt.Println("Validate cloud provider discovery accuracy")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --provider, -p string   Cloud provider (aws, azure, gcp, digitalocean, all)")
			fmt.Println("  --region, -r string     Specific region to validate (optional)")
			fmt.Println("  --output, -o string     Output file for validation report (optional)")
			fmt.Println("  --timeout, -t duration  Timeout for validation (default: 5m)")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr validate --provider aws")
			fmt.Println("  driftmgr validate --provider azure --region eastus")
			fmt.Println("  driftmgr validate --provider all --output report.txt")
			return
		}
	}

	if provider == "" {
		fmt.Println("Error: Provider is required. Use --provider flag")
		fmt.Println("Run 'driftmgr validate --help' for usage")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if provider == "all" {
		if err := validateAllProviders(ctx, output); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := validateSingleProvider(ctx, provider, region, output); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed for %s: %v\n", provider, err)
			os.Exit(1)
		}
	}

	fmt.Println("Validation completed successfully!")
}

func validateAllProviders(ctx context.Context, outputFile string) error {
	providers := []string{"aws", "azure", "gcp", "digitalocean"}

	fmt.Println("🔍 Starting comprehensive validation for all cloud providers...")
	fmt.Println(strings.Repeat("=", 80))

	allResults := make(map[string][]discovery.ValidationResult)

	for _, provider := range providers {
		fmt.Printf("\n📋 Validating %s...\n", strings.ToUpper(provider))

		validator := discovery.NewResourceCountValidator(provider)
		regions := []string{} // Use dynamic discovery

		results, err := validator.ValidateResourceCounts(ctx, regions)
		if err != nil {
			fmt.Printf("[ERROR] Validation failed for %s: %v\n", provider, err)
			continue
		}

		allResults[provider] = results
		printProviderSummary(provider, results)
	}

	if outputFile != "" {
		if err := generateReport(allResults, outputFile); err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}
		fmt.Printf("\n📄 Report saved to: %s\n", outputFile)
	}

	return nil
}

func validateSingleProvider(ctx context.Context, provider, region, outputFile string) error {
	fmt.Printf("🔍 Starting validation for %s", strings.ToUpper(provider))
	if region != "" {
		fmt.Printf(" in region %s", region)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))

	validator := discovery.NewResourceCountValidator(provider)

	var regions []string
	if region != "" {
		regions = []string{region}
	}

	results, err := validator.ValidateResourceCounts(ctx, regions)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Print detailed results
	for _, result := range results {
		printDetailedResult(result)
	}

	// Generate report
	report, _ := validator.GenerateValidationReport(results)

	if outputFile != "" {
		reportStr := fmt.Sprintf("%v", report)
		if err := os.WriteFile(outputFile, []byte(reportStr), 0644); err != nil {
			return fmt.Errorf("failed to save report: %w", err)
		}
		fmt.Printf("📄 Validation report saved to: %s\n", outputFile)
	} else {
		fmt.Println("\n📋 VALIDATION REPORT")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println(report)
	}

	return nil
}

func printProviderSummary(provider string, results []discovery.ValidationResult) {
	totalRegions := len(results)
	matchingRegions := 0
	totalResources := 0

	for _, result := range results {
		if result.Match {
			matchingRegions++
		}
		totalResources += result.DriftmgrCount
	}

	matchPercentage := float64(matchingRegions) / float64(totalRegions) * 100

	if matchingRegions == totalRegions {
		fmt.Printf("[OK] %s: %d/%d regions match (%.1f%%) - %d resources total\n",
			strings.ToUpper(provider), matchingRegions, totalRegions, matchPercentage, totalResources)
	} else {
		fmt.Printf("[ERROR] %s: %d/%d regions match (%.1f%%)\n",
			strings.ToUpper(provider), matchingRegions, totalRegions, matchPercentage)
	}
}

func printDetailedResult(result discovery.ValidationResult) {
	fmt.Printf("\n🌍 Region: %s\n", result.Region)
	fmt.Printf("   Driftmgr Count: %d\n", result.DriftmgrCount)
	fmt.Printf("   CLI Count: %d\n", result.CLICount)

	if result.Match {
		fmt.Printf("   Status: [OK] MATCH\n")
	} else {
		fmt.Printf("   Status: [ERROR] MISMATCH\n")
	}

	if result.Error != nil {
		fmt.Printf("   Error: %s\n", result.Error)
	}
}

func generateReport(allResults map[string][]discovery.ValidationResult, outputFile string) error {
	var report strings.Builder

	report.WriteString("DRIFTMGR VALIDATION REPORT\n")
	report.WriteString(strings.Repeat("=", 50) + "\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	for providerName, results := range allResults {
		validator := discovery.NewResourceCountValidator(providerName)
		providerReport, _ := validator.GenerateValidationReport(results)
		// Convert report to string
		reportStr := fmt.Sprintf("%v", providerReport)
		report.WriteString(reportStr)
		report.WriteString("\n")
	}

	return os.WriteFile(outputFile, []byte(report.String()), 0644)
}
