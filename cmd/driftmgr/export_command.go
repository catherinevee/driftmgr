package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// handleExportCommand handles exporting discovery results
func handleExportCommand(args []string) {
	var inputFile, outputFile, format, provider string
	var allAccounts bool
	format = "json" // default format

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--input":
			if i+1 < len(args) {
				inputFile = args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--all-accounts":
			allAccounts = true
		case "--help", "-h":
			showExportHelp()
			return
		}
	}

	// If no output file specified, generate one based on format
	if outputFile == "" {
		timestamp := time.Now().Format("20060102-150405")
		switch format {
		case "csv":
			outputFile = fmt.Sprintf("driftmgr-export-%s.csv", timestamp)
		case "html":
			outputFile = fmt.Sprintf("driftmgr-export-%s.html", timestamp)
		case "excel":
			outputFile = fmt.Sprintf("driftmgr-export-%s.xlsx", timestamp)
		case "terraform":
			outputFile = fmt.Sprintf("driftmgr-export-%s.tf", timestamp)
		default:
			outputFile = fmt.Sprintf("driftmgr-export-%s.json", timestamp)
		}
	}

	var resources []models.Resource

	// If input file is provided, read from it
	if inputFile != "" {
		data, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}

		// Parse as resources array
		if err := json.Unmarshal(data, &resources); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing input file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Perform discovery
		fmt.Println("Performing resource discovery...")

		if provider == "" && !allAccounts {
			fmt.Println("No input file provided. Discovering all configured providers...")
			provider = "all"
		}

		ctx := context.Background()
		discoveryService := discovery.NewService()

		if provider == "all" || provider == "" {
			// Discover all providers
			results := discoveryService.DiscoverAll(ctx)
			for _, result := range results {
				if result != nil {
					resources = append(resources, result.Resources...)
				}
			}
		} else {
			// Discover specific provider
			results := discoveryService.DiscoverAll(ctx)
			if result, ok := results[provider]; ok && result != nil {
				resources = result.Resources
			}
		}

		if len(resources) == 0 {
			fmt.Fprintf(os.Stderr, "No resources found\n")
			os.Exit(1)
		}
	}

	// Export based on format
	fmt.Printf("Exporting %d resources to %s format...\n", len(resources), format)

	var err error
	switch strings.ToLower(format) {
	case "csv":
		err = exportToCSV(resources, outputFile)
	case "html":
		// For now, export as formatted JSON for HTML
		err = exportToJSON(resources, outputFile)
		if err == nil {
			fmt.Println("Note: HTML export currently uses JSON format")
		}
	case "excel":
		// For now, export as CSV for Excel
		err = exportToCSV(resources, outputFile)
		if err == nil {
			fmt.Println("Note: Excel export currently uses CSV format (can be opened in Excel)")
		}
	case "terraform":
		// For now, export as JSON for Terraform
		err = exportToJSON(resources, outputFile)
		if err == nil {
			fmt.Println("Note: Terraform export currently uses JSON format")
		}
	case "json":
		err = exportToJSON(resources, outputFile)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported format: %s\n", format)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully exported to %s\n", outputFile)

	// Show summary
	fmt.Printf("\nExport Summary:\n")
	fmt.Printf("  Total Resources: %d\n", len(resources))

	// Count by provider
	providerCount := make(map[string]int)
	for _, resource := range resources {
		providerCount[resource.Provider]++
	}

	fmt.Printf("  By Provider:\n")
	for provider, count := range providerCount {
		fmt.Printf("    - %s: %d\n", provider, count)
	}

	// Count by type
	typeCount := make(map[string]int)
	for _, resource := range resources {
		typeCount[resource.Type]++
	}

	fmt.Printf("  Top Resource Types:\n")
	count := 0
	for resType, resCount := range typeCount {
		if count >= 5 {
			break
		}
		fmt.Printf("    - %s: %d\n", resType, resCount)
		count++
	}
}

// exportToCSV exports resources to CSV format
func exportToCSV(resources []models.Resource, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Name", "Type", "Provider", "Region", "State", "Created", "Tags"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write resources
	for _, resource := range resources {
		tags := ""
		tagsMap := resource.GetTagsAsMap()
		for k, v := range tagsMap {
			tags += fmt.Sprintf("%s=%s;", k, v)
		}

		// Handle State as interface{} - could be string or map
		stateStr := ""
		if s, ok := resource.State.(string); ok {
			stateStr = s
		} else if resource.State != nil {
			stateStr = fmt.Sprintf("%v", resource.State)
		}
		
		record := []string{
			resource.ID,
			resource.Name,
			resource.Type,
			resource.Provider,
			resource.Region,
			stateStr,
			resource.CreatedAt.Format(time.RFC3339),
			tags,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// exportToJSON exports resources to JSON format
func exportToJSON(resources []models.Resource, outputFile string) error {
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputFile, data, 0644)
}

// showExportHelp displays help for export command
func showExportHelp() {
	fmt.Println("Usage: driftmgr export [flags]")
	fmt.Println()
	fmt.Println("Export discovery results to various formats")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --input string     Input file (JSON) to export from")
	fmt.Println("  --output string    Output file path")
	fmt.Println("  --format string    Export format: json, csv, html, excel, terraform (default: json)")
	fmt.Println("  --provider string  Provider to discover and export (if no input file)")
	fmt.Println("  --all-accounts     Include all accessible accounts")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Export current discovery to CSV")
	fmt.Println("  driftmgr export --format csv")
	fmt.Println()
	fmt.Println("  # Export specific provider to HTML")
	fmt.Println("  driftmgr export --provider aws --format html")
	fmt.Println()
	fmt.Println("  # Export from saved discovery results")
	fmt.Println("  driftmgr export --input discovery.json --format excel")
	fmt.Println()
	fmt.Println("  # Export all accounts to Terraform")
	fmt.Println("  driftmgr export --all-accounts --format terraform")
}
