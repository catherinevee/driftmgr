package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/catherinevee/driftmgr/internal/importer"
)

// handleImport handles the import command
func handleImport(args []string) {
	config := importer.Config{
		Parallelism:    5,
		DryRun:         false,
		GenerateConfig: true,
		ValidateAfter:  true,
	}

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--input", "-i":
			if i+1 < len(args) {
				config.InputFile = args[i+1]
				i++
			}
		case "--parallelism", "-p":
			if i+1 < len(args) {
				if p, err := strconv.Atoi(args[i+1]); err == nil {
					config.Parallelism = p
				}
				i++
			}
		case "--dry-run":
			config.DryRun = true
		case "--no-generate":
			config.GenerateConfig = false
		case "--no-validate":
			config.ValidateAfter = false
		case "--help", "-h":
			printImportHelp()
			return
		}
	}

	// Check if input file is provided
	if config.InputFile == "" {
		fmt.Println("Error: Input file is required")
		fmt.Println("Use --input <file> to specify the resources file")
		fmt.Println()
		printImportHelp()
		os.Exit(1)
	}

	// Create import engine
	engine := importer.NewEngine()

	// Execute import
	fmt.Printf("🚀 Starting import from %s\n", config.InputFile)
	result, err := engine.Import(config)
	if err != nil {
		fmt.Printf("[ERROR] Import failed: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Println()
	fmt.Println("📊 Import Results:")
	fmt.Printf("  [OK] Successful: %d\n", result.Successful)
	fmt.Printf("  [ERROR] Failed: %d\n", result.Failed)
	fmt.Printf("  ⏱️  Duration: %v\n", result.Duration)

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("[WARNING]  Errors encountered:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s: %s\n", err.Resource, err.Error)
		}
	}

	if result.Successful > 0 && !config.DryRun {
		fmt.Println()
		fmt.Println("[OK] Import completed successfully!")
		if config.GenerateConfig {
			fmt.Println("📝 Terraform configuration files have been generated")
		}
	}
}

func printImportHelp() {
	fmt.Println("Usage: driftmgr import --input <file> [options]")
	fmt.Println()
	fmt.Println("Import existing cloud resources into Terraform")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --input, -i <file>        Input file (CSV or JSON) containing resources to import")
	fmt.Println("  --parallelism, -p <n>     Number of parallel imports (default: 5)")
	fmt.Println("  --dry-run                 Show what would be imported without executing")
	fmt.Println("  --no-generate             Don't generate Terraform configuration files")
	fmt.Println("  --no-validate             Skip validation after import")
	fmt.Println("  --help, -h                Show this help message")
	fmt.Println()
	fmt.Println("Input File Format (CSV):")
	fmt.Println("  provider,type,name,id")
	fmt.Println("  aws,aws_instance,web-server,i-1234567890")
	fmt.Println("  aws,aws_s3_bucket,my-bucket,my-bucket-name")
	fmt.Println()
	fmt.Println("Input File Format (JSON):")
	fmt.Println("  [")
	fmt.Println("    {")
	fmt.Println("      \"provider\": \"aws\",")
	fmt.Println("      \"type\": \"aws_instance\",")
	fmt.Println("      \"name\": \"web-server\",")
	fmt.Println("      \"id\": \"i-1234567890\"")
	fmt.Println("    }")
	fmt.Println("  ]")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  driftmgr import --input resources.csv")
	fmt.Println("  driftmgr import --input resources.json --dry-run")
	fmt.Println("  driftmgr import --input resources.csv --parallelism 10")
}
